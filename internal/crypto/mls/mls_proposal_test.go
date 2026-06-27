package mls

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestAddProposal(t *testing.T) {
	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	kp := NewKeyPackage("bob@example.com", signingPub, identityPub)
	ap := &AddProposal{KeyPackage: kp}

	if ap.Type() != ProposalTypeAdd {
		t.Errorf("proposal type: got %d, want %d", ap.Type(), ProposalTypeAdd)
	}

	serialized := ap.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized proposal is empty")
	}

	// Skip the proposal type byte in the serialized data
	deserialized, err := DeserializeAddProposal(serialized[1:])
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.KeyPackage == nil {
		t.Error("key package is nil after deserialization")
	}
}

func TestUpdateProposal(t *testing.T) {
	signingKey := make([]byte, 32)
	encryptionKey := make([]byte, 32)
	rand.Read(signingKey)
	rand.Read(encryptionKey)

	signingPub, _, _ := ed25519.GenerateKey(rand.Reader)
	identityPub, _, _ := ed25519.GenerateKey(rand.Reader)

	cred := NewCredential(signingPub, identityPub, "alice@example.com")
	ln := NewLeafNode("alice@example.com", signingKey, encryptionKey, cred)

	up := &UpdateProposal{
		LeafNodeIndex: 5,
		LeafNode:      ln,
	}

	if up.Type() != ProposalTypeUpdate {
		t.Errorf("proposal type: got %d, want %d", up.Type(), ProposalTypeUpdate)
	}

	serialized := up.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized proposal is empty")
	}

	// Skip the proposal type byte
	deserialized, err := DeserializeUpdateProposal(serialized[1:])
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.LeafNodeIndex != 5 {
		t.Errorf("leaf node index: got %d, want 5", deserialized.LeafNodeIndex)
	}
}

func TestRemoveProposal(t *testing.T) {
	rp := &RemoveProposal{LeafNodeIndex: 3}

	if rp.Type() != ProposalTypeRemove {
		t.Errorf("proposal type: got %d, want %d", rp.Type(), ProposalTypeRemove)
	}

	serialized := rp.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized proposal is empty")
	}

	// Skip the proposal type byte
	deserialized, err := DeserializeRemoveProposal(serialized[1:])
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.LeafNodeIndex != 3 {
		t.Errorf("leaf node index: got %d, want 3", deserialized.LeafNodeIndex)
	}
}

func TestProposalRef(t *testing.T) {
	rp := &RemoveProposal{LeafNodeIndex: 2}
	ref := NewProposalRef(rp)

	if len(ref.Hash) != 32 {
		t.Errorf("hash length: got %d, want 32", len(ref.Hash))
	}

	serialized := ref.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized ref is empty")
	}

	deserialized, err := DeserializeProposalRef(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if string(deserialized.Hash) != string(ref.Hash) {
		t.Error("hash mismatch")
	}
}

func TestMLSProposal(t *testing.T) {
	rp := &RemoveProposal{LeafNodeIndex: 1}
	mp := NewMLSProposal(rp)

	if mp.ProposalType != ProposalTypeRemove {
		t.Errorf("proposal type: got %d, want %d", mp.ProposalType, ProposalTypeRemove)
	}

	serialized := mp.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized mls proposal is empty")
	}

	deserialized, err := DeserializeMLSProposal(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if deserialized.ProposalType != ProposalTypeRemove {
		t.Errorf("proposal type: got %d, want %d", deserialized.ProposalType, ProposalTypeRemove)
	}
}
