package main

import (
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
	"github.com/unifiedcommunicationsprotocol/server/internal/router"
)

// TestRouterMessageToLocalRecipient tests routing to local recipients.
func TestRouterMessageToLocalRecipient(t *testing.T) {
	r := router.New()
	r.RegisterLocalRecipient("alice@localhost")
	r.RegisterLocalRecipient("bob@localhost")

	envelope := &models.UCPEnvelope{
		From: "alice@localhost",
		To:   []string{"bob@localhost"},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Errorf("route failed: %v", err)
		return
	}

	if len(local) != 1 || local[0] != "bob@localhost" {
		t.Errorf("expected bob as local recipient, got %v", local)
	}

	if len(remote) != 0 {
		t.Errorf("expected no remote recipients, got %v", remote)
	}
}

// TestRouterMessageToRemoteRecipient tests routing to remote recipients.
func TestRouterMessageToRemoteRecipient(t *testing.T) {
	r := router.New()
	r.RegisterLocalRecipient("alice@server1.example.com")

	envelope := &models.UCPEnvelope{
		From: "alice@server1.example.com",
		To:   []string{"bob@server2.example.com", "carol@server2.example.com"},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Errorf("route failed: %v", err)
		return
	}

	if len(local) != 0 {
		t.Errorf("expected no local recipients, got %v", local)
	}

	if len(remote) != 1 {
		t.Errorf("expected 1 remote domain, got %d", len(remote))
		return
	}

	if recipients, ok := remote["server2.example.com"]; !ok || len(recipients) != 2 {
		t.Errorf("expected 2 recipients for server2.example.com, got %v", remote)
	}
}

// TestRouterMessageMixed tests routing to both local and remote recipients.
func TestRouterMessageMixed(t *testing.T) {
	r := router.New()
	r.RegisterLocalRecipient("alice@server1.example.com")
	r.RegisterLocalRecipient("bob@server1.example.com")

	envelope := &models.UCPEnvelope{
		From: "alice@server1.example.com",
		To: []string{
			"bob@server1.example.com",           // local
			"carol@server2.example.com",         // remote
			"diana@server2.example.com",         // remote
			"eve@server3.example.com",           // different remote domain
		},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Errorf("route failed: %v", err)
		return
	}

	if len(local) != 1 || local[0] != "bob@server1.example.com" {
		t.Errorf("expected bob as local, got %v", local)
	}

	if len(remote) != 2 {
		t.Errorf("expected 2 remote domains, got %d", len(remote))
		return
	}

	if recipients, ok := remote["server2.example.com"]; !ok || len(recipients) != 2 {
		t.Errorf("server2.example.com recipients mismatch")
	}

	if recipients, ok := remote["server3.example.com"]; !ok || len(recipients) != 1 {
		t.Errorf("server3.example.com recipients mismatch")
	}
}

// TestFederationConnection tests establishing a federation connection.
func TestFederationConnection(t *testing.T) {
	r := router.New()

	conn, err := r.EstablishFederation("server2.example.com")
	if err != nil {
		t.Errorf("establish federation failed: %v", err)
		return
	}

	if conn.Domain != "server2.example.com" {
		t.Errorf("domain mismatch: %s", conn.Domain)
	}

	if conn.Established.IsZero() {
		t.Error("establishment time not set")
	}
}

// TestFederationConnectionCaching tests that connections are cached.
func TestFederationConnectionCaching(t *testing.T) {
	r := router.New()

	conn1, _ := r.GetFederationConnection("server2.example.com")
	conn2, _ := r.GetFederationConnection("server2.example.com")

	if conn1.Established != conn2.Established {
		t.Error("connections should be cached, not recreated")
	}
}

// TestRetryQueueEnqueue tests enqueuing a retry.
func TestRetryQueueEnqueue(t *testing.T) {
	rq := router.NewRetryQueue()

	attempt := rq.EnqueueRetry("msg-123", "bob@remote.example.com")

	if attempt.EnvelopeID != "msg-123" {
		t.Errorf("envelope ID mismatch: %s", attempt.EnvelopeID)
	}

	if attempt.Retries != 1 {
		t.Errorf("expected 1 retry attempt, got %d", attempt.Retries)
	}

	if attempt.NextRetry.Before(time.Now()) {
		t.Error("next retry should be in the future")
	}
}

// TestRetryQueueShouldRetry tests retry logic.
func TestRetryQueueShouldRetry(t *testing.T) {
	rq := router.NewRetryQueue()

	// Enqueue a message
	attempt := rq.EnqueueRetry("msg-123", "bob@remote.example.com")

	// Immediately should not retry (scheduled for 1 minute)
	if rq.ShouldRetry("msg-123") {
		t.Error("should not retry immediately")
	}

	// Override next retry to past
	attempt.NextRetry = time.Now().Add(-10 * time.Second)

	// Now should retry
	if !rq.ShouldRetry("msg-123") {
		t.Error("should retry after scheduled time")
	}
}

// TestRetryQueueExpiry tests that messages expire after 48 hours.
func TestRetryQueueExpiry(t *testing.T) {
	rq := router.NewRetryQueue()

	// Enqueue a message
	attempt := rq.EnqueueRetry("msg-123", "bob@remote.example.com")

	// Fake it being attempted 50 hours ago
	attempt.AttemptedAt = time.Now().Add(-50 * time.Hour)
	attempt.NextRetry = time.Now().Add(-1 * time.Hour) // Should have already retried by now

	// Should not retry (expired)
	if rq.ShouldRetry("msg-123") {
		t.Error("message should expire after 48 hours")
	}
}

// TestExponentialBackoff tests exponential backoff capping.
func TestExponentialBackoff(t *testing.T) {
	rq := router.NewRetryQueue()
	attempt := rq.EnqueueRetry("msg-123", "bob@remote.example.com")

	// Verify initial backoff is set
	if attempt.NextRetry.Before(time.Now()) {
		t.Error("initial retry should be scheduled in future")
	}

	// Simulate many retries
	for i := 0; i < 10; i++ {
		rq.IncrementRetry("msg-123")
	}

	// After many retries, backoff should be capped at 4 hours
	nextRetry := attempt.NextRetry.Sub(time.Now())
	maxBackoff := 4 * time.Hour

	if nextRetry > maxBackoff+10*time.Second {
		t.Errorf("backoff should be capped at 4 hours, got %v", nextRetry)
	}
}

// TestRetryQueueIncrementRetry tests incrementing retry count.
func TestRetryQueueIncrementRetry(t *testing.T) {
	rq := router.NewRetryQueue()
	attempt := rq.EnqueueRetry("msg-123", "bob@remote.example.com")

	if attempt.Retries != 1 {
		t.Errorf("initial retries should be 1, got %d", attempt.Retries)
	}

	rq.IncrementRetry("msg-123")

	if attempt.Retries != 2 {
		t.Errorf("after increment, retries should be 2, got %d", attempt.Retries)
	}
}

// TestRouterLocalRecipient tests local recipient registration.
func TestRouterLocalRecipient(t *testing.T) {
	r := router.New()

	if r.IsLocalRecipient("alice@localhost") {
		t.Error("unregistered address should not be local")
	}

	r.RegisterLocalRecipient("alice@localhost")

	if !r.IsLocalRecipient("alice@localhost") {
		t.Error("registered address should be local")
	}
}

// TestRouterMultipleRemoteDomains tests routing to multiple remote domains.
func TestRouterMultipleRemoteDomains(t *testing.T) {
	r := router.New()

	envelope := &models.UCPEnvelope{
		From: "alice@server1.example.com",
		To: []string{
			"bob@server2.example.com",
			"carol@server2.example.com",
			"diana@server3.example.com",
			"eve@server4.example.com",
		},
	}

	_, remote, _ := r.RouteMessage(envelope)

	if len(remote) != 3 {
		t.Errorf("expected 3 remote domains, got %d", len(remote))
	}

	if _, ok := remote["server2.example.com"]; !ok {
		t.Error("missing server2.example.com in remote")
	}

	if _, ok := remote["server3.example.com"]; !ok {
		t.Error("missing server3.example.com in remote")
	}

	if _, ok := remote["server4.example.com"]; !ok {
		t.Error("missing server4.example.com in remote")
	}
}
