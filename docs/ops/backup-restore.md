# Backup and Restore — ClarityIT

## What Gets Backed Up

| Component | Data | Method | Verified |
|-----------|------|--------|----------|
| PostgreSQL | All tables (37+), migrations state | `pg_dump` | ✅ Smoke-tested (96K dump) |
| MinIO | Uploaded files/attachments + bucket metadata | Docker volume `tar` | ✅ Smoke-tested (4.8K tarball, `.minio.sys` verified) |

## What Is NOT Backed Up

- `.env` file (contains secrets — manage separately)
- Docker images (rebuild from source)
- Redis state (ephemeral cache — repopulated on restart)
- NATS state (ephemeral transport — reprocessed from outbox)

## Automated Backup

```bash
# PostgreSQL
./scripts/backup-postgres.sh

# MinIO
./scripts/backup-minio.sh

# Verify
./scripts/verify-backup.sh
```

Add to crontab for daily backups:
```cron
0 2 * * * /opt/clarityit/scripts/backup-postgres.sh >> /var/log/clarityit-backup.log 2>&1
0 3 * * * /opt/clarityit/scripts/backup-minio.sh >> /var/log/clarityit-backup.log 2>&1
```

## Restore

### PostgreSQL Restore

```bash
# List available backups
ls -la /opt/clarityit/backups/postgresql_*.sql.gz

# Restore specific backup (prompts for confirmation)
./scripts/restore-postgres.sh /opt/clarityit/backups/postgresql_20260613_020000.sql.gz
```

### MinIO Restore

MinIO data lives in the Docker named volume `clarityit_miniodata`. To restore:

```bash
# Stop MinIO
docker compose stop minio

# Restore volume from backup
tar xzf /opt/clarityit/backups/minio_TIMESTAMP.tar.gz -C /var/lib/docker/volumes/clarityit_miniodata/_data/

# Start MinIO
docker compose start minio
```

On a fresh machine, create the volume first:
```bash
docker volume create clarityit_miniodata
tar xzf backup.tar.gz -C /var/lib/docker/volumes/clarityit_miniodata/_data/
```

On a new machine:

```bash
# 1. Install Docker
curl -fsSL https://get.docker.com | sh

# 2. Get the code
git clone <repo-url> /opt/clarityit
cd /opt/clarityit

# 3. Configure environment
cp services/api/.env.example .env
# Edit .env with production values

# 4. Start services
docker compose up -d

# 5. Wait for PostgreSQL to be healthy
docker compose ps

# 6. Restore backup
./scripts/restore-postgres.sh /path/to/postgresql_TIMESTAMP.sql.gz

# 7. Verify
make verify-deployment
```

## Backup Rotation

- PostgreSQL: Last 30 backups retained
- MinIO: Last 10 backups retained
- Rotation happens automatically during backup

## Migration Rollback

Migrations are numbered sequentially (001-019+). To rollback:

```bash
# Check current migration state
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename"

# Rollback specific migration manually (requires SQL knowledge)
# Each migration file is in migrations/NNN_description.sql
# Reverse the operations carefully
```

**Important**: There is no automated rollback. Test migrations on a backup before applying to production.
