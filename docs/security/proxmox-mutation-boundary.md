# ClarityIT v1.0 Proxmox Mutation Boundary

## Document Status
- **Version**: 1.0.0
- **Date**: 2026-06-14
- **Scope**: Proxmox VE integration — read-only operations and controlled mutation pipeline

---

## 1. Architecture

```
┌──────────────┐     HTTP API      ┌──────────────┐
│ ClarityIT    │ ─────────────────→│ Proxmox VE   │
│ API :8765    │  Token: root@pam  │ 192.168.3.5  │
│              │                   │ :8006         │
│ [Read-Only]  │                   │               │
│ [Mutation]   │                   │               │
└──────────────┘                   └──────────────┘
```

### ProxmoxClient Interface Methods

| Method | Purpose | Risk | Mutation? |
|--------|---------|------|-----------|
| `GetNodes()` | List cluster nodes | Low | No |
| `GetNodeStatus(node)` | Node details | Low | No |
| `GetVMs(node)` | List VMs/CTs on node | Low | No |
| `GetVMStatus(node, vmid)` | VM details | Low | No |
| `GetStorage(node)` | Storage pools | Low | No |
| `GetTasks(node, limit)` | Task history | Low | No |
| `GetTaskStatus(node, upid)` | Task result | Low | No |
| `StartVM(target)` | Start VM/CT | Medium | **Yes** |
| `ShutdownVM(target)` | Graceful shutdown | High | **Yes** |
| `StopVM(target)` | Force power-off | Critical | **Yes** |
| `SnapshotVM(target, name)` | Create snapshot | Medium | **Yes** |

### MutationTarget
```go
type MutationTarget struct {
    Node   string // Proxmox node name
    VMID   int    // VM/CT ID
    VMType string // "qemu" or "lxc"
}
```

Parsed from asset `external_id` format: `pve:{node}:{vmid}`

## 2. Allowed Mutation Routes

### Route Inventory (Exact)

```
POST /api/teams/{teamId}/assets/{assetId}/actions/proxmox/start
POST /api/teams/{teamId}/assets/{assetId}/actions/proxmox/shutdown
POST /api/teams/{teamId}/assets/{assetId}/actions/proxmox/stop
POST /api/teams/{teamId}/assets/{assetId}/actions/proxmox/snapshot
GET  /api/teams/{teamId}/assets/asset-actions
GET  /api/teams/{teamId}/assets/asset-actions/{actionId}
POST /api/teams/{teamId}/assets/asset-actions/{actionId}/execute
```

### Risk and Requirements Per Action

| Action | Risk Level | MFA | Approval | Min Approvers | Feature Flag |
|--------|-----------|-----|----------|---------------|-------------|
| `proxmox.start` | medium | ❌ | ✅ | 1 | Required |
| `proxmox.snapshot` | medium | ❌ | ✅ | 1 | Required |
| `proxmox.shutdown` | high | ✅ | ✅ | 1 | Required |
| `proxmox.stop` | critical | ✅ | ✅ | 2 | Required |

### Feature Flag
- Environment variable: `PROXMOX_MUTATION_ENABLED`
- Default: `false` (mutation disabled)
- When `false`: `ExecuteAction` returns 403 "Proxmox mutation is not enabled"
- Read-only operations (sync, status) work regardless of flag

## 3. Forbidden Actions (Absent From Codebase)

The following actions have **no route definition, no handler method, no client method, and no code path**:

| Action | Status | Verification |
|--------|--------|-------------|
| `proxmox.delete` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.migrate` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.clone` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.reset` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.firewall_modify` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.network_modify` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.storage_mutate` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.certificate_mutate` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.host_level_mutation` | **ABSENT** | No route, no client method, no switch case |
| `proxmox.bulk_mutation` | **ABSENT** | No route, no client method, no switch case |

### Enforcement Mechanism
```go
// action_handler.go — allowedActions map
var allowedActions = map[string]struct{...}{
    "proxmox.start":    {riskLevel: "medium", ...},
    "proxmox.snapshot": {riskLevel: "medium", ...},
    "proxmox.shutdown": {riskLevel: "high", ...},
    "proxmox.stop":     {riskLevel: "critical", ...},
}
// Any action NOT in this map → 404 "Action not found"
```

The switch statement in `ExecuteAction` only handles the 4 allowed actions:
```go
switch actionType {
case "proxmox.start":    → h.client.StartVM(ctx, target)
case "proxmox.shutdown": → h.client.ShutdownVM(ctx, target)
case "proxmox.stop":     → h.client.StopVM(ctx, target)
case "proxmox.snapshot": → h.client.SnapshotVM(ctx, target, snapName)
// No default case → falls through, no mutation occurs
}
```

## 4. Execution Guardrails (16 Total)

1. **Asset team ownership**: Asset must belong to requesting team
2. **Provider check**: Asset provider must be `proxmox`
3. **Metadata check**: Asset must have valid `pve:{node}:{vmid}` external_id
4. **Feature flag**: `PROXMOX_MUTATION_ENABLED` must be `true`
5. **Approval linked**: Action must have an approval_id
6. **Approval cross-team**: Approval team_id must match action team_id
7. **Approval action_type**: Must match the requested action
8. **Approval executed_at**: Must be NULL (not already used)
9. **Approval status**: Must be `approved`
10. **Approval expiry**: Must not be expired
11. **MFA**: Required for high (shutdown) and critical (stop)
12. **Min approvers**: Stop requires 2 distinct approvers
13. **Idempotency**: Idempotency-Key header required for execution
14. **Idempotent replay**: Already-succeeded actions return cached result
15. **Snapshot name**: Validated with regex `^[a-zA-Z0-9_-]{1,40}$`
16. **Sanitized audit/outbox**: No secrets in recorded payloads

## 5. Audit and Event Trail

### Audit Events
| Event | When | Payload |
|-------|------|---------|
| `asset.action.requested` | CreateAction | action_type, asset_id, risk_level, hostname |
| `asset.action.executed` | ExecuteAction success | action_type, task_id, hostname, approval_id |
| `asset.action.blocked` | ExecuteAction denied | action_id, reason |
| `asset.action.failed` | ExecuteAction error | action_type, sanitized error |

### Outbox Events
| Event Type | Subject |
|-----------|---------|
| `clarity.v1.asset.action.requested` | Published on creation |
| `clarity.v1.asset.action.executed` | Published on success |
| `clarity.v1.asset.action.blocked` | Published on denial |
| `clarity.v1.asset.action.failed` | Published on failure |

### Payload Content
All payloads contain only identity information:
- `action_type`, `asset_id`, `hostname`, `risk_level`, `task_id`, `approval_id`
- **Never**: Proxmox token, API secret, node credentials, raw snapshot contents

## 6. Client Implementation

### Real Client (`real_client.go`)
- Uses Proxmox API token (`root@pam!test`)
- HTTPS with configurable TLS verification
- Start: `POST /api2/json/nodes/{node}/{vmtype}/{vmid}/status/start`
- Shutdown: `POST /api2/json/nodes/{node}/{vmtype}/{vmid}/status/shutdown`
- Stop: `POST /api2/json/nodes/{node}/{vmtype}/{vmid}/status/stop`
- Snapshot: `POST /api2/json/nodes/{node}/{vmtype}/{vmid}/snapshot`

### Fake Client (Testing)
- Returns deterministic task UPIDs
- No network calls
- Used in all unit/integration tests

## 7. Validation Status

### Track 7 (Security Review)
- **`PROXMOX_MUTATION_ENABLED=false`** by default — mutation pipeline code exists but cannot execute without explicit operator enablement
- Real Proxmox mutation client implemented with full API call paths for start/shutdown/stop/snapshot
- All 16 guardrails verified via unit tests using FakeProxmoxClient (deterministic task UPIDs, no network calls)
- Mutation pipeline NOT live-exercised against real Proxmox in Track 7

### Track 8 (Release Verification) — DEFERRED
- Live Proxmox mutation validation against real PVE cluster
- End-to-end: create asset action → approve → execute → verify VM state change
- Task status polling verification
- Feature flag enable/disable cycle
- This deferral is acceptable because the feature is disabled by default and fully tested with fakes

## 8. UI Layer

### Frontend Routes
- `/admin/asset-actions` — Action history table (AdminAssetActions)
- `/assets/:id/actions` — Action request buttons (AssetActions)
- Risk indicators on each button (medium/high/critical color coding)
- Stop button displays "Critical • 2 approvers + MFA required" warning
- Snapshot name client-side validated (same regex as backend)

### API Client Methods
```typescript
createAssetAction(assetId, action, snapshotName?)  // POST mutation
listAssetActions(status?)                          // GET list
getAssetAction(id)                                 // GET detail
executeAssetAction(id)                             // POST execute
```
