package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Monitor runs a background loop that detects expiring and expired approvals,
// emits audit + outbox events, and transitions expired approvals to terminal state.
type Monitor struct {
	pool              *pgxpool.Pool
	cfg               *config.Config
	thresholdPercent  int           // percentage of total window that constitutes "expiring"
	interval          time.Duration // how often to poll
}

// NewMonitor creates an approval expiry monitor from config.
func NewMonitor(pool *pgxpool.Pool, cfg *config.Config) *Monitor {
	return &Monitor{
		pool:             pool,
		cfg:              cfg,
		thresholdPercent: cfg.ApprovalExpiringThresholdPercent,
		interval:         time.Duration(cfg.ApprovalMonitorIntervalSeconds) * time.Second,
	}
}

// Start launches the monitor loop. Blocks until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	if !m.cfg.ApprovalMonitorEnabled {
		config.Info("[approval-monitor] disabled by config", nil)
		return
	}

	config.Info("[approval-monitor] started", map[string]any{
		"interval_seconds":    m.cfg.ApprovalMonitorIntervalSeconds,
		"threshold_percent": m.thresholdPercent,
	})

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run once immediately on startup
	m.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			config.Info("[approval-monitor] stopped", nil)
			return
		case <-ticker.C:
			m.tick(ctx)
		}
	}
}

// tick performs one cycle of expiry checking.
func (m *Monitor) tick(ctx context.Context) {
	now := time.Now()

	// ── 1. Mark expired approvals ──
	m.processExpired(ctx, now)

	// ── 2. Notify expiring approvals ──
	m.processExpiring(ctx, now)
}

// processExpired finds pending approvals past their expiry and transitions them.
func (m *Monitor) processExpired(ctx context.Context, now time.Time) {
	rows, err := m.pool.Query(ctx, `
		SELECT id::text, team_id::text, action_type, risk_level, expires_at, created_at
		FROM approval_requests
		WHERE status = 'pending' AND expires_at < $1
	`, now)
	if err != nil {
		config.Error("[approval-monitor] expired query error", map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, teamIDStr, actionType, riskLevel string
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&id, &teamIDStr, &actionType, &riskLevel, &expiresAt, &createdAt); err != nil {
			continue
		}
		m.expireApproval(ctx, id, teamIDStr, actionType, riskLevel, expiresAt)
	}
}

// expireApproval transitions a single approval to expired and emits events.
func (m *Monitor) expireApproval(ctx context.Context, id, teamIDStr, actionType, riskLevel string, expiresAt time.Time) {
	approvalID, err := uuid.Parse(id)
	if err != nil {
		return
	}
	teamID, _ := uuid.Parse(teamIDStr)

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE approval_requests SET status='expired', updated_at=NOW() WHERE id=$1 AND status='pending'
	`, approvalID)
	if err != nil {
		config.Error("[approval-monitor] failed to expire approval", map[string]any{"approval_id": id, "error": err.Error()})
		return
	}

	// Audit
	auditMeta, _ := json.Marshal(map[string]any{
		"approval_id": id,
		"action_type": actionType,
		"risk_level":  riskLevel,
		"expires_at":  expiresAt.Format(time.RFC3339),
	})
	audit.Write(ctx, tx, audit.Event{
		ActorType:  "system",
		Action:     "approval.expired",
		EntityType: "approval_request",
		EntityID:   approvalID,
		TeamID:     &teamID,
		Summary:    fmt.Sprintf("Approval expired: %s (%s risk)", actionType, riskLevel),
		NewValue:   auditMeta,
	})

	// Outbox — sanitized payload, no raw action_target
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.approval.expired",
		AggregateType: "approval_request",
		AggregateID:   id,
		Payload:       buildExpiryPayload(id, teamIDStr, actionType, riskLevel, expiresAt, 0, "expired"),
	})

	if err := tx.Commit(ctx); err != nil {
		config.Error("[approval-monitor] failed to commit expiry", map[string]any{"approval_id": id, "error": err.Error()})
		return
	}

	config.Info("[approval-monitor] expired approval", map[string]any{
		"approval_id": id,
		"action_type": actionType,
	})
}

// processExpiring finds pending approvals within the threshold window and notifies once.
func (m *Monitor) processExpiring(ctx context.Context, now time.Time) {
	rows, err := m.pool.Query(ctx, `
		SELECT id::text, team_id::text, action_type, risk_level, expires_at, created_at
		FROM approval_requests
		WHERE status = 'pending'
	  		AND expires_at > $1
	  		AND expiring_notified_at IS NULL
	`, now)
	if err != nil {
		config.Error("[approval-monitor] expiring query error", map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, teamIDStr, actionType, riskLevel string
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&id, &teamIDStr, &actionType, &riskLevel, &expiresAt, &createdAt); err != nil {
			continue
		}

		// Check threshold
		totalWindow := expiresAt.Sub(createdAt)
		if totalWindow <= 0 {
			continue
		}
		remaining := expiresAt.Sub(now)
		thresholdDuration := time.Duration(float64(totalWindow) * float64(m.thresholdPercent) / 100.0)

		if remaining > thresholdDuration {
			continue // not yet in expiring window
		}

		remainingSeconds := int(remaining.Seconds())
		if remainingSeconds < 0 {
			remainingSeconds = 0
		}

		m.notifyExpiring(ctx, id, teamIDStr, actionType, riskLevel, expiresAt, remainingSeconds)
	}
}

// notifyExpiring sends the one-time expiring notification.
func (m *Monitor) notifyExpiring(ctx context.Context, id, teamIDStr, actionType, riskLevel string, expiresAt time.Time, remainingSeconds int) {
	approvalID, err := uuid.Parse(id)
	if err != nil {
		return
	}
	teamID, _ := uuid.Parse(teamIDStr)

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	// Set expiring_notified_at (prevents repeat notifications)
	_, err = tx.Exec(ctx, `
		UPDATE approval_requests SET expiring_notified_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND status='pending' AND expiring_notified_at IS NULL
	`, approvalID)
	if err != nil {
		config.Error("[approval-monitor] failed to set notified_at", map[string]any{"approval_id": id, "error": err.Error()})
		return
	}

	// Verify row was actually updated (race condition guard)
	var updated bool
	tx.QueryRow(ctx, `SELECT expiring_notified_at IS NOT NULL FROM approval_requests WHERE id=$1`, approvalID).Scan(&updated)
	if !updated {
		return // someone else already notified
	}

	// Audit
	auditMeta, _ := json.Marshal(map[string]any{
		"approval_id":       id,
		"action_type":       actionType,
		"risk_level":        riskLevel,
		"expires_at":        expiresAt.Format(time.RFC3339),
		"remaining_seconds": remainingSeconds,
	})
	audit.Write(ctx, tx, audit.Event{
		ActorType:  "system",
		Action:     "approval.expiring",
		EntityType: "approval_request",
		EntityID:   approvalID,
		TeamID:     &teamID,
		Summary:    fmt.Sprintf("Approval expiring soon: %s (%s risk, %ds remaining)", actionType, riskLevel, remainingSeconds),
		NewValue:   auditMeta,
	})

	// Outbox — sanitized payload
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.approval.expiring",
		AggregateType: "approval_request",
		AggregateID:   id,
		Payload:       buildExpiryPayload(id, teamIDStr, actionType, riskLevel, expiresAt, remainingSeconds, "expiring"),
	})

	if err := tx.Commit(ctx); err != nil {
		config.Error("[approval-monitor] failed to commit expiring", map[string]any{"approval_id": id, "error": err.Error()})
		return
	}

	config.Info("[approval-monitor] notified expiring approval", map[string]any{
		"approval_id":      id,
		"action_type":      actionType,
		"remaining_seconds": remainingSeconds,
	})
}

// buildExpiryPayload constructs a sanitized JSON payload for outbox events.
// Never includes raw action_target.
func buildExpiryPayload(approvalID, teamID, actionType, riskLevel string, expiresAt time.Time, remainingSeconds int, status string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"approval_id":"%s","team_id":"%s","action_type":"%s","risk_level":"%s","expires_at":"%s","remaining_seconds":%d,"status":"%s"}`,
		approvalID, teamID, actionType, riskLevel, expiresAt.Format(time.RFC3339), remainingSeconds, status,
	))
}
