// Package store manages Postgres persistence: messages, identities, sessions, and indexes.
package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/unifiedcommunicationsprotocol/server/internal/models"
)

// Store encapsulates all database operations.
type Store struct {
	db *sql.DB
}

// New creates a new Store with an open database connection.
func New(dsn string) (*Store, error) {
	fmt.Printf("DEBUG: Connecting with DSN: %s\n", dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*1000)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// StoreMessage stores a message envelope in the database.
func (s *Store) StoreMessage(ctx context.Context, envelope *models.UCPEnvelope, encryptedMLS []byte) error {
	const query = `
	INSERT INTO messages (message_id, thread_id, from_addr, to_addrs, signing_key, server_ts, mls_encrypted)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (message_id) DO NOTHING
	`

	// Generate unique message ID if not provided
	messageID := models.GenerateULID()

	_, err := s.db.ExecContext(ctx, query,
		string(messageID),
		string(envelope.ThreadID),
		envelope.From,
		pq.Array(envelope.To),
		envelope.SigningKey,
		envelope.ServerTs,
		encryptedMLS,
	)

	if err != nil {
		return fmt.Errorf("store message: %w", err)
	}

	return nil
}

// GetMessage retrieves a message envelope by ID.
func (s *Store) GetMessage(ctx context.Context, messageID models.ULID) (*models.UCPEnvelope, []byte, error) {
	const query = `
	SELECT thread_id, from_addr, to_addrs, signing_key, server_ts, mls_encrypted
	FROM messages
	WHERE id = $1
	`

	var envelope models.UCPEnvelope
	var to []string
	var mls []byte

	err := s.db.QueryRowContext(ctx, query, messageID).Scan(
		&envelope.ThreadID,
		&envelope.From,
		&to,
		&envelope.SigningKey,
		&envelope.ServerTs,
		&mls,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("message not found")
		}
		return nil, nil, fmt.Errorf("query message: %w", err)
	}

	envelope.To = to
	envelope.V = "ucp/1.0"

	return &envelope, mls, nil
}

// GetThreadMessages retrieves all messages in a thread, ordered by server_ts.
func (s *Store) GetThreadMessages(ctx context.Context, threadID models.ULID) ([]*models.UCPEnvelope, error) {
	const query = `
	SELECT id, thread_id, from_addr, to_addrs, signing_key, server_ts
	FROM messages
	WHERE thread_id = $1
	ORDER BY server_ts ASC
	`

	rows, err := s.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("query thread messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.UCPEnvelope
	for rows.Next() {
		var envelope models.UCPEnvelope
		var id models.ULID
		var to []string

		if err := rows.Scan(
			&id,
			&envelope.ThreadID,
			&envelope.From,
			&to,
			&envelope.SigningKey,
			&envelope.ServerTs,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		envelope.V = "ucp/1.0"
		envelope.To = to
		messages = append(messages, &envelope)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}

// StoreIdentity stores or updates an identity record.
func (s *Store) StoreIdentity(ctx context.Context, identity *models.Identity) error {
	const query = `
	INSERT INTO identities (address, identity_key, signing_keys_json, revocation_key, capabilities)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (address) DO UPDATE SET
		identity_key = $2,
		signing_keys_json = $3,
		revocation_key = $4,
		capabilities = $5,
		updated_at = NOW()
	`

	// For now, serialize signing_keys as JSON (real impl would normalize)
	signingKeysJSON, _ := marshalSigningKeys(identity.SigningKeys)

	_, err := s.db.ExecContext(ctx, query,
		identity.Address,
		identity.IdentityKey,
		signingKeysJSON,
		identity.RevocationKey,
		pq.Array(identity.Capabilities),
	)

	if err != nil {
		return fmt.Errorf("store identity: %w", err)
	}

	return nil
}

// GetIdentity retrieves an identity by address.
func (s *Store) GetIdentity(ctx context.Context, address string) (*models.Identity, error) {
	const query = `
	SELECT address, identity_key, signing_keys_json, revocation_key, capabilities
	FROM identities
	WHERE address = $1
	`

	var identity models.Identity
	var signingKeysJSON string

	err := s.db.QueryRowContext(ctx, query, address).Scan(
		&identity.Address,
		&identity.IdentityKey,
		&signingKeysJSON,
		&identity.RevocationKey,
		pq.Array(&identity.Capabilities),
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("identity not found")
		}
		return nil, fmt.Errorf("query identity: %w", err)
	}

	identity.SigningKeys, _ = unmarshalSigningKeys(signingKeysJSON)

	return &identity, nil
}

// CreateSession creates a new authenticated session.
func (s *Store) CreateSession(ctx context.Context, address string, token string, expiresAt int64) error {
	const query = `
	INSERT INTO sessions (address, token, expires_at)
	VALUES ($1, $2, to_timestamp($3))
	ON CONFLICT (token) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, address, token, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by token.
func (s *Store) GetSession(ctx context.Context, token string) (address string, err error) {
	const query = `
	SELECT address FROM sessions
	WHERE token = $1
	AND expires_at > NOW()
	`

	err = s.db.QueryRowContext(ctx, query, token).Scan(&address)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("session not found or expired")
		}
		return "", fmt.Errorf("query session: %w", err)
	}

	return address, nil
}

// RevokeSession revokes a session token.
func (s *Store) RevokeSession(ctx context.Context, token string) error {
	const query = `
	UPDATE sessions SET revoked_at = NOW()
	WHERE token = $1 AND revoked_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

// StoreAttachment stores an attachment reference.
func (s *Store) StoreAttachment(ctx context.Context, attachment *models.Attachment) error {
	const query = `
	INSERT INTO attachments (id, name, mime_type, size, sha256)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		attachment.ID,
		attachment.Name,
		attachment.MimeType,
		attachment.Size,
		attachment.SHA256,
	)

	if err != nil {
		return fmt.Errorf("store attachment: %w", err)
	}

	return nil
}

// GetAttachment retrieves an attachment by ID.
func (s *Store) GetAttachment(ctx context.Context, attachmentID models.ULID) (*models.Attachment, error) {
	const query = `
	SELECT id, name, mime_type, size, sha256
	FROM attachments
	WHERE id = $1
	`

	var attachment models.Attachment
	err := s.db.QueryRowContext(ctx, query, attachmentID).Scan(
		&attachment.ID,
		&attachment.Name,
		&attachment.MimeType,
		&attachment.Size,
		&attachment.SHA256,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("attachment not found")
		}
		return nil, fmt.Errorf("query attachment: %w", err)
	}

	return &attachment, nil
}

// Helper functions for JSON serialization (simplified for now).
func marshalSigningKeys(keys []models.SigningKey) (string, error) {
	// TODO: implement proper JSON marshaling
	return "[]", nil
}

func unmarshalSigningKeys(jsonStr string) ([]models.SigningKey, error) {
	// TODO: implement proper JSON unmarshaling
	return []models.SigningKey{}, nil
}
