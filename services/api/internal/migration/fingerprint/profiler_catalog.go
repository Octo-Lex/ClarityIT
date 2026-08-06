package fingerprint

// profiler_catalog.go — catalog queries for the source-profiler fingerprint.
// Each query is a verbatim translation of capture_schema.py, scoped to all user
// schemas (not just governed). Catalog incompleteness is always an error.
// Per-table row counts (row_counts) are deliberately NOT queried: they are in
// FINGERPRINT_EXCLUDE and never enter the hash.

import (
	"context"
	"fmt"
)

// queryPGInfo returns the postgres info block: {settings, extensions}.
func queryPGInfo(ctx context.Context, q pgxQuerier) (map[string]any, error) {
	// settings: only standard_conforming_strings + TimeZone, ordered by name.
	settings := map[string]string{}
	rows, err := q.Query(ctx, `SELECT name, setting FROM pg_settings WHERE name IN ('standard_conforming_strings','TimeZone') ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query pg settings: %w", err)
	}
	for rows.Next() {
		var name, setting string
		if err := rows.Scan(&name, &setting); err != nil {
			rows.Close()
			return nil, scanErr("pg settings scan", err)
		}
		settings[name] = setting
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("pg settings rows.Err", err)
	}

	// extensions: [{name, version}] ordered by extname.
	rows, err = q.Query(ctx, `SELECT extname, extversion FROM pg_extension ORDER BY extname`)
	if err != nil {
		return nil, fmt.Errorf("query extensions: %w", err)
	}
	var exts []any = []any{}
	for rows.Next() {
		var name, ver string
		if err := rows.Scan(&name, &ver); err != nil {
			rows.Close()
			return nil, scanErr("extensions scan", err)
		}
		exts = append(exts, map[string]any{"name": name, "version": ver})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, scanErr("extensions rows.Err", err)
	}
	return map[string]any{"settings": settings, "extensions": exts}, nil
}

// queryUserSchemas returns user schema names (excluding system schemas).
func queryUserSchemas(ctx context.Context, q pgxQuerier) ([]any, error) {
	rows, err := q.Query(ctx, `SELECT n.nspname FROM pg_namespace n
		WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
		AND n.nspname NOT LIKE 'pg_temp_%'
		AND n.nspname NOT LIKE 'pg_toast_temp_%'
		ORDER BY n.nspname`)
	if err != nil {
		return nil, fmt.Errorf("query schemas: %w", err)
	}
	defer rows.Close()
	// Empty (non-nil) so zero rows -> [] not null.
	out := []any{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, scanErr("schemas scan", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("schemas rows.Err", err)
	}
	return out, nil
}

// queryProfilerRelations returns relations with the 5-field profiler shape
// (schema, name, kind, persistence, type) — note: profiler includes the format_type
// "type" field that the governed projection drops.
func queryProfilerRelations(ctx context.Context, q pgxQuerier, schemas []any) ([]any, error) {
	const sql = `SELECT n.nspname, c.relname, c.relkind::text, c.relpersistence::text,
		pg_catalog.format_type(c.reltype, NULL)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		AND c.relkind IN ('r','v','m','S','f','p')
		ORDER BY n.nspname, c.relkind, c.relname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler relations: %w", err)
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var schema, name, kind, persist, typ string
		if err := rows.Scan(&schema, &name, &kind, &persist, &typ); err != nil {
			return nil, scanErr("profiler relations scan", err)
		}
		out = append(out, map[string]any{
			"schema": schema, "name": name, "kind": kind, "persistence": persist, "type": typ,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler relations rows.Err", err)
	}
	return out, nil
}

// queryProfilerColumns returns columns keyed by "schema.table" (list of column
// rows), matching capture_schema.columns.
func queryProfilerColumns(ctx context.Context, q pgxQuerier, schemas []any) (map[string]any, error) {
	const sql = `SELECT n.nspname, c.relname, a.attnum, a.attname,
		pg_catalog.format_type(a.atttypid, a.atttypmod),
		a.attnotnull, pg_get_expr(d.adbin, d.adrelid), a.attidentity::text
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = ANY($1) AND a.attnum > 0 AND NOT a.attisdropped
		AND c.relkind IN ('r','v','m','f','p')
		ORDER BY n.nspname, c.relname, a.attnum`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler columns: %w", err)
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var schema, table, name, typ, identity string
		var attnum int
		var notNull bool
		var def *string
		if err := rows.Scan(&schema, &table, &attnum, &name, &typ, &notNull, &def, &identity); err != nil {
			return nil, scanErr("profiler columns scan", err)
		}
		key := columnMapKey(schema, table)
		var defVal any
		if def != nil {
			defVal = *def
		}
		var keyList []any
		if existing, ok := out[key]; ok {
			keyList = existing.([]any)
		}
		keyList = append(keyList, map[string]any{
			"name": name, "type": typ, "not_null": notNull, "default": defVal, "identity": identity,
		})
		out[key] = keyList
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler columns rows.Err", err)
	}
	return out, nil
}

// queryProfilerConstraints returns constraints keyed by "schema.table".
func queryProfilerConstraints(ctx context.Context, q pgxQuerier, schemas []any) (map[string]any, error) {
	const sql = `SELECT n.nspname, c.relname, con.conname, con.contype::text,
		pg_get_constraintdef(con.oid, true)
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		ORDER BY n.nspname, c.relname, con.contype, con.conname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler constraints: %w", err)
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var schema, table, name, contype, def string
		if err := rows.Scan(&schema, &table, &name, &contype, &def); err != nil {
			return nil, scanErr("profiler constraints scan", err)
		}
		key := columnMapKey(schema, table)
		var keyList []any
		if existing, ok := out[key]; ok {
			keyList = existing.([]any)
		}
		keyList = append(keyList, map[string]any{"name": name, "type": contype, "definition": def})
		out[key] = keyList
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler constraints rows.Err", err)
	}
	return out, nil
}

// queryProfilerIndexes returns indexes keyed by "schema.table".
func queryProfilerIndexes(ctx context.Context, q pgxQuerier, schemas []any) (map[string]any, error) {
	const sql = `SELECT n.nspname, c.relname, i.relname,
		pg_get_indexdef(ix.indexrelid, 0, true),
		ix.indisunique, ix.indisprimary
		FROM pg_index ix
		JOIN pg_class c ON c.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		ORDER BY n.nspname, c.relname, i.relname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler indexes: %w", err)
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var schema, table, name, def string
		var unique, primary bool
		if err := rows.Scan(&schema, &table, &name, &def, &unique, &primary); err != nil {
			return nil, scanErr("profiler indexes scan", err)
		}
		key := columnMapKey(schema, table)
		var keyList []any
		if existing, ok := out[key]; ok {
			keyList = existing.([]any)
		}
		keyList = append(keyList, map[string]any{"name": name, "definition": def, "unique": unique, "primary": primary})
		out[key] = keyList
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler indexes rows.Err", err)
	}
	return out, nil
}

// queryProfilerTriggers returns triggers keyed by "schema.table".
func queryProfilerTriggers(ctx context.Context, q pgxQuerier, schemas []any) (map[string]any, error) {
	const sql = `SELECT n.nspname, c.relname, t.tgname, pg_get_triggerdef(t.oid, true)
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND NOT t.tgisinternal
		ORDER BY n.nspname, c.relname, t.tgname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler triggers: %w", err)
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var schema, table, name, def string
		if err := rows.Scan(&schema, &table, &name, &def); err != nil {
			return nil, scanErr("profiler triggers scan", err)
		}
		key := columnMapKey(schema, table)
		var keyList []any
		if existing, ok := out[key]; ok {
			keyList = existing.([]any)
		}
		keyList = append(keyList, map[string]any{"name": name, "definition": def})
		out[key] = keyList
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler triggers rows.Err", err)
	}
	return out, nil
}

// queryProfilerSequences returns sequences (full list, not keyed).
func queryProfilerSequences(ctx context.Context, q pgxQuerier, schemas []any) ([]any, error) {
	const sql = `SELECT n.nspname, c.relname,
		s.seqtypid::regtype, s.seqstart, s.seqincrement,
		s.seqmax, s.seqmin, s.seqcache, s.seqcycle
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_sequence s ON s.seqrelid = c.oid
		WHERE n.nspname = ANY($1) AND c.relkind = 'S'
		ORDER BY n.nspname, c.relname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler sequences: %w", err)
	}
	defer rows.Close()
	// Initialize to empty (non-nil) so zero rows serialize to [] not null,
	// matching Python's empty list.
	out := []any{}
	for rows.Next() {
		var schema, name, typ string
		var start, incr, mx, mn, cache int64
		var cycle bool
		if err := rows.Scan(&schema, &name, &typ, &start, &incr, &mx, &mn, &cache, &cycle); err != nil {
			return nil, scanErr("profiler sequences scan", err)
		}
		out = append(out, map[string]any{
			"schema": schema, "name": name, "type": typ, "start": start, "increment": incr,
			"max": mx, "min": mn, "cache": cache, "cycle": cycle,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler sequences rows.Err", err)
	}
	return out, nil
}

// queryProfilerFunctions returns all functions in user schemas (full list).
func queryProfilerFunctions(ctx context.Context, q pgxQuerier, schemas []any) ([]any, error) {
	const sql = `SELECT n.nspname, p.proname,
		pg_get_function_identity_arguments(p.oid),
		pg_get_functiondef(p.oid)
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = ANY($1) AND p.prokind IN ('f','p','w')
		ORDER BY n.nspname, p.proname,
		pg_get_function_identity_arguments(p.oid)`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler functions: %w", err)
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var schema, name, args, body string
		if err := rows.Scan(&schema, &name, &args, &body); err != nil {
			return nil, scanErr("profiler functions scan", err)
		}
		out = append(out, map[string]any{"schema": schema, "name": name, "args": args, "body": body})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler functions rows.Err", err)
	}
	return out, nil
}

// queryProfilerViews returns views (relkind v,m).
func queryProfilerViews(ctx context.Context, q pgxQuerier, schemas []any) ([]any, error) {
	const sql = `SELECT n.nspname, c.relname, pg_get_viewdef(c.oid, true)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND c.relkind IN ('v','m')
		ORDER BY n.nspname, c.relname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query profiler views: %w", err)
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var schema, name, def string
		if err := rows.Scan(&schema, &name, &def); err != nil {
			return nil, scanErr("profiler views scan", err)
		}
		out = append(out, map[string]any{"schema": schema, "name": name, "definition": def})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("profiler views rows.Err", err)
	}
	return out, nil
}

// queryRLSPolicies returns RLS policies.
func queryRLSPolicies(ctx context.Context, q pgxQuerier, schemas []any) ([]any, error) {
	const sql = `SELECT n.nspname, c.relname, p.polname, p.polcmd, p.polpermissive,
		pg_get_expr(p.polqual, p.polrelid),
		pg_get_expr(p.polwithcheck, p.polrelid),
		(SELECT array_agg(rolname) FROM pg_roles r2, unnest(p.polroles) AS proid WHERE r2.oid = proid)
		FROM pg_policy p
		JOIN pg_class c ON c.oid = p.polrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		ORDER BY c.relname, p.polname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query rls policies: %w", err)
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var schema, table, name, cmd string
		var permissive bool
		var using, withCheck, roles *string
		if err := rows.Scan(&schema, &table, &name, &cmd, &permissive, &using, &withCheck, &roles); err != nil {
			return nil, scanErr("rls policies scan", err)
		}
		var usingVal, withCheckVal, rolesVal any
		if using != nil {
			usingVal = *using
		}
		if withCheck != nil {
			withCheckVal = *withCheck
		}
		if roles != nil {
			rolesVal = *roles
		}
		out = append(out, map[string]any{
			"schema": schema, "table": table, "name": name, "cmd": cmd,
			"permissive": permissive, "using": usingVal, "with_check": withCheckVal, "roles": rolesVal,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("rls policies rows.Err", err)
	}
	return out, nil
}

// queryRLSState returns RLS state for tables with RLS enabled.
func queryRLSState(ctx context.Context, q pgxQuerier, schemas []any) ([]any, error) {
	const sql = `SELECT n.nspname, c.relname, c.relrowsecurity, c.relforcerowsecurity
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND c.relkind IN ('r','v','m')
		AND (c.relrowsecurity OR c.relforcerowsecurity)
		ORDER BY n.nspname, c.relname`
	rows, err := q.Query(ctx, sql, schemas)
	if err != nil {
		return nil, fmt.Errorf("query rls state: %w", err)
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var schema, table string
		var enabled, forced bool
		if err := rows.Scan(&schema, &table, &enabled, &forced); err != nil {
			return nil, scanErr("rls state scan", err)
		}
		out = append(out, map[string]any{
			"schema": schema, "table": table, "rls_enabled": enabled, "rls_forced": forced,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("rls state rows.Err", err)
	}
	return out, nil
}
