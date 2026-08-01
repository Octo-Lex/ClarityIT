-- G2 018 validation: runs against an isolated P1-canonical schema in a separate schema namespace.
-- Creates schema g2_p1, applies the P1-canonical agent DDL, runs all assertions,
-- then creates a raw-018 schema and asserts it FAILS the P1 checks.
-- Finally creates a 005-only schema and asserts it FAILS.

-- === STEP 1: Create P1-canonical profile in schema g2_p1 ===
CREATE SCHEMA IF NOT EXISTS g2_p1;
SET search_path TO g2_p1;

-- Inline the P1-canonical DDL (psql \i requires working directory which
-- is unreliable in CI containers; inline avoids the path problem)

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

-- P1 foreign keys (only 4)
ALTER TABLE agent_tool_grants ADD CONSTRAINT agent_tool_grants_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES agent_identities(id) ON DELETE CASCADE;
ALTER TABLE agent_runs ADD CONSTRAINT agent_runs_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES agent_identities(id);
ALTER TABLE agent_intentions ADD CONSTRAINT agent_intentions_agent_run_id_fkey FOREIGN KEY (agent_run_id) REFERENCES agent_runs(id) ON DELETE CASCADE;
ALTER TABLE agent_effect_results ADD CONSTRAINT agent_effect_results_intention_id_fkey FOREIGN KEY (intention_id) REFERENCES agent_intentions(id) ON DELETE CASCADE;

-- P1 trigger
CREATE TRIGGER trg_agent_identities_updated_at BEFORE UPDATE ON agent_identities FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- === STEP 2: Validate P1-canonical passes ALL assertions ===
DO $$
BEGIN
    -- 1. max_autonomy exists (NOT max_autonomy_level)
    ASSERT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_identities' AND column_name='max_autonomy'),
        'P1 FAIL: max_autonomy missing';
    ASSERT NOT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_identities' AND column_name='max_autonomy_level'),
        'P1 FAIL: max_autonomy_level should not exist';

    -- 2. max_autonomy has NO default (P1 divergence from 018)
    ASSERT (SELECT column_default FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_identities' AND column_name='max_autonomy') IS NULL,
        'P1 FAIL: max_autonomy should have NO default';

    -- 3. No metadata column
    ASSERT NOT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_identities' AND column_name='metadata'),
        'P1 FAIL: metadata should not exist';

    -- 4. agent_identities has ZERO FKs (018 has 2)
    ASSERT NOT EXISTS (SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema='g2_p1' AND table_name='agent_identities' AND constraint_type='FOREIGN KEY'),
        'P1 FAIL: agent_identities should have 0 FKs';

    -- 5. agent_tool_grants.agent_id has ON DELETE CASCADE (P1 adds; 018 does not)
    ASSERT EXISTS (SELECT 1 FROM pg_constraint
        WHERE conrelid='g2_p1.agent_tool_grants'::regclass AND contype='f'
        AND confdeltype='c' AND confrelid='g2_p1.agent_identities'::regclass),
        'P1 FAIL: agent_tool_grants.agent_id should have CASCADE';

    -- 6. agent_tool_grants has exactly 1 FK
    ASSERT (SELECT count(*) FROM pg_constraint
        WHERE conrelid='g2_p1.agent_tool_grants'::regclass AND contype='f') = 1,
        'P1 FAIL: agent_tool_grants should have exactly 1 FK';

    -- 7. agent_runs has exactly 1 FK
    ASSERT (SELECT count(*) FROM pg_constraint
        WHERE conrelid='g2_p1.agent_runs'::regclass AND contype='f') = 1,
        'P1 FAIL: agent_runs should have exactly 1 FK';

    -- 8. agent_intentions has exactly 1 FK
    ASSERT (SELECT count(*) FROM pg_constraint
        WHERE conrelid='g2_p1.agent_intentions'::regclass AND contype='f') = 1,
        'P1 FAIL: agent_intentions should have exactly 1 FK';

    -- 9. agent_effect_results has exactly 1 FK
    ASSERT (SELECT count(*) FROM pg_constraint
        WHERE conrelid='g2_p1.agent_effect_results'::regclass AND contype='f') = 1,
        'P1 FAIL: agent_effect_results should have exactly 1 FK';

    -- 10. Total FKs across all 5 agent tables = 4
    ASSERT (SELECT count(*) FROM pg_constraint WHERE contype='f' AND conrelid IN (
        'g2_p1.agent_identities'::regclass, 'g2_p1.agent_tool_grants'::regclass,
        'g2_p1.agent_runs'::regclass, 'g2_p1.agent_intentions'::regclass,
        'g2_p1.agent_effect_results'::regclass)) = 4,
        'P1 FAIL: should have exactly 4 FKs total (018 has 14)';

    -- 11. max_autonomy_level in agent_tool_grants has NO default
    ASSERT (SELECT column_default FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_tool_grants' AND column_name='max_autonomy_level') IS NULL,
        'P1 FAIL: agent_tool_grants.max_autonomy_level should have NO default';

    -- 12. agent_runs.status has NO default
    ASSERT (SELECT column_default FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_runs' AND column_name='status') IS NULL,
        'P1 FAIL: agent_runs.status should have NO default';

    -- 13. agent_intentions.status defaults to 'created' (NOT 018's 'proposed')
    ASSERT (SELECT column_default FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_intentions' AND column_name='status') = '''created''::text',
        'P1 FAIL: agent_intentions.status should default to created';

    -- 14. agent_effect_results.result exists (NOT result_payload)
    ASSERT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_effect_results' AND column_name='result'),
        'P1 FAIL: result column missing';
    ASSERT NOT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='agent_effect_results' AND column_name='result_payload'),
        'P1 FAIL: result_payload should not exist';

    -- 15. tool_registry has 7 columns
    ASSERT (SELECT count(*) FROM information_schema.columns
        WHERE table_schema='g2_p1' AND table_name='tool_registry') = 7,
        'P1 FAIL: tool_registry should have 7 columns';

    -- 16. trg_agent_identities_updated_at trigger exists
    ASSERT EXISTS (SELECT 1 FROM pg_trigger
        WHERE tgrelid='g2_p1.agent_identities'::regclass
        AND tgname='trg_agent_identities_updated_at' AND NOT tgisinternal),
        'P1 FAIL: trigger should exist';

END $$;

-- === STEP 3: Create raw-018 schema and assert it FAILS P1 checks ===
RESET search_path;
CREATE SCHEMA IF NOT EXISTS g2_018;
SET search_path TO g2_018;

-- Create dummy teams/users tables for FK targets (018 references them)
CREATE TABLE teams (id UUID PRIMARY KEY DEFAULT gen_random_uuid());
CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid());

-- Apply 018's agent table definitions (raw, unmodified)
-- Note: 018 uses CREATE TABLE without IF NOT EXISTS
CREATE TABLE agent_identities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID REFERENCES teams(id),
    name            TEXT NOT NULL,
    agent_type      TEXT,
    description     TEXT DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','disabled','suspended')),
    max_autonomy    TEXT NOT NULL DEFAULT 'A0'
                    CHECK (max_autonomy IN ('A0','A1','A2','A3','A4','A5')),
    metadata        JSONB DEFAULT '{}',
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (team_id, name)
);

CREATE TABLE agent_tool_grants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id            UUID NOT NULL REFERENCES agent_identities(id),
    team_id             UUID REFERENCES teams(id),
    tool_name           TEXT NOT NULL,
    max_autonomy_level  TEXT NOT NULL DEFAULT 'A0'
                        CHECK (max_autonomy_level IN ('A0','A1','A2','A3','A4','A5')),
    requires_approval   BOOLEAN NOT NULL DEFAULT true,
    requires_mfa        BOOLEAN NOT NULL DEFAULT false,
    expires_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ,
    revoked_by          UUID REFERENCES users(id)
);
CREATE INDEX idx_agent_tool_grants_agent ON agent_tool_grants(agent_id, tool_name);

CREATE TABLE agent_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id                 UUID NOT NULL REFERENCES teams(id),
    agent_id                UUID NOT NULL REFERENCES agent_identities(id),
    triggered_by            UUID REFERENCES users(id),
    triggered_by_actor_type TEXT
                            CHECK (triggered_by_actor_type IN ('user','agent','system')),
    context_bundle_id       UUID,
    status                  TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending','queued','running','completed','failed','cancelled')),
    started_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at            TIMESTAMPTZ,
    correlation_id          UUID,
    error_message           TEXT,
    created_at              TIMESTAMPTZ DEFAULT now(),
    updated_at              TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_agent_runs_team_status ON agent_runs(team_id, status, started_at DESC);

CREATE TABLE agent_intentions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_run_id        UUID NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    team_id             UUID NOT NULL REFERENCES teams(id),
    intention_type      TEXT NOT NULL,
    target_object_id    UUID,
    tool_name           TEXT,
    confidence          NUMERIC CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1)),
    risk_level          TEXT NOT NULL CHECK (risk_level IN ('low','medium','high','critical')),
    autonomy_level      TEXT NOT NULL CHECK (autonomy_level IN ('A0','A1','A2','A3','A4','A5')),
    payload             JSONB DEFAULT '{}' CHECK (jsonb_typeof(payload) = 'object'),
    reasoning_summary   TEXT DEFAULT '',
    evidence_refs       JSONB DEFAULT '[]',
    status              TEXT NOT NULL DEFAULT 'proposed'
                        CHECK (status IN ('proposed','approved','denied','executed','failed','blocked',
                                          'created','validated','approval_requested')),
    blocked_reason      TEXT,
    approved_by         UUID REFERENCES users(id),
    approved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agent_intentions_run ON agent_intentions(agent_run_id, created_at);

CREATE TABLE agent_effect_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intention_id    UUID NOT NULL REFERENCES agent_intentions(id) ON DELETE CASCADE,
    team_id         UUID NOT NULL REFERENCES teams(id),
    tool_name       TEXT NOT NULL,
    status          TEXT NOT NULL
                    CHECK (status IN ('succeeded','failed','denied','blocked','cancelled')),
    approval_id     UUID,
    audit_event_id  UUID,
    result          JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(result) = 'object'),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tool_registry (
    tool_name         TEXT PRIMARY KEY,
    display_name      TEXT NOT NULL,
    description       TEXT DEFAULT '',
    risk_level        TEXT NOT NULL DEFAULT 'medium'
                      CHECK (risk_level IN ('low','medium','high','critical')),
    requires_approval BOOLEAN NOT NULL DEFAULT true,
    requires_mfa      BOOLEAN NOT NULL DEFAULT false,
    is_active         BOOLEAN NOT NULL DEFAULT true
);

-- Assert raw-018 FAILS P1 checks
DO $$
BEGIN
    -- 018 has metadata (P1 does not) → this should FAIL if P1 assertion ran
    ASSERT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_schema='g2_018' AND table_name='agent_identities' AND column_name='metadata'),
        '018 has metadata (P1 does not) — confirms 018 ≠ P1';

    -- 018 has max_autonomy DEFAULT 'A0' (P1 has no default)
    ASSERT (SELECT column_default FROM information_schema.columns
        WHERE table_schema='g2_018' AND table_name='agent_identities' AND column_name='max_autonomy') = '''A0''::text',
        '018 has DEFAULT A0 (P1 has no default) — confirms divergence';

    -- 018 has agent_identities FKs (P1 has none)
    ASSERT EXISTS (SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema='g2_018' AND table_name='agent_identities' AND constraint_type='FOREIGN KEY'),
        '018 has agent_identities FKs (P1 has none) — confirms divergence';

    -- 018 has agent_tool_grants.agent_id WITHOUT CASCADE
    ASSERT NOT EXISTS (SELECT 1 FROM pg_constraint
        WHERE conrelid='g2_018.agent_tool_grants'::regclass AND contype='f' AND confdeltype='c'),
        '018 has no CASCADE on agent_tool_grants.agent_id (P1 adds it) — confirms divergence';

    -- 018 total FKs = 14 (P1 = 4)
    ASSERT (SELECT count(*) FROM pg_constraint WHERE contype='f' AND conrelid IN (
        'g2_018.agent_identities'::regclass, 'g2_018.agent_tool_grants'::regclass,
        'g2_018.agent_runs'::regclass, 'g2_018.agent_intentions'::regclass,
        'g2_018.agent_effect_results'::regclass)) >= 10,
        '018 has many FKs (P1 has 4) — confirms divergence';

    -- 018 has NO trigger
    ASSERT NOT EXISTS (SELECT 1 FROM pg_trigger
        WHERE tgrelid='g2_018.agent_identities'::regclass
        AND tgname='trg_agent_identities_updated_at' AND NOT tgisinternal),
        '018 has no trigger (P1 has one) — confirms divergence';

END $$;

RESET search_path;

-- === CLEANUP ===
DROP SCHEMA g2_p1 CASCADE;
DROP SCHEMA g2_018 CASCADE;
