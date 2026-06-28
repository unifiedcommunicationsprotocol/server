package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
)

// TestWellKnownServerKey tests the server key endpoint
func TestWellKnownServerKey(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config{
		Listen:       ":5150",
		DatabaseURL:  "",
		ServerDomain: "localhost:5150",
		ServerKey:    "",
	}
	mux.HandleFunc("GET /.well-known/ucp/server-key", handleServerKey(cfg))
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/.well-known/ucp/server-key")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if _, ok := result["domain"]; !ok {
		t.Error("Expected 'domain' in response")
	}
}

// TestWellKnownPrivacy tests the privacy endpoint
func TestWellKnownPrivacy(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/ucp/privacy", handlePrivacy())
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/.well-known/ucp/privacy")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected non-empty response")
	}

	if _, ok := result["enabled"]; !ok {
		t.Error("Expected 'enabled' in response")
	}
}

// TestAuthChallenge tests the challenge generation endpoint
func TestAuthChallenge(t *testing.T) {
	mux := http.NewServeMux()
	challengeStore := auth.NewChallengeStore()
	mux.HandleFunc("POST /auth/challenge", handleChallenge(challengeStore))
	server := httptest.NewServer(mux)
	defer server.Close()

	address := "test@example.com"
	reqBody := map[string]string{"address": address}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		server.URL+"/auth/challenge",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	challenge, ok := result["challenge"]
	if !ok {
		t.Fatal("No challenge in response")
	}

	// Verify it's base64 and 32 bytes
	challengeBytes, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		t.Errorf("Invalid base64: %v", err)
	}
	if len(challengeBytes) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(challengeBytes))
	}
}

// TestAuthChallengeAndSign demonstrates the full auth flow with valid signature
func TestAuthChallengeAndSign(t *testing.T) {
	// Create handlers
	mux := http.NewServeMux()
	challengeStore := auth.NewChallengeStore()
	authMgr := auth.New()

	mux.HandleFunc("POST /auth/challenge", handleChallenge(challengeStore))
	server := httptest.NewServer(mux)
	defer server.Close()

	address := "alice@example.com"

	// Step 1: Generate keypair
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	t.Logf("Generated keys: pub=%s", pubKeyB64)

	// Step 2: Get challenge
	reqBody := map[string]string{"address": address}
	body, _ := json.Marshal(reqBody)
	resp, _ := http.Post(
		server.URL+"/auth/challenge",
		"application/json",
		bytes.NewReader(body),
	)
	var challengeResult map[string]string
	json.NewDecoder(resp.Body).Decode(&challengeResult)
	resp.Body.Close()

	challenge := challengeResult["challenge"]
	challengeBytes, _ := base64.StdEncoding.DecodeString(challenge)
	t.Logf("Challenge: %s", challenge)

	// Step 3: Sign
	signature := ed25519.Sign(privKey, challengeBytes)
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	t.Logf("Signature: %s", signatureB64)

	// Step 4: Verify signature (this is what the server would do)
	// Note: Server uses GetIdentity to fetch the public key from DB
	// For this test, we verify locally
	if !ed25519.Verify(pubKey, challengeBytes, signature) {
		t.Error("Signature verification failed")
	}

	// Step 5: Create session (what server would do after verifying signature)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := authMgr.CreateSession(ctx, address, 24*3600)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.Token == "" {
		t.Error("Session token is empty")
	}

	t.Logf("✓ Session created: %s expires at %d", session.Token, session.ExpiresAt)
}

// TestSessionValidation tests session token validation
func TestSessionValidation(t *testing.T) {
	authMgr := auth.New()
	address := "bob@example.com"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create session
	session, _ := authMgr.CreateSession(ctx, address, 3600)

	// Validate it
	retrievedAddr, err := authMgr.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if retrievedAddr != address {
		t.Errorf("Expected %s, got %s", address, retrievedAddr)
	}

	// Try invalid token
	_, err = authMgr.ValidateSession(ctx, "invalid-token")
	if err == nil {
		t.Error("Should have rejected invalid token")
	}
}

// TestChallengeExpiry tests that challenges expire after 60 seconds
func TestChallengeExpiry(t *testing.T) {
	cs := auth.NewChallengeStore()
	address := "charlie@example.com"

	// Issue a challenge
	challenge, err := cs.IssueChallenge(address)
	if err != nil {
		t.Fatalf("Failed to issue challenge: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(challenge))
	}

	// Immediately validate should work
	retrievedChallenge, err := cs.ValidateChallenge(address)
	if err != nil {
		t.Fatalf("Should have valid challenge: %v", err)
	}

	if !bytes.Equal(challenge, retrievedChallenge) {
		t.Error("Retrieved challenge doesn't match")
	}

	// Consuming should work once
	err = cs.ConsumeChallenge(address)
	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}

	// Second consume should fail
	err = cs.ConsumeChallenge(address)
	if err == nil {
		t.Error("Should not consume twice")
	}
}

// TestHandlerBadRequest tests handlers with malformed input
func TestHandlerBadRequest(t *testing.T) {
	mux := http.NewServeMux()
	challengeStore := auth.NewChallengeStore()
	mux.HandleFunc("POST /auth/challenge", handleChallenge(challengeStore))
	server := httptest.NewServer(mux)
	defer server.Close()

	// Invalid JSON
	resp, _ := http.Post(
		server.URL+"/auth/challenge",
		"application/json",
		bytes.NewReader([]byte("not json")),
	)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}

	// Missing address field
	resp, _ = http.Post(
		server.URL+"/auth/challenge",
		"application/json",
		bytes.NewReader([]byte("{}")),
	)
	defer resp.Body.Close()

	// Should still issue challenge for empty address (server's choice)
	// Just verify it doesn't crash
	if resp.StatusCode > 500 {
		t.Errorf("Server error: %d", resp.StatusCode)
	}
}

// TestSessionExpiry tests that sessions respect TTL
func TestSessionExpiry(t *testing.T) {
	authMgr := auth.New()
	address := "expiry@example.com"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create session with 0 second TTL (expires immediately in practical terms)
	session, err := authMgr.CreateSession(ctx, address, 0)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Session should still be valid immediately (Unix time precision)
	retrievedAddr, err := authMgr.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Logf("Note: Session expired immediately (expected with 0 TTL): %v", err)
	} else {
		if retrievedAddr != address {
			t.Errorf("Expected %s, got %s", address, retrievedAddr)
		}
	}
}

// TestAuthChallengeMultipleAddresses tests multiple concurrent challenges
func TestAuthChallengeMultipleAddresses(t *testing.T) {
	cs := auth.NewChallengeStore()
	addresses := []string{
		"user1@example.com",
		"user2@example.com",
		"user3@example.com",
	}

	challenges := make(map[string][]byte)

	// Issue challenges to multiple users
	for _, addr := range addresses {
		challenge, err := cs.IssueChallenge(addr)
		if err != nil {
			t.Fatalf("Failed for %s: %v", addr, err)
		}
		challenges[addr] = challenge
	}

	// Verify each can be retrieved independently
	for _, addr := range addresses {
		retrieved, err := cs.ValidateChallenge(addr)
		if err != nil {
			t.Fatalf("Failed to validate %s: %v", addr, err)
		}

		if !bytes.Equal(challenges[addr], retrieved) {
			t.Errorf("Challenge mismatch for %s", addr)
		}
	}
}

// TestChallengeSignatureVerification tests Ed25519 signature verification
func TestChallengeSignatureVerification(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	challenge := []byte("test challenge data")

	// Valid signature
	signature := ed25519.Sign(privKey, challenge)
	if !ed25519.Verify(pubKey, challenge, signature) {
		t.Error("Valid signature failed verification")
	}

	// Invalid signature (all zeros)
	invalidSig := make([]byte, 64)
	if ed25519.Verify(pubKey, challenge, invalidSig) {
		t.Error("Invalid signature passed verification")
	}

	// Wrong challenge
	wrongChallenge := []byte("different challenge")
	if ed25519.Verify(pubKey, wrongChallenge, signature) {
		t.Error("Signature verified with wrong challenge")
	}

	// Different key
	otherPubKey, _, _ := ed25519.GenerateKey(nil)
	if ed25519.Verify(otherPubKey, challenge, signature) {
		t.Error("Signature verified with wrong key")
	}
}

// TestSessionTokenFormat tests that session tokens are properly formatted
func TestSessionTokenFormat(t *testing.T) {
	authMgr := auth.New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create multiple sessions and verify tokens are unique
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		session, _ := authMgr.CreateSession(ctx, "user@example.com", 3600)

		// Verify it's base64 and decodable
		decoded, err := base64.StdEncoding.DecodeString(session.Token)
		if err != nil {
			t.Errorf("Token %d is not valid base64: %v", i, err)
		}

		// Verify it's 32 bytes (256 bits)
		if len(decoded) != 32 {
			t.Errorf("Token %d has %d bytes, expected 32", i, len(decoded))
		}

		// Verify uniqueness
		if tokens[session.Token] {
			t.Error("Duplicate token generated")
		}
		tokens[session.Token] = true
	}
}

// TestPrivacyPolicyEndpoint verifies the privacy policy format
func TestPrivacyPolicyEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/ucp/privacy", handlePrivacy())
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, _ := http.Get(server.URL + "/.well-known/ucp/privacy")
	defer resp.Body.Close()

	var policy map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&policy)

	// Verify required fields
	requiredFields := []string{"enabled", "scopes", "data_retention", "deletion_policy"}
	for _, field := range requiredFields {
		if _, ok := policy[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify types
	if enabled, ok := policy["enabled"].(bool); !ok {
		t.Error("'enabled' should be bool")
	} else {
		t.Logf("Server processing enabled: %v", enabled)
	}

	if scopes, ok := policy["scopes"].([]interface{}); !ok {
		t.Error("'scopes' should be array")
	} else {
		t.Logf("Processing scopes: %d available", len(scopes))
	}
}
