# Unified Enterprise IAM Development Plan — Best Combined Build-Ready Version

**Version:** 4.0 — best-combined canonical baseline  
**Purpose:** Production-grade IAM and authorization platform for a team-based enterprise application  
**Status:** Recommended implementation blueprint, consolidating the strongest parts of the two prior versions

---

## Document Position

This version combines the best elements of both prior documents.

It adopts the build-ready structure and operational maturity of the canonical combined version while restoring the stricter least-privilege posture and several useful schema safeguards from the earlier master blueprint.

Key decisions retained from the canonical version:

```text
- explicit data-classification model
- corrected idempotency primary key and request fingerprint behavior
- encrypted webhook signing secrets
- token status and attempt tracking
- stronger outbox worker semantics
- rollback strategy
- expanded platform, SoD, webhook, idempotency, load, and privacy tests
- clearer API permissions
- key-management requirements
- full risk register
```

Key decisions restored or strengthened from the earlier master blueprint:

```text
- stricter default team-admin permission model
- explicit role-permission matrix as the source of truth
- provider_message_key / external delivery dedupe support in outbox_events
- request_id and correlation_id in audit logs
- persistent login lockout fields on users
- stricter duplicate invitation policy by team + invited email
- explicit no-PII audit allowlist discipline
- stricter platform-owner and last-team-owner service locking guidance
```

The controlling architectural rule remains:

```text
Users do not have one global application role.

Users receive team permissions only through:
1. active team membership, or
2. active temporary team access grant.

Platform roles are separate and exist only for global administration.
```

Example:

```text
User A = owner in Team Alpha
User A = viewer in Team Beta
User A = platform_admin globally
```

These are three separate authorization contexts.

---

# 1. Executive Summary

Build a production-grade enterprise IAM system supporting:

```text
User registration and authentication
Email verification
Password reset and password change
MFA enrollment, verification, recovery, and administrative reset
Short-lived access-token sessions
Refresh-token rotation and reuse detection
Session visibility and revocation
Team creation and team-scoped authorization
Concrete per-team roles and explicit permissions
Team invitations with verified-email binding
Temporary team access grants for JIT / break-glass workflows
Custom team roles
Separation-of-duties policy checks
Platform administration separated from team permissions
Platform role hierarchy and last-owner protection
Privacy-preserving immutable audit logs
Strict idempotency for mutating business APIs
Transactional encrypted outbox side effects
Webhook subscriptions and webhook delivery tracking
User suspension, reactivation, anonymization, and session revocation
Operational cleanup, recovery, retention, and security jobs
Observability, alerting, and structured redacted logs
Future enterprise extensions such as SAML, SCIM, service principals, API keys, ReBAC, and customer-managed keys
```

Core principle:

```text
Team role != platform role.
Platform role != team membership.
Authorization is always evaluated in the correct context.
```

---

# 2. Scope

## 2.1 Core Production Scope

| Area | Included |
|---|---|
| Authentication | Register, login, refresh, logout, password reset, password change |
| Email verification | Request verification, confirm verification, enforce verified email for invitation acceptance |
| MFA | Setup, enroll, verify, recovery codes, regenerate codes, disable, admin reset |
| Sessions | Active session visibility, per-session revocation, revoke all, high-risk introspection |
| Refresh tokens | Rotation, token family tracking, reuse detection, family/session revocation |
| Teams | Create, read, update, soft delete |
| Team RBAC | Owner, admin, manager, member, viewer, custom roles |
| Team members | List, role change, soft removal, membership history |
| Invitations | Create, list, resend, revoke/cancel, accept, expire |
| Temporary access | Grant, revoke, expire, audit, SoD evaluation |
| Platform admin | User list, suspend, reactivate, anonymize, revoke sessions, global audit |
| Platform roles | platform_support, platform_admin, platform_owner, hierarchy, last-owner protection |
| Audit logs | Sanitized append-only team and global audit |
| Idempotency | Required for mutating business-resource APIs |
| Outbox | Encrypted sensitive payloads, provider dedupe, retries, backoff, dead-letter, cleanup |
| Webhooks | Subscriptions, encrypted signing secrets, deliveries, retries, dead-letter visibility |
| Operations | Cleanup, recovery, dormant suspension, audit partitioning, retention |

## 2.2 Deferred Enterprise Scope

| Feature | Status |
|---|---|
| SAML SSO | Deferred until customer requirement |
| SCIM provisioning | Deferred until customer requirement |
| Service principals | Deferred |
| API keys | Deferred |
| ReBAC / relationship-based authorization | Deferred |
| Customer-managed keys | Deferred |
| Advanced policy engine | Deferred |
| Enterprise audit export | Deferred |
| Support impersonation | Avoid unless absolutely required; if added later, require separate high-risk design |

Deferred features must not block the production core system, but schema and boundaries should avoid making them difficult later.

---

# 3. Non-Negotiable Architecture Rules

| Area | Final Rule |
|---|---|
| Team-scoped authorization | Every team action requires `actor_user_id`, `team_id`, `resource`, and `action`. |
| Platform/team separation | Platform roles grant only global operations. They never imply team membership or team resource access. |
| Explicit grants | Permissions are granted through explicit role-permission rows, not broad rules such as “all read actions.” |
| Active-user enforcement | Suspended, anonymized, or deleted users cannot authorize actions. |
| Active-team enforcement | Deleted or suspended teams cannot authorize normal team operations. |
| Concrete team roles | Template roles are copied into concrete team roles during team creation. Memberships, invitations, and grants never reference template roles. |
| Same-team role enforcement | Memberships, invitations, and grants may only reference roles belonging to the same team. Enforce with composite foreign keys. |
| Last team-owner protection | A live team must always have at least one active owner unless the team is being deleted. Enforce in service code and deferred DB trigger. |
| Last platform-owner protection | The platform must always have at least one active `platform_owner`. Prevent removal, suspension, or anonymization of the final active owner. |
| Audit-first state changes | Every mutation writes sanitized audit evidence in the same database transaction. |
| Immutable audit | Audit events are append-only forensic evidence, not source-of-truth state. |
| No raw PII in immutable storage | Immutable stores must not contain raw emails, display names, IPs, user agents, tokens, secrets, cookies, authorization headers, or free-text PII. |
| HMAC identifiers | Use keyed HMAC for emails, IPs, user agents, tokens, ticket references, and low-entropy identifiers. |
| Outbox side effects | Emails, webhooks, and notifications are written to outbox inside the transaction and dispatched only after commit. |
| Sensitive outbox payloads | Raw emails and one-time tokens may appear only in encrypted sensitive outbox payloads with TTL cleanup. |
| Webhook signing secrets | Store webhook signing secrets encrypted. Hash-only storage is insufficient because the dispatcher must sign outbound requests. |
| Strict idempotency | Same scope + same key + different request fingerprint returns `409 Conflict`. |
| Idempotency reservation first | Reserve idempotency before mutation. Never write idempotency only after mutation. |
| Token issuance safety | Do not replay raw login or refresh token responses from idempotency storage. |
| Optimistic locking | Concurrently mutable records carry `version`; updates use `If-Match` or `expected_version`. |
| Invitation email binding | Invitation acceptance requires authenticated user’s verified canonical email to match the invited canonical email. |
| Invitation duplicate policy | Only one pending invitation may exist per team + canonical email unless a deliberate product exception is approved. |
| Role lifecycle safety | System roles cannot be renamed or soft-deleted through normal APIs. Custom roles are soft-deleted once referenced. |
| Deleted snapshots | Snapshots must be sanitized or encrypted with destroyable keys and retention controls. |
| Frontend trust boundary | UI hides unavailable actions, but backend and DB enforce all authorization and invariants. |

---

# 4. High-Level Architecture

```text
Frontend UI
   |
   v
API Gateway / Backend API
   |
   +--> Auth Service
   |       - registration
   |       - email verification
   |       - login
   |       - refresh
   |       - logout
   |       - password reset/change
   |       - MFA
   |
   +--> Idempotency Middleware
   |
   +--> Authorization Engine
   |       - team permission checks
   |       - temporary access grants
   |       - platform role checks
   |       - SoD enforcement
   |
   +--> IAM Domain Services
   |       - users
   |       - teams
   |       - roles
   |       - permissions
   |       - memberships
   |       - invitations
   |       - temporary access grants
   |       - sessions
   |       - platform roles
   |
   +--> Audit Writer
   |
   +--> Transactional Outbox Publisher
   |
   +--> PostgreSQL
           - users
           - email verification tokens
           - password reset tokens
           - mfa recovery codes
           - teams
           - roles
           - permissions
           - role permissions
           - invitations
           - team memberships
           - temporary access grants
           - platform roles
           - user platform roles
           - sessions
           - refresh tokens
           - audit logs
           - idempotency keys
           - outbox events
           - webhook subscriptions
           - webhook deliveries
           - SoD rules
           - deleted snapshots

Outbox Worker
   |
   +--> Email Service
   +--> Webhook Dispatcher
   +--> Notification Service

Operational Workers
   |
   +--> Invitation expiry
   +--> Verification/reset token expiry
   +--> Idempotency cleanup/recovery
   +--> Outbox recovery/dead-letter cleanup
   +--> Session cleanup
   +--> Refresh-token cleanup
   +--> Temporary access expiry
   +--> Dormant suspension
   +--> Audit partition maintenance
   +--> Deleted snapshot retention / crypto-shred
```

---

# 5. Data Classification and Privacy Model

| Storage area | Raw PII allowed? | Notes |
|---|---:|---|
| `users` | Yes | Mutable operational PII. Must be scrubbed during anonymization. |
| `invitations` | Yes | Mutable operational PII. Raw email allowed; audit stores only HMAC. |
| `user_sessions` | Yes | IP and user-agent allowed for security. Scrub during anonymization. |
| `audit_logs` | No | Store IDs, HMACs, safe enums, reason codes, sanitized JSON only. |
| `idempotency_keys.response_body` | No sensitive payloads | Store redacted response bodies only. Never store raw tokens. |
| `idempotency_keys.response_body_ciphertext` | Conditional | Encrypted safe response data only; token issuance responses must not be replayed. |
| `outbox_events.payload` | No raw PII | Plain payload must be sanitized. |
| `outbox_events.sensitive_payload_ciphertext` | Yes, encrypted | Raw email/token allowed only while needed and purged after send/dead-letter/TTL. |
| `deleted_entity_snapshots.snapshot_sanitized` | No | Must pass PII validator. |
| `deleted_entity_snapshots.snapshot_ciphertext` | Yes, encrypted | Key must be destroyable for retention/crypto-shred. |
| `webhook_deliveries` | Sanitized only | Response excerpts sanitized; full bodies not retained unless separately encrypted. |

Use keyed HMAC instead of plain SHA-256 for low-entropy or sensitive identifiers.

---

# 6. Authorization Model

## 6.1 Team Authorization

Team authorization answers:

```text
Can user X perform action Y on resource Z inside team T?
```

It depends on:

```text
users
teams
team_memberships
team_access_grants
roles
role_permissions
permissions
```

Required checks:

```text
User is active.
User is not deleted.
Team is active.
Team is not deleted.
Membership or grant belongs to the requested team.
Membership or grant is active.
Temporary grant has not expired or been revoked.
Role belongs to the same team.
Role is not deleted.
Role grants requested permission.
```

## 6.2 Platform Authorization

Platform authorization answers:

```text
Can user X perform global operation Y?
```

Examples:

```text
Suspend user
Reactivate user
Anonymize user
View global audit logs
Revoke global sessions
Assign platform role
Remove platform role
```

Platform hierarchy:

| Role | Level | Purpose |
|---|---:|---|
| `platform_support` | 40 | Read/support-only global operations |
| `platform_admin` | 80 | User lifecycle, global audit, global sessions |
| `platform_owner` | 100 | Full global control, including platform role assignment/removal |

Rule:

```text
platform_owner >= platform_admin >= platform_support
```

Platform roles do not grant team permissions.

---

# 7. Role Model

## 7.1 System Team Roles

| Role | Purpose |
|---|---|
| `owner` | Full team control, including team deletion. Cannot be removed/demoted if final owner. |
| `admin` | Manage operational team settings, members, invitations, projects, billing, and webhooks. Cannot delete team, manage roles, create/revoke temporary grants, or read audit logs by default. |
| `manager` | Manage projects and view members. |
| `member` | Create, read, and update project content. |
| `viewer` | Read project content and minimal team metadata. |

## 7.2 Custom Team Roles

```text
Custom roles are scoped to one team.
Custom roles cannot use reserved system role names unless created by system seed logic.
Custom roles can be soft-deleted.
Custom roles cannot be deleted while assigned to active memberships or active temporary grants.
Custom role permission changes must pass SoD validation.
```

## 7.3 Explicit Team Permission Matrix

This matrix is the default seed policy. Do not seed viewer with all `read` actions. Every permission assignment must be explicit.

| Permission | Owner | Admin | Manager | Member | Viewer |
|---|---:|---:|---:|---:|---:|
| `team:read` | yes | yes | yes | yes | yes |
| `team:update` | yes | yes | no | no | no |
| `team:delete` | yes | no | no | no | no |
| `team_member:invite` | yes | yes | no | no | no |
| `team_member:read` | yes | yes | yes | no | no |
| `team_member:update_role` | yes | yes | no | no | no |
| `team_member:remove` | yes | yes | no | no | no |
| `invitation:read` | yes | yes | no | no | no |
| `role:read` | yes | yes | yes | no | no |
| `role:create` | yes | no | no | no | no |
| `role:update` | yes | no | no | no | no |
| `role:delete` | yes | no | no | no | no |
| `access_grant:create` | yes | no | no | no | no |
| `access_grant:read` | yes | yes | no | no | no |
| `access_grant:revoke` | yes | no | no | no | no |
| `project:create` | yes | yes | yes | yes | no |
| `project:read` | yes | yes | yes | yes | yes |
| `project:update` | yes | yes | yes | yes | no |
| `project:delete` | yes | yes | yes | no | no |
| `billing:read` | yes | yes | no | no | no |
| `billing:update` | yes | yes | no | no | no |
| `audit_log:read` | yes | no | no | no | no |
| `webhook:create` | yes | yes | no | no | no |
| `webhook:read` | yes | yes | no | no | no |
| `webhook:update` | yes | yes | no | no | no |
| `webhook:delete` | yes | yes | no | no | no |

Product override option:

```text
If the product explicitly wants team admins to manage custom roles, temporary grants, or audit logs, create a separate elevated role such as security_admin instead of silently broadening admin.
```

---

# 8. Database Schema

## 8.1 Extensions and Shared Trigger

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

## 8.2 Bootstrap Lock

```sql
CREATE TABLE bootstrap_lock (
    id INT PRIMARY KEY CHECK (id = 1),
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    completed_by UUID
);

INSERT INTO bootstrap_lock (id, completed)
VALUES (1, FALSE)
ON CONFLICT (id) DO NOTHING;
```

Purpose:

```text
Prevent first-user bootstrap races.
The first successful bootstrap creates the first user, assigns platform_owner, creates the first team, and assigns concrete team owner.
```

## 8.3 Users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    email CITEXT NOT NULL,
    email_canonical TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,

    display_name TEXT,
    password_hash TEXT,

    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_secret_ciphertext BYTEA,
    mfa_secret_key_id TEXT,
    mfa_enabled_at TIMESTAMPTZ,

    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'anonymized')),
    status_reason_code TEXT,

    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,

    failed_login_count INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,

    last_login_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,

    version INT NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (email_canonical = LOWER(TRIM(email::TEXT))),
    CHECK ((status <> 'anonymized') OR (deleted_at IS NOT NULL))
);

CREATE UNIQUE INDEX uq_users_email_active
    ON users (email_canonical)
    WHERE deleted_at IS NULL AND status <> 'anonymized';

CREATE INDEX idx_users_status
    ON users (status)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_users_last_activity
    ON users (last_activity_at)
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Rules:

```text
email_canonical = lower(trim(email)).
Anonymized users receive a synthetic non-identifying email.
Anonymization must scrub display name, password hash, MFA data, recovery codes, sessions, and refresh tokens.
failed_login_count and locked_until may be mirrored in rate-limit infrastructure but remain useful for durable account lockout.
```

## 8.4 Email Verification Tokens

```sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token_hmac TEXT UNIQUE NOT NULL,
    token_hmac_key_id TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'used', 'expired', 'revoked')),

    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (expires_at > created_at)
);

CREATE INDEX idx_email_verification_tokens_user
    ON email_verification_tokens (user_id, status);

CREATE INDEX idx_email_verification_tokens_pending
    ON email_verification_tokens (status, expires_at)
    WHERE status = 'pending';

CREATE TRIGGER trg_email_verification_tokens_updated_at
BEFORE UPDATE ON email_verification_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

## 8.5 Password Reset Tokens

```sql
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token_hmac TEXT UNIQUE NOT NULL,
    token_hmac_key_id TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'used', 'expired', 'revoked')),

    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (expires_at > created_at)
);

CREATE INDEX idx_password_reset_tokens_user
    ON password_reset_tokens (user_id, status);

CREATE INDEX idx_password_reset_tokens_pending
    ON password_reset_tokens (status, expires_at)
    WHERE status = 'pending';

CREATE TRIGGER trg_password_reset_tokens_updated_at
BEFORE UPDATE ON password_reset_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

## 8.6 MFA Recovery Codes

```sql
CREATE TABLE mfa_recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    code_hash TEXT NOT NULL,
    hash_algorithm TEXT NOT NULL DEFAULT 'argon2id',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    UNIQUE (user_id, code_hash)
);

CREATE INDEX idx_mfa_recovery_codes_user_active
    ON mfa_recovery_codes (user_id)
    WHERE used_at IS NULL AND revoked_at IS NULL;
```

Rules:

```text
Recovery codes are shown once.
Recovery codes are hashed individually.
Using a recovery code marks only that code used.
Anonymization deletes or revokes all recovery codes.
```

## 8.7 Teams

```sql
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT NOT NULL,
    slug CITEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'suspended', 'deleted')),
    status_reason_code TEXT,

    created_by UUID REFERENCES users(id) ON DELETE SET NULL,

    version INT NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_teams_slug_active
    ON teams (LOWER(slug::TEXT))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_teams_status
    ON teams (status)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_teams_updated_at
BEFORE UPDATE ON teams
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Normal deletion is soft deletion:

```text
status = deleted
deleted_at = NOW()
```

Hard deletion is allowed only through controlled retention jobs outside the application role.

## 8.8 Roles

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    team_id UUID REFERENCES teams(id) ON DELETE RESTRICT,

    name TEXT NOT NULL,
    description TEXT,

    is_system_role BOOLEAN NOT NULL DEFAULT FALSE,

    deleted_at TIMESTAMPTZ,
    version INT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (TRIM(name) <> '')
);

CREATE UNIQUE INDEX uq_roles_team_name_active
    ON roles (team_id, LOWER(name))
    WHERE team_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX uq_roles_template_name_active
    ON roles (LOWER(name))
    WHERE team_id IS NULL AND deleted_at IS NULL;

ALTER TABLE roles
ADD CONSTRAINT uq_roles_id_team UNIQUE (id, team_id);

CREATE INDEX idx_roles_team_active
    ON roles (team_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_roles_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

System-role identity protection:

```sql
CREATE OR REPLACE FUNCTION prevent_system_role_identity_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.is_system_role = TRUE THEN
        IF NEW.name <> OLD.name THEN
            RAISE EXCEPTION 'system role names cannot be changed';
        END IF;

        IF NEW.deleted_at IS DISTINCT FROM OLD.deleted_at THEN
            RAISE EXCEPTION 'system roles cannot be soft-deleted by application role';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_system_role_update
BEFORE UPDATE ON roles
FOR EACH ROW
EXECUTE FUNCTION prevent_system_role_identity_change();
```

Rules:

```text
Template roles have team_id IS NULL.
Concrete team roles have team_id IS NOT NULL.
Memberships, invitations, and temporary grants must reference concrete team roles.
System roles cannot be renamed or soft-deleted through normal APIs.
Custom roles are soft-deleted, not hard-deleted, once referenced.
Retention hard-delete procedures run under a maintenance role and are not exposed to the application role.
```

## 8.9 Permissions and Role Permissions

```sql
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (resource, action),
    CHECK (TRIM(resource) <> ''),
    CHECK (TRIM(action) <> '')
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,

    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_permission
    ON role_permissions (permission_id);
```

## 8.10 Invitations

```sql
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,

    email CITEXT NOT NULL,
    email_canonical TEXT NOT NULL,
    email_hmac TEXT NOT NULL,
    email_hmac_key_id TEXT NOT NULL,

    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,

    role_id UUID NOT NULL,

    token_hmac TEXT UNIQUE NOT NULL,
    token_hmac_key_id TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'expired', 'revoked')),

    expires_at TIMESTAMPTZ NOT NULL,

    accepted_at TIMESTAMPTZ,
    accepted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    revoked_reason_code TEXT,

    send_count INT NOT NULL DEFAULT 1,
    last_sent_at TIMESTAMPTZ DEFAULT NOW(),

    version INT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invitation_role_same_team
        FOREIGN KEY (role_id, team_id)
        REFERENCES roles (id, team_id),

    CHECK (email_canonical = LOWER(TRIM(email::TEXT))),
    CHECK (expires_at > created_at),
    CHECK (
        (status = 'pending' AND accepted_at IS NULL AND revoked_at IS NULL)
        OR status IN ('accepted', 'expired', 'revoked')
    )
);

CREATE INDEX idx_invitations_status
    ON invitations (status, expires_at)
    WHERE status = 'pending';

CREATE INDEX idx_invitations_token_hmac
    ON invitations (token_hmac);

CREATE INDEX idx_invitations_team_email_hmac
    ON invitations (team_id, email_hmac);

CREATE UNIQUE INDEX uq_pending_invitation_team_email
    ON invitations (team_id, email_canonical)
    WHERE status = 'pending';

CREATE TRIGGER trg_invitations_updated_at
BEFORE UPDATE ON invitations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Important token rule:

```text
Do not store raw invitation token.
Do not store token plaintext prefix.
Only token_hmac is persisted.
Raw token appears only inside encrypted sensitive outbox payload.
```

Duplicate invitation rule:

```text
Default policy permits only one pending invitation per team + canonical email.
Resend should reuse or rotate the existing pending invitation according to product policy.
Creating another pending invitation to the same team/email returns 409 or an idempotent replay.
```

## 8.11 Team Memberships

```sql
CREATE TABLE team_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    role_id UUID NOT NULL,

    invitation_id UUID REFERENCES invitations(id) ON DELETE SET NULL,

    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'removed')),

    removed_at TIMESTAMPTZ,
    removed_by UUID REFERENCES users(id) ON DELETE SET NULL,

    version INT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_membership_role_same_team
        FOREIGN KEY (role_id, team_id)
        REFERENCES roles (id, team_id),

    CHECK (
        (status = 'active' AND removed_at IS NULL)
        OR
        (status = 'removed' AND removed_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_team_membership_one_active
    ON team_memberships (team_id, user_id)
    WHERE status = 'active';

CREATE INDEX idx_memberships_user
    ON team_memberships (user_id);

CREATE INDEX idx_memberships_team
    ON team_memberships (team_id);

CREATE INDEX idx_memberships_role
    ON team_memberships (role_id);

CREATE INDEX idx_memberships_active_team
    ON team_memberships (team_id)
    WHERE status = 'active';

CREATE TRIGGER trg_team_memberships_updated_at
BEFORE UPDATE ON team_memberships
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

## 8.12 Temporary Team Access Grants

```sql
CREATE TABLE team_access_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    role_id UUID NOT NULL,

    reason_code TEXT NOT NULL,
    ticket_reference_hmac TEXT,
    ticket_reference_hmac_key_id TEXT,

    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    expires_at TIMESTAMPTZ NOT NULL,

    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    revoked_reason_code TEXT,

    version INT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_access_grant_role_same_team
        FOREIGN KEY (role_id, team_id)
        REFERENCES roles (id, team_id),

    CHECK (expires_at > granted_at),
    CHECK (revoked_at IS NULL OR revoked_at >= granted_at)
);

CREATE INDEX idx_team_access_grants_active
    ON team_access_grants (team_id, user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_team_access_grants_user_active
    ON team_access_grants (user_id, expires_at)
    WHERE revoked_at IS NULL;

CREATE TRIGGER trg_team_access_grants_updated_at
BEFORE UPDATE ON team_access_grants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Rules:

```text
Temporary grants are for JIT/break-glass access.
They do not mutate permanent membership.
They must be audited.
They expire automatically.
They must participate in SoD checks.
High-risk or long-lived grants require recent MFA verification.
```

## 8.13 Platform Roles

```sql
CREATE TABLE platform_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT UNIQUE NOT NULL
        CHECK (name IN ('platform_support', 'platform_admin', 'platform_owner')),

    level INT UNIQUE NOT NULL CHECK (level > 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_platform_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform_role_id UUID NOT NULL REFERENCES platform_roles(id) ON DELETE CASCADE,

    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    version INT NOT NULL DEFAULT 1,

    PRIMARY KEY (user_id, platform_role_id)
);

CREATE INDEX idx_user_platform_roles_user
    ON user_platform_roles (user_id);

CREATE INDEX idx_user_platform_roles_role
    ON user_platform_roles (platform_role_id);
```

Seed hierarchy:

```sql
INSERT INTO platform_roles (name, level) VALUES
    ('platform_support', 40),
    ('platform_admin', 80),
    ('platform_owner', 100)
ON CONFLICT DO NOTHING;
```

## 8.14 User Sessions and Refresh Tokens

```sql
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    access_token_jti TEXT UNIQUE,
    refresh_token_family UUID NOT NULL DEFAULT gen_random_uuid(),

    ip_address INET,
    user_agent TEXT,
    device_label TEXT,

    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_reason_code TEXT,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    revoked_at TIMESTAMPTZ,

    access_expires_at TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ NOT NULL,

    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (is_revoked = FALSE OR revoked_at IS NOT NULL),
    CHECK (refresh_expires_at > created_at)
);

CREATE INDEX idx_sessions_user_active
    ON user_sessions (user_id)
    WHERE is_revoked = FALSE;

CREATE INDEX idx_sessions_family
    ON user_sessions (refresh_token_family);

CREATE INDEX idx_sessions_access_expires
    ON user_sessions (access_expires_at)
    WHERE is_revoked = FALSE;

CREATE TRIGGER trg_user_sessions_updated_at
BEFORE UPDATE ON user_sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    session_id UUID NOT NULL REFERENCES user_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token_family UUID NOT NULL,
    token_hash TEXT UNIQUE NOT NULL,
    token_hmac_key_id TEXT NOT NULL,

    previous_token_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    replaced_by_token_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,

    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    used_at TIMESTAMPTZ,
    rotated_at TIMESTAMPTZ,

    revoked_at TIMESTAMPTZ,
    revoked_reason_code TEXT,

    reuse_detected_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (expires_at > issued_at)
);

CREATE INDEX idx_refresh_tokens_session
    ON refresh_tokens (session_id, issued_at);

CREATE INDEX idx_refresh_tokens_user
    ON refresh_tokens (user_id, issued_at);

CREATE INDEX idx_refresh_tokens_family
    ON refresh_tokens (token_family);

CREATE INDEX idx_refresh_tokens_active
    ON refresh_tokens (session_id, expires_at)
    WHERE revoked_at IS NULL AND rotated_at IS NULL;

CREATE TRIGGER trg_refresh_tokens_updated_at
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Refresh-token rules:

```text
Only the latest unrotated token in a family is accepted.
Using a rotated token is token reuse.
Refresh-token reuse revokes the session and the whole token family.
Suspended, anonymized, or deleted users cannot refresh.
Refresh token hashes use HMAC or strong keyed hash, not plain SHA-256 without a secret.
```

## 8.15 Audit Logs

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL,
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),

    actor_id UUID,
    actor_type TEXT NOT NULL DEFAULT 'user'
        CHECK (actor_type IN ('user', 'system', 'service_principal')),

    team_id UUID,

    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,

    old_value JSONB NOT NULL DEFAULT '{}'::jsonb,
    new_value JSONB NOT NULL DEFAULT '{}'::jsonb,

    change_summary TEXT NOT NULL,

    ip_hmac TEXT,
    user_agent_hmac TEXT,
    hmac_key_id TEXT,

    idempotency_key TEXT,
    request_id UUID,
    correlation_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (id, created_at),
    UNIQUE (event_id, created_at),

    CHECK (jsonb_typeof(old_value) = 'object'),
    CHECK (jsonb_typeof(new_value) = 'object')
);

CREATE INDEX idx_audit_team
    ON audit_logs (team_id, created_at);

CREATE INDEX idx_audit_action
    ON audit_logs (action, created_at);

CREATE INDEX idx_audit_entity
    ON audit_logs (entity_type, entity_id);

CREATE INDEX idx_audit_actor
    ON audit_logs (actor_id, created_at);

CREATE INDEX idx_audit_idempotency
    ON audit_logs (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX idx_audit_request
    ON audit_logs (request_id)
    WHERE request_id IS NOT NULL;

CREATE INDEX idx_audit_correlation
    ON audit_logs (correlation_id)
    WHERE correlation_id IS NOT NULL;
```

Production rule:

```text
Partition audit_logs by created_at from day one.
Application audit writer receives INSERT and narrowly scoped SELECT only.
Application role must not UPDATE or DELETE audit rows.
One command may create multiple audit rows with the same idempotency key.
```

## 8.16 Idempotency Store

```sql
CREATE TABLE idempotency_keys (
    scope_type TEXT NOT NULL
        CHECK (scope_type IN ('user', 'anonymous', 'system')),

    scope_id TEXT NOT NULL,
    key TEXT NOT NULL,

    request_method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'processing'
        CHECK (status IN ('processing', 'completed', 'failed')),

    response_code INT,
    response_body JSONB,
    response_body_ciphertext BYTEA,
    response_body_key_id TEXT,

    error_code TEXT,
    locked_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (scope_type, scope_id, key),

    CHECK (expires_at > created_at),
    CHECK (response_body IS NULL OR jsonb_typeof(response_body) = 'object')
);

CREATE INDEX idx_idempotency_expires
    ON idempotency_keys (expires_at);

CREATE INDEX idx_idempotency_status
    ON idempotency_keys (status, created_at);

CREATE INDEX idx_idempotency_processing
    ON idempotency_keys (status, locked_until)
    WHERE status = 'processing';

CREATE TRIGGER trg_idempotency_updated_at
BEFORE UPDATE ON idempotency_keys
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Critical rule:

```text
The primary key is scope_type + scope_id + key.
The same key reused with a different method, path, query, or body fingerprint returns 409.
```

## 8.17 Outbox Events

```sql
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    event_type TEXT NOT NULL,

    aggregate_type TEXT,
    aggregate_id UUID,

    payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    sensitive_payload_ciphertext BYTEA,
    sensitive_payload_key_id TEXT,

    provider_message_key TEXT,

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'sent', 'failed', 'dead_letter')),

    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,

    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    locked_at TIMESTAMPTZ,
    locked_by TEXT,

    last_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    processed_at TIMESTAMPTZ,

    purge_after TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',

    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX idx_outbox_ready
    ON outbox_events (status, next_attempt_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX idx_outbox_processing
    ON outbox_events (status, locked_at)
    WHERE status = 'processing';

CREATE INDEX idx_outbox_purge
    ON outbox_events (purge_after);

CREATE INDEX idx_outbox_provider_message_key
    ON outbox_events (provider_message_key)
    WHERE provider_message_key IS NOT NULL;

CREATE TRIGGER trg_outbox_updated_at
BEFORE UPDATE ON outbox_events
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
```

Rules:

```text
payload must be sanitized and safe to log.
Raw invitation/reset/verification tokens exist only in sensitive_payload_ciphertext until dispatch/scrub.
provider_message_key supports provider-side idempotency and duplicate-send prevention.
Sensitive payloads are nulled after success, dead-letter, or TTL.
```

## 8.18 Separation of Duties Rules

```sql
CREATE TABLE sod_conflict_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    team_id UUID REFERENCES teams(id) ON DELETE CASCADE,

    name TEXT NOT NULL,

    permission_a_id UUID NOT NULL REFERENCES permissions(id),
    permission_b_id UUID NOT NULL REFERENCES permissions(id),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (team_id, permission_a_id, permission_b_id),
    CHECK (permission_a_id < permission_b_id)
);

CREATE INDEX idx_sod_rules_team_active
    ON sod_conflict_rules (team_id)
    WHERE is_active = TRUE;
```

## 8.19 Deleted Entity Snapshots

```sql
CREATE TABLE deleted_entity_snapshots (
    id BIGSERIAL PRIMARY KEY,

    audit_event_id UUID NOT NULL,

    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,

    snapshot_sanitized JSONB,
    snapshot_ciphertext BYTEA,
    encryption_key_id TEXT,

    deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (
        snapshot_sanitized IS NOT NULL
        OR snapshot_ciphertext IS NOT NULL
    ),

    CHECK (
        snapshot_ciphertext IS NULL
        OR encryption_key_id IS NOT NULL
    ),

    CHECK (
        snapshot_sanitized IS NULL
        OR jsonb_typeof(snapshot_sanitized) = 'object'
    )
);

CREATE INDEX idx_deleted_snapshots_entity
    ON deleted_entity_snapshots (entity_type, entity_id);

CREATE INDEX idx_deleted_snapshots_audit_event
    ON deleted_entity_snapshots (audit_event_id);
```

Rules:

```text
snapshot_sanitized must pass a PII validator.
snapshot_ciphertext must use a destroyable key.
No raw production PII in sanitized snapshots.
```

## 8.20 Webhooks

```sql
CREATE TABLE webhook_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,

    url TEXT NOT NULL,

    secret_ciphertext BYTEA NOT NULL,
    secret_key_id TEXT NOT NULL,
    secret_fingerprint_hmac TEXT NOT NULL,

    events JSONB NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    version INT NOT NULL DEFAULT 1,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (jsonb_typeof(events) = 'array')
);

CREATE INDEX idx_webhook_subscriptions_team_active
    ON webhook_subscriptions (team_id)
    WHERE is_active = TRUE;

CREATE TRIGGER trg_webhook_subscriptions_updated_at
BEFORE UPDATE ON webhook_subscriptions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    subscription_id UUID NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    outbox_event_id UUID REFERENCES outbox_events(id) ON DELETE SET NULL,

    request_body_sanitized JSONB NOT NULL,

    response_status INT,
    response_body_excerpt_sanitized TEXT,
    response_body_hash TEXT,

    success BOOLEAN,

    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CHECK (jsonb_typeof(request_body_sanitized) = 'object')
);

CREATE INDEX idx_webhook_deliveries_subscription
    ON webhook_deliveries (subscription_id, attempted_at);
```

Webhook secret rule:

```text
Outbound webhook signing requires access to a signing secret.
Therefore store the signing secret encrypted, not only hashed.
Use HMAC/fingerprint only for comparison, diagnostics, and key rotation metadata.
```

---

# 9. Database Invariant Triggers

## 9.1 Last Team Owner Protection

```sql
CREATE OR REPLACE FUNCTION enforce_team_has_owner()
RETURNS TRIGGER AS $$
DECLARE
    affected_team UUID;
    owner_count INT;
    team_is_live BOOLEAN;
BEGIN
    affected_team := COALESCE(OLD.team_id, NEW.team_id);

    SELECT EXISTS (
        SELECT 1
        FROM teams
        WHERE id = affected_team
          AND deleted_at IS NULL
          AND status <> 'deleted'
    )
    INTO team_is_live;

    IF team_is_live = FALSE THEN
        RETURN NULL;
    END IF;

    SELECT COUNT(*)
    INTO owner_count
    FROM team_memberships tm
    JOIN roles r ON r.id = tm.role_id
    WHERE tm.team_id = affected_team
      AND tm.status = 'active'
      AND r.team_id = affected_team
      AND r.deleted_at IS NULL
      AND r.name = 'owner';

    IF owner_count < 1 THEN
        RAISE EXCEPTION 'team must have at least one active owner';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_team_has_owner_after_membership_update
AFTER UPDATE OR DELETE ON team_memberships
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_team_has_owner();
```

Service code must also lock the team row before owner mutations:

```sql
SELECT id
FROM teams
WHERE id = :team_id
  AND deleted_at IS NULL
FOR UPDATE;
```

Alternative service lock:

```text
pg_advisory_xact_lock(hash(team_id, 'owner_mutation'))
```

## 9.2 Last Platform Owner Protection

```sql
CREATE OR REPLACE FUNCTION active_platform_owner_count()
RETURNS INT AS $$
DECLARE
    cnt INT;
BEGIN
    SELECT COUNT(*)
    INTO cnt
    FROM user_platform_roles upr
    JOIN platform_roles pr ON pr.id = upr.platform_role_id
    JOIN users u ON u.id = upr.user_id
    WHERE pr.name = 'platform_owner'
      AND u.status = 'active'
      AND u.deleted_at IS NULL;

    RETURN cnt;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION enforce_platform_has_owner()
RETURNS TRIGGER AS $$
BEGIN
    IF active_platform_owner_count() < 1 THEN
        RAISE EXCEPTION 'platform must have at least one active platform_owner';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER trg_platform_owner_after_role_delete
AFTER DELETE ON user_platform_roles
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_platform_has_owner();

CREATE CONSTRAINT TRIGGER trg_platform_owner_after_user_update
AFTER UPDATE OF status, deleted_at ON users
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
WHEN (
    OLD.status IS DISTINCT FROM NEW.status
    OR OLD.deleted_at IS DISTINCT FROM NEW.deleted_at
)
EXECUTE FUNCTION enforce_platform_has_owner();
```

Service-level rules:

```text
Cannot remove the final active platform_owner.
Cannot suspend the final active platform_owner.
Cannot anonymize the final active platform_owner.
platform_admin cannot suspend or anonymize a platform_owner; platform_owner required.
Only platform_owner may assign or remove platform roles.
Service code must acquire a platform-owner mutation lock before removing, suspending, or anonymizing a platform_owner.
```

---

# 10. Seed Data

## 10.1 Permissions

```sql
INSERT INTO permissions (resource, action, description) VALUES
    ('team', 'read', 'View team details'),
    ('team', 'update', 'Edit team settings'),
    ('team', 'delete', 'Delete team'),

    ('team_member', 'invite', 'Invite members'),
    ('team_member', 'read', 'View members'),
    ('team_member', 'update_role', 'Change member role'),
    ('team_member', 'remove', 'Remove member'),

    ('invitation', 'read', 'View team invitations'),

    ('role', 'read', 'View team roles'),
    ('role', 'create', 'Create custom team role'),
    ('role', 'update', 'Update custom team role'),
    ('role', 'delete', 'Delete custom team role'),

    ('access_grant', 'create', 'Create temporary access grant'),
    ('access_grant', 'read', 'View temporary access grants'),
    ('access_grant', 'revoke', 'Revoke temporary access grant'),

    ('project', 'create', 'Create projects'),
    ('project', 'read', 'View projects'),
    ('project', 'update', 'Edit projects'),
    ('project', 'delete', 'Delete projects'),

    ('billing', 'read', 'View billing'),
    ('billing', 'update', 'Update billing'),

    ('audit_log', 'read', 'View team audit logs'),

    ('webhook', 'create', 'Create webhooks'),
    ('webhook', 'read', 'View webhooks'),
    ('webhook', 'update', 'Update webhooks'),
    ('webhook', 'delete', 'Delete webhooks')
ON CONFLICT (resource, action) DO NOTHING;
```

## 10.2 Template Roles

```sql
INSERT INTO roles (team_id, name, description, is_system_role) VALUES
    (NULL, 'owner', 'Full team control, including team deletion.', TRUE),
    (NULL, 'admin', 'Manage team settings, members, invitations, projects, billing, and webhooks except team deletion, role administration, temporary access grants, and audit logs.', TRUE),
    (NULL, 'manager', 'Manage projects and view team/member/role basics.', TRUE),
    (NULL, 'member', 'Create, read, and update project content.', TRUE),
    (NULL, 'viewer', 'Limited read-only team and project access.', TRUE)
ON CONFLICT DO NOTHING;
```

## 10.3 Explicit Role Permission Seeding

```sql
-- Owner: all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.team_id IS NULL
  AND r.name = 'owner'
ON CONFLICT DO NOTHING;

-- Admin: explicit operational permissions, not team deletion, role mutation, grant creation/revocation, or audit logs
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON TRUE
WHERE r.team_id IS NULL
  AND r.name = 'admin'
  AND (
      (p.resource = 'team' AND p.action IN ('read', 'update'))
      OR (p.resource = 'team_member' AND p.action IN ('invite', 'read', 'update_role', 'remove'))
      OR (p.resource = 'invitation' AND p.action = 'read')
      OR (p.resource = 'role' AND p.action = 'read')
      OR (p.resource = 'access_grant' AND p.action = 'read')
      OR (p.resource = 'project' AND p.action IN ('create', 'read', 'update', 'delete'))
      OR (p.resource = 'billing' AND p.action IN ('read', 'update'))
      OR (p.resource = 'webhook' AND p.action IN ('create', 'read', 'update', 'delete'))
  )
ON CONFLICT DO NOTHING;

-- Manager: project management + read team/member/role basics
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON TRUE
WHERE r.team_id IS NULL
  AND r.name = 'manager'
  AND (
      (p.resource = 'team' AND p.action = 'read')
      OR (p.resource = 'team_member' AND p.action = 'read')
      OR (p.resource = 'role' AND p.action = 'read')
      OR (p.resource = 'project' AND p.action IN ('create', 'read', 'update', 'delete'))
  )
ON CONFLICT DO NOTHING;

-- Member: team read + project create/read/update
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON TRUE
WHERE r.team_id IS NULL
  AND r.name = 'member'
  AND (
      (p.resource = 'team' AND p.action = 'read')
      OR (p.resource = 'project' AND p.action IN ('create', 'read', 'update'))
  )
ON CONFLICT DO NOTHING;

-- Viewer: limited read-only access, not billing/audit/members/roles/webhooks by default
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON TRUE
WHERE r.team_id IS NULL
  AND r.name = 'viewer'
  AND (
      (p.resource = 'team' AND p.action = 'read')
      OR (p.resource = 'project' AND p.action = 'read')
  )
ON CONFLICT DO NOTHING;
```

## 10.4 Platform Roles

```sql
INSERT INTO platform_roles (name, level) VALUES
    ('platform_support', 40),
    ('platform_admin', 80),
    ('platform_owner', 100)
ON CONFLICT DO NOTHING;
```

---

# 11. Hashing, Canonicalization, and Sanitization

## 11.1 HMAC Helpers

```python
import hashlib
import hmac
import json
import re
from typing import Any

EMAIL_RE = re.compile(r"[^@\s]+@[^@\s]+\.[^@\s]+")
IPV4_RE = re.compile(r"\b(?:\d{1,3}\.){3}\d{1,3}\b")


def canonical_email(email: str) -> str:
    return email.strip().lower()


def hmac_identifier(secret: bytes, value: str) -> str:
    normalized = value.strip().lower()
    return hmac.new(secret, normalized.encode(), hashlib.sha256).hexdigest()


def email_hmac(secret: bytes, email: str) -> str:
    return hmac_identifier(secret, canonical_email(email))


def token_hmac(secret: bytes, raw_token: str) -> str:
    return hmac.new(secret, raw_token.encode(), hashlib.sha256).hexdigest()


def ip_hmac(secret: bytes, ip: str | None) -> str | None:
    return hmac_identifier(secret, ip) if ip else None


def user_agent_hmac(secret: bytes, user_agent: str | None) -> str | None:
    return hmac_identifier(secret, user_agent) if user_agent else None


def canonical_json(obj: dict[str, Any]) -> str:
    return json.dumps(obj or {}, sort_keys=True, separators=(",", ":"), default=str)


def request_fingerprint(method: str, path: str, query: dict[str, Any], body: dict[str, Any]) -> str:
    payload = {
        "method": method.upper(),
        "path": path,
        "query": query or {},
        "body": body or {},
    }
    return hashlib.sha256(canonical_json(payload).encode()).hexdigest()
```

## 11.2 Audit Allowlist and Sanitizer

```python
ALLOWED_AUDIT_FIELDS = {
    "status": str,
    "previous_status": str,
    "new_status": str,
    "role_id": str,
    "old_role_id": str,
    "new_role_id": str,
    "role_name": str,
    "old_role_name": str,
    "new_role_name": str,
    "permission_ids": list,
    "email_hmac": str,
    "email_hmac_key_id": str,
    "sessions_revoked": int,
    "refresh_tokens_revoked": int,
    "reason_code": str,
    "membership_id": str,
    "team_id": str,
    "target_user_id": str,
    "version": int,
    "grant_id": str,
    "expires_at": str,
    "webhook_id": str,
    "token_family_revoked": bool,
    "pii_scrubbed": bool,
}

SAFE_REASON_CODES = {
    "user_requested",
    "policy_violation",
    "security_risk",
    "inactive_account",
    "admin_action",
    "user_anonymized",
    "invitation_revoked",
    "temporary_access_expired",
    "refresh_token_reuse",
    "password_reset",
    "session_revoked",
    "platform_owner_protection",
}

MAX_AUDIT_STRING_LENGTH = 200


def contains_obvious_pii(value: str) -> bool:
    if EMAIL_RE.search(value):
        return True
    if IPV4_RE.search(value):
        return True

    lowered = value.lower()

    forbidden_fragments = [
        "bearer ",
        "authorization:",
        "set-cookie",
        "cookie",
        "refresh_token",
        "access_token",
        "password",
        "totp",
        "mfa_secret",
        "recovery_code",
        "token=",
    ]

    return any(fragment in lowered for fragment in forbidden_fragments)


def sanitize_audit_payload(payload: dict[str, Any]) -> dict[str, Any]:
    sanitized = {}

    for key, value in (payload or {}).items():
        if key not in ALLOWED_AUDIT_FIELDS:
            continue

        expected_type = ALLOWED_AUDIT_FIELDS[key]
        if not isinstance(value, expected_type):
            continue

        if isinstance(value, str):
            if len(value) > MAX_AUDIT_STRING_LENGTH:
                continue
            if contains_obvious_pii(value):
                continue

        if key == "reason_code" and value not in SAFE_REASON_CODES:
            continue

        if key == "permission_ids":
            if not all(isinstance(item, str) and len(item) <= 64 for item in value):
                continue

        sanitized[key] = value

    return sanitized
```

---

# 12. Authorization Engine

## 12.1 Team Permission Check

```python
from uuid import UUID

class PermissionDenied(Exception):
    pass


def check_team_permission(db, *, user_id: UUID, team_id: UUID, resource: str, action: str) -> bool:
    result = db.query_one("""
        SELECT EXISTS (
            SELECT 1
            FROM users u
            JOIN team_memberships tm ON tm.user_id = u.id
            JOIN teams t ON t.id = tm.team_id
            JOIN roles r ON r.id = tm.role_id
            JOIN role_permissions rp ON rp.role_id = r.id
            JOIN permissions p ON p.id = rp.permission_id
            WHERE u.id = :user_id
              AND u.status = 'active'
              AND u.deleted_at IS NULL
              AND t.id = :team_id
              AND t.status = 'active'
              AND t.deleted_at IS NULL
              AND tm.status = 'active'
              AND r.team_id = t.id
              AND r.deleted_at IS NULL
              AND p.resource = :resource
              AND p.action = :action

            UNION

            SELECT 1
            FROM users u
            JOIN team_access_grants tag ON tag.user_id = u.id
            JOIN teams t ON t.id = tag.team_id
            JOIN roles r ON r.id = tag.role_id
            JOIN role_permissions rp ON rp.role_id = r.id
            JOIN permissions p ON p.id = rp.permission_id
            WHERE u.id = :user_id
              AND u.status = 'active'
              AND u.deleted_at IS NULL
              AND t.id = :team_id
              AND t.status = 'active'
              AND t.deleted_at IS NULL
              AND tag.revoked_at IS NULL
              AND tag.expires_at > NOW()
              AND r.team_id = t.id
              AND r.deleted_at IS NULL
              AND p.resource = :resource
              AND p.action = :action
        ) AS authorized
    """, {
        "user_id": user_id,
        "team_id": team_id,
        "resource": resource,
        "action": action,
    })

    if not result or not result["authorized"]:
        raise PermissionDenied(f"Missing permission {resource}:{action}")

    return True
```

## 12.2 Platform Permission Check

```python
def check_platform_permission(db, *, user_id: UUID, required_role: str) -> bool:
    result = db.query_one("""
        SELECT EXISTS (
            SELECT 1
            FROM users u
            JOIN user_platform_roles upr ON upr.user_id = u.id
            JOIN platform_roles user_role ON user_role.id = upr.platform_role_id
            JOIN platform_roles required_role ON required_role.name = :required_role
            WHERE u.id = :user_id
              AND u.status = 'active'
              AND u.deleted_at IS NULL
              AND user_role.level >= required_role.level
        ) AS authorized
    """, {
        "user_id": user_id,
        "required_role": required_role,
    })

    if not result or not result["authorized"]:
        raise PermissionDenied(f"Missing platform role {required_role}")

    return True
```

Platform roles never satisfy team permission checks.

## 12.3 Active Session Check

Use for high-risk endpoints.

```python
def check_active_session(db, *, user_id: UUID, session_id: UUID) -> bool:
    result = db.query_one("""
        SELECT EXISTS (
            SELECT 1
            FROM user_sessions s
            JOIN users u ON u.id = s.user_id
            WHERE s.id = :session_id
              AND s.user_id = :user_id
              AND s.is_revoked = FALSE
              AND s.refresh_expires_at > NOW()
              AND u.status = 'active'
              AND u.deleted_at IS NULL
        ) AS active
    """, {
        "user_id": user_id,
        "session_id": session_id,
    })

    if not result or not result["active"]:
        raise PermissionDenied("Session is no longer active")

    return True
```

High-risk endpoints:

```text
Password change
MFA disable
MFA recovery-code regeneration
Role changes
Temporary access grants
Platform role assignment/removal
User suspension
User anonymization
Team deletion
Billing changes
Webhook secret rotation
```

---

# 13. Separation of Duties Enforcement

SoD rules are evaluated against the effective permission set, including:

```text
Permanent active membership role
Active temporary access grants
Proposed new role permissions
Proposed new temporary grant permissions
```

```python
class SeparationOfDutiesViolation(Exception):
    pass


def get_effective_permission_ids_for_user(db, *, user_id: UUID, team_id: UUID) -> set[str]:
    rows = db.query("""
        SELECT DISTINCT p.id::text AS permission_id
        FROM team_memberships tm
        JOIN roles r ON r.id = tm.role_id
        JOIN role_permissions rp ON rp.role_id = r.id
        JOIN permissions p ON p.id = rp.permission_id
        WHERE tm.user_id = :user_id
          AND tm.team_id = :team_id
          AND tm.status = 'active'
          AND r.deleted_at IS NULL

        UNION

        SELECT DISTINCT p.id::text AS permission_id
        FROM team_access_grants tag
        JOIN roles r ON r.id = tag.role_id
        JOIN role_permissions rp ON rp.role_id = r.id
        JOIN permissions p ON p.id = rp.permission_id
        WHERE tag.user_id = :user_id
          AND tag.team_id = :team_id
          AND tag.revoked_at IS NULL
          AND tag.expires_at > NOW()
          AND r.deleted_at IS NULL
    """, {"user_id": user_id, "team_id": team_id})

    return {row["permission_id"] for row in rows}


def enforce_sod(db, *, team_id: UUID, user_id: UUID, proposed_permission_ids: set[str]):
    existing = get_effective_permission_ids_for_user(db, user_id=user_id, team_id=team_id)
    combined = existing | proposed_permission_ids

    conflicts = db.query("""
        SELECT permission_a_id::text, permission_b_id::text, name
        FROM sod_conflict_rules
        WHERE is_active = TRUE
          AND (team_id = :team_id OR team_id IS NULL)
    """, {"team_id": team_id})

    for conflict in conflicts:
        if conflict["permission_a_id"] in combined and conflict["permission_b_id"] in combined:
            raise SeparationOfDutiesViolation(conflict["name"])
```

---

# 14. Idempotency Contract

## 14.1 API Rule

Mutating business-resource endpoints require:

```text
Idempotency-Key: <uuid>
```

Do not replay token responses for:

```text
/auth/login
/auth/refresh
```

Registration and password reset request may use anonymous idempotency scoped by `email_hmac` or another privacy-safe fingerprint.

## 14.2 Failure Semantics

| Failure type | Behavior |
|---|---|
| Same key, same scope, different fingerprint | `409 idempotency_conflict`; no mutation |
| Same request still processing | `409 idempotency_in_progress` or `425 Too Early`; no mutation |
| Deterministic validation failure before mutation | Store and replay completed 4xx response |
| Permission denied | Store completed 403 only if deterministic for same actor/request |
| Unexpected server error before mutation | Mark failed or let lock expire |
| Mutation committed | Store completed idempotency response in same transaction before commit |
| Token issuance response | Do not store replayable raw tokens |

## 14.3 Insert-First Reservation Pattern

```python
from datetime import datetime, timedelta, timezone

class IdempotencyConflict(Exception):
    pass

class IdempotencyInProgress(Exception):
    pass


def begin_idempotent_request(db, *, scope_type: str, scope_id: str, key: str, method: str, path: str, query: dict, body: dict) -> dict | None:
    fingerprint = request_fingerprint(method, path, query, body)
    expires_at = datetime.now(timezone.utc) + timedelta(hours=48)
    locked_until = datetime.now(timezone.utc) + timedelta(minutes=5)

    inserted = db.query_one("""
        INSERT INTO idempotency_keys (
            scope_type, scope_id, key,
            request_method, request_path, request_fingerprint,
            status, locked_until, expires_at
        )
        VALUES (
            :scope_type, :scope_id, :key,
            :method, :path, :fingerprint,
            'processing', :locked_until, :expires_at
        )
        ON CONFLICT DO NOTHING
        RETURNING *
    """, {
        "scope_type": scope_type,
        "scope_id": scope_id,
        "key": key,
        "method": method.upper(),
        "path": path,
        "fingerprint": fingerprint,
        "locked_until": locked_until,
        "expires_at": expires_at,
    })

    if inserted:
        return None

    existing = db.query_one("""
        SELECT *
        FROM idempotency_keys
        WHERE scope_type = :scope_type
          AND scope_id = :scope_id
          AND key = :key
          AND expires_at > NOW()
        FOR UPDATE
    """, {
        "scope_type": scope_type,
        "scope_id": scope_id,
        "key": key,
    })

    if not existing:
        raise IdempotencyConflict("Expired idempotency key cannot be reused immediately")

    if (
        existing["request_method"] != method.upper()
        or existing["request_path"] != path
        or existing["request_fingerprint"] != fingerprint
    ):
        raise IdempotencyConflict("Idempotency key reused with different request")

    if existing["status"] == "completed":
        return {
            "response_code": existing["response_code"],
            "response_body": existing["response_body"],
        }

    if existing["status"] == "processing" and existing["locked_until"] and existing["locked_until"] > datetime.now(timezone.utc):
        raise IdempotencyInProgress("Equivalent request is already processing")

    db.execute("""
        UPDATE idempotency_keys
        SET status = 'processing',
            locked_until = :locked_until,
            error_code = NULL
        WHERE scope_type = :scope_type
          AND scope_id = :scope_id
          AND key = :key
    """, {
        "locked_until": locked_until,
        "scope_type": scope_type,
        "scope_id": scope_id,
        "key": key,
    })

    return None
```

---

# 15. Audit Writer

```python
def generic_summary(action: str) -> str:
    allowed = {
        "platform.bootstrap_completed",
        "team.created",
        "team.updated",
        "team.deleted",
        "team.member_invited",
        "team.member_role_changed",
        "team.member_removed",
        "invitation.accepted",
        "invitation.revoked",
        "invitation.resent",
        "role.created",
        "role.updated",
        "role.deleted",
        "temporary_access.granted",
        "temporary_access.revoked",
        "user.registered",
        "user.email_verified",
        "user.suspended",
        "user.reactivated",
        "user.anonymized",
        "sessions.revoked",
        "auth.logout",
        "auth.refresh_token_reuse_detected",
        "mfa.enabled",
        "mfa.recovery_code_used",
        "webhook.created",
        "webhook.updated",
        "webhook.deleted",
        "webhook.secret_rotated",
    }
    return action if action in allowed else "unknown.event"


def audit(db, *, action: str, actor_id, team_id, entity_type: str, entity_id, old_value: dict | None, new_value: dict | None, request_context, idempotency_key: str | None = None, request_id=None, correlation_id=None):
    db.insert("audit_logs", {
        "actor_id": actor_id,
        "actor_type": "system" if actor_id is None else "user",
        "team_id": team_id,
        "action": action,
        "entity_type": entity_type,
        "entity_id": entity_id,
        "old_value": sanitize_audit_payload(old_value or {}),
        "new_value": sanitize_audit_payload(new_value or {}),
        "change_summary": generic_summary(action),
        "ip_hmac": ip_hmac(AUDIT_HMAC_SECRET, request_context.ip_address),
        "user_agent_hmac": user_agent_hmac(AUDIT_HMAC_SECRET, request_context.user_agent),
        "hmac_key_id": AUDIT_HMAC_KEY_ID,
        "idempotency_key": idempotency_key,
        "request_id": request_id,
        "correlation_id": correlation_id,
    })
```

Rules:

```text
Audit writer must run inside the same transaction as the mutation.
Audit payloads must be allowlisted, not denylisted.
Audit change_summary must use safe system-generated text, not arbitrary user text.
```

---

# 16. Core Workflows

## 16.1 Race-Safe First-User Bootstrap

Purpose:

```text
Prevent a production system with no platform owner.
```

Transaction:

```text
1. Lock bootstrap_lock row FOR UPDATE.
2. If completed = true, reject bootstrap.
3. Verify no existing active users and no active platform_owner.
4. Create first user.
5. Mark email verified only if the bootstrap flow is trusted/internal; otherwise require verification before login.
6. Assign platform_owner.
7. Create first team.
8. Copy template roles into concrete team roles.
9. Copy role_permissions to concrete roles.
10. Assign first user concrete team owner role.
11. Audit platform.bootstrap_completed.
12. Mark bootstrap_lock.completed = true.
13. Commit.
```

Service must use:

```sql
SELECT * FROM bootstrap_lock WHERE id = 1 FOR UPDATE;
```

After completion, the bootstrap endpoint must be disabled or permanently return `409 bootstrap_completed`.

## 16.2 Register User

```text
1. Use anonymous idempotency scoped by email_hmac.
2. Canonicalize email.
3. Return generic conflict-safe response if account enumeration is a concern.
4. Hash password with Argon2id or bcrypt cost >= 12.
5. Create user with email_verified = false.
6. Create email_verification_token with token_hmac.
7. Write outbox event with encrypted raw verification token and email.
8. Audit user.registered without raw email.
9. Complete idempotency.
```

## 16.3 Verify Email

```text
1. Receive raw token.
2. HMAC token.
3. Lock pending email_verification_token.
4. Validate expiry and attempts.
5. Set token status = used.
6. Set users.email_verified = true and email_verified_at = NOW().
7. Audit user.email_verified without raw email.
```

## 16.4 Login

```text
1. Rate-limit by IP HMAC, email_hmac, and endpoint.
2. Canonicalize email.
3. Load user by email_canonical.
4. Use fake password verification if user missing.
5. Return generic failure for unknown email or wrong password.
6. Enforce failed_login_count and locked_until if persistent lockout is enabled.
7. Verify password.
8. Require active, non-deleted user.
9. If MFA enabled, require TOTP or recovery code.
10. Create user_session.
11. Create refresh_tokens row.
12. Issue access token with 15-minute expiry and jti.
13. Issue refresh token.
14. Do not idempotently replay raw tokens.
15. Audit auth.login using sanitized payload if desired by policy.
```

Access token contains:

```text
sub/user_id
session_id
jti
iat
exp
```

It does not contain:

```text
email
full permissions
MFA secrets
refresh token
```

## 16.5 Refresh Token

```text
1. HMAC raw refresh token.
2. Look up refresh_tokens row FOR UPDATE.
3. If not found, return generic invalid token.
4. If token exists but rotated_at is not null, treat as reuse.
5. On reuse: mark reuse_detected_at, revoke whole token_family, revoke session, audit auth.refresh_token_reuse_detected.
6. Validate session active and user active.
7. Validate token not expired or revoked.
8. Mark old token used_at and rotated_at.
9. Insert replacement refresh token in same family.
10. Set replaced_by_token_id.
11. Issue new access token and raw refresh token.
```

## 16.6 Logout

```text
1. Revoke current session.
2. Revoke active refresh tokens for session.
3. Audit auth.logout.
```

## 16.7 Create Team

```text
1. Reserve idempotency.
2. Verify actor active.
3. Insert team.
4. Copy template roles into concrete team roles.
5. Copy role_permissions from templates to concrete roles.
6. Assign creator concrete owner role.
7. Audit team.created.
8. Complete idempotency.
```

## 16.8 Invite User

Validation:

```text
Actor has team_member:invite.
Team is active.
Role belongs to same team and is not deleted.
Invitation email is canonicalized.
Target user is not already an active member.
No existing pending invitation exists for the same team + canonical email, unless resend/replace flow is intentionally used.
Raw token is generated once.
Only token_hmac is stored.
Raw token and raw email go only into encrypted sensitive outbox payload.
Audit stores email_hmac only.
```

Outbox example:

```json
{
  "event_type": "team.invitation_created",
  "payload": {
    "invitation_id": "uuid",
    "team_id": "uuid"
  },
  "sensitive_payload_ciphertext": "encrypted({ email, raw_token })",
  "provider_message_key": "invitation:uuid:send_count"
}
```

## 16.9 Accept Invitation

Validation:

```text
Token HMAC exists.
Invitation status is pending.
Invitation is not expired.
Authenticated user exists.
Authenticated user is active.
Authenticated user has verified email.
Canonical authenticated email equals canonical invitation email.
User is not already an active team member.
Invitation role belongs to the same team and is not deleted.
SoD rules are not violated.
```

Do not use token prefix in the idempotency fingerprint. Use either:

```text
body = { "token_hmac": computed_token_hmac }
```

or:

```text
body = { "invitation_id": resolved_invitation_id }
```

The raw token must never enter logs, audit, idempotency storage, or error responses.

## 16.10 Change Member Role

Validation:

```text
Actor has team_member:update_role.
Path team_id and body team_id cannot disagree.
Target membership is active.
New role belongs to same team and is not deleted.
Expected membership version matches.
If demoting owner, service locks team row and checks active owner count.
DB trigger enforces last owner at commit.
SoD rules are not violated.
```

Update:

```sql
UPDATE team_memberships
SET role_id = :new_role_id,
    version = version + 1
WHERE id = :membership_id
  AND team_id = :team_id
  AND status = 'active'
  AND version = :expected_version;
```

Zero rows affected returns `409 Conflict` or `412 Precondition Failed` depending on API convention.

## 16.11 Remove Member

Validation:

```text
Actor has team_member:remove.
Target membership is active.
Expected version matches.
If target is owner, service locks team row and checks owner count.
Database trigger enforces last-owner invariant.
Membership is soft-removed, not hard-deleted.
```

Update:

```sql
UPDATE team_memberships
SET status = 'removed',
    removed_at = NOW(),
    removed_by = :actor_id,
    version = version + 1
WHERE id = :membership_id
  AND team_id = :team_id
  AND status = 'active'
  AND version = :expected_version;
```

## 16.12 Create / Update / Delete Custom Role

Create:

```text
Actor has role:create.
Role name is unique among active team roles.
Role name is not a reserved system role name.
Permission IDs exist.
System role flag cannot be set by normal API.
Permission set is checked against SoD policy.
```

Update:

```text
Actor has role:update.
Role belongs to team and is not deleted.
System role name cannot change.
Expected version matches.
Permission changes must pass SoD impact checks for all active users assigned the role.
```

Delete:

```text
System roles cannot be deleted.
Custom roles are soft-deleted.
Role with active memberships cannot be deleted.
Role with active temporary grants cannot be deleted.
Historical memberships may continue referencing the role.
```

## 16.13 Grant Temporary Access

Validation:

```text
Actor has access_grant:create.
Target user is active.
Team is active.
Role belongs to same team and is not deleted.
expires_at is bounded by maximum policy.
Reason code is safe enum.
Ticket reference is HMACed, not stored raw.
SoD is evaluated against effective permissions.
High-risk or long-lived grant requires recent MFA verification.
```

## 16.14 Suspend User Globally

Required role:

```text
platform_admin or platform_owner
```

Additional rules:

```text
platform_admin cannot suspend platform_owner.
Cannot suspend final active platform_owner.
Target must not be anonymized.
Expected version must match.
```

Transaction:

```text
1. Reserve idempotency.
2. Lock target user.
3. Check platform permission and last-platform-owner invariant.
4. Set status = suspended.
5. Revoke active sessions.
6. Revoke active refresh tokens.
7. Optionally revoke active temporary grants.
8. Audit user.suspended.
9. Write outbox event.
10. Complete idempotency.
```

## 16.15 Reactivate User

```text
platform_admin or platform_owner required.
Can reactivate suspended users.
Cannot reactivate anonymized users.
Does not automatically restore sessions.
Does not automatically restore expired grants.
```

## 16.16 Anonymize User

Required role:

```text
platform_admin or platform_owner
```

Rules:

```text
platform_admin cannot anonymize platform_owner.
Cannot anonymize final active platform_owner.
Expected version must match.
```

Transaction:

```text
1. Reserve idempotency.
2. Lock target user.
3. Compute email_hmac before scrubbing.
4. Replace email with synthetic non-identifying value.
5. Replace email_canonical consistently.
6. Null display_name.
7. Null password_hash.
8. Disable MFA.
9. Null MFA secret and key ID.
10. Revoke and remove recovery codes.
11. Revoke sessions.
12. Null session IP and user-agent.
13. Revoke refresh tokens.
14. Revoke active temporary access grants.
15. Revoke pending email/password tokens.
16. Set status = anonymized.
17. Set deleted_at.
18. Write sanitized audit event with email_hmac only.
19. Write sanitized or encrypted deleted snapshot.
20. Complete idempotency.
```

## 16.17 Delete Team

```text
Actor has team:delete.
High-risk action requires recent MFA verification.
Team is soft-deleted: status = deleted, deleted_at = NOW().
Active memberships may remain as historical rows or be marked removed by policy.
Outbox emits team.deleted notifications as needed.
Last-owner trigger allows this because the team is no longer live.
Hard delete occurs only through retention jobs under maintenance role.
```

---

# 17. Outbox Worker Design

## 17.1 Claim Batch

```sql
SELECT *
FROM outbox_events
WHERE status IN ('pending', 'failed')
  AND attempts < max_attempts
  AND next_attempt_at <= NOW()
ORDER BY created_at
LIMIT 50
FOR UPDATE SKIP LOCKED;
```

Then update each claimed event:

```sql
UPDATE outbox_events
SET status = 'processing',
    attempts = attempts + 1,
    locked_at = NOW(),
    locked_by = :worker_id
WHERE id = :id;
```

## 17.2 Backoff

```python
def backoff_seconds(attempts: int) -> int:
    return min(3600, (2 ** attempts) * 10)
```

## 17.3 Success Handling

```sql
UPDATE outbox_events
SET status = 'sent',
    processed_at = NOW(),
    locked_at = NULL,
    locked_by = NULL,
    sensitive_payload_ciphertext = NULL,
    sensitive_payload_key_id = NULL,
    purge_after = NOW() + INTERVAL '24 hours'
WHERE id = :id;
```

## 17.4 Failure Handling

```sql
UPDATE outbox_events
SET status = CASE
        WHEN attempts >= max_attempts THEN 'dead_letter'
        ELSE 'failed'
    END,
    next_attempt_at = NOW() + (:backoff_seconds || ' seconds')::interval,
    locked_at = NULL,
    locked_by = NULL,
    last_error = :truncated_sanitized_error
WHERE id = :id;
```

## 17.5 Recovery Job

```sql
UPDATE outbox_events
SET status = 'failed',
    locked_at = NULL,
    locked_by = NULL,
    next_attempt_at = NOW(),
    last_error = 'processing lock expired'
WHERE status = 'processing'
  AND locked_at < NOW() - INTERVAL '15 minutes';
```

## 17.6 Sensitive Payload Cleanup

```sql
UPDATE outbox_events
SET sensitive_payload_ciphertext = NULL,
    sensitive_payload_key_id = NULL
WHERE sensitive_payload_ciphertext IS NOT NULL
  AND (
      status IN ('sent', 'dead_letter')
      OR purge_after < NOW()
  );
```

---

# 18. API Contract

Base URL:

```text
/api/v1
```

## 18.1 Common Headers

Authenticated requests:

```text
Authorization: Bearer <access_token>
```

Mutating business-resource endpoints:

```text
Idempotency-Key: <uuid>
```

Optimistic locking:

```text
If-Match: <version>
```

Use `If-Match` as the standard. Accept `expected_version` in JSON only if there is a strong client compatibility reason.

## 18.2 Standard Error Shape

```json
{
  "error": {
    "code": "conflict",
    "message": "This record changed. Reload and retry.",
    "request_id": "uuid",
    "correlation_id": "uuid"
  }
}
```

Standard error codes:

```text
400 invalid_request
401 unauthenticated
403 permission_denied
404 not_found
409 conflict
409 idempotency_conflict
409 idempotency_in_progress
412 precondition_failed
422 validation_failed
425 too_early
429 rate_limited
500 internal_error
```

## 18.3 Authentication APIs

```text
POST   /auth/register
POST   /auth/email/verify/request
POST   /auth/email/verify/confirm
POST   /auth/login
POST   /auth/refresh
POST   /auth/logout
POST   /auth/password/change
POST   /auth/password/reset/request
POST   /auth/password/reset/confirm
POST   /auth/mfa/setup
POST   /auth/mfa/enroll
POST   /auth/mfa/verify
POST   /auth/mfa/recovery-code
POST   /auth/mfa/recovery-codes/regenerate
POST   /auth/mfa/reset
GET    /.well-known/jwks.json
```

## 18.4 Teams

```text
POST   /teams
GET    /teams
GET    /teams/{team_id}
PUT    /teams/{team_id}
DELETE /teams/{team_id}
```

Permissions:

```text
POST   /teams              active authenticated user
GET    /teams              active authenticated user
GET    /teams/{team_id}    team:read
PUT    /teams/{team_id}    team:update
DELETE /teams/{team_id}    team:delete + recent MFA
```

## 18.5 Team Members

```text
GET    /teams/{team_id}/members
PUT    /teams/{team_id}/members/{membership_id}/role
DELETE /teams/{team_id}/members/{membership_id}
```

Permissions:

```text
GET      team_member:read
PUT      team_member:update_role
DELETE   team_member:remove
```

## 18.6 Team Invitations

```text
POST   /teams/{team_id}/invitations
GET    /teams/{team_id}/invitations
POST   /teams/{team_id}/invitations/{invitation_id}/resend
DELETE /teams/{team_id}/invitations/{invitation_id}
POST   /invitations/accept
```

Acceptance rules:

```text
Authenticated active user required.
Email must be verified.
Authenticated canonical email must match invitation canonical email.
```

## 18.7 Team Roles

```text
GET    /teams/{team_id}/roles
POST   /teams/{team_id}/roles
GET    /teams/{team_id}/roles/{role_id}
PUT    /teams/{team_id}/roles/{role_id}
DELETE /teams/{team_id}/roles/{role_id}
```

## 18.8 Temporary Access Grants

```text
POST   /teams/{team_id}/access-grants
GET    /teams/{team_id}/access-grants
DELETE /teams/{team_id}/access-grants/{grant_id}
```

Permissions:

```text
POST     access_grant:create + recent MFA if long-lived/high-risk
GET      access_grant:read
DELETE   access_grant:revoke
```

## 18.9 Team Audit Logs

```text
GET /teams/{team_id}/audit-logs
```

Permission:

```text
audit_log:read
```

## 18.10 Sessions

```text
GET    /me/sessions
DELETE /me/sessions/{session_id}
POST   /me/sessions/revoke-all
GET    /admin/users/{user_id}/sessions
POST   /admin/users/{user_id}/sessions/revoke-all
```

## 18.11 Platform Admin

```text
GET    /admin/users
GET    /admin/users/{user_id}
POST   /admin/users/{user_id}/suspend
POST   /admin/users/{user_id}/reactivate
POST   /admin/users/{user_id}/anonymize
GET    /admin/audit-logs
```

Required role:

```text
platform_admin or platform_owner
```

## 18.12 Platform Role Management

```text
GET    /admin/platform-roles
POST   /admin/users/{user_id}/platform-roles
DELETE /admin/users/{user_id}/platform-roles/{platform_role_id}
```

Required role:

```text
platform_owner
```

## 18.13 Webhooks

```text
GET    /teams/{team_id}/webhooks
POST   /teams/{team_id}/webhooks
PUT    /teams/{team_id}/webhooks/{webhook_id}
DELETE /teams/{team_id}/webhooks/{webhook_id}
GET    /teams/{team_id}/webhooks/{webhook_id}/deliveries
POST   /teams/{team_id}/webhooks/{webhook_id}/rotate-secret
```

Permissions:

```text
GET      webhook:read
POST     webhook:create
PUT      webhook:update
DELETE   webhook:delete
rotate   webhook:update + recent MFA
```

---

# 19. UI Blueprint

## 19.1 Header

```text
[Team Selector] [Search] [Notifications] [User Menu]
```

Team selector shows only teams where the user has active membership or active temporary access.

## 19.2 Team Settings Tabs

```text
Members
Invitations
Roles
Temporary Access
Audit Logs
Webhooks
Billing
```

Rules:

```text
Hide controls when user lacks permission.
Disable apparent last-owner demotion/removal in UI.
Show server-side conflict if DB rejects concurrent owner changes.
Show optimistic-locking conflict as: “This record changed. Reload and retry.”
Never rely on UI enforcement alone.
```

## 19.3 Members UI

Columns:

```text
User ID / display label
Email if viewer has appropriate mutable-table permission
Role
Status
Joined
Version
Actions
```

## 19.4 Invitations UI

Columns:

```text
Email
Role
Expiration
Status
Send count
Last sent
Actions: Resend, Cancel
```

Acceptance errors:

```text
Invalid or expired invitation
Email must be verified
Invitation does not match this account
Already a member
Invitation role is no longer valid
```

## 19.5 Roles UI

Columns:

```text
Name
Description
System role
Permission count
Assigned active members
Version
Actions
```

Rules:

```text
System roles show locked badge.
Custom roles can be edited with role:update.
Custom roles can be deleted only if not actively assigned.
SoD conflicts display before save.
```

## 19.6 Temporary Access UI

Columns:

```text
User
Role
Reason code
Ticket reference fingerprint
Granted by
Granted at
Expires at
Status
Actions: Revoke
```

## 19.7 Audit UI

Display:

```text
Timestamp
Actor ID or system
Action
Entity type
Entity ID
Safe details
Request ID
Correlation ID
```

Do not display from audit records:

```text
Raw IP
Raw user-agent
Raw email
Raw display name
Raw token
Free-text sensitive reason
```

## 19.8 Platform Admin UI

Tabs:

```text
All Users
Sessions
Global Audit
Platform Roles
Security Events
Operational Jobs
```

Actions:

```text
Suspend user
Reactivate user
Anonymize user
Revoke sessions
Assign platform role
Remove platform role
```

Platform role assignment/removal visible only to `platform_owner`.

---

# 20. Security Controls

## 20.1 Passwords

```text
Prefer Argon2id.
If bcrypt is used, minimum cost = 12.
Support password hash migration.
Never store plaintext passwords.
Never log plaintext passwords or password hashes.
Password reset tokens are HMACed, expiring, and single-use.
Password reset completion revokes active sessions and refresh tokens unless policy explicitly says otherwise.
```

## 20.2 Tokens

```text
Access tokens expire after 15 minutes.
Refresh tokens are HMACed before storage.
Refresh tokens rotate on every refresh.
Refresh token family is tracked.
Reuse of an old refresh token revokes the entire family and session.
Suspended users cannot refresh.
Anonymized users cannot refresh.
Session revocation takes effect on next refresh or high-risk introspection check.
Verification/reset/invitation tokens are single-use where applicable.
Raw tokens are never stored outside encrypted outbox payloads.
```

JWT rules:

```text
Include jti.
Do not include full permissions.
Rotate signing keys.
Expose JWKS.
Use revocation list or session introspection for high-risk endpoints.
```

## 20.3 MFA

```text
TOTP secrets are envelope-encrypted.
Recovery codes are shown once.
Recovery codes are stored individually hashed.
Using a recovery code marks only that code as used.
Admin can force MFA reset.
High-risk actions require recent MFA verification.
```

High-risk actions:

```text
Disable MFA
Regenerate recovery codes
Anonymize user
Assign platform role
Remove platform role
Create long-lived temporary access grant
Rotate webhook secret
Change billing settings
Delete team
```

## 20.4 Rate Limiting

Apply limits to:

```text
/auth/register
/auth/login
/auth/refresh
/auth/password/reset/request
/auth/password/reset/confirm
/auth/email/verify/request
/auth/email/verify/confirm
/auth/mfa/verify
/invitations/accept
/team invitations create/resend
/webhook delivery retry paths
```

Dimensions:

```text
IP HMAC
email_hmac
user_id
team_id
endpoint
```

## 20.5 Browser Security

Cookie-based auth:

```text
SameSite=Lax or Strict
Secure
HttpOnly
CSRF token on mutating requests
```

Bearer token SPA:

```text
Do not store refresh tokens in localStorage.
Prefer secure HTTP-only refresh cookie.
Keep access token in memory.
```

## 20.6 Secrets and Key Management

Store in KMS or secret manager:

```text
Audit HMAC secret
Email HMAC secret
Invitation token HMAC secret
Email verification token HMAC secret
Password reset token HMAC secret
Refresh token HMAC secret
MFA encryption key
Outbox sensitive payload encryption key
Webhook signing secret encryption key
Webhook secret fingerprint HMAC secret
JWT signing keys
```

Each cryptographic key needs:

```text
key_id
rotation policy
decommission policy
emergency revoke procedure
```

---

# 21. Background Jobs

| Job | Frequency | Purpose |
|---|---:|---|
| Expire invitations | Every 15 minutes | Mark pending expired invitations as expired. |
| Expire email verification tokens | Every 15 minutes | Mark stale pending verification tokens expired. |
| Expire password reset tokens | Every 15 minutes | Mark stale pending reset tokens expired. |
| Cleanup idempotency | Hourly | Delete expired idempotency records. |
| Recover stuck idempotency | Every 5 minutes | Mark stale processing rows as failed. |
| Recover stuck outbox | Every 5 minutes | Move stale processing events back to failed. |
| Cleanup outbox sensitive payloads | Hourly | Null encrypted sensitive payload after send/dead-letter/TTL. |
| Session cleanup | Hourly | Revoke or purge expired sessions according to retention policy. |
| Refresh-token cleanup | Hourly | Purge expired token records after retention. |
| Dormant account suspension | Daily | Suspend inactive accounts after configured period. |
| Temporary grant expiry | Every 5 minutes | Expire/revoke temporary access grants. |
| Audit partition maintenance | Monthly | Create next partition and archive old partitions. |
| Deleted snapshot retention | Daily | Purge or crypto-shred snapshots past retention. |
| Webhook dead-letter review | Daily | Surface failed webhooks for admin action. |

---

# 22. Observability

## 22.1 Metrics

```text
auth_register_success_total
auth_login_success_total
auth_login_failed_total
auth_refresh_success_total
auth_refresh_failed_total
refresh_token_reuse_detected_total
mfa_verify_failed_total
mfa_enabled_total
mfa_disabled_total
authorization_denied_total
idempotency_replayed_total
idempotency_conflict_total
idempotency_in_progress_total
invitation_created_total
invitation_accepted_total
invitation_accept_failure_total
session_revoked_total
audit_inserted_total
audit_write_failure_total
outbox_pending_count
outbox_failed_count
outbox_dead_letter_count
outbox_processing_stale_count
suspended_user_auth_attempt_total
anonymization_completed_total
platform_role_assignment_total
platform_role_removed_total
platform_owner_protection_block_total
temporary_access_granted_total
temporary_access_revoked_total
webhook_delivery_success_total
webhook_delivery_failed_total
```

## 22.2 Alerts

Alert on:

```text
Outbox dead-letter count > threshold
Login failure spike
MFA failure spike
Refresh-token reuse detected
Permission denied spike
Platform role assignment
Final platform owner protection blocked mutation
User anonymization
Dormant suspension batch failure
Audit insert failure
Audit partition creation failure
Webhook delivery failure spike
```

## 22.3 Structured Logs

Include:

```text
correlation_id
request_id
actor_id
team_id
endpoint
method
status_code
error_code
idempotency_key_present
request_fingerprint
```

Never log:

```text
Raw Authorization header
Raw cookies
Raw invitation tokens
Raw refresh tokens
Raw access tokens
Passwords
Password hashes
TOTP secrets
Recovery codes
Full invitation links
Full password reset links
Encrypted payload plaintext
```

---

# 23. Testing Strategy

## 23.1 Authorization Tests

```text
Suspended user cannot authorize team action.
Anonymized user cannot authorize team action.
Deleted user cannot authorize team action.
Platform admin cannot access team data unless also team member or granted access.
Team admin in Team A cannot access Team B.
Viewer cannot read billing, audit logs, members, roles, or webhooks unless explicitly granted.
Manager cannot invite members.
Admin cannot delete team.
Admin cannot create/update/delete custom roles by default.
Admin cannot create/revoke temporary access grants by default.
Temporary grant grants permission before expiry.
Temporary grant stops granting permission after expiry.
Temporary grant stops granting permission after revocation.
Same-team role assignment cannot be violated manually in SQL.
Deleted role cannot authorize permissions.
```

## 23.2 Platform Tests

```text
platform_owner satisfies platform_admin checks.
platform_admin satisfies platform_support checks.
platform_support does not satisfy platform_admin checks.
platform_admin cannot assign platform_owner.
platform_owner is required for platform role assignment.
Platform roles do not grant team permissions.
Cannot remove final active platform_owner.
Cannot suspend final active platform_owner.
Cannot anonymize final active platform_owner.
```

## 23.3 Idempotency Tests

```text
Same key + same request returns original response.
Same key + different body returns 409.
Same key + different path returns 409.
Same key + different method returns 409.
Concurrent duplicate requests create only one invitation.
Concurrent duplicate requests do not pass select-then-insert race.
Processing request lock expiry works.
Committed mutation always has completed idempotency response.
Deterministic 4xx response can be replayed.
Token issuance responses are not replayed from idempotency storage.
```

## 23.4 Audit Tests

```text
No raw email in audit_logs.
No raw IP in audit_logs.
No raw user-agent in audit_logs.
No token in audit_logs.
No display name in audit_logs.
No free-text PII reason in audit_logs.
One command can create multiple audit events with same idempotency key.
Audit rows cannot be updated by application role.
Audit rows cannot be deleted by application role.
request_id and correlation_id are preserved for traceability.
```

## 23.5 Last-Owner Tests

```text
Cannot remove only team owner.
Cannot demote only team owner.
Two concurrent owner removals cannot leave zero owners.
Two concurrent owner demotions cannot leave zero owners.
Database trigger blocks invariant violation even if service bug exists.
Team soft deletion does not fail because of last-owner trigger.
```

## 23.6 Invitation Tests

```text
Expired token rejected.
Revoked token rejected.
Accepted token rejected on reuse.
Authenticated email must match invited email.
Email must be verified before acceptance.
Existing active member cannot accept duplicate invitation.
Duplicate pending invitation to same team/email is rejected or idempotently replayed.
Invitation role must belong to same team.
Invitation role cannot be deleted.
Raw token is never stored in database.
Raw token is never stored in audit, logs, or idempotency storage.
Invitation email appears only in mutable operational table and encrypted outbox payload.
```

## 23.7 Refresh-Token Tests

```text
Refresh rotates token.
Old token reuse is detected.
Reuse revokes token family.
Reuse revokes session.
Suspended user cannot refresh.
Anonymized user cannot refresh.
Revoked session cannot refresh.
Expired refresh token cannot refresh.
```

## 23.8 Privacy Tests

```text
Anonymization scrubs user email.
Anonymization scrubs display name.
Anonymization nulls password hash.
Anonymization nulls MFA secret.
Anonymization revokes and removes recovery codes.
Anonymization revokes sessions.
Anonymization revokes refresh tokens.
Anonymization nulls session IP.
Anonymization nulls session user-agent.
Deleted snapshots reject obvious raw PII.
Outbox sensitive payload is purged after send.
Webhook response body is sanitized or hashed.
```

## 23.9 SoD Tests

```text
Role creation rejects conflicting permission combinations if policy requires.
Role update checks all affected users.
Temporary grant cannot create conflicting effective permission set.
Permanent membership plus temporary grant is evaluated together.
Global SoD rule applies to all teams.
Team SoD rule applies only to that team.
```

## 23.10 Outbox and Webhook Tests

```text
Worker uses FOR UPDATE SKIP LOCKED.
Concurrent workers do not process same event.
provider_message_key is supplied to providers that support idempotency.
Sensitive payload decrypts only in worker.
Failed event retries with backoff.
Event moves to dead_letter after max attempts.
Stuck processing event recovers.
Sensitive payload is nulled after success.
Webhook secret can be decrypted for signing.
Webhook response body is sanitized or hashed.
```

## 23.11 Load Tests

```text
Permission check query latency.
Audit log insertion throughput.
Outbox processing throughput.
Login and refresh throughput.
Team member list pagination.
Audit log pagination.
Webhook delivery throughput.
Idempotency contention under duplicate requests.
```

---

# 24. Deployment and Migration Plan

## 24.1 Environments

```text
dev
test
staging
production
```

## 24.2 Migration Order

```text
1. Extensions and shared functions
2. Bootstrap lock
3. Users
4. Email verification and password reset token tables
5. MFA recovery codes
6. Teams
7. Roles and permissions
8. Role permissions
9. Invitations
10. Memberships
11. Temporary access grants
12. Platform roles and user platform roles
13. Sessions and refresh tokens
14. Audit logs and partitions
15. Idempotency store
16. Outbox events
17. SoD rules
18. Deleted snapshots
19. Webhooks
20. Triggers and constraints
21. Seed data
22. Database grants
```

## 24.3 Database Grants

```text
Application role can INSERT audit_logs.
Application role can SELECT audit_logs only through approved access paths.
Application role cannot UPDATE audit_logs.
Application role cannot DELETE audit_logs.
Outbox worker has minimal access to outbox_events and decrypt permissions only for required payloads.
Operational job role has limited cleanup permissions.
Retention hard-delete jobs run under a separate maintenance role.
```

## 24.4 Rollback Strategy

```text
Schema migrations must be backward-compatible where possible.
Avoid destructive migrations.
Use feature flags for new flows.
Deploy additive schema first, then code, then constraints.
Test rollback in staging.
Retention hard-delete jobs must be disabled during rollback windows.
```

---

# 25. Implementation Phases

## Phase 1 — Foundation, 3–4 weeks

Deliverables:

```text
Schema and migrations
Seed roles and permissions
First-user bootstrap with lock
User register/login/refresh/logout
Email verification
Password hashing
Refresh-token rotation basics
Persistent lockout fields and rate-limit integration
Idempotency reservation system
Sanitized audit writer
Outbox worker with encryption support
Team creation with concrete owner role
```

Exit criteria:

```text
First user can bootstrap platform_owner and first team owner.
Users can register and verify email.
Users can log in, refresh, and log out.
Refresh tokens rotate and reuse detection is test-covered.
Team roles are copied from templates.
Audit logs contain no raw PII.
Idempotency survives concurrent duplicate requests.
```

## Phase 2 — Team Management, 4 weeks

Deliverables:

```text
Member list
Invite/resend/cancel invitation
Accept invitation
Role change
Member removal
Last-owner service checks
Last-owner DB trigger
Team-scoped authorization middleware
Basic team settings UI
```

Exit criteria:

```text
Team owner can manage members.
Invitation acceptance requires matching verified email.
Same-team role FK blocks invalid assignments.
Last-owner invariant survives concurrent requests.
Duplicate pending invitations are handled safely.
```

## Phase 3 — Roles, SoD, and Temporary Access, 3–4 weeks

Deliverables:

```text
Custom role create/update/delete
Role permission editor
SoD conflict rules
SoD validation for role changes
Temporary team access grants
Temporary access revocation
Temporary access expiry job
```

Exit criteria:

```text
Custom roles work safely.
System roles cannot be renamed or deleted through normal APIs.
SoD conflicts are blocked.
Temporary grants affect authorization only while active.
```

## Phase 4 — Security Hardening, 4 weeks

Deliverables:

```text
MFA setup/enrollment/verification/recovery
Password reset and password change
Session management
Session revocation UI
Rate limiting
Audit viewer
High-risk recent-MFA checks
Security event metrics
```

Exit criteria:

```text
Suspended user cannot refresh tokens.
Session revocation works.
MFA recovery works.
Login abuse is rate-limited.
Audit viewer exposes sanitized data only.
```

## Phase 5 — Platform Admin, 3 weeks

Deliverables:

```text
Platform roles
Last platform_owner protection
Platform admin user list
Global suspend/reactivate/anonymize
Global session revocation
Global audit viewer
Platform role assignment UI
```

Exit criteria:

```text
Platform/team permissions remain separate.
Anonymization passes privacy tests.
Platform owner is required for platform role assignment.
Final platform owner cannot be removed, suspended, or anonymized.
Platform hierarchy works correctly.
```

## Phase 6 — Operations and Webhooks, 3 weeks

Deliverables:

```text
Invitation expiry job
Email verification/password reset expiry jobs
Session cleanup job
Refresh-token cleanup job
Outbox recovery/dead-letter handling
Outbox provider dedupe support
Outbox sensitive payload cleanup
Webhook subscriptions
Webhook encrypted signing secret storage
Webhook deliveries
Webhook retry/dead-letter UI
Audit partition maintenance
Deleted snapshot retention jobs
```

Exit criteria:

```text
Jobs are idempotent.
Failed outbox events retry safely.
Dead-letter events are visible.
Sensitive payloads are purged.
Webhook signing works with encrypted secrets.
Webhook payloads are sanitized.
Provider-side dedupe is supported where available.
```

## Phase 7 — Enterprise Extensions, Deferred

Deliver only when justified:

```text
SAML
SCIM
Service principals
API keys
JIT approval workflow
Resource-level authorization
ReBAC
Advanced policy engine
Customer-managed keys
Advanced audit export
```

---

# 26. Realistic Delivery Estimate

Assuming:

```text
2–3 backend engineers
1 frontend engineer
part-time security review
part-time DevOps/SRE support
```

| Scope | Estimate |
|---|---:|
| Lean functional MVP | 14–16 weeks |
| Production-ready core IAM | 20–24 weeks |
| Production-ready with strong operations and webhooks | 24–28 weeks |
| Enterprise extensions | Add 8–16+ weeks |

A 13-week delivery is plausible only for a lean internal MVP with pre-existing authentication, worker, migration, observability, and secrets-management foundations.

---

# 27. Risk Register

| Risk | Impact | Mitigation |
|---|---|---|
| Permission bug | Privilege escalation | Central authorization engine, DB same-team constraints, tests |
| Viewer over-permissioning | Sensitive data exposure | Explicit role-permission matrix |
| Admin over-permissioning | Privilege escalation | Keep stricter admin default; introduce separate elevated role if needed |
| Raw token leakage | Account/team compromise | HMAC tokens, encrypted outbox, no prefixes, log filters |
| Idempotency race | Duplicate mutation | Insert-first reservation, PK on scope/key |
| Idempotency key reused across endpoint | Wrong replay | Fingerprint includes method/path/query/body; mismatch 409 |
| Last team owner removed | Orphaned team | Service lock + deferred DB trigger |
| Last platform owner removed | Platform lockout | Service lock + deferred DB trigger |
| Audit contains PII | Compliance failure | Allowlist sanitizer + tests + restricted DB grants |
| Refresh-token reuse missed | Session hijack persists | Durable refresh_tokens table and family/session revocation |
| Outbox duplicate send | Duplicate emails/webhooks | provider_message_key, provider idempotency, attempts, dedupe IDs |
| Worker crash | Stuck side effects | Lock lease and recovery job |
| Webhook secret unavailable | Cannot sign delivery | Store encrypted signing secret, not hash-only |
| Team hard-delete conflicts with role triggers | Retention job failure | Soft delete in app; hard delete under maintenance procedure |
| Migration failure | Production outage | Staging migration, additive rollout, rollback testing |
| Persistent lockout abuse | Account denial-of-service | Rate-limit unlock policy, admin reset, careful thresholding |

---

# 28. Final Engineering Acceptance Checklist

| Requirement | Acceptance Condition |
|---|---|
| Team authorization | Every team endpoint checks `team_id`, resource, action, and active user state. |
| Platform authorization | Every global endpoint checks platform role hierarchy. |
| Platform/team separation | Platform roles do not grant team permissions. |
| Concrete role isolation | Memberships, invitations, and grants reference concrete same-team roles only. |
| Explicit viewer permissions | Viewer cannot read billing, audit logs, members, roles, or webhooks by default. |
| Strict admin default | Admin cannot delete team, read audit logs, create/revoke temporary grants, or mutate roles by default. |
| First-user bootstrap | Race-safe bootstrap creates first platform_owner and first concrete team owner. |
| Last team owner | Cannot remove/demote final active team owner. |
| Last platform owner | Cannot remove, suspend, or anonymize final active platform_owner. |
| Invitation token security | Raw token not stored in invitation table, audit, logs, or idempotency. |
| Invitation email binding | Accept requires verified matching canonical email. |
| Invitation duplicate policy | Only one pending invite exists per team + canonical email unless exception approved. |
| Refresh-token rotation | Refresh token rotates on every refresh. |
| Refresh-token reuse detection | Reuse revokes token family and session. |
| MFA | Secret encrypted; recovery codes individually hashed and tracked. |
| Email verification | Verification tokens are HMACed, expiring, single-use. |
| Password reset | Reset tokens are HMACed, expiring, single-use. |
| Idempotency | Same scope/key with different request returns `409`; mutation response stored before commit. |
| Audit immutability | App role cannot update/delete audit logs. |
| Audit privacy | Audit logs contain no raw PII, tokens, IPs, user agents, or free-text sensitive values. |
| Audit traceability | request_id and correlation_id are available for forensic tracing. |
| Outbox security | Sensitive payload encrypted and purged after send/dead-letter/TTL. |
| Outbox dedupe | provider_message_key or equivalent dedupe is available. |
| Webhook security | Signing secret stored encrypted; delivery records sanitized. |
| Anonymization | User email/name/MFA/password/session PII scrubbed and refresh tokens revoked. |
| UI | UI hides unavailable actions but backend and DB enforce all controls. |
| Cleanup | Expired sessions, refresh tokens, outbox, invitations, verification/reset tokens, snapshots, and idempotency records cleaned. |
| Observability | Metrics, alerts, correlation IDs, request IDs, and redacted structured logs implemented. |

---

# 29. Final Position

This Version 4.0 is the recommended master implementation blueprint.

It keeps the canonical combined version’s build-ready structure, stronger data-classification model, corrected idempotency semantics, encrypted webhook secret design, mature worker operations, expanded testing strategy, rollback planning, and risk register.

It also restores the earlier master blueprint’s stronger least-privilege defaults, especially the stricter team-admin permission model, explicit role-permission matrix, provider outbox dedupe field, persistent login lockout fields, request-level audit traceability, and stricter duplicate invitation policy.

The result is a more secure and implementation-ready enterprise IAM plan with fewer hidden privilege-escalation risks and clearer operational controls.
