-- 010_iam_invitations_access_grants.sql

-- Team invitations
CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id),
    token_hash TEXT NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (TRIM(email) <> '')
);

CREATE INDEX IF NOT EXISTS idx_invitations_team ON invitations (team_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invitations_token ON invitations (token_hash) WHERE accepted_at IS NULL;

-- Team access grants (explicit access grants beyond membership)
CREATE TABLE IF NOT EXISTS team_access_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_by UUID NOT NULL REFERENCES users(id),
    grant_type TEXT NOT NULL CHECK (grant_type IN ('explicit', 'delegated', 'temporary')),
    scope TEXT,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_team_access_grants_team ON team_access_grants (team_id);
CREATE INDEX IF NOT EXISTS idx_team_access_grants_user ON team_access_grants (user_id) WHERE revoked_at IS NULL;
