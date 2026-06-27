package router

import (
	"testing"

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
