package migration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestForwardG1Convergence proves the two repository-sanctioned Stage-A paths
// available in CI (fresh install and P3 adoption) converge through the identical
// least-privilege Stage-B package to the same frozen WP-01 G1 target.
func TestForwardG1Convergence(t *testing.T) {
	cases := []struct {
		name              string
		container         string
		port              int
		buildFoundation   func(*testing.T, int, string) *pgxpool.Pool
		wantStageAPath    Path
		wantSourceProfile string
	}{
		{
			name:      "fresh",
			container: "wp01-g1-forward-fresh",
			port:      56241,
			buildFoundation: func(t *testing.T, port int, container string) *pgxpool.Pool {
				return applyTestPool(t, container, port)
			},
			wantStageAPath: PathInstall,
		},
		{
			name:              "p3",
			container:         "wp01-g1-forward-p3",
			port:              56242,
			buildFoundation:   buildP3FixturePool,
			wantStageAPath:    PathAdopt,
			wantSourceProfile: P3ProfileID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			bootstrapPool := tc.buildFoundation(t, tc.port, tc.container)

			base := Apply(ctx, bootstrapPool, ApplyOptions{
				Actor:       "wp01-g1-stage-a-test",
				ReleaseID:   "wp01-g1-convergence",
				EvidenceRef: "sanitized-wp01-g1-convergence",
			})
			if base.Err != nil {
				t.Fatalf("Stage A: %v", base.Err)
			}
			if base.Path != tc.wantStageAPath {
				t.Fatalf("Stage A path=%q want=%q", base.Path, tc.wantStageAPath)
			}
			if base.GovernedFingerprint != GovernedTargetFingerprint {
				t.Fatalf("Stage A governed fp=%s want=%s", base.GovernedFingerprint, GovernedTargetFingerprint)
			}

			// Exact revision 0001 remains on the accepted WP-00 read model. This
			// routing probe requires no direct platform-table privilege.
			stageAConn, err := pgx.Connect(ctx, applyDSN(tc.port))
			if err != nil {
				t.Fatalf("Stage-A read connection: %v", err)
			}
			hasForward, err := HasForwardRevision(ctx, stageAConn)
			if err != nil {
				stageAConn.Close(ctx)
				t.Fatalf("HasForwardRevision at 0001: %v", err)
			}
			if hasForward {
				stageAConn.Close(ctx)
				t.Fatal("exact 0001 incorrectly routed to Stage B")
			}

			// Production SQL intentionally stores no password. The fixture gives
			// the actual least-privilege migrator a test-only password, then all
			// Stage-B execution/inspection uses that login and SET-only owner role.
			if _, err := stageAConn.Exec(ctx, `ALTER ROLE clarityit_migrator PASSWORD 'wp01-g1-test-only'`); err != nil {
				stageAConn.Close(ctx)
				t.Fatalf("provision fixture migrator auth: %v", err)
			}
			stageAConn.Close(ctx)

			migratorPool := openTestMigratorPool(t, ctx, tc.port)
			defer migratorPool.Close()

			forward := ApplyForward(ctx, migratorPool, ApplyOptions{
				Actor:       "clarityit_migrator@wp01-g1-test",
				ReleaseID:   "wp01-g1-convergence",
				EvidenceRef: "sanitized-wp01-g1-convergence",
			})
			if forward.Err != nil {
				t.Fatalf("Stage B: %v", forward.Err)
			}
			if forward.Path != Path("forward") {
				t.Fatalf("Stage B path=%q want=forward", forward.Path)
			}

			conn, err := migratorPool.Acquire(ctx)
			if err != nil {
				t.Fatalf("acquire migrator inspection connection: %v", err)
			}
			inspectConn := conn.Conn()
			ins, err := InspectForward(ctx, inspectConn)
			if err != nil {
				conn.Release()
				t.Fatalf("InspectForward: %v", err)
			}
			if !ins.Current || ins.CurrentVersion != ForwardTargetVersion {
				conn.Release()
				t.Fatalf("current=%v version=%s want=true/%s", ins.Current, ins.CurrentVersion, ForwardTargetVersion)
			}
			if ins.PackageDigest != ForwardPackageSHA256 {
				conn.Release()
				t.Fatalf("package=%s want=%s", ins.PackageDigest, ForwardPackageSHA256)
			}
			if ins.ManifestDigest != ForwardTargetManifestSHA256 {
				conn.Release()
				t.Fatalf("manifest=%s want=%s", ins.ManifestDigest, ForwardTargetManifestSHA256)
			}
			hasForward, err = HasForwardRevision(ctx, inspectConn)
			if err != nil || !hasForward {
				conn.Release()
				t.Fatalf("post-forward routing hasForward=%v err=%v", hasForward, err)
			}
			conn.Release()

			assertForwardLedgerAndPrivileges(t, ctx, migratorPool, tc.wantSourceProfile)

			// Replay is a verified no-op: no duplicate revision or migration run.
			replay := ApplyForward(ctx, migratorPool, ApplyOptions{
				Actor:       "clarityit_migrator@wp01-g1-test",
				ReleaseID:   "wp01-g1-convergence",
				EvidenceRef: "sanitized-wp01-g1-convergence-replay",
			})
			if replay.Err != nil {
				t.Fatalf("Stage B replay: %v", replay.Err)
			}
			if replay.Path != PathNoOp {
				t.Fatalf("Stage B replay path=%q want=%q", replay.Path, PathNoOp)
			}
			assertForwardLedgerAndPrivileges(t, ctx, migratorPool, tc.wantSourceProfile)
		})
	}
}

func openTestMigratorPool(t *testing.T, ctx context.Context, port int) *pgxpool.Pool {
	t.Helper()
	dsn := fmt.Sprintf("postgres://clarityit_migrator:wp01-g1-test-only@localhost:%d/clarityit?sslmode=disable", port)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("migrator pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("migrator ping: %v", err)
	}
	return pool
}

func assertForwardLedgerAndPrivileges(t *testing.T, ctx context.Context, pool *pgxpool.Pool, wantSourceProfile string) {
	t.Helper()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire ledger connection: %v", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin ledger verification: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE clarityit_owner`); err != nil {
		t.Fatalf("ledger verification SET ROLE: %v", err)
	}

	rows, err := tx.Query(ctx, `SELECT version,name,checksum FROM platform.schema_revisions ORDER BY version`)
	if err != nil {
		t.Fatalf("read revision ledger: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var version, name, checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			t.Fatalf("scan revision ledger: %v", err)
		}
		got = append(got, strings.Join([]string{version, name, checksum}, ":"))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("revision rows: %v", err)
	}
	cat, err := ForwardCatalog()
	if err != nil {
		t.Fatalf("ForwardCatalog: %v", err)
	}
	want := []string{"0001:" + ledger0001Name(t, tx) + ":" + BaselineChecksum}
	for _, rev := range cat {
		want = append(want, strings.Join([]string{rev.Version, rev.Name, rev.Checksum}, ":"))
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("revision ledger mismatch\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	var runCount int
	var sourceProfile string
	if err := tx.QueryRow(ctx, `
		SELECT count(*), COALESCE(max(source_profile_id),'')
		FROM platform.migration_runs
		WHERE target_version=$1 AND state='completed'`, ForwardTargetVersion).Scan(&runCount, &sourceProfile); err != nil {
		t.Fatalf("forward migration run: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("forward completed run count=%d want=1", runCount)
	}
	if sourceProfile != wantSourceProfile {
		t.Fatalf("forward source profile=%q want=%q", sourceProfile, wantSourceProfile)
	}

	var outboxPublishedUpdate, outboxPayloadUpdate bool
	var inboxProcessedUpdate, inboxPayloadUpdate, inboxDelete bool
	if err := tx.QueryRow(ctx, `SELECT
		has_column_privilege('clarityit_app','kernel.outbox_messages','published_at','UPDATE'),
		has_column_privilege('clarityit_app','kernel.outbox_messages','payload_digest','UPDATE'),
		has_column_privilege('clarityit_app','kernel.inbox_messages','processed_at','UPDATE'),
		has_column_privilege('clarityit_app','kernel.inbox_messages','payload_digest','UPDATE'),
		has_table_privilege('clarityit_app','kernel.inbox_messages','DELETE')`).Scan(
		&outboxPublishedUpdate, &outboxPayloadUpdate, &inboxProcessedUpdate, &inboxPayloadUpdate, &inboxDelete,
	); err != nil {
		t.Fatalf("message privilege verification: %v", err)
	}
	if !outboxPublishedUpdate || outboxPayloadUpdate || !inboxProcessedUpdate || inboxPayloadUpdate || inboxDelete {
		t.Fatalf("message privilege posture published=%v outbox_payload=%v processed=%v inbox_payload=%v inbox_delete=%v",
			outboxPublishedUpdate, outboxPayloadUpdate, inboxProcessedUpdate, inboxPayloadUpdate, inboxDelete)
	}
}

func ledger0001Name(t *testing.T, tx pgx.Tx) string {
	t.Helper()
	var name string
	if err := tx.QueryRow(context.Background(), `SELECT name FROM platform.schema_revisions WHERE version='0001'`).Scan(&name); err != nil {
		t.Fatalf("revision 0001 name: %v", err)
	}
	return name
}
