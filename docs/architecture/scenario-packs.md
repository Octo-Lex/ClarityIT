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
