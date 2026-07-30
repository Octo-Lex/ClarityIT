-- P0 CI-only environment prerequisite for legacy migration 029.
-- This is not an approved production role or grant inventory.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app') THEN
        CREATE ROLE clarityit_app NOLOGIN;
    END IF;
END $$;
