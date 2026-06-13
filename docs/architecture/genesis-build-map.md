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
