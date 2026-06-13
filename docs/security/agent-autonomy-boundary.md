# ClarityIT v1.0 Agent Autonomy Boundary

## Document Status
- **Version**: 1.0.0
- **Date**: 2026-06-14
- **Scope**: Agent runtime, tool gateway, policy enforcement, autonomy ladder, reasoning worker isolation

---

## 1. Autonomy Ladder (A0-A5)

| Level | Name | Execution Capability | Approval | MFA | v1.0 Status |
|-------|------|---------------------|----------|-----|-------------|
| A0 | Observant | None — read/observe only | — | — | ✅ Active |
| A1 | Advisory | Propose only, no execution | — | — | ✅ Active |
| A2 | Guided | Low-risk with approval | Required | No | ✅ Active |
| A3 | Supervised | Low/medium with approval | Required | No | ✅ Active |
| A4 | Operational | Any risk with approval + MFA | Required | For high/critical | ✅ Active |
| A5 | **Autonomous** | **Unrestricted** | — | — | **DISABLED** |

### A5 Enforcement
```go
// policy.go — Pre-check (before any DB lookup)
if req.AutonomyLevel == "A5" {
    return &Decision{
        Outcome:  OutcomeBlockedPolicy,
        Reason:   "a5_disabled",
        Check:    0,
        ToolName: req.ToolName,
    }, nil
}
```

A5 is rejected **before** checking agent status, grant existence, or any database query.
This ensures A5 cannot bypass security even if a grant or agent record claims A5.

## 2. PolicyEvaluator (13-Check Chain)

### Check Order (Exact)

| # | Check | Failure Outcome | Reason |
|---|-------|-----------------|--------|
| 0 | A5 pre-check | `blocked_policy` | `a5_disabled` |
| 1 | Agent active | `denied` | `agent_not_found` / `agent_disabled` |
| 2 | Run active | `denied` | `run_not_active` |
| 3 | Tool registered | `denied` | `tool_not_registered` |
| 4 | Grant active | `blocked_grant` | `no_active_grant` / `grant_expired` |
| 5 | Scope match | — | Always passes (v1.0 per-tool) |
| 6 | Autonomy ≤ agent max | `blocked_policy` | `autonomy_exceeds_agent` |
| 7 | Autonomy ≤ grant max | `blocked_policy` | `autonomy_exceeds_grant` |
| 8 | Risk allowed | `blocked_risk` | `risk_exceeds_autonomy` |
| 9 | Approval satisfied | `blocked_approval_required` | Multiple sub-reasons |
| 10 | MFA satisfied | `blocked_mfa_required` | `mfa_required` |
| 11 | Team permission | `denied` | `permission_denied` |
| 12 | Idempotency key | `denied` | `idempotency_key_required` |
| 13 | Target ownership | `denied` | `target_not_in_team` |

### Decision Outcomes
```
allowed          → All 13 checks passed, execute
denied           → Hard denial (agent/run/tool/permission/idempotency)
blocked_approval → Needs approval (check 9)
blocked_mfa      → Needs MFA (check 10)
blocked_policy   → Autonomy/policy violation (checks 0, 6, 7)
blocked_risk     → Risk exceeds autonomy (check 8)
blocked_grant    → No active grant (check 4)
blocked_scope    → Scope mismatch (check 5, unused in v1.0)
executed         → Already executed (idempotent replay)
failed           → Execution attempted but failed
```

## 3. Approval Verification (10 Sub-Checks)

When check 9 requires an approval, `verifyApproval()` runs:

| # | Guardrail | Failure Reason |
|---|-----------|---------------|
| 1 | `action_type` matches tool | `approval_action_type_mismatch` |
| 2 | `action_target` deep-equal match | `approval_target_mismatch` |
| 3 | Not expired | `approval_expired` |
| 4 | Status is `approved` | `approval_not_approved` |
| 5 | Same team | `approval_wrong_team` |
| 6 | `executed_at` IS NULL | `approval_already_executed` |
| 7 | (Marked executed in same tx as effect) | — |
| 8 | Denied payloads sanitized | — |
| 9 | Unknown tools denied before grant lookup | — |
| 10 | A5 fails before any DB lookup | — |

### Target Matching
```go
// Uses parsed JSON deep-equality, not raw byte comparison
// This handles JSON key ordering differences
func matchActionTarget(stored, requested json.RawMessage) bool {
    var s, r interface{}
    json.Unmarshal(stored, &s)
    json.Unmarshal(requested, &r)
    return reflect.DeepEqual(s, r)
}
```

## 4. Tool Gateway

### Registration
Tools must be registered in `tool_registry` table:
```sql
CREATE TABLE tool_registry (
    id UUID PRIMARY KEY,
    tool_name TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    risk_level TEXT NOT NULL DEFAULT 'low',
    requires_approval BOOLEAN DEFAULT false,
    requires_mfa BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at, updated_at
);
```

### Grants
Agents must have an active grant for each tool:
```sql
CREATE TABLE agent_tool_grants (
    id UUID PRIMARY KEY,
    agent_id UUID REFERENCES agent_identities(id),
    tool_name TEXT,
    team_id UUID,
    max_autonomy_level TEXT CHECK (max_autonomy_level IN ('A0','A1','A2','A3','A4')),
    requires_approval BOOLEAN DEFAULT false,
    requires_mfa BOOLEAN DEFAULT false,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at
);
```

### Key Constraint
- Grant `max_autonomy_level` has a CHECK constraint: only A0-A4 (A5 excluded at DB level)
- Agent `max_autonomy` also CHECK-constrained to A0-A4

## 5. Reasoning Worker Isolation

### Architecture
```
┌─────────────────────┐         HTTP only          ┌──────────────┐
│ Python Worker       │ ──────────────────────────→ │ Go API       │
│                     │   GET /api/auth/me          │ :8765        │
│ • Polls agent runs  │   GET /api/teams/{id}/...   │              │
│ • Generates intents │   POST /api/teams/{id}/...  │ All mutations│
│ • POSTs intentions  │                             │ happen here  │
│                     │                             └──────────────┘
│ NO: DATABASE_URL    │
│ NO: NATS_URL        │
│ NO: REDIS_URL       │
│ NO: MINIO_ENDPOINT  │
│ NO: Proxmox access  │
└─────────────────────┘
```

### Startup Validation
```python
# worker.py — fails closed if forbidden env vars present
for forbidden in ("DATABASE_URL", "NATS_URL", "REDIS_URL", "MINIO_ENDPOINT"):
    if os.environ.get(forbidden):
        log.error("Reasoning worker must NOT have %s — violates isolation boundary", forbidden)
        sys.exit(1)
```

### Communication Pattern
1. Worker authenticates with `WORKER_TOKEN` (JWT)
2. Polls `GET /api/teams/{teamId}/agent-runs`
3. For each pending run, generates an intention via ModelGateway
4. POSTs intention to `/api/teams/{teamId}/agent-runs/{runId}/intentions`
5. The Tool Gateway (Go) evaluates the intention through the 13-check chain
6. Worker NEVER directly mutates any infrastructure

### Model Gateway Constraints
- `chain_of_thought` always rejected/stripped
- `reasoning_summary` is the ONLY narrative output
- Output validated against `IntentionShape` schema
- Invalid intentions are rejected (not retried)

## 6. Remediation Agent Isolation

### System Agent Creation
- Remediation step execution auto-creates a `remediation-executor` system agent
- Max autonomy: A4 (never A5)
- Tool grants must be explicitly added for specific tools

### Step Execution Flow
```
Remediation Step (approved)
    ↓
Creates ToolRequest with:
  - agent_id: remediation-executor
  - run_id: (auto-created)
  - autonomy_level: A4
  - tool_name: step.tool_name
  - approval_id: remediation's approval
  - action_target: step.parameters (sanitized)
    ↓
PolicyEvaluator.Evaluate() — full 13-check chain
    ↓
ExecuteTool → records effect result
```

## 7. Audit Trail

Every agent action produces:

| Record | Table | Content |
|--------|-------|---------|
| Intention | `agent_intentions` | Tool name, autonomy, parameters (sanitized) |
| Effect result | `agent_effect_results` | Status (succeeded/denied/blocked/failed), reason |
| Audit log | `audit_logs` | Action, entity, sanitized payload |
| Outbox event | `outbox_events` | Event type, sanitized payload |

### Denied/Blocked Audit
Even denied/blocked actions produce audit + outbox records — full traceability for every attempted action.

## 8. Non-Negotiable Constraints (v1.0)

1. ❌ No unrestricted autonomy (A5 disabled)
2. ❌ No destructive mutation (delete/migrate/clone/firewall/network/storage/cert)
3. ✅ Allowed mutations only: start/shutdown/stop/snapshot
4. ✅ All high-risk requires: MFA + approval + policy + audit + outbox
5. ❌ No "act now review later" — approval before execution
6. ❌ Python workers never mutate directly
7. ✅ A5 disabled (hardcoded, fails before DB lookup)
