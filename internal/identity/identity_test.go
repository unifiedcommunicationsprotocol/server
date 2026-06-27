package identity

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

func TestGenerateKeyMaterial(t *testing.T) {
	km, err := GenerateKeyMaterial()
	if err != nil {
		t.Fatalf("GenerateKeyMaterial error: %v", err)
	}

	if km.IdentityPrivateKey == nil || km.IdentityPublicKey == nil {
		t.Error("Identity key not generated")
	}
	if km.SigningPrivateKey == nil || km.SigningPublicKey == nil {
		t.Error("Signing key not generated")
	}
	if km.RevocationPrivateKey == nil || km.RevocationPublicKey == nil {
		t.Error("Revocation key not generated")
	}

	// Verify key sizes
	if len(km.IdentityPublicKey) != ed25519.PublicKeySize {
		t.Errorf("Identity key size: got %d, want %d", len(km.IdentityPublicKey), ed25519.PublicKeySize)
	}
	if len(km.SigningPublicKey) != ed25519.PublicKeySize {
		t.Errorf("Signing key size: got %d, want %d", len(km.SigningPublicKey), ed25519.PublicKeySize)
	}
}

func TestPublicKeyEncoding(t *testing.T) {
	km, _ := GenerateKeyMaterial()

	// Encode
	encoded := EncodePublicKey(km.IdentityPublicKey)
	if encoded == "" {
		t.Error("EncodePublicKey returned empty string")
	}

	// Decode
	decoded, err := DecodePublicKey(encoded)
	if err != nil {
		t.Fatalf("DecodePublicKey error: %v", err)
	}

	// Verify round-trip
	if !km.IdentityPublicKey.Equal(decoded) {
		t.Error("Public key round-trip failed")
	}
}

func TestSigningKeyBindingString(t *testing.T) {
	km, _ := GenerateKeyMaterial()
	expiresAt := time.Now().Unix() + 60*24*3600

	bindingStr := SigningKeyBindingString(km.SigningPublicKey, expiresAt)

	// Verify format
	if bindingStr == "" {
		t.Error("SigningKeyBindingString returned empty string")
	}

	// Should start with "signing_key:"
	expected := "signing_key:"
	if len(bindingStr) < len(expected) || bindingStr[:len(expected)] != expected {
		t.Errorf("Binding string format wrong: %q", bindingStr)
	}
}

func TestSigningKeySignatureVerification(t *testing.T) {
	km, _ := GenerateKeyMaterial()
	expiresAt := time.Now().Unix() + 60*24*3600

	// Sign
	sig := SignSigningKey(km.IdentityPrivateKey, km.SigningPublicKey, expiresAt)

	// Verify signature is non-empty
	if sig == "" {
		t.Error("SignSigningKey returned empty signature")
	}

	// Verify it can be decoded
	_, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("Signature not valid base64: %v", err)
	}

	// Verify with correct identity key
	if err := VerifySigningKeySignature(km.IdentityPublicKey, km.SigningPublicKey, expiresAt, sig); err != nil {
		t.Fatalf("VerifySigningKeySignature error: %v", err)
	}

	// Verify with wrong identity key fails
	km2, _ := GenerateKeyMaterial()
	if err := VerifySigningKeySignature(km2.IdentityPublicKey, km.SigningPublicKey, expiresAt, sig); err == nil {
		t.Error("VerifySigningKeySignature should fail with wrong identity key")
	}

	// Verify with wrong expiry fails
	if err := VerifySigningKeySignature(km.IdentityPublicKey, km.SigningPublicKey, expiresAt+1, sig); err == nil {
		t.Error("VerifySigningKeySignature should fail with wrong expiry")
	}
}

func TestMessageSignatureVerification(t *testing.T) {
	km, _ := GenerateKeyMaterial()

	msg := []byte(`{"id":"01J3K","from":"alice@example.com","to":["bob@example.com"],"subject":"Hello"}`)

	// Sign
	sig := SignMessage(km.SigningPrivateKey, msg)
	if sig == "" {
		t.Error("SignMessage returned empty signature")
	}

	// Verify
	if err := VerifyMessageSignature(km.SigningPublicKey, msg, sig); err != nil {
		t.Fatalf("VerifyMessageSignature error: %v", err)
	}

	// Tampered message should fail
	tamperedMsg := []byte(`{"id":"01J3K","from":"attacker@example.com","to":["bob@example.com"],"subject":"Hello"}`)
	if err := VerifyMessageSignature(km.SigningPublicKey, tamperedMsg, sig); err == nil {
		t.Error("VerifyMessageSignature should fail with tampered message")
	}

	// Wrong signing key should fail
	km2, _ := GenerateKeyMaterial()
	if err := VerifyMessageSignature(km2.SigningPublicKey, msg, sig); err == nil {
		t.Error("VerifyMessageSignature should fail with wrong signing key")
	}
}

func TestRevocationSignatureVerification(t *testing.T) {
	km, _ := GenerateKeyMaterial()

	revRecord := &models.RevocationRecord{
		Type:        "revocation",
		Version:     "1",
		Identity:    "alice@example.com",
		IdentityKey: EncodePublicKey(km.IdentityPublicKey),
		Reason:      "compromised",
		Timestamp:   time.Now().Unix(),
	}

	// Sign
	revRecord.RevocationSig = SignRevocation(km.RevocationPrivateKey, revRecord)
	if revRecord.RevocationSig == "" {
		t.Error("SignRevocation returned empty signature")
	}

	// Verify
	if err := VerifyRevocationSignature(km.RevocationPublicKey, revRecord); err != nil {
		t.Fatalf("VerifyRevocationSignature error: %v", err)
	}

	// Tampered record should fail
	revRecord.Identity = "attacker@example.com"
	if err := VerifyRevocationSignature(km.RevocationPublicKey, revRecord); err == nil {
		t.Error("VerifyRevocationSignature should fail with tampered record")
	}
}

func TestCreateSigningKey(t *testing.T) {
	km, _ := GenerateKeyMaterial()

	sk := CreateSigningKey(km.IdentityPrivateKey, km.SigningPublicKey, 60)

	if sk.Key == "" {
		t.Error("CreateSigningKey returned empty key")
	}
	if sk.Status != "active" {
		t.Errorf("CreateSigningKey status: got %q, want %q", sk.Status, "active")
	}
	if sk.Expires <= sk.Issued {
		t.Error("CreateSigningKey expires should be after issued")
	}
	if sk.Sig == "" {
		t.Error("CreateSigningKey returned empty signature")
	}

	// Verify the signature in the key
	decodedPub, _ := DecodePublicKey(sk.Key)
	if err := VerifySigningKeySignature(km.IdentityPublicKey, decodedPub, sk.Expires, sk.Sig); err != nil {
		t.Fatalf("Signing key signature verification failed: %v", err)
	}
}

func TestRotateSigningKey(t *testing.T) {
	km, _ := GenerateKeyMaterial()

	newKey, err := RotateSigningKey(km.IdentityPrivateKey)
	if err != nil {
		t.Fatalf("RotateSigningKey error: %v", err)
	}

	if newKey.Key == "" {
		t.Error("RotateSigningKey returned empty key")
	}
	if newKey.Status != "active" {
		t.Error("New signing key should be active")
	}

	// Verify the new key's signature
	decodedPub, _ := DecodePublicKey(newKey.Key)
	if err := VerifySigningKeySignature(km.IdentityPublicKey, decodedPub, newKey.Expires, newKey.Sig); err != nil {
		t.Fatalf("New signing key signature verification failed: %v", err)
	}
}

func TestGenerateServerKey(t *testing.T) {
	serverKey, err := GenerateServerKey("ucp.example.com")
	if err != nil {
		t.Fatalf("GenerateServerKey error: %v", err)
	}

	if serverKey.PrivateKey == nil || serverKey.PublicKey == nil {
		t.Error("Server key not generated")
	}
	if serverKey.Domain != "ucp.example.com" {
		t.Errorf("Domain: got %q, want %q", serverKey.Domain, "ucp.example.com")
	}

	if len(serverKey.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("Server key size: got %d, want %d", len(serverKey.PublicKey), ed25519.PublicKeySize)
	}
}

func TestServerHelloSignature(t *testing.T) {
	serverKey, _ := GenerateServerKey("ucp.example.com")

	authToken := "abc123token"
	serverID := "ucp.example.com"

	// Sign
	sig := serverKey.SignServerHello(authToken, serverID)
	if sig == "" {
		t.Error("SignServerHello returned empty signature")
	}

	// Verify
	if err := VerifyServerHello(serverKey.PublicKey, authToken, serverID, sig); err != nil {
		t.Fatalf("VerifyServerHello error: %v", err)
	}

	// Wrong auth token should fail
	if err := VerifyServerHello(serverKey.PublicKey, "wrongtoken", serverID, sig); err == nil {
		t.Error("VerifyServerHello should fail with wrong auth token")
	}

	// Wrong server ID should fail
	if err := VerifyServerHello(serverKey.PublicKey, authToken, "wrong.com", sig); err == nil {
		t.Error("VerifyServerHello should fail with wrong server ID")
	}

	// Wrong server key should fail
	serverKey2, _ := GenerateServerKey("other.com")
	if err := VerifyServerHello(serverKey2.PublicKey, authToken, serverID, sig); err == nil {
		t.Error("VerifyServerHello should fail with wrong server key")
	}
}
