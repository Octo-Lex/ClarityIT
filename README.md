# ClarityIT — AI-Native IT Operations OS

Sovereign hybrid IT operations platform: Go control plane + Python reasoning workers, PostgreSQL canonical truth, NATS JetStream events, universal object spine, A0–A5 agent autonomy.

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  React 19   │───▶│  Go API      │───▶│ PostgreSQL  │
│  Frontend   │◀───│  (chi/pgx)   │◀───│  37 tables  │
└─────────────┘    └──────┬───────┘    └─────────────┘
                          │
              ┌───────────┼───────────┐
              ▼           ▼           ▼
       ┌──────────┐ ┌──────────┐ ┌──────────────┐
       │  NATS    │ │  Redis   │ │ Python Worker│
       │JetStream │ │  Pub/Sub │ │ (Reasoning)  │
       └──────────┘ └──────────┘ └──────────────┘
              │           │
       ┌──────────┐ ┌──────────┐
       │  Outbox  │ │   WS     │
       │  Worker  │ │   Hub    │
       └──────────┘ └──────────┘
       ┌──────────┐
       │ Context  │
       │  Worker  │
       └──────────┘
```

## Services (Docker Compose)

| Service | Port | Description |
|---------|------|-------------|
| `clarityit-api` | 8765 | Go control plane (chi router, pgx, JWT auth) |
| `clarityit-web` | 3000 | React 19 frontend (nginx proxy) |
| `clarityit-outbox-worker` | — | Go worker: PostgreSQL outbox → NATS publish + Redis fanout |
| `clarityit-context-worker` | — | Go worker: NATS consumer → context graph (nodes, edges, evidence) |
| `clarityit-reasoning-worker` | — | Python worker: poll pending runs → generate intentions via ModelGateway |
| `postgres` | 5432 | PostgreSQL 16 — 37 tables, migrations 001–019 |
| `nats` | 4222 | NATS JetStream — `CLARITY_EVENTS` + `CLARITY_DLQ` streams |
| `redis` | 6379 | Redis — WS pub/sub fanout |
| `minio` | 9000 | MinIO — object storage |

## Agent Runtime Architecture

Phase 7 implements the Genesis agent autonomy model:

### A0–A5 Autonomy Ladder

| Level | Meaning |
|-------|---------|
| A0 | Read-only — no mutations |
| A1 | Low-risk reads with logging |
| A2 | Low-risk writes (comments, timeline) |
| A3 | Standard operations (status changes, assignments) |
| A4 | Significant changes (create/close incidents) |
| A5 | Full autonomy (all tools, approval bypass) |

### Tool Gateway Enforcement

Every agent tool execution passes through:

1. **Agent active?** — soft-deleted/disabled agents blocked
2. **Run active?** — only `pending`/`running` runs accepted
3. **Tool registered?** — must exist in `tool_registry` with risk level
4. **Grant exists?** — `agent_tool_grants` must have active, non-expired, non-revoked entry
5. **Autonomy check** — requested level ≤ agent max AND ≤ grant max
6. **Approval block** — `requires_approval = true` → blocked
7. **MFA block** — `requires_mfa = true` → blocked (real MFA deferred)
8. **Risk block** — `medium+` risk → blocked (until approval workflow)

### Data Flow

```
User creates Agent Identity → Grants tools → Triggers Run
                                          ↓
Python Reasoning Worker polls pending runs
  → StubModelGateway generates Intention
  → POSTs structured intention to Go API
                                          ↓
Tool Gateway validates:
  ✓ Agent active, run active, grant exists
  ✓ Autonomy within bounds
  ✓ No approval/MFA required
  ✓ Risk level acceptable
  → Execute or Block (with audit + outbox event)
```

### Reasoning Worker Isolation

The Python reasoning worker has **NO direct access** to:
- PostgreSQL (`DATABASE_URL` not set)
- NATS (`NATS_URL` not set)
- Redis (`REDIS_URL` not set)
- MinIO (`MINIO_ENDPOINT` not set)

It communicates **only** through the Go API HTTP interface.

## Model Gateway

```python
from model_gateway import StubModelGateway, IntentionShape

gateway = StubModelGateway()
intention = gateway.generate_intention(
    agent_run_id="...",
    context={},
    tool_grants=[...],
)
# intention is a validated IntentionShape with reasoning_summary
# chain_of_thought is always rejected/stripped
```

Placeholders exist for: `OpenAICompatibleGateway`, `LiteLLMGateway`, `LocalOllamaGateway`.

## Database Schema

18 migrations create 37+ tables across:
- **IAM**: users, teams, roles, permissions, sessions, tokens, invitations, access grants
- **Core**: objects, work_items, incidents, projects, links, comments
- **Events**: audit_logs, outbox_events, idempotency_keys, context_nodes, context_edges
- **Agent**: agent_identities, agent_tool_grants, agent_runs, agent_intentions, agent_effect_results, tool_registry
- **Integration**: integration_api_keys, assets, alerts, object_attachments

## Testing

```bash
# Backend (142 tests)
cd services/api && go test -p 1 -count=1 -timeout 180s ./...

# Frontend (21 tests)
cd web && npm test

# Python model gateway (9 tests)
cd services/workers/reasoning && python -m pytest test_model_gateway.py -v
```

Total: **172 tests** (142 backend + 21 frontend + 9 Python)

## Phase 8: Integrations

| Feature | Route | Auth |
|---------|-------|------|
| Integration Keys | `POST/GET/DELETE /api/teams/{id}/integration-keys` | JWT |
| Webhook Receiver | `POST /api/webhooks/{source}` | Integration Key |
| Proxmox Status | `GET /api/teams/{id}/integrations/proxmox/status` | JWT |
| Proxmox Sync | `POST /api/teams/{id}/integrations/proxmox/sync` | JWT |
| Assets | `GET /api/teams/{id}/assets` | JWT |
| Deep Health | `GET /api/health/deep` | JWT |

## Deployment

```bash
# Build and deploy all
docker compose up -d --build

# Reasoning worker needs WORKER_TOKEN
WORKER_TOKEN=<token> TEAM_ID=<uuid> docker compose up -d clarityit-reasoning-worker
```

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
