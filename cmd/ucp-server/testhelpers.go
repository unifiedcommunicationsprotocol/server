package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
	"github.com/unifiedcommunicationsprotocol/server/internal/models"
	"github.com/unifiedcommunicationsprotocol/server/internal/store"
	"github.com/unifiedcommunicationsprotocol/server/internal/testutil"
)

// TestServer sets up a test HTTP server with all dependencies.
type TestServer struct {
	Server         *httptest.Server
	Store          *store.Store
	AuthMgr        *auth.Manager
	ChallengeStore *auth.ChallengeStore
	DB             *testutil.TestDB
	T              *testing.T
}

// StartTestServer initializes a test server with isolated database.
func StartTestServer(t *testing.T) *TestServer {
	db := testutil.SetupTestDB(t)

	// Initialize managers
	authMgr := auth.New()
	challengeStore := auth.NewChallengeStore()

	// Create a store from the test database
	s := &store.Store{}
	// Inject database by directly setting through a helper or accessor
	// For now, we'll use the store package directly with the test DB

	// Create HTTP server with handlers
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/ucp/server-key", handleServerKey(config{
		ServerDomain: "localhost:5150",
		ServerKey:    "",
	}))

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
		db.TeardownTestDB()
	})

	return &TestServer{
		Server:         server,
		Store:          s,
		AuthMgr:        authMgr,
		ChallengeStore: challengeStore,
		DB:             db,
		T:              t,
	}
}

// CreateTestIdentity creates a test identity with Ed25519 keypair.
func CreateTestIdentity(t *testing.T, db *testutil.TestDB, address string) (*models.Identity, ed25519.PrivateKey) {
	// Generate Ed25519 keypair
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate Ed25519 keypair: %v", err)
	}

	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	identity := &models.Identity{
		Address:       address,
		IdentityKey:   pubKeyB64,
		RevocationKey: "test-revocation-key",
		Capabilities:  []string{"ucp/1.0"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Store identity in database using raw SQL since we don't have Store accessor yet
	query := `
		INSERT INTO identities (address, identity_key, revocation_key, capabilities, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (address) DO NOTHING
	`
	_, err = db.Exec(ctx, query, address, pubKeyB64, identity.RevocationKey, "{\"ucp/1.0\"}", time.Now())
	if err != nil {
		t.Fatalf("failed to store test identity: %v", err)
	}

	return identity, privKey
}

// CreateTestSession creates an authenticated session for a user.
func CreateTestSession(t *testing.T, authMgr *auth.Manager, address string) string {
	session, err := authMgr.CreateSession(address, 24*3600)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	return session.Token
}

// CreateTestEnvelope creates a UCPEnvelope for testing.
func CreateTestEnvelope(threadID, from string, to []string) *models.UCPEnvelope {
	now := time.Now().Unix()
	return &models.UCPEnvelope{
		V:        "ucp/1.0",
		ThreadID: models.ULID(threadID),
		From:     from,
		To:       to,
		ServerTs: &now,
	}
}

// SendHTTPRequest sends an HTTP request to the test server.
func SendHTTPRequest(t *testing.T, ts *TestServer, method, path string, headers map[string]string, body []byte) *http.Response {
	req, err := http.NewRequest(method, ts.Server.URL+path, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	return resp
}

// AssertEnvelopeExists verifies an envelope is stored in the database.
func AssertEnvelopeExists(t *testing.T, db *testutil.TestDB, threadID models.ULID) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "SELECT COUNT(*) FROM messages WHERE thread_id = $1"
	var count int
	err := db.QueryRow(ctx, query, string(threadID)).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query messages: %v", err)
	}

	if count == 0 {
		t.Errorf("expected envelope in thread %s, but found none", threadID)
	}
}

// AssertStatusCode verifies the HTTP response status code.
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	if resp.StatusCode != expected {
		t.Errorf("expected status %d, got %d", expected, resp.StatusCode)
	}
}
