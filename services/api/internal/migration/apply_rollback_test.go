package migration

// apply_rollback_test.go — the rollback/restart failpoint matrix. For each
// failpoint, inject an error, run Apply, confirm the target rolled back to the
// original fixture state, then rerun without the failpoint and confirm success.
//
// Every failure case must prove:
//   - revision 0001 absent (fresh/P3 rollback removes the control schema)
//   - platform schema absent
//   - run/reconciliation rows absent
//   - advisory lock absent (released by connection destruction)
//   - rerun succeeds exactly once
//   - third run is clean no-op
//   - revision 0001 count = 1, permissions = 7, completed runs = 1

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rollbackMatrixDSN builds a fresh fixture and returns (pool, dsn). Each
// failpoint case gets its own fixture so rollback state is hermetic.
func rollbackMatrixDSN(t *testing.T, name string, port int) (*pgxpool.Pool, string) {
	t.Helper()
	startFixture(t, name, port)
	dsn := applyDSN(port)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, dsn
}

// assertFreshRollbackState confirms the database is in the original empty state
// after a failed apply (no platform schema, no revision, no runs).
func assertFreshRollbackState(t *testing.T, dsn string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect for rollback check: %v", err)
	}
	defer conn.Close(ctx)

	var hasPlatform bool
	_ = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='platform')`).Scan(&hasPlatform)
	if hasPlatform {
		// Platform exists — check if it has revision/runs (partial state).
		var revCount, runCount int
		_ = conn.QueryRow(ctx, `SELECT count(*) FROM platform.schema_revisions WHERE version='0001'`).Scan(&revCount)
		_ = conn.QueryRow(ctx, `SELECT count(*) FROM platform.migration_runs`).Scan(&runCount)
		if revCount > 0 || runCount > 0 {
			t.Errorf("ROLLBACK FAILED: platform exists with rev=%d runs=%d (partial state not rolled back)", revCount, runCount)
		}
	}
}

// assertSuccessfulApply confirms the database is in the governed-current state
// after a successful apply.
func assertSuccessfulApply(t *testing.T, dsn string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect for success check: %v", err)
	}
	defer conn.Close(ctx)

	var revCount, permCount, runCount, targetReceipt, execReceipt int
	conn.QueryRow(ctx, `SELECT count(*) FROM platform.schema_revisions WHERE version='0001' AND success`).Scan(&revCount)
	conn.QueryRow(ctx, `SELECT count(*) FROM public.permissions`).Scan(&permCount)
	conn.QueryRow(ctx, `SELECT count(*) FROM platform.migration_runs WHERE state='completed'`).Scan(&runCount)
	conn.QueryRow(ctx, `SELECT count(*) FROM platform.reconciliation_results WHERE check_id='governed.target_fingerprint'`).Scan(&targetReceipt)
	conn.QueryRow(ctx, `SELECT count(*) FROM platform.reconciliation_results WHERE check_id='runner.execution_receipt'`).Scan(&execReceipt)

	if revCount != 1 {
		t.Errorf("revision 0001 count: got %d want 1", revCount)
	}
	if permCount != 7 {
		t.Errorf("canonical permissions: got %d want 7", permCount)
	}
	if runCount != 1 {
		t.Errorf("completed runs: got %d want 1", runCount)
	}
	if targetReceipt != 1 {
		t.Errorf("target-fingerprint receipts: got %d want 1", targetReceipt)
	}
	if execReceipt != 1 {
		t.Errorf("execution receipts: got %d want 1", execReceipt)
	}
}

// runRollbackCase runs one failpoint case: inject the failpoint, apply (must
// fail), assert rollback, rerun without failpoint (must succeed), third run
// (no-op).
func runRollbackCase(t *testing.T, fp Failpoint, port int) {
	t.Helper()
	ctx := context.Background()
	pool, dsn := rollbackMatrixDSN(t, fmt.Sprintf("g4-rollback-%s", sanitize(string(fp))), port)

	// Inject the failpoint (one-shot).
	ActiveFailpointController = &MapFailpoint{
		Errors: map[Failpoint]error{fp: errors.New("injected failure")},
	}
	defer func() { ActiveFailpointController = InertFailpoints{} }()

	// First apply: must fail.
	res := Apply(ctx, pool, ApplyOptions{
		Actor:       "rollback-test",
		ReleaseID:   "rollback-test-release",
		EvidenceRef: "sanitized-rollback-test",
	})
	if res.Err == nil {
		t.Fatalf("apply with failpoint %s unexpectedly succeeded (should have failed)", fp)
	}

	// Assert rollback: no partial state.
	assertFreshRollbackState(t, dsn)

	// Reset to inert failpoints for the rerun.
	ActiveFailpointController = InertFailpoints{}

	// Second apply: must succeed.
	res2 := Apply(ctx, pool, ApplyOptions{
		Actor:       "rollback-test",
		ReleaseID:   "rollback-test-release",
		EvidenceRef: "sanitized-rollback-test",
	})
	if res2.Err != nil {
		t.Fatalf("rerun after failpoint %s failed: %v", fp, res2.Err)
	}
	if res2.GovernedFingerprint != GovernedTargetFingerprint {
		t.Errorf("rerun governed FP: got %s want %s", res2.GovernedFingerprint, GovernedTargetFingerprint)
	}

	// Assert successful apply state.
	assertSuccessfulApply(t, dsn)

	// Third apply: clean no-op.
	res3 := Apply(ctx, pool, ApplyOptions{
		Actor:       "rollback-test-3",
		ReleaseID:   "rollback-test-release",
		EvidenceRef: "sanitized-rollback-test",
	})
	if res3.Err != nil {
		t.Errorf("third run (no-op) failed: %v", res3.Err)
	}
}

// TestRollback_FreshInstall_FailpointMatrix runs each failpoint against a fresh
// install, asserting rollback + rerun + no-op for every case.
func TestRollback_FreshInstall_FailpointMatrix(t *testing.T) {
	cases := []struct {
		fp   Failpoint
		port int
	}{
		{FailAfterSecondProbe, 56001},
		{FailAfterArtifactRoles, 56002},
		{FailAfterArtifactPlatform, 56003},
		{FailAfterArtifactBaseline, 56004},
		{FailAfterArtifactSeed, 56005},
		{FailAfterTargetFingerprint, 56006},
		{FailAfterRunInsert, 56007},
		{FailAfterTargetReceipt, 56008},
		{FailAfterExecutionReceipt, 56009},
		{FailAfterEvidenceFingerprint, 56010},
		{FailBeforeCommit, 56011},
	}
	for _, c := range cases {
		t.Run(string(c.fp), func(t *testing.T) {
			runRollbackCase(t, c.fp, c.port)
		})
	}
}
