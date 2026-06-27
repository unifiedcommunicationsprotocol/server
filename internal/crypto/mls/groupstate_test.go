package mls

import (
	"testing"
)

func TestCreateGroup(t *testing.T) {
	gsm := NewGroupStateManager()

	members := []string{"alice@example.com", "bob@example.com"}
	managed, err := gsm.CreateGroup("thread_123", members)

	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if managed.Group.ThreadID != "thread_123" {
		t.Errorf("thread_id: got %q, want %q", managed.Group.ThreadID, "thread_123")
	}

	if len(managed.Members) != 2 {
		t.Errorf("member count: got %d, want 2", len(managed.Members))
	}

	if managed.State != StateCreated {
		t.Errorf("state: got %d, want %d", managed.State, StateCreated)
	}
}

func TestGetGroup(t *testing.T) {
	gsm := NewGroupStateManager()

	gsm.CreateGroup("thread_456", []string{"alice@example.com"})

	managed, err := gsm.GetGroup("thread_456")
	if err != nil {
		t.Fatalf("get group: %v", err)
	}

	if managed == nil {
		t.Error("managed group is nil")
	}
}

func TestProposeAdd(t *testing.T) {
	gsm := NewGroupStateManager()
	gsm.CreateGroup("thread_789", []string{"alice@example.com"})

	propRef, err := gsm.ProposeAdd("thread_789", "bob@example.com")
	if err != nil {
		t.Fatalf("propose add: %v", err)
	}

	if propRef == nil {
		t.Error("proposal ref is nil")
	}

	if propRef.Ref == nil {
		t.Error("ref is nil")
	}
}

func TestProposeRemove(t *testing.T) {
	gsm := NewGroupStateManager()
	gsm.CreateGroup("thread_abc", []string{"alice@example.com", "bob@example.com"})

	propRef, err := gsm.ProposeRemove("thread_abc", "alice@example.com")
	if err != nil {
		t.Fatalf("propose remove: %v", err)
	}

	if propRef == nil {
		t.Error("proposal ref is nil")
	}
}

func TestCommitProposals(t *testing.T) {
	gsm := NewGroupStateManager()
	gsm.CreateGroup("thread_def", []string{"alice@example.com"})

	// Propose adding a member
	gsm.ProposeAdd("thread_def", "bob@example.com")

	// Commit
	commit, err := gsm.CommitProposals("thread_def", "alice@example.com")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	if commit == nil {
		t.Error("commit is nil")
	}

	// Check epoch advanced
	epoch, _ := gsm.GetEpoch("thread_def")
	if epoch != 1 {
		t.Errorf("epoch: got %d, want 1", epoch)
	}
}

func TestEncryptDecryptMessage(t *testing.T) {
	gsm := NewGroupStateManager()
	gsm.CreateGroup("thread_ghi", []string{"alice@example.com"})

	plaintext := []byte("secret message")

	// Encrypt
	ciphertext, err := gsm.EncryptMessage("thread_ghi", plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("ciphertext is empty")
	}

	// Decrypt
	decrypted, err := gsm.DecryptMessage("thread_ghi", ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != "secret message" {
		t.Errorf("decrypted: got %q, want %q", string(decrypted), "secret message")
	}
}

func TestGetMembers(t *testing.T) {
	gsm := NewGroupStateManager()
	gsm.CreateGroup("thread_jkl", []string{"alice@example.com", "bob@example.com"})

	members, err := gsm.GetMembers("thread_jkl")
	if err != nil {
		t.Fatalf("get members: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("member count: got %d, want 2", len(members))
	}
}
