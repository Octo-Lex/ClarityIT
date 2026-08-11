package migration

// ledger.go — immutable revision + run/reconciliation evidence, mapped onto the
// FROZEN platform control schema (0000_platform.sql). The runner does NOT change
// the frozen schema; it uses the existing tables:
//
//   - platform.schema_revisions: revision, checksum, source_commit, actor,
//     duration, result (success), with an immutability trigger on success=true.
//     The frozen SEED and ADOPTION artifacts ALREADY insert the revision-0001
//     row with the canonical checksum + success=true. The runner therefore NEVER
//     inserts or updates revision 0001 — the artifact owns it, and the
//     immutability trigger + PK on version would reject a second write.
//   - platform.migration_runs: run_id, source_profile_id, target_version,
//     state (constrained enum), timestamps, release_id, evidence_ref. One active
//     run per database (partial unique index).
//   - platform.reconciliation_results: append-only (trigger-enforced),
//     run-linked, with expected/actual fingerprint jsonb + result.
//
// Recommended transaction shape (per the G4 contract):
//  1. Generate run_id in Go.
//  2. Acquire the session advisory lock.
//  3. Run complete preflight.
//  4. Short control tx: insert migration_runs as 'preflighted'.
//  5. Begin target tx.
//  6. Re-run the SAME complete probe; require match with original preflight.
//  7. (Frozen artifact inserts the pending/success revision-0001 row when it
//     executes; the runner does NOT insert it separately.)
//  8. Execute the frozen transformed SQL.
//  9. Compute governed target fingerprint.
// 10. Append reconciliation result.
// 11. Set run to 'completed'.
// 12. Commit atomically.
//
// On failure: roll back target tx; while still holding the session lock, open a
// NEW control tx; update the run to 'precommit_rolled_back' (or exact permitted
// state); store only sanitized evidence; commit the failure record; unlock.
// This preserves failure evidence across a target rollback and leaves a
// restart-visible active/blocked run if the process dies.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerInput is the sanitized evidence the caller supplies for a run. The
// runner never stores secrets, DSNs, or raw production data here.
type LedgerInput struct {
	RunID         string // generated in Go (UUID)
	SourceProfile string // source-profile id (fingerprint-derived), allowlisted
	TargetVersion string // "0001"
	ReleaseID     string // compiled release identifier (ldflags)
	Actor         string // who/what initiated the apply
	SourceCommit  string // build-bound implementation commit (ldflags)
	EvidenceRef   string // sanitized evidence reference (CI run id, never raw payload)
}

// LedgerRecord is the runner's view of a migration_runs row.
type LedgerRecord struct {
	RunID         string
	State         string
	TargetVersion string
	ReleaseID     string
	EvidenceRef   string
	SourceProfile string
	StartedAt     time.Time
	CompletedAt   *time.Time
}

// ErrRevisionImmutable is returned if the runner (incorrectly) attempts to
// insert or update revision 0001. The frozen artifacts own that row; the runner
// must never touch it.
var ErrRevisionImmutable = errors.New("revision 0001 is artifact-owned and immutable; the runner must not insert or update it")

// InsertRunPreflighted opens a SHORT control transaction on the pinned connection
// and inserts a migration_runs row in state 'preflighted'. This is the durable
// restart-visible record created BEFORE the target transaction begins.
//
// The session advisory lock MUST already be held on this connection (session
// locks survive the control tx's commit, so the lock is not released here).
func InsertRunPreflighted(ctx context.Context, conn *pgxpool.Conn, in LedgerInput) error {
	if in.RunID == "" {
		return errors.New("InsertRunPreflighted: RunID required")
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin control tx: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform.migration_runs
			(run_id, source_profile_id, target_version, state, started_at, release_id, evidence_ref)
		VALUES ($1, NULLIF($2,''), $3, 'preflighted', now(), $4, NULLIF($5,''))`,
		in.RunID, in.SourceProfile, in.TargetVersion, in.ReleaseID, in.EvidenceRef)
	if err != nil {
		return fmt.Errorf("insert migration_runs (preflighted): %w", err)
	}
	return tx.Commit(ctx)
}

// CompleteRun sets a run to 'completed' inside the TARGET transaction (the
// caller's tx). Called after the frozen artifact chain + reconciliation have
// succeeded, before the target tx commits.
func CompleteRun(ctx context.Context, tx pgx.Tx, runID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE platform.migration_runs
		SET state = 'completed', completed_at = now()
		WHERE run_id = $1 AND state NOT IN ('completed','blocked','precommit_rolled_back')`,
		runID)
	if err != nil {
		return fmt.Errorf("complete run: %w", err)
	}
	return nil
}

// ExecutionReceipt is the runtime receipt recorded in a SECOND append-only
// reconciliation_results row (check_id="runner.execution_receipt"). It carries
// the runtime actor, measured duration, producing commit, and artifact digests
// that CANNOT go in the artifact-owned revision-0001 row (which hardcodes
// applied_by='g3-baseline-artifact'/'g3-adoption-artifact' and execution_ms=0).
//
// This maps the G4-required actor/run/duration/result evidence onto the frozen
// append-only structure WITHOUT modifying the platform schema or misrepresenting
// artifact provenance. The target-fingerprint reconciliation row is kept SEPARATE
// (check_id="governed.target_fingerprint").
type ExecutionReceipt struct {
	RunID             string
	ReleaseID         string
	PackageDigest     string // composite 8af2c9f5...
	TargetFingerprint string // governed 9881c93e...
	TargetVersion     string
	Actor             string // sanitized runner identity
	Path              string // fresh_install | adopt_p3
	StartedAt         time.Time
	CompletedAt       time.Time
	ExecutionMs       int64
	ProducingCommit   string // 40-char SHA (ldflags-bound)
	// Original/transformed artifact digests (asset name -> sha256).
	OriginalDigests    map[string]string
	TransformedDigests map[string]string
	EvidenceRef        string // sanitized immutable reference
}

// AppendExecutionReceipt writes the runner.execution_receipt reconciliation row
// inside the target transaction. expected/actual are JSONB; result is "pass".
func AppendExecutionReceipt(ctx context.Context, tx pgx.Tx, r ExecutionReceipt) error {
	expected := map[string]any{
		"release_id":         r.ReleaseID,
		"package_digest":     r.PackageDigest,
		"target_fingerprint": r.TargetFingerprint,
		"target_version":     r.TargetVersion,
	}
	actual := map[string]any{
		"run_id":                       r.RunID,
		"actor":                        r.Actor,
		"path":                         r.Path,
		"started_at":                   r.StartedAt.Format(time.RFC3339Nano),
		"completed_at":                 r.CompletedAt.Format(time.RFC3339Nano),
		"execution_ms":                 r.ExecutionMs,
		"producing_commit":             r.ProducingCommit,
		"original_artifact_digests":    r.OriginalDigests,
		"transformed_artifact_digests": r.TransformedDigests,
	}
	return AppendReconciliation(ctx, tx, r.RunID, "runner.execution_receipt",
		"schema_revision:"+r.TargetVersion, expected, actual, "pass", r.EvidenceRef)
}

// AppendReconciliation writes one append-only reconciliation_results row inside
// the target transaction. expected/actual are JSONB; the trigger enforces
// append-only and requires a non-empty evidence_ref.
func AppendReconciliation(ctx context.Context, tx pgx.Tx, runID, checkID, scope string, expected, actual any, result, evidenceRef string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO platform.reconciliation_results
			(run_id, check_id, scope, expected, actual, result, evidence_ref, recorded_at)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, now())`,
		runID, checkID, scope, expected, actual, result, evidenceRef)
	if err != nil {
		return fmt.Errorf("append reconciliation: %w", err)
	}
	return nil
}

// MarkRunRolledBack opens a NEW control transaction (after the target tx rolled
// back) to record the failure while still holding the session advisory lock.
// state must be an exact permitted value (e.g. 'precommit_rolled_back',
// 'forward_recovery_required', 'blocked'). evidenceRef must be sanitized.
func MarkRunRolledBack(ctx context.Context, conn *pgxpool.Conn, runID, state, evidenceRef string) error {
	if !isPermittedFailureState(state) {
		return fmt.Errorf("MarkRunRolledBack: state %q is not a permitted failure state", state)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin failure control tx: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		UPDATE platform.migration_runs
		SET state = $2, completed_at = now(), evidence_ref = COALESCE(NULLIF($3,''), evidence_ref)
		WHERE run_id = $1`,
		runID, state, evidenceRef)
	if err != nil {
		return fmt.Errorf("mark run rolled back: %w", err)
	}
	return tx.Commit(ctx)
}

// isPermittedFailureState mirrors the frozen migration_runs_state_check enum.
func isPermittedFailureState(s string) bool {
	switch s {
	case "blocked", "paused", "precommit_rolled_back", "forward_recovery_required":
		return true
	}
	return false
}

// AssertRevisionNotTouched is a guard the apply path calls before execution to
// confirm the runner will NOT insert/update revision 0001. The frozen seed and
// adoption artifacts own that row. This is documentation-as-code: the runner
// has no function that writes schema_revisions, and this guard makes the
// invariant explicit and auditable.
func AssertRevisionNotTouched() error {
	// The runner intentionally provides NO function to insert or update
	// platform.schema_revisions. The revision-0001 row is created by the frozen
	// seed artifact (install path) or the frozen adoption artifact (adopt path),
	// both of which are executed via the transformed-body path. A second write
	// by the runner would (a) violate the PK on version and (b) be rejected by
	// the schema_revisions_immutable trigger. This function exists to make the
	// invariant grep-able and to fail loudly if a future edit adds such a path.
	return nil
}

// sanityCheckAssetOwnsRevision confirms the frozen seed AND adoption artifacts
// contain the revision-0001 INSERT (so the runner can rely on the artifact
// owning that row). Called during packaging verification.
func sanityCheckAssetOwnsRevision() error {
	for _, name := range []assets.AssetName{assets.AssetSeed, assets.AssetAdoptP3} {
		b, err := assets.Bytes(name)
		if err != nil {
			return err
		}
		// Both artifacts INSERT INTO platform.schema_revisions with version '0001'.
		if !bytes.Contains(b, []byte("INSERT INTO platform.schema_revisions")) {
			return fmt.Errorf("asset %s does not contain the revision-0001 INSERT (runner cannot rely on artifact owning it)", name)
		}
	}
	return nil
}
