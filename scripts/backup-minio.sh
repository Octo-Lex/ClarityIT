#!/bin/bash
# ClarityIT MinIO Backup Script
# Usage: ./scripts/backup-minio.sh
# Requires: mc (MinIO Client) or docker cp
# Output: backups/minio_<timestamp>.tar.gz

set -euo pipefail

BACKUP_DIR="/opt/clarityit/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/minio_${TIMESTAMP}.tar.gz"

mkdir -p "${BACKUP_DIR}"

echo "=== ClarityIT MinIO Backup ==="
echo "Timestamp: ${TIMESTAMP}"
echo "Output: ${BACKUP_FILE}"

# Use docker cp to backup MinIO data
docker exec clarityit-minio-1 tar czf - /data | cat > "${BACKUP_FILE}"

SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "Backup complete: ${SIZE}"

# Rotate: keep last 10 MinIO backups
ls -t "${BACKUP_DIR}"/minio_*.tar.gz | tail -n +11 | xargs -r rm --

echo "Old backups rotated (keeping last 10)"
