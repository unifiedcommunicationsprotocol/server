package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"
)

func TestGenerateChallenge(t *testing.T) {
	challenge, err := GenerateChallenge()
	if err != nil {
		t.Fatalf("GenerateChallenge error: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("Challenge length: got %d, want 32", len(challenge))
	}

	// Should be different each time
	challenge2, _ := GenerateChallenge()
	if string(challenge) == string(challenge2) {
		t.Error("Challenges should be unique")
	}
}

func TestVerifyChallengeResponse(t *testing.T) {
	// Generate a keypair
	pub, priv, _ := ed25519.GenerateKey(nil)

	challenge, _ := GenerateChallenge()

	// Sign the challenge
	sig := ed25519.Sign(priv, challenge)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	// Verify
	if err := VerifyChallengeResponse(challenge, pub, sigB64); err != nil {
		t.Fatalf("VerifyChallengeResponse error: %v", err)
	}

	// Wrong challenge should fail
	challenge2, _ := GenerateChallenge()
	if err := VerifyChallengeResponse(challenge2, pub, sigB64); err == nil {
		t.Error("VerifyChallengeResponse should fail with wrong challenge")
	}
}

func TestCreateSession(t *testing.T) {
	m := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := m.CreateSession(ctx, "alice@example.com", 3600)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	if session.Address != "alice@example.com" {
		t.Errorf("Address: got %q, want %q", session.Address, "alice@example.com")
	}

	if session.Token == "" {
		t.Error("Token is empty")
	}

	if session.ExpiresAt <= time.Now().Unix() {
		t.Error("ExpiresAt should be in the future")
	}

	if session.RevokedAt != nil {
		t.Error("RevokedAt should be nil for new session")
	}
}

func TestValidateSession(t *testing.T) {
	m := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, _ := m.CreateSession(ctx, "alice@example.com", 3600)

	// Valid session
	address, err := m.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Fatalf("ValidateSession error: %v", err)
	}

	if address != "alice@example.com" {
		t.Errorf("Address: got %q, want %q", address, "alice@example.com")
	}

	// Nonexistent token
	_, err = m.ValidateSession(ctx, "fake_token")
	if err == nil {
		t.Error("ValidateSession should fail for nonexistent token")
	}
}

func TestRevokeSession(t *testing.T) {
	m := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, _ := m.CreateSession(ctx, "alice@example.com", 3600)

	// Should be valid initially
	_, err := m.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Fatal("Valid session should validate")
	}

	// Revoke
	_ = m.RevokeSession(ctx, session.Token)

	// Should be invalid now
	_, err = m.ValidateSession(ctx, session.Token)
	if err == nil {
		t.Error("Revoked session should not validate")
	}
}

func TestSessionExpiry(t *testing.T) {
	m := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create with -1 second lifetime (already expired)
	session, _ := m.CreateSession(ctx, "alice@example.com", -1)

	// Should be expired
	_, err := m.ValidateSession(ctx, session.Token)
	if err == nil {
		t.Error("Expired session should not validate")
	}
}

func TestRefreshSession(t *testing.T) {
	m := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oldSession, _ := m.CreateSession(ctx, "alice@example.com", 3600)

	// Refresh
	newSession, err := m.RefreshSession(ctx, oldSession.Token, 7200)
	if err != nil {
		t.Fatalf("RefreshSession error: %v", err)
	}

	if newSession.Token == oldSession.Token {
		t.Error("New session should have different token")
	}

	if newSession.Address != oldSession.Address {
		t.Error("Address should be same")
	}

	// New session should be valid
	address, err := m.ValidateSession(ctx, newSession.Token)
	if err != nil {
		t.Fatalf("New session should validate: %v", err)
	}

	if address != "alice@example.com" {
		t.Error("Address mismatch")
	}
}

func TestSessionLifetimeCap(t *testing.T) {
	m := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Request 48-hour session
	session, _ := m.CreateSession(ctx, "alice@example.com", 48*3600)

	// Should be capped at 24 hours
	maxExpiry := time.Now().Unix() + 24*3600
	if session.ExpiresAt > maxExpiry {
		t.Errorf("Session lifetime exceeds 24-hour cap: %d > %d", session.ExpiresAt, maxExpiry)
	}
}

func TestChallengeStore(t *testing.T) {
	cs := NewChallengeStore()

	// Issue challenge
	challenge, err := cs.IssueChallenge("alice@example.com")
	if err != nil {
		t.Fatalf("IssueChallenge error: %v", err)
	}

	if len(challenge) != 32 {
		t.Errorf("Challenge length: got %d, want 32", len(challenge))
	}

	// Validate challenge
	retrieved, err := cs.ValidateChallenge("alice@example.com")
	if err != nil {
		t.Fatalf("ValidateChallenge error: %v", err)
	}

	if string(retrieved) != string(challenge) {
		t.Error("Retrieved challenge mismatch")
	}

	// Consume challenge
	if err := cs.ConsumeChallenge("alice@example.com"); err != nil {
		t.Fatalf("ConsumeChallenge error: %v", err)
	}

	// Should not be retrievable after consumption
	_, err = cs.ValidateChallenge("alice@example.com")
	if err == nil {
		t.Error("Consumed challenge should not validate")
	}
}

func TestChallengeNotFound(t *testing.T) {
	cs := NewChallengeStore()

	_, err := cs.ValidateChallenge("nonexistent@example.com")
	if err == nil {
		t.Error("ValidateChallenge should fail for nonexistent challenge")
	}
}
