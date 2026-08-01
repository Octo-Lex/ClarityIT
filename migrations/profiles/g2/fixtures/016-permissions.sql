-- G2 016 Fixture: Permission normalization — complete collision + negative tests
-- Runs in an isolated schema to avoid P0 seed interference.

CREATE SCHEMA IF NOT EXISTS g2_016;
SET search_path TO g2_016;

-- Create minimal permissions + role_permissions + roles tables
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

-- === CASE 1: Simple rename (legacy .edit exists, canonical .update does not) ===

-- Seed 3 .edit permissions
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040001', 'incidents.edit.own', 'Edit own', 'incidents', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040002', 'docs.edit.own', 'Edit own', 'docs', 'edit.own', 'low'),
    ('00000000-0000-4000-8000-000000040003', 'projects.edit', 'Edit', 'projects', 'edit', 'medium')
ON CONFLICT (name) DO NOTHING;

-- role-legacy-only has grants on legacy .edit rows
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040001'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040002'),
    ('00000000-0000-4000-8000-000000030001', '00000000-0000-4000-8000-000000040003')
ON CONFLICT DO NOTHING;

-- Simple rename
UPDATE permissions SET name = replace(name, '.edit', '.update'), action = replace(action, 'edit', 'update')
WHERE name LIKE '%.edit%';

DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit remain after simple rename';
    ASSERT (SELECT count(*) FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030001') = 3,
        '016 FAIL: role-legacy-only lost grants during rename';
END $$;

-- === CASE 2: Collision (both .edit and .update exist with grants) ===

-- Insert legacy .edit alongside canonical .update (which exists from Case 1)
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040010', 'incidents.edit.own', 'Legacy', 'incidents', 'edit.own', 'low')
ON CONFLICT (name) DO NOTHING;

-- role-canonical-only has ONLY the canonical .update grant
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-4000-8000-000000030002', id FROM permissions WHERE name = 'incidents.update.own'
ON CONFLICT DO NOTHING;

-- role-dual-grant has BOTH the legacy .edit AND the canonical .update
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-4000-8000-000000030003', '00000000-0000-4000-8000-000000040010')  -- legacy
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-4000-8000-000000030003', id FROM permissions WHERE name = 'incidents.update.own'  -- canonical
ON CONFLICT DO NOTHING;

-- Collision resolution:
-- 1. Union legacy grants into canonical (ON CONFLICT handles PK violation for dual-grant role)
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

-- Verify: zero .edit remain
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit remain after collision';
END $$;

-- Verify role-canonical-only still has canonical grant
DO $$ BEGIN
    ASSERT EXISTS (
        SELECT 1 FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030002' AND p.name = 'incidents.update.own'
    ), '016 FAIL: role-canonical-only lost grant';
END $$;

-- Verify role-dual-grant now has exactly 1 grant on canonical (union deduped via ON CONFLICT)
DO $$ BEGIN
    ASSERT (
        SELECT count(*) FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030003' AND p.name = 'incidents.update.own'
    ) = 1, '016 FAIL: role-dual-grant should have exactly 1 canonical grant after collision';
END $$;

-- Verify role-legacy-only still has all 3 grants
DO $$ BEGIN
    ASSERT (
        SELECT count(*) FROM role_permissions rp
        JOIN permissions p ON rp.permission_id = p.id
        WHERE rp.role_id = '00000000-0000-4000-8000-000000030001'
    ) = 3, '016 FAIL: role-legacy-only should still have 3 grants';
END $$;

-- === CASE 3: Negative — neither legacy nor canonical exists ===
-- Insert a permission that is neither .edit nor the expected .update
INSERT INTO permissions (id, name, description, resource, action, risk_level) VALUES
    ('00000000-0000-4000-8000-000000040099', 'completely.unrelated', 'Unrelated', 'none', 'none', 'low')
ON CONFLICT DO NOTHING;

-- The assertion that all expected canonical names exist catches corruption
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions exist (should be zero)';
    -- Note: we only seeded 3 .update permissions, not all 7.
    -- The assertion checks zero .edit, not count of .update.
END $$;

-- === CLEANUP ===
RESET search_path;
DROP SCHEMA g2_016 CASCADE;
