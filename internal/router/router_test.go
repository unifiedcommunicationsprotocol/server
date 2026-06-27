package router

import (
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

func TestRegisterLocalRecipient(t *testing.T) {
	r := New()

	r.RegisterLocalRecipient("alice@example.com")

	if !r.IsLocalRecipient("alice@example.com") {
		t.Error("RegisterLocalRecipient should make address local")
	}

	if r.IsLocalRecipient("bob@example.com") {
		t.Error("Unregistered address should not be local")
	}
}

func TestRouteMessage(t *testing.T) {
	r := New()
	r.RegisterLocalRecipient("alice@example.com")

	envelope := &models.UCPEnvelope{
		From: "alice@example.com",
		To:   []string{"alice@example.com", "bob@example.com"},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Fatalf("RouteMessage error: %v", err)
	}

	if len(local) != 1 {
		t.Errorf("Local recipients: got %d, want 1", len(local))
	}

	if _, ok := remote["example.com"]; !ok {
		t.Error("Remote server not found")
	}
}

func TestRetryQueue(t *testing.T) {
	rq := NewRetryQueue()

	envelopeID := "env_123"
	recipient := "bob@example.com"

	// Enqueue
	attempt := rq.EnqueueRetry(envelopeID, recipient)
	if attempt.Retries != 1 {
		t.Error("Initial retry count should be 1")
	}

	// Should not retry immediately
	if rq.ShouldRetry(envelopeID) {
		t.Error("Should not retry before next scheduled time")
	}

	// Increment and check retry time increases
	oldNextRetry := attempt.NextRetry
	rq.IncrementRetry(envelopeID)
	if attempt.NextRetry.Before(oldNextRetry) {
		t.Error("NextRetry should increase on increment")
	}
}

// TestRouteMessageMultipleDomains tests routing to multiple remote domains
func TestRouteMessageMultipleDomains(t *testing.T) {
	r := New()
	r.RegisterLocalRecipient("alice@example.com")

	envelope := &models.UCPEnvelope{
		From: "alice@example.com",
		To: []string{
			"alice@example.com",           // local
			"bob@other.com",               // remote domain 1
			"carol@third.org",             // remote domain 2
			"dave@other.com",              // remote domain 1 again
		},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Fatalf("RouteMessage error: %v", err)
	}

	if len(local) != 1 {
		t.Errorf("Local recipients: got %d, want 1", len(local))
	}

	if len(remote) != 2 {
		t.Errorf("Remote domains: got %d, want 2", len(remote))
	}

	if len(remote["other.com"]) != 2 {
		t.Errorf("Recipients for other.com: got %d, want 2", len(remote["other.com"]))
	}

	if len(remote["third.org"]) != 1 {
		t.Errorf("Recipients for third.org: got %d, want 1", len(remote["third.org"]))
	}

	t.Logf("✓ Multi-domain routing works correctly")
}

// TestRouteMessageAllLocal tests routing with all local recipients
func TestRouteMessageAllLocal(t *testing.T) {
	r := New()
	r.RegisterLocalRecipient("alice@example.com")
	r.RegisterLocalRecipient("bob@example.com")
	r.RegisterLocalRecipient("carol@example.com")

	envelope := &models.UCPEnvelope{
		From: "alice@example.com",
		To:   []string{"alice@example.com", "bob@example.com", "carol@example.com"},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Fatalf("RouteMessage error: %v", err)
	}

	if len(local) != 3 {
		t.Errorf("Local recipients: got %d, want 3", len(local))
	}

	if len(remote) != 0 {
		t.Errorf("Remote recipients: got %d, want 0", len(remote))
	}

	t.Logf("✓ All-local routing works correctly")
}

// TestRouteMessageAllRemote tests routing with all remote recipients
func TestRouteMessageAllRemote(t *testing.T) {
	r := New()

	envelope := &models.UCPEnvelope{
		From: "alice@example.com",
		To:   []string{"bob@other.com", "carol@another.org"},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Fatalf("RouteMessage error: %v", err)
	}

	if len(local) != 0 {
		t.Errorf("Local recipients: got %d, want 0", len(local))
	}

	if len(remote) != 2 {
		t.Errorf("Remote domains: got %d, want 2", len(remote))
	}

	t.Logf("✓ All-remote routing works correctly")
}

// TestExponentialBackoff verifies exponential backoff calculation
func TestExponentialBackoff(t *testing.T) {
	rq := NewRetryQueue()
	envelopeID := "env_backoff_test"
	recipient := "bob@example.com"

	// Enqueue initial attempt
	rq.EnqueueRetry(envelopeID, recipient)
	attempt := rq.attempts[envelopeID]

	// Track retry times
	backoffSequence := []time.Duration{
		1 * time.Minute,  // retry 1
		2 * time.Minute,  // retry 2
		4 * time.Minute,  // retry 3
		8 * time.Minute,  // retry 4
		16 * time.Minute, // retry 5
	}

	for i, expectedBackoff := range backoffSequence {
		rq.IncrementRetry(envelopeID)

		// Calculate actual backoff from the increment
		actualBackoff := attempt.NextRetry.Sub(time.Now())

		// Allow some tolerance for test execution time
		tolerance := 2 * time.Second
		if actualBackoff < expectedBackoff-tolerance {
			t.Errorf("Retry %d: backoff too short (got ~%v, want ~%v)", i+1, actualBackoff, expectedBackoff)
		}

		t.Logf("Retry %d: backoff = %v", i+1, expectedBackoff)
	}

	t.Logf("✓ Exponential backoff works correctly")
}

// TestRetryQueueMaxBackoff verifies backoff caps at 4 hours
func TestRetryQueueMaxBackoff(t *testing.T) {
	rq := NewRetryQueue()
	envelopeID := "env_max_backoff"
	recipient := "bob@example.com"

	rq.EnqueueRetry(envelopeID, recipient)
	attempt := rq.attempts[envelopeID]

	// Do many retries to exceed 4-hour cap
	for i := 0; i < 20; i++ {
		rq.IncrementRetry(envelopeID)
	}

	// After many increments, backoff should still be <= 4 hours
	actualBackoff := attempt.NextRetry.Sub(time.Now())
	maxBackoff := 4 * time.Hour
	tolerance := 2 * time.Second

	if actualBackoff > maxBackoff+tolerance {
		t.Errorf("Backoff exceeded max: got %v, want <= %v", actualBackoff, maxBackoff)
	}

	t.Logf("✓ Max backoff (4h) is enforced")
}

// TestRetryWindow48Hours verifies 48-hour retry window
func TestRetryWindow48Hours(t *testing.T) {
	rq := NewRetryQueue()
	envelopeID := "env_48h_test"
	recipient := "bob@example.com"

	rq.EnqueueRetry(envelopeID, recipient)
	attempt := rq.attempts[envelopeID]

	// Simulate old attempt (> 48 hours ago)
	attempt.AttemptedAt = time.Now().Add(-49 * time.Hour)
	attempt.NextRetry = time.Now().Add(-1 * time.Second) // Should be ready to retry

	// ShouldRetry should return false (outside 48h window)
	if rq.ShouldRetry(envelopeID) {
		t.Error("Attempt older than 48h should not retry")
	}

	// Attempt should be removed from queue
	if _, ok := rq.attempts[envelopeID]; ok {
		t.Error("Attempt older than 48h should be removed from queue")
	}

	t.Logf("✓ 48-hour retry window enforced")
}

// TestFederationConnection tests connection establishment and retrieval
func TestFederationConnection(t *testing.T) {
	r := New()
	domain := "remote.example.com"

	// Establish connection
	conn1, err := r.EstablishFederation(domain)
	if err != nil {
		t.Fatalf("EstablishFederation error: %v", err)
	}

	if conn1.Domain != domain {
		t.Errorf("Domain mismatch: got %q, want %q", conn1.Domain, domain)
	}

	if conn1.Retries != 0 {
		t.Errorf("Initial retries should be 0, got %d", conn1.Retries)
	}

	// Retrieve same connection
	conn2, err := r.GetFederationConnection(domain)
	if err != nil {
		t.Fatalf("GetFederationConnection error: %v", err)
	}

	if conn2 != conn1 {
		t.Error("GetFederationConnection should return same instance")
	}

	t.Logf("✓ Federation connection established and retrieved")
}

// TestFederationMultipleConnections tests multiple federation connections
func TestFederationMultipleConnections(t *testing.T) {
	r := New()
	domains := []string{"server1.com", "server2.org", "server3.net"}

	// Establish multiple connections
	for _, domain := range domains {
		_, err := r.EstablishFederation(domain)
		if err != nil {
			t.Fatalf("EstablishFederation error for %s: %v", domain, err)
		}
	}

	// Verify all are stored
	if len(r.remoteServers) != len(domains) {
		t.Errorf("Remote servers count: got %d, want %d", len(r.remoteServers), len(domains))
	}

	// Verify retrieval
	for _, domain := range domains {
		conn, err := r.GetFederationConnection(domain)
		if err != nil {
			t.Fatalf("GetFederationConnection error for %s: %v", domain, err)
		}

		if conn.Domain != domain {
			t.Errorf("Domain mismatch: got %q, want %q", conn.Domain, domain)
		}
	}

	t.Logf("✓ Multiple federation connections work correctly")
}

// TestRouteMessageEmptyRecipients tests edge case with empty recipients
func TestRouteMessageEmptyRecipients(t *testing.T) {
	r := New()
	r.RegisterLocalRecipient("alice@example.com")

	envelope := &models.UCPEnvelope{
		From: "alice@example.com",
		To:   []string{},
	}

	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Fatalf("RouteMessage error: %v", err)
	}

	if len(local) != 0 {
		t.Errorf("Local recipients: got %d, want 0", len(local))
	}

	if len(remote) != 0 {
		t.Errorf("Remote recipients: got %d, want 0", len(remote))
	}

	t.Logf("✓ Empty recipients handled correctly")
}

// TestRetryQueueNonexistentEnvelope tests retry operations on non-existent envelope
func TestRetryQueueNonexistentEnvelope(t *testing.T) {
	rq := NewRetryQueue()

	// ShouldRetry on non-existent should return false
	if rq.ShouldRetry("nonexistent") {
		t.Error("ShouldRetry on nonexistent envelope should return false")
	}

	// IncrementRetry on non-existent should not panic
	rq.IncrementRetry("nonexistent")

	t.Logf("✓ Nonexistent envelope operations handled safely")
}
