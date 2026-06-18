# Backup/Restore Drill Report

## Date: 2026-06-13 15:27 UTC+3

## Source

| Parameter | Value |
|-----------|-------|
| System | ClarityIT v0.9.0 (dev) |
| Backup timestamp | 20260613_152736 |
| PostgreSQL backup | `postgresql_20260613_152736.sql.gz` (176K) |
| MinIO backup | `minio_20260613_152736.tar.gz` (8.0K) |
| Source host | CT 150 (<your-host>) |

## Restore Target

| Parameter | Value |
|-----------|-------|
| Method | Temporary database on same PostgreSQL instance |
| Target database | `clarityit_restore_test` |

## Commands Run

```bash
# Create backup
./scripts/backup-postgres.sh
./scripts/backup-minio.sh
./scripts/verify-backup.sh

# Restore to temporary database
docker exec clarityit-postgres-1 psql -U clarityit -d postgres -c "CREATE DATABASE clarityit_restore_test"
gunzip -c /opt/clarityit/backups/postgresql_20260613_152736.sql.gz | \
  docker exec -i clarityit-postgres-1 psql -U clarityit -d clarityit_restore_test

# Verify
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit_restore_test -c \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'"
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit_restore_test -c \
  "SELECT count(*) FROM users"

# Cleanup
docker exec clarityit-postgres-1 psql -U clarityit -d postgres -c "DROP DATABASE clarityit_restore_test"
```

## Results

| Check | Expected | Actual | Status |
|-------|----------|--------|--------|
| Tables restored | 38+ | 38 | ✅ Pass |
| Users restored | 2 | 2 | ✅ Pass |
| Backup file non-empty | Yes | 176K | ✅ Pass |
| Restore completed | No errors | Clean | ✅ Pass |
| Drop cleanup | Success | Success | ✅ Pass |

## Time to Restore

- Backup creation: ~3 seconds
- Restore to temp DB: ~5 seconds
- Total: ~8 seconds

## Issues Found

None.

## MinIO Backup Verification

MinIO backup verified via `./scripts/verify-backup.sh`:
- File exists: ✅
- Non-empty: ✅ (8.0K)
- Contains `.minio.sys` metadata: ✅

## Conclusion

Backup and restore path is functional. PostgreSQL backup contains all 38 tables with correct data. The restore drill was completed without issues.

For a full fresh-machine restore, follow the steps in `docs/ops/backup-restore.md` under "Fresh VM/LXC Recovery".
