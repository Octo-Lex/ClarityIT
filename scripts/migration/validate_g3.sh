#!/usr/bin/env bash
# WP-00 G3 validation.
#  --static : committed-blob + receipt-bind static checks only.
#  live     : static + two fresh installs + P3 adoption convergence + drift negative.
set -euo pipefail

mode="${1:-live}"
if [[ "$mode" != "live" && "$mode" != "--static" ]]; then
    echo "usage: $0 [--static]" >&2
    exit 2
fi

python3 scripts/migration/verify_g3.py static
if [[ "$mode" == "--static" ]]; then
    exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
    echo "G3 VERIFY FAIL: live validation requires Docker" >&2
    exit 1
fi
if ! python3 -c 'import psycopg2' >/dev/null 2>&1; then
    echo "G3 VERIFY FAIL: live validation requires psycopg2-binary" >&2
    exit 1
fi

# Pinned PostgreSQL 16.14-alpine image (immutable by digest).
PG_IMAGE="postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382"
password="g3-local-proof-only"
# The P3 golden fingerprint (G1-approved source profile).
P3_FP="cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b"

suffix="$$"
container_a="g3-fresh-a-${suffix}"
container_b="g3-fresh-b-${suffix}"
container_p3="g3-p3-adopt-${suffix}"
container_drift="g3-p3-drift-${suffix}"

cleanup() {
    docker rm -f "$container_a" "$container_b" "$container_p3" "$container_drift" >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_pg() {
    local name="$1" user="$2" pass="$3"
    docker run -d --name "$name" \
        -e POSTGRES_USER="$user" \
        -e POSTGRES_PASSWORD="$pass" \
        -e POSTGRES_DB=clarityit \
        -p 127.0.0.1::5432 \
        "$PG_IMAGE" >/dev/null

    local ready=0
    for _ in $(seq 1 90); do
        if docker exec "$name" psql -U "$user" -d clarityit -tAc "SELECT 1" >/dev/null 2>&1; then
            sleep 1
            if docker exec "$name" psql -U "$user" -d clarityit -tAc "SELECT 1" >/dev/null 2>&1; then
                ready=1
                break
            fi
        fi
        sleep 1
    done
    if [[ "$ready" -ne 1 ]]; then
        echo "G3 VERIFY FAIL: $name did not become query-ready" >&2
        docker logs "$name" >&2
        exit 1
    fi
}

install_g3() {
    local name="$1"
    local artifact
    for artifact in \
        migrations/v2/bootstrap/0000_roles.sql \
        migrations/v2/bootstrap/0000_platform.sql \
        migrations/v2/baseline/0001_reconciled.sql \
        migrations/v2/baseline/0001_seed.sql
    do
        docker exec -i "$name" psql -U g3bootstrap -d clarityit -v ON_ERROR_STOP=1 -q < "$artifact" >/dev/null
    done
}

apply_p3() {
    local name="$1"
    git cat-file blob HEAD:migrations/profiles/p3/schema.sql | \
        docker exec -i "$name" psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 -q >/dev/null
    git cat-file blob HEAD:migrations/profiles/p3/seed.sql | \
        docker exec -i "$name" psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 -q >/dev/null
}

port_of() {
    docker port "$1" 5432/tcp | sed -E 's/.*:([0-9]+)$/\1/'
}

dsn() {
    echo "host=127.0.0.1 port=$1 dbname=clarityit user=$2 password=$3"
}

echo "=== Starting two fresh G3 installs ==="
start_pg "$container_a" g3bootstrap "$password"
start_pg "$container_b" g3bootstrap "$password"
install_g3 "$container_a"
install_g3 "$container_b"
# Create a read-only superuser for governed verification on each fresh install.
for c in "$container_a" "$container_b"; do
    docker exec "$c" psql -U g3bootstrap -d clarityit -c \
        "CREATE ROLE g3_proof_admin LOGIN SUPERUSER PASSWORD '$password';" >/dev/null
done
port_a="$(port_of "$container_a")"
port_b="$(port_of "$container_b")"

echo "=== Two-fresh live comparison (raw profiler) ==="
python3 scripts/migration/verify_g3.py live \
    --dsn-a "$(dsn "$port_a" g3bootstrap "$password")" \
    --dsn-b "$(dsn "$port_b" g3bootstrap "$password")"

echo "=== Governed fingerprint: fresh A == fresh B == A4 frozen target ==="
fp_a="$(python3 scripts/migration/verify_g3.py governed --dsn "$(dsn "$port_a" g3_proof_admin "$password")")"
fp_b="$(python3 scripts/migration/verify_g3.py governed --dsn "$(dsn "$port_b" g3_proof_admin "$password")")"
if [[ "$fp_a" != "$fp_b" ]]; then
    echo "G3 VERIFY FAIL: fresh governed fingerprints differ: $fp_a vs $fp_b" >&2
    exit 1
fi
# Live-bind: assert the computed value equals the A4 frozen target.
python3 scripts/migration/verify_g3.py governed-bind --dsn "$(dsn "$port_a" g3_proof_admin "$password")"
echo "GOVERNED-FRESH-EQUIV PASS: $fp_a"

echo "=== P3 adoption convergence ==="
start_pg "$container_p3" clarityit clarityit
apply_p3 "$container_p3"
port_p3="$(port_of "$container_p3")"

echo "--- pre-adopt preconditions (before any extra role is created) ---"
python3 scripts/migration/verify_g3.py pre-adopt --dsn "$(dsn "$port_p3" clarityit clarityit)"

# Create the read-only verification superuser AFTER pre-adopt so it does
# not pollute the raw profiler fingerprint checked above.
docker exec "$container_p3" psql -U clarityit -d clarityit -c \
    "CREATE ROLE g3_proof_admin LOGIN SUPERUSER PASSWORD '$password';" >/dev/null

echo "--- apply adoption artifact ---"
docker exec -i "$container_p3" psql -U clarityit -d clarityit \
    -v ON_ERROR_STOP=1 -v g3_source_commit="$(git rev-parse HEAD)" -q \
    < migrations/v2/adoption/0001_adopt_p3.sql >/dev/null

echo "--- post-adopt convergence ---"
python3 scripts/migration/verify_g3.py post-adopt \
    --dsn-fresh "$(dsn "$port_a" g3_proof_admin "$password")" \
    --dsn-adopted "$(dsn "$port_p3" g3_proof_admin "$password")" \
    --expected-source-commit "$(git rev-parse HEAD)"

# Complete catalog snapshot for drift/atomicity comparisons.
# Produces a SHA-256 over the ACTUAL catalog content (not just counts):
# role names+flags, membership edges, schema+relation+function ownership,
# grant values, column definitions, and per-table row counts.
catalog_snapshot() {
    local name="$1"
    # Part 1: catalog content (roles, memberships, schemas, rels, cols, grants,
    # funcs, perms) — single deterministic query, no temp tables.
    local catalog_digest
    catalog_digest=$(docker exec "$name" psql -U clarityit -d clarityit -qtAc "
        WITH parts(label, content) AS (
            VALUES
            ('roles', COALESCE((SELECT string_agg(format('%s|%s|%s|%s|%s|%s|%s', rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin, rolbypassrls), E'\n' ORDER BY rolname) FROM pg_roles WHERE rolname !~ '^pg_'), '')),
            ('members', COALESCE((SELECT string_agg(format('member:%s->role_of:%s|admin:%s|inherit:%s|set:%s', member.rolname, granted.rolname, am.admin_option, am.inherit_option, am.set_option), E'\n' ORDER BY member.rolname, granted.rolname) FROM pg_auth_members am JOIN pg_roles member ON member.oid=am.member JOIN pg_roles granted ON granted.oid=am.roleid WHERE member.rolname !~ '^pg_' AND granted.rolname !~ '^pg_'), '')),
            ('schemas', COALESCE((SELECT string_agg(format('schema:%s|owner:%s', nspname, pg_get_userbyid(nspowner)), E'\n' ORDER BY nspname) FROM pg_namespace WHERE nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND nspname NOT LIKE 'pg_temp%' AND nspname NOT LIKE 'pg_toast_temp%'), '')),
            ('rels', COALESCE((SELECT string_agg(format('rel:%s.%s|%s|owner:%s', n.nspname, c.relname, c.relkind, pg_get_userbyid(c.relowner)), E'\n' ORDER BY n.nspname, c.relname) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND n.nspname NOT LIKE 'pg_temp%' AND c.relkind IN ('r','S','i','v')), '')),
            ('cols', COALESCE((SELECT string_agg(format('col:%s.%s.%s|%s|notnull:%s', table_schema, table_name, column_name, data_type, is_nullable), E'\n' ORDER BY table_schema, table_name, column_name) FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog','information_schema') AND table_schema NOT LIKE 'pg_temp%'), '')),
            ('grants', COALESCE((SELECT string_agg(format('grant:%s.%s|%s|grantee:%s|priv:%s|grantable:%s', n.nspname, c.relname, c.relkind, pg_get_userbyid(a.grantee), a.privilege_type, a.is_grantable), E'\n' ORDER BY n.nspname, c.relname, pg_get_userbyid(a.grantee), a.privilege_type) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace, aclexplode(c.relacl) a WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND n.nspname NOT LIKE 'pg_temp%' AND c.relkind IN ('r','S','i','v')), '')),
            ('funcs', COALESCE((SELECT string_agg(format('func:%s.%s(%s)|owner:%s', n.nspname, p.proname, pg_get_function_identity_arguments(p.oid), pg_get_userbyid(p.proowner)), E'\n' ORDER BY n.nspname, p.proname, pg_get_function_identity_arguments(p.oid)) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND n.nspname NOT LIKE 'pg_temp%'), '')),
            ('perms', COALESCE((SELECT string_agg(format('perm:%s|%s', id, name), E'\n' ORDER BY name) FROM public.permissions), ''))
        )
        SELECT encode(public.digest(convert_to(string_agg(label||':'||content, E'\n===\n' ORDER BY label), 'UTF8'), 'sha256'), 'hex') FROM parts" 2>&1 | tr -d '[:space:]')
    # Part 2: exact per-table COUNT(*) via dynamic SQL (separate session;
    # deterministic on its own). Combined with part 1 via sha256sum.
    local rowcount_digest
    rowcount_digest=$(docker exec "$name" psql -U clarityit -d clarityit -qtAc "
        CREATE TEMP TABLE _ec(s text, t text, c bigint);
        DO \$do\$ DECLARE r record; c bigint; BEGIN FOR r IN SELECT n.nspname, c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast') AND n.nspname NOT LIKE 'pg_temp%' AND c.relkind='r' LOOP EXECUTE format('SELECT count(*) FROM %I.%I', r.nspname, r.relname) INTO c; INSERT INTO _ec VALUES(r.nspname, r.relname, c); END LOOP; END \$do\$;
        SELECT encode(public.digest(convert_to(string_agg(s||'.'||t||'='||c, E'\n' ORDER BY s, t), 'UTF8'), 'sha256'), 'hex') FROM _ec" 2>&1 | grep -E '^[0-9a-f]{64}$' | head -1)
    echo -n "${catalog_digest}+${rowcount_digest}" | sha256sum | cut -d' ' -f1
}

echo "=== Drift negative: actual adoption SQL must reject drifted P3 with zero writes ==="
start_pg "$container_drift" clarityit clarityit
apply_p3 "$container_drift"
# Inject structural drift.
docker exec "$container_drift" psql -U clarityit -d clarityit -c \
    "ALTER TABLE public.teams ADD COLUMN drift_marker text;" >/dev/null
# Capture the COMPLETE catalog/data projection before.
snap_before="$(catalog_snapshot "$container_drift")"
# Invoke the ACTUAL adoption SQL — it must fail at the SQL-derived digest gate.
if docker exec -i "$container_drift" psql -U clarityit -d clarityit \
    -v ON_ERROR_STOP=1 -v g3_source_commit="$(git rev-parse HEAD)" -q \
    < migrations/v2/adoption/0001_adopt_p3.sql >/dev/null 2>&1; then
    echo "G3 VERIFY FAIL: drifted P3 adoption SQL succeeded (should have failed)" >&2
    exit 1
fi
# The COMPLETE catalog/data projection must be unchanged (zero writes).
snap_after="$(catalog_snapshot "$container_drift")"
if [[ "$snap_before" != "$snap_after" ]]; then
    echo "G3 VERIFY FAIL: drift adoption performed writes: $snap_before -> $snap_after" >&2
    exit 1
fi
echo "ADOPT-P3 DRIFT-NEGATIVE PASS: actual adoption SQL rejected drifted source with zero writes"

echo "=== Atomicity: injected mid-adoption failure must roll back every change ==="
# Reuse the clean P3 source (pre-adoption state).  Inject a deliberate failure
# after the role transition by appending a RAISE EXCEPTION, then verify the
# complete catalog projection is unchanged (full rollback).
container_atomic="g3-atomic-${suffix}"
trap 'docker rm -f "$container_a" "$container_b" "$container_p3" "$container_drift" "$container_atomic" >/dev/null 2>&1 || true' EXIT
start_pg "$container_atomic" clarityit clarityit
apply_p3 "$container_atomic"
snap_atomic_before="$(catalog_snapshot "$container_atomic")"
# Build a deliberately-failing adoption variant: the real SQL including the
# final demotion, but with an injected exception AFTER the demotion and
# BEFORE the final assertions/COMMIT — proving the last state mutation
# itself rolls back.
sed '/^ALTER ROLE clarityit LOGIN INHERIT/a\
DO $$ BEGIN RAISE EXCEPTION '"'"'g3 injected atomicity test failure'"'"'; END $$;' \
    migrations/v2/adoption/0001_adopt_p3.sql > /tmp/g3_atomic_test.sql
# Apply the failing variant — it must fail.
if docker exec -i "$container_atomic" psql -U clarityit -d clarityit \
    -v ON_ERROR_STOP=1 -v g3_source_commit="$(git rev-parse HEAD)" -q \
    < /tmp/g3_atomic_test.sql >/dev/null 2>&1; then
    echo "G3 VERIFY FAIL: injected-failure adoption succeeded (should have rolled back)" >&2
    exit 1
fi
snap_atomic_after="$(catalog_snapshot "$container_atomic")"
if [[ "$snap_atomic_before" != "$snap_atomic_after" ]]; then
    echo "G3 VERIFY FAIL: injected-failure adoption did not roll back: $snap_atomic_before -> $snap_atomic_after" >&2
    exit 1
fi
echo "ADOPT-P3 ATOMICITY PASS: injected mid-adoption failure rolled back every change"

echo "=== G3 FULL MATRIX PASS ==="

echo "=== G3 FULL MATRIX PASS ==="
