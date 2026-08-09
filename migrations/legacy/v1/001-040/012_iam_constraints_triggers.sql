-- 012_iam_constraints_triggers.sql
-- Additional constraints, indexes, and triggers for IAM tables

-- Users: email normalization trigger (lowercase + trim)
CREATE OR REPLACE FUNCTION normalize_user_email()
RETURNS TRIGGER AS $$
BEGIN
    NEW.email := LOWER(TRIM(NEW.email));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_normalize_email
BEFORE INSERT OR UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION normalize_user_email();

-- Teams: slug normalization trigger (lowercase, alphanumeric + hyphens)
CREATE OR REPLACE FUNCTION normalize_team_slug()
RETURNS TRIGGER AS $$
BEGIN
    NEW.slug := LOWER(TRIM(REGEXP_REPLACE(NEW.slug, '[^a-z0-9-]', '-', 'g')));
    NEW.slug := TRIM(BOTH '-' FROM NEW.slug);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_teams_normalize_slug
BEFORE INSERT OR UPDATE ON teams
FOR EACH ROW EXECUTE FUNCTION normalize_team_slug();

-- Team memberships: prevent duplicate active memberships
-- (UNIQUE constraint already on table, this is defense-in-depth)

-- Prevent last owner from being removed or demoted
CREATE OR REPLACE FUNCTION protect_last_team_owner()
RETURNS TRIGGER AS $$
DECLARE
    owner_role_id UUID;
    admin_count INT;
BEGIN
    SELECT id INTO owner_role_id FROM roles WHERE name = 'owner';

    -- On role change away from owner
    IF TG_OP = 'UPDATE' AND OLD.role_id = owner_role_id AND NEW.role_id != owner_role_id THEN
        SELECT COUNT(*) INTO admin_count
        FROM team_memberships
        WHERE team_id = NEW.team_id AND role_id = owner_role_id;
        IF admin_count <= 1 THEN
            RAISE EXCEPTION 'Cannot remove the last owner from team';
        END IF;
    END IF;

    -- On deletion of an owner
    IF TG_OP = 'DELETE' AND OLD.role_id = owner_role_id THEN
        SELECT COUNT(*) INTO admin_count
        FROM team_memberships
        WHERE team_id = OLD.team_id AND role_id = owner_role_id;
        IF admin_count <= 1 THEN
            RAISE EXCEPTION 'Cannot remove the last owner from team';
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_protect_last_team_owner_update
BEFORE UPDATE ON team_memberships
FOR EACH ROW EXECUTE FUNCTION protect_last_team_owner();

CREATE TRIGGER trg_protect_last_team_owner_delete
BEFORE DELETE ON team_memberships
FOR EACH ROW EXECUTE FUNCTION protect_last_team_owner();

-- Bootstrap lock: prevent unlocking
CREATE OR REPLACE FUNCTION prevent_bootstrap_unlock()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.is_locked = TRUE AND NEW.is_locked = FALSE THEN
        RAISE EXCEPTION 'Bootstrap lock cannot be reversed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_bootstrap_lock
BEFORE UPDATE ON bootstrap_lock
FOR EACH ROW EXECUTE FUNCTION prevent_bootstrap_unlock();

-- Index for audit_logs by actor (from 002 migration already has entity/team indexes)
-- Add actor index
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs (actor_id) WHERE actor_id IS NOT NULL;

-- Integration API keys table (per Execution Directive §24)
CREATE TABLE IF NOT EXISTS integration_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    allowed_sources TEXT[] NOT NULL DEFAULT '{}',
    allowed_scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_integration_api_keys_prefix
    ON integration_api_keys (key_prefix) WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_integration_api_keys_team
    ON integration_api_keys (team_id) WHERE revoked_at IS NULL;
