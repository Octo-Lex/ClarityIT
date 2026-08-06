package fingerprint

import (
	"context"
	"fmt"
)

// queryRelations fetches governed relations, projected to the 4-field signed
// contract (schema, name, kind, persistence). Mirrors capture_schema.relations
// restricted to governed schemas.
func queryRelations(ctx context.Context, q pgxQuerier) ([]relationRow, error) {
	const sql = `
		SELECT n.nspname, c.relname, c.relkind::text, c.relpersistence::text
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		  AND c.relkind IN ('r','v','m','S','f','p')
		ORDER BY n.nspname, c.relkind, c.relname`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query relations: %w", err)
	}
	defer rows.Close()
	var out []relationRow
	for rows.Next() {
		var schema, name, kind, persist string
		if err := rows.Scan(&schema, &name, &kind, &persist); err != nil {
			return nil, scanErr("relations scan", err)
		}
		out = append(out, relationRow{Schema: schema, Name: name, Kind: kind, Persistence: persist})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("relations rows.Err", err)
	}
	return out, nil
}

// columnMapKey returns the "schema.table" key used by the columns/constraints/
// indexes/triggers maps in the governed projection.
func columnMapKey(schema, table string) string {
	return schema + "." + table
}

// queryColumns fetches governed columns keyed by "schema.table", ordered by attnum.
// The Default column is nullable; NULL becomes nil in the projection (matching
// Python's None, which canonicalizes to null).
func queryColumns(ctx context.Context, q pgxQuerier) (map[string][]columnRow, error) {
	const sql = `
		SELECT n.nspname, c.relname, a.attnum, a.attname,
		       pg_catalog.format_type(a.atttypid, a.atttypmod),
		       a.attnotnull, pg_get_expr(d.adbin, d.adrelid), a.attidentity::text
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = ANY($1) AND a.attnum > 0 AND NOT a.attisdropped
		  AND c.relkind IN ('r','v','m','f','p')
		ORDER BY n.nspname, c.relname, a.attnum`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()
	out := map[string][]columnRow{}
	for rows.Next() {
		var schema, table, name, typ, identity string
		var attnum int
		var notNull bool
		var def *string
		if err := rows.Scan(&schema, &table, &attnum, &name, &typ, &notNull, &def, &identity); err != nil {
			return nil, scanErr("columns scan", err)
		}
		var defVal any
		if def != nil {
			defVal = *def
		}
		key := columnMapKey(schema, table)
		out[key] = append(out[key], columnRow{Name: name, Type: typ, NotNull: notNull, Default: defVal, Identity: identity})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("columns rows.Err", err)
	}
	return out, nil
}

// queryConstraints fetches governed constraints keyed by "schema.table".
func queryConstraints(ctx context.Context, q pgxQuerier) (map[string][]constraintRow, error) {
	const sql = `
		SELECT n.nspname, c.relname, con.conname, con.contype::text,
		       pg_get_constraintdef(con.oid, true)
		FROM pg_constraint con
		JOIN pg_class c ON c.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		ORDER BY n.nspname, c.relname, con.contype, con.conname`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query constraints: %w", err)
	}
	defer rows.Close()
	out := map[string][]constraintRow{}
	for rows.Next() {
		var schema, table, name, contype, def string
		if err := rows.Scan(&schema, &table, &name, &contype, &def); err != nil {
			return nil, scanErr("constraints scan", err)
		}
		key := columnMapKey(schema, table)
		out[key] = append(out[key], constraintRow{Name: name, Type: contype, Def: def})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("constraints rows.Err", err)
	}
	return out, nil
}

// queryIndexes fetches governed indexes keyed by "schema.table".
func queryIndexes(ctx context.Context, q pgxQuerier) (map[string][]indexRow, error) {
	const sql = `
		SELECT n.nspname, c.relname, i.relname,
		       pg_get_indexdef(ix.indexrelid, 0, true),
		       ix.indisunique, ix.indisprimary
		FROM pg_index ix
		JOIN pg_class c ON c.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1)
		ORDER BY n.nspname, c.relname, i.relname`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query indexes: %w", err)
	}
	defer rows.Close()
	out := map[string][]indexRow{}
	for rows.Next() {
		var schema, table, name, def string
		var unique, primary bool
		if err := rows.Scan(&schema, &table, &name, &def, &unique, &primary); err != nil {
			return nil, scanErr("indexes scan", err)
		}
		key := columnMapKey(schema, table)
		out[key] = append(out[key], indexRow{Name: name, Def: def, Unique: unique, Primary: primary})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("indexes rows.Err", err)
	}
	return out, nil
}

// queryTriggers fetches governed non-internal triggers keyed by "schema.table".
func queryTriggers(ctx context.Context, q pgxQuerier) (map[string][]triggerRow, error) {
	const sql = `
		SELECT n.nspname, c.relname, t.tgname, pg_get_triggerdef(t.oid, true)
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ANY($1) AND NOT t.tgisinternal
		ORDER BY n.nspname, c.relname, t.tgname`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query triggers: %w", err)
	}
	defer rows.Close()
	out := map[string][]triggerRow{}
	for rows.Next() {
		var schema, table, name, def string
		if err := rows.Scan(&schema, &table, &name, &def); err != nil {
			return nil, scanErr("triggers scan", err)
		}
		key := columnMapKey(schema, table)
		out[key] = append(out[key], triggerRow{Name: name, Def: def})
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("triggers rows.Err", err)
	}
	return out, nil
}

// querySequences fetches governed sequences.
func querySequences(ctx context.Context, q pgxQuerier) ([]sequenceRow, error) {
	const sql = `
		SELECT n.nspname, c.relname,
		       s.seqtypid::regtype, s.seqstart, s.seqincrement,
		       s.seqmax, s.seqmin, s.seqcache, s.seqcycle
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_sequence s ON s.seqrelid = c.oid
		WHERE n.nspname = ANY($1) AND c.relkind = 'S'
		ORDER BY n.nspname, c.relname`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query sequences: %w", err)
	}
	defer rows.Close()
	var out []sequenceRow
	for rows.Next() {
		var s sequenceRow
		if err := rows.Scan(&s.Schema, &s.Name, &s.Type, &s.Start, &s.Increment, &s.Max, &s.Min, &s.Cache, &s.Cycle); err != nil {
			return nil, scanErr("sequences scan", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("sequences rows.Err", err)
	}
	return out, nil
}

// queryAppFunctions fetches governed application functions (signed set + platform
// functions), filtered to the app_signatures set. Returns full pg_get_functiondef.
func queryAppFunctions(ctx context.Context, q pgxQuerier, appSignatures map[funcKey]struct{}) ([]functionRow, error) {
	const sql = `
		SELECT n.nspname, p.proname,
		       pg_get_function_identity_arguments(p.oid),
		       pg_get_functiondef(p.oid)
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = ANY($1) AND p.prokind IN ('f','p','w')
		ORDER BY n.nspname, p.proname,
		         pg_get_function_identity_arguments(p.oid)`
	rows, err := q.Query(ctx, sql, GovernedSchemas)
	if err != nil {
		return nil, fmt.Errorf("query functions: %w", err)
	}
	defer rows.Close()
	var out []functionRow
	for rows.Next() {
		var schema, name, args, body string
		if err := rows.Scan(&schema, &name, &args, &body); err != nil {
			return nil, scanErr("functions scan", err)
		}
		if _, ok := appSignatures[funcKey{schema, name, args}]; ok {
			out = append(out, functionRow{Schema: schema, Name: name, Args: args, Body: body})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, scanErr("functions rows.Err", err)
	}
	return out, nil
}

// funcKey identifies a function by (schema, name, identity-args).
type funcKey struct {
	Schema string
	Name   string
	Args   string
}
