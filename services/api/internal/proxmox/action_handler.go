package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/clarityit/api/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActionHandler handles Proxmox asset mutation requests.
type ActionHandler struct {
	pool   *pgxpool.Pool
	client ProxmoxClient
	cfg    *config.Config
}

func NewActionHandler(pool *pgxpool.Pool, client ProxmoxClient, cfg *config.Config) *ActionHandler {
	return &ActionHandler{pool: pool, client: client, cfg: cfg}
}

func (h *ActionHandler) Routes() chi.Router {
	r := chi.NewRouter()
	// Allowed mutation routes — only these four exist
	r.Post("/{assetId}/actions/proxmox/start", h.CreateAction)
	r.Post("/{assetId}/actions/proxmox/shutdown", h.CreateAction)
	r.Post("/{assetId}/actions/proxmox/stop", h.CreateAction)
	r.Post("/{assetId}/actions/proxmox/snapshot", h.CreateAction)
	// List + detail + execute
	r.Get("/asset-actions", h.ListActions)
	r.Get("/asset-actions/{actionId}", h.GetAction)
	r.Post("/asset-actions/{actionId}/execute", h.ExecuteAction)
	return r
}

// allowedActions maps action paths to risk levels and requirements.
var allowedActions = map[string]struct {
	riskLevel      string
	requiresMFA    bool
	minApprovers   int
}{
	"proxmox.start":    {riskLevel: "medium", requiresMFA: false, minApprovers: 1},
	"proxmox.snapshot": {riskLevel: "medium", requiresMFA: false, minApprovers: 1},
	"proxmox.shutdown": {riskLevel: "high", requiresMFA: true, minApprovers: 1},
	"proxmox.stop":     {riskLevel: "critical", requiresMFA: true, minApprovers: 2},
}

// snapshotNameRegex allows only alphanumeric, hyphens, and underscores.
var snapshotNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,40}$`)

// ─── Create Action ───

func (h *ActionHandler) CreateAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	assetID, err := uuid.Parse(chi.URLParam(r, "assetId"))
	if err != nil {
		writeErr(w, 400, "Invalid asset ID")
		return
	}

	// Extract action type from URL path
	actionType := "proxmox." + chi.URLParam(r, "*")
	// The wildcard captures the last segment
	pathSegments := strings.Split(r.URL.Path, "/")
	lastSeg := pathSegments[len(pathSegments)-1]
	actionType = "proxmox." + lastSeg

	// Guardrail 8: Unknown or forbidden action routes must return 404, not a soft allow
	req, ok := allowedActions[actionType]
	if !ok {
		writeErr(w, 404, "Action not found")
		return
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	// Parse optional snapshot name
	var body struct {
		SnapshotName string `json:"snapshot_name"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Validate snapshot name if applicable
	if actionType == "proxmox.snapshot" {
		if body.SnapshotName == "" {
			// Auto-generate a safe name
			body.SnapshotName = fmt.Sprintf("clar-%d", time.Now().Unix())
		}
		if !snapshotNameRegex.MatchString(body.SnapshotName) {
			writeErr(w, 400, "Invalid snapshot name: only alphanumeric, hyphens, and underscores allowed (max 40 chars)")
			return
		}
	}

	// Guardrail 1: Asset must belong to the same team
	var provider, externalID, assetType, hostname string
	err = h.pool.QueryRow(ctx, `
		SELECT a.provider, a.external_id, a.asset_type, a.hostname
		FROM assets a
		JOIN objects o ON a.object_id = o.id
		WHERE a.object_id = $1 AND o.team_id = $2 AND o.deleted_at IS NULL
	`, assetID, teamID).Scan(&provider, &externalID, &assetType, &hostname)
	if err != nil {
		writeErr(w, 404, "Asset not found in team")
		return
	}

	// Guardrail 2: Asset must be provider=proxmox
	if provider != "proxmox" {
		writeErr(w, 400, "Asset is not a Proxmox-managed resource")
		return
	}

	// Guardrail 3: Asset must include node + vmid + vm_type
	target, err := parseMutationTarget(externalID, assetType)
	if err != nil {
		writeErr(w, 400, fmt.Sprintf("Asset missing required metadata: %v", err))
		return
	}

	// Create asset_action + approval_request in a single transaction
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	actionID := uuid.New()

	// Create approval_request
	approvalID := uuid.New()
	targetJSON, _ := json.Marshal(map[string]any{
		"vmid":   fmt.Sprintf("%d", target.VMID),
		"node":   target.Node,
		"vmtype": target.VMType,
	})

	_, err = tx.Exec(ctx, `
		INSERT INTO approval_requests (id, team_id, action_type, action_target, risk_level, description,
		                                requested_by, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', NOW() + interval '1 hour')
	`, approvalID, teamID, actionType, targetJSON, req.riskLevel,
		fmt.Sprintf("%s on %s", actionType, hostname), userID)
	if err != nil {
		writeErr(w, 500, "Failed to create approval request")
		return
	}

	// Create asset_action
	_, err = tx.Exec(ctx, `
		INSERT INTO asset_actions (id, team_id, asset_id, action_type, status, approval_id, requested_by, snapshot_name)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6, $7)
	`, actionID, teamID, assetID, actionType, approvalID, userID, body.SnapshotName)
	if err != nil {
		writeErr(w, 500, "Failed to create asset action")
		return
	}

	// Audit
	actionMeta, _ := json.Marshal(map[string]any{
		"action_type": actionType,
		"asset_id":    assetID.String(),
		"risk_level":  req.riskLevel,
		"hostname":    hostname,
	})
	_ = audit.Write(ctx, tx, audit.Event{
		TeamID:     &teamID,
		ActorID:    userID,
		Action:     "asset.action.requested",
		EntityType: "asset_action",
		EntityID:   actionID,
		NewValue:   actionMeta,
	})

	// Outbox — no raw secrets/target payloads beyond identity
	_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.asset.action.requested",
		AggregateType: "asset_action",
		AggregateID:   actionID.String(),
		Payload:       actionMeta,
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 201, map[string]any{
		"id":           actionID,
		"approval_id":  approvalID,
		"action_type":  actionType,
		"asset_id":     assetID,
		"status":       "pending",
		"risk_level":   req.riskLevel,
		"requires_mfa": req.requiresMFA,
		"hostname":     hostname,
	})
}

// ─── List Actions ───

func (h *ActionHandler) ListActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	statusFilter := r.URL.Query().Get("status")

	query := `SELECT id::text, asset_id::text, action_type, status, approval_id::text,
	                 proxmox_task_id, error_message, snapshot_name,
	                 created_at::text, executed_at::text, completed_at::text
	          FROM asset_actions WHERE team_id=$1`
	args := []any{teamID}
	if statusFilter != "" {
		query += " AND status=$2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, assetID, actionType, status string
		var approvalID, taskID, errMsg, snapName *string
		var createdAt, executedAt, completedAt *string
		rows.Scan(&id, &assetID, &actionType, &status, &approvalID, &taskID, &errMsg,
			&snapName, &createdAt, &executedAt, &completedAt)
		items = append(items, map[string]any{
			"id":              id,
			"asset_id":        assetID,
			"action_type":     actionType,
			"status":          status,
			"approval_id":     approvalID,
			"proxmox_task_id": taskID,
			"error_message":   errMsg,
			"snapshot_name":   snapName,
			"created_at":      createdAt,
			"executed_at":     executedAt,
			"completed_at":    completedAt,
		})
	}

	writeJSON(w, 200, items)
}

// ─── Get Action ───

func (h *ActionHandler) GetAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		writeErr(w, 400, "Invalid action ID")
		return
	}

	var id, assetID, actionType, status string
	var approvalID, taskID, errMsg, snapName *string
	var result json.RawMessage
	var createdAt, executedAt, completedAt *string

	err = h.pool.QueryRow(ctx, `
		SELECT id::text, asset_id::text, action_type, status, approval_id::text,
		       proxmox_task_id, result::text, error_message, snapshot_name,
		       created_at::text, executed_at::text, completed_at::text
		FROM asset_actions WHERE id=$1 AND team_id=$2
	`, actionID, teamID).Scan(&id, &assetID, &actionType, &status, &approvalID,
		&taskID, &result, &errMsg, &snapName, &createdAt, &executedAt, &completedAt)
	if err != nil {
		writeErr(w, 404, "Asset action not found")
		return
	}

	writeJSON(w, 200, map[string]any{
		"id":              id,
		"asset_id":        assetID,
		"action_type":     actionType,
		"status":          status,
		"approval_id":     approvalID,
		"proxmox_task_id": taskID,
		"result":          result,
		"error_message":   errMsg,
		"snapshot_name":   snapName,
		"created_at":      createdAt,
		"executed_at":     executedAt,
		"completed_at":    completedAt,
	})
}

// ─── Execute Action ───

func (h *ActionHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		writeErr(w, 400, "Invalid action ID")
		return
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	// Idempotency required
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		writeErr(w, 400, "Idempotency-Key header required")
		return
	}

	// Guardrail 7: Mutation disabled by default unless explicitly enabled
	if !h.cfg.ProxmoxMutationEnabled {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "mutation_disabled")
		writeErr(w, 403, "Proxmox mutation is not enabled")
		return
	}

	// Load asset_action
	var actionType, status, assetIDStr string
	var approvalID *uuid.UUID
	var snapshotName *string
	var assetID uuid.UUID
	err = h.pool.QueryRow(ctx, `
		SELECT action_type, status, asset_id, approval_id, snapshot_name
		FROM asset_actions WHERE id=$1 AND team_id=$2
	`, actionID, teamID).Scan(&actionType, &status, &assetID, &approvalID, &snapshotName)
	if err != nil {
		writeErr(w, 404, "Asset action not found")
		return
	}

	// Already executed? Idempotent response.
	if status == "succeeded" || status == "executing" {
		writeJSON(w, 200, map[string]any{
			"id":           actionID,
			"status":       status,
			"message":      "already executed",
		})
		return
	}

	// Check approval exists and is approved
	if approvalID == nil {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "no_approval_linked")
		writeErr(w, 403, "No approval linked to this action")
		return
	}

	// Verify approval via the same guardrails as Track 3
	var approvalStatus, approvalActionType string
	var approvalTarget json.RawMessage
	var approvalTeamID uuid.UUID
	var approvalExpiresAt time.Time
	var approvalExecutedAt *time.Time

	err = h.pool.QueryRow(ctx, `
		SELECT status, action_type, action_target, team_id, expires_at, executed_at
		FROM approval_requests WHERE id=$1
	`, *approvalID).Scan(&approvalStatus, &approvalActionType, &approvalTarget,
		&approvalTeamID, &approvalExpiresAt, &approvalExecutedAt)
	if err != nil {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "approval_not_found")
		writeErr(w, 403, "Approval not found")
		return
	}

	// Guardrail 10: Cross-team approval/action/asset mismatch
	if approvalTeamID != teamID {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "approval_wrong_team")
		writeErr(w, 403, "Approval belongs to a different team")
		return
	}

	// Guardrail 17: Approval action_type mismatch
	if approvalActionType != actionType {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "approval_action_type_mismatch")
		writeErr(w, 403, "Approval action_type does not match")
		return
	}

	// Approval already executed?
	if approvalExecutedAt != nil || approvalStatus == "executed" {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "approval_already_executed")
		writeErr(w, 403, "Approval already used")
		return
	}

	// Approval status must be approved
	if approvalStatus != "approved" {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "approval_not_approved")
		writeErr(w, 403, "Approval is not approved")
		return
	}

	// Expired?
	if time.Now().After(approvalExpiresAt) {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "approval_expired")
		writeErr(w, 403, "Approval has expired")
		return
	}

	// Action type risk requirements
	req, ok := allowedActions[actionType]
	if !ok {
		writeErr(w, 400, "Unknown action type")
		return
	}

	// MFA requirement
	if req.requiresMFA {
		if !h.hasRecentMFA(ctx, userID) {
			h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "mfa_required")
			writeErr(w, 403, "Recent MFA verification required")
			return
		}
	}

	// Stop requires 2 distinct approvers
	if req.minApprovers > 1 {
		var approvalCount int
		h.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM approval_decisions WHERE approval_id=$1 AND decision='approved'`,
			*approvalID).Scan(&approvalCount)
		if approvalCount < req.minApprovers {
			h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "insufficient_approvers")
			writeErr(w, 403, fmt.Sprintf("This action requires %d distinct approvers (have %d)", req.minApprovers, approvalCount))
			return
		}
	}

	// Load asset target
	var provider, externalID, assetType, hostname string
	err = h.pool.QueryRow(ctx, `
		SELECT a.provider, a.external_id, a.asset_type, a.hostname
		FROM assets a JOIN objects o ON a.object_id=o.id
		WHERE a.object_id=$1 AND o.team_id=$2
	`, assetID, teamID).Scan(&provider, &externalID, &assetType, &hostname)
	if err != nil {
		writeErr(w, 404, "Asset not found")
		return
	}

	target, err := parseMutationTarget(externalID, assetType)
	if err != nil {
		h.recordActionBlock(ctx, teamIDStr, teamID, userID, actionID, "invalid_asset_metadata")
		writeErr(w, 400, fmt.Sprintf("Asset metadata invalid: %v", err))
		return
	}

	// Execute mutation through Proxmox client
	var taskUPID string
	var execErr error

	switch actionType {
	case "proxmox.start":
		taskUPID, execErr = h.client.StartVM(ctx, target)
	case "proxmox.shutdown":
		taskUPID, execErr = h.client.ShutdownVM(ctx, target)
	case "proxmox.stop":
		taskUPID, execErr = h.client.StopVM(ctx, target)
	case "proxmox.snapshot":
		snapName := "clar-snap"
		if snapshotName != nil {
			snapName = *snapshotName
		}
		taskUPID, execErr = h.client.SnapshotVM(ctx, target, snapName)
	}

	if execErr != nil {
		// Record failure
		h.recordActionFailure(ctx, teamIDStr, teamID, userID, actionID, actionType, *approvalID, execErr.Error())
		writeErr(w, 502, fmt.Sprintf("Proxmox mutation failed: %s", sanitizeForLog(execErr.Error())))
		return
	}

	// Success — persist result in transaction with approval marking
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Mark asset_action
		tx.Exec(ctx, `
			UPDATE asset_actions
			SET status='succeeded', proxmox_task_id=$1, executed_at=NOW(), completed_at=NOW(), updated_at=NOW()
			WHERE id=$2
		`, taskUPID, actionID)

		// Guardrail 12: Mark approval executed in same tx
		tx.Exec(ctx, `
			UPDATE approval_requests SET status='executed', executed_at=NOW(), updated_at=NOW()
			WHERE id=$1 AND status='approved'
		`, *approvalID)

		// Audit — sanitized (no secrets, no raw target beyond identity)
		auditPayload, _ := json.Marshal(map[string]any{
			"action_type":   actionType,
			"task_id":       taskUPID,
			"hostname":      hostname,
			"asset_id":      assetIDStr,
			"approval_id":   approvalID.String(),
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID:     &teamID,
			ActorID:    userID,
			Action:     "asset.action.executed",
			EntityType: "asset_action",
			EntityID:   actionID,
			NewValue:   auditPayload,
		})

		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.asset.action.executed",
			AggregateType: "asset_action",
			AggregateID:   actionID.String(),
			Payload:       auditPayload,
		})

		return nil
	})
	if err != nil {
		writeErr(w, 500, "Failed to persist result")
		return
	}

	writeJSON(w, 200, map[string]any{
		"id":              actionID,
		"status":          "succeeded",
		"action_type":     actionType,
		"proxmox_task_id": taskUPID,
		"hostname":        hostname,
	})
}

// ─── Helpers ───

func (h *ActionHandler) hasRecentMFA(ctx context.Context, userID uuid.UUID) bool {
	var recentMFA *time.Time
	h.pool.QueryRow(ctx,
		`SELECT MAX(recent_mfa_at) FROM user_sessions
		 WHERE user_id=$1 AND revoked_at IS NULL`,
		userID).Scan(&recentMFA)
	if recentMFA == nil {
		return false
	}
	return time.Since(*recentMFA) < 5*time.Minute
}

func (h *ActionHandler) recordActionBlock(ctx context.Context, teamIDStr string, tid, userID, actionID uuid.UUID, reason string) {
	payload, _ := json.Marshal(map[string]any{
		"action_id": actionID.String(),
		"reason":    reason,
	})
	_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tx.Exec(ctx, `UPDATE asset_actions SET status='failed', error_message=$1, updated_at=NOW() WHERE id=$2`,
			reason, actionID)
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &tid, ActorID: userID, Action: "asset.action.blocked",
			EntityType: "asset_action", EntityID: actionID, NewValue: payload,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType: "clarity.v1.asset.action.blocked", AggregateType: "asset_action",
			AggregateID: actionID.String(), Payload: payload,
		})
		return nil
	})
}

func (h *ActionHandler) recordActionFailure(ctx context.Context, teamIDStr string, tid, userID, actionID uuid.UUID, actionType string, approvalID uuid.UUID, errMsg string) {
	sanitizedErr := sanitizeForLog(errMsg)
	payload, _ := json.Marshal(map[string]any{
		"action_type": actionType,
		"error":       sanitizedErr,
	})
	_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tx.Exec(ctx, `UPDATE asset_actions SET status='failed', error_message=$1, completed_at=NOW(), updated_at=NOW() WHERE id=$2`,
			sanitizedErr, actionID)
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &tid, ActorID: userID, Action: "asset.action.failed",
			EntityType: "asset_action", EntityID: actionID, NewValue: payload,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType: "clarity.v1.asset.action.failed", AggregateType: "asset_action",
			AggregateID: actionID.String(), Payload: payload,
		})
		return nil
	})
}

// parseMutationTarget extracts node, vmid, and vm_type from asset external_id.
// Format: pve:{node}:{vmid}
func parseMutationTarget(externalID, assetType string) (MutationTarget, error) {
	parts := strings.SplitN(externalID, ":", 3)
	if len(parts) < 3 {
		return MutationTarget{}, fmt.Errorf("external_id must be pve:node:vmid format")
	}
	node := parts[1]
	var vmid int
	fmt.Sscanf(parts[2], "%d", &vmid)
	if vmid == 0 {
		return MutationTarget{}, fmt.Errorf("invalid vmid in external_id")
	}
	vmType := "qemu"
	if strings.Contains(strings.ToLower(assetType), "lxc") || strings.Contains(strings.ToLower(assetType), "container") {
		vmType = "lxc"
	}
	return MutationTarget{Node: node, VMID: vmid, VMType: vmType}, nil
}

// sanitizeForLog removes any potential secrets from error messages.
func sanitizeForLog(s string) string {
	// Remove anything that looks like a token/secret
	s = regexp.MustCompile(`token=[^\s&]+`).ReplaceAllString(s, "token=[REDACTED]")
	s = regexp.MustCompile(`secret=[^\s&]+`).ReplaceAllString(s, "secret=[REDACTED]")
	return s
}
