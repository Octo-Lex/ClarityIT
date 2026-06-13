-- Migration 024: Asset Actions (Proxmox Controlled Mutation)
-- Tracks infrastructure mutation requests through their full lifecycle.

CREATE TABLE IF NOT EXISTS asset_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    asset_id        UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    action_type     TEXT NOT NULL
                    CHECK (action_type IN ('proxmox.start','proxmox.shutdown','proxmox.stop','proxmox.snapshot')),
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','approved','executing','succeeded','failed','cancelled')),
    approval_id     UUID REFERENCES approval_requests(id),
    requested_by    UUID NOT NULL REFERENCES users(id),
    proxmox_task_id TEXT,
    result          JSONB DEFAULT '{}',
    error_message   TEXT,
    snapshot_name   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    executed_at     TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_asset_actions_team ON asset_actions(team_id);
CREATE INDEX idx_asset_actions_status ON asset_actions(status);
CREATE INDEX idx_asset_actions_asset ON asset_actions(asset_id);

-- Permissions for asset actions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('assets.actions.create', 'Create asset action requests', 'assets', 'create_action', 'medium'),
    ('assets.actions.read', 'View asset actions', 'assets', 'read_action', 'low'),
    ('assets.actions.execute', 'Execute approved asset actions', 'assets', 'execute_action', 'high')
ON CONFLICT (name) DO NOTHING;

-- Assign to roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name IN ('assets.actions.create', 'assets.actions.read')
  AND r.name IN ('owner', 'admin', 'manager', 'on_call_engineer', 'infrastructure_engineer', 'security_admin')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
CROSS JOIN permissions p
WHERE p.name = 'assets.actions.execute'
  AND r.name IN ('owner', 'admin', 'infrastructure_engineer', 'security_admin')
ON CONFLICT DO NOTHING;
