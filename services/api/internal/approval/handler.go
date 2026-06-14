package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewHandler(pool *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{pool: pool, cfg: cfg}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{approvalId}", h.Get)
	r.With(requireIdempotency).
		Post("/{approvalId}/approve", h.Approve)
	r.With(requireIdempotency).
		Post("/{approvalId}/reject", h.Reject)
	r.With(requireIdempotency).
		Post("/{approvalId}/cancel", h.Cancel)
	return r
}

// ─── Create Approval Request ───
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	userIDStr := ctx.Value("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	var req struct {
		ActionType  string          `json:"action_type"`
		ActionTarget json.RawMessage `json:"action_target"`
		RiskLevel   string          `json:"risk_level"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid request body")
		return
	}
	if req.ActionType == "" {
		writeErr(w, 400, "action_type is required")
		return
	}
	if err := ValidateRiskLevel(req.RiskLevel); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if req.ActionTarget == nil {
		req.ActionTarget = json.RawMessage(`{}`)
	}

	// Resolve policy
	policy, err := ResolvePolicy(ctx, h.pool, teamID, req.RiskLevel)
	if err != nil {
		writeErr(w, 500, "Failed to resolve policy")
		return
	}

	approvalID := uuid.New()
	expiresAt := policy.ExpiresAt()

	// Sanitize action_target for storage — strip known sensitive keys
	sanitizedTarget := sanitizeActionTarget(req.ActionTarget)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	// Use NULL for policy_id when no DB policy exists (builtin fallback returns zero UUID)
	var policyIDArg any
	if policy.ID != (uuid.UUID{}) {
		policyIDArg = policy.ID
	} else {
		policyIDArg = nil
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO approval_requests (id, team_id, action_type, action_target, risk_level, description,
		                                requested_by, status, policy_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
	`, approvalID, teamID, req.ActionType, sanitizedTarget, req.RiskLevel, req.Description,
		userID, policyIDArg, expiresAt)
	if err != nil {
		writeErr(w, 500, "Failed to create approval request")
		return
	}

	// Auto-approve if policy allows (low-risk auto-approve)
	if policy.AutoApprove && policy.AllowSelfApprove {
		// Self-approve
		_, err = tx.Exec(ctx, `
			INSERT INTO approval_decisions (id, approval_id, decided_by, decision, reason, mfa_verified)
			VALUES ($1, $2, $3, 'approved', 'auto-approved by policy', false)
		`, uuid.New(), approvalID, userID)
		if err != nil {
			writeErr(w, 500, "Failed to auto-approve")
			return
		}
		_, err = tx.Exec(ctx, `UPDATE approval_requests SET status='approved', updated_at=NOW() WHERE id=$1`, approvalID)
		if err != nil {
			writeErr(w, 500, "Failed to update status")
			return
		}
	}

	// Audit — redacted payload
	auditMeta, _ := json.Marshal(map[string]string{
		"action_type": req.ActionType,
		"risk_level":  req.RiskLevel,
		"team_id":     teamIDStr,
	})
	audit.Write(ctx, tx, audit.Event{
		ActorID:    userID,
		ActorType:  "user",
		Action:     "approval.request.created",
		EntityType: "approval_request",
		EntityID:   approvalID,
		TeamID:     &teamID,
		Summary:    fmt.Sprintf("Approval request: %s (%s risk)", req.ActionType, req.RiskLevel),
		NewValue:   auditMeta,
	})

	// Outbox — no raw action_target
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.approval.request.created",
		AggregateType: "approval_request",
		AggregateID:   approvalID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"action_type":"%s","risk_level":"%s","team_id":"%s"}`, req.ActionType, req.RiskLevel, teamIDStr)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	// Return response
	status := "pending"
	if policy.AutoApprove && policy.AllowSelfApprove {
		status = "approved"
	}

	writeJSON(w, 201, map[string]any{
		"id":          approvalID,
		"team_id":     teamID,
		"action_type": req.ActionType,
		"risk_level":  req.RiskLevel,
		"description": req.Description,
		"status":      status,
		"expires_at":  expiresAt,
		"requested_by": userID,
	})
}

// ─── List Approvals ───
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	statusFilter := r.URL.Query().Get("status")

	query := `SELECT id::text, action_type, risk_level, description, status, requested_by::text,
	                  expires_at, created_at, updated_at,
	                  CASE WHEN status='pending' AND expires_at > NOW()
	                       THEN EXTRACT(EPOCH FROM (expires_at - NOW()))::int
	                       ELSE 0 END as remaining_seconds,
	                  (expiring_notified_at IS NOT NULL AND status='pending') as is_expiring
	           FROM approval_requests WHERE team_id=$1`
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
		var id, actionType, riskLevel, description, status, requestedBy string
		var expiresAt, createdAt, updatedAt time.Time
		var remainingSeconds int
		var isExpiring bool
		rows.Scan(&id, &actionType, &riskLevel, &description, &status, &requestedBy, &expiresAt, &createdAt, &updatedAt, &remainingSeconds, &isExpiring)
		items = append(items, map[string]any{
			"id":               id,
			"action_type":      actionType,
			"risk_level":       riskLevel,
			"description":      description,
			"status":           status,
			"requested_by":     requestedBy,
			"expires_at":       expiresAt,
			"created_at":       createdAt,
			"updated_at":       updatedAt,
			"remaining_seconds": remainingSeconds,
			"is_expiring":      isExpiring,
		})
	}

	writeJSON(w, 200, items)
}

// ─── Get Approval Detail ───
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	approvalID, err := uuid.Parse(chi.URLParam(r, "approvalId"))
	if err != nil {
		writeErr(w, 400, "Invalid approval ID")
		return
	}

	var id, actionType, riskLevel, description, status, requestedBy string
	var actionTarget json.RawMessage
	var policyID *uuid.UUID
	var expiresAt, executedAt, createdAt, updatedAt time.Time

	err = h.pool.QueryRow(ctx, `
		SELECT id::text, action_type, action_target, risk_level, description, status,
		       requested_by::text, policy_id, expires_at, executed_at, created_at, updated_at
		FROM approval_requests WHERE id=$1 AND team_id=$2
	`, approvalID, teamID).Scan(&id, &actionType, &actionTarget, &riskLevel, &description, &status,
		&requestedBy, &policyID, &expiresAt, &executedAt, &createdAt, &updatedAt)
	if err != nil {
		writeErr(w, 404, "Approval request not found")
		return
	}

	// Get decisions
	rows, _ := h.pool.Query(ctx, `
		SELECT id::text, decided_by::text, decision, reason, mfa_verified, created_at
		FROM approval_decisions WHERE approval_id=$1 ORDER BY created_at
	`, approvalID)
	defer rows.Close()

	decisions := []map[string]any{}
	for rows.Next() {
		var dID, decidedBy, decision, reason string
		var mfaVerified bool
		var dCreatedAt time.Time
		rows.Scan(&dID, &decidedBy, &decision, &reason, &mfaVerified, &dCreatedAt)
		decisions = append(decisions, map[string]any{
			"id":           dID,
			"decided_by":   decidedBy,
			"decision":     decision,
			"reason":       reason,
			"mfa_verified": mfaVerified,
			"created_at":   dCreatedAt,
		})
	}

	writeJSON(w, 200, map[string]any{
		"id":           id,
		"action_type":  actionType,
		"action_target": actionTarget,
		"risk_level":   riskLevel,
		"description":  description,
		"status":       status,
		"requested_by": requestedBy,
		"policy_id":    policyID,
		"expires_at":   expiresAt,
		"executed_at":  executedAt,
		"created_at":   createdAt,
		"updated_at":   updatedAt,
		"decisions":    decisions,
	})
}

// ─── Approve ───
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "approved")
}

// ─── Reject ───
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, "rejected")
}

// decide handles both approve and reject with shared logic.
func (h *Handler) decide(w http.ResponseWriter, r *http.Request, decision string) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	approvalID, err := uuid.Parse(chi.URLParam(r, "approvalId"))
	if err != nil {
		writeErr(w, 400, "Invalid approval ID")
		return
	}

	userIDStr := ctx.Value("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Load approval request
	var status, riskLevel, requestedBy string
	var policyID *uuid.UUID
	var expiresAt time.Time
	err = h.pool.QueryRow(ctx, `
		SELECT status, risk_level, requested_by::text, policy_id, expires_at
		FROM approval_requests WHERE id=$1 AND team_id=$2
	`, approvalID, teamID).Scan(&status, &riskLevel, &requestedBy, &policyID, &expiresAt)
	if err != nil {
		writeErr(w, 404, "Approval request not found")
		return
	}

	// Terminal state check
	if IsTerminalState(status) {
		writeErr(w, 409, fmt.Sprintf("Approval is in terminal state: %s", status))
		return
	}

	// Expired check
	if time.Now().After(expiresAt) {
		h.pool.Exec(ctx, `UPDATE approval_requests SET status='expired', updated_at=NOW() WHERE id=$1`, approvalID)
		writeErr(w, 409, "Approval request has expired")
		return
	}

	// Duplicate decision check
	alreadyDecided, err := HasUserDecided(ctx, h.pool, approvalID, userID)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	if alreadyDecided {
		writeErr(w, 409, "You have already made a decision on this request")
		return
	}

	// Self-approval check
	requestedUUID, _ := uuid.Parse(requestedBy)
	isSelfApproval := userID == requestedUUID

	// Resolve policy
	policy, _ := ResolvePolicy(ctx, h.pool, teamID, riskLevel)
	if policy == nil {
		policy = builtinPolicy(teamID, riskLevel)
	}

	// Self-approval blocked by default
	if isSelfApproval && !policy.AllowSelfApprove {
		writeErr(w, 403, "Self-approval is not allowed for this risk level")
		return
	}

	// MFA required check for high/critical
	mfaVerified := false
	if policy.RequiresMFA {
		mfaVerified = hasRecentMFA(ctx, h.pool, userID)
		if !mfaVerified {
			writeErr(w, 403, "Recent MFA verification required to approve this request")
			return
		}
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	// Write immutable decision
	decisionID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO approval_decisions (id, approval_id, decided_by, decision, reason, mfa_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, decisionID, approvalID, userID, decision, req.Reason, mfaVerified)
	if err != nil {
		writeErr(w, 500, "Failed to record decision")
		return
	}

	// Update request status based on decision
	if decision == "rejected" {
		tx.Exec(ctx, `UPDATE approval_requests SET status='rejected', updated_at=NOW() WHERE id=$1`, approvalID)
	} else if decision == "approved" {
		// Count total approvals
		approvalCount, _ := CountApprovals(ctx, h.pool, approvalID) // includes this one (not yet committed, so +1)
		approvalCount++ // Account for the one we just wrote

		if approvalCount >= policy.MinApprovers {
			tx.Exec(ctx, `UPDATE approval_requests SET status='approved', updated_at=NOW() WHERE id=$1`, approvalID)
		}
		// If min_approvers not met yet, status stays pending
	}

	// Audit — no raw target data
	auditMeta, _ := json.Marshal(map[string]string{
		"approval_id": approvalID.String(),
		"decision":    decision,
		"risk_level":  riskLevel,
		"mfa_verified": fmt.Sprintf("%v", mfaVerified),
	})
	audit.Write(ctx, tx, audit.Event{
		ActorID:    userID,
		ActorType:  "user",
		Action:     fmt.Sprintf("approval.%s", decision),
		EntityType: "approval_request",
		EntityID:   approvalID,
		TeamID:     &teamID,
		Summary:    fmt.Sprintf("Approval %s: %s", decision, req.Reason),
		NewValue:   auditMeta,
	})

	// Outbox
	teamIDStr := teamID.String()
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     fmt.Sprintf("clarity.v1.approval.%s", decision),
		AggregateType: "approval_request",
		AggregateID:   approvalID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"approval_id":"%s","decision":"%s","user_id":"%s"}`, approvalID, decision, userID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	// Return current status
	h.pool.QueryRow(ctx, `SELECT status FROM approval_requests WHERE id=$1`, approvalID).Scan(&status)

	writeJSON(w, 200, map[string]any{
		"approval_id": approvalID,
		"decision":    decision,
		"status":      status,
		"reason":      req.Reason,
		"mfa_verified": mfaVerified,
	})
}

// ─── Cancel ───
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	approvalID, err := uuid.Parse(chi.URLParam(r, "approvalId"))
	if err != nil {
		writeErr(w, 400, "Invalid approval ID")
		return
	}

	userIDStr := ctx.Value("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	// Load request
	var status, requestedBy string
	err = h.pool.QueryRow(ctx, `
		SELECT status, requested_by::text FROM approval_requests WHERE id=$1 AND team_id=$2
	`, approvalID, teamID).Scan(&status, &requestedBy)
	if err != nil {
		writeErr(w, 404, "Approval request not found")
		return
	}

	// Terminal state
	if IsTerminalState(status) {
		writeErr(w, 409, fmt.Sprintf("Cannot cancel: approval is in terminal state: %s", status))
		return
	}

	// Only requester or owner/admin can cancel
	requestedUUID, _ := uuid.Parse(requestedBy)
	isRequester := userID == requestedUUID
	isOwner := ctx.Value("team_role") == "owner"
	isAdmin := ctx.Value("team_role") == "admin"
	if !isRequester && !isOwner && !isAdmin {
		writeErr(w, 403, "Only the requester or an admin can cancel this approval")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE approval_requests SET status='cancelled', updated_at=NOW() WHERE id=$1`, approvalID)
	if err != nil {
		writeErr(w, 500, "Failed to cancel")
		return
	}

	teamIDStr := teamID.String()
	audit.Write(ctx, tx, audit.Event{
		ActorID:    userID,
		ActorType:  "user",
		Action:     "approval.cancelled",
		EntityType: "approval_request",
		EntityID:   approvalID,
		TeamID:     &teamID,
		Summary:    "Approval request cancelled",
	})

	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.approval.cancelled",
		AggregateType: "approval_request",
		AggregateID:   approvalID.String(),
		Payload:       json.RawMessage(fmt.Sprintf(`{"approval_id":"%s","user_id":"%s"}`, approvalID, userID)),
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"approval_id": approvalID,
		"status":      "cancelled",
	})
}

// ─── Helpers ───

// hasRecentMFA checks if user has recent MFA verification.
func hasRecentMFA(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) bool {
	var recentMFA *time.Time
	pool.QueryRow(ctx, `
		SELECT MAX(recent_mfa_at) FROM user_sessions
		WHERE user_id=$1 AND revoked_at IS NULL
	`, userID).Scan(&recentMFA)
	if recentMFA == nil {
		return false
	}
	return time.Since(*recentMFA) < 5*time.Minute
}

// sanitizeActionTarget strips known sensitive keys from action_target JSON.
// SanitizeActionTargetForTest exposes sanitizeActionTarget for security review tests.
func SanitizeActionTargetForTest(raw json.RawMessage) json.RawMessage {
	return sanitizeActionTarget(raw)
}

func sanitizeActionTarget(raw json.RawMessage) json.RawMessage {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return json.RawMessage(`{}`)
	}

	sensitiveKeys := []string{
		"token", "secret", "password", "key", "credential",
		"api_key", "auth", "authorization", "webhook_secret",
		"signing_secret", "mfa_code", "recovery_code", "otp",
		"proxmox_token", "proxmox_secret",
	}

	sanitizeMap(data, sensitiveKeys)

	result, _ := json.Marshal(data)
	return result
}

func sanitizeMap(m map[string]any, sensitive []string) {
	for k, v := range m {
		lk := strings.ToLower(k)
		for _, s := range sensitive {
			if strings.Contains(lk, s) {
				m[k] = "[REDACTED]"
				break
			}
		}
		if nested, ok := v.(map[string]any); ok {
			sanitizeMap(nested, sensitive)
		}
	}
}

// requireIdempotency wraps a handler with idempotency key checking.
// This is a lightweight wrapper — the real idempotency middleware handles
// the DB-level checking. This just requires the header.
func requireIdempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") == "" {
			writeErr(w, 400, "Idempotency-Key header required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

// Ensure imports
var _ = pgx.Tx(nil)
