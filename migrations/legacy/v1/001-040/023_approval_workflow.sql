-- Migration 023: Approval Workflow Engine
-- Durable approval system for human authorization of high-risk actions

-- Approval policies (team-level rules)
CREATE TABLE IF NOT EXISTS approval_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id             UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    risk_level          TEXT NOT NULL CHECK (risk_level IN ('low','medium','high','critical')),
    requires_mfa        BOOLEAN NOT NULL DEFAULT true,
    requires_approval   BOOLEAN NOT NULL DEFAULT true,
    auto_approve        BOOLEAN NOT NULL DEFAULT false,
    timeout_seconds     INT NOT NULL DEFAULT 3600,
    min_approvers       INT NOT NULL DEFAULT 1,
    allow_self_approve  BOOLEAN NOT NULL DEFAULT false,
    is_default          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(team_id, name)
);

-- Approval requests
CREATE TABLE IF NOT EXISTS approval_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    action_type     TEXT NOT NULL,
    action_target   JSONB NOT NULL DEFAULT '{}',
    risk_level      TEXT NOT NULL CHECK (risk_level IN ('low','medium','high','critical')),
    description     TEXT NOT NULL DEFAULT '',
    requested_by    UUID NOT NULL REFERENCES users(id),
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','approved','rejected','cancelled','expired','executed','failed')),
    policy_id       UUID REFERENCES approval_policies(id),
    expires_at      TIMESTAMPTZ NOT NULL,
    executed_at     TIMESTAMPTZ,
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Approval decisions (immutable — one row per decision event)
CREATE TABLE IF NOT EXISTS approval_decisions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    approval_id     UUID NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    decided_by      UUID NOT NULL REFERENCES users(id),
    decision        TEXT NOT NULL CHECK (decision IN ('approved','rejected')),
    reason          TEXT NOT NULL DEFAULT '',
    mfa_verified    BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(approval_id, decided_by)
);

CREATE INDEX idx_approval_requests_team ON approval_requests(team_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);
CREATE INDEX idx_approval_decisions_approval ON approval_decisions(approval_id);

-- Seed default policies for existing teams
INSERT INTO approval_policies (team_id, name, risk_level, requires_mfa, requires_approval, auto_approve, timeout_seconds, min_approvers, allow_self_approve, is_default)
SELECT id, 'default-low', 'low', false, false, true, 3600, 1, true, true FROM teams
WHERE NOT EXISTS (SELECT 1 FROM approval_policies WHERE team_id = teams.id AND risk_level = 'low')
ON CONFLICT (team_id, name) DO NOTHING;

INSERT INTO approval_policies (team_id, name, risk_level, requires_mfa, requires_approval, auto_approve, timeout_seconds, min_approvers, allow_self_approve, is_default)
SELECT id, 'default-medium', 'medium', false, true, false, 3600, 1, false, true FROM teams
WHERE NOT EXISTS (SELECT 1 FROM approval_policies WHERE team_id = teams.id AND risk_level = 'medium')
ON CONFLICT (team_id, name) DO NOTHING;

INSERT INTO approval_policies (team_id, name, risk_level, requires_mfa, requires_approval, auto_approve, timeout_seconds, min_approvers, allow_self_approve, is_default)
SELECT id, 'default-high', 'high', true, true, false, 1800, 1, false, true FROM teams
WHERE NOT EXISTS (SELECT 1 FROM approval_policies WHERE team_id = teams.id AND risk_level = 'high')
ON CONFLICT (team_id, name) DO NOTHING;

INSERT INTO approval_policies (team_id, name, risk_level, requires_mfa, requires_approval, auto_approve, timeout_seconds, min_approvers, allow_self_approve, is_default)
SELECT id, 'default-critical', 'critical', true, true, false, 900, 2, false, true FROM teams
WHERE NOT EXISTS (SELECT 1 FROM approval_policies WHERE team_id = teams.id AND risk_level = 'critical')
ON CONFLICT (team_id, name) DO NOTHING;

-- Permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('approvals.create', 'Create approval request', 'approvals', 'create', 'low'),
    ('approvals.read', 'View approval requests', 'approvals', 'read', 'low'),
    ('approvals.approve', 'Approve/reject requests', 'approvals', 'approve', 'medium'),
    ('approvals.cancel', 'Cancel own approval requests', 'approvals', 'cancel', 'low')
ON CONFLICT (name) DO NOTHING;

-- Assign to roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('approvals.create', 'approvals.read')
  AND r.name IN ('owner', 'admin', 'manager', 'member', 'on_call_engineer', 'infrastructure_engineer', 'security_admin', 'automation_operator')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('approvals.approve', 'approvals.cancel')
  AND r.name IN ('owner', 'admin', 'manager', 'security_admin')
ON CONFLICT DO NOTHING;
