# Admin Runbook — ClarityIT

## Service Management

```bash
# Start all services
docker compose up -d

# Stop all services
docker compose down

# Restart a single service
docker compose restart clarityit-api

# View logs
docker compose logs -f clarityit-api
docker compose logs -f clarityit-outbox-worker
docker compose logs -f clarityit-reasoning-worker

# Check service health
make verify-deployment
```

## Health Checks

```bash
# Basic health
curl http://localhost:8765/health

# Deep health (authenticated)
TOKEN=$(curl -sf -X POST http://localhost:8765/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"xxx"}' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])')
curl -H "Authorization: Bearer $TOKEN" http://localhost:8765/api/health/deep

# Prometheus metrics
curl http://localhost:8765/metrics
```

## Outbox Lag

If events aren't being processed:

```bash
# Check pending count
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL"

# Check dead letters
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT COUNT(*) FROM outbox_events WHERE dead_lettered_at IS NOT NULL"

# Restart outbox worker
docker compose restart clarityit-outbox-worker
```

## Dead Letters

```bash
# List recent dead letters
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT id::text, event_type, error_message, dead_lettered_at FROM outbox_events WHERE dead_lettered_at IS NOT NULL ORDER BY dead_lettered_at DESC LIMIT 10"
```

## Rotate Secrets

### JWT Secret

1. Generate new: `openssl rand -base64 48`
2. Update `.env` with new `JWT_SECRET`
3. Restart: `docker compose restart clarityit-api`
4. All existing sessions invalidated — users must re-login

### HMAC Key

1. Generate new: `openssl rand -base64 48`
2. Update `.env` with new `HMAC_KEY`
3. Restart: `docker compose restart clarityit-api`
4. All integration keys must be regenerated

## Backup and Restore

```bash
# Backup
./scripts/backup-postgres.sh
./scripts/backup-minio.sh

# Verify backups
./scripts/verify-backup.sh

# Restore (WARNING: replaces all data)
./scripts/restore-postgres.sh backups/postgresql_YYYYMMDD_HHMMSS.sql.gz
```

See [Backup/Restore](backup-restore.md) for full documentation.

## Reset Bootstrap (Dev Only)

```bash
# WARNING: This destroys all data
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "TRUNCATE users, teams, bootstrap_lock CASCADE"
# Re-run migrations
make migrate
# Re-bootstrap via POST /api/bootstrap
```

## Troubleshooting

### Login Issues
- Check JWT_SECRET is set and consistent across restarts
- Check user exists: `SELECT email FROM users WHERE email='...'`
- Check sessions table for token validity

### Webhook Ingestion Issues
- Check integration key is not revoked: `SELECT * FROM integration_api_keys WHERE key_prefix='clarity_...'`
- Check rate limits: `curl http://localhost:8765/metrics | grep webhook_rate_limited`
- Check source/scope match the webhook URL and key config

### Agent Worker Failures
- Verify WORKER_TOKEN is set in reasoning worker env
- Verify TEAM_ID is set and valid
- Check worker logs: `docker compose logs clarityit-reasoning-worker`
- Worker exits without WORKER_TOKEN — this is by design

### Stuck Workers
```bash
# Check worker health via deep health
curl -H "Authorization: Bearer $TOKEN" http://localhost:8765/api/health/deep | python3 -m json.tool

# Restart all workers
docker compose restart clarityit-outbox-worker clarityit-context-worker clarityit-reasoning-worker
```
