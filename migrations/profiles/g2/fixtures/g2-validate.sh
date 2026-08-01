-- G2 Fixture: Target grants inventory (closed-world)
-- Defines the required post-decision grant posture for the production target.
-- Every schema, table, sequence, and function has explicit grantee/privilege entries.
-- PUBLIC EXECUTE on functions is revoked by default.

-- === SCHEMA GRANTS ===
GRANT USAGE ON SCHEMA public TO clarityit_app;

-- === TABLE GRANTS (application runtime) ===
-- clarityit_app receives SELECT, INSERT, UPDATE on all application tables.
-- DELETE is NOT granted to clarityit_app by default (soft-delete model).
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO clarityit_app;

-- === SEQUENCE GRANTS ===
-- audit_logs_id_seq: application inserts audit records; needs USAGE
GRANT USAGE, SELECT ON SEQUENCE audit_logs_id_seq TO clarityit_app;

-- === FUNCTION GRANTS ===
-- PostgreSQL grants PUBLIC EXECUTE on all functions by default.
-- The production target REVOKEs this and grants explicitly.
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;

-- Grant EXECUTE on application functions to clarityit_app
-- (set_updated_at and related trigger functions are called by triggers, not the app;
--  trigger execution uses the table owner's privileges, not the caller's.)
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO clarityit_app;

-- === DEFAULT PRIVILEGES ===
-- Objects created by clarityit_owner should default to granting clarityit_app
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE ON TABLES TO clarityit_app;

ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO clarityit_app;

ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO clarityit_app;

-- === MEMBERSHIP OPTIONS ===
-- clarityit is a member of clarityit_app with INHERIT (inherits table privileges)
-- (GRANT ... WITH INHERIT OPTION is the default; explicit for clarity)
GRANT clarityit_app TO clarityit WITH INHERIT OPTION;

-- clarityit_migrator can SET ROLE clarityit_owner (ADMIN option for role switching)
GRANT clarityit_owner TO clarityit_migrator WITH ADMIN OPTION;

-- === GRANT OPTION ===
-- No role has WITH GRANT OPTION on any object (prevents privilege delegation)
