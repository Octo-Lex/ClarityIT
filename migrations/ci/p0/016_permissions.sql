-- P0 CI-only repair for legacy migration 016.
-- This is not an approved G2 permission decision.
DO $$
DECLARE
    mapping RECORD;
    legacy_id UUID;
    canonical_id UUID;
BEGIN
    FOR mapping IN
        SELECT *
        FROM (VALUES
            ('work.items.edit.own', 'work.items.update.own', 'work.items', 'update.own', 'Update own work items', 'low'),
            ('work.items.edit.any', 'work.items.update.any', 'work.items', 'update.any', 'Update any work item', 'medium'),
            ('projects.edit', 'projects.update', 'projects', 'update', 'Update projects', 'medium'),
            ('incidents.edit.own', 'incidents.update', 'incidents', 'update', 'Update incidents', 'medium'),
            ('incidents.edit.any', 'incidents.update', 'incidents', 'update', 'Update incidents', 'medium'),
            ('docs.edit.own', 'docs.update', 'docs', 'update', 'Update documents', 'medium'),
            ('docs.edit.any', 'docs.update', 'docs', 'update', 'Update documents', 'medium')
        ) AS mappings(legacy_name, canonical_name, resource_name, action_name, description_text, risk_name)
    LOOP
        SELECT id INTO legacy_id FROM permissions WHERE name = mapping.legacy_name;
        SELECT id INTO canonical_id FROM permissions WHERE name = mapping.canonical_name;

        IF canonical_id IS NULL AND legacy_id IS NULL THEN
            INSERT INTO permissions (name, description, resource, action, risk_level)
            VALUES (
                mapping.canonical_name,
                mapping.description_text,
                mapping.resource_name,
                mapping.action_name,
                mapping.risk_name
            )
            RETURNING id INTO canonical_id;
        ELSIF canonical_id IS NULL THEN
            UPDATE permissions
            SET name = mapping.canonical_name,
                description = mapping.description_text,
                resource = mapping.resource_name,
                action = mapping.action_name,
                risk_level = mapping.risk_name
            WHERE id = legacy_id
            RETURNING id INTO canonical_id;
        ELSE
            UPDATE permissions
            SET description = mapping.description_text,
                resource = mapping.resource_name,
                action = mapping.action_name,
                risk_level = mapping.risk_name
            WHERE id = canonical_id;

            IF legacy_id IS NOT NULL AND legacy_id <> canonical_id THEN
                INSERT INTO role_permissions (role_id, permission_id)
                SELECT role_id, canonical_id
                FROM role_permissions
                WHERE permission_id = legacy_id
                ON CONFLICT (role_id, permission_id) DO NOTHING;

                DELETE FROM role_permissions WHERE permission_id = legacy_id;
                DELETE FROM permissions WHERE id = legacy_id;
            END IF;
        END IF;

        legacy_id := NULL;
        canonical_id := NULL;
    END LOOP;
END $$;

DO $$
BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        'legacy edit permissions remain after P0 reconciliation';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'work.items.update.own'),
        'work.items.update.own missing';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'work.items.update.any'),
        'work.items.update.any missing';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'projects.update'),
        'projects.update missing';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'incidents.update'),
        'incidents.update missing';
    ASSERT EXISTS (SELECT 1 FROM permissions WHERE name = 'docs.update'),
        'docs.update missing';
END $$;
