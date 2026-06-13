# Clarity IT — Comprehensive Project Overview
## AI-Native IT Operations Platform (2026)

---

## 1. What Is Clarity IT?

Clarity IT is a **unified, AI-native platform** designed for IT teams, DevOps engineers, and technical project managers. It brings together five traditionally separate domains into one lightweight, fast, self-hostable application:

1. **IT Service Desk** (ticketing, queues, SLA management)
2. **Project Management** (Gantt, sprints, resource planning)
3. **Knowledge Base** (wiki, documentation, runbooks)
4. **Team Communication** (chat, channels, file sharing)
5. **Infrastructure Operations** (Proxmox, monitoring, alerting)

Instead of stitching together Mattermost + Jira + Confluence + Trello + Grafana, Clarity IT replaces them all with a single, cohesive system where **AI agents** handle the heavy lifting and **events** keep everything in sync.

---

## 2. The Problem It Solves

### Current State (The Tool Sprawl)
Today's IT teams operate across 5–10 disconnected tools:

| Need | Typical Tool | Problem |
|------|-------------|---------|
| Chat | Slack/Teams/Mattermost | Heavy, expensive, no AI integration |
| Tickets | Jira/ServiceNow | Bloated, slow, expensive per seat |
| Projects | Asana/Monday/Notion | No IT-specific features |
| Docs | Confluence/Notion | No Git sync, vendor lock-in |
| Infrastructure | Grafana + Proxmox UI | No connection to tickets or projects |
| AI | ChatGPT/Claude Code | No access to team data, no automation |

**Result**: Context switching, data silos, subscription fatigue, and AI that doesn't know your team's reality.

### Clarity IT State (The Unified Platform)
| Need | Clarity IT Module | How It's Different |
|------|-----------------|-------------------|
| Chat | The Hub | Integrated with tickets, projects, alerts |
| Tickets | The Queue | AI auto-triage, SLA prediction, runbook linking |
| Projects | The Project | Auto-scheduling, resource balancing, risk prediction |
| Docs | The Wiki | Git-synced, AI Q&A, semantic search |
| Infrastructure | The Grid | Proxmox-native, alert correlation, auto-remediation |
| AI | The Agents | 6 specialized agents with access to ALL team data |

**Result**: One platform, one subscription (or free self-hosted), AI that understands your entire operation.

---

## 3. Core Principles

### 3.1 Agent-Native
Every feature is designed as an AI capability first, UI second. The Queue Agent can triage tickets without a human opening the app. The Alert Agent can remediate issues at 3 AM. The UI is a window into what the agents are doing.

### 3.2 Event-Driven
All state changes flow through an event mesh (NATS/Redis Streams). When a ticket is created, an event fires. The Queue Agent sees it. The Project Agent updates the sprint burndown. The Hub posts a notification. Services are decoupled, reactive, and scalable.

### 3.3 Git-as-Source
Projects, docs, and tasks are stored as Markdown + YAML. Every change is a Git commit. Your data is portable, version-controlled, and never locked in. You can `git clone` your entire team's knowledge.

### 3.4 Edge-First
Cloudflare Workers handle authentication, caching, and routing at 300+ global locations. The origin server only handles heavy compute. Users get sub-100ms response times worldwide.

### 3.5 Single Binary
The core server compiles to one Go binary. Deployment is:
```bash
scp clarity-it-server $HOST:/opt/clarity-it/
ssh $HOST "cd /opt/clarity-it && ./clarity-it-server"
```
No Docker required. No dependency hell. No 47 microservices.

### 3.6 Protocol-First
- **A2A Protocol**: Agents talk to agents
- **OpenAPI 3.1**: APIs are documented and discoverable
- **Event Mesh**: Everything publishes and subscribes to topics

---

## 4. The Five Modules + AI Core

### 4.1 The Queue — IT Service Desk
**For**: Help desk, IT support, incident management

**Key Functions**:
- Kanban board with WIP limits and swimlanes
- SLA tracking with automatic escalation
- AI auto-triage (reads ticket, suggests category/priority/assignee)
- Round-robin and skill-based assignment
- Time tracking (timer + manual)
- Customer portal with self-service knowledge base
- Email-to-ticket conversion
- Bulk actions and automation rules

**AI Power**: Queue Agent predicts SLA breaches, suggests resolutions from past tickets, generates incident post-mortems, and finds similar resolved tickets.

### 4.2 The Project — Project Management
**For**: IT projects, migrations, deployments, sprints

**Key Functions**:
- Interactive Gantt chart with dependencies and critical path
- Sprint planning with velocity tracking
- Burndown/burnup charts
- Resource allocation heatmap
- Cross-project dependency mapping
- Budget tracking vs. actuals
- Risk register with impact scoring
- What-if scenario planning

**AI Power**: Project Agent auto-schedules based on dependencies, generates status reports from commits + tasks, predicts delays from velocity trends, and suggests resource rebalancing.

### 4.3 The Wiki — Knowledge Base
**For**: Runbooks, SOPs, architecture docs, onboarding

**Key Functions**:
- Block-based editor (Notion-style `/commands`)
- Real-time collaborative editing (Yjs CRDTs)
- Semantic search + AI Q&A ("How do we handle SSL certs?")
- Git sync (every save is a commit)
- Templates for runbooks, meeting notes, RFCs, post-mortems
- Import from Confluence/Notion; export to PDF/HTML
- Outdated document detection

**AI Power**: Doc Agent auto-summarizes, suggests related docs, answers questions from your knowledge base, detects broken links, and generates onboarding guides.

### 4.4 The Hub — Team Communication
**For**: Daily coordination, incident response, announcements

**Key Functions**:
- Public/private channels with threaded conversations
- Persistent voice rooms (Discord-style)
- Video calls with screen sharing and remote control
- Meeting recording + auto-transcription
- File sharing with inline previews and versioning
- Slash commands (`/ticket`, `/deploy`, `/status`)
- Webhook integrations (GitHub, GitLab, Jira)
- Smart replies and sentiment analysis

**AI Power**: Meeting transcription → auto-extracts action items → creates tasks in The Queue. Smart replies speed up responses. Sentiment monitoring flags escalating situations.

### 4.5 The Grid — Infrastructure & Operations
**For**: Proxmox management, monitoring, alerting, compliance

**Key Functions**:
- Proxmox VM/container dashboard with live metrics
- One-click start/stop/restart/migrate/backup
- Alert aggregation from Prometheus, PRTG, Nagios, Zabbix
- Alert correlation ("CPU high + disk full + backup fail = one incident")
- Runbook execution with approval gates
- Auto-remediation for approved scenarios
- Capacity forecasting ("Disk full in 14 days")
- Certificate expiry monitoring
- Cost tracking per VM/project

**AI Power**: Alert Agent detects anomalies, auto-remediates common issues, generates incident timelines, and creates on-call handoff summaries.

### 4.6 The Agents — AI Automation Core
**The brain of Clarity IT.** Six specialized agents communicate via the A2A protocol:

| Agent | Role | Example Task |
|-------|------|--------------|
| **Orchestrator** | Central router | "Plan the API migration" → decomposes into sub-tasks for other agents |
| **Queue Agent** | ITSM expert | Auto-triage ticket, suggest fix, predict SLA breach |
| **Project Agent** | PM assistant | Auto-schedule sprint, generate status report, predict delay |
| **Ops Agent** | Daily operations | Send standup summary, extract meeting action items, set reminders |
| **Alert Agent** | SRE companion | Detect anomaly, execute runbook, forecast capacity |
| **Doc Agent** | Knowledge keeper | Summarize doc, answer wiki questions, generate onboarding guide |

---

## 5. System Architecture

### 5.1 Layered Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  CLIENTS                                                    │
│  Web (React+Vite) · Mobile PWA · Desktop (Tauri) · CLI · API│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  EDGE / GATEWAY (Cloudflare)                                │
│  Workers · WAF · Auth(JWT/OIDC) · CDN · D1 + KV + R2      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  EVENT MESH (NATS / Redis Streams)                          │
│  Topics: ops.tasks.* · ops.projects.* · ops.alerts.*        │
│          ops.agents.* · ops.queues.*                        │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  AI AGENTS    │    │ CORE SERVICES │    │  DATA LAYER   │
│  Orchestrator │    │  Go 1.24      │    │  PostgreSQL   │
│  Queue        │    │  sqlc         │    │  SQLite (D1)  │
│  Project      │    │  Task Service │    │  Redis        │
│  Ops          │    │  Project Svc  │    │  S3/R2        │
│  Alert        │    │  Queue Svc    │    │  Git Repo     │
│  Doc          │    │  Doc Svc      │    │  Vector DB    │
│  A2A Protocol │    │  Infra Svc    │    │               │
└───────────────┘    └───────────────┘    └───────────────┘
```

### 5.2 Technology Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Frontend | React 19 + Vite + Tailwind + shadcn/ui | Fast, modern, accessible |
| Desktop | Tauri (Rust) | <5MB binaries, native performance |
| Mobile | Capacitor PWA | One codebase, works offline |
| CLI | Go (Cobra) | Single binary, cross-platform |
| Edge | Cloudflare Workers | Global edge compute, D1/KV/R2 |
| Event Mesh | NATS or Redis Streams | Lightweight, high throughput |
| Core Server | Go 1.24 + sqlc | Fast, type-safe, minimal deps |
| Database | PostgreSQL 16 + SQLite (D1) | Reliable + edge-compatible |
| Cache | Redis 7 | Sessions, rate limiting, pub/sub |
| Search | Typesense | Fast, typo-tolerant, faceted |
| AI | LiteLLM (BYO key) | Model-agnostic, local or cloud |
| Storage | S3-compatible (R2/MinIO) | Universal, cost-effective |
| Git | go-git | Pure Go, no CGO, embeddable |
| Docs | ProseMirror + Yjs | Real-time collaborative editing |
| Monitoring | Prometheus + Grafana | Industry standard |

---

## 6. Source Projects & What We Borrow

| Source | Idea Borrowed | How We Adapt It |
|--------|--------------|-----------------|
| **Solace Agent Mesh** | Event-driven multi-agent A2A protocol | Replace Solace broker with NATS; keep semantics |
| **Shockwave** | Integrated AI agent + SKILL.md system | Agent operates on event mesh, not just local files |
| **Obsidian-PM** | Gantt/Kanban/Table with Markdown storage | Real-time sync via event mesh, AI scheduling |
| **Vikunja** | Clean Go layered architecture | Replace XORM with sqlc; add event sourcing |
| **Outline** | Collaborative wiki with Markdown | CRDT sync (Yjs) instead of OT; Git backend |
| **Mattermost** | Team chat, threads, webhooks | PWA + Tauri instead of heavy mobile apps |
| **Focalboard** | Kanban with custom properties | Boards as views of tasks in event mesh |
| **AppFlowy** | Block editor, database blocks | Markdown-based blocks; ProseMirror editor |
| **Nextcloud** | File sync, sharing, versioning | S3/R2 primary; Git LFS for versioned assets |
| **Proxmox** | VM management, monitoring API | Events feed into alert agent; VM tasks → tickets |
| **Cloudflare** | Edge compute, D1, KV, R2 | Workers as front door; origin optional |

---

## 7. Deployment Models

### 7.1 Edge-First (Cloudflare)
```
Cloudflare Workers (Auth + API Gateway + Cache)
    ├── D1 (SQLite) — sessions, config, edge data
    ├── KV — rate limits, feature flags
    └── R2 — files, backups

Origin Server (optional) — heavy compute, AI inference
```
**Best for**: Teams wanting global low latency with minimal infrastructure.

### 7.2 Self-Hosted (Proxmox/Docker)
```
Nginx/Caddy (Reverse Proxy)
    └── Clarity IT Server (Go binary)
        ├── PostgreSQL
        ├── Redis
        ├── NATS
        └── MinIO/S3
```
**Best for**: Teams requiring full data sovereignty.

### 7.3 Hybrid (Recommended)
```
Cloudflare Workers (Edge: Auth, Cache, Static)
    └── Tailscale/WireGuard (Secure tunnel)
        └── Proxmox LXC/Docker (Origin: Core services)
            └── Local NAS/S3 (Storage)
```
**Best for**: Best of both worlds — edge performance + self-hosted data.

---

## 8. Development Roadmap (24 Weeks)

### Phase 1: Foundation (Weeks 1–4)
- Event mesh (NATS)
- Go server with sqlc
- PostgreSQL schema + migrations
- Auth (JWT + OIDC)
- Task CRUD API
- React frontend scaffold

### Phase 2: Task & Queue (Weeks 5–8)
- Kanban board view
- IT queue with SLA tracking
- Table view with filters
- Time tracking
- Basic automation rules
- Git sync for tasks

### Phase 3: Projects & Docs (Weeks 9–12)
- Gantt chart
- Block-based editor
- Collections + wiki structure
- Real-time collaboration (Yjs)
- Search (Typesense)
- Import/export tools

### Phase 4: AI & Agents (Weeks 13–16)
- Agent framework
- Orchestrator agent
- Queue + Project agents
- Doc + Alert agents
- A2A protocol implementation
- SKILL.md system

### Phase 5: Communication & Infra (Weeks 17–20)
- Channels + DMs
- File sharing
- Proxmox integration
- Alert aggregation
- Mobile PWA
- Desktop app (Tauri)

### Phase 6: Polish & Scale (Weeks 21–24)
- Performance optimization
- Security audit
- Documentation
- Community onboarding
- Plugin SDK
- Cloudflare Workers edge deploy

---

## 9. Success Metrics

| Metric | Target |
|--------|--------|
| Deploy Time | < 5 minutes |
| Binary Size | < 50 MB |
| Memory Footprint (Idle) | < 256 MB |
| Cold Start | < 100 ms |
| Concurrent Users | 10,000+ |
| Agent Response Time | < 2 seconds |
| Git Sync Latency | < 1 second |

---

## 10. Why Not Use Existing Tools As-Is?

| Tool | Limitation | Clarity IT Solution |
|------|-----------|---------------------|
| Mattermost | Heavy, no AI, mobile apps required | Lightweight PWA + Tauri, agent-native |
| Jira | Bloated, expensive, slow | Fast, single binary, AI-integrated |
| Confluence | No Git sync, vendor lock-in | Git-as-source, Markdown, AI Q&A |
| Notion | No IT-specific features, no self-hosting | IT queue, Proxmox, self-hosted |
| Trello/Asana | No ITSM, no SLA | Full IT queue with SLA + automation |
| Grafana | No connection to tickets/projects | Alerts become tickets, runbooks linked |
| ChatGPT/Claude | No access to team data | Agents with full context of your operation |

---

## 11. File Structure

```
clarity-it/
├── cmd/
│   ├── server/           # Main Go server
│   ├── cli/              # CLI tool
│   └── agent/            # Agent runner
├── internal/
│   ├── api/              # HTTP handlers
│   ├── domain/             # Business logic
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
│   └── models/           # Shared types
├── web/
│   ├── src/
│   │   ├── components/   # UI components
│   │   ├── views/        # Page views
│   │   ├── stores/       # Zustand state
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

## 12. Licensing & Community

- **Core Platform**: MIT License
- **Agent Framework**: Apache-2.0
- **Plugin SDK**: MIT License
- **Contributions**: Welcome via GitHub; all PRs require issue discussion first
- **No AI-generated bulk PRs**: Code must be human-readable and match project style

---

## 13. Next Steps

1. **Validate the architecture** — Does this solve your team's actual pain points?
2. **Prioritize modules** — Which module delivers the most value first?
3. **Design the database schema** — Core entities and relationships
4. **Build the event mesh** — Foundation everything else depends on
5. **Create the first agent** — Queue Agent is the highest-impact starting point

---

*Document Version: 1.0 | Generated: 2026-06-12 | Status: Design Phase*
