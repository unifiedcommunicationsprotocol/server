# Decision Log

Architectural Decision Records (ADRs) — listed newest first.

Run `/decision` to add an entry.

---

## ADR-001: Runtime — Bun over Node.js/Deno

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Admin dashboard needs a lightweight, TypeScript-first HTTP server. Primary goal: single binary deployment on Hetzner VPS with zero external runtime dependencies. Candidates were Node.js, Deno, and Bun.

**Options Considered:**
- **Node.js** — Mature ecosystem, large package registry; but heavy runtime (100+ MB), requires external SQLite driver (better-sqlite3, pg), slower startup
- **Deno** — Modern security model, native TypeScript; but immature ecosystem, still evolving (npm compat issues), smaller community
- **Bun** — TypeScript-native, native SQLite bindings (bun:sqlite), 100× faster startup, single executable, smallest binary size

**Decision:**
Bun. Aligns with UCP Server's "self-hostable single binary" principle. Native SQLite support eliminates need for external drivers. TypeScript runs natively without compilation overhead. Single executable deployable to any Linux VPS.

**Consequences:**
- ✅ Binary ~50 MB (vs Node 100+ MB, Deno ~70 MB)
- ✅ Startup <100ms (vs Node 500ms+)
- ✅ SQLite bindings built-in (no external package)
- ✅ Native TS execution (`tsgo` as compiler)
- ⚠️ Smaller ecosystem (workaround: npm compatibility layer)
- ⚠️ Less battle-tested in production (mitigated: Bun 1.0+ is stable)

---

## ADR-002: HTTP Framework — Hono

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Dashboard backend needs type-safe routing, automatic OpenAPI schema generation, and validation. Hono is lightweight, works on Bun without adapters, and has first-class TypeScript support.

**Options Considered:**
- **Express** — Mature, vast middleware ecosystem; but not TypeScript-first, verbose boilerplate, routing not type-safe
- **Fastify** — Fast, TypeScript-friendly; but heavier than needed for dashboard, more plugins required
- **Hono** — Minimal, TypeScript-native routing, works on Bun/CF Workers/Deno, OpenAPI middleware built-in, automates schema generation

**Decision:**
Hono. Provides typed routing with Zod validation. OpenAPI schema auto-generated (no manual spec maintenance). Runs on Bun with zero configuration. Middleware-based architecture matches auth/validation patterns.

**Consequences:**
- ✅ Automatic route type inference
- ✅ OpenAPI schema at /api/openapi.json (for Swagger UI, client generation)
- ✅ Works seamlessly with Bun runtime
- ✅ Lightweight (no framework bloat)
- ⚠️ Smaller community than Express (but growing fast)
- ⚠️ Some middleware must be implemented (vs Express plugins)

---

## ADR-003: Validation — Zod

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Admin API accepts user input (login credentials, filters, mutation payloads). Need runtime validation with good error messages and TypeScript type inference.

**Options Considered:**
- **io-ts** — Pure functional, composable; but verbose, steep learning curve
- **Joi** — Feature-rich validation; but not TypeScript-first, slower at runtime
- **Zod** — Simple, TypeScript-first, fast runtime validation, excellent error messages, growing adoption

**Decision:**
Zod. Validates at request boundary (controller pattern). Infers TS types automatically via `z.infer<typeof schema>`. Error messages human-readable. Works well with Hono (middleware integration).

**Consequences:**
- ✅ Runtime + type-level safety
- ✅ Clean error messages (validation failures → 400 with details)
- ✅ Types derived from schemas (DRY principle)
- ⚠️ Smaller ecosystem than Joi, less enterprise adoption

---

## ADR-004: Database — SQLite (bun:sqlite) + Drizzle ORM

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Dashboard needs to store: users, sessions (Better Auth), audit logs, server config mirror. Options were Postgres, SQLite, or embedded solutions.

**Options Considered:**
- **Postgres** — Enterprise-grade, full ACID, powerful queries; but external service to operate, network latency
- **SQLite** — Embedded, zero ops, fast local queries; but single-writer (not suitable for high-concurrency)
- **LevelDB / RocksDB** — Key-value stores; but not relational

**Decision:**
SQLite (bun:sqlite) with Drizzle ORM. Dashboard is single-writer (admin actions not high-concurrency). SQLite bundled in Bun binary (zero external dependencies). Drizzle provides type-safe schema + migrations via code generation (no hand-written SQL). Bun's native SQLite bindings are faster than any npm driver.

**Consequences:**
- ✅ Zero external DB to operate
- ✅ Database file travels with binary (single artifact)
- ✅ Type-safe schema via Drizzle
- ✅ Migrations via `drizzle-kit push` (no SQL files)
- ⚠️ Single writer (not suitable for horizontal scaling)
- ⚠️ Operator must back up .db file manually (documented)

---

## ADR-005: Authentication — Better Auth

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Admin dashboard needs user authentication. Choices were JWT-based, session-based, or auth platforms (Auth0, Firebase).

**Options Considered:**
- **JWT (Jsonwebtoken)** — Stateless, scalable; but requires external issuer, token revocation costly
- **Sessions (Express-session pattern)** — Stateful, simple revocation; but requires session store
- **Better Auth** — Framework-agnostic session library, multi-device support, OAuth included, fits Hono

**Decision:**
Better Auth. Provides session management without external provider lock-in. Supports email/password + OAuth (Google, Microsoft). Multi-device sessions built-in. Easy integration with Hono middleware. Sessions stored in SQLite (no external cache needed).

**Consequences:**
- ✅ Zero external auth service (fully self-hosted)
- ✅ OAuth support (Google, Microsoft) without custom integration
- ✅ Multi-device session tracking
- ✅ Session signing key rotatable via `BETTER_AUTH_SECRET`
- ⚠️ Sessions stateful (requires DB; OK given SQLite choice)
- ⚠️ Smaller community than Auth0 (but growing)

---

## ADR-006: Frontend — React 19 + Tailwind CSS v4

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Dashboard UI needs modern component library + styling. React is industry standard for SPAs. Tailwind provides utility CSS without large CSS bundles.

**Options Considered:**
- **Vue 3 + UnoCSS** — Simpler learning curve; but smaller ecosystem, fewer component libraries
- **Svelte + Tailwind** — Smaller bundle; but newer, less stable, fewer resources
- **React 19 + Tailwind v4** — Largest ecosystem, most stable, component libraries mature

**Decision:**
React 19 + Tailwind CSS v4. React's component model matches dashboard requirements (reusable panels, charts, tables). Tailwind v4 is lighter than v3 (CSS engine rewrite, smaller bundles). Both have massive ecosystems (shadcn/ui, recharts, etc.).

**Consequences:**
- ✅ Massive component library ecosystem (shadcn/ui, Headless UI, etc.)
- ✅ Tailwind v4 smaller bundle than v3
- ✅ Dark mode built-in (tailwind dark: mode)
- ✅ React 19 Server Components (future use)
- ⚠️ Larger JS bundle than Vue/Svelte (mitigated: code splitting, gzip)

---

## ADR-007: Type Checking — TypeScript 7 Preview (tsgo)

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Bun runs TypeScript natively via `tsgo` command (TS 7 preview compiler). Need to type-check code without runtime cost.

**Options Considered:**
- **tsc (TypeScript compiler)** — Traditional, slower, separate compilation step
- **esbuild --check** — Fast but less complete
- **tsgo (Bun's TS7 preview)** — Native, fast, integrated into Bun

**Decision:**
TypeScript 7 preview with `tsgo` command. Bun runs scripts directly without compilation; `tsgo` for CI type-checking. Single command fits established CI workflow.

**Consequences:**
- ✅ No separate compilation step (dev faster)
- ✅ Type errors caught in CI via `bun run typecheck`
- ✅ Simpler toolchain (no tsc config overhead)
- ⚠️ TS 7 is preview (not final); trade-off for speed/simplicity

---

## ADR-008: Formatting & Linting — Biome

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Team needs consistent code formatting + linting. Historically ESLint + Prettier are separate tools, slow, and require config. Biome unifies both.

**Options Considered:**
- **ESLint + Prettier** — Mature, widely known; but two separate tools, slower, configuration hell
- **Biome** — Single tool, much faster, simpler config, replaces ESLint + Prettier

**Decision:**
Biome. Single `biome.json` replaces `.eslintrc` + `.prettierrc` + `prettier ignore`. `bun run lint` and `bun run format` handle everything. 10× faster than ESLint.

**Consequences:**
- ✅ Single tool (less config, fewer deps)
- ✅ 10× faster than ESLint
- ✅ Built-in formatting (no Prettier needed)
- ✅ Pre-commit hooks optional (CI enforces via script)
- ⚠️ Rules subset of ESLint (acceptable; Biome has essentials)
- ⚠️ Smaller community (but rapid adoption)

---

## ADR-009: Dashboard Deployment — Embedded in Go Binary

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Admin dashboard is built separately (Bun, React, TypeScript) but must deploy with UCP Server. Options: separate binary + process, Docker compose, or embed in Go binary.

**Options Considered:**
- **Separate Bun + Go processes** — Independently scalable; but two binaries, coordination overhead, simpler but ops heavier
- **Docker Compose** — Both in containers; but adds container runtime, violates "single binary" principle
- **Embed React build in Go** — Single binary, simpler deployment, no process coordination

**Decision:**
Embed React build as static assets in Go binary using Go's `embed` package. Build process: React → `www/dist/`, then Go `embed` package includes files, `go build` → single executable.

**Deployment flow:**
1. `cd www && bun run build` — outputs to `dist/`
2. Go server embeds `dist/`
3. `go build ./cmd/ucp-server` → single binary serves both API + dashboard

**Consequences:**
- ✅ Single deployable artifact (simpler ops)
- ✅ No process coordination (dashboard served by Go)
- ✅ No external web server needed
- ⚠️ React rebuild required before server deploy
- ⚠️ Asset updates = server binary rebuild (acceptable for admin UI)

---

## ADR-010: Testing Strategy — No Database Mocking

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Tests for API handlers that query SQLite. Question: mock database or use real SQLite instance?

**Options Considered:**
- **Mock database** — Fast tests, no external deps; but mocks diverge from reality, false negatives
- **Real SQLite test instance** — Slower but reliable, catches real bugs

**Decision:**
Real SQLite test instance per test suite. Create empty DB, run migrations, run test, truncate tables (via `test-setup.ts` preload). Same database bindings as production (no mock divergence).

**Consequences:**
- ✅ Tests catch real database bugs
- ✅ No mock drift (code/tests stay in sync)
- ✅ Easier debugging (inspect actual DB state)
- ⚠️ Tests slower (DB operations are I/O)
- ⚠️ Requires test DB infrastructure (SQLite file, not an issue)

---

## ADR-011: Packaging — All Commands in package.json Scripts

**Date:** 2026-06-28  
**Status:** Accepted

**Context:**
Team needs consistent way to run dev, build, test, lint, deploy. Options: Makefile, custom scripts, npm package.json scripts, or shell aliases.

**Options Considered:**
- **Makefile** — Language-agnostic, widely known; but not portable to Windows
- **Custom bash scripts** — Flexible; but harder to discover
- **package.json scripts** — Standard in Node ecosystem, self-documenting via `bun run`

**Decision:**
All commands in `package.json` scripts. `bun run dev`, `bun run build`, `bun run test`, `bun run lint`, `bun run typecheck`, `bun run db:push`. Self-documenting and standard for Bun/JS projects.

**Consequences:**
- ✅ Self-documenting (`bun run` lists all commands)
- ✅ Standard in Bun/Node ecosystem
- ✅ Cross-platform (Windows + Unix)
- ⚠️ Scripts limited to shell commands (complex logic → separate script files)

---

*Last updated: 2026-06-28*
