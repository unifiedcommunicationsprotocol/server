# Architecture

## System Overview

UCP Server is a reference implementation of the Unified Communications Protocol — a modern replacement for IMAP/SMTP/CalDAV/CardDAV. The server operates as a zero-knowledge relay: it receives messages encrypted via MLS, routes them to recipients based on unencrypted headers, and stores ciphertext without decrypting (unless a user grants an opt-in key share for search/summary). The architecture is stateless and federation-ready: any server can talk to any other via mutually authenticated persistent connections. A single binary compiles from Go with no external runtime dependencies, designed to run on a VPS with Postgres for durability.

## Component Map

```
┌─────────────────────────────────────────────────────────────┐
│                      UCP Server                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Transport Layer (net/http + WebSocket/WebTransport)│  │
│  │  ├─ Connection negotiation (version, capabilities)  │  │
│  │  ├─ Session authentication (challenge-response)     │  │
│  │  └─ Persistent connection state management          │  │
│  └──────┬───────────────────────────────────────────────┘  │
│         │                                                   │
│  ┌──────▼──────────────────────────────────────────────┐  │
│  │  Request Router & Middleware                        │  │
│  │  ├─ Auth validation (session token, signing key)    │  │
│  │  ├─ Rate limiting                                   │  │
│  │  └─ Request dispatch (API, federation, bridge)      │  │
│  └──┬──────────┬──────────────────┬──────────┬──────────┘  │
│     │          │                  │          │              │
│  ┌──▼──┐  ┌──▼──┐  ┌──────────┐ ┌─▼──┐  ┌──▼────┐         │
│  │ API │  │Feder│  │  Bridge  │ │ AI │  │  Auth │         │
│  │     │  │ation│  │ (IMAP/   │ │Proc│  │       │         │
│  │     │  │     │  │  SMTP)   │ │ess │  │       │         │
│  └──┬──┘  └──┬──┘  └────┬─────┘ └─┬──┘  └───┬───┘         │
│     │       │           │        │          │              │
│  ┌──▼───────▼───────────▼────────▼──────────▼──────────┐  │
│  │  Crypto Layer (MLS RFC 9420)                        │  │
│  │  ├─ Group creation & membership                     │  │
│  │  ├─ Message encryption/decryption                   │  │
│  │  ├─ Epoch advancement & key rotation               │  │
│  │  └─ Key package management                          │  │
│  └──────────┬───────────────────────────────────────────┘  │
│             │                                               │
│  ┌──────────▼───────────────────────────────────────────┐  │
│  │  Store Layer (Message & Identity)                   │  │
│  │  ├─ Message envelope storage (encrypted)            │  │
│  │  ├─ Identity records (keys, DNS metadata)           │  │
│  │  ├─ Session & token management                      │  │
│  │  └─ Indexes (thread_id, to/from, server_ts)         │  │
│  └──────────┬───────────────────────────────────────────┘  │
│             │                                               │
│         ┌───▼────────────────┐                            │
│         │   Postgres 18+     │                            │
│         │  (persistence)     │                            │
│         └────────────────────┘                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘

External (Federation):
┌──────────────┐          ┌──────────────┐
│ Remote UCP   │◄────────►│ This Server  │
│ Server       │ Mutual   │ (federation) │
└──────────────┘ auth     └──────────────┘
```

## Transport Layer Detail

**Connection Negotiation:**
1. Client attempts **WebTransport** (HTTP/3 / QUIC) first — if available and connects within 500ms, use it
2. Fallback to **WebSocket** (HTTP/1.1 or HTTP/2) if WebTransport unavailable or fails
3. Both client and server MUST support WebSocket; WebTransport is SHOULD support (preferred path)
4. Single persistent connection handles push delivery, API calls, and federation
5. Keepalive ping every 30 seconds of inactivity; pong timeout 10 seconds

**Handshake (UCPHello → UCPHelloAck):**
- Client sends `UCPHello` with protocol version and session token
- Server responds with `UCPHelloAck` containing server ID, server signature (proves domain ownership via Ed25519), and optional stale key share list
- Version negotiation: both parties agree on lowest common version
- Server signature binds proof to session and server identity (independent of TLS)

## Data Flow

### Inbound Message (Send)

1. **Client connects** → Transport layer attempts WebTransport first, falls back to WebSocket; performs `UCPHello` handshake with server signature verification
2. **Client authenticates** → Challenge-response via signing key, issues session token
3. **Client sends message** → POST to `/api/message/send` with `UCPEnvelope` (unencrypted) + `MLSMessage` (encrypted)
4. **Auth validates** → Confirms session token, verifies signing key matches authenticated sender
5. **Crypto layer** → Deserializes MLS envelope, decrypts content (if server processing enabled), validates MLS group state
6. **Recipient resolution** → For each `to`/`cc` address, resolves via well-known endpoint to determine destination server(s)
7. **Store layer** → Writes envelope to `messages` table with `server_ts`, unencrypted routing metadata indexed
8. **Federation** → For remote recipients, queues delivery to remote server's federation endpoint
9. **Local delivery** → Streams message to connected clients in the thread group via persistent connection
10. **Response** → Returns `200 OK` with `envelope_id` for idempotency

### Inbound Message (Receive via Federation)

1. **Remote server connects** → Transport layer establishes federation connection, mutual Ed25519 authentication
2. **Remote server delivers** → Sends `UCPDeliver` or `UCPBundledDeliver` with envelope(s)
3. **Bundle dedup** → Checks `bundle_id` against bundle log; if committed, returns `UCPAck` immediately
4. **Store layer** → Writes envelope(s) to `messages` table with server-assigned `server_ts`
5. **Local delivery** → Streams to connected recipients via persistent connections
6. **ACK to sender** → Responds with `UCPAck` to confirm durable storage

### Bridge Inbound (SMTP → UCP)

1. **SMTP arrives** → Port 25 receives message, parses MIME structure
2. **Threading** → Looks up `References`/`In-Reply-To` in bridge threading map, or creates new thread
3. **KeyPackage check** → Fetches recipient's current KeyPackage from well-known endpoint
4. **Conversion** → HTML → blocks (lossy), extracts attachments, builds message payload
5. **Crypto** → Creates or joins MLS group, encrypts envelope to recipient
6. **Store** → Writes with `bridge_attestation` in place of signature
7. **Local delivery** → Streams to recipient's connected clients

### Message Read Path

1. **Client queries** → `GET /api/inbox?thread_id=...` with session token
2. **Auth validates** → Confirms session, checks permissions
3. **Store retrieves** → Fetches encrypted envelopes from `messages` table ordered by `server_ts`
4. **Stream to client** → Returns envelopes; client performs MLS decryption locally
5. **Client processes** → Verifies signatures, renders blocks, generates local AI metadata if needed

## Package Responsibilities

| Package | Responsibility |
|---------|---|
| `transport` | WebSocket/WebTransport connection negotiation, `UCPHello` handshake, frame encoding/decoding |
| `identity` | Ed25519 keypair generation, DNS TXT record parsing, signing key lifecycle, well-known endpoint responses |
| `crypto` | MLS group creation/state, envelope encryption/decryption, epoch advancement, KeyPackage validation |
| `router` | Federation connection state, message routing to local/remote recipients, retry/bounce logic |
| `store` | Postgres schema, message/identity/session/token persistence, indexes, transaction handling |
| `bridge` | IMAP connection pooling, SMTP inbound parsing, HTML↔blocks conversion, IMAP/SMTP header mapping |
| `ai` | Local client inference schema, server processing key derivation, opt-in decryption, stale key share handling |
| `api` | HTTP endpoint handlers, well-known routes, request validation, session token verification |
| `auth` | Challenge generation, signature verification, session token issuance/refresh/revocation |
| `models` | Type definitions (Message, Envelope, Identity, KeyPackage, etc.) — protocol-aligned structs |

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go 1.23+ | Single binary, cross-compile, no cgo, no runtime dependencies |
| HTTP Transport | `net/http` (stdlib) | Minimal dependencies, direct handler composition, WebSocket via stdlib upgrade |
| Database | Postgres 18+ | Enterprise-grade durability, JSONB for flexible metadata, row-level constraints for security |
| SQL Access | TBD: sqlc or pgx | Code generation (sqlc) or driver (pgx); no ORM |
| MLS Library | TBD | Must implement RFC 9420 exactly; evaluate mlspp bindings vs pure Go |
| Federation | Persistent connections | Stateful federation connections per remote domain reduce handshake overhead |
| Bridge | First-class subsystem | IMAP/SMTP conversion on server, not client — adoption path for legacy email users |
| Secrets Management | Environment variables at startup | No runtime config; typed struct injection into handlers |
| Deployment | Single binary + Postgres | Operationally simple; stateless (federation state is ephemeral) |

Full decision records: `docs/decisions.md`

## Message Content Types

The `UCPApplicationData` wrapper inside MLS encrypted payloads begins with a single-byte `content_type` to allow efficient dispatch:

| Type | Byte | Purpose | Defined |
|------|------|---------|---------|
| `message` | `0x01` | Email message (includes forwards) | UCP/1.0 |
| `0x02` | `0x02` | Reserved for `reaction` in UCP/1.1 — ignore in 1.0 | UCP/1.1 |
| `receipt` | `0x03` | Read receipt | UCP/1.0 |
| `edit` | `0x04` | Message edit (modifies prior `message`) | UCP/1.0 |
| `delete` | `0x05` | Message deletion | UCP/1.0 |
| `attachment` | `0x06` | Attachment content (used on `/content` download) | UCP/1.0 |

This separation of content type from JSON payload allows lightweight processing (receipts, edits) without full message parsing.

## Signing Key Lifecycle

**Timeline & Rotation:**
- **Lifetime:** Default 60 days (configurable 30-90 days)
- **Rotation window:** Must begin rotation no later than **7 days before expiry**
- **Grace period:** Old signing key valid for verification only for **48 hours** after rotation
- **Cutover:** Client begins signing new messages immediately with new key
- **Expiry:** After grace period, old key marked expired; removed from well-known response after 7 more days

**Why:** Forward secrecy at the identity layer. A compromised signing key is bounded to its 60-day window; MLS provides post-compromise security through epoch key deletion and automatic MLS Update on key rotation.

## Bundle Idempotency (Federation)

New threads require atomic delivery of Welcome + first message via `UCPBundledDeliver`:

- **Bundle ID:** Sender-generated ULID, stable across all retry attempts
- **Bundle Log:** Receiver maintains log for 72 hours (covers 48-hour retry window + buffer)
- **Atomic Commit:** Bundle log entry transitions from `pending` → `committed` in single database transaction alongside envelope storage
- **Commit-before-forward:** Receiver MUST NOT forward Welcome to clients until full commit succeeds

This ensures: (1) no duplicate threads if sender retries after crash, (2) no thread with Welcome but no first message, (3) recovery from mid-commit crashes.

**Spec Reference:** The normative UCP protocol specification lives at https://github.com/unifiedcommunicationsprotocol/spec. When this document says "follows RFC 9420" or "per spec/core.md", refer to the published spec for exact definitions.

## External Integrations

| Service | Purpose | Auth method |
|---------|---------|-------------|
| DNS (recursive resolver) | Identity bootstrapping, server discovery, federation endpoint resolution | None (read-only) |
| IMAP servers (Gmail, Outlook, Fastmail) | Account bridge inbound | OAuth2 (preferred) or app passwords |
| SMTP servers (Gmail, Outlook, Fastmail) | Account bridge outbound | OAuth2 (preferred) or app passwords |
| MX DNS | Inbound SMTP gateway discovery | None (read-only) |
| Postgres 18+ | Message/identity/session persistence, row-level security, JSONB metadata | Unix socket or TLS (environment-injected credentials) |

## Security Considerations

**Authentication & Authorization:**
- Session tokens issued post-challenge-response, verified at middleware before handler
- Tokens are opaque, short-lived (24h max), revocable, refreshable
- No user password — identity is Ed25519 keypair only
- Signing key rotation automatic every 60 days; grace period 48h covers in-flight messages

**Message Security:**
- All messages encrypted via MLS; server sees only routing metadata (from, to, thread_id, signing_key)
- Envelope signature verified before MLS decryption; payload signature verified after decryption
- Forward secrecy via MLS epoch key deletion; post-compromise security via signing key rotation
- Server blindness by default; opt-in server processing requires explicit user action and per-group key shares

**Data Isolation:**
- Postgres row-level constraints: each identity sees only messages they are a recipient of
- No cross-identity leakage in threading, BCC groups, or federation delivery
- Bridge attestation signed by server key; clients verify before rendering legacy messages
- Credential storage for account bridges encrypted at rest; never logged or transmitted in plaintext

**Federation:**
- Mutual Ed25519 authentication before any message delivery
- Sender's server responsible for retry/bounce logic; receiver acknowledges durable storage only
- Bundle idempotency via sender-generated bundle_id; receiver maintains 72h bundle log
- Rate limiting per-domain (implementation-defined, minimum 1000 msg/min baseline)

---

*Last updated: 2026-06-26*
