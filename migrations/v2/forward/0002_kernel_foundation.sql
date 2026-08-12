-- WP-01 forward revision 0002: authoritative kernel foundation.
-- Authority: WP01-AUTH-2026-08-12 / WP01-G0.
-- Compatibility phase 2 (expand) only. No live provider mutation, provider
-- credential, legacy migration replay, destructive v1 DDL, or writer cutover.
-- The accepted Go runner owns the transaction; no BEGIN/COMMIT/meta-commands here.

SET LOCAL ROLE clarityit_owner;
SET LOCAL search_path = pg_catalog, public;

CREATE SCHEMA kernel AUTHORIZATION clarityit_owner;
CREATE SCHEMA compat AUTHORIZATION clarityit_owner;
REVOKE ALL ON SCHEMA kernel, compat FROM PUBLIC, clarityit_app;

CREATE FUNCTION kernel.is_uuid_v7(value uuid) RETURNS boolean
LANGUAGE sql IMMUTABLE STRICT
AS $function$ SELECT substring(value::text from 15 for 1) = '7' $function$;

CREATE FUNCTION kernel.reject_immutable_mutation() RETURNS trigger
LANGUAGE plpgsql AS $function$
BEGIN
    RAISE EXCEPTION 'immutable kernel record cannot be updated or deleted';
END;
$function$;

CREATE FUNCTION kernel.prevent_packet_payload_mutation() RETURNS trigger
LANGUAGE plpgsql AS $function$
BEGIN
    IF OLD.state <> 'draft' AND (
        NEW.canonical_payload IS DISTINCT FROM OLD.canonical_payload OR
        NEW.canonical_digest IS DISTINCT FROM OLD.canonical_digest OR
        NEW.signature_envelope IS DISTINCT FROM OLD.signature_envelope OR
        NEW.packet_version IS DISTINCT FROM OLD.packet_version
    ) THEN
        RAISE EXCEPTION 'proposed operation packet payload is immutable';
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TABLE kernel.principal_refs (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    principal_type text NOT NULL,
    external_ref text NOT NULL,
    display_name text,
    source_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT principal_refs_pkey PRIMARY KEY (id),
    CONSTRAINT principal_refs_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT principal_refs_type_check CHECK (principal_type IN ('human','reasoning','service','policy','execution_workload','external_source')),
    CONSTRAINT principal_refs_external_present CHECK (btrim(external_ref) <> ''),
    CONSTRAINT principal_refs_source_object CHECK (jsonb_typeof(source_provenance) = 'object')
);
CREATE UNIQUE INDEX principal_refs_workspace_external_uq
ON kernel.principal_refs (workspace_id, principal_type, external_ref);

CREATE TABLE kernel.cases (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    objective text NOT NULL,
    accountability jsonb NOT NULL DEFAULT '{}'::jsonb,
    affected_resource_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    success_criteria jsonb NOT NULL DEFAULT '[]'::jsonb,
    lifecycle_state text NOT NULL DEFAULT 'open',
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT cases_pkey PRIMARY KEY (id),
    CONSTRAINT cases_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT cases_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT cases_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT cases_objective_present CHECK (btrim(objective) <> ''),
    CONSTRAINT cases_accountability_object CHECK (jsonb_typeof(accountability) = 'object'),
    CONSTRAINT cases_success_criteria_array CHECK (jsonb_typeof(success_criteria) = 'array'),
    CONSTRAINT cases_lifecycle_check CHECK (lifecycle_state IN ('open','investigating','decision_pending','authorized','executing','verifying','outcome_pending','accepted','correction_required','closed')),
    CONSTRAINT cases_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT cases_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.cases(workspace_id, id)
);
CREATE INDEX cases_workspace_state_idx ON kernel.cases (workspace_id, lifecycle_state, recorded_at DESC);

CREATE TABLE kernel.resources (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    resource_type text NOT NULL,
    provider_class text NOT NULL,
    environment text NOT NULL,
    resource_version text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    owner_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
    capability_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
    health_contract_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT resources_pkey PRIMARY KEY (id),
    CONSTRAINT resources_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT resources_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT resources_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT resources_type_present CHECK (btrim(resource_type) <> '' AND btrim(provider_class) <> '' AND btrim(resource_version) <> ''),
    CONSTRAINT resources_environment_check CHECK (environment IN ('development','test','staging','production','restricted')),
    CONSTRAINT resources_status_check CHECK (status IN ('active','quarantined','retired','unresolved')),
    CONSTRAINT resources_owner_refs_array CHECK (jsonb_typeof(owner_refs) = 'array'),
    CONSTRAINT resources_capability_refs_array CHECK (jsonb_typeof(capability_refs) = 'array'),
    CONSTRAINT resources_health_refs_array CHECK (jsonb_typeof(health_contract_refs) = 'array'),
    CONSTRAINT resources_attributes_object CHECK (jsonb_typeof(attributes) = 'object'),
    CONSTRAINT resources_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT resources_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.resources(workspace_id, id)
);
CREATE INDEX resources_workspace_type_idx ON kernel.resources (workspace_id, resource_type, status);

CREATE TABLE kernel.provider_bindings (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    adapter_id text NOT NULL,
    adapter_version text NOT NULL,
    route_identity text NOT NULL,
    external_identity jsonb NOT NULL,
    discovery_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    valid_from timestamptz NOT NULL DEFAULT now(),
    valid_until timestamptz,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT provider_bindings_pkey PRIMARY KEY (id),
    CONSTRAINT provider_bindings_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT provider_bindings_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT provider_bindings_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT provider_bindings_external_object CHECK (jsonb_typeof(external_identity) = 'object'),
    CONSTRAINT provider_bindings_provenance_object CHECK (jsonb_typeof(discovery_provenance) = 'object'),
    CONSTRAINT provider_bindings_validity CHECK (valid_until IS NULL OR valid_until > valid_from),
    CONSTRAINT provider_bindings_status_check CHECK (status IN ('active','quarantined','retired','unresolved')),
    CONSTRAINT provider_bindings_resource_fkey FOREIGN KEY (workspace_id, resource_id) REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT provider_bindings_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT provider_bindings_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.provider_bindings(workspace_id, id)
);
CREATE UNIQUE INDEX provider_bindings_active_identity_uq ON kernel.provider_bindings (workspace_id, resource_id, adapter_id, route_identity) WHERE status = 'active';

CREATE TABLE kernel.observations (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    provider_binding_id uuid,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    source_ref jsonb NOT NULL,
    observed_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    fieldset text[] NOT NULL,
    state jsonb NOT NULL,
    external_revision text,
    fresh_until timestamptz NOT NULL,
    fingerprint text NOT NULL,
    artifact_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT observations_pkey PRIMARY KEY (id),
    CONSTRAINT observations_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT observations_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT observations_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT observations_source_object CHECK (jsonb_typeof(source_ref) = 'object'),
    CONSTRAINT observations_state_object CHECK (jsonb_typeof(state) = 'object'),
    CONSTRAINT observations_artifacts_array CHECK (jsonb_typeof(artifact_refs) = 'array'),
    CONSTRAINT observations_time_order CHECK (received_at >= observed_at AND fresh_until >= observed_at),
    CONSTRAINT observations_fingerprint_sha256 CHECK (fingerprint ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT observations_resource_fkey FOREIGN KEY (workspace_id, resource_id) REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT observations_binding_fkey FOREIGN KEY (workspace_id, provider_binding_id) REFERENCES kernel.provider_bindings(workspace_id, id),
    CONSTRAINT observations_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT observations_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.observations(workspace_id, id)
);
CREATE INDEX observations_resource_time_idx ON kernel.observations (workspace_id, resource_id, observed_at DESC);
CREATE TRIGGER observations_immutable BEFORE UPDATE OR DELETE ON kernel.observations FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.verification_specs (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    verifier_type text NOT NULL,
    verifier_version text NOT NULL,
    required_sources jsonb NOT NULL DEFAULT '[]'::jsonb,
    thresholds jsonb NOT NULL DEFAULT '{}'::jsonb,
    freshness_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
    evaluation_rule jsonb NOT NULL,
    timeout_ms bigint NOT NULL,
    spec_digest text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT verification_specs_pkey PRIMARY KEY (id),
    CONSTRAINT verification_specs_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT verification_specs_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT verification_specs_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT verification_specs_sources_array CHECK (jsonb_typeof(required_sources) = 'array'),
    CONSTRAINT verification_specs_thresholds_object CHECK (jsonb_typeof(thresholds) = 'object'),
    CONSTRAINT verification_specs_freshness_object CHECK (jsonb_typeof(freshness_policy) = 'object'),
    CONSTRAINT verification_specs_rule_object CHECK (jsonb_typeof(evaluation_rule) = 'object'),
    CONSTRAINT verification_specs_timeout_positive CHECK (timeout_ms > 0),
    CONSTRAINT verification_specs_digest_sha256 CHECK (spec_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT verification_specs_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT verification_specs_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.verification_specs(workspace_id, id)
);
CREATE UNIQUE INDEX verification_specs_digest_uq ON kernel.verification_specs (workspace_id, spec_digest);
CREATE TRIGGER verification_specs_immutable BEFORE UPDATE OR DELETE ON kernel.verification_specs FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.operation_packets (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    provider_binding_id uuid NOT NULL,
    verification_spec_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    packet_version bigint NOT NULL DEFAULT 1,
    state text NOT NULL DEFAULT 'draft',
    objective text NOT NULL,
    capability_name text NOT NULL,
    capability_version integer NOT NULL,
    baseline_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
    parameters jsonb NOT NULL DEFAULT '{}'::jsonb,
    rationale jsonb NOT NULL DEFAULT '{}'::jsonb,
    predicted_effects jsonb NOT NULL DEFAULT '[]'::jsonb,
    preconditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    postconditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    stop_conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    risk jsonb NOT NULL DEFAULT '{}'::jsonb,
    authority_requirement jsonb NOT NULL DEFAULT '{}'::jsonb,
    compensation_candidate jsonb,
    valid_from timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    nonce text NOT NULL,
    policy_revision_hint text,
    canonical_payload jsonb,
    canonical_digest text,
    signature_envelope jsonb,
    proposed_at timestamptz,
    dispatch_started_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT operation_packets_pkey PRIMARY KEY (id),
    CONSTRAINT operation_packets_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT operation_packets_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT operation_packets_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0 AND packet_version > 0 AND capability_version > 0),
    CONSTRAINT operation_packets_state_check CHECK (state IN ('draft','proposed','superseded','withdrawn','expired')),
    CONSTRAINT operation_packets_validity CHECK (expires_at > valid_from),
    CONSTRAINT operation_packets_nonce_present CHECK (btrim(nonce) <> ''),
    CONSTRAINT operation_packets_baseline_array CHECK (jsonb_typeof(baseline_refs) = 'array'),
    CONSTRAINT operation_packets_parameters_object CHECK (jsonb_typeof(parameters) = 'object'),
    CONSTRAINT operation_packets_rationale_object CHECK (jsonb_typeof(rationale) = 'object'),
    CONSTRAINT operation_packets_effects_array CHECK (jsonb_typeof(predicted_effects) = 'array'),
    CONSTRAINT operation_packets_preconditions_array CHECK (jsonb_typeof(preconditions) = 'array'),
    CONSTRAINT operation_packets_postconditions_array CHECK (jsonb_typeof(postconditions) = 'array'),
    CONSTRAINT operation_packets_stop_array CHECK (jsonb_typeof(stop_conditions) = 'array'),
    CONSTRAINT operation_packets_risk_object CHECK (jsonb_typeof(risk) = 'object'),
    CONSTRAINT operation_packets_authority_object CHECK (jsonb_typeof(authority_requirement) = 'object'),
    CONSTRAINT operation_packets_compensation_object CHECK (compensation_candidate IS NULL OR jsonb_typeof(compensation_candidate) = 'object'),
    CONSTRAINT operation_packets_payload_object CHECK (canonical_payload IS NULL OR jsonb_typeof(canonical_payload) = 'object'),
    CONSTRAINT operation_packets_signature_object CHECK (signature_envelope IS NULL OR jsonb_typeof(signature_envelope) = 'object'),
    CONSTRAINT operation_packets_digest_shape CHECK (canonical_digest IS NULL OR canonical_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT operation_packets_proposed_fields CHECK (state = 'draft' OR (canonical_payload IS NOT NULL AND canonical_digest IS NOT NULL AND signature_envelope IS NOT NULL AND proposed_at IS NOT NULL)),
    CONSTRAINT operation_packets_expiry_before_dispatch CHECK (state <> 'expired' OR dispatch_started_at IS NULL),
    CONSTRAINT operation_packets_withdraw_before_dispatch CHECK (state <> 'withdrawn' OR dispatch_started_at IS NULL),
    CONSTRAINT operation_packets_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT operation_packets_resource_fkey FOREIGN KEY (workspace_id, resource_id) REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT operation_packets_binding_fkey FOREIGN KEY (workspace_id, provider_binding_id) REFERENCES kernel.provider_bindings(workspace_id, id),
    CONSTRAINT operation_packets_verifier_fkey FOREIGN KEY (workspace_id, verification_spec_id) REFERENCES kernel.verification_specs(workspace_id, id),
    CONSTRAINT operation_packets_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT operation_packets_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.operation_packets(workspace_id, id)
);
CREATE UNIQUE INDEX operation_packets_digest_uq ON kernel.operation_packets (workspace_id, canonical_digest) WHERE canonical_digest IS NOT NULL;
CREATE TRIGGER operation_packets_payload_immutable BEFORE UPDATE ON kernel.operation_packets FOR EACH ROW EXECUTE FUNCTION kernel.prevent_packet_payload_mutation();

CREATE TABLE kernel.policy_decisions (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    packet_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    packet_digest text NOT NULL,
    policy_revision text NOT NULL,
    decision text NOT NULL,
    requirements jsonb NOT NULL DEFAULT '{}'::jsonb,
    reason_codes text[] NOT NULL DEFAULT '{}'::text[],
    evaluated_context_digest text,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT policy_decisions_pkey PRIMARY KEY (id),
    CONSTRAINT policy_decisions_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT policy_decisions_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT policy_decisions_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT policy_decisions_digest_sha CHECK (packet_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT policy_decisions_decision_check CHECK (decision IN ('allow','deny','approval_required')),
    CONSTRAINT policy_decisions_requirements_object CHECK (jsonb_typeof(requirements) = 'object'),
    CONSTRAINT policy_decisions_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT policy_decisions_packet_fkey FOREIGN KEY (workspace_id, packet_id) REFERENCES kernel.operation_packets(workspace_id, id),
    CONSTRAINT policy_decisions_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT policy_decisions_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.policy_decisions(workspace_id, id)
);
CREATE INDEX policy_decisions_packet_idx ON kernel.policy_decisions (workspace_id, packet_id, recorded_at DESC);
CREATE TRIGGER policy_decisions_immutable BEFORE UPDATE OR DELETE ON kernel.policy_decisions FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.approval_decisions (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    packet_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    packet_digest text NOT NULL,
    decision text NOT NULL,
    approver_principal_id uuid NOT NULL,
    required_context jsonb NOT NULL DEFAULT '{}'::jsonb,
    mfa_verified boolean NOT NULL DEFAULT false,
    rationale text,
    decided_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT approval_decisions_pkey PRIMARY KEY (id),
    CONSTRAINT approval_decisions_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT approval_decisions_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT approval_decisions_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT approval_decisions_digest_sha CHECK (packet_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT approval_decisions_decision_check CHECK (decision IN ('approved','rejected')),
    CONSTRAINT approval_decisions_context_object CHECK (jsonb_typeof(required_context) = 'object'),
    CONSTRAINT approval_decisions_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT approval_decisions_packet_fkey FOREIGN KEY (workspace_id, packet_id) REFERENCES kernel.operation_packets(workspace_id, id),
    CONSTRAINT approval_decisions_approver_fkey FOREIGN KEY (workspace_id, approver_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT approval_decisions_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT approval_decisions_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.approval_decisions(workspace_id, id)
);
CREATE UNIQUE INDEX approval_decisions_packet_approver_uq ON kernel.approval_decisions (workspace_id, packet_id, approver_principal_id);
CREATE TRIGGER approval_decisions_immutable BEFORE UPDATE OR DELETE ON kernel.approval_decisions FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.authority_grants (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    packet_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    packet_digest text NOT NULL,
    resource_version text NOT NULL,
    capability_name text NOT NULL,
    capability_version integer NOT NULL,
    parameter_constraints jsonb NOT NULL DEFAULT '{}'::jsonb,
    subject_workload_principal_id uuid NOT NULL,
    route_identity text NOT NULL,
    policy_revision text NOT NULL,
    not_before timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    max_uses integer NOT NULL DEFAULT 1,
    uses_consumed integer NOT NULL DEFAULT 0,
    state text NOT NULL DEFAULT 'issued',
    reservation_id uuid,
    reserved_until timestamptz,
    nonce text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT authority_grants_pkey PRIMARY KEY (id),
    CONSTRAINT authority_grants_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT authority_grants_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT authority_grants_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0 AND capability_version > 0),
    CONSTRAINT authority_grants_digest_sha CHECK (packet_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT authority_grants_parameters_object CHECK (jsonb_typeof(parameter_constraints) = 'object'),
    CONSTRAINT authority_grants_state_check CHECK (state IN ('issued','reserved','consumed','revoked','expired')),
    CONSTRAINT authority_grants_validity CHECK (expires_at > not_before),
    CONSTRAINT authority_grants_use_check CHECK (max_uses > 0 AND uses_consumed >= 0 AND uses_consumed <= max_uses),
    CONSTRAINT authority_grants_reservation_check CHECK ((state = 'reserved' AND reservation_id IS NOT NULL AND reserved_until IS NOT NULL) OR state <> 'reserved'),
    CONSTRAINT authority_grants_nonce_present CHECK (btrim(nonce) <> ''),
    CONSTRAINT authority_grants_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT authority_grants_packet_fkey FOREIGN KEY (workspace_id, packet_id) REFERENCES kernel.operation_packets(workspace_id, id),
    CONSTRAINT authority_grants_resource_fkey FOREIGN KEY (workspace_id, resource_id) REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT authority_grants_subject_fkey FOREIGN KEY (workspace_id, subject_workload_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT authority_grants_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT authority_grants_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.authority_grants(workspace_id, id)
);
CREATE UNIQUE INDEX authority_grants_nonce_uq ON kernel.authority_grants (workspace_id, nonce);

CREATE TABLE kernel.grant_approval_refs (
    workspace_id uuid NOT NULL,
    grant_id uuid NOT NULL,
    approval_decision_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT grant_approval_refs_pkey PRIMARY KEY (workspace_id, grant_id, approval_decision_id),
    CONSTRAINT grant_approval_refs_grant_fkey FOREIGN KEY (workspace_id, grant_id) REFERENCES kernel.authority_grants(workspace_id, id),
    CONSTRAINT grant_approval_refs_approval_fkey FOREIGN KEY (workspace_id, approval_decision_id) REFERENCES kernel.approval_decisions(workspace_id, id)
);

CREATE TABLE kernel.execution_attempts (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    packet_id uuid NOT NULL,
    grant_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    provider_binding_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    state text NOT NULL DEFAULT 'created',
    logical_idempotency_key text NOT NULL,
    route_identity text NOT NULL,
    executor_workload_principal_id uuid NOT NULL,
    provider_operation_ref text,
    preflight_observation_id uuid,
    submitted_at timestamptz,
    terminal_at timestamptz,
    reason_code text,
    reason_detail jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT execution_attempts_pkey PRIMARY KEY (id),
    CONSTRAINT execution_attempts_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT execution_attempts_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT execution_attempts_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT execution_attempts_state_check CHECK (state IN ('created','preflight','dispatchable','submitting','submitted','running','provider_completed','provider_failed','blocked','cancelled','outcome_unknown')),
    CONSTRAINT execution_attempts_key_present CHECK (btrim(logical_idempotency_key) <> ''),
    CONSTRAINT execution_attempts_reason_object CHECK (jsonb_typeof(reason_detail) = 'object'),
    CONSTRAINT execution_attempts_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT execution_attempts_packet_fkey FOREIGN KEY (workspace_id, packet_id) REFERENCES kernel.operation_packets(workspace_id, id),
    CONSTRAINT execution_attempts_grant_fkey FOREIGN KEY (workspace_id, grant_id) REFERENCES kernel.authority_grants(workspace_id, id),
    CONSTRAINT execution_attempts_resource_fkey FOREIGN KEY (workspace_id, resource_id) REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT execution_attempts_binding_fkey FOREIGN KEY (workspace_id, provider_binding_id) REFERENCES kernel.provider_bindings(workspace_id, id),
    CONSTRAINT execution_attempts_preflight_observation_fkey FOREIGN KEY (workspace_id, preflight_observation_id) REFERENCES kernel.observations(workspace_id, id),
    CONSTRAINT execution_attempts_executor_fkey FOREIGN KEY (workspace_id, executor_workload_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT execution_attempts_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT execution_attempts_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.execution_attempts(workspace_id, id)
);
CREATE UNIQUE INDEX execution_attempts_logical_key_uq ON kernel.execution_attempts (workspace_id, logical_idempotency_key);
CREATE INDEX execution_attempts_case_state_idx ON kernel.execution_attempts (workspace_id, case_id, state, recorded_at DESC);

CREATE TABLE kernel.dispatch_records (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    grant_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    route_identity text NOT NULL,
    dispatch_payload_digest text NOT NULL,
    requested_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    CONSTRAINT dispatch_records_pkey PRIMARY KEY (id),
    CONSTRAINT dispatch_records_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT dispatch_records_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT dispatch_records_schema_positive CHECK (schema_version > 0),
    CONSTRAINT dispatch_records_digest_sha CHECK (dispatch_payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT dispatch_records_attempt_fkey FOREIGN KEY (workspace_id, attempt_id) REFERENCES kernel.execution_attempts(workspace_id, id),
    CONSTRAINT dispatch_records_grant_fkey FOREIGN KEY (workspace_id, grant_id) REFERENCES kernel.authority_grants(workspace_id, id),
    CONSTRAINT dispatch_records_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id)
);
CREATE UNIQUE INDEX dispatch_records_attempt_uq ON kernel.dispatch_records (workspace_id, attempt_id);
CREATE TRIGGER dispatch_records_immutable BEFORE UPDATE OR DELETE ON kernel.dispatch_records FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.provider_receipts (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    receipt_type text NOT NULL,
    source_identity text NOT NULL,
    adapter_id text NOT NULL,
    adapter_version text NOT NULL,
    route_identity text NOT NULL,
    provider_operation_ref text,
    provider_status text,
    raw_provider_code text,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    payload_digest text NOT NULL,
    occurred_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    CONSTRAINT provider_receipts_pkey PRIMARY KEY (id),
    CONSTRAINT provider_receipts_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT provider_receipts_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT provider_receipts_schema_positive CHECK (schema_version > 0),
    CONSTRAINT provider_receipts_type_check CHECK (receipt_type IN ('acknowledgement','progress','terminal_result','error','reconciliation')),
    CONSTRAINT provider_receipts_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT provider_receipts_digest_sha CHECK (payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT provider_receipts_attempt_fkey FOREIGN KEY (workspace_id, attempt_id) REFERENCES kernel.execution_attempts(workspace_id, id),
    CONSTRAINT provider_receipts_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id)
);
CREATE INDEX provider_receipts_attempt_time_idx ON kernel.provider_receipts (workspace_id, attempt_id, occurred_at);
CREATE TRIGGER provider_receipts_immutable BEFORE UPDATE OR DELETE ON kernel.provider_receipts FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.result_claims (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    claim_type text NOT NULL,
    normalized_status text NOT NULL,
    claim jsonb NOT NULL,
    source_receipt_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    claim_digest text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    CONSTRAINT result_claims_pkey PRIMARY KEY (id),
    CONSTRAINT result_claims_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT result_claims_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT result_claims_schema_positive CHECK (schema_version > 0),
    CONSTRAINT result_claims_claim_object CHECK (jsonb_typeof(claim) = 'object'),
    CONSTRAINT result_claims_digest_sha CHECK (claim_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT result_claims_attempt_fkey FOREIGN KEY (workspace_id, attempt_id) REFERENCES kernel.execution_attempts(workspace_id, id),
    CONSTRAINT result_claims_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id)
);
CREATE INDEX result_claims_attempt_idx ON kernel.result_claims (workspace_id, attempt_id, recorded_at);
CREATE TRIGGER result_claims_immutable BEFORE UPDATE OR DELETE ON kernel.result_claims FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.verifications (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    verification_spec_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    state text NOT NULL DEFAULT 'pending',
    spec_digest text NOT NULL,
    input_observation_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    input_manifest_digest text,
    result jsonb,
    reason_codes text[] NOT NULL DEFAULT '{}'::text[],
    started_at timestamptz,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT verifications_pkey PRIMARY KEY (id),
    CONSTRAINT verifications_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT verifications_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT verifications_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT verifications_state_check CHECK (state IN ('pending','running','passed','failed','inconclusive')),
    CONSTRAINT verifications_spec_digest_sha CHECK (spec_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT verifications_result_object CHECK (result IS NULL OR jsonb_typeof(result) = 'object'),
    CONSTRAINT verifications_time_order CHECK (completed_at IS NULL OR started_at IS NULL OR completed_at >= started_at),
    CONSTRAINT verifications_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT verifications_attempt_fkey FOREIGN KEY (workspace_id, attempt_id) REFERENCES kernel.execution_attempts(workspace_id, id),
    CONSTRAINT verifications_spec_fkey FOREIGN KEY (workspace_id, verification_spec_id) REFERENCES kernel.verification_specs(workspace_id, id),
    CONSTRAINT verifications_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT verifications_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.verifications(workspace_id, id)
);
CREATE INDEX verifications_attempt_idx ON kernel.verifications (workspace_id, attempt_id, recorded_at DESC);

CREATE TABLE kernel.verification_evidence (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    verification_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    source_ref jsonb NOT NULL,
    observed_at timestamptz NOT NULL,
    evidence_type text NOT NULL,
    artifact_ref text,
    artifact_digest text,
    normalized_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    payload_digest text NOT NULL,
    redaction_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    CONSTRAINT verification_evidence_pkey PRIMARY KEY (id),
    CONSTRAINT verification_evidence_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT verification_evidence_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT verification_evidence_schema_positive CHECK (schema_version > 0),
    CONSTRAINT verification_evidence_source_object CHECK (jsonb_typeof(source_ref) = 'object'),
    CONSTRAINT verification_evidence_payload_object CHECK (jsonb_typeof(normalized_payload) = 'object'),
    CONSTRAINT verification_evidence_redaction_object CHECK (jsonb_typeof(redaction_metadata) = 'object'),
    CONSTRAINT verification_evidence_payload_sha CHECK (payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT verification_evidence_artifact_sha CHECK (artifact_digest IS NULL OR artifact_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT verification_evidence_verification_fkey FOREIGN KEY (workspace_id, verification_id) REFERENCES kernel.verifications(workspace_id, id),
    CONSTRAINT verification_evidence_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id)
);
CREATE TRIGGER verification_evidence_immutable BEFORE UPDATE OR DELETE ON kernel.verification_evidence FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.outcome_decisions (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    verification_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    state text NOT NULL DEFAULT 'pending',
    accountable_principal_id uuid,
    rationale text,
    decided_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT outcome_decisions_pkey PRIMARY KEY (id),
    CONSTRAINT outcome_decisions_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT outcome_decisions_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT outcome_decisions_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT outcome_decisions_state_check CHECK (state IN ('pending','accepted','rejected','correction_required','compensation_required')),
    CONSTRAINT outcome_decisions_terminal_actor CHECK (state = 'pending' OR (accountable_principal_id IS NOT NULL AND decided_at IS NOT NULL)),
    CONSTRAINT outcome_decisions_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT outcome_decisions_attempt_fkey FOREIGN KEY (workspace_id, attempt_id) REFERENCES kernel.execution_attempts(workspace_id, id),
    CONSTRAINT outcome_decisions_verification_fkey FOREIGN KEY (workspace_id, verification_id) REFERENCES kernel.verifications(workspace_id, id),
    CONSTRAINT outcome_decisions_accountable_fkey FOREIGN KEY (workspace_id, accountable_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT outcome_decisions_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT outcome_decisions_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.outcome_decisions(workspace_id, id)
);
CREATE INDEX outcome_decisions_case_idx ON kernel.outcome_decisions (workspace_id, case_id, recorded_at DESC);

CREATE TABLE kernel.evidence_manifests (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    lineage_type text NOT NULL,
    manifest jsonb NOT NULL,
    manifest_digest text NOT NULL,
    redaction_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    sealed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_by_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    supersedes_id uuid,
    CONSTRAINT evidence_manifests_pkey PRIMARY KEY (id),
    CONSTRAINT evidence_manifests_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT evidence_manifests_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT evidence_manifests_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT evidence_manifests_lineage_check CHECK (lineage_type IN ('happy','blocked','rejected','failed','cancelled','unknown','inconclusive','superseded','compensation_required','compensated','successor')),
    CONSTRAINT evidence_manifests_manifest_object CHECK (jsonb_typeof(manifest) = 'object'),
    CONSTRAINT evidence_manifests_redaction_object CHECK (jsonb_typeof(redaction_metadata) = 'object'),
    CONSTRAINT evidence_manifests_digest_sha CHECK (manifest_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT evidence_manifests_case_fkey FOREIGN KEY (workspace_id, case_id) REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT evidence_manifests_created_by_fkey FOREIGN KEY (workspace_id, created_by_principal_id) REFERENCES kernel.principal_refs(workspace_id, id),
    CONSTRAINT evidence_manifests_supersedes_fkey FOREIGN KEY (workspace_id, supersedes_id) REFERENCES kernel.evidence_manifests(workspace_id, id)
);
CREATE UNIQUE INDEX evidence_manifests_digest_uq ON kernel.evidence_manifests (workspace_id, manifest_digest);
CREATE TRIGGER evidence_manifests_immutable BEFORE UPDATE OR DELETE ON kernel.evidence_manifests FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.audit_records (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    schema_version integer NOT NULL DEFAULT 1,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL,
    action text NOT NULL,
    actor_principal_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    payload_digest text NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_records_pkey PRIMARY KEY (id),
    CONSTRAINT audit_records_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT audit_records_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT audit_records_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT audit_records_payload_sha CHECK (payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT audit_records_actor_fkey FOREIGN KEY (workspace_id, actor_principal_id) REFERENCES kernel.principal_refs(workspace_id, id)
);
CREATE INDEX audit_records_aggregate_idx ON kernel.audit_records (workspace_id, aggregate_type, aggregate_id, aggregate_version);
CREATE TRIGGER audit_records_immutable BEFORE UPDATE OR DELETE ON kernel.audit_records FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE kernel.outbox_messages (
    message_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    message_type text NOT NULL,
    schema_version integer NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    actor_principal_id uuid NOT NULL,
    occurred_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    payload_digest text NOT NULL,
    payload jsonb NOT NULL,
    published_at timestamptz,
    publish_attempts integer NOT NULL DEFAULT 0,
    CONSTRAINT outbox_messages_pkey PRIMARY KEY (message_id),
    CONSTRAINT outbox_messages_workspace_id_key UNIQUE (workspace_id, message_id),
    CONSTRAINT outbox_messages_uuid_v7 CHECK (kernel.is_uuid_v7(message_id)),
    CONSTRAINT outbox_messages_versions_positive CHECK (schema_version > 0 AND aggregate_version > 0),
    CONSTRAINT outbox_messages_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT outbox_messages_payload_sha CHECK (payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT outbox_messages_attempts_nonnegative CHECK (publish_attempts >= 0),
    CONSTRAINT outbox_messages_actor_fkey FOREIGN KEY (workspace_id, actor_principal_id) REFERENCES kernel.principal_refs(workspace_id, id)
);
CREATE INDEX outbox_messages_pending_idx ON kernel.outbox_messages (recorded_at) WHERE published_at IS NULL;

CREATE TABLE kernel.inbox_messages (
    consumer_name text NOT NULL,
    message_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    schema_version integer NOT NULL,
    payload_digest text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    result_digest text,
    result jsonb,
    CONSTRAINT inbox_messages_pkey PRIMARY KEY (consumer_name, message_id),
    CONSTRAINT inbox_messages_workspace_message_key UNIQUE (workspace_id, consumer_name, message_id),
    CONSTRAINT inbox_messages_message_uuid_v7 CHECK (kernel.is_uuid_v7(message_id)),
    CONSTRAINT inbox_messages_schema_positive CHECK (schema_version > 0),
    CONSTRAINT inbox_messages_payload_sha CHECK (payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT inbox_messages_result_sha CHECK (result_digest IS NULL OR result_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT inbox_messages_result_object CHECK (result IS NULL OR jsonb_typeof(result) = 'object')
);

CREATE TABLE kernel.message_quarantine (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    consumer_name text NOT NULL,
    message_id uuid,
    schema_version integer,
    reason_code text NOT NULL,
    payload_digest text,
    evidence_ref text NOT NULL,
    terminal_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_quarantine_pkey PRIMARY KEY (id),
    CONSTRAINT message_quarantine_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT message_quarantine_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT message_quarantine_schema_positive CHECK (schema_version IS NULL OR schema_version > 0),
    CONSTRAINT message_quarantine_payload_sha CHECK (payload_digest IS NULL OR payload_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT message_quarantine_evidence_present CHECK (btrim(evidence_ref) <> '')
);
CREATE TRIGGER message_quarantine_immutable BEFORE UPDATE OR DELETE ON kernel.message_quarantine FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE compat.writer_ownership (
    object_family text NOT NULL,
    authoritative_writer text NOT NULL,
    owner_version integer NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    authority_ref text NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT writer_ownership_pkey PRIMARY KEY (object_family, owner_version),
    CONSTRAINT writer_ownership_version_positive CHECK (owner_version > 0),
    CONSTRAINT writer_ownership_time_order CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT writer_ownership_writer_check CHECK (authoritative_writer IN ('v1','v2')),
    CONSTRAINT writer_ownership_authority_present CHECK (btrim(authority_ref) <> '')
);
CREATE UNIQUE INDEX writer_ownership_one_current_uq ON compat.writer_ownership (object_family) WHERE effective_to IS NULL;

CREATE TABLE compat.identity_mappings (
    id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    object_family text NOT NULL,
    legacy_table text NOT NULL,
    legacy_key text NOT NULL,
    legacy_semantic_class text NOT NULL,
    v2_object_id uuid NOT NULL,
    v2_object_type text NOT NULL,
    source_profile_id text,
    source_run_id uuid,
    mapping_version integer NOT NULL DEFAULT 1,
    mapping_digest text NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identity_mappings_pkey PRIMARY KEY (id),
    CONSTRAINT identity_mappings_workspace_id_key UNIQUE (workspace_id, id),
    CONSTRAINT identity_mappings_uuid_v7 CHECK (kernel.is_uuid_v7(id)),
    CONSTRAINT identity_mappings_version_positive CHECK (mapping_version > 0),
    CONSTRAINT identity_mappings_digest_sha CHECK (mapping_digest ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT identity_mappings_legacy_unique UNIQUE (workspace_id, object_family, legacy_table, legacy_key),
    CONSTRAINT identity_mappings_v2_unique UNIQUE (workspace_id, v2_object_type, v2_object_id)
);
CREATE TRIGGER identity_mappings_immutable BEFORE UPDATE OR DELETE ON compat.identity_mappings FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE TABLE compat.backfill_checkpoints (
    mapping_name text NOT NULL,
    workspace_id uuid NOT NULL,
    checkpoint_version integer NOT NULL DEFAULT 1,
    source_position jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL,
    processed_count bigint NOT NULL DEFAULT 0,
    last_error_code text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT backfill_checkpoints_pkey PRIMARY KEY (mapping_name, workspace_id),
    CONSTRAINT backfill_checkpoints_version_positive CHECK (checkpoint_version > 0),
    CONSTRAINT backfill_checkpoints_position_object CHECK (jsonb_typeof(source_position) = 'object'),
    CONSTRAINT backfill_checkpoints_status_check CHECK (status IN ('pending','running','paused','completed','blocked')),
    CONSTRAINT backfill_checkpoints_count_nonnegative CHECK (processed_count >= 0)
);

CREATE TABLE compat.feature_flags (
    flag_name text NOT NULL,
    scope_key text NOT NULL DEFAULT '*',
    workspace_id uuid,
    enabled boolean NOT NULL DEFAULT false,
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    authority_ref text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT feature_flags_pkey PRIMARY KEY (flag_name, scope_key),
    CONSTRAINT feature_flags_scope_check CHECK ((scope_key = '*' AND workspace_id IS NULL) OR (scope_key <> '*' AND workspace_id IS NOT NULL)),
    CONSTRAINT feature_flags_config_object CHECK (jsonb_typeof(config) = 'object'),
    CONSTRAINT feature_flags_authority_present CHECK (btrim(authority_ref) <> '')
);
CREATE UNIQUE INDEX feature_flags_workspace_uq ON compat.feature_flags (flag_name, workspace_id) WHERE workspace_id IS NOT NULL;

INSERT INTO compat.writer_ownership (object_family, authoritative_writer, owner_version, effective_from, authority_ref)
VALUES
    ('v1_existing_product_families', 'v1', 1, TIMESTAMPTZ '2026-08-12T00:00:00Z', 'WP01-AUTH-2026-08-12'),
    ('v2_kernel_objects', 'v2', 1, TIMESTAMPTZ '2026-08-12T00:00:00Z', 'WP01-AUTH-2026-08-12');

INSERT INTO compat.feature_flags (flag_name, scope_key, workspace_id, enabled, config, authority_ref)
VALUES
    ('wp01.kernel.enabled', '*', NULL, false, '{}'::jsonb, 'WP01-AUTH-2026-08-12'),
    ('wp01.effect_broker.fake_route.enabled', '*', NULL, false, '{}'::jsonb, 'WP01-AUTH-2026-08-12'),
    ('wp01.live_provider_mutation.enabled', '*', NULL, false, '{"forbidden":true}'::jsonb, 'WP01-AUTH-2026-08-12');

REVOKE ALL ON ALL TABLES IN SCHEMA kernel, compat FROM PUBLIC;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA kernel, compat FROM PUBLIC;
GRANT USAGE ON SCHEMA kernel, compat TO clarityit_app;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA kernel, compat TO clarityit_app;
REVOKE DELETE ON ALL TABLES IN SCHEMA kernel, compat FROM clarityit_app;
GRANT DELETE ON kernel.inbox_messages TO clarityit_app;

ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA kernel REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA compat REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA kernel REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA compat REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
