package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/health"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OpsHandler provides read-only operator visibility endpoints.
type OpsHandler struct {
	pool         *pgxpool.Pool
	healthCheck  *health.Handler
}

func NewOpsHandler(pool *pgxpool.Pool, hc *health.Handler) *OpsHandler {
	return &OpsHandler{pool: pool, healthCheck: hc}
}

// Routes returns a chi.Router for ops endpoints.
func (h *OpsHandler) Routes() func(r chiRouter) {
	return nil // wired directly in main.go
}

// ─── Summary: aggregate health + counts ───

func (h *OpsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Gather system counts
	summary := map[string]any{}

	// Outbox
	var outboxPending, deadLetters int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE processed_at IS NULL`).Scan(&outboxPending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE dead_lettered_at IS NOT NULL`).Scan(&deadLetters)
	summary["outbox_pending"] = outboxPending
	summary["dead_letters"] = deadLetters

	// Agent runs
	var agentPending, agentRunning int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='pending'`).Scan(&agentPending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='running'`).Scan(&agentRunning)
	summary["agent_runs_pending"] = agentPending
	summary["agent_runs_running"] = agentRunning

	// Recent rejections/blocks (last 24h)
	var webhookRejects, agentBlocks, securityEvents int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action IN ('integration.webhook.rejected','integration.webhook.rate_limited') AND created_at > NOW() - INTERVAL '24 hours'`).Scan(&webhookRejects)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_effect_results WHERE status IN ('blocked','denied') AND created_at > NOW() - INTERVAL '24 hours'`).Scan(&agentBlocks)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%.failed' OR action LIKE '%.denied' OR action LIKE '%.blocked' AND created_at > NOW() - INTERVAL '24 hours'`).Scan(&securityEvents)

	summary["webhook_rejections_24h"] = webhookRejects
	summary["agent_blocks_24h"] = agentBlocks
	summary["security_events_24h"] = securityEvents

	// Integration keys
	var totalKeys, rotationRequired int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM integration_api_keys WHERE revoked_at IS NULL`).Scan(&totalKeys)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM integration_api_keys WHERE revoked_at IS NULL AND rotation_required = true`).Scan(&rotationRequired)
	summary["integration_keys_active"] = totalKeys
	summary["integration_keys_rotation_required"] = rotationRequired

	// User count
	var totalUsers int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&totalUsers)
	summary["total_users"] = totalUsers

	// Team count
	var totalTeams int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM teams WHERE deleted_at IS NULL`).Scan(&totalTeams)
	summary["total_teams"] = totalTeams

	writeJSON(w, http.StatusOK, summary)
}

// ─── Outbox detail ───

func (h *OpsHandler) Outbox(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := parseLimit(r.URL.Query().Get("limit"), 50)

	rows, err := h.pool.Query(ctx, `
		SELECT id::text, event_type, aggregate_type, aggregate_id, created_at, processed_at, dead_lettered_at, attempts, last_error
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load outbox")
		return
	}
	defer rows.Close()

	type OutboxEvent struct {
		ID             string     `json:"id"`
		EventType      string     `json:"event_type"`
		AggregateType  string     `json:"aggregate_type"`
		AggregateID    string     `json:"aggregate_id"`
		CreatedAt      time.Time  `json:"created_at"`
		ProcessedAt    *time.Time `json:"processed_at"`
		DeadLetteredAt *time.Time `json:"dead_lettered_at"`
		Attempts       int        `json:"attempts"`
		ErrorMesssage  string     `json:"error_message"`
	}

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		rows.Scan(&e.ID, &e.EventType, &e.AggregateType, &e.AggregateID, &e.CreatedAt, &e.ProcessedAt, &e.DeadLetteredAt, &e.Attempts, &e.ErrorMesssage)
		events = append(events, e)
	}
	if events == nil {
		events = []OutboxEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

// ─── Dead letters ───

func (h *OpsHandler) DeadLetters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := parseLimit(r.URL.Query().Get("limit"), 50)

	rows, err := h.pool.Query(ctx, `
		SELECT id::text, event_type, aggregate_type, aggregate_id, created_at, dead_lettered_at, attempts, last_error
		FROM outbox_events
		WHERE dead_lettered_at IS NOT NULL
		ORDER BY dead_lettered_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load dead letters")
		return
	}
	defer rows.Close()

	type DeadLetter struct {
		ID             string    `json:"id"`
		EventType      string    `json:"event_type"`
		AggregateType  string    `json:"aggregate_type"`
		AggregateID    string    `json:"aggregate_id"`
		CreatedAt      time.Time `json:"created_at"`
		DeadLetteredAt time.Time `json:"dead_lettered_at"`
		Attempts       int       `json:"attempts"`
		ErrorMessage   string    `json:"error_message"`
		// NOTE: payload intentionally excluded — never expose raw event payloads
	}

	var items []DeadLetter
	for rows.Next() {
		var d DeadLetter
		rows.Scan(&d.ID, &d.EventType, &d.AggregateType, &d.AggregateID, &d.CreatedAt, &d.DeadLetteredAt, &d.Attempts, &d.ErrorMessage)
		items = append(items, d)
	}
	if items == nil {
		items = []DeadLetter{}
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Workers ───

func (h *OpsHandler) Workers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type WorkerStatus struct {
		Name     string     `json:"name"`
		LastSeen *time.Time `json:"last_seen"`
		Status   string     `json:"status"`
	}

	var outboxLast, contextLast *time.Time
	h.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM audit_logs WHERE action='outbox.batch_processed'`).Scan(&outboxLast)
	h.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM audit_logs WHERE action LIKE 'context.%'`).Scan(&contextLast)

	workers := []WorkerStatus{
		{Name: "outbox_worker", LastSeen: outboxLast, Status: workerAge(outboxLast)},
		{Name: "context_worker", LastSeen: contextLast, Status: workerAge(contextLast)},
		{Name: "reasoning_worker", LastSeen: nil, Status: "no_data"},
	}

	writeJSON(w, http.StatusOK, workers)
}

// ─── Webhook rejections ───

func (h *OpsHandler) WebhookRejections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := parseLimit(r.URL.Query().Get("limit"), 50)

	rows, err := h.pool.Query(ctx, `
		SELECT event_id, action, change_summary, created_at
		FROM audit_logs
		WHERE action IN ('integration.webhook.rejected', 'integration.webhook.rate_limited', 'integration.webhook.received')
			AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load webhook events")
		return
	}
	defer rows.Close()

	type WebhookEvent struct {
		EventID   string    `json:"event_id"`
		Action    string    `json:"action"`
		Summary   string    `json:"summary"`
		CreatedAt time.Time `json:"created_at"`
		// NOTE: new_value excluded — contains payload_hash only, but redacted for safety
	}

	var events []WebhookEvent
	for rows.Next() {
		var e WebhookEvent
		rows.Scan(&e.EventID, &e.Action, &e.Summary, &e.CreatedAt)
		events = append(events, e)
	}
	if events == nil {
		events = []WebhookEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

// ─── Agent blocked actions ───

func (h *OpsHandler) AgentBlocks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := parseLimit(r.URL.Query().Get("limit"), 50)

	rows, err := h.pool.Query(ctx, `
		SELECT er.id::text, er.intention_id::text, er.status, er.tool_name, er.result, er.created_at
		FROM agent_effect_results er
		WHERE er.status IN ('blocked', 'denied')
		ORDER BY er.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load agent blocks")
		return
	}
	defer rows.Close()

	type AgentBlock struct {
		EffectID      string    `json:"effect_id"`
		IntentionID   string    `json:"intention_id"`
		Status        string    `json:"status"`
		ToolName      string    `json:"tool_name"`
		Result        string    `json:"result"`
		CreatedAt     time.Time `json:"created_at"`
	}

	var blocks []AgentBlock
	for rows.Next() {
		var b AgentBlock
		rows.Scan(&b.EffectID, &b.IntentionID, &b.Status, &b.ToolName, &b.Result, &b.CreatedAt)
		blocks = append(blocks, b)
	}
	if blocks == nil {
		blocks = []AgentBlock{}
	}
	writeJSON(w, http.StatusOK, blocks)
}

// ─── Helpers ───

func parseLimit(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
		if n > 500 {
			return 500
		}
	}
	if n == 0 {
		return def
	}
	return n
}

func workerAge(lastSeen *time.Time) string {
	if lastSeen == nil {
		return "no_data"
	}
	age := time.Since(*lastSeen)
	if age > 5*time.Minute {
		return "stale"
	}
	return "active"
}

// chiRouter is an interface to avoid importing chi in this file
type chiRouter interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}

var _ = json.Marshal
var _ = context.Background
