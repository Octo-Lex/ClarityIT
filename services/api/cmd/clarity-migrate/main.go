// Command clarity-migrate is the governed ClarityIT migration runner. WP-00
// Stage A remains the accepted version-0001 install/adoption path; WP-01 adds an
// ordered post-0001 Stage B in the same binary/package.
//
// It never offers the historical legacy 001-040 chain as a selectable path and
// has no provider credentials, provider clients, target-system access, effect
// dispatch, or general application authority.
//
// Usage:
//
//	clarity-migrate plan    [-dsn DATABASE_URL]
//	clarity-migrate apply   [-dsn DATABASE_URL] [-actor ACTOR] [-release RELEASE] [-evidence EVIDENCE_REF]
//	clarity-migrate forward [-dsn DATABASE_URL] [-actor ACTOR] [-release RELEASE] [-evidence EVIDENCE_REF]
//	clarity-migrate status  [-dsn DATABASE_URL]
//	clarity-migrate verify  [-dsn DATABASE_URL]
//
// `apply` is the unchanged WP-00 Stage-A operation and uses the privileged
// bootstrap/adoption connection. `forward` is WP-01 Stage B and requires a
// clarityit_migrator-capable connection. Read-only commands automatically use
// the forward-aware model once platform.schema_revisions exists.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/clarityit/api/internal/migration"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ProducingCommit string
	ReleaseID       string
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	sub := os.Args[1]
	if sub == "version" {
		fmt.Printf("clarity-migrate commit=%s release=%s\n", ProducingCommit, ReleaseID)
		return
	}

	fs := flag.NewFlagSet("clarity-migrate "+sub, flag.ExitOnError)
	dsn := fs.String("dsn", "", "database URL (defaults to DATABASE_URL env)")
	actor := fs.String("actor", "", "caller-supplied actor label (apply/forward)")
	release := fs.String("release", "", "release identifier (apply/forward)")
	evidence := fs.String("evidence", "", "sanitized evidence reference (apply/forward)")
	fs.Parse(os.Args[2:])

	if *dsn == "" {
		*dsn = os.Getenv("DATABASE_URL")
	}
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "clarity-migrate: no DSN (set DATABASE_URL or pass -dsn)")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	switch sub {
	case "plan":
		runReadOnlyAuto(ctx, *dsn, migration.Plan, migration.ForwardPlan)
	case "status":
		runReadOnlyAuto(ctx, *dsn, migration.Status, migration.ForwardStatus)
	case "verify":
		runReadOnlyAuto(ctx, *dsn, migration.Verify, migration.ForwardVerify)
	case "apply":
		runApply(ctx, *dsn, *actor, *release, *evidence)
	case "forward":
		runForward(ctx, *dsn, *actor, *release, *evidence)
	default:
		usage()
		os.Exit(2)
	}
}

func runReadOnlyAuto(
	ctx context.Context,
	dsn string,
	stageA func(context.Context, *pgx.Conn) (migration.Result, error),
	stageB func(context.Context, *pgx.Conn) (migration.Result, error),
) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		emitError("connect", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// Every subsequently opened transaction, including InspectForward's
	// SET-ROLE transaction, is enforced read-only by PostgreSQL.
	if _, err := conn.Exec(ctx, "SET default_transaction_read_only = on"); err != nil {
		emitError("set-read-only", err)
		os.Exit(1)
	}

	hasLedger, err := migration.HasPlatformLedger(ctx, conn)
	if err != nil {
		emitError("route-read-only", err)
		os.Exit(1)
	}
	fn := stageA
	if hasLedger {
		fn = stageB
	}
	res, err := fn(ctx, conn)
	if err != nil {
		emitError(sub, err)
		os.Exit(1)
	}
	migration.Emit(os.Stdout, res)
	if res.Status == "blocked" {
		os.Exit(1)
	}
}

// runApply preserves the accepted WP-00 Stage-A executor unchanged.
func runApply(ctx context.Context, dsn, actor, releaseID, evidenceRef string) {
	if actor == "" || releaseID == "" || evidenceRef == "" {
		fmt.Fprintln(os.Stderr, "clarity-migrate apply: -actor, -release, and -evidence are required")
		os.Exit(2)
	}
	migration.ProducingCommit = ProducingCommit
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		emitError("pool", err)
		os.Exit(1)
	}
	defer pool.Close()

	res := migration.Apply(ctx, pool, migration.ApplyOptions{Actor: actor, ReleaseID: releaseID, EvidenceRef: evidenceRef})
	emitApplyResult(res, "0001")
}

// runForward executes only WP-01 Stage B. The DSN must authenticate a principal
// permitted to SET ROLE clarityit_owner; production posture is clarityit_migrator.
func runForward(ctx context.Context, dsn, actor, releaseID, evidenceRef string) {
	if actor == "" || releaseID == "" || evidenceRef == "" {
		fmt.Fprintln(os.Stderr, "clarity-migrate forward: -actor, -release, and -evidence are required")
		os.Exit(2)
	}
	migration.ProducingCommit = ProducingCommit
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		emitError("pool", err)
		os.Exit(1)
	}
	defer pool.Close()

	res := migration.ApplyForward(ctx, pool, migration.ApplyOptions{Actor: actor, ReleaseID: releaseID, EvidenceRef: evidenceRef})
	emitApplyResult(res, migration.ForwardTargetVersion)
}

func emitApplyResult(res migration.ApplyResult, targetVersion string) {
	statusStr := "applied"
	if res.Path == migration.Path("no_op") {
		statusStr = "no_op"
	}
	out := migration.Result{
		Status: statusStr, Code: res.Code, Phase: migration.PhaseApply,
		DDLStarted: res.DDLStarted, Class: res.Class, Path: res.Path,
		GovernedFP: res.GovernedFingerprint, RunID: res.RunID,
		TargetVersion: targetVersion, DurationMs: res.ExecutionMs,
	}
	if res.Err != nil {
		out.Status = "blocked"
		out.Code = migration.CodeUnknown
		out.Diagnostics = []migration.Diag{{CheckID: "apply_error", Result: "fail", Detail: sanitizeErr(res.Err)}}
	}
	migration.Emit(os.Stdout, out)
	if res.Err != nil {
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: clarity-migrate <plan|apply|forward|status|verify|version> [flags]")
}

func emitError(ctx string, err error) {
	migration.Emit(os.Stdout, migration.Result{
		Status: "blocked", Code: migration.CodeUnknown, Phase: migration.PhasePreflight,
		Diagnostics: []migration.Diag{{CheckID: ctx, Result: "fail", Detail: sanitizeErr(err)}},
	})
}

func sanitizeErr(err error) string {
	if err == nil { return "" }
	s := err.Error()
	switch {
	case strings.Contains(s, "connect") || strings.Contains(s, "connection"):
		return "connection_failed"
	case strings.Contains(s, "preflight") || strings.Contains(s, "packaging"):
		return "preflight_failed"
	case strings.Contains(s, "lock"):
		return "lock_contention"
	case strings.Contains(s, "verify") || strings.Contains(s, "fingerprint") || strings.Contains(s, "manifest") || strings.Contains(s, "mismatch"):
		return "verification_failed"
	case strings.Contains(s, "commit") || strings.Contains(s, "rollback"):
		return "transaction_failed"
	default:
		return "apply_failed"
	}
}

var sub = "unknown"
