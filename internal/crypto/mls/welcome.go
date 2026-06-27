package mls

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

// GroupSecrets contains secrets needed to initialize group state.
type GroupSecrets struct {
	Epoch           uint64
	EpochSecret     []byte
	ConfirmationKey []byte
}

// Serialize encodes group secrets.
func (gs *GroupSecrets) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint64(gs.Epoch)
	s.WriteOpaque(gs.EpochSecret)
	s.WriteOpaque(gs.ConfirmationKey)
	return s.Bytes()
}

// DeserializeGroupSecrets decodes group secrets.
func DeserializeGroupSecrets(data []byte) (*GroupSecrets, error) {
	ser := &Serializer{}

	epoch, consumed, err := ser.DecodeUint64(data)
	if err != nil {
		return nil, fmt.Errorf("decode epoch: %w", err)
	}
	data = data[consumed:]

	epochSecret, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode epoch secret: %w", err)
	}
	data = data[consumed:]

	confirmationKey, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode confirmation key: %w", err)
	}

	return &GroupSecrets{
		Epoch:           epoch,
		EpochSecret:     epochSecret,
		ConfirmationKey: confirmationKey,
	}, nil
}

// EncryptedGroupSecrets holds encrypted group secrets for a recipient.
type EncryptedGroupSecrets struct {
	KeyPackageRef []byte
	Ciphertext    []byte // Encrypted secrets
}

// Serialize encodes encrypted secrets.
func (egs *EncryptedGroupSecrets) Serialize() []byte {
	s := NewBuilder()
	s.WriteOpaque(egs.KeyPackageRef)
	s.WriteOpaque(egs.Ciphertext)
	return s.Bytes()
}

// DeserializeEncryptedGroupSecrets decodes encrypted secrets.
func DeserializeEncryptedGroupSecrets(data []byte) (*EncryptedGroupSecrets, error) {
	ser := &Serializer{}

	kpRef, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode key package ref: %w", err)
	}
	data = data[consumed:]

	ciphertext, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	return &EncryptedGroupSecrets{
		KeyPackageRef: kpRef,
		Ciphertext:    ciphertext,
	}, nil
}

// MLSWelcome is sent to new members to initialize group state.
type MLSWelcome struct {
	Version            uint16
	CipherSuite        uint16
	Secrets            []EncryptedGroupSecrets
	EncryptedGroupInfo []byte // Encrypted tree info
}

// NewWelcome creates a welcome message.
func NewWelcome(groupSecrets *GroupSecrets, kpRefs [][]byte) *MLSWelcome {
	secrets := make([]EncryptedGroupSecrets, len(kpRefs))

	// Encrypt secrets for each key package
	for i, kpRef := range kpRefs {
		// Simplified: random ciphertext
		cipher := make([]byte, 64)
		rand.Read(cipher)

		secrets[i] = EncryptedGroupSecrets{
			KeyPackageRef: kpRef,
			Ciphertext:    cipher,
		}
	}

	return &MLSWelcome{
		Version:     0x0001,
		CipherSuite: 0x0001,
		Secrets:     secrets,
	}
}

// EncryptGroupInfo encrypts tree information for the welcome.
func (w *MLSWelcome) EncryptGroupInfo(groupInfo []byte, encryptionKey []byte) error {
	cs := SupportedCiphersuite()

	block, err := aes.NewCipher(encryptionKey[:cs.AEADKeyLength])
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, cs.AEADNonceLength)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	w.EncryptedGroupInfo = gcm.Seal(nonce, nonce, groupInfo, nil)
	return nil
}

// DecryptGroupInfo decrypts tree information from the welcome.
func (w *MLSWelcome) DecryptGroupInfo(encryptionKey []byte) ([]byte, error) {
	cs := SupportedCiphersuite()

	if len(w.EncryptedGroupInfo) < cs.AEADNonceLength+cs.AEADTagLength {
		return nil, fmt.Errorf("encrypted info too short")
	}

	block, err := aes.NewCipher(encryptionKey[:cs.AEADKeyLength])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := w.EncryptedGroupInfo[:cs.AEADNonceLength]
	ciphertext := w.EncryptedGroupInfo[cs.AEADNonceLength:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Serialize encodes the welcome.
func (w *MLSWelcome) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint16(w.Version)
	s.WriteUint16(w.CipherSuite)

	// Secrets
	s.WriteUint32(uint32(len(w.Secrets)))
	for _, secret := range w.Secrets {
		s.WriteOpaque(secret.Serialize())
	}

	// Group info
	s.WriteOpaque(w.EncryptedGroupInfo)

	return s.Bytes()
}

// DeserializeWelcome decodes a welcome.
func DeserializeWelcome(data []byte) (*MLSWelcome, error) {
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

	// Secrets
	secretCount, consumed, err := ser.DecodeUint32(data)
	if err != nil {
		return nil, fmt.Errorf("decode secret count: %w", err)
	}
	data = data[consumed:]

	secrets := make([]EncryptedGroupSecrets, secretCount)
	for i := 0; i < int(secretCount); i++ {
		secretData, consumed, err := ser.DecodeOpaque(data)
		if err != nil {
			return nil, fmt.Errorf("decode secret: %w", err)
		}
		data = data[consumed:]

		secret, err := DeserializeEncryptedGroupSecrets(secretData)
		if err != nil {
			return nil, fmt.Errorf("deserialize secret: %w", err)
		}
		secrets[i] = *secret
	}

	// Group info
	groupInfo, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode group info: %w", err)
	}

	return &MLSWelcome{
		Version:            version,
		CipherSuite:        ciphersuite,
		Secrets:            secrets,
		EncryptedGroupInfo: groupInfo,
	}, nil
}
