#!/usr/bin/env python3
"""
ClarityIT WP-00 schema capture (P1/P2).

Produces a comprehensive, read-only, secret-excluding schema capture from a
PostgreSQL database, per the v1-to-v2 Compatibility & Migration Specification
§2.2 / §4.3 and the WP-00 G1 requirements.

Captures:
  - PostgreSQL version + extensions
  - Schema-only dump (pg_dump --schema-only, redacted of secrets)
  - Deterministic schema fingerprint (SHA-256 over canonical manifest)
  - Tables, columns, constraints, indexes, sequences, functions, triggers,
    views, materialized views, and RLS policies
  - Migration-history state (schema_migrations / migration ledger if present)
  - Roles, memberships, ownership, grants, default privileges
  - Required table counts and integrity checks (no business data, no secrets)

INVARIANTS:
  - READ-ONLY. Enforces SET TRANSACTION READ ONLY.
  - SECRET-EXCLUDING. Never reads row data (counts only), never reads
    pg_authid.rolpassword, pg_shadow, or credential columns.
  - DETERMINISTIC. Same source + same profiler version => same fingerprint.
    Excludes volatile fields (timestamps, row counts, stats, OIDs).

Usage:
    python capture_schema.py capture \
        --host H --port 5432 --db DB --user clarityit_ro_profile \
        --label P1-production \
        --out migrations/profiles/p1-production/

    python capture_schema.py fingerprint --manifest manifest.json
    python capture_schema.py compare --a p1/manifest.json --b p2/manifest.json
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import subprocess
import sys
from datetime import datetime, timezone

PROFILER_VERSION = "3.1.0-p1p2"


# ─── Read-only connection ────────────────────────────────────────────────────

def connect(host, port, db, user, password):
    try:
        import psycopg2  # type: ignore
    except ImportError:
        sys.exit("ERROR: psycopg2 not installed. pip install psycopg2-binary")
    conn = psycopg2.connect(
        host=host, port=port, dbname=db, user=user,
        password=password, connect_timeout=10,
        options="-c default_transaction_read_only=on",
    )
    conn.autocommit = False
    cur = conn.cursor()
    cur.execute("SET TRANSACTION READ ONLY;")
    cur.execute("SHOW transaction_read_only;")
    if cur.fetchone()[0] != "on":
        sys.exit("ERROR: could not enforce read-only. Aborting.")
    return conn, cur


# ─── Catalog queries (each ORDERed for determinism) ─────────────────────────

USER_SCHEMAS = """
    SELECT n.nspname
    FROM pg_namespace n
    WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
      AND n.nspname NOT LIKE 'pg_temp_%%'
      AND n.nspname NOT LIKE 'pg_toast_temp_%%'
    ORDER BY n.nspname;
"""


def q(cur, sql, args=None):
    cur.execute(sql, args)
    return cur.fetchall()


def pg_info(cur):
    cur.execute("SELECT version();")
    version = cur.fetchone()[0]
    # Schema-affecting settings only. server_version and server_version_num are
    # version-specific (not schema) and are captured in pg_version_string (excluded
    # from fingerprint). This makes the fingerprint stable across PG patch versions.
    cur.execute(
        "SELECT name, setting FROM pg_settings "
        "WHERE name IN ('integer_datetime',"
        "'lc_collate','lc_ctype','standard_conforming_strings','TimeZone') ORDER BY name;"
    )
    settings = {r[0]: r[1] for r in cur.fetchall()}
    return {"settings": settings}


def pg_version_string(cur):
    """Full version string — reported but EXCLUDED from the fingerprint."""
    cur.execute("SELECT version();")
    return cur.fetchone()[0]


def extensions(cur):
    return [
        {"name": r[0], "version": r[1]}
        for r in q(cur, "SELECT extname, extversion FROM pg_extension ORDER BY extname;")
    ]


def schemas(cur):
    return [r[0] for r in q(cur, USER_SCHEMAS)]


def relations(cur, sch):
    if not sch:
        return []
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, c.relkind, c.relpersistence,
               pg_catalog.format_type(c.reltype, NULL)
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s)
          AND c.relkind IN ('r','v','m','S','f','p')
        ORDER BY n.nspname, c.relkind, c.relname;
        """,
        [sch],
    )
    return [
        {"schema": r[0], "name": r[1], "kind": r[2], "persistence": r[3], "type": r[4]}
        for r in rows
    ]


def columns(cur, sch):
    if not sch:
        return {}
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, a.attnum, a.attname,
               pg_catalog.format_type(a.atttypid, a.atttypmod),
               a.attnotnull, pg_get_expr(d.adbin, d.adrelid), a.attidentity
        FROM pg_attribute a
        JOIN pg_class c ON c.oid = a.attrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
        WHERE n.nspname = ANY(%s) AND a.attnum > 0 AND NOT a.attisdropped
          AND c.relkind IN ('r','v','m','f','p')
        ORDER BY n.nspname, c.relname, a.attnum;
        """,
        [sch],
    )
    out = {}
    for r in rows:
        key = f"{r[0]}.{r[1]}"
        out.setdefault(key, []).append(
            {"name": r[3], "type": r[4], "not_null": r[5], "default": r[6], "identity": r[7]}
        )
    return out


def constraints(cur, sch):
    if not sch:
        return {}
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, con.conname, con.contype,
               pg_get_constraintdef(con.oid, true)
        FROM pg_constraint con
        JOIN pg_class c ON c.oid = con.conrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s)
        ORDER BY n.nspname, c.relname, con.contype, con.conname;
        """,
        [sch],
    )
    out = {}
    for r in rows:
        out.setdefault(f"{r[0]}.{r[1]}", []).append(
            {"name": r[2], "type": r[3], "definition": r[4]}
        )
    return out


def indexes(cur, sch):
    if not sch:
        return {}
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, i.relname,
               pg_get_indexdef(ix.indexrelid, 0, true),
               ix.indisunique, ix.indisprimary
        FROM pg_index ix
        JOIN pg_class c ON c.oid = ix.indrelid
        JOIN pg_class i ON i.oid = ix.indexrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s)
        ORDER BY n.nspname, c.relname, i.relname;
        """,
        [sch],
    )
    out = {}
    for r in rows:
        out.setdefault(f"{r[0]}.{r[1]}", []).append(
            {"name": r[2], "definition": r[3], "unique": r[4], "primary": r[5]}
        )
    return out


def sequences(cur, sch):
    if not sch:
        return []
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname,
               s.seqtypid::regtype, s.seqstart, s.seqincrement,
               s.seqmax, s.seqmin, s.seqcache, s.seqcycle
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_sequence s ON s.seqrelid = c.oid
        WHERE n.nspname = ANY(%s) AND c.relkind = 'S'
        ORDER BY n.nspname, c.relname;
        """,
        [sch],
    )
    return [
        {
            "schema": r[0], "name": r[1], "type": r[2], "start": r[3],
            "increment": r[4], "max": r[5], "min": r[6], "cache": r[7], "cycle": r[8],
        }
        for r in rows
    ]


def functions(cur, sch):
    if not sch:
        return []
    rows = q(
        cur,
        """
        SELECT n.nspname, p.proname,
               pg_get_function_identity_arguments(p.oid),
               pg_get_functiondef(p.oid)
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = ANY(%s) AND p.prokind IN ('f','p','w')
        ORDER BY n.nspname, p.proname,
                 pg_get_function_identity_arguments(p.oid);
        """,
        [sch],
    )
    return [
        {"schema": r[0], "name": r[1], "args": r[2], "body": r[3]} for r in rows
    ]


def triggers(cur, sch):
    if not sch:
        return {}
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, t.tgname, pg_get_triggerdef(t.oid, true)
        FROM pg_trigger t
        JOIN pg_class c ON c.oid = t.tgrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s) AND NOT t.tgisinternal
        ORDER BY n.nspname, c.relname, t.tgname;
        """,
        [sch],
    )
    out = {}
    for r in rows:
        out.setdefault(f"{r[0]}.{r[1]}", []).append({"name": r[2], "definition": r[3]})
    return out


def views(cur, sch):
    if not sch:
        return []
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, pg_get_viewdef(c.oid, true)
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s) AND c.relkind IN ('v','m')
        ORDER BY n.nspname, c.relname;
        """,
        [sch],
    )
    return [{"schema": r[0], "name": r[1], "definition": r[2]} for r in rows]


def rls_policies(cur, sch):
    if not sch:
        return []
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, p.polname, p.polcmd, p.polpermissive,
               pg_get_expr(p.polqual, p.polrelid),
               pg_get_expr(p.polwithcheck, p.polrelid),
               (SELECT array_agg(rolname) FROM pg_roles r2, unnest(p.polroles) AS proid
                WHERE r2.oid = proid)
        FROM pg_policy p
        JOIN pg_class c ON c.oid = p.polrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s)
        ORDER BY n.nspname, c.relname, p.polname;
        """,
        [sch],
    )
    return [
        {
            "schema": r[0], "table": r[1], "name": r[2], "cmd": r[3],
            "permissive": r[4], "using": r[5], "with_check": r[6],
            "roles": list(r[7]) if r[7] else [],
        }
        for r in rows
    ]


def rls_state(cur, sch):
    """Per-table RLS enabled/forced flags (separate from policies)."""
    if not sch:
        return []
    rows = q(
        cur,
        """
        SELECT n.nspname, c.relname, c.relrowsecurity, c.relforcerowsecurity
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = ANY(%s) AND c.relkind IN ('r','v','m')
          AND (c.relrowsecurity OR c.relforcerowsecurity)
        ORDER BY n.nspname, c.relname;
        """,
        [sch],
    )
    return [
        {"schema": r[0], "table": r[1], "rls_enabled": r[2], "rls_forced": r[3]}
        for r in rows
    ]


def migration_state(cur):
    """Detect any migration ledger table and read its state (no row data).

    Uses catalog-driven column detection (not hardcoded column names) so it
    works regardless of whether the ledger uses created_at, applied_at, or
    another timestamp column. Wraps the read in a savepoint so a failure
    (missing column, permissions) does not abort the surrounding read-only
    transaction."""
    candidates = [
        "schema_migrations",
        "platform.schema_revisions",
        "golang_migrations",
        "goose_db_version",
    ]
    for tbl in candidates:
        schema, name = (tbl.split(".", 1) + [None])[:2] if "." in tbl else ("public", tbl)
        cur.execute(
            "SELECT EXISTS (SELECT 1 FROM information_schema.tables "
            "WHERE table_schema = %s AND table_name = %s);",
            [schema, name],
        )
        if not cur.fetchone()[0]:
            continue

        # Catalog-driven: find a timestamp column to use for "latest"
        cur.execute(
            """
            SELECT column_name FROM information_schema.columns
            WHERE table_schema = %s AND table_name = %s
              AND data_type LIKE '%%timestamp%%'
            ORDER BY column_name LIMIT 1;
            """,
            [schema, name],
        )
        row = cur.fetchone()
        ts_col = row[0] if row else None

        # Savepoint so a failure here doesn't abort the read-only transaction
        cur.execute("SAVEPOINT migration_state_probe")
        try:
            if ts_col:
                # qualify the column to avoid SQL injection from catalog content;
                # column names are validated to be identifiers via information_schema
                cur.execute(
                    'SELECT count(*), max("' + ts_col.replace('"', '""') + '"::text) '
                    "FROM " + schema + "." + name
                )
                cnt, latest = cur.fetchone()
                cur.execute("RELEASE SAVEPOINT migration_state_probe")
                return {
                    "table": tbl, "row_count": cnt,
                    "latest_column": ts_col, "latest_recorded_at": latest,
                }
            cur.execute("SELECT count(*) FROM " + schema + "." + name)
            cnt = cur.fetchone()[0]
            cur.execute("RELEASE SAVEPOINT migration_state_probe")
            return {"table": tbl, "row_count": cnt, "latest_column": None}
        except Exception:
            cur.execute("ROLLBACK TO SAVEPOINT migration_state_probe")
            cur.execute("RELEASE SAVEPOINT migration_state_probe")
            return {"table": tbl, "note": "exists but unreadable"}
    return {"table": None, "note": "no migration ledger table detected"}


def roles_and_grants(cur):
    """Roles, memberships, grants, default privileges. Ownership is captured
    SEPARATELY (see ownership()) so it can be excluded from the fingerprint
    per spec §4.3. rolpassword is never selected."""
    # Roles (no password column)
    cur.execute(
        """
        SELECT r.rolname, r.rolsuper, r.rolinherit, r.rolcreaterole,
               r.rolcreatedb, r.rolcanlogin, r.rolreplication, r.rolbypassrls
        FROM pg_roles r
        WHERE r.rolname !~ '^pg_'
        ORDER BY r.rolname;
        """
    )
    roles = [
        {
            "name": r[0], "superuser": r[1], "inherit": r[2], "createrole": r[3],
            "createdb": r[4], "canlogin": r[5], "replication": r[6], "bypassrls": r[7],
        }
        for r in cur.fetchall()
    ]
    # Memberships
    cur.execute(
        """
        SELECT r.rolname AS member, r2.rolname AS role_of
        FROM pg_auth_members m
        JOIN pg_roles r ON r.oid = m.member
        JOIN pg_roles r2 ON r2.oid = m.roleid
        WHERE r.rolname !~ '^pg_' AND r2.rolname !~ '^pg_'
        ORDER BY member, role_of;
        """
    )
    memberships = [{"member": r[0], "role_of": r[1]} for r in cur.fetchall()]

    # Comprehensive grants via aclexplode — covers ALL object classes:
    # tables, sequences, functions, schemas, types, databases, columns,
    # large objects, plus PUBLIC grantee. This supersedes the prior
    # information_schema.role_table_grants approach which missed most classes.
    grant_material = _all_grants_material(cur)
    grants_digest = hashlib.sha256(grant_material.encode("utf-8")).hexdigest()

    # Default privileges (ACL stored as defaclacl; render to text)
    cur.execute(
        """
        SELECT pg_get_userbyid(d.defaclrole), n.nspname, d.defaclobjtype,
               array_to_string(d.defaclacl, ',')
        FROM pg_default_acl d
        LEFT JOIN pg_namespace n ON n.oid = d.defaclnamespace
        ORDER BY 1,2,3;
        """
    )
    default_privs = [
        {"creator": r[0], "schema": r[1], "objtype": r[2], "acl": r[3]}
        for r in cur.fetchall()
    ]
    return {
        "roles": roles,
        "memberships": memberships,
        "grants_sha256": grants_digest,
        "default_privileges": default_privs,
    }


def ownership(cur):
    """Object ownership — captured but EXCLUDED from the fingerprint (spec §4.3
    excludes ownership). Reported for the manifest, not hashed."""
    cur.execute(
        """
        SELECT n.nspname, c.relname, r.rolname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_roles r ON r.oid = c.relowner
        WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
          AND n.nspname NOT LIKE 'pg_temp_%%'
          AND c.relkind IN ('r','v','m','S','f','p')
        ORDER BY n.nspname, c.relname;
        """
    )
    return [{"schema": r[0], "relation": r[1], "owner": r[2]} for r in cur.fetchall()]


def _all_grants_material(cur):
    """Build a canonical string of ALL ACL grants across object classes for
    digesting. Covers: relations (r), schemas (n), sequences (S), functions (f),
    databases (d), types (T), columns, large objects — including PUBLIC.
    Uses pg_acldatacl / aclexplode for completeness."""
    parts = []
    # Relation-level grants (tables, views, sequences, matviews) via pg_class.relacl
    cur.execute(
        """
        SELECT n.nspname, c.relname, c.relkind,
               pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
               a.privilege_type, a.is_grantable
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace,
        aclexplode(c.relacl) AS a(grantor, grantee, privilege_type, is_grantable)
        WHERE n.nspname !~ '^(pg_|information_schema|pg_toast)'
          AND c.relkind IN ('r','v','m','S','f','p')
        ORDER BY 1,2,3,4,5,6;
        """
    )
    for r in cur.fetchall():
        parts.append(f"rel|{r[0]}|{r[1]}|{r[2]}|{r[3]}|{r[4]}|{r[5]}|{r[6]}")

    # Database-level grants
    cur.execute(
        """
        SELECT datname, pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
               a.privilege_type, a.is_grantable
        FROM pg_database d,
        aclexplode(d.datacl) AS a(grantor, grantee, privilege_type, is_grantable)
        WHERE datname !~ '^template'
        ORDER BY 1,2,3,4;
        """
    )
    for r in cur.fetchall():
        parts.append(f"db|{r[0]}|{r[1]}|{r[2]}|{r[3]}|{r[4]}")

    # Schema-level grants
    cur.execute(
        """
        SELECT n.nspname, pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
               a.privilege_type, a.is_grantable
        FROM pg_namespace n,
        aclexplode(n.nspacl) AS a(grantor, grantee, privilege_type, is_grantable)
        WHERE n.nspname !~ '^(pg_|information_schema|pg_toast)'
        ORDER BY 1,2,3,4;
        """
    )
    for r in cur.fetchall():
        parts.append(f"schema|{r[0]}|{r[1]}|{r[2]}|{r[3]}|{r[4]}")

    # Function/procedure grants
    cur.execute(
        """
        SELECT n.nspname, p.proname,
               pg_get_function_identity_arguments(p.oid),
               pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
               a.privilege_type, a.is_grantable
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace,
        aclexplode(p.proacl) AS a(grantor, grantee, privilege_type, is_grantable)
        WHERE n.nspname !~ '^(pg_|information_schema)'
        ORDER BY 1,2,3,4,5,6;
        """
    )
    for r in cur.fetchall():
        parts.append(f"func|{r[0]}|{r[1]}|{r[2]}|{r[3]}|{r[4]}|{r[5]}|{r[6]}")

    # Sequence grants (column-level covered via relacl above for kind 'S')
    # Type grants (T)
    cur.execute(
        """
        SELECT n.nspname, t.typname,
               pg_get_userbyid(a.grantor), pg_get_userbyid(a.grantee),
               a.privilege_type, a.is_grantable
        FROM pg_type t
        JOIN pg_namespace n ON n.oid = t.typnamespace,
        aclexplode(t.typacl) AS a(grantor, grantee, privilege_type, is_grantable)
        WHERE n.nspname !~ '^(pg_|information_schema)'
        ORDER BY 1,2,3,4,5;
        """
    )
    for r in cur.fetchall():
        parts.append(f"type|{r[0]}|{r[1]}|{r[2]}|{r[3]}|{r[4]}|{r[5]}")

    return "\n".join(parts)


def table_counts(cur, rels):
    """Approximate row counts for tables only — for reconciliation.
    EXCLUDED from the fingerprint (data, not schema)."""
    counts = {}
    for rel in rels:
        if rel["kind"] != "r":
            continue
        key = f'{rel["schema"]}.{rel["name"]}'
        try:
            cur.execute(f'SELECT count(*) FROM "{rel["schema"]}"."{rel["name"]}";')  # noqa: S608
            counts[key] = cur.fetchone()[0]
        except Exception:
            counts[key] = "ERROR"
    return counts


def integrity_checks(cur, sch):
    """Structural integrity checks: orphan FKs, dup indexes, invalid constraints.
    No data exposure."""
    checks = {}
    # orphan foreign keys (references a table not in our schemas)
    try:
        cur.execute(
            """
            SELECT n.nspname, c.relname, con.conname
            FROM pg_constraint con
            JOIN pg_class c ON c.oid = con.conrelid
            JOIN pg_namespace n ON n.oid = c.relnamespace
            JOIN pg_class c2 ON c2.oid = con.confrelid
            JOIN pg_namespace n2 ON n2.oid = c2.relnamespace
            WHERE con.contype = 'f' AND n.nspname = ANY(%s)
              AND n2.nspname <> ANY(%s)
            ORDER BY 1,2,3;
            """,
            [sch, sch],
        )
        checks["orphan_foreign_keys"] = [
            {"schema": r[0], "table": r[1], "constraint": r[2]} for r in cur.fetchall()
        ]
    except Exception as e:
        checks["orphan_foreign_keys"] = f"error: {e}"
    # invalid constraints / indexes
    try:
        cur.execute(
            """
            SELECT n.nspname, c.relname, con.conname, con.contype
            FROM pg_constraint con
            JOIN pg_class c ON c.oid = con.conrelid
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = ANY(%s) AND NOT con.convalidated
            ORDER BY 1,2,3;
            """,
            [sch],
        )
        checks["invalid_constraints"] = [
            {"schema": r[0], "table": r[1], "constraint": r[2], "type": r[3]}
            for r in cur.fetchall()
        ]
    except Exception as e:
        checks["invalid_constraints"] = f"error: {e}"
    return checks


# ─── Manifest + fingerprint ─────────────────────────────────────────────────

# Fields excluded from the fingerprint (volatile or data, not schema).
FINGERPRINT_EXCLUDE = {
    "captured_at_utc",
    "row_counts",
    "source_label",
    "integrity_checks",
    "schema_dump_sha256",   # capture artifact, not schema
    "schema_dump_error",    # capture artifact, not schema
    "fingerprint_sha256",   # MUST be excluded: the digest cannot include itself
    "ownership",            # spec §4.3 explicitly excludes ownership from fingerprint
    "pg_version_string",    # build-specific (compiler/musl label); not schema
}


def build_manifest(cur, source_label):
    sch = schemas(cur)
    rels = relations(cur, sch)
    return {
        "profiler_version": PROFILER_VERSION,
        "captured_at_utc": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "source_label": source_label,
        "postgres": {**pg_info(cur), "extensions": extensions(cur)},
        "pg_version_string": pg_version_string(cur),
        "schemas": sch,
        "relations": rels,
        "columns": columns(cur, sch),
        "constraints": constraints(cur, sch),
        "indexes": indexes(cur, sch),
        "sequences": sequences(cur, sch),
        "functions": functions(cur, sch),
        "triggers": triggers(cur, sch),
        "views": views(cur, sch),
        "rls_policies": rls_policies(cur, sch),
        "rls_state": rls_state(cur, sch),
        "migration_state": migration_state(cur),
        "roles_and_grants": roles_and_grants(cur),
        "ownership": ownership(cur),
        "integrity_checks": integrity_checks(cur, sch),
        "row_counts": table_counts(cur, rels),
    }


def canonicalize(manifest):
    stable = {k: v for k, v in manifest.items() if k not in FINGERPRINT_EXCLUDE}
    return json.dumps(
        stable, sort_keys=True, ensure_ascii=True, separators=(",", ":")
    ).encode("utf-8")


def fingerprint_of(manifest):
    return hashlib.sha256(canonicalize(manifest)).hexdigest()


def schema_only_dump(host, port, db, user, password, out_path):
    """pg_dump --schema-only. Redacted of any inline secrets by the read-only
    nature of the source (no credential columns are in catalog DDL)."""
    env = dict(os.environ, PGPASSWORD=password)
    cmd = [
        "pg_dump",
        "--schema-only",
        "--no-owner",
        "--no-privileges",
        "-h", host, "-p", str(port), "-U", user, db,
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, env=env, timeout=120)
        if result.returncode != 0:
            return None, result.stderr.strip()
        with open(out_path, "w", encoding="utf-8") as f:
            f.write(result.stdout)
        return hashlib.sha256(result.stdout.encode("utf-8")).hexdigest(), None
    except FileNotFoundError:
        return None, "pg_dump not found on PATH"
    except Exception as e:
        return None, str(e)


# ─── CLI ─────────────────────────────────────────────────────────────────────


def cmd_capture(args):
    pw = os.environ.get("PGPASSWORD", "")
    conn, cur = connect(args.host, args.port, args.db, args.user, pw)
    os.makedirs(args.out, exist_ok=True)
    try:
        m = build_manifest(cur, args.label)
        m["fingerprint_sha256"] = fingerprint_of(m)

        manifest_path = os.path.join(args.out, "manifest.json")
        with open(manifest_path, "w", encoding="utf-8") as f:
            json.dump(m, f, indent=2, sort_keys=True)

        # schema-only dump (best-effort; may be unavailable if pg_dump absent)
        dump_path = os.path.join(args.out, "schema.sql")
        dump_digest, dump_err = schema_only_dump(
            args.host, args.port, args.db, args.user, pw, dump_path
        )
        m["schema_dump_sha256"] = dump_digest
        m["schema_dump_error"] = dump_err
        # rewrite manifest with dump digest
        with open(manifest_path, "w", encoding="utf-8") as f:
            json.dump(m, f, indent=2, sort_keys=True)

        print(f"wrote {manifest_path}")
        print(f"fingerprint_sha256 = {m['fingerprint_sha256']}")
        print(f"profiler_version   = {m['profiler_version']}")
        print(f"relations          = {len(m['relations'])}")
        print(f"schema_dump_sha256 = {dump_digest or 'N/A (' + str(dump_err) + ')'}")
    finally:
        conn.rollback()
        conn.close()


def cmd_fingerprint(args):
    m = json.load(open(args.manifest, encoding="utf-8"))
    print(fingerprint_of(m))


def cmd_compare(args):
    a = json.load(open(args.a, encoding="utf-8"))
    b = json.load(open(args.b, encoding="utf-8"))
    fa, fb = fingerprint_of(a), fingerprint_of(b)
    print(f"A ({a.get('source_label','?')}): {fa}")
    print(f"B ({b.get('source_label','?')}): {fb}")
    if fa == fb:
        print("MATCH — identical canonical schema.")
        return 0
    print("DIFFER — canonical schema differs. Sections differing:")
    for section in (
        "schemas", "relations", "columns", "constraints", "indexes",
        "sequences", "functions", "triggers", "views", "rls_policies", "rls_state",
        "migration_state", "roles_and_grants",
    ):
        sa, sb = a.get(section), b.get(section)
        # Sort list-of-dicts canonically to avoid false positives from ordering
        if isinstance(sa, list):
            sa = sorted([json.dumps(x, sort_keys=True) for x in sa])
        if isinstance(sb, list):
            sb = sorted([json.dumps(x, sort_keys=True) for x in sb])
        if sa != sb:
            print(f"  {section}")
    return 1


def main():
    p = argparse.ArgumentParser(description="ClarityIT WP-00 schema capture (P1/P2)")
    sub = p.add_subparsers(dest="cmd", required=True)

    pc = sub.add_parser("capture", help="capture a comprehensive schema profile")
    pc.add_argument("--host", required=True)
    pc.add_argument("--port", type=int, default=5432)
    pc.add_argument("--db", required=True)
    pc.add_argument("--user", required=True)
    pc.add_argument("--label", required=True, help="e.g. P1-production / P2-restored")
    pc.add_argument("--out", required=True, help="output directory")
    pc.set_defaults(func=cmd_capture)

    pf = sub.add_parser("fingerprint", help="recompute fingerprint from a manifest")
    pf.add_argument("--manifest", required=True)
    pf.set_defaults(func=cmd_fingerprint)

    pcmp = sub.add_parser("compare", help="compare two manifests by fingerprint")
    pcmp.add_argument("--a", required=True)
    pcmp.add_argument("--b", required=True)
    pcmp.set_defaults(func=cmd_compare)

    args = p.parse_args()
    sys.exit(args.func(args) or 0)


if __name__ == "__main__":
    main()
