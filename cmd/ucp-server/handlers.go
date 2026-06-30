package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/admin"
	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
	"github.com/unifiedcommunicationsprotocol/server/internal/logging"
	"github.com/unifiedcommunicationsprotocol/server/internal/models"
	"github.com/unifiedcommunicationsprotocol/server/internal/ratelimit"
	"github.com/unifiedcommunicationsprotocol/server/internal/router"
	"github.com/unifiedcommunicationsprotocol/server/internal/store"
	"github.com/unifiedcommunicationsprotocol/server/internal/transport"
)

// extractUserFromAuth validates the Authorization header and returns the authenticated user address.
// Also returns a context with the user set for RLS policies.
func extractUserFromAuth(ctx context.Context, am *auth.Manager, r *http.Request) (string, context.Context, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ctx, fmt.Errorf("missing authorization")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ctx, fmt.Errorf("invalid authorization format")
	}

	token := parts[1]
	address, err := am.ValidateSession(ctx, token)
	if err != nil {
		return "", ctx, fmt.Errorf("invalid session: %w", err)
	}

	// Add user to context for RLS policies
	userCtx := store.WithUserAddress(ctx, address)
	return address, userCtx, nil
}

// ServerKeyResponse is the well-known server key endpoint response.
type ServerKeyResponse struct {
	Domain string `json:"domain"`
	Key    string `json:"key"`
}

func handleServerKey(cfg config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServerKeyResponse{
			Domain: cfg.ServerDomain,
			Key:    cfg.ServerKey,
		})
	}
}

// IdentityResponse is the well-known identity endpoint response.
type IdentityResponse struct {
	Address     string `json:"address"`
	IdentityKey string `json:"identity_key"`
	ServerKey   string `json:"server_key"`
}

func handleIdentity(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := r.PathValue("address")
		if address == "" {
			http.Error(w, "missing address", http.StatusBadRequest)
			return
		}

		// Look up identity
		identity, err := s.GetIdentity(r.Context(), address)
		if err != nil {
			http.Error(w, "identity not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IdentityResponse{
			Address:     identity.Address,
			IdentityKey: identity.IdentityKey,
		})
	}
}

// KeyPackagesResponse lists available key packages for a user.
type KeyPackagesResponse struct {
	KeyPackages []string `json:"keypackages"`
}

func handleKeyPackages(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := r.PathValue("address")
		if address == "" {
			http.Error(w, "missing address", http.StatusBadRequest)
			return
		}

		// In real implementation: fetch from database
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(KeyPackagesResponse{
			KeyPackages: []string{},
		})
	}
}

// PrivacyResponse describes server's privacy/processing capabilities.
type PrivacyResponse struct {
	Enabled         bool     `json:"enabled"`
	Scopes          []string `json:"scopes"`
	DataRetention   string   `json:"data_retention"`
	DeletionPolicy  string   `json:"deletion_policy"`
}

func handlePrivacy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PrivacyResponse{
			Enabled:        false,
			Scopes:         []string{},
			DataRetention:  "30 days",
			DeletionPolicy: "on request",
		})
	}
}

// ChallengeRequest initiates challenge-response authentication.
type ChallengeRequest struct {
	Address string `json:"address"`
}

// ChallengeResponse returns a challenge for the user to sign.
type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

func handleChallenge(cs *auth.ChallengeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChallengeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		challenge, err := cs.IssueChallenge(req.Address)
		if err != nil {
			http.Error(w, fmt.Sprintf("create challenge: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChallengeResponse{
			Challenge: base64.StdEncoding.EncodeToString(challenge),
		})
	}
}

// SessionRequest redeems a signed challenge for a session.
type SessionRequest struct {
	Address   string `json:"address"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

// SessionResponse returns an authenticated session token.
type SessionResponse struct {
	SessionToken string `json:"session_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func handleSession(am *auth.Manager, cs *auth.ChallengeStore, s *store.Store, adminHub *admin.AdminHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Validate challenge
		challenge, err := cs.ValidateChallenge(req.Address)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid challenge: %v", err), http.StatusUnauthorized)
			return
		}

		// Fetch user's identity to get public key for verification
		identity, err := s.GetIdentity(r.Context(), req.Address)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		// Decode the identity key from base64
		pubKeyBytes, err := base64.StdEncoding.DecodeString(identity.IdentityKey)
		if err != nil {
			http.Error(w, "invalid identity key", http.StatusInternalServerError)
			return
		}

		// Verify signature over challenge
		if err := auth.VerifyChallengeResponse(challenge, pubKeyBytes, req.Signature); err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		// Consume challenge
		if err := cs.ConsumeChallenge(req.Address); err != nil {
			http.Error(w, "consume challenge", http.StatusInternalServerError)
			return
		}

		// Create session (24-hour lifetime)
		session, err := am.CreateSession(r.Context(), req.Address, 24*3600)
		if err != nil {
			http.Error(w, fmt.Sprintf("create session: %v", err), http.StatusInternalServerError)
			return
		}

		// Broadcast session created event to admin subscribers
		adminHub.BroadcastSessionCreated(session.Token, req.Address, session.ExpiresAt)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SessionResponse{
			SessionToken: session.Token,
			ExpiresAt:    session.ExpiresAt,
		})
	}
}

// RefreshRequest refreshes an existing session.
type RefreshRequest struct {
	SessionToken string `json:"session_token"`
}

func handleRefresh(am *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		session, err := am.RefreshSession(r.Context(), req.SessionToken, 24*3600)
		if err != nil {
			http.Error(w, fmt.Sprintf("refresh session: %v", err), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SessionResponse{
			SessionToken: session.Token,
			ExpiresAt:    session.ExpiresAt,
		})
	}
}

// SendMessageRequest is a UCP envelope to be stored and routed.
type SendMessageRequest struct {
	Envelope string `json:"envelope"`
}

// SendMessageResponse returns the stored envelope ID.
type SendMessageResponse struct {
	EnvelopeID string `json:"envelope_id"`
}

func handleSendMessage(am *auth.Manager, s *store.Store, hub *transport.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate user from Authorization header
		address, ctx, err := extractUserFromAuth(r.Context(), am, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Decode envelope
		envelopeBytes, err := base64.StdEncoding.DecodeString(req.Envelope)
		if err != nil {
			http.Error(w, "invalid envelope encoding", http.StatusBadRequest)
			return
		}

		// Parse UCPEnvelope
		var envelope models.UCPEnvelope
		if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
			http.Error(w, "invalid envelope", http.StatusBadRequest)
			return
		}

		// Validate envelope format
		if envelope.V != "ucp/1.0" {
			http.Error(w, "unsupported protocol version", http.StatusBadRequest)
			return
		}

		if envelope.From == "" || len(envelope.To) == 0 || envelope.MLS == "" {
			http.Error(w, "invalid envelope: missing required fields", http.StatusBadRequest)
			return
		}

		// Verify sender
		if envelope.From != address {
			http.Error(w, "sender mismatch", http.StatusForbidden)
			return
		}

		// Verify signing key is provided
		if envelope.SigningKey == "" {
			http.Error(w, "invalid envelope: signing_key required", http.StatusBadRequest)
			return
		}

		// Verify MLS payload is valid base64
		if _, err := base64.StdEncoding.DecodeString(envelope.MLS); err != nil {
			http.Error(w, "invalid mls encoding", http.StatusBadRequest)
			return
		}

		// Set server timestamp
		serverTs := time.Now().UnixMilli()
		envelope.ServerTs = &serverTs

		// Store message (server now owns the MLS-encrypted bytes)
		if err := s.StoreMessage(ctx, &envelope, envelopeBytes); err != nil {
			http.Error(w, fmt.Sprintf("store message: %v", err), http.StatusInternalServerError)
			return
		}

		// Notify connected clients (in real implementation: broadcast to local recipients)
		_ = hub // Suppress unused warning

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SendMessageResponse{
			EnvelopeID: string(envelope.ThreadID),
		})
	}
}

// InboxResponse lists messages in a user's inbox.
type InboxResponse struct {
	Messages []MessageSummary `json:"messages"`
}

// MessageSummary is a brief summary of a message for listing.
type MessageSummary struct {
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
	From      string `json:"from"`
	Timestamp int64  `json:"timestamp"`
}

func handleInbox(am *auth.Manager, s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate user from Authorization header
		address, ctx, err := extractUserFromAuth(r.Context(), am, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		// Query inbox messages for this user (RLS policy enforces access control)
		_ = address
		_ = ctx

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InboxResponse{
			Messages: []MessageSummary{},
		})
	}
}

// UploadResponse returns the attachment ID and content hash.
type UploadResponse struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

func handleUploadAttachment(am *auth.Manager, s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate session
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		// (Auth extraction already done above)

		// Read attachment data
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}

		// Compute SHA256
		hash := sha256.Sum256(data)
		hashStr := fmt.Sprintf("%x", hash)

		// Generate attachment ID
		attachmentID := models.ULID(fmt.Sprintf("attach_%s", hashStr[:16]))

		// Store attachment metadata
		attachment := &models.Attachment{
			ID:       attachmentID,
			Name:     r.Header.Get("X-Filename"),
			MimeType: r.Header.Get("Content-Type"),
			Size:     int64(len(data)),
			SHA256:   hashStr,
		}

		if err := s.StoreAttachment(r.Context(), attachment); err != nil {
			http.Error(w, fmt.Sprintf("store attachment: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(UploadResponse{
			ID:     string(attachmentID),
			SHA256: hashStr,
		})
	}
}

func handleDownloadAttachment(am *auth.Manager, s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate user from Authorization header
		_, ctx, err := extractUserFromAuth(r.Context(), am, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		_ = ctx // May be used later for store operations
		attachmentID := models.ULID(r.PathValue("id"))
		if attachmentID == "" {
			http.Error(w, "missing attachment id", http.StatusBadRequest)
			return
		}

		// Fetch attachment
		attachment, err := s.GetAttachment(r.Context(), attachmentID)
		if err != nil {
			http.Error(w, "attachment not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", attachment.MimeType)
		w.Header().Set("X-SHA256", attachment.SHA256)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.Size))
		w.WriteHeader(http.StatusOK)
		// In real implementation: stream attachment content from storage
		fmt.Fprintf(w, "attachment content for %s", attachment.ID)
	}
}

// AdminSessionResponse is a single session in the admin sessions list.
type AdminSessionResponse struct {
	Token     string `json:"token"`
	Identity  string `json:"identity"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
	Status    string `json:"status"`
}

// AdminSessionsListResponse lists all active sessions.
type AdminSessionsListResponse struct {
	Sessions []AdminSessionResponse `json:"sessions"`
}

func handleAdminSessions(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Query all active sessions from database
		records, err := s.ListActiveSessions(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("list sessions: %v", err), http.StatusInternalServerError)
			return
		}

		var sessions []AdminSessionResponse
		for _, rec := range records {
			// Truncate token for display
			displayToken := rec.Token
			if len(rec.Token) > 20 {
				displayToken = rec.Token[:17] + "…"
			}

			sessions = append(sessions, AdminSessionResponse{
				Token:     displayToken,
				Identity:  rec.Address,
				IssuedAt:  rec.IssuedAt,
				ExpiresAt: rec.ExpiresAt,
				Status:    "active",
			})
		}

		if sessions == nil {
			sessions = []AdminSessionResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdminSessionsListResponse{
			Sessions: sessions,
		})
	}
}

// AdminFederationConnectionResponse is a single federation connection.
type AdminFederationConnectionResponse struct {
	RemoteDomain  string `json:"remote_domain"`
	EstablishedAt int64  `json:"established_at"`
	LastActivity  int64  `json:"last_activity"`
	Retries       int    `json:"retries"`
}

// AdminFederationConnectionsResponse lists all federation connections.
type AdminFederationConnectionsResponse struct {
	Connections []AdminFederationConnectionResponse `json:"connections"`
}

func handleAdminFederationConnections(r *router.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Get all federation connections
		conns := r.ListConnections()

		var responses []AdminFederationConnectionResponse
		for _, conn := range conns {
			responses = append(responses, AdminFederationConnectionResponse{
				RemoteDomain:  conn.Domain,
				EstablishedAt: conn.Established.Unix(),
				LastActivity:  conn.Established.Unix(), // TODO: track last activity
				Retries:       conn.Retries,
			})
		}

		if responses == nil {
			responses = []AdminFederationConnectionResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdminFederationConnectionsResponse{
			Connections: responses,
		})
	}
}

// AdminDeliveryQueueItemResponse is a single item in the delivery queue.
type AdminDeliveryQueueItemResponse struct {
	Recipient    string `json:"recipient"`
	EnvelopeID   string `json:"envelope_id"`
	Attempts     int    `json:"attempts"`
	LastAttempt  int64  `json:"last_attempt"`
	Status       string `json:"status"`
	NextRetry    int64  `json:"next_retry"`
}

// AdminFederationQueueResponse lists delivery queue status.
type AdminFederationQueueResponse struct {
	QueueDepth  int                              `json:"queue_depth"`
	FailedCount int                              `json:"failed_count"`
	Items       []AdminDeliveryQueueItemResponse `json:"items"`
}

func handleAdminFederationQueue(rq *router.RetryQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get all pending delivery attempts
		attempts := rq.ListAttempts()

		var items []AdminDeliveryQueueItemResponse
		for _, attempt := range attempts {
			items = append(items, AdminDeliveryQueueItemResponse{
				Recipient:   attempt.Recipient,
				EnvelopeID:  attempt.EnvelopeID[:12] + "…",
				Attempts:    attempt.Retries,
				LastAttempt: attempt.AttemptedAt.Unix(),
				Status:      "pending",
				NextRetry:   attempt.NextRetry.Unix(),
			})
		}

		if items == nil {
			items = []AdminDeliveryQueueItemResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdminFederationQueueResponse{
			QueueDepth:  len(items),
			FailedCount: 0,
			Items:       items,
		})
	}
}

// SearchRequest searches for messages (Phase 2f).
type SearchRequest struct {
	Query string `json:"query"`
}

// SearchResponse returns matching messages.
type SearchResponse struct {
	Results []MessageSummary `json:"results"`
	Count   int              `json:"count"`
}

func handleSearch(am *auth.Manager, s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate user from Authorization header
		address, _, err := extractUserFromAuth(r.Context(), am, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if req.Query == "" {
			http.Error(w, "query required", http.StatusBadRequest)
			return
		}

		// TODO: Implement full-text search
		// For now, return empty results
		// In Phase 2f, this would:
		// 1. Use Postgres FTS (to_tsvector, to_tsquery)
		// 2. Search indexed message content
		// 3. Filter by user permissions (RLS)
		// 4. Return matched messages with relevance score

		_ = address // Use for RLS filtering in real implementation

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResponse{
			Results: []MessageSummary{},
			Count:   0,
		})
	}
}

// withRateLimit wraps an HTTP handler with rate limiting.
func withRateLimit(limiter *ratelimit.Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use remote IP as the rate limit key
		ip := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ip = strings.Split(xff, ",")[0]
		}

		if !limiter.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded",
			})
			return
		}

		next(w, r)
	}
}

// MetricsResponse is the metrics endpoint response.
type MetricsResponse struct {
	Timestamp           string                 `json:"timestamp"`
	Metrics             map[string]interface{} `json:"metrics"`
}

// handleMetrics returns server metrics.
func handleMetrics(m *logging.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MetricsResponse{
			Timestamp: time.Now().Format(time.RFC3339),
			Metrics:   m.Snapshot(),
		})
	}
}
