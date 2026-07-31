-- P3: ClarityIT sanitized CI legacy fixture (synthetic)
-- Deterministic; generated from P1 manifest structural metadata.
-- NO production data, identifiers, credentials, or hostnames.
-- Generated: 2026-07-31T03:11:14+00:00
-- DO NOT EDIT BY HAND — regenerate via scripts/profile/generate_p3.py

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE SEQUENCE public.audit_logs_id_seq AS bigint INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 START WITH 1;

-- App-defined functions (extension functions come via CREATE EXTENSION)
CREATE OR REPLACE FUNCTION public.adoc_set_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.kc_search_vector_update()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.heading, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'C');
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.ki_search_vector_update()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.summary, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.content_text, '')), 'C');
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.ki_set_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    -- Don't override if search_vector trigger already set it
    IF NEW.updated_at = OLD.updated_at THEN
        NEW.updated_at := NOW();
    END IF;
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.normalize_team_slug()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.slug := LOWER(TRIM(REGEXP_REPLACE(NEW.slug, '[^a-z0-9-]', '-', 'g')));
    NEW.slug := TRIM(BOTH '-' FROM NEW.slug);
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.normalize_user_email()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.email := LOWER(TRIM(NEW.email));
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.prevent_bootstrap_unlock()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    IF OLD.is_locked = TRUE AND NEW.is_locked = FALSE THEN
        RAISE EXCEPTION 'Bootstrap lock cannot be reversed';
    END IF;
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.protect_last_team_owner()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
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
$function$;
CREATE OR REPLACE FUNCTION public.set_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$function$;
CREATE OR REPLACE FUNCTION public.trg_artifacts_updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$;

CREATE TABLE public.action_outcomes (
    actual_result text,
    asset_action_id uuid,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    expected_result text,
    follow_up_recommendation text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    operator_feedback text,
    outcome_status text NOT NULL,
    remediation_proposal_id uuid,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT action_outcomes_pkey PRIMARY KEY (id),
    CONSTRAINT action_outcomes_outcome_status_check CHECK (outcome_status = ANY (ARRAY['successful'::text, 'partially_successful'::text, 'failed'::text, 'inconclusive'::text])),
    CONSTRAINT action_outcomes_source_check CHECK (asset_action_id IS NOT NULL OR remediation_proposal_id IS NOT NULL)
);

CREATE UNIQUE INDEX idx_action_outcomes_asset_action_unique ON action_outcomes USING btree (asset_action_id) WHERE asset_action_id IS NOT NULL;
CREATE UNIQUE INDEX idx_action_outcomes_remediation_unique ON action_outcomes USING btree (remediation_proposal_id) WHERE remediation_proposal_id IS NOT NULL;
CREATE INDEX idx_action_outcomes_team ON action_outcomes USING btree (team_id);

CREATE TRIGGER trg_action_outcomes_updated_at BEFORE UPDATE ON action_outcomes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.agent_effect_results (
    approval_id uuid,
    audit_event_id uuid,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    intention_id uuid NOT NULL,
    result jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL,
    team_id uuid NOT NULL,
    tool_name text NOT NULL,
    CONSTRAINT agent_effect_results_pkey PRIMARY KEY (id),
    CONSTRAINT agent_effect_results_result_payload_check CHECK (jsonb_typeof(result) = 'object'::text),
    CONSTRAINT agent_effect_results_status_check CHECK (status = ANY (ARRAY['succeeded'::text, 'failed'::text, 'denied'::text, 'blocked'::text, 'cancelled'::text]))
);


CREATE TABLE public.agent_evaluation_runs (
    average_score double precision NOT NULL DEFAULT 0.0,
    completed_at timestamp with time zone,
    correctness_score double precision NOT NULL DEFAULT 0.0,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    explainability_score double precision NOT NULL DEFAULT 0.0,
    failed_count integer NOT NULL DEFAULT 0,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    passed_count integer NOT NULL DEFAULT 0,
    quality_score double precision NOT NULL DEFAULT 0.0,
    result_summary jsonb NOT NULL DEFAULT '{}'::jsonb,
    run_status text NOT NULL DEFAULT 'completed'::text,
    safety_score double precision NOT NULL DEFAULT 0.0,
    scenario_count integer NOT NULL DEFAULT 0,
    team_id uuid,
    CONSTRAINT agent_evaluation_runs_pkey PRIMARY KEY (id),
    CONSTRAINT agent_evaluation_runs_average_score_check CHECK (average_score >= 0.0::double precision AND average_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_runs_correctness_score_check CHECK (correctness_score >= 0.0::double precision AND correctness_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_runs_explainability_score_check CHECK (explainability_score >= 0.0::double precision AND explainability_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_runs_quality_score_check CHECK (quality_score >= 0.0::double precision AND quality_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_runs_run_status_check CHECK (run_status = ANY (ARRAY['running'::text, 'completed'::text, 'failed'::text])),
    CONSTRAINT agent_evaluation_runs_safety_score_check CHECK (safety_score >= 0.0::double precision AND safety_score <= 1.0::double precision)
);

CREATE INDEX idx_eval_runs_created_at ON agent_evaluation_runs USING btree (created_at DESC);

CREATE TABLE public.agent_evaluation_scenario_results (
    actual_recommendation jsonb NOT NULL DEFAULT '{}'::jsonb,
    correctness_score double precision NOT NULL DEFAULT 0.0,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expected_criteria jsonb NOT NULL DEFAULT '{}'::jsonb,
    explainability_score double precision NOT NULL DEFAULT 0.0,
    failure_reasons jsonb NOT NULL DEFAULT '[]'::jsonb,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    passed boolean NOT NULL DEFAULT false,
    quality_score double precision NOT NULL DEFAULT 0.0,
    run_id uuid NOT NULL,
    safety_score double precision NOT NULL DEFAULT 0.0,
    scenario_id text NOT NULL,
    scenario_name text NOT NULL,
    score double precision NOT NULL DEFAULT 0.0,
    CONSTRAINT agent_evaluation_scenario_results_pkey PRIMARY KEY (id),
    CONSTRAINT agent_evaluation_scenario_results_correctness_score_check CHECK (correctness_score >= 0.0::double precision AND correctness_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_scenario_results_explainability_score_check CHECK (explainability_score >= 0.0::double precision AND explainability_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_scenario_results_quality_score_check CHECK (quality_score >= 0.0::double precision AND quality_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_scenario_results_safety_score_check CHECK (safety_score >= 0.0::double precision AND safety_score <= 1.0::double precision),
    CONSTRAINT agent_evaluation_scenario_results_score_check CHECK (score >= 0.0::double precision AND score <= 1.0::double precision)
);

CREATE INDEX idx_eval_scenario_results_run_id ON agent_evaluation_scenario_results USING btree (run_id);

CREATE TABLE public.agent_identities (
    agent_type text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    deleted_at timestamp with time zone,
    description text DEFAULT ''::text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    max_autonomy text NOT NULL,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'active'::text,
    team_id uuid,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT agent_identities_pkey PRIMARY KEY (id),
    CONSTRAINT agent_identities_team_id_name_key UNIQUE (team_id, name),
    CONSTRAINT agent_identities_max_autonomy_level_check CHECK (max_autonomy = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text])),
    CONSTRAINT agent_identities_status_check CHECK (status = ANY (ARRAY['active'::text, 'disabled'::text, 'suspended'::text]))
);


CREATE TRIGGER trg_agent_identities_updated_at BEFORE UPDATE ON agent_identities FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.agent_intentions (
    agent_run_id uuid NOT NULL,
    approved_at timestamp with time zone,
    approved_by uuid,
    autonomy_level text NOT NULL,
    blocked_reason text,
    confidence numeric,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    evidence_refs jsonb DEFAULT '[]'::jsonb,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    intention_type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb,
    reasoning_summary text DEFAULT ''::text,
    risk_level text NOT NULL,
    status text NOT NULL DEFAULT 'created'::text,
    target_object_id uuid,
    team_id uuid NOT NULL,
    tool_name text,
    CONSTRAINT agent_intentions_pkey PRIMARY KEY (id),
    CONSTRAINT agent_intentions_autonomy_level_check CHECK (autonomy_level = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text])),
    CONSTRAINT agent_intentions_confidence_check CHECK (confidence IS NULL OR confidence >= 0::numeric AND confidence <= 1::numeric),
    CONSTRAINT agent_intentions_payload_check CHECK (jsonb_typeof(payload) = 'object'::text),
    CONSTRAINT agent_intentions_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])),
    CONSTRAINT agent_intentions_status_check CHECK (status = ANY (ARRAY['proposed'::text, 'approved'::text, 'denied'::text, 'executed'::text, 'failed'::text, 'blocked'::text, 'created'::text, 'validated'::text, 'approval_requested'::text]))
);

CREATE INDEX idx_agent_intentions_run ON agent_intentions USING btree (agent_run_id, created_at);

CREATE TABLE public.agent_runs (
    agent_id uuid NOT NULL,
    completed_at timestamp with time zone,
    context_bundle_id uuid,
    correlation_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    error_message text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    started_at timestamp with time zone NOT NULL DEFAULT now(),
    status text NOT NULL,
    team_id uuid NOT NULL,
    triggered_by uuid,
    triggered_by_actor_type text,
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT agent_runs_pkey PRIMARY KEY (id),
    CONSTRAINT agent_runs_status_check CHECK (status = ANY (ARRAY['pending'::text, 'queued'::text, 'running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])),
    CONSTRAINT agent_runs_triggered_by_actor_type_check CHECK (triggered_by_actor_type = ANY (ARRAY['user'::text, 'agent'::text, 'system'::text]))
);

CREATE INDEX idx_agent_runs_team_status ON agent_runs USING btree (team_id, status, started_at DESC);

CREATE TABLE public.agent_tool_grants (
    agent_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    expires_at timestamp with time zone,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    max_autonomy_level text NOT NULL,
    requires_approval boolean NOT NULL DEFAULT true,
    requires_mfa boolean NOT NULL DEFAULT false,
    revoked_at timestamp with time zone,
    revoked_by uuid,
    team_id uuid,
    tool_name text NOT NULL,
    CONSTRAINT agent_tool_grants_pkey PRIMARY KEY (id),
    CONSTRAINT agent_tool_grants_max_autonomy_level_check CHECK (max_autonomy_level = ANY (ARRAY['A0'::text, 'A1'::text, 'A2'::text, 'A3'::text, 'A4'::text, 'A5'::text]))
);

CREATE INDEX idx_agent_tool_grants_agent ON agent_tool_grants USING btree (agent_id, tool_name);

CREATE TABLE public.alerts (
    acknowledged_at timestamp with time zone,
    fingerprint text NOT NULL,
    first_seen_at timestamp with time zone NOT NULL DEFAULT now(),
    last_seen_at timestamp with time zone,
    object_id uuid NOT NULL,
    severity text,
    source text NOT NULL,
    source_alert_id text,
    CONSTRAINT alerts_pkey PRIMARY KEY (object_id)
);

CREATE UNIQUE INDEX uq_alert_source_fingerprint ON alerts USING btree (source, fingerprint);

CREATE TABLE public.approval_decisions (
    approval_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    decided_by uuid NOT NULL,
    decision text NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    mfa_verified boolean NOT NULL DEFAULT false,
    reason text NOT NULL DEFAULT ''::text,
    CONSTRAINT approval_decisions_pkey PRIMARY KEY (id),
    CONSTRAINT approval_decisions_approval_id_decided_by_key UNIQUE (approval_id, decided_by),
    CONSTRAINT approval_decisions_decision_check CHECK (decision = ANY (ARRAY['approved'::text, 'rejected'::text]))
);

CREATE INDEX idx_approval_decisions_approval ON approval_decisions USING btree (approval_id);

CREATE TABLE public.approval_policies (
    allow_self_approve boolean NOT NULL DEFAULT false,
    auto_approve boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    is_default boolean NOT NULL DEFAULT false,
    min_approvers integer NOT NULL DEFAULT 1,
    name text NOT NULL,
    requires_approval boolean NOT NULL DEFAULT true,
    requires_mfa boolean NOT NULL DEFAULT true,
    risk_level text NOT NULL,
    team_id uuid NOT NULL,
    timeout_seconds integer NOT NULL DEFAULT 3600,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT approval_policies_pkey PRIMARY KEY (id),
    CONSTRAINT approval_policies_team_id_name_key UNIQUE (team_id, name),
    CONSTRAINT approval_policies_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))
);


CREATE TABLE public.approval_requests (
    action_target jsonb NOT NULL DEFAULT '{}'::jsonb,
    action_type text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    description text NOT NULL DEFAULT ''::text,
    executed_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    expiring_notified_at timestamp with time zone,
    failure_reason text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    policy_id uuid,
    requested_by uuid NOT NULL,
    risk_level text NOT NULL,
    status text NOT NULL DEFAULT 'pending'::text,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT approval_requests_pkey PRIMARY KEY (id),
    CONSTRAINT approval_requests_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])),
    CONSTRAINT approval_requests_status_check CHECK (status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'cancelled'::text, 'expired'::text, 'executed'::text, 'failed'::text]))
);

CREATE INDEX idx_approval_requests_status ON approval_requests USING btree (status);
CREATE INDEX idx_approval_requests_team ON approval_requests USING btree (team_id);

CREATE TABLE public.artifact_document_versions (
    artifact_id uuid NOT NULL,
    change_summary text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    document_json jsonb NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    source text NOT NULL,
    team_id uuid NOT NULL,
    version_number integer NOT NULL,
    word_count integer NOT NULL DEFAULT 0,
    CONSTRAINT artifact_document_versions_pkey PRIMARY KEY (id),
    CONSTRAINT adv_artifact_version_unique UNIQUE (artifact_id, version_number),
    CONSTRAINT adv_docjson_object CHECK (jsonb_typeof(document_json) = 'object'::text),
    CONSTRAINT adv_wordcount_nonneg CHECK (word_count >= 0),
    CONSTRAINT artifact_document_versions_source_check CHECK (source = ANY (ARRAY['user_save'::text, 'agent_assisted_edit'::text, 'generated'::text, 'template'::text, 'restore'::text]))
);

CREATE INDEX idx_adv_artifact_version ON artifact_document_versions USING btree (artifact_id, version_number DESC);
CREATE INDEX idx_adv_team_created ON artifact_document_versions USING btree (team_id, created_at DESC);

CREATE TABLE public.artifact_documents (
    artifact_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    document_json jsonb NOT NULL,
    document_type text NOT NULL,
    last_exported_storage_object_id uuid,
    schema_version integer NOT NULL DEFAULT 1,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    word_count integer NOT NULL DEFAULT 0,
    CONSTRAINT artifact_documents_pkey PRIMARY KEY (artifact_id),
    CONSTRAINT adoc_doc_json_object CHECK (jsonb_typeof(document_json) = 'object'::text),
    CONSTRAINT adoc_document_type_check CHECK (document_type = ANY (ARRAY['general_document'::text, 'decision_memo'::text, 'implementation_plan'::text, 'incident_summary'::text, 'training_doc'::text, 'architecture_doc'::text, 'project_report'::text, 'status_report'::text, 'meeting_summary'::text, 'executive_brief'::text])),
    CONSTRAINT adoc_schema_version CHECK (schema_version = 1),
    CONSTRAINT adoc_word_count_nonneg CHECK (word_count >= 0)
);

CREATE INDEX idx_adoc_document_type ON artifact_documents USING btree (document_type);
CREATE INDEX idx_adoc_updated_at ON artifact_documents USING btree (updated_at DESC);

CREATE TRIGGER trg_adoc_set_updated_at BEFORE UPDATE ON artifact_documents FOR EACH ROW EXECUTE FUNCTION adoc_set_updated_at();

CREATE TABLE public.artifact_meeting_data (
    action_items jsonb NOT NULL DEFAULT '[]'::jsonb,
    agenda_items jsonb NOT NULL DEFAULT '[]'::jsonb,
    artifact_id uuid NOT NULL,
    attendees jsonb NOT NULL DEFAULT '[]'::jsonb,
    decisions jsonb NOT NULL DEFAULT '[]'::jsonb,
    duration_minutes integer,
    meeting_date date,
    CONSTRAINT artifact_meeting_data_pkey PRIMARY KEY (artifact_id),
    CONSTRAINT amd_action_items_arr CHECK (jsonb_typeof(action_items) = 'array'::text),
    CONSTRAINT amd_agenda_arr CHECK (jsonb_typeof(agenda_items) = 'array'::text),
    CONSTRAINT amd_attendees_arr CHECK (jsonb_typeof(attendees) = 'array'::text),
    CONSTRAINT amd_decisions_arr CHECK (jsonb_typeof(decisions) = 'array'::text),
    CONSTRAINT amd_duration_cap CHECK (duration_minutes IS NULL OR duration_minutes <= 1440),
    CONSTRAINT amd_duration_nonneg CHECK (duration_minutes IS NULL OR duration_minutes >= 0)
);

CREATE INDEX idx_amd_meeting_date ON artifact_meeting_data USING btree (meeting_date);

CREATE TABLE public.artifact_templates (
    content_markdown text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    description text,
    document_json jsonb,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    is_system boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    name text NOT NULL,
    schema_version integer,
    team_id uuid,
    template_format text NOT NULL DEFAULT 'markdown'::text,
    template_type text NOT NULL,
    CONSTRAINT artifact_templates_pkey PRIMARY KEY (id),
    CONSTRAINT atpl_content_or_json CHECK (template_format = 'markdown'::text AND length(btrim(content_markdown)) > 0 OR template_format = 'document_json'::text AND document_json IS NOT NULL),
    CONSTRAINT atpl_docjson_object CHECK (template_format <> 'document_json'::text OR jsonb_typeof(document_json) = 'object'::text),
    CONSTRAINT atpl_format_check CHECK (template_format = ANY (ARRAY['markdown'::text, 'document_json'::text])),
    CONSTRAINT atpl_metadata_object CHECK (jsonb_typeof(metadata) = 'object'::text),
    CONSTRAINT atpl_name_nonempty CHECK (length(btrim(name)) > 0),
    CONSTRAINT atpl_schema_version_by_format CHECK (template_format = 'markdown'::text AND schema_version IS NULL OR template_format = 'document_json'::text AND schema_version IS NOT NULL AND schema_version = 1),
    CONSTRAINT atpl_system_no_team CHECK (is_system = true AND team_id IS NULL OR is_system = false AND team_id IS NOT NULL),
    CONSTRAINT atpl_template_type_check CHECK (template_type = ANY (ARRAY['document'::text, 'report'::text, 'meeting_summary'::text, 'status_report'::text, 'decision_memo'::text, 'training_deck'::text, 'presentation'::text]))
);

CREATE INDEX idx_atpl_system ON artifact_templates USING btree (is_system) WHERE is_system = true;
CREATE INDEX idx_atpl_team ON artifact_templates USING btree (team_id);
CREATE INDEX idx_atpl_type ON artifact_templates USING btree (template_type);

CREATE TABLE public.artifacts (
    artifact_type text NOT NULL,
    content_markdown text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    description text,
    file_format text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    source_data jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_type text,
    status text NOT NULL DEFAULT 'draft'::text,
    storage_object_id uuid,
    team_id uuid NOT NULL,
    title text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_by uuid,
    CONSTRAINT artifacts_pkey PRIMARY KEY (id),
    CONSTRAINT artifacts_artifact_type_check CHECK (artifact_type = ANY (ARRAY['document'::text, 'report'::text, 'presentation'::text, 'meeting_summary'::text, 'status_report'::text, 'decision_memo'::text, 'training_deck'::text])),
    CONSTRAINT artifacts_file_format_check CHECK (file_format IS NULL OR (file_format = ANY (ARRAY['pptx'::text, 'pdf'::text, 'md'::text]))),
    CONSTRAINT artifacts_status_check CHECK (status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text])),
    CONSTRAINT artifacts_title_check CHECK (length(title) > 0)
);

CREATE INDEX idx_artifacts_team_status ON artifacts USING btree (team_id, status, updated_at DESC);
CREATE INDEX idx_artifacts_team_title ON artifacts USING gin (to_tsvector('english'::regconfig, title));
CREATE INDEX idx_artifacts_team_type ON artifacts USING btree (team_id, artifact_type, created_at DESC);

CREATE TRIGGER artifacts_updated_at BEFORE UPDATE ON artifacts FOR EACH ROW EXECUTE FUNCTION trg_artifacts_updated_at();

CREATE TABLE public.asset_actions (
    action_type text NOT NULL,
    approval_id uuid,
    asset_id uuid NOT NULL,
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    error_message text,
    executed_at timestamp with time zone,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    proxmox_task_id text,
    requested_by uuid NOT NULL,
    result jsonb DEFAULT '{}'::jsonb,
    snapshot_name text,
    status text NOT NULL DEFAULT 'pending'::text,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT asset_actions_pkey PRIMARY KEY (id),
    CONSTRAINT asset_actions_action_type_check CHECK (action_type = ANY (ARRAY['proxmox.start'::text, 'proxmox.shutdown'::text, 'proxmox.stop'::text, 'proxmox.snapshot'::text])),
    CONSTRAINT asset_actions_status_check CHECK (status = ANY (ARRAY['pending'::text, 'approved'::text, 'executing'::text, 'succeeded'::text, 'failed'::text, 'cancelled'::text]))
);

CREATE INDEX idx_asset_actions_asset ON asset_actions USING btree (asset_id);
CREATE INDEX idx_asset_actions_status ON asset_actions USING btree (status);
CREATE INDEX idx_asset_actions_team ON asset_actions USING btree (team_id);

CREATE TABLE public.assets (
    asset_type text NOT NULL,
    external_id text,
    hostname text,
    object_id uuid NOT NULL,
    provider text,
    service_id uuid,
    CONSTRAINT assets_pkey PRIMARY KEY (object_id)
);


CREATE TABLE public.audit_logs (
    action text NOT NULL,
    actor_id uuid,
    actor_type text NOT NULL DEFAULT 'user'::text,
    change_summary text NOT NULL,
    correlation_id uuid,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    entity_id uuid NOT NULL,
    entity_type text NOT NULL,
    event_id uuid NOT NULL DEFAULT gen_random_uuid(),
    hmac_key_id text,
    id bigint NOT NULL DEFAULT nextval('audit_logs_id_seq'::regclass),
    idempotency_key text,
    ip_hmac text,
    new_value jsonb NOT NULL DEFAULT '{}'::jsonb,
    old_value jsonb NOT NULL DEFAULT '{}'::jsonb,
    request_id uuid,
    team_id uuid,
    user_agent_hmac text,
    CONSTRAINT audit_logs_pkey PRIMARY KEY (id, created_at),
    CONSTRAINT audit_logs_actor_type_check CHECK (actor_type = ANY (ARRAY['user'::text, 'agent'::text, 'system'::text])),
    CONSTRAINT audit_logs_new_value_check CHECK (jsonb_typeof(new_value) = 'object'::text),
    CONSTRAINT audit_logs_old_value_check CHECK (jsonb_typeof(old_value) = 'object'::text)
);

CREATE INDEX idx_audit_actor ON audit_logs USING btree (actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_correlation ON audit_logs USING btree (correlation_id) WHERE correlation_id IS NOT NULL;
CREATE INDEX idx_audit_entity ON audit_logs USING btree (entity_type, entity_id);
CREATE INDEX idx_audit_team ON audit_logs USING btree (team_id, created_at);

CREATE TABLE public.bootstrap_lock (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id integer NOT NULL DEFAULT 1,
    is_locked boolean NOT NULL DEFAULT false,
    locked_at timestamp with time zone,
    locked_by_user_id uuid,
    CONSTRAINT bootstrap_lock_pkey PRIMARY KEY (id),
    CONSTRAINT bootstrap_lock_id_check CHECK (id = 1)
);


CREATE TRIGGER trg_bootstrap_lock BEFORE UPDATE ON bootstrap_lock FOR EACH ROW EXECUTE FUNCTION prevent_bootstrap_unlock();

CREATE TABLE public.context_bundles (
    bundle_json jsonb NOT NULL,
    dimensions text[] NOT NULL DEFAULT '{}'::text[],
    expires_at timestamp with time zone,
    freshness timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    requested_by_actor_id uuid,
    subject_id uuid NOT NULL,
    subject_type text NOT NULL,
    target_type text NOT NULL,
    team_id uuid NOT NULL,
    CONSTRAINT context_bundles_pkey PRIMARY KEY (id),
    CONSTRAINT context_bundles_bundle_json_check CHECK (jsonb_typeof(bundle_json) = 'object'::text),
    CONSTRAINT context_bundles_target_type_check CHECK (target_type = ANY (ARRAY['user'::text, 'agent'::text, 'view'::text]))
);

CREATE INDEX idx_context_bundles_subject ON context_bundles USING btree (team_id, subject_type, subject_id, freshness DESC);

CREATE TABLE public.context_edge_evidence (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    edge_id uuid NOT NULL,
    evidence_event_id uuid NOT NULL,
    evidence_summary text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    CONSTRAINT context_edge_evidence_pkey PRIMARY KEY (id)
);

CREATE UNIQUE INDEX idx_context_edge_evidence_unique ON context_edge_evidence USING btree (edge_id, evidence_event_id);

CREATE TABLE public.context_edges (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone,
    from_node_id uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    relation_type text NOT NULL,
    team_id uuid NOT NULL,
    to_node_id uuid NOT NULL,
    weight numeric NOT NULL DEFAULT 1.0,
    CONSTRAINT context_edges_pkey PRIMARY KEY (id),
    CONSTRAINT context_edges_check CHECK (from_node_id <> to_node_id),
    CONSTRAINT context_edges_weight_check CHECK (weight >= 0::numeric)
);

CREATE INDEX idx_context_edges_from ON context_edges USING btree (from_node_id, relation_type);
CREATE INDEX idx_context_edges_to ON context_edges USING btree (to_node_id, relation_type);
CREATE UNIQUE INDEX idx_context_edges_unique ON context_edges USING btree (team_id, from_node_id, to_node_id, relation_type);

CREATE TABLE public.context_nodes (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    entity_id uuid NOT NULL,
    entity_type text NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    properties jsonb NOT NULL DEFAULT '{}'::jsonb,
    source text NOT NULL,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT context_nodes_pkey PRIMARY KEY (id),
    CONSTRAINT context_nodes_team_id_entity_type_entity_id_key UNIQUE (team_id, entity_type, entity_id),
    CONSTRAINT context_nodes_properties_check CHECK (jsonb_typeof(properties) = 'object'::text)
);

CREATE INDEX idx_context_nodes_type ON context_nodes USING btree (team_id, entity_type);

CREATE TRIGGER trg_context_nodes_updated_at BEFORE UPDATE ON context_nodes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.context_relation_reviews (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    quality_status text NOT NULL,
    reason text,
    relation_id uuid NOT NULL,
    reviewed_at timestamp with time zone NOT NULL DEFAULT now(),
    reviewed_by uuid,
    team_id uuid NOT NULL,
    CONSTRAINT context_relation_reviews_pkey PRIMARY KEY (id),
    CONSTRAINT context_relation_reviews_quality_status_check CHECK (quality_status = ANY (ARRAY['confirmed'::text, 'dismissed'::text]))
);

CREATE UNIQUE INDEX idx_context_rel_reviews_relation_unique ON context_relation_reviews USING btree (relation_id);
CREATE INDEX idx_context_rel_reviews_team ON context_relation_reviews USING btree (team_id);

CREATE TABLE public.docs (
    collection_id uuid,
    current_version_id uuid,
    doc_type text NOT NULL,
    git_path text,
    object_id uuid NOT NULL,
    CONSTRAINT docs_pkey PRIMARY KEY (object_id),
    CONSTRAINT docs_doc_type_check CHECK (doc_type = ANY (ARRAY['doc'::text, 'runbook'::text, 'postmortem'::text, 'rfc'::text, 'template'::text]))
);


CREATE TABLE public.idempotency_keys (
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    error_code text,
    expires_at timestamp with time zone NOT NULL,
    key text NOT NULL,
    locked_until timestamp with time zone,
    request_fingerprint text,
    request_method text NOT NULL,
    request_path text NOT NULL,
    response_body jsonb,
    response_code integer,
    scope_id text NOT NULL,
    scope_type text NOT NULL,
    status text NOT NULL DEFAULT 'processing'::text,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT idempotency_keys_pkey PRIMARY KEY (scope_type, scope_id, key),
    CONSTRAINT idempotency_keys_check CHECK (expires_at > created_at),
    CONSTRAINT idempotency_keys_response_body_check CHECK (response_body IS NULL OR jsonb_typeof(response_body) = 'object'::text),
    CONSTRAINT idempotency_keys_scope_type_check CHECK (scope_type = ANY (ARRAY['user'::text, 'anonymous'::text, 'system'::text, 'agent'::text, 'tool-gateway'::text])),
    CONSTRAINT idempotency_keys_status_check CHECK (status = ANY (ARRAY['processing'::text, 'completed'::text, 'failed'::text]))
);

CREATE INDEX idx_idempotency_expires ON idempotency_keys USING btree (expires_at);

CREATE TRIGGER trg_idempotency_updated_at BEFORE UPDATE ON idempotency_keys FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.incidents (
    affected_service_id uuid,
    commander_user_id uuid,
    impact text,
    object_id uuid NOT NULL,
    opened_at timestamp with time zone NOT NULL DEFAULT now(),
    resolved_at timestamp with time zone,
    severity text NOT NULL,
    CONSTRAINT incidents_pkey PRIMARY KEY (object_id),
    CONSTRAINT incidents_severity_check CHECK (severity = ANY (ARRAY['sev1'::text, 'sev2'::text, 'sev3'::text, 'sev4'::text, 'sev5'::text]))
);


CREATE TABLE public.integration_api_keys (
    allow_unsigned_dev boolean NOT NULL DEFAULT false,
    allowed_scopes text[] NOT NULL DEFAULT '{}'::text[],
    allowed_sources text[] NOT NULL DEFAULT '{}'::text[],
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid NOT NULL,
    expires_at timestamp with time zone,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    key_hash text NOT NULL,
    key_prefix text NOT NULL,
    name text NOT NULL,
    revoked_at timestamp with time zone,
    rotation_required boolean NOT NULL DEFAULT false,
    signing_secret_hash text,
    team_id uuid NOT NULL,
    CONSTRAINT integration_api_keys_pkey PRIMARY KEY (id)
);

CREATE INDEX idx_integration_api_keys_team ON integration_api_keys USING btree (team_id) WHERE revoked_at IS NULL;
CREATE UNIQUE INDEX uq_integration_api_keys_prefix ON integration_api_keys USING btree (key_prefix) WHERE revoked_at IS NULL;

CREATE TABLE public.invitations (
    accepted_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    email text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    invited_by uuid NOT NULL,
    role_id uuid NOT NULL,
    team_id uuid NOT NULL,
    token_hash text NOT NULL,
    CONSTRAINT invitations_pkey PRIMARY KEY (id),
    CONSTRAINT invitations_email_check CHECK (TRIM(BOTH FROM email) <> ''::text)
);

CREATE INDEX idx_invitations_team ON invitations USING btree (team_id, created_at DESC);
CREATE INDEX idx_invitations_token ON invitations USING btree (token_hash) WHERE accepted_at IS NULL;

CREATE TABLE public.knowledge_chunks (
    chunk_index integer NOT NULL DEFAULT 0,
    content_hash text,
    content_text text NOT NULL DEFAULT ''::text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    heading text NOT NULL DEFAULT ''::text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    knowledge_item_id uuid NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    search_vector tsvector,
    team_id uuid NOT NULL,
    token_estimate integer NOT NULL DEFAULT 0,
    CONSTRAINT knowledge_chunks_pkey PRIMARY KEY (id),
    CONSTRAINT knowledge_chunks_knowledge_item_id_chunk_index_key UNIQUE (knowledge_item_id, chunk_index),
    CONSTRAINT kc_content_hash_format CHECK (content_hash IS NULL OR content_hash ~ '^[a-f0-9]{64}$'::text),
    CONSTRAINT kc_metadata_object CHECK (jsonb_typeof(metadata) = 'object'::text),
    CONSTRAINT kc_token_estimate_nonneg CHECK (token_estimate >= 0)
);

CREATE INDEX idx_kc_item ON knowledge_chunks USING btree (knowledge_item_id, chunk_index);
CREATE INDEX idx_kc_search_vector ON knowledge_chunks USING gin (search_vector);
CREATE INDEX idx_kc_team ON knowledge_chunks USING btree (team_id, created_at DESC);

CREATE TRIGGER trg_kc_search_vector BEFORE INSERT OR UPDATE OF heading, content_text ON knowledge_chunks FOR EACH ROW EXECUTE FUNCTION kc_search_vector_update();

CREATE TABLE public.knowledge_collection_items (
    added_at timestamp with time zone NOT NULL DEFAULT now(),
    added_by uuid,
    collection_id uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    knowledge_item_id uuid,
    note text,
    source_id text NOT NULL,
    source_type text NOT NULL,
    team_id uuid NOT NULL,
    CONSTRAINT knowledge_collection_items_pkey PRIMARY KEY (id),
    CONSTRAINT chk_item_note CHECK (note IS NULL OR length(note) <= 1000)
);

CREATE UNIQUE INDEX uq_collection_item_source ON knowledge_collection_items USING btree (collection_id, source_type, source_id);

CREATE TABLE public.knowledge_collections (
    archived_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    description text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name text NOT NULL,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT knowledge_collections_pkey PRIMARY KEY (id),
    CONSTRAINT chk_collection_desc CHECK (description IS NULL OR length(description) <= 2000),
    CONSTRAINT chk_collection_name CHECK (length(TRIM(BOTH FROM name)) > 0)
);

CREATE UNIQUE INDEX uq_active_collection_name_team ON knowledge_collections USING btree (team_id, name) WHERE archived_at IS NULL;

CREATE TABLE public.knowledge_items (
    content_hash text,
    content_text text NOT NULL DEFAULT ''::text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    indexed_at timestamp with time zone NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    search_vector tsvector,
    source_id uuid NOT NULL,
    source_type text NOT NULL,
    source_updated_at timestamp with time zone,
    stale_after timestamp with time zone,
    summary text NOT NULL DEFAULT ''::text,
    team_id uuid NOT NULL,
    title text NOT NULL DEFAULT ''::text,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    visibility text NOT NULL DEFAULT 'team'::text,
    CONSTRAINT knowledge_items_pkey PRIMARY KEY (id),
    CONSTRAINT knowledge_items_team_id_source_type_source_id_key UNIQUE (team_id, source_type, source_id),
    CONSTRAINT ki_content_hash_format CHECK (content_hash IS NULL OR content_hash ~ '^[a-f0-9]{64}$'::text),
    CONSTRAINT ki_metadata_object CHECK (jsonb_typeof(metadata) = 'object'::text),
    CONSTRAINT ki_source_type_check CHECK (source_type = ANY (ARRAY['artifact'::text, 'clarity_document'::text, 'meeting_summary'::text, 'status_report'::text, 'presentation'::text, 'template'::text, 'work_item'::text, 'incident'::text, 'project'::text, 'asset'::text, 'remediation'::text, 'approval'::text, 'context_node'::text])),
    CONSTRAINT knowledge_items_visibility_check CHECK (visibility = ANY (ARRAY['team'::text, 'private'::text]))
);

CREATE INDEX idx_ki_content_hash ON knowledge_items USING btree (team_id, content_hash) WHERE content_hash IS NOT NULL;
CREATE INDEX idx_ki_search_vector ON knowledge_items USING gin (search_vector);
CREATE INDEX idx_ki_stale ON knowledge_items USING btree (team_id, stale_after) WHERE stale_after IS NOT NULL;
CREATE INDEX idx_ki_team ON knowledge_items USING btree (team_id, source_type, updated_at DESC);

CREATE TRIGGER trg_ki_search_vector BEFORE INSERT OR UPDATE OF title, summary, content_text ON knowledge_items FOR EACH ROW EXECUTE FUNCTION ki_search_vector_update();
CREATE TRIGGER trg_ki_updated_at BEFORE UPDATE ON knowledge_items FOR EACH ROW EXECUTE FUNCTION ki_set_updated_at();

CREATE TABLE public.mfa_challenges (
    challenge text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone NOT NULL,
    factor_id uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    verified boolean NOT NULL DEFAULT false,
    CONSTRAINT mfa_challenges_pkey PRIMARY KEY (id)
);


CREATE TABLE public.mfa_recovery_codes (
    code_hash text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    used_at timestamp with time zone,
    user_id uuid NOT NULL,
    CONSTRAINT mfa_recovery_codes_pkey PRIMARY KEY (id)
);


CREATE TABLE public.object_comments (
    author_id uuid NOT NULL,
    body_markdown text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    object_id uuid NOT NULL,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT object_comments_pkey PRIMARY KEY (id),
    CONSTRAINT object_comments_body_markdown_check CHECK (TRIM(BOTH FROM body_markdown) <> ''::text)
);

CREATE INDEX idx_object_comments_object ON object_comments USING btree (object_id, created_at);

CREATE TABLE public.object_links (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    from_object_id uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    relation_type text NOT NULL,
    team_id uuid NOT NULL,
    to_object_id uuid NOT NULL,
    CONSTRAINT object_links_pkey PRIMARY KEY (id),
    CONSTRAINT object_links_check CHECK (from_object_id <> to_object_id)
);

CREATE INDEX idx_object_links_from ON object_links USING btree (from_object_id, relation_type);
CREATE INDEX idx_object_links_to ON object_links USING btree (to_object_id, relation_type);

CREATE TABLE public.object_storage_refs (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    object_id uuid NOT NULL,
    ref_type text NOT NULL,
    storage_object_id uuid NOT NULL,
    team_id uuid NOT NULL,
    CONSTRAINT object_storage_refs_pkey PRIMARY KEY (id)
);

CREATE INDEX idx_object_storage_refs_object ON object_storage_refs USING btree (object_id, ref_type);

CREATE TABLE public.objects (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    deleted_at timestamp with time zone,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    metadata jsonb DEFAULT '{}'::jsonb,
    object_type text NOT NULL,
    owner_user_id uuid,
    priority text,
    status text NOT NULL,
    summary text,
    team_id uuid NOT NULL,
    title text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    version integer NOT NULL DEFAULT 1,
    CONSTRAINT objects_pkey PRIMARY KEY (id),
    CONSTRAINT objects_priority_check CHECK (priority = ANY (ARRAY['critical'::text, 'high'::text, 'medium'::text, 'low'::text, 'none'::text])),
    CONSTRAINT objects_title_check CHECK (TRIM(BOTH FROM title) <> ''::text)
);

CREATE INDEX idx_objects_status ON objects USING btree (team_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_objects_team_type ON objects USING btree (team_id, object_type) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_objects_updated_at BEFORE UPDATE ON objects FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.outbox_events (
    aggregate_id uuid,
    aggregate_type text,
    attempts integer NOT NULL DEFAULT 0,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    dead_lettered_at timestamp with time zone,
    event_type text NOT NULL,
    event_version integer NOT NULL DEFAULT 1,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    last_error text,
    locked_at timestamp with time zone,
    locked_by text,
    max_attempts integer NOT NULL DEFAULT 5,
    next_attempt_at timestamp with time zone NOT NULL DEFAULT now(),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    processed_at timestamp with time zone,
    provider_message_key text,
    purge_after timestamp with time zone NOT NULL DEFAULT (now() + '7 days'::interval),
    sensitive_payload_ciphertext bytea,
    sensitive_payload_key_id text,
    status text NOT NULL DEFAULT 'pending'::text,
    team_id uuid,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT outbox_events_pkey PRIMARY KEY (id),
    CONSTRAINT outbox_events_payload_check CHECK (jsonb_typeof(payload) = 'object'::text),
    CONSTRAINT outbox_events_status_check CHECK (status = ANY (ARRAY['pending'::text, 'processing'::text, 'sent'::text, 'failed'::text, 'dead_letter'::text]))
);

CREATE INDEX idx_outbox_processing ON outbox_events USING btree (status, locked_at) WHERE status = 'processing'::text;
CREATE INDEX idx_outbox_provider_key ON outbox_events USING btree (provider_message_key) WHERE provider_message_key IS NOT NULL;
CREATE INDEX idx_outbox_ready ON outbox_events USING btree (status, next_attempt_at, created_at) WHERE status = ANY (ARRAY['pending'::text, 'failed'::text]);
CREATE INDEX idx_outbox_team_id ON outbox_events USING btree (team_id) WHERE team_id IS NOT NULL;

CREATE TRIGGER trg_outbox_updated_at BEFORE UPDATE ON outbox_events FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.password_reset_tokens (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    token_hash text NOT NULL,
    used_at timestamp with time zone,
    user_id uuid NOT NULL,
    CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT password_reset_tokens_check CHECK (expires_at > created_at)
);

CREATE INDEX idx_password_reset_user ON password_reset_tokens USING btree (user_id, created_at DESC);

CREATE TABLE public.permissions (
    action text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    description text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name text NOT NULL,
    resource text NOT NULL,
    risk_level text NOT NULL DEFAULT 'low'::text,
    CONSTRAINT permissions_pkey PRIMARY KEY (id),
    CONSTRAINT permissions_name_key UNIQUE (name),
    CONSTRAINT permissions_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))
);


CREATE TABLE public.platform_roles (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    description text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name text NOT NULL,
    CONSTRAINT platform_roles_pkey PRIMARY KEY (id),
    CONSTRAINT platform_roles_name_key UNIQUE (name)
);


CREATE TABLE public.proxmox_mutation_windows (
    close_reason text,
    closed_at timestamp with time zone,
    closed_by uuid,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    opened_at timestamp with time zone NOT NULL DEFAULT now(),
    opened_by uuid NOT NULL,
    reason text NOT NULL,
    status text NOT NULL DEFAULT 'open'::text,
    team_id uuid,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT proxmox_mutation_windows_pkey PRIMARY KEY (id),
    CONSTRAINT proxmox_mutation_windows_status_check CHECK (status = ANY (ARRAY['open'::text, 'closed'::text, 'expired'::text]))
);

CREATE INDEX idx_proxmox_mutation_windows_status ON proxmox_mutation_windows USING btree (status);
CREATE INDEX idx_proxmox_mutation_windows_team ON proxmox_mutation_windows USING btree (team_id);

CREATE TRIGGER trg_proxmox_mutation_windows_updated_at BEFORE UPDATE ON proxmox_mutation_windows FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.recommendation_evidence (
    confidence_level text NOT NULL DEFAULT 'low'::text,
    confidence_score double precision NOT NULL DEFAULT 0.0,
    conflicting_evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    is_stale boolean NOT NULL DEFAULT false,
    missing_info jsonb NOT NULL DEFAULT '[]'::jsonb,
    recommendation_id uuid NOT NULL,
    recommendation_summary text NOT NULL DEFAULT ''::text,
    risk_notes text NOT NULL DEFAULT ''::text,
    source_id uuid NOT NULL,
    source_type text NOT NULL,
    stale_after timestamp with time zone,
    supporting_evidence jsonb NOT NULL DEFAULT '[]'::jsonb,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT recommendation_evidence_pkey PRIMARY KEY (id),
    CONSTRAINT recommendation_evidence_confidence_check CHECK (confidence_score >= 0.0::double precision AND confidence_score <= 1.0::double precision),
    CONSTRAINT recommendation_evidence_level_check CHECK (confidence_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text])),
    CONSTRAINT recommendation_evidence_source_type_check CHECK (source_type = ANY (ARRAY['remediation_proposal'::text, 'incident_suggestion'::text, 'agent_recommendation'::text]))
);

CREATE INDEX idx_recommendation_evidence_source ON recommendation_evidence USING btree (team_id, source_type, source_id);
CREATE INDEX idx_recommendation_evidence_team_rec ON recommendation_evidence USING btree (team_id, recommendation_id);

CREATE TABLE public.refresh_tokens (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone NOT NULL,
    family_id uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    replaced_by_token_id uuid,
    reuse_detected_at timestamp with time zone,
    revoked_at timestamp with time zone,
    rotated_at timestamp with time zone,
    session_id uuid NOT NULL,
    token_hash text NOT NULL,
    user_id uuid NOT NULL,
    CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT refresh_tokens_check CHECK (expires_at > created_at)
);

CREATE INDEX idx_refresh_tokens_family ON refresh_tokens USING btree (family_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens USING btree (token_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_session ON refresh_tokens USING btree (session_id);

CREATE TABLE public.remediation_proposals (
    agent_run_id uuid,
    approval_id uuid,
    approved_at timestamp with time zone,
    approved_by uuid,
    completed_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid NOT NULL,
    description text NOT NULL DEFAULT ''::text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    idempotency_key text,
    incident_id uuid,
    risk_level text NOT NULL DEFAULT 'low'::text,
    source text NOT NULL DEFAULT 'operator'::text,
    status text NOT NULL DEFAULT 'draft'::text,
    team_id uuid NOT NULL,
    title text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT remediation_proposals_pkey PRIMARY KEY (id),
    CONSTRAINT remediation_proposals_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])),
    CONSTRAINT remediation_proposals_source_check CHECK (source = ANY (ARRAY['agent'::text, 'operator'::text])),
    CONSTRAINT remediation_proposals_status_check CHECK (status = ANY (ARRAY['draft'::text, 'proposed'::text, 'approved'::text, 'executing'::text, 'completed'::text, 'failed'::text, 'cancelled'::text]))
);

CREATE INDEX idx_remediation_proposals_agent_run ON remediation_proposals USING btree (agent_run_id);
CREATE INDEX idx_remediation_proposals_incident ON remediation_proposals USING btree (incident_id);
CREATE INDEX idx_remediation_proposals_status ON remediation_proposals USING btree (status);
CREATE INDEX idx_remediation_proposals_team ON remediation_proposals USING btree (team_id);

CREATE TABLE public.remediation_steps (
    approval_id uuid,
    completed_at timestamp with time zone,
    continue_on_failure boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    effect_result_id uuid,
    error_message text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
    proposal_id uuid NOT NULL,
    risk_level text NOT NULL DEFAULT 'low'::text,
    started_at timestamp with time zone,
    status text NOT NULL DEFAULT 'pending'::text,
    step_order integer NOT NULL DEFAULT 0,
    team_id uuid NOT NULL,
    tool_name text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT remediation_steps_pkey PRIMARY KEY (id),
    CONSTRAINT remediation_steps_parameters_check CHECK (jsonb_typeof(parameters) = 'object'::text),
    CONSTRAINT remediation_steps_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text])),
    CONSTRAINT remediation_steps_status_check CHECK (status = ANY (ARRAY['pending'::text, 'executing'::text, 'succeeded'::text, 'failed'::text, 'skipped'::text]))
);

CREATE INDEX idx_remediation_steps_proposal ON remediation_steps USING btree (proposal_id, step_order);
CREATE INDEX idx_remediation_steps_status ON remediation_steps USING btree (status);

CREATE TABLE public.role_permissions (
    permission_id uuid NOT NULL,
    role_id uuid NOT NULL,
    CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id)
);


CREATE TABLE public.roles (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    description text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    is_system_role boolean NOT NULL DEFAULT false,
    name text NOT NULL,
    CONSTRAINT roles_pkey PRIMARY KEY (id),
    CONSTRAINT roles_name_key UNIQUE (name)
);


CREATE TABLE public.saved_knowledge_answers (
    answer text NOT NULL,
    collection_id uuid,
    confidence text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    question text NOT NULL,
    sources jsonb NOT NULL DEFAULT '[]'::jsonb,
    team_id uuid NOT NULL,
    CONSTRAINT saved_knowledge_answers_pkey PRIMARY KEY (id),
    CONSTRAINT chk_saved_answer CHECK (length(TRIM(BOTH FROM answer)) > 0),
    CONSTRAINT chk_saved_confidence CHECK (confidence = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text])),
    CONSTRAINT chk_saved_question CHECK (length(TRIM(BOTH FROM question)) > 0),
    CONSTRAINT chk_saved_sources CHECK (jsonb_typeof(sources) = ANY (ARRAY['array'::text, 'object'::text]))
);


CREATE TABLE public.storage_objects (
    bucket text NOT NULL,
    content_type text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    created_by uuid,
    encryption_status text NOT NULL DEFAULT 'provider_managed'::text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    object_key text NOT NULL,
    retention_policy text NOT NULL DEFAULT 'default'::text,
    sha256 text NOT NULL,
    size_bytes bigint,
    team_id uuid NOT NULL,
    CONSTRAINT storage_objects_pkey PRIMARY KEY (id),
    CONSTRAINT storage_objects_bucket_object_key_key UNIQUE (bucket, object_key),
    CONSTRAINT storage_objects_encryption_status_check CHECK (encryption_status = ANY (ARRAY['none'::text, 'provider_managed'::text, 'app_managed'::text])),
    CONSTRAINT storage_objects_size_bytes_check CHECK (size_bytes IS NULL OR size_bytes >= 0)
);

CREATE INDEX idx_storage_objects_team ON storage_objects USING btree (team_id, created_at DESC);

CREATE TABLE public.team_access_grants (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone,
    grant_type text NOT NULL,
    granted_by uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    revoked_at timestamp with time zone,
    role_id uuid,
    scope text,
    team_id uuid NOT NULL,
    user_id uuid NOT NULL,
    CONSTRAINT team_access_grants_pkey PRIMARY KEY (id),
    CONSTRAINT team_access_grants_grant_type_check CHECK (grant_type = ANY (ARRAY['explicit'::text, 'delegated'::text, 'temporary'::text]))
);

CREATE INDEX idx_team_access_grants_team ON team_access_grants USING btree (team_id);
CREATE INDEX idx_team_access_grants_user ON team_access_grants USING btree (user_id) WHERE revoked_at IS NULL;

CREATE TABLE public.team_memberships (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    joined_at timestamp with time zone NOT NULL DEFAULT now(),
    role_id uuid NOT NULL,
    team_id uuid NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    user_id uuid NOT NULL,
    CONSTRAINT team_memberships_pkey PRIMARY KEY (id),
    CONSTRAINT team_memberships_user_id_team_id_key UNIQUE (user_id, team_id)
);

CREATE INDEX idx_team_memberships_team ON team_memberships USING btree (team_id);
CREATE INDEX idx_team_memberships_user ON team_memberships USING btree (user_id);

CREATE TRIGGER trg_protect_last_team_owner_delete BEFORE DELETE ON team_memberships FOR EACH ROW EXECUTE FUNCTION protect_last_team_owner();
CREATE TRIGGER trg_protect_last_team_owner_update BEFORE UPDATE ON team_memberships FOR EACH ROW EXECUTE FUNCTION protect_last_team_owner();
CREATE TRIGGER trg_team_memberships_updated_at BEFORE UPDATE ON team_memberships FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.teams (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone,
    description text,
    icon text NOT NULL DEFAULT '🏢'::text,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name text NOT NULL,
    settings jsonb NOT NULL DEFAULT '{}'::jsonb,
    slug text NOT NULL,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT teams_pkey PRIMARY KEY (id),
    CONSTRAINT teams_name_check CHECK (TRIM(BOTH FROM name) <> ''::text),
    CONSTRAINT teams_settings_check CHECK (jsonb_typeof(settings) = 'object'::text),
    CONSTRAINT teams_slug_check CHECK (TRIM(BOTH FROM slug) <> ''::text)
);

CREATE UNIQUE INDEX uq_teams_slug_active ON teams USING btree (slug) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_teams_normalize_slug BEFORE INSERT OR UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION normalize_team_slug();
CREATE TRIGGER trg_teams_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.tool_registry (
    description text DEFAULT ''::text,
    display_name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    requires_approval boolean NOT NULL DEFAULT true,
    requires_mfa boolean NOT NULL DEFAULT false,
    risk_level text NOT NULL DEFAULT 'medium'::text,
    tool_name text NOT NULL,
    CONSTRAINT tool_registry_pkey PRIMARY KEY (tool_name),
    CONSTRAINT tool_registry_risk_level_check CHECK (risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text, 'critical'::text]))
);


CREATE TABLE public.user_mfa_factors (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    disabled_at timestamp with time zone,
    factor_type text NOT NULL DEFAULT 'totp'::text,
    failed_attempts integer NOT NULL DEFAULT 0,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    locked_until timestamp with time zone,
    secret bytea NOT NULL,
    status text NOT NULL DEFAULT 'pending'::text,
    user_id uuid NOT NULL,
    verified_at timestamp with time zone,
    CONSTRAINT user_mfa_factors_pkey PRIMARY KEY (id),
    CONSTRAINT user_mfa_factors_user_id_factor_type_key UNIQUE (user_id, factor_type)
);


CREATE TABLE public.user_platform_roles (
    granted_at timestamp with time zone NOT NULL DEFAULT now(),
    granted_by uuid,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    platform_role_id uuid NOT NULL,
    revoked_at timestamp with time zone,
    user_id uuid NOT NULL,
    CONSTRAINT user_platform_roles_pkey PRIMARY KEY (id)
);

CREATE INDEX idx_user_platform_roles_user ON user_platform_roles USING btree (user_id) WHERE revoked_at IS NULL;
CREATE UNIQUE INDEX uq_user_platform_role_active ON user_platform_roles USING btree (user_id, platform_role_id) WHERE revoked_at IS NULL;

CREATE TABLE public.user_sessions (
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    expires_at timestamp with time zone NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    ip_hmac text,
    recent_mfa_at timestamp with time zone,
    revoked_at timestamp with time zone,
    user_agent_hmac text,
    user_id uuid NOT NULL,
    CONSTRAINT user_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT user_sessions_check CHECK (expires_at > created_at)
);

CREATE INDEX idx_user_sessions_active ON user_sessions USING btree (user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_user_sessions_user ON user_sessions USING btree (user_id, created_at DESC);

CREATE TABLE public.user_webauthn_credentials (
    aaguid text NOT NULL DEFAULT ''::text,
    backup_eligible boolean NOT NULL DEFAULT false,
    backup_state boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    credential_id_bytes bytea NOT NULL,
    credential_id_hash text NOT NULL,
    device_type text NOT NULL DEFAULT ''::text,
    disabled_at timestamp with time zone,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    label text NOT NULL DEFAULT ''::text,
    last_used_at timestamp with time zone,
    public_key bytea NOT NULL,
    sign_count bigint NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active'::text,
    transports text[] NOT NULL DEFAULT '{}'::text[],
    user_id uuid NOT NULL,
    CONSTRAINT user_webauthn_credentials_pkey PRIMARY KEY (id),
    CONSTRAINT user_webauthn_credentials_status_check CHECK (status = ANY (ARRAY['active'::text, 'disabled'::text]))
);

CREATE UNIQUE INDEX idx_webauthn_cred_id_hash ON user_webauthn_credentials USING btree (credential_id_hash);
CREATE INDEX idx_webauthn_status ON user_webauthn_credentials USING btree (status);
CREATE INDEX idx_webauthn_user ON user_webauthn_credentials USING btree (user_id);

CREATE TABLE public.users (
    avatar_url text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone,
    email text NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    is_active boolean NOT NULL DEFAULT true,
    last_login_at timestamp with time zone,
    name text NOT NULL,
    password_hash text NOT NULL,
    token_version integer NOT NULL DEFAULT 1,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_email_check CHECK (TRIM(BOTH FROM email) <> ''::text),
    CONSTRAINT users_name_check CHECK (TRIM(BOTH FROM name) <> ''::text)
);

CREATE INDEX idx_users_active ON users USING btree (id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_users_email_active ON users USING btree (email) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_users_normalize_email BEFORE INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION normalize_user_email();
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE public.work_items (
    assignee_user_id uuid,
    completed_at timestamp with time zone,
    due_at timestamp with time zone,
    object_id uuid NOT NULL,
    project_id uuid,
    queue_id uuid,
    sla_policy_id uuid,
    started_at timestamp with time zone,
    work_item_type text NOT NULL,
    CONSTRAINT work_items_pkey PRIMARY KEY (object_id),
    CONSTRAINT work_items_work_item_type_check CHECK (work_item_type = ANY (ARRAY['task'::text, 'ticket'::text, 'incident'::text, 'change'::text, 'problem'::text, 'project_task'::text, 'alert_work_item'::text]))
);


-- Foreign keys
ALTER TABLE public.action_outcomes ADD CONSTRAINT action_outcomes_asset_action_id_fkey FOREIGN KEY (asset_action_id) REFERENCES asset_actions(id) ON DELETE CASCADE;
ALTER TABLE public.action_outcomes ADD CONSTRAINT action_outcomes_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.agent_effect_results ADD CONSTRAINT agent_effect_results_intention_id_fkey FOREIGN KEY (intention_id) REFERENCES agent_intentions(id) ON DELETE CASCADE;
ALTER TABLE public.agent_evaluation_runs ADD CONSTRAINT agent_evaluation_runs_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL;
ALTER TABLE public.agent_evaluation_scenario_results ADD CONSTRAINT agent_evaluation_scenario_results_run_id_fkey FOREIGN KEY (run_id) REFERENCES agent_evaluation_runs(id) ON DELETE CASCADE;
ALTER TABLE public.agent_intentions ADD CONSTRAINT agent_intentions_agent_run_id_fkey FOREIGN KEY (agent_run_id) REFERENCES agent_runs(id) ON DELETE CASCADE;
ALTER TABLE public.agent_runs ADD CONSTRAINT agent_runs_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES agent_identities(id);
ALTER TABLE public.agent_tool_grants ADD CONSTRAINT agent_tool_grants_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES agent_identities(id) ON DELETE CASCADE;
ALTER TABLE public.alerts ADD CONSTRAINT alerts_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.approval_decisions ADD CONSTRAINT approval_decisions_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES approval_requests(id) ON DELETE CASCADE;
ALTER TABLE public.approval_decisions ADD CONSTRAINT approval_decisions_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES users(id);
ALTER TABLE public.approval_policies ADD CONSTRAINT approval_policies_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.approval_requests ADD CONSTRAINT approval_requests_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES approval_policies(id);
ALTER TABLE public.approval_requests ADD CONSTRAINT approval_requests_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES users(id);
ALTER TABLE public.approval_requests ADD CONSTRAINT approval_requests_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.artifact_document_versions ADD CONSTRAINT artifact_document_versions_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE;
ALTER TABLE public.artifact_document_versions ADD CONSTRAINT artifact_document_versions_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.artifact_documents ADD CONSTRAINT artifact_documents_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE;
ALTER TABLE public.artifact_documents ADD CONSTRAINT artifact_documents_last_exported_storage_object_id_fkey FOREIGN KEY (last_exported_storage_object_id) REFERENCES storage_objects(id);
ALTER TABLE public.artifact_meeting_data ADD CONSTRAINT artifact_meeting_data_artifact_id_fkey FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE;
ALTER TABLE public.artifact_templates ADD CONSTRAINT artifact_templates_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.artifacts ADD CONSTRAINT artifacts_storage_object_id_fkey FOREIGN KEY (storage_object_id) REFERENCES storage_objects(id) ON DELETE SET NULL;
ALTER TABLE public.artifacts ADD CONSTRAINT artifacts_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.asset_actions ADD CONSTRAINT asset_actions_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES approval_requests(id);
ALTER TABLE public.asset_actions ADD CONSTRAINT asset_actions_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.asset_actions ADD CONSTRAINT asset_actions_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES users(id);
ALTER TABLE public.asset_actions ADD CONSTRAINT asset_actions_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.assets ADD CONSTRAINT assets_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.context_edge_evidence ADD CONSTRAINT context_edge_evidence_edge_id_fkey FOREIGN KEY (edge_id) REFERENCES context_edges(id) ON DELETE CASCADE;
ALTER TABLE public.context_edges ADD CONSTRAINT context_edges_from_node_id_fkey FOREIGN KEY (from_node_id) REFERENCES context_nodes(id) ON DELETE CASCADE;
ALTER TABLE public.context_edges ADD CONSTRAINT context_edges_to_node_id_fkey FOREIGN KEY (to_node_id) REFERENCES context_nodes(id) ON DELETE CASCADE;
ALTER TABLE public.context_relation_reviews ADD CONSTRAINT context_relation_reviews_relation_id_fkey FOREIGN KEY (relation_id) REFERENCES context_edges(id) ON DELETE CASCADE;
ALTER TABLE public.context_relation_reviews ADD CONSTRAINT context_relation_reviews_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.docs ADD CONSTRAINT docs_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.incidents ADD CONSTRAINT incidents_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.integration_api_keys ADD CONSTRAINT integration_api_keys_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id);
ALTER TABLE public.integration_api_keys ADD CONSTRAINT integration_api_keys_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.invitations ADD CONSTRAINT invitations_invited_by_fkey FOREIGN KEY (invited_by) REFERENCES users(id);
ALTER TABLE public.invitations ADD CONSTRAINT invitations_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE public.invitations ADD CONSTRAINT invitations_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.knowledge_chunks ADD CONSTRAINT knowledge_chunks_knowledge_item_id_fkey FOREIGN KEY (knowledge_item_id) REFERENCES knowledge_items(id) ON DELETE CASCADE;
ALTER TABLE public.knowledge_chunks ADD CONSTRAINT knowledge_chunks_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.knowledge_collection_items ADD CONSTRAINT knowledge_collection_items_added_by_fkey FOREIGN KEY (added_by) REFERENCES users(id);
ALTER TABLE public.knowledge_collection_items ADD CONSTRAINT knowledge_collection_items_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES knowledge_collections(id) ON DELETE CASCADE;
ALTER TABLE public.knowledge_collection_items ADD CONSTRAINT knowledge_collection_items_knowledge_item_id_fkey FOREIGN KEY (knowledge_item_id) REFERENCES knowledge_items(id) ON DELETE SET NULL;
ALTER TABLE public.knowledge_collection_items ADD CONSTRAINT knowledge_collection_items_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.knowledge_collections ADD CONSTRAINT knowledge_collections_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id);
ALTER TABLE public.knowledge_collections ADD CONSTRAINT knowledge_collections_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.knowledge_items ADD CONSTRAINT knowledge_items_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.mfa_challenges ADD CONSTRAINT mfa_challenges_factor_id_fkey FOREIGN KEY (factor_id) REFERENCES user_mfa_factors(id) ON DELETE CASCADE;
ALTER TABLE public.mfa_challenges ADD CONSTRAINT mfa_challenges_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.mfa_recovery_codes ADD CONSTRAINT mfa_recovery_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.object_comments ADD CONSTRAINT object_comments_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.object_links ADD CONSTRAINT object_links_from_object_id_fkey FOREIGN KEY (from_object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.object_links ADD CONSTRAINT object_links_to_object_id_fkey FOREIGN KEY (to_object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.object_storage_refs ADD CONSTRAINT object_storage_refs_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
ALTER TABLE public.object_storage_refs ADD CONSTRAINT object_storage_refs_storage_object_id_fkey FOREIGN KEY (storage_object_id) REFERENCES storage_objects(id) ON DELETE RESTRICT;
ALTER TABLE public.password_reset_tokens ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.proxmox_mutation_windows ADD CONSTRAINT proxmox_mutation_windows_closed_by_fkey FOREIGN KEY (closed_by) REFERENCES users(id);
ALTER TABLE public.proxmox_mutation_windows ADD CONSTRAINT proxmox_mutation_windows_opened_by_fkey FOREIGN KEY (opened_by) REFERENCES users(id);
ALTER TABLE public.proxmox_mutation_windows ADD CONSTRAINT proxmox_mutation_windows_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.recommendation_evidence ADD CONSTRAINT recommendation_evidence_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.refresh_tokens ADD CONSTRAINT refresh_tokens_replaced_by_token_id_fkey FOREIGN KEY (replaced_by_token_id) REFERENCES refresh_tokens(id);
ALTER TABLE public.refresh_tokens ADD CONSTRAINT refresh_tokens_session_id_fkey FOREIGN KEY (session_id) REFERENCES user_sessions(id) ON DELETE CASCADE;
ALTER TABLE public.refresh_tokens ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.remediation_proposals ADD CONSTRAINT remediation_proposals_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES approval_requests(id);
ALTER TABLE public.remediation_proposals ADD CONSTRAINT remediation_proposals_approved_by_fkey FOREIGN KEY (approved_by) REFERENCES users(id);
ALTER TABLE public.remediation_proposals ADD CONSTRAINT remediation_proposals_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id);
ALTER TABLE public.remediation_proposals ADD CONSTRAINT remediation_proposals_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.remediation_steps ADD CONSTRAINT remediation_steps_approval_id_fkey FOREIGN KEY (approval_id) REFERENCES approval_requests(id);
ALTER TABLE public.remediation_steps ADD CONSTRAINT remediation_steps_effect_result_id_fkey FOREIGN KEY (effect_result_id) REFERENCES agent_effect_results(id);
ALTER TABLE public.remediation_steps ADD CONSTRAINT remediation_steps_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES remediation_proposals(id) ON DELETE CASCADE;
ALTER TABLE public.remediation_steps ADD CONSTRAINT remediation_steps_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.role_permissions ADD CONSTRAINT role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;
ALTER TABLE public.role_permissions ADD CONSTRAINT role_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
ALTER TABLE public.saved_knowledge_answers ADD CONSTRAINT saved_knowledge_answers_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES knowledge_collections(id) ON DELETE SET NULL;
ALTER TABLE public.saved_knowledge_answers ADD CONSTRAINT saved_knowledge_answers_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id);
ALTER TABLE public.saved_knowledge_answers ADD CONSTRAINT saved_knowledge_answers_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.team_access_grants ADD CONSTRAINT team_access_grants_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES users(id);
ALTER TABLE public.team_access_grants ADD CONSTRAINT team_access_grants_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE public.team_access_grants ADD CONSTRAINT team_access_grants_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.team_access_grants ADD CONSTRAINT team_access_grants_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.team_memberships ADD CONSTRAINT team_memberships_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE public.team_memberships ADD CONSTRAINT team_memberships_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE public.team_memberships ADD CONSTRAINT team_memberships_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.user_mfa_factors ADD CONSTRAINT user_mfa_factors_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.user_platform_roles ADD CONSTRAINT user_platform_roles_granted_by_fkey FOREIGN KEY (granted_by) REFERENCES users(id);
ALTER TABLE public.user_platform_roles ADD CONSTRAINT user_platform_roles_platform_role_id_fkey FOREIGN KEY (platform_role_id) REFERENCES platform_roles(id) ON DELETE CASCADE;
ALTER TABLE public.user_platform_roles ADD CONSTRAINT user_platform_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.user_sessions ADD CONSTRAINT user_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.user_webauthn_credentials ADD CONSTRAINT user_webauthn_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE public.work_items ADD CONSTRAINT work_items_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;
