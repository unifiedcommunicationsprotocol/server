package mls

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

// Encryption handles AES-GCM encryption per RFC 9420.
type Encryption struct {
	keySchedule *KeySchedule
}

// NewEncryption creates a new encryption handler.
func NewEncryption(ks *KeySchedule) *Encryption {
	return &Encryption{keySchedule: ks}
}

// Encrypt encrypts a plaintext message with AES-128-GCM.
// Returns: IV (12 bytes) || ciphertext || tag (16 bytes).
func (e *Encryption) Encrypt(plaintext []byte) ([]byte, error) {
	cs := SupportedCiphersuite()

	block, err := aes.NewCipher(e.keySchedule.EncryptionSecret[:cs.AEADKeyLength])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// Generate random IV (12 bytes)
	nonce := make([]byte, cs.AEADNonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts a ciphertext with AES-128-GCM.
// Input format: IV (12 bytes) || ciphertext || tag (16 bytes).
func (e *Encryption) Decrypt(ciphertext []byte) ([]byte, error) {
	cs := SupportedCiphersuite()

	if len(ciphertext) < cs.AEADNonceLength+cs.AEADTagLength {
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(ciphertext))
	}

	block, err := aes.NewCipher(e.keySchedule.EncryptionSecret[:cs.AEADKeyLength])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:cs.AEADNonceLength]
	encrypted := ciphertext[cs.AEADNonceLength:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptWithSenderData encrypts with sender data key (for member encryption).
func (e *Encryption) EncryptWithSenderData(plaintext []byte) ([]byte, error) {
	cs := SupportedCiphersuite()

	block, err := aes.NewCipher(e.keySchedule.SenderDataSecret[:cs.AEADKeyLength])
	if err != nil {
		return nil, fmt.Errorf("create sender data cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create sender data GCM: %w", err)
	}

	nonce := make([]byte, cs.AEADNonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptWithSenderData decrypts with sender data key.
func (e *Encryption) DecryptWithSenderData(ciphertext []byte) ([]byte, error) {
	cs := SupportedCiphersuite()

	if len(ciphertext) < cs.AEADNonceLength+cs.AEADTagLength {
		return nil, fmt.Errorf("ciphertext too short: %d bytes", len(ciphertext))
	}

	block, err := aes.NewCipher(e.keySchedule.SenderDataSecret[:cs.AEADKeyLength])
	if err != nil {
		return nil, fmt.Errorf("create sender data cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create sender data GCM: %w", err)
	}

	nonce := ciphertext[:cs.AEADNonceLength]
	encrypted := ciphertext[cs.AEADNonceLength:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt sender data: %w", err)
	}

	return plaintext, nil
}

// EncryptionSecret represents per-epoch encryption material.
type EncryptionSecret struct {
	Epoch           uint64
	EpochSecret     []byte // 32 bytes (SHA-256)
	SenderSecret    []byte // 32 bytes
	EncryptionSecret []byte // 16 bytes (AES-128 key)
}

// DeriveEncryptionSecret derives encryption material from epoch secret.
func DeriveEncryptionSecret(epochSecret []byte, epoch uint64) *EncryptionSecret {
	// Simplified derivation: in real implementation, use HKDF-Expand
	encSecret := make([]byte, 16)
	for i := 0; i < 16 && i < len(epochSecret); i++ {
		encSecret[i] = epochSecret[i]
	}

	senderSecret := make([]byte, 32)
	copy(senderSecret, epochSecret)

	return &EncryptionSecret{
		Epoch:            epoch,
		EpochSecret:      epochSecret,
		SenderSecret:     senderSecret,
		EncryptionSecret: encSecret,
	}
}

// MLSCiphertext represents an encrypted MLS message.
type MLSCiphertext struct {
	GroupID             []byte
	Epoch               uint64
	ContentType         uint8 // 0=application, 1=proposal, 2=commit
	SenderDataEncrypted []byte
	EncryptedContent    []byte
}

// Serialize encodes the ciphertext to bytes.
func (c *MLSCiphertext) Serialize() []byte {
	s := NewBuilder()
	s.WriteOpaque(c.GroupID)
	s.WriteUint64(c.Epoch)
	s.WriteUint8(c.ContentType)
	s.WriteOpaque(c.SenderDataEncrypted)
	s.WriteOpaque(c.EncryptedContent)
	return s.Bytes()
}

// Deserialize decodes a ciphertext from bytes.
func DeserializeMLSCiphertext(data []byte) (*MLSCiphertext, error) {
	s := &Serializer{}

	groupID, consumed, err := s.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode group id: %w", err)
	}
	data = data[consumed:]

	epoch, consumed, err := s.DecodeUint64(data)
	if err != nil {
		return nil, fmt.Errorf("decode epoch: %w", err)
	}
	data = data[consumed:]

	contentType, consumed, err := s.DecodeUint8(data)
	if err != nil {
		return nil, fmt.Errorf("decode content type: %w", err)
	}
	data = data[consumed:]

	senderData, consumed, err := s.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode sender data: %w", err)
	}
	data = data[consumed:]

	content, _, err := s.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode content: %w", err)
	}

	return &MLSCiphertext{
		GroupID:             groupID,
		Epoch:               epoch,
		ContentType:         contentType,
		SenderDataEncrypted: senderData,
		EncryptedContent:    content,
	}, nil
}
