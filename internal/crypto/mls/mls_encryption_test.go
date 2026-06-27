package mls

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	ks := &KeySchedule{
		Epoch:              0,
		EpochSecret:        make([]byte, 32),
		SenderDataSecret:   make([]byte, 32),
		EncryptionSecret:   make([]byte, 16),
		ExporterSecret:     make([]byte, 32),
	}

	enc := NewEncryption(ks)

	plaintext := []byte("hello world")
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Errorf("encrypt: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("ciphertext is empty")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Errorf("decrypt: %v", err)
	}

	if string(decrypted) != "hello world" {
		t.Errorf("decrypted: got %q, want %q", string(decrypted), "hello world")
	}
}

func TestEncryptSenderData(t *testing.T) {
	ks := &KeySchedule{
		Epoch:              0,
		EpochSecret:        make([]byte, 32),
		SenderDataSecret:   make([]byte, 32),
		EncryptionSecret:   make([]byte, 16),
	}

	enc := NewEncryption(ks)

	plaintext := []byte("sender data")
	ciphertext, err := enc.EncryptWithSenderData(plaintext)
	if err != nil {
		t.Errorf("encrypt sender data: %v", err)
	}

	decrypted, err := enc.DecryptWithSenderData(ciphertext)
	if err != nil {
		t.Errorf("decrypt sender data: %v", err)
	}

	if string(decrypted) != "sender data" {
		t.Errorf("decrypted: got %q, want %q", string(decrypted), "sender data")
	}
}

func TestDeriveEncryptionSecret(t *testing.T) {
	epochSecret := make([]byte, 32)
	for i := 0; i < 32; i++ {
		epochSecret[i] = byte(i)
	}

	es := DeriveEncryptionSecret(epochSecret, 0)

	if es.Epoch != 0 {
		t.Errorf("epoch: got %d, want 0", es.Epoch)
	}

	if len(es.EncryptionSecret) != 16 {
		t.Errorf("encryption secret length: got %d, want 16", len(es.EncryptionSecret))
	}

	if len(es.SenderSecret) != 32 {
		t.Errorf("sender secret length: got %d, want 32", len(es.SenderSecret))
	}
}

func TestMLSCiphertextSerialization(t *testing.T) {
	ct := &MLSCiphertext{
		GroupID:             []byte("group_123"),
		Epoch:               5,
		ContentType:         0,
		SenderDataEncrypted: []byte("encrypted_sender"),
		EncryptedContent:    []byte("encrypted_content"),
	}

	serialized := ct.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized ciphertext is empty")
	}

	deserialized, err := DeserializeMLSCiphertext(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if string(deserialized.GroupID) != "group_123" {
		t.Errorf("group id: got %q, want %q", string(deserialized.GroupID), "group_123")
	}

	if deserialized.Epoch != 5 {
		t.Errorf("epoch: got %d, want 5", deserialized.Epoch)
	}

	if deserialized.ContentType != 0 {
		t.Errorf("content type: got %d, want 0", deserialized.ContentType)
	}
}

func TestDecryptionFailure(t *testing.T) {
	ks := &KeySchedule{
		Epoch:            0,
		EpochSecret:      make([]byte, 32),
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: make([]byte, 16),
	}

	enc := NewEncryption(ks)

	// Try to decrypt invalid ciphertext
	_, err := enc.Decrypt([]byte("invalid"))
	if err == nil {
		t.Error("expected decryption error for invalid ciphertext")
	}
}

func TestEncryptionKeyRotation(t *testing.T) {
	ks1 := &KeySchedule{
		Epoch:            0,
		EpochSecret:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	ks2 := &KeySchedule{
		Epoch:            1,
		EpochSecret:      []byte{17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: []byte{17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}

	enc1 := NewEncryption(ks1)
	enc2 := NewEncryption(ks2)

	plaintext := []byte("secret message")
	ciphertext1, _ := enc1.Encrypt(plaintext)

	// Different key schedule should fail to decrypt
	_, err := enc2.Decrypt(ciphertext1)
	if err == nil {
		t.Error("different epoch decryption should fail")
	}
}
