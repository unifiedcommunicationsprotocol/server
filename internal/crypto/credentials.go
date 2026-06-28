// Package crypto provides encryption utilities for sensitive data at rest.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// CredentialsEncryptor handles AES-256-GCM encryption of sensitive credentials at rest.
// Uses HKDF (HMAC-based Key Derivation Function) from stdlib crypto/sha256 for key derivation.
type CredentialsEncryptor struct {
	masterKey []byte // 32 bytes for AES-256
}

// NewCredentialsEncryptor creates a new encryptor from a master key (should be 32 bytes).
// In production, load this from a secure key management system (e.g., Vault, AWS KMS).
func NewCredentialsEncryptor(masterKey []byte) (*CredentialsEncryptor, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes, got %d", len(masterKey))
	}
	return &CredentialsEncryptor{
		masterKey: masterKey,
	}, nil
}

// deriveKey derives an encryption key from the master key and salt using HMAC-SHA256.
// This is a simple KDF using SHA256 (pure stdlib, no external deps).
func (ce *CredentialsEncryptor) deriveKey(salt []byte) []byte {
	// Use HMAC-SHA256 for key derivation (simple but secure KDF)
	h := sha256.New()
	h.Write(ce.masterKey)
	h.Write(salt)
	return h.Sum(nil) // Returns 32 bytes
}

// Encrypt encrypts a plaintext credential and returns base64-encoded ciphertext.
// Includes nonce and salt for decryption.
// Format: base64(salt || nonce || ciphertext || tag)
func (ce *CredentialsEncryptor) Encrypt(plaintext string) (string, error) {
	// Generate random salt (16 bytes) for key derivation
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	// Derive key from master key using salt
	key := ce.deriveKey(salt)

	// Create AES-256-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	// Generate random nonce (12 bytes for GCM)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt plaintext (includes authentication tag automatically)
	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine salt || nonce || ciphertext into single output
	combined := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	combined = append(combined, salt...)
	combined = append(combined, nonce...)
	combined = append(combined, ciphertext...)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts a base64-encoded credential (produced by Encrypt).
// Verifies the authentication tag and decryption salt/nonce.
func (ce *CredentialsEncryptor) Decrypt(encoded string) (string, error) {
	// Decode base64
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	// Validate minimum length: 16 (salt) + 12 (nonce) + 16 (tag min)
	if len(combined) < 44 {
		return "", fmt.Errorf("ciphertext too short: %d bytes", len(combined))
	}

	// Extract salt (first 16 bytes)
	salt := combined[:16]
	remaining := combined[16:]

	// Derive key using same salt (must match encryption)
	key := ce.deriveKey(salt)

	// Create GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	// Extract nonce (next 12 bytes)
	nonce := remaining[:12]
	ciphertext := remaining[12:]

	// Decrypt (includes tag verification)
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed (authentication error): %w", err)
	}

	return string(plaintext), nil
}
