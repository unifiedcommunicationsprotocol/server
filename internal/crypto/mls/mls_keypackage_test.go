package mls

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestNewKeyPackage(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("alice@example.com", signingPub, identityPub)

	if kp.Version != 0x0001 {
		t.Errorf("version: got %d, want 1", kp.Version)
	}

	if kp.CipherSuite != 0x0001 {
		t.Errorf("ciphersuite: got %d, want 1", kp.CipherSuite)
	}

	if len(kp.InitKey) != 32 {
		t.Errorf("init key length: got %d, want 32", len(kp.InitKey))
	}

	if kp.LeafNode == nil {
		t.Error("leaf node is nil")
	}
}

func TestKeyPackageSign(t *testing.T) {
	signingPub, signingPriv, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("alice@example.com", signingPub, identityPub)

	err := kp.Sign(signingPriv)
	if err != nil {
		t.Errorf("sign: %v", err)
	}

	if len(kp.Signature) == 0 {
		t.Error("signature is empty after signing")
	}
}

func TestKeyPackageVerify(t *testing.T) {
	signingPub, signingPriv, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("alice@example.com", signingPub, identityPub)
	kp.Sign(signingPriv)

	err := kp.Verify()
	if err != nil {
		t.Errorf("verify: %v", err)
	}
}

func TestKeyPackageHash(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("alice@example.com", signingPub, identityPub)
	hash := kp.Hash()

	if len(hash) != 32 {
		t.Errorf("hash length: got %d, want 32", len(hash))
	}

	// Deterministic
	hash2 := kp.Hash()
	if string(hash) != string(hash2) {
		t.Error("hash not deterministic")
	}
}

func TestKeyPackageSerialization(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("alice@example.com", signingPub, identityPub)

	serialized := kp.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized key package is empty")
	}

	deserialized, err := DeserializeKeyPackage(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.Version != kp.Version {
		t.Errorf("version: got %d, want %d", deserialized.Version, kp.Version)
	}

	if deserialized.CipherSuite != kp.CipherSuite {
		t.Errorf("ciphersuite: got %d, want %d", deserialized.CipherSuite, kp.CipherSuite)
	}
}

func TestKeyPackageRef(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("alice@example.com", signingPub, identityPub)
	ref := NewKeyPackageRef(kp)

	if ref.Algorithm != 0 {
		t.Errorf("algorithm: got %d, want 0", ref.Algorithm)
	}

	if len(ref.Value) != 32 {
		t.Errorf("value length: got %d, want 32", len(ref.Value))
	}

	serialized := ref.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized ref is empty")
	}

	deserialized, err := DeserializeKeyPackageRef(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if string(deserialized.Value) != string(ref.Value) {
		t.Error("ref value mismatch")
	}
}
