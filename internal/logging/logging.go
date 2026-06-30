// Package logging provides structured logging and metrics for UCP server.
package logging

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Level is the logging level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// Logger provides structured logging.
type Logger struct {
	mu    sync.Mutex
	level Level
	out   *log.Logger
}

// New creates a new logger.
func New(level Level) *Logger {
	return &Logger{
		level: level,
		out:   log.New(os.Stderr, "", log.LstdFlags),
	}
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, kv ...interface{}) {
	if l.level <= LevelDebug {
		l.log("DEBUG", msg, kv...)
	}
}

// Info logs at info level.
func (l *Logger) Info(msg string, kv ...interface{}) {
	if l.level <= LevelInfo {
		l.log("INFO", msg, kv...)
	}
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, kv ...interface{}) {
	if l.level <= LevelWarn {
		l.log("WARN", msg, kv...)
	}
}

// Error logs at error level.
func (l *Logger) Error(msg string, kv ...interface{}) {
	if l.level <= LevelError {
		l.log("ERROR", msg, kv...)
	}
}

// Fatal logs at fatal level and exits.
func (l *Logger) Fatal(msg string, kv ...interface{}) {
	l.log("FATAL", msg, kv...)
	os.Exit(1)
}

// log logs a message with key-value pairs.
func (l *Logger) log(level, msg string, kv ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fields := formatKV(kv...)
	timestamp := time.Now().Format(time.RFC3339)
	l.out.Printf("[%s] %s %s %s\n", timestamp, level, msg, fields)
}

// formatKV formats key-value pairs as a string.
func formatKV(kv ...interface{}) string {
	if len(kv) == 0 {
		return ""
	}

	var result string
	for i := 0; i < len(kv); i += 2 {
		if i+1 < len(kv) {
			key := fmt.Sprintf("%v", kv[i])
			val := fmt.Sprintf("%v", kv[i+1])
			if i > 0 {
				result += " "
			}
			result += fmt.Sprintf("%s=%s", key, val)
		}
	}
	return result
}

// Metrics tracks server statistics (Prometheus-compatible).
type Metrics struct {
	// Counters (monotonic)
	HTTPRequestsTotal       int64 // All HTTP requests
	HTTPErrorsTotal         int64 // HTTP errors (4xx, 5xx)
	AuthChallengesTotal     int64 // Challenge requests
	AuthSessionsTotal       int64 // Successful sessions
	AuthFailuresTotal       int64 // Failed auth attempts
	MessagesReceivedTotal   int64 // Inbound messages
	MessagesSentTotal       int64 // Outbound messages
	AttachmentsUploadedTotal int64 // File uploads
	FederationDeliveriesTotal int64 // Federation attempts
	FederationFailuresTotal int64 // Federation failures
	SearchQueriesTotal      int64 // Full-text searches

	// State metrics
	ActiveConnections       int64 // Current WebSocket connections
	PendingRetries          int64 // Messages in retry queue

	// Error tracking
	LastErrorTime           time.Time
	LastErrorMsg            string

	mu sync.RWMutex
}

// RecordHTTPRequest increments HTTP request counter.
func (m *Metrics) RecordHTTPRequest() {
	atomic.AddInt64(&m.HTTPRequestsTotal, 1)
}

// RecordHTTPError increments HTTP error counter.
func (m *Metrics) RecordHTTPError() {
	atomic.AddInt64(&m.HTTPErrorsTotal, 1)
}

// RecordAuthChallenge increments auth challenge counter.
func (m *Metrics) RecordAuthChallenge() {
	atomic.AddInt64(&m.AuthChallengesTotal, 1)
}

// RecordAuthSession increments successful session counter.
func (m *Metrics) RecordAuthSession() {
	atomic.AddInt64(&m.AuthSessionsTotal, 1)
}

// RecordAuthFailure increments failed auth counter.
func (m *Metrics) RecordAuthFailure() {
	atomic.AddInt64(&m.AuthFailuresTotal, 1)
}

// RecordMessage increments message counters.
func (m *Metrics) RecordMessage(sent bool) {
	if sent {
		atomic.AddInt64(&m.MessagesSentTotal, 1)
	} else {
		atomic.AddInt64(&m.MessagesReceivedTotal, 1)
	}
}

// RecordAttachmentUpload increments attachment counter.
func (m *Metrics) RecordAttachmentUpload() {
	atomic.AddInt64(&m.AttachmentsUploadedTotal, 1)
}

// RecordFederationAttempt increments federation counter.
func (m *Metrics) RecordFederationAttempt(success bool) {
	if success {
		atomic.AddInt64(&m.FederationDeliveriesTotal, 1)
	} else {
		atomic.AddInt64(&m.FederationFailuresTotal, 1)
	}
}

// RecordSearchQuery increments search query counter.
func (m *Metrics) RecordSearchQuery() {
	atomic.AddInt64(&m.SearchQueriesTotal, 1)
}

// UpdateActiveConnections sets current connection count.
func (m *Metrics) UpdateActiveConnections(count int64) {
	atomic.StoreInt64(&m.ActiveConnections, count)
}

// UpdatePendingRetries sets current retry queue size.
func (m *Metrics) UpdatePendingRetries(count int64) {
	atomic.StoreInt64(&m.PendingRetries, count)
}

// RecordError records an error event with timestamp.
func (m *Metrics) RecordError(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastErrorTime = time.Now()
	m.LastErrorMsg = msg
	atomic.AddInt64(&m.HTTPErrorsTotal, 1)
}


// Snapshot returns current metrics (Prometheus-compatible).
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		// HTTP counters
		"http_requests_total":           atomic.LoadInt64(&m.HTTPRequestsTotal),
		"http_errors_total":             atomic.LoadInt64(&m.HTTPErrorsTotal),

		// Auth counters
		"auth_challenges_total":         atomic.LoadInt64(&m.AuthChallengesTotal),
		"auth_sessions_total":           atomic.LoadInt64(&m.AuthSessionsTotal),
		"auth_failures_total":           atomic.LoadInt64(&m.AuthFailuresTotal),

		// Message counters
		"messages_received_total":       atomic.LoadInt64(&m.MessagesReceivedTotal),
		"messages_sent_total":           atomic.LoadInt64(&m.MessagesSentTotal),

		// Attachment counters
		"attachments_uploaded_total":    atomic.LoadInt64(&m.AttachmentsUploadedTotal),

		// Federation counters
		"federation_deliveries_total":   atomic.LoadInt64(&m.FederationDeliveriesTotal),
		"federation_failures_total":     atomic.LoadInt64(&m.FederationFailuresTotal),

		// Search counters
		"search_queries_total":          atomic.LoadInt64(&m.SearchQueriesTotal),

		// State metrics
		"active_connections":            atomic.LoadInt64(&m.ActiveConnections),
		"pending_retries":               atomic.LoadInt64(&m.PendingRetries),

		// Error tracking
		"last_error_time":               m.LastErrorTime.Unix(),
		"last_error_msg":                m.LastErrorMsg,
	}
}
