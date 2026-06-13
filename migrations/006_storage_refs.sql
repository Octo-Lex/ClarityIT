-- 006_storage_refs.sql
CREATE TABLE IF NOT EXISTS storage_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    bucket TEXT NOT NULL,
    object_key TEXT NOT NULL,
    content_type TEXT,
    size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
    sha256 TEXT NOT NULL,
    encryption_status TEXT NOT NULL DEFAULT 'provider_managed' CHECK (encryption_status IN ('none', 'provider_managed', 'app_managed')),
    retention_policy TEXT NOT NULL DEFAULT 'default',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bucket, object_key)
);

CREATE INDEX IF NOT EXISTS idx_storage_objects_team ON storage_objects (team_id, created_at DESC);

CREATE TABLE IF NOT EXISTS object_storage_refs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL,
    object_id UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    storage_object_id UUID NOT NULL REFERENCES storage_objects(id) ON DELETE RESTRICT,
    ref_type TEXT NOT NULL,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_object_storage_refs_object ON object_storage_refs (object_id, ref_type);
