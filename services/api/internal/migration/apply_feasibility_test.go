package migration

// apply_feasibility_test.go — two load-bearing feasibility tests that MUST pass
// before apply.go is built. They prove the frozen role transitions allow the
// runner's post-body evidence writes (migration_runs + reconciliation_results)
// inside the SAME target transaction, and that rollback restores empty/exact
// state.
//
// These resolve the structural question: can the runner record evidence after
// the frozen artifact chain executes, given that:
//   - the artifacts SET LOCAL ROLE clarityit_owner / RESET SESSION AUTHORIZATION;
//   - the adoption artifact demotes `clarityit` to NOSUPERUSER as its last
//     mutation;
//   - the platform schema (and thus migration_runs) is created INSIDE the
//     target transaction by the frozen artifacts.

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/clarityit/api/internal/migration/fingerprint"
	"github.com/jackc/pgx/v5"
)

// execTransformedOnTx executes a transformed artifact body via the simple query
// protocol on the given transaction, fully draining + closing the MultiResultReader
// before returning. Any result-level or close error is a failure.
func execTransformedOnTx(ctx context.Context, tx pgx.Tx, body string) error {
	mrr := tx.Conn().PgConn().Exec(ctx, body)
	for mrr.NextResult() {
		// drain every result
	}
	return mrr.Close()
}

// TestApplyFeasibility_FreshInstall proves the fresh-install path can, inside
// ONE outer transaction:
//  1. execute the transformed roles/platform/baseline/seed artifacts;
//  2. confirm revision 0001 exists with the frozen checksum + success;
//  3. confirm the active role can INSERT into platform.migration_runs AND
//     platform.reconciliation_results (the evidence writes the runner needs);
//  4. compute the governed fingerprint == 9881c93e…;
//  5. ROLL BACK and prove the database returns to empty (no platform schema,
//     no roles, no product tables).
func TestApplyFeasibility_FreshInstall(t *testing.T) {
	conn := startFixture(t, "g4-feas-fresh", 55701)
	ctx := context.Background()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	// Capture pre-state sentinels for the rollback-empty assertion.
	preUserSchemas := countRows(t, ctx, conn, `SELECT count(*) FROM pg_namespace WHERE nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND nspname NOT LIKE 'pg_toast_temp_%'`)
	preUserRoles := countRows(t, ctx, conn, `SELECT count(*) FROM pg_roles WHERE rolname !~ '^pg_' AND rolname NOT IN ('postgres')`)

	// Execute the transformed chain on the SAME transaction. SET ROLE NONE before
	// each fresh artifact body so each starts from current_user=session_user,
	// emulating what separate-session application (the frozen-identity derivation)
	// implicitly does. Without boundary normalization, SET LOCAL ROLE from an
	// earlier artifact (platform) persists into the baseline's CREATE EXTENSION,
	// changing extension ownership and producing a different governed fingerprint.
	// SET ROLE NONE is preferred over RESET ROLE because it deterministically
	// restores current_user=session_user regardless of connection-time role options.
	for _, name := range []assets.AssetName{assets.AssetRolesBootstrap, assets.AssetPlatformSchema, assets.AssetBaseline, assets.AssetSeed} {
		if _, err := tx.Exec(ctx, "SET ROLE NONE"); err != nil {
			tx.Rollback(ctx)
			t.Fatalf("set role none before %s: %v", name, err)
		}
		// Assert boundary normalization: current_user must equal session_user.
		var eq bool
		if err := tx.QueryRow(ctx, "SELECT current_user = session_user").Scan(&eq); err != nil {
			tx.Rollback(ctx)
			t.Fatalf("boundary check before %s: %v", name, err)
		}
		if !eq {
			tx.Rollback(ctx)
			t.Fatalf("boundary normalization failed before %s: current_user != session_user", name)
		}
		ts, err := Transform(name)
		if err != nil {
			t.Fatalf("transform %s: %v", name, err)
		}
		if err := execTransformedOnTx(ctx, tx, string(ts.Body)); err != nil {
			tx.Rollback(ctx)
			t.Fatalf("exec %s: %v", name, err)
		}
	}

	// 2. Confirm revision 0001 with frozen checksum + success.
	var revCount int
	var checksum string
	var success bool
	if err := tx.QueryRow(ctx, `SELECT count(*), (SELECT checksum FROM platform.schema_revisions WHERE version='0001' LIMIT 1), (SELECT success FROM platform.schema_revisions WHERE version='0001' LIMIT 1) FROM platform.schema_revisions WHERE version='0001'`).Scan(&revCount, &checksum, &success); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("query revision: %v", err)
	}
	if revCount != 1 || checksum != BaselineChecksum || !success {
		tx.Rollback(ctx)
		t.Fatalf("revision 0001: count=%d checksum=%s success=%v (want 1/%s/true)", revCount, checksum, success, BaselineChecksum)
	}

	// 4. Compute the governed fingerprint == 9881c93e… BEFORE the evidence
	//    inserts (the migration_runs/reconciliation rows are governed objects
	//    and would change the projection if present at verification time).
	signed, _ := loadSignedG2()
	control, _ := loadControl()
	cap, err := fingerprint.GovernedCapture(ctx, tx, signed, control)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("governed capture: %v", err)
	}
	fp, err := fingerprint.GovernedFingerprint(cap)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("governed fingerprint: %v", err)
	}
	if fp != GovernedTargetFingerprint {
		// Debug: dump the in-tx projection for diffing against the Python committed oracle.
		b, _ := json.MarshalIndent(cap, "", "  ")
		_ = os.WriteFile("fingerprint/testdata/governed_in_tx_GO_debug.json", b, 0644)
		tx.Rollback(ctx)
		t.Fatalf("governed fingerprint (in-tx): got %s want %s\n(in-tx projection written to testdata/governed_in_tx_GO_debug.json)", fp, GovernedTargetFingerprint)
	}

	// 3. Confirm the active role can INSERT the evidence rows.
	var sessionUser, currentUser string
	if err := tx.QueryRow(ctx, `SELECT session_user, current_user`).Scan(&sessionUser, &currentUser); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("query session/current user: %v", err)
	}
	t.Logf("after fresh chain: session_user=%s current_user=%s", sessionUser, currentUser)

	runID := "00000000-0000-0000-0000-000000000001"
	if _, err := tx.Exec(ctx, `INSERT INTO platform.migration_runs (run_id, target_version, state, started_at, release_id, evidence_ref) VALUES ($1, '0001', 'completed', now(), 'feas-test', 'sanitized')`, runID); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("insert migration_runs: %v (active role %s lacks privilege — DESIGN CONFLICT)", err, currentUser)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO platform.reconciliation_results (run_id, check_id, scope, expected, actual, result, evidence_ref, recorded_at) VALUES ($1, 'governed_fingerprint', 'target', '{"fingerprint":"9881c93e..."}'::jsonb, '{"fingerprint":"9881c93e..."}'::jsonb, 'pass', 'sanitized', now())`, runID); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("insert reconciliation_results: %v (active role %s lacks privilege — DESIGN CONFLICT)", err, currentUser)
	}

	// 5. ROLL BACK and prove empty restoration.
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	postUserSchemas := countRows(t, ctx, conn, `SELECT count(*) FROM pg_namespace WHERE nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND nspname NOT LIKE 'pg_toast_temp_%'`)
	postUserRoles := countRows(t, ctx, conn, `SELECT count(*) FROM pg_roles WHERE rolname !~ '^pg_' AND rolname NOT IN ('postgres')`)
	if postUserSchemas != preUserSchemas {
		t.Errorf("ROLLBACK FAILED: user schemas %d -> %d (not restored to empty)", preUserSchemas, postUserSchemas)
	}
	if postUserRoles != preUserRoles {
		t.Errorf("ROLLBACK FAILED: user roles %d -> %d (roles leaked past rollback)", preUserRoles, postUserRoles)
	}
	t.Logf("fresh feasibility OK: governed=%s, evidence writes succeeded as %s, rollback restored empty", fp[:12], currentUser)
}

// TestApplyFeasibility_P3Adoption proves the P3-adoption path can, inside ONE
// outer transaction:
//  1. execute the transformed adoption artifact against a prepared P3 source;
//  2. inspect session_user/current_user/role attributes after the artifact's
//     SET SESSION AUTHORIZATION + demotion;
//  3. attempt the migration_runs + reconciliation_results writes;
//  4. compute the governed fingerprint == 9881c93e…;
//  5. ROLL BACK and prove the P3 source is restored exactly.
//
// CRITICAL: the adoption artifact demotes `clarityit` to NOSUPERUSER as its
// last mutation. If the runner connects AS `clarityit` and the demotion removes
// the privileges needed for the post-body evidence writes, this is a G4 design
// conflict that must be recorded, NOT silently solved by grant changes or
// helper functions.
func TestApplyFeasibility_P3Adoption(t *testing.T) {
	// Build a P3 source fixture (clarityit superuser + P3 schema/seed) using the
	// pinned image that reproduces cedf689d.
	dsn := buildP3SourceFixture(t)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("P3 fixture unreachable: %v", err)
	}
	defer conn.Close(ctx)

	// Capture pre-state for the rollback-restoration assertion: the source FP.
	preSourceFP := computeSourceFP(t, ctx, conn)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	// Bind the runtime producing commit (the adoption artifact requires it).
	// Execute set_config as a parameterized extended-protocol statement FIRST.
	if _, err := tx.Exec(ctx, `SELECT set_config('g3.source_commit', $1, true)`, P3SourceCommit); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("set_config: %v", err)
	}

	// Execute the transformed adoption body (set_config line removed by Transform).
	ts, err := Transform(assets.AssetAdoptP3)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("transform adoption: %v", err)
	}
	if err := execTransformedOnTx(ctx, tx, string(ts.Body)); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("exec adoption: %v", err)
	}

	// 2. Inspect session_user/current_user + role attributes post-demotion. The
	// adoption artifact's RESET SESSION AUTHORIZATION restores the originally-
	// authenticated user — which was renamed from `clarityit` to
	// `legacy_ext_owner`. So the session lands on `legacy_ext_owner`, which
	// RETAINS the bootstrap role's rolsuper=true (only rolcanlogin was changed
	// to false). The evidence writes succeed through this residual superuser
	// authority, NOT through ordinary platform-table grants. This means the
	// adoption connection MUST be physically destroyed after apply (never
	// returned to the pool), because NOLOGIN prevents new logins but does not
	// de-privilege the existing superuser session.
	var sessionUser, currentUser string
	if err := tx.QueryRow(ctx, `SELECT session_user, current_user`).Scan(&sessionUser, &currentUser); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("query users: %v", err)
	}
	t.Logf("after adoption: session_user=%s current_user=%s", sessionUser, currentUser)
	// Assert the precise post-adoption identity.
	if sessionUser != "legacy_ext_owner" || currentUser != "legacy_ext_owner" {
		tx.Rollback(ctx)
		t.Fatalf("post-adoption identity: session_user=%s current_user=%s, want legacy_ext_owner/legacy_ext_owner", sessionUser, currentUser)
	}
	var rolsuper, rolcanlogin bool
	if err := tx.QueryRow(ctx, `SELECT rolsuper, rolcanlogin FROM pg_roles WHERE rolname='legacy_ext_owner'`).Scan(&rolsuper, &rolcanlogin); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("query legacy_ext_owner attrs: %v", err)
	}
	if !rolsuper {
		tx.Rollback(ctx)
		t.Fatalf("legacy_ext_owner.rolsuper=false; evidence writes succeed through residual superuser (this must be true)")
	}
	if rolcanlogin {
		tx.Rollback(ctx)
		t.Fatalf("legacy_ext_owner.rolcanlogin=true; adoption must have set it NOLOGIN")
	}
	t.Logf("legacy_ext_owner: rolsuper=%v rolcanlogin=%v (evidence writes via residual superuser)", rolsuper, rolcanlogin)

	// 3. Attempt the evidence writes.
	runID := "00000000-0000-0000-0000-000000000002"
	if _, err := tx.Exec(ctx, `INSERT INTO platform.migration_runs (run_id, source_profile_id, target_version, state, started_at, release_id, evidence_ref) VALUES ($1, $2, '0001', 'completed', now(), 'feas-test', 'sanitized')`, runID, P3ProfileID); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("insert migration_runs after adoption: %v (DEMOTION CONFLICT — post-body evidence cannot be written by %s; this is a G4 design conflict, record it)", err, currentUser)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO platform.reconciliation_results (run_id, check_id, scope, expected, actual, result, evidence_ref, recorded_at) VALUES ($1, 'governed_fingerprint', 'target', '{"fingerprint":"9881c93e..."}'::jsonb, '{"fingerprint":"9881c93e..."}'::jsonb, 'pass', 'sanitized', now())`, runID); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("insert reconciliation_results after adoption: %v (DEMOTION CONFLICT)", err)
	}

	// 4. Governed fingerprint == 9881c93e…
	signed, _ := loadSignedG2()
	control, _ := loadControl()
	cap, err := fingerprint.GovernedCapture(ctx, tx, signed, control)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("governed capture: %v", err)
	}
	fp, err := fingerprint.GovernedFingerprint(cap)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("governed fingerprint: %v", err)
	}
	if fp != GovernedTargetFingerprint {
		tx.Rollback(ctx)
		t.Fatalf("governed fingerprint after adoption: got %s want %s", fp, GovernedTargetFingerprint)
	}

	// 5. ROLL BACK and prove P3 source restored.
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	postSourceFP := computeSourceFP(t, ctx, conn)
	if postSourceFP != preSourceFP {
		t.Errorf("ROLLBACK FAILED: P3 source fingerprint changed %s -> %s (source not restored)", preSourceFP[:12], postSourceFP[:12])
	}
	t.Logf("adoption feasibility OK: governed=%s, evidence writes succeeded as %s, rollback restored P3 source", fp[:12], currentUser)
}

// buildP3SourceFixture brings up a P3 source DB (clarityit superuser, pinned
// image, schema+seed) and returns its DSN. Skips if Docker/image unavailable.
func buildP3SourceFixture(t *testing.T) string {
	t.Helper()
	name := "g4-feas-p3"
	exec.Command("docker", "rm", "-f", name).Run()
	if err := exec.Command("docker", "run", "-d", "--name", name,
		"-e", "POSTGRES_PASSWORD=clarityit",
		"-e", "POSTGRES_DB=clarityit",
		"-e", "POSTGRES_USER=clarityit",
		"-p", "55702:5432",
		"postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382").Run(); err != nil {
		t.Skipf("docker/pinned-image unavailable for P3 fixture: %v", err)
	}
	t.Cleanup(func() { exec.Command("docker", "rm", "-f", name).Run() })
	// Wait for connection.
	var err error
	for i := 0; i < 60; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		c, e := pgx.Connect(ctx, "postgres://clarityit:clarityit@localhost:55702/clarityit?sslmode=disable")
		cancel()
		if e == nil {
			c.Close(context.Background())
			err = nil
			break
		}
		err = e
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Skipf("P3 fixture connect failed: %v", err)
	}
	// Apply P3 schema + seed.
	for _, rel := range []string{"migrations/profiles/p3/schema.sql", "migrations/profiles/p3/seed.sql"} {
		sql, e := os.ReadFile("../../../../" + rel)
		if e != nil {
			t.Skipf("read %s: %v", rel, e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		c, e := pgx.Connect(ctx, "postgres://clarityit:clarityit@localhost:55702/clarityit?sslmode=disable")
		if e != nil {
			cancel()
			t.Skipf("connect for apply: %v", e)
		}
		// strip \set lines for pgx
		body := stripSetForP3(string(sql))
		mrr := c.PgConn().Exec(ctx, body)
		for mrr.NextResult() {
		}
		if e := mrr.Close(); e != nil {
			c.Close(ctx)
			cancel()
			t.Skipf("apply %s: %v", rel, e)
		}
		c.Close(ctx)
		cancel()
	}
	return "postgres://clarityit:clarityit@localhost:55702/clarityit?sslmode=disable"
}

func stripSetForP3(sql string) string {
	var out []string
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), `\set`) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func computeSourceFP(t *testing.T, ctx context.Context, conn *pgx.Conn) string {
	t.Helper()
	cap, err := fingerprint.ProfilerCapture(ctx, conn)
	if err != nil {
		t.Fatalf("source capture: %v", err)
	}
	fp, err := fingerprint.ProfilerFingerprint(cap)
	if err != nil {
		t.Fatalf("source fingerprint: %v", err)
	}
	return fp
}

func countRows(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(ctx, sql).Scan(&n); err != nil {
		t.Fatalf("count query failed: %v\nsql: %s", err, sql)
	}
	return n
}
