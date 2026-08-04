-- 034_meeting_artifacts.sql
-- v1.3 Track 3: Structured meeting summary data sidecar table.
-- 1:1 extension of artifacts where artifact_type = 'meeting_summary'.

CREATE TABLE IF NOT EXISTS artifact_meeting_data (
    artifact_id       UUID PRIMARY KEY REFERENCES artifacts(id) ON DELETE CASCADE,
    meeting_date      DATE,
    attendees         JSONB NOT NULL DEFAULT '[]',
    agenda_items      JSONB NOT NULL DEFAULT '[]',
    decisions         JSONB NOT NULL DEFAULT '[]',
    action_items      JSONB NOT NULL DEFAULT '[]',
    duration_minutes  INTEGER,

    CONSTRAINT amd_duration_nonneg CHECK (duration_minutes IS NULL OR duration_minutes >= 0),
    CONSTRAINT amd_duration_cap    CHECK (duration_minutes IS NULL OR duration_minutes <= 1440),
    CONSTRAINT amd_attendees_arr     CHECK (jsonb_typeof(attendees)    = 'array'),
    CONSTRAINT amd_agenda_arr        CHECK (jsonb_typeof(agenda_items) = 'array'),
    CONSTRAINT amd_decisions_arr     CHECK (jsonb_typeof(decisions)    = 'array'),
    CONSTRAINT amd_action_items_arr  CHECK (jsonb_typeof(action_items) = 'array')
);

-- Index for efficient join lookups
CREATE INDEX IF NOT EXISTS idx_amd_meeting_date ON artifact_meeting_data(meeting_date);
