-- 009_iam_roles_permissions_seed.sql
-- Seed 10 Genesis roles and permissions

-- ─── Platform Roles ───
-- NOTE: platform_roles is created and seeded by 011_iam_platform_roles.sql. The
-- duplicate seed that previously lived here referenced a table that does not
-- exist at step 009 (forward reference), breaking fresh-database migration.
-- Removed; 011 is the single source for platform role seeding.

-- ─── Team Roles ───
INSERT INTO roles (id, name, description, is_system_role) VALUES
    (gen_random_uuid(), 'owner',      'Full team administration. Manage members, settings, all objects.', TRUE),
    (gen_random_uuid(), 'admin',      'Team management. Invite members, change roles, manage all objects.', TRUE),
    (gen_random_uuid(), 'manager',    'Project and SLA management. Create/edit work items, manage projects.', TRUE),
    (gen_random_uuid(), 'member',     'Standard member. Create and edit own work items, comment, track time.', TRUE),
    (gen_random_uuid(), 'viewer',     'Read-only access to team objects.', TRUE),
    (gen_random_uuid(), 'on_call_engineer',        'Incident response and runbook access. Can acknowledge and update incidents.', TRUE),
    (gen_random_uuid(), 'infrastructure_engineer',  'Infrastructure read access, asset management. Proxmox inventory read.', TRUE),
    (gen_random_uuid(), 'security_admin',           'Security policies, audit access, SoD configuration.', TRUE),
    (gen_random_uuid(), 'auditor',                  'Full audit read access. No mutation capabilities.', TRUE),
    (gen_random_uuid(), 'automation_operator',      'Agent management, API key management, integration configuration.', TRUE)
ON CONFLICT (name) DO NOTHING;

-- ─── Permissions ───
-- Using resource.action naming convention

-- System / Platform permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('system.users.list',       'List all users',                'system.users',    'list',       'low'),
    ('system.users.manage',     'Create, update, deactivate users', 'system.users', 'manage',     'high'),
    ('system.users.delete',     'Soft-delete users',             'system.users',    'delete',     'critical'),
    ('system.teams.list',       'List all teams',                'system.teams',    'list',       'low'),
    ('system.teams.create',     'Create new teams',              'system.teams',    'create',     'high'),
    ('system.teams.delete',     'Soft-delete teams',             'system.teams',    'delete',     'critical'),
    ('system.settings',         'Manage platform settings',      'system.settings', 'manage',     'high'),
    ('system.audit.read',       'Read system-wide audit log',    'system.audit',    'read',       'low'),
    ('system.roles.manage',     'Manage roles and permissions',  'system.roles',    'manage',     'critical')
ON CONFLICT (name) DO NOTHING;

-- Team management permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('team.settings',           'Manage team settings',          'team',            'settings',   'medium'),
    ('team.members.list',       'List team members',             'team.members',    'list',       'low'),
    ('team.members.invite',     'Invite new members',            'team.members',    'invite',     'medium'),
    ('team.members.role',       'Change member roles',           'team.members',    'role',       'high'),
    ('team.members.remove',     'Remove members from team',      'team.members',    'remove',     'high'),
    ('team.audit.read',         'Read team audit log',           'team.audit',      'read',       'low')
ON CONFLICT (name) DO NOTHING;

-- Work item permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('work.items.view',         'View work items',               'work.items',      'view',       'low'),
    ('work.items.create',       'Create work items',             'work.items',      'create',     'low'),
    ('work.items.edit.own',     'Edit own work items',           'work.items',      'edit.own',   'low'),
    ('work.items.edit.any',     'Edit any work item',            'work.items',      'edit.any',   'medium'),
    ('work.items.delete.own',   'Delete own work items',         'work.items',      'delete.own', 'medium'),
    ('work.items.delete.any',   'Delete any work item',          'work.items',      'delete.any', 'high'),
    ('work.items.assign',       'Assign work items',             'work.items',      'assign',     'medium'),
    ('work.items.comment',      'Comment on work items',         'work.items',      'comment',    'low'),
    ('work.items.bulk',         'Bulk update work items',        'work.items',      'bulk',       'high')
ON CONFLICT (name) DO NOTHING;

-- Incident permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('incidents.view',          'View incidents',                'incidents',       'view',       'low'),
    ('incidents.create',        'Create incidents',              'incidents',       'create',     'low'),
    ('incidents.edit.own',      'Edit own incidents',            'incidents',       'edit.own',   'low'),
    ('incidents.edit.any',      'Edit any incident',             'incidents',       'edit.any',   'medium'),
    ('incidents.resolve',       'Resolve incidents',             'incidents',       'resolve',    'medium'),
    ('incidents.delete',        'Delete incidents',              'incidents',       'delete',     'high'),
    ('incidents.escalate',      'Escalate incidents',            'incidents',       'escalate',   'high')
ON CONFLICT (name) DO NOTHING;

-- Project permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('projects.view',           'View projects',                 'projects',        'view',       'low'),
    ('projects.create',         'Create projects',               'projects',        'create',     'low'),
    ('projects.edit',           'Edit projects',                 'projects',        'edit',       'medium'),
    ('projects.delete',         'Delete projects',               'projects',        'delete',     'high')
ON CONFLICT (name) DO NOTHING;

-- SLA permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('sla.view',                'View SLA policies',             'sla',             'view',       'low'),
    ('sla.manage',              'Manage SLA policies',           'sla',             'manage',     'medium')
ON CONFLICT (name) DO NOTHING;

-- Report permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('reports.view.aggregate',  'View aggregate reports',        'reports',         'view.aggregate', 'low'),
    ('reports.view.detailed',   'View detailed reports',         'reports',         'view.detailed',  'medium'),
    ('reports.export',          'Export reports',                'reports',         'export',         'medium')
ON CONFLICT (name) DO NOTHING;

-- Webhook permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('webhooks.manage',         'Manage webhooks',               'webhooks',        'manage',     'medium')
ON CONFLICT (name) DO NOTHING;

-- Time tracking permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('time.own',                'Track own time',                'time',            'own',        'low'),
    ('time.view.any',           'View anyone''s time entries',   'time',            'view.any',   'low')
ON CONFLICT (name) DO NOTHING;

-- Context and audit
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('context.view',            'View context graph',            'context',         'view',       'low'),
    ('audit.team.read',         'Read team audit',               'audit.team',      'read',       'low'),
    ('audit.system.read',       'Read system audit',             'audit.system',    'read',       'low')
ON CONFLICT (name) DO NOTHING;

-- Agent permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('agents.view',             'View agent identities',         'agents',          'view',       'low'),
    ('agents.manage',           'Create, update, disable agents', 'agents',         'manage',     'high'),
    ('agents.grants',           'Manage agent tool grants',       'agents.grants',  'manage',     'high'),
    ('agents.runs.view',        'View agent runs',               'agents.runs',     'view',       'low')
ON CONFLICT (name) DO NOTHING;

-- API key permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('apikeys.manage',          'Manage integration API keys',   'apikeys',         'manage',     'medium')
ON CONFLICT (name) DO NOTHING;

-- Infrastructure permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('infra.inventory.read',    'Read infrastructure inventory',  'infra.inventory', 'read',       'low'),
    ('infra.metrics.read',      'Read infrastructure metrics',   'infra.metrics',   'read',       'low'),
    ('infra.actions.execute',   'Execute infrastructure actions', 'infra.actions',  'execute',    'critical')
ON CONFLICT (name) DO NOTHING;

-- Wiki / docs permissions
INSERT INTO permissions (name, description, resource, action, risk_level) VALUES
    ('docs.view',               'View documents',                'docs',            'view',       'low'),
    ('docs.create',             'Create documents',              'docs',            'create',     'low'),
    ('docs.edit.own',           'Edit own documents',            'docs',            'edit.own',   'low'),
    ('docs.edit.any',           'Edit any document',             'docs',            'edit.any',   'medium'),
    ('docs.delete',             'Delete documents',              'docs',            'delete',     'high')
ON CONFLICT (name) DO NOTHING;

-- ─── Role-Permission Assignments ───

-- Helper: insert role_permissions using role/permission name lookups
-- We use a DO block to resolve IDs dynamically

DO $$
DECLARE
    -- Role IDs
    r_owner      UUID; r_admin UUID; r_manager UUID; r_member UUID; r_viewer UUID;
    r_oncall     UUID; r_infra UUID; r_security UUID; r_auditor UUID; r_automation UUID;
BEGIN
    SELECT id INTO r_owner      FROM roles WHERE name = 'owner';
    SELECT id INTO r_admin      FROM roles WHERE name = 'admin';
    SELECT id INTO r_manager    FROM roles WHERE name = 'manager';
    SELECT id INTO r_member     FROM roles WHERE name = 'member';
    SELECT id INTO r_viewer     FROM roles WHERE name = 'viewer';
    SELECT id INTO r_oncall     FROM roles WHERE name = 'on_call_engineer';
    SELECT id INTO r_infra      FROM roles WHERE name = 'infrastructure_engineer';
    SELECT id INTO r_security   FROM roles WHERE name = 'security_admin';
    SELECT id INTO r_auditor    FROM roles WHERE name = 'auditor';
    SELECT id INTO r_automation FROM roles WHERE name = 'automation_operator';

    -- owner: full team access
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_owner, id FROM permissions WHERE name IN (
        'team.settings', 'team.members.list', 'team.members.invite', 'team.members.role', 'team.members.remove',
        'team.audit.read',
        'work.items.view', 'work.items.create', 'work.items.edit.own', 'work.items.edit.any',
        'work.items.delete.own', 'work.items.delete.any', 'work.items.assign', 'work.items.comment', 'work.items.bulk',
        'incidents.view', 'incidents.create', 'incidents.edit.own', 'incidents.edit.any',
        'incidents.resolve', 'incidents.delete', 'incidents.escalate',
        'projects.view', 'projects.create', 'projects.edit', 'projects.delete',
        'sla.view', 'sla.manage',
        'reports.view.aggregate', 'reports.view.detailed', 'reports.export',
        'webhooks.manage',
        'time.own', 'time.view.any',
        'context.view', 'audit.team.read',
        'agents.view', 'agents.manage', 'agents.grants', 'agents.runs.view',
        'apikeys.manage',
        'docs.view', 'docs.create', 'docs.edit.own', 'docs.edit.any', 'docs.delete'
    ) ON CONFLICT DO NOTHING;

    -- admin: team management minus some critical ops
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_admin, id FROM permissions WHERE name IN (
        'team.settings', 'team.members.list', 'team.members.invite', 'team.members.role', 'team.members.remove',
        'team.audit.read',
        'work.items.view', 'work.items.create', 'work.items.edit.own', 'work.items.edit.any',
        'work.items.delete.own', 'work.items.delete.any', 'work.items.assign', 'work.items.comment', 'work.items.bulk',
        'incidents.view', 'incidents.create', 'incidents.edit.own', 'incidents.edit.any',
        'incidents.resolve', 'incidents.delete', 'incidents.escalate',
        'projects.view', 'projects.create', 'projects.edit', 'projects.delete',
        'sla.view', 'sla.manage',
        'reports.view.aggregate', 'reports.view.detailed', 'reports.export',
        'webhooks.manage',
        'time.own', 'time.view.any',
        'context.view', 'audit.team.read',
        'agents.view', 'agents.manage', 'agents.grants', 'agents.runs.view',
        'apikeys.manage',
        'docs.view', 'docs.create', 'docs.edit.own', 'docs.edit.any', 'docs.delete'
    ) ON CONFLICT DO NOTHING;

    -- manager: project/work management
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_manager, id FROM permissions WHERE name IN (
        'team.members.list',
        'work.items.view', 'work.items.create', 'work.items.edit.own', 'work.items.edit.any',
        'work.items.delete.own', 'work.items.assign', 'work.items.comment', 'work.items.bulk',
        'incidents.view', 'incidents.create', 'incidents.edit.own', 'incidents.edit.any',
        'incidents.resolve', 'incidents.escalate',
        'projects.view', 'projects.create', 'projects.edit',
        'sla.view', 'sla.manage',
        'reports.view.aggregate', 'reports.view.detailed', 'reports.export',
        'time.own', 'time.view.any',
        'context.view',
        'docs.view', 'docs.create', 'docs.edit.own', 'docs.edit.any'
    ) ON CONFLICT DO NOTHING;

    -- member: own work
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_member, id FROM permissions WHERE name IN (
        'team.members.list',
        'work.items.view', 'work.items.create', 'work.items.edit.own',
        'work.items.delete.own', 'work.items.comment',
        'incidents.view', 'incidents.create', 'incidents.edit.own', 'incidents.resolve',
        'projects.view',
        'sla.view',
        'reports.view.aggregate',
        'time.own',
        'context.view',
        'docs.view', 'docs.create', 'docs.edit.own'
    ) ON CONFLICT DO NOTHING;

    -- viewer: read only
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_viewer, id FROM permissions WHERE name IN (
        'team.members.list',
        'work.items.view',
        'incidents.view',
        'projects.view',
        'sla.view',
        'reports.view.aggregate',
        'context.view',
        'docs.view'
    ) ON CONFLICT DO NOTHING;

    -- on_call_engineer: incident response + runbook access
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_oncall, id FROM permissions WHERE name IN (
        'team.members.list',
        'work.items.view', 'work.items.create', 'work.items.edit.own', 'work.items.edit.any',
        'work.items.comment', 'work.items.assign',
        'incidents.view', 'incidents.create', 'incidents.edit.own', 'incidents.edit.any',
        'incidents.resolve', 'incidents.escalate',
        'projects.view',
        'sla.view', 'sla.manage',
        'reports.view.aggregate',
        'time.own',
        'context.view',
        'docs.view', 'docs.create', 'docs.edit.own', 'docs.edit.any'
    ) ON CONFLICT DO NOTHING;

    -- infrastructure_engineer: infra read + asset management
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_infra, id FROM permissions WHERE name IN (
        'team.members.list',
        'work.items.view', 'work.items.create', 'work.items.edit.own', 'work.items.comment',
        'incidents.view', 'incidents.create', 'incidents.edit.own', 'incidents.resolve',
        'projects.view',
        'infra.inventory.read', 'infra.metrics.read',
        'context.view',
        'docs.view', 'docs.create', 'docs.edit.own',
        'reports.view.aggregate',
        'time.own'
    ) ON CONFLICT DO NOTHING;

    -- security_admin: security policies, audit
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_security, id FROM permissions WHERE name IN (
        'team.members.list',
        'team.audit.read',
        'work.items.view',
        'incidents.view', 'incidents.escalate',
        'projects.view',
        'context.view',
        'audit.team.read', 'audit.system.read',
        'agents.view', 'agents.runs.view',
        'reports.view.aggregate', 'reports.view.detailed',
        'docs.view'
    ) ON CONFLICT DO NOTHING;

    -- auditor: read-only, full audit access, no mutations
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_auditor, id FROM permissions WHERE name IN (
        'team.members.list',
        'team.audit.read',
        'work.items.view',
        'incidents.view',
        'projects.view',
        'sla.view',
        'context.view',
        'audit.team.read', 'audit.system.read',
        'agents.view', 'agents.runs.view',
        'reports.view.aggregate', 'reports.view.detailed', 'reports.export',
        'docs.view'
    ) ON CONFLICT DO NOTHING;

    -- automation_operator: agents + API keys
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r_automation, id FROM permissions WHERE name IN (
        'team.members.list',
        'work.items.view', 'work.items.create', 'work.items.edit.own', 'work.items.comment',
        'incidents.view',
        'projects.view',
        'context.view',
        'agents.view', 'agents.manage', 'agents.grants', 'agents.runs.view',
        'apikeys.manage',
        'webhooks.manage',
        'reports.view.aggregate',
        'docs.view'
    ) ON CONFLICT DO NOTHING;
END$$;
