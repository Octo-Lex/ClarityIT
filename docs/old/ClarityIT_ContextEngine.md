# Clarity IT — The Context Engine
## The Missing Piece for Maximum Impact (2026)

> **Core Insight**: Context is the platform. Everything else — The Queue, The Project, The Wiki, The Hub, The Grid, The Agents — is just a view into it.

---

## 1. The Problem: Islands of Intelligence

We have designed six powerful modules and six specialized AI agents. But here is the critical gap:

**Every agent operates with partial information.**

| Agent | What It Knows | What It Is Missing |
|-------|--------------|-------------------|
| **Queue Agent** | Ticket content, SLA rules, past resolutions | Who is actually on call right now, what projects are in crunch mode, what was discussed in Slack about this exact issue last week |
| **Project Agent** | Tasks, dependencies, velocity | Which team members are overloaded with tickets, what infrastructure changes are scheduled, what risks were flagged in recent alerts |
| **Alert Agent** | Metrics, thresholds, alert history | Which project is affected, who owns the service, what runbook applies, what the team decided in the last incident review |
| **Doc Agent** | Document content, wiki structure | Recent Slack decisions that override the documented process, current ticket volume on related issues, recent code changes that make the doc outdated |
| **Ops Agent** | Calendar, tasks, meeting schedules | Real-time system health, active incidents, team availability, project deadlines |
| **Orchestrator** | Agent capabilities, task queues | The full picture of what the team is doing, thinking, and struggling with right now |

**The result**: Agents make "smart" decisions with blind spots. They suggest fixes for issues the team already decided to deprecate. They schedule work for people who are on vacation. They write docs that contradict recent Slack agreements.

This is not a feature gap. It is an architectural gap.

---

## 2. The Solution: The Context Engine

### 2.1 Definition

The **Context Engine** is a real-time, queryable, evolving knowledge graph that sits at the center of Clarity IT. It collects, correlates, and serves **context** — not just data — to every agent, every view, and every user.

**Context ≠ Data**
- **Data** is raw: "Ticket #4821 was created at 14:30"
- **Context** is meaningful: "Ticket #4821 is a P1 about the auth service, created while the team is in sprint review, assigned to the on-call engineer who is already handling two P2s, and similar to an incident last month that took 4 hours to resolve"

### 2.2 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     CONTEXT ENGINE                              │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │   INGEST    │  │  CORRELATE  │  │   SERVE     │           │
│  │             │  │             │  │             │           │
│  │ • Events    │  │ • Entity    │  │ • Graph     │           │
│  │ • APIs      │  │   Linking   │  │   Queries   │           │
│  │ • Webhooks  │  │ • Temporal  │  │ • Vector    │           │
│  │ • Polling   │  │   Analysis  │  │   Search    │           │
│  │ • Streams   │  │ • Semantic  │  │ • Context   │           │
│  │             │  │   Enrich    │  │   Bundles   │           │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘           │
│         │                │                │                   │
│         └────────────────┼────────────────┘                   │
│                          │                                      │
│                    ┌─────┴─────┐                                │
│                    │  GRAPH    │                                │
│                    │  STORE    │                                │
│                    │           │                                │
│                    │ • Nodes   │  Users, Tickets, Projects,     │
│                    │   (Entities) Docs, Alerts, VMs, Commits   │
│                    │ • Edges   │  created_by, blocks, relates_to│
│                    │   (Relations) mentions, caused, resolved_by│
│                    │ • Vectors │  Semantic embeddings for       │
│                    │           │  similarity search             │
│                    └───────────┘                                │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
   ┌─────────┐          ┌─────────┐          ┌─────────┐
   │  AGENTS │          │   VIEWS │          │  USERS  │
   │         │          │         │          │         │
   │ Queue   │          │ Kanban  │          │ "What   │
   │ Project │          │ Gantt   │          │  should │
   │ Alert   │◄────────►│ Wiki    │◄────────►│  I work │
   │ Doc     │  Context │ Hub     │  Context │  on     │
   │ Ops     │  Bundles │ Grid    │  Bundles │  now?"  │
   │         │          │         │          │         │
   └─────────┘          └─────────┘          └─────────┘
```

### 2.3 Context Types

The engine maintains six dimensions of context:

| Type | Description | Example |
|------|-------------|---------|
| **User Context** | Who is this person? What do they know? What are they working on? | "Alice is a senior SRE, on-call this week, owns the auth service, has 3 open tickets, last active 10 min ago" |
| **Team Context** | Who is available? Who is overloaded? What is the team's focus? | "Backend team is in crunch mode for v2.0 release. Frontend team has capacity. Two people on PTO." |
| **Project Context** | What is the goal? What is blocked? What changed recently? | "API v2 migration is 60% complete. Auth refactor is blocked by certificate renewal. Deploy scheduled for Friday." |
| **Time Context** | What is happening now? What is coming up? What happened recently? | "Sprint ends tomorrow. Quarterly review next week. Maintenance window in 2 hours. Last deploy caused 3 incidents." |
| **System Context** | What is the infrastructure state? What is healthy? What is failing? | "web-cluster: healthy, db-primary: 95% disk, cert-expiry: 14 days, backup: failed last night" |
| **Business Context** | What matters to the organization? What are the priorities? What are the constraints? | "Q3 goal: reduce MTTR by 30%. Budget: $50K remaining. Compliance audit in 6 weeks." |

---

## 3. How It Works

### 3.1 Ingestion Layer

Every module feeds the Context Engine in real-time:

```
The Queue    ──►  ticket.created, ticket.updated, ticket.resolved
The Project  ──►  task.moved, milestone.reached, sprint.started
The Wiki     ──►  doc.created, doc.edited, doc.commented
The Hub      ──►  message.sent, thread.replied, file.shared
The Grid     ──►  alert.triggered, vm.status_changed, backup.completed
The Agents   ──►  agent.action_taken, agent.decision_made, agent.escalated
External     ──►  github.push, gitlab.merge, prometheus.alert, proxmox.event
```

**Ingestion Methods**:
- **Event Mesh**: NATS/Redis topics → auto-ingested
- **API Pushes**: Modules push context updates via REST
- **Webhooks**: External systems push events
- **Polling**: For systems without push capability
- **Streams**: Real-time log tailing, metric streaming

### 3.2 Correlation Layer

Raw events become context through correlation:

**Entity Linking**:
```
Event: ticket.created {id: 4821, title: "Auth service down", service: "auth-api"}
  ↓
Links to:
  - User: alice (on-call engineer)
  - Project: api-v2-migration (affects auth service)
  - Doc: runbook/auth-service-recovery
  - Alert: prometheus.auth-500-errors (firing since 14:25)
  - VM: vm-auth-prod-03 (CPU 95%)
  - Message: hub.#incidents "Anyone seeing auth issues?" (14:28)
  - Commit: github/auth-service "Updated JWT validation" (14:15)
```

**Temporal Analysis**:
```
Ticket 4821 created at 14:30
  → Commit pushed at 14:15 (15 min before)
  → Alert fired at 14:25 (5 min before)
  → Message in #incidents at 14:28 (2 min before)
  → Correlation score: 0.94 (very likely related)
```

**Semantic Enrichment**:
```
Ticket: "Users can't log in"
  → Semantic embedding → similar to:
    - Doc: "Auth Service Troubleshooting"
    - Ticket #3912: "Login failures after JWT update" (resolved)
    - Runbook: "auth-service-recovery"
    - Alert: "auth-500-errors"
```

### 3.3 The Context Graph

The engine stores everything as a graph:

**Nodes** (Entities):
```yaml
node_types:
  - user
  - ticket
  - project
  - task
  - doc
  - alert
  - vm
  - service
  - channel
  - message
  - commit
  - runbook
  - team
  - sprint
  - milestone
```

**Edges** (Relations):
```yaml
edge_types:
  - created_by (ticket → user)
  - assigned_to (ticket → user)
  - blocks (task → task)
  - depends_on (task → task)
  - relates_to (ticket → doc)
  - caused_by (alert → commit)
  - resolved_by (ticket → runbook)
  - mentions (message → user)
  - affects (alert → service)
  - owns (user → service)
  - part_of (task → project)
  - scheduled_for (maintenance → time)
```

### 3.4 Context Bundles

When an agent or view requests context, the engine returns a **Context Bundle** — a curated snapshot of everything relevant:

```json
{
  "bundle_id": "ctx-uuid",
  "timestamp": "2026-06-12T14:40:00Z",
  "for": "queue_agent",
  "query": "ticket:4821",
  "context": {
    "subject": {
      "ticket": { "id": 4821, "title": "Auth service down", "priority": "P1" }
    },
    "user_context": {
      "assignee": { "name": "Alice", "role": "SRE", "on_call": true, "load": "2 P2s + this P1" },
      "reporter": { "name": "Bob", "team": "Frontend", "affected": true }
    },
    "team_context": {
      "backend_team": { "status": "crunch", "focus": "v2.0 release" },
      "available_sres": ["Charlie"],
      "on_call_rotation": { "primary": "Alice", "secondary": "Charlie" }
    },
    "project_context": {
      "affected_projects": ["api-v2-migration"],
      "blocked_tasks": ["Deploy auth changes to prod"],
      "upcoming_milestones": [{ "name": "v2.0 deploy", "date": "2026-06-14" }]
    },
    "time_context": {
      "now": "14:40",
      "sprint_ends": "tomorrow",
      "maintenance_window": "in 2 hours",
      "business_hours": true
    },
    "system_context": {
      "affected_services": ["auth-api", "user-dashboard"],
      "alerts_firing": ["auth-500-errors", "db-connection-pool-exhausted"],
      "vm_status": { "vm-auth-prod-03": "CPU 95%, Disk 87%" },
      "recent_changes": ["JWT validation update deployed 14:15"]
    },
    "business_context": {
      "mttr_goal": "< 1 hour",
      "current_mttr": "45 min",
      "customer_impact": "All users cannot log in",
      "revenue_at_risk": "$50K/hour"
    },
    "related_entities": {
      "similar_tickets": [3912, 4056],
      "relevant_docs": ["auth-service-recovery", "jwt-troubleshooting"],
      "recent_discussions": ["#incidents: Anyone seeing auth issues?"],
      "related_commits": ["github/auth-service: Updated JWT validation"]
    }
  },
  "confidence_scores": {
    "correlation_ticket_commit": 0.94,
    "correlation_ticket_alert": 0.91,
    "similarity_to_past_incidents": 0.87
  }
}
```

---

## 4. The Flywheel Effect

The Context Engine creates a self-reinforcing loop:

```
        ┌─────────────────────────────────────┐
        │                                     │
        ▼                                     │
   ┌─────────┐    ┌─────────┐    ┌─────────┐  │
   │  USE    │───►│ CONTEXT │───►│ BETTER  │  │
   │         │    │ GROWS   │    │   AI    │  │
   └─────────┘    └─────────┘    └─────────┘  │
        ▲                              │      │
        │                              │      │
        │    ┌─────────────────────────┘      │
        │    │                                │
        │    ▼                                │
        │ ┌─────────┐    ┌─────────┐         │
        └─┤  MORE   │◄───┤  MORE   │◄────────┘
          │  VALUE  │    │ USAGE   │
          └─────────┘    └─────────┘
```

1. **Team uses Clarity IT** → Every ticket, doc, chat, alert adds to the graph
2. **Context graph grows** → More connections, richer correlations, better embeddings
3. **AI gets better** → Agents make decisions with fuller context
4. **More value delivered** → Faster resolution, better predictions, less manual work
5. **More usage** → Team relies on Clarity IT more → Back to step 1

**This is the moat.** Competitors can copy features. They cannot copy your team's accumulated context.

---

## 5. Implementation

### 5.1 Technology Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Graph Store** | Neo4j or Dgraph | Native graph queries, relationship traversal |
| **Vector Store** | pgvector or Qdrant | Semantic similarity, RAG for AI |
| **Time-Series** | TimescaleDB or InfluxDB | Event history, temporal queries |
| **Cache** | Redis | Hot context bundles, session context |
| **Stream Processing** | NATS JetStream or Kafka | Real-time event correlation |
| **Embedding Model** | Local (Ollama) or OpenAI | Generate semantic embeddings |
| **Query Engine** | Custom Go service | Context bundle assembly, graph traversal |

### 5.2 Data Model

```yaml
# Context Node
context_node:
  id: uuid
  entity_type: user | ticket | project | doc | alert | vm | service | ...
  entity_id: string
  source: queue | project | wiki | hub | grid | agent | external
  properties: jsonb
  embedding: vector(1536)
  created_at: timestamp
  updated_at: timestamp
  ttl: duration  # auto-expire stale context

# Context Edge
context_edge:
  id: uuid
  from_node: uuid
  to_node: uuid
  relation_type: created_by | assigned_to | blocks | caused_by | ...
  weight: float  # correlation strength
  evidence: [event_id]  # events that created this edge
  created_at: timestamp
  expires_at: timestamp

# Context Bundle
context_bundle:
  id: uuid
  target_type: agent | view | user
  target_id: string
  query: string
  bundle: jsonb
  freshness: timestamp
  cache_ttl: duration
```

### 5.3 API

```go
// Query context for an agent
ctx, err := contextEngine.Query(ContextQuery{
    For:       "queue_agent",
    Subject:   Subject{Type: "ticket", ID: "4821"},
    Dimensions: []ContextDimension{User, Team, Project, Time, System, Business},
    Depth:     3,  // how many hops in the graph
    Freshness: 30 * time.Second,
})

// Subscribe to context changes
sub, err := contextEngine.Subscribe(Subscription{
    Filter: "subject.type == 'ticket' && subject.priority == 'P1'",
    Callback: func(bundle ContextBundle) {
        // Alert Agent gets notified of new P1 context
    },
})

// Push context update
err := contextEngine.Ingest(Event{
    Type: "ticket.created",
    Payload: ticket,
    Source: "queue",
    Timestamp: time.Now(),
})
```

### 5.4 Integration with Agents

Every agent receives a Context Bundle as part of its input:

```json
{
  "agent_request": {
    "task": "Triage ticket 4821",
    "context_bundle": { /* full context as shown above */ },
    "tools": ["search_docs", "query_metrics", "create_incident"]
  }
}
```

The agent uses the bundle to:
- **Understand** the full situation before acting
- **Decide** with awareness of team state, project priorities, and system health
- **Explain** its reasoning by referencing context ("I escalated because Alice is overloaded and this is similar to incident #3912")

---

## 6. Real-World Impact

### Before Context Engine
> **3 AM Alert**: "Disk full on db-prod-01"
> 
> Alert Agent: "Disk full. Suggest cleanup."
> 
> On-call engineer: "Which disk? What cleanup? Is this related to the migration? Who is the DBA?"
> 
> **Result**: 15 minutes of hunting for context before action.

### After Context Engine
> **3 AM Alert**: "Disk full on db-prod-01"
> 
> Context Bundle includes:
> - This VM hosts the auth service database
> - Auth service is part of the api-v2-migration project
> - Project deadline is tomorrow
> - Last cleanup was 30 days ago
> - Runbook "db-disk-cleanup" exists and was last successful
> - On-call DBA is Charlie (secondary), Alice (primary) is handling a P1
> - Similar alert last month: cleanup freed 200GB
> 
> Alert Agent: "db-prod-01 (auth DB, v2.0 migration, deploy tomorrow) is 95% full. Runbook 'db-disk-cleanup' succeeded last month, freeing 200GB. Charlie (DBA, on-call) is available. Alice (primary) is on a P1. Suggest: execute runbook, notify Charlie, schedule follow-up."
> 
> **Result**: One-click approval, 2-minute resolution, informed handoff.

---

## 7. Why This Is the Missing Piece

| Without Context Engine | With Context Engine |
|----------------------|---------------------|
| Agents are smart but blind | Agents are smart **and** aware |
| Each module is a silo | Every module enriches every other module |
| AI suggestions are generic | AI suggestions are **situationally precise** |
| Users hunt for information | Information finds the user |
| Platform is a collection of tools | Platform is a **living understanding** of the team |
| Value is linear with usage | Value is **exponential** with usage (flywheel) |

**The Context Engine is what transforms Clarity IT from "a better tool" into "an indispensable team member."**

---

*Document Version: 1.0 | Generated: 2026-06-12*
