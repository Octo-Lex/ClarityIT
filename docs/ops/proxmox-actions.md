# Proxmox Actions — ClarityIT v1.0 Operations Guide

## Overview

ClarityIT integrates with Proxmox VE for infrastructure visibility and controlled mutations. This guide covers read-only sync, mutation requests, and the approval/execution pipeline.

## Configuration

### Environment Variables
```ini
PROXMOX_ENABLED=true
PROXMOX_URL=https://192.168.3.5:8006
PROXMOX_TOKEN_ID=root@pam!test
PROXMOX_TOKEN_SECRET=<your-token-secret>
PROXMOX_VERIFY_TLS=false
PROXMOX_MUTATION_ENABLED=false   # Default: off. Set true only when operating.
```

### Feature Flag
`PROXMOX_MUTATION_ENABLED` controls whether mutation execution is allowed:
- `false` (default): All execute requests return 403 "Proxmox mutation is not enabled"
- `true`: Mutations can be executed after passing all guardrails

**Keep this `false` unless you are actively performing infrastructure operations.**

## Read-Only Operations

### Sync Assets
```bash
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/integrations/proxmox/sync \
  -H "Authorization: Bearer $TOKEN"
```
Pulls nodes, VMs, and CTs from Proxmox into ClarityIT assets.

### List Assets
```bash
curl http://192.168.3.20:8765/api/teams/$TEAM/assets \
  -H "Authorization: Bearer $TOKEN"
```

### Check Status
```bash
curl http://192.168.3.20:8765/api/teams/$TEAM/integrations/proxmox/status \
  -H "Authorization: Bearer $TOKEN"
```

## Mutation Actions

### Allowed Actions

| Action | Risk | MFA | Min Approvers | Description |
|--------|------|-----|---------------|-------------|
| `proxmox.start` | Medium | No | 1 | Power on a stopped VM/CT |
| `proxmox.snapshot` | Medium | No | 1 | Create a point-in-time snapshot |
| `proxmox.shutdown` | High | Yes | 1 | Graceful shutdown (ACPI) |
| `proxmox.stop` | Critical | Yes | 2 | Force power-off (data loss risk) |

### Forbidden Actions (Not Implemented)
- delete, migrate, clone, reset
- firewall modify, network modify, storage mutate
- certificate mutate, host-level mutation, bulk mutation

### Requesting an Action

#### Via Web UI
1. Navigate to the asset detail page or `/assets/{id}/actions`
2. Click the action button (Start, Snapshot, Shutdown, Stop)
3. For snapshots, enter a name (alphanumeric, hyphens, underscores; max 40 chars)
4. The system creates an approval request automatically

#### Via API
```bash
# Create action + approval request
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/assets/$ASSET_ID/actions/proxmox/snapshot \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"snapshot_name": "pre-patch-2026-06-14"}'
# Returns: { "id": "...", "approval_id": "...", "status": "pending" }
```

### Approving an Action

Medium/High risk:
```bash
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/approvals/$APPROVAL_ID/approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"reason": "Approved for maintenance window"}'
```

Critical (stop) requires 2 distinct approvers — have a second operator repeat the approve call.

For high/critical risk, the approver must have recent MFA (within 5 minutes).

### Executing an Action

```bash
# Ensure PROXMOX_MUTATION_ENABLED=true on the API
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/assets/asset-actions/$ACTION_ID/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)"
# Returns: { "status": "succeeded", "proxmox_task_id": "UPID:..." }
```

### Monitoring Actions

```bash
# List all actions
curl http://192.168.3.20:8765/api/teams/$TEAM/assets/asset-actions \
  -H "Authorization: Bearer $TOKEN"

# Get specific action detail
curl http://192.168.3.20:8765/api/teams/$TEAM/assets/asset-actions/$ACTION_ID \
  -H "Authorization: Bearer $TOKEN"
```

### Viewing in UI
Navigate to `/admin/asset-actions` for the action history table.

## Guardrail Summary

1. Asset must belong to requesting team
2. Asset provider must be `proxmox`
3. Asset must have valid `pve:{node}:{vmid}` external_id
4. `PROXMOX_MUTATION_ENABLED` must be `true`
5. Approval must be linked, approved, same team, not expired, not executed
6. MFA required for high/critical
7. Stop requires 2 distinct approvers
8. Idempotency-Key required for execution
9. Snapshot name regex validated
10. All actions recorded in audit_logs + outbox_events

## Post-Operation

After completing infrastructure operations:

1. Set `PROXMOX_MUTATION_ENABLED=false` in the API environment
2. Restart the API container: `docker compose restart clarityit-api`
3. Verify: attempt an execute and confirm 403 response

## Troubleshooting

| Error | Cause | Resolution |
|-------|-------|------------|
| "Proxmox mutation is not enabled" | Feature flag off | Set `PROXMOX_MUTATION_ENABLED=true` and restart API |
| "Approval not approved" | Approval still pending | Approve the request first |
| "Recent MFA required" | MFA window expired | Complete MFA challenge, then execute |
| "requires 2 distinct approvers" | Stop with <2 approvers | Have a second operator approve |
| "Asset not found in team" | Wrong team or asset ID | Verify asset belongs to your team |
| "Asset is not a Proxmox-managed resource" | Provider mismatch | Ensure asset was synced from Proxmox |
| "Invalid snapshot name" | Bad characters | Use only alphanumeric, hyphens, underscores (max 40) |
