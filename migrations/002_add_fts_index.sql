-- UCP Server Migration 002: Add Full-Text Search Index
-- Adds GIN index on messages.body for efficient full-text search queries
-- This improves search performance from O(n) ILIKE scans to O(log n) index lookups

-- Create full-text search index on message body
CREATE INDEX CONCURRENTLY idx_messages_body_fts ON messages
  USING GIN(to_tsvector('english', COALESCE(body, '')));

-- Comment explaining the index
COMMENT ON INDEX idx_messages_body_fts IS
  'Full-text search index for fast message content queries. Used by handleSearch() to find messages by keyword. Covers to_tsvector indexed queries.';
