# Deployment

## Platform

Bun HTTP server on Hetzner VPS. Pulumi manages infrastructure. Caddy handles TLS + reverse proxy. No Docker.

## Environments

| Environment | URL / Host | Branch | Deploy trigger |
|-------------|------------|--------|----------------|
| Production | [TODO] | `main` | GitHub Actions on push |

## Data Residency

| Region | Provider | Use for |
|--------|----------|---------|
| nbg1 | Hetzner | EU GDPR |
| hel1 | Hetzner | EU + likely UK GDPR adequate |
| lhr | Vultr | UK GDPR mandatory (healthcare, financial) |

## Prerequisites

- Pulumi CLI installed
- Bun installed locally
- VPS SSH key (Ed25519)
- GitHub repo with Actions enabled

## Provision VPS

```bash
cd infra/pulumi
bun install
pulumi stack init prod
pulumi config set --secret sshPublicKey "$(cat ~/.ssh/id_ed25519.pub)"
pulumi config set sshPort 2222
pulumi up
```

## DNS

Point your domain at the floating IP.

```bash
pulumi stack output serverIp
```

## First Deploy

Push to `main` — GitHub Actions handles it.

Required secrets: `VPS_HOST`, `VPS_USER` (deploy), `VPS_SSH_KEY`, `VPS_SSH_PORT` (2222).

## Connect to Server

```bash
pulumi stack output sshCommand
# → ssh deploy@<ip> -p 2222
```

## Deploy Process

```bash
# build binary
bun build --compile --target=bun src/index.ts --outfile=[project-name]

# deploy via GitHub Actions on push to main
```

## Environment Variables

> Never commit values. Document keys here.

| Key | Required | Description |
|-----|----------|-------------|
| `DATABASE_URL` | Yes | Postgres connection string |
| `BETTER_AUTH_SECRET` | Yes | Secret for signing sessions |
| `GOOGLE_CLIENT_ID` | No | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | No | Google OAuth client secret |
| `MICROSOFT_CLIENT_ID` | No | Microsoft Entra ID client ID |
| `MICROSOFT_CLIENT_SECRET` | No | Microsoft Entra ID client secret |
| `MICROSOFT_TENANT_ID` | No | Client's Azure AD tenant ID |
| `[TODO]` | [TODO] | [TODO] |

## Health Check

`GET /health` → `{ status: "ok" }`

---

*Last updated: [DATE]*
