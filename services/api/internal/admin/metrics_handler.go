package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MetricsHandler provides read-only operational metrics.
type MetricsHandler struct {
	pool *pgxpool.Pool
}

func NewMetricsHandler(pool *pgxpool.Pool) *MetricsHandler {
	return &MetricsHandler{pool: pool}
}

// Metrics computes operational throughput and outcome metrics from PostgreSQL.
// Returns aggregated counts and averages — no raw payloads, targets, or secrets.
func (h *MetricsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	result := map[string]any{
		"approvals":     h.approvalMetrics(ctx),
		"remediations":  h.remediationMetrics(ctx),
		"asset_actions": h.assetActionMetrics(ctx),
		"agents":        h.agentMetrics(ctx),
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MetricsHandler) approvalMetrics(ctx context.Context) map[string]any {
	var pending, approved, rejected, expired, executed, failed int

	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status='pending'`).Scan(&pending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status='approved'`).Scan(&approved)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status='rejected'`).Scan(&rejected)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status='expired'`).Scan(&expired)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status='executed'`).Scan(&executed)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status='failed'`).Scan(&failed)

	// Average time to decision: for approved/rejected only, measure from created_at to updated_at
	var avgTimeToDecision *float64
	h.pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (updated_at - created_at)))
		FROM approval_requests
		WHERE status IN ('approved', 'rejected')
	`).Scan(&avgTimeToDecision)

	avgDecision := 0.0
	if avgTimeToDecision != nil {
		avgDecision = *avgTimeToDecision
	}

	return map[string]any{
		"pending":                     pending,
		"approved":                    approved,
		"rejected":                    rejected,
		"expired":                     expired,
		"executed":                    executed,
		"failed":                      failed,
		"avg_time_to_decision_seconds": avgDecision,
	}
}

func (h *MetricsHandler) remediationMetrics(ctx context.Context) map[string]any {
	var draft, proposed, approved, executing, completed, failed, cancelled int

	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='draft'`).Scan(&draft)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='proposed'`).Scan(&proposed)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='approved'`).Scan(&approved)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='executing'`).Scan(&executing)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='completed'`).Scan(&completed)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='failed'`).Scan(&failed)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM remediation_proposals WHERE status='cancelled'`).Scan(&cancelled)

	return map[string]any{
		"draft":      draft,
		"proposed":   proposed,
		"approved":   approved,
		"executing":  executing,
		"completed":  completed,
		"failed":     failed,
		"cancelled":  cancelled,
	}
}

func (h *MetricsHandler) assetActionMetrics(ctx context.Context) map[string]any {
	// Counts by status
	var pending, approvedStatus, executing, succeeded, failed, cancelled int

	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE status='pending'`).Scan(&pending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE status='approved'`).Scan(&approvedStatus)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE status='executing'`).Scan(&executing)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE status='succeeded'`).Scan(&succeeded)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE status='failed'`).Scan(&failed)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE status='cancelled'`).Scan(&cancelled)

	// Counts by action type
	var startCount, shutdownCount, stopCount, snapshotCount int
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE action_type='proxmox.start'`).Scan(&startCount)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE action_type='proxmox.shutdown'`).Scan(&shutdownCount)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE action_type='proxmox.stop'`).Scan(&stopCount)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM asset_actions WHERE action_type='proxmox.snapshot'`).Scan(&snapshotCount)

	// Success rate: succeeded / (succeeded + failed) * 100
	successRate := 0.0
	denominator := succeeded + failed
	if denominator > 0 {
		successRate = float64(succeeded) / float64(denominator) * 100
	}

	return map[string]any{
		"by_status": map[string]any{
			"pending":    pending,
			"approved":   approvedStatus,
			"executing":  executing,
			"succeeded":  succeeded,
			"failed":     failed,
			"cancelled":  cancelled,
		},
		"by_type": map[string]any{
			"proxmox.start":    startCount,
			"proxmox.shutdown": shutdownCount,
			"proxmox.stop":     stopCount,
			"proxmox.snapshot": snapshotCount,
		},
		"success_rate_percent": successRate,
	}
}

func (h *MetricsHandler) agentMetrics(ctx context.Context) map[string]any {
	var runsPending, runsRunning, runsCompleted, runsFailed int

	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='pending'`).Scan(&runsPending)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='running'`).Scan(&runsRunning)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='completed'`).Scan(&runsCompleted)
	h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_runs WHERE status='failed'`).Scan(&runsFailed)

	// Average reasoning time: completed_at - started_at for completed runs
	var avgReasoningTime *float64
	h.pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (completed_at - started_at)))
		FROM agent_runs
		WHERE status='completed'
	`).Scan(&avgReasoningTime)

	avgReasoning := 0.0
	if avgReasoningTime != nil {
		avgReasoning = *avgReasoningTime
	}

	return map[string]any{
		"runs_pending":                runsPending,
		"runs_running":                runsRunning,
		"runs_completed":              runsCompleted,
		"runs_failed":                 runsFailed,
		"avg_reasoning_time_seconds":  avgReasoning,
	}
}

var _ = time.Now
