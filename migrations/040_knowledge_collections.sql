-- 040_knowledge_collections.sql
-- v1.5.0 Track 6 — Knowledge Collections and Saved Findings
-- Storage and organization only. No public sharing, external links, publishing.

-- ─── Tables ───

CREATE TABLE IF NOT EXISTS knowledge_collections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS knowledge_collection_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_id    UUID NOT NULL REFERENCES knowledge_collections(id) ON DELETE CASCADE,
    team_id          UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    source_type      TEXT NOT NULL,
    source_id        TEXT NOT NULL,
    knowledge_item_id UUID NULL REFERENCES knowledge_items(id) ON DELETE SET NULL,
    note             TEXT,
    added_by         UUID REFERENCES users(id),
    added_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saved_knowledge_answers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id       UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    collection_id UUID NULL REFERENCES knowledge_collections(id) ON DELETE SET NULL,
    question      TEXT NOT NULL,
    answer        TEXT NOT NULL,
    confidence    TEXT NOT NULL,
    sources       JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by    UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Constraints ───

-- Collection name must be non-empty
ALTER TABLE knowledge_collections
    ADD CONSTRAINT chk_collection_name CHECK (length(trim(name)) > 0);

-- Collection description bounded to 2000 chars
ALTER TABLE knowledge_collections
    ADD CONSTRAINT chk_collection_desc CHECK (description IS NULL OR length(description) <= 2000);

-- Unique active collection name per team (archived collections excluded)
CREATE UNIQUE INDEX uq_active_collection_name_team
    ON knowledge_collections (team_id, name)
    WHERE archived_at IS NULL;

-- Unique collection item per collection + source_type + source_id
CREATE UNIQUE INDEX uq_collection_item_source
    ON knowledge_collection_items (collection_id, source_type, source_id);

-- Note bounded to 1000 chars
ALTER TABLE knowledge_collection_items
    ADD CONSTRAINT chk_item_note CHECK (note IS NULL OR length(note) <= 1000);

-- Question must be non-empty
ALTER TABLE saved_knowledge_answers
    ADD CONSTRAINT chk_saved_question CHECK (length(trim(question)) > 0);

-- Answer must be non-empty
ALTER TABLE saved_knowledge_answers
    ADD CONSTRAINT chk_saved_answer CHECK (length(trim(answer)) > 0);

-- Confidence must be low|medium|high
ALTER TABLE saved_knowledge_answers
    ADD CONSTRAINT chk_saved_confidence CHECK (confidence IN ('low', 'medium', 'high'));

-- Sources must be valid JSON (array or object)
ALTER TABLE saved_knowledge_answers
    ADD CONSTRAINT chk_saved_sources CHECK (jsonb_typeof(sources) IN ('array', 'object'));

-- ─── Permissions ───

INSERT INTO permissions (name, description, resource, action, risk_level)
VALUES
    ('knowledge.collections.read',   'Read knowledge collections',          'knowledge', 'read',   'low'),
    ('knowledge.collections.create', 'Create knowledge collections',        'knowledge', 'create', 'low'),
    ('knowledge.collections.update', 'Update knowledge collections',        'knowledge', 'update', 'low'),
    ('knowledge.collections.delete', 'Delete knowledge collections',        'knowledge', 'delete', 'low')
ON CONFLICT (name) DO NOTHING;

-- ─── Role Assignments ───

-- All team roles can read collections
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE p.name = 'knowledge.collections.read'
  AND r.name IN ('owner', 'admin', 'manager', 'member', 'viewer', 'on_call_engineer', 'infrastructure_engineer', 'security_admin', 'auditor')
ON CONFLICT DO NOTHING;

-- Owner, admin, manager, member can create collections
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE p.name = 'knowledge.collections.create'
  AND r.name IN ('owner', 'admin', 'manager', 'member')
ON CONFLICT DO NOTHING;

-- Owner, admin, manager, member can update collections (incl. add/remove items)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE p.name = 'knowledge.collections.update'
  AND r.name IN ('owner', 'admin', 'manager', 'member')
ON CONFLICT DO NOTHING;

-- Owner, admin, manager can delete/archive collections
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE p.name = 'knowledge.collections.delete'
  AND r.name IN ('owner', 'admin', 'manager')
ON CONFLICT DO NOTHING;
