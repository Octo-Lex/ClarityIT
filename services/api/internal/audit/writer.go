package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Event struct {
	EventID        string
	EventType      string // maps to action
	ActorID        uuid.UUID
	ActorType      string
	TeamID         *uuid.UUID
	Action         string // maps to action
	EntityType     string
	EntityID       uuid.UUID
	OldValue       json.RawMessage
	NewValue       json.RawMessage
	Summary        string // maps to change_summary
	IPHMAC         string
	UserAgentHMAC  string
	IdempotencyKey string
	RequestID      string
	CorrelationID  string
}

// Write writes an audit log entry within the current transaction.
// This MUST be called inside the same transaction as the domain mutation.
func Write(ctx context.Context, tx pgx.Tx, e Event) error {
	if e.EventID == "" {
		e.EventID = uuid.New().String()
	}
	if e.ActorType == "" {
		e.ActorType = "user"
	}
	if e.Action == "" {
		e.Action = e.EventType // Use EventType as Action if Action not set
	}
	if e.OldValue == nil {
		e.OldValue = json.RawMessage(`{}`)
	}
	if e.NewValue == nil {
		e.NewValue = json.RawMessage(`{}`)
	}

	var requestID, correlationID *string
	if e.RequestID != "" {
		requestID = &e.RequestID
	}
	if e.CorrelationID != "" {
		correlationID = &e.CorrelationID
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			event_id, actor_id, actor_type, team_id,
			action, entity_type, entity_id,
			old_value, new_value, change_summary,
			ip_hmac, user_agent_hmac,
			idempotency_key, request_id, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, e.EventID, e.ActorID, e.ActorType, e.TeamID,
		e.Action, e.EntityType, e.EntityID,
		e.OldValue, e.NewValue, e.Summary,
		e.IPHMAC, e.UserAgentHMAC,
		e.IdempotencyKey, requestID, correlationID,
	)
	return err
}
