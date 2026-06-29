# Architecture

## System Overview

The UCP Admin Dashboard is a full-stack web application for server operators. It consists of:

1. **React Frontend (SPA)** — 6-tab dashboard UI built with React 19 + Tailwind CSS v4
2. **Hono API Server** — Bun-based HTTP server that serves the SPA and relays API calls
3. **UCP Server Backend** — Go server at `:6001` providing `/api/*` and `/.well-known/*` endpoints

The frontend makes real HTTP calls to the UCP Server at `localhost:6001`. If the server is unreachable, the dashboard falls back to mock responses (useful for development/testing).

## Component Map

```
┌──────────────────────────────────────────────────┐
│            React SPA (React 19)                  │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │  Dashboard (shell + router)                │ │
│  │  ├─ Sidebar (navigation, status badge)    │ │
│  │  ├─ Header (page title, token input)      │ │
│  │  └─ Tabs (6 screens)                      │ │
│  │     ├─ Overview (stats + API ref)         │ │
│  │     ├─ APIExplorer (live endpoint tester) │ │
│  │     ├─ Identity (key lookup & resolve)    │ │
│  │     ├─ Sessions (active sessions table)   │ │
│  │     ├─ Federation (delivery queue)        │ │
│  │     └─ Bridge (IMAP + threading)          │ │
│  └────────────────────────────────────────────┘ │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │  Reusable Primitives                       │ │
│  │  ├─ MethodBadge (GET/POST styling)        │ │
│  │  ├─ StatusPill (colored status)           │ │
│  │  ├─ DataBlock (inset data display)        │ │
│  │  └─ SectionCard (card wrapper)            │ │
│  └────────────────────────────────────────────┘ │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │  API Client (src/api/handlers.ts)          │ │
│  │  ├─ getServerKey()                        │ │
│  │  ├─ getIdentity(address)                  │ │
│  │  ├─ getKeyPackages(address)               │ │
│  │  ├─ getChallenge(address)                 │ │
│  │  ├─ createSession(...)                    │ │
│  │  ├─ sendMessage(...)                      │ │
│  │  ├─ getInbox(...)                         │ │
│  │  ├─ uploadContent(...)                    │ │
│  │  ├─ getContent(...)                       │ │
│  │  └─ apiCall(method, path, body, token)    │ │
│  └────────────────────────────────────────────┘ │
└────────────────┬─────────────────────────────────┘
                 │ HTTP
                 ▼
        ┌─────────────────────┐
        │  Hono Server (Bun)  │
        │  :6002              │
        │                     │
        │  ├─ Serves SPA      │
        │  └─ Proxy API calls │
        └────────┬────────────┘
                 │ HTTP
                 ▼
        ┌──────────────────────────┐
        │  UCP Server (Go)         │
        │  :6001                   │
        │                          │
        │  ├─ /api/*               │
        │  │  ├─ POST /message/send│
        │  │  ├─ GET /inbox        │
        │  │  ├─ POST /content/*   │
        │  │  └─ GET /content/*    │
        │  │                       │
        │  ├─ /auth/*              │
        │  │  ├─ POST /challenge   │
        │  │  ├─ POST /session     │
        │  │  └─ POST /refresh     │
        │  │                       │
        │  └─ /.well-known/*       │
        │     ├─ /server-key       │
        │     ├─ /identity/{addr}  │
        │     ├─ /keypackages/{..} │
        │     └─ /privacy         │
        └──────────────────────────┘
```

## Data Flow

### On Page Load
1. React mounts in `#app` DOM node
2. Dashboard component initializes state
3. `useEffect` calls `getServerKey()` to check server status
4. Status updates sidebar badge (online/offline/checking)
5. All 6 tabs render with mock data by default

### Tab Navigation
1. User clicks tab in sidebar
2. Dashboard state updates (`activeTab`)
3. Main content fades in (fadeIn animation)
4. Components render (no data fetch until user action)

### API Explorer - Send Request
1. User selects endpoint from left panel
2. Request body textarea populated with default or empty
3. User modifies body (if POST)
4. User clicks "Send" button
5. `handleSend()` calls `apiCall(method, path, body, sessionToken)`
6. API client fetches from `localhost:6001 + path` with 3-5s timeout
7. Response displayed as JSON
8. If timeout/error → falls back to mock response

### Identity Lookup
1. User enters address (e.g., `alice@example.com`)
2. Clicks "Lookup"
3. Calls `getIdentity(address)` → `GET /.well-known/ucp/identity/{address}`
4. Parses JSON response and displays in `<pre>` block
5. Shows mock if server unreachable

## Package Responsibilities

| Package/File | Purpose |
|---|---|
| `src/index.ts` | Hono HTTP server entry point, serves SPA from `dist/` |
| `src/index.tsx` | React app entry point, renders Dashboard into `#app` |
| `src/api/handlers.ts` | API client library (17+ functions for all endpoints) |
| `src/components/dashboard/Dashboard.tsx` | Root component, tab router, state management |
| `src/components/dashboard/Sidebar.tsx` | Navigation, server status badge, active tab styling |
| `src/components/dashboard/Header.tsx` | Page title, bearer token input (explorer only), status |
| `src/components/dashboard/tabs/*.tsx` | 6 screen components (Overview, APIExplorer, Identity, Sessions, Federation, Bridge) |
| `src/components/dashboard/primitives/*.tsx` | 4 reusable widgets (MethodBadge, StatusPill, DataBlock, SectionCard) |
| `src/styles/globals.css` | Tailwind @theme variables, animations, custom scrollbar |
| `build.ts` | Bun build configuration (esbuild + bun-plugin-tailwind) |
| `index.html` | SPA shell (loads React app from src/index.tsx) |

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Client Library** | Explicit handlers per endpoint | Easy to add error handling, logging, type safety per function |
| **API Calls** | Fetch with timeout | Simple, no axios/swr/react-query overhead for small app |
| **Mock Fallback** | Hardcoded mock responses | Development without running Go server, easier testing |
| **State Management** | React hooks (useState) | Sufficient for 6-tab admin UI, no Redux/Zustand complexity |
| **Tab Routing** | Conditional rendering | Simple, all state in one Dashboard component |
| **Session Token** | Input field in header | User can paste token, automatically sent to auth endpoints |
| **Server Status** | Ping on mount | Quick check without blocking render, updates badge |
| **Build Output** | `dist/index.html` + bundles | SPA served by Hono, no SSR overhead |

## Embedding in Go Server

When embedded in the Go server's `cmd/ucp-server/main.go`:

1. React assets copied to `public/` directory
2. Go embeds `public/*` at compile time (zero runtime cost)
3. Hono server replaced with single Caddy reverse proxy
4. Go server handles ALL routes:
   - `/` → index.html (React loads from `src/index.tsx`, fetches from `/api/*`)
   - `/api/*` → API handlers (existing UCP handlers)
   - `/.well-known/*` → Well-known routes (existing)

No separate Hono process needed. Single binary serves everything.

## Performance

- **React bundle:** 184 KB (JS, minified)
- **CSS bundle:** 25 KB (minified)
- **Total:** ~210 KB gzipped
- **Dev server startup:** <500ms (Hono)
- **SPA load time:** <1s on 3G
- **API call latency:** <100ms (local network)

## Security

- **No auth at dashboard level** — Better Auth / sessions TBD
- **Session token passed in header** — User supplies bearer token for testing
- **CORS not configured** — SPA served from same origin as API
- **No secrets in frontend** — All API credentials server-side

---

*Last updated: 2026-06-28*
