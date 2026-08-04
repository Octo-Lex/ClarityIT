-- 004_context_graph.sql
CREATE TABLE IF NOT EXISTS context_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    source TEXT NOT NULL,
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- embedding vector(1536), -- enable after pgvector extension is available
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (team_id, entity_type, entity_id),
    CHECK (jsonb_typeof(properties) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_context_nodes_type ON context_nodes (team_id, entity_type);

CREATE TRIGGER trg_context_nodes_updated_at
BEFORE UPDATE ON context_nodes
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS context_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    from_node_id UUID NOT NULL REFERENCES context_nodes(id) ON DELETE CASCADE,
    to_node_id UUID NOT NULL REFERENCES context_nodes(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    weight NUMERIC NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    CHECK (from_node_id <> to_node_id),
    CHECK (weight >= 0)
);

CREATE INDEX IF NOT EXISTS idx_context_edges_from ON context_edges (from_node_id, relation_type);
CREATE INDEX IF NOT EXISTS idx_context_edges_to ON context_edges (to_node_id, relation_type);

CREATE TABLE IF NOT EXISTS context_edge_evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    edge_id UUID NOT NULL REFERENCES context_edges(id) ON DELETE CASCADE,
    evidence_event_id UUID NOT NULL,
    evidence_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    subject_type TEXT NOT NULL,
    subject_id UUID NOT NULL,
    requested_by_actor_id UUID,
    target_type TEXT NOT NULL CHECK (target_type IN ('user', 'agent', 'view')),
    dimensions TEXT[] NOT NULL DEFAULT '{}',
    bundle_json JSONB NOT NULL,
    freshness TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    CHECK (jsonb_typeof(bundle_json) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_context_bundles_subject ON context_bundles (team_id, subject_type, subject_id, freshness DESC);
