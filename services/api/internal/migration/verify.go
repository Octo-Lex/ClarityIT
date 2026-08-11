package migration

// verify.go — the read-only `verify` command. Recomputes the governed target
// fingerprint, compares it against the frozen identity 9881c93e..., and runs a
// difference engine against the embedded G2 product + control manifests. Emits
// stable reason codes for any drift detected. Zero mutation.

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Verify runs the read-only verification and returns the result.
func Verify(ctx context.Context, conn *pgx.Conn) (Result, error) {
	// Verify embedded identities first.
	vres, err := VerifyAll()
	if err != nil {
		return blockedResult(CodePackagingMismatch, ClassUnknownDrifted), nil
	}

	// Begin a READ ONLY transaction.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhaseVerify, DDLStarted: false},
			fmt.Errorf("verify: begin read-only tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check platform exists.
	var hasPlatform bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='platform')`).Scan(&hasPlatform); err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhaseVerify, DDLStarted: false},
			fmt.Errorf("verify: probe platform: %w", err)
	}
	if !hasPlatform {
		return Result{
			Status: "blocked",
			Code:   CodeLedgerInconsistent,
			Phase:  PhaseVerify,
			Diagnostics: []Diag{{
				CheckID: "platform",
				Result:  "fail",
				Detail:  "platform schema does not exist; cannot verify an uninstalled database",
			}},
		}, nil
	}

	// Compute the governed fingerprint.
	signed, err := loadSignedG2()
	if err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhaseVerify}, fmt.Errorf("verify: load G2: %w", err)
	}
	control, err := loadControl()
	if err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhaseVerify}, fmt.Errorf("verify: load control: %w", err)
	}
	cap, err := fpGovernedCapture(ctx, tx, signed, control)
	if err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhaseVerify}, fmt.Errorf("verify: governed capture: %w", err)
	}
	fp, err := fpGovernedFingerprint(cap)
	if err != nil {
		return Result{Status: "blocked", Code: CodeUnknown, Phase: PhaseVerify}, fmt.Errorf("verify: governed fingerprint: %w", err)
	}

	res := Result{
		Status:     "verified",
		Code:       CodeOK,
		Phase:      PhaseVerify,
		DDLStarted: false,
		GovernedFP: fp,
		Composite:  vres.Composite,
	}

	// Compare against the frozen target.
	if fp != GovernedTargetFingerprint {
		res.Status = "blocked"
		res.Code = CodeDriftedGoverned
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "governed_fingerprint",
			Result:  "fail",
			Detail:  fmt.Sprintf("computed=%s frozen=%s (DRIFT)", fp[:12], GovernedTargetFingerprint[:12]),
		})
	} else {
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "governed_fingerprint",
			Result:  "pass",
			Detail:  fmt.Sprintf("computed=%s matches frozen target", fp[:12]),
		})
	}

	// Verify revision 0001 consistency. A missing revision or success=false is a
	// hard block — the structural fingerprint may match without the ledger being
	// correct, so these checks are independent.
	var revChecksum string
	var revSuccess bool
	err = tx.QueryRow(ctx, `SELECT checksum, success FROM platform.schema_revisions WHERE version='0001' LIMIT 1`).Scan(&revChecksum, &revSuccess)
	if err != nil {
		// Revision 0001 missing — block.
		res.Status = "blocked"
		res.Code = CodeLedgerInconsistent
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "revision_0001",
			Result:  "fail",
			Detail:  "revision 0001 is missing from the ledger",
		})
	} else {
		if revChecksum != BaselineChecksum {
			res.Diagnostics = append(res.Diagnostics, Diag{
				CheckID: "revision_checksum",
				Result:  "fail",
				Detail:  fmt.Sprintf("recorded=%s frozen=%s (MISMATCH)", revChecksum[:12], BaselineChecksum[:12]),
			})
			res.Status = "blocked"
			res.Code = CodeLedgerInconsistent
		} else {
			res.Diagnostics = append(res.Diagnostics, Diag{
				CheckID: "revision_checksum",
				Result:  "pass",
				Detail:  fmt.Sprintf("recorded=%s matches frozen baseline", revChecksum[:12]),
			})
		}
		if !revSuccess {
			res.Diagnostics = append(res.Diagnostics, Diag{
				CheckID: "revision_success",
				Result:  "fail",
				Detail:  "revision 0001 success=false",
			})
			res.Status = "blocked"
			res.Code = CodeLedgerInconsistent
		}
	}

	// Verify the complete revision set: at target version 0001, there must be
	// exactly one revision row. An unexpected extra/contradictory revision can
	// coexist with the exact governed structural fingerprint (ledger rows are
	// not part of the structural fingerprint), so this check is independent.
	var revCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM platform.schema_revisions`).Scan(&revCount); err != nil {
		return Result{
			Status:     "blocked",
			Code:       CodeLedgerInconsistent,
			Phase:      PhaseVerify,
			DDLStarted: false,
			Diagnostics: []Diag{{
				CheckID: "revision_count",
				Result:  "fail",
				Detail:  "unable to verify revision set",
			}},
		}, nil
	}
	if revCount != 1 {
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "revision_count",
			Result:  "fail",
			Detail:  fmt.Sprintf("expected exactly 1 revision row, found %d (contradictory/extra revision)", revCount),
		})
		res.Status = "blocked"
		res.Code = CodeLedgerInconsistent
	} else {
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "revision_count",
			Result:  "pass",
			Detail:  "exactly 1 revision row (0001)",
		})
	}

	return res, nil
}
