# Cloudflare Tunnel Deployment Guide

## Architecture

```
Internet → Cloudflare Edge → cloudflared → [nginx:3000 | api:8765]
                                                     ↓
                            Proxmox LXC (192.168.3.20)
                            ┌──────────────────────────┐
                            │  nginx (port 3000)        │
                            │  Go API (port 8765)       │
                            │  PostgreSQL (port 5432)   │ ← PRIVATE
                            │  NATS (port 4222)         │ ← PRIVATE
                            │  Redis (port 6379)        │ ← PRIVATE
                            │  MinIO (port 9000)        │ ← PRIVATE
                            │  Workers (no ports)       │ ← PRIVATE
                            └──────────────────────────┘
```

## Exposure Policy

| Service | Exposed | Reason |
|---------|---------|--------|
| nginx (frontend) | ✅ Yes | Static files + API proxy |
| Go API | ✅ Yes (via nginx) | REST API endpoints |
| PostgreSQL | ❌ No | Internal only, Docker network |
| NATS JetStream | ❌ No | Internal only, Docker network |
| Redis | ❌ No | Internal only, Docker network |
| MinIO | ❌ No | Internal only, S3 API via Go handler |
| Outbox Worker | ❌ No | No listening port |
| Context Worker | ❌ No | No listening port |
| Reasoning Worker | ❌ No | No listening port, HTTP client only |
| Proxmox VE | ❌ No | Management network only |

## Cloudflare Tunnel Setup

### 1. Install cloudflared

```bash
# On the LXC container
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o /usr/local/bin/cloudflared
chmod +x /usr/local/bin/cloudflared
```

### 2. Authenticate

```bash
cloudflared tunnel login
```

### 3. Create Tunnel

```bash
cloudflared tunnel create clarityit
# Note the tunnel ID from the output
```

### 4. Configure DNS

```bash
cloudflared tunnel route dns clarityit clarityit.yourdomain.com
```

### 5. Create Configuration

```ini
# ~/.cloudflared/config.yml
tunnel: <TUNNEL_ID>
credentials-file: /root/.cloudflared/<TUNNEL_ID>.json

ingress:
  - hostname: clarityit.yourdomain.com
    service: http://localhost:3000
  - service: http_status:404
```

### 6. Run as Service

```bash
cloudflared service install
systemctl enable cloudflared
systemctl start cloudflared
```

## Firewall Rules

Only the nginx port (3000) and API port (8765) should be accessible from the LAN. All other services bind to Docker internal networks only.

```bash
# UFW rules on the LXC
ufw allow 3000/tcp  # Frontend (nginx)
ufw allow 8765/tcp  # API (direct access for dev)
ufw deny 5432/tcp   # PostgreSQL
ufw deny 4222/tcp   # NATS
ufw deny 6379/tcp   # Redis
ufw deny 9000/tcp   # MinIO
```

## Rollback Steps

If a deployment fails:

1. **Container rollback**: `docker compose down && git checkout <previous-commit> && docker compose up -d --build`
2. **Database rollback**: `make migrate` to re-run migrations (idempotent where possible)
3. **Tunnel rollback**: `cloudflared tunnel delete clarityit && cloudflared tunnel create clarityit` (DNS records auto-update)
4. **Full reset**: `make migrate:fresh` (WARNING: destroys all data)

## Verification

```bash
make verify-deployment
```

This checks:
- Web frontend reachable at :3000
- API health at :8765/health
- Deep health at :8765/api/health/deep (authenticated)
- All Docker services healthy
- PostgreSQL, NATS, Redis, MinIO not externally exposed
