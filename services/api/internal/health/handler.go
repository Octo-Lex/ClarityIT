package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool     *pgxpool.Pool
	build    string
	version  string
	uptime   time.Time
}

func NewHandler(pool *pgxpool.Pool, build string) *Handler {
	return &Handler{pool: pool, build: build, version: build, uptime: time.Now()}
}

func NewHandlerWithVersion(pool *pgxpool.Pool, version, gitCommit string) *Handler {
	build := version
	if gitCommit != "" {
		build = version + "+" + gitCommit[:7]
	}
	return &Handler{pool: pool, build: build, version: version, uptime: time.Now()}
}

type DeepHealthResponse struct {
	Build     string                 `json:"build"`
	Version   string                 `json:"version,omitempty"`
	Uptime    string                 `json:"uptime"`
	Timestamp string                 `json:"timestamp"`
	Postgres  ComponentHealth        `json:"postgres"`
	NATS      ComponentHealth        `json:"nats,omitempty"`
	Redis     ComponentHealth        `json:"redis,omitempty"`
	MinIO     ComponentHealth        `json:"minio,omitempty"`
	Outbox    OutboxHealth           `json:"outbox"`
	Workers   WorkersHealth          `json:"workers"`
	AgentRuns AgentRunsHealth        `json:"agent_runs"`
}

type ComponentHealth struct {
	Status  string `json:"status"`
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

type WorkersHealth struct {
	Outbox   WorkerHeartbeat `json:"outbox_worker"`
	Context  WorkerHeartbeat `json:"context_worker"`
	Reasoning WorkerHeartbeat `json:"reasoning_worker"`
}

type WorkerHeartbeat struct {
	LastSeen *time.Time `json:"last_seen,omitempty"`
	Status   string     `json:"status"`
}

func (h *Handler) Deep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	health := DeepHealthResponse{
		Build:     h.build,
		Version:   h.version,
		Uptime:    time.Since(h.uptime).String(),
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

	// Worker heartbeats (from DB if available)
	h.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM audit_logs WHERE action='outbox.batch_processed'`).Scan(&health.Workers.Outbox.LastSeen)
	h.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM audit_logs WHERE action LIKE 'context.%'`).Scan(&health.Workers.Context.LastSeen)
	
	health.Workers.Outbox.Status = workerStatus(health.Workers.Outbox.LastSeen)
	health.Workers.Context.Status = workerStatus(health.Workers.Context.LastSeen)
	health.Workers.Reasoning.Status = "unknown" // No heartbeat table yet

	// Update metrics counters
	M.OutboxPendingCount.Store(int64(health.Outbox.Pending))
	M.OutboxDeadLetterCount.Store(int64(health.Outbox.DeadLetter))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func workerStatus(lastSeen *time.Time) string {
	if lastSeen == nil {
		return "no_data"
	}
	if time.Since(*lastSeen) > 5*time.Minute {
		return "stale"
	}
	return "active"
}
