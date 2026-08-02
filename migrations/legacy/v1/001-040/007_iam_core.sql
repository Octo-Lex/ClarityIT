-- 007_iam_core.sql
-- IAM core tables: bootstrap lock, users, teams

-- Bootstrap lock: single-row table to prevent multiple bootstrap calls
CREATE TABLE IF NOT EXISTS bootstrap_lock (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    locked_by_user_id UUID,
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert the single unlocked row
INSERT INTO bootstrap_lock (id, is_locked) VALUES (1, FALSE) ON CONFLICT DO NOTHING;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    avatar_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    token_version INT NOT NULL DEFAULT 1,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (TRIM(email) <> ''),
    CHECK (TRIM(name) <> '')
);

-- Partial unique index: only one active user per email
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email_active ON users (email) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_active ON users (id) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Teams
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    icon TEXT NOT NULL DEFAULT '🏢',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (TRIM(name) <> ''),
    CHECK (TRIM(slug) <> ''),
    CHECK (jsonb_typeof(settings) = 'object')
);

-- Partial unique index: only one active team per slug
CREATE UNIQUE INDEX IF NOT EXISTS uq_teams_slug_active ON teams (slug) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_teams_updated_at
BEFORE UPDATE ON teams
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Roles (for team-scoped roles)
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_system_role BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Permissions
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    risk_level TEXT NOT NULL DEFAULT 'low' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Role-Permission mapping
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Team memberships (role via role_id reference, not role name)
CREATE TABLE IF NOT EXISTS team_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_team_memberships_team ON team_memberships (team_id);
CREATE INDEX IF NOT EXISTS idx_team_memberships_user ON team_memberships (user_id);

CREATE TRIGGER trg_team_memberships_updated_at
BEFORE UPDATE ON team_memberships
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
