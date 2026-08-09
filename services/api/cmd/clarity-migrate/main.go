// Command clarity-migrate is the G4 Go migration runner. It provides plan,
// apply, status, and read-only verify behavior with stable machine-readable
// diagnostics and non-zero failure exits.
//
// The runner consumes only the frozen G3 artifacts embedded at build time. It
// never offers the legacy 001-040 chain as a selectable installation path. It
// holds no provider credentials, provider clients, target-system access, effect
// dispatch, or general application authority.
//
// Usage:
//
//	clarity-migrate plan    [-dsn DATABASE_URL]
//	clarity-migrate apply   [-dsn DATABASE_URL] [-actor ACTOR] [-release RELEASE] [-evidence EVIDENCE_REF]
//	clarity-migrate status  [-dsn DATABASE_URL]
//	clarity-migrate verify  [-dsn DATABASE_URL]
//
// The DSN defaults to the DATABASE_URL environment variable. Actor, release,
// and evidence-ref are required for apply. The producing commit is build-bound
// via -ldflags and is NOT accepted from CLI input.
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

// Build-time variables (set via -ldflags).
var (
	// ProducingCommit is the Git commit SHA this binary was built from. Set via:
	//   -ldflags "-X main.ProducingCommit=$(git rev-parse HEAD)"
	ProducingCommit string
	// ReleaseID is the release artifact identifier.
	ReleaseID string
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	sub := os.Args[1]

	// version doesn't need a DSN or flag parsing.
	if sub == "version" {
		fmt.Printf("clarity-migrate commit=%s release=%s\n", ProducingCommit, ReleaseID)
		return
	}

	fs := flag.NewFlagSet("clarity-migrate "+sub, flag.ExitOnError)
	dsn := fs.String("dsn", "", "database URL (defaults to DATABASE_URL env)")
	actor := fs.String("actor", "", "caller-supplied actor label (apply only)")
	release := fs.String("release", "", "release identifier (apply only)")
	evidence := fs.String("evidence", "", "sanitized evidence reference (apply only)")
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
		sub = "plan"
		runReadOnly(ctx, *dsn, migration.Plan)
	case "status":
		sub = "status"
		runReadOnly(ctx, *dsn, migration.Status)
	case "verify":
		sub = "verify"
		runReadOnly(ctx, *dsn, migration.Verify)
	case "apply":
		sub = "apply"
		runApply(ctx, *dsn, *actor, *release, *evidence)
	default:
		usage()
		os.Exit(2)
	}
}

// runReadOnly executes a read-only command (plan, status, verify) and emits the
// stable JSON result on stdout. Non-zero exit on failure.
func runReadOnly(ctx context.Context, dsn string, fn func(context.Context, *pgx.Conn) (migration.Result, error)) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		emitError("connect", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// Enforce read-only at the session level.
	if _, err := conn.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		emitError("set-read-only", err)
		os.Exit(1)
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

// runApply executes the apply command via a pool (Apply manages its own
// connection lifecycle — hijack + destroy).
func runApply(ctx context.Context, dsn, actor, releaseID, evidenceRef string) {
	if actor == "" || releaseID == "" || evidenceRef == "" {
		fmt.Fprintln(os.Stderr, "clarity-migrate apply: -actor, -release, and -evidence are required")
		os.Exit(2)
	}

	// Set the build-bound producing commit for the migration package.
	migration.ProducingCommit = ProducingCommit

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		emitError("pool", err)
		os.Exit(1)
	}
	defer pool.Close()

	res := migration.Apply(ctx, pool, migration.ApplyOptions{
		Actor:       actor,
		ReleaseID:   releaseID,
		EvidenceRef: evidenceRef,
	})

	// Emit the result document using the REAL ddl_started state from ApplyResult.
	statusStr := "applied"
	if res.Path == migration.Path("no_op") {
		statusStr = "no_op"
	}
	out := migration.Result{
		Status:    statusStr,
		Code:      res.Code,
		Phase:     migration.PhaseApply,
		DDLStarted: res.DDLStarted,
		Class:     res.Class,
		Path:      res.Path,
		GovernedFP: res.GovernedFingerprint,
		RunID:     res.RunID,
		TargetVersion: "0001",
		DurationMs: res.ExecutionMs,
	}
	if res.Err != nil {
		out.Status = "blocked"
		out.Code = migration.CodeUnknown
		out.DDLStarted = res.DDLStarted // preserve real state even on failure
		out.Diagnostics = []migration.Diag{{
			CheckID: "apply_error",
			Result:  "fail",
			Detail:  sanitizeErr(res.Err),
		}}
	}
	migration.Emit(os.Stdout, out)
	if res.Err != nil {
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: clarity-migrate <plan|apply|status|verify|version> [flags]")
}

func emitError(ctx string, err error) {
	migration.Emit(os.Stdout, migration.Result{
		Status: "blocked",
		Code:   migration.CodeUnknown,
		Phase:  migration.PhasePreflight,
		Diagnostics: []migration.Diag{{
			CheckID: ctx,
			Result:  "fail",
			Detail:  sanitizeErr(err),
		}},
	})
}

// sanitizeErr maps an error to a stable allowlisted diagnostic string. It never
// emits raw error text (which may contain DSNs, passwords, or SQL) to stdout.
// Raw errors go to stderr only; the result document carries a stable code.
func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	// Map to stable allowlisted reason codes based on the error content.
	switch {
	case strings.Contains(s, "connect") || strings.Contains(s, "connection"):
		return "connection_failed"
	case strings.Contains(s, "preflight") || strings.Contains(s, "packaging"):
		return "preflight_failed"
	case strings.Contains(s, "lock"):
		return "lock_contention"
	case strings.Contains(s, "verify") || strings.Contains(s, "fingerprint") || strings.Contains(s, "mismatch"):
		return "verification_failed"
	case strings.Contains(s, "commit") || strings.Contains(s, "rollback"):
		return "transaction_failed"
	default:
		return "apply_failed"
	}
}

// sub holds the current subcommand name for error context.
var sub = "unknown"
