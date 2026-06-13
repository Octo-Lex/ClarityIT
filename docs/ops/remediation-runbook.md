# Remediation Runbook — ClarityIT v1.0

## Overview

Remediation proposals are structured, multi-step action plans that execute through the Tool Gateway with full policy enforcement. Agents propose; operators decide.

## Proposal Lifecycle

```
draft → proposed → approved → executing → {completed, failed, cancelled}
```

| State | Description |
|-------|-------------|
| draft | Agent-created (operators must promote to proposed) |
| proposed | Ready for operator review |
| approved | Operator approved, ready for execution |
| executing | Steps are being executed through Tool Gateway |
| completed | All steps succeeded (or non-critical failures with `continue_on_failure`) |
| failed | Execution failed (critical step error without `continue_on_failure`) |
| cancelled | Withdrawn by operator |

## Creating Remediation Proposals

### By Operator
```bash
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/remediations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "title": "Restart web service on host-01",
    "description": "CPU spike detected, service restart recommended",
    "source": "operator",
    "risk_level": "medium",
    "steps": [
      {
        "step_order": 1,
        "tool_name": "objects.add_comment",
        "risk_level": "low",
        "parameters": {"object_id": "inc-123", "body": "Starting remediation"}
      },
      {
        "step_order": 2,
        "tool_name": "proxmox.start",
        "risk_level": "medium",
        "parameters": {"vmid": "100", "node": "pve"}
      }
    ]
  }'
```

### By Agent
Agent-created proposals are always `draft` status. An operator must review and approve before execution.

```json
{
  "source": "agent",
  "agent_run_id": "<active-run-id>",
  "title": "Auto-remediation: disk space cleanup",
  "risk_level": "low",
  "steps": [...]
}
```

## Approving

```bash
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/remediations/$REMEDIATION_ID/approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)"
```

Rules:
- High/critical risk: self-approval blocked
- Agent-sourced proposals: must be promoted from draft to proposed first

## Executing

```bash
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/remediations/$REMEDIATION_ID/execute \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)"
```

### What Happens During Execution
1. Each step is processed sequentially
2. A "remediation-executor" system agent (A4 max autonomy) is auto-created per team
3. Each step goes through the full 13-check PolicyEvaluator chain
4. If a step fails:
   - `continue_on_failure=true`: next step proceeds, step marked `failed`
   - `continue_on_failure=false`: execution stops, proposal marked `failed`
5. All step results recorded with status, output, and error messages
6. Sensitive parameters are redacted in all stored/audited data

### Idempotent Replay
If you call execute on a `completed` proposal, the same result is returned without re-execution.

## Cancelling

```bash
curl -X POST http://192.168.3.20:8765/api/teams/$TEAM/remediations/$REMEDIATION_ID/cancel \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $(uuidgen)"
```

Cancelled proposals cannot be executed (returns 409).

## Viewing in UI

Navigate to `/incidents/{id}/remediation` for incident-scoped proposals, or use the API to list all:

```bash
curl http://192.168.3.20:8765/api/teams/$TEAM/remediations \
  -H "Authorization: Bearer $TOKEN"
```

## Step Execution Safety

Every step passes through the Tool Gateway's 13-check enforcement chain:

1. A5 rejected (hardcoded)
2. Agent active
3. Run active
4. Tool registered
5. Grant active
6. Scope match
7. Autonomy ≤ agent max
8. Autonomy ≤ grant max
9. Risk level allowed
10. Approval satisfied (action_type, target, expiry, team, not-executed)
11. MFA satisfied (for high/critical)
12. Team permission verified
13. Target belongs to team

If any check fails, the step is blocked and the decision reason is recorded.

## Troubleshooting

| Error | Cause | Resolution |
|-------|-------|------------|
| "unknown tool" | Step tool_name not in tool_registry | Register the tool or use an existing one |
| "cannot execute draft" | Proposal still in draft | Approve first (promotes to proposed→approved) |
| "already completed" | Proposal was executed | Idempotent replay returns same result |
| "remediation cancelled" | Cancelled before execution | Create a new proposal |
| "cross-team incident" | Incident belongs to different team | Use an incident from your team |
| "cross-team agent_run" | Agent run belongs to different team | Use an agent run from your team |
| Step "blocked_approval_required" | Step risk requires approval | Ensure the remediation's approval covers the step's tool+target |
| Step "blocked_mfa_required" | High/critical step without recent MFA | Complete MFA challenge before executing |
