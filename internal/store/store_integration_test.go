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
