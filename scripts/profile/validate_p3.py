#!/usr/bin/env python3
"""
P3 validation harness — proves the P3 fixture is reproducible and clean.

Executes the full acceptance proof:
  1. Build two fresh PostgreSQL 16 databases (Docker containers).
  2. Apply P3 schema + seed to each.
  3. Capture both with the profiler.
  4. Recompute fingerprints from both manifests (self-consistency).
  5. Compare both captures (determinism).
  6. Compare both against the committed golden manifest.
  7. Run secret + production-identifier scan on the manifests.

Exit code 0 = all checks pass. Non-zero = failure (with diagnostics).

Usage:
    python scripts/profile/validate_p3.py
    python scripts/profile/validate_p3.py --p3-dir migrations/profiles/p3
    python scripts/profile/validate_p3.py --keep   # don't tear down containers

Requires Docker on PATH.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
import time

_HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(_HERE))
PROFILER = os.path.join(_HERE, "capture_schema.py")

# Import the profiler module so we use its canonical fingerprint_of (DRY —
# the harness must not duplicate fingerprint logic, or it drifts).
import importlib.util as _ilutil
_spec = _ilutil.spec_from_file_location("capture_schema", PROFILER)
cs = _ilutil.module_from_spec(_spec)
_spec.loader.exec_module(cs)

# Patterns that indicate production data leakage (never in a synthetic fixture).
# The scan must substantiate claims about tokens, credentials, emails, hostnames,
# and production identifiers. These cover both the SQL files and JSON manifests.
SECRET_PATTERNS = [
    re.compile(p, re.IGNORECASE) for p in [
        # --- Production network identifiers ---
        r"192\.168\.\d+\.\d+",            # private production IPs
        r"10\.\d+\.\d+\.\d+",             # private production IPs (10.x)
        r"pve0\d",                        # production Proxmox node names
        r"clarityit-postgres",            # production container names
        r"clarityit-context-worker",
        r"clarityit-api",
        # --- Production email domains ---
        r"@clarityit\.(com|dev|io|net)",  # production email domain
        r"@(?!example\.|p3-synthetic)[a-z]",  # any non-example/synthetic email local-part
        # --- Credentials / tokens ---
        r"(?:password|passwd|pwd)\s*[=:]"
        r"\s*['\"](?!\s*SYNTHETIC|change-me|replace)",  # real passwords (not markers)
        r"(?:api[_-]?key|secret|token)\s*[=:]"
        r"\s*['\"](?:gho_|ghp_|ghs_|sk_|pk_|AKIA|xox[bpoa]-)",  # known token prefixes
        r"(?:api[_-]?key|secret|token)\s*[=:]"
        r"\s*['\"][A-Za-z0-9]{32,}",      # long opaque strings that look like tokens
        r"-----BEGIN (?:RSA |EC |OPENSSH |)PRIVATE KEY-----",  # private keys
        # --- JWT / bearer tokens ---
        r"eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}",  # JWT-shaped strings
        # --- Connection strings with credentials ---
        r"(?:postgres|postgresql|redis|nats|amqp)://[^:\s]+:[^@\s]+@",  # URI-embedded passwords
    ]
]


def run(cmd, **kw):
    """Run a command, return CompletedProcess. Raises on failure."""
    kw.setdefault("capture_output", True)
    kw.setdefault("text", True)
    kw.setdefault("check", True)
    return subprocess.run(cmd, **kw)


# Pinned to match production (PostgreSQL 16.14). Floating tags like
# postgres:16-alpine cause golden mismatches across patch versions.
PG_IMAGE = os.environ.get("P3_PG_IMAGE", "postgres:16.14-alpine")


def docker_run_pg(name, network):
    """Start a fresh PostgreSQL container from the pinned image."""
    run([
        "docker", "run", "-d", "--name", name,
        "--network", network,
        "-e", "POSTGRES_USER=clarityit",
        "-e", "POSTGRES_PASSWORD=clarityit",
        "-e", "POSTGRES_DB=clarityit",
        PG_IMAGE,
    ])


def wait_ready(name, timeout=30):
    """Wait until postgres accepts connections."""
    for _ in range(timeout):
        r = subprocess.run(
            ["docker", "exec", name, "pg_isready", "-U", "clarityit"],
            capture_output=True,
        )
        if r.returncode == 0:
            return True
        time.sleep(1)
    return False


def apply_sql(name, sql_path):
    """Apply a SQL file to a container via stdin."""
    with open(sql_path, "rb") as f:
        subprocess.run(
            ["docker", "exec", "-i", name, "psql", "-U", "clarityit", "-d",
             "clarityit", "-v", "ON_ERROR_STOP=1"],
            stdin=f, capture_output=True, check=True,
        )


def capture(name, network, out_dir, label):
    """Run the profiler against a container, writing manifest to out_dir."""
    run([
        "docker", "run", "--rm",
        "--network", network,
        "-v", PROFILER + ":/app/capture_schema.py",
        "-v", os.path.abspath(out_dir) + ":/out",
        "-w", "/app",
        "-e", "PGPASSWORD=clarityit",
        "python:3.12-slim",
        "sh", "-c",
        f"pip install -q psycopg2-binary 2>/dev/null; "
        f"python3 capture_schema.py capture "
        f"--host {name} --port 5432 --db clarityit --user clarityit "
        f"--label {label} --out /out",
    ])


def fingerprint_of(manifest_path):
    """Recompute the fingerprint from a manifest file using the profiler's logic."""
    m = json.load(open(manifest_path, encoding="utf-8"))
    return cs.fingerprint_of(m)


def secret_scan(path):
    """Scan any file (manifest, SQL, JSON) for production data leakage.
    Returns list of findings."""
    text = open(path, encoding="utf-8").read()
    findings = []
    for pat in SECRET_PATTERNS:
        for m in pat.finditer(text):
            ctx = text[max(0, m.start() - 30):m.end() + 30]
            findings.append(f"{os.path.basename(path)}: pattern {pat.pattern!r} matched: ...{ctx}...")
    return findings


def main():
    ap = argparse.ArgumentParser(description="P3 validation harness")
    ap.add_argument("--p3-dir", default=os.path.join(REPO, "migrations", "profiles", "p3"))
    ap.add_argument("--keep", action="store_true", help="don't tear down containers")
    ap.add_argument("--update-golden", action="store_true",
                    help="write/overwrite golden-manifest.json from capture A")
    args = ap.parse_args()

    schema = os.path.join(args.p3_dir, "schema.sql")
    seed = os.path.join(args.p3_dir, "seed.sql")
    golden = os.path.join(args.p3_dir, "golden-manifest.json")

    for f in [schema, seed]:
        if not os.path.exists(f):
            print(f"FAIL: missing {f}", file=sys.stderr)
            return 1

    # Golden may not exist yet on first run (--update-golden establishes it)
    golden_fp = None
    if os.path.exists(golden):
        golden_fp = json.load(open(golden))["fingerprint_sha256"]

    net = "p3-validate-net"
    containers = ["p3-validate-a", "p3-validate-b"]
    tmp = tempfile.mkdtemp(prefix="p3-validate-")

    failures = []

    def check(label, condition, detail=""):
        status = "PASS" if condition else "FAIL"
        print(f"  [{status}] {label}" + (f" — {detail}" if detail else ""))
        if not condition:
            failures.append(label)

    try:
        print("=== Setting up Docker network ===")
        subprocess.run(["docker", "network", "rm", net], capture_output=True)
        run(["docker", "network", "create", net])

        for c in containers:
            print(f"=== Starting {c} (PostgreSQL 16) ===")
            run(["docker", "rm", "-f", c], capture_output=True)  # cleanup stale
            docker_run_pg(c, net)
            if not wait_ready(c):
                failures.append(f"{c} not ready")
                print(f"  [FAIL] {c} did not become ready")
                raise RuntimeError("postgres not ready")

        for c in containers:
            print(f"=== Applying P3 to {c} ===")
            apply_sql(c, schema)
            apply_sql(c, seed)
            print(f"  schema + seed applied to {c}")

        print("=== Capturing profiles ===")
        cap_a = os.path.join(tmp, "a")
        cap_b = os.path.join(tmp, "b")
        os.makedirs(cap_a)
        os.makedirs(cap_b)
        capture(containers[0], net, cap_a, "P3-validate-a")
        capture(containers[1], net, cap_b, "P3-validate-b")

        fp_a_stored = json.load(open(os.path.join(cap_a, "manifest.json")))["fingerprint_sha256"]
        fp_b_stored = json.load(open(os.path.join(cap_b, "manifest.json")))["fingerprint_sha256"]
        fp_a_recomp = fingerprint_of(os.path.join(cap_a, "manifest.json"))
        fp_b_recomp = fingerprint_of(os.path.join(cap_b, "manifest.json"))

        print("=== Validation checks ===")
        check("A: self-consistent (stored == recomputed)", fp_a_stored == fp_a_recomp,
              f"{fp_a_stored[:16]} vs {fp_a_recomp[:16]}")
        check("B: self-consistent (stored == recomputed)", fp_b_stored == fp_b_recomp,
              f"{fp_b_stored[:16]} vs {fp_b_recomp[:16]}")
        check("A == B (deterministic)", fp_a_stored == fp_b_stored,
              f"{fp_a_stored[:16]} vs {fp_b_stored[:16]}")

        if args.update_golden or golden_fp is None:
            # Establish/overwrite the golden from capture A (after self-consistency
            # and determinism are proven). This is how the golden is produced —
            # by the same process CI uses, not by hand-derivation.
            import shutil
            shutil.copy(os.path.join(cap_a, "manifest.json"), golden)
            golden_fp = fp_a_stored
            print(f"  [WRITE] golden-manifest.json established: {golden_fp}")
            check("golden established from capture A", True)
        else:
            check("A == golden (matches committed profile)", fp_a_stored == golden_fp,
                  f"{fp_a_stored[:16]} vs {golden_fp[:16]}")
            check("B == golden (matches committed profile)", fp_b_stored == golden_fp,
                  f"{fp_b_stored[:16]} vs {golden_fp[:16]}")

        print("=== Secret + production-identifier scan (all fixture + captured files) ===")
        scan_files = [
            ("fixture schema.sql", schema),
            ("fixture seed.sql", seed),
            ("golden-manifest.json", golden),
            ("capture A manifest", os.path.join(cap_a, "manifest.json")),
            ("capture B manifest", os.path.join(cap_b, "manifest.json")),
        ]
        all_findings = []
        for label, path in scan_files:
            findings = secret_scan(path)
            check(f"{label}: secret-scan clean", len(findings) == 0,
                  f"{len(findings)} findings" if findings else "")
            all_findings.extend(findings)
        for f in all_findings:
            print(f"    ! {f}")

    finally:
        if not args.keep:
            print("=== Tearing down ===")
            for c in containers:
                subprocess.run(["docker", "rm", "-f", c], capture_output=True)
            subprocess.run(["docker", "network", "rm", net], capture_output=True)

    print()
    if failures:
        print(f"RESULT: {len(failures)} check(s) FAILED: {', '.join(failures)}")
        return 1
    print("RESULT: ALL CHECKS PASSED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
