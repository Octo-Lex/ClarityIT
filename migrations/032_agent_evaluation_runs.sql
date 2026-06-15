-- Migration 032: Agent Recommendation Evaluation Runs
-- v1.2.0 Track 7 — Agent Recommendation Evaluation Harness
-- Controlled golden-scenario evaluation only. No live operations.

CREATE TABLE IF NOT EXISTS agent_evaluation_runs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id             UUID REFERENCES teams(id) ON DELETE SET NULL,
    run_status          TEXT NOT NULL DEFAULT 'completed'
                        CHECK (run_status IN ('running', 'completed', 'failed')),
    scenario_count      INTEGER NOT NULL DEFAULT 0,
    passed_count        INTEGER NOT NULL DEFAULT 0,
    failed_count        INTEGER NOT NULL DEFAULT 0,
    average_score       DOUBLE PRECISION NOT NULL DEFAULT 0.0
                        CHECK (average_score >= 0.0 AND average_score <= 1.0),
    safety_score        DOUBLE PRECISION NOT NULL DEFAULT 0.0
                        CHECK (safety_score >= 0.0 AND safety_score <= 1.0),
    explainability_score DOUBLE PRECISION NOT NULL DEFAULT 0.0
                        CHECK (explainability_score >= 0.0 AND explainability_score <= 1.0),
    correctness_score   DOUBLE PRECISION NOT NULL DEFAULT 0.0
                        CHECK (correctness_score >= 0.0 AND correctness_score <= 1.0),
    quality_score       DOUBLE PRECISION NOT NULL DEFAULT 0.0
                        CHECK (quality_score >= 0.0 AND quality_score <= 1.0),
    result_summary      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by          UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS agent_evaluation_scenario_results (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id               UUID NOT NULL REFERENCES agent_evaluation_runs(id) ON DELETE CASCADE,
    scenario_id          TEXT NOT NULL,
    scenario_name        TEXT NOT NULL,
    passed               BOOLEAN NOT NULL DEFAULT false,
    score                DOUBLE PRECISION NOT NULL DEFAULT 0.0
                         CHECK (score >= 0.0 AND score <= 1.0),
    correctness_score    DOUBLE PRECISION NOT NULL DEFAULT 0.0
                         CHECK (correctness_score >= 0.0 AND correctness_score <= 1.0),
    safety_score         DOUBLE PRECISION NOT NULL DEFAULT 0.0
                         CHECK (safety_score >= 0.0 AND safety_score <= 1.0),
    explainability_score DOUBLE PRECISION NOT NULL DEFAULT 0.0
                         CHECK (explainability_score >= 0.0 AND explainability_score <= 1.0),
    quality_score        DOUBLE PRECISION NOT NULL DEFAULT 0.0
                         CHECK (quality_score >= 0.0 AND quality_score <= 1.0),
    expected_criteria    JSONB NOT NULL DEFAULT '{}'::jsonb,
    actual_recommendation JSONB NOT NULL DEFAULT '{}'::jsonb,
    failure_reasons      JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_eval_scenario_results_run_id
    ON agent_evaluation_scenario_results(run_id);

CREATE INDEX IF NOT EXISTS idx_eval_runs_created_at
    ON agent_evaluation_runs(created_at DESC);

COMMENT ON TABLE agent_evaluation_runs IS
    'v1.2 Track 7: Agent recommendation evaluation runs. Controlled golden scenarios only.';
COMMENT ON TABLE agent_evaluation_scenario_results IS
    'v1.2 Track 7: Per-scenario results for evaluation runs. No live operational data.';
