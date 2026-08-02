-- Phase 3: Team management + invitations + access grants enhancements

-- Add role_id to team_access_grants (spec requires role reference, not role text)
ALTER TABLE team_access_grants ADD COLUMN IF NOT EXISTS role_id UUID REFERENCES roles(id);

-- Add missing permissions for Phase 3 endpoints
INSERT INTO permissions (name, resource, action, risk_level) VALUES
  ('team.settings.read',       'team.settings',    'read',   'low'),
  ('team.settings.update',     'team.settings',    'update', 'medium'),
  ('team.members.read',        'team.members',     'read',   'low'),
  ('team.members.update',      'team.members',     'update', 'medium'),
  ('team.invitations.create',  'team.invitations', 'create', 'medium'),
  ('team.invitations.read',    'team.invitations', 'read',   'low'),
  ('team.invitations.revoke',  'team.invitations', 'revoke', 'medium'),
  ('team.access_grants.read',  'team.access_grants', 'read',   'low'),
  ('team.access_grants.create','team.access_grants', 'create', 'high'),
  ('team.access_grants.revoke','team.access_grants', 'revoke', 'medium')
ON CONFLICT (name) DO NOTHING;

-- Map new permissions to roles
-- owner gets all team permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'owner'
  AND p.name IN (
    'team.settings.read', 'team.settings.update',
    'team.members.read', 'team.members.update', 'team.members.remove', 'team.members.role',
    'team.members.invite', 'team.members.list',
    'team.invitations.create', 'team.invitations.read', 'team.invitations.revoke',
    'team.access_grants.read', 'team.access_grants.create', 'team.access_grants.revoke',
    'team.audit.read', 'team.settings'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- admin gets most team permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN (
    'team.settings.read', 'team.settings.update',
    'team.members.read', 'team.members.update', 'team.members.remove', 'team.members.role',
    'team.members.invite', 'team.members.list',
    'team.invitations.create', 'team.invitations.read', 'team.invitations.revoke',
    'team.access_grants.read', 'team.access_grants.create', 'team.access_grants.revoke',
    'team.audit.read', 'team.settings'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- manager gets member management + invitations
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name IN (
    'team.settings.read',
    'team.members.read', 'team.members.list',
    'team.members.invite',
    'team.invitations.create', 'team.invitations.read',
    'team.access_grants.read'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- member gets read-only team access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name IN (
    'team.settings.read',
    'team.members.read', 'team.members.list'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- viewer gets minimal team read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'viewer'
  AND p.name IN (
    'team.settings.read',
    'team.members.read', 'team.members.list'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
