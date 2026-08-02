-- G2 029 Fixture: Disposable PostgreSQL 16 cluster validation
-- Runs in a SEPARATE database (g2_029_test) with a dedicated bootstrap admin.
-- Tests: exact 5-role posture, exact membership options (ADMIN/INHERIT/SET),
-- non-superuser clarityit.
-- This fixture is designed to be applied to a freshly created empty database.
--
-- The pre-mutation fail-closed check (no roles in fresh DB) is performed by the
-- harness BEFORE this file is applied. This file therefore assumes the fresh-DB
-- state and proceeds directly to bootstrap.
--
-- PostgreSQL 16 membership options: each pg_auth_members row stores three
-- independent booleans — admin_option, inherit_option, set_option. Any option
-- omitted on GRANT defaults to TRUE. We therefore specify all three explicitly
-- on every grant and validate all three below.

-- === STEP 1: Bootstrap the five-role posture ===
-- clarityit_app: NOLOGIN runtime grant group; INHERIT (matches manifest + decision)
CREATE ROLE clarityit_app NOLOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_owner: NOLOGIN object owner; NOINHERIT
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_migrator: LOGIN, NOINHERIT
CREATE ROLE clarityit_migrator LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_admin: LOGIN, CREATEROLE, NOINHERIT
CREATE ROLE clarityit_admin LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit: LOGIN, INHERIT, NON-superuser (production target)
CREATE ROLE clarityit LOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- === STEP 2: Memberships with explicit PostgreSQL 16 options ===
-- All three options stated explicitly because the omitted default is TRUE.

-- clarityit → clarityit_app: INHERIT TRUE (inherit app privileges), ADMIN FALSE (no delegation), SET FALSE
GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;

-- clarityit_migrator → clarityit_owner: SET TRUE (authorize SET ROLE), ADMIN FALSE (no delegation), INHERIT FALSE (no ambient owner)
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE;

-- === STEP 3: Validate exact role flags ===
DO $$ BEGIN
    -- clarityit_app: INHERIT (per manifest + decision)
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_app'
        AND rolcanlogin=false AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=true),
        '029 FAIL: clarityit_app flags wrong (expected INHERIT)';

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

    -- clarityit: NON-superuser (production target)
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit'
        AND rolcanlogin=true AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=true),
        '029 FAIL: clarityit should be non-superuser LOGIN INHERIT';

END $$;

-- === STEP 4: Validate ALL THREE membership options exactly ===
DO $$ BEGIN
    -- clarityit → clarityit_app: admin_option=false, inherit_option=true, set_option=false
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname='clarityit' AND role.rolname='clarityit_app'
        AND am.admin_option = false
        AND am.inherit_option = true
        AND am.set_option = false
    ), '029 FAIL: clarityit→clarityit_app must be (ADMIN=false, INHERIT=true, SET=false)';

    -- clarityit_migrator → clarityit_owner: admin_option=false, inherit_option=false, set_option=true
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname='clarityit_migrator' AND role.rolname='clarityit_owner'
        AND am.admin_option = false
        AND am.inherit_option = false
        AND am.set_option = true
    ), '029 FAIL: clarityit_migrator→clarityit_owner must be (ADMIN=false, INHERIT=false, SET=true)';

    -- No other memberships should exist
    ASSERT (
        SELECT count(*) FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname IN ('clarityit', 'clarityit_migrator', 'clarityit_admin')
        AND role.rolname IN ('clarityit_app', 'clarityit_owner')
    ) = 2, '029 FAIL: exactly 2 memberships expected';

    -- clarityit_admin NOT member of clarityit_app or clarityit_owner
    ASSERT NOT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname='clarityit_admin'
        AND role.rolname IN ('clarityit_app', 'clarityit_owner')
    ), '029 FAIL: clarityit_admin must not be member of app/owner';

    -- clarityit_owner NOT member of clarityit_app
    ASSERT NOT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname='clarityit_owner' AND role.rolname='clarityit_app'
    ), '029 FAIL: clarityit_owner must not be member of clarityit_app';

END $$;
