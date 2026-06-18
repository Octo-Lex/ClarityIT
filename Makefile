.PHONY: build test test-iam test-team deploy audit audit-prod verify-deployment

# Deployment target — override for your environment: make deploy DEPLOY_HOST=10.0.0.5
DEPLOY_HOST ?= localhost
API_PORT    ?= 8765
WEB_PORT    ?= 3000

# Docker run helper for Go commands
GO_TEST = docker run --rm -v /opt/clarityit/services/api:/app -w /app --network clarityit_clarityit-net golang:1.25-alpine

# Build the Go binary
build:
	$(GO_TEST) go build ./cmd/api

# Run all tests sequentially (shared DB requires -p 1)
test:
	$(GO_TEST) go test -p 1 -count=1 -timeout 300s ./...

# Run only IAM tests
test-iam:
	$(GO_TEST) go test -v -count=1 -timeout 60s ./internal/iam/

# Run only team tests
test-team:
	$(GO_TEST) go test -v -count=1 -timeout 60s ./internal/team/

# Run only domain tests
test-domain:
	$(GO_TEST) go test -v -count=1 -timeout 60s ./internal/domain/

# Run only outbox worker tests
test-outbox:
	$(GO_TEST) go test -v -count=1 -timeout 60s ./cmd/outbox-worker/

# Run only context tests
test-context:
	$(GO_TEST) go test -v -count=1 -timeout 60s ./internal/contextx/

# Run only WebSocket tests
test-ws:
	$(GO_TEST) go test -v -count=1 -timeout 30s ./internal/wsx/

# Build and deploy via Docker Compose
deploy:
	cd /opt/clarityit && docker compose up -d --build clarityit-api

# Build and deploy web frontend
deploy-web:
	cd /opt/clarityit && docker compose up -d --build clarityit-web

# Deploy all services
deploy-all:
	cd /opt/clarityit && docker compose up -d --build

# Apply pending migrations
migrate:
	for f in /opt/clarityit/migrations/*.sql; do \
		docker exec -i clarityit-postgres-1 psql -U clarityit -d clarityit < "$$f"; \
	done

# Live pipeline integration test
verify-pipeline:
	$(GO_TEST) go test -v -count=1 -run TestPhase5_LivePipeline ./cmd/context-worker/ -timeout 60s
	$(GO_TEST) go test -v -count=1 -run TestPhase5_OutboxWorker ./cmd/outbox-worker/ -timeout 60s

# Verify docker compose deployment
verify-deploy:
	curl -sf http://$(DEPLOY_HOST):$(API_PORT)/health
	docker logs clarityit-outbox-worker 2>&1 | tail -3
	docker logs clarityit-context-worker 2>&1 | tail -3

# Security and dependency audit
audit:
	@echo "=== Go Vet ==="
	$(GO_TEST) go vet ./...
	@echo "=== Go Tests ==="
	$(GO_TEST) go test -p 1 -count=1 -timeout 300s ./...
	@echo "=== Production Dependency Audit (runtime only) ==="
	cd web && npm audit --omit=dev --audit-level=high
	@echo "=== Dev Dependency Audit (informational) ==="
	cd web && npm audit --audit-level=high 2>&1 || echo "  (dev-only findings — see docs/security/risk-acceptance-v1.md)"
	@echo "=== Python Check ==="
	cd services/workers/reasoning && python3 -m pip check 2>&1 || true
	@echo "=== Audit Complete ==="

# Production-only dependency audit (must be clean)
audit-prod:
	@echo "=== Production Runtime Dependencies ==="
	cd web && npm audit --omit=dev --audit-level=high
	@echo "=== Python Runtime Dependencies ==="
	cd services/workers/reasoning && python3 -m pip check
	@echo "=== Go Module Check ==="
	$(GO_TEST) go vet ./...
	@echo "=== Production audit clean ==="

# Full deployment verification
verify-deployment:
	@echo "=== Web Frontend ==="
	@curl -sf http://$(DEPLOY_HOST):$(WEB_PORT) | head -c 100 && echo "... OK" || echo "FAIL"
	@echo "=== API Health ==="
	@curl -sf http://$(DEPLOY_HOST):$(API_PORT)/health || echo "FAIL"
	@echo ""
	@echo "=== Deep Health (requires token) ==="
	@TOKEN=$$(curl -sf -X POST http://$(DEPLOY_HOST):$(API_PORT)/api/auth/login -H 'Content-Type: application/json' -d "{\"email\":\"$$CLARITY_TEST_EMAIL\",\"password\":\"$$CLARITY_TEST_PASSWORD\"}" | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])') && curl -sf http://$(DEPLOY_HOST):$(API_PORT)/api/health/deep -H "Authorization: Bearer $$TOKEN" | python3 -m json.tool || echo "FAIL"
	@echo "=== Docker Services ==="
	@docker compose -f /opt/clarityit/docker-compose.yml ps --format "table {{.Name}}\t{{.Status}}"
	@echo "=== Port Exposure Check ==="
	@echo "PostgreSQL:" && (curl -sf --max-time 2 http://$(DEPLOY_HOST):5432 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "NATS:" && (curl -sf --max-time 2 http://$(DEPLOY_HOST):4222 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "Redis:" && (curl -sf --max-time 2 http://$(DEPLOY_HOST):6379 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "MinIO:" && (curl -sf --max-time 2 http://$(DEPLOY_HOST):9000 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "=== Backup Status ==="
	@TOKEN=$$(curl -sf -X POST http://$(DEPLOY_HOST):$(API_PORT)/api/auth/login -H 'Content-Type: application/json' -d "{\"email\":\"$$CLARITY_TEST_EMAIL\",\"password\":\"$$CLARITY_TEST_PASSWORD\"}" | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])') && curl -sf http://$(DEPLOY_HOST):$(API_PORT)/api/admin/backup-status -H "Authorization: Bearer $$TOKEN" | python3 -m json.tool || echo "FAIL"

# Playwright E2E smoke tests
test-e2e:
	cd e2e && npm install && npx playwright install --with-deps chromium && npx playwright test
