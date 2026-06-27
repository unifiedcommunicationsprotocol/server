package mls

import (
	"fmt"
	"sync"
	"time"
)

// GroupStateManager manages the MLS group state machine for a thread.
type GroupStateManager struct {
	mu              sync.RWMutex
	groups          map[string]*ManagedGroup
	pendingProposals map[string]*ProposalWithRef
}

// ManagedGroup wraps a Group with state machine logic.
type ManagedGroup struct {
	Group               *Group
	State               GroupStateEnum
	Encryption          *Encryption
	LastEpochTimestamp  int64
	TreeHash            []byte
	Members             map[string]*MemberState
	PendingProposals    []*ProposalWithRef
}

// GroupStateEnum represents the group state.
type GroupStateEnum int

const (
	StateCreated GroupStateEnum = iota
	StateActive
	StateCommitting
	StateUpdated
	StateArchived
)

// MemberState tracks a member's status in the group.
type MemberState struct {
	Address       string
	JoinedAt      int64
	SigningKey    string
	Status        MemberStatusEnum
	LastMessageAt int64
}

// MemberStatusEnum represents member status.
type MemberStatusEnum int

const (
	MemberActive MemberStatusEnum = iota
	MemberSuspended
	MemberRemoved
)

// ProposalWithRef wraps a proposal with its reference.
type ProposalWithRef struct {
	Ref       *ProposalRef
	Proposal  Proposal
	CreatedAt int64
	Author    string
}

// NewGroupStateManager creates a new group state manager.
func NewGroupStateManager() *GroupStateManager {
	return &GroupStateManager{
		groups:           make(map[string]*ManagedGroup),
		pendingProposals: make(map[string]*ProposalWithRef),
	}
}

// CreateGroup initializes a new group.
func (gsm *GroupStateManager) CreateGroup(threadID string, members []string) (*ManagedGroup, error) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if _, exists := gsm.groups[threadID]; exists {
		return nil, fmt.Errorf("group already exists for thread %s", threadID)
	}

	group := NewGroup(threadID, members)

	// Initialize encryption
	epochSecret := make([]byte, 32)
	ks := &KeySchedule{
		Epoch:            0,
		EpochSecret:      epochSecret,
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: make([]byte, 16),
		ExporterSecret:   make([]byte, 32),
	}

	enc := NewEncryption(ks)

	// Build member state map
	memberStates := make(map[string]*MemberState)
	now := time.Now().UnixMilli()
	for _, member := range members {
		memberStates[member] = &MemberState{
			Address:      member,
			JoinedAt:     now,
			Status:       MemberActive,
			LastMessageAt: 0,
		}
	}

	managed := &ManagedGroup{
		Group:              group,
		State:              StateCreated,
		Encryption:         enc,
		LastEpochTimestamp: now,
		Members:            memberStates,
		PendingProposals:   []*ProposalWithRef{},
	}

	gsm.groups[threadID] = managed
	return managed, nil
}

// GetGroup retrieves a managed group by thread ID.
func (gsm *GroupStateManager) GetGroup(threadID string) (*ManagedGroup, error) {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found for thread %s", threadID)
	}

	return group, nil
}

// ProposeAdd creates an Add proposal.
func (gsm *GroupStateManager) ProposeAdd(threadID, proposedMember string) (*ProposalWithRef, error) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found")
	}

	// Check if already a member
	if _, isMember := group.Members[proposedMember]; isMember {
		return nil, fmt.Errorf("already a member")
	}

	// Create proposal (KeyPackage would come from the new member)
	signingKey := make([]byte, 32)
	encryptionKey := make([]byte, 32)

	proposal := &AddProposal{
		KeyPackage: &KeyPackage{
			Version:     0x0001,
			CipherSuite: 0x0001,
			InitKey:     make([]byte, 32),
			LeafNode: &LeafNode{
				EncryptionKey: encryptionKey,
				SignatureKey:  signingKey,
				Credential: &Credential{
					CredentialType: "signing_key",
					SigningKey:     signingKey,
					Identity:       proposedMember,
					IdentityKey:    make([]byte, 32),
					IdentitySig:    make([]byte, 32),
				},
				Capabilities: &Capabilities{
					Versions:   []uint16{0x0001},
					Ciphers:    []uint16{0x0001},
					Extensions: []uint16{},
				},
				Extensions: []Extension{},
			},
			Extensions: []Extension{},
		},
	}

	ref := NewProposalRef(proposal)
	proposalWithRef := &ProposalWithRef{
		Ref:       ref,
		Proposal:  proposal,
		CreatedAt: time.Now().UnixMilli(),
		Author:    "system",
	}

	group.PendingProposals = append(group.PendingProposals, proposalWithRef)
	gsm.pendingProposals[fmt.Sprintf("%x", ref.Hash)] = proposalWithRef

	return proposalWithRef, nil
}

// ProposeRemove creates a Remove proposal.
func (gsm *GroupStateManager) ProposeRemove(threadID, memberToRemove string) (*ProposalWithRef, error) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found")
	}

	// Find member's leaf index
	var leafIndex uint32
	found := false
	for i, member := range group.Group.Members {
		if member == memberToRemove {
			leafIndex = uint32(i * 2)
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("member not found")
	}

	proposal := &RemoveProposal{LeafNodeIndex: leafIndex}
	ref := NewProposalRef(proposal)
	proposalWithRef := &ProposalWithRef{
		Ref:       ref,
		Proposal:  proposal,
		CreatedAt: time.Now().UnixMilli(),
		Author:    memberToRemove,
	}

	group.PendingProposals = append(group.PendingProposals, proposalWithRef)
	gsm.pendingProposals[fmt.Sprintf("%x", ref.Hash)] = proposalWithRef

	return proposalWithRef, nil
}

// CommitProposals bundles pending proposals into a commit and advances epoch.
func (gsm *GroupStateManager) CommitProposals(threadID, committer string) (*MLSCommit, error) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found")
	}

	if len(group.PendingProposals) == 0 {
		return nil, fmt.Errorf("no pending proposals")
	}

	// Build commit content
	content := &CommitContent{
		Proposals: []ProposalRef{},
		Updates:   []ProposalRef{},
		Removes:   []ProposalRef{},
	}

	for _, prop := range group.PendingProposals {
		content.Proposals = append(content.Proposals, *prop.Ref)
	}

	// Create commit
	commit := &MLSCommit{
		GroupID:      group.Group.ID,
		Epoch:        group.Group.Epoch,
		Content:      content,
		Confirmation: make([]byte, 32),
		Signature:    make([]byte, 64),
	}

	// Apply commit
	if err := group.Group.ApplyCommit(commit); err != nil {
		return nil, fmt.Errorf("apply commit: %w", err)
	}

	// Process proposals to update group state
	for _, prop := range group.PendingProposals {
		switch p := prop.Proposal.(type) {
		case *AddProposal:
			if p.KeyPackage.LeafNode != nil && p.KeyPackage.LeafNode.Credential != nil {
				member := p.KeyPackage.LeafNode.Credential.Identity
				group.Members[member] = &MemberState{
					Address:    member,
					JoinedAt:   time.Now().UnixMilli(),
					Status:     MemberActive,
				}
			}
		case *RemoveProposal:
			// Find and remove member
			for _, state := range group.Members {
				if state.Status == MemberActive {
					state.Status = MemberRemoved
					break
				}
			}
		}
	}

	// Clear pending proposals
	group.PendingProposals = []*ProposalWithRef{}

	// Advance state
	group.State = StateUpdated
	group.LastEpochTimestamp = time.Now().UnixMilli()

	// Re-derive encryption keys for new epoch
	epochSecret := make([]byte, 32)
	ks := &KeySchedule{
		Epoch:            group.Group.Epoch,
		EpochSecret:      epochSecret,
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: make([]byte, 16),
		ExporterSecret:   make([]byte, 32),
	}
	group.Encryption = NewEncryption(ks)

	return commit, nil
}

// EncryptMessage encrypts a message with the current group key.
func (gsm *GroupStateManager) EncryptMessage(threadID string, plaintext []byte) ([]byte, error) {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found")
	}

	return group.Encryption.Encrypt(plaintext)
}

// DecryptMessage decrypts a message with the current group key.
func (gsm *GroupStateManager) DecryptMessage(threadID string, ciphertext []byte) ([]byte, error) {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found")
	}

	return group.Encryption.Decrypt(ciphertext)
}

// GetMembers returns the active members of the group.
func (gsm *GroupStateManager) GetMembers(threadID string) ([]string, error) {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return nil, fmt.Errorf("group not found")
	}

	var members []string
	for addr, state := range group.Members {
		if state.Status == MemberActive {
			members = append(members, addr)
		}
	}

	return members, nil
}

// GetEpoch returns the current epoch number.
func (gsm *GroupStateManager) GetEpoch(threadID string) (uint64, error) {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	group, exists := gsm.groups[threadID]
	if !exists {
		return 0, fmt.Errorf("group not found")
	}

	return group.Group.Epoch, nil
}
