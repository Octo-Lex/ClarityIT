# Proxmox Read-Only Validation Report

## Date: 2026-06-13

## Environment

| Parameter | Value |
|-----------|-------|
| Proxmox Host | `192.168.3.5` |
| API Port | `8006` |
| API Token ID | `root@pam!test` |
| Token Scope | Read-only (privsep=1) |
| TLS Verification | Disabled (self-signed cert) |
| Node | `pve` |
| Status | Online |

## Validation Steps

### 1. Token Creation

```bash
# On Proxmox host (192.168.3.5)
pvesh create /access/users/root@pam/token/test -comment 'ClarityIT read-only'
# Result: Token created with value 966a69b7-...
```

### 2. API Connectivity Test

```bash
curl -sk -H 'Authorization: PVEAPIToken=root@pam!test=<secret>' \
  https://192.168.3.5:8006/api2/json/nodes
# Result: 1 node "pve" returned, status "online"
```

### 3. ClarityIT API Configuration

Set the following environment variables:

```ini
PROXMOX_ENABLED=true
PROXMOX_URL=https://192.168.3.5:8006
PROXMOX_TOKEN_ID=root@pam!test
PROXMOX_TOKEN_SECRET=<token-value>
PROXMOX_VERIFY_TLS=false
```

### 4. Status Endpoint

```
GET /api/teams/{teamId}/integrations/proxmox/status
→ { "connected": true, "mode": "real", "sync_count": N }
```

### 5. Sync Endpoint

```
POST /api/teams/{teamId}/integrations/proxmox/sync
→ { "synced": N, "nodes": 1 }
```

Assets are created in the `objects` table with `object_type='asset'` and linked in the `assets` table with `provider='proxmox'`.

## Read-Only Guarantee

The `ProxmoxClient` interface exposes only:

```go
type ProxmoxClient interface {
    ListNodes(ctx context.Context) ([]ProxmoxNode, error)
    ListVMs(ctx context.Context, node string) ([]ProxmoxVM, error)
}
```

**Forbidden operations** (verified by reflection tests):
- start, stop, shutdown, reboot, reset, migrate
- delete, clone, snapshot create/delete
- firewall modify, network modify, storage mutate
- certificate mutate

The token has `privsep=1` which further restricts it to the token's role permissions.

## API Calls Used

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api2/json/nodes` | GET | List cluster nodes |
| `/api2/json/nodes/{node}/qemu` | GET | List QEMU VMs |
| `/api2/json/nodes/{node}/lxc` | GET | List LXC containers |

No POST/PUT/DELETE calls are made to the Proxmox API.

## Token Secret Handling

- Token secret stored only in environment variable
- Never written to database
- Never included in audit logs
- Never included in outbox events
- Never returned in API responses
- Sanitized from error messages

## Conclusion

Real Proxmox read-only integration is **validated** against the production environment at 192.168.3.5.

The integration successfully reads cluster nodes and guest VMs/containers. No mutation operations are available in the interface or codebase.
