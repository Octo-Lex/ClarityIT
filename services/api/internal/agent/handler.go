package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.CreateAgent)
	r.Get("/", h.ListAgents)
	r.Route("/{agentId}", func(r chi.Router) {
		r.Get("/", h.GetAgent)
		r.Patch("/", h.UpdateAgent)
		r.Delete("/", h.DisableAgent)
		r.Post("/grants", h.CreateGrant)
		r.Get("/grants", h.ListGrants)
		r.Delete("/grants/{grantId}", h.RevokeGrant)
	})
	return r
}

func claims(r *http.Request) (*iam.TokenClaims, bool) { return iam.GetClaims(r) }

// ─── Create Agent ───

func (h *Handler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	cl, ok := claims(r)
	if !ok { writeErr(w, 401, "unauthorized"); return }
	actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		MaxAutonomy string `json:"max_autonomy"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" { writeErr(w, 400, "name required"); return }
	a := req.MaxAutonomy; if a == "" { a = "A1" }
	if !validAut(a) { writeErr(w, 400, "max_autonomy must be A0-A5"); return }

	var id string
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO agent_identities (team_id,name,description,max_autonomy,created_by) VALUES ($1,$2,$3,$4,$5) RETURNING id::text`, tid, req.Name, req.Description, a, actorID).Scan(&id); err != nil { return err }
		meta, _ := json.Marshal(map[string]any{"name": req.Name, "max_autonomy": a})
		eid, _ := uuid.Parse(id)
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.identity.created", EntityType: "agent_identity", EntityID: eid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.identity.created", AggregateType: "agent_identity", AggregateID: id, Payload: meta})
	})
	if err != nil { writeErr(w, 500, "Failed to create agent"); return }
	writeJSON(w, 201, map[string]any{"id": id})
}

func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, _ := h.pool.Query(ctx, `SELECT id::text,name,description,status,max_autonomy,created_at FROM agent_identities WHERE team_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, chi.URLParam(r, "teamId"))
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, n, d, s, a string; var t time.Time
		rows.Scan(&id, &n, &d, &s, &a, &t)
		out = append(out, map[string]any{"id": id, "name": n, "description": d, "status": s, "max_autonomy": a, "created_at": t})
	}
	if out == nil { out = []map[string]any{} }
	writeJSON(w, 200, out)
}

func (h *Handler) GetAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var id, n, d, s, a string; var c, u time.Time
	err := h.pool.QueryRow(ctx, `SELECT id::text,name,description,status,max_autonomy,created_at,updated_at FROM agent_identities WHERE id=$1 AND team_id=$2 AND deleted_at IS NULL`, chi.URLParam(r, "agentId"), chi.URLParam(r, "teamId")).Scan(&id, &n, &d, &s, &a, &c, &u)
	if err != nil { writeErr(w, 404, "Agent not found"); return }
	writeJSON(w, 200, map[string]any{"id": id, "name": n, "description": d, "status": s, "max_autonomy": a, "created_at": c, "updated_at": u})
}

func (h *Handler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	agentID := chi.URLParam(r, "agentId")
	cl, _ := claims(r)
	actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID)
	aid, _ := uuid.Parse(agentID)

	var req map[string]any
	json.NewDecoder(r.Body).Decode(&req)
	sets := []string{}; args := []any{}
	if n, ok := req["name"].(string); ok && n != "" { sets = append(sets, fmt.Sprintf("name=$%d", len(args)+1)); args = append(args, n) }
	if d, ok := req["description"].(string); ok { sets = append(sets, fmt.Sprintf("description=$%d", len(args)+1)); args = append(args, d) }
	if a, ok := req["max_autonomy"].(string); ok && validAut(a) { sets = append(sets, fmt.Sprintf("max_autonomy=$%d", len(args)+1)); args = append(args, a) }
	if len(sets) == 0 { writeErr(w, 400, "No fields to update"); return }
	sets = append(sets, fmt.Sprintf("updated_at=$%d", len(args)+1)); args = append(args, time.Now())
	args = append(args, aid, tid)

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := fmt.Sprintf("UPDATE agent_identities SET %s WHERE id=$%d AND team_id=$%d AND deleted_at IS NULL", strings.Join(sets, ","), len(args)-1, len(args))
		tag, err := tx.Exec(ctx, q, args...)
		if err != nil { return err }
		if tag.RowsAffected() == 0 { return fmt.Errorf("not found") }
		meta, _ := json.Marshal(req)
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.identity.updated", EntityType: "agent_identity", EntityID: aid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.identity.updated", AggregateType: "agent_identity", AggregateID: agentID, Payload: meta})
	})
	if err != nil { writeErr(w, 500, "Failed to update agent"); return }
	writeJSON(w, 200, map[string]any{"message": "updated"})
}

func (h *Handler) DisableAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId"); agentID := chi.URLParam(r, "agentId")
	cl, _ := claims(r); actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID); aid, _ := uuid.Parse(agentID)

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE agent_identities SET status='disabled',deleted_at=now(),updated_at=now() WHERE id=$1 AND team_id=$2 AND deleted_at IS NULL`, aid, tid)
		if err != nil { return err }
		if tag.RowsAffected() == 0 { return fmt.Errorf("not found") }
		meta, _ := json.Marshal(map[string]any{"status": "disabled"})
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.identity.disabled", EntityType: "agent_identity", EntityID: aid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.identity.disabled", AggregateType: "agent_identity", AggregateID: agentID, Payload: meta})
	})
	if err != nil { writeErr(w, 404, "Agent not found"); return }
	writeJSON(w, 200, map[string]any{"message": "disabled"})
}

// ─── Tool Grants ───

func (h *Handler) CreateGrant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId"); agentID := chi.URLParam(r, "agentId")
	cl, _ := claims(r); actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID); aid, _ := uuid.Parse(agentID)

	var req struct {
		ToolName         string `json:"tool_name"`
		MaxAutonomyLevel string `json:"max_autonomy_level"`
		RequiresApproval bool   `json:"requires_approval"`
		RequiresMFA      bool   `json:"requires_mfa"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.ToolName == "" { writeErr(w, 400, "tool_name required"); return }
	a := req.MaxAutonomyLevel; if a == "" { a = "A3" }
	if !validAut(a) { writeErr(w, 400, "max_autonomy_level must be A0-A5"); return }

	var id string
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var st string
		if err := tx.QueryRow(ctx, `SELECT status FROM agent_identities WHERE id=$1 AND team_id=$2 AND deleted_at IS NULL`, aid, tid).Scan(&st); err != nil { return fmt.Errorf("agent not found") }
		if st != "active" { return fmt.Errorf("agent not active") }
		if err := tx.QueryRow(ctx, `INSERT INTO agent_tool_grants (agent_id,team_id,tool_name,max_autonomy_level,requires_approval,requires_mfa,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`, aid, tid, req.ToolName, a, req.RequiresApproval, req.RequiresMFA, actorID).Scan(&id); err != nil { return err }
		meta, _ := json.Marshal(map[string]any{"tool_name": req.ToolName, "max_autonomy_level": a})
		gid, _ := uuid.Parse(id)
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.tool_grant.created", EntityType: "agent_tool_grant", EntityID: gid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.tool_grant.created", AggregateType: "agent_tool_grant", AggregateID: id, Payload: meta})
	})
	if err != nil {
		if err.Error() == "agent not found" { writeErr(w, 404, err.Error()) } else if err.Error() == "agent not active" { writeErr(w, 409, err.Error()) } else { writeErr(w, 500, "Failed to create grant") }
		return
	}
	writeJSON(w, 201, map[string]any{"id": id})
}

func (h *Handler) ListGrants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, _ := h.pool.Query(ctx, `SELECT id::text,tool_name,max_autonomy_level,requires_approval,requires_mfa,expires_at,created_at,revoked_at FROM agent_tool_grants WHERE agent_id=$1 AND team_id=$2 ORDER BY created_at DESC`, chi.URLParam(r, "agentId"), chi.URLParam(r, "teamId"))
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, tn, ma string; var ra, ea bool; var exp, rev *time.Time; var c time.Time
		rows.Scan(&id, &tn, &ma, &ra, &ea, &exp, &c, &rev)
		out = append(out, map[string]any{"id": id, "tool_name": tn, "max_autonomy_level": ma, "requires_approval": ra, "requires_mfa": ea, "expires_at": exp, "created_at": c, "revoked_at": rev})
	}
	if out == nil { out = []map[string]any{} }
	writeJSON(w, 200, out)
}

func (h *Handler) RevokeGrant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId"); agentID := chi.URLParam(r, "agentId"); grantID := chi.URLParam(r, "grantId")
	cl, _ := claims(r); actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID); aid, _ := uuid.Parse(agentID); gid, _ := uuid.Parse(grantID)

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE agent_tool_grants SET revoked_at=now(),revoked_by=$1 WHERE id=$2 AND agent_id=$3 AND team_id=$4 AND revoked_at IS NULL`, actorID, gid, aid, tid)
		if err != nil { return err }
		if tag.RowsAffected() == 0 { return fmt.Errorf("not found") }
		meta, _ := json.Marshal(map[string]any{"grant_id": grantID})
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.tool_grant.revoked", EntityType: "agent_tool_grant", EntityID: gid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.tool_grant.revoked", AggregateType: "agent_tool_grant", AggregateID: grantID, Payload: meta})
	})
	if err != nil { writeErr(w, 404, "Grant not found"); return }
	writeJSON(w, 200, map[string]any{"message": "revoked"})
}

// ─── Agent Runs ───

func (h *Handler) CreateRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId"); cl, _ := claims(r); actorID, _ := uuid.Parse(cl.UserID); tid, _ := uuid.Parse(teamID)
	var req struct{ AgentID string `json:"agent_id"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.AgentID == "" { writeErr(w, 400, "agent_id required"); return }
	aid, _ := uuid.Parse(req.AgentID)

	var id string
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var st string
		if err := tx.QueryRow(ctx, `SELECT status FROM agent_identities WHERE id=$1 AND team_id=$2 AND deleted_at IS NULL`, aid, tid).Scan(&st); err != nil { return fmt.Errorf("agent not found") }
		if st != "active" { return fmt.Errorf("agent not active") }
		if err := tx.QueryRow(ctx, `INSERT INTO agent_runs (team_id,agent_id,triggered_by,status) VALUES ($1,$2,$3,'pending') RETURNING id::text`, tid, aid, actorID).Scan(&id); err != nil { return err }
		meta, _ := json.Marshal(map[string]any{"agent_id": req.AgentID, "status": "pending"})
		rid, _ := uuid.Parse(id)
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.run.created", EntityType: "agent_run", EntityID: rid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.run.created", AggregateType: "agent_run", AggregateID: id, Payload: meta})
	})
	if err != nil { writeErr(w, 500, "Failed to create run"); return }
	writeJSON(w, 201, map[string]any{"id": id})
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, _ := h.pool.Query(ctx, `SELECT id::text,agent_id::text,status,triggered_by::text,created_at FROM agent_runs WHERE team_id=$1 ORDER BY created_at DESC LIMIT 50`, chi.URLParam(r, "teamId"))
	defer rows.Close()
	var out []map[string]any
	for rows.Next() { var id, aid, s, tb string; var c time.Time; rows.Scan(&id, &aid, &s, &tb, &c); out = append(out, map[string]any{"id": id, "agent_id": aid, "status": s, "triggered_by": tb, "created_at": c}) }
	if out == nil { out = []map[string]any{} }
	writeJSON(w, 200, out)
}

func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var id, aid, s, tb string; var sa, ca *time.Time; var em *string; var c time.Time
	err := h.pool.QueryRow(ctx, `SELECT id::text,agent_id::text,status,triggered_by::text,started_at,completed_at,error_message,created_at FROM agent_runs WHERE id=$1 AND team_id=$2`, chi.URLParam(r, "runId"), chi.URLParam(r, "teamId")).Scan(&id, &aid, &s, &tb, &sa, &ca, &em, &c)
	if err != nil { writeErr(w, 404, "Run not found"); return }
	writeJSON(w, 200, map[string]any{"id": id, "agent_id": aid, "status": s, "triggered_by": tb, "started_at": sa, "completed_at": ca, "error_message": em, "created_at": c})
}

// ─── Intentions ───

func (h *Handler) CreateIntention(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId"); runID := chi.URLParam(r, "runId")
	cl, _ := claims(r); actorID, _ := uuid.Parse(cl.UserID); tid, _ := uuid.Parse(teamID); rid, _ := uuid.Parse(runID)

	var req struct {
		IntentionType    string  `json:"intention_type"`
		RequestedTool    string  `json:"requested_tool"`
		Confidence       float32 `json:"confidence"`
		RiskLevel        string  `json:"risk_level"`
		AutonomyLevel    string  `json:"autonomy_level"`
		ReasoningSummary string  `json:"reasoning_summary"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.RequestedTool == "" || req.IntentionType == "" || req.ReasoningSummary == "" { writeErr(w, 400, "intention_type, requested_tool, reasoning_summary required"); return }
	if !validAut(req.AutonomyLevel) { writeErr(w, 400, "autonomy_level must be A0-A5"); return }
	conf := req.Confidence; if conf <= 0 { conf = 0.5 }
	rl := req.RiskLevel; if rl == "" { rl = "low" }

	var id string
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO agent_intentions (agent_run_id,team_id,intention_type,tool_name,confidence,risk_level,autonomy_level,reasoning_summary,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'proposed') RETURNING id::text`, rid, tid, req.IntentionType, req.RequestedTool, conf, rl, req.AutonomyLevel, req.ReasoningSummary).Scan(&id); err != nil { return err }
		meta, _ := json.Marshal(map[string]any{"tool": req.RequestedTool, "autonomy": req.AutonomyLevel})
		iid, _ := uuid.Parse(id)
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "agent.intent.created", EntityType: "agent_intention", EntityID: iid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.agent.intent.created", AggregateType: "agent_intention", AggregateID: id, Payload: meta})
	})
	if err != nil { writeErr(w, 500, "Failed to create intention"); return }
	writeJSON(w, 201, map[string]any{"id": id})
}

func (h *Handler) ListIntentions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, _ := h.pool.Query(ctx, `SELECT id::text,intention_type,tool_name,confidence,risk_level,autonomy_level,status,blocked_reason,created_at FROM agent_intentions WHERE agent_run_id=$1 AND team_id=$2 ORDER BY created_at DESC`, chi.URLParam(r, "runId"), chi.URLParam(r, "teamId"))
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, it, tn, rl, al, st string; var br *string; var conf float32; var c time.Time
		rows.Scan(&id, &it, &tn, &conf, &rl, &al, &st, &br, &c)
		out = append(out, map[string]any{"id": id, "intention_type": it, "requested_tool": tn, "confidence": conf, "risk_level": rl, "autonomy_level": al, "status": st, "blocked_reason": br, "created_at": c})
	}
	if out == nil { out = []map[string]any{} }
	writeJSON(w, 200, out)
}

// ─── Tool Gateway (Policy-Governed) ───

func (h *Handler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	cl, ok := claims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var req struct {
		AgentID        string          `json:"agent_id"`
		RunID          string          `json:"run_id"`
		IntentionID    string          `json:"intention_id"`
		ToolName       string          `json:"tool_name"`
		AutonomyLevel  string          `json:"autonomy_level"`
		ApprovalID     string          `json:"approval_id"`
		ActionTarget   json.RawMessage `json:"action_target"`
		TargetObjectID string          `json:"target_object_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.AgentID == "" || req.ToolName == "" || req.IntentionID == "" {
		writeErr(w, 400, "agent_id, tool_name, intention_id required")
		return
	}

	agentID, _ := uuid.Parse(req.AgentID)
	runID, _ := uuid.Parse(req.RunID)
	intentionID, _ := uuid.Parse(req.IntentionID)
	autonomy := req.AutonomyLevel
	if autonomy == "" {
		autonomy = "A3"
	}

	// Build policy request
	toolReq := ToolRequest{
		AgentID:        agentID,
		RunID:          runID,
		IntentionID:    intentionID,
		TeamID:         teamID,
		UserID:         actorID,
		ToolName:       req.ToolName,
		AutonomyLevel:  autonomy,
		ActionTarget:   req.ActionTarget,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	}
	if req.ApprovalID != "" {
		aid, err := uuid.Parse(req.ApprovalID)
		if err == nil {
			toolReq.ApprovalID = &aid
		}
	}
	if req.TargetObjectID != "" {
		toid, err := uuid.Parse(req.TargetObjectID)
		if err == nil {
			toolReq.TargetObjectID = &toid
		}
	}

	// Run the 13-check policy chain
	evaluator := NewPolicyEvaluator(h.pool)
	decision, _ := evaluator.Evaluate(ctx, toolReq)

	if decision.Denied() {
		// Record blocked/denied effect with sanitized payload
		h.recordDecision(ctx, teamIDStr, teamID, actorID, intentionID, decision)
		writeErr(w, decision.HTTPStatus(), decision.Reason)
		return
	}

	// Execute within a single transaction
	// Guardrail 7: Mark approval_request executed only in the same tx as the effect result.
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Mark intention executed
		tx.Exec(ctx, `UPDATE agent_intentions SET status='executed' WHERE id=$1`, intentionID)

		// Store effect result with linked approval
		payload, _ := json.Marshal(map[string]any{
			"tool":   req.ToolName,
			"status": "succeeded",
		})
		tx.Exec(ctx,
			`INSERT INTO agent_effect_results (intention_id, team_id, tool_name, status, approval_id, result)
			 VALUES ($1, $2, $3, 'succeeded', $4, $5)`,
			intentionID, teamID, req.ToolName, decision.ApprovalID, payload)

		// Guardrail 7: Mark approval as executed in same transaction
		if decision.ApprovalID != nil {
			tx.Exec(ctx,
				`UPDATE approval_requests SET status='executed', executed_at=NOW(), updated_at=NOW()
				 WHERE id=$1 AND status='approved'`,
				*decision.ApprovalID)
		}

		// Audit — no raw target payload (guardrail 8)
		auditPayload, _ := json.Marshal(map[string]any{
			"tool":        req.ToolName,
			"status":      "succeeded",
			"approval_id": decision.ApprovalID,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID:     &teamID,
			ActorID:    actorID,
			Action:     "agent.tool.execution_succeeded",
			EntityType: "agent_effect_result",
			EntityID:   intentionID,
			NewValue:   auditPayload,
		})

		// Outbox — no raw target (guardrail 8)
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.agent.tool.execution_succeeded",
			AggregateType: "agent_effect_result",
			AggregateID:   req.IntentionID,
			Payload:       auditPayload,
		})
		return nil
	})
	if err != nil {
		writeErr(w, 500, "Execution failed")
		return
	}
	writeJSON(w, 200, map[string]any{"status": "succeeded", "tool": req.ToolName})
}

// recordDecision persists a denied/blocked decision as an effect result,
// audit entry, and outbox event. Payloads are sanitized — no raw target data
// (guardrail 8).
func (h *Handler) recordDecision(ctx context.Context, teamIDStr string, tid, actorID, iid uuid.UUID, d *Decision) {
	effectStatus := d.EffectStatus()

	// Sanitized payload — guardrail 8: denied/block events must not include raw target payloads.
	payload, _ := json.Marshal(map[string]any{
		"tool":    d.ToolName,
		"status":  effectStatus,
		"reason":  d.Reason,
		"outcome": d.Outcome,
		"check":   d.Check,
	})

	_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tx.Exec(ctx, `UPDATE agent_intentions SET status='blocked', blocked_reason=$1 WHERE id=$2`, d.Reason, iid)
		tx.Exec(ctx,
			`INSERT INTO agent_effect_results (intention_id, team_id, tool_name, status, result)
			 VALUES ($1, $2, $3, $4, $5)`,
			iid, tid, d.ToolName, effectStatus, payload)
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID:     &tid,
			ActorID:    actorID,
			Action:     fmt.Sprintf("agent.tool.execution_%s", effectStatus),
			EntityType: "agent_effect_result",
			EntityID:   iid,
			NewValue:   payload,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     fmt.Sprintf("clarity.v1.agent.tool.execution_%s", effectStatus),
			AggregateType: "agent_effect_result",
			AggregateID:   iid.String(),
			Payload:       payload,
		})
		return nil
	})
}

func validAut(a string) bool { return a == "A0" || a == "A1" || a == "A2" || a == "A3" || a == "A4" || a == "A5" }
func autRank(a string) int { return map[string]int{"A0": 0, "A1": 1, "A2": 2, "A3": 3, "A4": 4, "A5": 5}[a] }

func writeJSON(w http.ResponseWriter, s int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(s); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, s int, m string) { writeJSON(w, s, map[string]string{"detail": m}) }
