-- v0.8.0: Add rotation_required flag to integration_api_keys
-- Keys created before migration 020 lack signing_secret_hash and must be rotated

ALTER TABLE integration_api_keys
  ADD COLUMN IF NOT EXISTS rotation_required boolean NOT NULL DEFAULT false;

-- Mark all existing keys without signing_secret_hash as requiring rotation
UPDATE integration_api_keys
  SET rotation_required = true
  WHERE signing_secret_hash IS NULL OR signing_secret_hash = '';

COMMENT ON COLUMN integration_api_keys.rotation_required IS 'True when key lacks signing_secret_hash and must be rotated for production webhook signing';
