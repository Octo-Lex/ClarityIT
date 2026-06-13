#!/bin/bash
# ClarityIT Backup Verification Script
# Usage: ./scripts/verify-backup.sh
# Checks that recent backups exist and are non-empty.

set -euo pipefail

BACKUP_DIR="/opt/clarityit/backups"
ERRORS=0

echo "=== ClarityIT Backup Verification ==="

# Check PostgreSQL backup
echo -n "PostgreSQL: "
PG_LATEST=$(ls -t "${BACKUP_DIR}"/postgresql_*.sql.gz 2>/dev/null | head -1)
if [ -z "${PG_LATEST}" ]; then
    echo "MISSING - no PostgreSQL backup found"
    ERRORS=$((ERRORS + 1))
elif [ ! -s "${PG_LATEST}" ]; then
    echo "EMPTY - ${PG_LATEST}"
    ERRORS=$((ERRORS + 1))
else
    SIZE=$(du -h "${PG_LATEST}" | cut -f1)
    AGE=$(( ($(date +%s) - $(date +%s -r "${PG_LATEST}")) / 3600 ))
    echo "OK (${SIZE}, ${AGE}h old)"
fi

# Check MinIO backup
echo -n "MinIO: "
M_LATEST=$(ls -t "${BACKUP_DIR}"/minio_*.tar.gz 2>/dev/null | head -1)
if [ -z "${M_LATEST}" ]; then
    echo "MISSING - no MinIO backup found"
    ERRORS=$((ERRORS + 1))
elif [ ! -s "${M_LATEST}" ]; then
    echo "EMPTY - ${M_LATEST}"
    ERRORS=$((ERRORS + 1))
else
    SIZE=$(du -h "${M_LATEST}" | cut -f1)
    AGE=$(( ($(date +%s) - $(date +%s -r "${M_LATEST}")) / 3600 ))
    echo "OK (${SIZE}, ${AGE}h old)"
fi

# Summary
if [ ${ERRORS} -eq 0 ]; then
    echo "All backups verified."
else
    echo "WARNING: ${ERRORS} issue(s) found!"
    exit 1
fi
