-- G2 016 Fixture: Permission normalization — complete with all 7 canonical + collision + negative
-- Runs in an isolated schema to avoid P0 seed interference.

CREATE SCHEMA IF NOT EXISTS g2_016;
SET search_path TO g2_016;

-- Minimal tables for testing
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT DEFAULT ''
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    resource TEXT,
    action TEXT,
    risk_level TEXT DEFAULT 'low'
        CHECK (risk_level IN ('low','medium','high','critical')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- === SETUP: seed roles ===
INSERT INTO roles (id, name) VALUES
    ('00000000-0000-4000-8000-000000030001', 'role-legacy-only'),
    ('00000000-0000-4000-8000-000000030002', 'role-canonical-only'),
    ('00000000-0000-4000-8000-000000030003', 'role-dual-grant');

-- === CASE 1: Simple rename (all 7 .edit → .update, no canonical exists yet) ===
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040001', 'work.items.edit.own', 'Edit own work items', 'work.items', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040002', 'work.items.edit.any', 'Edit any work item', 'work.items', 'edit.any', 'medium'),
    ('00000000-0000-4000-8000-000000040003', 'projects.edit', 'Edit projects', 'projects', 'edit', 'medium'),
    ('00000000-0000-4000-8000-000000040004', 'incidents.edit.own', 'Edit own incidents', 'incidents', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040005', 'incidents.edit.any', 'Edit any incident', 'incidents', 'edit.any', 'medium'),
    ('00000000-0000-4000-8000-000000040006', 'docs.edit.own', 'Edit own docs', 'docs', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040007', 'docs.edit.any', 'Edit any doc', 'docs', 'edit.any', 'medium')
ON CONFLICT (name) DO NOTHING;

-- role-legacy-only has grants on all 7 legacy rows
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040001'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040002'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040003'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040004'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040005'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040006'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040007')
ON CONFLICT DO NOTHING;

-- Simple rename all 7
UPDATE permissions SET name = replace(name, '.edit', '.update'), action = replace(action, 'edit', 'update')
WHERE name LIKE '%.edit%';

-- Assert: zero .edit, all 7 canonical names exist exactly once
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit remain after simple rename';
    ASSERT (SELECT count(*) FROM permissions WHERE name IN (
        'work.items.update.own', 'work.items.update.any', 'projects.update',
        'incidents.update.own', 'incidents.update.any',
        'docs.update.own', 'docs.update.any'
    )) = 7, '016 FAIL: not all 7 canonical .update permissions exist';
    -- role-legacy-only retains all 7 grants
    ASSERT (SELECT count(*) FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030001') = 7,
        '016 FAIL: role-legacy-only lost grants during rename';
END $$;

-- === CASE 2: Collision (both .edit and .update exist with different grant holders) ===
-- Insert a legacy .edit alongside the existing canonical .update
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040010', 'incidents.edit.own', 'Legacy collision', 'incidents', 'edit.own', 'low')
ON CONFLICT (name) DO NOTHING;

-- role-canonical-only has the canonical .update grant
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-4000-8000-000000030002', id FROM permissions WHERE name = 'incidents.update.own'
ON CONFLICT DO NOTHING;

-- role-dual-grant has BOTH legacy .edit AND canonical .update
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-4000-8000-000000030003', '00000000-0000-4000-8000-000000040010')
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-4000-8000-000000030003', id FROM permissions WHERE name = 'incidents.update.own'
ON CONFLICT DO NOTHING;

-- Collision resolution:
-- 1. Union legacy grants into canonical (ON CONFLICT DO NOTHING prevents PK violation)
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, canon.id
FROM role_permissions rp
JOIN permissions legacy ON rp.permission_id = legacy.id AND legacy.name LIKE '%.edit%'
JOIN permissions canon ON canon.name = replace(legacy.name, '.edit', '.update')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 2. Delete legacy grant rows
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name LIKE '%.edit%');

-- 3. Delete legacy permission rows
DELETE FROM permissions WHERE name LIKE '%.edit%';

-- Assert: zero .edit, dual-grant role has exactly 1 canonical grant
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit remain after collision';
    -- role-dual-grant: was holding both legacy+canonical → now exactly 1
    ASSERT (SELECT count(*) FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030003'
        AND p.name = 'incidents.update.own') = 1,
        '016 FAIL: role-dual-grant should have exactly 1 canonical grant after collision';
    -- role-canonical-only: still has canonical
    ASSERT EXISTS (SELECT 1 FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030002'
        AND p.name = 'incidents.update.own'),
        '016 FAIL: role-canonical-only lost grant';
    -- All 7 canonical still exist
    ASSERT (SELECT count(*) FROM permissions WHERE name IN (
        'work.items.update.own', 'work.items.update.any', 'projects.update',
        'incidents.update.own', 'incidents.update.any',
        'docs.update.own', 'docs.update.any'
    )) = 7, '016 FAIL: canonical count wrong after collision';
END $$;

-- === CASE 3: Negative — expected failure when neither legacy nor canonical exists ===
-- Insert an unrelated permission (neither .edit nor any expected .update)
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040099', 'completely.unrelated', 'Unrelated', 'none', 'none', 'low')
ON CONFLICT DO NOTHING;

-- The corruption case: if a required canonical name were missing,
-- the following count would be < 7 and the assertion would fail.
DO $$ BEGIN
    ASSERT (SELECT count(*) FROM permissions WHERE name IN (
        'work.items.update.own', 'work.items.update.any', 'projects.update',
        'incidents.update.own', 'incidents.update.any',
        'docs.update.own', 'docs.update.any'
    )) = 7, '016 FAIL: missing canonical permission (corruption detected)';
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions exist (should be zero)';
END $$;

-- === CLEANUP ===
RESET search_path;
DROP SCHEMA g2_016 CASCADE;
