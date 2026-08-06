package fingerprint

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// grantLine is one row of the closed-world ACL material. Python emits these as
// pipe-delimited strings: "kind|schema|name|[kind]|[grantee]|[priv]|[grantable]".
// The runner canonicalizes the sorted list of these strings (no object type
// struct — they are pre-formatted so the canonical form is byte-stable).
type grantLine struct {
	text string
}

// queryGrantMaterial returns the sorted, deduped closed-world ACL strings over
// the governed object inventory (governed relations + app functions + schemas),
// restricted to target roles plus PUBLIC. The grantor is excluded (environment-
// dependent). Mirrors governed_fingerprint._grant_material.
func queryGrantMaterial(ctx context.Context, q pgxQuerier, governedRelations [][2]string, appSignatures map[funcKey]struct{}) ([]string, error) {
	var material []string

	// Build the grantee IN-list: target roles + PUBLIC.
	grantees := make([]string, 0, len(TargetRoleNames)+1)
	grantees = append(grantees, "PUBLIC")
	for r := range TargetRoleNames {
		grantees = append(grantees, r)
	}
	sort.Strings(grantees)
	granteeList := quoteCSV(grantees)

	// --- Relation grants: governed inventory (product tables + sequence + platform tables) ---
	if len(governedRelations) > 0 {
		relFilter := orRelationFilter(governedRelations)
		sql := fmt.Sprintf(`SELECT n.nspname, c.relname, c.relkind::text, pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace, aclexplode(c.relacl) a
			WHERE (%s) AND c.relkind IN ('r','S','v','m','f','p')
			AND pg_get_userbyid(a.grantee) IN (%s)
			ORDER BY 1,2,3,4,5`, relFilter, granteeList)
		rows, err := q.Query(ctx, sql)
		if err != nil {
			return nil, fmt.Errorf("query relation grants: %w", err)
		}
		for rows.Next() {
			var nsp, rel, kind, grantee, priv string
			var grantable bool
			if err := rows.Scan(&nsp, &rel, &kind, &grantee, &priv, &grantable); err != nil {
				rows.Close()
				return nil, scanErr("relation grants scan", err)
			}
			material = append(material, fmt.Sprintf("rel|%s|%s|%s|%s|%s|%s", nsp, rel, kind, grantee, priv, pyBool(grantable)))
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, scanErr("relation grants rows.Err", err)
		}
	}

	// --- Function grants: signed app functions + platform functions ---
	if len(appSignatures) > 0 {
		funcFilter := orFunctionFilter(appSignatures)
		sql := fmt.Sprintf(`SELECT n.nspname, p.proname,
			pg_get_function_identity_arguments(p.oid),
			pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace,
			aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
			WHERE (%s) AND p.prokind IN ('f','p','w')
			AND pg_get_userbyid(a.grantee) IN (%s)
			ORDER BY 1,2,3,4,5`, funcFilter, granteeList)
		rows, err := q.Query(ctx, sql)
		if err != nil {
			return nil, fmt.Errorf("query function grants: %w", err)
		}
		for rows.Next() {
			var nsp, proname, args, grantee, priv string
			var grantable bool
			if err := rows.Scan(&nsp, &proname, &args, &grantee, &priv, &grantable); err != nil {
				rows.Close()
				return nil, scanErr("function grants scan", err)
			}
			material = append(material, fmt.Sprintf("func|%s|%s|%s|%s|%s|%s", nsp, proname, args, grantee, priv, pyBool(grantable)))
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, scanErr("function grants rows.Err", err)
		}
	}

	// --- Schema grants: public + platform ---
	schemaList := quoteCSV(GovernedSchemas)
	sql := fmt.Sprintf(`SELECT n.nspname, pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable
		FROM pg_namespace n, aclexplode(n.nspacl) a
		WHERE n.nspname IN (%s)
		AND pg_get_userbyid(a.grantee) IN (%s)
		ORDER BY 1,2,3`, schemaList, granteeList)
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query schema grants: %w", err)
	}
	for rows.Next() {
		var nsp, grantee, priv string
		var grantable bool
		if err := rows.Scan(&nsp, &grantee, &priv, &grantable); err != nil {
			rows.Close()
			return nil, scanErr("schema grants scan", err)
		}
		material = append(material, fmt.Sprintf("schema|%s|%s|%s|%s", nsp, grantee, priv, pyBool(grantable)))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("schema grants rows.Err", err)
	}

	// sorted(set(material)) — dedupe + sort, matching Python.
	return sortedDedup(material), nil
}

// ownershipRow is a governed projected-ownership record. The ownership projection
// includes database owner, schema owners, relation owners, and function owners.
type ownershipRow struct {
	Database    string `json:"database,omitempty"`
	Schemas     map[string]string `json:"schemas,omitempty"`
	Relations   map[string]string `json:"relations,omitempty"`
	Functions   map[string]string `json:"functions,omitempty"`
}

// queryProjectedOwnership returns the projected ownership posture: database
// owner + per-schema/relation/function owner identities, restricted to the
// governed inventory. Mirrors governed_fingerprint._projected_ownership.
func queryProjectedOwnership(ctx context.Context, q pgxQuerier, appSignatures map[funcKey]struct{}) (map[string]any, error) {
	// Database owner.
	var dbOwner string
	if err := q.QueryRow(ctx, `SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = current_database()`).Scan(&dbOwner); err != nil {
		return nil, scanErr("db owner", err)
	}

	schemaList := quoteCSV(GovernedSchemas)

	// Schema owners.
	schemaOwners := map[string]string{}
	rows, err := q.Query(ctx, fmt.Sprintf(`SELECT n.nspname, pg_get_userbyid(n.nspowner) FROM pg_namespace n WHERE n.nspname IN (%s) ORDER BY 1`, schemaList))
	if err != nil {
		return nil, fmt.Errorf("query schema owners: %w", err)
	}
	for rows.Next() {
		var nsp, owner string
		if err := rows.Scan(&nsp, &owner); err != nil {
			rows.Close()
			return nil, scanErr("schema owners scan", err)
		}
		schemaOwners[nsp] = owner
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("schema owners rows.Err", err)
	}

	// Relation owners (governed relkinds incl. indexes). Python keys these as
	// "schema.kind.name" (dot-joined), e.g. "platform.i.migration_runs_pkey".
	relationOwners := map[string]string{}
	rows, err = q.Query(ctx, fmt.Sprintf(`SELECT n.nspname, c.relname, c.relkind::text, pg_get_userbyid(c.relowner) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname IN (%s) AND c.relkind IN ('r','S','i','v','m','f','p') ORDER BY 1,2,3`, schemaList))
	if err != nil {
		return nil, fmt.Errorf("query relation owners: %w", err)
	}
	for rows.Next() {
		var nsp, rel, kind, owner string
		if err := rows.Scan(&nsp, &rel, &kind, &owner); err != nil {
			rows.Close()
			return nil, scanErr("relation owners scan", err)
		}
		relationOwners[nsp+"."+kind+"."+rel] = owner
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("relation owners rows.Err", err)
	}

	// Function owners (governed app functions only). Python keys these as
	// "schema.name(args)", e.g. "platform.protect_succeeded_revision()".
	functionOwners := map[string]string{}
	if len(appSignatures) > 0 {
		funcFilter := orFunctionFilter(appSignatures)
		rows, err = q.Query(ctx, fmt.Sprintf(`SELECT n.nspname, p.proname, pg_get_function_identity_arguments(p.oid), pg_get_userbyid(p.proowner) FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace WHERE (%s) AND p.prokind IN ('f','p','w') ORDER BY 1,2,3`, funcFilter))
		if err != nil {
			return nil, fmt.Errorf("query function owners: %w", err)
		}
		for rows.Next() {
			var nsp, proname, args, owner string
			if err := rows.Scan(&nsp, &proname, &args, &owner); err != nil {
				rows.Close()
				return nil, scanErr("function owners scan", err)
			}
			functionOwners[nsp+"."+proname+"("+args+")"] = owner
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, scanErr("function owners rows.Err", err)
		}
	}

	return map[string]any{
		"database_owner": dbOwner,
		"schemas":        schemaOwners,
		"relations":      relationOwners,
		"functions":      functionOwners,
	}, nil
}

// queryDefaultPrivilegesEffective returns sorted creator|schema|objtype|grantee|priv
// strings for default ACLs created by the target creators in governed schemas.
// Mirrors governed_fingerprint._default_privileges_effective (creator set =
// {"clarityit_owner"}).
func queryDefaultPrivilegesEffective(ctx context.Context, q pgxQuerier) ([]string, error) {
	creators := []string{"clarityit_owner"}
	creatorList := quoteCSV(creators)
	schemaList := quoteCSV(GovernedSchemas)
	sql := fmt.Sprintf(`SELECT pg_get_userbyid(d.defaclrole), n.nspname, d.defaclobjtype::text, pg_get_userbyid(a.grantee), a.privilege_type
		FROM pg_default_acl d
		LEFT JOIN pg_namespace n ON n.oid = d.defaclnamespace,
		aclexplode(d.defaclacl) a
		WHERE pg_get_userbyid(d.defaclrole) IN (%s)
		AND n.nspname IN (%s)
		ORDER BY 1,2,3,4,5`, creatorList, schemaList)
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query default privileges: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var creator, schema, objtype, grantee, priv string
		if err := rows.Scan(&creator, &schema, &objtype, &grantee, &priv); err != nil {
			return nil, scanErr("default privileges scan", err)
		}
		out = append(out, fmt.Sprintf("%s|%s|%s|%s|%s", creator, schema, objtype, grantee, priv))
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("default privileges rows.Err", err)
	}
	return out, nil
}

// queryExtensionOwnerInvariant returns extname -> owned-by-target-bool for each
// required extension. The invariant is violated (and must block) if any
// required extension is owned by a target role.
func queryExtensionOwnerInvariant(ctx context.Context, q pgxQuerier) (map[string]bool, error) {
	extList := quoteCSV(RequiredExtensions)
	sql := fmt.Sprintf(`SELECT e.extname, pg_get_userbyid(e.extowner) FROM pg_extension e WHERE e.extname IN (%s) ORDER BY 1`, extList)
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("query extension owners: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var extname, owner string
		if err := rows.Scan(&extname, &owner); err != nil {
			return nil, scanErr("extension owners scan", err)
		}
		_, isTarget := TargetRoleNames[owner]
		out[extname] = isTarget
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("extension owners rows.Err", err)
	}
	if len(out) != len(RequiredExtensions) {
		return nil, fmt.Errorf("%w: expected %d required extensions, found %d", ErrCatalogIncomplete, len(RequiredExtensions), len(out))
	}
	return out, nil
}

// quoteCSV returns 'a','b','c' for SQL IN-lists (single-quoted, comma-joined).
func quoteCSV(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return strings.Join(quoted, ",")
}

// orRelationFilter builds "(n.nspname = 'x' AND c.relname = 'y') OR ..." for
// each governed (schema, name) pair. Mirrors _grant_material's rel_filter.
func orRelationFilter(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = fmt.Sprintf("(n.nspname = '%s' AND c.relname = '%s')",
			strings.ReplaceAll(p[0], "'", "''"),
			strings.ReplaceAll(p[1], "'", "''"))
	}
	return strings.Join(parts, " OR ")
}

// orFunctionFilter builds the (schema, name, args) OR-filter for governed
// application functions. Mirrors _grant_material's func_filter.
func orFunctionFilter(sigs map[funcKey]struct{}) string {
	keys := make([]funcKey, 0, len(sigs))
	for k := range sigs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Schema != keys[j].Schema {
			return keys[i].Schema < keys[j].Schema
		}
		if keys[i].Name != keys[j].Name {
			return keys[i].Name < keys[j].Name
		}
		return keys[i].Args < keys[j].Args
	})
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("(n.nspname = '%s' AND p.proname = '%s' AND pg_get_function_identity_arguments(p.oid) = '%s')",
			strings.ReplaceAll(k.Schema, "'", "''"),
			strings.ReplaceAll(k.Name, "'", "''"),
			strings.ReplaceAll(k.Args, "'", "''"))
	}
	return strings.Join(parts, " OR ")
}

// pyBool renders a bool the way Python's f-string does: "True"/"False"
// (capitalized). The grant-material strings are pre-formatted to match the
// Python oracle byte-for-byte, so the bool field must use Python's form, not
// Go's %t ("true"/"false").
func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// sortedDedup returns the sorted unique set of strings. Python does
// sorted(set(material)); the canonical layer then hashes the resulting list.
func sortedDedup(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
