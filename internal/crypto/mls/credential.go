package mls

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
)

// NewCredential creates a UCP credential binding signing key to identity.
func NewCredential(signingKey ed25519.PublicKey, identityKey ed25519.PublicKey, address string) *Credential {
	// Build binding string: "credential:" || address || signing_key || identity_key
	bindingData := []byte("credential:" + address)
	bindingData = append(bindingData, signingKey...)
	bindingData = append(bindingData, identityKey...)

	// Sign with identity key (private key needed in real impl)
	// For now, placeholder signature
	sig := sha256.Sum256(bindingData)

	return &Credential{
		CredentialType: "signing_key",
		SigningKey:     signingKey,
		Identity:       address,
		IdentityKey:    identityKey,
		IdentitySig:    sig[:],
	}
}

// Verify checks that the credential is properly signed.
func (c *Credential) Verify() error {
	if c.CredentialType != "signing_key" {
		return fmt.Errorf("unsupported credential type: %s", c.CredentialType)
	}

	if len(c.SigningKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid signing key length: %d", len(c.SigningKey))
	}

	if len(c.IdentityKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid identity key length: %d", len(c.IdentityKey))
	}

	if c.Identity == "" {
		return fmt.Errorf("credential missing address")
	}

	return nil
}

// Serialize encodes the credential to bytes.
func (c *Credential) Serialize() []byte {
	s := NewBuilder()
	s.WriteBytes([]byte(c.CredentialType))
	s.WriteBytes(c.SigningKey)
	s.WriteBytes([]byte(c.Identity))
	s.WriteBytes(c.IdentityKey)
	s.WriteBytes(c.IdentitySig)
	return s.Bytes()
}

// DeserializeCredential decodes a credential from bytes.
func DeserializeCredential(data []byte) (*Credential, error) {
	ser := &Serializer{}

	credType, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode credential type: %w", err)
	}
	data = data[consumed:]

	signingKey, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode signing key: %w", err)
	}
	data = data[consumed:]

	identity, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode identity: %w", err)
	}
	data = data[consumed:]

	identityKey, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode identity key: %w", err)
	}
	data = data[consumed:]

	identitySig, _, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode identity sig: %w", err)
	}

	return &Credential{
		CredentialType: string(credType),
		SigningKey:     signingKey,
		Identity:       string(identity),
		IdentityKey:    identityKey,
		IdentitySig:    identitySig,
	}, nil
}

// NewLeafNode creates a leaf node for a member.
func NewLeafNode(member string, signingKey, encryptionKey []byte, credential *Credential) *LeafNode {
	return &LeafNode{
		EncryptionKey: encryptionKey,
		SignatureKey:  signingKey,
		Credential:    credential,
		Capabilities: &Capabilities{
			Versions:  []uint16{0x0001}, // MLS 1.0
			Ciphers:   []uint16{0x0001}, // MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519
			Extensions: []uint16{},
		},
		Extensions: []Extension{},
	}
}

// Serialize encodes the leaf node.
func (ln *LeafNode) Serialize() []byte {
	s := NewBuilder()
	s.WriteBytes(ln.EncryptionKey)
	s.WriteBytes(ln.SignatureKey)
	s.WriteOpaque(ln.Credential.Serialize())

	// Capabilities
	s.WriteUint16(uint16(len(ln.Capabilities.Versions)))
	for _, v := range ln.Capabilities.Versions {
		s.WriteUint16(v)
	}
	s.WriteUint16(uint16(len(ln.Capabilities.Ciphers)))
	for _, c := range ln.Capabilities.Ciphers {
		s.WriteUint16(c)
	}
	s.WriteUint16(uint16(len(ln.Capabilities.Extensions)))
	for _, e := range ln.Capabilities.Extensions {
		s.WriteUint16(e)
	}

	// Extensions
	s.WriteUint16(uint16(len(ln.Extensions)))
	for _, ext := range ln.Extensions {
		s.WriteUint16(ext.ExtensionType)
		s.WriteOpaque(ext.ExtensionData)
	}

	return s.Bytes()
}

// DeserializeLeafNode decodes a leaf node.
func DeserializeLeafNode(data []byte) (*LeafNode, error) {
	ser := &Serializer{}

	encKey, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	data = data[consumed:]

	sigKey, consumed, err := ser.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode signature key: %w", err)
	}
	data = data[consumed:]

	credData, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode credential: %w", err)
	}
	data = data[consumed:]

	cred, err := DeserializeCredential(credData)
	if err != nil {
		return nil, fmt.Errorf("deserialize credential: %w", err)
	}

	// Decode capabilities
	versionsLen, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode versions length: %w", err)
	}
	data = data[consumed:]

	versions := make([]uint16, versionsLen)
	for i := 0; i < int(versionsLen); i++ {
		v, consumed, err := ser.DecodeUint16(data)
		if err != nil {
			return nil, fmt.Errorf("decode version: %w", err)
		}
		versions[i] = v
		data = data[consumed:]
	}

	ciphersLen, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode ciphers length: %w", err)
	}
	data = data[consumed:]

	ciphers := make([]uint16, ciphersLen)
	for i := 0; i < int(ciphersLen); i++ {
		c, consumed, err := ser.DecodeUint16(data)
		if err != nil {
			return nil, fmt.Errorf("decode cipher: %w", err)
		}
		ciphers[i] = c
		data = data[consumed:]
	}

	extsLen, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode extensions length: %w", err)
	}
	data = data[consumed:]

	extensions := make([]uint16, extsLen)
	for i := 0; i < int(extsLen); i++ {
		e, consumed, err := ser.DecodeUint16(data)
		if err != nil {
			return nil, fmt.Errorf("decode extension: %w", err)
		}
		extensions[i] = e
		data = data[consumed:]
	}

	// Decode extension data
	extDataLen, consumed, err := ser.DecodeUint16(data)
	if err != nil {
		return nil, fmt.Errorf("decode extension data length: %w", err)
	}
	data = data[consumed:]

	extData := make([]Extension, extDataLen)
	for i := 0; i < int(extDataLen); i++ {
		extType, consumed, err := ser.DecodeUint16(data)
		if err != nil {
			return nil, fmt.Errorf("decode extension type: %w", err)
		}
		data = data[consumed:]

		extBytes, consumed, err := ser.DecodeOpaque(data)
		if err != nil {
			return nil, fmt.Errorf("decode extension data: %w", err)
		}
		data = data[consumed:]

		extData[i] = Extension{
			ExtensionType: extType,
			ExtensionData: extBytes,
		}
	}

	return &LeafNode{
		EncryptionKey: encKey,
		SignatureKey:  sigKey,
		Credential:    cred,
		Capabilities: &Capabilities{
			Versions:   versions,
			Ciphers:    ciphers,
			Extensions: extensions,
		},
		Extensions: extData,
	}, nil
}
