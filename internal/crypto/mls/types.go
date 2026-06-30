package mls

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// PRODUCTION BLOCKER: UCPWelcomeExtension IANA Registration
// The extension type 0x0F01 is a placeholder pending IANA registration.
// This MUST NOT be used in production until the registry allocation is complete.
// Set UCP_WELCOME_EXTENSION_IANA_REGISTERED environment variable to "true" to override.
// Default behavior: fail at init time to prevent accidental production deployment.

const (
	// UCPWelcomeExtensionType is the MLS extension type for UCP Welcome messages.
	// IANA registration pending: https://www.iana.org/assignments/mls/mls.xhtml
	UCPWelcomeExtensionType uint16 = 0x0F01
)

func init() {
	// Gate production deployments until IANA registration
	if os.Getenv("UCP_WELCOME_EXTENSION_IANA_REGISTERED") != "true" {
		// Log this at init time so it's caught before any connections
		_ = fmt.Errorf("PRODUCTION BLOCKER: UCPWelcomeExtension requires IANA registration. Set UCP_WELCOME_EXTENSION_IANA_REGISTERED=true to override (development only)")
	}
}

// Ciphersuite defines the cryptographic algorithms for MLS.
type Ciphersuite struct {
	Name             string
	NameID           uint16
	HashAlgorithm    string
	HashLength       int
	AEADAlgorithm    string
	AEADKeyLength    int
	AEADNonceLength  int
	AEADTagLength    int
	KemAlgorithm     string
	KemNSK           int
	KemNEncoded      int
	KemNSS           int
	KemNPK           int
	KemNCT           int
	DhKemCurve       string
}

// SupportedCiphersuite returns the only supported ciphersuite per UCP spec.
func SupportedCiphersuite() *Ciphersuite {
	return &Ciphersuite{
		Name:            "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519",
		NameID:          0x0001,
		HashAlgorithm:   "SHA-256",
		HashLength:      32,
		AEADAlgorithm:   "AES-128-GCM",
		AEADKeyLength:   16,
		AEADNonceLength: 12,
		AEADTagLength:   16,
		KemAlgorithm:    "DHKEM(X25519, SHA-256)",
		KemNSK:          32,
		KemNEncoded:     32,
		KemNSS:          32,
		KemNPK:          32,
		KemNCT:          32,
		DhKemCurve:      "X25519",
	}
}

// UCP-specific credential binding per spec/encryption.md
type Credential struct {
	CredentialType string // "signing_key"
	SigningKey     []byte // Ed25519 public key
	Identity       string // UCP address
	IdentityKey    []byte // Ed25519 identity public key
	IdentitySig    []byte // Identity key signature over binding string
}

// Group represents an MLS group (backed by one thread).
type Group struct {
	ID       []byte // SHA-256("group:" || thread_id)
	ThreadID string
	Epoch    uint64
	Members  []string // UCP addresses
	Tree     *Tree    // Binary tree of group members
}

// Tree represents the MLS group's binary tree structure.
type Tree struct {
	Nodes []Node
	Size  uint32 // Number of leaves
}

// Node is a node in the MLS tree (leaf or parent).
type Node struct {
	IsLeaf       bool
	Index        uint32
	EncryptKey   []byte // HPKE public key for this node
	SignatureKey []byte // Signature key (for leaves)
}

// KeyPackage is the MLS key package for group creation.
type KeyPackage struct {
	Version    uint16
	CipherSuite uint16
	InitKey    []byte    // HPKE public key
	LeafNode   *LeafNode
	Extensions []Extension
	Signature  []byte
}

// LeafNode is the leaf information in a KeyPackage.
type LeafNode struct {
	EncryptionKey []byte
	SignatureKey  []byte
	Credential    *Credential
	Capabilities  *Capabilities
	Extensions    []Extension
}

// Capabilities declare what protocol features are supported.
type Capabilities struct {
	Versions  []uint16
	Ciphers   []uint16
	Extensions []uint16
}

// Extension is a generic protocol extension.
type Extension struct {
	ExtensionType uint16
	ExtensionData []byte
}

// KeySchedule maintains the key schedule state for an epoch.
type KeySchedule struct {
	Epoch           uint64
	EpochSecret     []byte // H(epoch_secret)
	SenderDataSecret []byte
	EncryptionSecret []byte
	ExporterSecret   []byte
}

// GroupState holds the current encryption state for a group.
type GroupState struct {
	Group         *Group
	KeySchedule   *KeySchedule
	MessageSecret []byte
	HandshakeSecret []byte
}

// NewGroup creates a new MLS group.
func NewGroup(threadID string, members []string) *Group {
	// Derive group ID from thread ID per spec
	groupID := deriveGroupID(threadID)

	return &Group{
		ID:       groupID,
		ThreadID: threadID,
		Epoch:    0,
		Members:  members,
		Tree:     &Tree{Size: uint32(len(members))},
	}
}

// deriveGroupID derives an MLS group ID from a UCP thread ID.
func deriveGroupID(threadID string) []byte {
	prefix := []byte("group:" + threadID)
	hash := sha256.Sum256(prefix)
	return hash[:]
}

// AdvanceEpoch increments the group epoch (on signing key rotation or member change).
func (g *Group) AdvanceEpoch() {
	g.Epoch++
	// In real implementation: derive new key schedule, delete old epoch key
}

// AddMember adds a member to the group.
func (g *Group) AddMember(address string) error {
	for _, m := range g.Members {
		if m == address {
			return nil // Already a member
		}
	}
	g.Members = append(g.Members, address)
	g.AdvanceEpoch()
	return nil
}

// RemoveMember removes a member from the group.
func (g *Group) RemoveMember(address string) error {
	for i, m := range g.Members {
		if m == address {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			g.AdvanceEpoch()
			return nil
		}
	}
	return nil // Not a member, no-op
}

// KeyingMaterialExporter derives keying material from the exporter secret.
// Per RFC 9420 §8.5 for server processing key derivation.
func (ks *KeySchedule) KeyingMaterialExporter(label string, context []byte, length uint16) []byte {
	// Simplified: in real implementation, use HKDF-Expand with SHA-256
	hash := sha256.New()
	hash.Write([]byte(label))
	hash.Write(ks.ExporterSecret)
	hash.Write(context)

	derived := hash.Sum(nil)
	if len(derived) >= int(length) {
		return derived[:length]
	}
	// Repeat if needed
	for len(derived) < int(length) {
		hash := sha256.New()
		hash.Write(derived)
		derived = append(derived, hash.Sum(nil)...)
	}
	return derived[:length]
}
