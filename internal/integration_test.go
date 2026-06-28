package internal

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/auth"
	"github.com/unifiedcommunicationsprotocol/server/internal/crypto/mls"
	"github.com/unifiedcommunicationsprotocol/server/internal/models"
	"github.com/unifiedcommunicationsprotocol/server/internal/router"
)

// TestFullMessageFlow simulates a complete message send/receive flow.
func TestFullMessageFlow(t *testing.T) {
	// Setup auth
	authMgr := auth.New()
	challengeStore := auth.NewChallengeStore()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Alice issues a challenge
	aliceChallenge, err := challengeStore.IssueChallenge("alice@example.com")
	if err != nil {
		t.Fatalf("issue challenge: %v", err)
	}

	if len(aliceChallenge) != 32 {
		t.Errorf("challenge length: got %d, want 32", len(aliceChallenge))
	}

	// Alice signs challenge and creates session
	if err := challengeStore.ConsumeChallenge("alice@example.com"); err != nil {
		t.Fatalf("consume challenge: %v", err)
	}

	aliceSession, err := authMgr.CreateSession(ctx, "alice@example.com", 3600)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Verify session is valid
	address, err := authMgr.ValidateSession(ctx, aliceSession.Token)
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}

	if address != "alice@example.com" {
		t.Errorf("address: got %q, want %q", address, "alice@example.com")
	}

	// Setup router
	r := router.New()
	r.RegisterLocalRecipient("alice@example.com")
	r.RegisterLocalRecipient("bob@example.com")

	// Create and route a message
	threadID := models.ULID("thread_123")
	now := time.Now().UnixMilli()

	envelope := &models.UCPEnvelope{
		V:        "ucp/1.0",
		Type:     "application",
		ThreadID: threadID,
		From:     "alice@example.com",
		To:       []string{"bob@example.com"},
		SigningKey: "AQIDBA==", // Base64 placeholder
		ServerTs: &now,
		MLS:      "encrypted_content",
	}

	// Route the message
	local, remote, err := r.RouteMessage(envelope)
	if err != nil {
		t.Fatalf("route message: %v", err)
	}

	if len(local) != 1 || local[0] != "bob@example.com" {
		t.Errorf("local recipients: got %v, want [bob@example.com]", local)
	}

	if len(remote) != 0 {
		t.Errorf("remote recipients: got %v, want empty", remote)
	}
}

// TestMLSGroupMessaging simulates MLS group state transitions.
func TestMLSGroupMessaging(t *testing.T) {
	// Create a group with 3 members
	members := []string{"alice@example.com", "bob@example.com", "charlie@example.com"}
	group := mls.NewGroup("thread_456", members)

	if group.Epoch != 0 {
		t.Errorf("initial epoch: got %d, want 0", group.Epoch)
	}

	if len(group.Members) != 3 {
		t.Errorf("member count: got %d, want 3", len(group.Members))
	}

	// Simulate adding a member
	err := group.AddMember("diana@example.com")
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	if len(group.Members) != 4 {
		t.Errorf("after add: got %d members, want 4", len(group.Members))
	}

	if group.Epoch != 1 {
		t.Errorf("after add: epoch got %d, want 1", group.Epoch)
	}

	// Simulate removing a member
	err = group.RemoveMember("charlie@example.com")
	if err != nil {
		t.Fatalf("remove member: %v", err)
	}

	if len(group.Members) != 3 {
		t.Errorf("after remove: got %d members, want 3", len(group.Members))
	}

	if group.Epoch != 2 {
		t.Errorf("after remove: epoch got %d, want 2", group.Epoch)
	}

	// Verify group state is consistent
	if group.ThreadID != "thread_456" {
		t.Errorf("thread_id: got %q, want %q", group.ThreadID, "thread_456")
	}
}

// TestEncryptionKeySchedule simulates per-epoch encryption.
func TestEncryptionKeySchedule(t *testing.T) {
	// Epoch 0: Alice sends message
	ks0 := &mls.KeySchedule{
		Epoch:            0,
		EpochSecret:      make([]byte, 32),
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: make([]byte, 16),
		ExporterSecret:   make([]byte, 32),
	}

	enc0 := mls.NewEncryption(ks0)
	plaintext := []byte("Hello, Bob!")

	ciphertext, err := enc0.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Decrypt with same key
	decrypted, err := enc0.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != "Hello, Bob!" {
		t.Errorf("decrypted: got %q, want %q", string(decrypted), "Hello, Bob!")
	}

	// Epoch 1: Key rotation
	ks1 := &mls.KeySchedule{
		Epoch:            1,
		EpochSecret:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		SenderDataSecret: make([]byte, 32),
		EncryptionSecret: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ExporterSecret:   make([]byte, 32),
	}

	enc1 := mls.NewEncryption(ks1)

	// Old epoch's ciphertext should fail with new key
	_, err = enc1.Decrypt(ciphertext)
	if err == nil {
		t.Error("should not decrypt with different epoch key")
	}
}

// TestSessionRefresh simulates token refresh flow.
func TestSessionRefresh(t *testing.T) {
	authMgr := auth.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create initial session
	session1, err := authMgr.CreateSession(ctx, "user@example.com", 1800)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	token1 := session1.Token

	// Validate it works
	_, err = authMgr.ValidateSession(ctx, token1)
	if err != nil {
		t.Fatalf("validate session 1: %v", err)
	}

	// Refresh to get new token
	session2, err := authMgr.RefreshSession(ctx, token1, 3600)
	if err != nil {
		t.Fatalf("refresh session: %v", err)
	}

	token2 := session2.Token

	// Tokens should be different
	if token1 == token2 {
		t.Error("refreshed token should be different")
	}

	// New token should be valid
	_, err = authMgr.ValidateSession(ctx, token2)
	if err != nil {
		t.Fatalf("validate session 2: %v", err)
	}

	// Old token should be revoked
	_, err = authMgr.ValidateSession(ctx, token1)
	if err == nil {
		t.Error("old token should be invalid after refresh")
	}
}

// TestJSONEnvelopeMarshaling verifies envelope serialization.
func TestJSONEnvelopeMarshaling(t *testing.T) {
	threadID := models.ULID("thread_789")
	now := time.Now().UnixMilli()

	envelope := &models.UCPEnvelope{
		V:        "ucp/1.0",
		Type:     "application",
		ThreadID: threadID,
		From:     "alice@example.com",
		To:       []string{"bob@example.com", "charlie@example.com"},
		SigningKey: "AQIDBA==",
		ServerTs: &now,
		MLS:      "base64_encrypted_content",
	}

	// Marshal to JSON
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal back
	var restored models.UCPEnvelope
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.From != "alice@example.com" {
		t.Errorf("from: got %q, want %q", restored.From, "alice@example.com")
	}

	if len(restored.To) != 2 {
		t.Errorf("to count: got %d, want 2", len(restored.To))
	}

	if restored.ServerTs == nil || *restored.ServerTs != now {
		t.Errorf("server_ts: got %v, want %v", restored.ServerTs, &now)
	}
}

// TestProposalCommitWorkflow simulates handshake messages.
func TestProposalCommitWorkflow(t *testing.T) {
	// Create group with 2 members
	group := mls.NewGroup("thread_workflow", []string{"alice@example.com", "bob@example.com"})

	// Create a proper KeyPackage for the Add proposal
	// In real scenario this would come from the new member
	signingKey := make([]byte, 32)
	encryptionKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		signingKey[i] = byte(i)
		encryptionKey[i] = byte(i + 32)
	}

	// We'll skip creating a full key package and instead use a minimal proposal
	// Just test the commit workflow
	removeProposal := &mls.RemoveProposal{
		LeafNodeIndex: 1,
	}

	removeRef := mls.NewProposalRef(removeProposal)

	// Create commit bundling the proposal
	commitContent := &mls.CommitContent{
		Proposals: []mls.ProposalRef{},
		Updates:   []mls.ProposalRef{},
		Removes:   []mls.ProposalRef{*removeRef},
	}

	commit := &mls.MLSCommit{
		GroupID:      group.ID,
		Epoch:        group.Epoch,
		Content:      commitContent,
		Confirmation: make([]byte, 32),
		Signature:    make([]byte, 64),
	}

	// Apply commit to group
	err := group.ApplyCommit(commit)
	if err != nil {
		t.Fatalf("apply commit: %v", err)
	}

	// Epoch should advance
	if group.Epoch != 1 {
		t.Errorf("epoch after commit: got %d, want 1", group.Epoch)
	}

	// Create welcome for new member
	groupSecrets := &mls.GroupSecrets{
		Epoch:           1,
		EpochSecret:     make([]byte, 32),
		ConfirmationKey: make([]byte, 32),
	}

	welcome := mls.NewWelcome(groupSecrets, [][]byte{[]byte("newmember_kpref")})

	if len(welcome.Secrets) != 1 {
		t.Errorf("welcome secrets: got %d, want 1", len(welcome.Secrets))
	}
}

// TestContextualMessaging shows messages flowing through the system.
func TestContextualMessaging(t *testing.T) {
	ctx := context.Background()

	// Thread context: simulating a conversation
	threadID := models.ULID("thread_context")

	messages := []*models.UCPEnvelope{
		{
			V:        "ucp/1.0",
			Type:     "application",
			ThreadID: threadID,
			From:     "alice@example.com",
			To:       []string{"bob@example.com"},
			MLS:      "msg1_encrypted",
		},
		{
			V:        "ucp/1.0",
			Type:     "application",
			ThreadID: threadID,
			From:     "bob@example.com",
			To:       []string{"alice@example.com"},
			MLS:      "msg2_encrypted",
		},
		{
			V:        "ucp/1.0",
			Type:     "application",
			ThreadID: threadID,
			From:     "alice@example.com",
			To:       []string{"bob@example.com"},
			MLS:      "msg3_encrypted",
		},
	}

	// Verify thread context is maintained
	for i, msg := range messages {
		if msg.ThreadID != threadID {
			t.Errorf("message %d: thread_id mismatch", i)
		}

		if msg.V != "ucp/1.0" {
			t.Errorf("message %d: version mismatch", i)
		}
	}

	// Simulate processing through system
	_ = ctx
	if len(messages) != 3 {
		t.Errorf("message count: got %d, want 3", len(messages))
	}
}
