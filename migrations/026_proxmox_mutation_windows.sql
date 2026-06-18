-- Migration 026: Proxmox Mutation Change-Window Workflow
-- v1.1.0 Track 2 — Controlled, audited mutation windows

CREATE TABLE IF NOT EXISTS proxmox_mutation_windows (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id       uuid REFERENCES teams(id) ON DELETE CASCADE,
    status        text NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open', 'closed', 'expired')),
    reason        text NOT NULL,
    opened_by     uuid NOT NULL REFERENCES users(id),
    opened_at     timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    closed_by     uuid REFERENCES users(id),
    closed_at     timestamptz,
    close_reason  text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_proxmox_mutation_windows_status ON proxmox_mutation_windows(status);
CREATE INDEX idx_proxmox_mutation_windows_team ON proxmox_mutation_windows(team_id);

-- Trigger for updated_at
CREATE TRIGGER trg_proxmox_mutation_windows_updated_at
    BEFORE UPDATE ON proxmox_mutation_windows
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
