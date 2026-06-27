package mls

import (
	"crypto/sha256"
	"fmt"
)

// CommitContent holds the proposals being committed.
type CommitContent struct {
	Proposals []ProposalRef
	Updates   []ProposalRef // Updates to existing proposals
	Removes   []ProposalRef // Removals
}

// Serialize encodes the commit content.
func (cc *CommitContent) Serialize() []byte {
	s := NewBuilder()

	// Proposals
	s.WriteUint32(uint32(len(cc.Proposals)))
	for _, p := range cc.Proposals {
		s.WriteOpaque(p.Serialize())
	}

	// Updates
	s.WriteUint32(uint32(len(cc.Updates)))
	for _, u := range cc.Updates {
		s.WriteOpaque(u.Serialize())
	}

	// Removes
	s.WriteUint32(uint32(len(cc.Removes)))
	for _, r := range cc.Removes {
		s.WriteOpaque(r.Serialize())
	}

	return s.Bytes()
}

// DeserializeCommitContent decodes commit content.
func DeserializeCommitContent(data []byte) (*CommitContent, error) {
	ser := &Serializer{}

	// Decode proposals
	propCount, consumed, err := ser.DecodeUint32(data)
	if err != nil {
		return nil, fmt.Errorf("decode proposal count: %w", err)
	}
	data = data[consumed:]

	proposals := make([]ProposalRef, propCount)
	for i := 0; i < int(propCount); i++ {
		ref, consumed, err := ser.DecodeOpaque(data)
		if err != nil {
			return nil, fmt.Errorf("decode proposal ref: %w", err)
		}
		data = data[consumed:]

		propRef, err := DeserializeProposalRef(ref)
		if err != nil {
			return nil, fmt.Errorf("deserialize proposal ref: %w", err)
		}
		proposals[i] = *propRef
	}

	// Decode updates
	updateCount, consumed, err := ser.DecodeUint32(data)
	if err != nil {
		return nil, fmt.Errorf("decode update count: %w", err)
	}
	data = data[consumed:]

	updates := make([]ProposalRef, updateCount)
	for i := 0; i < int(updateCount); i++ {
		ref, consumed, err := ser.DecodeOpaque(data)
		if err != nil {
			return nil, fmt.Errorf("decode update ref: %w", err)
		}
		data = data[consumed:]

		updateRef, err := DeserializeProposalRef(ref)
		if err != nil {
			return nil, fmt.Errorf("deserialize update ref: %w", err)
		}
		updates[i] = *updateRef
	}

	// Decode removes
	removeCount, consumed, err := ser.DecodeUint32(data)
	if err != nil {
		return nil, fmt.Errorf("decode remove count: %w", err)
	}
	data = data[consumed:]

	removes := make([]ProposalRef, removeCount)
	for i := 0; i < int(removeCount); i++ {
		ref, consumed, err := ser.DecodeOpaque(data)
		if err != nil {
			return nil, fmt.Errorf("decode remove ref: %w", err)
		}
		data = data[consumed:]

		removeRef, err := DeserializeProposalRef(ref)
		if err != nil {
			return nil, fmt.Errorf("deserialize remove ref: %w", err)
		}
		removes[i] = *removeRef
	}

	return &CommitContent{
		Proposals: proposals,
		Updates:   updates,
		Removes:   removes,
	}, nil
}

// MLSCommit represents a handshake message committing proposals.
type MLSCommit struct {
	GroupID      []byte
	Epoch        uint64
	Content      *CommitContent
	Confirmation []byte // HMAC confirmation tag
	Signature    []byte
}

// ComputeConfirmation computes the confirmation tag per RFC 9420.
func (mc *MLSCommit) ComputeConfirmation(confirmationKey []byte) []byte {
	// Simplified: HMAC-SHA256(confirmation_key, tree_hash)
	// In real implementation: per RFC 9420 §8.5
	h := sha256.New()
	h.Write(confirmationKey)
	h.Write(mc.Content.Serialize())
	return h.Sum(nil)
}

// Serialize encodes the commit.
func (mc *MLSCommit) Serialize() []byte {
	s := NewBuilder()
	s.WriteOpaque(mc.GroupID)
	s.WriteUint64(mc.Epoch)
	s.WriteOpaque(mc.Content.Serialize())
	s.WriteOpaque(mc.Confirmation)
	s.WriteOpaque(mc.Signature)
	return s.Bytes()
}

// DeserializeMLSCommit decodes a commit.
func DeserializeMLSCommit(data []byte) (*MLSCommit, error) {
	ser := &Serializer{}

	groupID, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode group id: %w", err)
	}
	data = data[consumed:]

	epoch, consumed, err := ser.DecodeUint64(data)
	if err != nil {
		return nil, fmt.Errorf("decode epoch: %w", err)
	}
	data = data[consumed:]

	contentData, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode content: %w", err)
	}
	data = data[consumed:]

	content, err := DeserializeCommitContent(contentData)
	if err != nil {
		return nil, fmt.Errorf("deserialize content: %w", err)
	}

	confirmation, consumed, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode confirmation: %w", err)
	}
	data = data[consumed:]

	signature, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	return &MLSCommit{
		GroupID:      groupID,
		Epoch:        epoch,
		Content:      content,
		Confirmation: confirmation,
		Signature:    signature,
	}, nil
}

// ApplyCommitToGroup applies a commit to group state.
func (g *Group) ApplyCommit(commit *MLSCommit) error {
	if len(commit.GroupID) > 0 && string(commit.GroupID) != string(g.ID) {
		return fmt.Errorf("group id mismatch")
	}

	// Process proposals in order
	for _, propRef := range commit.Content.Proposals {
		// In real implementation: look up proposal by hash and apply
		_ = propRef
	}

	for _, updateRef := range commit.Content.Updates {
		// Process updates
		_ = updateRef
	}

	for _, removeRef := range commit.Content.Removes {
		// Process removals
		_ = removeRef
	}

	// Advance epoch
	g.AdvanceEpoch()
	return nil
}
