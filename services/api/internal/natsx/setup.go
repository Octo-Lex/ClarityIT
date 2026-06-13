package natsx

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go"
)

// Setup ensures the required JetStream streams exist.
func Setup(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}

	ctx := context.Background()

	// CLARITY_EVENTS — main event stream
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      "CLARITY_EVENTS",
		Subjects:  []string{"clarity.v1.>"},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		log.Printf("Stream CLARITY_EVENTS: %v (may already exist)", err)
	}

	// CLARITY_DLQ — dead letter queue
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      "CLARITY_DLQ",
		Subjects:  []string{"clarity.dlq.>"},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		log.Printf("Stream CLARITY_DLQ: %v (may already exist)", err)
	}

	return js, nil
}
