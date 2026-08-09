-- G3 deterministic seed contract: seven G2-canonical permission names
-- plus the immutable baseline revision record. No business/sample data.
-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.
\set ON_ERROR_STOP on
BEGIN;
DO $$ BEGIN ASSERT current_database() = 'clarityit', 'G3 seed requires POSTGRES_DB=clarityit'; END $$;
SET LOCAL ROLE clarityit_owner;
INSERT INTO public.permissions (id, name, description, resource, action, risk_level, created_at) VALUES
    ('53c0f4d2-6fec-5d57-84ca-ed58a8dfc19d', 'work.items.update.own', 'Update own work items', 'work.items', 'update.own', 'low', '2026-08-02T00:00:00Z'),
    ('4f2499c2-ad5d-5215-94cf-1919ca9fa865', 'work.items.update.any', 'Update any work item', 'work.items', 'update.any', 'medium', '2026-08-02T00:00:00Z'),
    ('6a53d14f-8ca0-5be7-9a77-f8775d36efaa', 'projects.update', 'Update projects', 'projects', 'update', 'medium', '2026-08-02T00:00:00Z'),
    ('678fd8d6-56e9-5335-8b72-06ec2cb09f97', 'incidents.update.own', 'Update own incidents', 'incidents', 'update.own', 'low', '2026-08-02T00:00:00Z'),
    ('4c73278f-fd39-585c-a8a8-2508e016bde3', 'incidents.update.any', 'Update any incident', 'incidents', 'update.any', 'medium', '2026-08-02T00:00:00Z'),
    ('bdb6f96a-8577-5763-9a48-19adff491206', 'docs.update.own', 'Update own documents', 'docs', 'update.own', 'low', '2026-08-02T00:00:00Z'),
    ('341bd87b-d622-525d-8c06-94308da39f99', 'docs.update.any', 'Update any document', 'docs', 'update.any', 'medium', '2026-08-02T00:00:00Z');
DO $$
BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM public.permissions WHERE name LIKE '%.edit%'), 'G3 seed contains legacy .edit permission';
    ASSERT (SELECT count(*) FROM public.permissions WHERE name IN ('work.items.update.own','work.items.update.any','projects.update','incidents.update.own','incidents.update.any','docs.update.own','docs.update.any')) = 7, 'G3 canonical permission set incomplete';
END
$$;
INSERT INTO platform.schema_revisions (version, name, checksum, source_commit, applied_at, applied_by, execution_ms, success)
VALUES ('0001', 'g3-reconciled-baseline', '1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508', 'f04f94faad0105d1c3274e9c7974d44f936a0d28', '2026-08-02T00:00:00Z', 'g3-baseline-artifact', 0, true);
COMMIT;
