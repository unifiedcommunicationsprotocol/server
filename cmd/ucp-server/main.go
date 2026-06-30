package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/unifiedcommunicationsprotocol/server/internal/admin"
	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
	"github.com/unifiedcommunicationsprotocol/server/internal/logging"
	"github.com/unifiedcommunicationsprotocol/server/internal/ratelimit"
	"github.com/unifiedcommunicationsprotocol/server/internal/router"
	"github.com/unifiedcommunicationsprotocol/server/internal/store"
	"github.com/unifiedcommunicationsprotocol/server/internal/transport"
)

//go:embed public/*
var publicFS embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load config from environment
	cfg := loadConfig()

	// Initialize logging
	logger := logging.New(logging.LevelInfo)
	metrics := &logging.Metrics{}
	logger.Info("starting ucp server", "version", "0.1.0", "listen", cfg.Listen)

	// Connect to Postgres
	s, err := store.New(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err.Error())
		return fmt.Errorf("connect to database: %w", err)
	}
	defer s.Close()
	logger.Info("connected to database")

	// Initialize auth manager with database persistence
	authMgr := auth.NewWithStore(s)
	challengeStore := auth.NewChallengeStore()

	// Initialize rate limiters
	authLimiter := ratelimit.New(10, 5)     // 10 burst, 5/sec
	messageLimiter := ratelimit.New(50, 10) // 50 burst, 10/sec

	// Initialize transport hub
	hub := transport.New()

	// Initialize federation router and delivery queue
	fedRouter := router.New()
	retryQueue := router.NewRetryQueue()

	// Initialize admin event hub
	adminHub := admin.New()

	// Create HTTP router
	mux := http.NewServeMux()

	// Register well-known endpoints
	mux.HandleFunc("GET /.well-known/ucp/server-key", handleServerKey(cfg))
	mux.HandleFunc("GET /.well-known/ucp/identity/{address}", handleIdentity(s))
	mux.HandleFunc("GET /.well-known/ucp/keypackages/{address}", handleKeyPackages(s))
	mux.HandleFunc("GET /.well-known/ucp/privacy", handlePrivacy())

	// Register auth endpoints (with rate limiting)
	mux.HandleFunc("POST /auth/challenge", withRateLimit(authLimiter, handleChallenge(challengeStore)))
	mux.HandleFunc("POST /auth/session", withRateLimit(authLimiter, handleSession(authMgr, challengeStore, s, adminHub)))
	mux.HandleFunc("POST /auth/session/refresh", withRateLimit(authLimiter, handleRefresh(authMgr)))

	// Register WebSocket connection endpoint (for persistent push delivery)
	mux.HandleFunc("GET /v1/connect", handleWebSocketConnect(hub, authMgr, cfg))

	// Register API endpoints (with rate limiting)
	mux.HandleFunc("POST /api/message/send", withRateLimit(messageLimiter, handleSendMessage(authMgr, s, hub)))
	mux.HandleFunc("GET /api/inbox", handleInbox(authMgr, s))
	mux.HandleFunc("POST /api/content/upload", withRateLimit(messageLimiter, handleUploadAttachment(authMgr, s)))
	mux.HandleFunc("GET /api/content/{id}", handleDownloadAttachment(authMgr, s))
	mux.HandleFunc("POST /api/search", handleSearch(authMgr, s))

	// Register metrics endpoint
	mux.HandleFunc("GET /metrics", handleMetrics(metrics))

	// Register admin endpoints
	mux.HandleFunc("GET /api/admin/sessions", handleAdminSessions(s))
	mux.HandleFunc("GET /api/admin/federation/connections", handleAdminFederationConnections(fedRouter))
	mux.HandleFunc("GET /api/admin/federation/queue", handleAdminFederationQueue(retryQueue))
	mux.HandleFunc("GET /api/admin/subscribe", handleAdminSubscribe(adminHub))

	// Serve React dashboard (SPA)
	publicFiles, err := fs.Sub(publicFS, "public")
	if err == nil {
		// Serve static files with fallback to index.html for SPA routing
		mux.Handle("/", serveIndexFallback(publicFiles))
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Listen,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		fmt.Printf("UCP Server listening on %s\n", cfg.Listen)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	fmt.Println("UCP Server stopped")
	return nil
}

type config struct {
	Listen      string
	DatabaseURL string
	ServerDomain string
	ServerKey   string // Base64-encoded Ed25519 private key
}

func loadConfig() config {
	dbURL := getEnv("DATABASE_URL", "postgres://localhost:6432/ucp")
	// Convert postgres:// URL to keyword/value format for pq
	// postgres://user:pass@host:port/db?sslmode=disable → user=X password=Y host=Z port=N dbname=...
	if strings.HasPrefix(dbURL, "postgres://") {
		dbURL = convertPgURL(dbURL)
	}
	return config{
		Listen:       getEnv("API_PORT", ":6001"),
		DatabaseURL: dbURL,
		ServerDomain: getEnv("API_URL", "localhost:6001"),
		ServerKey:    getEnv("UCP_SERVER_KEY", ""),
	}
}

func convertPgURL(pgURL string) string {
	// postgres://user:pass@host:port/db?sslmode=disable
	// → user=user password=pass host=host port=port dbname=db sslmode=disable
	pgURL = strings.TrimPrefix(pgURL, "postgres://")

	var user, pass, host, port, dbname, params string

	// Extract user:pass@
	if idx := strings.LastIndex(pgURL, "@"); idx != -1 {
		userpass := pgURL[:idx]
		pgURL = pgURL[idx+1:]
		if idx2 := strings.Index(userpass, ":"); idx2 != -1 {
			user = userpass[:idx2]
			pass = userpass[idx2+1:]
		} else {
			user = userpass
		}
	}

	// Extract host:port
	var hostport string
	if idx := strings.Index(pgURL, "/"); idx != -1 {
		hostport = pgURL[:idx]
		pgURL = pgURL[idx+1:]
	} else if idx := strings.Index(pgURL, "?"); idx != -1 {
		hostport = pgURL[:idx]
		pgURL = pgURL[idx:]
	} else {
		hostport = pgURL
		pgURL = ""
	}

	if idx := strings.Index(hostport, ":"); idx != -1 {
		host = hostport[:idx]
		port = hostport[idx+1:]
	} else {
		host = hostport
		port = "6432"
	}

	// Extract dbname and params
	if idx := strings.Index(pgURL, "?"); idx != -1 {
		dbname = pgURL[:idx]
		params = pgURL[idx+1:]
	} else {
		dbname = pgURL
	}

	// Build keyword/value string
	result := ""
	if user != "" {
		result += "user=" + user + " "
	}
	if pass != "" {
		result += "password=" + pass + " "
	}
	result += "host=" + host + " port=" + port + " dbname=" + dbname
	if params != "" {
		result += " " + params
	}

	return result
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// serveIndexFallback serves files from publicFS with fallback to index.html for SPA routing
func serveIndexFallback(publicFiles fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(publicFiles))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if file exists
		_, err := fs.Stat(publicFiles, path)
		if err == nil {
			// File exists, serve it normally
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found, check if it looks like an API route or asset
		// If it's not an asset extension and not a known API path, serve index.html for SPA
		if !strings.Contains(path, ".") || strings.HasSuffix(path, ".html") {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// Asset not found (e.g., missing .js, .css)
		http.Error(w, "Not Found", http.StatusNotFound)
	})
}
