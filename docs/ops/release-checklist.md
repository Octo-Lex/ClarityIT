# Release Checklist — ClarityIT v0.8.0

## Pre-Release

- [ ] All tests pass: `cd services/api && go test -p 1 -count=1 -timeout 180s ./...`
- [ ] Frontend tests pass: `cd web && npm test`
- [ ] Python tests pass: `cd services/workers/reasoning && python -m pytest test_model_gateway.py -v`
- [ ] Golden-thread tests pass: `cd services/api && go test -run TestGolden ./cmd/api/`
- [ ] No secrets in code: `git grep -i 'password.*=.*\"[a-z]' -- '*.go' '*.py' '*.ts'`
- [ ] Config validation passes in production mode
- [ ] Metrics endpoint responds: `curl /metrics`
- [ ] Deep health passes: `curl /api/health/deep`
- [ ] Backup scripts tested: `./scripts/backup-postgres.sh && ./scripts/verify-backup.sh`

## Build

- [ ] Version bumped in code (config, main.go, health handler)
- [ ] Git tag created: `git tag -a v0.8.0-hardening -m "..."`
- [ ] Docker images build cleanly: `docker compose build`

## Deploy

- [ ] Backup database before deploy: `./scripts/backup-postgres.sh`
- [ ] Deploy to staging: `docker compose up -d --build`
- [ ] Run migrations: `make migrate`
- [ ] Verify deployment: `make verify-deployment`
- [ ] Smoke test: login, create object, webhook, agent run
- [ ] Check logs for errors: `docker compose logs --tail=50 clarityit-api`

## Post-Release

- [ ] Update CHANGELOG
- [ ] Notify operators of changes
- [ ] Monitor metrics for 1 hour
- [ ] Verify backup schedule still running
