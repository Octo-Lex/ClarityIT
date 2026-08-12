package migration

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

var expectedKernelTables = []string{
	"approval_decisions", "audit_records", "authority_grants", "case_resource_refs", "cases",
	"dispatch_records", "evidence_manifests", "execution_attempts", "grant_approval_refs", "inbox_messages",
	"message_quarantine", "observations", "operation_packet_baseline_refs", "operation_packets", "outbox_messages",
	"outcome_decisions", "policy_decisions", "principal_refs", "provider_bindings", "provider_receipts",
	"resource_health_contract_refs", "resource_owner_refs", "resources", "result_claim_receipt_refs", "result_claims",
	"verification_evidence", "verification_observation_refs", "verification_specs", "verifications",
}
var expectedCompatTables = []string{"backfill_checkpoints", "feature_flags", "identity_mappings", "writer_ownership"}
var expectedKernelFunctions = []string{"is_uuid_v7", "prevent_packet_payload_mutation", "prevent_packet_return_to_draft", "reject_immutable_mutation"}

type forwardVerifierQuery interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// InspectForward performs the complete Stage-B read path as the real migration
// identity: enter a read-only transaction, SET ROLE clarityit_owner, pin
// canonical formatting, then validate exact revision ancestry and target state.
func InspectForward(ctx context.Context, conn *pgx.Conn) (ForwardInspection, error) {
	cat, err := ForwardCatalog()
	if err != nil {
		return ForwardInspection{}, err
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return ForwardInspection{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE clarityit_owner`); err != nil {
		return ForwardInspection{}, fmt.Errorf("forward inspection privilege boundary: %w", err)
	}
	if err := pinForwardManifestSession(ctx, tx); err != nil {
		return ForwardInspection{}, err
	}

	rows, err := readForwardHistory(ctx, tx)
	if err != nil {
		return ForwardInspection{}, err
	}
	state, err := validateForwardHistory(rows, cat)
	if err != nil {
		return ForwardInspection{}, err
	}
	ins := ForwardInspection{PackageDigest: ForwardPackageDigest(cat)}
	if ins.PackageDigest != ForwardPackageSHA256 {
		return ForwardInspection{}, fmt.Errorf("%w: package=%s frozen=%s", ErrForwardPackaging, ins.PackageDigest, ForwardPackageSHA256)
	}
	if state == "foundation" {
		gfp, ok, err := tryGovernedFingerprint(ctx, tx)
		if err != nil || !ok || gfp != GovernedTargetFingerprint {
			if err != nil {
				return ForwardInspection{}, fmt.Errorf("%w: governed capture: %v", ErrForwardFoundation, err)
			}
			return ForwardInspection{}, fmt.Errorf("%w: computed=%s expected=%s", ErrForwardFoundation, gfp, GovernedTargetFingerprint)
		}
		ins.FoundationOnly = true
		ins.CurrentVersion = "0001"
		return ins, nil
	}
	ins.Current = true
	ins.CurrentVersion = ForwardTargetVersion
	digest, err := verifyForwardTargetQuery(ctx, tx)
	if err != nil {
		return ForwardInspection{}, err
	}
	ins.ManifestDigest = digest
	return ins, nil
}

func verifyForwardTarget(ctx context.Context, conn *pgx.Conn) (string, error) {
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE clarityit_owner`); err != nil {
		return "", fmt.Errorf("forward verify privilege boundary: %w", err)
	}
	if err := pinForwardManifestSession(ctx, tx); err != nil {
		return "", err
	}
	return verifyForwardTargetQuery(ctx, tx)
}

func verifyForwardTargetTx(ctx context.Context, tx pgx.Tx) (string, error) {
	if err := pinForwardManifestSession(ctx, tx); err != nil {
		return "", err
	}
	return verifyForwardTargetQuery(ctx, tx)
}

func pinForwardManifestSession(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return fmt.Errorf("pin forward manifest timezone: %w", err)
	}
	if _, err := tx.Exec(ctx, `SET LOCAL DateStyle = 'ISO, YMD'`); err != nil {
		return fmt.Errorf("pin forward manifest datestyle: %w", err)
	}
	return nil
}

func verifyForwardTargetQuery(ctx context.Context, q forwardVerifierQuery) (string, error) {
	kernelTables, err := schemaTables(ctx, q, "kernel")
	if err != nil {
		return "", err
	}
	compatTables, err := schemaTables(ctx, q, "compat")
	if err != nil {
		return "", err
	}
	if !sameStrings(kernelTables, expectedKernelTables) {
		return "", fmt.Errorf("%w: kernel table inventory mismatch got=%v want=%v", ErrForwardManifest, sortedCopy(kernelTables), sortedCopy(expectedKernelTables))
	}
	if !sameStrings(compatTables, expectedCompatTables) {
		return "", fmt.Errorf("%w: compat table inventory mismatch got=%v want=%v", ErrForwardManifest, sortedCopy(compatTables), sortedCopy(expectedCompatTables))
	}
	funcs, err := kernelFunctions(ctx, q)
	if err != nil {
		return "", err
	}
	if !sameStrings(funcs, expectedKernelFunctions) {
		return "", fmt.Errorf("%w: kernel function inventory mismatch got=%v want=%v", ErrForwardManifest, sortedCopy(funcs), sortedCopy(expectedKernelFunctions))
	}

	var badOwners int
	if err := q.QueryRow(ctx, `
		SELECT count(*) FROM pg_class c
		JOIN pg_namespace n ON n.oid=c.relnamespace
		JOIN pg_roles r ON r.oid=c.relowner
		WHERE n.nspname IN ('kernel','compat') AND c.relkind IN ('r','p') AND r.rolname <> 'clarityit_owner'`).Scan(&badOwners); err != nil {
		return "", err
	}
	if badOwners != 0 {
		return "", fmt.Errorf("%w: %d tables not owned by clarityit_owner", ErrForwardManifest, badOwners)
	}

	var principalUUIDv7 bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM pg_constraint c
		JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace
		WHERE n.nspname='kernel' AND t.relname='principal_refs' AND c.conname='principal_refs_uuid_v7')`).Scan(&principalUUIDv7); err != nil {
		return "", err
	}
	if !principalUUIDv7 {
		return "", fmt.Errorf("%w: principal UUIDv7 constraint missing", ErrForwardManifest)
	}

	var inboxDelete bool
	if err := q.QueryRow(ctx, `SELECT has_table_privilege('clarityit_app','kernel.inbox_messages','DELETE')`).Scan(&inboxDelete); err != nil {
		return "", err
	}
	if inboxDelete {
		return "", fmt.Errorf("%w: clarityit_app has DELETE on durable inbox", ErrForwardManifest)
	}

	var flagCount, enabledCount int
	if err := q.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE enabled) FROM compat.feature_flags
		WHERE flag_name IN ('wp01.kernel.enabled','wp01.effect_broker.fake_route.enabled','wp01.live_provider_mutation.enabled')`).Scan(&flagCount, &enabledCount); err != nil {
		return "", err
	}
	if flagCount != 3 || enabledCount != 0 {
		return "", fmt.Errorf("%w: expected three disabled WP-01 flags", ErrForwardManifest)
	}
	var liveForbidden bool
	if err := q.QueryRow(ctx, `SELECT COALESCE((config->>'forbidden')::boolean,false)
		FROM compat.feature_flags WHERE flag_name='wp01.live_provider_mutation.enabled' AND scope_key='*'`).Scan(&liveForbidden); err != nil {
		return "", err
	}
	if !liveForbidden {
		return "", fmt.Errorf("%w: live-provider mutation is not explicitly forbidden", ErrForwardManifest)
	}

	var writerV1, writerV2 int
	if err := q.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE object_family='v1_existing_product_families' AND authoritative_writer='v1' AND effective_to IS NULL),
		count(*) FILTER (WHERE object_family='v2_kernel_objects' AND authoritative_writer='v2' AND effective_to IS NULL)
		FROM compat.writer_ownership`).Scan(&writerV1, &writerV2); err != nil {
		return "", err
	}
	if writerV1 != 1 || writerV2 != 1 {
		return "", fmt.Errorf("%w: writer ownership registry mismatch", ErrForwardManifest)
	}

	digest, err := forwardTargetManifestDigest(ctx, q)
	if err != nil {
		return "", fmt.Errorf("%w: build catalog manifest: %v", ErrForwardManifest, err)
	}
	if ForwardTargetManifestSHA256 != "" && digest != ForwardTargetManifestSHA256 {
		return "", fmt.Errorf("%w: computed=%s frozen=%s", ErrForwardManifest, digest, ForwardTargetManifestSHA256)
	}
	return digest, nil
}

func schemaTables(ctx context.Context, q forwardVerifierQuery, schema string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema=$1 AND table_type='BASE TABLE' ORDER BY table_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func kernelFunctions(ctx context.Context, q forwardVerifierQuery) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='kernel' ORDER BY p.proname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func sameStrings(got, want []string) bool {
	g := sortedCopy(got)
	w := sortedCopy(want)
	if len(g) != len(w) {
		return false
	}
	for i := range g {
		if g[i] != w[i] {
			return false
		}
	}
	return true
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
