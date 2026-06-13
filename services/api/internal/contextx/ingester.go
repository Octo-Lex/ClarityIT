package contextx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Envelope represents a Genesis event envelope from NATS.
type Envelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	TeamID        *string         `json:"team_id"`
	Payload       json.RawMessage `json:"payload"`
}

// Ingest processes one event and upserts context graph nodes/edges.
// Must be idempotent — safe to call on NATS redelivery.
func Ingest(ctx context.Context, pool *pgxpool.Pool, env Envelope) error {
	if env.AggregateType == "" || env.AggregateID == "" {
		return fmt.Errorf("missing aggregate_type or aggregate_id")
	}

	aggregateUUID, err := uuid.Parse(env.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid aggregate_id: %w", err)
	}

	var teamUUID uuid.UUID
	if env.TeamID != nil && *env.TeamID != "" {
		teamUUID, _ = uuid.Parse(*env.TeamID)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Upsert context node (idempotent via unique constraint)
	var nodeID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1, $2, $3, 'event', '{}')
		ON CONFLICT (team_id, entity_type, entity_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, teamUUID, env.AggregateType, aggregateUUID).Scan(&nodeID)
	if err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}

	// Edge mapping based on event type
	switch env.EventType {
	case "clarity.v1.object.linked":
		ingestLinkedEdge(ctx, tx, teamUUID, nodeID, env)
	case "clarity.v1.work.item.created":
		ingestWorkItemEdges(ctx, tx, teamUUID, nodeID, env)
	case "clarity.v1.incident.opened":
		// Basic node created above; no extra edges for now
	case "clarity.v1.object.commented":
		ingestCommentEdge(ctx, tx, teamUUID, nodeID, env)
	case "clarity.v1.incident.timeline_added":
		ingestTimelineEdge(ctx, tx, teamUUID, nodeID, env)
	}

	return tx.Commit(ctx)
}

func ingestLinkedEdge(ctx context.Context, tx pgx.Tx, teamID uuid.UUID, _ /* fromNodeID */ uuid.UUID, env Envelope) {
	var payload struct {
		From       string `json:"from"`
		To         string `json:"to"`
		Relation   string `json:"relation"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	if payload.From == "" || payload.To == "" {
		return
	}

	fromUUID, err1 := uuid.Parse(payload.From)
	toUUID, err2 := uuid.Parse(payload.To)
	if err1 != nil || err2 != nil {
		return
	}

	relationType := payload.Relation
	if relationType == "" {
		relationType = "linked_to"
	}

	// Upsert both nodes
	var fromNodeID, toNodeID uuid.UUID
	tx.QueryRow(ctx, `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1, 'object', $2, 'event', '{}')
		ON CONFLICT (team_id, entity_type, entity_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, teamID, fromUUID).Scan(&fromNodeID)
	tx.QueryRow(ctx, `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1, 'object', $2, 'event', '{}')
		ON CONFLICT (team_id, entity_type, entity_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, teamID, toUUID).Scan(&toNodeID)

	createDirectEdge(ctx, tx, teamID, fromNodeID, toNodeID, relationType, env)
}

func ingestWorkItemEdges(ctx context.Context, tx pgx.Tx, teamID, nodeID uuid.UUID, env Envelope) {
	var payload struct {
		ObjectID   string `json:"object_id"`
		OwnerID    string `json:"owner_id"`
		AssigneeID string `json:"assignee_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}

	// Edge to owner if present
	if payload.OwnerID != "" {
		ownerUUID, err := uuid.Parse(payload.OwnerID)
		if err == nil {
			// Upsert user node
			var userNodeID uuid.UUID
			err := tx.QueryRow(ctx, `
				INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
				VALUES ($1, 'user', $2, 'event', '{}')
				ON CONFLICT (team_id, entity_type, entity_id) DO UPDATE SET updated_at = NOW()
				RETURNING id
			`, teamID, ownerUUID).Scan(&userNodeID)
			if err == nil {
				createDirectEdge(ctx, tx, teamID, nodeID, userNodeID, "assigned_to_owner", env)
			}
		}
	}
}

func ingestCommentEdge(ctx context.Context, tx pgx.Tx, teamID, objectNodeID uuid.UUID, env Envelope) {
	var payload struct {
		CommentID string `json:"comment_id"`
		ObjectID  string `json:"object_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	if payload.CommentID == "" {
		return
	}

	commentUUID, err := uuid.Parse(payload.CommentID)
	if err != nil {
		return
	}

	// Upsert comment node
	var commentNodeID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1, 'comment', $2, 'event', '{}')
		ON CONFLICT (team_id, entity_type, entity_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, teamID, commentUUID).Scan(&commentNodeID)
	if err != nil {
		return
	}

	createDirectEdge(ctx, tx, teamID, objectNodeID, commentNodeID, "has_comment", env)
}

func ingestTimelineEdge(ctx context.Context, tx pgx.Tx, teamID, incidentNodeID uuid.UUID, env Envelope) {
	var payload struct {
		CommentID string `json:"comment_id"`
		ObjectID  string `json:"object_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	if payload.CommentID == "" {
		return
	}

	commentUUID, err := uuid.Parse(payload.CommentID)
	if err != nil {
		return
	}

	var commentNodeID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1, 'comment', $2, 'event', '{}')
		ON CONFLICT (team_id, entity_type, entity_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, teamID, commentUUID).Scan(&commentNodeID)
	if err != nil {
		return
	}

	createDirectEdge(ctx, tx, teamID, incidentNodeID, commentNodeID, "timeline_entry", env)
}

func createDirectEdge(ctx context.Context, tx pgx.Tx, teamID, fromNodeID, toNodeID uuid.UUID, relationType string, env Envelope) {
	eventUUID, err := uuid.Parse(env.EventID)
	if err != nil {
		return
	}

	// Upsert edge (idempotent via unique constraint)
	var edgeID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO context_edges (team_id, from_node_id, to_node_id, relation_type, weight)
		VALUES ($1, $2, $3, $4, 1.0)
		ON CONFLICT (team_id, from_node_id, to_node_id, relation_type) DO UPDATE SET weight = 1.0
		RETURNING id
	`, teamID, fromNodeID, toNodeID, relationType).Scan(&edgeID)
	if err != nil {
		log.Printf("createDirectEdge: %v", err)
		return
	}

	// Insert evidence (idempotent)
	tx.Exec(ctx, `
		INSERT INTO context_edge_evidence (edge_id, evidence_event_id, evidence_summary)
		VALUES ($1, $2, $3)
		ON CONFLICT (edge_id, evidence_event_id) DO NOTHING
	`, edgeID, eventUUID, env.EventType)
}
