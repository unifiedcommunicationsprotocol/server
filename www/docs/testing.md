# Testing

## Strategy

### Core Principle: No Database Mocking

Tests hit a real SQLite test database. Every test gets a clean database state, runs migrations, executes test code, then truncates tables for the next test. This approach catches real bugs that mocks would hide (schema mismatches, query errors, etc.).

### Layer Coverage

| Layer | Approach | Tool | When to use |
|-------|----------|------|-------------|
| **Unit** | Functions in isolation; math, formatting, validation | Bun test runner | Logic without I/O (Zod schemas, utilities) |
| **Integration** | API handlers + real SQLite (no mock) | Bun test runner + test DB | Auth handlers, database queries, migrations |
| **E2E** | Full HTTP requests, real database, real auth | Playwright or curl + test server | Critical user journeys (login → dashboard view) |

### What Must Always Be Tested

1. **Auth middleware** — session verification, role checks, token expiry
2. **API handlers** — request validation (Zod), response shape, error cases
3. **Database operations** — create/read/update/delete, migrations
4. **Business logic** — stats calculations, filtering, sorting

### What's Optional (Lower Priority)

- Frontend component rendering (can use snapshots, but not mandatory)
- External API calls (mock these to avoid real calls in CI)
- Edge cases on non-critical paths

## Tools

### Unit & Integration Testing

**Bun's built-in test runner** — no external test framework needed.

```bash
bun test                    # Run all tests
bun test --watch           # Watch mode (rerun on file change)
bun test src/auth/*.test.ts # Run specific files
bun test --timeout=5000    # Increase timeout (default 5s)
```

Coverage enabled via `bunfig.toml` (80% threshold).

```bash
bun test --coverage        # Generate coverage report
```

### E2E Testing (Optional)

For critical flows (login → view dashboard), use Playwright:

```bash
bunx @playwright/test init
bun test e2e/               # Runs .spec.ts files
```

## Test Database Setup

### Configuration

`bunfig.toml` specifies test preload:

```toml
[test]
preload = ["./src/db/test-setup.ts"]
```

### test-setup.ts

Runs before ALL tests. Initializes test database, runs migrations, sets up fixtures.

```typescript
// src/db/test-setup.ts
import { migrate } from "./migrate";
import { db } from "./index"; // SQLite connection

// Safety check: DATABASE_URL must contain "test"
if (!process.env.DATABASE_URL?.includes("test")) {
  throw new Error("DATABASE_URL must include 'test' (e.g., sqlite://db/test.db)");
}

// Initialize fresh database
const testDb = new Database(process.env.DATABASE_URL);
migrate(testDb);

export { testDb };
```

### Running Tests

First run — must apply migrations:

```bash
# Create test database and run migrations
bunx drizzle-kit migrate --database "sqlite://db/test.db"

# Now run tests
bun test
```

Subsequent runs:
```bash
bun test  # Migrations already applied
```

## Test File Structure

Co-locate test files with source:

```
src/
├── auth/
│   ├── middleware.ts
│   └── middleware.test.ts    # Same name, .test.ts suffix
├── api/
│   ├── handlers/
│   │   ├── stats.ts
│   │   └── stats.test.ts
│   └── routes.ts
└── db/
    ├── schema.ts
    └── schema.test.ts
```

## Example Tests

### Unit Test (No Database)

```typescript
// src/utils/format.test.ts
import { describe, it, expect } from "bun:test";
import { formatBytes, formatDuration } from "./format";

describe("format utilities", () => {
  it("should format bytes to human-readable", () => {
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1024 * 1024)).toBe("1 MB");
  });

  it("should format duration to hh:mm:ss", () => {
    expect(formatDuration(3661)).toBe("1:01:01");
  });
});
```

### Integration Test (With Database)

```typescript
// src/auth/middleware.test.ts
import { describe, it, expect, beforeEach } from "bun:test";
import { db } from "../db";
import { sessions, users } from "../db/schema";
import { verifySession } from "./middleware";

describe("Auth Middleware", () => {
  beforeEach(async () => {
    // Clean tables between tests
    await db.delete(sessions);
    await db.delete(users);
  });

  it("should verify valid session", async () => {
    // Insert test user and session
    await db.insert(users).values({
      id: "user-1",
      email: "alice@example.com",
      role: "admin",
    });

    const sessionToken = "valid-token-123";
    await db.insert(sessions).values({
      id: sessionToken,
      userId: "user-1",
      expiresAt: new Date(Date.now() + 86400000), // 24h from now
    });

    // Verify session
    const session = await verifySession(sessionToken);
    expect(session).toEqual({
      userId: "user-1",
      email: "alice@example.com",
      role: "admin",
    });
  });

  it("should reject expired session", async () => {
    // Insert expired session
    const expiredToken = "expired-token-123";
    await db.insert(sessions).values({
      id: expiredToken,
      userId: "user-1",
      expiresAt: new Date(Date.now() - 1000), // 1s ago
    });

    // Verify fails
    const session = await verifySession(expiredToken);
    expect(session).toBeNull();
  });
});
```

### API Handler Test

```typescript
// src/api/handlers/auth.test.ts
import { describe, it, expect } from "bun:test";
import { Hono } from "hono";
import { signinHandler } from "./auth";

describe("POST /api/auth/signin", () => {
  it("should return 400 on missing email", async () => {
    const app = new Hono();
    app.post("/api/auth/signin", signinHandler);

    const res = await app.request("POST /api/auth/signin", {
      method: "POST",
      body: JSON.stringify({ password: "test123" }),
    });

    expect(res.status).toBe(400);
    const json = await res.json();
    expect(json.error).toContain("email is required");
  });

  it("should return 400 on invalid email", async () => {
    const res = await app.request("POST /api/auth/signin", {
      method: "POST",
      body: JSON.stringify({
        email: "not-an-email",
        password: "test123",
      }),
    });

    expect(res.status).toBe(400);
  });

  it("should return 401 on wrong password", async () => {
    // Set up test user in database
    // ...

    const res = await app.request("POST /api/auth/signin", {
      method: "POST",
      body: JSON.stringify({
        email: "alice@example.com",
        password: "wrong-password",
      }),
    });

    expect(res.status).toBe(401);
  });
});
```

## Coverage Requirements

| Package | Minimum | Target |
|---------|---------|--------|
| `api/handlers` | 80% | 90% |
| `auth` | 85% | 95% |
| `db` | 70% | 85% |
| `utils` | 75% | 90% |

Run coverage check:

```bash
bun test --coverage
```

## CI/CD

### Pre-Commit (Local)

Before committing, run locally:

```bash
bun run typecheck
bun run lint
bun test
```

Or use Git hook (optional `.husky/` setup):

```bash
# Install husky
bunx husky install

# Add pre-commit hook
echo "bun run typecheck && bun run lint && bun test" > .husky/pre-commit
chmod +x .husky/pre-commit
```

### GitHub Actions

`.github/workflows/test.yml`:

```yaml
name: Test

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      sqlite:
        image: nouchka/sqlite3:latest  # Or just use file-based

    steps:
      - uses: actions/checkout@v3
      - uses: oven-sh/setup-bun@v1

      - run: bun install
      - run: bun run typecheck
      - run: bun run lint
      - run: bun test --coverage
      - run: |
          echo "Coverage report:"
          cat coverage.txt
```

## Performance Testing

### Load Testing (Optional, for optimization)

Use `autocannon` or similar:

```bash
bunx autocannon http://localhost:3000/api/stats --connections 100 --duration 10
```

Target: <50ms P99 latency on `/api/stats` endpoint.

## Debugging Tests

### Print Debug Info

```typescript
it("should do something", async () => {
  const result = await fetchStats();
  console.log("Result:", result); // Shows in test output
  expect(result).toBeDefined();
});
```

Run with output:

```bash
bun test --verbose
```

### Single Test

```bash
bun test src/auth/middleware.test.ts --only "should verify valid session"
```

### Connect to Database During Test

```typescript
it("should have correct session in db", async () => {
  const session = await db.query.sessions.findFirst({
    where: eq(sessions.id, "test-token"),
  });
  console.log("Session:", session);
  expect(session).toBeDefined();
});
```

## Mocking External Services

While we avoid mocking the database, external APIs should be mocked:

```typescript
// src/api/handlers/stats.test.ts
import { describe, it, expect, mock } from "bun:test";
import { getStats } from "./stats";

describe("getStats", () => {
  it("should fetch metrics from UCP Server", async () => {
    // Mock HTTP call to UCP Server
    const mockFetch = mock((url) => {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          messageCount: 1234,
          activeConnections: 42,
        }),
      });
    });

    global.fetch = mockFetch;

    const stats = await getStats();
    expect(stats.messageCount).toBe(1234);
    expect(mockFetch).toHaveBeenCalledWith("http://localhost:5150/api/stats");
  });
});
```

## Common Patterns

### Testing with Transaction Rollback (Optional)

For true database isolation without truncation:

```typescript
beforeEach(async () => {
  await db.transaction(async (tx) => {
    // Run test in transaction
    await runTest(tx);
    // Automatically rolls back after test
    throw new Error("ROLLBACK"); // Force rollback
  });
});
```

Note: Bun test runner doesn't support nested transactions well; truncation is simpler.

### Snapshot Testing (Optional)

For API responses:

```typescript
it("should return correct schema", async () => {
  const res = await app.request("GET /api/stats");
  const json = await res.json();
  expect(json).toMatchSnapshot();
});
```

Run with:

```bash
bun test --update-snapshots  # Update snapshots if correct
```

## Troubleshooting

### "DATABASE_URL must include 'test'"

```bash
# Export test database URL
export DATABASE_URL="sqlite://db/test.db"
bun test
```

Or set in `.env.test`:

```
DATABASE_URL=sqlite://db/test.db
```

### "Test timeout exceeded"

Increase timeout:

```bash
bun test --timeout=10000  # 10 seconds
```

Or in individual test:

```typescript
it("slow test", async () => {
  // ...
}, { timeout: 10000 });
```

### "Module not found"

Ensure `bunfig.toml` includes test preload:

```toml
[test]
preload = ["./src/db/test-setup.ts"]
root = "src"
```

### "Failed to run migrations"

Ensure Drizzle CLI is available:

```bash
bun add -d drizzle-kit
```

Run migrations manually:

```bash
bunx drizzle-kit migrate --database "sqlite://db/test.db"
bun test
```

---

*Last updated: 2026-06-28*
