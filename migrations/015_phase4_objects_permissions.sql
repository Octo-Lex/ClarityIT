-- Phase 4: Object spine + domain permissions

-- Add missing object-level permissions
INSERT INTO permissions (name, resource, action, risk_level) VALUES
  ('objects.create',           'objects',            'create', 'low'),
  ('objects.read',             'objects',            'read',   'low'),
  ('objects.update',           'objects',            'update', 'medium'),
  ('objects.delete',           'objects',            'delete', 'high'),
  ('objects.links.create',     'objects.links',      'create', 'low'),
  ('objects.links.read',       'objects.links',      'read',   'low'),
  ('objects.links.delete',     'objects.links',      'delete', 'medium'),
  ('objects.comments.create',  'objects.comments',   'create', 'low'),
  ('objects.comments.read',    'objects.comments',   'read',   'low'),
  ('objects.comments.update.own', 'objects.comments', 'update', 'low'),
  ('objects.comments.update.any', 'objects.comments', 'update', 'medium'),
  ('objects.comments.delete.own', 'objects.comments', 'delete', 'medium'),
  ('objects.comments.delete.any', 'objects.comments', 'delete', 'high'),
  ('incidents.read',           'incidents',          'read',   'low'),
  ('incidents.update',         'incidents',          'update', 'medium'),
  ('incidents.timeline.create','incidents.timeline', 'create', 'low')
ON CONFLICT (name) DO NOTHING;

-- Map to roles
-- owner: full access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'owner'
  AND p.name IN (
    'objects.create','objects.read','objects.update','objects.delete',
    'objects.links.create','objects.links.read','objects.links.delete',
    'objects.comments.create','objects.comments.read',
    'objects.comments.update.own','objects.comments.update.any',
    'objects.comments.delete.own','objects.comments.delete.any',
    'work.items.create','work.items.view','work.items.edit.own','work.items.edit.any',
    'work.items.delete.own','work.items.delete.any','work.items.assign','work.items.comment',
    'incidents.create','incidents.read','incidents.update','incidents.resolve',
    'incidents.escalate','incidents.timeline.create',
    'projects.create','projects.view','projects.edit','projects.delete'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- admin: same as owner
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN (
    'objects.create','objects.read','objects.update','objects.delete',
    'objects.links.create','objects.links.read','objects.links.delete',
    'objects.comments.create','objects.comments.read',
    'objects.comments.update.own','objects.comments.update.any',
    'objects.comments.delete.own','objects.comments.delete.any',
    'work.items.create','work.items.view','work.items.edit.own','work.items.edit.any',
    'work.items.delete.own','work.items.delete.any','work.items.assign',
    'incidents.create','incidents.read','incidents.update','incidents.resolve',
    'incidents.escalate','incidents.timeline.create',
    'projects.create','projects.view','projects.edit','projects.delete'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- member: create/read/edit own/delete own
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name IN (
    'objects.create','objects.read','objects.update',
    'objects.links.create','objects.links.read',
    'objects.comments.create','objects.comments.read',
    'objects.comments.update.own','objects.comments.delete.own',
    'work.items.create','work.items.view','work.items.edit.own','work.items.delete.own',
    'incidents.create','incidents.read',
    'projects.create','projects.view'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- viewer: read only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'viewer'
  AND p.name IN (
    'objects.read','objects.links.read','objects.comments.read',
    'work.items.view','incidents.read','projects.view'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Add metadata column to objects if not exists (for projects)
ALTER TABLE objects ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
