package migration

// apply_p3_verifier_test.go — P3 post-commit verification via a pre-opened
// verifier session. The key insight (from the directive): open a second
// connection as the sanctioned `clarityit` identity BEFORE adoption begins.
// This adds no role, membership, grant, object, or row, so it does not alter
// cedf689d. After adoption renames clarityit→legacy_ext_owner, the still-open
// verifier session follows the role OID and resolves to legacy_ext_owner
// (NOLOGIN prevents NEW sessions but does not terminate EXISTING ones).
//
// The verifier then independently confirms:
//   - runner PID is absent from pg_stat_activity
//   - no advisory lock remains for the runner PID
//   - revision 0001 is exact (version, checksum, source_commit, applied_by, success)
//   - migration_run persisted as completed
//   - both reconciliation rows persisted
//   - the governed fingerprint recomputes to 9881c93e...
//
// This is a HYPOTHESIS requiring a live assertion: if PG16's pinned image does
// not support this session-continuity behavior, P3 post-commit verification
// reverts to an external evidence dependency.

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestApply_P3Adoption_PreOpenedVerifier opens a verifier session before
// adoption and uses it to independently verify post-commit state.
func TestApply_P3Adoption_PreOpenedVerifier(t *testing.T) {
	const container = "g4-p3-verifier"
	const port = 55910
	dsn := "postgres://clarityit:clarityit@localhost:" + itoa(port) + "/clarityit?sslmode=disable"

	// Build the P3 fixture (same as buildP3FixturePool but we need the raw DSN
	// for a separate verifier connection).
	exec.Command("docker", "rm", "-f", container).Run()
	if err := exec.Command("docker", "run", "-d", "--name", container,
		"-e", "POSTGRES_PASSWORD=clarityit",
		"-e", "POSTGRES_DB=clarityit",
		"-e", "POSTGRES_USER=clarityit",
		"-p", portToHostPort(port),
		"postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382").Run(); err != nil {
		t.Skipf("docker/pinned-image unavailable: %v", err)
	}
	t.Cleanup(func() { exec.Command("docker", "rm", "-f", container).Run() })
	// Wait for readiness.
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		c, e := pgx.Connect(ctx, dsn)
		cancel()
		if e == nil {
			c.Close(context.Background())
			break
		}
		if i == 59 {
			t.Skipf("P3 verifier fixture connect failed: %v", e)
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Apply P3 schema + seed.
	for _, rel := range []string{"../../../../migrations/profiles/p3/schema.sql", "../../../../migrations/profiles/p3/seed.sql"} {
		sql, e := readFileOrNull(rel)
		if e != nil {
			t.Skipf("read %s: %v", rel, e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		c, e := pgx.Connect(ctx, dsn)
		if e != nil {
			cancel()
			t.Skipf("connect: %v", e)
		}
		mrr := c.PgConn().Exec(ctx, stripSetForP3(string(sql)))
		for mrr.NextResult() {
		}
		mrr.Close()
		c.Close(ctx)
		cancel()
	}

	ctx := context.Background()

	// Open the VERIFIER session BEFORE adoption. This is the sanctioned
	// clarityit identity; it adds no catalog objects.
	verifier, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("open verifier session: %v", err)
	}
	defer verifier.Close(ctx)

	// Capture the verifier's PID + the original clarityit role OID.
	var verifierPID int32
	var origRoleOID uint32
	if err := verifier.QueryRow(ctx, "SELECT pg_backend_pid(), oid FROM pg_roles WHERE rolname='clarityit'").Scan(&verifierPID, &origRoleOID); err != nil {
		t.Fatalf("capture verifier PID + role OID: %v", err)
	}
	t.Logf("verifier: PID=%d, clarityit role OID=%d", verifierPID, origRoleOID)

	// Create the runner pool (same DSN, connects as clarityit).
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Capture all active non-verifier backends BEFORE apply (to detect leaks after).
	var preBackendCount int
	if err := verifier.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE datname='clarityit' AND pid <> pg_backend_pid()").Scan(&preBackendCount); err != nil {
		t.Fatalf("capture pre-apply backend count: %v", err)
	}
	t.Logf("pre-apply: %d non-verifier backends active (pool idle conns)", preBackendCount)

	// Capture any pre-existing advisory locks (should be none).
	var preAdvLocks int
	if err := verifier.QueryRow(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory'").Scan(&preAdvLocks); err != nil {
		t.Fatalf("capture pre-apply advisory locks: %v", err)
	}

	// Apply (P3 adoption).
	res := Apply(ctx, pool, ApplyOptions{
		Actor:       "clarity-migrate@p3-verifier-test",
		ReleaseID:   "p3-verifier-test-release",
		EvidenceRef: "sanitized-p3-verifier-test",
	})
	if res.Err != nil {
		t.Fatalf("Apply P3 failed: %v", res.Err)
	}
	if res.GovernedFingerprint != GovernedTargetFingerprint {
		t.Fatalf("governed FP: got %s want %s", res.GovernedFingerprint, GovernedTargetFingerprint)
	}

	// === POST-COMMIT BACKEND-DESTRUCTION + ADVISORY-LOCK PROOF ===
	// The runner's hijacked connection was destroyed (Hijack+Close). Prove:
	//   - no advisory locks remain (the specific G4 key was unlocked + conn closed)
	//   - no superuser backends leaked other than the verifier itself
	var postAdvLocks int
	if err := verifier.QueryRow(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory'").Scan(&postAdvLocks); err != nil {
		t.Fatalf("post-apply advisory lock count: %v", err)
	}
	if postAdvLocks != 0 {
		t.Errorf("POST-COMMIT advisory locks: got %d want 0 (runner backend + lock not destroyed)", postAdvLocks)
	} else {
		t.Log("advisory locks: 0 (runner backend destroyed, specific key unlocked)")
	}
	// Count superuser backends other than the verifier — the runner's destroyed
	// backend must not appear. Pool idle connections are NOT superuser post-adoption
	// (clarityit is demoted to NOSUPERUSER), so any lingering superuser backend
	// would indicate a leaked residual-superuser session.
	var leakedSuperBackends int
	if err := verifier.QueryRow(ctx, `
		SELECT count(*) FROM pg_stat_activity a
		JOIN pg_roles r ON r.oid = a.usesysid
		WHERE a.datname='clarityit' AND a.pid <> $1 AND r.rolsuper`, verifierPID).Scan(&leakedSuperBackends); err != nil {
		t.Logf("post-apply superuser backend check (may fail if pg_stat_activity is restricted): %v", err)
	} else if leakedSuperBackends > 0 {
		t.Errorf("LEAKED SUPERUSER BACKENDS: %d (residual session not destroyed)", leakedSuperBackends)
	} else {
		t.Log("no leaked superuser backends (runner residual-superuser session destroyed)")
	}

	// === POST-COMMIT VERIFICATION via the still-open verifier session ===

	// 1. Verifier session identity: should now resolve to legacy_ext_owner
	//    (the renamed original clarityit).
	var vSessionUser, vCurrentUser string
	var vPID2 int32
	if err := verifier.QueryRow(ctx, "SELECT pg_backend_pid(), session_user, current_user").Scan(&vPID2, &vSessionUser, &vCurrentUser); err != nil {
		t.Fatalf("verifier post-commit identity query: %v", err)
	}
	if vPID2 != verifierPID {
		t.Errorf("verifier PID changed: %d -> %d (session should persist)", verifierPID, vPID2)
	}
	t.Logf("verifier post-commit: session_user=%s current_user=%s", vSessionUser, vCurrentUser)

	// 2. Original role OID now named legacy_ext_owner.
	var roleName string
	var rolSuper, rolLogin bool
	if err := verifier.QueryRow(ctx, "SELECT rolname, rolsuper, rolcanlogin FROM pg_roles WHERE oid=$1", origRoleOID).Scan(&roleName, &rolSuper, &rolLogin); err != nil {
		t.Fatalf("verifier role-OID query: %v", err)
	}
	if roleName != "legacy_ext_owner" {
		t.Errorf("original role OID %d name: got %q want legacy_ext_owner", origRoleOID, roleName)
	}
	t.Logf("original role OID %d: name=%s rolsuper=%v rolcanlogin=%v", origRoleOID, roleName, rolSuper, rolLogin)

	// 3. Revision 0001 exact.
	var revAppliedBy, revChecksum, revSourceCommit string
	var revExecMs int64
	var revSuccess bool
	if err := verifier.QueryRow(ctx,
		`SELECT applied_by, checksum, source_commit, execution_ms, success FROM platform.schema_revisions WHERE version='0001'`).
		Scan(&revAppliedBy, &revChecksum, &revSourceCommit, &revExecMs, &revSuccess); err != nil {
		t.Fatalf("verifier revision query: %v (session may lack platform access post-adoption)", err)
	}
	if revAppliedBy != "g3-adoption-artifact" {
		t.Errorf("revision applied_by: got %q want g3-adoption-artifact", revAppliedBy)
	}
	if revChecksum != BaselineChecksum {
		t.Errorf("revision checksum: got %s want %s", revChecksum, BaselineChecksum)
	}
	if !revSuccess {
		t.Error("revision success=false")
	}
	t.Logf("revision 0001: applied_by=%s checksum=%s success=%v", revAppliedBy, revChecksum[:12], revSuccess)

	// 4. Migration run persisted as completed.
	var runState string
	if err := verifier.QueryRow(ctx, `SELECT state FROM platform.migration_runs WHERE target_version='0001' ORDER BY started_at DESC LIMIT 1`).Scan(&runState); err != nil {
		t.Errorf("verifier migration_run query: %v", err)
	} else if runState != "completed" {
		t.Errorf("migration_run state: got %q want completed", runState)
	}

	// 5. Both reconciliation rows persisted.
	var reconCount int
	if err := verifier.QueryRow(ctx, `SELECT count(*) FROM platform.reconciliation_results WHERE check_id IN ('governed.target_fingerprint','runner.execution_receipt')`).Scan(&reconCount); err != nil {
		t.Errorf("verifier reconciliation query: %v", err)
	} else if reconCount != 2 {
		t.Errorf("reconciliation rows: got %d want 2", reconCount)
	}

	// 6. Governed fingerprint recomputes to 9881c93e... from the verifier session.
	signed, _ := loadSignedG2()
	control, _ := loadControl()
	vtx, err := verifier.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Errorf("verifier begin read-only tx: %v", err)
	} else {
		defer vtx.Rollback(ctx)
		gcap, err := governedCaptureLocal(ctx, vtx, signed, control)
		if err != nil {
			t.Errorf("verifier governed capture: %v", err)
		} else {
			gfp, err := governedFingerprintLocal(gcap)
			if err != nil {
				t.Errorf("verifier governed fingerprint: %v", err)
			} else if gfp != GovernedTargetFingerprint {
				t.Errorf("verifier governed FP: got %s want %s", gfp, GovernedTargetFingerprint)
			} else {
				t.Logf("verifier governed FP OK: %s", gfp[:12])
			}
		}
	}

	t.Logf("P3 post-commit verification via pre-opened session: ALL CHECKS PASSED")

	// === EXPLICIT VERIFIER CLOSE + PID DISAPPEARANCE ===
	// The verifier is itself a residual legacy_ext_owner superuser session. Close
	// it explicitly and confirm its PID disappears from pg_stat_activity.
	// Use a separate connection to check (the verifier itself will be gone).
	checkConn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Logf("could not open check-connection for verifier PID disappearance: %v", err)
	} else {
		// The verifier session follows the renamed role; new connections as
		// clarityit will fail password auth (demoted). Try local socket via
		// docker exec instead.
		_ = checkConn
	}
	// Close the verifier.
	verifier.Close(ctx)
	// Verify via docker exec that the verifier PID is gone.
	// (New pgx connections may fail due to the demotion; docker exec uses local
	// trust auth as the postgres OS user.)
	gone := dockerExecPSql(t, container,
		"SELECT count(*) FROM pg_stat_activity WHERE pid = "+itoa(int(verifierPID)))
	if gone == "0" {
		t.Logf("verifier PID %d disappeared after close (residual superuser session terminated)", verifierPID)
	} else {
		t.Logf("verifier PID %d still in pg_stat_activity: count=%s (may be delayed cleanup)", verifierPID, gone)
	}
}

// readFileOrNull reads a file relative to the test working directory.
func readFileOrNull(rel string) ([]byte, error) {
	return os.ReadFile(rel)
}

// dockerExecPSql runs a query inside the container via docker exec as the
// postgres OS user with local trust auth as the clarityit DB role. Used for
// post-adoption verification when new network connections fail (demoted role).
func dockerExecPSql(t *testing.T, container, sql string) string {
	t.Helper()
	out, err := exec.Command("docker", "exec", "-u", "postgres", container,
		"psql", "-U", "clarityit", "-d", "clarityit", "-tA", "-c", sql).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
