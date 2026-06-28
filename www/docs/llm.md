# UCP Admin Dashboard — AI Navigation Guide

> This file is the project constitution. It loads in every Claude Code session via symlinks at `CLAUDE.md` and `AGENTS.md`.
> Procedures belong in `.claude/skills/`, not here.

---

## Project Overview

**What:** Full-stack admin dashboard for the Unified Communications Protocol (UCP) server, providing real-time visibility and control over message routing, identity management, and federation status.

**Why:** UCP Server operators need a web UI to monitor health, manage users, view message logs, and control federation. Built as embedded SPA (React 19 + Tailwind) served alongside the API.

**Status:** ✅ COMPLETE & PRODUCTION READY (v0.1.0)
- All 6 tabs implemented (Overview, APIExplorer, Identity, Sessions, Federation, Bridge)
- React UI matches design 1:1 with Tailwind v4
- API client library with 17+ functions
- Ready to embed in Go server or run standalone

**Repo:** github.com/unifiedcommunicationsprotocol/server/tree/main/www

---

## Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Language** | TypeScript 7 (preview) | `@typescript/native-preview`; tsgo for type-checking |
| **Runtime** | Bun 1.0+ | Single binary, native SQLite, no Node.js |
| **HTTP Framework** | Hono + Zod | Type-safe routing, OpenAPI-first, automatic schema generation |
| **Frontend** | React 19 + Tailwind CSS v4 | Modern UI, utility CSS, dark mode built-in |
| **Database** | SQLite (bun:sqlite) + Drizzle ORM | Embedded, zero external deps, type-safe schema |
| **Authentication** | Better Auth | Session-based (not JWT), multi-device, OAuth support |
| **Formatting & Linting** | Biome | Single tool, replaces ESLint + Prettier |
| **Infrastructure** | Hetzner VPS + Caddy + Systemd | No Docker, self-hosted, single binary deployment |
| **Testing** | Bun test runner | Built-in, no Jest, real SQLite (no mocking) |

---

## Repository Structure

```
www/
├── src/
│   ├── index.ts                      # Hono app entry point
│   ├── api/
│   │   ├── routes.ts                 # Route definitions
│   │   ├── handlers/                 # Per-domain handlers (auth, stats, messages, etc.)
│   │   └── schemas.ts                # Zod validation schemas
│   ├── auth/
│   │   ├── config.ts                 # Better Auth setup
│   │   └── middleware.ts             # Session verification
│   ├── db/
│   │   ├── schema.ts                 # Drizzle tables
│   │   ├── migrate.ts                # Migration runner
│   │   └── test-setup.ts             # Test database init
│   ├── components/                   # React components (dashboard UI)
│   ├── hooks/                        # React hooks (useAuth, useAPI, etc.)
│   ├── pages/                        # Route-based pages
│   ├── styles/                       # Tailwind CSS
│   ├── types/                        # TypeScript types
│   └── utils/                        # Helper functions
├── public/                           # Static assets
├── index.html                        # SPA root
├── build.ts                          # Custom build script (esbuild + Tailwind)
├── bunfig.toml                       # Bun config (SQLite, test preload)
├── biome.json                        # Biome (lint + format)
├── tsconfig.json                     # TypeScript 7
├── package.json                      # Dependencies & scripts
├── docs/
│   ├── llm.md                        # This file (AI navigation)
│   ├── architecture.md               # System design & components
│   ├── decisions.md                  # Architectural decision records
│   ├── deployment.md                 # VPS deployment + ops
│   ├── testing.md                    # Test strategy & examples
│   ├── spec.md                       # Feature spec template
│   ├── context.md                    # Domain language glossary
│   └── constraints.md                # Hard rules & requirements
├── CLAUDE.md                         # Symlink to docs/llm.md
└── AGENTS.md                         # Symlink to docs/llm.md
```

---

## Coding Conventions

- **TypeScript 7 native** — `@typescript/native-preview` as the TS compiler; `tsgo` for type-checking in CI
- **Biome for all formatting and linting** — no ESLint, no Prettier (single `biome.json`)
- **ESM imports everywhere** — no CommonJS (`type: "module"` in package.json)
- **SQLite via bun:sqlite** — `drizzle-orm/bun-sql` for ORM; no `pg`, `better-sqlite3`, or external drivers
- **Dependency management** — `bun add <pkg>@latest` CLI only; never hand-write versions in package.json
- **Database schema** — Drizzle table definitions only; `bun run db:push` applies changes (never hand-edit migrations)
- **Naming conventions** — kebab-case for filenames (e.g., `user-service.ts`), PascalCase for types/components (e.g., `UserCard`)
- **Configuration** — typed constants injected at startup, not `process.env` in packages
- **Scripts** — all commands via `package.json` scripts (accessible via `bun run`); no Makefile

---

## Architecture Principles

1. **Single binary deployment** — React embedded as static assets in Go server; also compiles standalone Bun binary
2. **Type-safe end-to-end** — TypeScript 7 + Zod validation + OpenAPI schema generation
3. **Push-first real-time** — WebSocket connections for live metrics (future); polling fallback (current)
4. **Zero external dependencies** — SQLite embedded, no managed databases, no external auth provider
5. **Admin-centric UI** — Dashboard is for operators, not end users; focus on debugging and control
6. **Audit trail** — all admin actions logged to SQLite (immutable append-only)
7. **Self-hostable** — single VPS (Hetzner), single process, no Kubernetes/Docker/cloud services

---

## Hard Constraints

### Database
- **Never modify** Drizzle-generated migration files, snapshots, or journal — Drizzle owns these
- **Never modify** Better Auth tables manually — let Better Auth generate schema
- **Never use** `bun db:generate` + `migrate` pattern — use `bun run db:push` only
- **Never use** the `pg` package — use `drizzle-orm/bun-sql` only

### Code
- **Never use** Node.js built-ins or polyfills — Bun-native APIs only (e.g., `bun:sqlite`, `bun:env`)
- **Never write** package versions by hand in `package.json` — only via `bun add <pkg>@latest` CLI
- **Never use** `process.env` directly in packages — use typed build constants or config injection instead
- **Never commit** secrets, tokens, credentials, or `.env` files — document variable names only
- **Never modify** configs (tsconfig.json, biome.json, bunfig.toml) — only install/remove packages

### Architecture
- **Never write** auth logic in route handlers — middleware handles all session verification
- **Never add** Docker, managed cloud services, or external platforms (e.g., Auth0, Vercel)
- **Never use** ORM magic or hidden queries — Drizzle queries must be explicit and visible

---

## Key Contacts & Decisions

- **Decisions log:** `docs/decisions.md` (ADRs for tech choices)
- **Architecture doc:** `docs/architecture.md` (system design, data flows, package responsibilities)
- **Open specs:** `.claude/specs/` (feature specifications, use `/spec-create` to generate)
- **Domain language:** `docs/context.md` (shared vocabulary for AI + team)
- **Constraints:** `docs/constraints.md` (hard rules and non-negotiables)

---

## Skills Available

| Skill | When to invoke |
|-------|---|
| `/spec-create` | Starting a formal feature spec; generates template in `.claude/specs/` |
| `/spec-review` | Before implementing a spec; flags gaps, risks, ambiguities |
| `/tdd` | Building/fixing with tests as driver; red-green-refactor loop |
| `/diagnose` | Stuck on bug or unexpected behavior; disciplined diagnosis loop |
| `/zoom-out` | Losing big picture; need to explain code/decision in system context |
| `/decision` | Logging an architectural decision record (ADR) to `docs/decisions.md` |
| `/commit` | Creating a well-formed commit with proper message format |

---

## Environment Variables

At runtime, the app reads from `.env.local` (dev) or `.env.production` (deployed):

| Variable | Required | Description |
|----------|----------|-------------|
| `BETTER_AUTH_SECRET` | Yes | Session signing key (32+ bytes, base64) |
| `DATABASE_URL` | Yes | SQLite connection (`sqlite:///path/to/db.db` or `sqlite://db/test.db`) |
| `NODE_ENV` | No | `development` or `production` (default: `development`) |
| `PORT` | No | HTTP listen port (default: `5173` dev, `3000` prod) |

Never commit `.env` files. Document variable names in `docs/deployment.md` only.

---

## Common Commands

```bash
# Development
bun run dev              # Start dev server (hot reload, port 5173)

# Type checking
bun run typecheck        # Type-check via tsgo (runs in CI)

# Formatting & Linting
bun run lint             # Check for issues
bun run format           # Auto-fix formatting
bun run format:check     # Check without modifying

# Testing
bun test                 # Run all tests
bun test --watch        # Watch mode
bun test --coverage     # Coverage report (80% threshold)

# Database
bun run db:push         # Apply schema changes
bun run db:studio       # Open Drizzle Studio (local UI)

# Build & Deploy
bun run build           # Build React + Tailwind
bun build --compile --target=bun src/index.ts --outfile=ucp-dashboard  # Compile Bun binary
```

---

## Related Projects

- **UCP Server (Go)** — Main API server; embeds this dashboard as static assets
- **UCP Spec** — https://github.com/unifiedcommunicationsprotocol/spec (authoritative protocol definition)

---

*Last updated: 2026-06-28*
