package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/contextx"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/natsx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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
		nats.Name("clarityit-context-worker"),
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

	// Create durable consumer on CLARITY_EVENTS
	consumer, err := js.CreateConsumer(ctx, "CLARITY_EVENTS", jetstream.ConsumerConfig{
		Name:    "context-ingester",
		Durable: "context-ingester",
	})
	if err != nil {
		log.Printf("Consumer (may already exist): %v", err)
		consumer, err = js.Consumer(ctx, "CLARITY_EVENTS", "context-ingester")
		if err != nil {
			log.Fatalf("Get consumer: %v", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Context worker started, consuming CLARITY_EVENTS")

	// Consume messages
	cctx, ccancel := context.WithCancel(ctx)
	defer ccancel()

	msgs := make(chan jetstream.Msg, 100)

	_, err = consumer.Consume(func(msg jetstream.Msg) {
		msgs <- msg
	}, jetstream.PullMaxMessages(10))
	if err != nil {
		log.Fatalf("Consume: %v", err)
	}

	for {
		select {
		case <-sigCh:
			log.Println("Shutting down...")
			return
		case <-cctx.Done():
			return
		case msg := <-msgs:
			processMessage(ctx, pool, msg)
		}
	}
}

// maxIngestRetries bounds redelivery for a single event. After this many failed
// attempts the message is Terminated (Term = terminal Nak, no redelivery) so a
// single poison-pill event can't pin a core forever in a tight Nak loop.
// 13 days of CPU at ~100%/core in production traced to exactly this: one
// unprocessable object.commented event redelivered nonstop with no escape.
const maxIngestRetries = 10

func processMessage(ctx context.Context, pool *pgxpool.Pool, msg jetstream.Msg) {
	var env contextx.Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		// Unparseable JSON: cannot even build an Envelope. Record a durable
		// dead-letter with no aggregate_id before terminating so the operator
		// can see (but not replay) the poison event through /api/admin/ops.
		delivered := deliveryCount(msg)
		log.Printf("Invalid message (unparseable JSON, term): %v", err)
		recordDeadLetter(ctx, pool, dlqParams{
			dlqType:   "clarity.dlq.context.unparseable",
			delivered: delivered,
			source:    "unparseable_json",
			errMsg:    err.Error(),
			msg:       msg,
			// env is zero-valued: AggregateID is "", EventType is "".
		})
		termChecked(msg)
		return
	}

	if err := contextx.Ingest(ctx, pool, env); err != nil {
		// Count deliveries; if we've retried too many times, dead-letter (Term)
		// instead of Nak so the event stops redelivering.
		delivered := deliveryCount(msg)
		if delivered >= maxIngestRetries {
			log.Printf("Ingest failed %s after %d deliveries, terming (dead-letter): %v",
				env.EventType, delivered, err)
			// Recursion guard: a DLQ event that fails to ingest must not spawn
			// another DLQ row (prevents DLQ-of-DLQ proliferation).
			if !strings.HasPrefix(env.EventType, "clarity.dlq.") {
				recordDeadLetter(ctx, pool, dlqParams{
					dlqType:   "clarity.dlq.context.ingest_failed",
					env:       env,
					delivered: delivered,
					source:    "ingest_retry_exhausted",
					errMsg:    err.Error(),
					msg:       msg,
				})
			}
			termChecked(msg)
			return
		}
		log.Printf("Ingest failed %s (delivery %d, will retry): %v",
			env.EventType, delivered, err)
		msg.Nak()
		return
	}

	msg.Ack()
}

// deliveryCount returns JetStream's authoritative delivery count when metadata
// is available. A message received without metadata has still been delivered at
// least once, so the safe fallback is 1 rather than a misleading zero.
func deliveryCount(msg jetstream.Msg) uint64 {
	if meta, err := msg.Metadata(); err == nil && meta.NumDelivered > 0 {
		return meta.NumDelivered
	}
	return 1
}

// termChecked calls msg.Term and logs any error it returns. Term is the safety
// net that prevents a poison event from pinning a core in a redelivery loop;
// it MUST always execute, even if the durable dead-letter write failed.
func termChecked(msg jetstream.Msg) {
	if tErr := msg.Term(); tErr != nil {
		log.Printf("msg.Term error: %v", tErr)
	}
}

// dlqNamespace is the fixed UUIDv5 namespace used to derive deterministic
// disposition IDs for dead-letter rows. A uuid.UUID cannot be a const.
var dlqNamespace = uuid.NewSHA1(
	uuid.NameSpaceDNS,
	[]byte("clarityit/context-worker/dlq"),
)

// dlqParams carries the inputs needed to record one durable dead-letter row.
type dlqParams struct {
	dlqType   string              // clarity.dlq.context.ingest_failed | .unparseable
	env       contextx.Envelope   // zero-valued for the unparseable path
	delivered uint64              // JetStream NumDelivered (0 when unknown)
	source    string              // human-readable failure origin
	errMsg    string              // raw error, sanitized + truncated before storage
	msg       jetstream.Msg       // source of subject/stream/sequence traceability
}

// recordDeadLetter writes one durable outbox_events row with status='dead_letter'
// so the poison event is visible (but NOT replayable) through the existing
// /api/admin/ops/dead-letters and /api/admin/ops/outbox views and the health
// dead-letter gauge. The row carries a sanitized summary only — never the raw
// message bytes. Idempotent via a deterministic disposition ID + ON CONFLICT.
//
// Capability note: this is durable terminal disposition + operator visibility.
// It is NOT NATS-DLQ publication or replay/redrive (AC-00-27 replay is future).
func recordDeadLetter(ctx context.Context, pool *pgxpool.Pool, p dlqParams) {
	// Derive a deterministic disposition ID from stream+sequence so a crash
	// between insert and Term cannot create a duplicate on redelivery.
	subject, stream, streamSeq := streamIdentity(p.msg)
	dispID := uuid.New()
	if stream != "" {
		dispID = uuid.NewSHA1(dlqNamespace, []byte("context:"+stream+":"+strconv.FormatUint(streamSeq, 10)))
	} else {
		// Metadata unavailable: do not collapse all such events onto "":0.
		// Use a random ID and log that idempotency is degraded.
		log.Printf("idempotency degraded: stream metadata unavailable, using random DLQ id %s", dispID)
	}

	// aggregate_id: parse to UUID when valid, NULL otherwise. Never insert an
	// invalid string into the UUID column (an invalid UUID is itself a likely
	// ingest failure and would make the DLQ insert fail).
	var aggregateID any // nil → SQL NULL
	if p.env.AggregateID != "" {
		if parsed, pErr := uuid.Parse(p.env.AggregateID); pErr == nil {
			aggregateID = parsed
		}
	}

	// aggregate_type: normalize empty/unparseable to "unknown".
	aggregateType := p.env.AggregateType
	if aggregateType == "" {
		aggregateType = "unknown"
	}

	// Sanitized summary payload — exact-key allowlist, built with json.Marshal
	// (never string formatting). No raw message bytes.
	summary := map[string]any{
		"original_event_id": p.env.EventID,
		"event_type":        p.env.EventType,
		"subject":           subject,
		"stream":            stream,
		"stream_sequence":   streamSeq,
		"deliveries":        p.delivered,
		"source":            p.source,
		"error_summary":     truncateRunes(p.errMsg, 500),
	}
	payload, err := json.Marshal(summary)
	if err != nil {
		// Should be impossible for this map shape; fall back to a minimal object.
		log.Printf("DLQ payload marshal error: %v", err)
		payload = []byte(`{"source":"dlq_payload_marshal_error"}`)
	}

	_, execErr := pool.Exec(ctx, `
		INSERT INTO outbox_events
			(id, event_type, event_version, aggregate_type, aggregate_id,
			 payload, status, dead_lettered_at, attempts, last_error,
			 next_attempt_at, purge_after)
		VALUES ($1, $2, 1, $3, $4,
			    $5, 'dead_letter', NOW(), $6, $7,
			    NOW(), NOW() + INTERVAL '30 days')
		ON CONFLICT (id) DO NOTHING
	`, dispID, p.dlqType, aggregateType, aggregateID,
		payload, p.delivered, truncateRunes(p.errMsg, 500))
	if execErr != nil {
		// Log only — Term still runs. Never resurrect the redelivery loop.
		log.Printf("DLQ durable write failed for %s (terming anyway): %v", p.dlqType, execErr)
	}
}

// streamIdentity extracts subject/stream/stream_sequence from a JetStream
// message's metadata for traceability. Returns "" / "" / 0 when unavailable.
func streamIdentity(msg jetstream.Msg) (subject, stream string, streamSeq uint64) {
	meta, err := msg.Metadata()
	if err != nil {
		return "", "", 0
	}
	return msg.Subject(), meta.Stream, meta.Sequence.Stream
}

// truncateRunes truncates s to at most maxRunes Unicode code points, safe for
// multi-byte UTF-8 (byte slicing would split a rune).
func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}
