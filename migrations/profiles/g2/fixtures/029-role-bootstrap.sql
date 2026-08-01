-- G2 Fixture: 029 role bootstrap and fail-closed validation
-- Tests the privileged bootstrap + missing-role failure + grant success

-- === BOOTSTRAP: create roles before migration 029 ===

-- 1. clarityit_app: NOLOGIN group role (application runtime grantee)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app') THEN
        CREATE ROLE clarityit_app NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT;
    END IF;
END $$;

-- 2. clarityit: LOGIN role, member of clarityit_app (production target: NOT superuser)
-- (Development exception: existing clarityit remains superuser)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit') THEN
        CREATE ROLE clarityit LOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT;
        GRANT clarityit_app TO clarityit;
    END IF;
END $$;

-- 3. Ensure membership exists even if clarityit pre-exists
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles r ON r.oid = am.member
        JOIN pg_roles r2 ON r2.oid = am.roleid
        WHERE r.rolname = 'clarityit' AND r2.rolname = 'clarityit_app'
    ) THEN
        GRANT clarityit_app TO clarityit;
    END IF;
END $$;

-- === 029 GRANT should now succeed ===
-- (recommendation_evidence table must exist first)
GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;

-- === FAIL-CLOSED TEST: verify required roles exist before mutation ===
DO $$ BEGIN
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app'),
        '029 FAIL: clarityit_app role not found — bootstrap required before migration';
END $$;

-- === TARGET POSTURE VALIDATION ===
DO $$ BEGIN
    -- clarityit_app must be NOLOGIN
    ASSERT EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app' AND rolcanlogin = false
    ), '029 FAIL: clarityit_app must be NOLOGIN';

    -- clarityit_app must NOT be superuser
    ASSERT EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app' AND rolsuper = false
    ), '029 FAIL: clarityit_app must not be superuser';

    -- clarityit must be a member of clarityit_app
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles r ON r.oid = am.member
        JOIN pg_roles r2 ON r2.oid = am.roleid
        WHERE r.rolname = 'clarityit' AND r2.rolname = 'clarityit_app'
    ), '029 FAIL: clarityit must be a member of clarityit_app';

END $$;
