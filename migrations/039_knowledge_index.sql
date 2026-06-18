-- 039_knowledge_index.sql
-- v1.5 Track 1: Knowledge Index Foundation
-- Team-scoped full-text search index over all ClarityIT knowledge sources.
-- Uses PostgreSQL tsvector + GIN — no external search engine.

-- ─── knowledge_items: one row per indexable entity ───
CREATE TABLE IF NOT EXISTS knowledge_items (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id            UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    source_type        TEXT NOT NULL,
    source_id          UUID NOT NULL,
    title              TEXT NOT NULL DEFAULT '',
    summary            TEXT NOT NULL DEFAULT '',
    content_text       TEXT NOT NULL DEFAULT '',
    content_hash       TEXT,
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    search_vector      TSVECTOR,
    visibility         TEXT NOT NULL DEFAULT 'team'
                       CHECK (visibility IN ('team', 'private')),
    indexed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_updated_at  TIMESTAMPTZ,
    stale_after        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One knowledge item per (team, source_type, source_id)
    UNIQUE (team_id, source_type, source_id),

    -- Valid source types
    CONSTRAINT ki_source_type_check CHECK (
        source_type IN (
            'artifact',
            'clarity_document',
            'meeting_summary',
            'status_report',
            'presentation',
            'template',
            'work_item',
            'incident',
            'project',
            'asset',
            'remediation',
            'approval',
            'context_node'
        )
    ),

    -- content_hash must be a hex string if present
    CONSTRAINT ki_content_hash_format CHECK (
        content_hash IS NULL OR content_hash ~ '^[a-f0-9]{64}$'
    ),

    -- metadata must be a JSON object
    CONSTRAINT ki_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

-- GIN index for full-text search (most important index)
CREATE INDEX IF NOT EXISTS idx_ki_search_vector
    ON knowledge_items USING gin (search_vector);

-- Index for team-scoped queries
CREATE INDEX IF NOT EXISTS idx_ki_team
    ON knowledge_items (team_id, source_type, updated_at DESC);

-- Index for stale detection
CREATE INDEX IF NOT EXISTS idx_ki_stale
    ON knowledge_items (team_id, stale_after)
    WHERE stale_after IS NOT NULL;

-- Index for content_hash dedup
CREATE INDEX IF NOT EXISTS idx_ki_content_hash
    ON knowledge_items (team_id, content_hash)
    WHERE content_hash IS NOT NULL;

-- Trigger to auto-update search_vector on insert/update
CREATE OR REPLACE FUNCTION ki_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.summary, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'C');
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ki_search_vector ON knowledge_items;
CREATE TRIGGER trg_ki_search_vector
    BEFORE INSERT OR UPDATE OF title, summary, content_text ON knowledge_items
    FOR EACH ROW EXECUTE FUNCTION ki_search_vector_update();

-- ─── knowledge_chunks: sub-item segments for finer-grained search ───
CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_item_id  UUID NOT NULL REFERENCES knowledge_items(id) ON DELETE CASCADE,
    team_id            UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    chunk_index        INTEGER NOT NULL DEFAULT 0,
    heading            TEXT NOT NULL DEFAULT '',
    content_text       TEXT NOT NULL DEFAULT '',
    content_hash       TEXT,
    token_estimate     INTEGER NOT NULL DEFAULT 0,
    search_vector      TSVECTOR,
    metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One chunk per (knowledge_item_id, chunk_index)
    UNIQUE (knowledge_item_id, chunk_index),

    -- token_estimate must be non-negative
    CONSTRAINT kc_token_estimate_nonneg CHECK (token_estimate >= 0),

    -- metadata must be a JSON object
    CONSTRAINT kc_metadata_object CHECK (jsonb_typeof(metadata) = 'object'),

    -- content_hash must be a hex string if present
    CONSTRAINT kc_content_hash_format CHECK (
        content_hash IS NULL OR content_hash ~ '^[a-f0-9]{64}$'
    )
);

-- GIN index for chunk-level search
CREATE INDEX IF NOT EXISTS idx_kc_search_vector
    ON knowledge_chunks USING gin (search_vector);

-- Index for team-scoped chunk queries
CREATE INDEX IF NOT EXISTS idx_kc_team
    ON knowledge_chunks (team_id, created_at DESC);

-- Index for joining back to knowledge items
CREATE INDEX IF NOT EXISTS idx_kc_item
    ON knowledge_chunks (knowledge_item_id, chunk_index);

-- Trigger to auto-update chunk search_vector
CREATE OR REPLACE FUNCTION kc_search_vector_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.heading, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_kc_search_vector ON knowledge_chunks;
CREATE TRIGGER trg_kc_search_vector
    BEFORE INSERT OR UPDATE OF heading, content_text ON knowledge_chunks
    FOR EACH ROW EXECUTE FUNCTION kc_search_vector_update();

-- ─── Permissions ───
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('knowledge.search', 'Search team knowledge index', 'knowledge', 'read', 'low'),
    ('knowledge.read', 'Read knowledge items and collections', 'knowledge', 'read', 'low'),
    ('knowledge.create', 'Create knowledge collections and save findings', 'knowledge', 'create', 'low'),
    ('knowledge.update', 'Update knowledge collections', 'knowledge', 'update', 'low'),
    ('knowledge.delete', 'Delete knowledge collections', 'knowledge', 'delete', 'low'),
    ('knowledge.ask', 'Ask Clarity Q&A over team knowledge', 'knowledge', 'read', 'low')
ON CONFLICT DO NOTHING;

-- Assign permissions:
-- viewer+: search, read, ask
-- member+: search, read, create, update, ask
-- manager+: all including delete
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('viewer', 'member', 'manager', 'admin', 'owner')
  AND p.name IN ('knowledge.search', 'knowledge.read', 'knowledge.ask')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('member', 'manager', 'admin', 'owner')
  AND p.name IN ('knowledge.create', 'knowledge.update')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('manager', 'admin', 'owner')
  AND p.name = 'knowledge.delete'
ON CONFLICT DO NOTHING;

-- updated_at trigger for knowledge_items
CREATE OR REPLACE FUNCTION ki_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    -- Don't override if search_vector trigger already set it
    IF NEW.updated_at = OLD.updated_at THEN
        NEW.updated_at := NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ki_updated_at ON knowledge_items;
CREATE TRIGGER trg_ki_updated_at
    BEFORE UPDATE ON knowledge_items
    FOR EACH ROW
    EXECUTE FUNCTION ki_set_updated_at();
