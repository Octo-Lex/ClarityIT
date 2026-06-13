#!/bin/bash
# ClarityIT MinIO Backup Script
# Usage: ./scripts/backup-minio.sh
# Backs up the MinIO Docker volume directly.
# Output: backups/minio_<timestamp>.tar.gz

set -euo pipefail

BACKUP_DIR="/opt/clarityit/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/minio_${TIMESTAMP}.tar.gz"

mkdir -p "${BACKUP_DIR}"

echo "=== ClarityIT MinIO Backup ==="
echo "Timestamp: ${TIMESTAMP}"
echo "Output: ${BACKUP_FILE}"

# Back up the Docker named volume directly
# The volume path is managed by Docker and contains the MinIO data + metadata
MINIO_VOL="/var/lib/docker/volumes/clarityit_miniodata/_data"

if [ ! -d "${MINIO_VOL}" ]; then
    echo "ERROR: MinIO volume not found at ${MINIO_VOL}"
    echo "Falling back to container-based backup..."
    docker exec clarityit-minio-1 tar czf - /data | cat > "${BACKUP_FILE}"
else
    tar czf "${BACKUP_FILE}" -C "${MINIO_VOL}" .
fi

SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "Backup complete: ${SIZE}"

# Verify backup is not empty
if [ ! -s "${BACKUP_FILE}" ]; then
    echo "ERROR: Backup file is empty!"
    exit 1
fi

# Verify backup contains expected MinIO structure
if ! tar tzf "${BACKUP_FILE}" | grep -q '.minio.sys'; then
    echo "WARNING: MinIO metadata not found in backup — verify contents"
fi

# Rotate: keep last 10 MinIO backups
ls -t "${BACKUP_DIR}"/minio_*.tar.gz | tail -n +11 | xargs -r rm --

echo "Old backups rotated (keeping last 10)"
