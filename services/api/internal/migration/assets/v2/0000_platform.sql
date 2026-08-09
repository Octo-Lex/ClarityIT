-- G3 minimal migration-control schema (four tables).
-- Product-schema identity remains the signed G2 manifest; these objects
-- are governed by CONTROL-SCHEMA-MANIFEST.json.
-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.
\set ON_ERROR_STOP on
BEGIN;
DO $$ BEGIN ASSERT current_database() = 'clarityit', 'G3 platform bootstrap requires POSTGRES_DB=clarityit'; END $$;
SET LOCAL ROLE clarityit_owner;
CREATE SCHEMA platform AUTHORIZATION clarityit_owner;
REVOKE ALL ON SCHEMA platform FROM PUBLIC;
REVOKE ALL ON SCHEMA platform FROM clarityit_app;

CREATE TABLE platform.source_profiles (
    profile_id text NOT NULL,
    schema_fingerprint text NOT NULL,
    postgres_version text NOT NULL,
    postgres_major integer NOT NULL,
    extensions jsonb NOT NULL,
    roles_digest text NOT NULL,
    source_commit text NOT NULL,
    approved_by text NOT NULL,
    approved_at timestamp with time zone NOT NULL,
    CONSTRAINT source_profiles_pkey PRIMARY KEY (profile_id),
    CONSTRAINT source_profiles_fingerprint_sha256 CHECK (schema_fingerprint ~ '^[0-9a-f]{64}$'),
    CONSTRAINT source_profiles_roles_sha256 CHECK (roles_digest ~ '^[0-9a-f]{64}$'),
    CONSTRAINT source_profiles_pg16 CHECK (postgres_major = 16),
    CONSTRAINT source_profiles_extensions_array CHECK (jsonb_typeof(extensions) = 'array'),
    CONSTRAINT source_profiles_approver_present CHECK (btrim(approved_by) <> '')
);

CREATE TABLE platform.schema_revisions (
    version text NOT NULL,
    name text NOT NULL,
    checksum text NOT NULL,
    source_commit text NOT NULL,
    applied_at timestamp with time zone NOT NULL,
    applied_by text NOT NULL,
    execution_ms bigint NOT NULL,
    success boolean NOT NULL,
    CONSTRAINT schema_revisions_pkey PRIMARY KEY (version),
    CONSTRAINT schema_revisions_checksum_key UNIQUE (checksum),
    CONSTRAINT schema_revisions_checksum_sha256 CHECK (checksum ~ '^[0-9a-f]{64}$'),
    CONSTRAINT schema_revisions_execution_nonnegative CHECK (execution_ms >= 0)
);

CREATE TABLE platform.migration_runs (
    run_id uuid NOT NULL DEFAULT gen_random_uuid(),
    database_name text NOT NULL DEFAULT current_database(),
    source_profile_id text,
    target_version text NOT NULL,
    state text NOT NULL,
    started_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    release_id text NOT NULL,
    evidence_ref text,
    CONSTRAINT migration_runs_pkey PRIMARY KEY (run_id),
    CONSTRAINT migration_runs_source_profile_fkey FOREIGN KEY (source_profile_id) REFERENCES platform.source_profiles(profile_id),
    CONSTRAINT migration_runs_state_check CHECK (state IN ('planned','profiled','preflighted','expanding','backfilling','reconciling','cutover_ready','cutover_committed','observing','completed','blocked','paused','precommit_rolled_back','forward_recovery_required')),
    CONSTRAINT migration_runs_time_order CHECK (completed_at IS NULL OR completed_at >= started_at)
);
CREATE UNIQUE INDEX migration_runs_one_active_per_database ON platform.migration_runs (database_name) WHERE state NOT IN ('completed','blocked','precommit_rolled_back');

CREATE TABLE platform.reconciliation_results (
    run_id uuid NOT NULL,
    check_id text NOT NULL,
    scope text NOT NULL,
    expected jsonb NOT NULL,
    actual jsonb NOT NULL,
    result text NOT NULL,
    evidence_ref text NOT NULL,
    recorded_at timestamp with time zone NOT NULL,
    CONSTRAINT reconciliation_results_pkey PRIMARY KEY (run_id, check_id, scope),
    CONSTRAINT reconciliation_results_run_fkey FOREIGN KEY (run_id) REFERENCES platform.migration_runs(run_id),
    CONSTRAINT reconciliation_results_result_check CHECK (result IN ('pass','fail','blocked')),
    CONSTRAINT reconciliation_results_evidence_present CHECK (btrim(evidence_ref) <> '')
);

CREATE FUNCTION platform.protect_succeeded_revision() RETURNS trigger
LANGUAGE plpgsql AS $function$
BEGIN
    IF TG_OP = 'DELETE' OR OLD.success THEN
        RAISE EXCEPTION 'successful schema revision is immutable';
    END IF;
    RETURN NEW;
END;
$function$;
CREATE TRIGGER schema_revisions_immutable
BEFORE UPDATE OR DELETE ON platform.schema_revisions
FOR EACH ROW EXECUTE FUNCTION platform.protect_succeeded_revision();

CREATE FUNCTION platform.reject_reconciliation_mutation() RETURNS trigger
LANGUAGE plpgsql AS $function$
BEGIN
    RAISE EXCEPTION 'reconciliation result is append-only';
    RETURN NULL;
END;
$function$;
CREATE TRIGGER reconciliation_results_append_only
BEFORE UPDATE OR DELETE ON platform.reconciliation_results
FOR EACH ROW EXECUTE FUNCTION platform.reject_reconciliation_mutation();

REVOKE ALL ON ALL TABLES IN SCHEMA platform FROM PUBLIC, clarityit_app;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA platform FROM PUBLIC, clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA platform REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA platform REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
COMMIT;
