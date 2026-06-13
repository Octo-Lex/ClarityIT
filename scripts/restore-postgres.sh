#!/bin/bash
# ClarityIT PostgreSQL Restore Script
# Usage: ./scripts/restore-postgres.sh <backup_file>
# WARNING: This replaces all data. Requires explicit confirmation.

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    echo "Available backups:"
    ls -la /opt/clarityit/backups/postgresql_*.sql.gz 2>/dev/null || echo "  No backups found"
    exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "${BACKUP_FILE}" ]; then
    echo "ERROR: File not found: ${BACKUP_FILE}"
    exit 1
fi

echo "=== ClarityIT PostgreSQL Restore ==="
echo "Backup: ${BACKUP_FILE}"
echo ""
echo "WARNING: This will DROP and recreate all data!"
echo "Type 'RESTORE' to confirm:"
read -r CONFIRMATION

if [ "${CONFIRMATION}" != "RESTORE" ]; then
    echo "Aborted."
    exit 1
fi

echo "Restoring..."
gunzip -c "${BACKUP_FILE}" | docker exec -i clarityit-postgres-1 psql -U clarityit -d clarityit

echo "Restore complete. Run 'make verify-deployment' to verify."
