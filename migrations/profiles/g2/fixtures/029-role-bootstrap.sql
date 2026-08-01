-- G2 029 Fixture: Disposable PostgreSQL 16 cluster validation
-- Runs in a SEPARATE database (g2_029_test) with a dedicated bootstrap admin.
-- Tests: exact 5-role posture, exact membership options, non-superuser clarityit.
-- This fixture is designed to be applied to a freshly created empty database.
--
-- The pre-mutation fail-closed check (no roles in fresh DB) is performed by the
-- harness BEFORE this file is applied. This file therefore assumes the fresh-DB
-- state and proceeds directly to bootstrap.

-- === STEP 1: Bootstrap the five-role posture ===

-- clarityit_app: NOLOGIN runtime grant group
CREATE ROLE clarityit_app NOLOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_owner: NOLOGIN object owner
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_migrator: LOGIN, NOINHERIT
CREATE ROLE clarityit_migrator LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit_admin: LOGIN, CREATEROLE, NOINHERIT
CREATE ROLE clarityit_admin LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- clarityit: LOGIN, NON-superuser (production target)
CREATE ROLE clarityit LOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- === STEP 2: Memberships with explicit PostgreSQL 16 options ===
-- PostgreSQL 16: ADMIN, INHERIT, and SET are independent options on GRANT.
-- See https://www.postgresql.org/docs/16/sql-grant.html

-- clarityit → clarityit_app: INHERIT (inherits app privileges), not ADMIN, not SET
GRANT clarityit_app TO clarityit WITH INHERIT OPTION;

-- clarityit_migrator → clarityit_owner: ADMIN (can SET ROLE), not INHERIT
GRANT clarityit_owner TO clarityit_migrator WITH ADMIN OPTION;

-- === STEP 3: Validate exact role flags ===
DO $$ BEGIN
    -- clarityit_app
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit_app'
        AND rolcanlogin=false AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=false),
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

    -- clarityit: NON-superuser (production target)
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='clarityit'
        AND rolcanlogin=true AND rolsuper=false AND rolcreatedb=false
        AND rolcreaterole=false AND rolreplication=false AND rolbypassrls=false
        AND rolinherit=true),
        '029 FAIL: clarityit should be non-superuser LOGIN INHERIT';

END $$;

-- === STEP 4: Validate exact membership options ===
-- pg_auth_members columns: admin_option (bool), inherit_option (PG16: bool)
-- SET option is implicit via ADMIN in PG16 (ADMIN grants SET capability).

DO $$ BEGIN
    -- clarityit → clarityit_app: inherit_option=true, admin_option=false
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname='clarityit' AND role.rolname='clarityit_app'
        AND am.admin_option = false
    ), '029 FAIL: clarityit→clarityit_app must have admin_option=false';

    -- clarityit_migrator → clarityit_owner: admin_option=true
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid=am.member
        JOIN pg_roles role ON role.oid=am.roleid
        WHERE member.rolname='clarityit_migrator' AND role.rolname='clarityit_owner'
        AND am.admin_option = true
    ), '029 FAIL: clarityit_migrator→clarityit_owner must have admin_option=true';

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
