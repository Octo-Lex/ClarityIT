package fingerprint

// profiler.go — the source-profiler fingerprint (capture_schema.py port). This
// is the BROADER algorithm used for source-profile allowlist selection: it
// captures the full live schema (relations, columns, constraints, indexes,
// triggers, sequences, functions, views, rls, roles_and_grants, migration_state)
// minus the FINGERPRINT_EXCLUDE top-level keys, and hashes the canonical form
// with NO domain prefix.
//
// Targets: P1 89b7792d…, P3 cedf689d… . The fresh G3 install has its own source
// fingerprint (distinct from governed 9881c93e…) because the source profiler
// includes roles/ownership/ledger state the governed projection deliberately
// excludes.
//
// Layering (per G4 contract): catalog extraction returns typed data;
// canonicalization (separate package) produces deterministic bytes; this layer
// hashes only those canonical bytes.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/clarityit/api/internal/migration/canonicalize"
)

// ProfilerVersion must match capture_schema.PROFILER_VERSION exactly — it is a
// value in the hashed JSON.
const ProfilerVersion = "3.2.0-p1p2"

// fingerprintExclude is the set of top-level manifest keys excluded from the
// source fingerprint (capture_schema.FINGERPRINT_EXCLUDE). These are volatile
// or self-referential and must not enter the hash.
var fingerprintExclude = map[string]bool{
	"captured_at_utc":   true,
	"row_counts":        true,
	"source_label":      true,
	"integrity_checks":  true,
	"schema_dump_sha256": true,
	"schema_dump_error": true,
	"fingerprint_sha256": true, // the digest cannot include itself
	"ownership":         true, // spec excludes ownership from fingerprint
	"pg_version_string": true, // build-specific label
}

// ProfilerCapture builds the full source manifest from the live catalog. This
// is the Go port of capture_schema.build_manifest. Catalog incompleteness is
// always an error.
//
// Unlike the governed projection, this captures ALL user schemas (not just
// public+platform) and includes roles, memberships, grants, default privileges,
// views, RLS, and migration_state. It excludes the fingerprintExclude keys.
func ProfilerCapture(ctx context.Context, q pgxQuerier) (map[string]any, error) {
	pgInfo, err := queryPGInfo(ctx, q)
	if err != nil {
		return nil, err
	}
	schemas, err := queryUserSchemas(ctx, q)
	if err != nil {
		return nil, err
	}
	relations, err := queryProfilerRelations(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	columns, err := queryProfilerColumns(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	constraints, err := queryProfilerConstraints(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	indexes, err := queryProfilerIndexes(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	triggers, err := queryProfilerTriggers(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	sequences, err := queryProfilerSequences(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	functions, err := queryProfilerFunctions(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	views, err := queryProfilerViews(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	rlsPolicies, err := queryRLSPolicies(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	rlsState, err := queryRLSState(ctx, q, schemas)
	if err != nil {
		return nil, err
	}
	rolesAndGrants, err := queryRolesAndGrants(ctx, q)
	if err != nil {
		return nil, err
	}
	migrationState, err := queryMigrationState(ctx, q)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"profiler_version":   ProfilerVersion,
		"postgres":           pgInfo,
		"schemas":            schemas,
		"relations":          relations,
		"columns":            columns,
		"constraints":        constraints,
		"indexes":            indexes,
		"sequences":          sequences,
		"functions":          functions,
		"triggers":           triggers,
		"views":              views,
		"rls_policies":       rlsPolicies,
		"rls_state":          rlsState,
		"roles_and_grants":   rolesAndGrants,
		"migration_state":    migrationState,
		// Excluded keys (captured_at_utc, source_label, ownership, row_counts,
		// integrity_checks, schema_dump_*, fingerprint_sha256, pg_version_string)
		// are deliberately NOT added.
	}, nil
}

// ProfilerFingerprint computes SHA-256(canonical(stable)).hex where stable is
// the capture minus the fingerprintExclude keys. NO domain prefix (unlike the
// governed fingerprint).
func ProfilerFingerprint(capture map[string]any) (string, error) {
	stable := make(map[string]any, len(capture))
	for k, v := range capture {
		if !fingerprintExclude[k] {
			stable[k] = v
		}
	}
	payload, err := canonicalize.Marshal(stable)
	if err != nil {
		return "", fmt.Errorf("profiler canonicalize: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
