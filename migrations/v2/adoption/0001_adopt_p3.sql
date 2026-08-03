-- G3 deterministic P3 approved-source adoption artifact.
-- Reconciles an existing P3 source to the signed G2 governed posture.
-- No legacy replay, no product-table creation, no business-row mutation.
-- The only product-row write is the seven canonical permission inserts.
-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.
\set ON_ERROR_STOP on
BEGIN;

-- Bind the adoption producing commit and the verified source fingerprint
-- at execution time.  The proof harness computes the live profiler
-- fingerprint (via pre_adopt_verify) and passes it as g3_source_fingerprint;
-- the SQL asserts it matches the G1-approved P3 golden so adoption is
-- fail-closed against a drifted source even when invoked directly.
SELECT set_config('g3.source_commit', :'g3_source_commit', true);
SELECT set_config('g3.source_fingerprint', :'g3_source_fingerprint', true);
DO $$ BEGIN ASSERT current_setting('g3.source_commit', true) ~ '^[0-9a-f]{40}$',
    'g3.source_commit must be set to a 40-char lowercase hex SHA'; END $$;
DO $$ BEGIN ASSERT current_setting('g3.source_fingerprint', true) = 'cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b',
    'g3.source_fingerprint must equal the G1-approved P3 golden'; END $$;

-- ============================================================
-- Preflight (read-only): the source must be the approved P3 shape.
-- ============================================================
DO $g3_adopt_preflight$
BEGIN
    IF current_database() <> 'clarityit' THEN
        RAISE EXCEPTION 'G3 adoption requires database clarityit, got %', current_database();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN
        RAISE EXCEPTION 'G3 adoption requires a PostgreSQL superuser';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN
        RAISE EXCEPTION 'G3 adoption requires extension pgcrypto';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'citext') THEN
        RAISE EXCEPTION 'G3 adoption requires extension citext';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        RAISE EXCEPTION 'G3 adoption requires extension pg_trgm';
    END IF;
    -- clarityit must exist as the P3 bootstrap superuser owning the extensions.
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND rolsuper) THEN
        RAISE EXCEPTION 'G3 adoption requires the P3 bootstrap superuser clarityit';
    END IF;
    IF (SELECT count(DISTINCT e.extname) FROM pg_extension e JOIN pg_roles r ON r.oid = e.extowner
        WHERE e.extname IN ('pgcrypto','citext','pg_trgm') AND r.rolname = 'clarityit') <> 3 THEN
        RAISE EXCEPTION 'G3 adoption requires clarityit to own pgcrypto, citext, and pg_trgm';
    END IF;
    -- Target identities must be absent (single-shot adoption).
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN (
        'clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin',
        'legacy_ext_owner','g3_adopt_admin')) THEN
        RAISE EXCEPTION 'G3 adoption is single-shot: a target/legacy identity already exists';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'platform') THEN
        RAISE EXCEPTION 'G3 adoption requires no existing platform schema';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='action_outcomes' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.action_outcomes';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_effect_results' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_effect_results';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_evaluation_runs' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_evaluation_runs';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_evaluation_scenario_results' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_evaluation_scenario_results';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_identities' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_identities';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_intentions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_intentions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_runs' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_runs';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='agent_tool_grants' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.agent_tool_grants';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='alerts' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.alerts';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='approval_decisions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.approval_decisions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='approval_policies' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.approval_policies';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='approval_requests' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.approval_requests';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='artifact_document_versions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.artifact_document_versions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='artifact_documents' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.artifact_documents';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='artifact_meeting_data' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.artifact_meeting_data';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='artifact_templates' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.artifact_templates';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='artifacts' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.artifacts';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='asset_actions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.asset_actions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='assets' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.assets';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='audit_logs' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.audit_logs';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='bootstrap_lock' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.bootstrap_lock';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='context_bundles' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.context_bundles';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='context_edge_evidence' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.context_edge_evidence';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='context_edges' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.context_edges';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='context_nodes' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.context_nodes';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='context_relation_reviews' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.context_relation_reviews';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='docs' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.docs';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='idempotency_keys' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.idempotency_keys';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='incidents' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.incidents';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='integration_api_keys' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.integration_api_keys';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='invitations' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.invitations';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='knowledge_chunks' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.knowledge_chunks';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='knowledge_collection_items' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.knowledge_collection_items';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='knowledge_collections' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.knowledge_collections';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='knowledge_items' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.knowledge_items';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='mfa_challenges' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.mfa_challenges';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='mfa_recovery_codes' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.mfa_recovery_codes';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='object_comments' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.object_comments';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='object_links' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.object_links';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='object_storage_refs' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.object_storage_refs';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='objects' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.objects';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='outbox_events' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.outbox_events';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='password_reset_tokens' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.password_reset_tokens';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='permissions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.permissions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='platform_roles' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.platform_roles';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='proxmox_mutation_windows' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.proxmox_mutation_windows';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='recommendation_evidence' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.recommendation_evidence';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='refresh_tokens' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.refresh_tokens';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='remediation_proposals' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.remediation_proposals';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='remediation_steps' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.remediation_steps';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='role_permissions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.role_permissions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='roles' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.roles';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='saved_knowledge_answers' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.saved_knowledge_answers';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='storage_objects' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.storage_objects';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='team_access_grants' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.team_access_grants';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='team_memberships' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.team_memberships';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='teams' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.teams';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='tool_registry' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.tool_registry';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='user_mfa_factors' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.user_mfa_factors';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='user_platform_roles' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.user_platform_roles';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='user_sessions' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.user_sessions';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='user_webauthn_credentials' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.user_webauthn_credentials';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='users' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.users';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='work_items' AND c.relkind='r') THEN
        RAISE EXCEPTION 'G3 adoption requires product table public.work_items';
    END IF;
END
$g3_adopt_preflight$;

-- ============================================================
-- Role transition (atomic, inside this transaction).
-- The bootstrap clarityit owns the extensions; rename it to a fixed
-- NOLOGIN legacy-extension owner, then create the signed target
-- clarityit.  PostgreSQL forbids renaming the current session user,
-- so switch session authorization to a temporary administrator.
-- ============================================================
CREATE ROLE g3_adopt_admin NOLOGIN SUPERUSER;
SET SESSION AUTHORIZATION g3_adopt_admin;
ALTER ROLE clarityit RENAME TO legacy_ext_owner;
ALTER ROLE legacy_ext_owner NOLOGIN;
CREATE ROLE clarityit LOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_app NOLOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_owner NOLOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_migrator LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
CREATE ROLE clarityit_admin LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

GRANT clarityit_app TO clarityit WITH INHERIT TRUE, ADMIN FALSE, SET FALSE;
GRANT clarityit_owner TO clarityit_migrator WITH INHERIT FALSE, ADMIN FALSE, SET TRUE;
-- Transfer database ownership to clarityit_owner BEFORE platform
-- creation.  This gives clarityit_owner the CREATE privilege on the
-- database as its owner, so SET LOCAL ROLE clarityit_owner can
-- create the platform schema without any temporary GRANT CREATE
-- (which would leave an unnecessary database ACL difference).
ALTER DATABASE clarityit OWNER TO clarityit_owner;

-- ============================================================
-- Platform control schema (same statements as the fresh bootstrap,
-- rendered without its own BEGIN/COMMIT so the adoption stays atomic).
-- ============================================================
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
RESET ROLE;

-- ============================================================
-- Ownership transfer: product objects + public schema to
-- clarityit_owner.  Idempotent (a no-op if already owned).
-- (Database ownership was transferred above, before platform.)
-- ============================================================
ALTER SCHEMA public OWNER TO clarityit_owner;
ALTER TABLE public.action_outcomes OWNER TO clarityit_owner;
ALTER TABLE public.agent_effect_results OWNER TO clarityit_owner;
ALTER TABLE public.agent_evaluation_runs OWNER TO clarityit_owner;
ALTER TABLE public.agent_evaluation_scenario_results OWNER TO clarityit_owner;
ALTER TABLE public.agent_identities OWNER TO clarityit_owner;
ALTER TABLE public.agent_intentions OWNER TO clarityit_owner;
ALTER TABLE public.agent_runs OWNER TO clarityit_owner;
ALTER TABLE public.agent_tool_grants OWNER TO clarityit_owner;
ALTER TABLE public.alerts OWNER TO clarityit_owner;
ALTER TABLE public.approval_decisions OWNER TO clarityit_owner;
ALTER TABLE public.approval_policies OWNER TO clarityit_owner;
ALTER TABLE public.approval_requests OWNER TO clarityit_owner;
ALTER TABLE public.artifact_document_versions OWNER TO clarityit_owner;
ALTER TABLE public.artifact_documents OWNER TO clarityit_owner;
ALTER TABLE public.artifact_meeting_data OWNER TO clarityit_owner;
ALTER TABLE public.artifact_templates OWNER TO clarityit_owner;
ALTER TABLE public.artifacts OWNER TO clarityit_owner;
ALTER TABLE public.asset_actions OWNER TO clarityit_owner;
ALTER TABLE public.assets OWNER TO clarityit_owner;
ALTER TABLE public.audit_logs OWNER TO clarityit_owner;
ALTER TABLE public.bootstrap_lock OWNER TO clarityit_owner;
ALTER TABLE public.context_bundles OWNER TO clarityit_owner;
ALTER TABLE public.context_edge_evidence OWNER TO clarityit_owner;
ALTER TABLE public.context_edges OWNER TO clarityit_owner;
ALTER TABLE public.context_nodes OWNER TO clarityit_owner;
ALTER TABLE public.context_relation_reviews OWNER TO clarityit_owner;
ALTER TABLE public.docs OWNER TO clarityit_owner;
ALTER TABLE public.idempotency_keys OWNER TO clarityit_owner;
ALTER TABLE public.incidents OWNER TO clarityit_owner;
ALTER TABLE public.integration_api_keys OWNER TO clarityit_owner;
ALTER TABLE public.invitations OWNER TO clarityit_owner;
ALTER TABLE public.knowledge_chunks OWNER TO clarityit_owner;
ALTER TABLE public.knowledge_collection_items OWNER TO clarityit_owner;
ALTER TABLE public.knowledge_collections OWNER TO clarityit_owner;
ALTER TABLE public.knowledge_items OWNER TO clarityit_owner;
ALTER TABLE public.mfa_challenges OWNER TO clarityit_owner;
ALTER TABLE public.mfa_recovery_codes OWNER TO clarityit_owner;
ALTER TABLE public.object_comments OWNER TO clarityit_owner;
ALTER TABLE public.object_links OWNER TO clarityit_owner;
ALTER TABLE public.object_storage_refs OWNER TO clarityit_owner;
ALTER TABLE public.objects OWNER TO clarityit_owner;
ALTER TABLE public.outbox_events OWNER TO clarityit_owner;
ALTER TABLE public.password_reset_tokens OWNER TO clarityit_owner;
ALTER TABLE public.permissions OWNER TO clarityit_owner;
ALTER TABLE public.platform_roles OWNER TO clarityit_owner;
ALTER TABLE public.proxmox_mutation_windows OWNER TO clarityit_owner;
ALTER TABLE public.recommendation_evidence OWNER TO clarityit_owner;
ALTER TABLE public.refresh_tokens OWNER TO clarityit_owner;
ALTER TABLE public.remediation_proposals OWNER TO clarityit_owner;
ALTER TABLE public.remediation_steps OWNER TO clarityit_owner;
ALTER TABLE public.role_permissions OWNER TO clarityit_owner;
ALTER TABLE public.roles OWNER TO clarityit_owner;
ALTER TABLE public.saved_knowledge_answers OWNER TO clarityit_owner;
ALTER TABLE public.storage_objects OWNER TO clarityit_owner;
ALTER TABLE public.team_access_grants OWNER TO clarityit_owner;
ALTER TABLE public.team_memberships OWNER TO clarityit_owner;
ALTER TABLE public.teams OWNER TO clarityit_owner;
ALTER TABLE public.tool_registry OWNER TO clarityit_owner;
ALTER TABLE public.user_mfa_factors OWNER TO clarityit_owner;
ALTER TABLE public.user_platform_roles OWNER TO clarityit_owner;
ALTER TABLE public.user_sessions OWNER TO clarityit_owner;
ALTER TABLE public.user_webauthn_credentials OWNER TO clarityit_owner;
ALTER TABLE public.users OWNER TO clarityit_owner;
ALTER TABLE public.work_items OWNER TO clarityit_owner;
ALTER SEQUENCE public.audit_logs_id_seq OWNER TO clarityit_owner;
ALTER FUNCTION public.adoc_set_updated_at() OWNER TO clarityit_owner;
ALTER FUNCTION public.kc_search_vector_update() OWNER TO clarityit_owner;
ALTER FUNCTION public.ki_search_vector_update() OWNER TO clarityit_owner;
ALTER FUNCTION public.ki_set_updated_at() OWNER TO clarityit_owner;
ALTER FUNCTION public.normalize_team_slug() OWNER TO clarityit_owner;
ALTER FUNCTION public.normalize_user_email() OWNER TO clarityit_owner;
ALTER FUNCTION public.prevent_bootstrap_unlock() OWNER TO clarityit_owner;
ALTER FUNCTION public.protect_last_team_owner() OWNER TO clarityit_owner;
ALTER FUNCTION public.set_updated_at() OWNER TO clarityit_owner;
ALTER FUNCTION public.trg_artifacts_updated_at() OWNER TO clarityit_owner;

-- ============================================================
-- Signed grants (idempotent) and default privileges.
-- ============================================================
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.action_outcomes TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_effect_results TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_evaluation_runs TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_evaluation_scenario_results TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_identities TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_intentions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_runs TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.agent_tool_grants TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.alerts TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.approval_decisions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.approval_policies TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.approval_requests TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.artifact_document_versions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.artifact_documents TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.artifact_meeting_data TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.artifact_templates TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.artifacts TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.asset_actions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.assets TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.audit_logs TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.bootstrap_lock TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.context_bundles TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.context_edge_evidence TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.context_edges TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.context_nodes TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.context_relation_reviews TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.docs TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.idempotency_keys TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.incidents TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.integration_api_keys TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.invitations TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.knowledge_chunks TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.knowledge_collection_items TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.knowledge_collections TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.knowledge_items TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.mfa_challenges TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.mfa_recovery_codes TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.object_comments TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.object_links TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.object_storage_refs TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.objects TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.outbox_events TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.password_reset_tokens TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.permissions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.platform_roles TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.proxmox_mutation_windows TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.recommendation_evidence TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.refresh_tokens TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.remediation_proposals TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.remediation_steps TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.role_permissions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.roles TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.saved_knowledge_answers TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.storage_objects TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.team_access_grants TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.team_memberships TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.teams TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.tool_registry TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.user_mfa_factors TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.user_platform_roles TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.user_sessions TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.user_webauthn_credentials TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.users TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON TABLE public.work_items TO clarityit_app;
GRANT USAGE, SELECT ON SEQUENCE public.audit_logs_id_seq TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.adoc_set_updated_at() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.adoc_set_updated_at() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.kc_search_vector_update() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.kc_search_vector_update() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.ki_search_vector_update() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.ki_search_vector_update() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.ki_set_updated_at() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.ki_set_updated_at() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.normalize_team_slug() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.normalize_team_slug() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.normalize_user_email() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.normalize_user_email() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.prevent_bootstrap_unlock() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.prevent_bootstrap_unlock() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.protect_last_team_owner() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.protect_last_team_owner() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.set_updated_at() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.set_updated_at() TO clarityit_app;
REVOKE EXECUTE ON FUNCTION public.trg_artifacts_updated_at() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.trg_artifacts_updated_at() TO clarityit_app;

ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public GRANT SELECT, INSERT, UPDATE ON TABLES TO clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO clarityit_app;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

-- ============================================================
-- Seed + adoption ledger.  Performed before the final role transition.
-- No pre-existing P3 business rows are mutated.
-- ============================================================
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
INSERT INTO platform.source_profiles (profile_id, schema_fingerprint, postgres_version, postgres_major, extensions, roles_digest, source_commit, approved_by, approved_at) VALUES (
    '7c5cb0b9-1fb4-540d-9433-f0196ff6f7bb', 'cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b', 'PostgreSQL 16', 16, '["pgcrypto","citext","pg_trgm"]'::jsonb, '2273a104fa6145ebe699ffc570da41941d49df4584ee2b093f323ce8d5a0a7c3', '29c4cdcb4c7bd9f13209f5627b55f4fabbd08a33', '3b4a6fdeb35473e5f73ca74bafa479bd2648fb10', '2026-08-03T00:00:00Z');
INSERT INTO platform.schema_revisions (version, name, checksum, source_commit, applied_at, applied_by, execution_ms, success)
VALUES ('0001', 'adopt-p3', '1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508', current_setting('g3.source_commit', true), '2026-08-03T00:00:00Z', 'g3-adoption-artifact', 0, true);
RESET ROLE;

-- ============================================================
-- End of privileged operations.  Reset session authorization from
-- the temporary administrator back to the original session identity,
-- then drop the temporary administrator.  This occurs AFTER seed and
-- ledger insertion and BEFORE the final role transition, so the
-- temporary identity exists only while privileged operations are
-- still in progress.
-- ============================================================
RESET SESSION AUTHORIZATION;
DROP ROLE g3_adopt_admin;

-- ============================================================
-- Final bootstrap-role transition (LAST state mutation).
-- Demote the new clarityit to its signed non-superuser target posture
-- only after every privileged operation and assertion has succeeded.
-- ============================================================
ALTER ROLE clarityit LOGIN INHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;

-- Final read-only assertions.
DO $g3_adopt_validate$
BEGIN
    ASSERT (SELECT count(*) FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')) = 5, 'G3 adoption role count mismatch';
    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'legacy_ext_owner' AND rolcanlogin), 'legacy_ext_owner must be NOLOGIN';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND NOT rolsuper), 'clarityit must be demoted to NOSUPERUSER';
    ASSERT (SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = current_database()) = 'clarityit_owner', 'database must be owned by clarityit_owner';
    ASSERT NOT EXISTS (
        SELECT 1 FROM pg_extension e JOIN pg_roles r ON r.oid = e.extowner
        WHERE e.extname IN ('pgcrypto','citext','pg_trgm') AND r.rolname IN ('clarityit','clarityit_owner')),        'no extension may be owned by a target role';
END
$g3_adopt_validate$;
COMMIT;
