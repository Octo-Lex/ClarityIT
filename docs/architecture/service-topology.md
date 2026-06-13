# ClarityIT Genesis Service Topology

## Deployment shape

```text
Internet
  ↓
Cloudflare
  ↓
Cloudflare Tunnel
  ↓
Proxmox host / cluster
  ↓
Reverse proxy: Caddy or Traefik
  ↓
ClarityIT Web + API
```

## Internal services

```text
clarity-web
clarity-api
clarity-tool-gateway
clarity-agent-supervisor
clarity-python-agent-worker
clarity-context-worker
clarity-outbox-worker
clarity-integration-worker
postgres
nats-jetstream
valkey-or-redis
minio
cloudflared
```

## Exposure rule

Only the web/API entrypoint is exposed through Cloudflare Tunnel. Internal services remain private.

## Recommended early deployment

A single Proxmox VM or LXC with Docker Compose is acceptable for Genesis, provided service boundaries already match the target topology.

## Recommended production split

```text
Node/VM 1: web + api + reverse proxy
Node/VM 2: PostgreSQL
Node/VM 3: NATS + Valkey/Redis
Node/VM 4: MinIO
Node/VM 5: agent and integration workers
```
