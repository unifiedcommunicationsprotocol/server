package mls

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// NewKeyPackage creates a new MLS key package for a member.
func NewKeyPackage(member string, signingKey ed25519.PublicKey, identityKey ed25519.PublicKey) *KeyPackage {
	// Generate random HPKE init key (32 bytes for X25519)
	initKey := make([]byte, 32)
	rand.Read(initKey)

	// Generate encryption key
	encryptKey := make([]byte, 32)
	rand.Read(encryptKey)

	credential := NewCredential(signingKey, identityKey, member)
	leafNode := NewLeafNode(member, signingKey, encryptKey, credential)

	return &KeyPackage{
		Version:    0x0001, // MLS 1.0
		CipherSuite: 0x0001, // MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519
		InitKey:    initKey,
		LeafNode:   leafNode,
		Extensions: []Extension{},
		Signature:  []byte{}, // Will be signed
	}
}

// Sign signs the key package with the signing key.
func (kp *KeyPackage) Sign(signingKeyPrivate ed25519.PrivateKey) error {
	// Build the signing input: version || ciphersuite || init_key || leaf_node || extensions
	s := NewBuilder()
	s.WriteUint16(kp.Version)
	s.WriteUint16(kp.CipherSuite)
	s.WriteBytes(kp.InitKey)
	s.WriteOpaque(kp.LeafNode.Serialize())
	s.WriteUint16(uint16(len(kp.Extensions)))
	for _, ext := range kp.Extensions {
		s.WriteUint16(ext.ExtensionType)
		s.WriteOpaque(ext.ExtensionData)
	}

	signingInput := s.Bytes()

	// Sign with Ed25519
	sig := ed25519.Sign(signingKeyPrivate, signingInput)
	kp.Signature = sig

	return nil
}

// Verify verifies the key package signature.
func (kp *KeyPackage) Verify() error {
	// Rebuild signing input
	s := NewBuilder()
	s.WriteUint16(kp.Version)
	s.WriteUint16(kp.CipherSuite)
	s.WriteBytes(kp.InitKey)
	s.WriteOpaque(kp.LeafNode.Serialize())
	s.WriteUint16(uint16(len(kp.Extensions)))
	for _, ext := range kp.Extensions {
		s.WriteUint16(ext.ExtensionType)
		s.WriteOpaque(ext.ExtensionData)
	}

	signingInput := s.Bytes()

	// Verify signature
	if !ed25519.Verify(ed25519.PublicKey(kp.LeafNode.SignatureKey), signingInput, kp.Signature) {
		return fmt.Errorf("invalid key package signature")
	}

	return nil
}

// Hash computes the SHA-256 hash of the key package.
func (kp *KeyPackage) Hash() []byte {
	h := sha256.Sum256(kp.Serialize())
	return h[:]
}

// Serialize encodes the key package to bytes.
func (kp *KeyPackage) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint16(kp.Version)
	s.WriteUint16(kp.CipherSuite)
	s.WriteBytes(kp.InitKey)
	s.WriteOpaque(kp.LeafNode.Serialize())
	s.WriteUint16(uint16(len(kp.Extensions)))
	for _, ext := range kp.Extensions {
		s.WriteUint16(ext.ExtensionType)
		s.WriteOpaque(ext.ExtensionData)
	}
	s.WriteOpaque(kp.Signature)
	return s.Bytes()
}

// DeserializeKeyPackage decodes a key package.
func DeserializeKeyPackage(data []byte) (*KeyPackage, error) {
	ser := &Serializer{}

	version, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode version: %w", err)
	}
	data = data[consumed:]

	ciphersuite, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode ciphersuite: %w", err)
	}
	data = data[consumed:]

	initKey, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode init key: %w", err)
	}
	data = data[consumed:]

	leafNodeData, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode leaf node: %w", err)
	}
	data = data[consumed:]

	leafNode, err := DeserializeLeafNode(leafNodeData)
	if err != nil {
		return nil, fmt.Errorf("deserialize leaf node: %w", err)
	}

	// Decode extensions count
	extCount, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode extensions count: %w", err)
	}
	data = data[consumed:]

	extensions := make([]Extension, extCount)
	for i := 0; i < int(extCount); i++ {
		extType, consumed, err := ser.DecodeUint16(data)
		if err != nil {
			return nil, fmt.Errorf("decode extension type: %w", err)
		}
		data = data[consumed:]

		extData, consumed, err := ser.DecodeOpaque(data)
		if err != nil {
			return nil, fmt.Errorf("decode extension data: %w", err)
		}
		data = data[consumed:]

		extensions[i] = Extension{
			ExtensionType: extType,
			ExtensionData: extData,
		}
	}

	// Decode signature
	signature, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	return &KeyPackage{
		Version:    version,
		CipherSuite: ciphersuite,
		InitKey:    initKey,
		LeafNode:   leafNode,
		Extensions: extensions,
		Signature:  signature,
	}, nil
}

// KeyPackageRef is a reference to a key package (its hash).
type KeyPackageRef struct {
	Algorithm uint8  // 0 = sha256
	Value     []byte // Hash value
}

// NewKeyPackageRef creates a reference from a key package.
func NewKeyPackageRef(kp *KeyPackage) *KeyPackageRef {
	return &KeyPackageRef{
		Algorithm: 0, // SHA-256
		Value:     kp.Hash(),
	}
}

// Serialize encodes the reference.
func (kpr *KeyPackageRef) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint8(kpr.Algorithm)
	s.WriteOpaque(kpr.Value)
	return s.Bytes()
}

// DeserializeKeyPackageRef decodes a reference.
func DeserializeKeyPackageRef(data []byte) (*KeyPackageRef, error) {
	ser := &Serializer{}

	algo, consumed, err := ser.DecodeUint8(data)
	if err != nil {
		return nil, fmt.Errorf("decode algorithm: %w", err)
	}
	data = data[consumed:]

	value, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode value: %w", err)
	}

	return &KeyPackageRef{
		Algorithm: algo,
		Value:     value,
	}, nil
}
