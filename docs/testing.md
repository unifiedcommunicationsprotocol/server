# Testing

## Strategy

[TODO: overall approach — unit, integration, E2E and where each applies]

Core principle: no mocking the database. Tests hit a real Postgres test instance (see `test-setup.ts`).

## Tools

| Layer | Tool | Notes |
|-------|------|-------|
| Unit / Integration | Bun test runner | Built-in; no Jest needed |
| Coverage | Bun coverage | 80% threshold in `bunfig.toml` |
| [TODO] | [TODO] | [TODO] |

## test-setup.ts

Loaded via `bunfig.toml` preload. Runs migrations against the test DB and truncates between tests.

**Critical:** `DATABASE_URL` must contain `"test"` — `test-setup.ts` throws otherwise. Separate `postgres-test` service in `compose.yml` on port 5433.

## Running Tests

```sh
# all tests
bun test

# watch mode
bun test --watch

# single package
bun test src/[package]

# apply migrations to test DB first (required on first run)
bunx drizzle-kit migrate
```

## Conventions

- Test files co-located with source: `foo.ts` → `foo.test.ts`
- Use domain language from `docs/context.md` in test descriptions
- No mocking the database — tests hit the real test Postgres instance
- [TODO: mocking policy for external services]

## What Must Always Be Tested

- Auth middleware — session derivation and rejection
- [TODO: non-negotiable coverage requirements]

---

*Last updated: [DATE]*
