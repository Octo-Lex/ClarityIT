#!/usr/bin/env python3
"""
generate_g2_grants.py — closed-world, per-object grants inventory for G2 manifest.

Reads TARGET-SCHEMA-MANIFEST.json, classifies each function as application-owned
(plpgsql or sql defined by our migrations) vs extension-provided (c/internal from
pgcrypto/citext/pg_trgm), and emits a per-object grants block replacing the
aggregate ALL TABLES / ALL FUNCTIONS shorthand.

Application-owned function set is derived from the migrations themselves
(grep CREATE FUNCTION across migrations/001-040) and asserted against the
manifest to catch drift.

Output: writes a new target_grants block to stdout as canonical JSON.
"""
import json
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
MANIFEST = REPO / "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json"

# Application functions are defined by CREATE FUNCTION in migrations 001-040.
# Determined by scanning migration files (see validate_application_functions()).
APPLICATION_FUNCTIONS = {
    "set_updated_at",
    "normalize_user_email",
    "normalize_team_slug",
    "protect_last_team_owner",
    "prevent_bootstrap_unlock",
    "trg_artifacts_updated_at",
    "adoc_set_updated_at",
    "ki_search_vector_update",
    "kc_search_vector_update",
    "ki_set_updated_at",
}

# Table privileges granted to clarityit_app (soft-delete model: no DELETE).
APP_TABLE_PRIVS = ["SELECT", "INSERT", "UPDATE"]
# Sequence privileges for clarityit_app.
APP_SEQ_PRIVS = ["USAGE", "SELECT"]
# Schema privilege for clarityit_app.
APP_SCHEMA_PRIV = "USAGE"


def discover_application_functions():
    """Re-derive the application function set from migrations to catch drift."""
    found = set()
    for sql in sorted((REPO / "migrations").glob("0[0-9][0-9]*.sql")):
        text = sql.read_text(encoding="utf-8")
        for m in re.finditer(r"CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:public\.)?([a-z_]+)\s*\(", text, re.I):
            name = m.group(1)
            # Skip extension-provided functions that migrations CREATE EXTENSION
            if name in {"pgp_pub_decrypt_bytea", "pgp_pub_decrypt", "pgp_sym_encrypt_bytea",
                        "pgp_sym_encrypt", "pgp_sym_decrypt_bytea", "pgp_sym_decrypt",
                        "pgp_pub_encrypt_bytea", "pgp_pub_encrypt", "pgp_key_id",
                        "pgp_armor_headers", "gen_salt", "gen_random_uuid", "gen_random_bytes",
                        "encrypt_iv", "encrypt", "decrypt_iv", "decrypt", "dearmor",
                        "crypt", "hmac", "digest", "armor", "citext"}:
                continue
            found.add(name)
    return found


def main():
    d = json.loads(MANIFEST.read_text(encoding="utf-8"))

    # Validate application function set against migrations.
    discovered = discover_application_functions()
    if discovered != APPLICATION_FUNCTIONS:
        print(f"ERROR: application function set drifted.", file=sys.stderr)
        print(f"  hardcoded: {sorted(APPLICATION_FUNCTIONS)}", file=sys.stderr)
        print(f"  discovered: {sorted(discovered)}", file=sys.stderr)
        sys.exit(1)

    # Partition manifest functions into app vs extension.
    app_in_manifest = []
    ext_in_manifest = []
    for f in d["functions"]:
        name = f["name"]
        body = f.get("body", "")
        is_extension = (
            name not in APPLICATION_FUNCTIONS
            or re.search(r"LANGUAGE\s+(c|internal)\b", body, re.I)
            or "$libdir" in body
        )
        if is_extension:
            ext_in_manifest.append(name)
        else:
            app_in_manifest.append(name)

    # Sanity: every declared app function must be in the manifest.
    missing_in_manifest = APPLICATION_FUNCTIONS - set(app_in_manifest)
    if missing_in_manifest:
        print(f"ERROR: app functions missing from manifest: {missing_in_manifest}", file=sys.stderr)
        sys.exit(1)

    tables = sorted(d["tables"].keys())  # 64 entries like "public.action_outcomes"
    sequences = [s["name"] for s in d["sequences"]]
    schemas = [s if isinstance(s, str) else s.get("name", s) for s in d["schemas"]]
    schemas = [s for s in schemas if s]

    # Build per-object grants.
    table_grants = []
    for fqtn in tables:
        # fqtn is "public.<name>"
        schema, _, tname = fqtn.partition(".")
        table_grants.append({
            "schema": schema,
            "name": tname,
            "grantee": "clarityit_app",
            "privileges": list(APP_TABLE_PRIVS),
            "grant_option": False,
        })

    app_fn_grants = []
    for fn in sorted(set(app_in_manifest)):
        app_fn_grants.append({
            "schema": "public",
            "name": fn,
            "grantee": "clarityit_app",
            "privileges": ["EXECUTE"],
            "grant_option": False,
        })

    seq_grants = []
    for seq in sequences:
        seq_grants.append({
            "schema": "public",
            "name": seq,
            "grantee": "clarityit_app",
            "privileges": list(APP_SEQ_PRIVS),
            "grant_option": False,
        })

    schema_grants = []
    for sc in schemas:
        schema_grants.append({
            "schema": sc,
            "grantee": "clarityit_app",
            "privilege": APP_SCHEMA_PRIV,
            "grant_option": False,
        })

    new_block = {
        "tables": table_grants,
        "application_functions": app_fn_grants,
        "extension_functions": {
            "count": len(ext_in_manifest),
            "names": sorted(set(ext_in_manifest)),
            "acl_policy": (
                "Extension-provided functions (pgcrypto, citext, pg_trgm) retain their "
                "default ACL as installed by CREATE EXTENSION. The REVOKE EXECUTE FROM PUBLIC "
                "policy applies ONLY to application functions. Revoking PUBLIC EXECUTE from "
                "extension functions would break the extension operator classes and is out of scope."
            ),
        },
        "sequences": seq_grants,
        "schemas": schema_grants,
        "public_revoke_scope": {
            "tables": "No PUBLIC revocation needed — PostgreSQL grants no default table privileges to PUBLIC.",
            "application_functions": (
                "REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC applies to all 10 "
                "application functions. Per-object EXECUTE then granted to clarityit_app."
            ),
            "extension_functions": "EXCLUDED from PUBLIC revoke — managed by CREATE EXTENSION.",
        },
        "grant_options": {"no_role_has_with_grant_option": True},
    }

    print(json.dumps({
        "counts": {
            "tables": len(table_grants),
            "application_functions": len(app_fn_grants),
            "extension_functions_excluded": len(ext_in_manifest),
            "sequences": len(seq_grants),
            "schemas": len(schema_grants),
        },
        "target_grants": new_block,
    }, indent=2))


if __name__ == "__main__":
    main()
