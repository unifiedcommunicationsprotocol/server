package transport

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// RealtimeSyncMessage is a message sent over the sync channel.
type RealtimeSyncMessage struct {
	Type      string                 `json:"type"` // "message", "presence", "typing", "receipt"
	ThreadID  string                 `json:"thread_id,omitempty"`
	From      string                 `json:"from,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// SyncHub manages real-time connections and broadcasts.
type SyncHub struct {
	mu           sync.RWMutex
	connections  map[string]*SyncConnection // address -> connection
	subscribers  map[string][]chan *RealtimeSyncMessage // threadID -> channels
	messageQueue [](*RealtimeSyncMessage)
}

// SyncConnection represents an active real-time sync connection.
type SyncConnection struct {
	Address      string
	ConnectedAt  time.Time
	LastActivity time.Time
	SendChan     chan *RealtimeSyncMessage
	ReceiveChan  chan *RealtimeSyncMessage
	CloseChan    chan struct{}
}

// NewSyncHub creates a new sync hub.
func NewSyncHub() *SyncHub {
	return &SyncHub{
		connections: make(map[string]*SyncConnection),
		subscribers: make(map[string][]chan *RealtimeSyncMessage),
		messageQueue: make([](*RealtimeSyncMessage), 0),
	}
}

// RegisterConnection registers a new real-time connection.
func (sh *SyncHub) RegisterConnection(address string) *SyncConnection {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	conn := &SyncConnection{
		Address:      address,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		SendChan:     make(chan *RealtimeSyncMessage, 10),
		ReceiveChan:  make(chan *RealtimeSyncMessage, 10),
		CloseChan:    make(chan struct{}),
	}

	sh.connections[address] = conn
	return conn
}

// UnregisterConnection removes a connection.
func (sh *SyncHub) UnregisterConnection(address string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	conn, exists := sh.connections[address]
	if exists {
		close(conn.SendChan)
		close(conn.ReceiveChan)
		close(conn.CloseChan)
		delete(sh.connections, address)
	}
}

// Subscribe subscribes to messages for a thread.
func (sh *SyncHub) Subscribe(threadID string, address string) (<-chan *RealtimeSyncMessage, error) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	_, exists := sh.connections[address]
	if !exists {
		return nil, fmt.Errorf("connection not found for %s", address)
	}

	ch := make(chan *RealtimeSyncMessage, 10)
	sh.subscribers[threadID] = append(sh.subscribers[threadID], ch)

	return ch, nil
}

// Broadcast sends a message to all subscribers of a thread.
func (sh *SyncHub) Broadcast(msg *RealtimeSyncMessage) {
	sh.mu.RLock()
	subscribers, exists := sh.subscribers[msg.ThreadID]
	sh.mu.RUnlock()

	if exists {
		for _, ch := range subscribers {
			select {
			case ch <- msg:
			case <-time.After(1 * time.Second):
				// Subscriber timeout, skip
			}
		}
	}

	// Queue for offline delivery
	sh.mu.Lock()
	sh.messageQueue = append(sh.messageQueue, msg)
	sh.mu.Unlock()
}

// BroadcastToUser sends a message to a specific user.
func (sh *SyncHub) BroadcastToUser(address string, msg *RealtimeSyncMessage) error {
	sh.mu.RLock()
	conn, exists := sh.connections[address]
	sh.mu.RUnlock()

	if !exists {
		// Queue for later delivery
		sh.mu.Lock()
		sh.messageQueue = append(sh.messageQueue, msg)
		sh.mu.Unlock()
		return fmt.Errorf("connection not found, queued for delivery")
	}

	select {
	case conn.SendChan <- msg:
		return nil
	case <-time.After(1 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

// GetQueuedMessages retrieves messages queued while offline.
func (sh *SyncHub) GetQueuedMessages(address string) []*RealtimeSyncMessage {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	var messages []*RealtimeSyncMessage

	// Filter for messages addressed to this user
	for _, msg := range sh.messageQueue {
		if data, ok := msg.Data["to"].(string); ok && data == address {
			messages = append(messages, msg)
		}
	}

	// Clear processed messages
	sh.messageQueue = make([](*RealtimeSyncMessage), 0)

	return messages
}

// BroadcastPresence notifies of user presence.
func (sh *SyncHub) BroadcastPresence(address string, online bool) {
	msg := &RealtimeSyncMessage{
		Type:      "presence",
		From:      address,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"online": online,
		},
	}

	sh.mu.RLock()
	subscribers := sh.subscribers["presence"]
	sh.mu.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

// BroadcastTyping notifies that a user is typing.
func (sh *SyncHub) BroadcastTyping(threadID, address string, typing bool) {
	msg := &RealtimeSyncMessage{
		Type:      "typing",
		ThreadID:  threadID,
		From:      address,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"typing": typing,
		},
	}

	sh.Broadcast(msg)
}

// BroadcastReceipt notifies of message receipt.
func (sh *SyncHub) BroadcastReceipt(threadID, address string, messageID string) {
	msg := &RealtimeSyncMessage{
		Type:      "receipt",
		ThreadID:  threadID,
		From:      address,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"message_id": messageID,
		},
	}

	sh.Broadcast(msg)
}

// SyncMessageHandler processes incoming sync messages.
type SyncMessageHandler func(*RealtimeSyncMessage) error

// HandleIncomingMessage processes a message from a client.
func (sh *SyncHub) HandleIncomingMessage(address string, data []byte) (*RealtimeSyncMessage, error) {
	var msg RealtimeSyncMessage

	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("invalid message format: %w", err)
	}

	// Update connection activity
	sh.mu.Lock()
	if conn, exists := sh.connections[address]; exists {
		conn.LastActivity = time.Now()
	}
	sh.mu.Unlock()

	return &msg, nil
}

// GetConnectionStats returns connection statistics.
func (sh *SyncHub) GetConnectionStats() map[string]interface{} {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	activeCount := len(sh.connections)
	totalThreads := len(sh.subscribers)

	return map[string]interface{}{
		"active_connections": activeCount,
		"subscribed_threads": totalThreads,
		"queued_messages":    len(sh.messageQueue),
	}
}
