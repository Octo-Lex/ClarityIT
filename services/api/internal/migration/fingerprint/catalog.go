// Package fingerprint ports the two JSON-canonicalizing fingerprint algorithms
// from the Python G3 tooling into Go, so the runner can compute them directly
// (AC-00-21: read-only verify detects fingerprint mismatch).
//
// The package is deliberately split into three layers, per the G4 contract:
//
//   - catalog: typed extraction of the live PostgreSQL catalog (relations,
//     columns, constraints, indexes, triggers, sequences, functions, roles,
//     memberships, grants, ownership, default privileges, extension owners).
//     Catalog queries return typed data and treat incompleteness as an error.
//   - The canonicalize package (separate) produces deterministic bytes.
//   - Fingerprinting hashes only those canonical bytes.
//
// Two algorithms:
//
//   - Governed (governed_fingerprint.py): SHA-256(domain || canonical(projection))
//     over the governed projection. Target: 9881c93e… . Used for post-apply verify.
//   - Source profiler (capture_schema.py fingerprint_of): SHA-256(canonical(stable))
//     over the full capture minus FINGERPRINT_EXCLUDE. Targets: P1 89b7792d…,
//     P3 cedf689d… . Used for source-profile allowlist selection.
//
// The composite installation digest is binary-framed and lives in the migration
// package (catalog.go), not here.
package fingerprint

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GovernedSchemas are the schemas covered by the governed projection.
var GovernedSchemas = []string{"public", "platform"}

// TargetRoleNames is the fixed five-role target posture.
var TargetRoleNames = map[string]struct{}{
	"clarityit":          {},
	"clarityit_admin":    {},
	"clarityit_app":      {},
	"clarityit_migrator": {},
	"clarityit_owner":    {},
}

// RequiredExtensions are the extensions the governed posture requires.
var RequiredExtensions = []string{"pgcrypto", "citext", "pg_trgm"}

// ErrCatalogIncomplete is returned when a catalog query fails to scan all
// expected columns or a rows iteration ends with a non-nil Err. Incomplete
// catalog data must never produce a partial fingerprint.
var ErrCatalogIncomplete = errors.New("catalog extraction incomplete")

// scanErr wraps a scan/iteration error with the query context for diagnostics.
func scanErr(ctx string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", ErrCatalogIncomplete, ctx, err)
}

// roleRow is a governed role record: name + the seven boolean flags.
type roleRow struct {
	Name           string `json:"name"`
	RolSuper       bool   `json:"rolsuper"`
	RolInherit     bool   `json:"rolinherit"`
	RolCreateRole  bool   `json:"rolcreaterole"`
	RolCreateDB    bool   `json:"rolcreatedb"`
	RolCanLogin    bool   `json:"rolcanlogin"`
	RolReplication bool   `json:"rolreplication"`
	RolBypassRLS   bool   `json:"rolbypassrls"`
}

// membershipRow is a governed role-membership record.
type membershipRow struct {
	Member        string `json:"member"`
	RoleOf        string `json:"role_of"`
	AdminOption   bool   `json:"admin_option"`
	InheritOption bool   `json:"inherit_option"`
	SetOption     bool   `json:"set_option"`
}

// relationRow is a governed relation projected to the 4-field signed contract.
type relationRow struct {
	Schema       string `json:"schema"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Persistence  string `json:"persistence"`
}

// columnRow is a governed column record.
type columnRow struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	NotNull  bool   `json:"not_null"`
	Default  any    `json:"default"`
	Identity string `json:"identity"`
}

// constraintRow is a governed constraint record.
type constraintRow struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Def  string `json:"definition"`
}

// indexRow is a governed index record.
type indexRow struct {
	Name      string `json:"name"`
	Def       string `json:"definition"`
	Unique    bool   `json:"unique"`
	Primary   bool   `json:"primary"`
}

// triggerRow is a governed trigger record.
type triggerRow struct {
	Name string `json:"name"`
	Def  string `json:"definition"`
}

// sequenceRow is a governed sequence record.
type sequenceRow struct {
	Schema    string `json:"schema"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Start     int64  `json:"start"`
	Increment int64  `json:"increment"`
	Max       int64  `json:"max"`
	Min       int64  `json:"min"`
	Cache     int64  `json:"cache"`
	Cycle     bool   `json:"cycle"`
}

// functionRow is a governed application function record (body is the full
// pg_get_functiondef output).
type functionRow struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Args   string `json:"args"`
	Body   string `json:"body"`
}

// queryRoles fetches the target role rows, ordered by rolname.
func queryRoles(ctx context.Context, q pgxQuerier) ([]roleRow, error) {
	roles := make([]string, 0, len(TargetRoleNames))
	for r := range TargetRoleNames {
		roles = append(roles, r)
	}
	const sql = `SELECT rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin, rolreplication, rolbypassrls
		FROM pg_roles WHERE rolname = ANY($1) ORDER BY rolname`
	rows, err := q.Query(ctx, sql, roles)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()
	var out []roleRow
	for rows.Next() {
		var r roleRow
		if err := rows.Scan(&r.Name, &r.RolSuper, &r.RolInherit, &r.RolCreateRole, &r.RolCreateDB, &r.RolCanLogin, &r.RolReplication, &r.RolBypassRLS); err != nil {
			return nil, scanErr("roles scan", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("roles rows.Err", err)
	}
	if len(out) != len(TargetRoleNames) {
		return nil, fmt.Errorf("%w: expected %d target roles, found %d", ErrCatalogIncomplete, len(TargetRoleNames), len(out))
	}
	return out, nil
}

// queryMemberships fetches target role memberships, ordered by member, role_of.
func queryMemberships(ctx context.Context, q pgxQuerier) ([]membershipRow, error) {
	roles := make([]string, 0, len(TargetRoleNames))
	for r := range TargetRoleNames {
		roles = append(roles, r)
	}
	const sql = `SELECT member.rolname, granted.rolname, am.admin_option, am.inherit_option, am.set_option
		FROM pg_auth_members am
		JOIN pg_roles member ON member.oid = am.member
		JOIN pg_roles granted ON granted.oid = am.roleid
		WHERE member.rolname = ANY($1) AND granted.rolname = ANY($2)
		ORDER BY 1,2`
	rows, err := q.Query(ctx, sql, roles, roles)
	if err != nil {
		return nil, fmt.Errorf("query memberships: %w", err)
	}
	defer rows.Close()
	var out []membershipRow
	for rows.Next() {
		var m membershipRow
		if err := rows.Scan(&m.Member, &m.RoleOf, &m.AdminOption, &m.InheritOption, &m.SetOption); err != nil {
			return nil, scanErr("memberships scan", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("memberships rows.Err", err)
	}
	return out, nil
}

// pgxQuerier is the minimal pgx surface the catalog layer needs. A *pgx.Tx or
// *pgx.Conn satisfies it. Using a narrow interface keeps the catalog layer
// testable and prevents it from reaching for connection-management APIs.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
