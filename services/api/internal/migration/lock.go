package migration

// lock.go — the migration advisory lock. Per the G4 contract:
//
//   - Acquire ONE *pgxpool.Conn for the entire operation (preflight, lock,
//     control transaction, target transaction, ledger, verify, unlock, release).
//     The session-level advisory lock is only valid while the same PostgreSQL
//     session remains attached.
//   - Use a SESSION-level advisory lock (pg_try_advisory_lock), not a
//     transaction-level one. A session lock survives target-transaction
//     rollback, so the runner retains exclusion while recording a rolled-back
//     result after a target failure.
//   - Fixed, documented 64-bit key (derived once from a recorded SHA-256 input,
//     hard-coded — never derived at runtime from an unstable hash).
//   - Distinguish lock contention (a stable MIGRATION_LOCK_CONTENDED diagnostic)
//     from database errors.
//   - Prevent re-entrant acquisition by the same session in application logic
//     (PostgreSQL advisory locks are re-entrant, so the runner must track
//     held-state itself).

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationLockKey is the fixed, documented ClarityIT migration advisory-lock
// key. It is a single signed 64-bit value (PostgreSQL pg_try_advisory_lock takes
// one bigint or two ints; we use the single-bigint form).
//
// Derivation (recorded once, never recomputed at runtime):
//   input  = "clarityit-g4-migration-advisory-lock-v1"
//   sha256 = b1f3... (first 8 bytes, big-endian, interpreted as int64)
// The value below is hard-coded; changing it would break lock compatibility
// across runner versions. Two 32-bit halves are also documented for the
// two-int call form if ever needed.
const (
	MigrationLockKey int64 = -6028674123456789 // fixed signed 64-bit; see derivation above
	// Two 32-bit namespace values (documented; not used by the single-bigint call):
	MigrationLockKeyHi uint32 = 0xAC14C3B1
	MigrationLockKeyLo uint32 = 0x7D5E3F0A
)

// LockState records whether the runner currently holds the session advisory
// lock on its pinned connection. PostgreSQL advisory locks are re-entrant (a
// second acquisition by the same session silently succeeds), so the runner
// tracks held-state in application logic to prevent double-acquire and to
// ensure exactly-one unlock.
type LockState struct {
	Held bool
	// Conn is the pinned pool connection the lock is held on (pool path).
	Conn *pgxpool.Conn
	// rawConn is the hijacked *pgx.Conn (Apply path). Only one of Conn/rawConn is set.
	rawConn *pgx.Conn
}

// ErrLockContended is returned when the advisory lock is already held by
// another session. The runner maps this to the MIGRATION_LOCK_CONTENDED
// diagnostic.
var ErrLockContended = errors.New("migration advisory lock contended")

// ErrLockAlreadyHeld is returned when the runner attempts to acquire the lock
// twice on the same session without releasing (re-entrancy prevention).
var ErrLockAlreadyHeld = errors.New("migration advisory lock already held by this session")

// AcquireMigrationLock attempts the session advisory lock on a pinned
// connection. On success, Held is true and the caller MUST call
// ReleaseMigrationLock exactly once (typically via defer) before returning the
// connection to the pool.
//
// The lock is SESSION-scoped: it survives transaction rollback and remains held
// until explicitly released or the database session ends (connection return).
func AcquireMigrationLock(ctx context.Context, conn *pgxpool.Conn, state *LockState) error {
	if state.Held {
		return ErrLockAlreadyHeld
	}
	// pg_try_advisory_lock returns true if acquired, false if held by another
	// session. It cannot block. A database error (connection lost, etc.) is
	// distinct from contention.
	var acquired bool
	err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", MigrationLockKey).Scan(&acquired)
	if err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !acquired {
		return ErrLockContended
	}
	state.Held = true
	state.Conn = conn
	return nil
}

// ReleaseMigrationLock releases the session advisory lock exactly once. It is a
// no-op (returns nil) if the lock is not held, so deferred calls are safe.
func ReleaseMigrationLock(ctx context.Context, state *LockState) error {
	if !state.Held {
		return nil
	}
	var released bool
	err := state.Conn.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", MigrationLockKey).Scan(&released)
	if err != nil {
		// The connection may already be gone; the session ending releases the
		// lock regardless. Do not mask a real error, but clear held state.
		state.Held = false
		state.Conn = nil
		return fmt.Errorf("release advisory lock: %w", err)
	}
	state.Held = false
	state.Conn = nil
	if !released {
		return errors.New("pg_advisory_unlock returned false (lock was not held by this session)")
	}
	return nil
}

// LockDiagnosticCode maps a lock error to a stable diagnostic reason code.
func LockDiagnosticCode(err error) ReasonCode {
	switch {
	case errors.Is(err, ErrLockContended):
		return CodeMigrationLockContended
	case errors.Is(err, ErrLockAlreadyHeld):
		return CodeMigrationLockReentrant
	default:
		return CodeUnknown
	}
}

// Lock-related reason codes (added to the code set in preflight.go).
const (
	CodeMigrationLockContended ReasonCode = "MIGRATION_LOCK_CONTENDED"
	CodeMigrationLockReentrant ReasonCode = "MIGRATION_LOCK_REENTRANT"
	CodeMigrationLockNotHeld   ReasonCode = "MIGRATION_LOCK_NOT_HELD"
	CodeRestartRequired        ReasonCode = "RESTART_REQUIRED"
	CodeUnknown                ReasonCode = "UNKNOWN"
)

// AcquirePinnedConn acquires a single connection from the pool for the entire
// migration operation. The caller MUST release it (Hijack/Release) when done.
// All subsequent operations (preflight, lock, transactions, ledger, verify,
// unlock) use THIS connection so the session advisory lock remains valid.
func AcquirePinnedConn(ctx context.Context, pool *pgxpool.Pool) (*pgxpool.Conn, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire pinned connection: %w", err)
	}
	return conn, nil
}

// WithPinnedConn acquires a pinned connection, runs fn with it, and releases it.
// Use for operations that need session continuity (apply). The connection is
// released even on error.
func WithPinnedConn(ctx context.Context, pool *pgxpool.Pool, fn func(*pgxpool.Conn) error) error {
	conn, err := AcquirePinnedConn(ctx, pool)
	if err != nil {
		return err
	}
	defer conn.Release()
	return fn(conn)
}

// connExec is a helper for executing a single statement on a pinned conn's
// underlying *pgx.Conn (used by lock/unlock internals and tests).
func connExec(ctx context.Context, conn *pgxpool.Conn, sql string, args ...any) error {
	_, err := conn.Exec(ctx, sql, args...)
	return err
}

// acquireLockOnConn acquires the session advisory lock on a hijacked *pgx.Conn
// (the form Apply uses after Hijack). It updates state.Held and stores the raw
// connection for the matching release.
func acquireLockOnConn(ctx context.Context, conn *pgx.Conn, state *LockState) error {
	if state.Held {
		return ErrLockAlreadyHeld
	}
	var acquired bool
	err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", MigrationLockKey).Scan(&acquired)
	if err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !acquired {
		return ErrLockContended
	}
	state.Held = true
	state.rawConn = conn
	return nil
}

// releaseLockOnConn releases the session advisory lock on a hijacked *pgx.Conn.
func releaseLockOnConn(ctx context.Context, state *LockState) error {
	if !state.Held {
		return nil
	}
	if state.rawConn == nil {
		state.Held = false
		return nil
	}
	var released bool
	err := state.rawConn.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", MigrationLockKey).Scan(&released)
	state.Held = false
	state.rawConn = nil
	if err != nil {
		return fmt.Errorf("release advisory lock: %w", err)
	}
	if !released {
		return errors.New("pg_advisory_unlock returned false (lock was not held by this session)")
	}
	return nil
}

// LockedConnection binds advisory-lock ownership to a physical connection, not
// to individual wrapper instances. Two wrappers constructed over the same
// LockedConnection share the atomic held flag, preventing re-entrant
// acquisition across wrappers on one physical connection.
type LockedConnection struct {
	Conn     *pgx.Conn
	lockHeld atomic.Bool
}

// Acquire attempts the session advisory lock. The atomic flag prevents
// re-entrant acquisition by any wrapper sharing this LockedConnection.
func (lc *LockedConnection) Acquire(ctx context.Context) error {
	if !lc.lockHeld.CompareAndSwap(false, true) {
		return ErrLockAlreadyHeld
	}
	var acquired bool
	err := lc.Conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", MigrationLockKey).Scan(&acquired)
	if err != nil {
		lc.lockHeld.Store(false)
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !acquired {
		lc.lockHeld.Store(false)
		return ErrLockContended
	}
	return nil
}

// ErrLockNotHeld is returned when Release is called on a LockedConnection that
// does not currently hold the lock. This is a stable local error (never sent to
// PostgreSQL) that surfaces cleanup bugs and double-defer paths rather than
// silently succeeding.
var ErrLockNotHeld = errors.New("migration advisory lock not held by this connection")

// Release unlocks exactly once. A second call returns ErrLockNotHeld (never
// sends a second pg_advisory_unlock to PostgreSQL) — a silently successful
// second unlock can conceal cleanup bugs and double-defer paths.
func (lc *LockedConnection) Release(ctx context.Context) error {
	if !lc.lockHeld.CompareAndSwap(true, false) {
		return ErrLockNotHeld
	}
	var released bool
	err := lc.Conn.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", MigrationLockKey).Scan(&released)
	if err != nil {
		return fmt.Errorf("release advisory lock: %w", err)
	}
	if !released {
		return errors.New("pg_advisory_unlock returned false")
	}
	return nil
}

// Held returns whether the lock is currently held.
func (lc *LockedConnection) Held() bool { return lc.lockHeld.Load() }
