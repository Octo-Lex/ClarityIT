-- Phase 4 closure: Normalize permission names
-- Rename work.items.edit.* → work.items.update.*
-- Rename projects.edit → projects.update

-- 1. Rename work.items.edit.own → work.items.update.own
UPDATE permissions SET name = 'work.items.update.own' WHERE name = 'work.items.edit.own';
-- 2. Rename work.items.edit.any → work.items.update.any
UPDATE permissions SET name = 'work.items.update.any' WHERE name = 'work.items.edit.any';
-- 3. Rename projects.edit → projects.update
UPDATE permissions SET name = 'projects.update' WHERE name = 'projects.edit';

-- Verify
DO $$
BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        'Old edit permissions still exist';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'work.items.update.own'),
        'work.items.update.own missing';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'work.items.update.any'),
        'work.items.update.any missing';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'projects.update'),
        'projects.update missing';
END $$;
