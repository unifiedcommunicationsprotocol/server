package crypto

import (
	"testing"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

func TestCreateGroup(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	members := []string{"alice@example.com", "bob@example.com"}

	group, err := m.CreateGroup(threadID, members)
	if err != nil {
		t.Fatalf("CreateGroup error: %v", err)
	}

	if group.ThreadID != threadID {
		t.Errorf("ThreadID mismatch: got %q, want %q", group.ThreadID, threadID)
	}

	if len(group.Members) != 2 {
		t.Errorf("Members length: got %d, want 2", len(group.Members))
	}

	if group.Epoch != 0 {
		t.Errorf("Epoch: got %d, want 0", group.Epoch)
	}

	if len(group.EncryptKey) != 32 {
		t.Errorf("EncryptKey length: got %d, want 32", len(group.EncryptKey))
	}
}

func TestCreateGroupEmptyMembers(t *testing.T) {
	m := New()

	_, err := m.CreateGroup("01J3K...", []string{})
	if err == nil {
		t.Error("CreateGroup should fail with empty members")
	}
}

func TestGetGroup(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	members := []string{"alice@example.com"}

	group, _ := m.CreateGroup(threadID, members)

	retrieved, err := m.GetGroup(group.ID)
	if err != nil {
		t.Fatalf("GetGroup error: %v", err)
	}

	if retrieved.ThreadID != group.ThreadID {
		t.Error("Retrieved group mismatch")
	}
}

func TestGetGroupNotFound(t *testing.T) {
	m := New()

	_, err := m.GetGroup([]byte("nonexistent"))
	if err == nil {
		t.Error("GetGroup should fail for nonexistent group")
	}
}

func TestEncryptDecryptMessage(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com"})

	plaintext := []byte(`{"id":"01J3K","from":"alice@example.com","subject":"Hello"}`)

	// Encrypt
	ciphertext, err := m.EncryptMessage(group.ID, plaintext)
	if err != nil {
		t.Fatalf("EncryptMessage error: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("Ciphertext is empty")
	}

	// Plaintext and ciphertext should be different
	if string(ciphertext) == string(plaintext) {
		t.Error("Ciphertext equals plaintext")
	}

	// Decrypt
	decrypted, err := m.DecryptMessage(group.ID, ciphertext)
	if err != nil {
		t.Fatalf("DecryptMessage error: %v", err)
	}

	// Should match original
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted mismatch:\ngot:  %s\nwant: %s", string(decrypted), string(plaintext))
	}
}

func TestDecryptTamperedMessage(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com"})

	plaintext := []byte("Hello, World!")

	ciphertext, _ := m.EncryptMessage(group.ID, plaintext)

	// Tamper with ciphertext
	if len(ciphertext) > 20 {
		ciphertext[20] ^= 0xFF
	}

	// Should fail to decrypt
	_, err := m.DecryptMessage(group.ID, ciphertext)
	if err == nil {
		t.Error("DecryptMessage should fail with tampered ciphertext")
	}
}

func TestAddMember(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com"})

	initialEpoch := group.Epoch

	// Add member
	err := m.AddMember(group.ID, "charlie@example.com")
	if err != nil {
		t.Fatalf("AddMember error: %v", err)
	}

	if len(group.Members) != 2 {
		t.Errorf("Members length: got %d, want 2", len(group.Members))
	}

	if group.Epoch != initialEpoch+1 {
		t.Errorf("Epoch: got %d, want %d", group.Epoch, initialEpoch+1)
	}
}

func TestAddDuplicateMember(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com"})

	// Try to add existing member
	err := m.AddMember(group.ID, "alice@example.com")
	if err == nil {
		t.Error("AddMember should fail for existing member")
	}
}

func TestRemoveMember(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com", "bob@example.com"})

	initialEpoch := group.Epoch

	// Remove member
	err := m.RemoveMember(group.ID, "bob@example.com")
	if err != nil {
		t.Fatalf("RemoveMember error: %v", err)
	}

	if len(group.Members) != 1 {
		t.Errorf("Members length: got %d, want 1", len(group.Members))
	}

	if group.Epoch != initialEpoch+1 {
		t.Errorf("Epoch: got %d, want %d", group.Epoch, initialEpoch+1)
	}
}

func TestRemoveNonexistentMember(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com"})

	// Try to remove non-existent member
	err := m.RemoveMember(group.ID, "charlie@example.com")
	if err == nil {
		t.Error("RemoveMember should fail for non-existent member")
	}
}

func TestAdvanceEpoch(t *testing.T) {
	m := New()

	group, _ := m.CreateGroup("01J3K...", []string{"alice@example.com"})

	initialEpoch := group.Epoch
	oldKey := group.EncryptKey

	// Advance epoch (e.g., signing key rotation)
	err := m.AdvanceEpoch(group.ID)
	if err != nil {
		t.Fatalf("AdvanceEpoch error: %v", err)
	}

	if group.Epoch != initialEpoch+1 {
		t.Errorf("Epoch: got %d, want %d", group.Epoch, initialEpoch+1)
	}

	// Key should be different
	if string(group.EncryptKey) == string(oldKey) {
		t.Error("EncryptKey should change on epoch advance")
	}
}

func TestDeriveBCCGroupID(t *testing.T) {
	threadID := models.ULID("01J3K...")
	recipient := "bob@example.com"

	groupID := DeriveBCCGroupID(threadID, recipient)

	if len(groupID) == 0 {
		t.Error("DeriveBCCGroupID returned empty")
	}

	// Consistent derivation
	groupID2 := DeriveBCCGroupID(threadID, recipient)
	if string(groupID) != string(groupID2) {
		t.Error("DeriveBCCGroupID not deterministic")
	}

	// Different recipient -> different ID
	groupID3 := DeriveBCCGroupID(threadID, "alice@example.com")
	if string(groupID) == string(groupID3) {
		t.Error("DeriveBCCGroupID should differ for different recipients")
	}
}

func TestCreateKeyPackage(t *testing.T) {
	m := New()

	groupID := []byte("test_group_id")

	kp, err := m.CreateKeyPackage(groupID)
	if err != nil {
		t.Fatalf("CreateKeyPackage error: %v", err)
	}

	if len(kp.InitKey) != 32 {
		t.Errorf("InitKey length: got %d, want 32", len(kp.InitKey))
	}

	if len(kp.SigningKey) != 32 {
		t.Errorf("SigningKey length: got %d, want 32", len(kp.SigningKey))
	}
}

func TestEncodeDecodeKeyPackage(t *testing.T) {
	m := New()

	groupID := []byte("test_group_id")
	kp, _ := m.CreateKeyPackage(groupID)

	// Encode
	encoded := EncodeKeyPackage(kp)
	if encoded == "" {
		t.Error("EncodeKeyPackage returned empty")
	}

	// Decode
	decoded, err := DecodeKeyPackage(encoded)
	if err != nil {
		t.Fatalf("DecodeKeyPackage error: %v", err)
	}

	// Verify round-trip
	if len(decoded.InitKey) != 32 || len(decoded.SigningKey) != 32 {
		t.Error("Decoded key package mismatch")
	}
}
