package logging

import (
	"testing"
)

func TestLogger(t *testing.T) {
	logger := New(LevelInfo)

	// These should not panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

func TestMetrics(t *testing.T) {
	m := &Metrics{}

	m.RecordMessage(false)
	m.RecordMessage(true)
	m.RecordAuth(false)
	m.RecordAuth(true)
	m.RecordAttachment()
	m.RecordError("test error")

	snapshot := m.Snapshot()

	if snapshot["messages_received"] != int64(1) {
		t.Errorf("messages_received: got %v, want 1", snapshot["messages_received"])
	}

	if snapshot["messages_sent"] != int64(1) {
		t.Errorf("messages_sent: got %v, want 1", snapshot["messages_sent"])
	}

	if snapshot["auth_challenges"] != int64(1) {
		t.Errorf("auth_challenges: got %v, want 1", snapshot["auth_challenges"])
	}

	if snapshot["auth_sessions"] != int64(1) {
		t.Errorf("auth_sessions: got %v, want 1", snapshot["auth_sessions"])
	}

	if snapshot["attachments_uploaded"] != int64(1) {
		t.Errorf("attachments_uploaded: got %v, want 1", snapshot["attachments_uploaded"])
	}

	if snapshot["errors"] != int64(1) {
		t.Errorf("errors: got %v, want 1", snapshot["errors"])
	}
}
