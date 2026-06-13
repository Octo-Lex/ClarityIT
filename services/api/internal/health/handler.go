package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// S3BucketChecker is a minimal interface for checking S3-compatible storage.
type S3BucketChecker interface {
	HeadBucket(ctx context.Context, bucket string) error
}

type Handler struct {
	pool    *pgxpool.Pool
	build   string
	version string
	uptime  time.Time
	nats    *nats.Conn
	rdb     *redis.Client
	s3      S3BucketChecker
	bucket  string
}

func NewHandler(pool *pgxpool.Pool, build string) *Handler {
	return &Handler{pool: pool, build: build, version: build, uptime: time.Now()}
}

func NewHandlerWithVersion(pool *pgxpool.Pool, version, gitCommit string) *Handler {
	build := version
	if gitCommit != "" && len(gitCommit) >= 7 {
		build = version + "+" + gitCommit[:7]
	}
	return &Handler{pool: pool, build: build, version: version, uptime: time.Now()}
}

// NewHandlerWithDeps creates a handler with real dependency clients.
func NewHandlerWithDeps(pool *pgxpool.Pool, version, gitCommit string, nc *nats.Conn, rdb *redis.Client, s3 S3BucketChecker, bucket string) *Handler {
	build := version
	if gitCommit != "" && len(gitCommit) >= 7 {
		build = version + "+" + gitCommit[:7]
	}
	return &Handler{
		pool:    pool,
		build:   build,
		version: version,
		uptime:  time.Now(),
		nats:    nc,
		rdb:     rdb,
		s3:      s3,
		bucket:  bucket,
	}
}

type DeepHealthResponse struct {
	Build     string          `json:"build"`
	Version   string          `json:"version,omitempty"`
	Uptime    string          `json:"uptime"`
	Timestamp string          `json:"timestamp"`
	Postgres  ComponentHealth `json:"postgres"`
	NATS      ComponentHealth `json:"nats"`
	Redis     ComponentHealth `json:"redis"`
	MinIO     ComponentHealth `json:"minio"`
	Outbox    OutboxHealth    `json:"outbox"`
	Workers   WorkersHealth   `json:"workers"`
	AgentRuns AgentRunsHealth `json:"agent_runs"`
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
	Pending int `json:"pending"`
	Running int `json:"running"`
}

type WorkersHealth struct {
	Outbox    WorkerHeartbeat `json:"outbox_worker"`
	Context   WorkerHeartbeat `json:"context_worker"`
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

	// NATS — real connectivity check
	health.NATS = h.checkNATS(ctx)

	// Redis — PING
	health.Redis = h.checkRedis(ctx)

	// MinIO/S3 — HeadBucket
	health.MinIO = h.checkMinIO(ctx)

	// Outbox counts
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL`).Scan(&health.Outbox.Pending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE dead_lettered_at IS NOT NULL`).Scan(&health.Outbox.DeadLetter)

	// Agent runs
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='pending'`).Scan(&health.AgentRuns.Pending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='running'`).Scan(&health.AgentRuns.Running)

	// Worker heartbeats
	h.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM audit_logs WHERE action='outbox.batch_processed'`).Scan(&health.Workers.Outbox.LastSeen)
	h.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM audit_logs WHERE action LIKE 'context.%'`).Scan(&health.Workers.Context.LastSeen)

	health.Workers.Outbox.Status = workerStatus(health.Workers.Outbox.LastSeen)
	health.Workers.Context.Status = workerStatus(health.Workers.Context.LastSeen)
	health.Workers.Reasoning.Status = "no_data"

	// Update metrics counters
	M.OutboxPendingCount.Store(int64(health.Outbox.Pending))
	M.OutboxDeadLetterCount.Store(int64(health.Outbox.DeadLetter))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// checkNATS performs a real connectivity check against NATS.
func (h *Handler) checkNATS(ctx context.Context) ComponentHealth {
	if h.nats == nil {
		return ComponentHealth{Status: "unknown"}
	}
	start := time.Now()

	switch h.nats.Status() {
	case nats.CONNECTED:
		// Try a JetStream API info request with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_, err := h.nats.RequestWithContext(timeoutCtx, "$JS.API.INFO", nil)
		latency := time.Since(start).String()
		if err != nil {
			return ComponentHealth{Status: "degraded", Latency: latency}
		}
		return ComponentHealth{Status: "up", Latency: latency}
	case nats.CLOSED:
		return ComponentHealth{Status: "down"}
	default:
		return ComponentHealth{Status: "degraded"}
	}
}

// checkRedis performs a PING against Redis.
func (h *Handler) checkRedis(ctx context.Context) ComponentHealth {
	if h.rdb == nil {
		return ComponentHealth{Status: "unknown"}
	}
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := h.rdb.Ping(timeoutCtx).Result()
	if err != nil {
		return ComponentHealth{Status: "down"}
	}
	return ComponentHealth{Status: "up", Latency: time.Since(start).String()}
}

// checkMinIO performs a HeadBucket against the S3-compatible endpoint.
func (h *Handler) checkMinIO(ctx context.Context) ComponentHealth {
	if h.s3 == nil || h.bucket == "" {
		return ComponentHealth{Status: "unknown"}
	}
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := h.s3.HeadBucket(timeoutCtx, h.bucket)
	if err != nil {
		return ComponentHealth{Status: "down"}
	}
	return ComponentHealth{Status: "up", Latency: time.Since(start).String()}
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
