-- WP-01 forward revision 0005: lineage and message integrity hardening.
-- Authority: WP01-AUTH-2026-08-12 / WP01-G0.
-- Storage-level integrity only; transition/service semantics remain WP01-G2+.
-- No live provider mutation, provider credential, legacy replay, revision-ledger
-- write, BEGIN/COMMIT, psql meta-command, destructive v1 DDL, or writer cutover.

SET LOCAL ROLE clarityit_owner;
SET LOCAL search_path = pg_catalog, public;

-- ---------------------------------------------------------------------------
-- Exact packet baseline binding: a signed packet baseline must resolve to a
-- real Observation in the same workspace, for the packet's exact Resource, and
-- with the exact Observation fingerprint it records.
-- ---------------------------------------------------------------------------
ALTER TABLE kernel.observations
    ADD CONSTRAINT observations_workspace_id_fingerprint_resource_key
    UNIQUE (workspace_id, id, fingerprint, resource_id);

ALTER TABLE kernel.operation_packets
    ADD CONSTRAINT operation_packets_workspace_id_resource_key
    UNIQUE (workspace_id, id, resource_id);

ALTER TABLE kernel.operation_packet_baseline_refs
    ADD COLUMN resource_id uuid NOT NULL,
    ADD CONSTRAINT operation_packet_baseline_refs_packet_resource_fkey
        FOREIGN KEY (workspace_id, packet_id, resource_id)
        REFERENCES kernel.operation_packets(workspace_id, id, resource_id),
    ADD CONSTRAINT operation_packet_baseline_refs_observation_exact_fkey
        FOREIGN KEY (workspace_id, observation_id, observation_fingerprint, resource_id)
        REFERENCES kernel.observations(workspace_id, id, fingerprint, resource_id);

-- ---------------------------------------------------------------------------
-- ResultClaim lineage: every declared ProviderReceipt must belong to the same
-- workspace and exact ExecutionAttempt as the immutable claim.
-- ---------------------------------------------------------------------------
ALTER TABLE kernel.result_claims
    ADD CONSTRAINT result_claims_workspace_id_attempt_key
    UNIQUE (workspace_id, id, attempt_id);

ALTER TABLE kernel.provider_receipts
    ADD CONSTRAINT provider_receipts_workspace_id_attempt_key
    UNIQUE (workspace_id, id, attempt_id);

ALTER TABLE kernel.result_claim_receipt_refs
    ADD COLUMN attempt_id uuid NOT NULL,
    ADD CONSTRAINT result_claim_receipt_refs_claim_attempt_fkey
        FOREIGN KEY (workspace_id, result_claim_id, attempt_id)
        REFERENCES kernel.result_claims(workspace_id, id, attempt_id),
    ADD CONSTRAINT result_claim_receipt_refs_receipt_attempt_fkey
        FOREIGN KEY (workspace_id, provider_receipt_id, attempt_id)
        REFERENCES kernel.provider_receipts(workspace_id, id, attempt_id);

-- ---------------------------------------------------------------------------
-- Verification input lineage: each Observation must belong to the same
-- workspace and Resource as the Verification's ExecutionAttempt.
-- ---------------------------------------------------------------------------
ALTER TABLE kernel.execution_attempts
    ADD CONSTRAINT execution_attempts_workspace_id_resource_key
    UNIQUE (workspace_id, id, resource_id);

ALTER TABLE kernel.verifications
    ADD CONSTRAINT verifications_workspace_id_attempt_key
    UNIQUE (workspace_id, id, attempt_id);

ALTER TABLE kernel.observations
    ADD CONSTRAINT observations_workspace_id_resource_key
    UNIQUE (workspace_id, id, resource_id);

ALTER TABLE kernel.verification_observation_refs
    ADD COLUMN attempt_id uuid NOT NULL,
    ADD COLUMN resource_id uuid NOT NULL,
    ADD CONSTRAINT verification_observation_refs_verification_attempt_fkey
        FOREIGN KEY (workspace_id, verification_id, attempt_id)
        REFERENCES kernel.verifications(workspace_id, id, attempt_id),
    ADD CONSTRAINT verification_observation_refs_observation_resource_fkey
        FOREIGN KEY (workspace_id, observation_id, resource_id)
        REFERENCES kernel.observations(workspace_id, id, resource_id),
    ADD CONSTRAINT verification_observation_refs_attempt_resource_fkey
        FOREIGN KEY (workspace_id, attempt_id, resource_id)
        REFERENCES kernel.execution_attempts(workspace_id, id, resource_id);

-- ---------------------------------------------------------------------------
-- EvidenceManifest typed lineage. The JSON manifest remains the canonical
-- serialized/digested representation, but authoritative record identities are
-- additionally persisted in this relational index. Exactly one typed target is
-- present per row, every target is workspace-scoped by FK, and a deferred
-- constraint trigger prevents a sealed manifest from committing without at
-- least one validated typed lineage reference.
-- ---------------------------------------------------------------------------
CREATE TABLE kernel.evidence_manifest_refs (
    workspace_id uuid NOT NULL,
    evidence_manifest_id uuid NOT NULL,
    ordinal integer NOT NULL,
    ref_type text NOT NULL,
    operation_packet_id uuid,
    policy_decision_id uuid,
    approval_decision_id uuid,
    authority_grant_id uuid,
    execution_attempt_id uuid,
    provider_receipt_id uuid,
    result_claim_id uuid,
    observation_id uuid,
    verification_id uuid,
    verification_evidence_id uuid,
    outcome_decision_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evidence_manifest_refs_pkey
        PRIMARY KEY (workspace_id, evidence_manifest_id, ordinal),
    CONSTRAINT evidence_manifest_refs_ordinal_nonnegative CHECK (ordinal >= 0),
    CONSTRAINT evidence_manifest_refs_type_check CHECK (ref_type IN (
        'operation_packet','policy_decision','approval_decision','authority_grant',
        'execution_attempt','provider_receipt','result_claim','observation',
        'verification','verification_evidence','outcome_decision'
    )),
    CONSTRAINT evidence_manifest_refs_exactly_one_target CHECK (
        num_nonnulls(
            operation_packet_id, policy_decision_id, approval_decision_id,
            authority_grant_id, execution_attempt_id, provider_receipt_id,
            result_claim_id, observation_id, verification_id,
            verification_evidence_id, outcome_decision_id
        ) = 1
        AND (
            (ref_type = 'operation_packet' AND operation_packet_id IS NOT NULL) OR
            (ref_type = 'policy_decision' AND policy_decision_id IS NOT NULL) OR
            (ref_type = 'approval_decision' AND approval_decision_id IS NOT NULL) OR
            (ref_type = 'authority_grant' AND authority_grant_id IS NOT NULL) OR
            (ref_type = 'execution_attempt' AND execution_attempt_id IS NOT NULL) OR
            (ref_type = 'provider_receipt' AND provider_receipt_id IS NOT NULL) OR
            (ref_type = 'result_claim' AND result_claim_id IS NOT NULL) OR
            (ref_type = 'observation' AND observation_id IS NOT NULL) OR
            (ref_type = 'verification' AND verification_id IS NOT NULL) OR
            (ref_type = 'verification_evidence' AND verification_evidence_id IS NOT NULL) OR
            (ref_type = 'outcome_decision' AND outcome_decision_id IS NOT NULL)
        )
    ),
    CONSTRAINT evidence_manifest_refs_manifest_fkey
        FOREIGN KEY (workspace_id, evidence_manifest_id)
        REFERENCES kernel.evidence_manifests(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_packet_fkey
        FOREIGN KEY (workspace_id, operation_packet_id)
        REFERENCES kernel.operation_packets(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_policy_fkey
        FOREIGN KEY (workspace_id, policy_decision_id)
        REFERENCES kernel.policy_decisions(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_approval_fkey
        FOREIGN KEY (workspace_id, approval_decision_id)
        REFERENCES kernel.approval_decisions(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_grant_fkey
        FOREIGN KEY (workspace_id, authority_grant_id)
        REFERENCES kernel.authority_grants(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_attempt_fkey
        FOREIGN KEY (workspace_id, execution_attempt_id)
        REFERENCES kernel.execution_attempts(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_receipt_fkey
        FOREIGN KEY (workspace_id, provider_receipt_id)
        REFERENCES kernel.provider_receipts(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_claim_fkey
        FOREIGN KEY (workspace_id, result_claim_id)
        REFERENCES kernel.result_claims(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_observation_fkey
        FOREIGN KEY (workspace_id, observation_id)
        REFERENCES kernel.observations(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_verification_fkey
        FOREIGN KEY (workspace_id, verification_id)
        REFERENCES kernel.verifications(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_verification_evidence_fkey
        FOREIGN KEY (workspace_id, verification_evidence_id)
        REFERENCES kernel.verification_evidence(workspace_id, id),
    CONSTRAINT evidence_manifest_refs_outcome_fkey
        FOREIGN KEY (workspace_id, outcome_decision_id)
        REFERENCES kernel.outcome_decisions(workspace_id, id)
);
CREATE INDEX evidence_manifest_refs_target_type_idx
    ON kernel.evidence_manifest_refs (workspace_id, ref_type, evidence_manifest_id);
CREATE TRIGGER evidence_manifest_refs_immutable
BEFORE UPDATE OR DELETE ON kernel.evidence_manifest_refs
FOR EACH ROW EXECUTE FUNCTION kernel.reject_immutable_mutation();

CREATE FUNCTION kernel.require_evidence_manifest_typed_lineage() RETURNS trigger
LANGUAGE plpgsql AS $function$
DECLARE
    ref_count bigint;
BEGIN
    SELECT count(*) INTO ref_count
    FROM kernel.evidence_manifest_refs
    WHERE workspace_id = NEW.workspace_id
      AND evidence_manifest_id = NEW.id;
    IF ref_count = 0 THEN
        RAISE EXCEPTION 'sealed evidence manifest requires typed workspace-scoped lineage references';
    END IF;
    RETURN NULL;
END;
$function$;

CREATE CONSTRAINT TRIGGER evidence_manifests_typed_lineage_required
AFTER INSERT ON kernel.evidence_manifests
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION kernel.require_evidence_manifest_typed_lineage();

COMMENT ON COLUMN kernel.evidence_manifests.manifest IS
    'Canonical serialized/digested evidence index; authoritative target identities are additionally validated by kernel.evidence_manifest_refs.';

-- ---------------------------------------------------------------------------
-- Durable message envelopes. Runtime code may insert complete messages and
-- update only processing metadata; workspace/aggregate identities, payloads and
-- digests are not mutable through clarityit_app.
-- ---------------------------------------------------------------------------
REVOKE UPDATE ON kernel.outbox_messages, kernel.inbox_messages FROM clarityit_app;
GRANT UPDATE (published_at, publish_attempts)
    ON kernel.outbox_messages TO clarityit_app;
GRANT UPDATE (processed_at, result_digest, result)
    ON kernel.inbox_messages TO clarityit_app;
REVOKE DELETE ON kernel.outbox_messages, kernel.inbox_messages FROM clarityit_app;

-- New typed lineage table follows the same non-destructive runtime posture.
GRANT SELECT, INSERT ON kernel.evidence_manifest_refs TO clarityit_app;
REVOKE UPDATE, DELETE ON kernel.evidence_manifest_refs FROM clarityit_app;
REVOKE EXECUTE ON FUNCTION kernel.require_evidence_manifest_typed_lineage() FROM PUBLIC, clarityit_app;
