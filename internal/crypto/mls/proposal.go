package mls

import (
	"crypto/sha256"
	"fmt"
)

// ProposalType defines the type of MLS proposal.
type ProposalType uint8

const (
	ProposalTypeAdd    ProposalType = 0
	ProposalTypeUpdate ProposalType = 1
	ProposalTypeRemove ProposalType = 2
)

// Proposal is the base interface for all proposal types.
type Proposal interface {
	Type() ProposalType
	Serialize() []byte
}

// AddProposal adds a member to the group.
type AddProposal struct {
	KeyPackage *KeyPackage
}

func (ap *AddProposal) Type() ProposalType {
	return ProposalTypeAdd
}

func (ap *AddProposal) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint8(uint8(ProposalTypeAdd))
	s.WriteOpaque(ap.KeyPackage.Serialize())
	return s.Bytes()
}

// DeserializeAddProposal decodes an Add proposal.
func DeserializeAddProposal(data []byte) (*AddProposal, error) {
	ser := &Serializer{}

	kpData, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode key package: %w", err)
	}

	kp, err := DeserializeKeyPackage(kpData)
	if err != nil {
		return nil, fmt.Errorf("deserialize key package: %w", err)
	}

	return &AddProposal{KeyPackage: kp}, nil
}

// UpdateProposal updates a member's encryption key.
type UpdateProposal struct {
	LeafNodeIndex uint32
	LeafNode      *LeafNode
}

func (up *UpdateProposal) Type() ProposalType {
	return ProposalTypeUpdate
}

func (up *UpdateProposal) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint8(uint8(ProposalTypeUpdate))
	s.WriteUint32(up.LeafNodeIndex)
	s.WriteOpaque(up.LeafNode.Serialize())
	return s.Bytes()
}

// DeserializeUpdateProposal decodes an Update proposal.
func DeserializeUpdateProposal(data []byte) (*UpdateProposal, error) {
	ser := &Serializer{}

	index, consumed, err := ser.DecodeUint32(data)
	if err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}
	data = data[consumed:]

	leafData, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode leaf node: %w", err)
	}

	leaf, err := DeserializeLeafNode(leafData)
	if err != nil {
		return nil, fmt.Errorf("deserialize leaf node: %w", err)
	}

	return &UpdateProposal{
		LeafNodeIndex: index,
		LeafNode:      leaf,
	}, nil
}

// RemoveProposal removes a member from the group.
type RemoveProposal struct {
	LeafNodeIndex uint32
	Member        string // UCP address of member being removed
}

func (rp *RemoveProposal) Type() ProposalType {
	return ProposalTypeRemove
}

func (rp *RemoveProposal) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint8(uint8(ProposalTypeRemove))
	s.WriteUint32(rp.LeafNodeIndex)
	return s.Bytes()
}

// DeserializeRemoveProposal decodes a Remove proposal.
func DeserializeRemoveProposal(data []byte) (*RemoveProposal, error) {
	ser := &Serializer{}

	index, _, err := ser.DecodeUint32(data)
	if err != nil {
		return nil, fmt.Errorf("decode index: %w", err)
	}

	return &RemoveProposal{LeafNodeIndex: index}, nil
}

// ProposalRef is a reference to a proposal (its SHA-256 hash).
type ProposalRef struct {
	Hash []byte
}

// NewProposalRef creates a reference from a proposal.
func NewProposalRef(proposal Proposal) *ProposalRef {
	data := proposal.Serialize()
	h := sha256.Sum256(data)
	return &ProposalRef{Hash: h[:]}
}

// Serialize encodes the reference.
func (pr *ProposalRef) Serialize() []byte {
	s := NewBuilder()
	s.WriteOpaque(pr.Hash)
	return s.Bytes()
}

// DeserializeProposalRef decodes a reference.
func DeserializeProposalRef(data []byte) (*ProposalRef, error) {
	ser := &Serializer{}

	hash, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode hash: %w", err)
	}

	return &ProposalRef{Hash: hash}, nil
}

// MLSProposal is the wire format for a proposal.
type MLSProposal struct {
	ProposalType ProposalType
	Content      []byte // Serialized proposal
}

// NewMLSProposal wraps a proposal for transmission.
func NewMLSProposal(proposal Proposal) *MLSProposal {
	return &MLSProposal{
		ProposalType: proposal.Type(),
		Content:      proposal.Serialize(),
	}
}

// Serialize encodes the MLS proposal.
func (mp *MLSProposal) Serialize() []byte {
	s := NewBuilder()
	s.WriteUint8(uint8(mp.ProposalType))
	s.WriteOpaque(mp.Content)
	return s.Bytes()
}

// DeserializeMLSProposal decodes an MLS proposal.
func DeserializeMLSProposal(data []byte) (*MLSProposal, error) {
	ser := &Serializer{}

	propType, consumed, err := ser.DecodeUint8(data)
	if err != nil {
		return nil, fmt.Errorf("decode proposal type: %w", err)
	}
	data = data[consumed:]

	content, _, err := ser.DecodeOpaque(data)
	if err != nil {
		return nil, fmt.Errorf("decode content: %w", err)
	}

	return &MLSProposal{
		ProposalType: ProposalType(propType),
		Content:      content,
	}, nil
}
