package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/natsx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
)

const (
	batchSize   = 50
	pollRate    = 500 * time.Millisecond
	maxAttempts = 5
	workerID    = "outbox-worker-1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config: %v", err)
	}

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB: %v", err)
	}
	defer pool.Close()

	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("clarityit-outbox-worker"),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(60),
	)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer nc.Close()

	js, err := natsx.Setup(nc)
	if err != nil {
		log.Fatalf("JetStream setup: %v", err)
	}

	// Redis for WebSocket fanout
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURLHost()})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis not available: %v", err)
		rdb = nil
	} else {
		log.Println("Redis connected")
	}

	log.Println("Outbox worker started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(pollRate)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			log.Println("Shutting down...")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			processBatch(ctx, pool, js, nc, rdb)
		}
	}
}

type outboxEvent struct {
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	Payload       json.RawMessage `json:"payload"`
	TeamID        *uuid.UUID      `json:"team_id"`
	Attempts      int             `json:"attempts"`
}

func processBatch(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, nc *nats.Conn, rdb *redis.Client) {
	ids, err := claimEvents(ctx, pool)
	if err != nil || len(ids) == 0 {
		return
	}

	events, err := fetchClaimed(ctx, pool, ids)
	if err != nil {
		return
	}

	for _, evt := range events {
		publishOne(ctx, pool, js, nc, rdb, evt)
	}
}

func claimEvents(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id FROM outbox_events
		WHERE status IN ('pending', 'failed')
		  AND next_attempt_at <= NOW()
		  AND (locked_at IS NULL OR locked_at < NOW() - INTERVAL '5 minutes')
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE outbox_events SET locked_at = NOW(), locked_by = $1, status = 'processing'
		WHERE id = ANY($2)
	`, workerID, ids)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return ids, nil
}

func fetchClaimed(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID) ([]outboxEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, event_type, aggregate_type, aggregate_id, payload, team_id, attempts
		FROM outbox_events WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []outboxEvent
	for rows.Next() {
		var e outboxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.AggregateType, &e.AggregateID, &e.Payload, &e.TeamID, &e.Attempts); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}

func publishOne(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, nc *nats.Conn, rdb *redis.Client, evt outboxEvent) {
	subject := evt.EventType
	if subject == "" {
		subject = "clarity.v1.unknown"
	}

	data, err := json.Marshal(map[string]any{
		"event_id":       evt.ID.String(),
		"event_type":     evt.EventType,
		"aggregate_type": evt.AggregateType,
		"aggregate_id":   evt.AggregateID,
		"team_id":        evt.TeamID,
		"payload":        evt.Payload,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		markFailed(ctx, pool, evt.ID, evt.EventType, "json marshal failed")
		return
	}

	_, err = js.Publish(ctx, subject, data)
	if err != nil {
		log.Printf("Publish failed %s: %v", evt.EventType, err)
		markFailed(ctx, pool, evt.ID, evt.EventType, err.Error())
		return
	}

	markSent(ctx, pool, evt.ID)

	// Publish sanitized fanout to Redis for WebSocket clients
	if rdb != nil && evt.TeamID != nil {
		fanout, _ := json.Marshal(map[string]string{
			"team_id":       evt.TeamID.String(),
			"event_type":     evt.EventType,
			"aggregate_type": evt.AggregateType,
			"aggregate_id":   evt.AggregateID,
			"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		})
		rdb.Publish(ctx, "clarity:ws:events", string(fanout))
	}
}

func markSent(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) {
	pool.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'sent', processed_at = NOW(), locked_at = NULL, locked_by = NULL
		WHERE id = $1
	`, id)
}

func markFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, eventType string, errMsg string) {
	sanitized := errMsg
	if len(sanitized) > 500 {
		sanitized = sanitized[:500]
	}

	var attempts int
	pool.QueryRow(ctx, "SELECT attempts FROM outbox_events WHERE id = $1", id).Scan(&attempts)
	attempts++

	backoff := time.Duration(math.Min(float64(int(math.Pow(2, float64(attempts)))), 300)) * time.Second

	if attempts >= maxAttempts {
		// Dead letter
		pool.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'dead_letter', dead_lettered_at = NOW(),
			    attempts = $2, last_error = $3, locked_at = NULL, locked_by = NULL
			WHERE id = $1
		`, id, attempts, sanitized)

		// Recursion guard: do NOT create DLQ events for DLQ events
		if !strings.HasPrefix(eventType, "clarity.dlq.") {
			pool.Exec(ctx, `
				INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, next_attempt_at, purge_after)
				VALUES ($1, 'clarity.dlq.outbox.publish_failed', 'outbox_event', $2, $3, 'pending', NOW(), NOW() + INTERVAL '30 days')
			`, uuid.New(), id, fmt.Sprintf(
				`{"outbox_event_id":"%s","event_type":"publish_failed","attempts":%d,"error_summary":"%s"}`,
				id.String(), attempts, sanitizeForJSON(sanitized),
			))
		}
	} else {
		pool.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'failed', attempts = $2, last_error = $3,
			    next_attempt_at = NOW() + $4, locked_at = NULL, locked_by = NULL
			WHERE id = $1
		`, id, attempts, sanitized, backoff)
	}
}

func sanitizeForJSON(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
