-- Migration 022: MFA TOTP
-- Real TOTP MFA foundation for high-risk approval and execution

-- User MFA factors (one row per enrolled factor)
CREATE TABLE IF NOT EXISTS user_mfa_factors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    factor_type TEXT NOT NULL DEFAULT 'totp',
    secret      BYTEA NOT NULL,                  -- AES-256-GCM encrypted TOTP secret
    status      TEXT NOT NULL DEFAULT 'pending', -- pending, active, disabled
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until   TIMESTAMPTZ,
    UNIQUE(user_id, factor_type)
);

-- MFA challenge records (short-lived)
CREATE TABLE IF NOT EXISTS mfa_challenges (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    factor_id   UUID NOT NULL REFERENCES user_mfa_factors(id) ON DELETE CASCADE,
    challenge   TEXT NOT NULL,                    -- random nonce
    verified    BOOLEAN NOT NULL DEFAULT false,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Recovery codes (hashed, single-use)
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT NOT NULL,                    -- HMAC-SHA256 hash
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Recent MFA state on session
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS recent_mfa_at TIMESTAMPTZ;

-- Seed permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('mfa.enroll', 'Enroll MFA factor', 'mfa', 'enroll', 'low'),
    ('mfa.manage', 'Manage own MFA factors', 'mfa', 'manage', 'low'),
    ('mfa.disable', 'Disable MFA factor', 'mfa', 'disable', 'high')
ON CONFLICT (name) DO NOTHING;

-- Assign MFA permissions to all authenticated roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('mfa.enroll', 'mfa.manage', 'mfa.disable')
  AND r.name IN ('owner', 'admin', 'manager', 'member', 'on_call_engineer', 'infrastructure_engineer', 'security_admin', 'automation_operator')
ON CONFLICT DO NOTHING;
