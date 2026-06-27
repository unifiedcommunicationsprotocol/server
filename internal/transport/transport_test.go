package transport

import (
	"testing"
	"time"
)

func TestRegisterConnection(t *testing.T) {
	h := New()

	conn := &Connection{
		ID:      "conn_1",
		Address: "alice@example.com",
	}

	if err := h.RegisterConnection(conn); err != nil {
		t.Fatalf("RegisterConnection error: %v", err)
	}

	retrieved, err := h.GetConnection("conn_1")
	if err != nil {
		t.Fatalf("GetConnection error: %v", err)
	}

	if retrieved.Address != "alice@example.com" {
		t.Errorf("Address mismatch: got %q, want %q", retrieved.Address, "alice@example.com")
	}
}

func TestGetConnectionsByAddress(t *testing.T) {
	h := New()

	h.RegisterConnection(&Connection{ID: "conn_1", Address: "alice@example.com"})
	h.RegisterConnection(&Connection{ID: "conn_2", Address: "alice@example.com"})
	h.RegisterConnection(&Connection{ID: "conn_3", Address: "bob@example.com"})

	aliceConns := h.GetConnectionsByAddress("alice@example.com")
	if len(aliceConns) != 2 {
		t.Errorf("Expected 2 connections for alice, got %d", len(aliceConns))
	}

	bobConns := h.GetConnectionsByAddress("bob@example.com")
	if len(bobConns) != 1 {
		t.Errorf("Expected 1 connection for bob, got %d", len(bobConns))
	}
}

func TestUnregisterConnection(t *testing.T) {
	h := New()

	h.RegisterConnection(&Connection{ID: "conn_1", Address: "alice@example.com"})

	h.UnregisterConnection("conn_1")

	_, err := h.GetConnection("conn_1")
	if err == nil {
		t.Error("UnregisterConnection should remove connection")
	}
}

func TestValidateUCPHello(t *testing.T) {
	cm := NewConnectionManager(New())

	hello := map[string]interface{}{
		"version": "ucp/1.0",
	}

	version, err := cm.ValidateUCPHello(hello)
	if err != nil {
		t.Fatalf("ValidateUCPHello error: %v", err)
	}

	if version != "ucp/1.0" {
		t.Errorf("Version: got %q, want %q", version, "ucp/1.0")
	}
}

func TestValidateUCPHelloWrongVersion(t *testing.T) {
	cm := NewConnectionManager(New())

	hello := map[string]interface{}{
		"version": "ucp/2.0",
	}

	_, err := cm.ValidateUCPHello(hello)
	if err == nil {
		t.Error("ValidateUCPHello should reject unsupported version")
	}
}

func TestCreateUCPHelloAck(t *testing.T) {
	cm := NewConnectionManager(New())

	ack := cm.CreateUCPHelloAck("token_abc", "ucp.example.com")

	if ack["version"] != "ucp/1.0" {
		t.Error("Version mismatch in hello ack")
	}

	if ack["server_id"] != "ucp.example.com" {
		t.Error("ServerID mismatch in hello ack")
	}
}

func TestKeepalive(t *testing.T) {
	k := NewKeepalive()

	now := time.Now()

	// After 35 seconds - should send ping (interval is 30s)
	if !k.ShouldSendPing(now.Add(-35 * time.Second)) {
		t.Error("ShouldSendPing should return true after 35 seconds")
	}

	// Too recent (5 seconds) - should not send ping
	if k.ShouldSendPing(now.Add(-5 * time.Second)) {
		t.Error("ShouldSendPing should return false within interval")
	}

	// Connection alive check (35 seconds - within timeout)
	if !k.IsConnectionAlive(now.Add(-35 * time.Second)) {
		t.Error("IsConnectionAlive should return true after 35 seconds (interval + some delay)")
	}

	// Connection dead (50+ seconds > 30s interval + 10s timeout)
	if k.IsConnectionAlive(now.Add(-50 * time.Second)) {
		t.Error("IsConnectionAlive should return false after 50 seconds")
	}
}

func TestBackoffReconnect(t *testing.T) {
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 60s, 60s...
	expected := []time.Duration{
		1 * time.Second,   // 1 << 0
		2 * time.Second,   // 1 << 1
		4 * time.Second,   // 1 << 2
		8 * time.Second,   // 1 << 3
		16 * time.Second,  // 1 << 4
		32 * time.Second,  // 1 << 5
		60 * time.Second,  // 1 << 6 = 64, capped at 60
		60 * time.Second,  // 1 << 7 = 128, capped at 60
	}

	for i, exp := range expected {
		got := BackoffReconnect(i)
		if got != exp {
			t.Errorf("Backoff[%d]: got %v, want %v", i, got, exp)
		}
	}
}

func TestEncodeDecodeFrame(t *testing.T) {
	frame := &Frame{
		Type:    "application",
		Payload: []byte(`{"id":"01J3K"}`),
		Seq:     1,
	}

	// Encode
	encoded, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("EncodeFrame error: %v", err)
	}

	// Decode
	decoded, err := DecodeFrame(encoded)
	if err != nil {
		t.Fatalf("DecodeFrame error: %v", err)
	}

	if decoded.Type != "application" {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Type, "application")
	}

	if decoded.Seq != 1 {
		t.Errorf("Seq mismatch: got %d, want 1", decoded.Seq)
	}
}
