-- P1-canonical agent schema (generated from P1 production manifest)
-- This is the authoritative production shape for the 5 agent tables + tool_registry.
-- Used as the reference profile for 018 fixture validation.

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TABLE agent_identities (
    agent_type text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    deleted_at timestamp with time zone,
    description text DEFAULT ''::text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    max_autonomy text NOT NULL,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'active'::text,
    team_id uuid,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT agent_identities_pkey PRIMARY KEY (id),
    CONSTRAINT agent_identities_team_id_name_key UNIQUE (team_id, name),
    CONSTRAINT agent_identities_max_autonomy_level_check CHECK (max_autonomy = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text])),
    CONSTRAINT agent_identities_status_check CHECK (status = ANY (ARRAY['active'::text, 'disabled'::text, 'suspended'::text]))
);

CREATE TABLE agent_tool_grants (
    agent_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    expires_at timestamp with time zone,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    max_autonomy_level text NOT NULL,
    requires_approval boolean NOT NULL DEFAULT true,
    requires_mfa boolean NOT NULL DEFAULT false,
    revoked_at timestamp with time zone,
    revoked_by uuid,
    team_id uuid,
    tool_name text NOT NULL,
    CONSTRAINT agent_tool_grants_pkey PRIMARY KEY (id),
    CONSTRAINT agent_tool_grants_max_autonomy_level_check CHECK (max_autonomy_level = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text]))
);

CREATE TABLE agent_runs (
    agent_id uuid NOT NULL,
    completed_at timestamp with time zone,
    context_bundle_id uuid,
    correlation_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    error_message text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    started_at timestamp with time zone NOT NULL DEFAULT now(),
    status text NOT NULL,
    team_id uuid NOT NULL,
    triggered_by uuid,
    triggered_by_actor_type text,
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT agent_runs_pkey PRIMARY KEY (id),
    CONSTRAINT agent_runs_status_check CHECK (status = ANY (ARRAY['pending'::text, 'queued'::text, 'running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])),
    CONSTRAINT agent_runs_triggered_by_actor_type_check CHECK (triggered_by_actor_type = ANY (ARRAY['user'::text, 'agent'::text, 'system'::text]))
);

CREATE TABLE agent_intentions (
    agent_run_id uuid NOT NULL,
    approved_at timestamp with time zone,
    approved_by uuid,
    autonomy_level text NOT NULL,
    blocked_reason text,
    confidence numeric,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    evidence_refs jsonb DEFAULT '[]'::jsonb,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    intention_type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb,
    reasoning_summary text DEFAULT ''::text,
    risk_level text NOT NULL,
    status text NOT NULL DEFAULT 'created'::text,
    target_object_id uuid,
    team_id uuid NOT NULL,
    tool_name text,
    CONSTRAINT agent_intentions_pkey PRIMARY KEY (id),
    CONSTRAINT agent_intentions_autonomy_level_check CHECK (autonomy_level = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text])),
    CONSTRAINT agent_intentions_confidence_check CHECK (confidence IS NULL OR confidence >= 0::numeric AND confidence <= 1::numeric),
    CONSTRAINT agent_intentions_payload_check CHECK (jsonb_typeof(payload) = 'object'::text),
    CONSTRAINT agent_intentions_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])),
    CONSTRAINT agent_intentions_status_check CHECK (status = ANY (ARRAY['proposed'::text, 'approved'::text, 'denied'::text, 'executed'::text, 'failed'::text, 'blocked'::text, 'created'::text, 'validated'::text, 'approval_requested'::text]))
);

CREATE TABLE agent_effect_results (
    approval_id uuid,
    audit_event_id uuid,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    intention_id uuid NOT NULL,
    result jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL,
    team_id uuid NOT NULL,
    tool_name text NOT NULL,
    CONSTRAINT agent_effect_results_pkey PRIMARY KEY (id),
    CONSTRAINT agent_effect_results_result_payload_check CHECK (jsonb_typeof(result) = 'object'::text),
    CONSTRAINT agent_effect_results_status_check CHECK (status = ANY (ARRAY['succeeded'::text, 'failed'::text, 'denied'::text, 'blocked'::text, 'cancelled'::text]))
);

CREATE TABLE tool_registry (
    description text DEFAULT ''::text,
    display_name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    requires_approval boolean NOT NULL DEFAULT true,
    requires_mfa boolean NOT NULL DEFAULT false,
    risk_level text NOT NULL DEFAULT 'medium'::text,
    tool_name text NOT NULL,
    CONSTRAINT tool_registry_pkey PRIMARY KEY (tool_name),
    CONSTRAINT tool_registry_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))
);

-- P1 foreign keys (only 4 — 10 FKs from 018 are absent)
ALTER TABLE agent_tool_grants ADD CONSTRAINT agent_tool_grants_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES agent_identities(id) ON DELETE CASCADE;
ALTER TABLE agent_runs ADD CONSTRAINT agent_runs_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES agent_identities(id);
ALTER TABLE agent_intentions ADD CONSTRAINT agent_intentions_agent_run_id_fkey FOREIGN KEY (agent_run_id) REFERENCES agent_runs(id) ON DELETE CASCADE;
ALTER TABLE agent_effect_results ADD CONSTRAINT agent_effect_results_intention_id_fkey FOREIGN KEY (intention_id) REFERENCES agent_intentions(id) ON DELETE CASCADE;

-- P1 trigger (absent from 018)
CREATE TRIGGER trg_agent_identities_updated_at BEFORE UPDATE ON agent_identities FOR EACH ROW EXECUTE FUNCTION set_updated_at();
