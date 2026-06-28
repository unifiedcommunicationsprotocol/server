# Deployment

## Architecture

The admin dashboard can be deployed in two ways:

### Option 1: Embedded in Go Server (Recommended)
Single binary serves dashboard + API. Simplest deployment.

### Option 2: Standalone Bun Server
Separate process, easier development iteration. Calls UCP Server at `:5150`.

## Environments

| Environment | URL | Setup | Deploy |
|-------------|-----|-------|--------|
| Development | `http://localhost:5173` | Standalone Bun | `bun run dev` |
| Production | `https://admin.ucp.example.com` | Embedded in Go | `go build ./cmd/ucp-server` |

## Embedding in Go Server (Production)

### Step 1: Build React Dashboard

```bash
cd www
bun install          # Already done
bun run build        # Compiles to dist/
```

Output: `dist/index.html` + `dist/chunk-*.js` + `dist/chunk-*.css` (~210 KB total)

### Step 2: Copy Assets to Go Server

```bash
# From repo root
cp -r www/dist/* cmd/ucp-server/public/
```

This copies the React bundle into Go server's embed directory.

### Step 3: Embed in Go Binary

In `cmd/ucp-server/main.go`:

```go
package main

import (
    "embed"
    "net/http"
    // ... existing imports ...
)

// Embed the dashboard assets
//go:embed public/*
var public embed.FS

func main() {
    // ... existing UCP server setup ...

    // Serve dashboard on / (before registering API routes)
    http.Handle("/", http.FileServer(http.FS(public)))
    
    // API routes (unchanged)
    http.HandleFunc("/api/message/send", handleMessageSend)
    http.HandleFunc("/auth/challenge", handleChallenge)
    // ... rest of API routes ...

    port := ":5150"
    log.Printf("🚀 UCP Server with Admin Dashboard running on %s\n", port)
    log.Fatal(http.ListenAndServe(port, nil))
}
```

### Step 4: Build Single Binary

```bash
cd ..
go build -o ucp-server ./cmd/ucp-server
```

Result: Single `ucp-server` binary that serves:
- `/` → React dashboard (HTML + JS + CSS)
- `/api/*` → UCP API endpoints
- `/.well-known/*` → Identity & key endpoints

### Step 5: Deploy

```bash
# Copy to VPS
scp ucp-server deploy@<VPS_IP>:/opt/ucp/

# SSH and run
ssh deploy@<VPS_IP>
cd /opt/ucp
./ucp-server
```

Binary size: ~15-20 MB (Go binary only, React embedded)

## Standalone Bun Server (Development)

### Single Process (Alternative)

Dashboard served by Bun binary on port `5173`. Caddy reverse proxy on port `443` handles TLS (optional).

```
User → HTTPS (port 443, Caddy)
       ↓
       Caddy reverse proxy
       ↓
       Bun server (port 3000)
       ├─ /api/* → Hono API handlers
       ├─ / → React SPA (index.html + bundle.js)
       └─ /static/* → Other assets
```

### Separate UCP Server + Dashboard (Alternative)

If dashboard runs on separate server than UCP API:

```
┌──────────────────────────┐         ┌──────────────────────────┐
│ VPS 1 (admin.ucp.com)    │         │ VPS 2 (api.ucp.com)      │
├──────────────────────────┤         ├──────────────────────────┤
│ Caddy (443)              │         │ Caddy (443)              │
├──────────────────────────┤         ├──────────────────────────┤
│ Dashboard Bun (3000)     │         │ UCP Server Go (5150)     │
│ - React SPA              │         │ - /api/ucp/*             │
│ - Auth (Better Auth)     │         │ - /api/well-known/*      │
│ - SQLite session store   │         │ - Message routing        │
└──────────────────────────┘         └──────────────────────────┘
```

Current architecture: **Dashboard + UCP Server on same VPS** (simpler).

## Prerequisites

- **Bun 1.0+** installed locally (for building)
- **Hetzner VPS** with Ubuntu 22.04+ or Debian 12+
- **SSH key** (Ed25519) for VPS access
- **Domain name** pointing to VPS IP

## Data Residency

| Region | Provider | Use case |
|--------|----------|---------|
| nbg1 (Nuremberg) | Hetzner | EU GDPR compliant |
| hel1 (Helsinki) | Hetzner | EU + UK GDPR adequate |
| lhr (London) | Vultr | UK GDPR mandatory |

Hetzner's data centers are in EU; set `location` in Pulumi config to preferred region.

## Build & Compile

### Step 1: Build React

```bash
cd www

# Install dependencies
bun install

# Type-check
bun run typecheck

# Lint
bun run lint

# Build React to static assets (outputs to dist/)
bun run build
```

This produces `www/dist/` with index.html + bundle.js.

### Step 2: Compile Bun Binary

```bash
bun build --compile --target=bun src/index.ts --outfile=ucp-dashboard
```

Creates `ucp-dashboard` executable (~50 MB, statically linked).

Alternatively, if embedding in Go Server:

```bash
# In Go server repo
cp -r www/dist/* cmd/ucp-server/public/
go build ./cmd/ucp-server
```

The Go binary now serves dashboard + API.

## Provision VPS

### Manual Setup (Recommended for learning)

1. **Create VPS** on Hetzner, record IP address
2. **SSH into server:**
   ```bash
   ssh root@<IP>
   ```
3. **Install Bun** (if running dashboard separately):
   ```bash
   curl https://bun.sh/install | bash
   ```
4. **Create deploy user:**
   ```bash
   useradd -m -s /bin/bash deploy
   ```
5. **Configure SSH** for deploy user:
   ```bash
   mkdir -p /home/deploy/.ssh
   echo "your-public-key" > /home/deploy/.ssh/authorized_keys
   chown -R deploy:deploy /home/deploy/.ssh
   chmod 600 /home/deploy/.ssh/authorized_keys
   ```
6. **Install Caddy** (TLS reverse proxy):
   ```bash
   apt update && apt install -y caddy
   ```
7. **Create app directory:**
   ```bash
   mkdir -p /opt/ucp-dashboard
   chown deploy:deploy /opt/ucp-dashboard
   ```

### Via Pulumi (Infrastructure as Code)

```bash
cd infra/pulumi
bun install
pulumi stack init prod
pulumi config set region nbg1
pulumi config set sshPublicKey "$(cat ~/.ssh/id_ed25519.pub)"
pulumi up
```

This provisions VPS, configures SSH, installs Caddy, creates app directory. Outputs server IP + SSH command.

## Deploy Binary

### Method 1: Manual SCP

```bash
# Build locally
bun run build
bun build --compile --target=bun src/index.ts --outfile=ucp-dashboard

# Copy to VPS
scp -P 2222 ucp-dashboard deploy@<VPS_IP>:/opt/ucp-dashboard/
scp -P 2222 .env.production deploy@<VPS_IP>:/opt/ucp-dashboard/.env

# SSH and start
ssh deploy@<VPS_IP> -p 2222
cd /opt/ucp-dashboard
chmod +x ucp-dashboard
./ucp-dashboard
```

### Method 2: GitHub Actions CI/CD

Push to `main` branch, Actions automatically:
1. Builds React + compiles Bun binary
2. Uploads to VPS via SSH
3. Restarts systemd service

Required GitHub secrets:
- `VPS_HOST` — VPS IP or hostname
- `VPS_USER` — SSH user (deploy)
- `VPS_SSH_KEY` — SSH private key (base64)
- `VPS_SSH_PORT` — SSH port (default 2222)

Example workflow (`.github/workflows/deploy.yml`):

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: oven-sh/setup-bun@v1
      - run: cd www && bun install && bun run build && bun build --compile --target=bun src/index.ts --outfile=ucp-dashboard
      - run: |
          mkdir -p ~/.ssh
          echo "${{ secrets.VPS_SSH_KEY }}" | base64 -d > ~/.ssh/id_ed25519
          chmod 600 ~/.ssh/id_ed25519
          ssh-keyscan -H ${{ secrets.VPS_HOST }} >> ~/.ssh/known_hosts
          scp -P ${{ secrets.VPS_SSH_PORT }} www/ucp-dashboard ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }}:/opt/ucp-dashboard/
          ssh -p ${{ secrets.VPS_SSH_PORT }} ${{ secrets.VPS_USER }}@${{ secrets.VPS_HOST }} "systemctl restart ucp-dashboard"
```

## Configuration

### Environment Variables

Never commit these values. Populate at deployment time.

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `BETTER_AUTH_SECRET` | Yes | Session signing key (32+ bytes, base64) | `base64 /dev/urandom \| head -c 32` |
| `DATABASE_URL` | Yes | SQLite connection path | `sqlite:///opt/ucp-dashboard/dashboard.db` |
| `NODE_ENV` | Yes | Deployment environment | `production` |
| `PORT` | No | HTTP listen port (default 3000) | `3000` |
| `LOG_LEVEL` | No | Logging verbosity | `info` or `debug` |

Store in `/opt/ucp-dashboard/.env.production`:

```
BETTER_AUTH_SECRET=your-secret-here
DATABASE_URL=sqlite:///opt/ucp-dashboard/dashboard.db
NODE_ENV=production
PORT=3000
```

Load at startup via systemd EnvironmentFile.

### Caddy Reverse Proxy

Create `/etc/caddy/Caddyfile`:

```
admin.ucp.example.com {
  encode gzip
  
  reverse_proxy localhost:3000 {
    header_up X-Forwarded-Proto https
    header_up X-Real-IP {http.request.remote.host}
  }
  
  # Redirect www to naked domain
  @www host www.admin.ucp.example.com
  redir @www https://admin.ucp.example.com{uri}
  
  # Security headers
  header Strict-Transport-Security "max-age=31536000; includeSubDomains"
  header X-Content-Type-Options "nosniff"
  header X-Frame-Options "DENY"
}
```

Reload:

```bash
sudo systemctl reload caddy
```

## Systemd Service

Create `/etc/systemd/system/ucp-dashboard.service`:

```ini
[Unit]
Description=UCP Admin Dashboard
After=network.target
Wants=caddy.service

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/ucp-dashboard
ExecStart=/opt/ucp-dashboard/ucp-dashboard
Restart=on-failure
RestartSec=5s

# Environment file with secrets
EnvironmentFile=/opt/ucp-dashboard/.env.production

# Logging
StandardOutput=append:/var/log/ucp-dashboard/access.log
StandardError=append:/var/log/ucp-dashboard/error.log

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ucp-dashboard
sudo systemctl start ucp-dashboard

# Check status
sudo systemctl status ucp-dashboard

# View logs
sudo journalctl -u ucp-dashboard -f
```

## Database Initialization

On first deployment, run migrations:

```bash
# SSH into server
ssh deploy@<VPS_IP> -p 2222

# Create database and run migrations
cd /opt/ucp-dashboard
./ucp-dashboard  # Starts server
# (Migrations run automatically on startup if DATABASE_URL points to a new .db file)
```

Migrations are baked into the binary (Drizzle migrations compiled with code). No separate migration step needed.

## Health Check

```bash
curl https://admin.ucp.example.com/api/health
# → { "status": "ok" }
```

Caddy health probes this regularly. If endpoint returns non-2xx, Caddy logs warning but doesn't restart (Bun process manager handles restarts).

## Monitoring & Logs

### System Logs

```bash
sudo journalctl -u ucp-dashboard -n 50 -f
```

### Application Logs

```bash
tail -f /var/log/ucp-dashboard/error.log
```

### Error Tracking (Optional)

Configure Sentry for production error reporting:

```typescript
// src/index.ts
import * as Sentry from "@sentry/bun";

Sentry.init({
  dsn: process.env.SENTRY_DSN,
  environment: process.env.NODE_ENV,
});
```

Set `SENTRY_DSN` environment variable for production.

## Backup & Recovery

### Database Backup

SQLite database file: `/opt/ucp-dashboard/dashboard.db`

Backup to S3 or offsite storage:

```bash
# Automated backup script
#!/bin/bash
BACKUP_DATE=$(date +%Y-%m-%d_%H-%M-%S)
sqlite3 /opt/ucp-dashboard/dashboard.db ".backup /tmp/dashboard-$BACKUP_DATE.db"
aws s3 cp /tmp/dashboard-$BACKUP_DATE.db s3://my-backup-bucket/dashboard/
rm /tmp/dashboard-$BACKUP_DATE.db
```

Schedule via cron (daily):

```bash
0 2 * * * /opt/ucp-dashboard/backup.sh
```

### Restore from Backup

```bash
# Download backup
aws s3 cp s3://my-backup-bucket/dashboard/dashboard-2026-06-28_02-00-00.db .

# Stop service
sudo systemctl stop ucp-dashboard

# Restore
sqlite3 dashboard-backup.db ".backup /opt/ucp-dashboard/dashboard.db"

# Start service
sudo systemctl start ucp-dashboard
```

## Rollback

Keep previous binaries:

```bash
/opt/ucp-dashboard/
├── ucp-dashboard           # Current
├── ucp-dashboard.previous  # Previous version
└── .env.production
```

On deploy:

```bash
# Backup current
cp /opt/ucp-dashboard/ucp-dashboard /opt/ucp-dashboard/ucp-dashboard.backup

# Deploy new
scp ucp-dashboard deploy@<VPS_IP>:/opt/ucp-dashboard/

# If issues, revert
ssh deploy@<VPS_IP>
cp /opt/ucp-dashboard/ucp-dashboard.backup /opt/ucp-dashboard/ucp-dashboard
sudo systemctl restart ucp-dashboard
```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 3000
lsof -i :3000

# Kill process
kill -9 <PID>
```

### Permission Denied on Database File

```bash
sudo chown deploy:deploy /opt/ucp-dashboard/dashboard.db
sudo chmod 644 /opt/ucp-dashboard/dashboard.db
```

### Caddy TLS Certificate Not Renewing

```bash
# Check Caddy logs
sudo journalctl -u caddy -n 50 -f

# Manually renew
sudo caddy reload --config /etc/caddy/Caddyfile
```

### Out of Disk Space

```bash
# Check disk usage
df -h

# Clean old logs
sudo journalctl --vacuum=30d
sudo rm /var/log/ucp-dashboard/*.log.*.gz
```

---

*Last updated: 2026-06-28*
