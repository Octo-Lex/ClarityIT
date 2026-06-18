-- Migration 031: Context Relation Reviews (v1.2 Track 6)
-- Sidecar table for operator review state on context graph relations.
-- Advisory only — does not delete nodes or relations.

CREATE TABLE IF NOT EXISTS context_relation_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    relation_id     UUID NOT NULL REFERENCES context_edges(id) ON DELETE CASCADE,
    quality_status  TEXT NOT NULL CHECK (quality_status IN ('confirmed', 'dismissed')),
    reason          TEXT,
    reviewed_by     UUID,
    reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One review per relation (latest wins via upsert)
CREATE UNIQUE INDEX IF NOT EXISTS idx_context_rel_reviews_relation_unique
    ON context_relation_reviews (relation_id);

-- Index for team-scoped lookups
CREATE INDEX IF NOT EXISTS idx_context_rel_reviews_team
    ON context_relation_reviews (team_id);

COMMENT ON TABLE context_relation_reviews IS 'Operator review state for context graph relations (v1.2 Track 6 — advisory only)';
