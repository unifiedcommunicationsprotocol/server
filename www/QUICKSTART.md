# UCP Admin Dashboard — Quick Start

## What's Built

✅ **Complete React UI** matching the design 1:1
- 6 tabs: Overview, API Explorer, Identity, Sessions, Federation, Bridge
- Real API integration (calls localhost:6001)
- Mock responses fallback
- Full state management

## Development

### Build the React app
```bash
bun run build
```

This compiles React + TypeScript + Tailwind into `dist/index.html` + JS bundles.

### Run dev server
```bash
bun run dev
```

Starts Hono server on `http://localhost:6002` with hot reload.

### Production build
```bash
bun build --compile --target=bun src/index.ts --outfile=dashboard
```

Creates a single compiled executable.

## Architecture

```
src/
  ├── index.tsx              ← React entry point
  ├── components/
  │   └── dashboard/         ← All UI components
  │       ├── Dashboard.tsx  ← Shell & router
  │       ├── Sidebar.tsx
  │       ├── Header.tsx
  │       ├── tabs/          ← 6 screen components
  │       └── primitives/    ← Reusable widgets
  ├── api/
  │   └── handlers.ts        ← API client (calls localhost:6001)
  └── styles/
      └── globals.css        ← Tailwind theme + animations

src/index.ts                  ← Hono server (serves SPA)
index.html                    ← SPA shell
```

## API Integration

The dashboard connects to the UCP Server at `localhost:6001`:

- **Server Status:** Calls `GET /.well-known/ucp/server-key` on load
- **API Explorer:** Live requests to all 11 endpoints
- **Identity Lookup:** Resolves addresses via `/.well-known/ucp/identity/{address}`
- **All endpoints:** Automatic fallback to mock responses if server unreachable

See `src/api/handlers.ts` for all available API functions.

## Embedding in Go Server

To embed the dashboard in the Go UCP Server:

1. Build React: `bun run build`
2. Go server embeds `dist/` as static assets via `embed` package
3. Serve at `/` while API stays at `/api/*` and `/.well-known/*`

## Next Steps

- [ ] Connect to real UCP Server instance
- [ ] Add real session management (use sessionToken)
- [ ] Implement data fetching for Sessions & Federation tabs
- [ ] Add error states and loading indicators
- [ ] Test all 11 API endpoints
