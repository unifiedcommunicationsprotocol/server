// Package logging provides structured logging and metrics for UCP server.
package logging

import (
	"fmt"
	"log"
	"os"
	"sync"
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

// Metrics tracks server statistics.
type Metrics struct {
	mu                  sync.RWMutex
	MessagesReceived    int64
	MessagesSent        int64
	AuthChallenges      int64
	AuthSessions        int64
	AttachmentsUploaded int64
	Errors              int64
	LastError           string
}

// RecordMessage records a message.
func (m *Metrics) RecordMessage(sent bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sent {
		m.MessagesSent++
	} else {
		m.MessagesReceived++
	}
}

// RecordAuth records auth event.
func (m *Metrics) RecordAuth(isSession bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if isSession {
		m.AuthSessions++
	} else {
		m.AuthChallenges++
	}
}

// RecordAttachment records attachment upload.
func (m *Metrics) RecordAttachment() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AttachmentsUploaded++
}

// RecordError records an error.
func (m *Metrics) RecordError(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors++
	m.LastError = err
}

// Snapshot returns current metrics.
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"messages_received":    m.MessagesReceived,
		"messages_sent":        m.MessagesSent,
		"auth_challenges":      m.AuthChallenges,
		"auth_sessions":        m.AuthSessions,
		"attachments_uploaded": m.AttachmentsUploaded,
		"errors":               m.Errors,
		"last_error":           m.LastError,
	}
}
