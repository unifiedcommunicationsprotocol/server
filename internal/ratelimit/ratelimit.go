// Package ratelimit provides rate limiting for UCP endpoints.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter enforces rate limits per key (e.g., IP address or user).
type Limiter struct {
	mu       sync.RWMutex
	limits   map[string]*bucketState
	maxBurst int
	refillMs int64
}

// bucketState tracks the token bucket for a key.
type bucketState struct {
	tokens    int
	lastFill  time.Time
	expiresAt time.Time
}

// New creates a new rate limiter.
// maxBurst: maximum tokens in the bucket
// refillPerSec: tokens to add per second
func New(maxBurst int, refillPerSec int) *Limiter {
	return &Limiter{
		limits:   make(map[string]*bucketState),
		maxBurst: maxBurst,
		refillMs: 1000 / int64(refillPerSec),
	}
}

// Allow checks if an action is allowed for the given key.
// Returns true if within rate limit, false otherwise.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.limits[key]

	// Initialize or expire old bucket
	if !exists || now.After(bucket.expiresAt) {
		bucket = &bucketState{
			tokens:    l.maxBurst,
			lastFill:  now,
			expiresAt: now.Add(5 * time.Minute),
		}
		l.limits[key] = bucket
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastFill).Milliseconds()
	tokensToAdd := int(elapsed / l.refillMs)

	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.tokens+tokensToAdd, l.maxBurst)
		bucket.lastFill = now
	}

	// Try to consume a token
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// Reset clears all limits (for testing).
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.limits = make(map[string]*bucketState)
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
