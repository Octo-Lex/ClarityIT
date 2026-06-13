# ClarityIT Genesis Architecture Decision Pack - Combined Review

# ClarityIT Genesis Architecture Decision Pack

This pack formalizes the current ClarityIT Genesis Build baseline.

It is intended to become the initial control-plane contract for the repository: deployment, runtime split, events, object model, agents, storage boundaries, schema governance, and validation strategy.

## Locked architecture baseline

| Area | Decision |
|---|---|
| Deployment | Sovereign Hybrid: self-hosted Proxmox origin + Cloudflare Tunnel + web interface |
| Runtime | Go control plane + Python reasoning workers |
| Agent effects | All effects pass through the Go Tool Gateway |
| Events | PostgreSQL outbox is canonical; NATS JetStream transports events |
| Event subject pattern | `clarity.v1.<domain>.<entity>.<action>` |
| Object model | Universal object spine + typed extension tables |
| Agent model | Internal agent identities + tool grants + A0-A5 autonomy ladder |
| Storage | PostgreSQL truth, NATS movement, MinIO/S3 artifacts, Redis/Valkey speed, Git knowledge/config, vector indexes semantic projection |
| Schema governance | CUE as build-time contract compiler only; no request hot-path CUE evaluation |
| Validation | Genesis scenario packs / golden threads, not a first MVP scenario |

## Contents

```text
clarityit-genesis-architecture-decision-pack/
  README.md
  docs/
    adr/                         # ADR-001 through ADR-020
    architecture/
      genesis-build-map.md
      scenario-packs.md
      service-topology.md
  schemas/
    cue/
      autonomy.cue
      events.cue
      ontology.cue
      permissions.cue
      storage.cue
      tools.cue
      views.cue
  migrations/
    001_core_extensions.sql
    002_outbox_and_audit.sql
    003_object_spine.sql
    004_context_graph.sql
    005_agent_esaa.sql
    006_storage_refs.sql
```

## How to use this pack

1. Copy `docs/adr` into the project repository.
2. Treat `schemas/cue` as the contract source for generated artifacts.
3. Treat `migrations` as the initial SQL skeleton, not final production migrations.
4. Implement code generation from CUE to:
   - SQL validation constants
   - Go types
   - TypeScript types
   - OpenAPI schemas
   - Agent tool manifests
   - UI view metadata
   - test fixtures

## Non-negotiable runtime rule

CUE is not evaluated in normal request paths. It is used for build-time validation and generation only.

```text
CUE contracts → CI validation → generated artifacts → runtime services
```

## Next implementation task

Create the repository skeleton and wire the first generated contracts:

```text
apps/web
services/api
services/agents
services/context
services/workers
packages/contracts
schemas/cue
migrations
```


---
# ADR Index

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


---

# ADR-001: Sovereign Hybrid Deployment

## Status

Accepted.

## Decision

Adopt self-hosted Proxmox origin with Cloudflare Tunnel and web-first access.

## Rationale

Data sovereignty, secure ingress, operational realism, and low deployment friction.

## Consequences

Only web/API ingress is exposed through Cloudflare Tunnel. Databases, NATS, Redis/Valkey, MinIO, Proxmox APIs, and workers remain private.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-002: PostgreSQL Primary with Graph Tables

## Status

Accepted.

## Decision

Use PostgreSQL as the canonical data store with relational tables plus graph-style context tables.

## Rationale

Keeps source-of-truth, audit, IAM, context, idempotency, and transactional integrity in one operational database.

## Consequences

Graph database can be introduced later as a projection if traversal complexity exceeds PostgreSQL recursive CTE capabilities.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-003: Go Modular Monolith API

## Status

Accepted.

## Decision

Use a Go modular monolith for the primary API and domain services.

## Rationale

Avoids premature microservice complexity while preserving module boundaries.

## Consequences

Modules include IAM, Work, Queue, Project, Wiki, Hub, Grid, Context API, and Tool Gateway.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-004: Separate Worker Runtime

## Status

Accepted.

## Decision

Run background workloads outside the API process.

## Rationale

Workers isolate slow, retryable, probabilistic, or integration-heavy work from user-facing request paths.

## Consequences

Workers include outbox, context ingestion, event handling, integrations, and agents.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-005: Go Control Plane and Python Reasoning Workers

## Status

Accepted.

## Decision

Use Go for deterministic control-plane services and Python for agent reasoning workers.

## Rationale

Go provides strong operational reliability; Python accelerates AI/LLM development.

## Consequences

Python workers cannot mutate state directly and must call the Go Tool Gateway.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-006: PostgreSQL Outbox Canonical, NATS Transport

## Status

Accepted.

## Decision

Persist domain events in PostgreSQL outbox inside the transaction, then publish to NATS JetStream asynchronously.

## Rationale

Prevents event loss and supports replay, retries, dedupe, and auditability.

## Consequences

NATS is transport, not canonical state.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-007: ESAA Structured Agent Intentions

## Status

Accepted.

## Decision

Agents emit structured intentions rather than directly mutating state.

## Rationale

Separates probabilistic reasoning from deterministic effect application.

## Consequences

Intentions are validated by schema, IAM, autonomy policy, approvals, MFA, SoD, idempotency, and audit before execution.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-008: S3-Compatible Storage with MinIO Profile

## Status

Accepted.

## Decision

Use S3-compatible object storage and provide a MinIO deployment profile.

## Rationale

Supports self-hosting on Proxmox while avoiding hard dependency on one object store.

## Consequences

PostgreSQL owns metadata and permissions for every object.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-009: Redis-Compatible Ephemeral Runtime Cache

## Status

Accepted.

## Decision

Use Redis-compatible backend, preferably Valkey by default, for ephemeral acceleration.

## Rationale

Supports cache, rate limiting, presence, leases, and short-lived context bundles.

## Consequences

Redis/Valkey must not store canonical audit, permissions, domain state, or idempotency truth.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-010: CUE Build-Time Contract Compiler Only

## Status

Accepted.

## Decision

Use CUE for contracts and generation, not request hot-path validation.

## Rationale

Provides strong cross-contract validation without runtime overhead.

## Consequences

Generated artifacts are used by Go, TypeScript, SQL, OpenAPI, and agent tools.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-011: REST and OpenAPI First

## Status

Accepted.

## Decision

Expose external APIs through REST with OpenAPI 3.1.

## Rationale

Simplifies authorization, audit, generated clients, and agent tool contracts.

## Consequences

GraphQL may be considered later if projection needs justify it.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-012: Web-First PWA

## Status

Accepted.

## Decision

Build the primary interface as a web-first PWA.

## Rationale

Matches Cloudflare Tunnel access and avoids premature desktop/mobile complexity.

## Consequences

Tauri and native mobile remain future clients.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-013: Universal Object Spine with Typed Extensions

## Status

Accepted.

## Decision

Represent all important entities through a shared object spine and domain-specific extension tables.

## Rationale

Prevents internal tool sprawl while preserving domain semantics.

## Consequences

Objects share links, comments, refs, ownership, status, permissions, context, and audit behavior.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-014: PostgreSQL FTS and pgvector First

## Status

Accepted.

## Decision

Use PostgreSQL full-text search and pgvector for Genesis semantic retrieval.

## Rationale

Avoids additional storage engines while enabling search and embeddings.

## Consequences

Typesense/Qdrant can be added later as projections.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-015: Internal Agent Identity Model

## Status

Accepted.

## Decision

Create internal agent identities with tool grants and team scope.

## Rationale

Agents need auditable authority without introducing broad service principals.

## Consequences

Agent authority is bounded by team, tool, autonomy level, risk, approval, MFA, and expiry.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-016: A0-A5 Autonomy Ladder

## Status

Accepted.

## Decision

Classify every agent capability by autonomy level A0 through A5.

## Rationale

Enables full-system design while controlling operational risk.

## Consequences

High-risk actions default to A4 approval-gated with recent MFA.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-017: Least-Privilege IT Role Matrix

## Status

Accepted.

## Decision

Seed explicit IT roles and permissions with least privilege.

## Rationale

ClarityIT touches infrastructure and must prevent broad admin authority.

## Consequences

Roles include owner, admin, manager, member, viewer, on_call_engineer, infrastructure_engineer, security_admin, auditor, and automation_operator.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-018: Proxmox Read-Rich, Action-Gated Integration

## Status

Accepted.

## Decision

Start Proxmox integration with inventory, metrics, backup status, and alert correlation; gate actions.

## Rationale

Provides value without unsafe automation defaults.

## Consequences

Start/stop/restart/migrate require approval and MFA; destructive actions disabled by default.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-019: Git for Knowledge and Config, PostgreSQL for Operations

## Status

Accepted.

## Decision

Use Git for runbooks, docs, skills, schemas, deployment templates, and config snapshots.

## Rationale

Git is excellent for human-readable knowledge and configuration history, not live operations.

## Consequences

Tickets, incidents, audit, sessions, permissions, idempotency, and agent runs remain in PostgreSQL.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---

# ADR-020: Golden Thread Scenario Packs

## Status

Accepted.

## Decision

Use scenario packs as integration validation, not as MVP scope.

## Rationale

Tests whether identity, events, context, agents, views, audit, approvals, storage, and workflows compose correctly.

## Consequences

Required packs include incident response, service desk triage, infrastructure action approval, knowledge drift correction, project risk detection, and denied agent action.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.


---
# Service Topology

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


---
# Genesis Build Map

# Genesis Build Map

## System thesis

ClarityIT is an AI-native IT operations OS built around identity, events, context, agents, operational objects, and controlled effects.

## Primary layers

```text
Web interface
  ↓
Go API and Tool Gateway
  ↓
IAM / permissions / audit / idempotency
  ↓
Universal object runtime
  ↓
PostgreSQL source of truth
  ↓
Outbox → NATS JetStream
  ↓
Workers: context, integrations, agents, outbox
  ↓
Context graph, object storage, cache, Git-backed knowledge
```

## Genesis rule

The first complete system should be shallow across modules but deep in control-plane integrity.

Required surfaces:

- Command Center
- Queue
- Project
- Wiki
- Hub
- Grid
- Agent Console
- Approval Inbox
- Audit Timeline
- Context Graph

Required control-plane primitives:

- IAM
- Events
- Audit
- Idempotency
- Outbox
- Context graph
- Tool Gateway
- Agent identities
- Autonomy levels
- Storage boundaries


---
# Scenario Packs

# Genesis Scenario Packs

Scenario packs are not MVP scope boundaries. They are golden-thread integration tests for the AI-native operating system.

## Pack 1: Incident response

```text
alert.triggered
  → context bundle assembled
  → alert agent creates recommendation
  → incident opened
  → incident room created
  → runbook suggested
  → approval requested for high-risk action
  → human approves with MFA
  → tool gateway executes
  → audit written
  → event emitted
  → context graph updated
  → summary generated
```

## Pack 2: Service desk triage

Validates ticket creation, SLA, queue assignment, knowledge suggestions, draft reply, context links, and audit.

## Pack 3: Infrastructure action approval

Validates Proxmox action gating, MFA, approval workflow, SoD policy, tool execution, evidence capture, and rollback notes.

## Pack 4: Knowledge drift correction

Validates incident outcome → runbook update draft → approval → Git-backed doc update → context graph refresh.

## Pack 5: Project risk detection

Validates project schedule, linked incidents, capacity context, risk recommendation, and manager approval.

## Pack 6: Denied agent action

Validates an agent attempting an action outside its tool grant or autonomy level. The system must deny, audit, and explain the denial.
