# Incident Response — ClarityIT

## Severity Levels

| Level | Description | Response Time |
|-------|-------------|---------------|
| P1 | Service completely down | 15 minutes |
| P2 | Major feature broken | 1 hour |
| P3 | Minor feature degraded | 4 hours |
| P4 | Cosmetic or non-critical | Next business day |

## P1: Complete Outage

1. **Assess**: `make verify-deployment`
2. **Check**: `docker compose ps` — are containers running?
3. **Logs**: `docker compose logs --tail=50 clarityit-api`
4. **Recovery**:
   ```bash
   docker compose restart clarityit-api
   # If that fails:
   docker compose down && docker compose up -d --build
   ```
5. **Verify**: `curl http://localhost:8765/health`
6. **Notify**: Post to operations channel

## P1: Database Down

1. **Check**: `docker compose ps clarityit-postgres-1`
2. **Logs**: `docker compose logs clarityit-postgres-1 --tail=30`
3. **Recovery**:
   ```bash
   docker compose restart clarityit-postgres-1
   # Wait for healthy
   docker compose ps  # Should show "(healthy)"
   # Restart dependent services
   docker compose restart clarityit-api clarityit-outbox-worker clarityit-context-worker
   ```
4. **Worst case**: Restore from backup
   ```bash
   ./scripts/restore-postgres.sh backups/postgresql_TIMESTAMP.sql.gz
   ```

## P2: Webhooks Not Processing

1. **Check integration key**: `SELECT * FROM integration_api_keys WHERE revoked_at IS NULL`
2. **Check rate limits**: `curl localhost:8765/metrics | grep webhook_rate_limited`
3. **Check outbox**: `SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL`
4. **Restart workers**: `docker compose restart clarityit-outbox-worker`

## P2: Agent Tool Execution Blocked

1. **Check**: `SELECT * FROM agent_effect_results WHERE status='blocked' ORDER BY created_at DESC LIMIT 10`
2. **Verify**: Tool exists in registry, grant is valid, autonomy level is within bounds
3. **Resolution**: Update tool registry or grant configuration

## P3: Outbox Lag Growing

1. **Monitor**: `curl localhost:8765/metrics | grep outbox_pending`
2. **Check outbox worker**: `docker compose logs clarityit-outbox-worker --tail=20`
3. **Check NATS**: `docker compose logs clarityit-nats-1 --tail=20`
4. **Restart**: `docker compose restart clarityit-outbox-worker`

## Post-Incident

1. **Document**: What happened, timeline, root cause, resolution
2. **Action items**: Prevent recurrence
3. **Review**: Check audit logs for the incident period
4. **Backup**: Take a backup after recovery
