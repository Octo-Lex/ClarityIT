package migration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const forwardTestSocketDSN = "user=clarityit_migrator dbname=clarityit host=/var/run/postgresql sslmode=disable"

// TestForwardG1Convergence proves the two repository-sanctioned Stage-A paths
// available in CI (fresh install and P3 adoption) converge through the identical
// compiled Stage-B binary/package to the same frozen WP-01 G1 target.
//
// Stage B runs inside each disposable PostgreSQL container over its trusted Unix
// socket as the real database role clarityit_migrator. No password or role
// mutation is introduced; the binary must SET ROLE clarityit_owner through the
// frozen SET-only membership before forward DDL.
func TestForwardG1Convergence(t *testing.T) {
	binaryPath, producingCommit := buildForwardTestCLI(t)

	cases := []struct {
		name              string
		container         string
		port              int
		buildFoundation   func(*testing.T, string, int) *pgxpool.Pool
		wantStageAPath    Path
		wantSourceProfile string
	}{
		{
			name:      "fresh",
			container: "wp01-g1-forward-fresh",
			port:      56241,
			buildFoundation: func(t *testing.T, container string, port int) *pgxpool.Pool {
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
			bootstrapPool := tc.buildFoundation(t, tc.container, tc.port)

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

			// Exact 0001 must remain on the frozen WP-00 read model. Recheck the
			// fresh path here; the inherited G4/G5 matrix separately proves P3.
			if tc.name == "fresh" {
				stageAConn, err := pgx.Connect(ctx, applyDSN(tc.port))
				if err != nil {
					t.Fatalf("Stage-A read connection: %v", err)
				}
				hasForward, err := HasForwardRevision(ctx, stageAConn)
				stageAConn.Close(ctx)
				if err != nil {
					t.Fatalf("HasForwardRevision at 0001: %v", err)
				}
				if hasForward {
					t.Fatal("exact 0001 incorrectly routed to Stage B")
				}
			}

			installForwardTestCLI(t, tc.container, binaryPath)
			forwardOut := runForwardTestCLI(t, tc.container, "forward",
				"-actor", "clarityit_migrator@wp01-g1-test",
				"-release", "wp01-g1-convergence",
				"-evidence", "sanitized-wp01-g1-convergence",
			)
			if !strings.Contains(forwardOut, ForwardTargetVersion) {
				t.Fatalf("forward output missing target version %s: %s", ForwardTargetVersion, forwardOut)
			}

			// verify must pass exact ancestry, package digest and frozen target
			// manifest through the same binary and least-privilege role.
			verifyOut := runForwardTestCLI(t, tc.container, "verify")
			if !strings.Contains(verifyOut, ForwardTargetVersion) ||
				!strings.Contains(verifyOut, ForwardTargetManifestSHA256) {
				t.Fatalf("verify output missing frozen target identity: %s", verifyOut)
			}

			assertForwardLedgerAndPrivilegesContainer(
				t, tc.container, tc.wantSourceProfile, producingCommit,
			)

			// Replay is a verified no-op and must add neither a revision nor a run.
			replayOut := runForwardTestCLI(t, tc.container, "forward",
				"-actor", "clarityit_migrator@wp01-g1-test",
				"-release", "wp01-g1-convergence",
				"-evidence", "sanitized-wp01-g1-convergence-replay",
			)
			if !strings.Contains(replayOut, "no_op") {
				t.Fatalf("forward replay was not a no-op: %s", replayOut)
			}
			assertForwardLedgerAndPrivilegesContainer(
				t, tc.container, tc.wantSourceProfile, producingCommit,
			)
		})
	}
}

func buildForwardTestCLI(t *testing.T) (string, string) {
	t.Helper()
	sha := strings.TrimSpace(os.Getenv("GITHUB_SHA"))
	if ValidateProducingCommit(sha) != nil {
		out, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
		if err != nil {
			t.Fatalf("resolve test producing commit: %v: %s", err, strings.TrimSpace(string(out)))
		}
		sha = strings.TrimSpace(string(out))
	}
	if err := ValidateProducingCommit(sha); err != nil {
		t.Fatalf("test producing commit %q is invalid: %v", sha, err)
	}

	binaryPath := filepath.Join(t.TempDir(), "clarity-migrate")
	cmd := exec.Command(
		"go", "build", "-trimpath",
		"-ldflags", "-X main.ProducingCommit="+sha,
		"-o", binaryPath,
		"../../cmd/clarity-migrate",
	)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build clarity-migrate test binary: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return binaryPath, sha
}

func installForwardTestCLI(t *testing.T, container, binaryPath string) {
	t.Helper()
	if out, err := exec.Command("docker", "cp", binaryPath, container+":/tmp/clarity-migrate").CombinedOutput(); err != nil {
		t.Fatalf("copy clarity-migrate into %s: %v: %s", container, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("docker", "exec", container, "chmod", "0555", "/tmp/clarity-migrate").CombinedOutput(); err != nil {
		t.Fatalf("chmod clarity-migrate in %s: %v: %s", container, err, strings.TrimSpace(string(out)))
	}
}

func runForwardTestCLI(t *testing.T, container, operation string, extra ...string) string {
	t.Helper()
	args := []string{
		"exec", "-u", "postgres", container,
		"/tmp/clarity-migrate", operation,
		"-dsn", forwardTestSocketDSN,
	}
	args = append(args, extra...)
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("clarity-migrate %s in %s: %v: %s", operation, container, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out))
}

func assertForwardLedgerAndPrivilegesContainer(
	t *testing.T,
	container, wantSourceProfile, producingCommit string,
) {
	t.Helper()
	cat, err := ForwardCatalog()
	if err != nil {
		t.Fatalf("ForwardCatalog: %v", err)
	}

	ledger := runForwardPSQL(t, container, `
		SET ROLE clarityit_owner;
		SELECT version || '|' || checksum || '|' || name || '|' || source_commit
		FROM platform.schema_revisions ORDER BY version;
	`)
	lines := nonEmptyLines(ledger)
	if len(lines) != 1+len(cat) {
		t.Fatalf("revision ledger row count=%d want=%d: %q", len(lines), 1+len(cat), ledger)
	}
	first := strings.Split(lines[0], "|")
	if len(first) != 4 || first[0] != "0001" || first[1] != BaselineChecksum {
		t.Fatalf("revision 0001 ledger mismatch: %q", lines[0])
	}
	for i, rev := range cat {
		parts := strings.Split(lines[i+1], "|")
		if len(parts) != 4 || parts[0] != rev.Version || parts[1] != rev.Checksum ||
			parts[2] != rev.Name || parts[3] != producingCommit {
			t.Fatalf("forward revision %s ledger mismatch: %q", rev.Version, lines[i+1])
		}
	}

	runState := strings.TrimSpace(runForwardPSQL(t, container, fmt.Sprintf(`
		SET ROLE clarityit_owner;
		SELECT count(*)::text || '|' || COALESCE(max(source_profile_id),'')
		FROM platform.migration_runs
		WHERE target_version='%s' AND state='completed';
	`, ForwardTargetVersion)))
	if runState != "1|"+wantSourceProfile {
		t.Fatalf("forward run/source profile=%q want=%q", runState, "1|"+wantSourceProfile)
	}

	privileges := strings.TrimSpace(runForwardPSQL(t, container, `
		SET ROLE clarityit_owner;
		SELECT
			has_column_privilege('clarityit_app','kernel.outbox_messages','published_at','UPDATE')::int || '|' ||
			has_column_privilege('clarityit_app','kernel.outbox_messages','payload_digest','UPDATE')::int || '|' ||
			has_column_privilege('clarityit_app','kernel.inbox_messages','processed_at','UPDATE')::int || '|' ||
			has_column_privilege('clarityit_app','kernel.inbox_messages','payload_digest','UPDATE')::int || '|' ||
			has_table_privilege('clarityit_app','kernel.inbox_messages','DELETE')::int;
	`))
	if privileges != "1|0|1|0|0" {
		t.Fatalf("message privilege posture=%q want=1|0|1|0|0", privileges)
	}
}

func runForwardPSQL(t *testing.T, container, sql string) string {
	t.Helper()
	args := []string{
		"exec", "-u", "postgres", container,
		"psql", "-qAt", "-v", "ON_ERROR_STOP=1",
		"-U", "clarityit_migrator", "-d", "clarityit", "-c", sql,
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("psql migrator inspection in %s: %v: %s", container, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out))
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
