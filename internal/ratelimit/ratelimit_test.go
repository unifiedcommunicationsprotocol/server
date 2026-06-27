package ratelimit

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// 10 burst, 10 refill per second
	limiter := New(10, 10)

	// First 10 requests should succeed
	for i := 0; i < 10; i++ {
		if !limiter.Allow("user1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 11th request should fail
	if limiter.Allow("user1") {
		t.Error("11th request should be rate limited")
	}

	// Wait for refill (100ms = 1 token)
	time.Sleep(150 * time.Millisecond)

	// Should have 1-2 tokens now
	if !limiter.Allow("user1") {
		t.Error("should have refilled at least 1 token")
	}
}

func TestRateLimiterPerKey(t *testing.T) {
	limiter := New(3, 10)

	// Each key should have independent limits
	if !limiter.Allow("user1") {
		t.Error("user1 first request should be allowed")
	}

	if !limiter.Allow("user2") {
		t.Error("user2 first request should be allowed")
	}

	// Exhaust user1's limit
	limiter.Allow("user1")
	limiter.Allow("user1")

	// user1 is at limit, user2 should still have tokens
	if limiter.Allow("user1") {
		t.Error("user1 should be rate limited")
	}

	if !limiter.Allow("user2") {
		t.Error("user2 should not be rate limited")
	}
}

func TestRateLimiterReset(t *testing.T) {
	limiter := New(5, 10)

	// Exhaust limit
	for i := 0; i < 5; i++ {
		limiter.Allow("test")
	}

	if limiter.Allow("test") {
		t.Error("should be rate limited")
	}

	// Reset
	limiter.Reset()

	// Should work again
	if !limiter.Allow("test") {
		t.Error("should work after reset")
	}
}

func TestRateLimiterExpiry(t *testing.T) {
	limiter := New(1, 10)

	// Use the one token
	if !limiter.Allow("test") {
		t.Error("first request should be allowed")
	}

	// Rate limited
	if limiter.Allow("test") {
		t.Error("should be rate limited")
	}

	// The expiry is 5 minutes, so we can't test it directly
	// But we can test reset() and other behavior
}
