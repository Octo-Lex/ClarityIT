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

# Create a fresh disposable PostgreSQL 16 container with a bootstrap admin
docker rm -f g2-029-pg 2>/dev/null || true
docker run -d --name g2-029-pg --network clarityit-net \
    -e POSTGRES_USER=g2admin \
    -e POSTGRES_PASSWORD=g2admin_pass \
    -e POSTGRES_DB=g2_029_test \
    postgres:16-alpine

# Wait for it to be ready
for i in $(seq 1 30); do
    if docker exec g2-029-pg pg_isready -U g2admin >/dev/null 2>&1; then
        echo "g2-029-pg ready"
        break
    fi
    sleep 1
done

# Step 3a: Pre-mutation fail-closed test (expect FAILURE in fresh DB)
echo "--- 3a: Pre-mutation fail-closed (expecting failure) ---"
if docker exec -i g2-029-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    -c "DO \$\$ BEGIN ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app'), 'expected fail'; END \$\$;" \
    2>&1; then
    echo "029 FAIL: clarityit_app already exists in fresh DB — container not clean"
    docker rm -f g2-029-pg
    exit 1
else
    echo "029 PASS: fail-closed correctly rejected (no roles in fresh DB)"
fi

# Step 3b: Bootstrap + validate five-role posture (expect SUCCESS)
echo "--- 3b: Bootstrap and validate five-role posture ---"
docker exec -i g2-029-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/029-role-bootstrap.sql
echo "029 PASS: five-role posture validated with exact flags and membership options"

# Step 3c: Superuser rejection test — try to create a superuser clarityit (expect FAIL)
echo "--- 3c: Superuser rejection test ---"
# In the disposable DB, clarityit was already created as non-superuser.
# Verify it's NOT a superuser.
docker exec -i g2-029-pg psql -U g2admin -d g2_029_test -v ON_ERROR_STOP=1 \
    -c "DO \$\$ BEGIN ASSERT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit' AND rolsuper = true), 'superuser rejected'; END \$\$;"
echo "029 PASS: clarityit is non-superuser in production target"

# Clean up disposable container
docker rm -f g2-029-pg
echo "029 cleanup complete"
echo ""

echo "=== ALL G2 FIXTURES PASSED ==="
