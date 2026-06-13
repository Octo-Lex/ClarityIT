-- 011_iam_platform_roles.sql

-- Platform roles (separate from team roles)
-- Platform roles grant global administrative authority, NOT team-level permissions.
CREATE TABLE IF NOT EXISTS platform_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed platform roles
INSERT INTO platform_roles (name, description) VALUES
    ('platform_owner',  'Full platform administration. Can manage users, teams, settings, and audit.'),
    ('platform_admin',  'Platform administration without owner-level changes.'),
    ('platform_viewer', 'Read-only platform visibility.')
ON CONFLICT (name) DO NOTHING;

-- User ↔ Platform role assignments
CREATE TABLE IF NOT EXISTS user_platform_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform_role_id UUID NOT NULL REFERENCES platform_roles(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- One active platform role per user per role type
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_platform_role_active
    ON user_platform_roles (user_id, platform_role_id) WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_platform_roles_user ON user_platform_roles (user_id) WHERE revoked_at IS NULL;
