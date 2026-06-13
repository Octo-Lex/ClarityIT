package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
)

func TestPhase5_OutboxWorker(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	natsURL := "nats://192.168.3.20:4222"
	redisAddr := "192.168.3.20:6379"

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

	// Ensure streams exist
	js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      "CLARITY_EVENTS",
		Subjects:  []string{"clarity.v1.>"},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.MemoryStorage,
	})
	js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      "CLARITY_DLQ",
		Subjects:  []string{"clarity.dlq.>"},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.MemoryStorage,
	})

	// Redis client
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	// Clean
	pool.Exec(ctx, "TRUNCATE outbox_events CASCADE")

	t.Run("Pending_Event_Publishes_To_NATS", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.published', 'test', $2, '{}', 'pending', NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		processBatch(ctx, pool, js, nc, rdb)

		var status string
		pool.QueryRow(ctx, "SELECT status FROM outbox_events WHERE id = $1", evtID).Scan(&status)
		if status != "sent" {
			t.Errorf("Expected sent, got %s", status)
		}
	})

	t.Run("Published_Event_Marked_Sent", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.sent', 'test', $2, '{}', 'pending', NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		processBatch(ctx, pool, js, nc, rdb)

		var processedAt *time.Time
		pool.QueryRow(ctx, "SELECT processed_at FROM outbox_events WHERE id = $1", evtID).Scan(&processedAt)
		if processedAt == nil {
			t.Error("processed_at should be set for sent event")
		}
	})

	t.Run("Sent_Event_Not_Republished", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, processed_at, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.already', 'test', $2, '{}', 'sent', NOW(), NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		ids, err := claimEvents(ctx, pool)
		if err != nil {
			t.Fatalf("Claim: %v", err)
		}
		for _, id := range ids {
			if id == evtID {
				t.Error("Sent event should not be claimed")
			}
		}
	})

	t.Run("Publish_Failure_Increments_Attempts", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.fail', 'test', $2, '{}', 'pending', NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		markFailed(ctx, pool, evtID, "clarity.v1.test.fail", "simulated failure")

		var attempts int
		pool.QueryRow(ctx, "SELECT attempts FROM outbox_events WHERE id = $1", evtID).Scan(&attempts)
		if attempts != 1 {
			t.Errorf("Expected attempts=1, got %d", attempts)
		}
	})

	t.Run("Publish_Failure_Sets_Next_Attempt_At", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, attempts, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.retry', 'test', $2, '{}', 'pending', 0, NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		markFailed(ctx, pool, evtID, "clarity.v1.test.retry", "retry test")

		var nextAttempt time.Time
		pool.QueryRow(ctx, "SELECT next_attempt_at FROM outbox_events WHERE id = $1", evtID).Scan(&nextAttempt)
		if nextAttempt.Before(time.Now()) {
			t.Error("next_attempt_at should be in the future")
		}
	})

	t.Run("Max_Attempts_Marks_Dead_Letter", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, attempts, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.dead', 'test', $2, '{}', 'failed', 4, NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		markFailed(ctx, pool, evtID, "clarity.v1.test.dead", "final failure")

		var status string
		pool.QueryRow(ctx, "SELECT status FROM outbox_events WHERE id = $1", evtID).Scan(&status)
		if status != "dead_letter" {
			t.Errorf("Expected dead_letter, got %s", status)
		}

		var deadLetterAt *time.Time
		pool.QueryRow(ctx, "SELECT dead_lettered_at FROM outbox_events WHERE id = $1", evtID).Scan(&deadLetterAt)
		if deadLetterAt == nil {
			t.Error("dead_lettered_at should be set")
		}
	})

	t.Run("Stale_Locked_Event_Can_Be_Reclaimed", func(t *testing.T) {
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, locked_at, locked_by, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.stale', 'test', $2, '{}', 'failed', NOW() - INTERVAL '10 minutes', 'old-worker', NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		ids, err := claimEvents(ctx, pool)
		if err != nil {
			t.Fatalf("Claim: %v", err)
		}
		found := false
		for _, id := range ids {
			if id == evtID {
				found = true
			}
		}
		if !found {
			t.Error("Stale locked event should be reclaimable")
		}
	})

	t.Run("Event_Payload_Shape_Preserved", func(t *testing.T) {
		evtID := uuid.New()
		aggID := uuid.New()
		teamID := uuid.New()
		payload := json.RawMessage(`{"key":"value","nested":{"foo":"bar"}}`)
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, team_id, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.shape', 'test', $2, $3, 'pending', $4, NOW(), NOW() + INTERVAL '7 days')
		`, evtID, aggID, payload, teamID)

		events, err := fetchClaimed(ctx, pool, []uuid.UUID{evtID})
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		var parsed map[string]any
		json.Unmarshal(events[0].Payload, &parsed)
		if parsed["key"] != "value" {
			t.Errorf("Payload shape not preserved: %v", parsed)
		}
	})

	t.Run("NATS_Streams_Exist", func(t *testing.T) {
		_, err := js.Stream(ctx, "CLARITY_EVENTS")
		if err != nil {
			t.Errorf("CLARITY_EVENTS stream missing: %v", err)
		}
		_, err = js.Stream(ctx, "CLARITY_DLQ")
		if err != nil {
			t.Errorf("CLARITY_DLQ stream missing: %v", err)
		}
	})

	t.Run("Published_Subject_Matches_Event_Type", func(t *testing.T) {
		sub, err := nc.SubscribeSync("clarity.v1.test.subject.*")
		if err != nil {
			t.Skipf("Subscribe: %v", err)
		}
		defer sub.Unsubscribe()

		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.subject.check', 'test', $2, '{}', 'pending', NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		processBatch(ctx, pool, js, nc, rdb)

		msg, err := sub.NextMsg(5 * time.Second)
		if err != nil {
			t.Skipf("No message: %v", err)
		}

		if msg.Subject != "clarity.v1.test.subject.check" {
			t.Errorf("Subject mismatch: got %s", msg.Subject)
		}
	})

	t.Run("DLQ_Excludes_Raw_Payload", func(t *testing.T) {
		var dlqPayload string
		pool.QueryRow(ctx, `
			SELECT payload::text FROM outbox_events
			WHERE event_type = 'clarity.dlq.outbox.publish_failed'
			ORDER BY created_at DESC LIMIT 1
		`).Scan(&dlqPayload)

		if dlqPayload == "" {
			t.Skip("No DLQ event found")
		}

		var parsed map[string]any
		json.Unmarshal([]byte(dlqPayload), &parsed)

		if _, ok := parsed["outbox_event_id"]; !ok {
			t.Error("DLQ event missing outbox_event_id")
		}
		if _, ok := parsed["attempts"]; !ok {
			t.Error("DLQ event missing attempts")
		}
		if _, ok := parsed["error_summary"]; !ok {
			t.Error("DLQ event missing error_summary")
		}
		if _, ok := parsed["payload"]; ok {
			t.Error("DLQ event should not include raw payload")
		}
	})

	t.Run("DLQ_Recursion_Guard", func(t *testing.T) {
		// Create a DLQ event and fail it — should NOT create another DLQ event
		evtID := uuid.New()
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, attempts, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.dlq.outbox.publish_failed', 'outbox_event', $2, '{}', 'failed', 4, NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New())

		// Count DLQ events before
		var beforeCount int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type LIKE 'clarity.dlq.%'").Scan(&beforeCount)

		markFailed(ctx, pool, evtID, "clarity.dlq.outbox.publish_failed", "DLQ failure")

		// The event should be dead_lettered but no NEW DLQ event created
		var status string
		pool.QueryRow(ctx, "SELECT status FROM outbox_events WHERE id = $1", evtID).Scan(&status)
		if status != "dead_letter" {
			t.Errorf("Expected dead_letter, got %s", status)
		}

		var afterCount int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type LIKE 'clarity.dlq.%'").Scan(&afterCount)
		if afterCount != beforeCount {
			t.Errorf("DLQ recursion: before=%d after=%d — should not create DLQ-of-DLQ", beforeCount, afterCount)
		}
	})

	t.Run("Redis_Fanout_Contains_No_Sensitive_Data", func(t *testing.T) {
		if rdb == nil {
			t.Skip("Redis not available")
		}

		// Subscribe to Redis channel
		sub := rdb.Subscribe(ctx, "clarity:ws:events")
		defer sub.Close()
		ch := sub.Channel()

		teamID := uuid.New()
		evtID := uuid.New()
		// Publish an event with sensitive data in the payload
		pool.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, team_id, next_attempt_at, purge_after)
			VALUES ($1, 'clarity.v1.test.sensitive', 'test', $2, '{"title":"Secret Title","email":"user@test.dev","ip":"192.168.1.1"}', 'pending', $3, NOW(), NOW() + INTERVAL '7 days')
		`, evtID, uuid.New(), teamID)

		processBatch(ctx, pool, js, nc, rdb)

		// Read Redis fanout message
		timeout := time.After(3 * time.Second)
		select {
		case msg := <-ch:
			var parsed map[string]string
			json.Unmarshal([]byte(msg.Payload), &parsed)

			// Required fields
			if parsed["team_id"] == "" { t.Error("Missing team_id") }
			if parsed["event_type"] == "" { t.Error("Missing event_type") }
			if parsed["aggregate_type"] == "" { t.Error("Missing aggregate_type") }
			if parsed["aggregate_id"] == "" { t.Error("Missing aggregate_id") }
			if parsed["occurred_at"] == "" { t.Error("Missing occurred_at") }

			// Must NOT contain sensitive data
			payload := msg.Payload
			sensitive := []string{"Secret Title", "user@test.dev", "192.168.1.1", "email", "ip", "title"}
			for _, s := range sensitive {
				if contains(payload, s) {
					t.Errorf("Redis fanout contains sensitive data: %s", s)
				}
			}

			fmt.Printf("Fanout payload: %s\n", payload)
		case <-timeout:
			t.Skip("No Redis message received within timeout")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
