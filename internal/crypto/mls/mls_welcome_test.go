package mls

import (
	"testing"
)

func TestGroupSecrets(t *testing.T) {
	gs := &GroupSecrets{
		Epoch:           1,
		EpochSecret:     make([]byte, 32),
		ConfirmationKey: make([]byte, 32),
	}

	serialized := gs.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized group secrets is empty")
	}

	deserialized, err := DeserializeGroupSecrets(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.Epoch != 1 {
		t.Errorf("epoch: got %d, want 1", deserialized.Epoch)
	}
}

func TestEncryptedGroupSecrets(t *testing.T) {
	egs := &EncryptedGroupSecrets{
		KeyPackageRef: []byte("kp_ref_123"),
		Ciphertext:    make([]byte, 64),
	}

	serialized := egs.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized encrypted secrets is empty")
	}

	deserialized, err := DeserializeEncryptedGroupSecrets(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if string(deserialized.KeyPackageRef) != "kp_ref_123" {
		t.Errorf("key package ref: got %q, want %q", string(deserialized.KeyPackageRef), "kp_ref_123")
	}
}

func TestNewWelcome(t *testing.T) {
	gs := &GroupSecrets{
		Epoch:           1,
		EpochSecret:     make([]byte, 32),
		ConfirmationKey: make([]byte, 32),
	}

	kpRefs := [][]byte{
		[]byte("kp_ref_1"),
		[]byte("kp_ref_2"),
	}

	welcome := NewWelcome(gs, kpRefs)

	if welcome.Version != 0x0001 {
		t.Errorf("version: got %d, want 1", welcome.Version)
	}

	if len(welcome.Secrets) != 2 {
		t.Errorf("secrets count: got %d, want 2", len(welcome.Secrets))
	}
}

func TestWelcomeGroupInfoEncryption(t *testing.T) {
	welcome := &MLSWelcome{
		Version:     0x0001,
		CipherSuite: 0x0001,
	}

	groupInfo := []byte("group_info_data")
	encryptionKey := make([]byte, 32)

	err := welcome.EncryptGroupInfo(groupInfo, encryptionKey)
	if err != nil {
		t.Errorf("encrypt: %v", err)
	}

	if len(welcome.EncryptedGroupInfo) == 0 {
		t.Error("encrypted group info is empty")
	}

	// Decrypt should work with same key
	decrypted, err := welcome.DecryptGroupInfo(encryptionKey)
	if err != nil {
		t.Errorf("decrypt: %v", err)
	}

	if string(decrypted) != "group_info_data" {
		t.Errorf("decrypted: got %q, want %q", string(decrypted), "group_info_data")
	}
}

func TestWelcomeSerialization(t *testing.T) {
	gs := &GroupSecrets{
		Epoch:           1,
		EpochSecret:     make([]byte, 32),
		ConfirmationKey: make([]byte, 32),
	}

	kpRefs := [][]byte{[]byte("kp_ref_1")}
	welcome := NewWelcome(gs, kpRefs)

	serialized := welcome.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized welcome is empty")
	}

	deserialized, err := DeserializeWelcome(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.Version != 0x0001 {
		t.Errorf("version: got %d, want 1", deserialized.Version)
	}

	if len(deserialized.Secrets) != 1 {
		t.Errorf("secrets count: got %d, want 1", len(deserialized.Secrets))
	}
}

func TestDecryptGroupInfoFailure(t *testing.T) {
	welcome := &MLSWelcome{
		EncryptedGroupInfo: []byte("invalid"),
	}

	_, err := welcome.DecryptGroupInfo(make([]byte, 32))
	if err == nil {
		t.Error("expected decryption error for invalid ciphertext")
	}
}
