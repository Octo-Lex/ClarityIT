-- 038_document_versions.sql
-- v1.4 Track 7: Document Version History.
-- Non-destructive version snapshots for native ClarityDocs documents.
-- Restore creates new versions — never overwrites or deletes old ones.

CREATE TABLE IF NOT EXISTS artifact_document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    document_json JSONB NOT NULL,
    version_number INTEGER NOT NULL,
    word_count INTEGER NOT NULL DEFAULT 0,
    change_summary TEXT,
    source TEXT NOT NULL CHECK (source IN ('user_save','agent_assisted_edit','generated','template','restore')),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- document_json must be a JSON object
ALTER TABLE artifact_document_versions
  DROP CONSTRAINT IF EXISTS adv_docjson_object;
ALTER TABLE artifact_document_versions
  ADD CONSTRAINT adv_docjson_object CHECK (jsonb_typeof(document_json) = 'object');

-- word_count must be non-negative
ALTER TABLE artifact_document_versions
  DROP CONSTRAINT IF EXISTS adv_wordcount_nonneg;
ALTER TABLE artifact_document_versions
  ADD CONSTRAINT adv_wordcount_nonneg CHECK (word_count >= 0);

-- version_number must be unique per artifact
ALTER TABLE artifact_document_versions
  DROP CONSTRAINT IF EXISTS adv_artifact_version_unique;
ALTER TABLE artifact_document_versions
  ADD CONSTRAINT adv_artifact_version_unique UNIQUE (artifact_id, version_number);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_adv_artifact_version
    ON artifact_document_versions (artifact_id, version_number DESC);

CREATE INDEX IF NOT EXISTS idx_adv_team_created
    ON artifact_document_versions (team_id, created_at DESC);
