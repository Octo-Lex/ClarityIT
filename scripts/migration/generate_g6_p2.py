#!/usr/bin/env python3
"""G6 P2 successor adoption artifact generator.

Consumes the exact deterministic P3 output (migrations/v2/adoption/0001_adopt_p3.sql)
and produces a complete P2 adoption artifact (migrations/v2/adoption/0001_adopt_p2.sql)
through explicit, count-checked transformations.

The P3 artifact is NEVER modified. Its SHA-256 must remain:
  a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359
"""
from __future__ import annotations

import hashlib
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
P3_ADOPTION = REPO_ROOT / "migrations/v2/adoption/0001_adopt_p3.sql"
P2_ADOPTION = REPO_ROOT / "migrations/v2/adoption/0001_adopt_p2.sql"
P3_ARTIFACT_SHA256 = "a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359"

# Frozen P2 identities
P2_FP = "57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb"
P3_FP = "cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b"
P2_UUID = "7b5b8b87-3467-5fd5-9bac-3dbcdd858178"
P3_UUID = "7c5cb0b9-1fb4-540d-9433-f0196ff6f7bb"
P2_COMMIT = "b9d15877583e84c45ab5478dfaa6087966926fc5"
P3_COMMIT = "29c4cdcb4c7bd9f13209f5627b55f4fabbd08a33"

PAIRS = [
    ("work.items.edit.own", "work.items.update.own"),
    ("work.items.edit.any", "work.items.update.any"),
    ("projects.edit", "projects.update"),
    ("incidents.edit.own", "incidents.update.own"),
    ("incidents.edit.any", "incidents.update.any"),
    ("docs.edit.own", "docs.update.own"),
    ("docs.edit.any", "docs.update.any"),
]
CANON_IDS = {
    "work.items.update.own": "53c0f4d2-6fec-5d57-84ca-ed58a8dfc19d",
    "work.items.update.any": "4f2499c2-ad5d-5215-94cf-1919ca9fa865",
    "projects.update": "6a53d14f-8ca0-5be7-9a77-f8775d36efaa",
    "incidents.update.own": "678fd8d6-56e9-5335-8b72-06ec2cb09c97",
    "incidents.update.any": "4c73278f-fd39-585c-a8a8-2508e016bde3",
    "docs.update.own": "bdb6f96a-8577-5763-9a48-19adff491206",
    "docs.update.any": "341bd87b-d622-525d-8c06-94308da39f99",
}


def assert_count(text, pattern, expected, desc):
    n = len(re.findall(pattern, text, re.MULTILINE))
    if n != expected:
        raise SystemExit(f"FAIL: expected {expected}x {desc}, found {n}")


def generate(p3: str) -> str:
    t = p3

    # 1. Header
    t = t.replace(
        "-- G3 deterministic P3 approved-source adoption artifact.\n"
        "-- Reconciles an existing P3 source to the signed G2 governed posture.\n"
        "-- No legacy replay, no product-table creation, no business-row mutation.\n"
        "-- The only product-row write is the seven canonical permission inserts.\n"
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g3.py.",
        "-- G6 deterministic P2 approved-source adoption artifact (successor to P3).\n"
        "-- Reconciles an existing P1/P2 source to the signed G2 governed posture.\n"
        "-- No legacy replay, no product-table creation, no business-row mutation.\n"
        "-- Product-row writes: Decision-016 permission reconciliation + seven canonical inserts.\n"
        "-- DO NOT EDIT BY HAND -- regenerate with scripts/migration/generate_g6_p2.py.",
    )

    # 2. Global error message renames (do before targeted replacements)
    t = t.replace("'G3 adoption requires", "'G6 P2 adoption requires")
    t = t.replace("'G3 adoption source", "'G6 P2 adoption source")
    t = t.replace("'G3 adoption is single-shot", "'G6 P2 adoption is single-shot")
    t = t.replace("$g3_adopt_preflight$", "$g6_p2_adopt_preflight$")
    t = t.replace("$g3_adopt_validate$", "$g6_p2_adopt_validate$")
    t = t.replace("g3_adopt_admin", "g6_p2_adopt_admin")

    # 3. pg_trgm preflight: P3 requires it present; P2 requires it ABSENT
    t = t.replace(
        "IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN\n"
        "        RAISE EXCEPTION 'G6 P2 adoption requires extension pg_trgm';\n"
        "    END IF;",
        "-- P2 source does NOT have pg_trgm; the adoption artifact creates it.\n"
        "    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN\n"
        "        RAISE EXCEPTION 'G6 P2 adoption requires pg_trgm to be ABSENT (created during adoption)';\n"
        "    END IF;",
    )

    # 4. Extension ownership check: P2 has 2 (pgcrypto+citext), not 3
    t = t.replace(
        "WHERE e.extname IN ('pgcrypto','citext','pg_trgm') AND r.rolname = 'clarityit') <> 3 THEN\n"
        "        RAISE EXCEPTION 'G6 P2 adoption requires clarityit to own pgcrypto, citext, and pg_trgm';",
        "WHERE e.extname IN ('pgcrypto','citext') AND r.rolname = 'clarityit') <> 2 THEN\n"
        "        RAISE EXCEPTION 'G6 P2 adoption requires clarityit to own pgcrypto and citext';",
    )

    # 5. Insert pg_trgm creation after preflight closes
    marker = "$g6_p2_adopt_preflight$;\n"
    assert_count(t, re.escape(marker), 1, "preflight end marker")
    pg_trgm_block = (
        "\n"
        "-- ============================================================\n"
        "-- P2-specific: create pg_trgm (absent from P1/P2 source).\n"
        "-- ============================================================\n"
        "CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;\n"
        "\n"
    )
    t = t.replace(marker, marker + pg_trgm_block, 1)

    # 6. Replace P3 permission block with D016 reconciliation
    perm_start = t.find("SET LOCAL ROLE clarityit_owner;\nINSERT INTO public.permissions")
    assert perm_start != -1, "FAIL: cannot find P3 permission INSERT"
    perm_end = t.find("$$;\n", perm_start) + len("$$;\n")

    recon_lines = [
        "-- ============================================================\n",
        "-- Decision-016 permission reconciliation.\n",
        "-- For each signed .edit -> .update pair:\n",
        "--   legacy only -> rename to .update, preserving ID and grants\n",
        "--   canonical only -> leave unchanged\n",
        "--   both -> copy legacy grants to canonical, delete legacy\n",
        "--   neither -> fail closed\n",
        "-- ============================================================\n",
        "DO $g6_d016_reconcile$\n",
        "BEGIN\n",
    ]
    for ed, up in PAIRS:
        cid = CANON_IDS[up]
        recon_lines.extend([
            f"    -- {ed} -> {up}\n",
            f"    IF EXISTS (SELECT 1 FROM public.permissions WHERE name = '{ed}') THEN\n",
            f"        IF EXISTS (SELECT 1 FROM public.permissions WHERE name = '{up}') THEN\n",
            f"            INSERT INTO public.role_permissions (role_id, permission_id)\n",
            f"            SELECT role_id, '{cid}'::uuid FROM public.role_permissions rp\n",
            f"            JOIN public.permissions p ON p.id = rp.permission_id\n",
            f"            WHERE p.name = '{ed}' ON CONFLICT DO NOTHING;\n",
            f"            DELETE FROM public.role_permissions WHERE permission_id =\n",
            f"                (SELECT id FROM public.permissions WHERE name = '{ed}');\n",
            f"            DELETE FROM public.permissions WHERE name = '{ed}';\n",
            f"        ELSE\n",
            f"            UPDATE public.permissions SET name = '{up}',\n",
            f"                action = replace(action, 'edit', 'update'),\n",
            f"                resource = replace(resource, 'edit', 'update')\n",
            f"            WHERE name = '{ed}';\n",
            f"        END IF;\n",
            f"    ELSE\n",
            f"        IF NOT EXISTS (SELECT 1 FROM public.permissions WHERE name = '{up}') THEN\n",
            f"            RAISE EXCEPTION 'G6 D016: neither {ed} nor {up} exists';\n",
            f"        END IF;\n",
            f"    END IF;\n",
        ])
    recon_lines.extend([
        "    ASSERT NOT EXISTS (SELECT 1 FROM public.permissions WHERE name LIKE '%.edit%'),\n",
        "        'G6 D016: legacy .edit permissions remain';\n",
        "    ASSERT (SELECT count(*) FROM public.permissions WHERE name IN (\n",
        "        'work.items.update.own','work.items.update.any','projects.update',\n",
        "        'incidents.update.own','incidents.update.any','docs.update.own','docs.update.any'\n",
        "    )) = 7, 'G6 canonical permission set incomplete';\n",
        "END\n",
        "$g6_d016_reconcile$;\n",
        "\n",
    ])
    t = t[:perm_start] + "".join(recon_lines) + t[perm_end:]

    # 7. Source profile row
    t = t.replace(
        f"'{P3_UUID}', '{P3_FP}', 'PostgreSQL 16', 16, "
        "'[\"pgcrypto\",\"citext\",\"pg_trgm\"]'::jsonb, "
        f"'2273a104fa6145ebe699ffc570da41941d49df4584ee2b093f323ce8d5a0a7c3', "
        f"'{P3_COMMIT}', '3b4a6fdeb35473e5f73ca74bafa479bd2648fb10', '2026-08-03T00:00:00Z'",
        f"'{P2_UUID}', '{P2_FP}', 'PostgreSQL 16', 16, "
        "'[\"pgcrypto\",\"citext\",\"pg_trgm\"]'::jsonb, "
        "'2273a104fa6145ebe699ffc570da41941d49df4584ee2b093f323ce8d5a0a7c3', "
        f"'{P2_COMMIT}', 'G6-TERMINAL-CLOSURE-AUTH-2026-08-11', '2026-08-11T00:00:00Z'",
    )

    # 8. Revision name + artifact name
    t = t.replace("'adopt-p3',", "'adopt-p2-v32',")
    t = t.replace("'g3-adoption-artifact',", "'g6-p2-adoption-artifact',")

    return t


def main():
    check = "--check" in sys.argv
    p3_bytes = P3_ADOPTION.read_bytes()
    p3_sha = hashlib.sha256(p3_bytes).hexdigest()
    if p3_sha != P3_ARTIFACT_SHA256:
        print(f"ERROR: P3 SHA-256 mismatch: {p3_sha}")
        sys.exit(1)
    print(f"P3 verified: {p3_sha}")

    p2_text = generate(p3_bytes.decode("utf-8"))
    p2_text = p2_text.rstrip() + "\n"
    p2_bytes = p2_text.encode("utf-8")
    p2_sha = hashlib.sha256(p2_bytes).hexdigest()
    print(f"P2 generated: sha256={p2_sha} size={len(p2_bytes)}")

    if check:
        if P2_ADOPTION.exists():
            ok = hashlib.sha256(P2_ADOPTION.read_bytes()).hexdigest() == p2_sha
            print(f"CHECK: {'PASS' if ok else 'FAIL'}")
            sys.exit(0 if ok else 1)
        print("CHECK: FAIL (not found)")
        sys.exit(1)

    P2_ADOPTION.write_bytes(p2_bytes)
    print(f"Written: {P2_ADOPTION}")


if __name__ == "__main__":
    main()
