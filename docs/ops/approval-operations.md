# Approval Operations — ClarityIT v1.0

## Overview

The approval workflow gates medium, high, and critical risk actions behind human review. Low-risk actions auto-approve per policy.

## Risk-Based Policy

| Risk | Auto-Approve | Requires Approval | Requires MFA | Min Approvers | TTL |
|------|-------------|-------------------|-------------|---------------|-----|
| Low | ✅ Yes | No | No | 0 | 1 hour |
| Medium | No | Yes | No | 1 | 1 hour |
| High | No | Yes | Yes (5 min) | 1 | 1 hour |
| Critical | No | Yes | Yes (5 min) | 2 | 1 hour |

## Approval States

```
pending → {approved, rejected, cancelled, expired} → {executed, failed}
```

- **pending**: Created, awaiting decision
- **approved**: Approved, ready for execution
- **rejected**: Denied, terminal
- **cancelled**: Withdrawn by requester, terminal
- **expired**: TTL elapsed, terminal
- **executed**: Action completed successfully
- **failed**: Execution attempted but errored

## Creating Approval Requests

### Via Proxmox Action (Automatic)
When an operator requests a Proxmox action (start/shutdown/stop/snapshot), an approval request is created automatically.

### Manual Creation
```bash
TOKEN=<your_access_token>
TEAM=<team_id>

curl -X POST http://<your-host>:8765/api/teams/$TEAM/approvals \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "action_type": "custom.action",
    "risk_level": "medium",
    "description": "Manual approval request",
    "action_target": {"key": "value"}
  }'
```

## Approving/Rejecting

### Approve
```bash
curl -X POST http://<your-host>:8765/api/teams/$TEAM/approvals/$APPROVAL_ID/approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"reason": "Approved by operator"}'
```

### Reject
```bash
curl -X POST http://<your-host>:8765/api/teams/$TEAM/approvals/$APPROVAL_ID/reject \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"reason": "Risk too high"}'
```

## Rules

1. **Self-approval prevented**: You cannot approve your own request (except low-risk auto-approve)
2. **Immutable decisions**: Once approved/rejected, the decision cannot be changed
3. **One decision per user**: UNIQUE(approval_id, decided_by) — each user decides once
4. **Critical risk**: Requires 2 distinct approvers
5. **High/critical**: Requires recent MFA (within 5 minutes) to approve
6. **Expiry**: Approvals expire after 1 hour
7. **Single execution**: Once executed, cannot be reused

## Timeout Behavior

- **Approval TTL**: 1 hour from creation. After expiry, status → `expired`, cannot be executed.
- **MFA window**: 5 minutes from `recent_mfa_at`. If MFA expires before execution, the action is blocked with `mfa_required`.
- **No retry escalation**: Expired approvals do not auto-escalate. A new request must be created.

## Monitoring

### List Pending Approvals (API)
```bash
curl http://<your-host>:8765/api/teams/$TEAM/approvals?status=pending \
  -H "Authorization: Bearer $TOKEN"
```

### View in UI
Navigate to `/admin/approvals` in the web interface.

### Audit Trail
All approval events (created, approved, rejected, cancelled, executed, failed) are recorded in:
- `audit_logs` table (sanitized payload)
- `outbox_events` table (event type `clarity.v1.approval.*`)

## Troubleshooting

| Problem | Cause | Resolution |
|---------|-------|------------|
| "cannot self-approve" | Requested by same user | Have another team member approve |
| "Recent MFA required" | MFA window expired | Complete MFA challenge, then approve |
| "requires 2 distinct approvers" | Critical risk, only 1 approver | Have a second team member approve |
| "already executed" | Approval was used | Create a new approval request |
| "expired" | TTL elapsed (>1 hour) | Create a new approval request |
