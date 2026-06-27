// Package models defines UCP protocol types: Message, Envelope, Identity, KeyPackage, etc.
package models

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"time"
)

// ULID is a 128-bit, lexicographically sortable unique identifier.
type ULID string

// GenerateULID creates a new ULID (timestamp + random bytes).
func GenerateULID() ULID {
	// ULID format: timestamp (48 bits) + randomness (80 bits)
	// For simplicity, use timestamp + random string
	timestamp := time.Now().UnixMilli()
	randomBytes := make([]byte, 10)
	rand.Read(randomBytes)

	// Encode as base32 for readability (similar to actual ULID)
	randomStr := base32.StdEncoding.EncodeToString(randomBytes)[:16]
	return ULID(fmt.Sprintf("%013d%s", timestamp, randomStr))
}

// Message represents a UCP message (email, chat, etc.).
type Message struct {
	ID              ULID               `json:"id"`
	Version         string             `json:"version"` // "ucp/1.0"
	Type            string             `json:"type"`    // "message.email"
	From            string             `json:"from"`
	To              []string           `json:"to"`
	Cc              []string           `json:"cc,omitempty"`
	Bcc             []string           `json:"bcc,omitempty"`
	Subject         string             `json:"subject"`
	Body            *Body              `json:"body"`
	Attachments     []Attachment       `json:"attachments,omitempty"`
	ThreadID        ULID               `json:"thread_id"`
	InReplyTo       *ULID              `json:"in_reply_to,omitempty"`
	References      []ULID             `json:"references,omitempty"`
	Priority        int                `json:"priority,omitempty"` // 1-5, default 3
	ClientTs        int64              `json:"client_ts"`          // Unix timestamp
	Origin          *Origin            `json:"origin,omitempty"`
	Meta            *MessageMeta       `json:"meta,omitempty"`
	Signature       string             `json:"signature,omitempty"`          // Ed25519 sig
	BridgeAttestation *BridgeAttestation `json:"bridge_attestation,omitempty"` // Instead of signature
}

// Body is a message body with structured blocks and optional HTML.
type Body struct {
	Blocks []Block `json:"blocks"`
	HTML   string  `json:"html,omitempty"`
}

// Block is a single content block (paragraph, heading, list, code, blockquote, image, divider).
type Block interface {
	BlockType() string
}

// ParagraphBlock represents a text paragraph.
type ParagraphBlock struct {
	Type    string       `json:"type"` // "paragraph"
	Content []InlineSpan `json:"content"`
}

func (ParagraphBlock) BlockType() string { return "paragraph" }

// HeadingBlock represents a heading.
type HeadingBlock struct {
	Type    string       `json:"type"` // "heading"
	Level   int          `json:"level"` // 1-3
	Content []InlineSpan `json:"content"`
}

func (HeadingBlock) BlockType() string { return "heading" }

// ListBlock represents an ordered or unordered list.
type ListBlock struct {
	Type    string      `json:"type"` // "list"
	Ordered bool        `json:"ordered"`
	Items   []ListItem  `json:"items"`
}

func (ListBlock) BlockType() string { return "list" }

// ListItem is a single list item.
type ListItem struct {
	Content []InlineSpan `json:"content"`
}

// CodeBlock represents a code block.
type CodeBlock struct {
	Type    string `json:"type"` // "code"
	Lang    string `json:"lang,omitempty"`
	Content string `json:"content"`
}

func (CodeBlock) BlockType() string { return "code" }

// BlockquoteBlock represents a quoted section.
type BlockquoteBlock struct {
	Type    string  `json:"type"` // "blockquote"
	Content []Block `json:"content"`
}

func (BlockquoteBlock) BlockType() string { return "blockquote" }

// ImageBlock represents an embedded or external image.
type ImageBlock struct {
	Type          string `json:"type"` // "image"
	AttachmentID  *ULID  `json:"attachment_id,omitempty"`
	ExternalURL   string `json:"external_url,omitempty"`
	Alt           string `json:"alt"`
	Width         *int   `json:"width,omitempty"`
	Height        *int   `json:"height,omitempty"`
}

func (ImageBlock) BlockType() string { return "image" }

// DividerBlock represents a horizontal rule.
type DividerBlock struct {
	Type string `json:"type"` // "divider"
}

func (DividerBlock) BlockType() string { return "divider" }

// InlineSpan is a text span with optional marks (bold, italic, code, link, strikethrough).
type InlineSpan struct {
	Text  string      `json:"text"`
	Marks []InlineMark `json:"marks,omitempty"`
}

// InlineMark represents text formatting.
type InlineMark struct {
	Type string `json:"type"` // "bold", "italic", "code", "link", "strikethrough"
	Href string `json:"href,omitempty"` // For "link" type
}

// Attachment represents a message attachment.
type Attachment struct {
	ID       ULID   `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"` // Hex-encoded
}

// Origin indicates a message is a forward.
type Origin struct {
	MessageID *ULID  `json:"message_id,omitempty"` // Null for bridge forwards
	From      string `json:"from"`
	ClientTs  int64  `json:"client_ts"` // Unix timestamp
}

// MessageMeta holds AI metadata, labels, and legacy headers.
type MessageMeta struct {
	AI      *AIMetadata       `json:"ai,omitempty"`
	Labels  []string          `json:"labels,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// AIMetadata is AI-generated or sender-supplied metadata.
type AIMetadata struct {
	Summary    string    `json:"summary,omitempty"`
	Category   string    `json:"category,omitempty"` // work, personal, newsletter, notification, transactional, social
	Priority   *int      `json:"priority,omitempty"` // 1-5
	Embeddings []float32 `json:"embeddings,omitempty"` // Local semantic search only
}

// BridgeAttestation is a server signature for bridge-converted messages.
type BridgeAttestation struct {
	Source        string `json:"source"` // "smtp"
	SMTPFrom      string `json:"smtp_from"`
	SMTPMessageID string `json:"smtp_message_id"`
	ReceivedAt    int64  `json:"received_at"` // Unix timestamp
	ReceivingServer string `json:"receiving_server"`
	ThreadingGap  bool   `json:"threading_gap,omitempty"`
	DKIM          string `json:"dkim,omitempty"` // "pass", "fail", "none"
	ServerSig     string `json:"server_sig"`     // Ed25519 signature
}

// Edit represents a message edit.
type Edit struct {
	Type      string `json:"type"` // "edit"
	MessageID ULID   `json:"message_id"`
	ThreadID  ULID   `json:"thread_id"`
	Body      *Body  `json:"body"`
	ClientTs  int64  `json:"client_ts"`
	Signature string `json:"signature"`
}

// Delete represents a message deletion.
type Delete struct {
	Type      string `json:"type"` // "delete"
	MessageID ULID   `json:"message_id"`
	ThreadID  ULID   `json:"thread_id"`
	Reason    *string `json:"reason,omitempty"` // "recalled", "policy"
	ClientTs  int64  `json:"client_ts"`
	Signature string `json:"signature"`
}

// Receipt represents a read receipt.
type Receipt struct {
	Type      string `json:"type"` // "receipt"
	MessageID ULID   `json:"message_id"`
	ThreadID  ULID   `json:"thread_id"`
	ClientTs  int64  `json:"client_ts"`
}

// UCPEnvelope is the wire format wrapping an MLS message.
type UCPEnvelope struct {
	V         string `json:"v"` // "ucp/1.0"
	Type      string `json:"type"` // "welcome", "handshake", "application"
	ThreadID  ULID   `json:"thread_id"`
	From      string `json:"from"`
	To        []string `json:"to"`
	SigningKey string `json:"signing_key,omitempty"` // Base64-encoded Ed25519 pubkey
	ServerTs  *int64 `json:"server_ts,omitempty"` // Assigned by receiver
	MLS       string `json:"mls"` // Base64-encoded TLS-serialized MLSMessage
}

// Identity represents a UCP identity record.
type Identity struct {
	Address       string          `json:"address"`
	IdentityKey   string          `json:"identity_key"` // Base64-encoded Ed25519 pubkey
	SigningKeys   []SigningKey    `json:"signing_keys"`
	RevocationKey string          `json:"revocation_key"` // Base64-encoded Ed25519 pubkey
	Revocation    *RevocationRecord `json:"revocation,omitempty"`
	Server        string          `json:"server"`
	Capabilities  []string        `json:"capabilities"` // ["ucp/1.0"]
	Preferences   *Preferences    `json:"preferences,omitempty"`
	ServerProcessing *ServerProcessing `json:"server_processing,omitempty"`
}

// SigningKey is an active, grace, or expired signing key.
type SigningKey struct {
	Key       string `json:"key"` // Base64-encoded Ed25519 pubkey
	Expires   int64  `json:"expires"` // Unix timestamp
	Issued    int64  `json:"issued"` // Unix timestamp
	Sig       string `json:"sig"` // Base64-encoded identity key signature
	Status    string `json:"status"` // "active", "grace", "expired"
}

// RevocationRecord indicates an identity has been revoked.
type RevocationRecord struct {
	Type          string `json:"type"` // "revocation"
	Version       string `json:"version"` // "1"
	Identity      string `json:"identity"`
	IdentityKey   string `json:"identity_key"` // Base64-encoded
	Reason        string `json:"reason"` // "compromised", "lost", "rotation"
	Timestamp     int64  `json:"timestamp"`
	RevocationSig string `json:"revocation_sig"` // Base64-encoded
}

// Preferences declares user preferences for message delivery.
type Preferences struct {
	Rendering      string `json:"rendering,omitempty"` // "html" or "blocks", default "html"
	ReadReceipts   bool   `json:"read_receipts,omitempty"` // Default false
	ExternalImages bool   `json:"external_images,omitempty"` // Default false
	Language       string `json:"language,omitempty"` // ISO 639-1, default "en"
}

// ServerProcessing declares opt-in server-side decryption.
type ServerProcessing struct {
	Enabled   bool     `json:"enabled"`
	Scopes    []string `json:"scopes,omitempty"` // "search", "summary", "routing"
	GrantedAt *int64   `json:"granted_at,omitempty"`
}

// KeyPackage is MLS encryption material for group creation.
type KeyPackage struct {
	Version    string `json:"version"` // "mls10"
	CipherSuite string `json:"cipher_suite"` // "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
	InitKey    string `json:"init_key"` // Base64-encoded HPKE pubkey
	LeafNode   *LeafNode `json:"leaf_node"`
	Extensions []string `json:"extensions,omitempty"`
	Signature  string   `json:"signature"` // Base64-encoded
}

// LeafNode is the MLS leaf node in a KeyPackage.
type LeafNode struct {
	EncryptionKey string     `json:"encryption_key"` // Base64-encoded HPKE pubkey
	SignatureKey  string     `json:"signature_key"` // Base64-encoded Ed25519 pubkey
	Credential    *Credential `json:"credential"`
	Capabilities  *Capabilities `json:"capabilities,omitempty"`
	LeafNodeSource string    `json:"leaf_node_source"` // "key_package"
	Extensions    []string   `json:"extensions,omitempty"`
}

// Credential binds MLS to UCP identity.
type Credential struct {
	CredentialType string `json:"credential_type"` // "signing_key"
	SigningKey     string `json:"signing_key"` // Base64-encoded Ed25519 pubkey
	Identity       string `json:"identity"`
	IdentityKey    string `json:"identity_key"` // Base64-encoded
	IdentitySig    string `json:"identity_sig"` // Base64-encoded identity key signature
}

// Capabilities declares what protocol features are supported.
type Capabilities struct {
	Extensions []string `json:"extensions,omitempty"` // ["ucp/1.0"]
}

// Session represents an authenticated user session.
type Session struct {
	Token     string `json:"session_token"`
	ExpiresAt int64  `json:"expires_at"` // Unix timestamp
}

// SystemMessage represents a server-generated or client-generated system event.
type SystemMessage struct {
	Type      string `json:"type"` // "system.delivery_failure", "system.member_added", "system.member_removed"
	ThreadID  ULID   `json:"thread_id"`
	MessageID *ULID  `json:"message_id,omitempty"` // For delivery_failure
	Recipient string `json:"recipient,omitempty"` // For delivery_failure
	AttemptedUntil *int64 `json:"attempted_until,omitempty"` // Unix timestamp, for delivery_failure
	Reason    string `json:"reason,omitempty"` // For delivery_failure
	ServerSig string `json:"server_sig,omitempty"` // For server-generated
}

// DeliveryFailure is the payload of a system.delivery_failure message.
type DeliveryFailure struct {
	Type           string `json:"type"` // "system.delivery_failure"
	ThreadID       ULID   `json:"thread_id"`
	MessageID      ULID   `json:"message_id"`
	Recipient      string `json:"recipient"`
	AttemptedUntil int64  `json:"attempted_until"`
	Reason         string `json:"reason"` // "unknown_recipient", "identity_suspended", "server_unreachable", "keypackage_unavailable"
	ServerSig      string `json:"server_sig"` // Ed25519 signature
}

// UCPError is a standard error response.
type UCPError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	RetryAfter *int   `json:"retry_after,omitempty"` // Seconds
}

// UCPHello is the initial client handshake message.
type UCPHello struct {
	Version      string   `json:"version"` // e.g., "ucp/1.0"
	AuthToken    string   `json:"auth_token"`
	Capabilities []string `json:"capabilities"` // ["ucp/1.0"]
}

// UCPHelloAck is the server response to UCPHello.
type UCPHelloAck struct {
	Version         string   `json:"version"`
	ServerID        string   `json:"server_id"` // Canonical domain
	ServerSig       string   `json:"server_sig"` // Ed25519 signature
	Capabilities    []string `json:"capabilities"`
	StaleKeyShares  []string `json:"stale_key_shares,omitempty"` // Base64-encoded group IDs
}

// KeySharePayload is used to submit MLS key shares for server processing.
type KeySharePayload struct {
	Shares []KeyShare `json:"shares"`
}

// KeyShare is a single MLS key share for server processing.
type KeyShare struct {
	GroupID ULID   `json:"group_id"` // Base64-encoded
	Epoch   int    `json:"epoch"`
	Key     string `json:"key"` // Base64-encoded server processing key
}

// Now returns current Unix timestamp.
func Now() int64 {
	return time.Now().Unix()
}

// RawBlock is used for JSON marshaling/unmarshaling blocks with type discrimination.
type RawBlock struct {
	Type    string          `json:"type"`
	Level   int             `json:"level,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Ordered bool            `json:"ordered,omitempty"`
	Items   json.RawMessage `json:"items,omitempty"`
	Lang    string          `json:"lang,omitempty"`
	AttachmentID *ULID      `json:"attachment_id,omitempty"`
	ExternalURL  string     `json:"external_url,omitempty"`
	Alt     string          `json:"alt,omitempty"`
	Width   *int            `json:"width,omitempty"`
	Height  *int            `json:"height,omitempty"`
}

// UnmarshalBlock unmarshals JSON data into a Block of the appropriate type.
func UnmarshalBlock(data []byte) (Block, error) {
	var rb RawBlock
	if err := json.Unmarshal(data, &rb); err != nil {
		return nil, err
	}

	switch rb.Type {
	case "paragraph":
		var content []InlineSpan
		if err := json.Unmarshal(rb.Content, &content); err != nil {
			return nil, err
		}
		return ParagraphBlock{Type: "paragraph", Content: content}, nil
	case "heading":
		var content []InlineSpan
		if err := json.Unmarshal(rb.Content, &content); err != nil {
			return nil, err
		}
		return HeadingBlock{Type: "heading", Level: rb.Level, Content: content}, nil
	case "list":
		var items []ListItem
		if err := json.Unmarshal(rb.Items, &items); err != nil {
			return nil, err
		}
		return ListBlock{Type: "list", Ordered: rb.Ordered, Items: items}, nil
	case "code":
		var codeContent string
		if err := json.Unmarshal(rb.Content, &codeContent); err != nil {
			return nil, err
		}
		return CodeBlock{Type: "code", Lang: rb.Lang, Content: codeContent}, nil
	case "blockquote":
		// For blockquotes, recursively unmarshal nested blocks
		var rawBlocks []json.RawMessage
		if err := json.Unmarshal(rb.Content, &rawBlocks); err != nil {
			return nil, err
		}
		var content []Block
		for _, raw := range rawBlocks {
			block, err := UnmarshalBlock(raw)
			if err != nil {
				continue
			}
			if block != nil {
				content = append(content, block)
			}
		}
		return BlockquoteBlock{Type: "blockquote", Content: content}, nil
	case "image":
		return ImageBlock{
			Type:         "image",
			AttachmentID: rb.AttachmentID,
			ExternalURL:  rb.ExternalURL,
			Alt:          rb.Alt,
			Width:        rb.Width,
			Height:       rb.Height,
		}, nil
	case "divider":
		return DividerBlock{Type: "divider"}, nil
	default:
		return nil, nil
	}
}

// MarshalBlock marshals a Block to JSON with type discrimination.
func MarshalBlock(b Block) ([]byte, error) {
	switch block := b.(type) {
	case ParagraphBlock:
		return json.Marshal(map[string]interface{}{
			"type":    "paragraph",
			"content": block.Content,
		})
	case HeadingBlock:
		return json.Marshal(map[string]interface{}{
			"type":    "heading",
			"level":   block.Level,
			"content": block.Content,
		})
	case ListBlock:
		return json.Marshal(map[string]interface{}{
			"type":    "list",
			"ordered": block.Ordered,
			"items":   block.Items,
		})
	case CodeBlock:
		return json.Marshal(map[string]interface{}{
			"type":    "code",
			"lang":    block.Lang,
			"content": block.Content,
		})
	case BlockquoteBlock:
		return json.Marshal(map[string]interface{}{
			"type":    "blockquote",
			"content": block.Content,
		})
	case ImageBlock:
		return json.Marshal(block)
	case DividerBlock:
		return json.Marshal(map[string]interface{}{
			"type": "divider",
		})
	default:
		return nil, nil
	}
}

// UnmarshalJSON unmarshals the Body's blocks field.
func (b *Body) UnmarshalJSON(data []byte) error {
	type Alias Body
	aux := &struct {
		Blocks []json.RawMessage `json:"blocks"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	for _, raw := range aux.Blocks {
		block, err := UnmarshalBlock(raw)
		if err != nil {
			continue
		}
		if block != nil {
			b.Blocks = append(b.Blocks, block)
		}
	}
	return nil
}

// MarshalJSON marshals the Body's blocks field.
func (b Body) MarshalJSON() ([]byte, error) {
	type Alias Body
	aux := struct {
		Blocks []json.RawMessage `json:"blocks"`
		*Alias
	}{
		Alias: (*Alias)(&b),
	}
	for _, block := range b.Blocks {
		data, err := MarshalBlock(block)
		if err != nil {
			continue
		}
		aux.Blocks = append(aux.Blocks, data)
	}
	return json.Marshal(aux)
}
