-- Phase 8.1: Webhook signature verification support
-- Adds signing_secret_hash and allow_unsigned_dev to integration_api_keys

ALTER TABLE integration_api_keys
  ADD COLUMN IF NOT EXISTS signing_secret_hash text,
  ADD COLUMN IF NOT EXISTS allow_unsigned_dev boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN integration_api_keys.signing_secret_hash IS 'HMAC-SHA256 hash of the signing secret used for webhook signature verification';
COMMENT ON COLUMN integration_api_keys.allow_unsigned_dev IS 'Allow unsigned webhooks in development mode only';
