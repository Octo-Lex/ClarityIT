#!/usr/bin/env python3
"""Generate the ClarityIT WP-00 G3 reconciled baseline artifacts.

The signed G2 manifest is a read-only input.  This generator intentionally does
not reuse generate_p3.py: G2 stores columns, constraints, indexes, and triggers
inside each table object, whereas P3 stores those collections at the top level.

All outputs are deterministic.  No clock, hostname, current branch name,
environment identity, credential, or database-generated value enters an
artifact.  The legacy archive is copied from the signed G2 commit's Git blobs,
not from mutable working-tree files.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
import uuid
from pathlib import Path, PurePosixPath
from typing import Iterable


ROOT = Path(__file__).resolve().parents[2]

FROZEN_G2_COMMIT = "f04f94faad0105d1c3274e9c7974d44f936a0d28"
G2_MANIFEST_PATH = "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json"
G2_CHECKSUM_PATH = "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.sha256"
G2_RECEIPT_PATH = "migrations/profiles/g2/G2-APPROVALS.md"
G2_MANIFEST_SHA256 = "1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63"
G2_MANIFEST_SIZE = 284_064
POSTGRES_DB = "clarityit"
GENERATOR_VERSION = "g3-baseline-generator-v1"
BASELINE_VERSION = "0001"

# P3 adoption-source constants (G1-approved sanitized legacy fixture).
P3_GOLDEN_FINGERPRINT = "cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b"
P3_SOURCE_COMMIT = "29c4cdcb4c7bd9f13209f5627b55f4fabbd08a33"
G1_APPROVAL_REF = "3b4a6fdeb35473e5f73ca74bafa479bd2648fb10"
G3_ARTIFACT_DATE = "2026-08-03T00:00:00Z"
ADOPTION_SOURCE_PROFILE_NAMESPACE = uuid.UUID("a4d9c3f1-7e2b-4a6f-9d18-3b5c8e1f0a27")
P3_PROFILE_ID = str(uuid.uuid5(
    ADOPTION_SOURCE_PROFILE_NAMESPACE,
    f"clarityit:g3:source-profile:{P3_GOLDEN_FINGERPRINT}",
))

# SQL-derived structural digest constants for the approved P3 source shape.
# These are computed INSIDE the adoption transaction from the live catalog
# (not supplied by the caller) and compared against these frozen values, so
# the adoption is fail-closed against a drifted source even when the SQL is
# invoked directly without the Python pre-check.
#   P3_COLUMN_DIGEST: SHA-256 over ordered (table.column.data_type.charlen)
#   P3_APPFN_DIGEST:  SHA-256 over the 10 signed application-function signatures
# Complete SQL-derived structural digests for the drift gate.  Each covers
# the full property set so that function-body changes, column default/
# nullability/identity changes, renamed constraints, or changed sequence
# cache all fail the gate.  All are computed inside the adoption transaction
# from the live catalog (not caller-supplied).
P3_COLUMN_DIGEST = "33ac369b7cb896ce22c91766c1caf12755506b11c799d929a4cb22aeb4ef2303"
P3_APPFN_SIG_DIGEST = "53eafc8837007c94620c786edbdb5c0db3c11c5e3675a987f8be231ae2357ab0"
P3_APPFN_BODY_DIGEST = "143cc88d07fa638c9e4d2a515140b987db210c6f381537ed7d6e75dff664f0f8"
P3_CONSTRAINT_DIGEST = "87372790e05c745ee3867cfe89d06df1017c9247615f0b7d98b8d55eba99fdf3"
P3_INDEX_DIGEST = "6dc397c2f5f5e36a6b946efb8cf39052e04fef311e8bb913506bb345a8190cf3"
P3_TRIGGER_DIGEST = "b379674f75f67a40c684c3ee9133019972ddca591c287659cbf076e22dca7333"
P3_SEQUENCE_DIGEST = "876fc2ed13992bedea3a31fab0a67a35770b4294a9d5718ee7f33138a90df216"
P3_APPFN_NAMES = (
    "adoc_set_updated_at", "kc_search_vector_update", "ki_search_vector_update",
    "ki_set_updated_at", "normalize_team_slug", "normalize_user_email",
    "prevent_bootstrap_unlock", "protect_last_team_owner", "set_updated_at",
    "trg_artifacts_updated_at",
)

# The governed target fingerprint (clarityit-g3-governed-v1) is the convergence
# target: both fresh installs and the P3-adopted database reach exact equality
# on this projection.  It is recorded in the A4 manifest and the receipt.
GOVERNED_TARGET_FINGERPRINT = "9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6"

LEGACY_DIR = Path("migrations/legacy/v1/001-040")
LEGACY_SUMS = Path("migrations/legacy/v1/SHA256SUMS")
ROLES_SQL = Path("migrations/v2/bootstrap/0000_roles.sql")
PLATFORM_SQL = Path("migrations/v2/bootstrap/0000_platform.sql")
BASELINE_SQL = Path("migrations/v2/baseline/0001_reconciled.sql")
SEED_SQL = Path("migrations/v2/baseline/0001_seed.sql")
CONTROL_MANIFEST = Path("migrations/v2/manifests/CONTROL-SCHEMA-MANIFEST.json")
A4_MANIFEST = Path("migrations/v2/manifests/G3-A4-MANIFEST.json")
V2_SUMS = Path("migrations/v2/manifests/SHA256SUMS")
ADOPTION_SQL = Path("migrations/v2/adoption/0001_adopt_p3.sql")

REQUIRED_EXTENSIONS = ("pgcrypto", "citext", "pg_trgm")

CANONICAL_PERMISSIONS = (
    ("work.items.update.own", "Update own work items", "work.items", "update.own", "low"),
    ("work.items.update.any", "Update any work item", "work.items", "update.any", "medium"),
    ("projects.update", "Update projects", "projects", "update", "medium"),
    ("incidents.update.own", "Update own incidents", "incidents", "update.own", "low"),
    ("incidents.update.any", "Update any incident", "incidents", "update.any", "medium"),
    ("docs.update.own", "Update own documents", "docs", "update.own", "low"),
    ("docs.update.any", "Update any document", "docs", "update.any", "medium"),
)

CONTROL_TABLES = {
    "source_profiles": {
        "columns": (
            ("profile_id", "text", "NOT NULL"),
            ("schema_fingerprint", "text", "NOT NULL"),
            ("postgres_version", "text", "NOT NULL"),
            ("postgres_major", "integer", "NOT NULL"),
            ("extensions", "jsonb", "NOT NULL"),
            ("roles_digest", "text", "NOT NULL"),
            ("source_commit", "text", "NOT NULL"),
            ("approved_by", "text", "NOT NULL"),
            ("approved_at", "timestamp with time zone", "NOT NULL"),
        ),
        "constraints": (
            "CONSTRAINT source_profiles_pkey PRIMARY KEY (profile_id)",
            "CONSTRAINT source_profiles_fingerprint_sha256 CHECK (schema_fingerprint ~ '^[0-9a-f]{64}$')",
            "CONSTRAINT source_profiles_roles_sha256 CHECK (roles_digest ~ '^[0-9a-f]{64}$')",
            "CONSTRAINT source_profiles_pg16 CHECK (postgres_major = 16)",
            "CONSTRAINT source_profiles_extensions_array CHECK (jsonb_typeof(extensions) = 'array')",
            "CONSTRAINT source_profiles_approver_present CHECK (btrim(approved_by) <> '')",
        ),
    },
    "schema_revisions": {
        "columns": (
            ("version", "text", "NOT NULL"),
            ("name", "text", "NOT NULL"),
            ("checksum", "text", "NOT NULL"),
            ("source_commit", "text", "NOT NULL"),
            ("applied_at", "timestamp with time zone", "NOT NULL"),
            ("applied_by", "text", "NOT NULL"),
            ("execution_ms", "bigint", "NOT NULL"),
            ("success", "boolean", "NOT NULL"),
        ),
        "constraints": (
            "CONSTRAINT schema_revisions_pkey PRIMARY KEY (version)",
            "CONSTRAINT schema_revisions_checksum_key UNIQUE (checksum)",
            "CONSTRAINT schema_revisions_checksum_sha256 CHECK (checksum ~ '^[0-9a-f]{64}$')",
            "CONSTRAINT schema_revisions_execution_nonnegative CHECK (execution_ms >= 0)",
        ),
    },
    "migration_runs": {
        "columns": (
            ("run_id", "uuid", "NOT NULL DEFAULT gen_random_uuid()"),
            ("database_name", "text", "NOT NULL DEFAULT current_database()"),
            ("source_profile_id", "text", ""),
            ("target_version", "text", "NOT NULL"),
            ("state", "text", "NOT NULL"),
            ("started_at", "timestamp with time zone", "NOT NULL"),
            ("completed_at", "timestamp with time zone", ""),
            ("release_id", "text", "NOT NULL"),
            ("evidence_ref", "text", ""),
        ),
        "constraints": (
            "CONSTRAINT migration_runs_pkey PRIMARY KEY (run_id)",
            "CONSTRAINT migration_runs_source_profile_fkey FOREIGN KEY (source_profile_id) REFERENCES platform.source_profiles(profile_id)",
            "CONSTRAINT migration_runs_state_check CHECK (state IN ('planned','profiled','preflighted','expanding','backfilling','reconciling','cutover_ready','cutover_committed','observing','completed','blocked','paused','precommit_rolled_back','forward_recovery_required'))",
            "CONSTRAINT migration_runs_time_order CHECK (completed_at IS NULL OR completed_at >= started_at)",
        ),
        "indexes": (
            "CREATE UNIQUE INDEX migration_runs_one_active_per_database ON platform.migration_runs (database_name) WHERE state NOT IN ('completed','blocked','precommit_rolled_back')",
        ),
    },
    "reconciliation_results": {
        "columns": (
            ("run_id", "uuid", "NOT NULL"),
            ("check_id", "text", "NOT NULL"),
            ("scope", "text", "NOT NULL"),
            ("expected", "jsonb", "NOT NULL"),
            ("actual", "jsonb", "NOT NULL"),
            ("result", "text", "NOT NULL"),
            ("evidence_ref", "text", "NOT NULL"),
            ("recorded_at", "timestamp with time zone", "NOT NULL"),
        ),
        "constraints": (
            "CONSTRAINT reconciliation_results_pkey PRIMARY KEY (run_id, check_id, scope)",
            "CONSTRAINT reconciliation_results_run_fkey FOREIGN KEY (run_id) REFERENCES platform.migration_runs(run_id)",
            "CONSTRAINT reconciliation_results_result_check CHECK (result IN ('pass','fail','blocked'))",
            "CONSTRAINT reconciliation_results_evidence_present CHECK (btrim(evidence_ref) <> '')",
        ),
    },
}


def git_blob(ref: str, path: str) -> bytes:
    proc = subprocess.run(
        ["git", "cat-file", "blob", f"{ref}:{path}"],
        cwd=ROOT,
        capture_output=True,
    )
    if proc.returncode:
        raise RuntimeError(
            f"cannot read committed blob {ref}:{path}: "
            f"{proc.stderr.decode('utf-8', errors='replace').strip()}"
        )
    return proc.stdout


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def json_bytes(value: object) -> bytes:
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + "\n").encode()


def sql_bytes(lines: Iterable[str]) -> bytes:
    return ("\n".join(lines).rstrip() + "\n").encode()


def canonical_repo_path(path: "Path | str") -> str:
    """Normalize a repository path to canonical forward-slash form.

    Generated artifacts (A4 manifest keys, detached checksum lines, archive
    inventory) embed repository-relative paths as bytes.  On Windows,
    ``str(Path("a/b"))`` yields ``"a\\b"`` and breaks determinism: the same
    generator committed on POSIX (forward slashes) cannot reproduce its own
    bytes on Windows.

    This helper forces forward slashes on every platform.  It cannot rely
    on ``Path.as_posix()`` alone, because on POSIX a backslash is a legal
    filename character (not a separator), so ``as_posix()`` leaves
    backslashes unchanged — a backslash-containing string or
    ``PureWindowsPath`` would pass through unnormalized.  Instead the
    string form is explicitly backslash-replaced before normalization.
    """
    text = str(path)
    text = text.replace("\\", "/")
    return str(PurePosixPath(text))


def validate_frozen_inputs() -> tuple[dict, bytes]:
    frozen = git_blob(FROZEN_G2_COMMIT, G2_MANIFEST_PATH)
    if sha256(frozen) != G2_MANIFEST_SHA256 or len(frozen) != G2_MANIFEST_SIZE:
        raise RuntimeError("frozen G2 manifest blob does not match the signed identity")

    current = git_blob("HEAD", G2_MANIFEST_PATH)
    if current != frozen:
        raise RuntimeError("G3 input drift: HEAD G2 manifest differs from signed f04f94f")
    if (ROOT / G2_MANIFEST_PATH).read_bytes() != frozen:
        raise RuntimeError("G3 input drift: working-tree G2 manifest differs from signed blob")

    for path in (G2_CHECKSUM_PATH, G2_RECEIPT_PATH):
        if git_blob("HEAD", path) != git_blob(FROZEN_G2_COMMIT, path):
            raise RuntimeError(f"G3 input drift: {path} differs from signed f04f94f")
        if (ROOT / path).read_bytes() != git_blob(FROZEN_G2_COMMIT, path):
            raise RuntimeError(f"G3 input drift: working-tree {path} differs from signed blob")

    manifest = json.loads(frozen)
    forbidden_top_level = {"columns", "constraints", "indexes", "triggers"}
    if forbidden_top_level & manifest.keys():
        raise RuntimeError("G2 structural contract violated: table members must remain nested")
    if len(manifest.get("tables", {})) != 64:
        raise RuntimeError("signed G2 manifest must contain exactly 64 tables")
    if any(
        set(table) != {"columns", "constraints", "indexes", "triggers"}
        for table in manifest["tables"].values()
    ):
        raise RuntimeError("unexpected nested G2 table shape")
    if len(manifest["target_grants"]["application_functions"]) != 10:
        raise RuntimeError("signed G2 manifest must identify exactly 10 application functions")
    if manifest.get("views") or manifest.get("rls_policies") or manifest.get("rls_state"):
        raise RuntimeError("generator v1 does not support non-empty views or RLS sections")
    return manifest, frozen


def legacy_paths() -> list[str]:
    proc = subprocess.run(
        ["git", "ls-tree", "-r", "--name-only", FROZEN_G2_COMMIT, "migrations"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=True,
    )
    paths = sorted(
        p for p in proc.stdout.splitlines()
        if re.fullmatch(r"migrations/0(?:0[1-9]|[1-3][0-9]|40)_[^/]+\.sql", p)
    )
    if len(paths) != 40:
        raise RuntimeError(f"expected 40 legacy migrations at signed G2 commit, got {len(paths)}")
    if [int(Path(p).name[:3]) for p in paths] != list(range(1, 41)):
        raise RuntimeError("legacy migrations are not the complete ordered range 001-040")
    return paths


def generate_roles_sql(manifest: dict) -> bytes:
    role_by_name = {role["name"]: role for role in manifest["target_roles"]}
    expected_names = {
        "clarityit_app", "clarityit", "clarityit_owner",
        "clarityit_migrator", "clarityit_admin",
    }
    if set(role_by_name) != expected_names:
        raise RuntimeError("G2 five-role posture changed")

    def role_options(role: dict) -> str:
        f = role["flags"]
        options = [
            "LOGIN" if f["canlogin"] else "NOLOGIN",
            "INHERIT" if f["inherit"] else "NOINHERIT",
            "CREATEDB" if f["createdb"] else "NOCREATEDB",
            "CREATEROLE" if f["createrole"] else "NOCREATEROLE",
            "SUPERUSER" if f["superuser"] else "NOSUPERUSER",
            "REPLICATION" if f["replication"] else "NOREPLICATION",
            "BYPASSRLS" if f["bypassrls"] else "NOBYPASSRLS",
        ]
        return " ".join(options)

    lines = [
        "-- G3 privileged bootstrap: exact signed five-role posture.",
        "-- Administrator-only; contains no passwords or environment identities.",
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.",
        r"\set ON_ERROR_STOP on",
        "BEGIN;",
        "DO $g3_preflight$",
        "BEGIN",
        f"    IF current_database() <> '{POSTGRES_DB}' THEN",
        f"        RAISE EXCEPTION 'G3 role bootstrap requires database {POSTGRES_DB}, got %', current_database();",
        "    END IF;",
        "    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN",
        "        RAISE EXCEPTION 'G3 role bootstrap requires a PostgreSQL superuser';",
        "    END IF;",
        "    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')) THEN",
        "        RAISE EXCEPTION 'G3 role bootstrap is fresh-install only: target role already exists';",
        "    END IF;",
        "END",
        "$g3_preflight$;",
        "",
    ]
    for name in ("clarityit_app", "clarityit", "clarityit_owner", "clarityit_migrator", "clarityit_admin"):
        lines.append(f"CREATE ROLE {name} {role_options(role_by_name[name])};")
    lines.append("")
    for membership in manifest["target_memberships"]:
        lines.append(membership["sql"] + ";")
    lines.extend([
        "",
        f"ALTER DATABASE {POSTGRES_DB} OWNER TO clarityit_owner;",
        "ALTER SCHEMA public OWNER TO clarityit_owner;",
        "REVOKE CREATE ON SCHEMA public FROM PUBLIC;",
        "GRANT USAGE ON SCHEMA public TO clarityit_app;",
        "",
    ])
    for default in manifest["target_default_privileges"]:
        privileges = ", ".join(default["privileges"])
        lines.append(
            f"ALTER DEFAULT PRIVILEGES FOR ROLE {default['creator']} "
            f"IN SCHEMA {default['schema']} GRANT {privileges} ON {default['object_type']} "
            f"TO {default['grantee']};"
        )
    for revoke in manifest["target_grants"]["default_privileges_public_revoke"]:
        if revoke["action"] != "REVOKE EXECUTE FROM PUBLIC":
            raise RuntimeError("unsupported G2 default-privilege PUBLIC revoke")
        lines.append(
            f"ALTER DEFAULT PRIVILEGES FOR ROLE {revoke['creator']} "
            f"IN SCHEMA {revoke['schema']} REVOKE EXECUTE ON {revoke['object_type']} FROM PUBLIC;"
        )
    lines.extend([
        "",
        "DO $g3_validate$",
        "BEGIN",
        "    ASSERT (SELECT count(*) FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')) = 5, 'G3 role count mismatch';",
    ])
    for name in sorted(role_by_name):
        f = role_by_name[name]["flags"]
        conditions = " AND ".join([
            f"rolsuper = {str(f['superuser']).lower()}",
            f"rolinherit = {str(f['inherit']).lower()}",
            f"rolcreaterole = {str(f['createrole']).lower()}",
            f"rolcreatedb = {str(f['createdb']).lower()}",
            f"rolcanlogin = {str(f['canlogin']).lower()}",
            f"rolreplication = {str(f['replication']).lower()}",
            f"rolbypassrls = {str(f['bypassrls']).lower()}",
        ])
        lines.append(
            f"    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '{name}' AND {conditions}), "
            f"'G3 role flags mismatch: {name}';"
        )
    for membership in manifest["target_memberships"]:
        lines.extend([
            "    ASSERT EXISTS (",
            "        SELECT 1 FROM pg_auth_members am",
            "        JOIN pg_roles member ON member.oid = am.member",
            "        JOIN pg_roles granted ON granted.oid = am.roleid",
            f"        WHERE member.rolname = '{membership['member']}'",
            f"          AND granted.rolname = '{membership['role_of']}'",
            f"          AND am.admin_option = {str(membership['admin_option']).lower()}",
            f"          AND am.inherit_option = {str(membership['inherit_option']).lower()}",
            f"          AND am.set_option = {str(membership['set_option']).lower()}",
            f"    ), 'G3 membership mismatch: {membership['member']} -> {membership['role_of']}';",
        ])
    lines.extend([
        "END",
        "$g3_validate$;",
        "COMMIT;",
    ])
    return sql_bytes(lines)


def render_create_table(name: str, spec: dict) -> list[str]:
    definitions = [
        f"    {col_name} {col_type}{(' ' + extras) if extras else ''}"
        for col_name, col_type, extras in spec["columns"]
    ]
    definitions.extend(f"    {constraint}" for constraint in spec["constraints"])
    return [
        f"CREATE TABLE platform.{name} (",
        ",\n".join(definitions),
        ");",
    ]


def generate_platform_sql() -> bytes:
    lines = [
        "-- G3 minimal migration-control schema (four tables).",
        "-- Product-schema identity remains the signed G2 manifest; these objects",
        "-- are governed by CONTROL-SCHEMA-MANIFEST.json.",
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.",
        r"\set ON_ERROR_STOP on",
        "BEGIN;",
        f"DO $$ BEGIN ASSERT current_database() = '{POSTGRES_DB}', 'G3 platform bootstrap requires POSTGRES_DB={POSTGRES_DB}'; END $$;",
        "SET LOCAL ROLE clarityit_owner;",
        "CREATE SCHEMA platform AUTHORIZATION clarityit_owner;",
        "REVOKE ALL ON SCHEMA platform FROM PUBLIC;",
        "REVOKE ALL ON SCHEMA platform FROM clarityit_app;",
        "",
    ]
    for name in ("source_profiles", "schema_revisions", "migration_runs", "reconciliation_results"):
        lines.extend(render_create_table(name, CONTROL_TABLES[name]))
        for index in CONTROL_TABLES[name].get("indexes", ()):
            lines.append(index + ";")
        lines.append("")

    lines.extend([
        "CREATE FUNCTION platform.protect_succeeded_revision() RETURNS trigger",
        "LANGUAGE plpgsql AS $function$",
        "BEGIN",
        "    IF TG_OP = 'DELETE' OR OLD.success THEN",
        "        RAISE EXCEPTION 'successful schema revision is immutable';",
        "    END IF;",
        "    RETURN NEW;",
        "END;",
        "$function$;",
        "CREATE TRIGGER schema_revisions_immutable",
        "BEFORE UPDATE OR DELETE ON platform.schema_revisions",
        "FOR EACH ROW EXECUTE FUNCTION platform.protect_succeeded_revision();",
        "",
        "CREATE FUNCTION platform.reject_reconciliation_mutation() RETURNS trigger",
        "LANGUAGE plpgsql AS $function$",
        "BEGIN",
        "    RAISE EXCEPTION 'reconciliation result is append-only';",
        "    RETURN NULL;",
        "END;",
        "$function$;",
        "CREATE TRIGGER reconciliation_results_append_only",
        "BEFORE UPDATE OR DELETE ON platform.reconciliation_results",
        "FOR EACH ROW EXECUTE FUNCTION platform.reject_reconciliation_mutation();",
        "",
        "REVOKE ALL ON ALL TABLES IN SCHEMA platform FROM PUBLIC, clarityit_app;",
        "REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA platform FROM PUBLIC, clarityit_app;",
        "ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA platform REVOKE ALL ON TABLES FROM PUBLIC;",
        "ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA platform REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;",
        "COMMIT;",
    ])
    return sql_bytes(lines)


def app_functions(manifest: dict) -> list[dict]:
    signatures = {
        (grant["schema"], grant["name"], grant["args"])
        for grant in manifest["target_grants"]["application_functions"]
    }
    matched = [
        function for function in manifest["functions"]
        if (function["schema"], function["name"], function["args"]) in signatures
    ]
    matched_signatures = {(f["schema"], f["name"], f["args"]) for f in matched}
    if matched_signatures != signatures or len(matched) != len(signatures):
        raise RuntimeError("application-function inventory does not bind one-to-one to function bodies")
    return sorted(matched, key=lambda f: (f["schema"], f["name"], f["args"]))


def generate_baseline_sql(manifest: dict) -> bytes:
    lines = [
        "-- G3 reconciled clean-install product baseline.",
        f"-- Signed G2 product manifest: sha256 {G2_MANIFEST_SHA256}, {G2_MANIFEST_SIZE} bytes.",
        "-- Deterministic: no data, timestamps, credentials, or environment owners.",
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.",
        r"\set ON_ERROR_STOP on",
        "BEGIN;",
        f"DO $$ BEGIN ASSERT current_database() = '{POSTGRES_DB}', 'G3 baseline requires POSTGRES_DB={POSTGRES_DB}'; END $$;",
        "",
        "-- Extension functions retain their installer-default PUBLIC ACL.",
    ]
    for extension in REQUIRED_EXTENSIONS:
        lines.append(f"CREATE EXTENSION IF NOT EXISTS {extension} WITH SCHEMA public;")
    lines.extend([
        "",
        "-- Every product object below is owned by the signed NOLOGIN owner role.",
        "SET LOCAL ROLE clarityit_owner;",
        # Keep pg_catalog first so unqualified built-ins (notably
        # gen_random_uuid()) bind to the same objects as the signed G2 source.
        "SET LOCAL search_path = pg_catalog, public;",
        "",
    ])

    for sequence in sorted(manifest["sequences"], key=lambda s: (s["schema"], s["name"])):
        cycle = " CYCLE" if sequence["cycle"] else " NO CYCLE"
        lines.append(
            f"CREATE SEQUENCE {sequence['schema']}.{sequence['name']} AS {sequence['type']} "
            f"INCREMENT BY {sequence['increment']} MINVALUE {sequence['min']} "
            f"MAXVALUE {sequence['max']} START WITH {sequence['start']} "
            f"CACHE {sequence['cache']}{cycle};"
        )
    lines.append("")

    lines.append("-- The signed inventory identifies exactly ten application functions.")
    for function in app_functions(manifest):
        body = function["body"].rstrip()
        lines.append(body if body.endswith(";") else body + ";")
    lines.append("")

    table_names = sorted(manifest["tables"])
    for key in table_names:
        schema, name = key.split(".", 1)
        table = manifest["tables"][key]
        definitions = []
        for column in sorted(table["columns"], key=lambda c: c["name"]):
            default = f" DEFAULT {column['default']}" if column.get("default") is not None else ""
            not_null = " NOT NULL" if column["not_null"] else ""
            definitions.append(f"    {column['name']} {column['type']}{default}{not_null}")
        for kind in ("p", "u", "c"):
            for constraint in sorted(
                (c for c in table["constraints"] if c["type"] == kind),
                key=lambda c: c["name"],
            ):
                definitions.append(
                    f"    CONSTRAINT {constraint['name']} {constraint['definition']}"
                )
        lines.extend([
            f"CREATE TABLE {schema}.{name} (",
            ",\n".join(definitions),
            ");",
        ])
        constraint_indexes = {
            c["name"] for c in table["constraints"] if c["type"] in {"p", "u"}
        }
        for index in sorted(table["indexes"], key=lambda item: item["name"]):
            if index["primary"] or index["name"] in constraint_indexes:
                continue
            lines.append(index["definition"] + ";")
        for trigger in sorted(table["triggers"], key=lambda item: item["name"]):
            lines.append(trigger["definition"] + ";")
        lines.append("")

    lines.append("-- Foreign keys are deferred until every referenced table exists.")
    for key in table_names:
        schema, name = key.split(".", 1)
        foreign_keys = sorted(
            (c for c in manifest["tables"][key]["constraints"] if c["type"] == "f"),
            key=lambda c: c["name"],
        )
        for constraint in foreign_keys:
            lines.append(
                f"ALTER TABLE {schema}.{name} ADD CONSTRAINT {constraint['name']} "
                f"{constraint['definition']};"
            )
    lines.append("")

    grants = manifest["target_grants"]
    for grant in grants["tables"]:
        lines.append(
            f"GRANT {', '.join(grant['privileges'])} ON TABLE {grant['schema']}.{grant['name']} "
            f"TO {grant['grantee']};"
        )
    for grant in grants["sequences"]:
        lines.append(
            f"GRANT {', '.join(grant['privileges'])} ON SEQUENCE {grant['schema']}.{grant['name']} "
            f"TO {grant['grantee']};"
        )
    for grant in grants["application_functions"]:
        lines.append(grant["public_revoke_sql"] + ";")
        for recipient in grant["grant_to"]:
            lines.append(recipient["grant_sql"] + ";")
    lines.extend(["COMMIT;"])
    return sql_bytes(lines)


def permission_uuid(name: str) -> str:
    namespace = uuid.UUID("8f22c14f-62c4-5e1b-82d5-64abf8ae1000")
    return str(uuid.uuid5(namespace, f"clarityit:g3:permission:{name}"))


def generate_seed_sql(baseline_sha: str) -> bytes:
    lines = [
        "-- G3 deterministic seed contract: seven G2-canonical permission names",
        "-- plus the immutable baseline revision record. No business/sample data.",
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.",
        r"\set ON_ERROR_STOP on",
        "BEGIN;",
        f"DO $$ BEGIN ASSERT current_database() = '{POSTGRES_DB}', 'G3 seed requires POSTGRES_DB={POSTGRES_DB}'; END $$;",
        "SET LOCAL ROLE clarityit_owner;",
        "INSERT INTO public.permissions (id, name, description, resource, action, risk_level, created_at) VALUES",
    ]
    rows = []
    for name, description, resource, action, risk in CANONICAL_PERMISSIONS:
        escaped_description = description.replace("'", "''")
        rows.append(
            f"    ('{permission_uuid(name)}', '{name}', '{escaped_description}', "
            f"'{resource}', '{action}', '{risk}', '2026-08-02T00:00:00Z')"
        )
    lines.append(",\n".join(rows) + ";")
    lines.extend([
        "DO $$",
        "BEGIN",
        "    ASSERT NOT EXISTS (SELECT 1 FROM public.permissions WHERE name LIKE '%.edit%'), 'G3 seed contains legacy .edit permission';",
        "    ASSERT (SELECT count(*) FROM public.permissions WHERE name IN ('work.items.update.own','work.items.update.any','projects.update','incidents.update.own','incidents.update.any','docs.update.own','docs.update.any')) = 7, 'G3 canonical permission set incomplete';",
        "END",
        "$$;",
        "INSERT INTO platform.schema_revisions (version, name, checksum, source_commit, applied_at, applied_by, execution_ms, success)",
        f"VALUES ('{BASELINE_VERSION}', 'g3-reconciled-baseline', '{baseline_sha}', '{FROZEN_G2_COMMIT}', '2026-08-02T00:00:00Z', 'g3-baseline-artifact', 0, true);",
        "COMMIT;",
    ])
    return sql_bytes(lines)


def _platform_statements() -> list[str]:
    """The platform-control DDL statements, without the BEGIN/COMMIT wrapper.

    Used by both the fresh bootstrap (``generate_platform_sql`` owns its
    own transaction) and the adoption artifact (which wraps these in its
    single atomic transaction).  The statements are identical so the
    platform bytes are preserved; only the transaction boundary differs.
    """
    lines = [
        f"DO $$ BEGIN ASSERT current_database() = '{POSTGRES_DB}', 'G3 platform bootstrap requires POSTGRES_DB={POSTGRES_DB}'; END $$;",
        "SET LOCAL ROLE clarityit_owner;",
        "CREATE SCHEMA platform AUTHORIZATION clarityit_owner;",
        "REVOKE ALL ON SCHEMA platform FROM PUBLIC;",
        "REVOKE ALL ON SCHEMA platform FROM clarityit_app;",
        "",
    ]
    for name in ("source_profiles", "schema_revisions", "migration_runs", "reconciliation_results"):
        lines.extend(render_create_table(name, CONTROL_TABLES[name]))
        for index in CONTROL_TABLES[name].get("indexes", ()):
            lines.append(index + ";")
        lines.append("")
    lines.extend([
        "CREATE FUNCTION platform.protect_succeeded_revision() RETURNS trigger",
        "LANGUAGE plpgsql AS $function$",
        "BEGIN",
        "    IF TG_OP = 'DELETE' OR OLD.success THEN",
        "        RAISE EXCEPTION 'successful schema revision is immutable';",
        "    END IF;",
        "    RETURN NEW;",
        "END;",
        "$function$;",
        "CREATE TRIGGER schema_revisions_immutable",
        "BEFORE UPDATE OR DELETE ON platform.schema_revisions",
        "FOR EACH ROW EXECUTE FUNCTION platform.protect_succeeded_revision();",
        "",
        "CREATE FUNCTION platform.reject_reconciliation_mutation() RETURNS trigger",
        "LANGUAGE plpgsql AS $function$",
        "BEGIN",
        "    RAISE EXCEPTION 'reconciliation result is append-only';",
        "    RETURN NULL;",
        "END;",
        "$function$;",
        "CREATE TRIGGER reconciliation_results_append_only",
        "BEFORE UPDATE OR DELETE ON platform.reconciliation_results",
        "FOR EACH ROW EXECUTE FUNCTION platform.reject_reconciliation_mutation();",
        "",
        "REVOKE ALL ON ALL TABLES IN SCHEMA platform FROM PUBLIC, clarityit_app;",
        "REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA platform FROM PUBLIC, clarityit_app;",
        "ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA platform REVOKE ALL ON TABLES FROM PUBLIC;",
        "ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner IN SCHEMA platform REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;",
    ])
    return lines


def generate_adoption_sql(manifest: dict, baseline_sha: str) -> bytes:
    """Generate the deterministic P3 approved-source adoption artifact.

    Adoption reconciles an existing P3 source database (whose 64 product
    tables already conform to the signed G2 shape) to the governed target
    posture: it creates the five target roles, transfers ownership of all
    product objects + the database + public schema to ``clarityit_owner``,
    installs the ``platform`` control schema, applies the signed grants and
    default privileges, seeds the seven canonical permissions and the
    adoption ledger rows, and demotes the application login to its signed
    non-superuser posture.  It performs no legacy-replay, no product-table
    creation, and no mutation of pre-existing P3 business rows; the only
    product-row write is the seven canonical permission inserts.

    Everything runs in a single transaction.  The bootstrap superuser
    ``clarityit`` owns the extensions, so it is renamed to a fixed
    ``NOLOGIN`` legacy-extension owner before the signed target
    ``clarityit`` is created.  PostgreSQL prohibits renaming the current
    session user, so a temporary ``NOLOGIN SUPERUSER`` administrator is
    created, session authorization is switched to it for the rename, and
    it is dropped before commit.  The final ``clarityit`` demotion is the
    transaction's last state mutation, after all privileged operations and
    assertions.
    """
    role_by_name = {role["name"]: role for role in manifest["target_roles"]}

    def role_options(role: dict) -> str:
        f = role["flags"]
        return " ".join([
            "LOGIN" if f["canlogin"] else "NOLOGIN",
            "INHERIT" if f["inherit"] else "NOINHERIT",
            "CREATEDB" if f["createdb"] else "NOCREATEDB",
            "CREATEROLE" if f["createrole"] else "NOCREATEROLE",
            "NOSUPERUSER", "NOREPLICATION", "NOBYPASSRLS",
        ])

    def target_role_flags_clause(role: dict) -> str:
        f = role["flags"]
        return " ".join([
            "LOGIN" if f["canlogin"] else "NOLOGIN",
            "INHERIT" if f["inherit"] else "NOINHERIT",
            "CREATEDB" if f["createdb"] else "NOCREATEDB",
            "CREATEROLE" if f["createrole"] else "NOCREATEROLE",
            "SUPERUSER" if f["superuser"] else "NOSUPERUSER",
            "REPLICATION" if f["replication"] else "NOREPLICATION",
            "BYPASSRLS" if f["bypassrls"] else "NOBYPASSRLS",
        ])

    table_names = list(manifest["tables"].keys())
    app_grants = manifest["target_grants"]["application_functions"]
    app_signatures = [(g["schema"], g["name"], g["args"]) for g in app_grants]

    lines = [
        "-- G3 deterministic P3 approved-source adoption artifact.",
        "-- Reconciles an existing P3 source to the signed G2 governed posture.",
        "-- No legacy replay, no product-table creation, no business-row mutation.",
        "-- The only product-row write is the seven canonical permission inserts.",
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.",
        r"\set ON_ERROR_STOP on",
        "BEGIN;",
        "",
        "-- Bind the adoption producing commit at execution time (runtime-bound,",
        "-- not a fingerprint).  The proof harness passes the exact implementation",
        "-- commit SHA via the g3_source_commit psql variable.",
        "SELECT set_config('g3.source_commit', :'g3_source_commit', true);",
        "DO $$ BEGIN ASSERT current_setting('g3.source_commit', true) ~ '^[0-9a-f]{40}$',",
        "    'g3.source_commit must be set to a 40-char lowercase hex SHA'; END $$;",
        "",
        "-- ============================================================",
        "-- Preflight (read-only): the source must be the approved P3 shape.",
        "-- ============================================================",
        f"DO $g3_adopt_preflight$",
        "BEGIN",
        f"    IF current_database() <> '{POSTGRES_DB}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption requires database {POSTGRES_DB}, got %', current_database();",
        "    END IF;",
        "    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN",
        "        RAISE EXCEPTION 'G3 adoption requires a PostgreSQL superuser';",
        "    END IF;",
        f"    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN",
        "        RAISE EXCEPTION 'G3 adoption requires extension pgcrypto';",
        "    END IF;",
        f"    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'citext') THEN",
        "        RAISE EXCEPTION 'G3 adoption requires extension citext';",
        "    END IF;",
        f"    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN",
        "        RAISE EXCEPTION 'G3 adoption requires extension pg_trgm';",
        "    END IF;",
        "    -- clarityit must exist as the P3 bootstrap superuser owning the extensions.",
        "    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND rolsuper) THEN",
        "        RAISE EXCEPTION 'G3 adoption requires the P3 bootstrap superuser clarityit';",
        "    END IF;",
        "    IF (SELECT count(DISTINCT e.extname) FROM pg_extension e JOIN pg_roles r ON r.oid = e.extowner",
        "        WHERE e.extname IN ('pgcrypto','citext','pg_trgm') AND r.rolname = 'clarityit') <> 3 THEN",
        "        RAISE EXCEPTION 'G3 adoption requires clarityit to own pgcrypto, citext, and pg_trgm';",
        "    END IF;",
        f"    -- SQL-derived structural digests: fail-closed against drift, computed",
        f"    -- from the live catalog (NOT caller-supplied).  Each covers the full",
        f"    -- property set so body/default/nullability/identity/constraint-name/cache",
        f"    -- changes all fail the gate.",
        f"    -- Column digest: name, type, NOT NULL, default, identity, charlen,",
        f"    -- precision, scale, collation, generated.",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        format('%s.%s.%s|notnull:%s|default:%s|identity:%s|charlen:%s|precision:%s|scale:%s|collation:%s|generated:%s',",
        f"        table_name, column_name, data_type, is_nullable,",
        f"        COALESCE(column_default, ''), COALESCE(is_identity, ''),",
        f"        COALESCE(character_maximum_length::text, ''),",
        f"        COALESCE(numeric_precision::text, ''),",
        f"        COALESCE(numeric_scale::text, ''),",
        f"        COALESCE(collation_name, ''),",
        f"        COALESCE(is_generated, '')), E'\\n'",
        f"        ORDER BY table_name, column_name), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM information_schema.columns WHERE table_schema='public')",
        f"        <> '{P3_COLUMN_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source column properties drifted (digest mismatch)';",
        f"    END IF;",
        f"    -- Application-function signature digest (explicit ORDER BY columns).",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        format('%s.%s(%s)', n.nspname, p.proname,",
        f"        pg_get_function_identity_arguments(p.oid)), E'\\n'",
        f"        ORDER BY n.nspname, p.proname, pg_get_function_identity_arguments(p.oid)), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace",
        f"        WHERE n.nspname='public' AND p.prokind='f' AND p.proname = ANY(ARRAY[{','.join(repr(n) for n in P3_APPFN_NAMES)}]::text[]))",
        f"        <> '{P3_APPFN_SIG_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source application-function signatures drifted (digest mismatch)';",
        f"    END IF;",
        f"    -- Application-function BODY digest (full pg_get_functiondef).",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        pg_get_functiondef(p.oid), E'\\n' ORDER BY n.nspname, p.proname), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace",
        f"        WHERE n.nspname='public' AND p.prokind='f' AND p.proname = ANY(ARRAY[{','.join(repr(n) for n in P3_APPFN_NAMES)}]::text[]))",
        f"        <> '{P3_APPFN_BODY_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source application-function bodies drifted (digest mismatch)';",
        f"    END IF;",
        f"    -- Index digest.",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        pg_get_indexdef(ix.indexrelid, 0, true), E'\\n'",
        f"        ORDER BY n.nspname, c.relname, i.relname), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM pg_index ix JOIN pg_class c ON c.oid=ix.indrelid",
        f"        JOIN pg_class i ON i.oid=ix.indexrelid",
        f"        JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public')",
        f"        <> '{P3_INDEX_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source index inventory drifted (digest mismatch)';",
        f"    END IF;",
        f"    -- Trigger digest.",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        pg_get_triggerdef(t.oid, true), E'\\n'",
        f"        ORDER BY n.nspname, c.relname, t.tgname), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid",
        f"        JOIN pg_namespace n ON n.oid=c.relnamespace",
        f"        WHERE n.nspname='public' AND NOT t.tgisinternal)",
        f"        <> '{P3_TRIGGER_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source trigger inventory drifted (digest mismatch)';",
        f"    END IF;",
        f"    -- Sequence digest: type, start, increment, min, max, cache, cycle, owner.",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        format('%s.%s|type:%s|start:%s|inc:%s|min:%s|max:%s|cache:%s|cycle:%s|owner:%s',",
        f"        n.nspname, c.relname, s.seqtypid::regtype, s.seqstart, s.seqincrement,",
        f"        s.seqmin, s.seqmax, s.seqcache, s.seqcycle,",
        f"        pg_get_userbyid(c.relowner)), E'\\n'",
        f"        ORDER BY n.nspname, c.relname), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM pg_sequence s JOIN pg_class c ON c.oid=s.seqrelid",
        f"        JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public')",
        f"        <> '{P3_SEQUENCE_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source sequence properties drifted (digest mismatch)';",
        f"    END IF;",
        f"    -- Constraint digest: name + definition (all 287, names included).",
        f"    IF (SELECT encode(public.digest(convert_to(string_agg(",
        f"        format('%s.%s.%s|%s', n.nspname, c.relname, con.conname,",
        f"        pg_get_constraintdef(con.oid, true)), E'\\n'",
        f"        ORDER BY n.nspname, c.relname, con.conname), 'UTF8'), 'sha256'), 'hex')",
        f"        FROM pg_constraint con JOIN pg_class c ON c.oid=con.conrelid",
        f"        JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind='r')",
        f"        <> '{P3_CONSTRAINT_DIGEST}' THEN",
        f"        RAISE EXCEPTION 'G3 adoption source constraints drifted (digest mismatch)';",
        f"    END IF;",
        "    -- Target identities must be absent (single-shot adoption).",
        "    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN (",
        "        'clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin',",
        "        'legacy_ext_owner','g3_adopt_admin')) THEN",
        "        RAISE EXCEPTION 'G3 adoption is single-shot: a target/legacy identity already exists';",
        "    END IF;",
        "    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'platform') THEN",
        "        RAISE EXCEPTION 'G3 adoption requires no existing platform schema';",
        "    END IF;",
    ]
    # Assert the 64 product tables + 1 sequence exist (shape conformance is
    # verified by the pre_adopt_verify Python check; here we assert presence).
    for key in table_names:
        schema, name = key.split(".", 1)
        lines.append(
            f"    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace "
            f"WHERE n.nspname='{schema}' AND c.relname='{name}' AND c.relkind='r') THEN"
        )
        lines.append(f"        RAISE EXCEPTION 'G3 adoption requires product table {key}';")
        lines.append("    END IF;")
    lines.extend([
        "END",
        "$g3_adopt_preflight$;",
        "",
        "-- ============================================================",
        "-- Role transition (atomic, inside this transaction).",
        "-- The bootstrap clarityit owns the extensions; rename it to a fixed",
        "-- NOLOGIN legacy-extension owner, then create the signed target",
        "-- clarityit.  PostgreSQL forbids renaming the current session user,",
        "-- so switch session authorization to a temporary administrator.",
        "-- ============================================================",
        "CREATE ROLE g3_adopt_admin NOLOGIN SUPERUSER;",
        "SET SESSION AUTHORIZATION g3_adopt_admin;",
        "ALTER ROLE clarityit RENAME TO legacy_ext_owner;",
        "ALTER ROLE legacy_ext_owner NOLOGIN;",
        f"CREATE ROLE clarityit {role_options(role_by_name['clarityit'])};",
    ])
    # Create the other four target roles.
    for name in ("clarityit_app", "clarityit_owner", "clarityit_migrator", "clarityit_admin"):
        lines.append(f"CREATE ROLE {name} {role_options(role_by_name[name])};")
    lines.append("")
    # Memberships.
    for membership in manifest["target_memberships"]:
        lines.append(membership["sql"] + ";")
    lines.extend([
        "-- Transfer database ownership to clarityit_owner BEFORE platform",
        "-- creation.  This gives clarityit_owner the CREATE privilege on the",
        "-- database as its owner, so SET LOCAL ROLE clarityit_owner can",
        "-- create the platform schema without any temporary GRANT CREATE",
        "-- (which would leave an unnecessary database ACL difference).",
        f"ALTER DATABASE {POSTGRES_DB} OWNER TO clarityit_owner;",
        "",
        "-- ============================================================",
        "-- Platform control schema (same statements as the fresh bootstrap,",
        "-- rendered without its own BEGIN/COMMIT so the adoption stays atomic).",
        "-- ============================================================",
    ])
    lines.extend(_platform_statements())
    lines.extend([
        "RESET ROLE;",
        "",
        "-- ============================================================",
        "-- Ownership transfer: product objects + public schema to",
        "-- clarityit_owner.  Idempotent (a no-op if already owned).",
        "-- (Database ownership was transferred above, before platform.)",
        "-- ============================================================",
        "ALTER SCHEMA public OWNER TO clarityit_owner;",
    ])
    for key in table_names:
        schema, name = key.split(".", 1)
        lines.append(f"ALTER TABLE {schema}.{name} OWNER TO clarityit_owner;")
    for seq in manifest["sequences"]:
        lines.append(f"ALTER SEQUENCE {seq['schema']}.{seq['name']} OWNER TO clarityit_owner;")
    for sig in app_signatures:
        schema, name, args = sig
        lines.append(f"ALTER FUNCTION {schema}.{name}({args}) OWNER TO clarityit_owner;")
    lines.extend([
        "",
        "-- ============================================================",
        "-- Signed grants (idempotent) and default privileges.",
        "-- ============================================================",
        "REVOKE CREATE ON SCHEMA public FROM PUBLIC;",
        "GRANT USAGE ON SCHEMA public TO clarityit_app;",
    ])
    # Table grants.
    for grant in manifest["target_grants"]["tables"]:
        lines.append(
            f"GRANT {', '.join(grant['privileges'])} ON TABLE {grant['schema']}.{grant['name']} "
            f"TO {grant['grantee']};"
        )
    # Sequence grants.
    for grant in manifest["target_grants"]["sequences"]:
        lines.append(
            f"GRANT {', '.join(grant['privileges'])} ON SEQUENCE {grant['schema']}.{grant['name']} "
            f"TO {grant['grantee']};"
        )
    # Application-function per-signature revoke+grant (NOT bulk; preserve ext ACL).
    for grant in app_grants:
        lines.append(grant["public_revoke_sql"] + ";")
        for recipient in grant["grant_to"]:
            lines.append(recipient["grant_sql"] + ";")
    lines.append("")
    # Default privileges (public schema).
    for default in manifest["target_default_privileges"]:
        privileges = ", ".join(default["privileges"])
        lines.append(
            f"ALTER DEFAULT PRIVILEGES FOR ROLE {default['creator']} "
            f"IN SCHEMA {default['schema']} GRANT {privileges} ON {default['object_type']} "
            f"TO {default['grantee']};"
        )
    for revoke in manifest["target_grants"]["default_privileges_public_revoke"]:
        if revoke["action"] != "REVOKE EXECUTE FROM PUBLIC":
            raise RuntimeError("unsupported G2 default-privilege PUBLIC revoke")
        lines.append(
            f"ALTER DEFAULT PRIVILEGES FOR ROLE {revoke['creator']} "
            f"IN SCHEMA {revoke['schema']} REVOKE EXECUTE ON {revoke['object_type']} FROM PUBLIC;"
        )
    lines.extend([
        "",
        "-- ============================================================",
        "-- Seed + adoption ledger.  Performed before the final role transition.",
        "-- No pre-existing P3 business rows are mutated.",
        "-- ============================================================",
        "SET LOCAL ROLE clarityit_owner;",
    ])
    # Seven canonical permissions.
    perm_rows = []
    for name, description, resource, action, risk in CANONICAL_PERMISSIONS:
        escaped = description.replace("'", "''")
        perm_rows.append(
            f"    ('{permission_uuid(name)}', '{name}', '{escaped}', "
            f"'{resource}', '{action}', '{risk}', '2026-08-02T00:00:00Z')"
        )
    lines.append("INSERT INTO public.permissions (id, name, description, resource, action, risk_level, created_at) VALUES")
    lines.append(",\n".join(perm_rows) + ";")
    lines.extend([
        "DO $$",
        "BEGIN",
        "    ASSERT NOT EXISTS (SELECT 1 FROM public.permissions WHERE name LIKE '%.edit%'), 'G3 seed contains legacy .edit permission';",
        "    ASSERT (SELECT count(*) FROM public.permissions WHERE name IN ('work.items.update.own','work.items.update.any','projects.update','incidents.update.own','incidents.update.any','docs.update.own','docs.update.any')) = 7, 'G3 canonical permission set incomplete';",
        "END",
        "$$;",
        f"INSERT INTO platform.source_profiles (profile_id, schema_fingerprint, postgres_version, postgres_major, extensions, roles_digest, source_commit, approved_by, approved_at) VALUES (",
        f"    '{P3_PROFILE_ID}', '{P3_GOLDEN_FINGERPRINT}', 'PostgreSQL 16', 16, "
        f"'[\"pgcrypto\",\"citext\",\"pg_trgm\"]'::jsonb, "
        f"'{roles_digest_for_manifest(manifest)}', '{P3_SOURCE_COMMIT}', '{G1_APPROVAL_REF}', '{G3_ARTIFACT_DATE}');",
        "INSERT INTO platform.schema_revisions (version, name, checksum, source_commit, applied_at, applied_by, execution_ms, success)",
        f"VALUES ('{BASELINE_VERSION}', 'adopt-p3', '{baseline_sha}', current_setting('g3.source_commit', true), '{G3_ARTIFACT_DATE}', 'g3-adoption-artifact', 0, true);",
        "RESET ROLE;",
        "",
        "-- ============================================================",
        "-- End of privileged operations.  Reset session authorization from",
        "-- the temporary administrator back to the original session identity,",
        "-- then drop the temporary administrator.  This occurs AFTER seed and",
        "-- ledger insertion and BEFORE the final role transition, so the",
        "-- temporary identity exists only while privileged operations are",
        "-- still in progress.",
        "-- ============================================================",
        "RESET SESSION AUTHORIZATION;",
        "DROP ROLE g3_adopt_admin;",
        "",
        "-- ============================================================",
        "-- Final bootstrap-role transition (LAST state mutation).",
        "-- Demote the new clarityit to its signed non-superuser target posture",
        "-- only after every privileged operation and assertion has succeeded.",
        "-- ============================================================",
        f"ALTER ROLE clarityit {target_role_flags_clause(role_by_name['clarityit'])};",
        "",
        "-- Final read-only assertions.",
        "DO $g3_adopt_validate$",
        "BEGIN",
        "    ASSERT (SELECT count(*) FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')) = 5, 'G3 adoption role count mismatch';",
        "    ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'legacy_ext_owner' AND rolcanlogin), 'legacy_ext_owner must be NOLOGIN';",
        "    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND NOT rolsuper), 'clarityit must be demoted to NOSUPERUSER';",
        "    ASSERT (SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = current_database()) = 'clarityit_owner', 'database must be owned by clarityit_owner';",
        "    ASSERT NOT EXISTS (",
        "        SELECT 1 FROM pg_extension e JOIN pg_roles r ON r.oid = e.extowner",
        "        WHERE e.extname IN ('pgcrypto','citext','pg_trgm') AND r.rolname IN ('clarityit','clarityit_owner')),"
        "        'no extension may be owned by a target role';",
        "END",
        "$g3_adopt_validate$;",
        "COMMIT;",
    ])
    return sql_bytes(lines)


def roles_digest_for_manifest(manifest: dict) -> str:
    """Deterministic roles_digest for the source_profiles row."""
    import hashlib as _hs
    role_rows = sorted(
        ({"name": r["name"], "flags": r["flags"]} for r in manifest["target_roles"]),
        key=lambda x: x["name"],
    )
    membership_rows = sorted(
        ({
            "member": m["member"], "role_of": m["role_of"],
            "admin_option": m["admin_option"], "inherit_option": m["inherit_option"],
            "set_option": m["set_option"],
        } for m in manifest["target_memberships"]),
        key=lambda x: (x["member"], x["role_of"]),
    )
    payload = [
        json.dumps(role_rows, sort_keys=True, ensure_ascii=True, separators=(",", ":")),
        json.dumps(membership_rows, sort_keys=True, ensure_ascii=True, separators=(",", ":")),
    ]
    return _hs.sha256(json.dumps(payload, sort_keys=True, ensure_ascii=True, separators=(",", ":")).encode("utf-8")).hexdigest()


def generate_control_manifest(platform_sql: bytes) -> bytes:
    value = {
        "format_version": 1,
        "description": "G3 platform migration-control schema; separate from signed G2 product identity.",
        "schema": "platform",
        "owner": "clarityit_owner",
        "tables": {
            name: {
                "columns": [
                    {"name": c[0], "type": c[1], "sql_suffix": c[2]}
                    for c in spec["columns"]
                ],
                "constraints": list(spec["constraints"]),
                "indexes": list(spec.get("indexes", ())),
            }
            for name, spec in sorted(CONTROL_TABLES.items())
        },
        "functions": [
            "platform.protect_succeeded_revision()",
            "platform.reject_reconciliation_mutation()",
        ],
        "triggers": [
            "platform.schema_revisions.schema_revisions_immutable",
            "platform.reconciliation_results.reconciliation_results_append_only",
        ],
        "runtime_grants": [],
        "platform_sql_sha256": sha256(platform_sql),
    }
    return json_bytes(value)


def composite_digest(components: list[tuple[str, bytes]]) -> str:
    digest = hashlib.sha256()
    digest.update(b"clarityit-g3-composite-v1\0")
    for label, data in components:
        label_bytes = label.encode("utf-8")
        digest.update(len(label_bytes).to_bytes(4, "big"))
        digest.update(label_bytes)
        digest.update(len(data).to_bytes(8, "big"))
        digest.update(data)
    return digest.hexdigest()


def expected_files() -> tuple[dict[Path, bytes], dict]:
    manifest, _ = validate_frozen_inputs()
    files: dict[Path, bytes] = {}

    legacy_lines = [
        "# G3 ordered legacy provenance inventory: signed G2 Git blobs, 001-040.",
        "# These files are evidence only and MUST NOT be selected for execution.",
    ]
    for source in legacy_paths():
        data = git_blob(FROZEN_G2_COMMIT, source)
        destination = LEGACY_DIR / Path(source).name
        files[destination] = data
        legacy_lines.append(f"{sha256(data)}  001-040/{destination.name}")
    files[LEGACY_SUMS] = sql_bytes(legacy_lines)

    files[ROLES_SQL] = generate_roles_sql(manifest)
    files[PLATFORM_SQL] = generate_platform_sql()
    files[BASELINE_SQL] = generate_baseline_sql(manifest)
    files[SEED_SQL] = generate_seed_sql(sha256(files[BASELINE_SQL]))
    files[CONTROL_MANIFEST] = generate_control_manifest(files[PLATFORM_SQL])
    files[ADOPTION_SQL] = generate_adoption_sql(manifest, sha256(files[BASELINE_SQL]))

    composite_components = [
        ("product_manifest_blob_sha256", G2_MANIFEST_SHA256.encode("ascii")),
        ("control_manifest", files[CONTROL_MANIFEST]),
        ("baseline_sql", files[BASELINE_SQL]),
        ("seed_sql", files[SEED_SQL]),
        ("role_bootstrap_sql", files[ROLES_SQL]),
        ("legacy_checksum_inventory", files[LEGACY_SUMS]),
    ]
    installation_sha = composite_digest(composite_components)
    a4 = {
        "format_version": 1,
        "artifact": "ClarityIT WP-00 G3 reconciled baseline (A4)",
        "generator": GENERATOR_VERSION,
        "frozen_g2": {
            "commit": FROZEN_G2_COMMIT,
            "manifest_path": G2_MANIFEST_PATH,
            "manifest_blob_sha256": G2_MANIFEST_SHA256,
            "manifest_blob_size": G2_MANIFEST_SIZE,
        },
        "database_name": POSTGRES_DB,
        "postgres_major": 16,
        "counts": {
            "legacy_migrations": 40,
            "product_tables": 64,
            "application_functions": 10,
            "platform_tables": 4,
            "canonical_permissions": 7,
        },
        "composite": {
            "algorithm": "SHA-256(domain + repeated uint32be(label_len), label, uint64be(data_len), data)",
            "domain": "clarityit-g3-composite-v1\\0",
            "component_order": [label for label, _ in composite_components],
            "sha256": installation_sha,
        },
        "components": {
            canonical_repo_path(path): {"sha256": sha256(data), "size": len(data)}
            for path, data in sorted(files.items(), key=lambda item: canonical_repo_path(item[0]))
            if path in {LEGACY_SUMS, ROLES_SQL, PLATFORM_SQL, BASELINE_SQL, SEED_SQL, CONTROL_MANIFEST}
        },
        "adoption": {
            "description": "P3 approved-source adoption artifact; distinct from fresh-install composite.",
            "p3_golden_fingerprint": P3_GOLDEN_FINGERPRINT,
            "p3_source_commit": P3_SOURCE_COMMIT,
            "g1_approval_ref": G1_APPROVAL_REF,
            "p3_profile_id": P3_PROFILE_ID,
            "baseline_checksum": sha256(files[BASELINE_SQL]),
            "adoption_sql_sha256": sha256(files[ADOPTION_SQL]),
            "adoption_sql_size": len(files[ADOPTION_SQL]),
            "p3_column_digest": P3_COLUMN_DIGEST,
            "p3_appfn_sig_digest": P3_APPFN_SIG_DIGEST,
            "p3_appfn_body_digest": P3_APPFN_BODY_DIGEST,
            "p3_constraint_digest": P3_CONSTRAINT_DIGEST,
            "p3_index_digest": P3_INDEX_DIGEST,
            "p3_trigger_digest": P3_TRIGGER_DIGEST,
            "p3_sequence_digest": P3_SEQUENCE_DIGEST,
        },
        "governed_fingerprint": {
            "algorithm": "clarityit-g3-governed-v1",
            "domain": "clarityit-g3-governed-v1\\0",
            "description": (
                "Deterministic projection over the signed G2 contract: governed "
                "product+platform objects, five target roles/memberships, "
                "closed-inventory grants (excluding grantor + extension objects), "
                "projected ownership (including database owner), effective default "
                "privileges, and the extension-owner invariant (boolean, not name). "
                "Fresh installs and P3-adopted databases converge on this fingerprint."
            ),
            "target_sha256": GOVERNED_TARGET_FINGERPRINT,
        },
    }
    files[A4_MANIFEST] = json_bytes(a4)

    checksum_paths = (
        LEGACY_SUMS, ROLES_SQL, PLATFORM_SQL, BASELINE_SQL,
        SEED_SQL, CONTROL_MANIFEST, A4_MANIFEST, ADOPTION_SQL,
    )
    checksum_lines = [
        "# G3 detached artifact checksums (repository-relative paths).",
        *[f"{sha256(files[path])}  {canonical_repo_path(path)}" for path in checksum_paths],
    ]
    files[V2_SUMS] = sql_bytes(checksum_lines)
    return files, a4


def write_files(files: dict[Path, bytes]) -> None:
    for relative, data in sorted(files.items(), key=lambda item: canonical_repo_path(item[0])):
        target = ROOT / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)


def check_files(files: dict[Path, bytes]) -> list[str]:
    failures = []
    for relative, expected in sorted(files.items(), key=lambda item: canonical_repo_path(item[0])):
        target = ROOT / relative
        if not target.exists():
            failures.append(f"missing: {relative}")
        elif target.read_bytes() != expected:
            failures.append(f"byte mismatch: {relative}")
    if (ROOT / LEGACY_DIR).exists():
        expected_archive = {str(path) for path in files if path.parent == LEGACY_DIR}
        actual_archive = {
            str(path.relative_to(ROOT))
            for path in (ROOT / LEGACY_DIR).iterdir()
            if path.is_file()
        }
        for extra in sorted(actual_archive - expected_archive):
            failures.append(f"unexpected legacy archive file: {extra}")
    return failures


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--write", action="store_true", help="write deterministic artifacts (default)")
    mode.add_argument("--check", action="store_true", help="fail if committed/generated bytes differ")
    mode.add_argument("--print-identity", action="store_true", help="print the composite installation identity")
    args = parser.parse_args()

    try:
        files, a4 = expected_files()
    except (KeyError, ValueError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"G3 generation failed: {exc}", file=sys.stderr)
        return 1

    if args.check:
        failures = check_files(files)
        if failures:
            print("G3 generated-artifact check failed:", file=sys.stderr)
            for failure in failures:
                print(f"  {failure}", file=sys.stderr)
            return 1
        print(f"BASELINE-GEN PASS: {len(files)} generated files match deterministic bytes")
    elif args.print_identity:
        print(a4["composite"]["sha256"])
    else:
        write_files(files)
        print(f"wrote {len(files)} deterministic G3 files")
        print(f"product identity   = {G2_MANIFEST_SHA256} ({G2_MANIFEST_SIZE} bytes)")
        print(f"control identity   = {sha256(files[CONTROL_MANIFEST])}")
        print(f"installation identity = {a4['composite']['sha256']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
