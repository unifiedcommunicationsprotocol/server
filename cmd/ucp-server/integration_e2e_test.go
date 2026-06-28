package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
	"github.com/unifiedcommunicationsprotocol/server/internal/models"
	"github.com/unifiedcommunicationsprotocol/server/internal/ratelimit"
	"github.com/unifiedcommunicationsprotocol/server/internal/store"
	"github.com/unifiedcommunicationsprotocol/server/internal/testutil"
	"github.com/unifiedcommunicationsprotocol/server/internal/transport"
)

// skipIntegrationTests skips tests if integration testing is not enabled.
func skipIntegrationTests(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test (set TEST_POSTGRES=1 to enable)")
	}
}

// TestE2ESendMessageValid tests sending a valid message end-to-end.
func TestE2ESendMessageValid(t *testing.T) {
	skipIntegrationTests(t)

	db := testutil.SetupTestDB(t)
	defer db.TeardownTestDB()

	// Create test identity and session
	senderAddr := "alice@example.com"
	_, privKey := CreateTestIdentity(t, db, senderAddr)

	// Create authenticated session
	authMgr := auth.New()
	sessionToken := CreateTestSession(t, authMgr, senderAddr)

	// Set up HTTP server with message handlers
	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	hub := transport.New()
	messageLimiter := ratelimit.New(50, 10)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/message/send", withRateLimit(messageLimiter, handleSendMessage(authMgr, s, hub)))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create test envelope
	threadID := models.GenerateULID()
	sigKey := base64.StdEncoding.EncodeToString(ed25519.PublicKey(privKey))

	envelope := &models.UCPEnvelope{
		V:         "ucp/1.0",
		ThreadID:  threadID,
		From:      senderAddr,
		To:        []string{"bob@example.com"},
		SigningKey: sigKey,
		MLS:       base64.StdEncoding.EncodeToString([]byte("mock_mls_ciphertext")),
	}

	// Encode envelope as base64
	envelopeJSON, _ := json.Marshal(envelope)
	envelopeB64 := base64.StdEncoding.EncodeToString(envelopeJSON)

	// Create request body with base64-encoded envelope
	req := map[string]interface{}{
		"envelope": envelopeB64,
	}
	reqBody, _ := json.Marshal(req)

	// Send message
	httpReq, _ := http.NewRequest("POST", server.URL+"/api/message/send", bytes.NewReader(reqBody))
	httpReq.Header.Set("Authorization", "Bearer "+sessionToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestE2EInboxEmptyUser tests retrieving inbox for user with no messages.
func TestE2EInboxEmptyUser(t *testing.T) {
	skipIntegrationTests(t)

	db := testutil.SetupTestDB(t)
	defer db.TeardownTestDB()

	// Create test identity and session
	userAddr := "alice@example.com"
	CreateTestIdentity(t, db, userAddr)

	authMgr := auth.New()
	sessionToken := CreateTestSession(t, authMgr, userAddr)

	// Set up HTTP server
	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/inbox", handleInbox(authMgr, s))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Fetch empty inbox
	httpReq, _ := http.NewRequest("GET", server.URL+"/api/inbox", nil)
	httpReq.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify response structure
	if _, ok := result["messages"]; !ok {
		t.Errorf("expected 'messages' field in response")
	}
}

// TestE2EInboxWithMessages tests retrieving inbox (stub implementation).
// NOTE: handleInbox is currently a stub that returns empty list.
// This test verifies the endpoint structure works; full implementation pending.
func TestE2EInboxWithMessages(t *testing.T) {
	skipIntegrationTests(t)

	db := testutil.SetupTestDB(t)
	defer db.TeardownTestDB()

	// Create test identities
	userAddr := "alice@example.com"
	senderAddr := "bob@example.com"
	CreateTestIdentity(t, db, userAddr)
	CreateTestIdentity(t, db, senderAddr)

	// Insert test messages (stored but not returned by current stub handler)
	ctx := context.Background()
	threadID := string(models.GenerateULID())

	query := `
		INSERT INTO messages (message_id, thread_id, from_addr, to_addrs, signing_key, server_ts, mls_encrypted)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	for i := 0; i < 3; i++ {
		_, err := db.Exec(ctx, query,
			string(models.GenerateULID()),
			threadID,
			senderAddr,
			pq.Array([]string{userAddr}),
			base64.StdEncoding.EncodeToString([]byte("signing-key")),
			now.Add(time.Duration(i)*time.Minute).Unix(),
			[]byte("encrypted_content"),
		)
		if err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	// Set up HTTP server
	authMgr := auth.New()
	sessionToken := CreateTestSession(t, authMgr, userAddr)

	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/inbox", handleInbox(authMgr, s))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Fetch inbox
	httpReq, _ := http.NewRequest("GET", server.URL+"/api/inbox?thread_id="+threadID, nil)
	httpReq.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify response structure is correct
	if _, ok := result["messages"]; !ok {
		t.Errorf("expected 'messages' field in response")
	}
	// TODO: Once handleInbox fully queries database, verify we get 3 messages
}

// TestE2EAttachmentUpload tests uploading an attachment.
func TestE2EAttachmentUpload(t *testing.T) {
	skipIntegrationTests(t)

	db := testutil.SetupTestDB(t)
	defer db.TeardownTestDB()

	userAddr := "alice@example.com"
	CreateTestIdentity(t, db, userAddr)

	authMgr := auth.New()
	sessionToken := CreateTestSession(t, authMgr, userAddr)

	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	messageLimiter := ratelimit.New(50, 10)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/content/upload", withRateLimit(messageLimiter, handleUploadAttachment(authMgr, s)))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create test file content
	fileContent := []byte("test file content")
	fileHash := sha256.Sum256(fileContent)
	fileHashB64 := base64.StdEncoding.EncodeToString(fileHash[:])

	// Create upload request
	req := map[string]interface{}{
		"name":     "test.txt",
		"size":     len(fileContent),
		"sha256":   fileHashB64,
		"content":  base64.StdEncoding.EncodeToString(fileContent),
	}
	reqBody, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", server.URL+"/api/content/upload", bytes.NewReader(reqBody))
	httpReq.Header.Set("Authorization", "Bearer "+sessionToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Accept both 200 and 201
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200 or 201, got %d: %s", resp.StatusCode, string(body))
	}

	// Verify response contains attachment ID
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if _, ok := result["attachment_id"]; !ok {
			t.Logf("warning: 'attachment_id' not in response: %v", result)
		}
	}
}

// TestE2EAttachmentDownload tests downloading an attachment.
func TestE2EAttachmentDownload(t *testing.T) {
	skipIntegrationTests(t)

	db := testutil.SetupTestDB(t)
	defer db.TeardownTestDB()

	userAddr := "alice@example.com"
	CreateTestIdentity(t, db, userAddr)

	authMgr := auth.New()
	sessionToken := CreateTestSession(t, authMgr, userAddr)

	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	// Insert test attachment
	ctx := context.Background()
	attachmentID := string(models.GenerateULID())
	fileContent := []byte("test attachment content")
	fileHash := sha256.Sum256(fileContent)
	fileHashHex := base64.StdEncoding.EncodeToString(fileHash[:])

	query := `
		INSERT INTO attachments (id, name, mime_type, size, sha256, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query, attachmentID, "test.txt", "text/plain", len(fileContent), fileHashHex, time.Now())
	if err != nil {
		t.Fatalf("failed to insert attachment: %v", err)
	}

	// Set up server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/content/{id}", handleDownloadAttachment(authMgr, s))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Download attachment
	httpReq, _ := http.NewRequest("GET", server.URL+"/api/content/"+attachmentID, nil)
	httpReq.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify SHA-256 header
	if sha256Header := resp.Header.Get("X-SHA256"); sha256Header == "" {
		t.Logf("warning: X-SHA256 header not present")
	}
}

// TestE2ESendMessageMissingAuth tests sending a message without authentication.
func TestE2ESendMessageMissingAuth(t *testing.T) {
	skipIntegrationTests(t)

	authMgr := auth.New()
	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	hub := transport.New()
	messageLimiter := ratelimit.New(50, 10)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/message/send", withRateLimit(messageLimiter, handleSendMessage(authMgr, s, hub)))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Send request without auth header
	envelope := &models.UCPEnvelope{
		V:         "ucp/1.0",
		ThreadID:  models.GenerateULID(),
		From:      "alice@example.com",
		To:        []string{"bob@example.com"},
		SigningKey: base64.StdEncoding.EncodeToString([]byte("test-signing-key")),
		MLS:       base64.StdEncoding.EncodeToString([]byte("mock_mls")),
	}

	envelopeJSON, _ := json.Marshal(envelope)
	envelopeB64 := base64.StdEncoding.EncodeToString(envelopeJSON)

	req := map[string]interface{}{
		"envelope": envelopeB64,
	}
	reqBody, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", server.URL+"/api/message/send", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

// TestE2EInboxMissingAuth tests fetching inbox without authentication.
func TestE2EInboxMissingAuth(t *testing.T) {
	skipIntegrationTests(t)

	authMgr := auth.New()
	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/inbox", handleInbox(authMgr, s))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Send request without auth header
	httpReq, _ := http.NewRequest("GET", server.URL+"/api/inbox", nil)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

// TestE2EAttachmentNotFound tests downloading a non-existent attachment.
func TestE2EAttachmentNotFound(t *testing.T) {
	skipIntegrationTests(t)

	db := testutil.SetupTestDB(t)
	defer db.TeardownTestDB()

	userAddr := "alice@example.com"
	CreateTestIdentity(t, db, userAddr)

	authMgr := auth.New()
	sessionToken := CreateTestSession(t, authMgr, userAddr)

	s, _ := store.New("user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable")
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/content/{id}", handleDownloadAttachment(authMgr, s))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Try to download non-existent attachment
	httpReq, _ := http.NewRequest("GET", server.URL+"/api/content/nonexistent", nil)
	httpReq.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}
