# Architecture Decision Records

| ADR | Title | Decision |
|---|---|---|
| [ADR-001](./ADR-001-sovereign-hybrid-deployment.md) | Sovereign Hybrid Deployment | Adopt self-hosted Proxmox origin with Cloudflare Tunnel and web-first access. |
| [ADR-002](./ADR-002-postgresql-primary-with-graph-tables.md) | PostgreSQL Primary with Graph Tables | Use PostgreSQL as the canonical data store with relational tables plus graph-style context tables. |
| [ADR-003](./ADR-003-go-modular-monolith-api.md) | Go Modular Monolith API | Use a Go modular monolith for the primary API and domain services. |
| [ADR-004](./ADR-004-separate-worker-runtime.md) | Separate Worker Runtime | Run background workloads outside the API process. |
| [ADR-005](./ADR-005-go-control-plane-and-python-reasoning-workers.md) | Go Control Plane and Python Reasoning Workers | Use Go for deterministic control-plane services and Python for agent reasoning workers. |
| [ADR-006](./ADR-006-postgresql-outbox-canonical-nats-transport.md) | PostgreSQL Outbox Canonical, NATS Transport | Persist domain events in PostgreSQL outbox inside the transaction, then publish to NATS JetStream asynchronously. |
| [ADR-007](./ADR-007-esaa-structured-agent-intentions.md) | ESAA Structured Agent Intentions | Agents emit structured intentions rather than directly mutating state. |
| [ADR-008](./ADR-008-s3-compatible-storage-with-minio-profile.md) | S3-Compatible Storage with MinIO Profile | Use S3-compatible object storage and provide a MinIO deployment profile. |
| [ADR-009](./ADR-009-redis-compatible-ephemeral-runtime-cache.md) | Redis-Compatible Ephemeral Runtime Cache | Use Redis-compatible backend, preferably Valkey by default, for ephemeral acceleration. |
| [ADR-010](./ADR-010-cue-build-time-contract-compiler-only.md) | CUE Build-Time Contract Compiler Only | Use CUE for contracts and generation, not request hot-path validation. |
| [ADR-011](./ADR-011-rest-and-openapi-first.md) | REST and OpenAPI First | Expose external APIs through REST with OpenAPI 3.1. |
| [ADR-012](./ADR-012-web-first-pwa.md) | Web-First PWA | Build the primary interface as a web-first PWA. |
| [ADR-013](./ADR-013-universal-object-spine-with-typed-extensions.md) | Universal Object Spine with Typed Extensions | Represent all important entities through a shared object spine and domain-specific extension tables. |
| [ADR-014](./ADR-014-postgresql-fts-and-pgvector-first.md) | PostgreSQL FTS and pgvector First | Use PostgreSQL full-text search and pgvector for Genesis semantic retrieval. |
| [ADR-015](./ADR-015-internal-agent-identity-model.md) | Internal Agent Identity Model | Create internal agent identities with tool grants and team scope. |
| [ADR-016](./ADR-016-a0-a5-autonomy-ladder.md) | A0-A5 Autonomy Ladder | Classify every agent capability by autonomy level A0 through A5. |
| [ADR-017](./ADR-017-least-privilege-it-role-matrix.md) | Least-Privilege IT Role Matrix | Seed explicit IT roles and permissions with least privilege. |
| [ADR-018](./ADR-018-proxmox-read-rich-action-gated-integration.md) | Proxmox Read-Rich, Action-Gated Integration | Start Proxmox integration with inventory, metrics, backup status, and alert correlation; gate actions. |
| [ADR-019](./ADR-019-git-for-knowledge-and-config-postgresql-for-operations.md) | Git for Knowledge and Config, PostgreSQL for Operations | Use Git for runbooks, docs, skills, schemas, deployment templates, and config snapshots. |
| [ADR-020](./ADR-020-golden-thread-scenario-packs.md) | Golden Thread Scenario Packs | Use scenario packs as integration validation, not as MVP scope. |
