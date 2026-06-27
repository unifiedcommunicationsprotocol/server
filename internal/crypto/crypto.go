// Package crypto implements MLS (RFC 9420) group management, encryption/decryption, and key rotation.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// Group represents an MLS group for a thread.
type Group struct {
	ID          []byte // SHA-256 of "group:" || thread_id
	ThreadID    models.ULID
	Epoch       uint64
	Members     []string
	EncryptKey  []byte // For testing/demo; real MLS uses key schedule
}

// Manager manages MLS groups and encryption/decryption.
type Manager struct {
	groups map[string]*Group
}

// New creates a new crypto Manager.
func New() *Manager {
	return &Manager{
		groups: make(map[string]*Group),
	}
}

// CreateGroup creates a new MLS group for a thread.
// Real implementation: use MLS RFC 9420 library to set up group with KeyPackages.
func (m *Manager) CreateGroup(threadID models.ULID, members []string) (*Group, error) {
	if len(members) == 0 {
		return nil, fmt.Errorf("group must have at least one member")
	}

	// Derive group ID from thread ID (per spec: SHA-256("group:" || thread_id))
	groupID := deriveGroupID(threadID)

	group := &Group{
		ID:       groupID,
		ThreadID: threadID,
		Epoch:    0,
		Members:  members,
	}

	// Generate a test encryption key; real MLS uses key schedule
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate encryption key: %w", err)
	}
	group.EncryptKey = key

	m.groups[string(groupID)] = group
	return group, nil
}

// GetGroup retrieves a group by ID.
func (m *Manager) GetGroup(groupID []byte) (*Group, error) {
	group, ok := m.groups[string(groupID)]
	if !ok {
		return nil, fmt.Errorf("group not found")
	}
	return group, nil
}

// EncryptMessage encrypts a message for a group using the group's encryption key.
// Real implementation: use MLS PrivateMessage with proper key derivation and framing.
// For now: use AES-256-GCM for demo/testing.
func (m *Manager) EncryptMessage(groupID []byte, plaintext []byte) ([]byte, error) {
	group, err := m.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(group.EncryptKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptMessage decrypts a message from a group.
// Real implementation: use MLS PrivateMessage deserialization and decryption.
func (m *Manager) DecryptMessage(groupID []byte, ciphertext []byte) ([]byte, error) {
	group, err := m.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(group.EncryptKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// AddMember adds a member to a group.
// Real implementation: commit Add proposal, advance epoch, send Welcome to new member.
func (m *Manager) AddMember(groupID []byte, member string) error {
	group, err := m.GetGroup(groupID)
	if err != nil {
		return err
	}

	// Check if already a member
	for _, existing := range group.Members {
		if existing == member {
			return fmt.Errorf("member already in group")
		}
	}

	group.Members = append(group.Members, member)
	group.Epoch++

	return nil
}

// RemoveMember removes a member from a group.
// Real implementation: commit Remove proposal, advance epoch.
func (m *Manager) RemoveMember(groupID []byte, member string) error {
	group, err := m.GetGroup(groupID)
	if err != nil {
		return err
	}

	for i, existing := range group.Members {
		if existing == member {
			group.Members = append(group.Members[:i], group.Members[i+1:]...)
			group.Epoch++
			return nil
		}
	}

	return fmt.Errorf("member not in group")
}

// AdvanceEpoch advances the group epoch (on signing key rotation).
// Real implementation: commit Update proposal with new signing key credential.
func (m *Manager) AdvanceEpoch(groupID []byte) error {
	group, err := m.GetGroup(groupID)
	if err != nil {
		return err
	}

	group.Epoch++

	// In real MLS: derive new epoch key, delete old one (forward secrecy)
	// For demo: regenerate encryption key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate new encryption key: %w", err)
	}
	group.EncryptKey = key

	return nil
}

// Helper to derive group ID from thread ID (per spec).
func deriveGroupID(threadID models.ULID) []byte {
	// Real implementation: SHA-256("group:" || thread_id)
	// For now, just return base64 encoding
	return []byte(base64.StdEncoding.EncodeToString([]byte("group:" + string(threadID))))
}

// DeriveBCCGroupID derives a BCC group ID (per spec).
// Schema: SHA-256("group_bcc:" || thread_id || ":" || recipient_address)
func DeriveBCCGroupID(threadID models.ULID, recipientAddress string) []byte {
	// Real implementation: SHA-256 hash
	return []byte(base64.StdEncoding.EncodeToString([]byte("group_bcc:" + string(threadID) + ":" + recipientAddress)))
}

// KeyPackage represents an MLS key package.
type KeyPackage struct {
	GroupID    []byte
	InitKey    []byte // HPKE public key
	SigningKey []byte // Ed25519 public key
}

// CreateKeyPackage creates a new key package for group creation.
// Real implementation: generate HPKE and Ed25519 keys, build per RFC 9420.
func (m *Manager) CreateKeyPackage(groupID []byte) (*KeyPackage, error) {
	// In reality, these would be proper HPKE and Ed25519 keys
	initKey := make([]byte, 32)
	if _, err := rand.Read(initKey); err != nil {
		return nil, fmt.Errorf("generate init key: %w", err)
	}

	signingKey := make([]byte, 32)
	if _, err := rand.Read(signingKey); err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}

	return &KeyPackage{
		GroupID:    groupID,
		InitKey:    initKey,
		SigningKey: signingKey,
	}, nil
}

// EncodeKeyPackage encodes a key package as base64.
func EncodeKeyPackage(kp *KeyPackage) string {
	// In reality: TLS-serialize per RFC 9420, then base64
	// For now: concatenate with length prefixes
	data := make([]byte, 0)
	data = append(data, byte(len(kp.GroupID)))
	data = append(data, kp.GroupID...)
	data = append(data, byte(len(kp.InitKey)))
	data = append(data, kp.InitKey...)
	data = append(data, byte(len(kp.SigningKey)))
	data = append(data, kp.SigningKey...)
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeKeyPackage decodes a base64-encoded key package.
func DecodeKeyPackage(encoded string) (*KeyPackage, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Simplified parsing with length prefixes
	if len(data) < 4 {
		return nil, fmt.Errorf("key package too short")
	}

	pos := 0
	groupIDLen := int(data[pos])
	pos++

	if pos+groupIDLen > len(data) {
		return nil, fmt.Errorf("invalid group ID length")
	}
	groupID := data[pos : pos+groupIDLen]
	pos += groupIDLen

	if pos >= len(data) {
		return nil, fmt.Errorf("missing init key")
	}
	initKeyLen := int(data[pos])
	pos++

	if pos+initKeyLen > len(data) {
		return nil, fmt.Errorf("invalid init key length")
	}
	initKey := data[pos : pos+initKeyLen]
	pos += initKeyLen

	if pos >= len(data) {
		return nil, fmt.Errorf("missing signing key")
	}
	signingKeyLen := int(data[pos])
	pos++

	if pos+signingKeyLen > len(data) {
		return nil, fmt.Errorf("invalid signing key length")
	}
	signingKey := data[pos : pos+signingKeyLen]

	kp := &KeyPackage{
		GroupID:    groupID,
		InitKey:    initKey,
		SigningKey: signingKey,
	}

	return kp, nil
}
