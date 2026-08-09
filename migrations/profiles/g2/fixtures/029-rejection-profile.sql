-- G2 029 Fixture: incorrect-role rejection profile
-- Applied to a SEPARATE disposable cluster that has been seeded with BAD states.
-- Each case is wrapped in a transaction with ROLLBACK so cases are independent.
-- Every assertion here is EXPECTED TO FAIL — the harness confirms the failure.

-- Helper: a validator that encodes the production-target acceptance criteria.
-- Any of these failing means the posture is NOT signature-ready.
CREATE OR REPLACE FUNCTION _reject_if_bad() RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    n_roles int;
    n_memberships int;
BEGIN
    -- Exactly 5 application roles
    SELECT count(*) INTO n_roles FROM pg_roles
        WHERE rolname IN ('clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin','clarityit');
    ASSERT n_roles = 5, 'rejection: expected exactly 5 application roles, got ' || n_roles;

    -- clarityit must be NON-superuser (production target)
    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit' AND rolsuper=true),
        'rejection: clarityit is a superuser — fails production target';

    -- clarityit_app must be INHERIT (matches manifest + decision)
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_app' AND rolinherit=true),
        'rejection: clarityit_app must be INHERIT';

    -- migrator→owner must be (ADMIN=false, INHERIT=false, SET=true)
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles m ON m.oid=am.member JOIN pg_roles r ON r.oid=am.roleid
        WHERE m.rolname='clarityit_migrator' AND r.rolname='clarityit_owner'
        AND am.admin_option=false AND am.inherit_option=false AND am.set_option=true
    ), 'rejection: migrator→owner options wrong';

    -- Exactly 2 memberships
    SELECT count(*) INTO n_memberships FROM pg_auth_members am
        JOIN pg_roles m ON m.oid=am.member JOIN pg_roles r ON r.oid=am.roleid
        WHERE m.rolname IN ('clarityit','clarityit_migrator','clarityit_admin')
        AND r.rolname IN ('clarityit_app','clarityit_owner');
    ASSERT n_memberships = 2, 'rejection: expected 2 memberships, got ' || n_memberships;
END $$;

-- ============================================================
-- CASE R1: clarityit is a SUPERUSER (sole deviation from correct posture)
-- Seeds the COMPLETE correct 5-role posture with clarityit as the ONLY
-- deviation (SUPERUSER). This isolates the superuser assertion: if the
-- validator rejects for any OTHER reason (e.g. a missing role), R1 FAILS.
-- The handler binds to the specific superuser message via SQLERRM.
-- ============================================================
BEGIN;
-- Full correct posture...
CREATE ROLE clarityit_app NOLOGIN INHERIT;
CREATE ROLE clarityit LOGIN INHERIT;
GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT;
CREATE ROLE clarityit_migrator LOGIN NOINHERIT;
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE;
CREATE ROLE clarityit_admin LOGIN NOINHERIT CREATEROLE;
-- ...then introduce the SOLE deviation: make clarityit a superuser.
ALTER ROLE clarityit SUPERUSER;
DO $$ BEGIN
    PERFORM _reject_if_bad();
    RAISE EXCEPTION 'R1 FAIL: superuser clarityit was NOT rejected';
EXCEPTION
    WHEN assert_failure THEN
        -- Must be the superuser-specific assertion, not a different one.
        IF SQLERRM LIKE '%superuser%' THEN
            RAISE NOTICE 'R1 PASS: superuser clarityit correctly rejected';
        ELSE
            RAISE EXCEPTION 'R1 FAIL: rejected for wrong reason: %', SQLERRM;
        END IF;
END $$;
ROLLBACK;

-- ============================================================
-- CASE R2: clarityit_app created with WRONG flags (NOINHERIT)
-- Full correct posture except clarityit_app is NOINHERIT (sole deviation).
-- Handler binds to the specific clarityit_app-INHERIT assertion.
-- ============================================================
BEGIN;
CREATE ROLE clarityit_app NOLOGIN NOINHERIT;
CREATE ROLE clarityit LOGIN INHERIT;
GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT;
CREATE ROLE clarityit_migrator LOGIN NOINHERIT;
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE;
CREATE ROLE clarityit_admin LOGIN NOINHERIT CREATEROLE;
DO $$ BEGIN
    PERFORM _reject_if_bad();
    RAISE EXCEPTION 'R2 FAIL: NOINHERIT clarityit_app was NOT rejected';
EXCEPTION
    WHEN assert_failure THEN
        IF SQLERRM LIKE '%clarityit_app must be INHERIT%' THEN
            RAISE NOTICE 'R2 PASS: wrong clarityit_app flags correctly rejected';
        ELSE
            RAISE EXCEPTION 'R2 FAIL: rejected for wrong reason: %', SQLERRM;
        END IF;
END $$;
ROLLBACK;

-- ============================================================
-- CASE R3: migrator→owner granted with ADMIN TRUE (delegation risk)
-- Full correct posture except migrator→owner has ADMIN TRUE (sole deviation).
-- Handler binds to the specific migrator→owner-options assertion.
-- ============================================================
BEGIN;
CREATE ROLE clarityit_app NOLOGIN INHERIT;
CREATE ROLE clarityit LOGIN INHERIT;
GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT;
CREATE ROLE clarityit_migrator LOGIN NOINHERIT;
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN TRUE, SET TRUE;
CREATE ROLE clarityit_admin LOGIN NOINHERIT CREATEROLE;
DO $$ BEGIN
    PERFORM _reject_if_bad();
    RAISE EXCEPTION 'R3 FAIL: ADMIN TRUE on migrator→owner was NOT rejected';
EXCEPTION
    WHEN assert_failure THEN
        IF SQLERRM LIKE '%migrator→owner options wrong%' THEN
            RAISE NOTICE 'R3 PASS: ADMIN TRUE delegation risk correctly rejected';
        ELSE
            RAISE EXCEPTION 'R3 FAIL: rejected for wrong reason: %', SQLERRM;
        END IF;
END $$;
ROLLBACK;

-- ============================================================
-- CASE R4: partial posture (only 3 of 5 roles — missing admin + migrator)
-- Handler binds to the specific role-count assertion.
-- ============================================================
BEGIN;
CREATE ROLE clarityit_app NOLOGIN INHERIT;
CREATE ROLE clarityit LOGIN INHERIT;
GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT;
-- clarityit_migrator and clarityit_admin deliberately absent
DO $$ BEGIN
    PERFORM _reject_if_bad();
    RAISE EXCEPTION 'R4 FAIL: partial posture (3 roles) was NOT rejected';
EXCEPTION
    WHEN assert_failure THEN
        IF SQLERRM LIKE '%expected exactly 5 application roles%' THEN
            RAISE NOTICE 'R4 PASS: partial posture correctly rejected';
        ELSE
            RAISE EXCEPTION 'R4 FAIL: rejected for wrong reason: %', SQLERRM;
        END IF;
END $$;
ROLLBACK;

-- ============================================================
-- CASE R5: extraneous membership (clarityit_admin wrongly in clarityit_app)
-- Full correct posture plus one extra membership (sole deviation).
-- Handler binds to the specific membership-count assertion.
-- ============================================================
BEGIN;
CREATE ROLE clarityit_app NOLOGIN INHERIT;
CREATE ROLE clarityit LOGIN INHERIT;
GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT;
CREATE ROLE clarityit_migrator LOGIN NOINHERIT;
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE;
CREATE ROLE clarityit_admin LOGIN NOINHERIT CREATEROLE;
GRANT clarityit_app TO clarityit_admin WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
DO $$ BEGIN
    PERFORM _reject_if_bad();
    RAISE EXCEPTION 'R5 FAIL: extraneous admin membership was NOT rejected';
EXCEPTION
    WHEN assert_failure THEN
        IF SQLERRM LIKE '%expected 2 memberships%' THEN
            RAISE NOTICE 'R5 PASS: extraneous membership correctly rejected';
        ELSE
            RAISE EXCEPTION 'R5 FAIL: rejected for wrong reason: %', SQLERRM;
        END IF;
END $$;
ROLLBACK;

DROP FUNCTION _reject_if_bad();
