package migration

// plan.go — the read-only `plan` command. Resolves the installation path
// (empty/install, approved_source/adopt, governed_current/no-op, or block)
// and lists the exact revisions and checksums that would apply. Performs
// zero mutation. All inspection runs in a READ ONLY transaction.

import (
	"context"
	"fmt"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
)

// Plan runs the read-only classification and returns the path + revisions.
// It reuses the full Preflight logic (the same one Apply consumes) so the plan
// is always consistent with what Apply would do.
func Plan(ctx context.Context, conn *pgx.Conn) (Result, error) {
	// Verify embedded identities first (same as Apply).
	vres, err := VerifyAll()
	if err != nil {
		return blockedResult(CodePackagingMismatch, ClassUnknownDrifted), nil
	}

	// Run the full preflight (read-only).
	pf, err := PreflightWithVerifier(ctx, conn, defaultPackageVerifier{})
	if err != nil && pf.Path != PathBlock {
		return Result{Status: "blocked", Code: pf.Code, Phase: PhasePreflight, DDLStarted: false, Class: pf.Class}, pfErr(err)
	}

	res := Result{
		Status:        "ok",
		Code:          pf.Code,
		Phase:         PhasePreflight,
		DDLStarted:    false,
		Class:         pf.Class,
		Path:          pf.Path,
		Composite:     vres.Composite,
		SourceProfile: pf.SourceFingerprint,
		GovernedFP:    pf.GovernedFingerprint,
		TargetVersion: "0001",
	}

	// List the revisions/checksums for the resolved path.
	switch pf.Path {
	case PathInstall:
		for _, name := range assets.FreshInstallChain {
			ts, err := Transform(name)
			if err != nil {
				continue
			}
			res.Diagnostics = append(res.Diagnostics, Diag{
				CheckID: "revision",
				Scope:   string(name),
				Result:  "planned",
				Detail:  fmt.Sprintf("source=%s transformed=%s", ts.SourceSHA256[:12], ts.TransformedSHA256[:12]),
			})
		}
	case PathAdopt:
		ts, err := Transform(assets.AssetAdoptP3)
		if err == nil {
			res.Diagnostics = append(res.Diagnostics, Diag{
				CheckID: "revision",
				Scope:   "adopt_p3",
				Result:  "planned",
				Detail:  fmt.Sprintf("source=%s transformed=%s", ts.SourceSHA256[:12], ts.TransformedSHA256[:12]),
			})
		}
	case PathNoOp:
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "classification",
			Result:  "no_op",
			Detail:  "database is already at the governed target",
		})
	case PathBlock:
		res.Status = "blocked"
	}

	return res, nil
}

func pfErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("plan: %w", err)
}
