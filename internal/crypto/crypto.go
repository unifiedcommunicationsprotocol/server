// Package crypto implements MLS (RFC 9420) group management, encryption/decryption, and key rotation.
package crypto

import (
	"fmt"
	"sync"

	"github.com/unifiedcommunicationsprotocol/server/internal/crypto/mls"
	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// Group wraps an MLS ManagedGroup with thread ID.
type Group struct {
	ThreadID    models.ULID
	ManagedGroup *mls.ManagedGroup
}

// Manager manages MLS groups and encryption/decryption using RFC 9420 implementation.
type Manager struct {
	mu     sync.RWMutex
	gsm    *mls.GroupStateManager
	groups map[string]*Group // keyed by thread ID
}

// New creates a new crypto Manager with real MLS implementation.
func New() *Manager {
	return &Manager{
		gsm:    mls.NewGroupStateManager(),
		groups: make(map[string]*Group),
	}
}

// CreateGroup creates a new MLS group for a thread using RFC 9420.
// Sets up the group with initial members, encryption key schedule, and tree structure.
func (m *Manager) CreateGroup(threadID models.ULID, members []string) (*Group, error) {
	if len(members) == 0 {
		return nil, fmt.Errorf("group must have at least one member")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	threadIDStr := string(threadID)

	// Check if already exists
	if _, exists := m.groups[threadIDStr]; exists {
		return nil, fmt.Errorf("group already exists for thread %s", threadIDStr)
	}

	// Create group using real MLS GroupStateManager
	managedGroup, err := m.gsm.CreateGroup(threadIDStr, members)
	if err != nil {
		return nil, fmt.Errorf("create MLS group: %w", err)
	}

	group := &Group{
		ThreadID:     threadID,
		ManagedGroup: managedGroup,
	}

	m.groups[threadIDStr] = group
	return group, nil
}

// GetGroup retrieves a group by thread ID.
func (m *Manager) GetGroup(threadID models.ULID) (*Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[string(threadID)]
	if !ok {
		return nil, fmt.Errorf("group not found for thread %s", threadID)
	}
	return group, nil
}

// EncryptMessage encrypts a message for a group using MLS encryption with proper key schedule.
// Uses the group's AES-128-GCM encryption with the current epoch's encryption secret.
func (m *Manager) EncryptMessage(threadID models.ULID, plaintext []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[string(threadID)]
	if !ok {
		return nil, fmt.Errorf("group not found for thread %s", threadID)
	}

	// Use MLS encryption with the group's encryption handler
	if group.ManagedGroup.Encryption == nil {
		return nil, fmt.Errorf("group encryption not initialized")
	}

	ciphertext, err := group.ManagedGroup.Encryption.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt message: %w", err)
	}

	return ciphertext, nil
}

// DecryptMessage decrypts a message from a group using MLS decryption.
// Uses the group's AES-128-GCM decryption with the current epoch's encryption secret.
func (m *Manager) DecryptMessage(threadID models.ULID, ciphertext []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[string(threadID)]
	if !ok {
		return nil, fmt.Errorf("group not found for thread %s", threadID)
	}

	if group.ManagedGroup.Encryption == nil {
		return nil, fmt.Errorf("group encryption not initialized")
	}

	plaintext, err := group.ManagedGroup.Encryption.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt message: %w", err)
	}

	return plaintext, nil
}

// AddMember adds a member to a group via MLS Add proposal.
// Proposes the addition and commits the proposal to advance the epoch.
func (m *Manager) AddMember(threadID models.ULID, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	threadIDStr := string(threadID)

	// Propose add via GroupStateManager
	_, err := m.gsm.ProposeAdd(threadIDStr, member)
	if err != nil {
		return fmt.Errorf("propose add member: %w", err)
	}

	// Commit the proposal to advance epoch
	_, err = m.gsm.CommitProposals(threadIDStr, "system")
	if err != nil {
		return fmt.Errorf("commit add proposal: %w", err)
	}

	return nil
}

// RemoveMember removes a member from a group via MLS Remove proposal.
// Proposes the removal and commits the proposal to advance the epoch.
func (m *Manager) RemoveMember(threadID models.ULID, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	threadIDStr := string(threadID)

	// Propose remove via GroupStateManager
	_, err := m.gsm.ProposeRemove(threadIDStr, member)
	if err != nil {
		return fmt.Errorf("propose remove member: %w", err)
	}

	// Commit the proposal to advance epoch
	_, err = m.gsm.CommitProposals(threadIDStr, "system")
	if err != nil {
		return fmt.Errorf("commit remove proposal: %w", err)
	}

	return nil
}

// AdvanceEpoch advances the group epoch (on signing key rotation).
// Uses MLS Update proposal to signal key rotation and derive new epoch secrets.
func (m *Manager) AdvanceEpoch(threadID models.ULID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	threadIDStr := string(threadID)

	group, ok := m.groups[threadIDStr]
	if !ok {
		return fmt.Errorf("group not found for thread %s", threadID)
	}

	// Advance epoch on the MLS group
	group.ManagedGroup.Group.AdvanceEpoch()

	return nil
}

// GetGroupMembers returns the current members of a group.
func (m *Manager) GetGroupMembers(threadID models.ULID) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[string(threadID)]
	if !ok {
		return nil, fmt.Errorf("group not found for thread %s", threadID)
	}

	return group.ManagedGroup.Group.Members, nil
}

// GetGroupEpoch returns the current epoch of a group.
func (m *Manager) GetGroupEpoch(threadID models.ULID) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[string(threadID)]
	if !ok {
		return 0, fmt.Errorf("group not found for thread %s", threadID)
	}

	return group.ManagedGroup.Group.Epoch, nil
}
