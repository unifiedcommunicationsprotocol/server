package transport

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSyncHubRegistration(t *testing.T) {
	hub := NewSyncHub()

	conn := hub.RegisterConnection("alice@example.com")

	if conn == nil {
		t.Error("connection is nil")
	}

	if conn.Address != "alice@example.com" {
		t.Errorf("address: got %q, want %q", conn.Address, "alice@example.com")
	}
}

func TestSyncHubUnregistration(t *testing.T) {
	hub := NewSyncHub()

	hub.RegisterConnection("alice@example.com")
	hub.UnregisterConnection("alice@example.com")

	// Should not panic
}

func TestBroadcastMessage(t *testing.T) {
	hub := NewSyncHub()

	hub.RegisterConnection("alice@example.com")
	ch, err := hub.Subscribe("thread_123", "alice@example.com")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	msg := &RealtimeSyncMessage{
		Type:      "message",
		ThreadID:  "thread_123",
		From:      "bob@example.com",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"body": "hello alice",
		},
	}

	go hub.Broadcast(msg)

	select {
	case received := <-ch:
		if received.Type != "message" {
			t.Errorf("type: got %q, want %q", received.Type, "message")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for broadcast message")
	}
}

func TestBroadcastPresence(t *testing.T) {
	hub := NewSyncHub()

	hub.RegisterConnection("alice@example.com")
	ch, err := hub.Subscribe("presence", "alice@example.com")
	if err != nil {
		t.Fatalf("subscribe to presence: %v", err)
	}

	hub.BroadcastPresence("bob@example.com", true)

	select {
	case msg := <-ch:
		if msg.Type != "presence" {
			t.Errorf("type: got %q, want %q", msg.Type, "presence")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for presence")
	}
}

func TestBroadcastTyping(t *testing.T) {
	hub := NewSyncHub()

	hub.RegisterConnection("alice@example.com")
	ch, err := hub.Subscribe("thread_456", "alice@example.com")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	hub.BroadcastTyping("thread_456", "bob@example.com", true)

	select {
	case msg := <-ch:
		if msg.Type != "typing" {
			t.Errorf("type: got %q, want %q", msg.Type, "typing")
		}
		if typing, ok := msg.Data["typing"].(bool); !ok || !typing {
			t.Error("typing flag not set")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for typing notification")
	}
}

func TestMessageQueuing(t *testing.T) {
	hub := NewSyncHub()

	// Send message before user connects
	msg := &RealtimeSyncMessage{
		Type:      "message",
		ThreadID:  "thread_789",
		From:      "alice@example.com",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"to": "bob@example.com",
		},
	}

	hub.Broadcast(msg)

	// User connects and retrieves queued messages
	hub.RegisterConnection("bob@example.com")
	queued := hub.GetQueuedMessages("bob@example.com")

	if len(queued) != 1 {
		t.Errorf("queued messages: got %d, want 1", len(queued))
	}
}

func TestHandleIncomingMessage(t *testing.T) {
	hub := NewSyncHub()

	hub.RegisterConnection("alice@example.com")

	msgData := &RealtimeSyncMessage{
		Type:      "message",
		ThreadID:  "thread_abc",
		Timestamp: time.Now().UnixMilli(),
	}

	data, _ := json.Marshal(msgData)
	parsed, err := hub.HandleIncomingMessage("alice@example.com", data)

	if err != nil {
		t.Errorf("handle incoming: %v", err)
	}

	if parsed.Type != "message" {
		t.Errorf("type: got %q, want %q", parsed.Type, "message")
	}
}

func TestConnectionStats(t *testing.T) {
	hub := NewSyncHub()

	hub.RegisterConnection("alice@example.com")
	hub.RegisterConnection("bob@example.com")
	hub.Subscribe("thread_123", "alice@example.com")
	hub.Subscribe("thread_456", "alice@example.com")

	stats := hub.GetConnectionStats()

	if active, ok := stats["active_connections"].(int); !ok || active != 2 {
		t.Errorf("active_connections: got %v", stats["active_connections"])
	}

	if threads, ok := stats["subscribed_threads"].(int); !ok || threads != 2 {
		t.Errorf("subscribed_threads: got %v", stats["subscribed_threads"])
	}
}
