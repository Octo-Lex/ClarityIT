#!/bin/bash
# G2 Test Harness: executes all fixture profiles against isolated PostgreSQL 16 schemas.
# 018 and 016 use isolated schemas on the shared P0 database.
# 029 uses a SEPARATE disposable PostgreSQL 16 container for production-target validation.
set -euo pipefail

echo "=== G2 Fixture Validation Harness ==="
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# === Step 0: Closed-world grant inventory + blob-digest assertion (fail-closed) ===
# Two independent fail-closed checks:
#   (a) GRANT-INV: regenerate the target_grants block from the manifest's own
#       object inventory and assert it matches the committed manifest (canonical
#       JSON). Catches drift between the generator and the committed inventory.
#   (b) BLOB-DIGEST: assert the committed Git BLOB's SHA-256 and byte size
#       against the detached checksum file. The blob is read via
#       `git cat-file blob` — the repository artifact CI tests — NOT the
#       working-tree file, whose line endings vary by platform (CRLF on
#       Windows, LF on Linux). This makes the cited digest match the artifact
#       CI tests and fail-closes any divergence between the receipt and the
#       committed bytes.
echo "=== Step 0: Closed-world grant inventory + blob-digest assertion ==="
python3 - <<'PYEOF'
import json, subprocess, sys, hashlib, re

MANIFEST = "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json"
CHECKSUM = "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.sha256"

# --- (a) GRANT-INV: generated vs committed target_grants ---
proc = subprocess.run(
    ["python3", "scripts/profile/generate_g2_grants.py"],
    capture_output=True, text=True,
)
if proc.returncode != 0:
    print("GRANT-INV FAIL: generate_g2_grants.py exited non-zero")
    print(proc.stderr)
    sys.exit(1)

generated = json.loads(proc.stdout)["target_grants"]
with open(MANIFEST, encoding="utf-8") as fh:
    committed = json.load(fh)["target_grants"]

gen_canon = json.dumps(generated, sort_keys=True, ensure_ascii=False)
com_canon = json.dumps(committed, sort_keys=True, ensure_ascii=False)
if gen_canon != com_canon:
    print("GRANT-INV FAIL: generated target_grants != committed target_grants")
    print("The manifest was hand-edited, OR a migration added/changed an object")
    print("and the manifest was not regenerated. Re-run generate_g2_grants.py")
    print("and commit the updated manifest + checksum.")
    import difflib
    diff = list(difflib.unified_diff(
        com_canon.splitlines(), gen_canon.splitlines(),
        fromfile="committed", tofile="generated", lineterm="", n=2))
    print("\n".join(diff[:40]) or "(diff too dense to summarize — inspect manually)")
    sys.exit(1)

# --- (b) BLOB-DIGEST: committed Git blob vs detached checksum file ---
# Read the repository artifact via git cat-file, NOT the working-tree file.
blob = subprocess.run(
    ["git", "cat-file", "blob", f"HEAD:{MANIFEST}"],
    capture_output=True, check=True,
).stdout
actual_sha = hashlib.sha256(blob).hexdigest()
actual_size = len(blob)

# Parse the detached checksum file for the expected values.
checksum_text = open(CHECKSUM, encoding="utf-8").read()
m_sha = re.search(r"^[^:\s]+:([0-9a-f]{64})\s*$", checksum_text, re.M)
m_size = re.search(r"^[^:\s]+:size:(\d+)\s*$", checksum_text, re.M)
if not m_sha or not m_size:
    print(f"BLOB-DIGEST FAIL: checksum file {CHECKSUM} malformed")
    print(checksum_text)
    sys.exit(1)
expected_sha = m_sha.group(1)
expected_size = int(m_size.group(1))

if actual_sha != expected_sha:
    print("BLOB-DIGEST FAIL: committed blob SHA-256 != checksum file")
    print(f"  committed blob (git cat-file HEAD): {actual_sha}")
    print(f"  checksum file expects:              {expected_sha}")
    print("The manifest changed but the checksum was not regenerated, OR the")
    print("checksum cites a working-tree (CRLF) digest instead of the blob.")
    print("Recompute via: git cat-file blob HEAD:<path> | sha256sum")
    sys.exit(1)
if actual_size != expected_size:
    print("BLOB-DIGEST FAIL: committed blob size != checksum file")
    print(f"  committed blob: {actual_size} bytes")
    print(f"  checksum file:  {expected_size} bytes")
    sys.exit(1)

print(f"GRANT-INV PASS: generated == committed ({len(committed['tables'])} tables, "
      f"{len(committed['application_functions'])} app functions, "
      f"{committed['extension_functions']['count']} extension excluded, "
      f"{len(committed['sequences'])} sequences, {len(committed['schemas'])} schemas)")
print(f"BLOB-DIGEST PASS: committed blob sha256 {actual_sha} ({actual_size} bytes) == checksum file")
PYEOF
echo ""

echo "=== Step 1: 018 Agent P1-Canonical Validation (isolated schemas on shared DB) ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/018-agent-p1-validation.sql
echo "018 PASS: P1-canonical validated; raw-018 and 005-only divergences confirmed"
echo ""

echo "=== Step 2: 016 Permission Normalization (isolated schema on shared DB) ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/016-permissions.sql
echo "016 PASS: all 7 canonical names, dual-grant collision, negative case validated"
echo ""

echo "=== Step 3: 029 Role Bootstrap (disposable PostgreSQL 16 cluster) ==="

# Create a fresh disposable PostgreSQL 16 container with a bootstrap admin.
# NOTE: postgres:16-alpine first boots an interim "bootstrap" server to create
# POSTGRES_DB, then restarts. pg_isready succeeds against the interim server,
# so we cannot rely on it alone. A real SELECT through the socket is the only
# proof that the final server is accepting connections.
docker rm -f g2-029-pg 2>/dev/null || true
docker run -d --name g2-029-pg --network clarityit-net \
    -e POSTGRES_USER=g2admin \
    -e POSTGRES_PASSWORD=g2admin_pass \
    -e POSTGRES_DB=g2_029_test \
    postgres:16-alpine

# Bounded readiness probe: a real query that survives the postgres restart.
ready=0
for i in $(seq 1 60); do
    # Two consecutive successful real queries prove we're past the restart.
    if docker exec g2-029-pg psql -U g2admin -d g2_029_test -tAc "SELECT 1" >/dev/null 2>&1; then
        sleep 1
        if docker exec g2-029-pg psql -U g2admin -d g2_029_test -tAc "SELECT 1" >/dev/null 2>&1; then
            ready=1
            echo "g2-029-pg ready (real-query probe passed twice)"
            break
        fi
    fi
    sleep 1
done
if [ "$ready" -ne 1 ]; then
    echo "029 FAIL: g2-029-pg did not become query-ready within 60s"
    docker logs g2-029-pg 2>&1 | tail -20
    docker rm -f g2-029-pg
    exit 1
fi

# Also prove the fresh DB truly has none of the five application roles before
# we mutate it — otherwise the fail-closed test below is meaningless.
pre_count=$(docker exec g2-029-pg psql -U g2admin -d g2_029_test -tAc \
    "SELECT count(*) FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')")
if [ "$pre_count" != "0" ]; then
    echo "029 FAIL: fresh DB already contains $pre_count application role(s) — container not clean"
    docker rm -f g2-029-pg
    exit 1
fi
echo "029 pre-check: 0 application roles in fresh DB (confirmed clean)"

# Step 3a: Pre-mutation fail-closed test (expect ASSERT FAILURE in fresh DB).
# We must distinguish the expected assert_failure from a connection failure.
# The psql command is EXPECTED to exit non-zero, so we disable errexit around it.
echo "--- 3a: Pre-mutation fail-closed (expecting assert_failure, not conn error) ---"
set +e
fail_3a_output=$(docker exec -i g2-029-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    -c "DO \$\$ BEGIN ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app'), 'expected fail'; END \$\$;" \
    2>&1)
fail_3a_rc=$?
set -e
if [ "$fail_3a_rc" -eq 0 ]; then
    echo "029 FAIL: clarityit_app already exists in fresh DB — fail-closed did NOT trigger"
    docker rm -f g2-029-pg
    exit 1
fi
# Reject connection/infra errors — only assert_failure is the expected outcome.
case "$fail_3a_output" in
    *"expected fail"*)
        echo "029 PASS: fail-closed correctly rejected via assert_failure"
        ;;
    *"connection"*|*"FATAL"*|*"No such file"*|*"shutting down"*|*"refused"*)
        echo "029 FAIL: 3a aborted on a CONNECTION error, not assert_failure:"
        echo "$fail_3a_output"
        docker rm -f g2-029-pg
        exit 1
        ;;
    *)
        echo "029 FAIL: 3a failed for an unexpected reason (rc=$fail_3a_rc):"
        echo "$fail_3a_output"
        docker rm -f g2-029-pg
        exit 1
        ;;
esac

# Step 3b: Bootstrap + validate five-role posture (expect SUCCESS)
echo "--- 3b: Bootstrap and validate five-role posture ---"
docker exec -i g2-029-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/029-role-bootstrap.sql
echo "029 PASS: five-role posture validated with exact flags and membership options"

# Step 3c: Superuser rejection — clarityit must be non-superuser (production target)
echo "--- 3c: Superuser rejection test ---"
docker exec -i g2-029-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    -c "DO \$\$ BEGIN ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND rolsuper = true), 'superuser rejected'; END \$\$;"
echo "029 PASS: clarityit is non-superuser in production target"

# Clean up the bootstrap cluster
docker rm -f g2-029-pg
echo "029 bootstrap cluster cleanup complete"

# Step 3d: Incorrect-role REJECTION profile on a SECOND disposable cluster.
# Each case seeds a bad posture inside BEGIN...ROLLBACK, so the cluster must
# start empty (no application roles) — hence a fresh container, not the
# already-bootstrapped one.
echo "--- 3d: Incorrect-role rejection profile (5 cases) ---"
docker rm -f g2-029-rej-pg 2>/dev/null || true
docker run -d --name g2-029-rej-pg --network clarityit-net \
    -e POSTGRES_USER=g2admin \
    -e POSTGRES_PASSWORD=g2admin_pass \
    -e POSTGRES_DB=g2_029_test \
    postgres:16-alpine >/dev/null

# Same real-query readiness probe as the bootstrap cluster.
ready=0
for i in $(seq 1 60); do
    if docker exec g2-029-rej-pg psql -U g2admin -d g2_029_test -tAc "SELECT 1" >/dev/null 2>&1; then
        sleep 1
        if docker exec g2-029-rej-pg psql -U g2admin -d g2_029_test -tAc "SELECT 1" >/dev/null 2>&1; then
            ready=1; break
        fi
    fi
    sleep 1
done
if [ "$ready" -ne 1 ]; then
    echo "029 FAIL: g2-029-rej-pg did not become query-ready within 60s"
    docker logs g2-029-rej-pg 2>&1 | tail -20
    docker rm -f g2-029-rej-pg
    exit 1
fi

# Pre-check: rejection cluster must start clean
pre_count=$(docker exec g2-029-rej-pg psql -U g2admin -d g2_029_test -tAc \
    "SELECT count(*) FROM pg_roles WHERE rolname IN ('clarityit','clarityit_app','clarityit_owner','clarityit_migrator','clarityit_admin')")
if [ "$pre_count" != "0" ]; then
    echo "029 FAIL: rejection cluster not clean ($pre_count app roles present)"
    docker rm -f g2-029-rej-pg
    exit 1
fi

# Run the rejection profile ONCE and capture output (each case uses ROLLBACK,
# so the cluster returns to clean after every case — no double-application).
rej_log=$(docker exec -i g2-029-rej-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/029-rejection-profile.sql 2>&1)
echo "$rej_log"

# Confirm every rejection case produced its PASS notice (all 5 must appear).
for label in R1 R2 R3 R4 R5; do
    if ! echo "$rej_log" | grep -q "$label PASS"; then
        echo "029 FAIL: rejection case $label did not PASS"
        docker rm -f g2-029-rej-pg
        exit 1
    fi
done
echo "029 PASS: all 5 incorrect-role cases rejected (R1 superuser, R2 wrong flags, R3 ADMIN-TRUE delegation, R4 partial posture, R5 extraneous membership)"

docker rm -f g2-029-rej-pg
echo "029 rejection cluster cleanup complete"
echo ""

echo "=== ALL G2 FIXTURES PASSED ==="
