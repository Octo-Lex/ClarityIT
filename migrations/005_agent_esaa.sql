-- 005_agent_esaa.sql
CREATE TABLE IF NOT EXISTS agent_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID,
    name TEXT NOT NULL,
    agent_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'suspended')),
    max_autonomy_level TEXT NOT NULL CHECK (max_autonomy_level IN ('A0', 'A1', 'A2', 'A3', 'A4', 'A5')),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (team_id, name)
);

CREATE TRIGGER trg_agent_identities_updated_at
BEFORE UPDATE ON agent_identities
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS agent_tool_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agent_identities(id) ON DELETE CASCADE,
    team_id UUID,
    tool_name TEXT NOT NULL,
    max_autonomy_level TEXT NOT NULL CHECK (max_autonomy_level IN ('A0', 'A1', 'A2', 'A3', 'A4', 'A5')),
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    requires_mfa BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_tool_grants_agent ON agent_tool_grants (agent_id, tool_name);

CREATE TABLE IF NOT EXISTS agent_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    agent_id UUID NOT NULL REFERENCES agent_identities(id),
    triggered_by_actor_id UUID,
    triggered_by_actor_type TEXT NOT NULL CHECK (triggered_by_actor_type IN ('user', 'agent', 'system')),
    context_bundle_id UUID,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    correlation_id UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_team_status ON agent_runs (team_id, status, started_at DESC);

CREATE TABLE IF NOT EXISTS agent_intentions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    agent_run_id UUID NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    intention_type TEXT NOT NULL,
    target_object_id UUID,
    tool_name TEXT,
    confidence NUMERIC CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1)),
    risk_level TEXT NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    autonomy_level TEXT NOT NULL CHECK (autonomy_level IN ('A0', 'A1', 'A2', 'A3', 'A4', 'A5')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'created' CHECK (status IN ('created', 'validated', 'denied', 'approval_requested', 'approved', 'executed', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_agent_intentions_run ON agent_intentions (agent_run_id, created_at);

CREATE TABLE IF NOT EXISTS agent_effect_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    intention_id UUID NOT NULL REFERENCES agent_intentions(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('succeeded', 'failed', 'denied', 'cancelled')),
    approval_id UUID,
    audit_event_id UUID,
    result_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (jsonb_typeof(result_payload) = 'object')
);
