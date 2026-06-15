-- Migration 033: Artifacts (v1.3.0 Track 1 — Internal Document/Report Workspace)
-- Team-scoped work artifacts: documents, reports, presentations, meeting summaries, etc.

CREATE TABLE IF NOT EXISTS artifacts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id           UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    artifact_type     TEXT NOT NULL
                      CHECK (artifact_type IN ('document', 'report', 'presentation',
                                               'meeting_summary', 'status_report',
                                               'decision_memo', 'training_deck')),
    title             TEXT NOT NULL CHECK (length(title) > 0),
    description       TEXT,
    content_markdown  TEXT,
    status            TEXT NOT NULL DEFAULT 'draft'
                      CHECK (status IN ('draft', 'published', 'archived')),
    source_type       TEXT,
    source_data       JSONB NOT NULL DEFAULT '{}'::jsonb,
    storage_object_id UUID REFERENCES storage_objects(id) ON DELETE SET NULL,
    file_format       TEXT
                      CHECK (file_format IS NULL OR file_format IN ('pptx', 'pdf', 'md')),
    created_by        UUID,
    updated_by        UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_artifacts_team_type
    ON artifacts (team_id, artifact_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_team_status
    ON artifacts (team_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_team_title
    ON artifacts USING gin (to_tsvector('english', title));

-- updated_at trigger
CREATE OR REPLACE FUNCTION trg_artifacts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER artifacts_updated_at
    BEFORE UPDATE ON artifacts
    FOR EACH ROW
    EXECUTE FUNCTION trg_artifacts_updated_at();

-- Seed permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('artifacts.create', 'Create team artifacts', 'artifacts', 'create', 'low'),
    ('artifacts.read', 'Read team artifacts', 'artifacts', 'read', 'low'),
    ('artifacts.update', 'Update team artifacts', 'artifacts', 'update', 'low'),
    ('artifacts.delete', 'Archive/delete team artifacts', 'artifacts', 'delete', 'low')
ON CONFLICT DO NOTHING;

-- Assign to roles: member, manager, admin, owner get all artifact permissions
-- viewer gets read only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.resource = 'artifacts'
  AND r.name IN ('member', 'manager', 'admin', 'owner')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.resource = 'artifacts' AND p.action = 'read'
  AND r.name = 'viewer'
ON CONFLICT DO NOTHING;

-- Platform roles also get artifact access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.resource = 'artifacts'
  AND r.name IN ('on_call_engineer', 'infrastructure_engineer', 'security_admin', 'auditor', 'automation_operator')
ON CONFLICT DO NOTHING;

COMMENT ON TABLE artifacts IS
    'v1.3 Track 1: Team-scoped work artifacts. No operational control path.';
