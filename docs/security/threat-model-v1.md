# ClarityIT v1.0 Threat Model

## Document Status
- **Version**: 1.0.0
- **Date**: 2026-06-14
- **Scope**: Full platform — control plane, reasoning workers, event transport, operator UI, integrations

---

## 1. System Overview

ClarityIT is an IT operations platform with:
- **Go control plane** (chi router, pgx) — canonical truth, all mutations
- **PostgreSQL** — relational store, 40+ tables, migrations 001-025
- **NATS JetStream** — event transport (outbox pattern)
- **Redis** — WebSocket fanout, rate limiting
- **MinIO** — object storage for attachments
- **Python reasoning workers** — isolated, communicate via HTTP only
- **React 19 frontend** — SPA, in-memory tokens, permission-gated UI
- **Proxmox** — infrastructure management (read-only + controlled mutation)

## 2. Trust Boundaries

```
┌─────────────────────────────────────────────────────────┐
│                     Internet / LAN                       │
│                    (untrusted)                           │
│                        │                                 │
│           ┌────────────┼────────────┐                    │
│           ▼            ▼            ▼                    │
│    [Cloudflare]   [Web :3000]   [API :8765]             │
│     Tunnel/TLS    nginx→SPA     Go chi router            │
│                                      │                   │
│                    ┌─────────────────┤                   │
│                    ▼                 ▼                   │
│              [PostgreSQL]     [NATS JetStream]           │
│              [Redis]          [MinIO]                    │
│                    │                                     │
│           ┌────────┘                                     │
│           ▼                                              │
│    [Python Workers] ── HTTP only ──→ [API :8765]        │
│    (no DB/NATS/Redis access)                             │
│                                                         │
│    [Proxmox :8006] ←── API token ──← [API :8765]       │
│                        (read + controlled mutation)      │
└─────────────────────────────────────────────────────────┘
```

### Trust Zones
| Zone | Trust Level | Access |
|------|------------|--------|
| Internet/LAN | Untrusted | Via Cloudflare Tunnel only |
| API :8765 | Semi-trusted (authenticated) | All services, DB, NATS, Redis, MinIO, Proxmox |
| Python Workers | Low-trust | API HTTP only — no infrastructure access |
| PostgreSQL/NATS/Redis/MinIO | Trusted (private) | Only from API container network |
| Proxmox | Semi-trusted (API token) | Read + controlled mutation (start/shutdown/stop/snapshot) |

## 3. Threat Catalogue (STRIDE)

### Spoofing

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Password brute-force | bcrypt(12) hashing, rate limiting | ✅ Implemented |
| Token theft/replay | Token families with rotation + reuse detection; access tokens in memory only (not localStorage) | ✅ Implemented |
| Session hijacking | httpOnly refresh cookies, CSRF-safe (same-site), session revocation | ✅ Implemented |
| MFA bypass | TOTP required for high/critical risk; `RequireRecentMFA` middleware; 5-min validity window | ✅ Implemented |
| Bootstrap spoofing | Bootstrap lock (first user only), subsequent calls rejected | ✅ Implemented |
| Integration key forgery | HMAC-SHA256 hashed keys (never stored plaintext) | ✅ Implemented |
| Webhook spoofing | HMAC-SHA256 signature verification; timestamp freshness check | ✅ Implemented |

### Tampering

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Audit log tampering | Append-only audit_logs; IP/user-agent stored as HMAC (not plaintext) | ✅ Implemented |
| Event payload tampering | PostgreSQL outbox pattern (atomic with domain mutation); NATS JetStream durable streams | ✅ Implemented |
| Optimistic locking violation | `version` column on objects with 409 on conflict | ✅ Implemented |
| Approval replay | Immutable approval decisions; UNIQUE(approval_id, decided_by); executed_at marking | ✅ Implemented |
| Snapshot name injection | Regex validation `^[a-zA-Z0-9_-]{1,40}$` | ✅ Implemented |
| Approval payload tampering | `action_type` and `action_target` match verified at execution time | ✅ Implemented |

### Repudiation

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Action denial | Every mutation writes audit + outbox in same transaction | ✅ Implemented |
| Agent action denial | Agent effect results recorded with intention, decision, and outcome | ✅ Implemented |
| Approval decision denial | Immutable decisions; decided_by + decided_at + mfa_verified recorded | ✅ Implemented |
| Webhook receipt denial | Payload hash stored in audit (not raw payload) | ✅ Implemented |

### Information Disclosure

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Secret in audit_logs | Sanitized payloads — 15+ sensitive key patterns redacted before audit.Write | ✅ Verified |
| Secret in outbox_events | Sanitized payloads — same 15+ patterns redacted before outbox.Write | ✅ Verified |
| Secret in NATS events | Outbox worker publishes sanitized payload (5-field whitelist) | ✅ Verified |
| Secret in Redis fanout | WS fanout carries only entity identity (no titles/bodies) | ✅ Verified |
| TOTP secret exposure | AES-256-GCM encrypted at rest; raw secret shown ONCE at enrollment | ✅ Verified |
| Recovery code exposure | HMAC-SHA256 hashed (never stored plaintext); shown once at enrollment | ✅ Verified |
| Integration key exposure | HMAC-SHA256 hashed (never stored plaintext); shown once at creation | ✅ Verified |
| PII in context nodes | Properties always `{}` — no titles/bodies/summaries stored | ✅ Verified |
| Email enumeration | Forgot-password always returns success | ✅ Verified |
| Raw IP in audit | IP stored as HMAC-SHA256 using HMAC_KEY | ✅ Verified |
| Log secret exposure | `sanitizeForLog()` strips token=/secret= from Proxmox error messages | ✅ Verified |

### Denial of Service

| Threat | Mitigation | Status |
|--------|-----------|--------|
| API flood | Rate limiting on webhooks (per key-prefix, per source, per IP) | ✅ Implemented |
| MFA brute-force | 5 attempts → 15-min lockout | ✅ Implemented |
| Event storm | DLQ recursion guard (`clarity.dlq.*` events never spawn secondary DLQ) | ✅ Implemented |
| DB connection exhaustion | pgxpool with configurable max connections | ✅ Implemented |
| Request timeout | 30-second chi middleware timeout | ✅ Implemented |

### Elevation of Privilege

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Unauthorized route access | RequireAuth + RequirePermission middleware on every route | ✅ Implemented |
| Platform role escalation | Separate platform_roles + user_platform_roles tables | ✅ Implemented |
| Team role escalation | Team-scoped permissions (team_memberships → roles → permissions) | ✅ Implemented |
| Agent autonomy escalation | 13-check PolicyEvaluator chain; A5 hardcoded rejection | ✅ Implemented |
| Cross-team data access | Every query scoped by `team_id`; cross-team approval/action blocked | ✅ Implemented |
| Last-owner removal | Database trigger prevents removing last owner | ✅ Implemented |
| Self-approval | Self-approval prevented (requested_by ≠ decided_by) | ✅ Implemented |
| Unauthorized Proxmox mutation | Only 4 allowed actions; feature-flagged off by default | ✅ Implemented |
| Unrestricted agent action | Agents can only use granted tools; autonomy ≤ grant max ≤ agent max | ✅ Implemented |

## 4. Attack Surface Inventory

### Exposed Ports (Public)
| Port | Service | Auth |
|------|---------|------|
| 3000 | nginx (React SPA) | None (static files) |
| 8765 | Go API | JWT (most endpoints) |

### Exposed Ports (Private — Docker Network Only)
| Port | Service | Network |
|------|---------|---------|
| 5432 | PostgreSQL | clarityit-net only |
| 4222 | NATS | clarityit-net only |
| 6379 | Redis | clarityit-net only |
| 9000 | MinIO | clarityit-net only |

### Unauthenticated Endpoints
| Endpoint | Purpose | Protection |
|----------|---------|-----------|
| POST /api/bootstrap | First-user setup | Bootstrap lock (one-time) |
| POST /api/auth/login | Authentication | bcrypt + rate limiting |
| POST /api/auth/refresh | Token rotation | httpOnly cookie + reuse detection |
| POST /api/auth/forgot-password | Password reset | Always returns success (no enumeration) |
| POST /api/webhooks/{source} | Webhook receiver | Integration key + HMAC signature |
| GET /health | Health check | No sensitive data |

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation Status |
|------|-----------|--------|-------------------|
| Compromised Proxmox token | Low | Critical | Token only used by API; controlled mutation only; feature-flagged |
| Agent runs amok | Medium | High | A4 max autonomy; A5 disabled; tool grants required; approval + MFA for high/critical |
| Secret leakage in logs | Low | High | Sanitization at all persistence layers; tested |
| Cross-team data access | Low | High | team_id scoping on every query; tested |
| MFA bypass via timing | Low | Medium | 5-min window; server-side validation only |

## 6. Assumptions and Constraints

1. **Sovereign deployment**: Proxmox LXC, Cloudflare Tunnel — no cloud provider access to data
2. **Single-tenant**: One deployment per organization (for now)
3. **MIT/Apache-2.0 licensed**: Open-source; security through design, not obscurity
4. **No destructive autonomy**: Agents cannot delete/migrate/clone/firewall — only start/shutdown/stop/snapshot
5. **Human-in-the-loop**: All high-risk actions require human approval; agents propose, operators decide
