#!/usr/bin/env bash
# WP-00 G3 validation. Default: static + two independent PostgreSQL 16 installs.
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

suffix="$$"
container_a="g3-fresh-a-${suffix}"
container_b="g3-fresh-b-${suffix}"
password="g3-local-proof-only"

cleanup() {
    docker rm -f "$container_a" "$container_b" >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_fresh() {
    local name="$1"
    docker run -d --name "$name" \
        -e POSTGRES_USER=g3bootstrap \
        -e POSTGRES_PASSWORD="$password" \
        -e POSTGRES_DB=clarityit \
        -p 127.0.0.1::5432 \
        postgres:16-alpine >/dev/null

    local ready=0
    for _ in $(seq 1 90); do
        if docker exec "$name" psql -U g3bootstrap -d clarityit -tAc "SELECT 1" >/dev/null 2>&1; then
            sleep 1
            if docker exec "$name" psql -U g3bootstrap -d clarityit -tAc "SELECT 1" >/dev/null 2>&1; then
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
        docker exec -i "$name" psql -U g3bootstrap -d clarityit -v ON_ERROR_STOP=1 < "$artifact"
    done
}

start_fresh "$container_a"
start_fresh "$container_b"
install_g3 "$container_a"
install_g3 "$container_b"

port_a="$(docker port "$container_a" 5432/tcp | sed -E 's/.*:([0-9]+)$/\1/')"
port_b="$(docker port "$container_b" 5432/tcp | sed -E 's/.*:([0-9]+)$/\1/')"

python3 scripts/migration/verify_g3.py live \
    --dsn-a "host=127.0.0.1 port=${port_a} dbname=clarityit user=g3bootstrap password=${password}" \
    --dsn-b "host=127.0.0.1 port=${port_b} dbname=clarityit user=g3bootstrap password=${password}"
