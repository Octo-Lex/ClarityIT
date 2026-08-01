#!/bin/bash
# G2 Test Harness: executes all fixture profiles against PostgreSQL 16.
# Positive cases must pass; negative cases must fail with unchanged catalog.
set -euo pipefail

P0_SCHEMA="/tmp/clarityit-ci-p0.sql"
P0_VERIFY="migrations/ci/p0/verify.sql"
P0_SEED="migrations/ci/p0/seed.sql"

echo "=== G2 Fixture Validation Harness ==="
echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# Ensure we have a P0 database to test against
echo "=== Step 0: Verify P0 schema is applied ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < "$P0_VERIFY" > /dev/null 2>&1 || {
    echo "FAIL: P0 schema not applied. Applying..."
    docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 < "$P0_SCHEMA"
    docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 < "$P0_VERIFY"
    docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 < "$P0_SEED"
}

echo "=== Step 1: 018 Agent Schema Validation ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/018-agent-schema.sql
echo "018 PASS: canonical P1 agent shape validated"
echo ""

echo "=== Step 2: 016 Permission Normalization ==="
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/016-permissions.sql
echo "016 PASS: permission normalization validated (simple rename + collision + negative)"
echo ""

echo "=== Step 3: 029 Role Bootstrap ==="
# 029 requires roles to exist first — create them
docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 << 'SQL'
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app') THEN
        CREATE ROLE clarityit_app NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS INHERIT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_owner') THEN
        CREATE ROLE clarityit_owner NOLOGIN NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_migrator') THEN
        CREATE ROLE clarityit_migrator LOGIN NOINHERIT NOCREATEDB NOCREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_admin') THEN
        CREATE ROLE clarityit_admin LOGIN NOINHERIT NOCREATEDB CREATEROLE NOSUPERUSER NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_auth_members am JOIN pg_roles r ON r.oid=am.member JOIN pg_roles r2 ON r2.oid=am.roleid WHERE r.rolname='clarityit' AND r2.rolname='clarityit_app') THEN
        GRANT clarityit_app TO clarityit;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_auth_members am JOIN pg_roles r ON r.oid=am.member JOIN pg_roles r2 ON r2.oid=am.roleid WHERE r.rolname='clarityit_migrator' AND r2.rolname='clarityit_owner') THEN
        GRANT clarityit_owner TO clarityit_migrator WITH ADMIN OPTION;
    END IF;
END $$;
SQL

docker exec -i postgres psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 \
    < migrations/profiles/g2/fixtures/029-role-bootstrap.sql
echo "029 PASS: role bootstrap validated"
echo ""

echo "=== ALL G2 FIXTURES PASSED ==="
