# Troubleshooting Guide — ClarityIT

## Service Won't Start

```bash
# Check logs
docker compose logs clarityit-api --tail=50

# Common causes:
# - Config validation failure (missing JWT_SECRET, weak HMAC_KEY)
# - Database connection failure (check DATABASE_URL)
# - Port already in use (check PORT)

# Fix: verify .env configuration
cat .env | grep -E 'JWT_SECRET|HMAC_KEY|DATABASE_URL'
```

## Database Connection Errors

```bash
# Check PostgreSQL is running
docker compose ps clarityit-postgres-1

# Test connection
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit -c "SELECT 1"

# Check URL format
# Correct: postgres://user:pass@hostname:5432/dbname?sslmode=disable
# Docker network: postgres://clarityit:clarityit@postgres:5432/clarityit
```

## Outbox Events Stuck

```bash
# Count pending
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL"

# Check outbox worker logs
docker compose logs clarityit-outbox-worker --tail=20

# Restart worker
docker compose restart clarityit-outbox-worker
```

## Webhooks Return 401/403

```bash
# 401 = invalid integration key
# - Check key is not revoked
# - Check key hash matches (HMAC-SHA256)

# 403 = source or scope not allowed
# - Check allowed_sources includes the webhook source
# - Check allowed_scopes includes 'webhooks:ingest'

# 429 = rate limited
# - Wait 60 seconds and retry
# - Check metrics: curl localhost:8765/metrics | grep rate_limited
```

## Reasoning Worker Restarts

```bash
# Check logs
docker compose logs clarityit-reasoning-worker --tail=20

# Common causes:
# - WORKER_TOKEN not set → exits immediately
# - TEAM_ID not set → exits immediately
# - Forbidden env vars set (DATABASE_URL, NATS_URL, etc.)
# - API unreachable → check API_BASE_URL
```

## Login Session Issues

```bash
# Check user exists
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT id, email FROM users WHERE email='user@example.com'"

# Check sessions
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT id, user_id, expires_at FROM sessions WHERE user_id='<uuid>' ORDER BY created_at DESC LIMIT 5"

# Force logout: truncate sessions
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "DELETE FROM sessions WHERE user_id='<uuid>'"
```

## Context Graph Empty

Context nodes are populated by the context worker from NATS events.

```bash
# Check context worker is running
docker compose ps clarityit-context-worker

# Check worker logs
docker compose logs clarityit-context-worker --tail=20

# Check context nodes
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT COUNT(*) FROM context_nodes"
```

## Agent Runs Stuck in Pending

```bash
# Check reasoning worker
docker compose ps clarityit-reasoning-worker
docker compose logs clarityit-reasoning-worker --tail=20

# If worker is restarting, check WORKER_TOKEN and TEAM_ID
docker exec clarityit-reasoning-worker env | grep -E 'WORKER_TOKEN|TEAM_ID'
```
