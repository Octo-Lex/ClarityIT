# ClarityIT v1.1 MFA and Approval Model

## Document Status
- **Version**: 1.1.0
- **Date**: 2026-06-14
- **Scope**: TOTP MFA, WebAuthn/FIDO2 MFA, approval workflow, and their interaction

## 1. MFA Implementation

### 1.0 WebAuthn Scope

WebAuthn/FIDO2 is an **MFA step-up verification only** (v1.1 Track 5).

- WebAuthn is used to refresh `recent_mfa_at` after an authenticated session exists.
- WebAuthn is **not** integrated into the primary login flow.
- WebAuthn does **not** replace password-based authentication.
- WebAuthn does **not** change approval, Tool Gateway, Proxmox mutation, or autonomy semantics.
- `RequireRecentMFA` accepts any MFA verification (TOTP, recovery code, or WebAuthn) because all three set `recent_mfa_at` on the user's session.

### 1.1 TOTP Enrollment

**Flow**:
1. User calls `POST /api/auth/mfa/totp/enroll` → receives provisioning URI + raw secret (shown ONCE)
2. Secret is AES-256-GCM encrypted at rest using key derived from `MFA_KEY` via HKDF-SHA256
3. User adds to authenticator app, enters 6-digit code via `POST /api/auth/mfa/totp/verify-enrollment`
4. On success: factor status → `active`, 10 recovery codes generated and shown once

**Constraints**:
- Only one active factor per user (409 if existing)
- Pending enrollments deleted before creating new one
- Raw secret returned exactly once in enrollment response
- All subsequent responses omit the secret

### 1.2 TOTP Validation

- **Algorithm**: SHA1 (RFC 6238 standard for TOTP)
- **Digits**: 6
- **Period**: 30 seconds
- **Window**: Current time only (no ±1 drift window — stricter)
- **Implementation**: `internal/mfa/totp.go` using `crypto/hmac` + `crypto/sha1`

### 1.3 MFA Challenge Flow

```
POST /api/auth/mfa/challenge → { password } → { challenge_id, expires_at }
POST /api/auth/mfa/verify → { challenge_id, code | recovery_code } → sets recent_mfa_at on ALL active sessions
```

- Challenge TTL: 5 minutes
- Challenge is single-use (verified flag)
- Failed attempts: counter on factor, 5 → 15-minute lockout
- Recovery codes: HMAC-SHA256 hashed, single-use, 10 generated

### 1.4 MFA Validity Window

- `recent_mfa_at` set on ALL active `user_sessions` for the user
- Validity: 5 minutes from `recent_mfa_at`
- Checked by: `RequireRecentMFA` middleware, `PolicyEvaluator.hasRecentMFA()`, `ActionHandler.hasRecentMFA()`
- Formula: `time.Since(recentMFA) < 5 * time.Minute`

### 1.5 Sensitive Operations Requiring Recent MFA

| Operation | Enforcement Point |
|-----------|-------------------|
| Disable MFA factor | `mfa.Handler.DisableFactor` — checks `hasRecentMFA` |
| Regenerate recovery codes | `mfa.Handler.RegenerateRecoveryCodes` — checks `hasRecentMFA` |
| High-risk tool execution (A4) | `PolicyEvaluator` check 10 |
| High-risk Proxmox action (shutdown) | `ActionHandler.ExecuteAction` |
| Critical Proxmox action (stop) | `ActionHandler.ExecuteAction` |

### 1.6 MFA Secret Storage

```
Raw TOTP Secret (20 bytes)
    ↓ AES-256-GCM Encrypt
Encrypted Secret (nonce + ciphertext)
    ↓ Stored in
user_mfa_factors.secret (bytea column)
```

**Key Derivation**:
```
MFA_KEY (env var, ≥16 chars)
    ↓ HKDF-SHA256
    ↓ info="clarityit-mfa-v1", salt="aes-256-gcm"
32-byte AES-256 key
    ↓ Zeroed from memory after cipher init
```

### 1.7 Recovery Code Storage

```
Raw Recovery Code (string)
    ↓ HMAC-SHA256(MFA_KEY, code)
Hash (hex string)
    ↓ Stored in
mfa_recovery_codes.code_hash (text column, used_at nullable)
```

- 10 codes generated at enrollment
- Shown once in enrollment response
- Single-use: `used_at` set on consumption
- Regeneration requires recent MFA

## 2. Approval Workflow Engine

### 2.1 Data Model

```sql
approval_requests
  - id (UUID, PK)
  - team_id (UUID, FK → teams)
  - action_type (TEXT) — e.g., "proxmox.start"
  - action_target (JSONB) — sanitized, identity-only
  - risk_level (TEXT) — low|medium|high|critical
  - description (TEXT)
  - requested_by (UUID, FK → users)
  - status (TEXT) — pending|approved|rejected|cancelled|expired|executed|failed
  - policy_id (UUID NULLABLE, FK → approval_policies)
  - expires_at (TIMESTAMPTZ)
  - executed_at (TIMESTAMPTZ NULLABLE)
  - created_at, updated_at

approval_decisions
  - id (UUID, PK)
  - approval_id (UUID, FK → approval_requests)
  - decided_by (UUID, FK → users)
  - decision (TEXT) — approved|rejected
  - reason (TEXT)
  - mfa_verified (BOOLEAN)
  - decided_at (TIMESTAMPTZ)
  - UNIQUE(approval_id, decided_by) — immutable, one decision per user

approval_policies
  - id (UUID, PK)
  - team_id (UUID, FK → teams)
  - risk_level (TEXT)
  - auto_approve (BOOLEAN)
  - allow_self_approve (BOOLEAN)
  - min_approvers (INT)
  - require_mfa (BOOLEAN)
  - ttl_seconds (INT)
```

### 2.2 Risk-Based Policy Resolution

```go
// internal/approval/policy.go
func ResolvePolicy(teamID, riskLevel) Policy {
    // 1. Check team-specific DB policy
    // 2. Fall back to builtin defaults:
    //    low: auto-approve, self-approve, 0 approvers, no MFA, 1h TTL
    //    medium: no auto-approve, 1 approver, no MFA, 1h TTL
    //    high: no auto-approve, 1 approver, MFA required, 1h TTL
    //    critical: no auto-approve, 2 approvers, MFA required, 1h TTL
}
```

### 2.3 Self-Approval Prevention

- `requested_by` recorded on creation
- `decided_by` must differ from `requested_by` (except low-risk auto-approve)
- Enforced in `Approve()` handler: `if cl.UserID == req.RequestedBy → 403 "cannot self-approve"`

### 2.4 Critical Risk: Two Distinct Approvers

- `min_approvers = 2` for critical risk
- Enforced by counting `approval_decisions WHERE decision='approved'`
- The UNIQUE(approval_id, decided_by) constraint ensures two DIFFERENT users

### 2.5 Approval Lifecycle States

```
                    ┌──────────┐
                    │ pending  │
                    └────┬─────┘
            ┌──────────┼──────────┐
            ▼          ▼          ▼
       approved    rejected   cancelled
            │
            ▼
       executed ←─── failed (execution error)
```

- `approved` → action can be executed
- `rejected` / `cancelled` → terminal, no execution
- `expired` → `expires_at` passed, no execution
- `executed` → action completed successfully
- `failed` → execution attempted but failed

### 2.6 Approval Verification (10-Point Check)

Used by both `PolicyEvaluator.verifyApproval()` and `ActionHandler.ExecuteAction()`:

1. Approval exists in DB
2. Cross-team blocked (`team_id` match)
3. Already-executed blocked (`executed_at IS NULL`)
4. Status must be `approved`
5. Not expired (`expires_at > NOW`)
6. `action_type` must match requested tool
7. `action_target` must deep-equal match (parsed JSON comparison)
8. Denied payloads sanitized before audit
9. Unknown tools denied before grant lookup
10. A5 fails closed before any DB lookup

## 3. Interaction Between MFA and Approvals

### Decision Matrix

| Risk | Approval Required | MFA Required | Who Decides |
|------|-------------------|-------------|-------------|
| Low | Auto-approved by policy | No | System |
| Medium | 1 human approver | No | Operator |
| High | 1 human approver + recent MFA | Yes | Operator (MFA-verified) |
| Critical | 2 distinct approvers + recent MFA | Yes | 2 Operators (MFA-verified) |

### MFA Check Points

```
Action Request
    ↓
[Approval Workflow]
    ↓ approved?
    ↓
[MFA Check] ← requires recent_mfa_at within 5 min
    ↓ MFA valid?
    ↓
[PolicyEvaluator Check 10] ← hasRecentMFA() for high/critical
    ↓
[Execution] ← ActionHandler also checks hasRecentMFA for shutdown/stop
    ↓
[Audit + Outbox] ← sanitized
```

## 4. Endpoint Reference

### MFA Routes (authenticated, `/api/auth/mfa`)
| Method | Path | Purpose | Recent MFA? |
|--------|------|---------|-------------|
| POST | `/totp/enroll` | Start TOTP enrollment | No |
| POST | `/totp/verify-enrollment` | Verify code, activate factor | No |
| POST | `/challenge` | Create MFA challenge | No |
| POST | `/verify` | Verify TOTP/recovery code | No |
| POST | `/recovery-codes/regenerate` | Generate new recovery codes | ✅ Yes |
| DELETE | `/factors/{factorId}` | Disable MFA factor | ✅ Yes |
| GET | `/factors` | List factors (no secrets) | No |
| GET | `/status` | MFA status | No |

### Approval Routes (authenticated, `/api/teams/{teamId}/approvals`)
| Method | Path | Permission | Idempotency |
|--------|------|-----------|-------------|
| POST | `/` | approvals.create | Required |
| GET | `/` | approvals.read | — |
| GET | `/{approvalId}` | approvals.read | — |
| POST | `/{approvalId}/approve` | approvals.approve | Required |
| POST | `/{approvalId}/reject` | approvals.approve | Required |
| POST | `/{approvalId}/cancel` | approvals.create | Required |
