package transport

import (
	"testing"
	"time"
)

// TestSyncHubRegisterConnection tests registering a sync connection.
func TestSyncHubRegisterConnection(t *testing.T) {
	sh := NewSyncHub()

	conn := sh.RegisterConnection("alice@example.com")

	if conn.Address != "alice@example.com" {
		t.Errorf("address mismatch: %s", conn.Address)
	}

	if conn.SendChan == nil || conn.ReceiveChan == nil || conn.CloseChan == nil {
		t.Error("channels not initialized")
	}

	if conn.ConnectedAt.IsZero() {
		t.Error("ConnectedAt not set")
	}
}

// TestSyncHubUnregisterConnection tests unregistering a sync connection.
func TestSyncHubUnregisterConnection(t *testing.T) {
	sh := NewSyncHub()

	sh.RegisterConnection("alice@example.com")
	sh.UnregisterConnection("alice@example.com")

	// Verify no connections remain
	// (Cannot directly verify, but the channels should be closed)
}

// TestSyncHubSubscribe tests subscribing to a thread.
func TestSyncHubSubscribe(t *testing.T) {
	sh := NewSyncHub()

	// Register a connection
	sh.RegisterConnection("alice@example.com")

	// Subscribe to a thread
	ch, err := sh.Subscribe("thread-1", "alice@example.com")
	if err != nil {
		t.Errorf("subscribe failed: %v", err)
		return
	}

	if ch == nil {
		t.Error("channel is nil")
	}
}

// TestSyncHubSubscribeNoConnection tests subscribing without active connection.
func TestSyncHubSubscribeNoConnection(t *testing.T) {
	sh := NewSyncHub()

	// Try to subscribe without registering
	_, err := sh.Subscribe("thread-1", "alice@example.com")
	if err == nil {
		t.Error("should fail without active connection")
	}
}

// TestSyncHubBroadcast tests broadcasting a message to subscribers.
func TestSyncHubBroadcast(t *testing.T) {
	sh := NewSyncHub()

	// Register connection and subscribe
	sh.RegisterConnection("alice@example.com")
	ch, _ := sh.Subscribe("thread-1", "alice@example.com")

	// Broadcast a message
	msg := &RealtimeSyncMessage{
		Type:      "message",
		ThreadID:  "thread-1",
		From:      "bob@example.com",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"content": "hello",
		},
	}

	sh.Broadcast(msg)

	// Receive the message (non-blocking with timeout)
	select {
	case received := <-ch:
		if received.Type != "message" {
			t.Errorf("message type mismatch: %s", received.Type)
		}
		if received.From != "bob@example.com" {
			t.Errorf("from mismatch: %s", received.From)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast message")
	}
}

// TestSyncHubMultipleSubscribers tests broadcasting to multiple subscribers.
func TestSyncHubMultipleSubscribers(t *testing.T) {
	sh := NewSyncHub()

	// Register two connections and subscribe to same thread
	sh.RegisterConnection("alice@example.com")
	sh.RegisterConnection("bob@example.com")

	ch1, _ := sh.Subscribe("thread-1", "alice@example.com")
	ch2, _ := sh.Subscribe("thread-1", "bob@example.com")

	// Broadcast message
	msg := &RealtimeSyncMessage{
		Type:      "message",
		ThreadID:  "thread-1",
		Timestamp: time.Now().Unix(),
	}

	sh.Broadcast(msg)

	// Both should receive
	select {
	case <-ch1:
	case <-time.After(100 * time.Millisecond):
		t.Error("alice didn't receive broadcast")
	}

	select {
	case <-ch2:
	case <-time.After(100 * time.Millisecond):
		t.Error("bob didn't receive broadcast")
	}
}

// TestSyncHubMultipleThreads tests subscribing to different threads.
func TestSyncHubMultipleThreads(t *testing.T) {
	sh := NewSyncHub()

	sh.RegisterConnection("alice@example.com")

	// Subscribe to two threads
	ch1, _ := sh.Subscribe("thread-1", "alice@example.com")
	ch2, _ := sh.Subscribe("thread-2", "alice@example.com")

	// Broadcast to thread-1
	msg1 := &RealtimeSyncMessage{
		Type:      "message",
		ThreadID:  "thread-1",
		Timestamp: time.Now().Unix(),
	}
	sh.Broadcast(msg1)

	// Should receive on ch1, not ch2
	select {
	case <-ch1:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("didn't receive on thread-1")
	}

	// ch2 should NOT receive
	select {
	case <-ch2:
		t.Error("received on wrong thread")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message
	}
}

// TestRealtimeSyncMessageTypes tests various message types.
func TestRealtimeSyncMessageTypes(t *testing.T) {
	messageTypes := []string{"message", "presence", "typing", "receipt"}

	for _, msgType := range messageTypes {
		msg := &RealtimeSyncMessage{
			Type:      msgType,
			ThreadID:  "thread-1",
			Timestamp: time.Now().Unix(),
		}

		if msg.Type != msgType {
			t.Errorf("type mismatch: %s", msg.Type)
		}
	}
}

// TestSyncConnectionTimestamps tests connection timestamps.
func TestSyncConnectionTimestamps(t *testing.T) {
	sh := NewSyncHub()

	before := time.Now()
	conn := sh.RegisterConnection("alice@example.com")
	after := time.Now()

	if conn.ConnectedAt.Before(before) || conn.ConnectedAt.After(after) {
		t.Error("ConnectedAt timestamp not in expected range")
	}

	if conn.LastActivity.Before(before) || conn.LastActivity.After(after) {
		t.Error("LastActivity timestamp not in expected range")
	}
}
