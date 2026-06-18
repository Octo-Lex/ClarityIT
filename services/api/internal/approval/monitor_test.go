package approval

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// monitorTestSetup creates a pool, cleans up, and returns the team ID.
func monitorTestSetup(t *testing.T) (*pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool, _ := pgxpool.New(t.Context(), testDBURL)
	t.Cleanup(func() { pool.Close() })

	ctx := t.Context()
	// Clean up
	pool.Exec(ctx, "DELETE FROM approval_decisions WHERE approval_id IN (SELECT id FROM approval_requests)")
	pool.Exec(ctx, "DELETE FROM approval_requests")
	pool.Exec(ctx, "DELETE FROM audit_logs WHERE entity_type='approval_request'")
	pool.Exec(ctx, "DELETE FROM outbox_events WHERE aggregate_type='approval_request'")

	var teamIDStr string
	pool.QueryRow(ctx, "SELECT id::text FROM teams LIMIT 1").Scan(&teamIDStr)
	teamID, _ := uuid.Parse(teamIDStr)

	var userIDStr string
	pool.QueryRow(ctx, "SELECT id::text FROM users LIMIT 1").Scan(&userIDStr)

	return pool, teamID
}

func createMonitorApproval(t *testing.T, pool *pgxpool.Pool, teamID uuid.UUID, riskLevel string, createdAt, expiresAt time.Time) uuid.UUID {
	t.Helper()
	ctx := t.Context()
	approvalID := uuid.New()

	var userIDStr string
	pool.QueryRow(ctx, "SELECT id::text FROM users LIMIT 1").Scan(&userIDStr)

	_, err := pool.Exec(ctx, `
		INSERT INTO approval_requests (id, team_id, action_type, action_target, risk_level, description,
		                                requested_by, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, 'proxmox.test', '{}', $3, 'test approval',
		        $4, 'pending', $5, $6, $6)
	`, approvalID, teamID, riskLevel, userIDStr, expiresAt, createdAt)
	if err != nil {
		t.Fatalf("failed to create test approval: %v", err)
	}
	return approvalID
}

// Test 1: pending approval approaching expiry emits approval.expiring once.
func TestMonitor_ExpiringApprovalNotifiesOnce(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	// 80-second total window, 70s elapsed → ~10s remaining → within 25% threshold (20s)
	createdAt := time.Now().Add(-70 * time.Second)
	expiresAt := createdAt.Add(80 * time.Second)

	approvalID := createMonitorApproval(t, pool, teamID, "medium", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           true,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)

	// First tick should notify
	monitor.tick(ctx)

	var notifiedAt *time.Time
	pool.QueryRow(ctx, `SELECT expiring_notified_at FROM approval_requests WHERE id=$1`, approvalID).Scan(&notifiedAt)
	if notifiedAt == nil {
		t.Fatal("expected expiring_notified_at to be set after first tick")
	}

	// Second tick should NOT re-notify
	time.Sleep(10 * time.Millisecond)
	monitor.tick(ctx)

	var notifiedAt2 *time.Time
	pool.QueryRow(ctx, `SELECT expiring_notified_at FROM approval_requests WHERE id=$1`, approvalID).Scan(&notifiedAt2)
	if notifiedAt2 == nil || !notifiedAt2.Equal(*notifiedAt) {
		t.Error("expiring_notified_at was modified on second tick — notification should not repeat")
	}

	// Verify exactly 1 audit event
	var auditCount int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE entity_id=$1 AND action='approval.expiring'`, approvalID).Scan(&auditCount)
	if auditCount != 1 {
		t.Errorf("expected exactly 1 expiring audit event, got %d", auditCount)
	}
}

// Test 2: expired pending approval is marked expired and writes audit + outbox.
func TestMonitor_ExpiredApprovalMarkedExpired(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	createdAt := time.Now().Add(-2 * time.Hour)
	expiresAt := createdAt.Add(1 * time.Hour) // expired 1 hour ago

	approvalID := createMonitorApproval(t, pool, teamID, "high", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           true,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)
	monitor.tick(ctx)

	var status string
	pool.QueryRow(ctx, `SELECT status FROM approval_requests WHERE id=$1`, approvalID).Scan(&status)
	if status != "expired" {
		t.Errorf("expected status 'expired', got '%s'", status)
	}

	var auditCount, outboxCount int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE entity_id=$1 AND action='approval.expired'`, approvalID).Scan(&auditCount)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE aggregate_id=$1 AND event_type='clarity.v1.approval.expired'`, approvalID).Scan(&outboxCount)
	if auditCount != 1 {
		t.Errorf("expected 1 expired audit event, got %d", auditCount)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 expired outbox event, got %d", outboxCount)
	}
}

// Test 3: approval well within window is not notified as expiring.
func TestMonitor_NotExpiringSkipped(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	createdAt := time.Now()
	expiresAt := createdAt.Add(1 * time.Hour) // 100% remaining

	approvalID := createMonitorApproval(t, pool, teamID, "medium", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           true,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)
	monitor.tick(ctx)

	var notifiedAt *time.Time
	pool.QueryRow(ctx, `SELECT expiring_notified_at FROM approval_requests WHERE id=$1`, approvalID).Scan(&notifiedAt)
	if notifiedAt != nil {
		t.Error("expiring_notified_at should not be set for approval with plenty of time left")
	}
}

// Test 4: expired approval cannot be approved (terminal state).
func TestMonitor_ExpiredApprovalCannotBeApproved(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	createdAt := time.Now().Add(-2 * time.Hour)
	expiresAt := createdAt.Add(1 * time.Hour)
	approvalID := createMonitorApproval(t, pool, teamID, "medium", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           true,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)
	monitor.tick(ctx)

	var status string
	pool.QueryRow(ctx, `SELECT status FROM approval_requests WHERE id=$1`, approvalID).Scan(&status)
	if status != "expired" {
		t.Fatalf("expected expired, got %s", status)
	}
	if !IsTerminalState(status) {
		t.Error("expired should be a terminal state")
	}
}

// Test 5: payload redaction — no raw action_target in outbox.
func TestMonitor_PayloadRedaction(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	createdAt := time.Now().Add(-2 * time.Hour)
	expiresAt := createdAt.Add(1 * time.Hour)
	approvalID := createMonitorApproval(t, pool, teamID, "critical", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           true,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)
	monitor.tick(ctx)

	var payload []byte
	pool.QueryRow(ctx,
		`SELECT payload FROM outbox_events WHERE aggregate_id=$1 AND event_type='clarity.v1.approval.expired'`,
		approvalID).Scan(&payload)
	if payload == nil {
		t.Fatal("expected outbox event payload")
	}

	var p map[string]any
	json.Unmarshal(payload, &p)

	// Verify no action_target key in payload
	if _, hasTarget := p["action_target"]; hasTarget {
		t.Error("payload should not contain action_target")
	}

	// Verify required sanitized fields exist
	requiredKeys := []string{"approval_id", "team_id", "risk_level", "action_type", "expires_at", "remaining_seconds", "status"}
	for _, key := range requiredKeys {
		if _, ok := p[key]; !ok {
			t.Errorf("payload missing required key: %s", key)
		}
	}
}

// Test 6: monitor disabled by config does nothing.
func TestMonitor_DisabledByConfig(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	createdAt := time.Now().Add(-2 * time.Hour)
	expiresAt := createdAt.Add(1 * time.Hour)
	approvalID := createMonitorApproval(t, pool, teamID, "medium", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           false,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)

	// Start should return immediately when disabled
	done := make(chan bool)
	go func() {
		monitor.Start(ctx)
		done <- true
	}()

	select {
	case <-done:
		// Good
	case <-time.After(1 * time.Second):
		t.Fatal("monitor.Start should return immediately when disabled")
	}

	var status string
	pool.QueryRow(ctx, `SELECT status FROM approval_requests WHERE id=$1`, approvalID).Scan(&status)
	if status != "pending" {
		t.Errorf("expected pending (monitor disabled), got '%s'", status)
	}
}

// Test 7: expiring approval writes both audit and outbox.
func TestMonitor_ExpiringWritesAuditAndOutbox(t *testing.T) {
	pool, teamID := monitorTestSetup(t)
	ctx := t.Context()
	defer pool.Exec(ctx, "DELETE FROM approval_requests")

	// 80s window, 70s elapsed → ~10s remaining → within 25% (20s)
	createdAt := time.Now().Add(-70 * time.Second)
	expiresAt := createdAt.Add(80 * time.Second)
	approvalID := createMonitorApproval(t, pool, teamID, "medium", createdAt, expiresAt)

	cfg := &config.Config{
		ApprovalMonitorEnabled:           true,
		ApprovalMonitorIntervalSeconds:   5,
		ApprovalExpiringThresholdPercent: 25,
	}
	monitor := NewMonitor(pool, cfg)
	monitor.tick(ctx)

	var auditCount, outboxCount int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE entity_id=$1 AND action='approval.expiring'`, approvalID).Scan(&auditCount)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE aggregate_id=$1 AND event_type='clarity.v1.approval.expiring'`, approvalID).Scan(&outboxCount)
	if auditCount != 1 {
		t.Errorf("expected 1 expiring audit event, got %d", auditCount)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 expiring outbox event, got %d", outboxCount)
	}
}

// Test 8: buildExpiryPayload produces correct sanitized output.
func TestBuildExpiryPayload(t *testing.T) {
	teamID := uuid.New()
	approvalID := uuid.New()
	expiresAt := time.Now().Add(10 * time.Second)

	payload := buildExpiryPayload(approvalID.String(), teamID.String(), "proxmox.start", "medium", expiresAt, 10, "expiring")

	var p map[string]any
	json.Unmarshal(payload, &p)

	if p["approval_id"] != approvalID.String() {
		t.Errorf("approval_id mismatch")
	}
	if p["team_id"] != teamID.String() {
		t.Errorf("team_id mismatch")
	}
	if p["action_type"] != "proxmox.start" {
		t.Errorf("action_type mismatch")
	}
	if p["risk_level"] != "medium" {
		t.Errorf("risk_level mismatch")
	}
	if p["remaining_seconds"] != float64(10) {
		t.Errorf("remaining_seconds mismatch")
	}
	if p["status"] != "expiring" {
		t.Errorf("status mismatch")
	}
	// Verify no action_target key
	if _, ok := p["action_target"]; ok {
		t.Error("payload should not contain action_target")
	}
}
