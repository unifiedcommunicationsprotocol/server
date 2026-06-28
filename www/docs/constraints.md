# Constraints

Hard constraints for this codebase. Must be respected unconditionally.

---

## Never Do

### Architecture
- **Never write auth logic in route handlers** — middleware handles all session verification and role checks
- **Never add Docker or managed cloud services** — keep binary self-hostable (Hetzner VPS + Caddy only)
- **Never use external auth providers** (Auth0, Firebase, etc.) — Better Auth on SQLite only
- **Never run production without TLS** — Caddy reverse proxy mandatory (enforces HTTPS)

### Database
- **Never modify** Drizzle-generated migration files, snapshots, or journal — only `drizzle-kit` owns these
- **Never modify** Better Auth tables manually — let Better Auth schema generator create them
- **Never use** `bun db:generate` + manual `migrate` pattern — use `bun run db:push` only
- **Never use** the `pg` package or any external SQLite driver — only `bun:sqlite` (Bun native)
- **Never use** ORMs beyond Drizzle (no Prisma, no TypeORM, no SQLAlchemy bindings)

### Code
- **Never use** Node.js built-ins or polyfills — only Bun-native APIs (`bun:sqlite`, `bun:env`, `bun:http`)
- **Never write** package versions by hand in `package.json` — only via `bun add <pkg>@latest` CLI
- **Never use** `process.env` directly in packages — use typed build constants or config injection
- **Never commit** secrets, tokens, credentials, `.env` files, or private keys — document variable names only in `docs/deployment.md`
- **Never modify** configuration files (tsconfig.json, biome.json, bunfig.toml) — only install/remove packages via CLI

### Frontend
- **Never serve** unminified or uncompressed assets in production — `bun run build` must produce gzipped bundles
- **Never log** message content, user credentials, or session tokens to browser console
- **Never use** localStorage for session tokens — HTTP-only cookies only (Better Auth default)
- **Never hardcode** API URLs — use environment variables or config injection

### Deployment
- **Never disable TLS** even for testing — always use HTTPS (enforce via Caddy)
- **Never share** `BETTER_AUTH_SECRET` across environments — regenerate for each deployment
- **Never run** multiple instances without shared SQLite (not suitable for horizontal scaling)

---

## Always Do

### Database Management
- **Always run** `drizzle-kit` to generate migrations before pushing to prod
- **Always test** migrations locally first (run `bun test` with clean DB)
- **Always keep** `drizzle.config.ts` in sync with `src/db/schema.ts` (no manual editing)

### Code Quality
- **Always run** `bun run typecheck` before committing — no type errors in CI
- **Always run** `bun run lint` before committing — Biome must pass
- **Always run** `bun test` before committing — all tests passing
- **Always add** tests for new handlers, schemas, and middleware
- **Always update** `docs/context.md` when introducing new domain concepts

### Documentation
- **Always keep** `docs/llm.md` (CLAUDE.md) factual and concise — procedures go in `.claude/skills/`
- **Always update** `docs/decisions.md` when making architectural choices (use `/decision` skill)
- **Always document** environment variables in `docs/deployment.md` (never in code)

### Secrets & Security
- **Always rotate** `BETTER_AUTH_SECRET` when deploying to new environment
- **Always use** Ed25519 SSH keys (not RSA) for VPS access
- **Always enable** systemd hardening (PrivateTmp, NoNewPrivileges, ReadOnlyPaths)
- **Always verify** session signature before using session data (Better Auth does this)

### Deployment
- **Always build locally** and test before deploying (`bun run build && bun test`)
- **Always backup** SQLite DB before major schema changes
- **Always keep** previous binary version for quick rollback

---

## Critical Path (Must Never Break)

| Path | Responsibility | Test Coverage |
|------|---|---|
| `POST /api/auth/signin` | User login, session creation | 100% |
| Session verification middleware | Every API request validates token | 95%+ |
| `GET /api/stats` | Server health, message count | 80%+ |
| Database migrations | Schema changes applied correctly | 95%+ (integration tests) |
| Zod validation | Invalid input rejected cleanly | 90%+ |

---

## External API / Rate Limit Notes

| Service | Limit | Notes |
|---------|-------|-------|
| Hetzner API | 3600 req/hour | Used only during provisioning (Pulumi) |
| Local SQLite | No external limit | Bound by disk I/O; no network latency |
| Better Auth OAuth | Google/Microsoft limits | Rare, during login only; no rate limit within our app |

---

## Regulatory / Legal

### Data Protection
- **GDPR:** No user data collected beyond login (email + role). Dashboard is internal admin tool, not user-facing.
- **Data Residency:** SQLite on Hetzner VPS in chosen region (EU: nbg1/hel1, UK: lhr via Vultr).
- **Data Retention:** Dashboard audit logs retained for 90 days (operator-configurable); UCP Server message retention is separate.

### Compliance
- **No third-party analytics** — no Sentry, Mixpanel, etc. in default config (optional for errors only)
- **No external logging** — all logs local to filesystem (operator can ship to centralized logging if desired)
- **Session security:** Signed sessions, no plaintext storage, automatic expiry at 24h

### Acceptable Use
- Dashboard is for authorized admins only — never expose publicly
- All admin actions logged immutably — audit trail for compliance
- Message encryption end-to-end (MLS) — server is zero-knowledge by default

---

*Last updated: 2026-06-28*
