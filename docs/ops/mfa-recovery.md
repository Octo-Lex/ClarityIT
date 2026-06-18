# MFA Recovery — ClarityIT v1.0

## Overview

ClarityIT uses RFC 6238 TOTP (Time-based One-Time Password) for multi-factor authentication. This document covers enrollment, daily use, and recovery procedures.

## MFA Enrollment

### Prerequisites
- User must be authenticated
- No existing active MFA factor

### Steps

1. Navigate to `/account/security` in the web UI
2. Click "Enroll TOTP"
3. A provisioning secret is displayed (shown ONCE)
4. Add to your authenticator app (Google Authenticator, Authy, 1Password, etc.)
   - Scan the QR code or enter the secret manually
5. Enter the 6-digit code from your authenticator app
6. **Save your recovery codes** — they are shown only once

### API Enrollment
```bash
# Step 1: Enroll
curl -X POST http://<your-host>:8765/api/auth/mfa/totp/enroll \
  -H "Authorization: Bearer $TOKEN"
# Returns: factor_id, secret (base32), provisioning_uri

# Step 2: Verify with TOTP code
curl -X POST http://<your-host>:8765/api/auth/mfa/totp/verify-enrollment \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"code": "123456"}'
# Returns: recovery_codes (10 codes — SAVE THESE)
```

## Recovery Codes

### What They Are
- 10 single-use codes generated at enrollment
- Each code can substitute for a TOTP code once
- HMAC-SHA256 hashed at rest (never stored plaintext)

### When to Use
- Lost or replaced phone
- Authenticator app deleted
- TOTP clock drift

### Using a Recovery Code
```bash
# Step 1: Create MFA challenge
curl -X POST http://<your-host>:8765/api/auth/mfa/challenge \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"password": "your-password"}'

# Step 2: Verify with recovery code instead of TOTP
curl -X POST http://<your-host>:8765/api/auth/mfa/verify \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"challenge_id": "...", "recovery_code": "your-recovery-code"}'
```

### Regenerating Recovery Codes
Requires recent MFA verification (within 5 minutes):
```bash
curl -X POST http://<your-host>:8765/api/auth/mfa/recovery-codes/regenerate \
  -H "Authorization: Bearer $TOKEN"
# Returns: new set of 10 recovery codes (old codes invalidated)
```

## Lost MFA Factor (Admin Recovery)

If a user loses both their authenticator app AND recovery codes:

### Option A: Admin Resets MFA (Database)
```sql
-- Connect to PostgreSQL
docker exec -it clarityit-postgres-1 psql -U clarityit -d clarityit

-- Disable all factors for the user
UPDATE user_mfa_factors SET status='disabled', disabled_at=NOW()
WHERE user_id = (SELECT id FROM users WHERE email='user@example.com');

-- Invalidate all recovery codes
UPDATE mfa_recovery_codes SET used_at=NOW()
WHERE user_id = (SELECT id FROM users WHERE email='user@example.com')
  AND used_at IS NULL;

-- Clear MFA window
UPDATE user_sessions SET recent_mfa_at=NULL
WHERE user_id = (SELECT id FROM users WHERE email='user@example.com');
```

The user can then re-enroll MFA from `/account/security`.

### Option B: Platform Owner Disables via Admin
Currently there is no admin API endpoint for MFA reset. Use Option A (direct SQL) until this is added in a future release.

## MFA for High-Risk Actions

These actions require recent MFA (within 5 minutes):

| Action | Risk | MFA Required |
|--------|------|-------------|
| Disable MFA factor | — | Yes |
| Regenerate recovery codes | — | Yes |
| Proxmox shutdown | High | Yes |
| Proxmox stop (force) | Critical | Yes |
| A4 agent tool execution (high/critical) | High+ | Yes |

### Verifying MFA Status
```bash
curl http://<your-host>:8765/api/auth/mfa/status \
  -H "Authorization: Bearer $TOKEN"
# Returns: { "enrolled": true, "recent_mfa_at": "...", "mfa_valid": true/false }
```

## Throttling

- **5 failed attempts** → 15-minute lockout
- Lockout applies per-factor
- Lockout clears automatically after 15 minutes

## Security Notes

- TOTP secrets are AES-256-GCM encrypted at rest using `MFA_KEY`
- Key derived via HKDF-SHA256 (info: `clarityit-mfa-v1`)
- Recovery codes are HMAC-SHA256 hashed (64 hex chars)
- Raw secrets/recovery codes never appear in audit logs, outbox events, or API responses (after initial display)
- `MFA_KEY` must be at least 16 characters and must NOT be the same as `HMAC_KEY`
