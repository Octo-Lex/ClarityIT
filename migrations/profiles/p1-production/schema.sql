--
-- PostgreSQL database dump
--

\restrict LDGKWThi8U5JhI4JcgUhWyfrWsOSBoabry9xnnxAUMfJ7BkbSjnhh2tI164jqvE

-- Dumped from database version 16.14
-- Dumped by pg_dump version 16.14

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: adoc_set_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.adoc_set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: kc_search_vector_update(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.kc_search_vector_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.heading, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'C');
    RETURN NEW;
END;
$$;


--
-- Name: ki_search_vector_update(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.ki_search_vector_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.summary, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'C');
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;


--
-- Name: ki_set_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.ki_set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Don't override if search_vector trigger already set it
    IF NEW.updated_at = OLD.updated_at THEN
        NEW.updated_at := NOW();
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: normalize_team_slug(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.normalize_team_slug() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.slug := LOWER(TRIM(REGEXP_REPLACE(NEW.slug, '[^a-z0-9-]', '-', 'g')));
    NEW.slug := TRIM(BOTH '-' FROM NEW.slug);
    RETURN NEW;
END;
$$;


--
-- Name: normalize_user_email(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.normalize_user_email() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.email := LOWER(TRIM(NEW.email));
    RETURN NEW;
END;
$$;


--
-- Name: prevent_bootstrap_unlock(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.prevent_bootstrap_unlock() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF OLD.is_locked = TRUE AND NEW.is_locked = FALSE THEN
        RAISE EXCEPTION 'Bootstrap lock cannot be reversed';
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: protect_last_team_owner(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.protect_last_team_owner() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    owner_role_id UUID;
    admin_count INT;
BEGIN
    SELECT id INTO owner_role_id FROM roles WHERE name = 'owner';

    -- On role change away from owner
    IF TG_OP = 'UPDATE' AND OLD.role_id = owner_role_id AND NEW.role_id != owner_role_id THEN
        SELECT COUNT(*) INTO admin_count
        FROM team_memberships
        WHERE team_id = NEW.team_id AND role_id = owner_role_id;
        IF admin_count <= 1 THEN
            RAISE EXCEPTION 'Cannot remove the last owner from team';
        END IF;
    END IF;

    -- On deletion of an owner
    IF TG_OP = 'DELETE' AND OLD.role_id = owner_role_id THEN
        SELECT COUNT(*) INTO admin_count
        FROM team_memberships
        WHERE team_id = OLD.team_id AND role_id = owner_role_id;
        IF admin_count <= 1 THEN
            RAISE EXCEPTION 'Cannot remove the last owner from team';
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: set_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;


--
-- Name: trg_artifacts_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.trg_artifacts_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: action_outcomes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.action_outcomes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    asset_action_id uuid,
    remediation_proposal_id uuid,
    expected_result text,
    actual_result text,
    operator_feedback text,
    outcome_status text NOT NULL,
    follow_up_recommendation text,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT action_outcomes_outcome_status_check CHECK ((outcome_status = ANY (ARRAY['successful'::text, 'partially_successful'::text, 'failed'::text, 'inconclusive'::text]))),
    CONSTRAINT action_outcomes_source_check CHECK (((asset_action_id IS NOT NULL) OR (remediation_proposal_id IS NOT NULL)))
);


--
-- Name: TABLE action_outcomes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.action_outcomes IS 'Post-action outcome tracking for asset actions and remediation proposals (v1.2 Track 5)';


--
-- Name: agent_effect_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_effect_results (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    intention_id uuid NOT NULL,
    tool_name text NOT NULL,
    status text NOT NULL,
    approval_id uuid,
    audit_event_id uuid,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT agent_effect_results_result_payload_check CHECK ((jsonb_typeof(result) = 'object'::text)),
    CONSTRAINT agent_effect_results_status_check CHECK ((status = ANY (ARRAY['succeeded'::text, 'failed'::text, 'denied'::text, 'blocked'::text, 'cancelled'::text])))
);


--
-- Name: agent_evaluation_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_evaluation_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid,
    run_status text DEFAULT 'completed'::text NOT NULL,
    scenario_count integer DEFAULT 0 NOT NULL,
    passed_count integer DEFAULT 0 NOT NULL,
    failed_count integer DEFAULT 0 NOT NULL,
    average_score double precision DEFAULT 0.0 NOT NULL,
    safety_score double precision DEFAULT 0.0 NOT NULL,
    explainability_score double precision DEFAULT 0.0 NOT NULL,
    correctness_score double precision DEFAULT 0.0 NOT NULL,
    quality_score double precision DEFAULT 0.0 NOT NULL,
    result_summary jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    CONSTRAINT agent_evaluation_runs_average_score_check CHECK (((average_score >= (0.0)::double precision) AND (average_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_runs_correctness_score_check CHECK (((correctness_score >= (0.0)::double precision) AND (correctness_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_runs_explainability_score_check CHECK (((explainability_score >= (0.0)::double precision) AND (explainability_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_runs_quality_score_check CHECK (((quality_score >= (0.0)::double precision) AND (quality_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_runs_run_status_check CHECK ((run_status = ANY (ARRAY['running'::text, 'completed'::text, 'failed'::text]))),
    CONSTRAINT agent_evaluation_runs_safety_score_check CHECK (((safety_score >= (0.0)::double precision) AND (safety_score <= (1.0)::double precision)))
);


--
-- Name: TABLE agent_evaluation_runs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.agent_evaluation_runs IS 'v1.2 Track 7: Agent recommendation evaluation runs. Controlled golden scenarios only.';


--
-- Name: agent_evaluation_scenario_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_evaluation_scenario_results (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    run_id uuid NOT NULL,
    scenario_id text NOT NULL,
    scenario_name text NOT NULL,
    passed boolean DEFAULT false NOT NULL,
    score double precision DEFAULT 0.0 NOT NULL,
    correctness_score double precision DEFAULT 0.0 NOT NULL,
    safety_score double precision DEFAULT 0.0 NOT NULL,
    explainability_score double precision DEFAULT 0.0 NOT NULL,
    quality_score double precision DEFAULT 0.0 NOT NULL,
    expected_criteria jsonb DEFAULT '{}'::jsonb NOT NULL,
    actual_recommendation jsonb DEFAULT '{}'::jsonb NOT NULL,
    failure_reasons jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT agent_evaluation_scenario_results_correctness_score_check CHECK (((correctness_score >= (0.0)::double precision) AND (correctness_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_scenario_results_explainability_score_check CHECK (((explainability_score >= (0.0)::double precision) AND (explainability_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_scenario_results_quality_score_check CHECK (((quality_score >= (0.0)::double precision) AND (quality_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_scenario_results_safety_score_check CHECK (((safety_score >= (0.0)::double precision) AND (safety_score <= (1.0)::double precision))),
    CONSTRAINT agent_evaluation_scenario_results_score_check CHECK (((score >= (0.0)::double precision) AND (score <= (1.0)::double precision)))
);


--
-- Name: TABLE agent_evaluation_scenario_results; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.agent_evaluation_scenario_results IS 'v1.2 Track 7: Per-scenario results for evaluation runs. No live operational data.';


--
-- Name: agent_identities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_identities (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid,
    name text NOT NULL,
    agent_type text,
    status text DEFAULT 'active'::text NOT NULL,
    max_autonomy text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    description text DEFAULT ''::text,
    deleted_at timestamp with time zone,
    CONSTRAINT agent_identities_max_autonomy_level_check CHECK ((max_autonomy = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text]))),
    CONSTRAINT agent_identities_status_check CHECK ((status = ANY (ARRAY['active'::text, 'disabled'::text, 'suspended'::text])))
);


--
-- Name: agent_intentions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_intentions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    agent_run_id uuid NOT NULL,
    intention_type text NOT NULL,
    target_object_id uuid,
    tool_name text,
    confidence numeric,
    risk_level text NOT NULL,
    autonomy_level text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb,
    status text DEFAULT 'created'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    reasoning_summary text DEFAULT ''::text,
    evidence_refs jsonb DEFAULT '[]'::jsonb,
    blocked_reason text,
    approved_by uuid,
    approved_at timestamp with time zone,
    CONSTRAINT agent_intentions_autonomy_level_check CHECK ((autonomy_level = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text]))),
    CONSTRAINT agent_intentions_confidence_check CHECK (((confidence IS NULL) OR ((confidence >= (0)::numeric) AND (confidence <= (1)::numeric)))),
    CONSTRAINT agent_intentions_payload_check CHECK ((jsonb_typeof(payload) = 'object'::text)),
    CONSTRAINT agent_intentions_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))),
    CONSTRAINT agent_intentions_status_check CHECK ((status = ANY (ARRAY['proposed'::text, 'approved'::text, 'denied'::text, 'executed'::text, 'failed'::text, 'blocked'::text, 'created'::text, 'validated'::text, 'approval_requested'::text])))
);


--
-- Name: agent_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    triggered_by uuid,
    triggered_by_actor_type text,
    context_bundle_id uuid,
    status text NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    correlation_id uuid,
    error_message text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT agent_runs_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'queued'::text, 'running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text]))),
    CONSTRAINT agent_runs_triggered_by_actor_type_check CHECK ((triggered_by_actor_type = ANY (ARRAY['user'::text, 'agent'::text, 'system'::text])))
);


--
-- Name: agent_tool_grants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_tool_grants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid NOT NULL,
    team_id uuid,
    tool_name text NOT NULL,
    max_autonomy_level text NOT NULL,
    requires_approval boolean DEFAULT true NOT NULL,
    requires_mfa boolean DEFAULT false NOT NULL,
    expires_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone,
    revoked_by uuid,
    CONSTRAINT agent_tool_grants_max_autonomy_level_check CHECK ((max_autonomy_level = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text])))
);


--
-- Name: alerts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alerts (
    object_id uuid NOT NULL,
    source text NOT NULL,
    source_alert_id text,
    severity text,
    fingerprint text NOT NULL,
    first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_at timestamp with time zone,
    acknowledged_at timestamp with time zone
);


--
-- Name: approval_decisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.approval_decisions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    approval_id uuid NOT NULL,
    decided_by uuid NOT NULL,
    decision text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    mfa_verified boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT approval_decisions_decision_check CHECK ((decision = ANY (ARRAY['approved'::text, 'rejected'::text])))
);


--
-- Name: approval_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.approval_policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    name text NOT NULL,
    risk_level text NOT NULL,
    requires_mfa boolean DEFAULT true NOT NULL,
    requires_approval boolean DEFAULT true NOT NULL,
    auto_approve boolean DEFAULT false NOT NULL,
    timeout_seconds integer DEFAULT 3600 NOT NULL,
    min_approvers integer DEFAULT 1 NOT NULL,
    allow_self_approve boolean DEFAULT false NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT approval_policies_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])))
);


--
-- Name: approval_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.approval_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    action_type text NOT NULL,
    action_target jsonb DEFAULT '{}'::jsonb NOT NULL,
    risk_level text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    requested_by uuid NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    policy_id uuid,
    expires_at timestamp with time zone NOT NULL,
    executed_at timestamp with time zone,
    failure_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    expiring_notified_at timestamp with time zone,
    CONSTRAINT approval_requests_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))),
    CONSTRAINT approval_requests_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'cancelled'::text, 'expired'::text, 'executed'::text, 'failed'::text])))
);


--
-- Name: artifact_document_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifact_document_versions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    artifact_id uuid NOT NULL,
    team_id uuid NOT NULL,
    document_json jsonb NOT NULL,
    version_number integer NOT NULL,
    word_count integer DEFAULT 0 NOT NULL,
    change_summary text,
    source text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT adv_docjson_object CHECK ((jsonb_typeof(document_json) = 'object'::text)),
    CONSTRAINT adv_wordcount_nonneg CHECK ((word_count >= 0)),
    CONSTRAINT artifact_document_versions_source_check CHECK ((source = ANY (ARRAY['user_save'::text, 'agent_assisted_edit'::text, 'generated'::text, 'template'::text, 'restore'::text])))
);


--
-- Name: artifact_documents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifact_documents (
    artifact_id uuid NOT NULL,
    document_type text NOT NULL,
    document_json jsonb NOT NULL,
    schema_version integer DEFAULT 1 NOT NULL,
    word_count integer DEFAULT 0 NOT NULL,
    last_exported_storage_object_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT adoc_doc_json_object CHECK ((jsonb_typeof(document_json) = 'object'::text)),
    CONSTRAINT adoc_document_type_check CHECK ((document_type = ANY (ARRAY['general_document'::text, 'decision_memo'::text, 'implementation_plan'::text, 'incident_summary'::text, 'training_doc'::text, 'architecture_doc'::text, 'project_report'::text, 'status_report'::text, 'meeting_summary'::text, 'executive_brief'::text]))),
    CONSTRAINT adoc_schema_version CHECK ((schema_version = 1)),
    CONSTRAINT adoc_word_count_nonneg CHECK ((word_count >= 0))
);


--
-- Name: artifact_meeting_data; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifact_meeting_data (
    artifact_id uuid NOT NULL,
    meeting_date date,
    attendees jsonb DEFAULT '[]'::jsonb NOT NULL,
    agenda_items jsonb DEFAULT '[]'::jsonb NOT NULL,
    decisions jsonb DEFAULT '[]'::jsonb NOT NULL,
    action_items jsonb DEFAULT '[]'::jsonb NOT NULL,
    duration_minutes integer,
    CONSTRAINT amd_action_items_arr CHECK ((jsonb_typeof(action_items) = 'array'::text)),
    CONSTRAINT amd_agenda_arr CHECK ((jsonb_typeof(agenda_items) = 'array'::text)),
    CONSTRAINT amd_attendees_arr CHECK ((jsonb_typeof(attendees) = 'array'::text)),
    CONSTRAINT amd_decisions_arr CHECK ((jsonb_typeof(decisions) = 'array'::text)),
    CONSTRAINT amd_duration_cap CHECK (((duration_minutes IS NULL) OR (duration_minutes <= 1440))),
    CONSTRAINT amd_duration_nonneg CHECK (((duration_minutes IS NULL) OR (duration_minutes >= 0)))
);


--
-- Name: artifact_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifact_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid,
    template_type text NOT NULL,
    name text NOT NULL,
    description text,
    content_markdown text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_system boolean DEFAULT false NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    template_format text DEFAULT 'markdown'::text NOT NULL,
    document_json jsonb,
    schema_version integer,
    CONSTRAINT atpl_content_or_json CHECK ((((template_format = 'markdown'::text) AND (length(btrim(content_markdown)) > 0)) OR ((template_format = 'document_json'::text) AND (document_json IS NOT NULL)))),
    CONSTRAINT atpl_docjson_object CHECK (((template_format <> 'document_json'::text) OR (jsonb_typeof(document_json) = 'object'::text))),
    CONSTRAINT atpl_format_check CHECK ((template_format = ANY (ARRAY['markdown'::text, 'document_json'::text]))),
    CONSTRAINT atpl_metadata_object CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT atpl_name_nonempty CHECK ((length(btrim(name)) > 0)),
    CONSTRAINT atpl_schema_version_by_format CHECK ((((template_format = 'markdown'::text) AND (schema_version IS NULL)) OR ((template_format = 'document_json'::text) AND (schema_version IS NOT NULL) AND (schema_version = 1)))),
    CONSTRAINT atpl_system_no_team CHECK ((((is_system = true) AND (team_id IS NULL)) OR ((is_system = false) AND (team_id IS NOT NULL)))),
    CONSTRAINT atpl_template_type_check CHECK ((template_type = ANY (ARRAY['document'::text, 'report'::text, 'meeting_summary'::text, 'status_report'::text, 'decision_memo'::text, 'training_deck'::text, 'presentation'::text])))
);


--
-- Name: artifacts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifacts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    artifact_type text NOT NULL,
    title text NOT NULL,
    description text,
    content_markdown text,
    status text DEFAULT 'draft'::text NOT NULL,
    source_type text,
    source_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    storage_object_id uuid,
    file_format text,
    created_by uuid,
    updated_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT artifacts_artifact_type_check CHECK ((artifact_type = ANY (ARRAY['document'::text, 'report'::text, 'presentation'::text, 'meeting_summary'::text, 'status_report'::text, 'decision_memo'::text, 'training_deck'::text]))),
    CONSTRAINT artifacts_file_format_check CHECK (((file_format IS NULL) OR (file_format = ANY (ARRAY['pptx'::text, 'pdf'::text, 'md'::text])))),
    CONSTRAINT artifacts_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))),
    CONSTRAINT artifacts_title_check CHECK ((length(title) > 0))
);


--
-- Name: TABLE artifacts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.artifacts IS 'v1.3 Track 1: Team-scoped work artifacts. No operational control path.';


--
-- Name: asset_actions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.asset_actions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    asset_id uuid NOT NULL,
    action_type text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    approval_id uuid,
    requested_by uuid NOT NULL,
    proxmox_task_id text,
    result jsonb DEFAULT '{}'::jsonb,
    error_message text,
    snapshot_name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    executed_at timestamp with time zone,
    completed_at timestamp with time zone,
    CONSTRAINT asset_actions_action_type_check CHECK ((action_type = ANY (ARRAY['proxmox.start'::text, 'proxmox.shutdown'::text, 'proxmox.stop'::text, 'proxmox.snapshot'::text]))),
    CONSTRAINT asset_actions_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'executing'::text, 'succeeded'::text, 'failed'::text, 'cancelled'::text])))
);


--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.assets (
    object_id uuid NOT NULL,
    asset_type text NOT NULL,
    provider text,
    external_id text,
    hostname text,
    service_id uuid
);


--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_logs (
    id bigint NOT NULL,
    event_id uuid DEFAULT gen_random_uuid() NOT NULL,
    actor_id uuid,
    actor_type text DEFAULT 'user'::text NOT NULL,
    team_id uuid,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    old_value jsonb DEFAULT '{}'::jsonb NOT NULL,
    new_value jsonb DEFAULT '{}'::jsonb NOT NULL,
    change_summary text NOT NULL,
    ip_hmac text,
    user_agent_hmac text,
    hmac_key_id text,
    idempotency_key text,
    request_id uuid,
    correlation_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT audit_logs_actor_type_check CHECK ((actor_type = ANY (ARRAY['user'::text, 'agent'::text, 'system'::text]))),
    CONSTRAINT audit_logs_new_value_check CHECK ((jsonb_typeof(new_value) = 'object'::text)),
    CONSTRAINT audit_logs_old_value_check CHECK ((jsonb_typeof(old_value) = 'object'::text))
);


--
-- Name: audit_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.audit_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.audit_logs_id_seq OWNED BY public.audit_logs.id;


--
-- Name: bootstrap_lock; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bootstrap_lock (
    id integer DEFAULT 1 NOT NULL,
    is_locked boolean DEFAULT false NOT NULL,
    locked_by_user_id uuid,
    locked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT bootstrap_lock_id_check CHECK ((id = 1))
);


--
-- Name: context_bundles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.context_bundles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    subject_type text NOT NULL,
    subject_id uuid NOT NULL,
    requested_by_actor_id uuid,
    target_type text NOT NULL,
    dimensions text[] DEFAULT '{}'::text[] NOT NULL,
    bundle_json jsonb NOT NULL,
    freshness timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone,
    CONSTRAINT context_bundles_bundle_json_check CHECK ((jsonb_typeof(bundle_json) = 'object'::text)),
    CONSTRAINT context_bundles_target_type_check CHECK ((target_type = ANY (ARRAY['user'::text, 'agent'::text, 'view'::text])))
);


--
-- Name: context_edge_evidence; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.context_edge_evidence (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    edge_id uuid NOT NULL,
    evidence_event_id uuid NOT NULL,
    evidence_summary text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: context_edges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.context_edges (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    from_node_id uuid NOT NULL,
    to_node_id uuid NOT NULL,
    relation_type text NOT NULL,
    weight numeric DEFAULT 1.0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone,
    CONSTRAINT context_edges_check CHECK ((from_node_id <> to_node_id)),
    CONSTRAINT context_edges_weight_check CHECK ((weight >= (0)::numeric))
);


--
-- Name: context_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.context_nodes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    source text NOT NULL,
    properties jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT context_nodes_properties_check CHECK ((jsonb_typeof(properties) = 'object'::text))
);


--
-- Name: context_relation_reviews; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.context_relation_reviews (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    relation_id uuid NOT NULL,
    quality_status text NOT NULL,
    reason text,
    reviewed_by uuid,
    reviewed_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT context_relation_reviews_quality_status_check CHECK ((quality_status = ANY (ARRAY['confirmed'::text, 'dismissed'::text])))
);


--
-- Name: TABLE context_relation_reviews; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.context_relation_reviews IS 'Operator review state for context graph relations (v1.2 Track 6 — advisory only)';


--
-- Name: docs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.docs (
    object_id uuid NOT NULL,
    collection_id uuid,
    doc_type text NOT NULL,
    git_path text,
    current_version_id uuid,
    CONSTRAINT docs_doc_type_check CHECK ((doc_type = ANY (ARRAY['doc'::text, 'runbook'::text, 'postmortem'::text, 'rfc'::text, 'template'::text])))
);


--
-- Name: idempotency_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.idempotency_keys (
    scope_type text NOT NULL,
    scope_id text NOT NULL,
    key text NOT NULL,
    request_method text NOT NULL,
    request_path text NOT NULL,
    request_fingerprint text,
    status text DEFAULT 'processing'::text NOT NULL,
    response_code integer,
    response_body jsonb,
    error_code text,
    locked_until timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT idempotency_keys_check CHECK ((expires_at > created_at)),
    CONSTRAINT idempotency_keys_response_body_check CHECK (((response_body IS NULL) OR (jsonb_typeof(response_body) = 'object'::text))),
    CONSTRAINT idempotency_keys_scope_type_check CHECK ((scope_type = ANY (ARRAY['user'::text, 'anonymous'::text, 'system'::text, 'agent'::text, 'tool-gateway'::text]))),
    CONSTRAINT idempotency_keys_status_check CHECK ((status = ANY (ARRAY['processing'::text, 'completed'::text, 'failed'::text])))
);


--
-- Name: incidents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.incidents (
    object_id uuid NOT NULL,
    severity text NOT NULL,
    impact text,
    affected_service_id uuid,
    opened_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    commander_user_id uuid,
    CONSTRAINT incidents_severity_check CHECK ((severity = ANY (ARRAY['sev1'::text, 'sev2'::text, 'sev3'::text, 'sev4'::text, 'sev5'::text])))
);


--
-- Name: integration_api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.integration_api_keys (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    name text NOT NULL,
    key_hash text NOT NULL,
    key_prefix text NOT NULL,
    allowed_sources text[] DEFAULT '{}'::text[] NOT NULL,
    allowed_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    expires_at timestamp with time zone,
    revoked_at timestamp with time zone,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    signing_secret_hash text,
    allow_unsigned_dev boolean DEFAULT false NOT NULL,
    rotation_required boolean DEFAULT false NOT NULL
);


--
-- Name: invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.invitations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    email text NOT NULL,
    role_id uuid NOT NULL,
    token_hash text NOT NULL,
    invited_by uuid NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    accepted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT invitations_email_check CHECK ((TRIM(BOTH FROM email) <> ''::text))
);


--
-- Name: knowledge_chunks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.knowledge_chunks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    knowledge_item_id uuid NOT NULL,
    team_id uuid NOT NULL,
    chunk_index integer DEFAULT 0 NOT NULL,
    heading text DEFAULT ''::text NOT NULL,
    content_text text DEFAULT ''::text NOT NULL,
    content_hash text,
    token_estimate integer DEFAULT 0 NOT NULL,
    search_vector tsvector,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT kc_content_hash_format CHECK (((content_hash IS NULL) OR (content_hash ~ '^[a-f0-9]{64}$'::text))),
    CONSTRAINT kc_metadata_object CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT kc_token_estimate_nonneg CHECK ((token_estimate >= 0))
);


--
-- Name: knowledge_collection_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.knowledge_collection_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    collection_id uuid NOT NULL,
    team_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id text NOT NULL,
    knowledge_item_id uuid,
    note text,
    added_by uuid,
    added_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_item_note CHECK (((note IS NULL) OR (length(note) <= 1000)))
);


--
-- Name: knowledge_collections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.knowledge_collections (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone,
    CONSTRAINT chk_collection_desc CHECK (((description IS NULL) OR (length(description) <= 2000))),
    CONSTRAINT chk_collection_name CHECK ((length(TRIM(BOTH FROM name)) > 0))
);


--
-- Name: knowledge_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.knowledge_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    title text DEFAULT ''::text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    content_text text DEFAULT ''::text NOT NULL,
    content_hash text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    search_vector tsvector,
    visibility text DEFAULT 'team'::text NOT NULL,
    indexed_at timestamp with time zone DEFAULT now() NOT NULL,
    source_updated_at timestamp with time zone,
    stale_after timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ki_content_hash_format CHECK (((content_hash IS NULL) OR (content_hash ~ '^[a-f0-9]{64}$'::text))),
    CONSTRAINT ki_metadata_object CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT ki_source_type_check CHECK ((source_type = ANY (ARRAY['artifact'::text, 'clarity_document'::text, 'meeting_summary'::text, 'status_report'::text, 'presentation'::text, 'template'::text, 'work_item'::text, 'incident'::text, 'project'::text, 'asset'::text, 'remediation'::text, 'approval'::text, 'context_node'::text]))),
    CONSTRAINT knowledge_items_visibility_check CHECK ((visibility = ANY (ARRAY['team'::text, 'private'::text])))
);


--
-- Name: mfa_challenges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mfa_challenges (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    factor_id uuid NOT NULL,
    challenge text NOT NULL,
    verified boolean DEFAULT false NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: mfa_recovery_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mfa_recovery_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    code_hash text NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: object_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.object_comments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    object_id uuid NOT NULL,
    author_id uuid NOT NULL,
    body_markdown text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT object_comments_body_markdown_check CHECK ((TRIM(BOTH FROM body_markdown) <> ''::text))
);


--
-- Name: object_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.object_links (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    from_object_id uuid NOT NULL,
    to_object_id uuid NOT NULL,
    relation_type text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT object_links_check CHECK ((from_object_id <> to_object_id))
);


--
-- Name: object_storage_refs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.object_storage_refs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    object_id uuid NOT NULL,
    storage_object_id uuid NOT NULL,
    ref_type text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: objects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.objects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    object_type text NOT NULL,
    title text NOT NULL,
    summary text,
    status text NOT NULL,
    priority text,
    owner_user_id uuid,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    CONSTRAINT objects_priority_check CHECK ((priority = ANY (ARRAY['critical'::text, 'high'::text, 'medium'::text, 'low'::text, 'none'::text]))),
    CONSTRAINT objects_title_check CHECK ((TRIM(BOTH FROM title) <> ''::text))
);


--
-- Name: outbox_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.outbox_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type text NOT NULL,
    event_version integer DEFAULT 1 NOT NULL,
    aggregate_type text,
    aggregate_id uuid,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    sensitive_payload_ciphertext bytea,
    sensitive_payload_key_id text,
    provider_message_key text,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    max_attempts integer DEFAULT 5 NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    locked_at timestamp with time zone,
    locked_by text,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    processed_at timestamp with time zone,
    purge_after timestamp with time zone DEFAULT (now() + '7 days'::interval) NOT NULL,
    team_id uuid,
    dead_lettered_at timestamp with time zone,
    CONSTRAINT outbox_events_payload_check CHECK ((jsonb_typeof(payload) = 'object'::text)),
    CONSTRAINT outbox_events_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'sent'::text, 'failed'::text, 'dead_letter'::text])))
);


--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.password_reset_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT password_reset_tokens_check CHECK ((expires_at > created_at))
);


--
-- Name: permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.permissions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    description text,
    resource text NOT NULL,
    action text NOT NULL,
    risk_level text DEFAULT 'low'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT permissions_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])))
);


--
-- Name: platform_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.platform_roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: proxmox_mutation_windows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.proxmox_mutation_windows (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid,
    status text DEFAULT 'open'::text NOT NULL,
    reason text NOT NULL,
    opened_by uuid NOT NULL,
    opened_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    closed_by uuid,
    closed_at timestamp with time zone,
    close_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT proxmox_mutation_windows_status_check CHECK ((status = ANY (ARRAY['open'::text, 'closed'::text, 'expired'::text])))
);


--
-- Name: recommendation_evidence; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.recommendation_evidence (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    recommendation_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    recommendation_summary text DEFAULT ''::text NOT NULL,
    supporting_evidence jsonb DEFAULT '[]'::jsonb NOT NULL,
    conflicting_evidence jsonb DEFAULT '[]'::jsonb NOT NULL,
    confidence_score double precision DEFAULT 0.0 NOT NULL,
    confidence_level text DEFAULT 'low'::text NOT NULL,
    risk_notes text DEFAULT ''::text NOT NULL,
    missing_info jsonb DEFAULT '[]'::jsonb NOT NULL,
    is_stale boolean DEFAULT false NOT NULL,
    stale_after timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT recommendation_evidence_confidence_check CHECK (((confidence_score >= (0.0)::double precision) AND (confidence_score <= (1.0)::double precision))),
    CONSTRAINT recommendation_evidence_level_check CHECK ((confidence_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text]))),
    CONSTRAINT recommendation_evidence_source_type_check CHECK ((source_type = ANY (ARRAY['remediation_proposal'::text, 'incident_suggestion'::text, 'agent_recommendation'::text])))
);


--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refresh_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    session_id uuid NOT NULL,
    token_hash text NOT NULL,
    family_id uuid NOT NULL,
    replaced_by_token_id uuid,
    rotated_at timestamp with time zone,
    reuse_detected_at timestamp with time zone,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT refresh_tokens_check CHECK ((expires_at > created_at))
);


--
-- Name: remediation_proposals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.remediation_proposals (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'draft'::text NOT NULL,
    risk_level text DEFAULT 'low'::text NOT NULL,
    source text DEFAULT 'operator'::text NOT NULL,
    incident_id uuid,
    agent_run_id uuid,
    created_by uuid NOT NULL,
    approved_by uuid,
    approval_id uuid,
    idempotency_key text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    approved_at timestamp with time zone,
    completed_at timestamp with time zone,
    CONSTRAINT remediation_proposals_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))),
    CONSTRAINT remediation_proposals_source_check CHECK ((source = ANY (ARRAY['agent'::text, 'operator'::text]))),
    CONSTRAINT remediation_proposals_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'proposed'::text, 'approved'::text, 'executing'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
);


--
-- Name: remediation_steps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.remediation_steps (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    proposal_id uuid NOT NULL,
    team_id uuid NOT NULL,
    step_order integer DEFAULT 0 NOT NULL,
    tool_name text NOT NULL,
    risk_level text DEFAULT 'low'::text NOT NULL,
    parameters jsonb DEFAULT '{}'::jsonb NOT NULL,
    approval_id uuid,
    effect_result_id uuid,
    status text DEFAULT 'pending'::text NOT NULL,
    continue_on_failure boolean DEFAULT false NOT NULL,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    CONSTRAINT remediation_steps_parameters_check CHECK ((jsonb_typeof(parameters) = 'object'::text)),
    CONSTRAINT remediation_steps_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))),
    CONSTRAINT remediation_steps_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'executing'::text, 'succeeded'::text, 'failed'::text, 'skipped'::text])))
);


--
-- Name: role_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_permissions (
    role_id uuid NOT NULL,
    permission_id uuid NOT NULL
);


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    description text,
    is_system_role boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: saved_knowledge_answers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.saved_knowledge_answers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    collection_id uuid,
    question text NOT NULL,
    answer text NOT NULL,
    confidence text NOT NULL,
    sources jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_saved_answer CHECK ((length(TRIM(BOTH FROM answer)) > 0)),
    CONSTRAINT chk_saved_confidence CHECK ((confidence = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text]))),
    CONSTRAINT chk_saved_question CHECK ((length(TRIM(BOTH FROM question)) > 0)),
    CONSTRAINT chk_saved_sources CHECK ((jsonb_typeof(sources) = ANY (ARRAY['array'::text, 'object'::text])))
);


--
-- Name: storage_objects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.storage_objects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    bucket text NOT NULL,
    object_key text NOT NULL,
    content_type text,
    size_bytes bigint,
    sha256 text NOT NULL,
    encryption_status text DEFAULT 'provider_managed'::text NOT NULL,
    retention_policy text DEFAULT 'default'::text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT storage_objects_encryption_status_check CHECK ((encryption_status = ANY (ARRAY['none'::text, 'provider_managed'::text, 'app_managed'::text]))),
    CONSTRAINT storage_objects_size_bytes_check CHECK (((size_bytes IS NULL) OR (size_bytes >= 0)))
);


--
-- Name: team_access_grants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_access_grants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    team_id uuid NOT NULL,
    user_id uuid NOT NULL,
    granted_by uuid NOT NULL,
    grant_type text NOT NULL,
    scope text,
    expires_at timestamp with time zone,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id uuid,
    CONSTRAINT team_access_grants_grant_type_check CHECK ((grant_type = ANY (ARRAY['explicit'::text, 'delegated'::text, 'temporary'::text])))
);


--
-- Name: team_memberships; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_memberships (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    team_id uuid NOT NULL,
    role_id uuid NOT NULL,
    joined_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teams (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    description text,
    icon text DEFAULT '🏢'::text NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT teams_name_check CHECK ((TRIM(BOTH FROM name) <> ''::text)),
    CONSTRAINT teams_settings_check CHECK ((jsonb_typeof(settings) = 'object'::text)),
    CONSTRAINT teams_slug_check CHECK ((TRIM(BOTH FROM slug) <> ''::text))
);


--
-- Name: tool_registry; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tool_registry (
    tool_name text NOT NULL,
    display_name text NOT NULL,
    description text DEFAULT ''::text,
    risk_level text DEFAULT 'medium'::text NOT NULL,
    requires_approval boolean DEFAULT true NOT NULL,
    requires_mfa boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    CONSTRAINT tool_registry_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])))
);


--
-- Name: user_mfa_factors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_mfa_factors (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    factor_type text DEFAULT 'totp'::text NOT NULL,
    secret bytea NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    verified_at timestamp with time zone,
    disabled_at timestamp with time zone,
    failed_attempts integer DEFAULT 0 NOT NULL,
    locked_until timestamp with time zone
);


--
-- Name: user_platform_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_platform_roles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    platform_role_id uuid NOT NULL,
    granted_by uuid,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone
);


--
-- Name: user_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    ip_hmac text,
    user_agent_hmac text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    recent_mfa_at timestamp with time zone,
    CONSTRAINT user_sessions_check CHECK ((expires_at > created_at))
);


--
-- Name: user_webauthn_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_webauthn_credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    credential_id_hash text NOT NULL,
    credential_id_bytes bytea NOT NULL,
    public_key bytea NOT NULL,
    sign_count bigint DEFAULT 0 NOT NULL,
    device_type text DEFAULT ''::text NOT NULL,
    backup_eligible boolean DEFAULT false NOT NULL,
    backup_state boolean DEFAULT false NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    aaguid text DEFAULT ''::text NOT NULL,
    transports text[] DEFAULT '{}'::text[] NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone,
    disabled_at timestamp with time zone,
    CONSTRAINT user_webauthn_credentials_status_check CHECK ((status = ANY (ARRAY['active'::text, 'disabled'::text])))
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    name text NOT NULL,
    avatar_url text,
    is_active boolean DEFAULT true NOT NULL,
    token_version integer DEFAULT 1 NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT users_email_check CHECK ((TRIM(BOTH FROM email) <> ''::text)),
    CONSTRAINT users_name_check CHECK ((TRIM(BOTH FROM name) <> ''::text))
);


--
-- Name: work_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.work_items (
    object_id uuid NOT NULL,
    work_item_type text NOT NULL,
    due_at timestamp with time zone,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    sla_policy_id uuid,
    assignee_user_id uuid,
    queue_id uuid,
    project_id uuid,
    CONSTRAINT work_items_work_item_type_check CHECK ((work_item_type = ANY (ARRAY['task'::text, 'ticket'::text, 'incident'::text, 'change'::text, 'problem'::text, 'project_task'::text, 'alert_work_item'::text])))
);


--
-- Name: audit_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs ALTER COLUMN id SET DEFAULT nextval('public.audit_logs_id_seq'::regclass);


--
-- Name: action_outcomes action_outcomes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.action_outcomes
    ADD CONSTRAINT action_outcomes_pkey PRIMARY KEY (id);


--
-- Name: artifact_document_versions adv_artifact_version_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_document_versions
    ADD CONSTRAINT adv_artifact_version_unique UNIQUE (artifact_id, version_number);


--
-- Name: agent_effect_results agent_effect_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_effect_results
    ADD CONSTRAINT agent_effect_results_pkey PRIMARY KEY (id);


--
-- Name: agent_evaluation_runs agent_evaluation_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_evaluation_runs
    ADD CONSTRAINT agent_evaluation_runs_pkey PRIMARY KEY (id);


--
-- Name: agent_evaluation_scenario_results agent_evaluation_scenario_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_evaluation_scenario_results
    ADD CONSTRAINT agent_evaluation_scenario_results_pkey PRIMARY KEY (id);


--
-- Name: agent_identities agent_identities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_identities
    ADD CONSTRAINT agent_identities_pkey PRIMARY KEY (id);


--
-- Name: agent_identities agent_identities_team_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_identities
    ADD CONSTRAINT agent_identities_team_id_name_key UNIQUE (team_id, name);


--
-- Name: agent_intentions agent_intentions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_intentions
    ADD CONSTRAINT agent_intentions_pkey PRIMARY KEY (id);


--
-- Name: agent_runs agent_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_runs
    ADD CONSTRAINT agent_runs_pkey PRIMARY KEY (id);


--
-- Name: agent_tool_grants agent_tool_grants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_tool_grants
    ADD CONSTRAINT agent_tool_grants_pkey PRIMARY KEY (id);


--
-- Name: alerts alerts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_pkey PRIMARY KEY (object_id);


--
-- Name: approval_decisions approval_decisions_approval_id_decided_by_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_decisions
    ADD CONSTRAINT approval_decisions_approval_id_decided_by_key UNIQUE (approval_id, decided_by);


--
-- Name: approval_decisions approval_decisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_decisions
    ADD CONSTRAINT approval_decisions_pkey PRIMARY KEY (id);


--
-- Name: approval_policies approval_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_policies
    ADD CONSTRAINT approval_policies_pkey PRIMARY KEY (id);


--
-- Name: approval_policies approval_policies_team_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_policies
    ADD CONSTRAINT approval_policies_team_id_name_key UNIQUE (team_id, name);


--
-- Name: approval_requests approval_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_pkey PRIMARY KEY (id);


--
-- Name: artifact_document_versions artifact_document_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_document_versions
    ADD CONSTRAINT artifact_document_versions_pkey PRIMARY KEY (id);


--
-- Name: artifact_documents artifact_documents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_documents
    ADD CONSTRAINT artifact_documents_pkey PRIMARY KEY (artifact_id);


--
-- Name: artifact_meeting_data artifact_meeting_data_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_meeting_data
    ADD CONSTRAINT artifact_meeting_data_pkey PRIMARY KEY (artifact_id);


--
-- Name: artifact_templates artifact_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_templates
    ADD CONSTRAINT artifact_templates_pkey PRIMARY KEY (id);


--
-- Name: artifacts artifacts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifacts
    ADD CONSTRAINT artifacts_pkey PRIMARY KEY (id);


--
-- Name: asset_actions asset_actions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_actions
    ADD CONSTRAINT asset_actions_pkey PRIMARY KEY (id);


--
-- Name: assets assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (object_id);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id, created_at);


--
-- Name: bootstrap_lock bootstrap_lock_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bootstrap_lock
    ADD CONSTRAINT bootstrap_lock_pkey PRIMARY KEY (id);


--
-- Name: context_bundles context_bundles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_bundles
    ADD CONSTRAINT context_bundles_pkey PRIMARY KEY (id);


--
-- Name: context_edge_evidence context_edge_evidence_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_edge_evidence
    ADD CONSTRAINT context_edge_evidence_pkey PRIMARY KEY (id);


--
-- Name: context_edges context_edges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_edges
    ADD CONSTRAINT context_edges_pkey PRIMARY KEY (id);


--
-- Name: context_nodes context_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_nodes
    ADD CONSTRAINT context_nodes_pkey PRIMARY KEY (id);


--
-- Name: context_nodes context_nodes_team_id_entity_type_entity_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_nodes
    ADD CONSTRAINT context_nodes_team_id_entity_type_entity_id_key UNIQUE (team_id, entity_type, entity_id);


--
-- Name: context_relation_reviews context_relation_reviews_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_relation_reviews
    ADD CONSTRAINT context_relation_reviews_pkey PRIMARY KEY (id);


--
-- Name: docs docs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.docs
    ADD CONSTRAINT docs_pkey PRIMARY KEY (object_id);


--
-- Name: idempotency_keys idempotency_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (scope_type, scope_id, key);


--
-- Name: incidents incidents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidents
    ADD CONSTRAINT incidents_pkey PRIMARY KEY (object_id);


--
-- Name: integration_api_keys integration_api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integration_api_keys
    ADD CONSTRAINT integration_api_keys_pkey PRIMARY KEY (id);


--
-- Name: invitations invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_pkey PRIMARY KEY (id);


--
-- Name: knowledge_chunks knowledge_chunks_knowledge_item_id_chunk_index_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_chunks
    ADD CONSTRAINT knowledge_chunks_knowledge_item_id_chunk_index_key UNIQUE (knowledge_item_id, chunk_index);


--
-- Name: knowledge_chunks knowledge_chunks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_chunks
    ADD CONSTRAINT knowledge_chunks_pkey PRIMARY KEY (id);


--
-- Name: knowledge_collection_items knowledge_collection_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collection_items
    ADD CONSTRAINT knowledge_collection_items_pkey PRIMARY KEY (id);


--
-- Name: knowledge_collections knowledge_collections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collections
    ADD CONSTRAINT knowledge_collections_pkey PRIMARY KEY (id);


--
-- Name: knowledge_items knowledge_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_pkey PRIMARY KEY (id);


--
-- Name: knowledge_items knowledge_items_team_id_source_type_source_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_team_id_source_type_source_id_key UNIQUE (team_id, source_type, source_id);


--
-- Name: mfa_challenges mfa_challenges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_challenges
    ADD CONSTRAINT mfa_challenges_pkey PRIMARY KEY (id);


--
-- Name: mfa_recovery_codes mfa_recovery_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_recovery_codes
    ADD CONSTRAINT mfa_recovery_codes_pkey PRIMARY KEY (id);


--
-- Name: object_comments object_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_comments
    ADD CONSTRAINT object_comments_pkey PRIMARY KEY (id);


--
-- Name: object_links object_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_links
    ADD CONSTRAINT object_links_pkey PRIMARY KEY (id);


--
-- Name: object_storage_refs object_storage_refs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_storage_refs
    ADD CONSTRAINT object_storage_refs_pkey PRIMARY KEY (id);


--
-- Name: objects objects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.objects
    ADD CONSTRAINT objects_pkey PRIMARY KEY (id);


--
-- Name: outbox_events outbox_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbox_events
    ADD CONSTRAINT outbox_events_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_name_key UNIQUE (name);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: platform_roles platform_roles_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.platform_roles
    ADD CONSTRAINT platform_roles_name_key UNIQUE (name);


--
-- Name: platform_roles platform_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.platform_roles
    ADD CONSTRAINT platform_roles_pkey PRIMARY KEY (id);


--
-- Name: proxmox_mutation_windows proxmox_mutation_windows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.proxmox_mutation_windows
    ADD CONSTRAINT proxmox_mutation_windows_pkey PRIMARY KEY (id);


--
-- Name: recommendation_evidence recommendation_evidence_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.recommendation_evidence
    ADD CONSTRAINT recommendation_evidence_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: remediation_proposals remediation_proposals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_proposals
    ADD CONSTRAINT remediation_proposals_pkey PRIMARY KEY (id);


--
-- Name: remediation_steps remediation_steps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_steps
    ADD CONSTRAINT remediation_steps_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id);


--
-- Name: roles roles_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_name_key UNIQUE (name);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: saved_knowledge_answers saved_knowledge_answers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_knowledge_answers
    ADD CONSTRAINT saved_knowledge_answers_pkey PRIMARY KEY (id);


--
-- Name: storage_objects storage_objects_bucket_object_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.storage_objects
    ADD CONSTRAINT storage_objects_bucket_object_key_key UNIQUE (bucket, object_key);


--
-- Name: storage_objects storage_objects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.storage_objects
    ADD CONSTRAINT storage_objects_pkey PRIMARY KEY (id);


--
-- Name: team_access_grants team_access_grants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_access_grants
    ADD CONSTRAINT team_access_grants_pkey PRIMARY KEY (id);


--
-- Name: team_memberships team_memberships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_memberships
    ADD CONSTRAINT team_memberships_pkey PRIMARY KEY (id);


--
-- Name: team_memberships team_memberships_user_id_team_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_memberships
    ADD CONSTRAINT team_memberships_user_id_team_id_key UNIQUE (user_id, team_id);


--
-- Name: teams teams_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_pkey PRIMARY KEY (id);


--
-- Name: tool_registry tool_registry_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tool_registry
    ADD CONSTRAINT tool_registry_pkey PRIMARY KEY (tool_name);


--
-- Name: user_mfa_factors user_mfa_factors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_factors
    ADD CONSTRAINT user_mfa_factors_pkey PRIMARY KEY (id);


--
-- Name: user_mfa_factors user_mfa_factors_user_id_factor_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_factors
    ADD CONSTRAINT user_mfa_factors_user_id_factor_type_key UNIQUE (user_id, factor_type);


--
-- Name: user_platform_roles user_platform_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_platform_roles
    ADD CONSTRAINT user_platform_roles_pkey PRIMARY KEY (id);


--
-- Name: user_sessions user_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_pkey PRIMARY KEY (id);


--
-- Name: user_webauthn_credentials user_webauthn_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_webauthn_credentials
    ADD CONSTRAINT user_webauthn_credentials_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: work_items work_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.work_items
    ADD CONSTRAINT work_items_pkey PRIMARY KEY (object_id);


--
-- Name: idx_action_outcomes_asset_action_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_action_outcomes_asset_action_unique ON public.action_outcomes USING btree (asset_action_id) WHERE (asset_action_id IS NOT NULL);


--
-- Name: idx_action_outcomes_remediation_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_action_outcomes_remediation_unique ON public.action_outcomes USING btree (remediation_proposal_id) WHERE (remediation_proposal_id IS NOT NULL);


--
-- Name: idx_action_outcomes_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_action_outcomes_team ON public.action_outcomes USING btree (team_id);


--
-- Name: idx_adoc_document_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_adoc_document_type ON public.artifact_documents USING btree (document_type);


--
-- Name: idx_adoc_updated_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_adoc_updated_at ON public.artifact_documents USING btree (updated_at DESC);


--
-- Name: idx_adv_artifact_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_adv_artifact_version ON public.artifact_document_versions USING btree (artifact_id, version_number DESC);


--
-- Name: idx_adv_team_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_adv_team_created ON public.artifact_document_versions USING btree (team_id, created_at DESC);


--
-- Name: idx_agent_intentions_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_intentions_run ON public.agent_intentions USING btree (agent_run_id, created_at);


--
-- Name: idx_agent_runs_team_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_runs_team_status ON public.agent_runs USING btree (team_id, status, started_at DESC);


--
-- Name: idx_agent_tool_grants_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_tool_grants_agent ON public.agent_tool_grants USING btree (agent_id, tool_name);


--
-- Name: idx_amd_meeting_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_amd_meeting_date ON public.artifact_meeting_data USING btree (meeting_date);


--
-- Name: idx_approval_decisions_approval; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_decisions_approval ON public.approval_decisions USING btree (approval_id);


--
-- Name: idx_approval_requests_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_requests_status ON public.approval_requests USING btree (status);


--
-- Name: idx_approval_requests_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_approval_requests_team ON public.approval_requests USING btree (team_id);


--
-- Name: idx_artifacts_team_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_artifacts_team_status ON public.artifacts USING btree (team_id, status, updated_at DESC);


--
-- Name: idx_artifacts_team_title; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_artifacts_team_title ON public.artifacts USING gin (to_tsvector('english'::regconfig, title));


--
-- Name: idx_artifacts_team_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_artifacts_team_type ON public.artifacts USING btree (team_id, artifact_type, created_at DESC);


--
-- Name: idx_asset_actions_asset; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_asset_actions_asset ON public.asset_actions USING btree (asset_id);


--
-- Name: idx_asset_actions_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_asset_actions_status ON public.asset_actions USING btree (status);


--
-- Name: idx_asset_actions_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_asset_actions_team ON public.asset_actions USING btree (team_id);


--
-- Name: idx_atpl_system; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_atpl_system ON public.artifact_templates USING btree (is_system) WHERE (is_system = true);


--
-- Name: idx_atpl_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_atpl_team ON public.artifact_templates USING btree (team_id);


--
-- Name: idx_atpl_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_atpl_type ON public.artifact_templates USING btree (template_type);


--
-- Name: idx_audit_actor; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_actor ON public.audit_logs USING btree (actor_id) WHERE (actor_id IS NOT NULL);


--
-- Name: idx_audit_correlation; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_correlation ON public.audit_logs USING btree (correlation_id) WHERE (correlation_id IS NOT NULL);


--
-- Name: idx_audit_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_entity ON public.audit_logs USING btree (entity_type, entity_id);


--
-- Name: idx_audit_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_team ON public.audit_logs USING btree (team_id, created_at);


--
-- Name: idx_context_bundles_subject; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_context_bundles_subject ON public.context_bundles USING btree (team_id, subject_type, subject_id, freshness DESC);


--
-- Name: idx_context_edge_evidence_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_context_edge_evidence_unique ON public.context_edge_evidence USING btree (edge_id, evidence_event_id);


--
-- Name: idx_context_edges_from; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_context_edges_from ON public.context_edges USING btree (from_node_id, relation_type);


--
-- Name: idx_context_edges_to; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_context_edges_to ON public.context_edges USING btree (to_node_id, relation_type);


--
-- Name: idx_context_edges_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_context_edges_unique ON public.context_edges USING btree (team_id, from_node_id, to_node_id, relation_type);


--
-- Name: idx_context_nodes_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_context_nodes_type ON public.context_nodes USING btree (team_id, entity_type);


--
-- Name: idx_context_rel_reviews_relation_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_context_rel_reviews_relation_unique ON public.context_relation_reviews USING btree (relation_id);


--
-- Name: idx_context_rel_reviews_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_context_rel_reviews_team ON public.context_relation_reviews USING btree (team_id);


--
-- Name: idx_eval_runs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_eval_runs_created_at ON public.agent_evaluation_runs USING btree (created_at DESC);


--
-- Name: idx_eval_scenario_results_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_eval_scenario_results_run_id ON public.agent_evaluation_scenario_results USING btree (run_id);


--
-- Name: idx_idempotency_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_idempotency_expires ON public.idempotency_keys USING btree (expires_at);


--
-- Name: idx_integration_api_keys_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_integration_api_keys_team ON public.integration_api_keys USING btree (team_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_invitations_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_invitations_team ON public.invitations USING btree (team_id, created_at DESC);


--
-- Name: idx_invitations_token; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_invitations_token ON public.invitations USING btree (token_hash) WHERE (accepted_at IS NULL);


--
-- Name: idx_kc_item; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kc_item ON public.knowledge_chunks USING btree (knowledge_item_id, chunk_index);


--
-- Name: idx_kc_search_vector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kc_search_vector ON public.knowledge_chunks USING gin (search_vector);


--
-- Name: idx_kc_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kc_team ON public.knowledge_chunks USING btree (team_id, created_at DESC);


--
-- Name: idx_ki_content_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ki_content_hash ON public.knowledge_items USING btree (team_id, content_hash) WHERE (content_hash IS NOT NULL);


--
-- Name: idx_ki_search_vector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ki_search_vector ON public.knowledge_items USING gin (search_vector);


--
-- Name: idx_ki_stale; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ki_stale ON public.knowledge_items USING btree (team_id, stale_after) WHERE (stale_after IS NOT NULL);


--
-- Name: idx_ki_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ki_team ON public.knowledge_items USING btree (team_id, source_type, updated_at DESC);


--
-- Name: idx_object_comments_object; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_object_comments_object ON public.object_comments USING btree (object_id, created_at);


--
-- Name: idx_object_links_from; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_object_links_from ON public.object_links USING btree (from_object_id, relation_type);


--
-- Name: idx_object_links_to; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_object_links_to ON public.object_links USING btree (to_object_id, relation_type);


--
-- Name: idx_object_storage_refs_object; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_object_storage_refs_object ON public.object_storage_refs USING btree (object_id, ref_type);


--
-- Name: idx_objects_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_objects_status ON public.objects USING btree (team_id, status) WHERE (deleted_at IS NULL);


--
-- Name: idx_objects_team_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_objects_team_type ON public.objects USING btree (team_id, object_type) WHERE (deleted_at IS NULL);


--
-- Name: idx_outbox_processing; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_outbox_processing ON public.outbox_events USING btree (status, locked_at) WHERE (status = 'processing'::text);


--
-- Name: idx_outbox_provider_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_outbox_provider_key ON public.outbox_events USING btree (provider_message_key) WHERE (provider_message_key IS NOT NULL);


--
-- Name: idx_outbox_ready; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_outbox_ready ON public.outbox_events USING btree (status, next_attempt_at, created_at) WHERE (status = ANY (ARRAY['pending'::text, 'failed'::text]));


--
-- Name: idx_outbox_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_outbox_team_id ON public.outbox_events USING btree (team_id) WHERE (team_id IS NOT NULL);


--
-- Name: idx_password_reset_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_password_reset_user ON public.password_reset_tokens USING btree (user_id, created_at DESC);


--
-- Name: idx_proxmox_mutation_windows_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_proxmox_mutation_windows_status ON public.proxmox_mutation_windows USING btree (status);


--
-- Name: idx_proxmox_mutation_windows_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_proxmox_mutation_windows_team ON public.proxmox_mutation_windows USING btree (team_id);


--
-- Name: idx_recommendation_evidence_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_recommendation_evidence_source ON public.recommendation_evidence USING btree (team_id, source_type, source_id);


--
-- Name: idx_recommendation_evidence_team_rec; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_recommendation_evidence_team_rec ON public.recommendation_evidence USING btree (team_id, recommendation_id);


--
-- Name: idx_refresh_tokens_family; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_family ON public.refresh_tokens USING btree (family_id);


--
-- Name: idx_refresh_tokens_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_hash ON public.refresh_tokens USING btree (token_hash) WHERE (revoked_at IS NULL);


--
-- Name: idx_refresh_tokens_session; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_session ON public.refresh_tokens USING btree (session_id);


--
-- Name: idx_remediation_proposals_agent_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_remediation_proposals_agent_run ON public.remediation_proposals USING btree (agent_run_id);


--
-- Name: idx_remediation_proposals_incident; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_remediation_proposals_incident ON public.remediation_proposals USING btree (incident_id);


--
-- Name: idx_remediation_proposals_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_remediation_proposals_status ON public.remediation_proposals USING btree (status);


--
-- Name: idx_remediation_proposals_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_remediation_proposals_team ON public.remediation_proposals USING btree (team_id);


--
-- Name: idx_remediation_steps_proposal; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_remediation_steps_proposal ON public.remediation_steps USING btree (proposal_id, step_order);


--
-- Name: idx_remediation_steps_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_remediation_steps_status ON public.remediation_steps USING btree (status);


--
-- Name: idx_storage_objects_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_storage_objects_team ON public.storage_objects USING btree (team_id, created_at DESC);


--
-- Name: idx_team_access_grants_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_team_access_grants_team ON public.team_access_grants USING btree (team_id);


--
-- Name: idx_team_access_grants_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_team_access_grants_user ON public.team_access_grants USING btree (user_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_team_memberships_team; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_team_memberships_team ON public.team_memberships USING btree (team_id);


--
-- Name: idx_team_memberships_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_team_memberships_user ON public.team_memberships USING btree (user_id);


--
-- Name: idx_user_platform_roles_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_platform_roles_user ON public.user_platform_roles USING btree (user_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_user_sessions_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_sessions_active ON public.user_sessions USING btree (user_id) WHERE (revoked_at IS NULL);


--
-- Name: idx_user_sessions_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_sessions_user ON public.user_sessions USING btree (user_id, created_at DESC);


--
-- Name: idx_users_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_active ON public.users USING btree (id) WHERE (deleted_at IS NULL);


--
-- Name: idx_webauthn_cred_id_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_webauthn_cred_id_hash ON public.user_webauthn_credentials USING btree (credential_id_hash);


--
-- Name: idx_webauthn_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webauthn_status ON public.user_webauthn_credentials USING btree (status);


--
-- Name: idx_webauthn_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webauthn_user ON public.user_webauthn_credentials USING btree (user_id);


--
-- Name: uq_active_collection_name_team; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_active_collection_name_team ON public.knowledge_collections USING btree (team_id, name) WHERE (archived_at IS NULL);


--
-- Name: uq_alert_source_fingerprint; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_alert_source_fingerprint ON public.alerts USING btree (source, fingerprint);


--
-- Name: uq_collection_item_source; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_collection_item_source ON public.knowledge_collection_items USING btree (collection_id, source_type, source_id);


--
-- Name: uq_integration_api_keys_prefix; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_integration_api_keys_prefix ON public.integration_api_keys USING btree (key_prefix) WHERE (revoked_at IS NULL);


--
-- Name: uq_teams_slug_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_teams_slug_active ON public.teams USING btree (slug) WHERE (deleted_at IS NULL);


--
-- Name: uq_user_platform_role_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_user_platform_role_active ON public.user_platform_roles USING btree (user_id, platform_role_id) WHERE (revoked_at IS NULL);


--
-- Name: uq_users_email_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_users_email_active ON public.users USING btree (email) WHERE (deleted_at IS NULL);


--
-- Name: artifacts artifacts_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER artifacts_updated_at BEFORE UPDATE ON public.artifacts FOR EACH ROW EXECUTE FUNCTION public.trg_artifacts_updated_at();


--
-- Name: action_outcomes trg_action_outcomes_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_action_outcomes_updated_at BEFORE UPDATE ON public.action_outcomes FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: artifact_documents trg_adoc_set_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_adoc_set_updated_at BEFORE UPDATE ON public.artifact_documents FOR EACH ROW EXECUTE FUNCTION public.adoc_set_updated_at();


--
-- Name: agent_identities trg_agent_identities_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_agent_identities_updated_at BEFORE UPDATE ON public.agent_identities FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: bootstrap_lock trg_bootstrap_lock; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_bootstrap_lock BEFORE UPDATE ON public.bootstrap_lock FOR EACH ROW EXECUTE FUNCTION public.prevent_bootstrap_unlock();


--
-- Name: context_nodes trg_context_nodes_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_context_nodes_updated_at BEFORE UPDATE ON public.context_nodes FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: idempotency_keys trg_idempotency_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_idempotency_updated_at BEFORE UPDATE ON public.idempotency_keys FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: knowledge_chunks trg_kc_search_vector; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_kc_search_vector BEFORE INSERT OR UPDATE OF heading, content_text ON public.knowledge_chunks FOR EACH ROW EXECUTE FUNCTION public.kc_search_vector_update();


--
-- Name: knowledge_items trg_ki_search_vector; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_ki_search_vector BEFORE INSERT OR UPDATE OF title, summary, content_text ON public.knowledge_items FOR EACH ROW EXECUTE FUNCTION public.ki_search_vector_update();


--
-- Name: knowledge_items trg_ki_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_ki_updated_at BEFORE UPDATE ON public.knowledge_items FOR EACH ROW EXECUTE FUNCTION public.ki_set_updated_at();


--
-- Name: objects trg_objects_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_objects_updated_at BEFORE UPDATE ON public.objects FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: outbox_events trg_outbox_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_outbox_updated_at BEFORE UPDATE ON public.outbox_events FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: team_memberships trg_protect_last_team_owner_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_protect_last_team_owner_delete BEFORE DELETE ON public.team_memberships FOR EACH ROW EXECUTE FUNCTION public.protect_last_team_owner();


--
-- Name: team_memberships trg_protect_last_team_owner_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_protect_last_team_owner_update BEFORE UPDATE ON public.team_memberships FOR EACH ROW EXECUTE FUNCTION public.protect_last_team_owner();


--
-- Name: proxmox_mutation_windows trg_proxmox_mutation_windows_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_proxmox_mutation_windows_updated_at BEFORE UPDATE ON public.proxmox_mutation_windows FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: team_memberships trg_team_memberships_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_team_memberships_updated_at BEFORE UPDATE ON public.team_memberships FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: teams trg_teams_normalize_slug; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_teams_normalize_slug BEFORE INSERT OR UPDATE ON public.teams FOR EACH ROW EXECUTE FUNCTION public.normalize_team_slug();


--
-- Name: teams trg_teams_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_teams_updated_at BEFORE UPDATE ON public.teams FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: users trg_users_normalize_email; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_users_normalize_email BEFORE INSERT OR UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.normalize_user_email();


--
-- Name: users trg_users_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: action_outcomes action_outcomes_asset_action_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.action_outcomes
    ADD CONSTRAINT action_outcomes_asset_action_id_fkey FOREIGN KEY (asset_action_id) REFERENCES public.asset_actions(id) ON DELETE CASCADE;


--
-- Name: action_outcomes action_outcomes_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.action_outcomes
    ADD CONSTRAINT action_outcomes_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: agent_effect_results agent_effect_results_intention_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_effect_results
    ADD CONSTRAINT agent_effect_results_intention_id_fkey FOREIGN KEY (intention_id) REFERENCES public.agent_intentions(id) ON DELETE CASCADE;


--
-- Name: agent_evaluation_runs agent_evaluation_runs_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_evaluation_runs
    ADD CONSTRAINT agent_evaluation_runs_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE SET NULL;


--
-- Name: agent_evaluation_scenario_results agent_evaluation_scenario_results_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_evaluation_scenario_results
    ADD CONSTRAINT agent_evaluation_scenario_results_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.agent_evaluation_runs(id) ON DELETE CASCADE;


--
-- Name: agent_intentions agent_intentions_agent_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_intentions
    ADD CONSTRAINT agent_intentions_agent_run_id_fkey FOREIGN KEY (agent_run_id) REFERENCES public.agent_runs(id) ON DELETE CASCADE;


--
-- Name: agent_runs agent_runs_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_runs
    ADD CONSTRAINT agent_runs_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent_identities(id);


--
-- Name: agent_tool_grants agent_tool_grants_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_tool_grants
    ADD CONSTRAINT agent_tool_grants_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent_identities(id) ON DELETE CASCADE;


--
-- Name: alerts alerts_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: approval_decisions approval_decisions_approval_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_decisions
    ADD CONSTRAINT approval_decisions_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES public.approval_requests(id) ON DELETE CASCADE;


--
-- Name: approval_decisions approval_decisions_decided_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_decisions
    ADD CONSTRAINT approval_decisions_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES public.users(id);


--
-- Name: approval_policies approval_policies_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_policies
    ADD CONSTRAINT approval_policies_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: approval_requests approval_requests_policy_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES public.approval_policies(id);


--
-- Name: approval_requests approval_requests_requested_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES public.users(id);


--
-- Name: approval_requests approval_requests_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.approval_requests
    ADD CONSTRAINT approval_requests_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: artifact_document_versions artifact_document_versions_artifact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_document_versions
    ADD CONSTRAINT artifact_document_versions_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES public.artifacts(id) ON DELETE CASCADE;


--
-- Name: artifact_document_versions artifact_document_versions_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_document_versions
    ADD CONSTRAINT artifact_document_versions_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: artifact_documents artifact_documents_artifact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_documents
    ADD CONSTRAINT artifact_documents_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES public.artifacts(id) ON DELETE CASCADE;


--
-- Name: artifact_documents artifact_documents_last_exported_storage_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_documents
    ADD CONSTRAINT artifact_documents_last_exported_storage_object_id_fkey FOREIGN KEY (last_exported_storage_object_id) REFERENCES public.storage_objects(id);


--
-- Name: artifact_meeting_data artifact_meeting_data_artifact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_meeting_data
    ADD CONSTRAINT artifact_meeting_data_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES public.artifacts(id) ON DELETE CASCADE;


--
-- Name: artifact_templates artifact_templates_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_templates
    ADD CONSTRAINT artifact_templates_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: artifacts artifacts_storage_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifacts
    ADD CONSTRAINT artifacts_storage_object_id_fkey FOREIGN KEY (storage_object_id) REFERENCES public.storage_objects(id) ON DELETE SET NULL;


--
-- Name: artifacts artifacts_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifacts
    ADD CONSTRAINT artifacts_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: asset_actions asset_actions_approval_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_actions
    ADD CONSTRAINT asset_actions_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES public.approval_requests(id);


--
-- Name: asset_actions asset_actions_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_actions
    ADD CONSTRAINT asset_actions_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: asset_actions asset_actions_requested_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_actions
    ADD CONSTRAINT asset_actions_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES public.users(id);


--
-- Name: asset_actions asset_actions_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.asset_actions
    ADD CONSTRAINT asset_actions_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: assets assets_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: context_edge_evidence context_edge_evidence_edge_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_edge_evidence
    ADD CONSTRAINT context_edge_evidence_edge_id_fkey FOREIGN KEY (edge_id) REFERENCES public.context_edges(id) ON DELETE CASCADE;


--
-- Name: context_edges context_edges_from_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_edges
    ADD CONSTRAINT context_edges_from_node_id_fkey FOREIGN KEY (from_node_id) REFERENCES public.context_nodes(id) ON DELETE CASCADE;


--
-- Name: context_edges context_edges_to_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_edges
    ADD CONSTRAINT context_edges_to_node_id_fkey FOREIGN KEY (to_node_id) REFERENCES public.context_nodes(id) ON DELETE CASCADE;


--
-- Name: context_relation_reviews context_relation_reviews_relation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_relation_reviews
    ADD CONSTRAINT context_relation_reviews_relation_id_fkey FOREIGN KEY (relation_id) REFERENCES public.context_edges(id) ON DELETE CASCADE;


--
-- Name: context_relation_reviews context_relation_reviews_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_relation_reviews
    ADD CONSTRAINT context_relation_reviews_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: docs docs_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.docs
    ADD CONSTRAINT docs_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: incidents incidents_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.incidents
    ADD CONSTRAINT incidents_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: integration_api_keys integration_api_keys_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integration_api_keys
    ADD CONSTRAINT integration_api_keys_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: integration_api_keys integration_api_keys_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integration_api_keys
    ADD CONSTRAINT integration_api_keys_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: invitations invitations_invited_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_invited_by_fkey FOREIGN KEY (invited_by) REFERENCES public.users(id);


--
-- Name: invitations invitations_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: invitations invitations_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: knowledge_chunks knowledge_chunks_knowledge_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_chunks
    ADD CONSTRAINT knowledge_chunks_knowledge_item_id_fkey FOREIGN KEY (knowledge_item_id) REFERENCES public.knowledge_items(id) ON DELETE CASCADE;


--
-- Name: knowledge_chunks knowledge_chunks_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_chunks
    ADD CONSTRAINT knowledge_chunks_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: knowledge_collection_items knowledge_collection_items_added_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collection_items
    ADD CONSTRAINT knowledge_collection_items_added_by_fkey FOREIGN KEY (added_by) REFERENCES public.users(id);


--
-- Name: knowledge_collection_items knowledge_collection_items_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collection_items
    ADD CONSTRAINT knowledge_collection_items_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.knowledge_collections(id) ON DELETE CASCADE;


--
-- Name: knowledge_collection_items knowledge_collection_items_knowledge_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collection_items
    ADD CONSTRAINT knowledge_collection_items_knowledge_item_id_fkey FOREIGN KEY (knowledge_item_id) REFERENCES public.knowledge_items(id) ON DELETE SET NULL;


--
-- Name: knowledge_collection_items knowledge_collection_items_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collection_items
    ADD CONSTRAINT knowledge_collection_items_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: knowledge_collections knowledge_collections_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collections
    ADD CONSTRAINT knowledge_collections_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: knowledge_collections knowledge_collections_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_collections
    ADD CONSTRAINT knowledge_collections_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: knowledge_items knowledge_items_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: mfa_challenges mfa_challenges_factor_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_challenges
    ADD CONSTRAINT mfa_challenges_factor_id_fkey FOREIGN KEY (factor_id) REFERENCES public.user_mfa_factors(id) ON DELETE CASCADE;


--
-- Name: mfa_challenges mfa_challenges_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_challenges
    ADD CONSTRAINT mfa_challenges_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: mfa_recovery_codes mfa_recovery_codes_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mfa_recovery_codes
    ADD CONSTRAINT mfa_recovery_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: object_comments object_comments_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_comments
    ADD CONSTRAINT object_comments_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: object_links object_links_from_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_links
    ADD CONSTRAINT object_links_from_object_id_fkey FOREIGN KEY (from_object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: object_links object_links_to_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_links
    ADD CONSTRAINT object_links_to_object_id_fkey FOREIGN KEY (to_object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: object_storage_refs object_storage_refs_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_storage_refs
    ADD CONSTRAINT object_storage_refs_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- Name: object_storage_refs object_storage_refs_storage_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.object_storage_refs
    ADD CONSTRAINT object_storage_refs_storage_object_id_fkey FOREIGN KEY (storage_object_id) REFERENCES public.storage_objects(id) ON DELETE RESTRICT;


--
-- Name: password_reset_tokens password_reset_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: proxmox_mutation_windows proxmox_mutation_windows_closed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.proxmox_mutation_windows
    ADD CONSTRAINT proxmox_mutation_windows_closed_by_fkey FOREIGN KEY (closed_by) REFERENCES public.users(id);


--
-- Name: proxmox_mutation_windows proxmox_mutation_windows_opened_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.proxmox_mutation_windows
    ADD CONSTRAINT proxmox_mutation_windows_opened_by_fkey FOREIGN KEY (opened_by) REFERENCES public.users(id);


--
-- Name: proxmox_mutation_windows proxmox_mutation_windows_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.proxmox_mutation_windows
    ADD CONSTRAINT proxmox_mutation_windows_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: recommendation_evidence recommendation_evidence_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.recommendation_evidence
    ADD CONSTRAINT recommendation_evidence_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: refresh_tokens refresh_tokens_replaced_by_token_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_replaced_by_token_id_fkey FOREIGN KEY (replaced_by_token_id) REFERENCES public.refresh_tokens(id);


--
-- Name: refresh_tokens refresh_tokens_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.user_sessions(id) ON DELETE CASCADE;


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: remediation_proposals remediation_proposals_approval_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_proposals
    ADD CONSTRAINT remediation_proposals_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES public.approval_requests(id);


--
-- Name: remediation_proposals remediation_proposals_approved_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_proposals
    ADD CONSTRAINT remediation_proposals_approved_by_fkey FOREIGN KEY (approved_by) REFERENCES public.users(id);


--
-- Name: remediation_proposals remediation_proposals_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_proposals
    ADD CONSTRAINT remediation_proposals_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: remediation_proposals remediation_proposals_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_proposals
    ADD CONSTRAINT remediation_proposals_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: remediation_steps remediation_steps_approval_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_steps
    ADD CONSTRAINT remediation_steps_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES public.approval_requests(id);


--
-- Name: remediation_steps remediation_steps_effect_result_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_steps
    ADD CONSTRAINT remediation_steps_effect_result_id_fkey FOREIGN KEY (effect_result_id) REFERENCES public.agent_effect_results(id);


--
-- Name: remediation_steps remediation_steps_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_steps
    ADD CONSTRAINT remediation_steps_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.remediation_proposals(id) ON DELETE CASCADE;


--
-- Name: remediation_steps remediation_steps_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.remediation_steps
    ADD CONSTRAINT remediation_steps_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: role_permissions role_permissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE;


--
-- Name: role_permissions role_permissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: saved_knowledge_answers saved_knowledge_answers_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_knowledge_answers
    ADD CONSTRAINT saved_knowledge_answers_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.knowledge_collections(id) ON DELETE SET NULL;


--
-- Name: saved_knowledge_answers saved_knowledge_answers_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_knowledge_answers
    ADD CONSTRAINT saved_knowledge_answers_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: saved_knowledge_answers saved_knowledge_answers_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_knowledge_answers
    ADD CONSTRAINT saved_knowledge_answers_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: team_access_grants team_access_grants_granted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_access_grants
    ADD CONSTRAINT team_access_grants_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES public.users(id);


--
-- Name: team_access_grants team_access_grants_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_access_grants
    ADD CONSTRAINT team_access_grants_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: team_access_grants team_access_grants_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_access_grants
    ADD CONSTRAINT team_access_grants_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: team_access_grants team_access_grants_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_access_grants
    ADD CONSTRAINT team_access_grants_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: team_memberships team_memberships_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_memberships
    ADD CONSTRAINT team_memberships_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: team_memberships team_memberships_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_memberships
    ADD CONSTRAINT team_memberships_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: team_memberships team_memberships_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_memberships
    ADD CONSTRAINT team_memberships_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_mfa_factors user_mfa_factors_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_mfa_factors
    ADD CONSTRAINT user_mfa_factors_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_platform_roles user_platform_roles_granted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_platform_roles
    ADD CONSTRAINT user_platform_roles_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES public.users(id);


--
-- Name: user_platform_roles user_platform_roles_platform_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_platform_roles
    ADD CONSTRAINT user_platform_roles_platform_role_id_fkey FOREIGN KEY (platform_role_id) REFERENCES public.platform_roles(id) ON DELETE CASCADE;


--
-- Name: user_platform_roles user_platform_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_platform_roles
    ADD CONSTRAINT user_platform_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_sessions user_sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT user_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_webauthn_credentials user_webauthn_credentials_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_webauthn_credentials
    ADD CONSTRAINT user_webauthn_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: work_items work_items_object_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.work_items
    ADD CONSTRAINT work_items_object_id_fkey FOREIGN KEY (object_id) REFERENCES public.objects(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict LDGKWThi8U5JhI4JcgUhWyfrWsOSBoabry9xnnxAUMfJ7BkbSjnhh2tI164jqvE

