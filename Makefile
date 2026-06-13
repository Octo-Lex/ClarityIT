.PHONY: build test test-iam test-team deploy

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
