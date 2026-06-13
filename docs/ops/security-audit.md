# Security Audit — ClarityIT

## Running the Audit

```bash
make audit
```

This runs:
1. `go vet ./...` — static analysis for Go code
2. `go test -p 1 ./...` — full test suite (includes security tests)
3. `npm audit --audit-level=high` — frontend dependency audit
4. `python -m pip check` — Python dependency consistency

## What's Checked

### Go Backend
- **go vet**: Detects common Go programming errors
- **Integration tests**: Webhook signature verification, HMAC key hashing, PII redaction in audit
- **Config tests**: Production secret strength validation, dev default enforcement
- **Health tests**: Metrics endpoint, deep health structure

### Frontend
- **npm audit**: Known vulnerabilities in npm dependencies (high severity threshold)
- Checks: XSS vectors, prototype pollution, path traversal in dependencies

### Python Worker
- **pip check**: Dependency consistency, version conflicts

## Manual Security Checks

### Secret Exposure
```bash
# Verify no secrets in code
git grep -i 'password.*=.*\"[a-z]' -- '*.go' '*.py' '*.ts'
git grep -i 'secret.*=.*\"[a-z]' -- '*.go' '*.py' '*.ts'

# Verify no raw keys in audit logs (requires DB access)
docker exec clarityit-postgres-1 psql -U clarityit -d clarityit \
  -c "SELECT new_value::text FROM audit_logs WHERE action LIKE 'integration.%' ORDER BY created_at DESC LIMIT 5"
```

### Webhook Signature Verification
```bash
# Create a key with allow_unsigned_dev=false
curl -X POST .../integration-keys -d '{"name":"test","allowed_sources":["*"],"allowed_scopes":["*"]}'

# Verify unsigned request is rejected
curl -X POST .../webhooks/test -H 'X-ClarityIT-Integration-Key: <key>'
# Should return 401 (missing signature)

# Verify signed request works
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY='{"name":"test","severity":"low","source_id":"test"}'
SIG="v1=$(echo -n "${TIMESTAMP}.${BODY}" | openssl dgst -sha256 -hmac "<key>" | awk '{print $2}')"
curl -X POST .../webhooks/test \
  -H "X-ClarityIT-Integration-Key: <key>" \
  -H "X-ClarityIT-Signature: ${SIG}" \
  -H "X-ClarityIT-Timestamp: ${TIMESTAMP}" \
  -d "${BODY}"
```

### Container Security
```bash
# Check containers run as non-root
docker exec clarityit-api id
docker exec clarityit-web id

# Check exposed ports (only 8765 and 3000 should be public)
docker compose ps --format "table {{.Name}}\t{{.Ports}}"
```

## Acceptance Criteria

- [ ] `make audit` completes without errors
- [ ] No secrets in git history
- [ ] Webhook signatures verified in production
- [ ] All tests pass including integration tests
- [ ] No raw PII in audit logs
- [ ] Containers hardened (non-root, minimal images)
