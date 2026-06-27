// +build postgres

package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

func TestPostgresConnection(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	s, err := New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	// Connection should be valid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestStoreAndGetIdentity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	s, err := New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Store identity
	identity := &models.Identity{
		Address:       "alice@example.com",
		IdentityKey:   "base64_ed25519_pubkey",
		SigningKeys:   []models.SigningKey{},
		RevocationKey: "revocation_key_base64",
		Server:        "ucp.example.com",
		Capabilities:  []string{"ucp/1.0"},
	}

	if err := s.StoreIdentity(ctx, identity); err != nil {
		t.Fatalf("store identity: %v", err)
	}

	// Retrieve identity
	retrieved, err := s.GetIdentity(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("get identity: %v", err)
	}

	if retrieved.Address != "alice@example.com" {
		t.Errorf("address: got %q, want %q", retrieved.Address, "alice@example.com")
	}

	if retrieved.IdentityKey != "base64_ed25519_pubkey" {
		t.Errorf("identity_key mismatch")
	}
}

func TestStoreAndGetMessage(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	s, err := New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// First store the sender identity
	identity := &models.Identity{
		Address:       "alice@example.com",
		IdentityKey:   "key",
		SigningKeys:   []models.SigningKey{},
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	if err := s.StoreIdentity(ctx, identity); err != nil {
		t.Fatalf("store identity: %v", err)
	}

	// Store message
	now := time.Now().UnixMilli()
	envelope := &models.UCPEnvelope{
		V:        "ucp/1.0",
		Type:     "application",
		ThreadID: "thread_123",
		From:     "alice@example.com",
		To:       []string{"bob@example.com"},
		SigningKey: "alice_sig_key",
		ServerTs: &now,
		MLS:      "encrypted_content_base64",
	}

	encryptedMLS := []byte("encrypted_mls_bytes")
	if err := s.StoreMessage(ctx, envelope, encryptedMLS); err != nil {
		t.Fatalf("store message: %v", err)
	}

	// Retrieve messages in thread
	messages, err := s.GetThreadMessages(ctx, "thread_123")
	if err != nil {
		t.Fatalf("get thread messages: %v", err)
	}

	if len(messages) == 0 {
		t.Error("no messages retrieved")
	}

	if len(messages) > 0 {
		msg := messages[0]
		if msg.From != "alice@example.com" {
			t.Errorf("from: got %q, want %q", msg.From, "alice@example.com")
		}

		if msg.ThreadID != "thread_123" {
			t.Errorf("thread_id: got %q, want %q", msg.ThreadID, "thread_123")
		}
	}
}

func TestCreateAndValidateSession(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	s, err := New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// First store identity
	identity := &models.Identity{
		Address:       "alice@example.com",
		IdentityKey:   "key",
		SigningKeys:   []models.SigningKey{},
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	if err := s.StoreIdentity(ctx, identity); err != nil {
		t.Fatalf("store identity: %v", err)
	}

	// Create session
	token := "test_session_token_" + time.Now().Format("20060102150405")
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	if err := s.CreateSession(ctx, "alice@example.com", token, expiresAt); err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Get session
	address, err := s.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}

	if address != "alice@example.com" {
		t.Errorf("address: got %q, want %q", address, "alice@example.com")
	}
}

func TestStoreAndGetAttachment(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	s, err := New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Store attachment
	attachment := &models.Attachment{
		ID:       "attach_abc123",
		Name:     "document.pdf",
		MimeType: "application/pdf",
		Size:     1024000,
		SHA256:   "abc123def456",
	}

	if err := s.StoreAttachment(ctx, attachment); err != nil {
		t.Fatalf("store attachment: %v", err)
	}

	// Get attachment
	retrieved, err := s.GetAttachment(ctx, "attach_abc123")
	if err != nil {
		t.Fatalf("get attachment: %v", err)
	}

	if retrieved.Name != "document.pdf" {
		t.Errorf("name: got %q, want %q", retrieved.Name, "document.pdf")
	}

	if retrieved.Size != 1024000 {
		t.Errorf("size: got %d, want 1024000", retrieved.Size)
	}

	if retrieved.SHA256 != "abc123def456" {
		t.Errorf("sha256: got %q, want %q", retrieved.SHA256, "abc123def456")
	}
}
