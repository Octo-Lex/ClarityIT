#!/bin/bash
# G2 Test Harness: executes all fixture profiles against isolated PostgreSQL 16 schemas.
# Each fixture creates and cleans up its own schema namespace.
# ON_ERROR_STOP=1 for all psql calls — no false passes.
set -euo pipefail

echo "=== G2 Fixture Validation Harness ==="
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

echo "=== Step 1: 018 Agent P1-Canonical Validation (isolated schemas) ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/018-agent-p1-validation.sql
echo "018 PASS: P1-canonical validated; raw-018 and 005-only divergences confirmed"
echo ""

echo "=== Step 2: 016 Permission Normalization (isolated schema) ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/016-permissions.sql
echo "016 PASS: all 7 canonical names, dual-grant collision, negative case validated"
echo ""

echo "=== Step 3: 029 Role Bootstrap (fail-closed + 5-role posture) ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/029-role-bootstrap.sql
echo "029 PASS: five-role posture validated with exact flags and memberships"
echo ""

echo "=== ALL G2 FIXTURES PASSED ==="
