package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
	"github.com/unifiedcommunicationsprotocol/server/internal/logging"
	"github.com/unifiedcommunicationsprotocol/server/internal/ratelimit"
	"github.com/unifiedcommunicationsprotocol/server/internal/store"
	"github.com/unifiedcommunicationsprotocol/server/internal/transport"
)

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

	// Initialize auth manager
	authMgr := auth.New()
	challengeStore := auth.NewChallengeStore()

	// Initialize rate limiters
	authLimiter := ratelimit.New(10, 5)     // 10 burst, 5/sec
	messageLimiter := ratelimit.New(50, 10) // 50 burst, 10/sec

	// Initialize transport hub
	hub := transport.New()

	// Create HTTP router
	mux := http.NewServeMux()

	// Register well-known endpoints
	mux.HandleFunc("GET /.well-known/ucp/server-key", handleServerKey(cfg))
	mux.HandleFunc("GET /.well-known/ucp/identity/{address}", handleIdentity(s))
	mux.HandleFunc("GET /.well-known/ucp/keypackages/{address}", handleKeyPackages(s))
	mux.HandleFunc("GET /.well-known/ucp/privacy", handlePrivacy())

	// Register auth endpoints (with rate limiting)
	mux.HandleFunc("POST /auth/challenge", withRateLimit(authLimiter, handleChallenge(challengeStore)))
	mux.HandleFunc("POST /auth/session", withRateLimit(authLimiter, handleSession(authMgr, challengeStore, s)))
	mux.HandleFunc("POST /auth/session/refresh", withRateLimit(authLimiter, handleRefresh(authMgr)))

	// Register API endpoints (with rate limiting)
	mux.HandleFunc("POST /api/message/send", withRateLimit(messageLimiter, handleSendMessage(authMgr, s, hub)))
	mux.HandleFunc("GET /api/inbox", handleInbox(authMgr, s))
	mux.HandleFunc("POST /api/content/upload", withRateLimit(messageLimiter, handleUploadAttachment(authMgr, s)))
	mux.HandleFunc("GET /api/content/{id}", handleDownloadAttachment(authMgr, s))

	// Register metrics endpoint
	mux.HandleFunc("GET /metrics", handleMetrics(metrics))

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
	return config{
		Listen:       getEnv("API_PORT", ":5150"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/ucp"),
		ServerDomain: getEnv("API_URL", "localhost:5150"),
		ServerKey:    getEnv("UCP_SERVER_KEY", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
