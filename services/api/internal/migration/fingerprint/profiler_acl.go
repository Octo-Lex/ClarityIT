package fingerprint

// profiler_acl.go — the roles_and_grants and migration_state queries for the
// source-profiler fingerprint. These capture the full ACL posture (roles,
// memberships, grants across relation/function/schema/database/type objects,
// default privileges) and the migration-ledger state.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// queryRolesAndGrants builds the roles_and_grants block: roles, memberships,
// grants_sha256 (a precomputed digest over the canonical ACL grants string),
// and default_privileges.
func queryRolesAndGrants(ctx context.Context, q pgxQuerier) (map[string]any, error) {
	// roles: all non-pg_ roles with 8 boolean flags.
	rows, err := q.Query(ctx, `SELECT r.rolname, r.rolsuper, r.rolinherit, r.rolcreaterole,
		r.rolcreatedb, r.rolcanlogin, r.rolreplication, r.rolbypassrls
		FROM pg_roles r
		WHERE r.rolname !~ '^pg_'
		ORDER BY r.rolname`)
	if err != nil {
		return nil, fmt.Errorf("query profiler roles: %w", err)
	}
	var roles []any = []any{}
	for rows.Next() {
		var name string
		var sup, inh, cr, cdb, cl, rep, byp bool
		if err := rows.Scan(&name, &sup, &inh, &cr, &cdb, &cl, &rep, &byp); err != nil {
			rows.Close()
			return nil, scanErr("profiler roles scan", err)
		}
		roles = append(roles, map[string]any{
			"name": name, "superuser": sup, "inherit": inh, "createrole": cr,
			"createdb": cdb, "canlogin": cl, "replication": rep, "bypassrls": byp,
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler roles rows.Err", err)
	}

	// memberships: {member, role_of} for non-pg_ roles.
	rows, err = q.Query(ctx, `SELECT r.rolname AS member, r2.rolname AS role_of
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.member
		JOIN pg_roles r2 ON r2.oid = m.roleid
		WHERE r.rolname !~ '^pg_' AND r2.rolname !~ '^pg_'
		ORDER BY member, role_of`)
	if err != nil {
		return nil, fmt.Errorf("query profiler memberships: %w", err)
	}
	var memberships []any = []any{}
	for rows.Next() {
		var member, roleOf string
		if err := rows.Scan(&member, &roleOf); err != nil {
			rows.Close()
			return nil, scanErr("profiler memberships scan", err)
		}
		memberships = append(memberships, map[string]any{"member": member, "role_of": roleOf})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler memberships rows.Err", err)
	}

	// grants: the canonical ACL material across all object kinds. Python builds
	// a sorted string list and hashes it. The Go port builds the same list and
	// canonicalizes it.
	grants, err := queryAllGrants(ctx, q)
	if err != nil {
		return nil, err
	}
	grantsSHA, err := grantsDigest(grants)
	if err != nil {
		return nil, err
	}

	// default_privileges: {creator, schema, objtype, acl} where acl is the raw
	// array_to_string of defaclacl.
	rows, err = q.Query(ctx, `SELECT pg_get_userbyid(d.defaclrole), n.nspname, d.defaclobjtype::text,
		array_to_string(d.defaclacl, ',')
		FROM pg_default_acl d
		LEFT JOIN pg_namespace n ON n.oid = d.defaclnamespace
		ORDER BY 1,2,3`)
	if err != nil {
		return nil, fmt.Errorf("query profiler default privileges: %w", err)
	}
	var defPrivs []any = []any{}
	for rows.Next() {
		var creator, schema, objtype, acl string
		if err := rows.Scan(&creator, &schema, &objtype, &acl); err != nil {
			rows.Close()
			return nil, scanErr("profiler default privileges scan", err)
		}
		defPrivs = append(defPrivs, map[string]any{
			"creator": creator, "schema": schema, "objtype": objtype, "acl": acl,
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler default privileges rows.Err", err)
	}

	return map[string]any{
		"roles":              roles,
		"memberships":        memberships,
		"grants_sha256":      grantsSHA,
		"default_privileges": defPrivs,
	}, nil
}

// queryAllGrants builds the sorted canonical ACL material across relation,
// database, schema, function, and type objects (matching _all_grants_material).
// Each line: "kind|...fields..." joined into a list, then canonicalized+hashed.
func queryAllGrants(ctx context.Context, q pgxQuerier) ([]string, error) {
	var material []string

	// Relation grants.
	rows, err := q.Query(ctx, `SELECT n.nspname, c.relname, c.relkind::text,
		pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
		a.privilege_type, a.is_grantable
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace,
		aclexplode(c.relacl) AS a(grantor, grantee, privilege_type, is_grantable)
		WHERE n.nspname !~ '^(pg_|information_schema|pg_toast)'
		AND n.nspname NOT LIKE 'pg_temp_%'
		AND c.relkind IN ('r','S','v','m','f','p')
		ORDER BY 1,2,3,4,5,6,7`)
	if err != nil {
		return nil, fmt.Errorf("query relation grants (profiler): %w", err)
	}
	for rows.Next() {
		var nsp, rel, kind, grantor, grantee, priv string
		var grantable bool
		if err := rows.Scan(&nsp, &rel, &kind, &grantor, &grantee, &priv, &grantable); err != nil {
			rows.Close()
			return nil, scanErr("relation grants (profiler) scan", err)
		}
		material = append(material, strings.Join([]string{"rel", nsp, rel, kind, grantor, grantee, priv, pyBool(grantable)}, "|"))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("relation grants (profiler) rows.Err", err)
	}

	// Database grants.
	rows, err = q.Query(ctx, `SELECT datname, pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
		a.privilege_type, a.is_grantable
		FROM pg_database d,
		aclexplode(d.datacl) AS a(grantor, grantee, privilege_type, is_grantable)
		WHERE datname !~ '^template'
		ORDER BY 1,2,3,4`)
	if err != nil {
		return nil, fmt.Errorf("query database grants: %w", err)
	}
	for rows.Next() {
		var db, grantor, grantee, priv string
		var grantable bool
		if err := rows.Scan(&db, &grantor, &grantee, &priv, &grantable); err != nil {
			rows.Close()
			return nil, scanErr("database grants scan", err)
		}
		material = append(material, strings.Join([]string{"db", db, grantor, grantee, priv, pyBool(grantable)}, "|"))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("database grants rows.Err", err)
	}

	// Schema grants.
	rows, err = q.Query(ctx, `SELECT n.nspname, pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
		a.privilege_type, a.is_grantable
		FROM pg_namespace n,
		aclexplode(n.nspacl) AS a(grantor, grantee, privilege_type, is_grantable)
		WHERE n.nspname !~ '^(pg_|information_schema|pg_toast)'
		ORDER BY 1,2,3,4`)
	if err != nil {
		return nil, fmt.Errorf("query schema grants: %w", err)
	}
	for rows.Next() {
		var nsp, grantor, grantee, priv string
		var grantable bool
		if err := rows.Scan(&nsp, &grantor, &grantee, &priv, &grantable); err != nil {
			rows.Close()
			return nil, scanErr("schema grants scan", err)
		}
		material = append(material, strings.Join([]string{"schema", nsp, grantor, grantee, priv, pyBool(grantable)}, "|"))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("schema grants rows.Err", err)
	}

	// Function grants.
	rows, err = q.Query(ctx, `SELECT n.nspname, p.proname,
		pg_get_function_identity_arguments(p.oid),
		pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
		a.privilege_type, a.is_grantable
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace,
		aclexplode(p.proacl) AS a(grantor, grantee, privilege_type, is_grantable)
		WHERE n.nspname !~ '^(pg_|information_schema|pg_toast)'
		AND p.prokind IN ('f','p','w')
		ORDER BY 1,2,3,4,5,6,7`)
	if err != nil {
		return nil, fmt.Errorf("query function grants (profiler): %w", err)
	}
	for rows.Next() {
		var nsp, proname, args, grantor, grantee, priv string
		var grantable bool
		if err := rows.Scan(&nsp, &proname, &args, &grantor, &grantee, &priv, &grantable); err != nil {
			rows.Close()
			return nil, scanErr("function grants (profiler) scan", err)
		}
		material = append(material, strings.Join([]string{"func", nsp, proname, args, grantor, grantee, priv, pyBool(grantable)}, "|"))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("function grants (profiler) rows.Err", err)
	}

	// Type grants.
	rows, err = q.Query(ctx, `SELECT n.nspname, t.typname,
		pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
		a.privilege_type, a.is_grantable
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace,
		aclexplode(t.typacl) AS a(grantor, grantee, privilege_type, is_grantable)
		WHERE n.nspname !~ '^(pg_|information_schema)'
		ORDER BY 1,2,3,4,5`)
	if err != nil {
		return nil, fmt.Errorf("query type grants: %w", err)
	}
	for rows.Next() {
		var nsp, typ, grantor, grantee, priv string
		var grantable bool
		if err := rows.Scan(&nsp, &typ, &grantor, &grantee, &priv, &grantable); err != nil {
			rows.Close()
			return nil, scanErr("type grants scan", err)
		}
		material = append(material, strings.Join([]string{"type", nsp, typ, grantor, grantee, priv, pyBool(grantable)}, "|"))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("type grants rows.Err", err)
	}

	// NOTE: deliberately NOT sorted or deduped. capture_schema joins parts in
	// SQL ORDER BY order with "\n". Sorting/deduping would change the digest.
	return material, nil
}

// grantsDigest computes grants_sha256 EXACTLY as capture_schema does:
//
//	grant_material = "\n".join(parts)   # parts is the ordered list of grant lines
//	grants_sha256 = sha256(grant_material.encode("utf-8")).hexdigest()
//
// NOTE: this is NOT canonical JSON — it is a raw SHA-256 over the newline-joined
// grant lines string, in SQL ORDER BY order, NOT sorted/deduped. The lines use
// Python's capitalized True/False for the is_grantable field (via pyBool).
func grantsDigest(grants []string) (string, error) {
	material := strings.Join(grants, "\n")
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:]), nil
}

// queryMigrationState probes for a migration ledger table and returns its state.
// Probes (in order): schema_migrations, platform.schema_revisions,
// golang_migrations, goose_db_version. Returns {table, row_count,
// latest_column, latest_recorded_at} or {table, note} if none found.
func queryMigrationState(ctx context.Context, q pgxQuerier) (map[string]any, error) {
	candidates := []struct{ schema, table string}{
		{"public", "schema_migrations"},
		{"platform", "schema_revisions"},
		{"public", "golang_migrations"},
		{"public", "goose_db_version"},
	}
	for _, c := range candidates {
		var exists bool
		err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`,
			c.schema, c.table).Scan(&exists)
		if err != nil {
			return nil, scanErr("migration_state exists probe", err)
		}
		if !exists {
			continue
		}
		// Find the timestamp column (first column with 'timestamp' type).
		var tsCol *string
		row := q.QueryRow(ctx, `SELECT column_name FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2
			AND data_type LIKE '%timestamp%'
			ORDER BY column_name LIMIT 1`, c.schema, c.table)
		var col string
		if err := row.Scan(&col); err == nil {
			tsCol = &col
		}
		fq := fmt.Sprintf(`"%s"."%s"`, c.schema, c.table)
		if tsCol != nil {
			// row_count + max(timestamp::text)
			var count int64
			var maxTs *string
			q2 := fmt.Sprintf(`SELECT count(*), max("%s"::text) FROM %s`, *tsCol, fq)
			if err := q.QueryRow(ctx, q2).Scan(&count, &maxTs); err != nil {
				return nil, scanErr("migration_state count/max probe", err)
			}
			var maxTsVal any
			if maxTs != nil {
				maxTsVal = *maxTs
			}
			return map[string]any{
				"table":           c.schema + "." + c.table,
				"row_count":       count,
				"latest_column":   *tsCol,
				"latest_recorded_at": maxTsVal,
			}, nil
		}
		// No timestamp column: just row count.
		var count int64
		if err := q.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s`, fq)).Scan(&count); err != nil {
			return nil, scanErr("migration_state count probe", err)
		}
		return map[string]any{
			"table":         c.schema + "." + c.table,
			"row_count":     count,
			"latest_column": nil,
		}, nil
	}
	return map[string]any{"table": nil, "note": "no migration ledger table detected"}, nil
}
