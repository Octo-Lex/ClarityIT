-- G3 privileged bootstrap: exact signed five-role posture.
-- Administrator-only; contains no passwords or environment identities.
-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.
\set ON_ERROR_STOP on
BEGIN;
DO $g3_preflight$
BEGIN
    IF current_database() <> 'clarityit' THEN
        RAISE EXCEPTION 'G3 role bootstrap requires database clarityit, got %', current_database();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN
        RAISE EXCEPTION 'G3 role bootstrap requires a PostgreSQL superuser';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')) THEN
        RAISE EXCEPTION 'G3 role bootstrap is fresh-install only: target role already exists';
    END IF;
END
$g3_preflight$;

CREATE ROLE clarityit_app NOLOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit LOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_migrator LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_admin LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE;

ALTER DATABASE clarityit OWNER TO clarityit_owner;
ALTER SCHEMA public OWNER TO clarityit_owner;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO clarityit_app;

ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public GRANT SELECT, INSERT, UPDATE ON TABLES TO clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

DO $g3_validate$
BEGIN
    ASSERT (SELECT count(*) FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')) = 5, 'G3 role count mismatch';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND rolsuper = false AND rolinherit = true AND rolcreaterole = false AND rolcreatedb = false AND rolcanlogin = true AND rolreplication = false AND rolbypassrls = false), 'G3 role flags mismatch: clarityit';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_admin' AND rolsuper = false AND rolinherit = false AND rolcreaterole = true AND rolcreatedb = false AND rolcanlogin = true AND rolreplication = false AND rolbypassrls = false), 'G3 role flags mismatch: clarityit_admin';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app' AND rolsuper = false AND rolinherit = true AND rolcreaterole = false AND rolcreatedb = false AND rolcanlogin = false AND rolreplication = false AND rolbypassrls = false), 'G3 role flags mismatch: clarityit_app';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_migrator' AND rolsuper = false AND rolinherit = false AND rolcreaterole = false AND rolcreatedb = false AND rolcanlogin = true AND rolreplication = false AND rolbypassrls = false), 'G3 role flags mismatch: clarityit_migrator';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_owner' AND rolsuper = false AND rolinherit = false AND rolcreaterole = false AND rolcreatedb = false AND rolcanlogin = false AND rolreplication = false AND rolbypassrls = false), 'G3 role flags mismatch: clarityit_owner';
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid = am.member
        JOIN pg_roles granted ON granted.oid = am.roleid
        WHERE member.rolname = 'clarityit'
          AND granted.rolname = 'clarityit_app'
          AND am.admin_option = false
          AND am.inherit_option = true
          AND am.set_option = false
    ), 'G3 membership mismatch: clarityit -> clarityit_app';
    ASSERT EXISTS (
        SELECT 1 FROM pg_auth_members am
        JOIN pg_roles member ON member.oid = am.member
        JOIN pg_roles granted ON granted.oid = am.roleid
        WHERE member.rolname = 'clarityit_migrator'
          AND granted.rolname = 'clarityit_owner'
          AND am.admin_option = false
          AND am.inherit_option = false
          AND am.set_option = true
    ), 'G3 membership mismatch: clarityit_migrator -> clarityit_owner';
END
$g3_validate$;
COMMIT;
