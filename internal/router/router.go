// Package router handles federation, message routing to local/remote recipients, and retry logic.
package router

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// Router manages message routing to local and remote recipients.
type Router struct {
	localRecipients map[string]bool
	remoteServers   map[string]*FederationConnection
	serverKey       ed25519.PrivateKey // Server's Ed25519 private key for mutual auth
	serverID        string              // Server domain identifier
}

// New creates a new Router.
func New() *Router {
	return &Router{
		localRecipients: make(map[string]bool),
		remoteServers:   make(map[string]*FederationConnection),
	}
}

// NewWithServer creates a Router with server credentials for mutual auth.
func NewWithServer(serverID string, serverKey ed25519.PrivateKey) *Router {
	return &Router{
		localRecipients: make(map[string]bool),
		remoteServers:   make(map[string]*FederationConnection),
		serverID:        serverID,
		serverKey:       serverKey,
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

// Federation protocol messages for mutual authentication
type UCPFedChallenge struct {
	Challenge string `json:"challenge"` // Base64-encoded 32-byte challenge
}

type UCPFedProof struct {
	Challenge string `json:"challenge"` // Base64-encoded challenge
	Signature string `json:"signature"` // Base64-encoded Ed25519 signature
	ServerID  string `json:"server_id"`
}

type UCPDeliver struct {
	Envelope   json.RawMessage `json:"envelope"`
	ServerSig  string          `json:"server_sig"`
}

type UCPAck struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// EstablishFederation establishes a federation connection with mutual authentication.
func (r *Router) EstablishFederation(domain string) (*FederationConnection, error) {
	// Generate random challenge
	challengeBytes := make([]byte, 32)
	_, err := rand.Read(challengeBytes)
	if err != nil {
		return nil, fmt.Errorf("generate challenge: %w", err)
	}
	challenge := base64.StdEncoding.EncodeToString(challengeBytes)

	// Issue challenge to remote server
	remoteURL := fmt.Sprintf("https://%s/.well-known/ucp/federation/challenge", domain)
	resp, err := http.Post(remoteURL, "application/json", bytes.NewBuffer(mustMarshalJSON(UCPFedChallenge{Challenge: challenge})))
	if err != nil {
		return nil, fmt.Errorf("connect to remote server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote server rejected connection: %d", resp.StatusCode)
	}

	// Receive challenge from remote server
	var remoteChallenge UCPFedChallenge
	err = json.NewDecoder(resp.Body).Decode(&remoteChallenge)
	if err != nil {
		return nil, fmt.Errorf("decode remote challenge: %w", err)
	}

	// Sign remote challenge
	remoteBytes, err := base64.StdEncoding.DecodeString(remoteChallenge.Challenge)
	if err != nil {
		return nil, fmt.Errorf("decode remote challenge: %w", err)
	}

	proof := ed25519.Sign(r.serverKey, remoteBytes)
	proofB64 := base64.StdEncoding.EncodeToString(proof)

	// Send proof to remote server
	proofPayload := UCPFedProof{
		Challenge: remoteChallenge.Challenge,
		Signature: proofB64,
		ServerID:  r.serverID,
	}

	resp2, err := http.Post(
		fmt.Sprintf("https://%s/.well-known/ucp/federation/proof", domain),
		"application/json",
		bytes.NewBuffer(mustMarshalJSON(proofPayload)),
	)
	if err != nil {
		return nil, fmt.Errorf("send proof: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proof rejected: %d", resp2.StatusCode)
	}

	// Connection established
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

// ListConnections returns all active federation connections.
func (r *Router) ListConnections() []*FederationConnection {
	conns := make([]*FederationConnection, 0, len(r.remoteServers))
	for _, conn := range r.remoteServers {
		conns = append(conns, conn)
	}
	return conns
}

// ListRetries returns all pending delivery attempts from the queue.
func (rq *RetryQueue) ListAttempts() []*DeliveryAttempt {
	attempts := make([]*DeliveryAttempt, 0, len(rq.attempts))
	for _, attempt := range rq.attempts {
		attempts = append(attempts, attempt)
	}
	return attempts
}

// DeliverMessage sends an envelope to a remote server via federation.
func (r *Router) DeliverMessage(ctx context.Context, remoteDomain string, envelopeJSON json.RawMessage) error {
	// Ensure federation connection established
	conn, err := r.GetFederationConnection(remoteDomain)
	if err != nil {
		return fmt.Errorf("establish federation: %w", err)
	}

	// Sign the envelope with server key (proof of origin)
	sig := ed25519.Sign(r.serverKey, envelopeJSON)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	// Build delivery payload
	delivery := UCPDeliver{
		Envelope:  envelopeJSON,
		ServerSig: sigB64,
	}

	payload, err := json.Marshal(delivery)
	if err != nil {
		return fmt.Errorf("marshal delivery: %w", err)
	}

	// POST to remote server's federation endpoint
	remoteURL := fmt.Sprintf("https://%s/.well-known/ucp/federation/deliver", conn.Domain)
	req, err := http.NewRequestWithContext(ctx, "POST", remoteURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver to %s: %w", conn.Domain, err)
	}
	defer resp.Body.Close()

	// Check for success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("delivery failed: %d - %v", resp.StatusCode, errResp)
	}

	// Verify ACK
	var ack UCPAck
	err = json.NewDecoder(resp.Body).Decode(&ack)
	if err != nil || !ack.Success {
		return fmt.Errorf("remote server nacked delivery: %v", ack.Message)
	}

	return nil
}

// ProcessRetryQueue sends pending messages from the retry queue.
func (rq *RetryQueue) ProcessRetryQueue(ctx context.Context, router *Router) {
	for envelopeID := range rq.attempts {
		if rq.ShouldRetry(envelopeID) {
			// In production: fetch envelope from database and retry delivery
			// For now: mark retry scheduled
			rq.IncrementRetry(envelopeID)
		}
	}
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

// mustMarshalJSON marshals to JSON or panics
func mustMarshalJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
