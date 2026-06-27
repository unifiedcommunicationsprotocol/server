package bridge

import (
	"testing"
)

func TestConnectIMAP(t *testing.T) {
	b := New()

	conn, err := b.ConnectIMAP("acc_1", "imap.gmail.com", 993, "user@gmail.com")
	if err != nil {
		t.Fatalf("ConnectIMAP error: %v", err)
	}

	if !conn.Connected {
		t.Error("Connection should be marked as connected")
	}

	if conn.Host != "imap.gmail.com" {
		t.Error("Host mismatch")
	}
}

func TestConnectSMTP(t *testing.T) {
	b := New()

	client, err := b.ConnectSMTP("acc_1", "smtp.gmail.com", 587, "user@gmail.com")
	if err != nil {
		t.Fatalf("ConnectSMTP error: %v", err)
	}

	if !client.Connected {
		t.Error("Client should be marked as connected")
	}
}

func TestThreadingMap(t *testing.T) {
	tm := NewThreadingMap()

	smtpID := "<msg123@gmail.com>"
	ucpID := "01J3K..."

	tm.MapSMTPToUCP(smtpID, ucpID)

	retrieved, err := tm.GetUCPID(smtpID)
	if err != nil {
		t.Fatalf("GetUCPID error: %v", err)
	}

	if retrieved != ucpID {
		t.Errorf("Retrieved ID mismatch: got %q, want %q", retrieved, ucpID)
	}

	smtpRetrieved, err := tm.GetSMTPID(ucpID)
	if err != nil {
		t.Fatalf("GetSMTPID error: %v", err)
	}

	if smtpRetrieved != smtpID {
		t.Error("Reverse mapping mismatch")
	}
}

func TestConverter(t *testing.T) {
	c := NewConverter()

	mimeData := []byte("test")
	msg, err := c.ConvertMIMEToUCP(mimeData, "sender@example.com")
	if err != nil {
		t.Fatalf("ConvertMIMEToUCP error: %v", err)
	}

	if msg["type"] != "message.email" {
		t.Error("Message type mismatch")
	}

	t.Log("✓ Converter works")
}

// TestThreadingEngine tests IMAP message threading
func TestThreadingEngine(t *testing.T) {
	te := NewThreadingEngine()

	// Map a message
	threadID, err := te.MapMessage("msg-123@gmail.com", "Test Subject", "", 1234567890)
	if err != nil {
		t.Fatalf("MapMessage error: %v", err)
	}

	if threadID == "" {
		t.Error("Thread ID should not be empty")
	}

	// Retrieve thread ID
	retrieved, found := te.GetThreadID("msg-123@gmail.com")
	if !found {
		t.Error("Thread ID not found")
	}

	if retrieved != threadID {
		t.Errorf("Thread ID mismatch: got %q, want %q", retrieved, threadID)
	}

	t.Logf("✓ Threading engine: derived thread ID %q", threadID)
}

// TestDeriveThreadID tests thread ID derivation
func TestDeriveThreadID(t *testing.T) {
	// DeriveThreadID strips "Re:" prefixes and creates deterministic hashes
	// Same subject should always produce same thread ID
	id1 := DeriveThreadID("Hello", "")
	id2 := DeriveThreadID("Re: Hello", "")
	id3 := DeriveThreadID("RE: Hello", "")

	if id1 != id2 {
		t.Errorf("'Hello' and 'Re: Hello' should have same thread ID")
	}

	if id1 != id3 {
		t.Errorf("'Hello' and 'RE: Hello' should have same thread ID")
	}

	// Different subjects should (usually) have different IDs
	id4 := DeriveThreadID("Different", "")
	if id1 == id4 {
		t.Error("Different subjects should (usually) have different IDs")
	}

	// Thread IDs should be consistently formatted
	if len(id1) == 0 {
		t.Error("Thread ID should not be empty")
	}

	if len(id1) < 10 { // "thread_" (7) + some hash
		t.Errorf("Thread ID seems too short: %q", id1)
	}

	t.Logf("✓ Thread ID derivation: 'Hello' → %q", id1)
}

// TestBridgeMultipleConnections tests managing multiple connections
func TestBridgeMultipleConnections(t *testing.T) {
	b := New()

	// Connect multiple IMAP accounts
	conn1, _ := b.ConnectIMAP("acc1", "imap1.example.com", 993, "user1@example.com")
	conn2, _ := b.ConnectIMAP("acc2", "imap2.example.com", 993, "user2@example.com")

	if conn1.AccountID != "acc1" {
		t.Error("Connection 1 account ID mismatch")
	}

	if conn2.AccountID != "acc2" {
		t.Error("Connection 2 account ID mismatch")
	}

	t.Logf("✓ Bridge manages %d IMAP connections", 2)
}

// TestThreadingMapRoundtrip tests bidirectional mapping
func TestThreadingMapRoundtrip(t *testing.T) {
	tm := NewThreadingMap()

	smtpID := "<msg@sender.com>"
	ucpID := "ucp-ulid-12345"

	tm.MapSMTPToUCP(smtpID, ucpID)

	// Forward lookup
	retrieved, _ := tm.GetUCPID(smtpID)
	if retrieved != ucpID {
		t.Error("Forward mapping failed")
	}

	// Reverse lookup
	retrieved, _ = tm.GetSMTPID(ucpID)
	if retrieved != smtpID {
		t.Error("Reverse mapping failed")
	}

	t.Log("✓ Bidirectional threading map works")
}

// TestConverterUCPToMIME tests UCP to MIME conversion
func TestConverterUCPToMIME(t *testing.T) {
	c := NewConverter()

	ucpMsg := map[string]interface{}{
		"type":    "message.email",
		"from":    "sender@example.com",
		"to":      []string{"recipient@example.com"},
		"subject": "Test",
		"body":    "Test body",
	}

	mime, err := c.ConvertUCPToMIME(ucpMsg)
	if err != nil {
		t.Fatalf("ConvertUCPToMIME error: %v", err)
	}

	if len(mime) == 0 {
		t.Error("MIME output should not be empty")
	}

	t.Logf("✓ UCP to MIME conversion produced %d bytes", len(mime))
}

// TestBridgeErrorHandling tests error cases
func TestBridgeErrorHandling(t *testing.T) {
	b := New()

	// Invalid IMAP config
	_, err := b.ConnectIMAP("acc", "", 993, "user")
	if err == nil {
		t.Error("Should error on empty host")
	}

	_, err = b.ConnectIMAP("acc", "host", -1, "user")
	if err == nil {
		t.Error("Should error on invalid port")
	}

	// Invalid SMTP config
	_, err = b.ConnectSMTP("acc", "", 587, "user")
	if err == nil {
		t.Error("Should error on empty host")
	}

	t.Log("✓ Error handling works")
}

// TestThreadingEngineEdgeCases tests edge cases
func TestThreadingEngineEdgeCases(t *testing.T) {
	te := NewThreadingEngine()

	// Empty message ID should error
	_, err := te.MapMessage("", "Subject", "", 1234567890)
	if err == nil {
		t.Error("Should error on empty message ID")
	}

	// Non-existent message should not be found
	_, found := te.GetThreadID("nonexistent@example.com")
	if found {
		t.Error("Should not find nonexistent message")
	}

	t.Log("✓ Edge cases handled correctly")
}
