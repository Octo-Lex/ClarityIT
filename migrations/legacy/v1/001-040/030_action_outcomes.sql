-- Migration 030: Action Outcomes (v1.2 Track 5)
-- Records expected result, actual result, operator feedback, outcome status,
-- and follow-up recommendation after asset actions and remediation execution.
-- Feedback capture only — no autonomous retry or follow-up execution.

CREATE TABLE IF NOT EXISTS action_outcomes (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id                    UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    asset_action_id            UUID REFERENCES asset_actions(id) ON DELETE CASCADE,
    remediation_proposal_id    UUID,
    expected_result            TEXT,
    actual_result              TEXT,
    operator_feedback          TEXT,
    outcome_status             TEXT NOT NULL CHECK (outcome_status IN ('successful', 'partially_successful', 'failed', 'inconclusive')),
    follow_up_recommendation   TEXT,
    created_by                 UUID,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- At least one of asset_action_id or remediation_proposal_id must be non-null
ALTER TABLE action_outcomes ADD CONSTRAINT action_outcomes_source_check
    CHECK (asset_action_id IS NOT NULL OR remediation_proposal_id IS NOT NULL);

-- One current outcome per asset_action (unique)
CREATE UNIQUE INDEX IF NOT EXISTS idx_action_outcomes_asset_action_unique
    ON action_outcomes (asset_action_id) WHERE asset_action_id IS NOT NULL;

-- One current outcome per remediation_proposal (unique)
CREATE UNIQUE INDEX IF NOT EXISTS idx_action_outcomes_remediation_unique
    ON action_outcomes (remediation_proposal_id) WHERE remediation_proposal_id IS NOT NULL;

-- Index for team-scoped lookups
CREATE INDEX IF NOT EXISTS idx_action_outcomes_team ON action_outcomes (team_id);

-- Trigger for updated_at
CREATE TRIGGER trg_action_outcomes_updated_at
    BEFORE UPDATE ON action_outcomes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE action_outcomes IS 'Post-action outcome tracking for asset actions and remediation proposals (v1.2 Track 5)';
