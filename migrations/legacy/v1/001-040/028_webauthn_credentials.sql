-- Migration 028: WebAuthn/FIDO2 credentials
-- v1.1.0 Track 5: WebAuthn as an additional MFA option

CREATE TABLE IF NOT EXISTS user_webauthn_credentials (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id_hash      TEXT NOT NULL,           -- SHA-256 hash of credential_id (for lookup)
    credential_id_bytes     BYTEA NOT NULL,           -- raw credential_id bytes (needed for assertion)
    public_key              BYTEA NOT NULL,           -- COSE public key
    sign_count              BIGINT NOT NULL DEFAULT 0,
    device_type             TEXT NOT NULL DEFAULT '', -- platform, cross-platform
    backup_eligible         BOOLEAN NOT NULL DEFAULT false,
    backup_state            BOOLEAN NOT NULL DEFAULT false,
    label                   TEXT NOT NULL DEFAULT '',
    aaguid                  TEXT NOT NULL DEFAULT '',
    transports              TEXT[] NOT NULL DEFAULT '{}',
    status                  TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'disabled')),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at            TIMESTAMPTZ,
    disabled_at             TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_webauthn_cred_id_hash ON user_webauthn_credentials(credential_id_hash);
CREATE INDEX IF NOT EXISTS idx_webauthn_user ON user_webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_status ON user_webauthn_credentials(status);

-- Permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('mfa.webauthn.manage', 'Register and manage WebAuthn credentials', 'mfa', 'webauthn', 'low')
ON CONFLICT (name) DO NOTHING;

-- Assign to all roles (every authenticated user can manage their own MFA)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name = 'mfa.webauthn.manage'
ON CONFLICT DO NOTHING;
