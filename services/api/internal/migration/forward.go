package migration

// forward.go — WP-01 post-0001 forward-series support.
//
// This file deliberately wraps rather than rewrites the accepted WP-00
// version-0001 executor. Stage A remains Apply()/Preflight exactly as accepted.
// Stage B starts only after an exact successful revision 0001 exists.
//
// G1 uses one bounded atomic batch for 0002+0003+0004. Therefore the only
// accepted persisted histories are:
//   - exact 0001 foundation; or
//   - exact 0001,0002,0003,0004 current state.
// An intermediate forward prefix is contradictory/manual state and fails closed.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ForwardTargetVersion = "0004"
	// These two identities are frozen after the first exact integrated G1
	// rehearsal. Empty means the structural verifier still runs but identity
	// freeze has not yet been completed; G1 cannot close while either is empty.
	ForwardPackageSHA256        = ""
	ForwardTargetManifestSHA256 = ""
)

var (
	ErrForwardHistory       = errors.New("forward revision history is inconsistent")
	ErrForwardPackaging     = errors.New("forward package identity mismatch")
	ErrForwardManifest      = errors.New("forward target manifest mismatch")
	ErrForwardIntermediate  = errors.New("intermediate forward revision state is not an accepted G1 checkpoint")
	ErrForwardFoundation    = errors.New("WP-00 foundation does not match the accepted pre-forward identity")
)

type ForwardRevision struct {
	Version string
	Name    string
	Asset   assets.AssetName
	Checksum string
}

type forwardLedgerRow struct {
	Version  string
	Name     string
	Checksum string
	Success  bool
}

type ForwardInspection struct {
	CurrentVersion string
	Current        bool
	FoundationOnly bool
	ManifestDigest string
	PackageDigest  string
}

// ForwardCatalog returns the exact ordered post-0001 catalog. Expected
// checksums come from the package-wide FrozenDigest map so VerifyAll and the
// forward executor cannot disagree about accepted bytes.
func ForwardCatalog() ([]ForwardRevision, error) {
	specs := []struct {
		version string
		name    string
		asset   assets.AssetName
	}{
		{"0002", "wp01-kernel-foundation", assets.AssetForward0002},
		{"0003", "wp01-kernel-integrity-hardening", assets.AssetForward0003},
		{"0004", "wp01-packet-immutability-barrier", assets.AssetForward0004},
	}
	out := make([]ForwardRevision, 0, len(specs))
	for _, s := range specs {
		want, ok := FrozenDigest[s.asset]
		if !ok || want == "" {
			return nil, fmt.Errorf("%w: no frozen SHA-256 for %s", ErrForwardPackaging, s.asset)
		}
		got, err := assets.SHA256(s.asset)
		if err != nil {
			return nil, fmt.Errorf("%w: hash %s: %v", ErrForwardPackaging, s.asset, err)
		}
		if got != want {
			return nil, fmt.Errorf("%w: %s embedded=%s frozen=%s", ErrForwardPackaging, s.asset, got, want)
		}
		out = append(out, ForwardRevision{Version: s.version, Name: s.name, Asset: s.asset, Checksum: want})
	}
	if err := validateForwardCatalog(out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateForwardCatalog(cat []ForwardRevision) error {
	if len(cat) != 3 {
		return fmt.Errorf("%w: expected 3 G1 revisions, got %d", ErrForwardPackaging, len(cat))
	}
	for i, r := range cat {
		want := fmt.Sprintf("%04d", i+2)
		if r.Version != want {
			return fmt.Errorf("%w: catalog[%d]=%s want %s", ErrForwardPackaging, i, r.Version, want)
		}
		if r.Name == "" || r.Checksum == "" {
			return fmt.Errorf("%w: incomplete catalog entry %s", ErrForwardPackaging, r.Version)
		}
	}
	return nil
}

// ForwardPackageDigest binds ordered version/name/checksum tuples. It does not
// include Git metadata or runtime data, so all builds carrying identical
// accepted forward bytes reproduce the same value.
func ForwardPackageDigest(cat []ForwardRevision) string {
	h := sha256.New()
	h.Write([]byte("clarityit-wp01-forward-package-v1\x00"))
	for _, r := range cat {
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00", r.Version, r.Name, r.Checksum)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// InspectForward is read-only and recognizes post-forward databases without
// invoking the WP-00 classifier (which correctly treats post-forward structure
// as different from pre-forward 9881c93e...).
func InspectForward(ctx context.Context, conn *pgx.Conn) (ForwardInspection, error) {
	cat, err := ForwardCatalog()
	if err != nil {
		return ForwardInspection{}, err
	}
	rows, err := readForwardHistory(ctx, conn)
	if err != nil {
		return ForwardInspection{}, err
	}
	state, err := validateForwardHistory(rows, cat)
	if err != nil {
		return ForwardInspection{}, err
	}
	ins := ForwardInspection{PackageDigest: ForwardPackageDigest(cat)}
	if ForwardPackageSHA256 != "" && ins.PackageDigest != ForwardPackageSHA256 {
		return ForwardInspection{}, fmt.Errorf("%w: package=%s frozen=%s", ErrForwardPackaging, ins.PackageDigest, ForwardPackageSHA256)
	}
	switch state {
	case "foundation":
		ins.FoundationOnly = true
		ins.CurrentVersion = "0001"
	case "current":
		ins.Current = true
		ins.CurrentVersion = ForwardTargetVersion
		digest, err := verifyForwardTarget(ctx, conn)
		if err != nil {
			return ForwardInspection{}, err
		}
		ins.ManifestDigest = digest
	}
	return ins, nil
}

// ApplyAll establishes exact 0001 through the unchanged accepted executor when
// necessary, then runs Stage B. It is the WP-01 migration orchestration entry.
func ApplyAll(ctx context.Context, pool *pgxpool.Pool, opts ApplyOptions) ApplyResult {
	has0001, err := hasExactRevision0001(ctx, pool)
	if err != nil {
		return ApplyResult{StartedAt: time.Now(), CompletedAt: time.Now(), Err: fmt.Errorf("inspect 0001: %w", err)}
	}
	if !has0001 {
		base := Apply(ctx, pool, opts)
		if base.Err != nil {
			return base
		}
		// Stage A has just established the accepted 0001 foundation. Do not use
		// its governed-current no-op as the final WP-01 result; continue to B.
	}
	return ApplyForward(ctx, pool, opts)
}

// ApplyForward upgrades an exact WP-00 foundation to the complete WP-01 G1
// forward target in one transaction, or verifies/no-ops a complete current DB.
func ApplyForward(ctx context.Context, pool *pgxpool.Pool, opts ApplyOptions) ApplyResult {
	res := ApplyResult{StartedAt: time.Now(), Path: Path("forward")}
	finish := func(err error) ApplyResult {
		res.Err = err
		res.CompletedAt = time.Now()
		res.ExecutionMs = res.CompletedAt.Sub(res.StartedAt).Milliseconds()
		return res
	}

	producingCommit, err := ResolveProducingCommit()
	if err != nil {
		return finish(fmt.Errorf("forward provenance: %w", err))
	}
	if _, err := VerifyAll(); err != nil {
		return finish(fmt.Errorf("forward package verify: %w", err))
	}
	cat, err := ForwardCatalog()
	if err != nil {
		return finish(err)
	}
	pkgDigest := ForwardPackageDigest(cat)
	if ForwardPackageSHA256 != "" && pkgDigest != ForwardPackageSHA256 {
		return finish(fmt.Errorf("%w: package=%s frozen=%s", ErrForwardPackaging, pkgDigest, ForwardPackageSHA256))
	}

	conn, err := AcquirePinnedConn(ctx, pool)
	if err != nil {
		return finish(err)
	}
	defer conn.Release()
	var lockState LockState
	if err := AcquireMigrationLock(ctx, conn, &lockState); err != nil {
		res.Code = LockDiagnosticCode(err)
		return finish(err)
	}
	defer ReleaseMigrationLock(context.Background(), &lockState)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return finish(fmt.Errorf("forward begin: %w", err))
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE clarityit_owner"); err != nil {
		return finish(fmt.Errorf("forward set owner role: %w", err))
	}

	rows, err := readForwardHistory(ctx, tx)
	if err != nil {
		return finish(err)
	}
	state, err := validateForwardHistory(rows, cat)
	if err != nil {
		return finish(err)
	}

	if state == "current" {
		digest, err := verifyForwardTargetTx(ctx, tx)
		if err != nil {
			return finish(err)
		}
		res.Path = PathNoOp
		res.Code = CodeOK
		res.Class = ClassGovernedCurrent
		res.TargetVersion = ForwardTargetVersion
		res.ForwardManifest = digest
		_ = tx.Rollback(ctx)
		res.CompletedAt = time.Now()
		res.ExecutionMs = res.CompletedAt.Sub(res.StartedAt).Milliseconds()
		return res
	}

	// Foundation-only DB must still be the exact accepted WP-00 structure.
	gfp, ok, err := tryGovernedFingerprint(ctx, tx)
	if err != nil || !ok || gfp != GovernedTargetFingerprint {
		if err != nil {
			return finish(fmt.Errorf("%w: governed capture: %v", ErrForwardFoundation, err))
		}
		return finish(fmt.Errorf("%w: computed=%s expected=%s", ErrForwardFoundation, gfp, GovernedTargetFingerprint))
	}

	res.DDLStarted = true
	for _, r := range cat {
		body, err := assets.Bytes(r.Asset)
		if err != nil {
			return finish(fmt.Errorf("forward asset %s: %w", r.Version, err))
		}
		if err := validateForwardSQL(body); err != nil {
			return finish(fmt.Errorf("forward SQL %s: %w", r.Version, err))
		}
		revStarted := time.Now()
		if err := execSimpleProtocolDrained(ctx, tx, string(body)); err != nil {
			return finish(fmt.Errorf("forward exec %s: %w", r.Version, err))
		}
		execMS := time.Since(revStarted).Milliseconds()
		if _, err := tx.Exec(ctx, `
			INSERT INTO platform.schema_revisions
				(version, name, checksum, source_commit, applied_at, applied_by, execution_ms, success)
			VALUES ($1,$2,$3,$4,now(),session_user,$5,true)`,
			r.Version, r.Name, r.Checksum, producingCommit, execMS); err != nil {
			return finish(fmt.Errorf("forward ledger %s: %w", r.Version, err))
		}
	}

	manifestDigest, err := verifyForwardTargetTx(ctx, tx)
	if err != nil {
		return finish(err)
	}
	if ForwardTargetManifestSHA256 != "" && manifestDigest != ForwardTargetManifestSHA256 {
		return finish(fmt.Errorf("%w: computed=%s frozen=%s", ErrForwardManifest, manifestDigest, ForwardTargetManifestSHA256))
	}

	runID := newRunID()
	res.RunID = runID
	sourceProfile := foundationSourceProfile(ctx, tx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO platform.migration_runs
			(run_id, source_profile_id, target_version, state, started_at, completed_at, release_id, evidence_ref)
		VALUES ($1,NULLIF($2,''),$3,'completed',$4,now(),$5,NULLIF($6,''))`,
		runID, sourceProfile, ForwardTargetVersion, res.StartedAt, opts.ReleaseID, opts.EvidenceRef); err != nil {
		return finish(fmt.Errorf("forward migration_run: %w", err))
	}
	if err := AppendReconciliation(ctx, tx, runID, "wp01.forward_manifest", "kernel+compat",
		map[string]any{"target_version": ForwardTargetVersion, "package_digest": pkgDigest, "manifest_digest": ForwardTargetManifestSHA256},
		map[string]any{"target_version": ForwardTargetVersion, "package_digest": pkgDigest, "manifest_digest": manifestDigest},
		"pass", opts.EvidenceRef); err != nil {
		return finish(fmt.Errorf("forward reconciliation: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return finish(fmt.Errorf("forward commit: %w", err))
	}
	committed = true
	res.Code = CodeOK
	res.Class = ClassGovernedCurrent
	res.TargetVersion = ForwardTargetVersion
	res.ForwardManifest = manifestDigest
	return finish(nil)
}

func hasExactRevision0001(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Release()
	var hasTable bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema='platform' AND table_name='schema_revisions')`).Scan(&hasTable); err != nil {
		return false, err
	}
	if !hasTable {
		return false, nil
	}
	var checksum string
	var success bool
	err = conn.QueryRow(ctx, `SELECT checksum, success FROM platform.schema_revisions WHERE version='0001'`).Scan(&checksum, &success)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// A contradictory existing 0001 is not treated as absent (which would pass
	// it into Stage A); fail here so no alternate path can overwrite/recreate it.
	if checksum != BaselineChecksum || !success {
		return false, fmt.Errorf("%w: revision 0001 checksum/success mismatch", ErrForwardHistory)
	}
	return true, nil
}

type forwardQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readForwardHistory(ctx context.Context, q forwardQueryer) ([]forwardLedgerRow, error) {
	rows, err := q.Query(ctx, `SELECT version, name, checksum, success FROM platform.schema_revisions ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read revision history: %w", err)
	}
	defer rows.Close()
	var out []forwardLedgerRow
	for rows.Next() {
		var r forwardLedgerRow
		if err := rows.Scan(&r.Version, &r.Name, &r.Checksum, &r.Success); err != nil {
			return nil, fmt.Errorf("scan revision history: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func validateForwardHistory(rows []forwardLedgerRow, cat []ForwardRevision) (string, error) {
	if len(rows) == 0 || rows[0].Version != "0001" || rows[0].Checksum != BaselineChecksum || !rows[0].Success {
		return "", fmt.Errorf("%w: exact successful 0001 ancestry required", ErrForwardHistory)
	}
	if len(rows) == 1 {
		return "foundation", nil
	}
	if len(rows) != 1+len(cat) {
		return "", fmt.Errorf("%w: got %d rows; expected 1 or %d", ErrForwardIntermediate, len(rows), 1+len(cat))
	}
	for i, expected := range cat {
		got := rows[i+1]
		if got.Version != expected.Version || got.Checksum != expected.Checksum || !got.Success {
			return "", fmt.Errorf("%w: %s mismatch", ErrForwardHistory, expected.Version)
		}
	}
	return "current", nil
}

func validateForwardSQL(body []byte) error {
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "\\") {
			return fmt.Errorf("psql meta-command forbidden: %q", line)
		}
		u := strings.ToUpper(line)
		if u == "BEGIN;" || u == "COMMIT;" {
			return fmt.Errorf("runner-owned transaction boundary found: %s", line)
		}
	}
	return nil
}

func foundationSourceProfile(ctx context.Context, tx pgx.Tx) string {
	var profile string
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(source_profile_id,'')
		FROM platform.migration_runs
		WHERE target_version='0001' AND state='completed'
		ORDER BY completed_at DESC NULLS LAST, started_at DESC
		LIMIT 1`).Scan(&profile)
	return profile
}

var expectedKernelTables = []string{
	"audit_records", "approval_decisions", "authority_grants", "case_resource_refs", "cases",
	"dispatch_records", "evidence_manifests", "execution_attempts", "grant_approval_refs", "inbox_messages",
	"message_quarantine", "observations", "operation_packet_baseline_refs", "operation_packets", "outbox_messages",
	"policy_decisions", "principal_refs", "provider_bindings", "provider_receipts", "resource_health_contract_refs",
	"resource_owner_refs", "resources", "result_claim_receipt_refs", "result_claims", "verification_evidence",
	"verification_observation_refs", "verification_specs", "verifications", "outcome_decisions",
}

var expectedCompatTables = []string{"backfill_checkpoints", "feature_flags", "identity_mappings", "writer_ownership"}
var expectedKernelFunctions = []string{"is_uuid_v7", "prevent_packet_payload_mutation", "prevent_packet_return_to_draft", "reject_immutable_mutation"}

// verifyForwardTargetTx checks the G1-owned target invariant set and returns a
// deterministic manifest digest over the verified semantic inventory.
func verifyForwardTargetTx(ctx context.Context, tx pgx.Tx) (string, error) {
	return verifyForwardTargetQuery(ctx, tx)
}

func verifyForwardTarget(ctx context.Context, conn *pgx.Conn) (string, error) {
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	return verifyForwardTargetQuery(ctx, tx)
}

type forwardVerifierQuery interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
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
		return "", fmt.Errorf("%w: kernel tables got=%v want=%v", ErrForwardManifest, kernelTables, sortedCopy(expectedKernelTables))
	}
	if !sameStrings(compatTables, expectedCompatTables) {
		return "", fmt.Errorf("%w: compat tables got=%v want=%v", ErrForwardManifest, compatTables, sortedCopy(expectedCompatTables))
	}
	funcs, err := kernelFunctions(ctx, q)
	if err != nil {
		return "", err
	}
	if !sameStrings(funcs, expectedKernelFunctions) {
		return "", fmt.Errorf("%w: kernel functions got=%v want=%v", ErrForwardManifest, funcs, sortedCopy(expectedKernelFunctions))
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
		return "", fmt.Errorf("%w: %d kernel/compat tables not owned by clarityit_owner", ErrForwardManifest, badOwners)
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
		return "", fmt.Errorf("%w: expected three disabled WP-01 feature flags", ErrForwardManifest)
	}
	var liveForbidden bool
	if err := q.QueryRow(ctx, `SELECT COALESCE((config->>'forbidden')::boolean,false) FROM compat.feature_flags WHERE flag_name='wp01.live_provider_mutation.enabled' AND scope_key='*'`).Scan(&liveForbidden); err != nil {
		return "", err
	}
	if !liveForbidden {
		return "", fmt.Errorf("%w: live-provider mutation flag is not explicitly forbidden", ErrForwardManifest)
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

	manifestLines := []string{"wp01-g1-schema-manifest-v1"}
	for _, t := range sortedCopy(kernelTables) {
		manifestLines = append(manifestLines, "table:kernel."+t)
	}
	for _, t := range sortedCopy(compatTables) {
		manifestLines = append(manifestLines, "table:compat."+t)
	}
	for _, f := range sortedCopy(funcs) {
		manifestLines = append(manifestLines, "function:kernel."+f)
	}
	manifestLines = append(manifestLines,
		"owner:kernel+compat=clarityit_owner",
		"privilege:clarityit_app:kernel.inbox_messages:DELETE=false",
		"flag:wp01.kernel.enabled=false",
		"flag:wp01.effect_broker.fake_route.enabled=false",
		"flag:wp01.live_provider_mutation.enabled=false;forbidden=true",
		"writer:v1_existing_product_families=v1",
		"writer:v2_kernel_objects=v2",
	)
	sum := sha256.Sum256([]byte(strings.Join(manifestLines, "\n") + "\n"))
	digest := hex.EncodeToString(sum[:])
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
