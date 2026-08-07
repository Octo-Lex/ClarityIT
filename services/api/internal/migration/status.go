package migration

// status.go — the read-only `status` command. Shows the current migration state:
// platform ledger (migration_runs, schema_revisions), source profile, governed
// fingerprint, and compatibility. Emits {"state":"uninstalled"} when the
// platform schema is absent. Zero mutation; all inspection is read-only.

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Status runs the read-only status query and returns the current migration state.
func Status(ctx context.Context, conn *pgx.Conn) (Result, error) {
	// Verify embedded identities first.
	vres, err := VerifyAll()
	if err != nil {
		return blockedResult(CodePackagingMismatch, ClassUnknownDrifted), nil
	}

	// Begin a READ ONLY transaction.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhasePreflight, DDLStarted: false},
			fmt.Errorf("status: begin read-only tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if the platform schema exists.
	var hasPlatform bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='platform')`).Scan(&hasPlatform); err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhasePreflight, DDLStarted: false},
			fmt.Errorf("status: probe platform: %w", err)
	}

	if !hasPlatform {
		// No platform schema → uninstalled (a valid status, not an error).
		return Result{
			Status:    "ok",
			Code:      CodeOK,
			Phase:     PhasePreflight,
			DDLStarted: false,
			Class:     ClassEmptyInstall,
			Path:      PathInstall,
			Composite: vres.Composite,
			Diagnostics: []Diag{{
				CheckID: "platform",
				Result:  "absent",
				Detail:  "platform schema does not exist; database is uninstalled",
			}},
		}, nil
	}

	// Platform exists — read the ledger state.
	var revVersion, revName, revChecksum, revAppliedBy string
	var revSuccess bool
	_ = tx.QueryRow(ctx, `SELECT version, name, checksum, applied_by, success FROM platform.schema_revisions ORDER BY version DESC LIMIT 1`).
		Scan(&revVersion, &revName, &revChecksum, &revAppliedBy, &revSuccess)

	var runCount, activeCount int
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM platform.migration_runs`).Scan(&runCount)
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM platform.migration_runs WHERE state NOT IN ('completed','blocked','precommit_rolled_back')`).Scan(&activeCount)

	// Classify — compute fingerprints like Preflight does (probeAll doesn't).
	probe, _ := probeAll(ctx, tx)
	if !probe.Fresh {
		// Source fingerprint (meaningful on a non-empty DB).
		if cap, err := profilerCaptureAdapter(ctx, tx); err == nil {
			if fp, err := profilerFingerprintAdapter(cap); err == nil {
				probe.SourceFingerprint = fp
			}
		}
		// Governed fingerprint.
		if gfp, ok, _ := tryGovernedFingerprint(ctx, tx); ok {
			probe.GovernedFingerprint = gfp
		}
	}
	class, path, code := Classify(probe)

	return Result{
		Status:          "ok",
		Code:            code,
		Phase:           PhasePreflight,
		DDLStarted:      false,
		Class:           class,
		Path:            path,
		Composite:       vres.Composite,
		TargetVersion:   revVersion,
		SourceProfile:   probe.SourceFingerprint,
		GovernedFP:      probe.GovernedFingerprint,
		Diagnostics: []Diag{
			{CheckID: "revision", Scope: revVersion, Result: fmt.Sprintf("success=%v", revSuccess), Detail: fmt.Sprintf("%s checksum=%s applied_by=%s", revName, revChecksum[:12], revAppliedBy)},
			{CheckID: "runs", Scope: "migration_runs", Result: fmt.Sprintf("total=%d active=%d", runCount, activeCount)},
		},
	}, nil
}
