-- Migration 029: Recommendation Evidence Packs (v1.2 Track 1)
-- Stores structured evidence for AI-generated recommendations.
-- No chain-of-thought, raw tool parameters, or secrets are persisted.

CREATE TABLE IF NOT EXISTS recommendation_evidence (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id               UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    recommendation_id     UUID NOT NULL,
    source_type           TEXT NOT NULL,  -- 'remediation_proposal', 'incident_suggestion', etc.
    source_id             UUID NOT NULL,  -- the actual proposal/incident/object ID
    recommendation_summary TEXT NOT NULL DEFAULT '',
    supporting_evidence   JSONB NOT NULL DEFAULT '[]'::jsonb,
    conflicting_evidence  JSONB NOT NULL DEFAULT '[]'::jsonb,
    confidence_score      DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    confidence_level      TEXT NOT NULL DEFAULT 'low',  -- low, medium, high
    risk_notes            TEXT NOT NULL DEFAULT '',
    missing_info          JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_stale              BOOLEAN NOT NULL DEFAULT false,
    stale_after           TIMESTAMPTZ,    -- when evidence should be considered stale
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Confidence score must be bounded 0.0-1.0
    CONSTRAINT recommendation_evidence_confidence_check CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    -- Confidence level must be valid
    CONSTRAINT recommendation_evidence_level_check CHECK (confidence_level IN ('low', 'medium', 'high')),
    -- Source type must be recognized
    CONSTRAINT recommendation_evidence_source_type_check CHECK (source_type IN ('remediation_proposal', 'incident_suggestion', 'agent_recommendation'))
);

-- Index for team-scoped lookup by recommendation_id
CREATE INDEX idx_recommendation_evidence_team_rec ON recommendation_evidence (team_id, recommendation_id);
-- Index for lookup by source
CREATE INDEX idx_recommendation_evidence_source ON recommendation_evidence (team_id, source_type, source_id);

GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;
