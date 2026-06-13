.PHONY: build test test-iam test-team deploy audit

# Build the Go binary
build:
	cd services/api && go build ./cmd/api

# Run all tests sequentially (shared DB requires -p 1)
# TODO: implement per-package test DB isolation to remove -p 1 requirement
test:
	cd services/api && go test -p 1 -count=1 -timeout 180s ./...

# Run only IAM tests
test-iam:
	cd services/api && go test -v -count=1 -timeout 60s ./internal/iam/

# Run only team tests
test-team:
	cd services/api && go test -v -count=1 -timeout 60s ./internal/team/

# Run only domain tests
test-domain:
	cd services/api && go test -v -count=1 -timeout 60s ./internal/domain/

# Run only outbox worker tests
test-outbox:
	cd services/api && go test -v -count=1 -timeout 60s ./cmd/outbox-worker/

# Run only context tests
test-context:
	cd services/api && go test -v -count=1 -timeout 60s ./internal/contextx/

# Run only WebSocket tests
test-ws:
	cd services/api && go test -v -count=1 -timeout 30s ./internal/wsx/

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
	cd services/api && go test -v -count=1 -run TestPhase5_LivePipeline ./cmd/context-worker/ -timeout 60s
	cd services/api && go test -v -count=1 -run TestPhase5_OutboxWorker ./cmd/outbox-worker/ -timeout 60s

# Verify docker compose deployment
verify-deploy:
	curl -sf http://192.168.3.20:8765/health
	docker logs clarityit-outbox-worker 2>&1 | tail -3
	docker logs clarityit-context-worker 2>&1 | tail -3

# Security and dependency audit
audit:
	@echo "=== Go Vet ==="
	cd services/api && go vet ./...
	@echo "=== Go Tests ==="
	cd services/api && go test -p 1 -count=1 -timeout 180s ./...
	@echo "=== Frontend Audit ==="
	cd web && npm audit --audit-level=high 2>&1 || true
	@echo "=== Python Check ==="
	cd services/workers/reasoning && python -m pip check 2>&1 || true
	@echo "=== Audit Complete ==="

# Full deployment verification
verify-deployment:
	@echo "=== Web Frontend ==="
	@curl -sf http://192.168.3.20:3000 | head -c 100 && echo "... OK" || echo "FAIL"
	@echo "=== API Health ==="
	@curl -sf http://192.168.3.20:8765/health || echo "FAIL"
	@echo ""
	@echo "=== Deep Health (requires token) ==="
	@TOKEN=$$(curl -sf -X POST http://192.168.3.20:8765/api/auth/login -H 'Content-Type: application/json' -d '{"email":"owner@test.dev","password":"password12"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])') && curl -sf http://192.168.3.20:8765/api/health/deep -H "Authorization: Bearer $$TOKEN" | python3 -m json.tool || echo "FAIL"
	@echo "=== Docker Services ==="
	@docker compose -f /opt/clarityit/docker-compose.yml ps --format "table {{.Name}}\t{{.Status}}"
	@echo "=== Port Exposure Check ==="
	@echo "PostgreSQL:" && (curl -sf --max-time 2 http://192.168.3.20:5432 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "NATS:" && (curl -sf --max-time 2 http://192.168.3.20:4222 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "Redis:" && (curl -sf --max-time 2 http://192.168.3.20:6379 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
	@echo "MinIO:" && (curl -sf --max-time 2 http://192.168.3.20:9000 2>/dev/null && echo "EXPOSED (WARN)" || echo "private (OK)")
