package migration

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const CodeForwardPending ReasonCode = "FORWARD_REVISIONS_PENDING"

// HasPlatformLedger reports whether the accepted WP-00 revision ledger exists.
// to_regclass is intentionally used so the routing probe does not require SELECT
// on the protected platform control tables.
func HasPlatformLedger(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var exists bool
	if err := conn.QueryRow(ctx, `SELECT to_regclass('platform.schema_revisions') IS NOT NULL`).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// HasForwardRevision is the CLI routing boundary. It deliberately does not read
// platform.schema_revisions because the ordinary application login has no
// platform-table authority and only clarityit_migrator may SET ROLE owner. The
// additive forward schemas are themselves the visible Stage-B footprint:
//
//   - exact 0001 has no kernel/compat objects and remains on the frozen Stage-A
//     plan/status/verify model used by the G4/G5 oracle;
//   - any persisted forward footprint routes into InspectForward, which then
//     escalates only through the migrator role and verifies exact ledger ancestry
//     plus the frozen target manifest. A manually partial footprint therefore
//     cannot escape fail-closed inspection.
//
// The routing probe uses PostgreSQL system catalogs instead of to_regclass so a
// least-privilege caller does not need USAGE on kernel/compat merely to detect
// that those schemas contain a forward object.
func HasForwardRevision(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var exists bool
	if err := conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			WHERE (n.nspname = 'kernel' AND c.relname = 'principal_refs')
			   OR (n.nspname = 'compat' AND c.relname = 'writer_ownership')
		)`).Scan(&exists); err != nil {
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
		Status:        "ok",
		Code:          CodeOK,
		Phase:         PhasePreflight,
		DDLStarted:    false,
		Class:         ClassGovernedCurrent,
		Path:          Path("forward"),
		TargetVersion: ForwardTargetVersion,
		Diagnostics:   []Diag{{CheckID: "forward_package", Result: "pass", Detail: ins.PackageDigest}},
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
			CheckID: "revision",
			Scope:   rev.Version,
			Result:  "planned",
			Detail:  fmt.Sprintf("%s checksum=%s", rev.Name, rev.Checksum),
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
		Status:        "ok",
		Code:          CodeOK,
		Phase:         PhasePreflight,
		DDLStarted:    false,
		Class:         ClassGovernedCurrent,
		Path:          Path("forward"),
		TargetVersion: ins.CurrentVersion,
		Diagnostics:   []Diag{{CheckID: "forward_package", Result: "pass", Detail: ins.PackageDigest}},
	}
	if ins.Current {
		res.Path = PathNoOp
		res.Diagnostics = append(res.Diagnostics, Diag{CheckID: "forward_manifest", Result: "pass", Detail: ins.ManifestDigest})
	} else {
		res.Code = CodeForwardPending
		res.Diagnostics = append(res.Diagnostics, Diag{
			CheckID: "forward_revision",
			Scope:   "0001",
			Result:  "foundation_current",
			Detail:  "WP-01 forward revisions pending",
		})
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
			Status:        "blocked",
			Code:          CodeForwardPending,
			Phase:         PhaseVerify,
			DDLStarted:    false,
			Class:         ClassGovernedCurrent,
			Path:          Path("forward"),
			TargetVersion: "0001",
			Diagnostics: []Diag{{
				CheckID: "forward_revision",
				Result:  "fail",
				Detail:  fmt.Sprintf("accepted WP-00 foundation is intact but WP-01 forward target %s is not applied", ForwardTargetVersion),
			}},
		}, nil
	}
	return Result{
		Status:        "verified",
		Code:          CodeOK,
		Phase:         PhaseVerify,
		DDLStarted:    false,
		Class:         ClassGovernedCurrent,
		Path:          PathNoOp,
		TargetVersion: ForwardTargetVersion,
		Diagnostics: []Diag{
			{CheckID: "forward_package", Result: "pass", Detail: ins.PackageDigest},
			{CheckID: "forward_manifest", Result: "pass", Detail: ins.ManifestDigest},
		},
	}, nil
}

func forwardBlocked(phase Phase, err error) Result {
	code := CodeLedgerInconsistent
	if errors.Is(err, ErrForwardPackaging) || errors.Is(err, ErrForwardManifest) {
		code = CodePackagingMismatch
	}
	return Result{
		Status:     "blocked",
		Code:       code,
		Phase:      phase,
		DDLStarted: false,
		Class:      ClassUnknownDrifted,
		Path:       PathBlock,
		Diagnostics: []Diag{{
			CheckID: "forward",
			Result:  "fail",
			Detail:  "forward state failed closed verification",
		}},
	}
}
