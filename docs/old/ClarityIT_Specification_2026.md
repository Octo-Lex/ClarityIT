# ClarityIT — AI-Native IT Operations Platform

## Project Specification & Architecture Blueprint (2026)

> **Status**: Design Phase | **Target**: Q3 2026 MVP | **License**: MIT/Apache-2.0

---

## 1. Executive Summary

**ClarityIT** is a lightweight, event-driven, AI-native platform that unifies **daily operational tasks**, **technical IT queues**, and **high-level project management** into a single, modern application. It is designed for IT teams, DevOps engineers, and technical project managers who need a fast, self-hostable solution that doesn't sacrifice power for simplicity.

Unlike monolithic suites (Mattermost + Focalboard + Outline + Vikunja), ClarityIT borrows the **best ideas** from each, reimagines them through an **agent-first, event-driven lens**, and delivers them as a cohesive, lightweight system deployable anywhere—from a single Cloudflare Worker to a Proxmox cluster.

---

## 2. Core Philosophy

| Principle | Description |
|-----------|-------------|
| **Agent-Native** | Every feature is exposed as an AI agent capability, not just a UI button |
| **Event-Driven** | All state changes flow through an event mesh; services are decoupled and reactive |
| **Git-as-Source** | Projects, docs, and tasks are stored as Markdown + YAML; Git is the sync layer |
| **Edge-First** | Cloudflare Workers handle auth, caching, and routing; compute happens at the edge |
| **Single Binary** | The core server compiles to one Go binary; deployment is `scp && ./ClarityIT` |
| **Protocol-First** | A2A (Agent-to-Agent) protocol for inter-agent communication; OpenAPI for everything else |

---

## 3. Borrowed Ideas & Sources

### 3.1 Solace Agent Mesh → Event Mesh + Agent Orchestration

- **Idea**: Event-driven multi-agent architecture with A2A protocol
- **Adopted**: The event mesh backbone, agent discovery, and delegation patterns
- **Simplified**: Replace Solace broker with NATS/Redis Streams; keep the A2A semantics
- **Code to borrow**: A2A protocol definitions, agent lifecycle management, YAML-based agent configuration

### 3.2 Shockwave → Local-First AI Coding Agent

- **Idea**: Integrated AI agent that reads/edits local files directly
- **Adopted**: The "bring your own key" agent pattern, SKILL.md system, encrypted secrets
- **Reimagined**: Agent operates on the event mesh, not just local files; can trigger workflows
- **Code to borrow**: SKILL.md loader, encrypted secret store, voice input pipeline

### 3.3 Obsidian-PM → Project Management Views

- **Idea**: Table, Gantt, and Kanban views with plain Markdown storage
- **Adopted**: YAML frontmatter for task metadata, drag-and-drop Gantt, dependency arrows
- **Enhanced**: Real-time sync via event mesh, AI-generated schedule suggestions
- **Code to borrow**: Gantt rendering engine, dependency resolution algorithm, custom field system

### 3.4 Vikunja → Task Service Architecture

- **Idea**: Clean Go backend with layered architecture (Models → Services → Routes)
- **Adopted**: CRUDable interface pattern, XORM-like generic handlers, permission system
- **Modernized**: Replace XORM with sqlc for type-safe SQL; add event sourcing layer
- **Code to borrow**: Generic web handler pattern, three-tier permission model, migration system

### 3.5 Outline → Documentation & Knowledge Base

- **Idea**: Beautiful, real-time collaborative wiki with Markdown
- **Adopted**: Document tree structure, real-time collaborative editing, search
- **Simplified**: CRDT-based sync (Yjs) instead of operational transforms; Git as backend
- **Code to borrow**: Document permission model, collection hierarchy, full-text search indexing

### 3.6 Mattermost → Team Communication Hub

- **Idea**: Channel-based messaging with threads, reactions, and integrations
- **Adopted**: Threaded conversations, @mentions, slash commands, webhook system
- **Streamlined**: Remove heavy React Native mobile app; use PWA + Tauri desktop
- **Code to borrow**: WebSocket event broadcast, plugin architecture, bot account system

### 3.7 Focalboard → Board Views & Cards

- **Idea**: Kanban boards with custom properties and card-based organization
- **Adopted**: Board/card data model, property types (select, multi-select, date, person)
- **Integrated**: Boards are just another view of tasks/projects in the event mesh
- **Code to borrow**: Property type system, board view rendering, card template engine

### 3.8 AppFlowy → Block-Based Editor

- **Idea**: Notion-like block editor for documents and databases
- **Adopted**: Block-based content model, database-as-blocks, AI-assisted writing
- **Adapted**: Blocks stored as Markdown with custom syntax; editor is ProseMirror-based
- **Code to borrow**: Block serialization format, database block logic, AI completion hooks

### 3.9 Nextcloud → File Sync & Sharing

- **Idea**: Self-hosted file storage with sharing, versioning, and federation
- **Adopted**: File versioning, share links, federated sharing, WebDAV support
- **Simplified**: S3/R2 as primary storage; Git LFS for versioned assets
- **Code to borrow**: Share permission model, activity feed, file conflict resolution

### 3.10 Proxmox → Infrastructure Integration

- **Idea**: API-driven VM/container management with monitoring
- **Adopted**: REST API patterns, resource metrics, backup/restore workflows
- **Integrated**: Proxmox events feed into ClarityIT alert agent; VM tasks become ClarityIT tasks
- **Code to borrow**: API client patterns, metric collection, task queue integration

### 3.11 Cloudflare → Edge Architecture

- **Idea**: Serverless edge computing with global distribution
- **Adopted**: Workers for auth/gateway, D1 for edge SQLite, R2 for object storage, KV for sessions
- **Architecture**: Cloudflare Workers as the "front door"; origin server optional for heavy compute
- **Code to borrow**: D1 connection pooling, KV session management, R2 multipart upload

---

## 4. System Architecture

### 4.1 High-Level Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CLIENT LAYER                                      │
│  Web (React+Vite)  Mobile (PWA)  Desktop (Tauri)  CLI (Go)  Bots (Slack)  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        EDGE / GATEWAY (Cloudflare)                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Workers   │  │  WAF/Rate   │  │  Auth/JWT   │  │  CDN/Cache  │        │
│  │   (D1+KV)   │  │   Limit     │  │   OIDC      │  │   (R2)      │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         EVENT MESH (NATS/Redis)                             │
│     Topics: ops.tasks.* | ops.projects.* | ops.alerts.* | ops.agents.*        │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
┌────────────────────────┐ ┌─────────────────┐ ┌────────────────────────┐
│     AI AGENT MESH      │ │  CORE SERVICES  │ │      DATA LAYER        │
│  ┌──────────────────┐  │ │  ┌───────────┐  │ │  ┌──────────────────┐  │
│  │  Orchestrator    │  │ │  │  Task     │  │ │  │  PostgreSQL      │  │
│  │  (Route/Delegate)│  │ │  │  Service  │  │ │  │  (Primary)       │  │
│  ├──────────────────┤  │ │  ├───────────┤  │ │  ├──────────────────┤  │
│  │  Queue Agent     │  │ │  │  Project  │  │ │  │  Redis           │  │
│  │  (ITSM/SLA)      │  │ │  │  Service  │  │ │  │  (Cache/PubSub)  │  │
│  ├──────────────────┤  │ │  ├───────────┤  │ │  ├──────────────────┤  │
│  │  Project Agent   │  │ │  │  Queue    │  │ │  │  S3/R2           │  │
│  │  (PM/Gantt)      │  │ │  │  Service  │  │ │  │  (Files)         │  │
│  ├──────────────────┤  │ │  ├───────────┤  │ │  ├──────────────────┤  │
│  │  Ops Agent       │  │ │  │  Doc      │  │ │  │  Git Repo        │  │
│  │  (Daily Tasks)   │  │ │  │  Service  │  │ │  │  (Markdown)      │  │
│  ├──────────────────┤  │ │  ├───────────┤  │ │  ├──────────────────┤  │
│  │  Alert Agent     │  │ │  │  Infra    │  │ │  │  Vector DB       │  │
│  │  (Monitoring)    │  │ │  │  Service  │  │ │  │  (Embeddings)    │  │
│  ├──────────────────┤  │ │  └───────────┘  │ │  └──────────────────┘  │
│  │  Doc Agent       │  │ │                 │ │                        │
│  │  (Wiki/KB)       │  │ │                 │ │                        │
│  └──────────────────┘  │ │                 │ │                        │
└────────────────────────┘ └─────────────────┘ └────────────────────────┘
```

### 4.2 Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Frontend** | React 19 + Vite + Tailwind + shadcn/ui | Fast, modern, accessible |
| **Desktop** | Tauri (Rust) | <5MB binaries, native performance |
| **Mobile** | Capacitor PWA | One codebase, app-store optional |
| **CLI** | Go (Cobra) | Single binary, fast, cross-platform |
| **API Gateway** | Cloudflare Workers | Edge compute, D1/KV/R2 integration |
| **Event Mesh** | NATS (or Redis Streams) | Lightweight, high throughput |
| **Core Server** | Go 1.24 + stdlib + sqlc | Fast, type-safe, minimal dependencies |
| **Database** | PostgreSQL 16 (primary) + SQLite (edge) | Reliable, feature-rich, D1-compatible |
| **Cache** | Redis 7 | Sessions, rate limiting, pub/sub |
| **Search** | Typesense or Meilisearch | Fast, typo-tolerant, faceted |
| **AI/LLM** | OpenAI/Anthropic/local via LiteLLM | Bring-your-own-key, model-agnostic |
| **Storage** | S3-compatible (R2/MinIO) | Universal, cost-effective |
| **Git Sync** | go-git | Pure Go, no CGO, embeddable |
| **Docs** | ProseMirror + Yjs | Real-time collaborative editing |
| **Monitoring** | Prometheus + Grafana | Industry standard |

---

## 5. Feature Modules

### 5.1 Task & Queue Management ("The Queue")

**Inspired by**: Vikunja + Focalboard + Mattermost

- **Kanban Board**: Drag-and-drop with swimlanes, WIP limits, automation rules
- **IT Queue**: SLA tracking, priority escalation, assignment round-robin
- **Table View**: Sortable, filterable, bulk actions, custom fields
- **Calendar View**: Deadline visualization, recurring tasks
- **Time Tracking**: Built-in timer, manual entry, time reports
- **Automation**: "When X happens, do Y" — powered by event mesh

**AI Integration**:

- Auto-prioritize queue based on SLA risk
- Suggest task decomposition for large items
- Generate incident post-mortems from ticket history

### 5.2 Project Management ("The Project")

**Inspired by**: Obsidian-PM + AppFlowy

- **Gantt Chart**: Interactive timeline, dependencies, critical path, milestones
- **Table View**: Hierarchical tasks, custom fields, saved filters
- **Wiki Pages**: Block-based editor, embedded databases, templates
- **Sprints**: Scrum/Kanban hybrid, velocity tracking, burndown charts
- **Roadmap**: High-level initiative tracking, cross-project dependencies

**AI Integration**:

- Auto-schedule based on dependencies and resource availability
- Generate project status reports from commit + task data
- Risk prediction based on historical velocity

### 5.3 Documentation & Knowledge Base ("The Wiki")

**Inspired by**: Outline + Shockwave

- **Collections**: Hierarchical document organization
- **Real-time Collab**: Multi-user editing with cursors and comments
- **Search**: Full-text + semantic (vector) search across all docs
- **Templates**: Reusable document structures
- **Import/Export**: Markdown, PDF, HTML, Confluence
- **Git Sync**: Every doc change is a Git commit; full history

**AI Integration**:

- Auto-generate doc summaries
- Suggest related docs based on content
- "Ask the wiki" — RAG-powered Q&A

### 5.4 Team Communication ("The Hub")

**Inspired by**: Mattermost + Nextcloud

- **Channels**: Public/private, threaded conversations
- **Direct Messages**: 1:1 and group chats
- **File Sharing**: Drag-and-drop, previews, versioning
- **Video/Voice**: WebRTC integration (optional Jitsi)
- **Integrations**: Webhooks, slash commands, bot framework
- **Activity Feed**: Unified notification center

**AI Integration**:

- Meeting transcription + action item extraction
- Smart reply suggestions
- Channel sentiment analysis

### 5.5 Infrastructure Integration ("The Grid")

**Inspired by**: Proxmox + Cloudflare

- **VM/Container Dashboard**: Resource usage, status, quick actions
- **Alert Management**: Aggregate alerts from Proxmox, Prometheus, etc.
- **Runbooks**: Link alerts to wiki docs and automated responses
- **Backup Status**: Visual backup health across all systems
- **Cost Tracking**: Resource allocation vs. budget

**AI Integration**:

- Anomaly detection on metrics
- Auto-remediation for common alerts
- Capacity planning recommendations

---

## 6. Data Model

### 6.1 Core Entities (Event-Sourced)

```yaml
# Task (the universal work item)
task:
  id: uuid
  type: task | queue_item | project_task | alert | incident
  title: string
  description: markdown
  status: backlog | todo | in_progress | blocked | review | done | cancelled
  priority: critical | high | medium | low
  assignees: [user_id]
  labels: [string]
  due_date: datetime
  start_date: datetime
  estimated_hours: float
  logged_hours: [time_log]
  parent_id: uuid  # for subtasks
  dependencies: [uuid]
  custom_fields: {key: value}
  source: manual | github | gitlab | jira | proxmox | prometheus
  source_id: string  # external ID
  events: [domain_event]  # event sourcing
  created_at: datetime
  updated_at: datetime

# Project
project:
  id: uuid
  name: string
  description: markdown
  color: hex
  icon: emoji
  status: planning | active | on_hold | completed | cancelled
  owner_id: user_id
  members: [{user_id, role}]
  start_date: datetime
  target_date: datetime
  budget: {amount, currency}
  views: [kanban | gantt | table | board | calendar]
  settings: {workflows, automations}
  git_repo: string  # optional sync target

# Document
doc:
  id: uuid
  title: string
  content: block[]  # ProseMirror JSON or Markdown
  collection_id: uuid
  parent_id: uuid  # for nested docs
  authors: [user_id]
  tags: [string]
  published: bool
  version: int
  git_sha: string  # sync anchor
  events: [domain_event]

# Queue (IT Service Desk)
queue:
  id: uuid
  name: string
  type: incident | service_request | change | problem
  sla_policy: {response_time, resolution_time, escalation_rules}
  categories: [string]
  auto_assign_rules: [rule]
  templates: [ticket_template]
```

### 6.2 Event Schema

```yaml
event:
  id: uuid
  type: task.created | task.updated | task.status_changed | 
       project.milestone_reached | alert.triggered | agent.delegated
  aggregate_id: uuid  # the entity this event belongs to
  aggregate_type: task | project | doc | queue | alert
  payload: json
  metadata:
    user_id: uuid
    agent_id: uuid  # if triggered by AI
    timestamp: datetime
    correlation_id: uuid  # for distributed tracing
    causation_id: uuid  # previous event that caused this
```

---

## 7. Agent Architecture

### 7.1 Agent Types

| Agent | Role | Capabilities |
|-------|------|-------------|
| **Orchestrator** | Central router | Task decomposition, agent discovery, workflow orchestration |
| **Queue Agent** | ITSM expert | Ticket triage, SLA monitoring, escalation, runbook execution |
| **Project Agent** | PM assistant | Scheduling, dependency analysis, risk assessment, reporting |
| **Ops Agent** | Daily ops | Standup summaries, task reminders, meeting notes, action tracking |
| **Alert Agent** | SRE companion | Alert correlation, anomaly detection, auto-remediation, on-call |
| **Doc Agent** | Knowledge keeper | Doc generation, Q&A, content suggestions, link maintenance |
| **Code Agent** | Dev helper | PR summaries, code review, commit message generation, debugging |

### 7.2 Agent Communication (A2A Protocol)

```json
{
  "message_type": "task_request",
  "from": "orchestrator@ClarityIT.local",
  "to": "queue_agent@ClarityIT.local",
  "task": {
    "id": "task-uuid",
    "description": "Triage incoming alert: CPU > 90% on vm-prod-03",
    "context": { "alert": {...}, "vm": {...}, "runbook": "cpu-spike" },
    "expected_output": "incident_ticket or auto_remediation_result"
  },
  "deadline": "2026-06-12T03:00:00Z",
  "callback_topic": "ops.agents.response.orchestrator"
}
```

### 7.3 Agent Skills (Shockwave-inspired)

Agents load capabilities from `SKILL.md` files:

```markdown
# Skill: Incident Triage

## Description
Analyze monitoring alerts and create prioritized incident tickets.

## Tools
- search_docs(query)
- create_ticket(payload)
- query_metrics(vm_id, metric, timeframe)
- run_remediation(runbook_id, params)

## Examples
- "CPU spike on web-01" → Check runbook "cpu-spike", create P2 ticket
- "Disk full on db-03" → Check runbook "disk-cleanup", attempt auto-remediation
```

---

## 8. Deployment Models

### 8.1 Edge-First (Cloudflare)

```
Cloudflare Workers (Auth + API Gateway)
    ↓
D1 (SQLite) — edge data, sessions, config
    ↓
R2 (Object Storage) — files, backups
    ↓
KV (Key-Value) — rate limits, feature flags
    ↓
Origin Server (optional) — heavy compute, AI inference
```

### 8.2 Self-Hosted (Proxmox/Docker)

```
Nginx / Caddy (Reverse Proxy)
    ↓
ClarityIT Server (Go binary)
    ↓
PostgreSQL + Redis + NATS
    ↓
MinIO / S3 (Object Storage)
    ↓
Typesense (Search)
```

### 8.3 Hybrid (Recommended)

```
Cloudflare Workers (Edge: Auth, Cache, Static)
    ↓
Tailscale / WireGuard (Secure tunnel)
    ↓
Proxmox LXC / Docker (Origin: Core services)
    ↓
Local NAS / S3 (Storage)
```

---

## 9. Development Roadmap

### Phase 1: Foundation (Weeks 1-4)

- [ ] Event mesh setup (NATS)
- [ ] Core Go server with sqlc
- [ ] PostgreSQL schema + migrations
- [ ] Basic auth (JWT + OIDC)
- [ ] Task CRUD API
- [ ] React frontend scaffold

### Phase 2: Task & Queue (Weeks 5-8)

- [ ] Kanban board view
- [ ] IT queue with SLA
- [ ] Table view with filters
- [ ] Time tracking
- [ ] Basic automation rules
- [ ] Git sync for tasks

### Phase 3: Projects & Docs (Weeks 9-12)

- [ ] Gantt chart
- [ ] Block-based editor
- [ ] Collections & wiki
- [ ] Real-time collab (Yjs)
- [ ] Search (Typesense)
- [ ] Import/export

### Phase 4: AI & Agents (Weeks 13-16)

- [ ] Agent framework
- [ ] Orchestrator agent
- [ ] Queue agent (ITSM)
- [ ] Project agent (PM)
- [ ] Doc agent (RAG)
- [ ] A2A protocol

### Phase 5: Communication & Infra (Weeks 17-20)

- [ ] Channels & DMs
- [ ] File sharing
- [ ] Proxmox integration
- [ ] Alert aggregation
- [ ] Mobile PWA
- [ ] Desktop app (Tauri)

### Phase 6: Polish & Scale (Weeks 21-24)

- [ ] Performance optimization
- [ ] Security audit
- [ ] Documentation
- [ ] Community onboarding
- [ ] Plugin SDK
- [ ] Cloudflare Workers edge deploy

---

## 10. File Structure

```
ClarityIT/
├── cmd/
│   ├── server/           # Main Go server
│   ├── cli/              # CLI tool
│   └── agent/            # Agent runner
├── internal/
│   ├── api/              # HTTP handlers
│   ├── domain/           # Business logic
│   │   ├── task/
│   │   ├── project/
│   │   ├── doc/
│   │   ├── queue/
│   │   └── alert/
│   ├── events/           # Event sourcing
│   ├── agents/           # AI agent framework
│   ├── mesh/             # Event mesh client
│   ├── storage/          # DB + cache + search
│   ├── auth/             # Authentication
│   └── git/              # Git sync engine
├── pkg/
│   ├── a2a/              # Agent-to-Agent protocol
│   └── models/             # Shared types
├── web/
│   ├── src/
│   │   ├── components/   # UI components
│   │   ├── views/        # Page views
│   │   ├── stores/       # State management (Zustand)
│   │   ├── agents/       # Agent UI integration
│   │   └── lib/          # Utilities
│   └── public/
├── desktop/              # Tauri desktop app
├── mobile/               # Capacitor PWA
├── skills/               # Agent SKILL.md files
├── docs/                 # Project documentation
├── migrations/           # DB migrations
├── docker/               # Docker configs
├── scripts/              # Build & deploy
└── Makefile
```

---

## 11. Why Not Use These Projects As-Is?

| Project | Limitation | ClarityIT Solution |
|---------|-----------|----------------|
| Mattermost | Heavy, mobile apps required | Lightweight PWA + Tauri, event-driven |
| Outline | BSL license, no self-hosted AI | MIT license, built-in agent mesh |
| Vikunja | No AI, no real-time collab | Agent-native, Yjs collaboration |
| Focalboard | Unmaintained, no backend | Active development, event-sourced |
| AppFlowy | Complex Rust/Flutter stack | Go + React, simpler deployment |
| Nextcloud | PHP monolith, slow | Go microservices, edge-first |
| Proxmox | No task/queue integration | Unified ops platform |
| Solace SAM | Requires Solace broker | NATS/Redis, zero vendor lock-in |

---

## 12. Success Metrics

- **Deploy time**: < 5 minutes from download to running
- **Binary size**: < 50MB server, < 10MB CLI
- **Memory footprint**: < 256MB idle
- **Cold start**: < 100ms (Cloudflare Workers)
- **Concurrent users**: 10,000+ per instance
- **Agent response**: < 2s for common tasks
- **Git sync latency**: < 1s for doc changes

---

*Generated: 2026-06-12 | Next review: 2026-06-26*
