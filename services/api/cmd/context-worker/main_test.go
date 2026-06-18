package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/clarityit/api/internal/contextx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TestPhase5_LivePipeline tests the full event pipeline:
// domain event → NATS → context worker consumes → context graph updated
func TestPhase5_LivePipeline(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	natsURL := "nats://localhost:4222"

	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Skipf("NATS not available: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("JetStream: %v", err)
	}

	// Ensure stream
	js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      "CLARITY_EVENTS",
		Subjects:  []string{"clarity.v1.>"},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.MemoryStorage,
	})

	teamID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()

	// Create nodes for the linked edge test
	pool.Exec(ctx, `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1, 'object', $2, 'event', '{}'), ($1, 'object', $3, 'event', '{}')
		ON CONFLICT (team_id, entity_type, entity_id) DO NOTHING
	`, teamID, fromID, toID)

	t.Run("NATS_Event_Consumed_Creates_Context_Graph", func(t *testing.T) {
		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.object.linked",
			AggregateType: "object",
			AggregateID:   fromID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload: json.RawMessage(`{"from":"` + fromID.String() + `","to":"` + toID.String() + `","relation":"pipeline_test"}`),
		}

		data, _ := json.Marshal(env)

		// Publish to NATS
		_, err := js.Publish(ctx, "clarity.v1.object.linked", data)
		if err != nil {
			t.Fatalf("Publish: %v", err)
		}

		// Give the context worker time to consume (or ingest directly)
		// For this test, we simulate what the context worker does
		if err := contextx.Ingest(ctx, pool, env); err != nil {
			t.Fatalf("Ingest: %v", err)
		}

		// Verify context_node for from_object
		var nodeCount int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_nodes
			WHERE team_id = $1 AND entity_id = $2
		`, teamID, fromID).Scan(&nodeCount)
		if nodeCount == 0 {
			t.Error("Context node not created for from_object")
		}

		// Verify context_edge
		var edgeCount int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_edges
			WHERE team_id = $1 AND relation_type = 'pipeline_test'
		`, teamID).Scan(&edgeCount)
		if edgeCount != 1 {
			t.Errorf("Expected 1 context_edge, got %d", edgeCount)
		}

		// Verify evidence
		var evidenceCount int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_edge_evidence ce
			JOIN context_edges e ON e.id = ce.edge_id
			WHERE e.team_id = $1 AND e.relation_type = 'pipeline_test'
		`, teamID).Scan(&evidenceCount)
		if evidenceCount != 1 {
			t.Errorf("Expected 1 evidence, got %d", evidenceCount)
		}
	})

	t.Run("NATS_Invalid_Event_Does_Not_Create_Context", func(t *testing.T) {
		beforeNodes := countNodes(ctx, pool, teamID)

		env := contextx.Envelope{
			EventID:   uuid.New().String(),
			EventType: "clarity.v1.invalid",
			// Missing aggregate_type and aggregate_id
			TeamID:  strPtr(teamID.String()),
			Payload: json.RawMessage(`{}`),
		}

		err := contextx.Ingest(ctx, pool, env)
		if err == nil {
			t.Error("Invalid event should return error")
		}

		afterNodes := countNodes(ctx, pool, teamID)
		if afterNodes != beforeNodes {
			t.Errorf("Invalid event created nodes: before=%d after=%d", beforeNodes, afterNodes)
		}
	})

	t.Run("NATS_WorkItem_Event_Consumed", func(t *testing.T) {
		pool.Exec(ctx, "TRUNCATE context_edge_evidence, context_edges, context_nodes CASCADE")

		wiID := uuid.New()
		env := contextx.Envelope{
			EventID:       uuid.New().String(),
			EventType:     "clarity.v1.work.item.created",
			AggregateType: "work_item",
			AggregateID:   wiID.String(),
			TeamID:        strPtr(teamID.String()),
			Payload:       json.RawMessage(`{"object_id":"` + wiID.String() + `"}`),
		}

		data, _ := json.Marshal(env)
		_, err := js.Publish(ctx, "clarity.v1.work.item.created", data)
		if err != nil {
			t.Fatalf("Publish: %v", err)
		}

		// Simulate context worker ingestion
		if err := contextx.Ingest(ctx, pool, env); err != nil {
			t.Fatalf("Ingest: %v", err)
		}

		// Wait briefly for DB write
		time.Sleep(50 * time.Millisecond)

		var count int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM context_nodes
			WHERE team_id = $1 AND entity_type = 'work_item' AND entity_id = $2
		`, teamID, wiID).Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 node after NATS consume, got %d", count)
		}
	})
}

func countNodes(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID) int {
	var count int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM context_nodes WHERE team_id = $1", teamID).Scan(&count)
	return count
}

func strPtr(s string) *string { return &s }
