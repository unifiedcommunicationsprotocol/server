package ai

import (
	"testing"
)

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		valid    bool
	}{
		{"work", true},
		{"personal", true},
		{"newsletter", true},
		{"notification", true},
		{"transactional", true},
		{"social", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsValidCategory(tt.category)
		if got != tt.valid {
			t.Errorf("IsValidCategory(%q): got %v, want %v", tt.category, got, tt.valid)
		}
	}
}

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		scope string
		valid bool
	}{
		{"search", true},
		{"summary", true},
		{"routing", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsValidScope(tt.scope)
		if got != tt.valid {
			t.Errorf("IsValidScope(%q): got %v, want %v", tt.scope, got, tt.valid)
		}
	}
}

func TestInferMetadata(t *testing.T) {
	content := "This is a test message about work"

	metadata := InferMetadata(content)

	if metadata == nil {
		t.Error("InferMetadata should return metadata")
	}

	if metadata.Summary == "" {
		t.Error("Summary should not be empty")
	}

	if !IsValidCategory(metadata.Category) {
		t.Error("Category should be valid")
	}
}

func TestStaleKeyShareHandler(t *testing.T) {
	h := NewStaleKeyShareHandler()

	groupID := "group_123"

	if h.HasStaleShare(groupID) {
		t.Error("Should not have stale share initially")
	}

	h.RecordStaleShare(groupID)

	if !h.HasStaleShare(groupID) {
		t.Error("Should have stale share after recording")
	}

	shares := h.GetStaleShares()
	if len(shares) != 1 {
		t.Errorf("GetStaleShares: got %d, want 1", len(shares))
	}

	h.RemoveStaleShare(groupID)

	if h.HasStaleShare(groupID) {
		t.Error("Should not have stale share after removal")
	}
}

func TestMetadata(t *testing.T) {
	priority := 2
	metadata := &Metadata{
		Summary:  "Test summary",
		Category: "work",
		Priority: &priority,
	}

	if metadata.Summary != "Test summary" {
		t.Error("Summary mismatch")
	}

	if !IsValidCategory(metadata.Category) {
		t.Error("Invalid category")
	}

	if metadata.Priority == nil || *metadata.Priority != 2 {
		t.Error("Priority mismatch")
	}
}

func TestPromoteMetadata(t *testing.T) {
	metadata := &Metadata{
		Summary:  "Local inference",
		Category: "personal",
	}

	promoted := PromoteMetadata(metadata)

	if promoted.Summary != metadata.Summary {
		t.Error("Promoted metadata mismatch")
	}
}
