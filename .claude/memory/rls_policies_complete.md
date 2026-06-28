---
name: rls_policies_complete
description: Postgres Row-Level Security policies implement database-level data isolation
metadata:
  type: project
---

## Postgres Row-Level Security (RLS) Implementation Complete

**Status:** ✅ DONE (2026-06-28)

**What:** RLS policies enforce user-level data isolation at the database layer on 6 sensitive tables.

**Why:** Database-layer security provides zero-trust data isolation. Even if application code is compromised, Postgres enforces access control. Architectural requirement: "Row-level constraints provide data isolation guarantees."

**Affected Tables:**
- `identities` - users see only their own identity
- `sessions` - users see only their own sessions
- `messages` - users see only messages where they're recipient
- `key_packages` - users see only their own key packages
- `key_shares` - users see only their own key shares
- `bridge_imap_accounts` - users see only their own IMAP accounts

**How It Works:**
1. User authenticates → session token created
2. Protected handler extracts user from token → calls `store.WithUserAddress(ctx, address)`
3. Store method receives context → calls `setRLSUserContext(ctx)` to set Postgres session variable
4. Query executes → RLS policy (using `current_user_addr()` function) filters results
5. Database enforces policy; even `SELECT *` returns only user's data

**Implementation:**
- Helper function `current_user_addr()` reads from `app.current_user_addr` session variable
- Each table has `SELECT`, `INSERT`, `UPDATE`, `DELETE` policies (as applicable)
- RLS enabled with `ALTER TABLE ... ENABLE ROW LEVEL SECURITY`
- Context helpers: `WithUserAddress()`, `getUserAddress()`, `setRLSUserContext()`

**How to Apply:**
- Run migrations to enable RLS: `migrations/001_init_schema.sql`
- Store.GetIdentity(), GetThreadMessages() automatically enforce RLS
- For custom queries: call `setRLSUserContext(ctx)` before executing
- Verify RLS works: try querying as user A → should not see user B's data

**Security Notes:**
- RLS policies use `@>` array operator for efficient recipient lookups on `to_addrs`
- `GetSession()` deliberately NOT RLS-protected (needed for public auth lookups)
- Production: integrate with Postgres JWT extension for better auth integration
- Recommend: periodic audit of RLS policies (can view via `pg_policies`)

**Testing:**
- All store tests passing with RLS enabled
- No data leakage between users verified by tests
- Integration tests confirm context-based user isolation
