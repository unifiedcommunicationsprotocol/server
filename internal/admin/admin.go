// Package admin provides admin dashboard functionality: event broadcasting, monitoring, and controls.
package admin

import (
	"sync"
	"time"
)

// EventType identifies the kind of admin event.
type EventType string

const (
	EventSessionCreated    EventType = "session.created"
	EventSessionRevoked    EventType = "session.revoked"
	EventFederationConnect EventType = "federation.connected"
	EventQueueUpdated      EventType = "queue.updated"
)

// Event represents an admin notification sent to dashboard subscribers.
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Subscriber receives admin events over a channel.
type Subscriber struct {
	ID    string
	Events chan *Event
}

// AdminHub manages admin event subscriptions and broadcasting.
type AdminHub struct {
	subscribers map[string]*Subscriber
	mu          sync.RWMutex
	subscribe   chan *Subscriber
	unsubscribe chan string
	broadcast   chan *Event
}

// New creates a new AdminHub.
func New() *AdminHub {
	hub := &AdminHub{
		subscribers: make(map[string]*Subscriber),
		subscribe:   make(chan *Subscriber, 10),
		unsubscribe: make(chan string, 10),
		broadcast:   make(chan *Event, 100),
	}

	// Start the event loop
	go hub.loop()

	return hub
}

// Subscribe registers a new admin subscriber.
func (h *AdminHub) Subscribe(id string) *Subscriber {
	sub := &Subscriber{
		ID:     id,
		Events: make(chan *Event, 50),
	}
	h.subscribe <- sub
	return sub
}

// Unsubscribe unregisters a subscriber.
func (h *AdminHub) Unsubscribe(id string) {
	h.unsubscribe <- id
}

// Broadcast sends an event to all subscribers.
func (h *AdminHub) Broadcast(event *Event) {
	h.broadcast <- event
}

// BroadcastSessionCreated broadcasts a session created event.
func (h *AdminHub) BroadcastSessionCreated(token, address string, expiresAt int64) {
	h.Broadcast(&Event{
		Type:      EventSessionCreated,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"token":      token,
			"identity":   address,
			"expires_at": expiresAt,
		},
	})
}

// BroadcastSessionRevoked broadcasts a session revoked event.
func (h *AdminHub) BroadcastSessionRevoked(token string) {
	h.Broadcast(&Event{
		Type:      EventSessionRevoked,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"token": token,
		},
	})
}

// BroadcastFederationConnected broadcasts a federation connection event.
func (h *AdminHub) BroadcastFederationConnected(remoteDomain string) {
	h.Broadcast(&Event{
		Type:      EventFederationConnect,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"remote_domain": remoteDomain,
		},
	})
}

// BroadcastQueueUpdated broadcasts a queue update event.
func (h *AdminHub) BroadcastQueueUpdated(queueDepth int, failedCount int) {
	h.Broadcast(&Event{
		Type:      EventQueueUpdated,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"queue_depth":  queueDepth,
			"failed_count": failedCount,
		},
	})
}

// loop processes subscriptions and broadcasts in a single goroutine.
func (h *AdminHub) loop() {
	for {
		select {
		case sub := <-h.subscribe:
			h.mu.Lock()
			h.subscribers[sub.ID] = sub
			h.mu.Unlock()

		case id := <-h.unsubscribe:
			h.mu.Lock()
			if sub, ok := h.subscribers[id]; ok {
				close(sub.Events)
				delete(h.subscribers, id)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			for _, sub := range h.subscribers {
				// Non-blocking send; drop if subscriber is slow
				select {
				case sub.Events <- event:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}
