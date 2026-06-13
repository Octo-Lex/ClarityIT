Architecture Baseline

Deployment: Sovereign Hybrid
Proxmox self-hosted origin + Cloudflare Tunnel + web interface.

Runtime: Go control plane + Python reasoning workers.
All agent effects pass through the Go Tool Gateway.

Events: PostgreSQL outbox is canonical; NATS JetStream transports events.
Subject pattern: clarity.v1.<domain>.<entity>.<action>.

Object model: Universal object spine + typed extension tables.

Agent model: Internal agent identities + tool grants + A0-A5 autonomy ladder.

Storage: PostgreSQL truth, NATS movement, MinIO/S3 artifacts, Redis/Valkey speed, Git knowledge/config, vector index semantic projection.

Schema governance: CUE as build-time contract compiler only; no hot-path runtime CUE evaluation.

Validation: Genesis scenario packs / golden threads, not a first MVP scenario.
