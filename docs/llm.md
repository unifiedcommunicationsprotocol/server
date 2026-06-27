# UCP Server — AI Navigation Guide

> This file is the project constitution. It loads in every Claude Code session.
> Procedures belong in `.claude/skills/`, not here.

---

## Project Overview

**What:** Reference implementation of UCP — the Unified Communications Protocol, an open protocol unifying email, messaging, calendar, contacts, and notes over a single push-first, E2E encrypted connection.

**Why:** IMAP (1990s), SMTP, CalDAV, and CardDAV are fragmented, pull-based, insecure by default, and agent-hostile. UCP replaces this legacy stack with a modern foundation: push delivery, mandatory MLS encryption, structured JSON messages, portable DNS-anchored identity, and AI-native metadata.

**Status:** UCP/1.0 specification is draft-complete and ready for implementation. Reference server bootstrap phase. One known spec production blocker: IANA registration pending for `UCPWelcomeExtension` type — see `spec/encryption.md` for details.

**Repo:** github.com/unifiedcommunicationsprotocol/server

---

## Tech Stack

| Layer      | Technology                          | Notes                                     |
|------------|-------------------------------------|-------------------------------------------|
| Language   | Go 1.23+                            | Single compiled binary; no cgo dependency |
| HTTP       | `net/http` (stdlib)                 | No framework; direct handler composition  |
| Database   | Postgres 18+                        | TBD: sqlc, pgx, or raw stdlib sql package |
| Auth       | TBD                                 | Session tokens, Ed25519 signing; no OAuth framework yet |
| Infra      | Hetzner VPS                         | Pulumi provisioning; Caddy TLS; no Docker |
| Testing    | `testing` (stdlib)                  | Table-driven tests; no external frameworks |
| Deploy     | Single binary                       | `go build ./cmd/ucp-server` → production binary |

---

## Repository Structure

```
server/
├── CLAUDE.md                # → symlink to docs/llm.md (AI navigation guide)
├── AGENTS.md                # → symlink to docs/llm.md (same as CLAUDE.md)
├── cmd/
│   └── ucp-server/          # Binary entry point
├── internal/
│   ├── transport/           # WebSocket + WebTransport (HTTP/3) negotiation
│   ├── identity/            # Ed25519 keypairs, DNS resolution, signing key rotation
│   ├── crypto/              # MLS (RFC 9420) group management, encryption
│   ├── router/              # Federation, server-to-server routing
│   ├── store/               # Message persistence layer (Postgres)
│   ├── bridge/              # IMAP/SMTP bridge, HTML↔blocks conversion
│   ├── ai/                  # AI metadata surface (client-generated, opt-in server processing)
│   ├── api/                 # Client-facing HTTP endpoints, well-known routes
│   ├── auth/                # Session tokens, challenge-response, key management
│   └── models/              # UCP types (Message, Envelope, Identity, etc.)
├── spec/                    # UCP protocol specification (read-only)
├── docs/
│   ├── llm.md               # This file — the authoritative source (loaded in every session)
│   ├── architecture.md      # System design, data flows, component structure
│   ├── decisions.md         # Architecture decision records (ADRs)
│   └── IMPLEMENTATION.md    # Package status, HTTP endpoints, database schema
├── go.mod / go.sum          # Dependencies
└── Makefile                 # Build, test, lint targets
```

**Note:** `CLAUDE.md` and `AGENTS.md` are symlinks to `docs/llm.md`. Edit `docs/llm.md` directly; the symlinks ensure the AI navigation guide loads in every session.

---

## Coding Conventions

- Go 1.23+ with zero external stdlib dependencies where feasible
- `go fmt` for formatting (standard library style)
- `goimports` for import organization
- Table-driven tests (stdlib `testing` package only)
- Package structure: `internal/` for non-exported, `cmd/` for binaries, no `pkg/`
- Error handling: explicit `if err != nil` with wrapped context via `fmt.Errorf`
- Naming: CamelCase for exported types/funcs, lowercase for package names (no dashes)
- Configuration: environment variables at startup, no runtime `os.Getenv()` in handler paths
- No cgo — pure Go only (enables single binary, cross-compilation)
- No ORM without explicit approval — `sqlc`, `pgx`, or stdlib `database/sql` only

---

## Architecture Principles

**From the UCP spec:**

1. **Push-first** — persistent WebSocket/WebTransport connections; clients never poll
2. **Structured by default** — typed JSON messages, not MIME blobs; blocks-first body model
3. **E2E encrypted by protocol** — MLS (RFC 9420) mandatory, not optional; server is zero-knowledge by default
4. **Portable identity** — Ed25519 keypairs anchored in DNS; users own their identity independent of server
5. **Unified async + real-time** — email, messages, calendar, notes over a single connection and auth flow
6. **AI-native** — summaries, embeddings, categories are first-class fields; zero-knowledge defaults with opt-in server processing
7. **Federated** — server-to-server delivery; no central registry or single point of control
8. **Self-hostable** — reference implementation ships as a single binary with zero external runtime dependencies

**Server-specific patterns:**

- **Zero-knowledge relay** — server stores and forwards encrypted envelopes without decrypting unless user grants key share
- **Identity-centric routing** — all federation and delivery keyed to UCP addresses, not accounts
- **IMAP/SMTP bridge as first-class** — not an afterthought; bridge attestation is the adoption path for legacy email users
- **Layered security** — MLS forward secrecy + signing key rotation + identity separation + revocation key offline

---

## Hard Constraints

- **CRITICAL: Never run git commands from the project root's parent directories.** Always `cd` into the project directory first before any git operation. Git commands run from parent paths can pollute state or affect unrelated repos.
- Never modify database migration files by hand — use schema versioning tool only
- Never add external dependencies without explicit approval (keep binary lean)
- Never commit secrets, tokens, credentials, or private keys
- No cgo — pure Go only (enables single binary, cross-compilation, reproducible builds)
- No ORM magic — explicit SQL queries or code generation (sqlc) only
- MLS implementation must follow RFC 9420 exactly; no algorithm substitutions or custom variants
- Never skip signature verification for envelope or message payloads
- Server must treat all user data as encrypted in transit; no plaintext logging of message content

---

## Key Contacts & Decisions

- Decisions log: `docs/decisions.md`
- Architecture doc: `docs/architecture.md`
- UCP Spec (authoritative): https://github.com/unifiedcommunicationsprotocol/spec

---

## Skills Available

| Skill          | When to invoke                          |
|----------------|-----------------------------------------|
| `/align`       | Before any non-trivial work             |
| `/spec-create` | Starting a formal feature spec          |
| `/spec-review` | Before implementing a spec              |
| `/tdd`         | Building or fixing with tests as driver |
| `/diagnose`    | Stuck on a bug or unexpected behaviour  |
| `/zoom-out`    | Losing the big picture; pre-refactor    |
| `/decision`    | Logging an architectural decision       |
| `/commit`      | Creating a well-formed commit           |

---

*Last updated: 2026-06-26*
