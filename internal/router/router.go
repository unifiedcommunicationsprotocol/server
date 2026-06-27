// Package router handles federation, message routing to local/remote recipients, and retry logic.
package router

import (
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// Router manages message routing to local and remote recipients.
type Router struct {
	localRecipients map[string]bool
	remoteServers   map[string]*FederationConnection
}

// New creates a new Router.
func New() *Router {
	return &Router{
		localRecipients: make(map[string]bool),
		remoteServers:   make(map[string]*FederationConnection),
	}
}

// RegisterLocalRecipient registers a local address.
func (r *Router) RegisterLocalRecipient(address string) {
	r.localRecipients[address] = true
}

// IsLocalRecipient checks if an address is local.
func (r *Router) IsLocalRecipient(address string) bool {
	return r.localRecipients[address]
}

// RouteMessage determines where a message should go.
func (r *Router) RouteMessage(envelope *models.UCPEnvelope) (local []string, remote map[string][]string, err error) {
	local = []string{}
	remote = make(map[string][]string)

	for _, to := range envelope.To {
		if r.IsLocalRecipient(to) {
			local = append(local, to)
		} else {
			// In reality: resolve domain via DNS, get server address
			domain := extractDomain(to)
			remote[domain] = append(remote[domain], to)
		}
	}

	return local, remote, nil
}

// FederationConnection represents a connection to a remote server.
type FederationConnection struct {
	Domain string
	Established time.Time
	Retries int
}

// EstablishFederation establishes a federation connection.
func (r *Router) EstablishFederation(domain string) (*FederationConnection, error) {
	// In reality: perform mutual authentication handshake
	conn := &FederationConnection{
		Domain:      domain,
		Established: time.Now(),
	}
	r.remoteServers[domain] = conn
	return conn, nil
}

// GetFederationConnection retrieves or establishes a federation connection.
func (r *Router) GetFederationConnection(domain string) (*FederationConnection, error) {
	if conn, ok := r.remoteServers[domain]; ok {
		return conn, nil
	}
	return r.EstablishFederation(domain)
}

// DeliveryAttempt tracks a message delivery attempt.
type DeliveryAttempt struct {
	EnvelopeID string
	Recipient  string
	AttemptedAt time.Time
	NextRetry   time.Time
	Retries     int
}

// RetryQueue manages message retries.
type RetryQueue struct {
	attempts map[string]*DeliveryAttempt
}

// NewRetryQueue creates a new retry queue.
func NewRetryQueue() *RetryQueue {
	return &RetryQueue{
		attempts: make(map[string]*DeliveryAttempt),
	}
}

// EnqueueRetry enqueues a message for retry.
func (rq *RetryQueue) EnqueueRetry(envelopeID, recipient string) *DeliveryAttempt {
	attempt := &DeliveryAttempt{
		EnvelopeID: envelopeID,
		Recipient:  recipient,
		AttemptedAt: time.Now(),
		NextRetry:  time.Now().Add(1 * time.Minute), // Start with 1-minute backoff
		Retries:    1,
	}
	rq.attempts[envelopeID] = attempt
	return attempt
}

// ShouldRetry checks if an attempt should be retried.
func (rq *RetryQueue) ShouldRetry(envelopeID string) bool {
	attempt, ok := rq.attempts[envelopeID]
	if !ok {
		return false
	}

	// Max 48-hour retry window
	if time.Since(attempt.AttemptedAt) > 48*time.Hour {
		delete(rq.attempts, envelopeID)
		return false
	}

	return time.Now().After(attempt.NextRetry)
}

// IncrementRetry increases the retry count and schedules next retry.
func (rq *RetryQueue) IncrementRetry(envelopeID string) {
	attempt, ok := rq.attempts[envelopeID]
	if !ok {
		return
	}

	attempt.Retries++
	// Exponential backoff: 1m, 2m, 4m, ..., up to 4 hours
	backoff := time.Duration(1<<uint(attempt.Retries)) * time.Minute
	if backoff > 4*time.Hour {
		backoff = 4 * time.Hour
	}
	attempt.NextRetry = time.Now().Add(backoff)
}

func extractDomain(address string) string {
	// Extract domain from address@domain
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] == '@' {
			return address[i+1:]
		}
	}
	return ""
}
