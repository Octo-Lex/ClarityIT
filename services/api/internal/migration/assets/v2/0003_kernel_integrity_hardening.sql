-- WP-01 forward revision 0003: kernel integrity hardening.
-- Authority: WP01-AUTH-2026-08-12 / WP01-G0.
-- Companion to 0002. G1 acceptance/current state requires the complete ordered
-- forward series through 0003; 0002 alone is not an accepted WP-01 target.
-- No live provider mutation, provider credential, legacy replay, destructive
-- v1 DDL, revision-ledger write, BEGIN/COMMIT, or psql meta-command appears here.

SET LOCAL ROLE clarityit_owner;
SET LOCAL search_path = pg_catalog, public;

-- K-01 storage invariant: after proposal, the complete operation subject is
-- immutable. G2 owns legal lifecycle transitions; this trigger only prevents a
-- lifecycle update from smuggling a material packet edit under an unchanged
-- canonical digest/signature.
CREATE OR REPLACE FUNCTION kernel.prevent_packet_payload_mutation() RETURNS trigger
LANGUAGE plpgsql AS $function$
BEGIN
    IF OLD.state <> 'draft' AND (
        NEW.id IS DISTINCT FROM OLD.id OR
        NEW.workspace_id IS DISTINCT FROM OLD.workspace_id OR
        NEW.case_id IS DISTINCT FROM OLD.case_id OR
        NEW.resource_id IS DISTINCT FROM OLD.resource_id OR
        NEW.provider_binding_id IS DISTINCT FROM OLD.provider_binding_id OR
        NEW.verification_spec_id IS DISTINCT FROM OLD.verification_spec_id OR
        NEW.schema_version IS DISTINCT FROM OLD.schema_version OR
        NEW.packet_version IS DISTINCT FROM OLD.packet_version OR
        NEW.objective IS DISTINCT FROM OLD.objective OR
        NEW.capability_name IS DISTINCT FROM OLD.capability_name OR
        NEW.capability_version IS DISTINCT FROM OLD.capability_version OR
        NEW.baseline_refs IS DISTINCT FROM OLD.baseline_refs OR
        NEW.parameters IS DISTINCT FROM OLD.parameters OR
        NEW.rationale IS DISTINCT FROM OLD.rationale OR
        NEW.predicted_effects IS DISTINCT FROM OLD.predicted_effects OR
        NEW.preconditions IS DISTINCT FROM OLD.preconditions OR
        NEW.postconditions IS DISTINCT FROM OLD.postconditions OR
        NEW.stop_conditions IS DISTINCT FROM OLD.stop_conditions OR
        NEW.risk IS DISTINCT FROM OLD.risk OR
        NEW.authority_requirement IS DISTINCT FROM OLD.authority_requirement OR
        NEW.compensation_candidate IS DISTINCT FROM OLD.compensation_candidate OR
        NEW.valid_from IS DISTINCT FROM OLD.valid_from OR
        NEW.expires_at IS DISTINCT FROM OLD.expires_at OR
        NEW.nonce IS DISTINCT FROM OLD.nonce OR
        NEW.policy_revision_hint IS DISTINCT FROM OLD.policy_revision_hint OR
        NEW.canonical_payload IS DISTINCT FROM OLD.canonical_payload OR
        NEW.canonical_digest IS DISTINCT FROM OLD.canonical_digest OR
        NEW.signature_envelope IS DISTINCT FROM OLD.signature_envelope OR
        NEW.proposed_at IS DISTINCT FROM OLD.proposed_at OR
        NEW.created_at IS DISTINCT FROM OLD.created_at OR
        NEW.created_by_principal_id IS DISTINCT FROM OLD.created_by_principal_id OR
        NEW.correlation_id IS DISTINCT FROM OLD.correlation_id OR
        NEW.causation_id IS DISTINCT FROM OLD.causation_id OR
        NEW.supersedes_id IS DISTINCT FROM OLD.supersedes_id
    ) THEN
        RAISE EXCEPTION 'proposed operation packet subject is immutable';
    END IF;
    RETURN NEW;
END;
$function$;

-- PrincipalRef is a canonical WP-01 identity record. Bind its ID to the same
-- UUIDv7 contract used by canonical aggregates/messages.
ALTER TABLE kernel.principal_refs
    ADD CONSTRAINT principal_refs_uuid_v7 CHECK (kernel.is_uuid_v7(id));

-- Canonical workspace-safe relationship for Case -> affected Resource.
CREATE TABLE kernel.case_resource_refs (
    workspace_id uuid NOT NULL,
    case_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT case_resource_refs_pkey PRIMARY KEY (workspace_id, case_id, resource_id),
    CONSTRAINT case_resource_refs_case_fkey FOREIGN KEY (workspace_id, case_id)
        REFERENCES kernel.cases(workspace_id, id),
    CONSTRAINT case_resource_refs_resource_fkey FOREIGN KEY (workspace_id, resource_id)
        REFERENCES kernel.resources(workspace_id, id)
);
COMMENT ON COLUMN kernel.cases.affected_resource_ids IS
    'Derived/cache projection only; canonical workspace-safe membership is kernel.case_resource_refs.';

-- Canonical workspace-safe relationship for Resource -> owner PrincipalRef.
CREATE TABLE kernel.resource_owner_refs (
    workspace_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    owner_role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resource_owner_refs_pkey PRIMARY KEY (workspace_id, resource_id, principal_id, owner_role),
    CONSTRAINT resource_owner_refs_role_check CHECK (owner_role IN ('accountable','technical')),
    CONSTRAINT resource_owner_refs_resource_fkey FOREIGN KEY (workspace_id, resource_id)
        REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT resource_owner_refs_principal_fkey FOREIGN KEY (workspace_id, principal_id)
        REFERENCES kernel.principal_refs(workspace_id, id)
);
COMMENT ON COLUMN kernel.resources.owner_refs IS
    'Derived/cache projection only; canonical workspace-safe ownership is kernel.resource_owner_refs.';

-- Canonical versioned verifier/health-contract reference for Resource.
CREATE TABLE kernel.resource_health_contract_refs (
    workspace_id uuid NOT NULL,
    resource_id uuid NOT NULL,
    verification_spec_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resource_health_contract_refs_pkey PRIMARY KEY (workspace_id, resource_id, verification_spec_id),
    CONSTRAINT resource_health_contract_refs_resource_fkey FOREIGN KEY (workspace_id, resource_id)
        REFERENCES kernel.resources(workspace_id, id),
    CONSTRAINT resource_health_contract_refs_spec_fkey FOREIGN KEY (workspace_id, verification_spec_id)
        REFERENCES kernel.verification_specs(workspace_id, id)
);
COMMENT ON COLUMN kernel.resources.health_contract_refs IS
    'Derived/cache projection only; canonical workspace-safe verifier references are kernel.resource_health_contract_refs.';

-- OperationPacket baselines are authoritative approval/execution inputs and must
-- not be representable as cross-workspace/nonexistent Observation references.
CREATE TABLE kernel.operation_packet_baseline_refs (
    workspace_id uuid NOT NULL,
    packet_id uuid NOT NULL,
    observation_id uuid NOT NULL,
    observation_fingerprint text NOT NULL,
    ordinal integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT operation_packet_baseline_refs_pkey PRIMARY KEY (workspace_id, packet_id, observation_id),
    CONSTRAINT operation_packet_baseline_refs_ordinal_uq UNIQUE (workspace_id, packet_id, ordinal),
    CONSTRAINT operation_packet_baseline_refs_ordinal_nonnegative CHECK (ordinal >= 0),
    CONSTRAINT operation_packet_baseline_refs_fingerprint_sha CHECK (observation_fingerprint ~ '^(sha256:)?[0-9a-f]{64}$'),
    CONSTRAINT operation_packet_baseline_refs_packet_fkey FOREIGN KEY (workspace_id, packet_id)
        REFERENCES kernel.operation_packets(workspace_id, id),
    CONSTRAINT operation_packet_baseline_refs_observation_fkey FOREIGN KEY (workspace_id, observation_id)
        REFERENCES kernel.observations(workspace_id, id)
);
COMMENT ON COLUMN kernel.operation_packets.baseline_refs IS
    'Canonical packet bytes retain the typed baseline payload; relational existence/workspace integrity is kernel.operation_packet_baseline_refs.';

-- ResultClaim lineage must reference real receipts from the same workspace.
CREATE TABLE kernel.result_claim_receipt_refs (
    workspace_id uuid NOT NULL,
    result_claim_id uuid NOT NULL,
    provider_receipt_id uuid NOT NULL,
    ordinal integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT result_claim_receipt_refs_pkey PRIMARY KEY (workspace_id, result_claim_id, provider_receipt_id),
    CONSTRAINT result_claim_receipt_refs_ordinal_uq UNIQUE (workspace_id, result_claim_id, ordinal),
    CONSTRAINT result_claim_receipt_refs_ordinal_nonnegative CHECK (ordinal >= 0),
    CONSTRAINT result_claim_receipt_refs_claim_fkey FOREIGN KEY (workspace_id, result_claim_id)
        REFERENCES kernel.result_claims(workspace_id, id),
    CONSTRAINT result_claim_receipt_refs_receipt_fkey FOREIGN KEY (workspace_id, provider_receipt_id)
        REFERENCES kernel.provider_receipts(workspace_id, id)
);
COMMENT ON COLUMN kernel.result_claims.source_receipt_ids IS
    'Derived/cache projection only; canonical workspace-safe lineage is kernel.result_claim_receipt_refs.';

-- Verification inputs must reference real Observations in the same workspace.
CREATE TABLE kernel.verification_observation_refs (
    workspace_id uuid NOT NULL,
    verification_id uuid NOT NULL,
    observation_id uuid NOT NULL,
    ordinal integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT verification_observation_refs_pkey PRIMARY KEY (workspace_id, verification_id, observation_id),
    CONSTRAINT verification_observation_refs_ordinal_uq UNIQUE (workspace_id, verification_id, ordinal),
    CONSTRAINT verification_observation_refs_ordinal_nonnegative CHECK (ordinal >= 0),
    CONSTRAINT verification_observation_refs_verification_fkey FOREIGN KEY (workspace_id, verification_id)
        REFERENCES kernel.verifications(workspace_id, id),
    CONSTRAINT verification_observation_refs_observation_fkey FOREIGN KEY (workspace_id, observation_id)
        REFERENCES kernel.observations(workspace_id, id)
);
COMMENT ON COLUMN kernel.verifications.input_observation_ids IS
    'Derived/cache projection only; canonical workspace-safe inputs are kernel.verification_observation_refs.';

-- Inbox rows are durable dedupe/replay evidence. Runtime may insert/update a
-- processing result but may not delete the dedupe record through clarityit_app.
REVOKE DELETE ON kernel.inbox_messages FROM clarityit_app;

-- The generic grants from 0002 already cover SELECT/INSERT/UPDATE on newly
-- created tables only if reissued after table creation. Grant the minimum G1
-- runtime shape for these link tables; DELETE remains revoked.
GRANT SELECT, INSERT, UPDATE ON
    kernel.case_resource_refs,
    kernel.resource_owner_refs,
    kernel.resource_health_contract_refs,
    kernel.operation_packet_baseline_refs,
    kernel.result_claim_receipt_refs,
    kernel.verification_observation_refs
TO clarityit_app;
REVOKE DELETE ON
    kernel.case_resource_refs,
    kernel.resource_owner_refs,
    kernel.resource_health_contract_refs,
    kernel.operation_packet_baseline_refs,
    kernel.result_claim_receipt_refs,
    kernel.verification_observation_refs
FROM clarityit_app;
