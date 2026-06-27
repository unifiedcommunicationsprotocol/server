package mls

import (
	"testing"
)

func TestCommitContent(t *testing.T) {
	rp := &RemoveProposal{LeafNodeIndex: 1}
	ref := NewProposalRef(rp)

	cc := &CommitContent{
		Proposals: []ProposalRef{*ref},
		Updates:   []ProposalRef{},
		Removes:   []ProposalRef{*ref},
	}

	serialized := cc.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized content is empty")
	}

	deserialized, err := DeserializeCommitContent(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if len(deserialized.Proposals) != 1 {
		t.Errorf("proposals count: got %d, want 1", len(deserialized.Proposals))
	}

	if len(deserialized.Removes) != 1 {
		t.Errorf("removes count: got %d, want 1", len(deserialized.Removes))
	}
}

func TestMLSCommit(t *testing.T) {
	rp := &RemoveProposal{LeafNodeIndex: 0}
	ref := NewProposalRef(rp)

	cc := &CommitContent{
		Proposals: []ProposalRef{*ref},
		Updates:   []ProposalRef{},
		Removes:   []ProposalRef{},
	}

	commit := &MLSCommit{
		GroupID:      []byte("group_123"),
		Epoch:        1,
		Content:      cc,
		Confirmation: make([]byte, 32),
		Signature:    make([]byte, 64),
	}

	serialized := commit.Serialize()
	if len(serialized) == 0 {
		t.Error("serialized commit is empty")
	}

	deserialized, err := DeserializeMLSCommit(serialized)
	if err != nil {
		t.Errorf("deserialize: %v", err)
	}

	if string(deserialized.GroupID) != "group_123" {
		t.Errorf("group id: got %q, want %q", string(deserialized.GroupID), "group_123")
	}

	if deserialized.Epoch != 1 {
		t.Errorf("epoch: got %d, want 1", deserialized.Epoch)
	}
}

func TestComputeConfirmation(t *testing.T) {
	rp := &RemoveProposal{LeafNodeIndex: 0}
	ref := NewProposalRef(rp)

	cc := &CommitContent{
		Proposals: []ProposalRef{*ref},
		Updates:   []ProposalRef{},
		Removes:   []ProposalRef{},
	}

	commit := &MLSCommit{
		GroupID: []byte("group_123"),
		Epoch:   1,
		Content: cc,
	}

	confirmationKey := make([]byte, 32)
	confirmation := commit.ComputeConfirmation(confirmationKey)

	if len(confirmation) != 32 {
		t.Errorf("confirmation length: got %d, want 32", len(confirmation))
	}

	// Deterministic
	confirmation2 := commit.ComputeConfirmation(confirmationKey)
	if string(confirmation) != string(confirmation2) {
		t.Error("confirmation not deterministic")
	}
}

func TestApplyCommit(t *testing.T) {
	members := []string{"alice@example.com", "bob@example.com"}
	group := NewGroup("thread_123", members)

	rp := &RemoveProposal{LeafNodeIndex: 1}
	ref := NewProposalRef(rp)

	cc := &CommitContent{
		Proposals: []ProposalRef{},
		Updates:   []ProposalRef{},
		Removes:   []ProposalRef{*ref},
	}

	commit := &MLSCommit{
		GroupID:      group.ID,
		Epoch:        0,
		Content:      cc,
		Confirmation: make([]byte, 32),
		Signature:    make([]byte, 64),
	}

	initialEpoch := group.Epoch
	err := group.ApplyCommit(commit)
	if err != nil {
		t.Errorf("apply commit: %v", err)
	}

	if group.Epoch != initialEpoch+1 {
		t.Errorf("epoch: got %d, want %d", group.Epoch, initialEpoch+1)
	}
}
