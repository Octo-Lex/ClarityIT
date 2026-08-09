package migration

// probe.go — live, read-only database probes that populate the Probe struct
// consumed by Classify. Every probe runs inside a READ ONLY transaction on a
// single dedicated connection (the same connection apply will later use for its
// advisory lock + execution). Apply must NOT recompute a weaker subset — it
// consumes the immutable PreflightResult returned here.
//
// All inspection is read-only. Catalog-query incompleteness is an error, never
// a partial profile. The runner verifies embedded identities (VerifyAll) BEFORE
// any database classification.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/clarityit/api/internal/migration/fingerprint"
	"github.com/jackc/pgx/v5"
)

// PreflightResult is the immutable outcome of a live preflight. Apply and plan
// consume this; neither recomputes a weaker subset.
type PreflightResult struct {
	Probe              Probe
	Class              Class
	Path               Path
	Code               ReasonCode
	Packaging          VerifyResult
	GovernedFingerprint string // computed live (empty if not governable)
	SourceFingerprint   string // computed live (empty if DB is empty)
}

// PackageVerifier is the identity-verification dependency. Production wiring
// uses the real embedded-byte verifier (defaultPackageVerifier -> VerifyAll).
// Tests inject a deterministic mismatch to exercise the checksum-mutation path
// live, without any filesystem or arbitrary SQL source.
type PackageVerifier interface {
	VerifyAll() (VerifyResult, error)
}

// defaultPackageVerifier is the production wiring: always the real embedded-byte
// verifier. Never overridden outside tests.
type defaultPackageVerifier struct{}

func (defaultPackageVerifier) VerifyAll() (VerifyResult, error) { return VerifyAll() }

// Preflight performs the full live preflight on a dedicated connection using
// the PRODUCTION identity verifier (real embedded bytes). It is the wiring used
// by the runner CLI. Every probe is read-only; zero mutation.
func Preflight(ctx context.Context, conn *pgx.Conn) (PreflightResult, error) {
	return PreflightWithVerifier(ctx, conn, defaultPackageVerifier{})
}

// PreflightWithVerifier is the orchestration entry point with an injected
// identity verifier. Tests use it to exercise the packaging-mismatch rejection
// path live (injecting a verifier that returns a deterministic mismatch) without
// any filesystem or arbitrary SQL source. Production callers use Preflight,
// which wires the real verifier.
func PreflightWithVerifier(ctx context.Context, conn *pgx.Conn, v PackageVerifier) (PreflightResult, error) {
	res := PreflightResult{}

	// 1. Verify embedded packaging identities BEFORE any DB work. A mismatch here
	//    stops preflight with PACKAGING_MISMATCH before any connection work.
	vres, err := v.VerifyAll()
	res.Packaging = vres
	if err != nil {
		res.Class, res.Path, res.Code = ClassUnknownDrifted, PathBlock, CodePackagingMismatch
		return res, fmt.Errorf("packaging verify: %w", err)
	}

	// 2. Begin a READ ONLY transaction. Read-only is enforced at the PG level,
	//    so even a bug in probe logic cannot mutate.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return res, fmt.Errorf("begin read-only tx: %w", err)
	}
	defer tx.Rollback(ctx) // safe: no-op after commit (we never commit here)

	// 3. Identity + structural probes.
	p, err := probeAll(ctx, tx)
	if err != nil {
		return res, err
	}

	// 4. Fingerprints (read-only; they re-query the catalog under the same tx).
	if !p.Fresh {
		// Source fingerprint only meaningful on a non-empty DB.
		if cap, err := fingerprint.ProfilerCapture(ctx, tx); err == nil {
			if fp, err := fingerprint.ProfilerFingerprint(cap); err == nil {
				p.SourceFingerprint = fp
			}
		}
		// Governed fingerprint: attempt always (governed projection works on any
		// DB with the governed schemas present; on an unknown DB it errors and
		// stays empty).
		if gfp, ok, err := tryGovernedFingerprint(ctx, tx); err == nil && ok {
			p.GovernedFingerprint = gfp
		}
	}

	res.Probe = p
	res.Class, res.Path, res.Code = Classify(p)
	res.GovernedFingerprint = p.GovernedFingerprint
	res.SourceFingerprint = p.SourceFingerprint
	return res, nil
}

// probeAll runs the structural probes (identity, extensions, roles, ledger,
// emptiness) and returns a populated Probe. All read-only.
func probeAll(ctx context.Context, tx pgx.Tx) (Probe, error) {
	var p Probe
	var err error

	p.DatabaseName, p.PGMajor, err = probeIdentity(ctx, tx)
	if err != nil {
		return p, err
	}
	p.ExtensionsPresent, err = probeExtensions(ctx, tx)
	if err != nil {
		return p, err
	}
	p.RolesPresent, err = probeRoles(ctx, tx)
	if err != nil {
		return p, err
	}
	p.PlatformSchemaPresent, err = probePlatformSchema(ctx, tx)
	if err != nil {
		return p, err
	}
	p.Revision0001Present, p.Recorded0001Checksum, err = probeRevision(ctx, tx)
	if err != nil {
		return p, err
	}
	p.Fresh, err = probeFresh(ctx, tx)
	if err != nil {
		return p, err
	}
	return p, nil
}

// probeIdentity returns (database_name, pg_major).
func probeIdentity(ctx context.Context, tx pgx.Tx) (string, int, error) {
	var db string
	var ver string
	if err := tx.QueryRow(ctx, `SELECT current_database(), current_setting('server_version_num')`).Scan(&db, &ver); err != nil {
		return "", 0, fmt.Errorf("probe identity: %w", err)
	}
	var major int
	if _, err := fmt.Sscanf(ver, "%d", &major); err != nil {
		return "", 0, fmt.Errorf("parse pg version %q: %w", ver, err)
	}
	// server_version_num for PG16 is "16000x"; major = version/10000.
	major = major / 10000
	return db, major, nil
}

// probeExtensions returns the presence map for required extensions.
func probeExtensions(ctx context.Context, tx pgx.Tx) (map[string]bool, error) {
	present := map[string]bool{}
	rows, err := tx.Query(ctx, `SELECT extname FROM pg_extension`)
	if err != nil {
		return nil, fmt.Errorf("probe extensions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("probe extensions scan: %w", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("probe extensions rows.Err: %w", err)
	}
	return present, nil
}

// probeRoles returns the presence map for the five target roles.
func probeRoles(ctx context.Context, tx pgx.Tx) (map[string]bool, error) {
	present := map[string]bool{}
	roles := make([]string, 0, len(fingerprint.TargetRoleNames))
	for r := range fingerprint.TargetRoleNames {
		roles = append(roles, r)
	}
	rows, err := tx.Query(ctx, `SELECT rolname FROM pg_roles WHERE rolname = ANY($1)`, roles)
	if err != nil {
		return nil, fmt.Errorf("probe roles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("probe roles scan: %w", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("probe roles rows.Err: %w", err)
	}
	return present, nil
}

// probePlatformSchema reports whether the platform control schema exists.
func probePlatformSchema(ctx context.Context, tx pgx.Tx) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'platform')`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("probe platform schema: %w", err)
	}
	return exists, nil
}

// probeRevision reports whether revision 0001 exists and its recorded checksum.
func probeRevision(ctx context.Context, tx pgx.Tx) (bool, string, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='platform' AND table_name='schema_revisions')`).Scan(&exists)
	if err != nil {
		return false, "", fmt.Errorf("probe revision table: %w", err)
	}
	if !exists {
		return false, "", nil
	}
	var present bool
	var checksum *string
	err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM platform.schema_revisions WHERE version='0001'), (SELECT checksum FROM platform.schema_revisions WHERE version='0001' LIMIT 1)`).Scan(&present, &checksum)
	if err != nil {
		return false, "", fmt.Errorf("probe revision 0001: %w", err)
	}
	if checksum == nil {
		return present, "", nil
	}
	return present, *checksum, nil
}

// probeFresh returns true only for a closed-world empty database: no non-system,
// non-approved-extension user relations, functions, schemas, or conflicting
// roles. This prevents classifying a database with unrelated objects as empty.
func probeFresh(ctx context.Context, tx pgx.Tx) (bool, error) {
	// 1. No user schemas other than the PostgreSQL default 'public' (which the
	//    G3 artifacts CREATE objects IN but do not CREATE the schema itself).
	var userSchemaCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_namespace n WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast','public') AND n.nspname NOT LIKE 'pg_temp_%' AND n.nspname NOT LIKE 'pg_toast_temp_%'`).Scan(&userSchemaCount); err != nil {
		return false, fmt.Errorf("probe fresh schemas: %w", err)
	}
	if userSchemaCount != 0 {
		return false, nil
	}
	// 2. No user relations.
	var userRelCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND n.nspname NOT LIKE 'pg_temp_%' AND c.relkind IN ('r','v','m','S','f','p')`).Scan(&userRelCount); err != nil {
		return false, fmt.Errorf("probe fresh relations: %w", err)
	}
	if userRelCount != 0 {
		return false, nil
	}
	// 3. No user functions.
	var userFuncCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND p.prokind IN ('f','p','w')`).Scan(&userFuncCount); err != nil {
		return false, fmt.Errorf("probe fresh functions: %w", err)
	}
	if userFuncCount != 0 {
		return false, nil
	}
	// 4. No non-system roles other than the bootstrap (and no target roles yet).
	var userRoleCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_roles WHERE rolname !~ '^pg_' AND rolname NOT IN ('postgres','clarityit')`).Scan(&userRoleCount); err != nil {
		return false, fmt.Errorf("probe fresh roles: %w", err)
	}
	if userRoleCount != 0 {
		return false, nil
	}
	return true, nil
}

// tryGovernedFingerprint computes the governed fingerprint if the DB has the
// governed schemas/objects; returns (fp, true, nil) on success, ("", false, nil)
// if the DB is not governable (unknown/empty).
func tryGovernedFingerprint(ctx context.Context, tx pgx.Tx) (string, bool, error) {
	signed, err := loadSignedG2()
	if err != nil {
		return "", false, err
	}
	control, err := loadControl()
	if err != nil {
		return "", false, err
	}
	cap, err := fingerprint.GovernedCapture(ctx, tx, signed, control)
	if err != nil {
		// Not governable (missing schemas/roles/etc).
		return "", false, nil
	}
	fp, err := fingerprint.GovernedFingerprint(cap)
	if err != nil {
		return "", false, err
	}
	return fp, true, nil
}

func loadSignedG2() (*fingerprint.SignedG2Manifest, error) {
	raw, err := assets.Bytes(assets.AssetG2Manifest)
	if err != nil {
		return nil, err
	}
	var s fingerprint.SignedG2Manifest
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("unmarshal G2 manifest: %w", err)
	}
	return &s, nil
}

func loadControl() (*fingerprint.ControlManifestFunctions, error) {
	raw, err := assets.Bytes(assets.AssetControlManifest)
	if err != nil {
		return nil, err
	}
	var c fingerprint.ControlManifestFunctions
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("unmarshal control manifest: %w", err)
	}
	return &c, nil
}
