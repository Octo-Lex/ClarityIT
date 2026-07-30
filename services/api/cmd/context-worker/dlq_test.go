package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clarityit/api/internal/contextx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// dlqDBURL is the test database (PostgreSQL only — no NATS needed for DLQ tests).
const dlqDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

// invalidUUID reliably fails uuid.Parse, forcing contextx.Ingest to error at
// its aggregate_id validation (ingester.go:31) regardless of aggregate_type.
const invalidUUID = "not-a-valid-uuid"

// fakeMsg is a minimal jetstream.Msg implementation for processMessage tests.
// It captures Ack/Nak/Term decisions and returns canned metadata/data/subject.
// var _ jetstream.Msg enforces the FULL interface at compile time, so if
// jetstream v1.52 adds or renames a method, this fails to build.
var _ jetstream.Msg = (*fakeMsg)(nil)

type fakeMsg struct {
	mu       sync.Mutex
	data     []byte
	subject  string
	meta     *jetstream.MsgMetadata
	ackState string // "" | "ack" | "nak" | "term"
	termCb   func() // invoked when Term() is called (e.g. to record row state)
}

func (f *fakeMsg) Ack() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackState = "ack"
	return nil
}
func (f *fakeMsg) DoubleAck(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackState = "ack"
	return nil
}
func (f *fakeMsg) Nak() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackState = "nak"
	return nil
}
func (f *fakeMsg) NakWithDelay(_ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ackState = "nak"
	return nil
}
func (f *fakeMsg) InProgress() error { return nil }
func (f *fakeMsg) Term() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.termCb != nil {
		f.termCb()
	}
	f.ackState = "term"
	return nil
}
func (f *fakeMsg) TermWithReason(_ string) error { return f.Term() }
func (f *fakeMsg) Data() []byte                   { return f.data }
func (f *fakeMsg) Subject() string                { return f.subject }
func (f *fakeMsg) Reply() string                  { return "" }
func (f *fakeMsg) Headers() nats.Header           { return nil }
func (f *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	if f.meta == nil {
		return nil, fmt.Errorf("no metadata")
	}
	return f.meta, nil
}

// newFakeMsg builds a fakeMsg with given data, subject, stream, and stream sequence.
func newFakeMsg(data []byte, stream string, streamSeq uint64, delivered uint64) *fakeMsg {
	return &fakeMsg{
		data:    data,
		subject: "clarity.v1.test",
		meta: &jetstream.MsgMetadata{
			Stream:       stream,
			Sequence:     jetstream.SequencePair{Stream: streamSeq},
			NumDelivered: delivered,
		},
	}
}

// poolForDLQ connects to the test DB, skipping if unavailable.
func poolForDLQ(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), dlqDBURL)
	if err != nil {
		t.Skipf("DB unavailable: %v", err)
	}
	return pool
}

// findDLQRow looks up a dead-letter row by the original_event_id stored in its
// sanitized payload (the row's event_type column holds the dlqType, not the
// original event). Returns the payload, status, attempts, aggregate_id, last_error.
func findDLQRow(t *testing.T, pool *pgxpool.Pool, originalEventID string) (payload []byte, status string, attempts int, aggregateID *string, lastErr *string) {
	t.Helper()
	err := pool.QueryRow(t.Context(), `
		SELECT payload, status, attempts, aggregate_id::text, last_error
		FROM outbox_events
		WHERE payload->>'original_event_id' = $1
	`, originalEventID).Scan(&payload, &status, &attempts, &aggregateID, &lastErr)
	if err != nil {
		t.Fatalf("DLQ row for original_event_id %s not found: %v", originalEventID, err)
	}
	return
}

// uniqueEventID returns a fresh UUID string for use as env.EventID, guaranteeing
// test isolation for findDLQRow lookups.
func uniqueEventID() string { return uuid.New().String() }

// cleanupDLQ deletes the row for a given original_event_id.
func cleanupDLQ(pool *pgxpool.Pool, originalEventID string) {
	pool.Exec(context.Background(),
		"DELETE FROM outbox_events WHERE payload->>'original_event_id' = $1", originalEventID)
}

// TestProcessMessageRetriesBelowThresholdNaks: deliveries < maxIngestRetries → Nak, no DLQ row.
func TestProcessMessageRetriesBelowThresholdNaks(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.naktest",
		AggregateType: "object",
		AggregateID:   invalidUUID, // forces Ingest error reliably
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9101, 3) // delivery 3 < 10

	processMessage(t.Context(), pool, msg)

	if msg.ackState != "nak" {
		t.Fatalf("ackState = %q; want nak", msg.ackState)
	}
	var n int
	pool.QueryRow(t.Context(),
		"SELECT count(*) FROM outbox_events WHERE payload->>'original_event_id' = $1", eventID,
	).Scan(&n)
	if n != 0 {
		t.Errorf("expected 0 DLQ rows below threshold; got %d", n)
	}
}

// TestProcessMessageAtThresholdTermsAndRecordsDLQ: deliveries >= maxIngestRetries →
// Term + a durable dead-letter row.
func TestProcessMessageAtThresholdTermsAndRecordsDLQ(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.threshold",
		AggregateType: "object",
		AggregateID:   invalidUUID, // invalid → Ingest errors; tests NULL aggregate_id
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9102, uint64(maxIngestRetries))
	defer cleanupDLQ(pool, eventID)

	processMessage(t.Context(), pool, msg)

	if msg.ackState != "term" {
		t.Fatalf("ackState = %q; want term", msg.ackState)
	}

	_, status, attempts, aggregateID, lastErr := findDLQRow(t, pool, eventID)
	if status != "dead_letter" {
		t.Errorf("status = %q; want dead_letter", status)
	}
	if attempts != maxIngestRetries {
		t.Errorf("attempts = %d; want %d", attempts, maxIngestRetries)
	}
	if lastErr == nil || *lastErr == "" {
		t.Error("last_error is NULL/empty")
	}
	// invalid aggregate_id must be stored as NULL, not fail the insert
	if aggregateID != nil {
		t.Errorf("aggregate_id = %v; want nil (NULL)", *aggregateID)
	}
}

// TestProcessMessageUnparseableJSONTermsAndRecordsDLQ: bad bytes → DLQ with
// clarity.dlq.context.unparseable, aggregate_id NULL.
func TestProcessMessageUnparseableJSONTermsAndRecordsDLQ(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	const delivered = uint64(7)
	msg := newFakeMsg([]byte("not json at all"), "CLARITY_EVENTS", 9103, delivered)
	dispositionID := uuid.NewSHA1(
		dlqNamespace,
		[]byte("context:CLARITY_EVENTS:9103"),
	)
	_, _ = pool.Exec(t.Context(), "DELETE FROM outbox_events WHERE id=$1", dispositionID)
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE id=$1", dispositionID)
	})

	processMessage(t.Context(), pool, msg)

	if msg.ackState != "term" {
		t.Fatalf("ackState = %q; want term", msg.ackState)
	}

	// The unparseable path has no env, so no original_event_id; look it up by
	// its dlqType and source directly.
	var (
		eventType   string
		aggregateID *string
		attempts    int
		deliveries  int
	)
	err := pool.QueryRow(t.Context(), `
		SELECT event_type, aggregate_id::text, attempts,
		       (payload->>'deliveries')::int
		FROM outbox_events
		WHERE id = $1
	`, dispositionID).Scan(&eventType, &aggregateID, &attempts, &deliveries)
	if err != nil {
		t.Fatalf("unparseable DLQ row not found: %v", err)
	}
	if eventType != "clarity.dlq.context.unparseable" {
		t.Errorf("event_type = %q; want clarity.dlq.context.unparseable", eventType)
	}
	if aggregateID != nil {
		t.Errorf("aggregate_id = %v; want nil", *aggregateID)
	}
	if attempts != int(delivered) {
		t.Errorf("attempts = %d; want %d", attempts, delivered)
	}
	if deliveries != int(delivered) {
		t.Errorf("payload deliveries = %d; want %d", deliveries, delivered)
	}
}

// TestProcessMessageDLQRecursionGuard: a clarity.dlq.* event that fails to ingest
// must NOT create another DLQ row.
func TestProcessMessageDLQRecursionGuard(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.dlq.outbox.publish_failed", // recursion-guarded
		AggregateType: "object",
		AggregateID:   invalidUUID,
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9104, uint64(maxIngestRetries))

	before := countDLQRows(t, pool)
	processMessage(t.Context(), pool, msg)
	after := countDLQRows(t, pool)

	if msg.ackState != "term" {
		t.Errorf("ackState = %q; want term (still terminates)", msg.ackState)
	}
	if after != before {
		t.Errorf("DLQ rows increased %d → %d; recursion guard failed", before, after)
	}
}

func countDLQRows(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	pool.QueryRow(t.Context(),
		"SELECT count(*) FROM outbox_events WHERE status='dead_letter'",
	).Scan(&n)
	return n
}

// TestProcessMessageInvalidAggregateUUIDStillRecords: an invalid aggregate_id
// must not break the DLQ insert; it is stored as NULL.
func TestProcessMessageInvalidAggregateUUIDStillRecords(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.baduuid",
		AggregateType: "object",
		AggregateID:   invalidUUID,
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9105, uint64(maxIngestRetries))
	defer cleanupDLQ(pool, eventID)

	processMessage(t.Context(), pool, msg)

	_, _, _, aggregateID, _ := findDLQRow(t, pool, eventID)
	if aggregateID != nil {
		t.Errorf("aggregate_id = %v; want nil", *aggregateID)
	}
}

// TestProcessMessageDLQWriteFailureStillTerms: if the DB write fails (forced via
// a cancelled context), Term must still be called. processMessage receives a
// pool, not a tx; we cancel the context to induce Exec failure.
func TestProcessMessageDLQWriteFailureStillTerms(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.dlqfail",
		AggregateType: "object",
		AggregateID:   invalidUUID,
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9106, uint64(maxIngestRetries))

	// Cancelled context → both Ingest and the DLQ Exec fail; Term must still run.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	processMessage(ctx, pool, msg)

	if msg.ackState != "term" {
		t.Errorf("ackState = %q; want term even after DLQ write failure", msg.ackState)
	}
}

// TestProcessMessageRowExistsWhenTermInvoked: the DLQ row must be committed
// BEFORE Term fires. We capture the row count inside Term's callback.
func TestProcessMessageRowExistsWhenTermInvoked(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.ordering",
		AggregateType: "object",
		AggregateID:   invalidUUID,
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9107, uint64(maxIngestRetries))
	defer cleanupDLQ(pool, eventID)

	rowsAtTerm := -1
	msg.termCb = func() {
		var n int
		pool.QueryRow(context.Background(),
			"SELECT count(*) FROM outbox_events WHERE payload->>'original_event_id' = $1", eventID,
		).Scan(&n)
		rowsAtTerm = n
	}

	processMessage(t.Context(), pool, msg)

	if msg.ackState != "term" {
		t.Fatalf("ackState = %q; want term", msg.ackState)
	}
	if rowsAtTerm != 1 {
		t.Errorf("rows at Term time = %d; want 1 (row must exist before Term)", rowsAtTerm)
	}
}

// TestProcessMessageRepeatedTerminalNoDuplicates: processing the same terminal
// message twice yields exactly one DLQ row (deterministic ID + ON CONFLICT).
func TestProcessMessageRepeatedTerminalNoDuplicates(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.dupes",
		AggregateType: "object",
		AggregateID:   invalidUUID,
	}
	data, _ := json.Marshal(env)

	// Same stream+sequence → same deterministic disposition ID → ON CONFLICT.
	msg1 := newFakeMsg(data, "CLARITY_EVENTS", 9108, uint64(maxIngestRetries))
	msg2 := newFakeMsg(data, "CLARITY_EVENTS", 9108, uint64(maxIngestRetries))
	defer cleanupDLQ(pool, eventID)

	processMessage(t.Context(), pool, msg1)
	processMessage(t.Context(), pool, msg2)

	var n int
	pool.QueryRow(t.Context(),
		"SELECT count(*) FROM outbox_events WHERE payload->>'original_event_id' = $1", eventID,
	).Scan(&n)
	if n != 1 {
		t.Errorf("expected 1 DLQ row after duplicate terminal processing; got %d", n)
	}
}

// TestProcessMessageCanaryPayloadExcludesRawBytes: a unique secret placed in
// env.Payload must NOT appear in the persisted DLQ payload, and the stored
// payload must have exactly the allowlist keys (no raw-bytes field).
func TestProcessMessageCanaryPayloadExcludesRawBytes(t *testing.T) {
	pool := poolForDLQ(t)
	defer pool.Close()

	eventID := uniqueEventID()
	secret := "CANARY_SECRET_" + uuid.New().String()
	env := contextx.Envelope{
		EventID:       eventID,
		EventType:     "clarity.v1.object.canary",
		AggregateType: "object",
		AggregateID:   invalidUUID,
		Payload:       json.RawMessage(`{"token":"` + secret + `","data":"sensitive"}`),
	}
	data, _ := json.Marshal(env)
	msg := newFakeMsg(data, "CLARITY_EVENTS", 9109, uint64(maxIngestRetries))
	defer cleanupDLQ(pool, eventID)

	processMessage(t.Context(), pool, msg)

	payload, _, _, _, _ := findDLQRow(t, pool, eventID)

	// (a) The secret must be absent from the persisted payload.
	if strings.Contains(string(payload), secret) {
		t.Errorf("persisted DLQ payload contains the canary secret %q:\n%s", secret, payload)
	}

	// (b) Exact key allowlist — no raw-bytes / payload-equivalent field.
	var stored map[string]any
	if err := json.Unmarshal(payload, &stored); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	allowed := map[string]bool{
		"original_event_id": true,
		"event_type":        true,
		"subject":           true,
		"stream":            true,
		"stream_sequence":   true,
		"deliveries":        true,
		"source":            true,
		"error_summary":     true,
	}
	for k := range stored {
		if !allowed[k] {
			t.Errorf("payload has unexpected key %q (not in allowlist)", k)
		}
	}
}
