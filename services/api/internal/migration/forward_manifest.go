package migration

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/jackc/pgx/v5"
)

// forwardTargetManifestDigest is the deterministic G1 identity for the evolved
// kernel/compat target. It hashes semantic PostgreSQL catalog projections rather
// than OIDs or physical creation identifiers, so independent fresh/P2/P3 paths
// converge when their schema, constraints, indexes, triggers, functions,
// ownership, grants/default ACLs and frozen compatibility controls are equal.
func forwardTargetManifestDigest(ctx context.Context, q forwardVerifierQuery) (string, error) {
	h := sha256.New()
	h.Write([]byte("clarityit-wp01-forward-target-v1\x00"))

	// Schema ownership.
	rows, err := q.Query(ctx, `
		SELECT n.nspname, r.rolname
		FROM pg_namespace n JOIN pg_roles r ON r.oid=n.nspowner
		WHERE n.nspname IN ('kernel','compat') ORDER BY n.nspname`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, owner string
		if err := rows.Scan(&schema, &owner); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "schema", schema, owner)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Table identity/ownership/security posture. Indexes are captured separately.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, c.relname, c.relkind::text, r.rolname,
		       c.relrowsecurity, c.relforcerowsecurity
		FROM pg_class c
		JOIN pg_namespace n ON n.oid=c.relnamespace
		JOIN pg_roles r ON r.oid=c.relowner
		WHERE n.nspname IN ('kernel','compat') AND c.relkind IN ('r','p','S')
		ORDER BY n.nspname, c.relname`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, name, kind, owner string
		var rls, forceRLS bool
		if err := rows.Scan(&schema, &name, &kind, &owner, &rls, &forceRLS); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "relation", schema, name, kind, owner, fmt.Sprint(rls), fmt.Sprint(forceRLS))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Columns are canonicalized by name, not attnum, to avoid physical-order
	// sensitivity while retaining every semantic column property used by G1.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, c.relname, a.attname,
		       pg_catalog.format_type(a.atttypid,a.atttypmod),
		       a.attnotnull, COALESCE(pg_get_expr(d.adbin,d.adrelid),''),
		       COALESCE(coll.collname,''), a.attidentity::text, a.attgenerated::text
		FROM pg_attribute a
		JOIN pg_class c ON c.oid=a.attrelid
		JOIN pg_namespace n ON n.oid=c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
		LEFT JOIN pg_collation coll ON coll.oid=a.attcollation
		WHERE n.nspname IN ('kernel','compat')
		  AND c.relkind IN ('r','p') AND a.attnum>0 AND NOT a.attisdropped
		ORDER BY n.nspname, c.relname, a.attname`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, table, column, typ, def, coll, identity, generated string
		var notnull bool
		if err := rows.Scan(&schema, &table, &column, &typ, &notnull, &def, &coll, &identity, &generated); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "column", schema, table, column, typ, fmt.Sprint(notnull), def, coll, identity, generated)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Constraints including PK/UQ/FK/CHECK semantics.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, t.relname, c.conname, c.contype::text,
		       pg_get_constraintdef(c.oid,false)
		FROM pg_constraint c
		JOIN pg_class t ON t.oid=c.conrelid
		JOIN pg_namespace n ON n.oid=t.relnamespace
		WHERE n.nspname IN ('kernel','compat')
		ORDER BY n.nspname, t.relname, c.conname`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, table, name, typ, def string
		if err := rows.Scan(&schema, &table, &name, &typ, &def); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "constraint", schema, table, name, typ, def)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// All indexes, including constraint-backed indexes.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, t.relname, i.relname, pg_get_indexdef(i.oid)
		FROM pg_index x
		JOIN pg_class t ON t.oid=x.indrelid
		JOIN pg_class i ON i.oid=x.indexrelid
		JOIN pg_namespace n ON n.oid=t.relnamespace
		WHERE n.nspname IN ('kernel','compat')
		ORDER BY n.nspname, t.relname, i.relname`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, table, name, def string
		if err := rows.Scan(&schema, &table, &name, &def); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "index", schema, table, name, def)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// User-defined triggers.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, c.relname, t.tgname, pg_get_triggerdef(t.oid,false)
		FROM pg_trigger t
		JOIN pg_class c ON c.oid=t.tgrelid
		JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE n.nspname IN ('kernel','compat') AND NOT t.tgisinternal
		ORDER BY n.nspname, c.relname, t.tgname`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, table, name, def string
		if err := rows.Scan(&schema, &table, &name, &def); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "trigger", schema, table, name, def)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Function definitions and ownership.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, p.proname, pg_get_function_identity_arguments(p.oid),
		       r.rolname, pg_get_functiondef(p.oid)
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid=p.pronamespace
		JOIN pg_roles r ON r.oid=p.proowner
		WHERE n.nspname IN ('kernel','compat')
		ORDER BY n.nspname, p.proname, pg_get_function_identity_arguments(p.oid)`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, name, args, owner, def string
		if err := rows.Scan(&schema, &name, &args, &owner, &def); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "function", schema, name, args, owner, def)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Explicit/effective table ACLs, canonicalized by role/privilege.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, c.relname,
		       COALESCE(gor.rolname,'PUBLIC'), COALESCE(gee.rolname,'PUBLIC'),
		       a.privilege_type, a.is_grantable
		FROM pg_class c
		JOIN pg_namespace n ON n.oid=c.relnamespace
		CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl, acldefault(CASE WHEN c.relkind='S' THEN 'S'::"char" ELSE 'r'::"char" END,c.relowner))) a
		LEFT JOIN pg_roles gor ON gor.oid=a.grantor
		LEFT JOIN pg_roles gee ON gee.oid=a.grantee
		WHERE n.nspname IN ('kernel','compat') AND c.relkind IN ('r','p','S')
		ORDER BY n.nspname,c.relname,COALESCE(gee.rolname,'PUBLIC'),a.privilege_type,COALESCE(gor.rolname,'PUBLIC')`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, rel, grantor, grantee, privilege string
		var grantable bool
		if err := rows.Scan(&schema, &rel, &grantor, &grantee, &privilege, &grantable); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "relation_acl", schema, rel, grantor, grantee, privilege, fmt.Sprint(grantable))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Schema ACLs.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, COALESCE(gor.rolname,'PUBLIC'), COALESCE(gee.rolname,'PUBLIC'),
		       a.privilege_type, a.is_grantable
		FROM pg_namespace n
		CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl, acldefault('n',n.nspowner))) a
		LEFT JOIN pg_roles gor ON gor.oid=a.grantor
		LEFT JOIN pg_roles gee ON gee.oid=a.grantee
		WHERE n.nspname IN ('kernel','compat')
		ORDER BY n.nspname,COALESCE(gee.rolname,'PUBLIC'),a.privilege_type,COALESCE(gor.rolname,'PUBLIC')`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, grantor, grantee, privilege string
		var grantable bool
		if err := rows.Scan(&schema, &grantor, &grantee, &privilege, &grantable); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "schema_acl", schema, grantor, grantee, privilege, fmt.Sprint(grantable))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Function ACLs.
	rows, err = q.Query(ctx, `
		SELECT n.nspname, p.proname, pg_get_function_identity_arguments(p.oid),
		       COALESCE(gor.rolname,'PUBLIC'), COALESCE(gee.rolname,'PUBLIC'),
		       a.privilege_type, a.is_grantable
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid=p.pronamespace
		CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl, acldefault('f',p.proowner))) a
		LEFT JOIN pg_roles gor ON gor.oid=a.grantor
		LEFT JOIN pg_roles gee ON gee.oid=a.grantee
		WHERE n.nspname IN ('kernel','compat')
		ORDER BY n.nspname,p.proname,pg_get_function_identity_arguments(p.oid),COALESCE(gee.rolname,'PUBLIC'),a.privilege_type,COALESCE(gor.rolname,'PUBLIC')`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, fn, args, grantor, grantee, privilege string
		var grantable bool
		if err := rows.Scan(&schema, &fn, &args, &grantor, &grantee, &privilege, &grantable); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "function_acl", schema, fn, args, grantor, grantee, privilege, fmt.Sprint(grantable))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// Default ACLs owned by clarityit_owner for the forward schemas.
	rows, err = q.Query(ctx, `
		SELECT COALESCE(n.nspname,''), r.rolname, d.defaclobjtype::text,
		       COALESCE(gor.rolname,'PUBLIC'), COALESCE(gee.rolname,'PUBLIC'),
		       a.privilege_type, a.is_grantable
		FROM pg_default_acl d
		JOIN pg_roles r ON r.oid=d.defaclrole
		LEFT JOIN pg_namespace n ON n.oid=d.defaclnamespace
		CROSS JOIN LATERAL aclexplode(d.defaclacl) a
		LEFT JOIN pg_roles gor ON gor.oid=a.grantor
		LEFT JOIN pg_roles gee ON gee.oid=a.grantee
		WHERE r.rolname='clarityit_owner' AND (n.nspname IN ('kernel','compat') OR n.nspname IS NULL)
		ORDER BY COALESCE(n.nspname,''),d.defaclobjtype::text,COALESCE(gee.rolname,'PUBLIC'),a.privilege_type,COALESCE(gor.rolname,'PUBLIC')`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var schema, owner, objtype, grantor, grantee, privilege string
		var grantable bool
		if err := rows.Scan(&schema, &owner, &objtype, &grantor, &grantee, &privilege, &grantable); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "default_acl", schema, owner, objtype, grantor, grantee, privilege, fmt.Sprint(grantable))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	// G1 compatibility control data is part of the accepted target identity.
	rows, err = q.Query(ctx, `SELECT flag_name,scope_key,COALESCE(workspace_id::text,''),enabled,config::text,authority_ref FROM compat.feature_flags ORDER BY flag_name,scope_key`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var name, scope, workspace, config, authority string
		var enabled bool
		if err := rows.Scan(&name, &scope, &workspace, &enabled, &config, &authority); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "feature_flag", name, scope, workspace, fmt.Sprint(enabled), config, authority)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	rows, err = q.Query(ctx, `SELECT object_family,authoritative_writer,owner_version,effective_from::text,COALESCE(effective_to::text,''),authority_ref FROM compat.writer_ownership ORDER BY object_family,owner_version`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var family, writer, from, to, authority string
		var version int
		if err := rows.Scan(&family, &writer, &version, &from, &to, &authority); err != nil {
			rows.Close()
			return "", err
		}
		manifestRecord(h, "writer_ownership", family, writer, fmt.Sprint(version), from, to, authority)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	return hex.EncodeToString(h.Sum(nil)), nil
}

func manifestRecord(h hash.Hash, kind string, fields ...string) {
	writeManifestField(h, kind)
	for _, field := range fields {
		writeManifestField(h, field)
	}
}

func writeManifestField(h hash.Hash, value string) {
	var lenbuf [8]byte
	b := []byte(value)
	binary.BigEndian.PutUint64(lenbuf[:], uint64(len(b)))
	h.Write(lenbuf[:])
	h.Write(b)
}

// compile-time assertion that pgx remains part of the manifest query contract;
// retained here because this file intentionally hashes PostgreSQL semantic
// projections rather than parsing SQL text.
var _ pgx.Rows
