package migration

import (
	"context"
	"strings"
	"testing"
)

// TestForwardG1RuntimeNegatives proves the two live Stage-B failure properties
// required by G1 without adding production failpoints or changing any migration
// artifact bytes:
//   1. a failure after 0002 DDL has begun rolls back the entire 0002..0005 batch
//      and all >=0002 ledger rows, then a clean rerun converges;
//   2. the inherited fixed migration advisory lock serializes Stage B before any
//      forward DDL, and release permits a clean rerun.
func TestForwardG1RuntimeNegatives(t *testing.T) {
	binaryPath, _ := buildForwardTestCLI(t)

	t.Run("atomic_rollback_and_rerun", func(t *testing.T) {
		const (
			container = "wp01-g1-forward-rollback"
			port      = 56243
		)
		ctx := context.Background()
		pool := applyTestPool(t, container, port)
		base := Apply(ctx, pool, ApplyOptions{
			Actor:       "wp01-g1-stage-a-rollback-test",
			ReleaseID:   "wp01-g1-runtime-negatives",
			EvidenceRef: "sanitized-wp01-g1-rollback",
		})
		if base.Err != nil || base.GovernedFingerprint != GovernedTargetFingerprint {
			t.Fatalf("Stage A foundation: err=%v fp=%s", base.Err, base.GovernedFingerprint)
		}
		installForwardTestCLI(t, container, binaryPath)

		// Test-only database fault injection. 0003 creates
		// kernel.case_resource_refs after the complete 0002 body has executed in
		// the same transaction. Raising at ddl_command_end therefore proves a
		// mid-batch failure rolls back both 0002 and the partial 0003 work.
		conn, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire fault-injection connection: %v", err)
		}
		_, err = conn.Exec(ctx, `
			CREATE OR REPLACE FUNCTION public.wp01_g1_fail_forward_0003()
			RETURNS event_trigger
			LANGUAGE plpgsql
			AS $function$
			DECLARE cmd record;
			BEGIN
				FOR cmd IN SELECT * FROM pg_event_trigger_ddl_commands() LOOP
					IF cmd.schema_name = 'kernel'
					   AND cmd.object_identity = 'kernel.case_resource_refs' THEN
						RAISE EXCEPTION 'wp01-g1-forward-rollback-injected';
					END IF;
				END LOOP;
			END;
			$function$;
			CREATE EVENT TRIGGER wp01_g1_fail_forward_0003
				ON ddl_command_end
				EXECUTE FUNCTION public.wp01_g1_fail_forward_0003();
		`)
		conn.Release()
		if err != nil {
			t.Fatalf("install rollback fault: %v", err)
		}

		out, err := runForwardTestCLIRaw(container, "forward",
			"-actor", "clarityit_migrator@wp01-g1-rollback-test",
			"-release", "wp01-g1-runtime-negatives",
			"-evidence", "sanitized-wp01-g1-rollback",
		)
		if err == nil || !strings.Contains(out, `"status":"blocked"`) {
			t.Fatalf("injected forward failure did not block: err=%v out=%s", err, out)
		}

		conn, err = pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire rollback inspection connection: %v", err)
		}
		var forwardRows int
		var schemasAbsent bool
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM platform.schema_revisions WHERE version >= '0002'`).Scan(&forwardRows); err != nil {
			conn.Release()
			t.Fatalf("forward ledger after rollback: %v", err)
		}
		if err := conn.QueryRow(ctx, `SELECT to_regnamespace('kernel') IS NULL AND to_regnamespace('compat') IS NULL`).Scan(&schemasAbsent); err != nil {
			conn.Release()
			t.Fatalf("schema rollback inspection: %v", err)
		}
		gfp, ok, fpErr := tryGovernedFingerprint(ctx, conn)
		if _, err := conn.Exec(ctx, `
			DROP EVENT TRIGGER wp01_g1_fail_forward_0003;
			DROP FUNCTION public.wp01_g1_fail_forward_0003();
		`); err != nil {
			conn.Release()
			t.Fatalf("remove rollback fault: %v", err)
		}
		conn.Release()
		if forwardRows != 0 || !schemasAbsent || fpErr != nil || !ok || gfp != GovernedTargetFingerprint {
			t.Fatalf("rollback not exact: forward_rows=%d schemas_absent=%v fp=%s ok=%v err=%v",
				forwardRows, schemasAbsent, gfp, ok, fpErr)
		}

		// A retry from the rolled-back foundation must converge without manual
		// schema/data repair.
		retryOut := runForwardTestCLI(t, container, "forward",
			"-actor", "clarityit_migrator@wp01-g1-rollback-test",
			"-release", "wp01-g1-runtime-negatives",
			"-evidence", "sanitized-wp01-g1-rollback-retry",
		)
		if !strings.Contains(retryOut, ForwardTargetVersion) {
			t.Fatalf("rollback retry did not reach %s: %s", ForwardTargetVersion, retryOut)
		}
		verifyOut := runForwardTestCLI(t, container, "verify")
		if !strings.Contains(verifyOut, ForwardTargetManifestSHA256) {
			t.Fatalf("rollback retry target verification missing frozen manifest: %s", verifyOut)
		}
	})

	t.Run("advisory_lock_serializes_forward", func(t *testing.T) {
		const (
			container = "wp01-g1-forward-lock"
			port      = 56244
		)
		ctx := context.Background()
		pool := applyTestPool(t, container, port)
		base := Apply(ctx, pool, ApplyOptions{
			Actor:       "wp01-g1-stage-a-lock-test",
			ReleaseID:   "wp01-g1-runtime-negatives",
			EvidenceRef: "sanitized-wp01-g1-lock",
		})
		if base.Err != nil || base.GovernedFingerprint != GovernedTargetFingerprint {
			t.Fatalf("Stage A foundation: err=%v fp=%s", base.Err, base.GovernedFingerprint)
		}
		installForwardTestCLI(t, container, binaryPath)

		lockConn, err := AcquirePinnedConn(ctx, pool)
		if err != nil {
			t.Fatalf("acquire lock-holder connection: %v", err)
		}
		var state LockState
		if err := AcquireMigrationLock(ctx, lockConn, &state); err != nil {
			lockConn.Release()
			t.Fatalf("hold migration lock: %v", err)
		}

		out, runErr := runForwardTestCLIRaw(container, "forward",
			"-actor", "clarityit_migrator@wp01-g1-lock-test",
			"-release", "wp01-g1-runtime-negatives",
			"-evidence", "sanitized-wp01-g1-lock",
		)
		if runErr == nil || !strings.Contains(out, `"status":"blocked"`) {
			_ = ReleaseMigrationLock(ctx, &state)
			lockConn.Release()
			t.Fatalf("contended forward did not block: err=%v out=%s", runErr, out)
		}

		var forwardRows int
		var schemasAbsent bool
		if err := lockConn.QueryRow(ctx, `SELECT count(*) FROM platform.schema_revisions WHERE version >= '0002'`).Scan(&forwardRows); err != nil {
			_ = ReleaseMigrationLock(ctx, &state)
			lockConn.Release()
			t.Fatalf("forward ledger under contention: %v", err)
		}
		if err := lockConn.QueryRow(ctx, `SELECT to_regnamespace('kernel') IS NULL AND to_regnamespace('compat') IS NULL`).Scan(&schemasAbsent); err != nil {
			_ = ReleaseMigrationLock(ctx, &state)
			lockConn.Release()
			t.Fatalf("forward schemas under contention: %v", err)
		}
		if forwardRows != 0 || !schemasAbsent {
			_ = ReleaseMigrationLock(ctx, &state)
			lockConn.Release()
			t.Fatalf("lock contention allowed mutation: forward_rows=%d schemas_absent=%v", forwardRows, schemasAbsent)
		}
		if err := ReleaseMigrationLock(ctx, &state); err != nil {
			lockConn.Release()
			t.Fatalf("release migration lock: %v", err)
		}
		lockConn.Release()

		// The exact same package must proceed after the lock holder releases.
		retryOut := runForwardTestCLI(t, container, "forward",
			"-actor", "clarityit_migrator@wp01-g1-lock-test",
			"-release", "wp01-g1-runtime-negatives",
			"-evidence", "sanitized-wp01-g1-lock-retry",
		)
		if !strings.Contains(retryOut, ForwardTargetVersion) {
			t.Fatalf("post-lock retry did not reach %s: %s", ForwardTargetVersion, retryOut)
		}
	})
}

func runForwardTestCLIRaw(container, operation string, extra ...string) (string, error) {
	args := []string{
		"exec", "-u", "postgres", container,
		"/tmp/clarity-migrate", operation,
		"-dsn", forwardTestSocketDSN,
	}
	args = append(args, extra...)
	out, err := execCommandCombined("docker", args...)
	return strings.TrimSpace(out), err
}

// Small seam keeps raw CLI execution readable in the negative tests while the
// success-path helper remains fail-fast.
var execCommandCombined = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}
