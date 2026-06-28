-- UCP Server Schema v1
-- Initialize all core tables for messages, identities, sessions, attachments

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Identities: UCP user identities with keypairs
CREATE TABLE identities (
  id BIGSERIAL PRIMARY KEY,
  address TEXT UNIQUE NOT NULL,
  identity_key TEXT NOT NULL,
  signing_keys_json JSONB NOT NULL DEFAULT '[]',
  revocation_key TEXT NOT NULL,
  revocation_record JSONB,
  server TEXT,
  capabilities TEXT[] DEFAULT ARRAY['ucp/1.0'],
  preferences JSONB DEFAULT '{"rendering":"html","read_receipts":false,"external_images":false,"language":"en"}',
  server_processing JSONB DEFAULT '{"enabled":false,"scopes":[],"granted_at":null}',
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_identities_address ON identities(address);

-- Sessions: authenticated user sessions
CREATE TABLE sessions (
  id BIGSERIAL PRIMARY KEY,
  address TEXT NOT NULL REFERENCES identities(address) ON DELETE CASCADE,
  token TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  revoked_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW(),
  last_used_at TIMESTAMP
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_address ON sessions(address);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Messages: encrypted message envelopes
CREATE TABLE messages (
  id BIGSERIAL PRIMARY KEY,
  message_id TEXT UNIQUE NOT NULL,
  thread_id TEXT NOT NULL,
  from_addr TEXT NOT NULL,
  to_addrs TEXT[] NOT NULL,
  signing_key TEXT NOT NULL,
  mls_encrypted BYTEA NOT NULL,
  server_ts BIGINT NOT NULL,
  client_ts BIGINT,
  message_type TEXT DEFAULT 'application',
  created_at TIMESTAMP DEFAULT NOW(),
  FOREIGN KEY (from_addr) REFERENCES identities(address)
);

CREATE INDEX idx_messages_thread_id ON messages(thread_id);
CREATE INDEX idx_messages_from_addr ON messages(from_addr);
CREATE INDEX idx_messages_server_ts ON messages(server_ts);
CREATE INDEX idx_messages_to_addrs ON messages USING GIN(to_addrs);

-- Attachments: message attachment metadata
CREATE TABLE attachments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size BIGINT NOT NULL,
  sha256 TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Message attachments: join table
CREATE TABLE message_attachments (
  id BIGSERIAL PRIMARY KEY,
  message_id TEXT NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
  attachment_id TEXT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_message_attachments_message_id ON message_attachments(message_id);

-- MLS Groups: for tracking MLS group state
CREATE TABLE mls_groups (
  id TEXT PRIMARY KEY,
  thread_id TEXT NOT NULL,
  epoch BIGINT DEFAULT 0,
  members TEXT[] NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_mls_groups_thread_id ON mls_groups(thread_id);

-- Key Packages: for group creation
CREATE TABLE key_packages (
  id TEXT PRIMARY KEY,
  address TEXT NOT NULL REFERENCES identities(address) ON DELETE CASCADE,
  group_id TEXT,
  init_key TEXT NOT NULL,
  signature_key TEXT NOT NULL,
  signature TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_key_packages_address ON key_packages(address);
CREATE INDEX idx_key_packages_group_id ON key_packages(group_id);

-- Server processing key shares: for opt-in server-side decryption
CREATE TABLE key_shares (
  id BIGSERIAL PRIMARY KEY,
  address TEXT NOT NULL REFERENCES identities(address) ON DELETE CASCADE,
  group_id TEXT NOT NULL,
  epoch BIGINT NOT NULL,
  key_material TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_key_shares_address_group_id ON key_shares(address, group_id);

-- Federation: remote server connections and retry state
CREATE TABLE federation_connections (
  id BIGSERIAL PRIMARY KEY,
  remote_domain TEXT UNIQUE NOT NULL,
  established_at TIMESTAMP,
  last_heartbeat TIMESTAMP,
  status TEXT DEFAULT 'pending',
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_federation_connections_remote_domain ON federation_connections(remote_domain);

-- Delivery retry queue: for federation message retry
CREATE TABLE delivery_queue (
  id BIGSERIAL PRIMARY KEY,
  envelope_id TEXT NOT NULL,
  recipient TEXT NOT NULL,
  thread_id TEXT NOT NULL,
  attempted_at TIMESTAMP NOT NULL,
  next_retry TIMESTAMP NOT NULL,
  retries INT DEFAULT 0,
  status TEXT DEFAULT 'pending',
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_delivery_queue_status ON delivery_queue(status);
CREATE INDEX idx_delivery_queue_next_retry ON delivery_queue(next_retry);

-- Bridge IMAP connections: for account bridging
CREATE TABLE bridge_imap_accounts (
  id TEXT PRIMARY KEY,
  address TEXT NOT NULL REFERENCES identities(address) ON DELETE CASCADE,
  imap_host TEXT NOT NULL,
  imap_port INT NOT NULL,
  imap_username TEXT NOT NULL,
  auth_token TEXT NOT NULL,
  last_sync TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_bridge_imap_accounts_address ON bridge_imap_accounts(address);

-- Bridge threading map: SMTP Message-ID ↔ UCP ULID
CREATE TABLE bridge_threading_map (
  id BIGSERIAL PRIMARY KEY,
  smtp_message_id TEXT NOT NULL,
  ucp_message_id TEXT NOT NULL,
  thread_id TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_bridge_threading_map_smtp_id ON bridge_threading_map(smtp_message_id);
CREATE INDEX idx_bridge_threading_map_ucp_id ON bridge_threading_map(ucp_message_id);
CREATE INDEX idx_bridge_threading_map_thread_id ON bridge_threading_map(thread_id);

-- Bundle log: for federation delivery idempotency
CREATE TABLE federation_bundle_log (
  id BIGSERIAL PRIMARY KEY,
  bundle_id TEXT UNIQUE NOT NULL,
  status TEXT DEFAULT 'pending',
  received_at TIMESTAMP NOT NULL,
  committed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_federation_bundle_log_status ON federation_bundle_log(status);
CREATE INDEX idx_federation_bundle_log_committed_at ON federation_bundle_log(committed_at);

-- Row-Level Security (RLS) Policies
-- Enable RLS on all tables that contain user-specific data

ALTER TABLE identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE key_packages ENABLE ROW LEVEL SECURITY;
ALTER TABLE key_shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE bridge_imap_accounts ENABLE ROW LEVEL SECURITY;

-- RLS Policy: identities — each user can only view their own identity
CREATE POLICY identities_select_policy ON identities FOR SELECT
  USING (address = current_user_addr());

CREATE POLICY identities_insert_policy ON identities FOR INSERT
  WITH CHECK (address = current_user_addr());

CREATE POLICY identities_update_policy ON identities FOR UPDATE
  USING (address = current_user_addr())
  WITH CHECK (address = current_user_addr());

-- RLS Policy: sessions — each user can only view/manage their own sessions
CREATE POLICY sessions_select_policy ON sessions FOR SELECT
  USING (address = current_user_addr());

CREATE POLICY sessions_update_policy ON sessions FOR UPDATE
  USING (address = current_user_addr())
  WITH CHECK (address = current_user_addr());

CREATE POLICY sessions_delete_policy ON sessions FOR DELETE
  USING (address = current_user_addr());

-- RLS Policy: messages — users can only view messages where they are a recipient
-- This requires checking the from_addr or to_addrs array
CREATE POLICY messages_select_policy ON messages FOR SELECT
  USING (
    from_addr = current_user_addr() OR
    to_addrs @> ARRAY[current_user_addr()]
  );

CREATE POLICY messages_insert_policy ON messages FOR INSERT
  WITH CHECK (from_addr = current_user_addr());

-- RLS Policy: key_packages — users can only view their own key packages
CREATE POLICY key_packages_select_policy ON key_packages FOR SELECT
  USING (address = current_user_addr());

CREATE POLICY key_packages_insert_policy ON key_packages FOR INSERT
  WITH CHECK (address = current_user_addr());

CREATE POLICY key_packages_delete_policy ON key_packages FOR DELETE
  USING (address = current_user_addr());

-- RLS Policy: key_shares — users can only view their own key shares
CREATE POLICY key_shares_select_policy ON key_shares FOR SELECT
  USING (address = current_user_addr());

CREATE POLICY key_shares_insert_policy ON key_shares FOR INSERT
  WITH CHECK (address = current_user_addr());

CREATE POLICY key_shares_update_policy ON key_shares FOR UPDATE
  USING (address = current_user_addr())
  WITH CHECK (address = current_user_addr());

-- RLS Policy: bridge_imap_accounts — users can only view their own bridge accounts
CREATE POLICY bridge_imap_accounts_select_policy ON bridge_imap_accounts FOR SELECT
  USING (address = current_user_addr());

CREATE POLICY bridge_imap_accounts_insert_policy ON bridge_imap_accounts FOR INSERT
  WITH CHECK (address = current_user_addr());

CREATE POLICY bridge_imap_accounts_update_policy ON bridge_imap_accounts FOR UPDATE
  USING (address = current_user_addr())
  WITH CHECK (address = current_user_addr());

CREATE POLICY bridge_imap_accounts_delete_policy ON bridge_imap_accounts FOR DELETE
  USING (address = current_user_addr());

-- Helper function: get current authenticated user address
-- This is called from RLS policies to determine the current user
-- In production, this should be set via a Postgres role or jwt claims extension
CREATE OR REPLACE FUNCTION current_user_addr() RETURNS TEXT AS $$
  SELECT current_setting('app.current_user_addr', true)::TEXT;
$$ LANGUAGE SQL STABLE;
