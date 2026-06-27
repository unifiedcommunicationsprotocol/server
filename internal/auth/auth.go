// Package auth handles challenge-response, session tokens, and signature verification.
package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// Manager handles authentication and session management.
type Manager struct {
	sessions map[string]*Session
}

// Session represents an authenticated user session.
type Session struct {
	Address   string
	Token     string
	ExpiresAt int64
	RevokedAt *int64
}

// New creates a new auth Manager.
func New() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// GenerateChallenge creates a 32-byte random challenge for authentication.
func GenerateChallenge() ([]byte, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("generate challenge: %w", err)
	}
	return challenge, nil
}

// VerifyChallengeResponse verifies that a signature is valid for a challenge and signing key.
func VerifyChallengeResponse(challenge []byte, signingPubKey ed25519.PublicKey, sigBase64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	if !ed25519.Verify(signingPubKey, challenge, sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// CreateSession creates a new session token for an authenticated user.
// Token is opaque and short-lived (max 24 hours).
func (m *Manager) CreateSession(address string, maxLifetimeSecs int) (*Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	token := base64.StdEncoding.EncodeToString(tokenBytes)
	now := time.Now().Unix()
	expiresAt := now + int64(maxLifetimeSecs)

	// Cap at 24 hours
	maxExpiry := now + 24*3600
	if expiresAt > maxExpiry {
		expiresAt = maxExpiry
	}

	session := &Session{
		Address:   address,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	m.sessions[token] = session
	return session, nil
}

// ValidateSession checks if a session token is still valid.
func (m *Manager) ValidateSession(token string) (address string, err error) {
	session, ok := m.sessions[token]
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	// Check if revoked
	if session.RevokedAt != nil {
		return "", fmt.Errorf("session revoked")
	}

	// Check expiry
	if time.Now().Unix() > session.ExpiresAt {
		return "", fmt.Errorf("session expired")
	}

	return session.Address, nil
}

// RefreshSession issues a new token for an existing valid session.
func (m *Manager) RefreshSession(oldToken string, maxLifetimeSecs int) (*Session, error) {
	// Validate old token first
	address, err := m.ValidateSession(oldToken)
	if err != nil {
		return nil, err
	}

	// Create new session
	newSession, err := m.CreateSession(address, maxLifetimeSecs)
	if err != nil {
		return nil, err
	}

	// Optionally revoke old session
	m.RevokeSession(oldToken)

	return newSession, nil
}

// RevokeSession revokes a session token immediately.
func (m *Manager) RevokeSession(token string) {
	session, ok := m.sessions[token]
	if ok {
		now := time.Now().Unix()
		session.RevokedAt = &now
	}
}

// Challenger represents a one-time challenge for authentication.
type Challenger struct {
	Challenge   []byte
	ExpiresAt   int64
	UsedAt      *int64
}

// ChallengeStore manages active authentication challenges.
type ChallengeStore struct {
	challenges map[string]*Challenger
}

// NewChallengeStore creates a new challenge store.
func NewChallengeStore() *ChallengeStore {
	return &ChallengeStore{
		challenges: make(map[string]*Challenger),
	}
}

// IssueChallenge creates a new challenge for an address (valid for 60 seconds).
func (cs *ChallengeStore) IssueChallenge(address string) ([]byte, error) {
	challenge, err := GenerateChallenge()
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	expiresAt := now + 60

	cs.challenges[address] = &Challenger{
		Challenge: challenge,
		ExpiresAt: expiresAt,
	}

	return challenge, nil
}

// ValidateChallenge checks if a challenge is still valid for an address.
func (cs *ChallengeStore) ValidateChallenge(address string) ([]byte, error) {
	challenger, ok := cs.challenges[address]
	if !ok {
		return nil, fmt.Errorf("challenge not found")
	}

	// Check expiry
	if time.Now().Unix() > challenger.ExpiresAt {
		delete(cs.challenges, address)
		return nil, fmt.Errorf("challenge expired")
	}

	// Check if already used
	if challenger.UsedAt != nil {
		return nil, fmt.Errorf("challenge already used")
	}

	return challenger.Challenge, nil
}

// ConsumeChallenge marks a challenge as used.
func (cs *ChallengeStore) ConsumeChallenge(address string) error {
	challenger, ok := cs.challenges[address]
	if !ok {
		return fmt.Errorf("challenge not found")
	}

	if challenger.UsedAt != nil {
		return fmt.Errorf("challenge already used")
	}

	now := time.Now().Unix()
	challenger.UsedAt = &now

	// Delete after a short time
	delete(cs.challenges, address)

	return nil
}
