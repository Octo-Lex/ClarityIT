-- 036_document_artifacts.sql
-- v1.4 Track 1: Native Document Artifact Model.
-- Sidecar table for structured documents (JSONB block model).
-- artifact_documents.artifact_id is a FK to artifacts(id) with CASCADE delete.
-- The artifacts row has artifact_type='document'.

CREATE TABLE IF NOT EXISTS artifact_documents (
    artifact_id UUID PRIMARY KEY REFERENCES artifacts(id) ON DELETE CASCADE,
    document_type TEXT NOT NULL,
    document_json JSONB NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    word_count INTEGER NOT NULL DEFAULT 0,
    last_exported_storage_object_id UUID NULL REFERENCES storage_objects(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT adoc_schema_version CHECK (schema_version = 1),
    CONSTRAINT adoc_word_count_nonneg CHECK (word_count >= 0),
    CONSTRAINT adoc_doc_json_object CHECK (jsonb_typeof(document_json) = 'object'),
    CONSTRAINT adoc_document_type_check CHECK (
        document_type IN (
            'general_document',
            'decision_memo',
            'implementation_plan',
            'incident_summary',
            'training_doc',
            'architecture_doc',
            'project_report',
            'status_report',
            'meeting_summary',
            'executive_brief'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_adoc_document_type ON artifact_documents(document_type);
CREATE INDEX IF NOT EXISTS idx_adoc_updated_at ON artifact_documents(updated_at DESC);

-- updated_at trigger (consistent with artifacts table pattern)
CREATE OR REPLACE FUNCTION adoc_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_adoc_set_updated_at ON artifact_documents;
CREATE TRIGGER trg_adoc_set_updated_at
    BEFORE UPDATE ON artifact_documents
    FOR EACH ROW
    EXECUTE FUNCTION adoc_set_updated_at();
