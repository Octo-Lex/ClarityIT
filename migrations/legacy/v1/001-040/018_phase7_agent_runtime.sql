-- Phase 7: Agent Runtime — identities, tool grants, runs, intentions, effect results
-- Migration 018
-- Replaces all prior 018 versions + manual ALTER TABLE patches.
-- A fresh `make migrate:fresh` must produce this exact schema.

BEGIN;

-- ─── Agent identities ───

CREATE TABLE agent_identities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID REFERENCES teams(id),
    name            TEXT NOT NULL,
    agent_type      TEXT,                                    -- nullable: future use (reactive, proactive, etc.)
    description     TEXT DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','disabled','suspended')),
    max_autonomy    TEXT NOT NULL DEFAULT 'A0'
                    CHECK (max_autonomy IN ('A0','A1','A2','A3','A4','A5')),
    metadata        JSONB DEFAULT '{}',
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,                             -- soft delete / disable

    UNIQUE (team_id, name)
);

-- ─── Agent tool grants ───

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

-- ─── Agent runs ───

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

-- ─── Agent intentions ───

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

-- ─── Agent effect results ───

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

-- ─── Tool registry (static seed) ───

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

INSERT INTO tool_registry (tool_name, display_name, description, risk_level, requires_approval, requires_mfa) VALUES
    ('work_items.create',        'Create Work Item',        'Create a new work item',          'medium', true,  false),
    ('work_items.update_status', 'Update Work Item Status',  'Change work item status',         'medium', true,  false),
    ('objects.add_comment',      'Add Comment',             'Add a comment to an object',       'low',    false, false),
    ('incidents.add_timeline',   'Add Timeline Entry',      'Add a timeline entry to incident', 'low',    false, false),
    ('incidents.summarize',      'Summarize Incident',      'Generate a summary draft',         'low',    false, false);

-- ─── Agent permissions ───

INSERT INTO permissions (name, description) VALUES
    ('agents.create',            'Create agent identities'),
    ('agents.read',              'Read agent identities'),
    ('agents.update',            'Update agent identities'),
    ('agents.disable',           'Disable agent identities'),
    ('agents.grants.create',     'Create agent tool grants'),
    ('agents.grants.read',       'Read agent tool grants'),
    ('agents.grants.revoke',     'Revoke agent tool grants'),
    ('agents.runs.create',       'Create agent runs'),
    ('agents.runs.read',         'Read agent runs'),
    ('agents.runs.cancel',       'Cancel agent runs'),
    ('agents.intentions.create', 'Create agent intentions'),
    ('agents.intentions.read',   'Read agent intentions'),
    ('agents.tools.execute',     'Execute tools via agent gateway')
ON CONFLICT (name) DO NOTHING;

-- ─── Role grants ───

DO $$
DECLARE
    r_owner  UUID; r_admin UUID; r_manager UUID; r_auto UUID;
    r_oncall UUID; r_infra UUID; r_seca    UUID;
BEGIN
    SELECT id INTO r_owner   FROM roles WHERE name = 'owner';
    SELECT id INTO r_admin   FROM roles WHERE name = 'admin';
    SELECT id INTO r_manager FROM roles WHERE name = 'manager';
    SELECT id INTO r_auto    FROM roles WHERE name = 'automation_operator';
    SELECT id INTO r_oncall  FROM roles WHERE name = 'on_call_engineer';
    SELECT id INTO r_infra   FROM roles WHERE name = 'infrastructure_engineer';
    SELECT id INTO r_seca    FROM roles WHERE name = 'security_admin';

    -- Owner / Admin / Automation Operator: full agent access
    FOREACH r_owner IN ARRAY ARRAY[r_owner, r_admin, r_auto] LOOP
        INSERT INTO role_permissions (role_id, permission_id)
            SELECT r_owner, id FROM permissions WHERE name LIKE 'agents.%'
            ON CONFLICT DO NOTHING;
    END LOOP;

    -- Manager: read + create/update agents, read grants, runs, intentions, execute
    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_manager, id FROM permissions WHERE name IN (
            'agents.create','agents.read','agents.update',
            'agents.grants.read',
            'agents.runs.create','agents.runs.read','agents.runs.cancel',
            'agents.intentions.read','agents.tools.execute')
        ON CONFLICT DO NOTHING;

    -- On-call / Infra: read + execute
    FOREACH r_oncall IN ARRAY ARRAY[r_oncall, r_infra] LOOP
        INSERT INTO role_permissions (role_id, permission_id)
            SELECT r_oncall, id FROM permissions WHERE name IN (
                'agents.read','agents.runs.read','agents.intentions.read','agents.tools.execute')
            ON CONFLICT DO NOTHING;
    END LOOP;

    -- Security admin: read all
    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_seca, id FROM permissions WHERE name IN (
            'agents.read','agents.grants.read','agents.runs.read','agents.intentions.read')
        ON CONFLICT DO NOTHING;
END $$;

-- ─── Extend idempotency scope for tool-gateway ───

ALTER TABLE idempotency_keys DROP CONSTRAINT IF EXISTS idempotency_keys_scope_type_check;
ALTER TABLE idempotency_keys ADD CONSTRAINT idempotency_keys_scope_type_check
    CHECK (scope_type IN ('user','anonymous','system','agent','tool-gateway'));

COMMIT;
