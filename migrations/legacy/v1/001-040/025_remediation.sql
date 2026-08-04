-- Migration 025: Operator-Controlled Remediation
-- Agents propose, operators decide. Draft-first remediation with step-level execution.

-- Remediation proposals
CREATE TABLE IF NOT EXISTS remediation_proposals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','proposed','approved','executing','completed','failed','cancelled')),
    risk_level      TEXT NOT NULL DEFAULT 'low'
                    CHECK (risk_level IN ('low','medium','high','critical')),
    source          TEXT NOT NULL DEFAULT 'operator'
                    CHECK (source IN ('agent','operator')),
    incident_id     UUID,
    agent_run_id    UUID,
    created_by      UUID NOT NULL REFERENCES users(id),
    approved_by     UUID REFERENCES users(id),
    approval_id     UUID REFERENCES approval_requests(id),
    idempotency_key TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at     TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_remediation_proposals_team ON remediation_proposals(team_id);
CREATE INDEX idx_remediation_proposals_status ON remediation_proposals(status);
CREATE INDEX idx_remediation_proposals_incident ON remediation_proposals(incident_id);
CREATE INDEX idx_remediation_proposals_agent_run ON remediation_proposals(agent_run_id);

-- Remediation steps
CREATE TABLE IF NOT EXISTS remediation_steps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id         UUID NOT NULL REFERENCES remediation_proposals(id) ON DELETE CASCADE,
    team_id             UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    step_order          INT NOT NULL DEFAULT 0,
    tool_name           TEXT NOT NULL,
    risk_level          TEXT NOT NULL DEFAULT 'low'
                        CHECK (risk_level IN ('low','medium','high','critical')),
    parameters          JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(parameters) = 'object'),
    approval_id         UUID REFERENCES approval_requests(id),
    effect_result_id    UUID REFERENCES agent_effect_results(id),
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','executing','succeeded','failed','skipped')),
    continue_on_failure BOOLEAN NOT NULL DEFAULT false,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ
);

CREATE INDEX idx_remediation_steps_proposal ON remediation_steps(proposal_id, step_order);
CREATE INDEX idx_remediation_steps_status ON remediation_steps(status);

-- Permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('remediations.create', 'Create remediation proposals', 'remediations', 'create', 'low'),
    ('remediations.read', 'View remediation proposals', 'remediations', 'read', 'low'),
    ('remediations.approve', 'Approve remediation proposals', 'remediations', 'approve', 'medium'),
    ('remediations.execute', 'Execute approved remediation proposals', 'remediations', 'execute', 'high'),
    ('remediations.cancel', 'Cancel remediation proposals', 'remediations', 'cancel', 'low')
ON CONFLICT (name) DO NOTHING;

-- Assign to roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('remediations.create', 'remediations.read', 'remediations.cancel')
  AND r.name IN ('owner', 'admin', 'manager', 'on_call_engineer', 'infrastructure_engineer', 'security_admin')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('remediations.approve')
  AND r.name IN ('owner', 'admin', 'manager', 'security_admin')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('remediations.execute')
  AND r.name IN ('owner', 'admin', 'infrastructure_engineer', 'security_admin')
ON CONFLICT DO NOTHING;
