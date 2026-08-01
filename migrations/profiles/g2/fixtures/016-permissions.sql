-- G2 Fixture: 016 permission collision and resolution
-- Tests both the simple rename case and the dual-row collision case.

-- Setup: seed all 7 .edit permissions (simulating migration 009)
INSERT INTO permissions (id, name, description) VALUES
    ('00000000-0000-4000-8000-000000010001', 'work.items.edit.own', 'Edit own work items'),
    ('00000000-0000-4000-8000-000000010002', 'work.items.edit.any', 'Edit any work item'),
    ('00000000-0000-4000-8000-000000010003', 'projects.edit', 'Edit projects'),
    ('00000000-0000-4000-8000-000000010004', 'incidents.edit.own', 'Edit own incidents'),
    ('00000000-0000-4000-8000-000000010005', 'incidents.edit.any', 'Edit any incident'),
    ('00000000-0000-4000-8000-000000010006', 'docs.edit.own', 'Edit own docs'),
    ('00000000-0000-4000-8000-000000010007', 'docs.edit.any', 'Edit any doc')
ON CONFLICT (name) DO NOTHING;

-- Case 1: Simple rename (no canonical .update row exists)
UPDATE permissions SET name = replace(name, '.edit', '.update') WHERE name LIKE '%.edit%';

-- Verify: zero .edit remain
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions still exist after simple rename';
END $$;

-- Case 2: Collision (both .edit and .update rows exist)
-- Reset: recreate .edit rows alongside .update
INSERT INTO permissions (id, name, description) VALUES
    ('00000000-0000-4000-8000-000000020001', 'incidents.edit.own', 'Legacy edit own incidents'),
    ('00000000-0000-4000-8000-000000020002', 'docs.edit.any', 'Legacy edit any doc')
ON CONFLICT (name) DO NOTHING;

-- Collision resolution: repoint role_permissions to canonical, then delete legacy
-- (Assumes canonical .update rows already exist from Case 1)
UPDATE role_permissions rp SET permission_id = canon.id
FROM permissions legacy, permissions canon
WHERE rp.permission_id = legacy.id
  AND legacy.name LIKE '%.edit%'
  AND canon.name = replace(legacy.name, '.edit', '.update')
  AND canon.id <> legacy.id;

DELETE FROM permissions WHERE name LIKE '%.edit%';

-- Final assertion: zero .edit remain
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        '016 FAIL: .edit permissions still exist after collision resolution';
END $$;
