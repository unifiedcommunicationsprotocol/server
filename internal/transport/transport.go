// Package transport handles WebSocket connections, protocol negotiation, and frame encoding.
package transport

import (
	"encoding/json"
	"fmt"
	"time"
)

// Connection represents an active UCP client connection.
type Connection struct {
	ID        string
	Address   string
	SessionToken string
	Capabilities []string
	LastHeartbeat time.Time
	Closed    bool
}

// Hub manages all active connections.
type Hub struct {
	connections map[string]*Connection
	register    chan *Connection
	unregister  chan *Connection
	broadcast   chan interface{}
}

// New creates a new connection hub.
func New() *Hub {
	return &Hub{
		connections: make(map[string]*Connection),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		broadcast:   make(chan interface{}),
	}
}

// RegisterConnection registers a new client connection.
func (h *Hub) RegisterConnection(conn *Connection) error {
	if conn.ID == "" {
		return fmt.Errorf("connection ID required")
	}
	if conn.Address == "" {
		return fmt.Errorf("address required")
	}

	h.connections[conn.ID] = conn
	return nil
}

// UnregisterConnection removes a client connection.
func (h *Hub) UnregisterConnection(connID string) {
	delete(h.connections, connID)
}

// GetConnection retrieves a connection by ID.
func (h *Hub) GetConnection(connID string) (*Connection, error) {
	conn, ok := h.connections[connID]
	if !ok {
		return nil, fmt.Errorf("connection not found")
	}
	return conn, nil
}

// GetConnectionsByAddress retrieves all connections for an address.
func (h *Hub) GetConnectionsByAddress(address string) []*Connection {
	var conns []*Connection
	for _, conn := range h.connections {
		if conn.Address == address {
			conns = append(conns, conn)
		}
	}
	return conns
}

// BroadcastMessage sends a message to all connections of an address.
func (h *Hub) BroadcastMessage(address string, message interface{}) error {
	conns := h.GetConnectionsByAddress(address)
	if len(conns) == 0 {
		return fmt.Errorf("no active connections for address")
	}

	// In real implementation: send via WebSocket to each connection
	// For now: just mark as broadcast
	return nil
}

// ConnectionManager manages UCP connections and handshake.
type ConnectionManager struct {
	hub *Hub
}

// NewConnectionManager creates a new manager.
func NewConnectionManager(hub *Hub) *ConnectionManager {
	return &ConnectionManager{hub: hub}
}

// ValidateUCPHello validates a client hello message and negotiates version.
func (cm *ConnectionManager) ValidateUCPHello(hello map[string]interface{}) (string, error) {
	// Extract and validate client version
	version, ok := hello["version"].(string)
	if !ok || version != "ucp/1.0" {
		return "", fmt.Errorf("unsupported version")
	}

	// In real implementation: version negotiation, capability exchange
	return "ucp/1.0", nil
}

// CreateUCPHelloAck creates a server hello response.
func (cm *ConnectionManager) CreateUCPHelloAck(authToken, serverID string) map[string]interface{} {
	return map[string]interface{}{
		"version":      "ucp/1.0",
		"server_id":    serverID,
		"server_sig":   "base64_signature",
		"capabilities": []string{"ucp/1.0"},
	}
}

// Keepalive manages connection keepalive pings.
type Keepalive struct {
	interval time.Duration
	timeout  time.Duration
}

// NewKeepalive creates a new keepalive handler (30s interval, 10s timeout).
func NewKeepalive() *Keepalive {
	return &Keepalive{
		interval: 30 * time.Second,
		timeout:  10 * time.Second,
	}
}

// ShouldSendPing checks if it's time to send a ping.
func (k *Keepalive) ShouldSendPing(lastHeartbeat time.Time) bool {
	elapsed := time.Since(lastHeartbeat)
	return elapsed > k.interval
}

// IsConnectionAlive checks if a connection should be considered dead.
func (k *Keepalive) IsConnectionAlive(lastHeartbeat time.Time) bool {
	elapsed := time.Since(lastHeartbeat)
	deadlineAfterPing := k.interval + k.timeout
	return elapsed < deadlineAfterPing
}

// Frame represents a UCP protocol frame.
type Frame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Seq     uint64          `json:"seq,omitempty"`
}

// EncodeFrame encodes a frame to JSON.
func EncodeFrame(frame *Frame) ([]byte, error) {
	return json.Marshal(frame)
}

// DecodeFrame decodes a frame from JSON.
func DecodeFrame(data []byte) (*Frame, error) {
	var frame Frame
	if err := json.Unmarshal(data, &frame); err != nil {
		return nil, fmt.Errorf("decode frame: %w", err)
	}
	return &frame, nil
}

// BackoffReconnect calculates exponential backoff for reconnection.
func BackoffReconnect(attempt int) time.Duration {
	// Start at 1s, double each attempt, max 60s
	backoff := time.Duration(1<<uint(attempt)) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second
	}
	return backoff
}
