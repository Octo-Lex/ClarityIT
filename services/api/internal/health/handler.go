package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool  *pgxpool.Pool
	build string
}

func NewHandler(pool *pgxpool.Pool, build string) *Handler {
	return &Handler{pool: pool, build: build}
}

type DeepHealth struct {
	Build        string                 `json:"build"`
	Postgres     ComponentHealth        `json:"postgres"`
	Outbox       OutboxHealth           `json:"outbox"`
	AgentRuns    AgentRunsHealth        `json:"agent_runs"`
	Timestamp    string                 `json:"timestamp"`
}

type ComponentHealth struct {
	Status string `json:"status"`
	Latency string `json:"latency,omitempty"`
}

type OutboxHealth struct {
	Pending    int `json:"pending"`
	DeadLetter int `json:"dead_letter"`
}

type AgentRunsHealth struct {
	Pending  int `json:"pending"`
	Running  int `json:"running"`
}

func (h *Handler) Deep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	health := DeepHealth{
		Build:     h.build,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// PostgreSQL
	start := time.Now()
	if err := h.pool.Ping(ctx); err != nil {
		health.Postgres = ComponentHealth{Status: "down"}
	} else {
		health.Postgres = ComponentHealth{Status: "up", Latency: time.Since(start).String()}
	}

	// Outbox counts
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL`).Scan(&health.Outbox.Pending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE dead_lettered_at IS NOT NULL`).Scan(&health.Outbox.DeadLetter)

	// Agent runs
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='pending'`).Scan(&health.AgentRuns.Pending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='running'`).Scan(&health.AgentRuns.Running)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
