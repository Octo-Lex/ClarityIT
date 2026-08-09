package migration

// lock_twowrapper_test.go — the connection-level lock invariant. Two wrappers
// over the same physical connection share the LockedConnection's atomic held
// flag, so the second acquisition is rejected in application logic before
// reaching PostgreSQL's re-entrant pg_try_advisory_lock.

import (
	"context"
	"errors"
	"testing"
)

// TestLock_TwoWrappers_PreventReentrantAcquisition proves two wrappers sharing
// one LockedConnection cannot both acquire. The second must receive
// MIGRATION_LOCK_REENTRANT without calling pg_try_advisory_lock.
func TestLock_TwoWrappers_PreventReentrantAcquisition(t *testing.T) {
	conn := startFixture(t, "g4-lock-2wrap", 55608)
	defer conn.Close(context.Background())
	// Belt-and-suspenders: force-release any leftover lock from prior runs.
	_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", MigrationLockKey)

	lc := &LockedConnection{Conn: conn}
	ctx := context.Background()

	// Wrapper 1 acquires through the shared LockedConnection.
	if err := lc.Acquire(ctx); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if !lc.Held() {
		t.Fatal("Held()=false after successful acquire")
	}

	// Wrapper 2 (same LockedConnection) tries to acquire — must be rejected by
	// the atomic flag WITHOUT calling pg_try_advisory_lock.
	err := lc.Acquire(ctx)
	if !errors.Is(err, ErrLockAlreadyHeld) {
		t.Fatalf("second acquire: got %v want ErrLockAlreadyHeld", err)
	}
	if code := LockDiagnosticCode(err); code != CodeMigrationLockReentrant {
		t.Errorf("diagnostic code: got %q want %q", code, CodeMigrationLockReentrant)
	}

	// PostgreSQL should report exactly ONE held advisory lock for this session.
	// (pg_try_advisory_lock with a single bigint key; query all advisory locks
	// held by this backend since the key encoding into classid/objid is
	// implementation-specific.)
	var lockCount int
	if err := conn.QueryRow(ctx,
		`SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND pid = pg_backend_pid()`).Scan(&lockCount); err != nil {
		t.Fatalf("count advisory locks: %v", err)
	}
	if lockCount != 1 {
		t.Errorf("expected exactly 1 held advisory lock, got %d (re-entrant acquisition may have reached PG)", lockCount)
	}

	// First Release: exactly one unlock sent to PG.
	if err := lc.Release(ctx); err != nil {
		t.Fatalf("first release: %v", err)
	}
	if lc.Held() {
		t.Error("Held()=true after release")
	}

	// Second Release: returns ErrLockNotHeld, never sent to PG (atomic CAS
	// fails). A silently-successful second unlock can conceal cleanup bugs.
	if err := lc.Release(ctx); !errors.Is(err, ErrLockNotHeld) {
		t.Errorf("second release: got %v want ErrLockNotHeld", err)
	}

	// No path uses pg_advisory_unlock_all (verified by TestApply_NoUnlockAllInSources).
}
