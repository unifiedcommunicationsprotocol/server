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

	if group.ManagedGroup == nil {
		t.Error("ManagedGroup is nil")
	}

	if len(group.ManagedGroup.Group.Members) != 2 {
		t.Errorf("Members length: got %d, want 2", len(group.ManagedGroup.Group.Members))
	}

	if group.ManagedGroup.Group.Epoch != 0 {
		t.Errorf("Epoch: got %d, want 0", group.ManagedGroup.Group.Epoch)
	}
}

func TestCreateGroupEmptyMembers(t *testing.T) {
	m := New()

	_, err := m.CreateGroup(models.ULID("01J3K..."), []string{})
	if err == nil {
		t.Error("CreateGroup should fail with empty members")
	}
}

func TestGetGroup(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	members := []string{"alice@example.com"}

	group, _ := m.CreateGroup(threadID, members)

	retrieved, err := m.GetGroup(threadID)
	if err != nil {
		t.Fatalf("GetGroup error: %v", err)
	}

	if retrieved.ThreadID != group.ThreadID {
		t.Error("Retrieved group mismatch")
	}
}

func TestGetGroupNotFound(t *testing.T) {
	m := New()

	_, err := m.GetGroup(models.ULID("nonexistent"))
	if err == nil {
		t.Error("GetGroup should fail for nonexistent group")
	}
}

func TestEncryptDecryptMessage(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	_, err := m.CreateGroup(threadID, []string{"alice@example.com"})
	if err != nil {
		t.Fatalf("CreateGroup error: %v", err)
	}

	plaintext := []byte(`{"id":"01J3K","from":"alice@example.com","subject":"Hello"}`)

	// Encrypt
	ciphertext, err := m.EncryptMessage(threadID, plaintext)
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
	decrypted, err := m.DecryptMessage(threadID, ciphertext)
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

	threadID := models.ULID("01J3K...")
	m.CreateGroup(threadID, []string{"alice@example.com"})

	plaintext := []byte("Hello, World!")

	ciphertext, _ := m.EncryptMessage(threadID, plaintext)

	// Tamper with ciphertext
	if len(ciphertext) > 20 {
		ciphertext[20] ^= 0xFF
	}

	// Should fail to decrypt
	_, err := m.DecryptMessage(threadID, ciphertext)
	if err == nil {
		t.Error("DecryptMessage should fail with tampered ciphertext")
	}
}

func TestAddMember(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	m.CreateGroup(threadID, []string{"alice@example.com"})

	members, _ := m.GetGroupMembers(threadID)
	initialLen := len(members)

	// Add member
	err := m.AddMember(threadID, "charlie@example.com")
	if err != nil {
		t.Fatalf("AddMember error: %v", err)
	}

	members, _ = m.GetGroupMembers(threadID)
	if len(members) != initialLen+1 {
		t.Errorf("Members length: got %d, want %d", len(members), initialLen+1)
	}
}

func TestAddDuplicateMember(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	m.CreateGroup(threadID, []string{"alice@example.com"})

	// Try to add existing member
	err := m.AddMember(threadID, "alice@example.com")
	if err == nil {
		t.Error("AddMember should fail for existing member")
	}
}

func TestRemoveMember(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	m.CreateGroup(threadID, []string{"alice@example.com", "bob@example.com"})

	members, _ := m.GetGroupMembers(threadID)
	initialLen := len(members)

	// Remove member
	err := m.RemoveMember(threadID, "bob@example.com")
	if err != nil {
		t.Fatalf("RemoveMember error: %v", err)
	}

	members, _ = m.GetGroupMembers(threadID)
	if len(members) != initialLen-1 {
		t.Errorf("Members length: got %d, want %d", len(members), initialLen-1)
	}
}

func TestAdvanceEpoch(t *testing.T) {
	m := New()

	threadID := models.ULID("01J3K...")
	m.CreateGroup(threadID, []string{"alice@example.com"})

	epoch, _ := m.GetGroupEpoch(threadID)
	initialEpoch := epoch

	// Advance epoch (e.g., signing key rotation)
	err := m.AdvanceEpoch(threadID)
	if err != nil {
		t.Fatalf("AdvanceEpoch error: %v", err)
	}

	epoch, _ = m.GetGroupEpoch(threadID)
	if epoch != initialEpoch+1 {
		t.Errorf("Epoch: got %d, want %d", epoch, initialEpoch+1)
	}
}
