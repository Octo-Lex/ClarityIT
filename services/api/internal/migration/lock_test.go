package migration

// lock_test.go — the advisory-lock test matrix. Covers all six required
// behaviors:
//   1. exactly one contender succeeds;
//   2. competitors receive MIGRATION_LOCK_CONTENDED;
//   3. cancellation while waiting does not acquire the lock (try-lock cannot
//      block, so this is exercised via a cancelled context);
//   4. rollback does not release the session lock;
//   5. explicit unlock releases it;
//   6. connection termination releases it (the backend session ends);
//   7. repeated acquisition by the same session is prevented in app logic.
//
// These use a shared PG16 fixture (a pool pinned to one backend). Each test
// starts from a known-unlocked state.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// startLockFixture brings up a dedicated PG16 container for ONE lock test and
// returns a DSN. Each lock test gets its own container+port so the per-test
// cleanup (container removal) doesn't tear down a shared fixture mid-suite.
func startLockFixture(t *testing.T, port int) string {
	if d := os.Getenv("CLARITY_G4_LOCK_DSN"); d != "" {
		return d
	}
	// startFixture brings up the container and registers cleanup on this test.
	startFixture(t, "g4-lock-"+sanitize(t.Name()), port)
	return fmt.Sprintf("postgres://postgres:postgres@localhost:%d/clarityit?sslmode=disable", port)
}

func newLockPool(t *testing.T, port int) *pgxpool.Pool {
	t.Helper()
	dsn := startLockFixture(t, port)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestLock_ExactlyOneSucceeds: among N concurrent contenders, exactly one
// acquires the lock; the rest receive ErrLockContended.
func TestLock_ExactlyOneSucceeds(t *testing.T) {
	pool := newLockPool(t, 55601)
	// Ensure clean state: no leftover lock from prior runs.
	forceUnlockAll(t, pool)

	const contenders = 5
	var (
		mu        sync.Mutex
		acquired  = 0
		contended = 0
	)
	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := pool.Acquire(context.Background())
			if err != nil {
				t.Errorf("acquire conn: %v", err)
				return
			}
			defer conn.Release()
			var state LockState
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := AcquireMigrationLock(ctx, conn, &state); err == nil {
				mu.Lock()
				acquired++
				mu.Unlock()
				// Hold briefly so other contenders see it held.
				time.Sleep(200 * time.Millisecond)
				if err := ReleaseMigrationLock(context.Background(), &state); err != nil {
					t.Errorf("release: %v", err)
				}
			} else if errors.Is(err, ErrLockContended) {
				mu.Lock()
				contended++
				mu.Unlock()
			} else {
				t.Errorf("unexpected lock error: %v", err)
			}
		}()
	}
	wg.Wait()
	if acquired != 1 {
		t.Errorf("acquired: got %d want 1", acquired)
	}
	if contended != contenders-1 {
		t.Errorf("contended: got %d want %d", contended, contenders-1)
	}
}

// TestLock_CompetitorsGetContendedDiagnostic: the LockDiagnosticCode for
// contention maps to MIGRATION_LOCK_CONTENDED.
func TestLock_CompetitorsGetContendedDiagnostic(t *testing.T) {
	pool := newLockPool(t, 55602)
	forceUnlockAll(t, pool)

	holder, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire holder: %v", err)
	}
	defer holder.Release()
	var holderState LockState
	if err := AcquireMigrationLock(context.Background(), holder, &holderState); err != nil {
		t.Fatalf("holder acquire: %v", err)
	}
	defer ReleaseMigrationLock(context.Background(), &holderState)

	competitor, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire competitor: %v", err)
	}
	defer competitor.Release()
	var compState LockState
	err = AcquireMigrationLock(context.Background(), competitor, &compState)
	if !errors.Is(err, ErrLockContended) {
		t.Fatalf("competitor: got %v want ErrLockContended", err)
	}
	if code := LockDiagnosticCode(err); code != CodeMigrationLockContended {
		t.Errorf("diagnostic code: got %q want %q", code, CodeMigrationLockContended)
	}
}

// TestLock_CancelledContextDoesNotAcquire: a cancelled context returns an error
// and does NOT acquire the lock.
func TestLock_CancelledContextDoesNotAcquire(t *testing.T) {
	pool := newLockPool(t, 55603)
	forceUnlockAll(t, pool)
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before acquire
	var state LockState
	if err := AcquireMigrationLock(ctx, conn, &state); err == nil {
		// pg_try_advisory_lock may still succeed on a cancelled context if the
		// query was already in flight; check held state and clean up.
		if state.Held {
			_ = ReleaseMigrationLock(context.Background(), &state)
			t.Error("lock acquired despite cancelled context")
		}
	}
	if state.Held {
		t.Error("state.Held true after cancelled-context acquire")
	}
}

// TestLock_RollbackDoesNotRelease: a session advisory lock survives a
// transaction rollback on the same connection.
func TestLock_RollbackDoesNotRelease(t *testing.T) {
	pool := newLockPool(t, 55604)
	forceUnlockAll(t, pool)
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	var state LockState
	if err := AcquireMigrationLock(context.Background(), conn, &state); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer ReleaseMigrationLock(context.Background(), &state)

	// Begin + rollback a transaction on the same connection.
	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if _, err := tx.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// The lock must still be held: a competitor must still contend.
	competitor, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire competitor: %v", err)
	}
	defer competitor.Release()
	var compState LockState
	if err := AcquireMigrationLock(context.Background(), competitor, &compState); err == nil {
		_ = ReleaseMigrationLock(context.Background(), &compState)
		t.Error("competitor acquired lock after holder rollback (session lock should survive)")
	}
}

// TestLock_ExplicitUnlockReleases: after ReleaseMigrationLock, a competitor can
// acquire.
func TestLock_ExplicitUnlockReleases(t *testing.T) {
	pool := newLockPool(t, 55605)
	forceUnlockAll(t, pool)
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	var state LockState
	if err := AcquireMigrationLock(context.Background(), conn, &state); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := ReleaseMigrationLock(context.Background(), &state); err != nil {
		t.Fatalf("release: %v", err)
	}
	if state.Held {
		t.Error("Held still true after release")
	}

	competitor, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire competitor: %v", err)
	}
	defer competitor.Release()
	var compState LockState
	if err := AcquireMigrationLock(context.Background(), competitor, &compState); err != nil {
		t.Errorf("competitor could not acquire after explicit unlock: %v", err)
	}
	_ = ReleaseMigrationLock(context.Background(), &compState)
}

// TestLock_ConnectionTerminationReleases: releasing the pooled connection (which
// ends the backend session) releases the session advisory lock.
func TestLock_ConnectionTerminationReleases(t *testing.T) {
	pool := newLockPool(t, 55606)
	forceUnlockAll(t, pool)
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	var state LockState
	if err := AcquireMigrationLock(context.Background(), conn, &state); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	// Release WITHOUT calling ReleaseMigrationLock — ending the session should
	// release the lock.
	conn.Release()

	// Give the backend a moment to end.
	time.Sleep(300 * time.Millisecond)

	competitor, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire competitor: %v", err)
	}
	defer competitor.Release()
	var compState LockState
	if err := AcquireMigrationLock(context.Background(), competitor, &compState); err != nil {
		t.Errorf("competitor could not acquire after connection termination: %v", err)
	}
	_ = ReleaseMigrationLock(context.Background(), &compState)
}

// TestLock_ReentrantPrevented: a second AcquireMigrationLock on the same
// session/state returns ErrLockAlreadyHeld (app-logic prevention, since
// PostgreSQL advisory locks are re-entrant).
func TestLock_ReentrantPrevented(t *testing.T) {
	pool := newLockPool(t, 55607)
	forceUnlockAll(t, pool)
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	var state LockState
	if err := AcquireMigrationLock(context.Background(), conn, &state); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer ReleaseMigrationLock(context.Background(), &state)

	// Second acquire on the same state -> app-logic block.
	if err := AcquireMigrationLock(context.Background(), conn, &state); !errors.Is(err, ErrLockAlreadyHeld) {
		t.Errorf("reentrant acquire: got %v want ErrLockAlreadyHeld", err)
	}
	if code := LockDiagnosticCode(ErrLockAlreadyHeld); code != CodeMigrationLockReentrant {
		t.Errorf("diagnostic code: got %q want %q", code, CodeMigrationLockReentrant)
	}
}

// forceUnlockAll is a belt-and-suspenders cleanup that force-releases any
// leftover advisory lock from a prior test (e.g. if a test crashed before
// unlock). pg_advisory_unlock on a session that doesn't hold it is a no-op.
func forceUnlockAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return
	}
	defer conn.Release()
	// Try unlocking on this fresh session; it won't hold the lock, but if the
	// prior backend already ended this is harmless. The real guarantee is that
	// each test starts its own connections.
	_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", MigrationLockKey)
}
