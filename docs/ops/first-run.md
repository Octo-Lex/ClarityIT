# First Run Guide — ClarityIT

## Prerequisites

- Docker 20+ and Docker Compose v2
- A machine (LXC, VM, or bare metal) with at least 4GB RAM
- Network access (LAN or internet-facing via Cloudflare Tunnel)

## Step 1: Get the Code

```bash
git clone <repo-url> /opt/clarityit
cd /opt/clarityit
```

## Step 2: Configure Environment

```bash
cp services/api/.env.example .env
```

Edit `.env` and set **at minimum**:

```ini
CLARITY_ENV=development  # Use "production" for real deployments
DATABASE_URL=postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable
JWT_SECRET=<generate with: openssl rand -base64 48>
HMAC_KEY=<generate with: openssl rand -base64 48>
```

For production, also set:
```ini
CLARITY_ENV=production
NATS_URL=nats://nats:4222
REDIS_URL=redis://redis:6379/0
```

## Step 3: Start Services

```bash
docker compose up -d
```

Wait for PostgreSQL to be healthy:
```bash
docker compose ps  # All services should show "Up"
```

## Step 4: Run Migrations

```bash
make migrate
```

## Step 5: Bootstrap First User

The first user must be created via the bootstrap endpoint (no registration):

```bash
curl -X POST http://localhost:8765/api/bootstrap \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@yourcompany.com",
    "password": "change-me-immediately",
    "name": "Admin User",
    "team_name": "My Team"
  }'
```

This creates:
- The first user (owner + admin)
- The first team
- A personal access token

**The bootstrap endpoint is locked after first use.**

## Step 6: Verify

```bash
# Health check
curl http://localhost:8765/health

# Web UI
open http://localhost:3000
```

Login with the credentials from Step 5.

## Step 7: Configure Reasoning Worker (Optional)

The reasoning worker needs a service token:

1. Login as owner
2. Create an integration key with wildcard scope
3. Set in `.env`:
```ini
WORKER_TOKEN=<integration-key-from-api>
TEAM_ID=<team-uuid>
API_BASE_URL=http://clarityit-api:8765
```

Restart:
```bash
docker compose up -d clarityit-reasoning-worker
```

## Next Steps

- [Cloudflare Tunnel Setup](deployment/cloudflare-tunnel.md)
- [Admin Runbook](admin-runbook.md)
- [Backup/Restore](backup-restore.md)
