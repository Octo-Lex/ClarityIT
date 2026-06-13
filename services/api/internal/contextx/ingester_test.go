package contextx_test

import (
	"encoding/json"
	"testing"

	"github.com/clarityit/api/internal/contextx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhase5_ContextIngestion(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	// Clean context tables
	pool.Exec(ctx, "TRUNCATE context_edge_evidence, context_edges, context_nodes CASCADE")

	teamID := uuid.New()

	t.Run("WorkItem_Created_Upserts_Context_Node", func(t *testing.T) {
		aggregateID := uuid.New()
		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.work.item.created",
			AggregateType: "work_item",
			AggregateID:   aggregateID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload:       json.RawMessage(`{"object_id":"` + aggregateID.String() + `"}`),
		}

		if err := contextx.Ingest(ctx, pool, env); err != nil {
			t.Fatalf("Ingest: %v", err)
		}

		var count int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_nodes
			WHERE team_id = $1 AND entity_type = 'work_item' AND entity_id = $2
		`, teamID, aggregateID).Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 context_node, got %d", count)
		}
	})

	t.Run("Object_Linked_Creates_Context_Edge", func(t *testing.T) {
		fromID := uuid.New()
		toID := uuid.New()

		// Create nodes first (the edge creator expects them)
		pool.Exec(ctx, `
			INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
			VALUES ($1, 'object', $2, 'event', '{}'), ($1, 'object', $3, 'event', '{}')
			ON CONFLICT (team_id, entity_type, entity_id) DO NOTHING
		`, teamID, fromID, toID)

		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.object.linked",
			AggregateType: "object",
			AggregateID:   fromID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload:       json.RawMessage(`{"from":"` + fromID.String() + `","to":"` + toID.String() + `","relation":"depends_on"}`),
		}

		if err := contextx.Ingest(ctx, pool, env); err != nil {
			t.Fatalf("Ingest: %v", err)
		}

		var edgeCount int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_edges
			WHERE team_id = $1 AND relation_type = 'depends_on'
		`, teamID).Scan(&edgeCount)
		if edgeCount != 1 {
			t.Errorf("Expected 1 context_edge, got %d", edgeCount)
		}

		var evidenceCount int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_edge_evidence ce
			JOIN context_edges e ON e.id = ce.edge_id
			WHERE e.team_id = $1 AND e.relation_type = 'depends_on'
		`, teamID).Scan(&evidenceCount)
		if evidenceCount != 1 {
			t.Errorf("Expected 1 context_edge_evidence, got %d", evidenceCount)
		}
	})

	t.Run("Redelivery_Does_Not_Duplicate_Node", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE context_edge_evidence, context_edges, context_nodes CASCADE")
		aggregateID := uuid.New()
		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.work.item.created",
			AggregateType: "work_item",
			AggregateID:   aggregateID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload:       json.RawMessage(`{"object_id":"` + aggregateID.String() + `"}`),
		}

		// Ingest twice (simulating redelivery)
		contextx.Ingest(ctx, pool, env)
		contextx.Ingest(ctx, pool, env)

		var count int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_nodes
			WHERE team_id = $1 AND entity_type = 'work_item' AND entity_id = $2
		`, teamID, aggregateID).Scan(&count)
		if count != 1 {
			t.Errorf("Redelivery created %d nodes, expected 1", count)
		}
	})

	t.Run("Redelivery_Does_Not_Duplicate_Edge", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE context_edge_evidence, context_edges, context_nodes CASCADE")
		fromID := uuid.New()
		toID := uuid.New()

		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.object.linked",
			AggregateType: "object",
			AggregateID:   fromID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload:       json.RawMessage(`{"from":"` + fromID.String() + `","to":"` + toID.String() + `","relation":"blocks"}`),
		}

		contextx.Ingest(ctx, pool, env)
		contextx.Ingest(ctx, pool, env)

		var count int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_edges
			WHERE team_id = $1 AND relation_type = 'blocks'
		`, teamID).Scan(&count)
		if count != 1 {
			t.Errorf("Redelivery created %d edges, expected 1", count)
		}
	})

	t.Run("Context_Node_Label_Excludes_Sensitive_Data", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE context_edge_evidence, context_edges, context_nodes CASCADE")
		aggregateID := uuid.New()
		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.work.item.created",
			AggregateType: "work_item",
			AggregateID:   aggregateID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload:       json.RawMessage(`{"object_id":"` + aggregateID.String() + `","title":"Secret Title Here"}`),
		}

		contextx.Ingest(ctx, pool, env)

		var properties string
		pool.QueryRow(ctx, `
			SELECT properties::text FROM context_nodes
			WHERE team_id = $1 AND entity_id = $2
		`, teamID, aggregateID).Scan(&properties)

		// Properties should be empty {} — no title/body
		if properties != "{}" {
			t.Errorf("Properties should be empty, got: %s", properties)
		}
	})

	t.Run("Invalid_Event_Missing_Aggregate_Rejected", func(t *testing.T) {
		env := contextx.Envelope{
			EventID:   uuid.New().String(),
			EventType: "clarity.v1.unknown",
			TeamID:    strPtr(teamID.String()),
			Payload:   json.RawMessage(`{}`),
		}

		err := contextx.Ingest(ctx, pool, env)
		if err == nil {
			t.Error("Should reject event with missing aggregate")
		}
	})
}

func strPtr(s string) *string { return &s }
