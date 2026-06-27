// Package ai manages AI metadata surface, zero-knowledge defaults, and opt-in server processing with key shares.
package ai

// Metadata represents AI-generated or sender-supplied metadata.
type Metadata struct {
	Summary    string    `json:"summary,omitempty"`
	Category   string    `json:"category,omitempty"`
	Priority   *int      `json:"priority,omitempty"`
	Embeddings []float32 `json:"embeddings,omitempty"`
}

// Categories are defined AI metadata categories.
var Categories = []string{
	"work",
	"personal",
	"newsletter",
	"notification",
	"transactional",
	"social",
}

// IsValidCategory checks if a category is defined.
func IsValidCategory(category string) bool {
	for _, c := range Categories {
		if c == category {
			return true
		}
	}
	return false
}

// KeyShare represents an MLS key share for server processing.
type KeyShare struct {
	GroupID string
	Epoch   int
	Key     string
}

// ServerProcessing declares opt-in server-side decryption.
type ServerProcessing struct {
	Enabled   bool
	Scopes    []string
	GrantedAt *int64
}

// IsValidScope checks if a scope is authorized for server processing.
func IsValidScope(scope string) bool {
	validScopes := []string{"search", "summary", "routing"}
	for _, s := range validScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// InferMetadata generates metadata locally (placeholder for AI inference).
func InferMetadata(messageContent string) *Metadata {
	// In reality: local inference model to generate summary, category, priority
	return &Metadata{
		Summary:  "Message summary",
		Category: "personal",
		Priority: intPtr(3),
	}
}

// PromoteMetadata promotes locally-generated metadata to outbound payload.
func PromoteMetadata(localMetadata *Metadata) *Metadata {
	// On forward/quote, include locally-generated metadata in outbound message
	return localMetadata
}

// StaleKeyShareHandler manages stale key share recovery.
type StaleKeyShareHandler struct {
	staleShares map[string]int64
}

// NewStaleKeyShareHandler creates a new handler.
func NewStaleKeyShareHandler() *StaleKeyShareHandler {
	return &StaleKeyShareHandler{
		staleShares: make(map[string]int64),
	}
}

// RecordStaleShare records a stale key share for a group.
func (h *StaleKeyShareHandler) RecordStaleShare(groupID string) {
	// Record timestamp for later cleanup (7-day window)
	h.staleShares[groupID] = 0 // In reality: current unix timestamp
}

// HasStaleShare checks if a group has a stale share.
func (h *StaleKeyShareHandler) HasStaleShare(groupID string) bool {
	_, ok := h.staleShares[groupID]
	return ok
}

// GetStaleShares returns all groups with stale shares.
func (h *StaleKeyShareHandler) GetStaleShares() []string {
	var shares []string
	for groupID := range h.staleShares {
		shares = append(shares, groupID)
	}
	return shares
}

// RemoveStaleShare removes a stale share (after re-sharing).
func (h *StaleKeyShareHandler) RemoveStaleShare(groupID string) {
	delete(h.staleShares, groupID)
}

// PrivacyPolicy describes server processing privacy and policies.
type PrivacyPolicy struct {
	Enabled    bool
	Scopes     []string
	DataRetention string
	DeletionPolicy string
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}
