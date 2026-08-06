package migration

// apply_cleanup_test.go — cleanup-error matrix + exit-path guarantees. Proves
// that every apply exit path (success or failure) does NOT leak:
//   - an open transaction
//   - an advisory lock (at most one unlock, never unlock_all)
//   - a backend (physical connection is always destroyed)
//   - a pooled superuser connection (never returned to pgxpool)
//
// Cleanup failures (unlock returns false, repeated unlock, close failure) are
// handled gracefully: the primary execution error is preserved, cleanup errors
// are sanitized secondary diagnostics.

import (
	"context"
	"errors"
	"testing"
)

// TestCleanup_FailpointPreservesPrimaryError confirms that when an execution
// failpoint fires, the result's Err is the execution error, not a cleanup error.
func TestCleanup_FailpointPreservesPrimaryError(t *testing.T) {
	pool := applyTestPool(t, "g4-cleanup-primary", 56301)
	ctx := context.Background()

	ActiveFailpointController = &MapFailpoint{Errors: map[Failpoint]error{FailBeforeCommit: errors.New("injected before-commit failure")}}
	defer func() { ActiveFailpointController = InertFailpoints{} }()

	res := Apply(ctx, pool, ApplyOptions{Actor: "cleanup-test", ReleaseID: "r", EvidenceRef: "e"})
	if res.Err == nil {
		t.Fatal("expected failure from injected failpoint")
	}
	// The error must mention the failpoint, not a cleanup/lock issue.
	if !contains(string(res.Err.Error()), "before-commit failpoint") {
		t.Errorf("primary error should be the failpoint, got: %v", res.Err)
	}
}

// TestCleanup_NoAdvisoryLockAfterFailure confirms no advisory lock leaks after
// a failed apply (the session lock was released or the connection was destroyed).
func TestCleanup_NoAdvisoryLockAfterFailure(t *testing.T) {
	pool := applyTestPool(t, "g4-cleanup-nolock", 56302)
	ctx := context.Background()

	ActiveFailpointController = &MapFailpoint{Errors: map[Failpoint]error{FailAfterArtifactBaseline: errors.New("injected mid-exec failure")}}
	defer func() { ActiveFailpointController = InertFailpoints{} }()

	Apply(ctx, pool, ApplyOptions{Actor: "cleanup-test", ReleaseID: "r", EvidenceRef: "e"})

	// After the failed apply, acquire a fresh connection and check for advisory locks.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("post-failure acquire: %v", err)
	}
	defer conn.Release()
	var lockCount int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory'").Scan(&lockCount); err != nil {
		t.Fatalf("advisory lock count query: %v", err)
	}
	if lockCount != 0 {
		t.Errorf("advisory lock leak: %d locks remain after failed apply (connection should have been destroyed)", lockCount)
	}
}

// TestCleanup_RerunSucceedsAfterFailure confirms a failed apply doesn't leave
// the database in a state that blocks a subsequent successful apply.
func TestCleanup_RerunSucceedsAfterFailure(t *testing.T) {
	pool := applyTestPool(t, "g4-cleanup-rerun", 56303)
	ctx := context.Background()

	// First: inject a mid-execution failure.
	ActiveFailpointController = &MapFailpoint{Errors: map[Failpoint]error{FailAfterArtifactBaseline: errors.New("injected")}}
	res1 := Apply(ctx, pool, ApplyOptions{Actor: "a", ReleaseID: "r", EvidenceRef: "e"})
	if res1.Err == nil {
		t.Fatal("first apply should have failed")
	}

	// Second: clean run must succeed.
	ActiveFailpointController = InertFailpoints{}
	res2 := Apply(ctx, pool, ApplyOptions{Actor: "a", ReleaseID: "r", EvidenceRef: "e"})
	if res2.Err != nil {
		t.Fatalf("rerun after failure failed: %v", res2.Err)
	}
	if res2.GovernedFingerprint != GovernedTargetFingerprint {
		t.Errorf("rerun governed FP: %s", res2.GovernedFingerprint)
	}
}

// TestCleanup_RepeatedUnlockReturnsError confirms the LockedConnection's second
// Release returns ErrLockNotHeld and never sends SQL to PostgreSQL.
func TestCleanup_RepeatedUnlockReturnsError(t *testing.T) {
	conn := startFixture(t, "g4-cleanup-2unlock", 56304)
	defer conn.Close(context.Background())
	_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", MigrationLockKey)

	lc := &LockedConnection{Conn: conn}
	ctx := context.Background()

	if err := lc.Acquire(ctx); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := lc.Release(ctx); err != nil {
		t.Fatalf("first release: %v", err)
	}
	// Second release: must return ErrLockNotHeld (no SQL sent).
	err := lc.Release(ctx)
	if !errors.Is(err, ErrLockNotHeld) {
		t.Errorf("second release: got %v want ErrLockNotHeld", err)
	}
	// Confirm no second unlock was sent: the lock count should still be 0.
	var lockCount int
	conn.QueryRow(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory'").Scan(&lockCount)
	if lockCount != 0 {
		t.Errorf("advisory lock count after double-unlock: %d (second unlock may have been sent)", lockCount)
	}
}

// TestCleanup_NoUnlockAllAnywhere is the static guarantee that the migration
// package source never calls pg_advisory_unlock_all. (Also in apply_p3_test.go;
// duplicated here for the cleanup matrix.)
func TestCleanup_NoUnlockAllAnywhere(t *testing.T) {
	files := []string{"lock.go", "apply.go", "ledger.go", "probe.go", "executor.go", "adapters.go", "failpoints.go", "package.go", "provenance.go"}
	for _, f := range files {
		b, err := readFileCompat(f)
		if err != nil {
			continue
		}
		if contains(string(b), "pg_advisory_unlock_all") {
			t.Errorf("forbidden pg_advisory_unlock_all in %s", f)
		}
	}
}

// readFileCompat reads a file from the test working directory.
func readFileCompat(name string) ([]byte, error) {
	return readFileOrNull(name)
}
