package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const CodeForwardPending ReasonCode = "FORWARD_REVISIONS_PENDING"

// HasPlatformLedger is a minimal read-only router for the CLI. The platform
// schema/revision table is created only by the accepted WP-00 Stage-A path; once
// present, the WP-01 release must use the forward-aware read model rather than
// the pre-forward 9881... classifier.
func HasPlatformLedger(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var exists bool
	if err := conn.QueryRow(ctx, `SELECT to_regclass('platform.schema_revisions') IS NOT NULL`).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func ForwardPlan(ctx context.Context, conn *pgx.Conn) (Result, error) {
	ins, err := InspectForward(ctx, conn)
	if err != nil {
		return forwardBlocked(PhasePreflight, err), nil
	}
	res := Result{
		Status: "ok", Code: CodeOK, Phase: PhasePreflight, DDLStarted: false,
		Class: ClassGovernedCurrent, Path: Path("forward"), TargetVersion: ForwardTargetVersion,
		Diagnostics: []Diag{{CheckID: "forward_package", Result: "pass", Detail: ins.PackageDigest}},
	}
	if ins.Current {
		res.Path = PathNoOp
		res.Diagnostics = append(res.Diagnostics,
			Diag{CheckID: "forward_revision", Scope: ForwardTargetVersion, Result: "current"},
			Diag{CheckID: "forward_manifest", Result: "pass", Detail: ins.ManifestDigest})
		return res, nil
	}
	cat, err := ForwardCatalog()
	if err != nil {
		return forwardBlocked(PhasePreflight, err), nil
	}
	for _, rev := range cat {
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "revision", Scope: rev.Version, Result: "planned",
			Detail: fmt.Sprintf("%s checksum=%s", rev.Name, rev.Checksum),
		})
	}
	return res, nil
}

func ForwardStatus(ctx context.Context, conn *pgx.Conn) (Result, error) {
	ins, err := InspectForward(ctx, conn)
	if err != nil {
		return forwardBlocked(PhasePreflight, err), nil
	}
	res := Result{
		Status: "ok", Code: CodeOK, Phase: PhasePreflight, DDLStarted: false,
		Class: ClassGovernedCurrent, Path: Path("forward"), TargetVersion: ins.CurrentVersion,
		Diagnostics: []Diag{{CheckID: "forward_package", Result: "pass", Detail: ins.PackageDigest}},
	}
	if ins.Current {
		res.Path = PathNoOp
		res.Diagnostics = append(res.Diagnostics, Diag{CheckID: "forward_manifest", Result: "pass", Detail: ins.ManifestDigest})
	} else {
		res.Code = CodeForwardPending
		res.Diagnostics = append(res.Diagnostics, Diag{CheckID: "forward_revision", Scope: "0001", Result: "foundation_current", Detail: "WP-01 forward revisions pending"})
	}
	return res, nil
}

func ForwardVerify(ctx context.Context, conn *pgx.Conn) (Result, error) {
	ins, err := InspectForward(ctx, conn)
	if err != nil {
		return forwardBlocked(PhaseVerify, err), nil
	}
	if !ins.Current {
		return Result{
			Status: "blocked", Code: CodeForwardPending, Phase: PhaseVerify, DDLStarted: false,
			Class: ClassGovernedCurrent, Path: Path("forward"), TargetVersion: "0001",
			Diagnostics: []Diag{{CheckID: "forward_revision", Result: "fail", Detail: "accepted WP-00 foundation is intact but WP-01 forward target 0004 is not applied"}},
		}, nil
	}
	return Result{
		Status: "verified", Code: CodeOK, Phase: PhaseVerify, DDLStarted: false,
		Class: ClassGovernedCurrent, Path: PathNoOp, TargetVersion: ForwardTargetVersion,
		Diagnostics: []Diag{
			{CheckID: "forward_package", Result: "pass", Detail: ins.PackageDigest},
			{CheckID: "forward_manifest", Result: "pass", Detail: ins.ManifestDigest},
		},
	}, nil
}

func forwardBlocked(phase Phase, err error) Result {
	code := CodeLedgerInconsistent
	if err != nil && (containsForwardPackaging(err) || containsForwardManifest(err)) {
		code = CodePackagingMismatch
	}
	return Result{
		Status: "blocked", Code: code, Phase: phase, DDLStarted: false,
		Class: ClassUnknownDrifted, Path: PathBlock,
		Diagnostics: []Diag{{CheckID: "forward", Result: "fail", Detail: "forward state failed closed verification"}},
	}
}

func containsForwardPackaging(err error) bool {
	return err != nil && (err == ErrForwardPackaging || containsErr(err, ErrForwardPackaging))
}

func containsForwardManifest(err error) bool {
	return err != nil && (err == ErrForwardManifest || containsErr(err, ErrForwardManifest))
}

func containsErr(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
