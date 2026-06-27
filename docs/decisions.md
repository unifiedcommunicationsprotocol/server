# Decision Log

Architectural Decision Records (ADRs) — listed newest first.

Run `/decision` to add an entry.

---

## ADR-001: Language Choice — Go 1.23+

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
UCP Server must ship as a single binary with zero external runtime dependencies, run on minimal Hetzner VPS infrastructure, and be deployable without Docker. Reference implementation serves as a template for other operators and language communities. Performance and operational simplicity are both critical.

**Options Considered:**
- **Go** — compiled, single binary, no runtime, cross-compile friendly, stdlib covers most needs
- **Rust** — compiled, single binary, no runtime; steeper learning curve, slower initial development
- **TypeScript/Bun** — single binary, zero Node deps, fast iteration; less mature ecosystem for cryptography
- **Python** — simplicity; requires runtime, not suitable for single-binary deployment

**Decision:**
Go 1.23+. Balances fast development (simpler syntax than Rust) with production robustness, mature crypto libraries, and strong stdlib. Single binary deployment aligns with "self-hostable" design principle. No cgo dependency keeps cross-compilation and reproducible builds tractable.

**Consequences:**
- ✅ Single statically-linked binary; easy deployment on minimal VPS
- ✅ Strong stdlib (`net/http`, `net`, `crypto`, `database/sql`); minimal external deps
- ✅ Fast startup, low memory footprint, excellent performance for protocol handler layers
- ⚠️ MLS implementation: no mature pure-Go RFC 9420 library yet; may require bindings or partial custom implementation
- ⚠️ Learning curve for developers unfamiliar with Go; explicit error handling verbosity

---

## ADR-002: HTTP Framework — net/http (stdlib)

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
UCP Server uses persistent WebSocket connections for push delivery, federation, and client-to-server communication. HTTP routing is present but lightweight — most of the protocol lives above HTTP (WebSocket + MLS). A full framework adds unnecessary complexity and dependencies.

**Options Considered:**
- **net/http** (stdlib) — minimal, handlers compose directly, no extra deps, WebSocket via stdlib upgrade
- **Gin** — popular, routing sugar; adds dependency and opinionated middleware
- **Echo** — lightweight; still a dependency and convention layer
- **Fiber** — express-like; not suitable for federation/persistent-connection handling

**Decision:**
Go `net/http` stdlib. WebSocket upgrade is built-in; federation connections are long-lived and hand-managed. Request routing can be explicit (fast, verifiable). No external framework keeps binary size down and makes credential injection into handlers straightforward.

**Consequences:**
- ✅ Zero external HTTP dependency
- ✅ Explicit routing and middleware; easier to audit for security
- ✅ Native WebSocket support via `golang.org/x/net/websocket` or similar (still minimal)
- ⚠️ More boilerplate than framework-based approach (acceptable tradeoff for security-critical code)
- ⚠️ Developers must write middleware composition explicitly (not a disadvantage, but requires discipline)

---

## ADR-003: Database — Postgres 18+

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
Messages must survive server restarts and network partitions. Postgres is ubiquitous, battle-tested, and offers row-level security constraints for multi-tenancy. SQLite is unsuitable for a network service (connection pooling, concurrent writes under load). Managed databases (RDS, Cloud SQL) introduce vendor lock-in contrary to "self-hostable" principle.

**Options Considered:**
- **Postgres 15+** — mature, JSONB, row-level constraints; operators must manage backups
- **Postgres 18+** (chosen) — latest stable, improved performance, better JSON/JSONB handling
- **MySQL 8+** — comparable; weaker JSON support, less suitable for flexible AI metadata
- **CockroachDB** — distributed; operational complexity for single-server deployment

**Decision:**
Postgres 18+ (self-hosted via Hetzner VPS). Row-level constraints provide data isolation guarantees (each identity sees only their messages). JSONB column for flexible AI metadata without schema churn. Operators control backups and can run Postgres in a container on the same VPS without adding external service dependencies.

**Consequences:**
- ✅ Row-level security (RLS) for identity isolation built into DB layer
- ✅ JSONB for `meta.ai`, preferences, and future-proofing
- ✅ Replication and failover are operator concerns; self-hosted aligns with spec's "federation" model
- ⚠️ Operators must manage backups, updates, monitoring (acceptable; detailed in deployment docs)
- ⚠️ Postgres 18 is bleeding-edge; later decisions may pin to 15+ for stability

---

## ADR-004: SQL Access Pattern — Code Generation (sqlc) or Driver (pgx)

**Date:** 2026-06-26
**Status:** Pending

**Context:**
Go's `database/sql` stdlib is flexible but verbose. Two common patterns exist: (1) sqlc, which code-generates type-safe queries from SQL; (2) pgx driver, which provides better error handling and prepared statement caching. Both avoid the implicit complexity of ORMs.

**Options Considered:**
- **sqlc** — SQL-first, generated Go code, type-safe, zero runtime deps beyond Postgres driver
- **pgx** — richer driver API, better perf, still requires manual SQL or query builders
- **gorm, ent, sqlc** — full ORM; violates "no hidden magic" principle
- **Raw database/sql** — verbose; acceptable for critical security code but tedious at scale

**Decision:**
TBD. Recommend starting with `sqlc` (pure code generation, no runtime wrapper) and evaluating `pgx` if query complexity grows beyond sqlc's scope. Decision deferred until schema stabilizes.

**Consequences:**
- (Deferred)

---

## ADR-005: MLS Implementation — RFC 9420 Conformance

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
UCP mandates MLS (RFC 9420) for all message encryption. The spec section `encryption.md` defines the ciphersuite, credential binding, group ID derivation, and envelope schema. Non-compliance will fragment the ecosystem before the protocol is established.

**Options Considered:**
- **Pure-Go implementation** — full control, auditability; massive development effort and crypto risk
- **mlspp bindings** — mature C++ MLS library, Go cgo bindings; introduces cgo (violates single-binary goal)
- **Partial wrapping** — reuse cryptographic primitives (AES, SHA, Ed25519) from `crypto/` stdlib, hand-code MLS state machine; moderate effort, auditable
- **Wait for Go library** — defer launch until a pure-Go RFC 9420 lib exists; unacceptable schedule slip

**Decision:**
Commit to RFC 9420 spec compliance exactly as written in `spec/encryption.md`. Evaluate mlspp bindings (cgo acceptable if necessary) vs partial hand-coding based on schedule. Implementation detail deferred to sprint planning; choice does not affect protocol surface.

**Consequences:**
- ✅ Spec-compliant encryption; no custom algorithm risk
- ✅ Future interoperability with other UCP implementations
- ⚠️ If mlspp bindings required, cgo breaks single-binary goal (mitigated: static linking possible but complex)
- ⚠️ Crypto implementation is security-critical; requires expert review before shipping

---

## ADR-006: Bridge — IMAP/SMTP as First-Class Subsystem

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
UCP's adoption path depends on legacy email integration. Users can immediately start receiving Gmail/Outlook in their UCP client without migrating their address. IMAP/SMTP bridge runs on the server, not the client, to handle the complexity of protocol conversion (MIME↔blocks, threading mapping, DKIM signing).

**Options Considered:**
- **Server-side bridge** (chosen) — centralized, handles conversion at ingress, adopters don't need custom client logic
- **Client-side bridge** — each client implements; complexity sprawl, harder to maintain threading map
- **Separate bridge service** — federation-like; added operational burden and failure modes

**Decision:**
Bridge runs as a subsystem of the UCP Server. Inbound SMTP port 25 (STARTTLS required), IMAP client for account bridging. Conversion happens server-side; bridge attestation (signed by server key) indicates to clients that legacy messages are not E2E encrypted.

**Consequences:**
- ✅ Adopters can use unmodified UCP clients; no custom logic per-client
- ✅ Threading map is server-authoritative; simpler state management
- ✅ Inbound gateway mode (MX records point to UCP server) is a natural use case
- ⚠️ Server must parse MIME, manage IMAP state, implement DKIM/SPF/DMARC — operational complexity
- ⚠️ Account bridge requires secure credential storage (AES-256-GCM at rest); documented in privacy policy

---

## ADR-007: Secrets Management — Environment Variables at Startup

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
Postgres credentials, TLS keys, signing keys, and OAuth tokens must never be hardcoded or logged. The server runs on Hetzner VPS with systemd or Docker; environment variables are the standard secret injection mechanism. Runtime config calls encourage accidental logging; startup injection is safer.

**Options Considered:**
- **Environment variables at startup** — injected by systemd/Docker, typed struct at init, handlers receive immutable config
- **Runtime environment reads** — `os.Getenv()` in handler paths; risk of accidental logging, harder to audit
- **Secrets manager** (Vault, AWS Secrets Manager) — operational complexity for single VPS; out of scope
- **Config file** — requires encryption; file permissions are weaker than env var injection

**Decision:**
Load all configuration from environment variables at server startup. Populate a typed `Config` struct. Pass `Config` (or subsets) to handler functions via dependency injection. Handler code never calls `os.Getenv()`. This makes credential access points auditable and logging safer.

**Consequences:**
- ✅ Single audit point (main func, at startup)
- ✅ Handlers are pure functions of injected config; testable without env state
- ✅ Systemd/Docker can manage secret injection without app-level wrappers
- ⚠️ Config mutations at runtime are impossible (acceptable; almost never needed)
- ⚠️ Longer startup up-front (negligible; typed struct init is fast)

---

## ADR-008: Deployment — Single Binary + Postgres Only

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
"Self-hostable" means operators can run UCP Server on a modest Hetzner VPS without Docker, Kubernetes, load balancers, or managed services. The server is stateless (federation state is ephemeral); persistence needs only one database.

**Options Considered:**
- **Single binary + Postgres** (chosen) — operator runs `./ucp-server` and `systemctl start postgres`; minimal infra
- **Docker + Docker Compose** — easier for some operators; masks underlying complexity, adds Docker management burden
- **Kubernetes** — overkill for single-server deployment; appropriate only after multi-server federation is operational
- **Managed database + Lambda** — vendor lock-in contradicts "self-hostable" principle

**Decision:**
Release ucp-server as a single statically-linked binary compiled via `go build ./cmd/ucp-server`. Operators install Postgres 18+ via distro package manager (apt, dnf, etc.). Systemd units for ucp-server and postgres management included in docs. No Docker, no Kubernetes, no managed services required.

**Consequences:**
- ✅ Minimal operational footprint; runs on $5/month Hetzner VPS
- ✅ Operators control data residency (no cloud vendor)
- ✅ Binary deployment is fast and reproducible
- ⚠️ Operators must manage Postgres updates and backups (documented; not a limitation)
- ⚠️ Multi-server HA is deferred (addressed later via federation and async replication)

---

## ADR-009: MLS Implementation — Pure Go RFC 9420

**Date:** 2026-06-26
**Status:** Accepted

**Context:**
MLS (RFC 9420) is mandatory for UCP encryption. The question is how to implement it in Go while respecting the "no cgo" hard constraint (single binary, cross-compile friendly).

**Options Considered:**
- **mlspp bindings** — Mature C++ library via cgo. Pros: production-ready. Cons: violates no-cgo; breaks single-binary goal.
- **Pure-Go RFC 9420** — Implement in Go alone. Pros: no cgo, single binary. Cons: no mature pure-Go lib exists; significant dev effort or wait.
- **Keep mock, defer** — Leave AES-GCM placeholder. Pros: unblocks work. Cons: not spec-compliant.
- **Hybrid: core subset** — Implement only what UCP needs. Pros: pure Go. Cons: incomplete spec compliance.

**Decision:**
Build pure-Go RFC 9420 implementation directly in the server (`internal/crypto/mls` subpackage), fully isolated so it can be extracted to a standalone `go-mls` module later. Current AES-GCM mock is the placeholder until MLS is complete.

**Architecture for extraction:**
- `internal/crypto/mls/` — self-contained MLS package with no UCP dependencies
- RFC 9420 types, operations, key schedule, group management, serialization — all isolated
- `internal/crypto/manager.go` — wraps MLS package, integrates with UCP (groups, encryption, routing)
- When stable: `go-mls` becomes a separate public module; server imports it like any other lib

**Consequences:**
- ✅ Single binary maintained (no cgo, no cross-compile friction)
- ✅ Full RFC 9420 compliance built in-house
- ✅ Clean boundaries enable extraction without refactoring
- ✅ UCP owns a production-grade pure-Go MLS lib (community asset later)
- ⚠️ Significant implementation effort upfront (~3-6 months for production-ready)
- ⚠️ Until complete, mock placeholder in use (known limitation on go-live)
