-- Phase 5: Outbox worker fields + context idempotency indexes

-- Add team_id to outbox_events for routing
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS team_id uuid;
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS dead_lettered_at timestamptz;

-- Add provider_message_key column for NATS ack tracking if not present
-- (already exists from earlier migration, skip)

-- Index for team-based outbox queries
CREATE INDEX IF NOT EXISTS idx_outbox_team_id ON outbox_events(team_id) WHERE team_id IS NOT NULL;

-- Context edge idempotency: prevent duplicate edges
CREATE UNIQUE INDEX IF NOT EXISTS idx_context_edges_unique
  ON context_edges(team_id, from_node_id, to_node_id, relation_type);

-- Context edge evidence idempotency: prevent duplicate evidence
CREATE UNIQUE INDEX IF NOT EXISTS idx_context_edge_evidence_unique
  ON context_edge_evidence(edge_id, evidence_event_id);

-- Backfill team_id from payload where possible
-- This is a best-effort backfill; new events will have team_id set by the writer
UPDATE outbox_events SET team_id = (payload->>'team_id')::uuid
WHERE team_id IS NULL AND payload->>'team_id' IS NOT NULL;
