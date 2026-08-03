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

echo "=== Governed fingerprint: fresh A == fresh B ==="
fp_a="$(python3 scripts/migration/verify_g3.py governed --dsn "$(dsn "$port_a" g3_proof_admin "$password")")"
fp_b="$(python3 scripts/migration/verify_g3.py governed --dsn "$(dsn "$port_b" g3_proof_admin "$password")")"
if [[ "$fp_a" != "$fp_b" ]]; then
    echo "G3 VERIFY FAIL: fresh governed fingerprints differ: $fp_a vs $fp_b" >&2
    exit 1
fi
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
    -v ON_ERROR_STOP=1 -v g3_source_commit="$(git rev-parse HEAD)" \
    -v g3_source_fingerprint="$P3_FP" -q \
    < migrations/v2/adoption/0001_adopt_p3.sql >/dev/null

echo "--- post-adopt convergence ---"
python3 scripts/migration/verify_g3.py post-adopt \
    --dsn-fresh "$(dsn "$port_a" g3_proof_admin "$password")" \
    --dsn-adopted "$(dsn "$port_p3" g3_proof_admin "$password")"

echo "=== Drift negative: drifted P3 must fail pre-adopt with zero writes ==="
start_pg "$container_drift" clarityit clarityit
apply_p3 "$container_drift"
# Inject structural drift.
docker exec "$container_drift" psql -U clarityit -d clarityit -c \
    "ALTER TABLE public.teams ADD COLUMN drift_marker text;" >/dev/null
port_drift="$(port_of "$container_drift")"
# Capture catalog snapshot before (no extra role yet).
snap_before="$(docker exec "$container_drift" psql -U clarityit -d clarityit -tAc \
    "SELECT (SELECT count(*) FROM pg_roles WHERE rolname !~ '^pg_') || ':' || (SELECT count(*) FROM pg_namespace WHERE nspname NOT IN ('pg_catalog','information_schema','pg_toast')) || ':' || (SELECT count(*) FROM information_schema.columns WHERE table_schema='public')")"
# pre-adopt must fail (run before creating any extra role).
if python3 scripts/migration/verify_g3.py pre-adopt --dsn "$(dsn "$port_drift" clarityit clarityit)" 2>/dev/null; then
    echo "G3 VERIFY FAIL: drifted P3 passed pre-adopt (should have failed)" >&2
    exit 1
fi
# Catalog must be unchanged (zero writes).
snap_after="$(docker exec "$container_drift" psql -U clarityit -d clarityit -tAc \
    "SELECT (SELECT count(*) FROM pg_roles WHERE rolname !~ '^pg_') || ':' || (SELECT count(*) FROM pg_namespace WHERE nspname NOT IN ('pg_catalog','information_schema','pg_toast')) || ':' || (SELECT count(*) FROM information_schema.columns WHERE table_schema='public')")"
if [[ "$snap_before" != "$snap_after" ]]; then
    echo "G3 VERIFY FAIL: drift pre-adopt performed writes: $snap_before -> $snap_after" >&2
    exit 1
fi
echo "ADOPT-P3 DRIFT-NEGATIVE PASS: drifted source rejected with zero writes ($snap_before unchanged)"

echo "=== G3 FULL MATRIX PASS ==="
