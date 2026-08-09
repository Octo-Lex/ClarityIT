-- 002_outbox_and_audit.sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL,
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    actor_id UUID,
    actor_type TEXT NOT NULL DEFAULT 'user' CHECK (actor_type IN ('user', 'agent', 'system')),
    team_id UUID,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    old_value JSONB NOT NULL DEFAULT '{}'::jsonb,
    new_value JSONB NOT NULL DEFAULT '{}'::jsonb,
    change_summary TEXT NOT NULL,
    ip_hmac TEXT,
    user_agent_hmac TEXT,
    hmac_key_id TEXT,
    idempotency_key TEXT,
    request_id UUID,
    correlation_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at),
    CHECK (jsonb_typeof(old_value) = 'object'),
    CHECK (jsonb_typeof(new_value) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_audit_team ON audit_logs (team_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_correlation ON audit_logs (correlation_id) WHERE correlation_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS idempotency_keys (
    scope_type TEXT NOT NULL CHECK (scope_type IN ('user', 'anonymous', 'system', 'agent')),
    scope_id TEXT NOT NULL,
    key TEXT NOT NULL,
    request_method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'processing' CHECK (status IN ('processing', 'completed', 'failed')),
    response_code INT,
    response_body JSONB,
    error_code TEXT,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (scope_type, scope_id, key),
    CHECK (expires_at > created_at),
    CHECK (response_body IS NULL OR jsonb_typeof(response_body) = 'object')
);

CREATE TRIGGER trg_idempotency_updated_at
BEFORE UPDATE ON idempotency_keys
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    event_version INT NOT NULL DEFAULT 1,
    aggregate_type TEXT,
    aggregate_id UUID,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    sensitive_payload_ciphertext BYTEA,
    sensitive_payload_key_id TEXT,
    provider_message_key TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'sent', 'failed', 'dead_letter')),
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    locked_by TEXT,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    purge_after TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',
    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_outbox_ready ON outbox_events (status, next_attempt_at, created_at) WHERE status IN ('pending', 'failed');
CREATE INDEX IF NOT EXISTS idx_outbox_processing ON outbox_events (status, locked_at) WHERE status = 'processing';
CREATE INDEX IF NOT EXISTS idx_outbox_provider_key ON outbox_events (provider_message_key) WHERE provider_message_key IS NOT NULL;

CREATE TRIGGER trg_outbox_updated_at
BEFORE UPDATE ON outbox_events
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
