#!/usr/bin/env python3
"""
Generate the P3 sanitized CI fixture from a P1 manifest.

P3 is a DETERMINISTIC, ENTIRELY SYNTHETIC legacy database fixture that matches
P1's schema shape (tables, columns, types, constraints, indexes, functions,
triggers) plus minimal legacy-truth seed rows for later migration tests.

It contains NO production rows, identifiers, credentials, hostnames, or
infrastructure details. All seeded UUIDs/timestamps are fixed literals.

P3 represents the single captured P1 source shape. It does NOT resolve
migrations 016/018/029 and does NOT include competing 005/018-shaped variants.
Migration decisions belong to G2 after profile approval.

Usage:
    python generate_p3.py \\
        --p1-manifest /path/to/p1-production/manifest.json \\
        --output migrations/profiles/p3/

This produces:
    p3/schema.sql      — synthetic DDL matching P1 shape
    p3/seed.sql        — minimal legacy-truth seed rows (5 legacy-truth cases)

The golden manifest (golden-manifest.json) is NOT generated here. It is
produced by applying schema.sql + seed.sql to a fresh PostgreSQL 16.14
database and running the profiler (capture_schema.py). Use validate_p3.py
to build, capture, and verify the golden profile end-to-end.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys

# Import the capture profiler for fingerprint computation
_HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _HERE)
import importlib.util
_spec = importlib.util.spec_from_file_location(
    "capture_schema", os.path.join(_HERE, "capture_schema.py")
)
cs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cs)

# Extensions required by P1's function set (pgcrypto → armor/gen_random_uuid,
# citext, pg_trgm for tsvector). These are declared so function bodies resolve.
REQUIRED_EXTENSIONS = ["pgcrypto", "citext", "pg_trgm"]

# Fixed synthetic seed UUIDs (deterministic — not from production).
SEED_TEAM = "00000000-0000-0000-0000-000000000001"
SEED_USER = "00000000-0000-0000-0000-000000000002"
SEED_OBJECT = "00000000-0000-0000-0000-000000000003"
SEED_ASSET = "00000000-0000-0000-0000-000000000004"
SEED_ACTION = "00000000-0000-0000-0000-000000000005"
SEED_EFFECT = "00000000-0000-0000-0000-000000000006"
SEED_AGENT = "00000000-0000-0000-0000-000000000007"
SEED_APPROVAL = "00000000-0000-0000-0000-000000000008"
SEED_PROPOSAL = "00000000-0000-0000-0000-000000000009"
SEED_STEP = "00000000-0000-0000-0000-000000000010"
FIXED_TS = "2026-01-01T00:00:00Z"


def map_type(pg_type: str) -> str:
    """Pass through PG types as-is (they come from format_type, already canonical)."""
    return pg_type


def column_default(col: dict, table_name: str) -> str:
    """Reproduce a column's DEFAULT if it has one, sanitized."""
    d = col.get("default")
    if not d:
        return ""
    return f" DEFAULT {d}"


def generate_schema_ddl(manifest: dict) -> str:
    """Generate synthetic DDL matching P1's schema shape from its manifest."""
    lines = [
        "-- P3: ClarityIT sanitized CI legacy fixture (synthetic)",
        "-- Deterministic; generated from P1 manifest structural metadata.",
        "-- NO production data, identifiers, credentials, or hostnames.",
        "-- Deterministic: identical P1 input produces identical bytes.",
        "-- DO NOT EDIT BY HAND — regenerate via scripts/profile/generate_p3.py",
        "",
    ]

    # Extensions
    for ext in REQUIRED_EXTENSIONS:
        lines.append(f"CREATE EXTENSION IF NOT EXISTS {ext};")
    lines.append("")

    # Sequences (must exist before tables that reference them via nextval)
    seqs = [r for r in manifest["relations"] if r["kind"] == "S"]
    # Also check the sequences section of the manifest for properties
    seq_props = {f"{s['schema']}.{s['name']}": s for s in manifest.get("sequences", [])}
    for sq in sorted(seqs, key=lambda x: x["name"]):
        key = f"{sq['schema']}.{sq['name']}"
        props = seq_props.get(key, {})
        start = props.get("start", 1)
        incr = props.get("increment", 1)
        lines.append(
            f"CREATE SEQUENCE {sq['schema']}.{sq['name']} "
            f"AS bigint INCREMENT BY {incr} MINVALUE 1 MAXVALUE 9223372036854775807 START WITH {start};"
        )
    if seqs:
        lines.append("")

    # App-defined functions from P1 (extension functions come via CREATE EXTENSION;
    # only emit functions whose body pg_get_functiondef returned — these are the
    # app-defined ones needed by triggers and constraints).
    # Filter to public-schema functions that are NOT extension-provided
    # (extension functions are created by CREATE EXTENSION and would error if
    # we tried to redefine them).
    EXT_PROVIDED = {
        "armor", "dearmor", "pgp_sym_encrypt", "pgp_sym_decrypt",
        "pgp_pub_encrypt", "pgp_pub_decrypt", "gen_random_uuid",
        "gen_random_bytes", "digest", "hmac", "crypt", "gen_salt",
        "pgp_sym_encrypt_bytea", "pgp_sym_decrypt_bytea",
        "pgp_pub_encrypt_bytea", "pgp_pub_decrypt_bytea",
        "pgp_armor_headers", "pgp_key_id",
        "encrypt", "decrypt", "encrypt_iv", "decrypt_iv",
        # citext operators/functions
        "citext_eq", "citext_ne", "citext_lt", "citext_le", "citext_gt",
        "citext_ge", "citext_cmp", "citext_hash", "citext_hash_extended",
        "citext_larger", "citext_smaller", "citext_pattern_cmp",
        "citext_pattern_lt", "citext_pattern_le", "citext_pattern_gt",
        "citext_pattern_ge", "citext", "citext_in", "citext_out",
        "citext_recv", "citext_send", "citextin", "citextout",
        "citextrecv", "citextsend",
        # pg_trgm citext overloads
        "regexp_match", "regexp_matches", "regexp_replace",
        "regexp_split_to_array", "regexp_split_to_table",
        "replace", "split_part", "strpos",
        "texticlike", "texticnlike", "texticregexeq", "texticregexne",
        "translate", "similarity", "word_similarity",
        "show_trgm", "show_limit", "set_limit",
    }
    app_fns = [
        f for f in manifest["functions"]
        if f["schema"] == "public" and f["name"] not in EXT_PROVIDED
    ]
    lines.append("-- App-defined functions (extension functions come via CREATE EXTENSION)")
    for f in sorted(app_fns, key=lambda x: (x["name"], x["args"])):
        # The body from pg_get_functiondef is a full CREATE OR REPLACE FUNCTION statement
        lines.append(f["body"].strip() + ";")
    lines.append("")

    # Tables in deterministic order
    tables = sorted(
        [r for r in manifest["relations"] if r["kind"] == "r"],
        key=lambda r: r["name"],
    )

    for tbl in tables:
        key = f"{tbl['schema']}.{tbl['name']}"
        cols = manifest["columns"].get(key, [])
        col_defs = []
        for c in sorted(cols, key=lambda x: x["name"]):
            nn = " NOT NULL" if c.get("not_null") else ""
            dflt = column_default(c, tbl["name"])
            col_defs.append(f"    {c['name']} {map_type(c['type'])}{nn}{dflt}")

        # Constraints from manifest
        cons = manifest["constraints"].get(key, [])
        # Primary keys first, then unique, then checks (FKs added after all tables)
        pk = [c for c in cons if c["type"] == "p"]
        uq = [c for c in cons if c["type"] == "u"]
        ck = [c for c in cons if c["type"] == "c"]

        for c in pk:
            col_defs.append(f"    CONSTRAINT {c['name']} {c['definition']}")
        for c in uq:
            col_defs.append(f"    CONSTRAINT {c['name']} {c['definition']}")
        for c in ck:
            col_defs.append(f"    CONSTRAINT {c['name']} {c['definition']}")

        lines.append(f"CREATE TABLE {tbl['schema']}.{tbl['name']} (")
        lines.append(",\n".join(col_defs))
        lines.append(");\n")

        # Indexes (skip primary-key indexes — they're created by the PK constraint;
        # also skip unique indexes that duplicate a UNIQUE constraint)
        idxs = manifest["indexes"].get(key, [])
        cons_names = {c["name"] for c in cons if c["type"] in ("p", "u")}
        for ix in sorted(idxs, key=lambda x: x["name"]):
            if ix.get("primary") or ix["name"] in cons_names:
                continue  # already created by the constraint
            # The definition from pg_get_indexdef is a full CREATE statement
            lines.append(ix["definition"] + ";")
        if idxs:
            lines.append("")

        # Triggers (reproduce set_updated_at triggers)
        trigs = manifest["triggers"].get(key, [])
        for t in sorted(trigs, key=lambda x: x["name"]):
            lines.append(t["definition"] + ";")
        if trigs:
            lines.append("")

    # Foreign keys (after all tables created)
    lines.append("-- Foreign keys")
    for tbl in tables:
        key = f"{tbl['schema']}.{tbl['name']}"
        fks = [c for c in manifest["constraints"].get(key, []) if c["type"] == "f"]
        for fk in sorted(fks, key=lambda x: x["name"]):
            lines.append(
                f"ALTER TABLE {tbl['schema']}.{tbl['name']} "
                f"ADD CONSTRAINT {fk['name']} {fk['definition']};"
            )
    lines.append("")

    # Views (none in P1, but reproduce if present)
    for v in sorted(manifest.get("views", []), key=lambda x: x["name"]):
        lines.append(f"CREATE VIEW {v['schema']}.{v['name']} AS {v['definition']};")

    return "\n".join(lines)


def generate_seed_sql(manifest: dict) -> str:
    """Generate minimal legacy-truth seed rows for migration classification tests.

    These exercise the historical-truth classifications (Migration spec §6.1):
    - agent_effect_results.status='succeeded' → legacy_unverified
    - asset_actions with proxmox_task_id → legacy_submitted_unverified
    - asset_actions executing → legacy_outcome_unknown
    - approval_requests approved → legacy_decision_evidence
    - action_outcomes → legacy_operator_assessment

    All UUIDs/timestamps are fixed literals. No production data.
    """
    return f"""-- P3 seed: minimal legacy-truth cases for migration classification tests.
-- All values are synthetic fixed literals. NO production data.

-- Bootstrap: one team, one user, one object (required by FKs)
INSERT INTO teams (id, name, slug, created_at, updated_at)
VALUES ('{SEED_TEAM}', 'p3-synthetic-team', 'p3-synthetic', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, password_hash, name, is_active, created_at, updated_at)
VALUES ('{SEED_USER}', 'p3-synthetic@example.invalid',
        'SYNTHETIC-HASH-NOT-A-REAL-CREDENTIAL', 'P3 Synthetic User', true, '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO objects (id, team_id, object_type, title, status, created_by, created_at, updated_at)
VALUES ('{SEED_OBJECT}', '{SEED_TEAM}', 'asset', 'p3-synthetic-asset', 'active', '{SEED_USER}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 1: agent_effect_results.status='succeeded' (→ legacy_unverified)
INSERT INTO agent_identities (id, team_id, name, agent_type, status, max_autonomy, created_by, created_at, updated_at)
VALUES ('{SEED_AGENT}', '{SEED_TEAM}', 'p3-synthetic-agent', 'assistant', 'active', 'A0', '{SEED_USER}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_runs (id, team_id, agent_id, triggered_by, triggered_by_actor_type, status, started_at, correlation_id, created_at, updated_at)
VALUES ('{SEED_OBJECT}', '{SEED_TEAM}', '{SEED_AGENT}', '{SEED_USER}', 'user', 'completed', '{FIXED_TS}', '{SEED_OBJECT}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_intentions (id, team_id, agent_run_id, intention_type, risk_level, autonomy_level, status, created_at)
VALUES ('{SEED_EFFECT}', '{SEED_TEAM}', '{SEED_OBJECT}', 'action', 'medium', 'A2', 'executed', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_effect_results (id, team_id, intention_id, tool_name, status, approval_id, result, created_at)
VALUES ('{SEED_EFFECT}', '{SEED_TEAM}', '{SEED_EFFECT}', 'synthetic_tool', 'succeeded', NULL,
        '{{"synthetic": true}}'::jsonb, '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 2: asset_actions succeeded WITH proxmox_task_id (→ legacy_submitted_unverified)
-- assets uses object_id (not a separate id column); asset_actions references asset_id
INSERT INTO asset_actions (id, team_id, asset_id, action_type, status, proxmox_task_id,
                           requested_by, created_at, updated_at)
VALUES ('{SEED_ACTION}', '{SEED_TEAM}', '{SEED_OBJECT}', 'proxmox.start', 'succeeded',
        'UPID:p3:0000ABC:00000000:00000000', '{SEED_USER}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 3: approval_requests approved (→ legacy_decision_evidence)
INSERT INTO approval_requests (id, team_id, action_type, action_target, risk_level,
                               description, status, requested_by, expires_at, created_at, updated_at)
VALUES ('{SEED_APPROVAL}', '{SEED_TEAM}', 'proxmox.start',
        '{{"asset_id":"{SEED_OBJECT}"}}'::jsonb, 'medium',
        'p3 synthetic approval', 'approved', '{SEED_USER}', '{FIXED_TS}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 4: asset_actions.status='executing' (→ legacy_outcome_unknown)
INSERT INTO asset_actions (id, team_id, asset_id, action_type, status, proxmox_task_id,
                           requested_by, created_at, updated_at)
VALUES ('{SEED_STEP}', '{SEED_TEAM}', '{SEED_OBJECT}', 'proxmox.start', 'executing',
        'UPID:p3:0000DEF:00000000:00000001', '{SEED_USER}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 5: action_outcomes (→ legacy_operator_assessment)
INSERT INTO action_outcomes (id, team_id, asset_action_id, expected_result, actual_result,
                             operator_feedback, outcome_status, created_by, created_at, updated_at)
VALUES ('{SEED_PROPOSAL}', '{SEED_TEAM}', '{SEED_ACTION}', 'VM running',
        'VM started successfully', 'Operator confirmed workload healthy', 'successful',
        '{SEED_USER}', '{FIXED_TS}', '{FIXED_TS}')
ON CONFLICT (id) DO NOTHING;
"""


def generate(manifest_path: str, output_dir: str):
    manifest = json.load(open(manifest_path, encoding="utf-8"))

    os.makedirs(output_dir, exist_ok=True)

    schema_sql = generate_schema_ddl(manifest)
    seed_sql = generate_seed_sql(manifest)

    schema_path = os.path.join(output_dir, "schema.sql")
    seed_path = os.path.join(output_dir, "seed.sql")

    with open(schema_path, "w", encoding="utf-8") as f:
        f.write(schema_sql)
    with open(seed_path, "w", encoding="utf-8") as f:
        f.write(seed_sql)

    schema_hash = hashlib.sha256(schema_sql.encode("utf-8")).hexdigest()
    seed_hash = hashlib.sha256(seed_sql.encode("utf-8")).hexdigest()

    print(f"wrote {schema_path}")
    print(f"wrote {seed_path}")
    print(f"schema.sql sha256 = {schema_hash}")
    print(f"seed.sql sha256   = {seed_hash}")
    print(f"tables generated  = {len([r for r in manifest['relations'] if r['kind']=='r'])}")


def main():
    p = argparse.ArgumentParser(description="Generate P3 sanitized CI fixture from P1 manifest")
    p.add_argument("--p1-manifest", required=True, help="path to P1 production manifest.json")
    p.add_argument("--output", required=True, help="output directory for P3 fixture")
    args = p.parse_args()
    generate(args.p1_manifest, args.output)


if __name__ == "__main__":
    main()
