-- Phase 8: Integration + Validation — missing permissions
-- Migration 019

BEGIN;

-- ─── Missing integration permissions ───

INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('integrations.keys.create',   'Create integration API keys',       'integrations.keys', 'create', 'medium'),
    ('integrations.keys.read',     'Read integration API keys',         'integrations.keys', 'read',   'low'),
    ('integrations.keys.revoke',   'Revoke integration API keys',       'integrations.keys', 'revoke', 'medium'),
    ('integrations.proxmox.read',  'Read Proxmox integration status',   'integrations.proxmox', 'read', 'low'),
    ('integrations.proxmox.sync',  'Trigger Proxmox inventory sync',    'integrations.proxmox', 'sync', 'medium'),
    ('assets.read',                'Read infrastructure assets',         'assets',           'read',   'low'),
    ('objects.attachments.create', 'Upload object attachments',         'objects.attachments', 'create', 'low'),
    ('objects.attachments.read',   'Read/download object attachments',  'objects.attachments', 'read',   'low')
ON CONFLICT (name) DO NOTHING;

-- ─── Role grants ───

DO $$
DECLARE
    r_owner  UUID; r_admin UUID; r_manager UUID; r_oncall UUID; r_infra UUID; r_auto UUID;
BEGIN
    SELECT id INTO r_owner   FROM roles WHERE name = 'owner';
    SELECT id INTO r_admin   FROM roles WHERE name = 'admin';
    SELECT id INTO r_manager FROM roles WHERE name = 'manager';
    SELECT id INTO r_oncall  FROM roles WHERE name = 'on_call_engineer';
    SELECT id INTO r_infra   FROM roles WHERE name = 'infrastructure_engineer';
    SELECT id INTO r_auto    FROM roles WHERE name = 'automation_operator';

    -- Owner/Admin: full integration access
    FOREACH r_owner IN ARRAY ARRAY[r_owner, r_admin] LOOP
        INSERT INTO role_permissions (role_id, permission_id)
            SELECT r_owner, id FROM permissions WHERE name IN (
                'integrations.keys.create','integrations.keys.read','integrations.keys.revoke',
                'integrations.proxmox.read','integrations.proxmox.sync',
                'assets.read',
                'objects.attachments.create','objects.attachments.read',
                'webhooks.manage','webhooks.ingest')
            ON CONFLICT DO NOTHING;
    END LOOP;

    -- Manager: read + create keys, read assets, attachments
    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_manager, id FROM permissions WHERE name IN (
            'integrations.keys.create','integrations.keys.read',
            'integrations.proxmox.read',
            'assets.read',
            'objects.attachments.create','objects.attachments.read')
        ON CONFLICT DO NOTHING;

    -- On-call / Infra: proxmox read + sync, assets, attachments read
    FOREACH r_oncall IN ARRAY ARRAY[r_oncall, r_infra] LOOP
        INSERT INTO role_permissions (role_id, permission_id)
            SELECT r_oncall, id FROM permissions WHERE name IN (
                'integrations.proxmox.read','integrations.proxmox.sync',
                'assets.read',
                'objects.attachments.read',
                'infra.inventory.read','infra.metrics.read')
            ON CONFLICT DO NOTHING;
    END LOOP;

    -- Automation operator: proxmox read, assets, webhooks
    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_auto, id FROM permissions WHERE name IN (
            'integrations.proxmox.read','integrations.proxmox.sync',
            'assets.read','webhooks.manage',
            'objects.attachments.read')
        ON CONFLICT DO NOTHING;
END $$;

COMMIT;
