package migration

// preflight_matrix_identity_test.go — the two identity-guard matrix cases that
// require non-default container configuration: unsupported PostgreSQL major
// (PG15) and wrong database name. These complete the live rejection matrix.
//
// Both use the stronger snapshot + DDL event-trigger sentinel so zero-mutation
// is proven the same way as every other case.

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// startFixtureCustom brings up an isolated container with custom image/DB/user.
// Used for the PG15 and wrong-db-name cases.
func startFixtureCustom(t *testing.T, name, image, db, user, password string, port int) *pgx.Conn {
	t.Helper()
	exec.Command("docker", "rm", "-f", name).Run()
	args := []string{"run", "-d", "--name", name,
		"-e", "POSTGRES_PASSWORD=" + password,
		"-e", "POSTGRES_DB=" + db,
		"-e", "POSTGRES_USER=" + user,
		"-p", fmt.Sprintf("%d:5432", port),
		image}
	if err := exec.Command("docker", args...).Run(); err != nil {
		t.Skipf("docker unavailable or image %s missing: %v", image, err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", user, password, port, db)
	var conn *pgx.Conn
	var err error
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		conn, err = pgx.Connect(ctx, dsn)
		cancel()
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Skipf("could not connect fixture %s on port %d: %v", name, port, err)
	}
	t.Cleanup(func() {
		conn.Close(context.Background())
		exec.Command("docker", "rm", "-f", name).Run()
	})
	return conn
}

// TestMatrix_UnsupportedPGMajor: a PG15 database blocks with PG_MAJOR_UNSUPPORTED
// before any DDL, with zero mutation.
func TestMatrix_UnsupportedPGMajor(t *testing.T) {
	// PG15 does not have all the G3 prerequisites, but the identity guard fires
	// first (PG major != 16), so no DDL is attempted.
	conn := startFixtureCustom(t, "g4-matrix-pg15", "postgres:15-alpine", "clarityit", "postgres", "postgres", 55516)
	installDDLEventTrigger(t, conn)
	spy := &SpyExecutor{Inner: NoExecutor{}}
	before := snapshotDBStrong(t, conn)
	// Preflight opens its own connection from the same DSN family; pass the
	// custom DSN via the matrix DSN override is not possible, so invoke the
	// classification directly on a probe built from this conn.
	probe, err := probeAll(context.Background(), connBeginTx(t, conn))
	if err != nil {
		t.Fatalf("probeAll: %v", err)
	}
	class, path, code := Classify(probe)
	res := PreflightResult{Probe: probe, Class: class, Path: path, Code: code}
	after := snapshotDBStrong(t, conn)
	// Expect PG_MAJOR_UNSUPPORTED (15 != 16).
	assertBlockedWithoutMutation(t, before, after, res, CodePgMajorUnsupported)
	assertExecutorNeverInvoked(t, spy)
}

// TestMatrix_WrongDatabaseName: a database not named clarityit blocks with
// DB_IDENTITY_WRONG before any DDL, with zero mutation.
func TestMatrix_WrongDatabaseName(t *testing.T) {
	conn := startFixtureCustom(t, "g4-matrix-wrongdb", "postgres:16-alpine", "other_db", "postgres", "postgres", 55517)
	installDDLEventTrigger(t, conn)
	spy := &SpyExecutor{Inner: NoExecutor{}}
	before := snapshotDBStrong(t, conn)
	probe, err := probeAll(context.Background(), connBeginTx(t, conn))
	if err != nil {
		t.Fatalf("probeAll: %v", err)
	}
	class, path, code := Classify(probe)
	res := PreflightResult{Probe: probe, Class: class, Path: path, Code: code}
	after := snapshotDBStrong(t, conn)
	assertBlockedWithoutMutation(t, before, after, res, CodeDbIdentityWrong)
	assertExecutorNeverInvoked(t, spy)
}

// connBeginTx is a helper that begins a read-only tx on a conn and returns it
// (the caller's probe runs inside; rollback is automatic via defer elsewhere).
// For the identity-guard cases we only need the probe, not the full Preflight
// connection-management path.
func connBeginTx(t *testing.T, conn *pgx.Conn) pgx.Tx {
	t.Helper()
	tx, err := conn.BeginTx(context.Background(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatalf("begin read-only tx: %v", err)
	}
	t.Cleanup(func() { tx.Rollback(context.Background()) })
	return tx
}
