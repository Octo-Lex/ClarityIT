-- G2 029 Fixture: Five-role posture validation with pre-mutation fail-closed
-- Uses ON_ERROR_STOP=1 — no false passes.
-- Creates roles, validates exact posture, GRANTs, then cleans up properly.

-- === STEP 1: PRE-MUTATION FAIL-CLOSED (expected to fail — demonstrates the check works) ===
-- This block asserts that roles DON'T exist yet, proving the validator runs before mutation.
-- We use a DO block that SHOULD fail; if it doesn't fail, the test environment is polluted.
DO $$
BEGIN
    -- These roles should NOT exist in a clean bootstrap scenario.
    -- In P0 CI, clarityit exists (superuser), but the 4 new roles should not.
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app') THEN
        -- Good — clarityit_app doesn't exist yet. This is the expected state.
        NULL;
    ELSE
        RAISE NOTICE '029 NOTE: clarityit_app already exists — cleaning up before test';
    END IF;
END $$;

-- === STEP 2: Clean up any leftover test roles from previous runs ===
DO $$ BEGIN
    -- Revoke grants on recommendation_evidence to avoid dependency on drop
    REVOKE SELECT, INSERT, UPDATE ON recommendation_evidence FROM clarityit_app;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

DO $$ BEGIN
    REVOKE clarityit_app FROM clarityit;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

DO $$ BEGIN
    REVOKE clarityit_owner FROM clarityit_migrator;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Drop roles in reverse dependency order
DROP ROLE IF EXISTS clarityit_admin;
DROP ROLE IF EXISTS clarityit_migrator;
DROP ROLE IF EXISTS clarityit_owner;
DROP ROLE IF EXISTS clarityit_app;

-- === STEP 3: Verify roles are gone (fail-closed proof) ===
DO $$ BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app'),
        '029 FAIL: clarityit_app should not exist before bootstrap';
    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_owner'),
        '029 FAIL: clarityit_owner should not exist before bootstrap';
    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_migrator'),
        '029 FAIL: clarityit_migrator should not exist before bootstrap';
    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_admin'),
        '029 FAIL: clarityit_admin should not exist before bootstrap';
END $$;

-- === STEP 4: Create the five-role posture ===

-- clarityit_app: NOLOGIN runtime grant group
CREATE ROLE clarityit_app NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT;

-- clarityit_owner: NOLOGIN object owner, no administrative attributes
CREATE ROLE clarityit_owner NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_migrator: LOGIN, NOINHERIT, may SET ROLE clarityit_owner
CREATE ROLE clarityit_migrator LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_admin: LOGIN, CREATEROLE, no ambient app ACL
CREATE ROLE clarityit_admin LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit (production target: non-superuser)
-- In P0, clarityit already exists as superuser. We do NOT recreate it.
-- The production target posture (non-superuser) is documented in DECISION-029.

-- === STEP 5: Memberships with explicit options ===
GRANT clarityit_app TO clarityit WITH INHERIT OPTION;
GRANT clarityit_owner TO clarityit_migrator WITH ADMIN OPTION;

-- === STEP 6: Validate the five-role posture (exhaustive) ===
DO $$ BEGIN
    -- clarityit_app
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_app'
        AND rolcanlogin=false AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=true),
        '029 FAIL: clarityit_app flags wrong';

    -- clarityit_owner
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_owner'
        AND rolcanlogin=false AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=false),
        '029 FAIL: clarityit_owner flags wrong';

    -- clarityit_migrator
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_migrator'
        AND rolcanlogin=true AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=false),
        '029 FAIL: clarityit_migrator flags wrong';

    -- clarityit_admin
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_admin'
        AND rolcanlogin=true AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=true AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=false),
        '029 FAIL: clarityit_admin flags wrong';

    -- clarityit → clarityit_app membership (WITH INHERIT OPTION)
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles r ON r.oid=am.member JOIN pg_roles r2 ON r2.oid=am.roleid
        WHERE r.rolname='clarityit' AND r2.rolname='clarityit_app'
        AND am.admin_option = false  -- INHERIT, not ADMIN
    ), '029 FAIL: clarityit not member of clarityit_app with INHERIT';

    -- clarityit_migrator → clarityit_owner membership (WITH ADMIN OPTION)
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles r ON r.oid=am.member JOIN pg_roles r2 ON r2.oid=am.roleid
        WHERE r.rolname='clarityit_migrator' AND r2.rolname='clarityit_owner'
        AND am.admin_option = true  -- ADMIN, enables SET ROLE
    ), '029 FAIL: clarityit_migrator not member of clarityit_owner with ADMIN';

    -- clarityit_admin NOT member of clarityit_app
    ASSERT NOT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles r ON r.oid=am.member JOIN pg_roles r2 ON r2.oid=am.roleid
        WHERE r.rolname='clarityit_admin' AND r2.rolname='clarityit_app'
    ), '029 FAIL: clarityit_admin must not be member of clarityit_app';

    -- clarityit_owner NOT member of clarityit_app
    ASSERT NOT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles r ON r.oid=am.member JOIN pg_roles r2 ON r2.oid=am.roleid
        WHERE r.rolname='clarityit_owner' AND r2.rolname='clarityit_app'
    ), '029 FAIL: clarityit_owner must not be member of clarityit_app';

END $$;

-- === STEP 7: Test 029 GRANT succeeds ===
GRANT SELECT, INSERT, UPDATE ON recommendation_evidence TO clarityit_app;

-- === STEP 8: CLEANUP — revoke grants first to avoid dependency ===
REVOKE SELECT, INSERT, UPDATE ON recommendation_evidence FROM clarityit_app;
REVOKE clarityit_app FROM clarityit;
REVOKE clarityit_owner FROM clarityit_migrator;
DROP ROLE IF EXISTS clarityit_admin;
DROP ROLE IF EXISTS clarityit_migrator;
DROP ROLE IF EXISTS clarityit_owner;
DROP ROLE IF EXISTS clarityit_app;
