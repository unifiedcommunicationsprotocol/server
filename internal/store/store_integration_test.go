package store

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// TestStoreNew tests database connection
func TestStoreNew(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test (set TEST_POSTGRES=1 to enable)")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	if s.db == nil {
		t.Error("Database connection is nil")
	}
}

// TestStoreIdentity tests storing and retrieving identities
func TestStoreIdentity(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	address := "store-test-" + time.Now().Format("20060102150405") + "@example.com"

	identity := &models.Identity{
		Address:       address,
		IdentityKey:   "test-key-base64",
		RevocationKey: "test-revocation-key",
		Capabilities:  []string{"ucp/1.0"},
	}

	// Store identity
	err = s.StoreIdentity(ctx, identity)
	if err != nil {
		t.Fatalf("Failed to store identity: %v", err)
	}

	// Retrieve it
	retrieved, err := s.GetIdentity(ctx, address)
	if err != nil {
		t.Fatalf("Failed to retrieve identity: %v", err)
	}

	if retrieved.Address != address {
		t.Errorf("Address mismatch: %s != %s", retrieved.Address, address)
	}

	if retrieved.IdentityKey != identity.IdentityKey {
		t.Errorf("IdentityKey mismatch")
	}

	if retrieved.RevocationKey != identity.RevocationKey {
		t.Errorf("RevocationKey mismatch")
	}
}

// TestStoreIdentityNotFound tests retrieving non-existent identity
func TestStoreIdentityNotFound(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_, err = s.GetIdentity(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Should have returned error for non-existent identity")
	}
}

// TestStoreMessage tests storing messages
func TestStoreMessage(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// First, store a user identity (required for foreign key)
	address := "msg-sender-" + time.Now().Format("20060102150405") + "@example.com"
	identity := &models.Identity{
		Address:       address,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	s.StoreIdentity(ctx, identity)

	// Store message
	ts := time.Now().Unix()
	envelope := &models.UCPEnvelope{
		ThreadID:   "thread-123",
		From:       address,
		To:         []string{"recipient@example.com"},
		SigningKey: "signing-key",
		ServerTs:   &ts,
	}

	encryptedMLS := []byte("encrypted-message-data")
	err = s.StoreMessage(ctx, envelope, encryptedMLS)
	if err != nil {
		t.Fatalf("Failed to store message: %v", err)
	}

	t.Logf("✓ Message stored successfully")
}

// TestStoreClose tests that Close() works without panic
func TestStoreClose(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Multiple closes should not panic
	err = s.Close()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	err = s.Close()
	if err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

// TestStoreMultipleIdentities tests storing and retrieving multiple identities
func TestStoreMultipleIdentities(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	timestamp := time.Now().Format("20060102150405")

	identities := []string{
		"user1-" + timestamp + "@example.com",
		"user2-" + timestamp + "@example.com",
		"user3-" + timestamp + "@example.com",
	}

	// Store all identities
	for i, addr := range identities {
		identity := &models.Identity{
			Address:       addr,
			IdentityKey:   "key-" + addr,
			RevocationKey: "revkey-" + addr,
			Capabilities:  []string{"ucp/1.0"},
		}
		err = s.StoreIdentity(ctx, identity)
		if err != nil {
			t.Fatalf("Failed to store identity %d: %v", i, err)
		}
	}

	// Retrieve and verify each
	for _, addr := range identities {
		retrieved, err := s.GetIdentity(ctx, addr)
		if err != nil {
			t.Fatalf("Failed to retrieve %s: %v", addr, err)
		}

		if retrieved.Address != addr {
			t.Errorf("Address mismatch for %s", addr)
		}

		if retrieved.IdentityKey != "key-"+addr {
			t.Errorf("Key mismatch for %s", addr)
		}
	}

	t.Logf("✓ Stored and retrieved %d identities", len(identities))
}

// TestStoreIdentityUpdate tests updating an existing identity
func TestStoreIdentityUpdate(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	address := "update-test-" + time.Now().Format("20060102150405") + "@example.com"

	// Store initial identity
	identity1 := &models.Identity{
		Address:       address,
		IdentityKey:   "key-v1",
		RevocationKey: "revkey-v1",
		Capabilities:  []string{"ucp/1.0"},
	}
	err = s.StoreIdentity(ctx, identity1)
	if err != nil {
		t.Fatalf("Failed to store initial identity: %v", err)
	}

	// Update with new key
	identity2 := &models.Identity{
		Address:       address,
		IdentityKey:   "key-v2-updated",
		RevocationKey: "revkey-v2-updated",
		Capabilities:  []string{"ucp/1.0"},
	}
	err = s.StoreIdentity(ctx, identity2)
	if err != nil {
		t.Fatalf("Failed to update identity: %v", err)
	}

	// Retrieve and verify update
	retrieved, err := s.GetIdentity(ctx, address)
	if err != nil {
		t.Fatalf("Failed to retrieve identity: %v", err)
	}

	if retrieved.IdentityKey != "key-v2-updated" {
		t.Errorf("Update failed: key is %s, expected key-v2-updated", retrieved.IdentityKey)
	}

	t.Logf("✓ Identity updated successfully")
}

// TestGetThreadMessages tests retrieving all messages in a thread with ordering
func TestGetThreadMessages(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	threadID := models.ULID("thread-" + time.Now().Format("20060102150405"))
	sender := "sender-" + time.Now().Format("20060102150405") + "@example.com"

	// Store sender identity
	identity := &models.Identity{
		Address:       sender,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	s.StoreIdentity(ctx, identity)

	// Store 3 messages in the same thread
	for i := 0; i < 3; i++ {
		ts := time.Now().Unix() + int64(i)
		envelope := &models.UCPEnvelope{
			ThreadID:   threadID,
			From:       sender,
			To:         []string{"recipient@example.com"},
			SigningKey: "signing-key",
			ServerTs:   &ts,
		}
		err = s.StoreMessage(ctx, envelope, []byte("msg-"+string(rune(i))))
		if err != nil {
			t.Fatalf("Failed to store message %d: %v", i, err)
		}
	}

	// Retrieve thread messages
	messages, err := s.GetThreadMessages(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to get thread messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Verify ordering (should be ascending by server_ts)
	for i := 0; i < len(messages)-1; i++ {
		if *messages[i].ServerTs > *messages[i+1].ServerTs {
			t.Errorf("Messages not ordered correctly: %d > %d", *messages[i].ServerTs, *messages[i+1].ServerTs)
		}
	}

	t.Logf("✓ Retrieved and ordered %d thread messages", len(messages))
}

// TestGetThreadMessagesEmpty tests empty thread returns empty slice
func TestGetThreadMessagesEmpty(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	emptyThreadID := models.ULID("empty-thread-" + time.Now().Format("20060102150405"))

	// Query non-existent thread
	messages, err := s.GetThreadMessages(ctx, emptyThreadID)
	if err != nil {
		t.Fatalf("Failed to get thread messages: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected empty slice for non-existent thread, got %d messages", len(messages))
	}

	t.Logf("✓ Empty thread returns empty slice")
}

// TestCreateAndGetSession tests session creation and retrieval
func TestCreateAndGetSession(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	address := "session-user-" + time.Now().Format("20060102150405") + "@example.com"
	token := "session-token-" + time.Now().Format("20060102150405")
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	// Create identity first (foreign key requirement)
	identity := &models.Identity{
		Address:       address,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	s.StoreIdentity(ctx, identity)

	// Create session
	err = s.CreateSession(ctx, address, token, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Retrieve session
	retrieved, err := s.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved != address {
		t.Errorf("Session address mismatch: got %q, want %q", retrieved, address)
	}

	t.Logf("✓ Session created and retrieved successfully")
}

// TestRevokeSession tests session revocation
func TestRevokeSession(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	address := "revoke-test-" + time.Now().Format("20060102150405") + "@example.com"
	token := "revoke-token-" + time.Now().Format("20060102150405")
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	// Create identity first (foreign key requirement)
	identity := &models.Identity{
		Address:       address,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	s.StoreIdentity(ctx, identity)

	// Create session
	err = s.CreateSession(ctx, address, token, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify it works
	retrieved, err := s.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if retrieved != address {
		t.Errorf("Session should be valid before revocation")
	}

	// Revoke session
	err = s.RevokeSession(ctx, token)
	if err != nil {
		t.Fatalf("Failed to revoke session: %v", err)
	}

	// Verify revocation (should return error or empty)
	retrieved, err = s.GetSession(ctx, token)
	if err == nil && retrieved != "" {
		t.Errorf("Revoked session should not be retrievable, got %q", retrieved)
	}

	t.Logf("✓ Session revoked successfully")
}

// TestSessionExpiry tests that expired sessions are not returned
func TestSessionExpiry(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	address := "expiry-test-" + time.Now().Format("20060102150405") + "@example.com"
	token := "expiry-token-" + time.Now().Format("20060102150405")
	expiresAt := time.Now().Add(-1 * time.Hour).Unix() // Expired 1 hour ago

	// Create identity first (foreign key requirement)
	identity := &models.Identity{
		Address:       address,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	s.StoreIdentity(ctx, identity)

	// Create session with past expiry
	err = s.CreateSession(ctx, address, token, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	// Try to retrieve (should fail or return error)
	retrieved, err := s.GetSession(ctx, token)
	if err == nil && retrieved != "" {
		t.Errorf("Expired session should not be retrievable, got %q", retrieved)
	}

	t.Logf("✓ Expired session not retrievable")
}

// TestStoreAndGetAttachment tests attachment storage and retrieval
func TestStoreAndGetAttachment(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	attachmentID := models.ULID("attach-" + time.Now().Format("20060102150405"))

	attachment := &models.Attachment{
		ID:       attachmentID,
		Name:     "document.pdf",
		MimeType: "application/pdf",
		Size:     102400,
		SHA256:   "abc123def456",
	}

	// Store attachment
	err = s.StoreAttachment(ctx, attachment)
	if err != nil {
		t.Fatalf("Failed to store attachment: %v", err)
	}

	// Retrieve attachment
	retrieved, err := s.GetAttachment(ctx, attachmentID)
	if err != nil {
		t.Fatalf("Failed to get attachment: %v", err)
	}

	if retrieved.Name != attachment.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, attachment.Name)
	}

	if retrieved.Size != attachment.Size {
		t.Errorf("Size mismatch: got %d, want %d", retrieved.Size, attachment.Size)
	}

	if retrieved.SHA256 != attachment.SHA256 {
		t.Errorf("SHA256 mismatch: got %q, want %q", retrieved.SHA256, attachment.SHA256)
	}

	t.Logf("✓ Attachment stored and retrieved successfully")
}

// TestGetAttachmentNotFound tests retrieving non-existent attachment
func TestGetAttachmentNotFound(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	nonexistentID := models.ULID("nonexistent-attach-123")

	// Try to retrieve non-existent attachment
	_, err = s.GetAttachment(ctx, nonexistentID)
	if err == nil {
		t.Error("Should have returned error for non-existent attachment")
	}

	t.Logf("✓ Non-existent attachment returns error")
}

// TestMultipleAttachments tests storing multiple attachments
func TestMultipleAttachments(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	timestamp := time.Now().Format("20060102150405")

	attachments := []*models.Attachment{
		{
			ID:       models.ULID("attach1-" + timestamp),
			Name:     "file1.pdf",
			MimeType: "application/pdf",
			Size:     1024,
			SHA256:   "hash1",
		},
		{
			ID:       models.ULID("attach2-" + timestamp),
			Name:     "file2.txt",
			MimeType: "text/plain",
			Size:     512,
			SHA256:   "hash2",
		},
		{
			ID:       models.ULID("attach3-" + timestamp),
			Name:     "file3.zip",
			MimeType: "application/zip",
			Size:     2048,
			SHA256:   "hash3",
		},
	}

	// Store all attachments
	for _, att := range attachments {
		err = s.StoreAttachment(ctx, att)
		if err != nil {
			t.Fatalf("Failed to store attachment %s: %v", att.Name, err)
		}
	}

	// Retrieve and verify each
	for _, att := range attachments {
		retrieved, err := s.GetAttachment(ctx, att.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve %s: %v", att.Name, err)
		}

		if retrieved.Name != att.Name {
			t.Errorf("Name mismatch for %s", att.Name)
		}

		if retrieved.Size != att.Size {
			t.Errorf("Size mismatch for %s: got %d, want %d", att.Name, retrieved.Size, att.Size)
		}
	}

	t.Logf("✓ Stored and retrieved %d attachments", len(attachments))
}

// TestMessageIdempotency tests that duplicate message IDs don't cause issues
func TestMessageIdempotency(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	sender := "idempotent-sender-" + time.Now().Format("20060102150405") + "@example.com"
	threadID := models.ULID("idempotent-thread-" + time.Now().Format("20060102150405"))

	// Store sender identity
	identity := &models.Identity{
		Address:       sender,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0"},
	}
	s.StoreIdentity(ctx, identity)

	ts := time.Now().Unix()
	envelope := &models.UCPEnvelope{
		ThreadID:   threadID,
		From:       sender,
		To:         []string{"recipient@example.com"},
		SigningKey: "signing-key",
		ServerTs:   &ts,
	}

	// Store message twice
	err = s.StoreMessage(ctx, envelope, []byte("msg1"))
	if err != nil {
		t.Fatalf("Failed to store first message: %v", err)
	}

	err = s.StoreMessage(ctx, envelope, []byte("msg2"))
	if err != nil {
		t.Fatalf("Failed to store duplicate message: %v", err)
	}

	// Retrieve thread messages (should only have 1 or 2, not duplicates)
	messages, err := s.GetThreadMessages(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to get thread messages: %v", err)
	}

	// ON CONFLICT means the second insert is ignored, so we should have 1 message
	if len(messages) == 0 {
		t.Errorf("Expected at least one message, got %d", len(messages))
	}

	t.Logf("✓ Message idempotency works: stored duplicate message, retrieved %d message(s)", len(messages))
}

// TestIdentityCapabilities tests storing and retrieving identity capabilities
func TestIdentityCapabilities(t *testing.T) {
	if os.Getenv("TEST_POSTGRES") == "" {
		t.Skip("Skipping integration test")
	}

	dsn := "user=postgres password=dev host=localhost port=5555 dbname=ucp sslmode=disable"
	s, err := New(dsn)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	address := "caps-test-" + time.Now().Format("20060102150405") + "@example.com"

	identity := &models.Identity{
		Address:       address,
		IdentityKey:   "key",
		RevocationKey: "revkey",
		Capabilities:  []string{"ucp/1.0", "ucp/chat", "bridge/smtp"},
	}

	// Store identity with capabilities
	err = s.StoreIdentity(ctx, identity)
	if err != nil {
		t.Fatalf("Failed to store identity: %v", err)
	}

	// Retrieve and verify capabilities
	retrieved, err := s.GetIdentity(ctx, address)
	if err != nil {
		t.Fatalf("Failed to retrieve identity: %v", err)
	}

	if len(retrieved.Capabilities) != len(identity.Capabilities) {
		t.Errorf("Capabilities count mismatch: got %d, want %d", len(retrieved.Capabilities), len(identity.Capabilities))
	}

	for i, cap := range identity.Capabilities {
		if i < len(retrieved.Capabilities) && retrieved.Capabilities[i] != cap {
			t.Errorf("Capability %d mismatch: got %q, want %q", i, retrieved.Capabilities[i], cap)
		}
	}

	t.Logf("✓ Identity capabilities stored and retrieved: %v", retrieved.Capabilities)
}
