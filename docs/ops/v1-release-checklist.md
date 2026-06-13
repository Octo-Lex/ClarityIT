# v1.0.0 Release Checklist — ClarityIT

## Pre-Release

### Code Quality
- [ ] `go vet ./...` passes clean
- [ ] `go test -p 1 -count=1 -timeout 180s ./...` passes (309 tests)
- [ ] `cd web && npx vitest run` passes (33 tests)
- [ ] Python model gateway tests pass (9 tests)
- [ ] No compiler warnings

### Security
- [ ] `make audit-prod` passes clean (0 production vulnerabilities)
- [ ] `npm audit --omit=dev --audit-level=high` → 0 vulnerabilities
- [ ] `pip check` → no broken requirements
- [ ] Secret leakage scan clean (no hardcoded secrets)
- [ ] All 16 security tests pass
- [ ] `docs/security/risk-acceptance-v1.md` reviewed and current

### Database
- [ ] All migrations applied (001-025)
- [ ] 47 tables present
- [ ] 121 permissions seeded
- [ ] 10 team roles + 3 platform roles seeded

### Backup
- [ ] PostgreSQL backup completed and verified
- [ ] MinIO backup completed and verified
- [ ] Restore drill passed (temp DB restore, table/user/permission count verified)
- [ ] Backup rotation working (30 PostgreSQL / 10 MinIO retained)

## Deployment

### Core Services
- [ ] `docker compose up -d --build` completes
- [ ] All 8 core services healthy (api, web, postgres, nats, redis, minio, outbox-worker, context-worker)
- [ ] `curl -sf http://192.168.3.20:8765/health` returns OK
- [ ] `curl -sf http://192.168.3.20:3000/` returns HTML
- [ ] Deep health check passes (authenticated)

### Agent Profile
- [ ] `docker compose --profile agent up -d` starts reasoning worker
- [ ] Reasoning worker connects to API (no forbidden env vars)
- [ ] Worker polls agent runs successfully

### Container Hardening
- [ ] API container runs as uid=100 (non-root)
- [ ] Web container runs as uid=101 (nginx-unprivileged)
- [ ] Web container read_only=true
- [ ] Web container no-new-privileges=true
- [ ] No privileged containers
- [ ] Only ports 8765 and 3000 publicly exposed

### Proxmox Integration
- [ ] Read-only sync works against 192.168.3.5
- [ ] `PROXMOX_MUTATION_ENABLED=false` by default
- [ ] Mutation validation completed safely (snapshot/start/shutdown)
- [ ] `PROXMOX_MUTATION_ENABLED` reset to false after validation

## Post-Deployment

### Functional Verification
- [ ] Bootstrap lock prevents second bootstrap
- [ ] Login + token rotation works
- [ ] MFA enrollment + verification works
- [ ] Approval workflow: create → approve → verify
- [ ] Asset action: create → approve → execute (with mutation disabled, verify 403)
- [ ] Remediation: create → approve → execute through Tool Gateway
- [ ] WebSocket invalidation works
- [ ] No raw secrets in API responses

### Playwright E2E
- [ ] All v0.9.0 smoke tests pass (11 tests)
- [ ] All v1.0 UI tests pass (12 tests)
- [ ] No sensitive data visible in any page

### Operator Documentation
- [ ] `docs/ops/approval-operations.md` reviewed
- [ ] `docs/ops/mfa-recovery.md` reviewed
- [ ] `docs/ops/proxmox-actions.md` reviewed
- [ ] `docs/ops/remediation-runbook.md` reviewed

## Rollback Path

If v1.0.0 must be rolled back:

1. Stop services: `docker compose down`
2. Restore PostgreSQL: `./scripts/restore-postgres.sh /opt/clarityit/backups/postgresql_TIMESTAMP.sql.gz`
3. Checkout previous tag: `git checkout v0.9.0-operator-readiness`
4. Rebuild: `docker compose up -d --build`
5. Verify: `make verify-deployment`

**Note**: Migrations 022-025 add new tables but do not modify existing ones. Rolling back the code without rolling back the database is safe — the new tables will exist but be unused.

## Sign-Off

- [ ] Release engineer verified all checks
- [ ] Security review accepted (Track 7)
- [ ] Backup/restore drill passed
- [ ] Playwright E2E passed
- [ ] Tag created: `v1.0.0-sovereign-operations`
