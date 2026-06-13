#!/bin/bash
# ClarityIT PostgreSQL Backup Script
# Usage: ./scripts/backup-postgres.sh
# Requires: docker, docker compose
# Output: backups/postgresql_<timestamp>.sql.gz

set -euo pipefail

BACKUP_DIR="/opt/clarityit/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/postgresql_${TIMESTAMP}.sql.gz"

mkdir -p "${BACKUP_DIR}"

echo "=== ClarityIT PostgreSQL Backup ==="
echo "Timestamp: ${TIMESTAMP}"
echo "Output: ${BACKUP_FILE}"

docker exec clarityit-postgres-1 pg_dump -U clarityit -d clarityit --clean --if-exists | gzip > "${BACKUP_FILE}"

SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "Backup complete: ${SIZE}"

# Verify backup is not empty
if [ ! -s "${BACKUP_FILE}" ]; then
    echo "ERROR: Backup file is empty!"
    exit 1
fi

# Rotate: keep last 30 backups
ls -t "${BACKUP_DIR}"/postgresql_*.sql.gz | tail -n +31 | xargs -r rm --

echo "Old backups rotated (keeping last 30)"
