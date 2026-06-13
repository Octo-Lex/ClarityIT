# Clarity IT + Unified Enterprise IAM Synthesis
## Security, Identity & Authorization Architecture (2026)

> **Purpose**: A comprehensive study of the Unified Enterprise IAM Development Plan (v4.0), identifying what Clarity IT must adopt for production-grade security, and how to integrate enterprise IAM into the Clarity IT architecture.

---

## 1. What This Document Is

The Unified Enterprise IAM Development Plan v4.0 is a **production-grade, build-ready blueprint** for enterprise identity and access management. It is not a theoretical framework—it is a canonical implementation specification with:

- Complete PostgreSQL schema (20+ tables)
- Python authorization engine code
- SQL triggers and constraints
- API contract definitions
- UI blueprints
- Testing strategy (11 categories, 80+ test cases)
- Deployment and migration plan
- Risk register with 20+ identified risks
- 28-week implementation roadmap

**Key architectural principle**: Users do not have one global application role. Users receive team permissions only through active team membership or active temporary team access grants. Platform roles are separate and exist only for global administration.

---

## 2. Critical Assessment: What This IAM System Gets Right

### 2.1 Separation of Concerns
The document enforces a strict separation between:
- **Team authorization** (can user X do action Y in team T?)
- **Platform authorization** (can user X perform global operation Z?)
- **Temporary access** (JIT/break-glass grants with expiry)

This maps perfectly to Clarity IT's multi-module architecture where The Queue, The Project, The Wiki, The Hub, and The Grid each operate within team contexts.

### 2.2 The Triple-Layer Security Model

| Layer | Implementation | Clarity IT Application |
|-------|---------------|----------------------|
| **Authentication** | JWT access tokens (15min), rotating refresh tokens, MFA (TOTP), recovery codes | All Clarity IT modules |
| **Authorization** | Team RBAC + Platform roles + SoD + Temporary grants | The Queue (ITSM), The Project (PM), The Grid (infra) |
| **Audit** | Immutable, sanitized, append-only logs with HMAC identifiers | Context Engine audit trail |

### 2.3 Field-Level Security Innovations

| Feature | Implementation | Why It Matters |
|---------|---------------|----------------|
| **Column Masking** | `column_masking` table: context x user x field x visibility/editability | External stakeholders see only whitelisted fields |
| **Same-Team FK** | Composite FK `(role_id, team_id)` ensures roles never cross teams | Prevents accidental privilege escalation |
| **System Role Protection** | DB trigger prevents renaming/deleting system roles | Immutable baseline permissions |
| **Last Owner Protection** | Deferred DB trigger + service-level lock | Prevents orphaned teams |
| **Platform Owner Protection** | Cannot suspend/anonymize final platform_owner | Prevents platform lockout |

### 2.4 Privacy-First Design

| Storage Area | Raw PII Allowed? | Clarity IT Context |
|-------------|------------------|-------------------|
| `users` | Yes | Operational PII, scrubbed on anonymization |
| `audit_logs` | **No** | HMAC identifiers only, no emails/IPs/tokens |
| `idempotency_keys` | **No sensitive payloads** | Redacted responses, no token replay |
| `outbox_events` | **No raw PII in plaintext** | Encrypted sensitive payloads with TTL purge |
| `deleted_snapshots` | **No** | Sanitized or encrypted with destroyable keys |
| `webhook_deliveries` | **Sanitized only** | Response excerpts hashed |

This is the most rigorous privacy model I have seen in any IAM specification. Clarity IT must adopt this wholesale.

### 2.5 The Idempotency Contract

The document specifies a strict idempotency pattern:
- **Reservation-first**: Insert idempotency record BEFORE mutation
- **Fingerprint-based**: Same key + different request body = 409 Conflict
- **No token replay**: Login/refresh responses are never stored replayable
- **Processing lock**: 5-minute lease with recovery job

This is critical for Clarity IT's event-driven architecture where duplicate events are inevitable.

### 2.6 Outbox Worker Design

The outbox pattern is implemented with:
- **Encrypted sensitive payloads** (raw emails/tokens only in ciphertext)
- **Provider dedupe** (`provider_message_key` for external idempotency)
- **Exponential backoff** (2^attempts x 10 seconds, capped at 1 hour)
- **Dead-letter handling** (max 5 attempts, then visible for admin action)
- **Sensitive payload cleanup** (nulled after send/dead-letter/TTL)

This is the correct way to handle side effects in an event-driven system. Clarity IT's event mesh should adopt this pattern.

---

## 3. What Clarity IT Must Adopt

### 3.1 Immediate Adoptions (Use as-specified)

| IAM Feature | Clarity IT Module | Integration Point |
|------------|-------------------|-----------------|
| **Team RBAC** | All modules | Authorization middleware on every API endpoint |
| **Platform Roles** | Platform Admin | Global user management, separate from team permissions |
| **Temporary Access Grants** | The Queue, The Grid | JIT access for incident response, break-glass |
| **SoD Rules** | The Queue, The Project | Prevent conflicting permissions (e.g., approve + execute) |
| **Column Masking** | All modules | External stakeholder portals, client views |
| **Audit Logs** | Context Engine | Immutable, sanitized, append-only |
| **Idempotency** | Event Mesh | Reservation-first pattern for all mutating events |
| **Outbox Pattern** | Event Mesh | Encrypted payloads, provider dedupe, dead-letter |
| **Refresh Token Rotation** | Auth Service | Family tracking, reuse detection, session revocation |
| **MFA (TOTP)** | Auth Service | Encrypted secrets, recovery codes, admin reset |
| **Email Verification** | Auth Service | HMAC tokens, verified-email binding for invitations |
| **Password Reset** | Auth Service | HMAC tokens, single-use, session revocation on completion |
| **Rate Limiting** | API Gateway | IP HMAC, email HMAC, user_id, endpoint dimensions |
| **Bootstrap Lock** | Platform Admin | Race-safe first-user creation with platform_owner |

### 3.2 Adaptations (Modify for IT context)

| IAM Feature | Generic | Clarity IT Adaptation |
|------------|---------|----------------------|
| **Team Roles** | Owner, Admin, Manager, Member, Viewer | Add `on_call_engineer`, `security_admin`, `auditor` |
| **Permissions** | Generic (team:read, project:create) | IT-specific (incident:respond, runbook:execute, alert:acknowledge) |
| **Temporary Grants** | Generic JIT access | Incident-specific: auto-grant on P1 alert, auto-revoke on resolution |
| **SoD Rules** | Generic (permission_a + permission_b) | IT-specific: `alert:acknowledge` + `alert:remediate` cannot coexist |
| **Audit Actions** | Generic (team.created, user.suspended) | IT-specific (incident.resolved, alert.acknowledged, runbook.executed) |
| **High-Risk Actions** | Generic (MFA disable, password change) | IT-specific (infrastructure restart, certificate rotation, firewall change) |
| **Anonymization** | Generic user scrub | IT-specific: preserve incident attribution ("anonymized user resolved #4821") |
| **Session Context** | Generic IP/user-agent | Add `proxmox_vm_id`, `alert_id`, `incident_id` for infrastructure sessions |

### 3.3 Rejections (Not needed for Clarity IT)

| IAM Feature | Why Reject | Clarity IT Alternative |
|------------|-----------|----------------------|
| **SAML/SCIM** | Deferred scope | Keep schema extensible but don't implement yet |
| **Service Principals** | Deferred scope | API keys for integrations instead |
| **ReBAC** | Overkill for v1 | RBAC + SoD sufficient for initial release |
| **Customer-Managed Keys** | Enterprise feature | Platform-managed keys with rotation policy |
| **Support Impersonation** | Security risk | Never implement; use audit trail instead |
| **Billing Integration** | Not IT-focused | IT Cost Tracking module instead |
| **Advanced Policy Engine** | Overkill for v1 | SoD rules + custom roles sufficient |

---

## 4. The Unified IAM + Clarity IT Architecture

### 4.1 Layered Security Architecture

```
CLIENT LAYER
  Web · PWA · Desktop (Tauri) · CLI · Slack/Teams Bots

EDGE / GATEWAY
  Cloudflare Workers · WAF · Rate Limiting · Auth · Cache

API GATEWAY (Go)
  JWT Validation · Session Introspection · Idempotency Check
  Rate Limiting · Request Fingerprinting · Correlation IDs

AUTHORIZATION ENGINE
  Team Auth (RBAC + SoD) · Platform Auth (Hierarchy) · Temporary Access Grants

IAM DOMAIN SERVICES
  Users · Teams · Roles · Permissions · Memberships · Invitations
  Sessions · Refresh Tokens · MFA · Platform Roles · Audit

CLARITY IT MODULES
  The Queue · The Project · The Wiki · The Hub · The Grid
  Each module checks team permissions before any action

CONTEXT ENGINE
  Ingest · Correlate · Graph · Serve (with audit trail)

AI AGENT MESH
  Orchestrator · Queue · Project · Ops · Alert · Doc
  Agents check permissions via Authorization Engine

EVENT MESH (NATS)
  All events carry actor_id, team_id, correlation_id, request_id

DATA LAYER
  PostgreSQL 16 · SQLite (D1/Local) · Redis · S3/R2 · Git
  Audit logs partitioned · Encrypted outbox · HMAC identifiers
```

### 4.2 Key Integration Points

| Integration | How It Works |
|------------|-------------|
| **Auth to Event Mesh** | Every login/logout/refresh publishes auth events |
| **Auth to Context Engine** | Session creation updates user context |
| **Auth to Agents** | Agent actions require valid session; high-risk requires recent MFA |
| **Team Auth to Modules** | Every module endpoint calls permission check before execution |
| **Team Auth to Agents** | Agents operate with service principal + team context |
| **Audit to Context Engine** | Audit events feed into context graph |
| **Audit to Outbox** | Audit events are immutable; outbox handles side effects |
| **Idempotency to Event Mesh** | Duplicate events detected by key + fingerprint |
| **SoD to Agents** | Agent cannot execute conflicting actions |
| **Column Masking to Views** | View engine applies masks based on user role + context |

---

## 5. The IAM-Enabled Clarity IT Data Model

### 5.1 Core Tables (from IAM spec, adapted)

```yaml
users:
  id: UUID PRIMARY KEY
  email: CITEXT NOT NULL
  email_canonical: TEXT NOT NULL
  email_verified: BOOLEAN DEFAULT FALSE
  display_name: TEXT
  password_hash: TEXT
  mfa_enabled: BOOLEAN DEFAULT FALSE
  mfa_secret_ciphertext: BYTEA
  status: ENUM(active, suspended, anonymized)
  failed_login_count: INT DEFAULT 0
  locked_until: TIMESTAMPTZ
  last_login_at: TIMESTAMPTZ
  last_activity_at: TIMESTAMPTZ
  version: INT DEFAULT 1
  deleted_at: TIMESTAMPTZ

teams:
  id: UUID PRIMARY KEY
  name: TEXT NOT NULL
  slug: CITEXT NOT NULL
  status: ENUM(active, suspended, deleted)
  created_by: UUID REFERENCES users
  version: INT DEFAULT 1
  deleted_at: TIMESTAMPTZ

roles:
  id: UUID PRIMARY KEY
  team_id: UUID REFERENCES teams
  name: TEXT NOT NULL
  description: TEXT
  is_system_role: BOOLEAN DEFAULT FALSE
  deleted_at: TIMESTAMPTZ
  version: INT DEFAULT 1

permissions:
  id: UUID PRIMARY KEY
  resource: TEXT NOT NULL
  action: TEXT NOT NULL
  description: TEXT

role_permissions:
  role_id: UUID REFERENCES roles
  permission_id: UUID REFERENCES permissions
  PRIMARY KEY (role_id, permission_id)

team_memberships:
  id: UUID PRIMARY KEY
  team_id: UUID REFERENCES teams
  user_id: UUID REFERENCES users
  role_id: UUID REFERENCES roles
  invitation_id: UUID REFERENCES invitations
  status: ENUM(active, removed)
  removed_at: TIMESTAMPTZ
  removed_by: UUID REFERENCES users
  version: INT DEFAULT 1

team_access_grants:
  id: UUID PRIMARY KEY
  team_id: UUID REFERENCES teams
  user_id: UUID REFERENCES users
  role_id: UUID REFERENCES roles
  reason_code: TEXT NOT NULL
  ticket_reference_hmac: TEXT
  granted_by: UUID REFERENCES users
  granted_at: TIMESTAMPTZ
  expires_at: TIMESTAMPTZ
  revoked_at: TIMESTAMPTZ
  revoked_by: UUID REFERENCES users
  version: INT DEFAULT 1

platform_roles:
  id: UUID PRIMARY KEY
  name: TEXT UNIQUE NOT NULL
  level: INT UNIQUE NOT NULL

user_platform_roles:
  user_id: UUID REFERENCES users
  platform_role_id: UUID REFERENCES platform_roles
  assigned_by: UUID REFERENCES users
  assigned_at: TIMESTAMPTZ
  PRIMARY KEY (user_id, platform_role_id)

user_sessions:
  id: UUID PRIMARY KEY
  user_id: UUID REFERENCES users
  access_token_jti: TEXT UNIQUE
  refresh_token_family: UUID
  ip_address: INET
  user_agent: TEXT
  device_label: TEXT
  is_revoked: BOOLEAN DEFAULT FALSE
  access_expires_at: TIMESTAMPTZ
  refresh_expires_at: TIMESTAMPTZ
  last_activity_at: TIMESTAMPTZ

refresh_tokens:
  id: UUID PRIMARY KEY
  session_id: UUID REFERENCES user_sessions
  user_id: UUID REFERENCES users
  token_family: UUID
  token_hash: TEXT UNIQUE
  previous_token_id: UUID REFERENCES refresh_tokens
  replaced_by_token_id: UUID REFERENCES refresh_tokens
  issued_at: TIMESTAMPTZ
  expires_at: TIMESTAMPTZ
  used_at: TIMESTAMPTZ
  rotated_at: TIMESTAMPTZ
  revoked_at: TIMESTAMPTZ
  reuse_detected_at: TIMESTAMPTZ

audit_logs:
  id: BIGSERIAL
  event_id: UUID DEFAULT gen_random_uuid()
  actor_id: UUID
  actor_type: ENUM(user, system, service_principal)
  team_id: UUID
  action: TEXT NOT NULL
  entity_type: TEXT NOT NULL
  entity_id: UUID NOT NULL
  old_value: JSONB
  new_value: JSONB
  change_summary: TEXT NOT NULL
  ip_hmac: TEXT
  user_agent_hmac: TEXT
  hmac_key_id: TEXT
  idempotency_key: TEXT
  request_id: UUID
  correlation_id: UUID
  created_at: TIMESTAMPTZ

idempotency_keys:
  scope_type: ENUM(user, anonymous, system)
  scope_id: TEXT
  key: TEXT
  request_method: TEXT
  request_path: TEXT
  request_fingerprint: TEXT
  status: ENUM(processing, completed, failed)
  response_code: INT
  response_body: JSONB
  response_body_ciphertext: BYTEA
  error_code: TEXT
  locked_until: TIMESTAMPTZ
  expires_at: TIMESTAMPTZ
  PRIMARY KEY (scope_type, scope_id, key)

outbox_events:
  id: UUID PRIMARY KEY
  event_type: TEXT
  aggregate_type: TEXT
  aggregate_id: UUID
  payload: JSONB
  sensitive_payload_ciphertext: BYTEA
  sensitive_payload_key_id: TEXT
  provider_message_key: TEXT
  status: ENUM(pending, processing, sent, failed, dead_letter)
  attempts: INT DEFAULT 0
  max_attempts: INT DEFAULT 5
  next_attempt_at: TIMESTAMPTZ
  locked_at: TIMESTAMPTZ
  locked_by: TEXT
  last_error: TEXT
  purge_after: TIMESTAMPTZ

sod_conflict_rules:
  id: UUID PRIMARY KEY
  team_id: UUID REFERENCES teams
  name: TEXT NOT NULL
  permission_a_id: UUID REFERENCES permissions
  permission_b_id: UUID REFERENCES permissions
  is_active: BOOLEAN DEFAULT TRUE

column_masking:
  context_id: UUID
  user_id: UUID
  field_name: TEXT
  permission: ENUM(visible, editable, hidden)
  redaction_text: TEXT
```

### 5.2 Clarity IT-Specific Extensions

```yaml
permissions_it:
  - resource: incident, action: create
  - resource: incident, action: respond
  - resource: incident, action: resolve
  - resource: incident, action: escalate
  - resource: alert, action: acknowledge
  - resource: alert, action: remediate
  - resource: runbook, action: read
  - resource: runbook, action: execute
  - resource: vm, action: read
  - resource: vm, action: start
  - resource: vm, action: stop
  - resource: vm, action: restart
  - resource: vm, action: migrate
  - resource: certificate, action: read
  - resource: certificate, action: rotate
  - resource: backup, action: read
  - resource: backup, action: trigger
  - resource: network, action: read
  - resource: network, action: modify

roles_it:
  - name: on_call_engineer
    permissions: [incident:respond, alert:acknowledge, runbook:execute, vm:read]
  - name: security_admin
    permissions: [audit_log:read, sod_rule:create, sod_rule:update, role:create, role:update]
  - name: infrastructure_engineer
    permissions: [vm:start, vm:stop, vm:restart, vm:migrate, certificate:rotate, backup:trigger, network:modify]
  - name: auditor
    permissions: [audit_log:read, team:read, project:read, incident:read, alert:read]

incident_access_grants:
  id: UUID PRIMARY KEY
  incident_id: UUID REFERENCES incidents
  user_id: UUID REFERENCES users
  role_id: UUID REFERENCES roles
  granted_by: UUID REFERENCES users
  granted_at: TIMESTAMPTZ
  auto_revoke_on_resolution: BOOLEAN DEFAULT TRUE
  revoked_at: TIMESTAMPTZ

infra_sessions:
  id: UUID PRIMARY KEY
  session_id: UUID REFERENCES user_sessions
  proxmox_vm_id: TEXT
  alert_id: UUID REFERENCES alerts
  incident_id: UUID REFERENCES incidents
  runbook_id: UUID REFERENCES runbooks
```

---

## 6. The Authorization Engine for Clarity IT

### 6.1 Team Permission Check (extended)

```python
def check_team_permission(db, user_id, team_id, resource, action, context=None):
    # Base permission check (from IAM spec)
    result = db.query_one(
        "SELECT EXISTS ("
        "  SELECT 1 FROM users u"
        "  JOIN team_memberships tm ON tm.user_id = u.id"
        "  JOIN teams t ON t.id = tm.team_id"
        "  JOIN roles r ON r.id = tm.role_id"
        "  JOIN role_permissions rp ON rp.role_id = r.id"
        "  JOIN permissions p ON p.id = rp.permission_id"
        "  WHERE u.id = :user_id AND u.status = 'active' AND u.deleted_at IS NULL"
        "    AND t.id = :team_id AND t.status = 'active' AND t.deleted_at IS NULL"
        "    AND tm.status = 'active' AND r.team_id = t.id AND r.deleted_at IS NULL"
        "    AND p.resource = :resource AND p.action = :action"
        "  UNION"
        "  SELECT 1 FROM users u"
        "  JOIN team_access_grants tag ON tag.user_id = u.id"
        "  JOIN teams t ON t.id = tag.team_id"
        "  JOIN roles r ON r.id = tag.role_id"
        "  JOIN role_permissions rp ON rp.role_id = r.id"
        "  JOIN permissions p ON p.id = rp.permission_id"
        "  WHERE u.id = :user_id AND u.status = 'active' AND u.deleted_at IS NULL"
        "    AND t.id = :team_id AND t.status = 'active' AND t.deleted_at IS NULL"
        "    AND tag.revoked_at IS NULL AND tag.expires_at > NOW()"
        "    AND r.team_id = t.id AND r.deleted_at IS NULL"
        "    AND p.resource = :resource AND p.action = :action"
        "  UNION"
        "  SELECT 1 FROM users u"
        "  JOIN incident_access_grants iag ON iag.user_id = u.id"
        "  JOIN incidents i ON i.id = iag.incident_id"
        "  JOIN teams t ON t.id = i.team_id"
        "  JOIN roles r ON r.id = iag.role_id"
        "  JOIN role_permissions rp ON rp.role_id = r.id"
        "  JOIN permissions p ON p.id = rp.permission_id"
        "  WHERE u.id = :user_id AND u.status = 'active' AND u.deleted_at IS NULL"
        "    AND t.id = :team_id AND t.status = 'active' AND t.deleted_at IS NULL"
        "    AND iag.revoked_at IS NULL AND i.status = 'active'"
        "    AND r.team_id = t.id AND r.deleted_at IS NULL"
        "    AND p.resource = :resource AND p.action = :action"
        ") AS authorized",
        {"user_id": user_id, "team_id": team_id, "resource": resource, "action": action}
    )

    if not result or not result["authorized"]:
        raise PermissionDenied(f"Missing permission {resource}:{action}")

    # SoD check
    if context and context.get("check_sod", True):
        effective_permissions = get_effective_permission_ids(db, user_id=user_id, team_id=team_id)
        enforce_sod(db, team_id=team_id, user_id=user_id, proposed_permission_ids=effective_permissions)

    return True
```

### 6.2 High-Risk Action Check (Clarity IT extension)

```python
HIGH_RISK_ACTIONS = {
    ("vm", "restart"): True,
    ("vm", "migrate"): True,
    ("certificate", "rotate"): True,
    ("network", "modify"): True,
    ("incident", "escalate"): True,
    ("alert", "remediate"): True,
    ("runbook", "execute"): True,
}

def check_high_risk_action(db, user_id, team_id, resource, action, session_id):
    if (resource, action) not in HIGH_RISK_ACTIONS:
        return True

    session = db.query_one(
        "SELECT mfa_verified_at FROM user_sessions"
        " WHERE id = :session_id AND user_id = :user_id"
        "   AND is_revoked = FALSE"
        "   AND mfa_verified_at > NOW() - INTERVAL '15 minutes'",
        {"session_id": session_id, "user_id": user_id}
    )

    if not session:
        raise PermissionDenied("High-risk action requires recent MFA verification")

    return True
```

---

## 7. The Audit-First Context Engine

### 7.1 Integration: Audit Logs to Context Graph

Every audit event feeds into the Context Engine:

```python
def audit_with_context(db, action, actor_id, team_id, entity_type, entity_id, old_value, new_value, request_context):
    # 1. Write sanitized audit log
    audit_event = write_audit_log(db, action=action, actor_id=actor_id, team_id=team_id,
                                   entity_type=entity_type, entity_id=entity_id,
                                   old_value=old_value, new_value=new_value,
                                   request_context=request_context)

    # 2. Publish to Context Engine
    context_event = {
        "type": "audit.event_created",
        "audit_event_id": audit_event["id"],
        "actor_id": actor_id,
        "team_id": team_id,
        "entity_type": entity_type,
        "entity_id": entity_id,
        "action": action,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "correlation_id": request_context.get("correlation_id"),
        "request_id": request_context.get("request_id"),
    }
    event_mesh.publish("ops.audit.*", context_event)

    # 3. Update context graph
    context_engine.ingest({
        "type": "entity.action_taken",
        "entity_type": entity_type,
        "entity_id": entity_id,
        "actor_id": actor_id,
        "action": action,
        "team_id": team_id,
        "timestamp": context_event["timestamp"],
    })
```

### 7.2 Context Bundle with Audit History

```python
def build_context_bundle_with_audit(db, user_id, team_id, subject_type, subject_id):
    bundle = build_base_context_bundle(db, user_id=user_id, team_id=team_id,
                                        subject_type=subject_type, subject_id=subject_id)

    recent_audits = db.query(
        "SELECT action, actor_id, created_at, change_summary"
        " FROM audit_logs"
        " WHERE entity_type = :subject_type AND entity_id = :subject_id"
        "   AND created_at > NOW() - INTERVAL '7 days'"
        " ORDER BY created_at DESC LIMIT 10",
        {"subject_type": subject_type, "subject_id": subject_id}
    )

    bundle["audit_history"] = [
        {"action": a["action"], "actor_id": a["actor_id"],
         "timestamp": a["created_at"], "summary": a["change_summary"]}
        for a in recent_audits
    ]

    return bundle
```

---

## 8. Implementation Roadmap (IAM + Clarity IT)

### Phase 1: Foundation (Weeks 1-4)
- PostgreSQL schema (all IAM tables + Clarity IT extensions)
- Bootstrap lock + first-user creation
- User registration, login, refresh, logout
- Email verification + password reset
- Refresh token rotation + reuse detection
- Idempotency middleware
- Sanitized audit writer
- Outbox worker with encryption
- Event mesh setup (NATS)
- Context Engine scaffold

### Phase 2: Team & Authorization (Weeks 5-8)
- Team creation with concrete roles
- Role permission matrix
- Team membership management
- Invitation system
- Last-owner protection
- Team-scoped authorization middleware
- The Queue module (Kanban, SLA)
- The Hub module (channels, DMs)

### Phase 3: Security Hardening (Weeks 9-12)
- MFA setup, enrollment, verification, recovery
- Password change + reset completion
- Session management + revocation
- Rate limiting
- High-risk action checks
- SoD conflict rules
- Temporary access grants
- The Project module (Gantt, sprints)
- The Wiki module (block editor, Git sync)

### Phase 4: Platform Admin & Operations (Weeks 13-16)
- Platform roles
- Last platform owner protection
- User suspension, reactivation, anonymization
- Global session revocation
- Global audit viewer
- Background jobs
- The Grid module (Proxmox, alerts)
- Agent mesh (Orchestrator, Queue, Project)

### Phase 5: AI & Intelligence (Weeks 17-20)
- Context Engine full implementation
- AI agents with permission checks
- Agent-to-agent communication (A2A)
- Recovery Mode
- Mental Load Zero-Out
- All 6 agents operational
- NLP pipeline with disambiguation chips

### Phase 6: Edge & Scale (Weeks 21-24)
- Cloudflare Workers deployment
- Edge caching + D1 integration
- Webhook subscriptions + delivery
- Plugin SDK
- Performance optimization
- Security audit
- Documentation

---

## 9. Risk Register (IAM + Clarity IT)

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Permission bug | Privilege escalation | Central auth engine, DB constraints, 80+ tests |
| Raw token leakage | Account compromise | HMAC tokens, encrypted outbox, no prefixes |
| Idempotency race | Duplicate mutations | Insert-first reservation, PK on scope/key |
| Last owner removed | Orphaned team | Service lock + deferred DB trigger |
| Last platform owner removed | Platform lockout | Service lock + deferred DB trigger |
| Audit contains PII | Compliance failure | Allowlist sanitizer + tests + restricted DB grants |
| Refresh token reuse missed | Session hijack | Family tracking + session revocation |
| Agent over-permissioning | Unauthorized actions | Agents subject to same permission checks as users |
| Context Engine data leakage | Privacy breach | Same HMAC rules as audit logs |
| Event mesh duplicate events | Inconsistent state | Idempotency + CRDT merge |
| Outbox duplicate send | Duplicate emails/webhooks | provider_message_key + dedupe |
| Worker crash | Stuck side effects | Lock lease + recovery job |
| High-risk action without MFA | Infrastructure damage | Recent MFA check for all infra actions |
| SoD violation by agent | Conflicting actions | SoD checks before agent execution |
| Anonymization breaks audit trail | Compliance failure | Preserve HMAC identifiers, anonymize only mutable PII |

---

## 10. Final Position

The Unified Enterprise IAM Development Plan v4.0 is the most rigorous, production-ready IAM specification I have analyzed. It combines:

- **Academic rigor** (CRDTs, Lamport timestamps, vector clocks)
- **Enterprise pragmatism** (SoD, platform roles, audit trails)
- **Privacy-by-design** (HMAC identifiers, encrypted outbox, no PII in immutable storage)
- **Operational maturity** (background jobs, observability, rollback strategy)

For Clarity IT, this is not optional. A platform that manages IT infrastructure, handles incidents, and executes runbooks must have this level of security. The IAM system is the foundation upon which everything else rests.

**The synthesis creates a platform that is:**
- **Secure**: Enterprise-grade IAM with field-level permissions and audit trails
- **Intelligent**: AI agents with full situational awareness and proper authorization
- **Resilient**: Event-driven, offline-capable, with conflict-free replication
- **Compliant**: Privacy-preserving, with immutable audit logs and data classification
- **Scalable**: Edge-deployable, with horizontal scaling and zero-knowledge relay

---

*Document Version: 1.0 | Generated: 2026-06-12 | Status: Synthesis Complete*
