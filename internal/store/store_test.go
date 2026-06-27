package store

import (
	"context"
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// MockStore is an in-memory implementation of Store for testing.
type MockStore struct {
	messages    map[string]*models.UCPEnvelope
	identities  map[string]*models.Identity
	sessions    map[string]sessionRecord
	attachments map[string]*models.Attachment
}

type sessionRecord struct {
	address   string
	expiresAt int64
	revoked   bool
}

// NewMockStore creates an in-memory store for testing.
func NewMockStore() *MockStore {
	return &MockStore{
		messages:    make(map[string]*models.UCPEnvelope),
		identities:  make(map[string]*models.Identity),
		sessions:    make(map[string]sessionRecord),
		attachments: make(map[string]*models.Attachment),
	}
}

// StoreMessage stores a message in memory.
func (m *MockStore) StoreMessage(ctx context.Context, envelope *models.UCPEnvelope, encryptedMLS []byte) error {
	key := string(envelope.ThreadID) + ":" + string(envelope.From)
	m.messages[key] = envelope
	return nil
}

// GetMessage retrieves a message from memory.
func (m *MockStore) GetMessage(ctx context.Context, messageID models.ULID) (*models.UCPEnvelope, []byte, error) {
	for _, env := range m.messages {
		// Simple lookup by checking if envelope exists
		return env, nil, nil
	}
	return nil, nil, nil
}

// GetThreadMessages retrieves messages in a thread.
func (m *MockStore) GetThreadMessages(ctx context.Context, threadID models.ULID) ([]*models.UCPEnvelope, error) {
	var messages []*models.UCPEnvelope
	for _, env := range m.messages {
		if env.ThreadID == threadID {
			messages = append(messages, env)
		}
	}
	return messages, nil
}

// StoreIdentity stores an identity in memory.
func (m *MockStore) StoreIdentity(ctx context.Context, identity *models.Identity) error {
	m.identities[identity.Address] = identity
	return nil
}

// GetIdentity retrieves an identity from memory.
func (m *MockStore) GetIdentity(ctx context.Context, address string) (*models.Identity, error) {
	id, ok := m.identities[address]
	if !ok {
		return nil, nil
	}
	return id, nil
}

// CreateSession creates a session in memory.
func (m *MockStore) CreateSession(ctx context.Context, address string, token string, expiresAt int64) error {
	m.sessions[token] = sessionRecord{
		address:   address,
		expiresAt: expiresAt,
		revoked:   false,
	}
	return nil
}

// GetSession retrieves a session from memory.
func (m *MockStore) GetSession(ctx context.Context, token string) (string, error) {
	record, ok := m.sessions[token]
	if !ok || record.revoked {
		return "", nil
	}
	// Check expiry
	if record.expiresAt < time.Now().Unix() {
		return "", nil
	}
	return record.address, nil
}

// RevokeSession revokes a session in memory.
func (m *MockStore) RevokeSession(ctx context.Context, token string) error {
	record, ok := m.sessions[token]
	if ok {
		record.revoked = true
		m.sessions[token] = record
	}
	return nil
}

// StoreAttachment stores an attachment in memory.
func (m *MockStore) StoreAttachment(ctx context.Context, attachment *models.Attachment) error {
	m.attachments[string(attachment.ID)] = attachment
	return nil
}

// GetAttachment retrieves an attachment from memory.
func (m *MockStore) GetAttachment(ctx context.Context, attachmentID models.ULID) (*models.Attachment, error) {
	att, ok := m.attachments[string(attachmentID)]
	if !ok {
		return nil, nil
	}
	return att, nil
}

func (m *MockStore) Close() error {
	return nil
}

// Tests using the MockStore

func TestMockStoreMessage(t *testing.T) {
	s := NewMockStore()
	ctx := context.Background()

	envelope := &models.UCPEnvelope{
		V:        "ucp/1.0",
		Type:     "application",
		ThreadID: "01J3K...",
		From:     "alice@example.com",
		To:       []string{"bob@example.com"},
		SigningKey: "base64_key",
	}

	// Store
	if err := s.StoreMessage(ctx, envelope, []byte("encrypted")); err != nil {
		t.Fatalf("StoreMessage error: %v", err)
	}

	// Retrieve thread messages
	messages, err := s.GetThreadMessages(ctx, envelope.ThreadID)
	if err != nil {
		t.Fatalf("GetThreadMessages error: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("GetThreadMessages: got %d messages, want 1", len(messages))
	}

	if messages[0].From != envelope.From {
		t.Errorf("From mismatch: got %q, want %q", messages[0].From, envelope.From)
	}
}

func TestMockStoreIdentity(t *testing.T) {
	s := NewMockStore()
	ctx := context.Background()

	identity := &models.Identity{
		Address:     "alice@example.com",
		IdentityKey: "base64_identity_key",
		SigningKeys: []models.SigningKey{
			{
				Key:     "base64_signing_key",
				Expires: time.Now().Unix() + 60*24*3600,
				Issued:  time.Now().Unix(),
				Status:  "active",
			},
		},
		RevocationKey: "base64_revocation_key",
		Capabilities:  []string{"ucp/1.0"},
	}

	// Store
	if err := s.StoreIdentity(ctx, identity); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	// Retrieve
	retrieved, err := s.GetIdentity(ctx, identity.Address)
	if err != nil {
		t.Fatalf("GetIdentity error: %v", err)
	}

	if retrieved.Address != identity.Address {
		t.Errorf("Address mismatch: got %q, want %q", retrieved.Address, identity.Address)
	}

	if len(retrieved.SigningKeys) != 1 {
		t.Errorf("SigningKeys length: got %d, want 1", len(retrieved.SigningKeys))
	}
}

func TestMockStoreSession(t *testing.T) {
	s := NewMockStore()
	ctx := context.Background()

	address := "alice@example.com"
	token := "session_token_abc123"
	expiresAt := time.Now().Unix() + 24*3600

	// Create
	if err := s.CreateSession(ctx, address, token, expiresAt); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Retrieve
	retrieved, err := s.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}

	if retrieved != address {
		t.Errorf("Address mismatch: got %q, want %q", retrieved, address)
	}

	// Revoke
	if err := s.RevokeSession(ctx, token); err != nil {
		t.Fatalf("RevokeSession error: %v", err)
	}

	// Retrieve again (should be revoked)
	retrieved, err = s.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}

	if retrieved != "" {
		t.Errorf("Revoked session should return empty address, got %q", retrieved)
	}
}

func TestMockStoreAttachment(t *testing.T) {
	s := NewMockStore()
	ctx := context.Background()

	attachment := &models.Attachment{
		ID:       "01J3K_ATTACH",
		Name:     "document.pdf",
		MimeType: "application/pdf",
		Size:     204800,
		SHA256:   "abc123def456...",
	}

	// Store
	if err := s.StoreAttachment(ctx, attachment); err != nil {
		t.Fatalf("StoreAttachment error: %v", err)
	}

	// Retrieve
	retrieved, err := s.GetAttachment(ctx, attachment.ID)
	if err != nil {
		t.Fatalf("GetAttachment error: %v", err)
	}

	if retrieved.Name != attachment.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, attachment.Name)
	}

	if retrieved.Size != attachment.Size {
		t.Errorf("Size mismatch: got %d, want %d", retrieved.Size, attachment.Size)
	}
}

func TestMockStoreExpiredSession(t *testing.T) {
	s := NewMockStore()
	ctx := context.Background()

	address := "alice@example.com"
	token := "expired_token"
	expiresAt := time.Now().Unix() - 3600 // Expired 1 hour ago

	// Create (with past expiry)
	if err := s.CreateSession(ctx, address, token, expiresAt); err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	// Retrieve (should be expired)
	retrieved, err := s.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}

	if retrieved != "" {
		t.Errorf("Expired session should return empty address, got %q", retrieved)
	}
}
