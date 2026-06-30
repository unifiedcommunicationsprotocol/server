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
	m.RecordAuthChallenge()
	m.RecordAuthSession()
	m.RecordAttachmentUpload()
	m.RecordError("test error")

	snapshot := m.Snapshot()

	if snapshot["messages_received_total"] != int64(1) {
		t.Errorf("messages_received_total: got %v, want 1", snapshot["messages_received_total"])
	}

	if snapshot["messages_sent_total"] != int64(1) {
		t.Errorf("messages_sent_total: got %v, want 1", snapshot["messages_sent_total"])
	}

	if snapshot["auth_challenges_total"] != int64(1) {
		t.Errorf("auth_challenges_total: got %v, want 1", snapshot["auth_challenges_total"])
	}

	if snapshot["auth_sessions_total"] != int64(1) {
		t.Errorf("auth_sessions_total: got %v, want 1", snapshot["auth_sessions_total"])
	}

	if snapshot["attachments_uploaded_total"] != int64(1) {
		t.Errorf("attachments_uploaded_total: got %v, want 1", snapshot["attachments_uploaded_total"])
	}

	if snapshot["http_errors_total"] != int64(1) {
		t.Errorf("http_errors_total: got %v, want 1", snapshot["http_errors_total"])
	}
}
