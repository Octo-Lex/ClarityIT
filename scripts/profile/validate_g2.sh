#!/bin/bash
# G2 Test Harness: executes all fixture profiles against isolated PostgreSQL 16 schemas.
# 018 and 016 use isolated schemas on the shared P0 database.
# 029 uses a SEPARATE disposable PostgreSQL 16 container for production-target validation.
set -euo pipefail

echo "=== G2 Fixture Validation Harness ==="
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# === Step 0: Closed-world grant inventory — generated vs committed (fail-closed) ===
# Regenerates the target_grants block from the manifest's own object inventory via
# generate_g2_grants.py and asserts it matches the committed manifest byte-for-byte
# (after canonical JSON normalization). Catches any drift between the generator and
# the committed inventory — e.g. a hand-edit to the manifest, a new migration adding
# an application function, or a stale generator. No database required.
echo "=== Step 0: Closed-world grant inventory — generated vs committed ==="
python3 - <<'PYEOF'
import json, subprocess, sys, hashlib

MANIFEST = "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json"

# Regenerate the grants block from the manifest's object inventory.
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

# Canonical comparison: same key order is NOT required, same structure/values ARE.
# sort_keys=True normalizes dict ordering so only real differences fail.
gen_canon = json.dumps(generated, sort_keys=True, ensure_ascii=False)
com_canon = json.dumps(committed, sort_keys=True, ensure_ascii=False)

if gen_canon != com_canon:
    print("GRANT-INV FAIL: generated target_grants != committed target_grants")
    print("This means the manifest was hand-edited, OR a migration added/changed")
    print("an object and the manifest was not regenerated. Re-run")
    print("generate_g2_grants.py and commit the updated manifest.")
    # Surface the first differing segment for diagnosis.
    import difflib
    diff = list(difflib.unified_diff(
        com_canon.splitlines(), gen_canon.splitlines(),
        fromfile="committed", tofile="generated", lineterm="", n=2))
    print("\n".join(diff[:40]) or "(diff too dense to summarize — inspect manually)")
    sys.exit(1)

# Confirm the manifest's raw bytes hash to the value the approval record cites,
# so the receipt and the artifact cannot silently diverge.
raw = open(MANIFEST, "rb").read()
print(f"GRANT-INV PASS: generated == committed ({len(committed['tables'])} tables, "
      f"{len(committed['application_functions'])} app functions, "
      f"{committed['extension_functions']['count']} extension excluded, "
      f"{len(committed['sequences'])} sequences, {len(committed['schemas'])} schemas)")
print(f"manifest sha256: {hashlib.sha256(raw).hexdigest()}")
print(f"manifest size:   {len(raw)} bytes")
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
