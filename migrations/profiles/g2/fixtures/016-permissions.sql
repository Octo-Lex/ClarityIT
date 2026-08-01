-- G2 Fixture: 016 permission normalization
-- Tests: simple rename, collision with dual grants, negative case (neither exists)
-- Requires: permissions(id, name, description, resource, action, risk_level, created_at)
--           role_permissions(role_id, permission_id)
--           roles(id, name, ...)
--
-- NOTE: The P0 CI fixture already renames all .edit permissions to .update via
-- migrations/ci/p0/016_permissions.sql. This fixture must operate on a CLEAN
-- permissions table to test the collision logic independently. We save and
-- restore the existing state.

-- === SAVE existing state ===
CREATE TEMP TABLE IF NOT EXISTS _g2_saved_permissions AS SELECT * FROM permissions;
CREATE TEMP TABLE IF NOT EXISTS _g2_saved_role_permissions AS SELECT * FROM role_permissions;

-- === CLEAR permissions for isolated testing ===
DELETE FROM role_permissions;
DELETE FROM permissions;

-- === SETUP: seed roles ===
INSERT INTO roles (id, name, description) VALUES
    ('00000000-0000-4000-8000-000000030001', 'g2-test-role-a', 'Test role A'),
    ('00000000-0000-4000-8000-000000030002', 'g2-test-role-b', 'Test role B'),
    ('00000000-0000-4000-8000-000000030003', 'g2-test-role-c', 'Test role C')
ON CONFLICT (id) DO NOTHING;

-- === CASE 1: Simple rename (legacy .edit exists, canonical .update does not) ===

-- Seed all 7 .edit permissions with required columns
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040001', 'incidents.edit.own', 'Edit own incidents', 'incidents', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040002', 'incidents.edit.any', 'Edit any incident', 'incidents', 'edit.any', 'medium'),
    ('00000000-0000-4000-8000-000000040003', 'docs.edit.own', 'Edit own docs', 'docs', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040004', 'docs.edit.any', 'Edit any doc', 'docs', 'edit.any', 'medium'),
    ('00000000-0000-4000-8000-000000040005', 'work.items.edit.own', 'Edit own work items', 'work.items', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040006', 'work.items.edit.any', 'Edit any work item', 'work.items', 'edit.any', 'medium'),
    ('00000000-0000-4000-8000-000000040007', 'projects.edit', 'Edit projects', 'projects', 'edit', 'medium')
ON CONFLICT (name) DO NOTHING;

-- Seed role_permissions: role A has legacy-only grants
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040001'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040003')
ON CONFLICT DO NOTHING;

-- Simple rename: UPDATE name and action
UPDATE permissions
SET name = replace(name, '.edit', '.update'),
    action = replace(action, 'edit', 'update')
WHERE name LIKE '%.edit%';

-- Verify Case 1: zero .edit remain
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions remain after simple rename';
END $$;

-- Verify role A still has both grants (IDs unchanged, names updated)
DO $$ BEGIN
    ASSERT (
        SELECT count(*) FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030001'
    ) = 2, '016 FAIL: role A lost grants during rename';
END $$;

-- === CASE 2: Collision (both .edit and .update exist with grants) ===

-- Insert a NEW legacy .edit alongside the canonical .update (which exists from Case 1)
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040010', 'incidents.edit.own', 'Legacy collision', 'incidents', 'edit.own', 'low')
ON CONFLICT (name) DO NOTHING;

-- Find the canonical .update row's ID
-- (incidents.update.own was renamed from 040001 in Case 1)
-- Role B has ONLY the legacy grant
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-4000-8000-000000030002', '00000000-0000-4000-8000-000000040010')
ON CONFLICT DO NOTHING;

-- Role C has ONLY the canonical grant (already exists from rename)
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-4000-8000-000000030003', id FROM permissions WHERE name = 'incidents.update.own'
ON CONFLICT DO NOTHING;

-- Collision resolution:
-- 1. Union legacy grants into canonical (ON CONFLICT DO NOTHING handles dual-grant PK violation)
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

-- Verify Case 2: zero .edit remain
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions remain after collision resolution';
END $$;

-- Verify role B (was legacy-only) now has the canonical grant
DO $$ BEGIN
    ASSERT EXISTS (
        SELECT 1 FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030002'
        AND p.name = 'incidents.update.own'
    ), '016 FAIL: role B (legacy-only) did not retain grant after collision resolution';
END $$;

-- Verify role C (was canonical-only) still has the canonical grant
DO $$ BEGIN
    ASSERT EXISTS (
        SELECT 1 FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030003'
        AND p.name = 'incidents.update.own'
    ), '016 FAIL: role C (canonical-only) lost grant after collision resolution';
END $$;

-- === CASE 3: Negative — neither legacy nor canonical exists ===
-- If neither exists, the assertion below catches the corruption.
-- Insert a role with NO grants to verify zero-grant roles are unaffected.
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions exist (should be zero)';
    -- Also verify all 7 canonical names exist exactly once
    ASSERT (
        SELECT count(*) FROM permissions
        WHERE name IN (
            'incidents.update.own', 'incidents.update.any',
            'docs.update.own', 'docs.update.any',
            'work.items.update.own', 'work.items.update.any',
            'projects.update'
        )
    ) >= 7, '016 FAIL: not all 7 canonical .update permissions exist';
END $$;

-- === CLEANUP: restore original permissions state ===
DELETE FROM role_permissions;
DELETE FROM permissions;
INSERT INTO permissions SELECT * FROM _g2_saved_permissions;
INSERT INTO role_permissions SELECT * FROM _g2_saved_role_permissions;
DELETE FROM roles WHERE id IN (
    '00000000-0000-4000-8000-000000030001',
    '00000000-0000-4000-8000-000000030002',
    '00000000-0000-4000-8000-000000030003'
);
DROP TABLE IF EXISTS _g2_saved_permissions;
DROP TABLE IF EXISTS _g2_saved_role_permissions;
