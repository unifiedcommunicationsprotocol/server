package mls

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestNewCredential(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	address := "alice@example.com"
	cred := NewCredential(signingPub, identityPub, address)

	if cred.CredentialType != "signing_key" {
		t.Errorf("credential type: got %q, want %q", cred.CredentialType, "signing_key")
	}

	if cred.Identity != address {
		t.Errorf("identity: got %q, want %q", cred.Identity, address)
	}

	if len(cred.SigningKey) != ed25519.PublicKeySize {
		t.Errorf("signing key length: got %d, want %d", len(cred.SigningKey), ed25519.PublicKeySize)
	}
}

func TestCredentialVerify(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	cred := NewCredential(signingPub, identityPub, "alice@example.com")

	err := cred.Verify()
	if err != nil {
		t.Errorf("verify: %v", err)
	}
}

func TestCredentialSerialization(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	cred := NewCredential(signingPub, identityPub, "alice@example.com")

	serialized := cred.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized credential is empty")
	}

	deserialized, err := DeserializeCredential(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.Identity != "alice@example.com" {
		t.Errorf("identity: got %q, want %q", deserialized.Identity, "alice@example.com")
	}

	if string(deserialized.SigningKey) != string(signingPub) {
		t.Error("signing key mismatch")
	}
}

func TestNewLeafNode(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	signingKey := make([]byte, 32)
	encryptionKey := make([]byte, 32)

	cred := NewCredential(signingPub, identityPub, "alice@example.com")
	ln := NewLeafNode("alice@example.com", signingKey, encryptionKey, cred)

	if len(ln.EncryptionKey) != 32 {
		t.Errorf("encryption key length: got %d, want 32", len(ln.EncryptionKey))
	}

	if len(ln.SignatureKey) != 32 {
		t.Errorf("signature key length: got %d, want 32", len(ln.SignatureKey))
	}

	if ln.Credential == nil {
		t.Error("credential is nil")
	}

	if ln.Capabilities == nil {
		t.Error("capabilities is nil")
	}
}

func TestLeafNodeSerialization(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	signingKey := make([]byte, 32)
	encryptionKey := make([]byte, 32)

	cred := NewCredential(signingPub, identityPub, "alice@example.com")
	ln := NewLeafNode("alice@example.com", signingKey, encryptionKey, cred)

	serialized := ln.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized leaf node is empty")
	}

	deserialized, err := DeserializeLeafNode(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if len(deserialized.EncryptionKey) != 32 {
		t.Errorf("encryption key length: got %d, want 32", len(deserialized.EncryptionKey))
	}

	if deserialized.Credential == nil {
		t.Error("credential is nil after deserialization")
	}
}
