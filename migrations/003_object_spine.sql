-- 003_object_spine.sql
CREATE TABLE IF NOT EXISTS objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    object_type TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT,
    status TEXT NOT NULL,
    priority TEXT CHECK (priority IN ('critical', 'high', 'medium', 'low', 'none')),
    owner_user_id UUID,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    version INT NOT NULL DEFAULT 1,
    CHECK (TRIM(title) <> '')
);

CREATE INDEX IF NOT EXISTS idx_objects_team_type ON objects (team_id, object_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_objects_status ON objects (team_id, status) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_objects_updated_at
BEFORE UPDATE ON objects
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS object_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    from_object_id UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    to_object_id UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (from_object_id <> to_object_id)
);

CREATE INDEX IF NOT EXISTS idx_object_links_from ON object_links (from_object_id, relation_type);
CREATE INDEX IF NOT EXISTS idx_object_links_to ON object_links (to_object_id, relation_type);

CREATE TABLE IF NOT EXISTS object_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    object_id UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    author_id UUID NOT NULL,
    body_markdown TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    CHECK (TRIM(body_markdown) <> '')
);

CREATE INDEX IF NOT EXISTS idx_object_comments_object ON object_comments (object_id, created_at);

CREATE TABLE IF NOT EXISTS work_items (
    object_id UUID PRIMARY KEY REFERENCES objects(id) ON DELETE CASCADE,
    work_item_type TEXT NOT NULL CHECK (work_item_type IN ('task', 'ticket', 'incident', 'change', 'problem', 'project_task', 'alert_work_item')),
    due_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    sla_policy_id UUID,
    assignee_user_id UUID,
    queue_id UUID,
    project_id UUID
);

CREATE TABLE IF NOT EXISTS incidents (
    object_id UUID PRIMARY KEY REFERENCES objects(id) ON DELETE CASCADE,
    severity TEXT NOT NULL CHECK (severity IN ('sev1', 'sev2', 'sev3', 'sev4', 'sev5')),
    impact TEXT,
    affected_service_id UUID,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    commander_user_id UUID
);

CREATE TABLE IF NOT EXISTS alerts (
    object_id UUID PRIMARY KEY REFERENCES objects(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    source_alert_id TEXT,
    severity TEXT,
    fingerprint TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_alert_source_fingerprint ON alerts (source, fingerprint);

CREATE TABLE IF NOT EXISTS docs (
    object_id UUID PRIMARY KEY REFERENCES objects(id) ON DELETE CASCADE,
    collection_id UUID,
    doc_type TEXT NOT NULL CHECK (doc_type IN ('doc', 'runbook', 'postmortem', 'rfc', 'template')),
    git_path TEXT,
    current_version_id UUID
);

CREATE TABLE IF NOT EXISTS assets (
    object_id UUID PRIMARY KEY REFERENCES objects(id) ON DELETE CASCADE,
    asset_type TEXT NOT NULL,
    provider TEXT,
    external_id TEXT,
    hostname TEXT,
    service_id UUID
);
