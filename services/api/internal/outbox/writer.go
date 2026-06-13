package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Event struct {
	ID            string
	EventType     string
	EventVersion  int
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
}

// Write writes an outbox event within the current transaction.
// This MUST be called inside the same transaction as the domain mutation.
// The outbox worker will publish to NATS after commit.
func Write(ctx context.Context, tx pgx.Tx, teamID *string, e Event) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	if e.EventVersion == 0 {
		e.EventVersion = 1
	}
	if e.Payload == nil {
		e.Payload = json.RawMessage(`{}`)
	}

	purgeAfter := time.Now().Add(7 * 24 * time.Hour)

	var teamUUID *uuid.UUID
	if teamID != nil && *teamID != "" {
		parsed, err := uuid.Parse(*teamID)
		if err == nil {
			teamUUID = &parsed
		}
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (
			id, event_type, event_version,
			aggregate_type, aggregate_id,
			payload, status, team_id,
			next_attempt_at, purge_after
		) VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, NOW(), $8)
	`, e.ID, e.EventType, e.EventVersion,
		e.AggregateType, e.AggregateID,
		e.Payload, teamUUID, purgeAfter,
	)
	return err
}
