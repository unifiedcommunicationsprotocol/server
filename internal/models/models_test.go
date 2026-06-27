package models

import (
	"encoding/json"
	"testing"
)

func TestMessageMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr bool
	}{
		{
			name: "basic email message",
			message: &Message{
				ID:       "01J3K...",
				Version:  "ucp/1.0",
				Type:     "message.email",
				From:     "alice@example.com",
				To:       []string{"bob@example.com"},
				Subject:  "Hello",
				Body: &Body{
					Blocks: []Block{
						ParagraphBlock{
							Type: "paragraph",
							Content: []InlineSpan{
								{Text: "Hello, Bob!"},
							},
						},
					},
					HTML: "<p>Hello, Bob!</p>",
				},
				ThreadID: "01J3K...",
				ClientTs: 1720000000,
				Priority: 3,
				Signature: "base64sig",
			},
			wantErr: false,
		},
		{
			name: "email with forward origin",
			message: &Message{
				ID:       "01J3K_FWD",
				Version:  "ucp/1.0",
				Type:     "message.email",
				From:     "alice@example.com",
				To:       []string{"bob@example.com"},
				Subject:  "Fwd: Important",
				ThreadID: "01J3K_FWD",
				ClientTs: 1720000000,
				Origin: &Origin{
					MessageID: Ptr(ULID("01J3K_ORIG")),
					From:      "charlie@example.com",
					ClientTs:  1719900000,
				},
				Body: &Body{
					Blocks: []Block{
						ParagraphBlock{
							Type:    "paragraph",
							Content: []InlineSpan{{Text: "See below."}},
						},
					},
				},
				Signature: "base64sig",
			},
			wantErr: false,
		},
		{
			name: "email with attachments",
			message: &Message{
				ID:       "01J3K...",
				Version:  "ucp/1.0",
				Type:     "message.email",
				From:     "alice@example.com",
				To:       []string{"bob@example.com"},
				Subject:  "Document",
				ThreadID: "01J3K...",
				ClientTs: 1720000000,
				Body: &Body{
					Blocks: []Block{
						ParagraphBlock{
							Type: "paragraph",
							Content: []InlineSpan{
								{Text: "See attached."},
							},
						},
						ImageBlock{
							Type:         "image",
							AttachmentID: Ptr(ULID("01J3K_ATTACH")),
							Alt:          "Document screenshot",
						},
					},
				},
				Attachments: []Attachment{
					{
						ID:       "01J3K_ATTACH",
						Name:     "document.pdf",
						MimeType: "application/pdf",
						Size:     204800,
						SHA256:   "abc123def456...",
					},
				},
				Signature: "base64sig",
			},
			wantErr: false,
		},
		{
			name: "reply with quoted content",
			message: &Message{
				ID:        "01J3K_REPLY",
				Version:   "ucp/1.0",
				Type:      "message.email",
				From:      "bob@example.com",
				To:        []string{"alice@example.com"},
				Subject:   "Re: Hello",
				ThreadID:  "01J3K...",
				InReplyTo: Ptr(ULID("01J3K...")),
				References: []ULID{"01J3K..."},
				ClientTs:  1720000100,
				Body: &Body{
					Blocks: []Block{
						ParagraphBlock{
							Type: "paragraph",
							Content: []InlineSpan{
								{Text: "Thanks for reaching out!"},
							},
						},
						BlockquoteBlock{
							Type: "blockquote",
							Content: []Block{
								ParagraphBlock{
									Type: "paragraph",
									Content: []InlineSpan{
										{Text: "Hello, Bob!"},
									},
								},
							},
						},
					},
				},
				Signature: "base64sig",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Errorf("Unmarshal error = %v", err)
				return
			}

			// Verify key fields round-trip
			if msg.ID != tt.message.ID {
				t.Errorf("ID mismatch: got %q, want %q", msg.ID, tt.message.ID)
			}
			if msg.From != tt.message.From {
				t.Errorf("From mismatch: got %q, want %q", msg.From, tt.message.From)
			}
			if msg.Subject != tt.message.Subject {
				t.Errorf("Subject mismatch: got %q, want %q", msg.Subject, tt.message.Subject)
			}
		})
	}
}

func TestBlockTypes(t *testing.T) {
	tests := []struct {
		block    Block
		wantType string
	}{
		{ParagraphBlock{}, "paragraph"},
		{HeadingBlock{}, "heading"},
		{ListBlock{}, "list"},
		{CodeBlock{}, "code"},
		{BlockquoteBlock{}, "blockquote"},
		{ImageBlock{}, "image"},
		{DividerBlock{}, "divider"},
	}

	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			if got := tt.block.BlockType(); got != tt.wantType {
				t.Errorf("BlockType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

func TestIdentityMarshaling(t *testing.T) {
	identity := &Identity{
		Address:     "alice@example.com",
		IdentityKey: "base64_identity_pubkey",
		SigningKeys: []SigningKey{
			{
				Key:     "base64_signing_pubkey",
				Expires: 1720000000 + 60*24*60*60,
				Issued:  1720000000,
				Sig:     "base64_identity_sig",
				Status:  "active",
			},
		},
		RevocationKey: "base64_revocation_pubkey",
		Server:        "https://ucp.example.com",
		Capabilities:  []string{"ucp/1.0"},
		Preferences: &Preferences{
			Rendering:      "html",
			ReadReceipts:   false,
			ExternalImages: false,
			Language:       "en",
		},
		ServerProcessing: &ServerProcessing{
			Enabled: false,
			Scopes:  []string{},
		},
	}

	data, err := json.Marshal(identity)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if id.Address != identity.Address {
		t.Errorf("Address mismatch: got %q, want %q", id.Address, identity.Address)
	}
	if len(id.SigningKeys) != 1 {
		t.Errorf("SigningKeys length = %d, want 1", len(id.SigningKeys))
	}
	if id.SigningKeys[0].Status != "active" {
		t.Errorf("SigningKey status = %q, want %q", id.SigningKeys[0].Status, "active")
	}
}

func TestKeyPackageMarshaling(t *testing.T) {
	kp := &KeyPackage{
		Version:     "mls10",
		CipherSuite: "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519",
		InitKey:     "base64_hpke_pubkey",
		LeafNode: &LeafNode{
			EncryptionKey: "base64_hpke_pubkey",
			SignatureKey:  "base64_ed25519_pubkey",
			Credential: &Credential{
				CredentialType: "signing_key",
				SigningKey:     "base64_ed25519_pubkey",
				Identity:       "alice@example.com",
				IdentityKey:    "base64_identity_pubkey",
				IdentitySig:    "base64_identity_sig",
			},
			Capabilities: &Capabilities{
				Extensions: []string{"ucp/1.0"},
			},
			LeafNodeSource: "key_package",
		},
		Extensions: []string{},
		Signature:  "base64_signature",
	}

	data, err := json.Marshal(kp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var kp2 KeyPackage
	if err := json.Unmarshal(data, &kp2); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if kp2.Version != kp.Version {
		t.Errorf("Version mismatch: got %q, want %q", kp2.Version, kp.Version)
	}
	if kp2.LeafNode.Credential.CredentialType != "signing_key" {
		t.Errorf("Credential type mismatch")
	}
}

func TestUCPEnvelopeMarshaling(t *testing.T) {
	ts := int64(1720000000)
	envelope := &UCPEnvelope{
		V:        "ucp/1.0",
		Type:     "application",
		ThreadID: "01J3K...",
		From:     "alice@example.com",
		To:       []string{"bob@example.com"},
		SigningKey: "base64_signing_pubkey",
		ServerTs: &ts,
		MLS:      "base64_mls_message",
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var env UCPEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if env.V != "ucp/1.0" {
		t.Errorf("Version mismatch: got %q, want %q", env.V, "ucp/1.0")
	}
	if env.ServerTs == nil || *env.ServerTs != ts {
		t.Errorf("ServerTs mismatch")
	}
}

func TestUCPError(t *testing.T) {
	err := &UCPError{
		Code:    "version_mismatch",
		Message: "Server does not support requested UCP version",
	}

	data, _ := json.Marshal(err)
	var err2 UCPError
	json.Unmarshal(data, &err2)

	if err2.Code != "version_mismatch" {
		t.Errorf("Code mismatch: got %q, want %q", err2.Code, "version_mismatch")
	}
}

func TestDeliveryFailureMarshaling(t *testing.T) {
	df := &DeliveryFailure{
		Type:           "system.delivery_failure",
		ThreadID:       "01J3K...",
		MessageID:      "01J3K...",
		Recipient:      "bob@example.com",
		AttemptedUntil: 1720172800,
		Reason:         "unknown_recipient",
		ServerSig:      "base64_server_sig",
	}

	data, err := json.Marshal(df)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var df2 DeliveryFailure
	if err := json.Unmarshal(data, &df2); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if df2.Reason != "unknown_recipient" {
		t.Errorf("Reason mismatch: got %q, want %q", df2.Reason, "unknown_recipient")
	}
}

// Helper function to create pointers.
func Ptr[T any](v T) *T {
	return &v
}
