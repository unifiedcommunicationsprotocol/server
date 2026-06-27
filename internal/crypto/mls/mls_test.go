package mls

import (
	"testing"
)

func TestSerialization(t *testing.T) {
	s := &Serializer{}

	// Test uint8
	encoded := s.EncodeUint8(42)
	decoded, _, _ := s.DecodeUint8(encoded)
	if decoded != 42 {
		t.Errorf("uint8 round-trip: got %d, want 42", decoded)
	}

	// Test uint16
	encoded = s.EncodeUint16(1000)
	decoded16, _, _ := s.DecodeUint16(encoded)
	if decoded16 != 1000 {
		t.Errorf("uint16 round-trip: got %d, want 1000", decoded16)
	}

	// Test uint32
	encoded = s.EncodeUint32(100000)
	decoded32, _, _ := s.DecodeUint32(encoded)
	if decoded32 != 100000 {
		t.Errorf("uint32 round-trip: got %d, want 100000", decoded32)
	}

	// Test bytes
	data := []byte("hello world")
	encoded = s.EncodeBytes(data)
	decoded_bytes, _, _ := s.DecodeBytes(encoded)
	if string(decoded_bytes) != "hello world" {
		t.Errorf("bytes round-trip: got %q, want %q", string(decoded_bytes), "hello world")
	}
}

func TestBuilder(t *testing.T) {
	b := NewBuilder()

	b.WriteUint8(1)
	b.WriteUint16(256)
	b.WriteBytes([]byte("test"))

	data := b.Bytes()
	if len(data) == 0 {
		t.Error("Builder produced empty data")
	}
}

func TestCiphersuite(t *testing.T) {
	cs := SupportedCiphersuite()

	if cs.NameID != 0x0001 {
		t.Errorf("Ciphersuite ID: got %d, want 1", cs.NameID)
	}

	if cs.HashLength != 32 {
		t.Errorf("Hash length: got %d, want 32", cs.HashLength)
	}

	if cs.AEADKeyLength != 16 {
		t.Errorf("AEAD key length: got %d, want 16", cs.AEADKeyLength)
	}
}

func TestGroupCreation(t *testing.T) {
	threadID := "thread_123"
	members := []string{"alice@example.com", "bob@example.com"}

	group := NewGroup(threadID, members)

	if group.ThreadID != threadID {
		t.Errorf("ThreadID: got %q, want %q", group.ThreadID, threadID)
	}

	if len(group.Members) != 2 {
		t.Errorf("Members: got %d, want 2", len(group.Members))
	}

	if group.Epoch != 0 {
		t.Errorf("Initial epoch: got %d, want 0", group.Epoch)
	}
}

func TestGroupMembership(t *testing.T) {
	group := NewGroup("thread_123", []string{"alice@example.com"})

	// Add member
	group.AddMember("bob@example.com")
	if len(group.Members) != 2 {
		t.Errorf("After add: got %d members, want 2", len(group.Members))
	}

	if group.Epoch != 1 {
		t.Errorf("Epoch after add: got %d, want 1", group.Epoch)
	}

	// Remove member
	group.RemoveMember("alice@example.com")
	if len(group.Members) != 1 {
		t.Errorf("After remove: got %d members, want 1", len(group.Members))
	}

	if group.Epoch != 2 {
		t.Errorf("Epoch after remove: got %d, want 2", group.Epoch)
	}
}

func TestKeyScheduleExporter(t *testing.T) {
	ks := &KeySchedule{
		Epoch:          0,
		EpochSecret:    []byte("epoch_secret_test"),
		ExporterSecret: []byte("exporter_secret"),
	}

	label := "server_processing"
	context := []byte("")
	length := uint16(32)

	derived := ks.KeyingMaterialExporter(label, context, length)

	if len(derived) != int(length) {
		t.Errorf("Derived key length: got %d, want %d", len(derived), length)
	}

	// Deterministic
	derived2 := ks.KeyingMaterialExporter(label, context, length)
	if string(derived) != string(derived2) {
		t.Error("Key derivation not deterministic")
	}
}

func TestGroupIDDerivation(t *testing.T) {
	threadID := "01J3K..."
	groupID := deriveGroupID(threadID)

	if len(groupID) != 32 {
		t.Errorf("Group ID length: got %d, want 32 (SHA-256)", len(groupID))
	}

	// Same thread ID should produce same group ID
	groupID2 := deriveGroupID(threadID)
	if string(groupID) != string(groupID2) {
		t.Error("Group ID derivation not deterministic")
	}

	// Different thread ID should produce different group ID
	groupID3 := deriveGroupID("different_thread")
	if string(groupID) == string(groupID3) {
		t.Error("Different thread IDs should produce different group IDs")
	}
}
