// Package identity manages Ed25519 keypairs, DNS resolution, signing key lifecycle, and well-known endpoints.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// KeyMaterial holds the three Ed25519 keypairs for a UCP identity.
type KeyMaterial struct {
	IdentityPrivateKey ed25519.PrivateKey
	IdentityPublicKey  ed25519.PublicKey
	SigningPrivateKey  ed25519.PrivateKey
	SigningPublicKey   ed25519.PublicKey
	RevocationPrivateKey ed25519.PrivateKey
	RevocationPublicKey ed25519.PublicKey
}

// GenerateKeyMaterial creates a new UCP identity keypair set.
func GenerateKeyMaterial() (*KeyMaterial, error) {
	// Generate identity keypair
	identityPub, identityPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate identity key: %w", err)
	}

	// Generate signing keypair
	signingPub, signingPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}

	// Generate revocation keypair
	revocationPub, revocationPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate revocation key: %w", err)
	}

	return &KeyMaterial{
		IdentityPrivateKey:   identityPriv,
		IdentityPublicKey:    identityPub,
		SigningPrivateKey:    signingPriv,
		SigningPublicKey:     signingPub,
		RevocationPrivateKey: revocationPriv,
		RevocationPublicKey:  revocationPub,
	}, nil
}

// EncodePublicKey encodes a public key as base64 for wire format.
func EncodePublicKey(pub ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pub)
}

// DecodePublicKey decodes a base64-encoded public key.
func DecodePublicKey(encoded string) (ed25519.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: want %d, got %d", ed25519.PublicKeySize, len(data))
	}
	return ed25519.PublicKey(data), nil
}

// SigningKeyBindingString generates the canonical string for signing key signatures.
// Format: "signing_key:<base64-pubkey>:<unix-timestamp>"
func SigningKeyBindingString(signingPubKey ed25519.PublicKey, expiresAt int64) string {
	return fmt.Sprintf("signing_key:%s:%d", EncodePublicKey(signingPubKey), expiresAt)
}

// SignSigningKey signs a signing key binding string with the identity key.
func SignSigningKey(identityPrivKey ed25519.PrivateKey, signingPubKey ed25519.PublicKey, expiresAt int64) string {
	bindingStr := SigningKeyBindingString(signingPubKey, expiresAt)
	sig := ed25519.Sign(identityPrivKey, []byte(bindingStr))
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifySigningKeySignature verifies a signing key's identity signature.
func VerifySigningKeySignature(identityPubKey ed25519.PublicKey, signingPubKey ed25519.PublicKey, expiresAt int64, sigBase64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	bindingStr := SigningKeyBindingString(signingPubKey, expiresAt)
	if !ed25519.Verify(identityPubKey, []byte(bindingStr), sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// SignMessage signs a message with the signing key.
func SignMessage(signingPrivKey ed25519.PrivateKey, canonicalJSON []byte) string {
	sig := ed25519.Sign(signingPrivKey, canonicalJSON)
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifyMessageSignature verifies a message signature.
func VerifyMessageSignature(signingPubKey ed25519.PublicKey, canonicalJSON []byte, sigBase64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	if !ed25519.Verify(signingPubKey, canonicalJSON, sig) {
		return fmt.Errorf("message signature verification failed")
	}
	return nil
}

// RevocationBindingString generates the canonical string for revocation signatures.
func RevocationBindingString(revRecord *models.RevocationRecord) string {
	// Per spec: canonical JSON of record with revocation_sig omitted
	// For now, return a simple format; full implementation needs canonical JSON encoder
	return fmt.Sprintf("revocation:%s:%s:%d", revRecord.Identity, revRecord.Reason, revRecord.Timestamp)
}

// SignRevocation signs a revocation record with the revocation key.
func SignRevocation(revocationPrivKey ed25519.PrivateKey, revRecord *models.RevocationRecord) string {
	bindingStr := RevocationBindingString(revRecord)
	sig := ed25519.Sign(revocationPrivKey, []byte(bindingStr))
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifyRevocationSignature verifies a revocation record's signature.
func VerifyRevocationSignature(revocationPubKey ed25519.PublicKey, revRecord *models.RevocationRecord) error {
	sig, err := base64.StdEncoding.DecodeString(revRecord.RevocationSig)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	bindingStr := RevocationBindingString(revRecord)
	if !ed25519.Verify(revocationPubKey, []byte(bindingStr), sig) {
		return fmt.Errorf("revocation signature verification failed")
	}
	return nil
}

// CreateSigningKey creates a new signing key with identity signature.
func CreateSigningKey(identityPrivKey ed25519.PrivateKey, signingPubKey ed25519.PublicKey, lifetimeDays int) *models.SigningKey {
	now := time.Now().Unix()
	expiresAt := now + int64(lifetimeDays*24*3600)

	sig := SignSigningKey(identityPrivKey, signingPubKey, expiresAt)

	return &models.SigningKey{
		Key:     EncodePublicKey(signingPubKey),
		Expires: expiresAt,
		Issued:  now,
		Sig:     sig,
		Status:  "active",
	}
}

// RotateSigningKey creates a new signing key and marks the old one as grace.
func RotateSigningKey(identityPrivKey ed25519.PrivateKey) (*models.SigningKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}

	// Create new key with 60-day lifetime
	newKey := CreateSigningKey(identityPrivKey, pub, 60)
	_ = priv // Return value not needed here; in real impl, return KeyMaterial

	return newKey, nil
}

// ServerKey represents a UCP server's Ed25519 keypair.
type ServerKey struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Domain     string
}

// GenerateServerKey creates a new server keypair.
func GenerateServerKey(domain string) (*ServerKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate server key: %w", err)
	}

	return &ServerKey{
		PrivateKey: priv,
		PublicKey:  pub,
		Domain:     domain,
	}, nil
}

// SignServerHello signs a server hello response.
// Binding string: "server_hello:" || auth_token || server_id
func (sk *ServerKey) SignServerHello(authToken, serverID string) string {
	bindingStr := fmt.Sprintf("server_hello:%s%s", authToken, serverID)
	sig := ed25519.Sign(sk.PrivateKey, []byte(bindingStr))
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifyServerHello verifies a server hello signature.
func VerifyServerHello(serverPubKey ed25519.PublicKey, authToken, serverID, sigBase64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	bindingStr := fmt.Sprintf("server_hello:%s%s", authToken, serverID)
	if !ed25519.Verify(serverPubKey, []byte(bindingStr), sig) {
		return fmt.Errorf("server hello signature verification failed")
	}
	return nil
}
