# ClarityIT v1.0 High-Risk Actions

## Document Status
- **Version**: 1.0.0
- **Date**: 2026-06-14
- **Scope**: All actions classified as medium, high, or critical risk

---

## 1. Risk Classification

| Level | Definition | Examples |
|-------|-----------|----------|
| **Low** | No infrastructure impact; reversible | Add comment, create work item, link objects |
| **Medium** | Infrastructure impact; reversible | Start VM, create snapshot |
| **High** | Infrastructure impact; may cause downtime | Shutdown VM |
| **Critical** | Infrastructure impact; may cause data loss | Stop VM (force power-off) |

## 2. Risk-to-Requirements Matrix

| Risk Level | Auto-Execute (A0-A3) | A4 Execution | Approval | MFA | Min Approvers |
|-----------|---------------------|-------------|----------|-----|---------------|
| Low | ✅ Yes | ✅ Yes | ❌ No | ❌ No | 0 |
| Medium | ❌ Blocked | ✅ With approval | ✅ Required | ❌ No | 1 |
| High | ❌ Blocked | ✅ With approval | ✅ Required | ✅ Required | 1 |
| Critical | ❌ Blocked | ✅ With approval | ✅ Required | ✅ Required | 2 |

### Enforcement Chain (13 checks, exact order)
1. **Pre-check**: A5 autonomy → hardcoded rejection (before any DB lookup)
2. Agent is active
3. Agent run is active (pending or running)
4. Tool is registered in `tool_registry`
5. Tool grant exists and is active (not revoked, not expired)
6. Grant scope matches (per-tool for v1.0)
7. Requested autonomy ≤ agent max_autonomy
8. Requested autonomy ≤ grant max_autonomy
9. Risk level allowed (A0-A3 blocked for high/critical; A4 can execute any risk)
10. Approval satisfied (for medium+ risk): status=approved, not expired, action_type match, action_target match, same team, not already executed
11. MFA satisfied (for high+ risk): `recent_mfa_at` within 5 minutes
12. Team permission: `agents.tools.execute` verified
13. Target object belongs to team

## 3. Proxmox Mutation Actions

| Action | Risk | MFA | Approval | Min Approvers | Feature Flag |
|--------|------|-----|----------|---------------|-------------|
| `proxmox.start` | Medium | ❌ | ✅ | 1 | `PROXMOX_MUTATION_ENABLED` |
| `proxmox.snapshot` | Medium | ❌ | ✅ | 1 | `PROXMOX_MUTATION_ENABLED` |
| `proxmox.shutdown` | High | ✅ | ✅ | 1 | `PROXMOX_MUTATION_ENABLED` |
| `proxmox.stop` | Critical | ✅ | ✅ | 2 | `PROXMOX_MUTATION_ENABLED` |

### Forbidden Actions (Not Implemented)
The following actions have **no route, no handler, no client method**:

- `proxmox.delete` — VM/CT deletion
- `proxmox.migrate` — VM migration between nodes
- `proxmox.clone` — VM cloning
- `proxmox.reset` — VM reset
- `proxmox.firewall_modify` — Firewall rule changes
- `proxmox.network_modify` — Network configuration changes
- `proxmox.storage_mutate` — Storage pool changes
- `proxmox.certificate_mutate` — Certificate management
- `proxmox.host_level_mutation` — Host-level changes (apt, systemd, etc.)
- `proxmox.bulk_mutation` — Bulk operations

### Guardrails
1. Asset must belong to the same team (cross-team blocked)
2. Asset provider must be `proxmox`
3. Asset must include node + vmid + vm_type metadata
4. Unknown action types return 404 (never soft-allow)
5. Snapshot name validated with regex `^[a-zA-Z0-9_-]{1,40}$` (shell injection prevention)
6. Mutation disabled by default (`PROXMOX_MUTATION_ENABLED=false`)
7. Idempotency key required for execution
8. Approval verified via 10-point check before mutation
9. MFA required for high/critical
10. Stop requires 2 distinct approvers (enforced by `approval_decisions` count)
11. Success/failure recorded with sanitized audit + outbox
12. Approval marked `executed` in same transaction as effect result

## 4. Agent Tool Execution

### Tool Gateway Risk Policy
| Tool Risk | Requires Approval | Requires MFA | A0-A3 | A4 |
|-----------|-------------------|-------------|-------|-----|
| Low | If tool_registry flag set | If tool_registry flag set | ✅ | ✅ |
| Medium | ✅ | ❌ | ❌ Blocked | ✅ With approval |
| High | ✅ | ✅ | ❌ Blocked | ✅ With approval + MFA |
| Critical | ✅ | ✅ | ❌ Blocked | ✅ With approval + MFA |

### Agent Autonomy Ladder
| Level | Name | Can Do |
|-------|------|--------|
| A0 | Observant | Read only, no tool execution |
| A1 | Advisory | Propose actions, no execution |
| A2 | Guided | Execute low-risk with approval |
| A3 | Supervised | Execute low/medium with approval |
| A4 | Operational | Execute any risk with approval + MFA |
| A5 | **DISABLED** | **Hardcoded rejection** — never allowed in v1.0 |

## 5. Approval Workflow

### Approval States
```
pending → {approved, rejected, cancelled, expired} → {executed, failed}
```

### Risk-Based Policy
| Risk | Auto-Approve | Approval | MFA | Min Approvers |
|------|-------------|----------|-----|---------------|
| Low | ✅ Yes (self-approve) | — | — | 0 |
| Medium | ❌ | ✅ Required | ❌ | 1 |
| High | ❌ | ✅ Required | ✅ Required | 1 |
| Critical | ❌ | ✅ Required | ✅ Required | 2 |

### Approval Verification (10 Guardrails)
1. `action_type` matches requested tool
2. `action_target` deep-equality matches requested target
3. Not expired (`expires_at` > NOW)
4. Status is `approved`
5. Same team (`team_id` match)
6. Not already executed (`executed_at` IS NULL)
7. (Action handler marks `executed_at` in same tx as effect)
8. Denied payloads sanitized
9. Unknown tools denied before grant lookup
10. A5 fails closed before any DB lookup

## 6. Remediation Workflow

### Proposal States
```
draft → proposed → approved → executing → {completed, failed, cancelled}
```

- Agent-created proposals are always `draft` (operators must promote)
- Steps execute through Tool Gateway (full 13-check chain)
- `continue_on_failure` flag allows partial success
- High/critical self-approval blocked
- Cancelled proposals return 409 on subsequent actions
- Completed proposals are idempotent (return same result)

## 7. Sensitive Data Sanitization

### Sanitized Before Audit/Outbox/Storage
These key patterns are redacted to `[REDACTED]` before any persistence:

```
token, secret, password, key, credential, api_key, auth, authorization,
webhook_secret, signing_secret, mfa_code, recovery_code, otp,
proxmox_token, proxmox_secret
```

### Implementation
- **Approval handler**: `sanitizeActionTarget()` — 15 sensitive key patterns
- **Remediation handler**: `sanitizeParameters()` — regex match on 11 patterns, recursive
- **Proxmox action handler**: `sanitizeForLog()` — strips token=/secret= from error messages
- **Outbox worker**: 5-field whitelist for Redis WS fanout (entity_id, entity_type, action, team_id, timestamp)
- **Audit writer**: IP/user-agent stored as HMAC-SHA256, never raw
- **MFA handler**: TOTP secret encrypted at rest; recovery codes HMAC-SHA256 hashed; raw values never in audit/outbox/events

## 8. Idempotency Requirements

All mutation endpoints require `Idempotency-Key` header:
- Prevents duplicate execution on network retry
- Replay returns same response
- JSON comparison uses parsed values (not raw bytes) to handle key ordering

### Exempt Endpoints
- `POST /api/auth/login` — inherently idempotent (returns new tokens)
- `POST /api/auth/refresh` — inherently idempotent (rotates tokens)
