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
}
