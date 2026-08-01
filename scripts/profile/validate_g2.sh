#!/bin/bash
# G2 Test Harness: executes all fixture profiles against isolated PostgreSQL 16 schemas.
# 018 and 016 use isolated schemas on the shared P0 database.
# 029 uses a SEPARATE disposable PostgreSQL 16 container for production-target validation.
set -euo pipefail

echo "=== G2 Fixture Validation Harness ==="
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
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

# Clean up disposable container
docker rm -f g2-029-pg
echo "029 cleanup complete"
echo ""

echo "=== ALL G2 FIXTURES PASSED ==="
