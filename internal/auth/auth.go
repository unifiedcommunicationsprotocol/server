// Package auth handles challenge-response, session tokens, and signature verification.
package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// SessionStore defines the interface for session persistence.
type SessionStore interface {
	CreateSession(ctx context.Context, address string, token string, expiresAt int64) error
	GetSession(ctx context.Context, token string) (address string, err error)
	RevokeSession(ctx context.Context, token string) error
}

// Manager handles authentication and session management with database persistence.
type Manager struct {
	store    SessionStore
	cacheMu  sync.RWMutex
	cache    map[string]*Session // Optional in-memory cache for performance
	cacheTTL time.Duration
}

// Session represents an authenticated user session.
type Session struct {
	Address   string
	Token     string
	ExpiresAt int64
	RevokedAt *int64
}

// New creates a new auth Manager with database backing and optional in-memory cache.
// Deprecated: Use NewWithStore instead. Kept for backward compatibility with tests.
func New() *Manager {
	return &Manager{
		store:    nil,
		cache:    make(map[string]*Session),
		cacheTTL: 5 * time.Minute,
	}
}

// NewWithStore creates a new auth Manager with database persistence.
func NewWithStore(store SessionStore) *Manager {
	return &Manager{
		store:    store,
		cache:    make(map[string]*Session),
		cacheTTL: 5 * time.Minute,
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
// Token is opaque and short-lived (max 24 hours). Persisted to database.
func (m *Manager) CreateSession(ctx context.Context, address string, maxLifetimeSecs int) (*Session, error) {
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

	// Persist to database
	if m.store != nil {
		if err := m.store.CreateSession(ctx, address, token, expiresAt); err != nil {
			return nil, fmt.Errorf("persist session: %w", err)
		}
	}

	// Cache for performance
	m.cacheMu.Lock()
	m.cache[token] = session
	m.cacheMu.Unlock()

	return session, nil
}

// ValidateSession checks if a session token is still valid.
// Checks cache first, then database.
func (m *Manager) ValidateSession(ctx context.Context, token string) (address string, err error) {
	// Check cache first
	m.cacheMu.RLock()
	session, inCache := m.cache[token]
	m.cacheMu.RUnlock()

	if inCache {
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

	// Fall back to database
	if m.store != nil {
		address, err := m.store.GetSession(ctx, token)
		if err != nil {
			return "", err
		}
		// Cache it for next lookup
		m.cacheMu.Lock()
		m.cache[token] = &Session{
			Address:   address,
			Token:     token,
			ExpiresAt: 0,
		}
		m.cacheMu.Unlock()
		return address, nil
	}

	return "", fmt.Errorf("session not found")
}

// RefreshSession issues a new token for an existing valid session.
func (m *Manager) RefreshSession(ctx context.Context, oldToken string, maxLifetimeSecs int) (*Session, error) {
	// Validate old token first
	address, err := m.ValidateSession(ctx, oldToken)
	if err != nil {
		return nil, err
	}

	// Create new session
	newSession, err := m.CreateSession(ctx, address, maxLifetimeSecs)
	if err != nil {
		return nil, err
	}

	// Revoke old session
	_ = m.RevokeSession(ctx, oldToken)

	return newSession, nil
}

// RevokeSession revokes a session token immediately.
func (m *Manager) RevokeSession(ctx context.Context, token string) error {
	// Revoke in database
	if m.store != nil {
		if err := m.store.RevokeSession(ctx, token); err != nil {
			return fmt.Errorf("revoke session: %w", err)
		}
	}

	// Revoke in cache
	m.cacheMu.Lock()
	if session, ok := m.cache[token]; ok {
		now := time.Now().Unix()
		session.RevokedAt = &now
	}
	m.cacheMu.Unlock()

	return nil
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
