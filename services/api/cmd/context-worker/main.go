package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/contextx"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/natsx"
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

func processMessage(ctx context.Context, pool *pgxpool.Pool, msg jetstream.Msg) {
	var env contextx.Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		log.Printf("Invalid message: %v", err)
		msg.Nak()
		return
	}

	if err := contextx.Ingest(ctx, pool, env); err != nil {
		log.Printf("Ingest failed %s: %v", env.EventType, err)
		msg.Nak()
		return
	}

	msg.Ack()
}
