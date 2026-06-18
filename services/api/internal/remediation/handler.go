package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/clarityit/api/internal/agent"
	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(requireIdempotency).Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{remediationId}", h.Get)
	r.With(requireIdempotency).Post("/{remediationId}/approve", h.Approve)
	r.With(requireIdempotency).Post("/{remediationId}/execute", h.Execute)
	r.With(requireIdempotency).Post("/{remediationId}/cancel", h.Cancel)
	return r
}

// ─── Create ───

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	var req struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Source       string `json:"source"`
		RiskLevel    string `json:"risk_level"`
		IncidentID   string `json:"incident_id"`
		AgentRunID   string `json:"agent_run_id"`
		Steps        []struct {
			StepOrder          int             `json:"step_order"`
			ToolName           string          `json:"tool_name"`
			RiskLevel          string          `json:"risk_level"`
			Parameters         json.RawMessage `json:"parameters"`
			ContinueOnFailure  bool            `json:"continue_on_failure"`
		} `json:"steps"`
		// v1.2 Track 1: Evidence pack (optional but required for agent-sourced recommendations)
		Evidence *EvidenceInput `json:"evidence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid request body")
		return
	}
	if req.Title == "" {
		writeErr(w, 400, "title is required")
		return
	}
	if req.Source == "" {
		req.Source = "operator"
	}
	if req.Source != "agent" && req.Source != "operator" {
		writeErr(w, 400, "source must be 'agent' or 'operator'")
		return
	}
	if req.RiskLevel == "" {
		req.RiskLevel = "low"
	}
	if err := validateRiskLevel(req.RiskLevel); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if len(req.Steps) == 0 {
		writeErr(w, 400, "at least one step is required")
		return
	}

	// Guardrail 1: Agent-created remediation must be tied to a valid active agent_run
	var incidentUUID, agentRunUUID *uuid.UUID
	if req.Source == "agent" {
		if req.AgentRunID == "" {
			writeErr(w, 400, "agent-sourced remediation requires agent_run_id")
			return
		}
		aid, err := uuid.Parse(req.AgentRunID)
		if err != nil {
			writeErr(w, 400, "Invalid agent_run_id")
			return
		}
		// Validate agent_run belongs to team and is active
		var runStatus string
		err = h.pool.QueryRow(ctx,
			`SELECT status FROM agent_runs WHERE id=$1 AND team_id=$2`, aid, teamID).Scan(&runStatus)
		if err != nil {
			writeErr(w, 400, "agent_run not found in team")
			return
		}
		if runStatus != "pending" && runStatus != "running" {
			writeErr(w, 400, "agent_run is not active")
			return
		}
		agentRunUUID = &aid
	} else if req.AgentRunID != "" {
		// Operator can optionally link to agent_run
		aid, err := uuid.Parse(req.AgentRunID)
		if err == nil {
			// Guardrail 5: Cross-team agent_run blocked
			var exists bool
			h.pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM agent_runs WHERE id=$1 AND team_id=$2)`, aid, teamID).Scan(&exists)
			if !exists {
				writeErr(w, 400, "agent_run not found in team")
				return
			}
			agentRunUUID = &aid
		}
	}

	// Incident linking
	if req.IncidentID != "" {
		iid, err := uuid.Parse(req.IncidentID)
		if err != nil {
			writeErr(w, 400, "Invalid incident_id")
			return
		}
		// Guardrail 5: Cross-team incident blocked
		var exists bool
		h.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM objects WHERE id=$1 AND team_id=$2 AND object_type='incident' AND deleted_at IS NULL)`,
			iid, teamID).Scan(&exists)
		if !exists {
			writeErr(w, 400, "incident not found in team")
			return
		}
		incidentUUID = &iid
	}

	// Validate all step tool_names exist in tool registry
	for i, step := range req.Steps {
		if step.ToolName == "" {
			writeErr(w, 400, fmt.Sprintf("step %d: tool_name is required", i))
			return
		}
		// Guardrail 3: Step tool_name must exist in Tool Gateway registry
		var toolExists bool
		h.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tool_registry WHERE tool_name=$1 AND is_active)`, step.ToolName).Scan(&toolExists)
		if !toolExists {
			writeErr(w, 400, fmt.Sprintf("step %d: unknown tool '%s'", i, step.ToolName))
			return
		}
		if step.RiskLevel == "" {
			req.Steps[i].RiskLevel = "low"
		}
		if err := validateRiskLevel(step.RiskLevel); err != nil {
			writeErr(w, 400, fmt.Sprintf("step %d: %v", i, err))
			return
		}
		if step.Parameters == nil {
			req.Steps[i].Parameters = json.RawMessage(`{}`)
		}
	}

	// Agent-created proposals are always draft
	proposalStatus := "proposed"
	if req.Source == "agent" {
		proposalStatus = "draft"
	}

	idemKey := r.Header.Get("Idempotency-Key")

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	proposalID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO remediation_proposals
			(id, team_id, title, description, status, risk_level, source,
			 incident_id, agent_run_id, created_by, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, proposalID, teamID, req.Title, req.Description, proposalStatus, req.RiskLevel, req.Source,
		incidentUUID, agentRunUUID, userID, idemKey)
	if err != nil {
		writeErr(w, 500, "Failed to create proposal")
		return
	}

	// Insert steps
	for _, step := range req.Steps {
		stepID := uuid.New()
		sanitizedParams := sanitizeParameters(step.Parameters)
		_, err = tx.Exec(ctx, `
			INSERT INTO remediation_steps
				(id, proposal_id, team_id, step_order, tool_name, risk_level, parameters, continue_on_failure)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, stepID, proposalID, teamID, step.StepOrder, step.ToolName, step.RiskLevel, sanitizedParams, step.ContinueOnFailure)
		if err != nil {
			writeErr(w, 500, "Failed to create step")
			return
		}
	}

	// Audit — sanitized
	auditMeta, _ := json.Marshal(map[string]any{
		"title":      req.Title,
		"source":     req.Source,
		"risk_level": req.RiskLevel,
		"step_count": len(req.Steps),
		"status":     proposalStatus,
	})
	_ = audit.Write(ctx, tx, audit.Event{
		TeamID: &teamID, ActorID: userID, Action: "remediation.proposal.created",
		EntityType: "remediation_proposal", EntityID: proposalID, NewValue: auditMeta,
	})

	_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType: "clarity.v1.remediation.proposal.created",
		AggregateType: "remediation_proposal", AggregateID: proposalID.String(), Payload: auditMeta,
	})

	// v1.2 Track 1: Persist evidence pack if provided
	// Agent-sourced recommendations should include evidence; operator-sourced are optional
	if req.Evidence != nil {
		sanitized, err := ValidateEvidenceInput(*req.Evidence)
		if err != nil {
			writeErr(w, 400, fmt.Sprintf("Invalid evidence: %v", err))
			return
		}
		// recommendation_id = proposal_id (the recommendation IS the proposal)
		if err := PersistEvidence(ctx, h.pool, teamID, proposalID, "remediation_proposal", proposalID, sanitized); err != nil {
			// Log but don't fail the proposal creation
			// Evidence is supplementary, not a gate on proposal creation
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 201, map[string]any{
		"id":          proposalID,
		"status":      proposalStatus,
		"title":       req.Title,
		"risk_level":  req.RiskLevel,
		"source":      req.Source,
		"step_count":  len(req.Steps),
	})
}

// ─── List ───

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	statusFilter := r.URL.Query().Get("status")

	query := `SELECT id::text, title, description, status, risk_level, source,
	                  incident_id::text, agent_run_id::text, created_by::text,
	                  created_at::text, updated_at::text, approved_at::text, completed_at::text
	           FROM remediation_proposals WHERE team_id=$1`
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
		var id, title, desc, status, risk, source, createdBy string
		var incidentID, agentRunID, approvedAt, completedAt *string
		var createdAt, updatedAt string
		rows.Scan(&id, &title, &desc, &status, &risk, &source, &incidentID, &agentRunID,
			&createdBy, &createdAt, &updatedAt, &approvedAt, &completedAt)
		items = append(items, map[string]any{
			"id": id, "title": title, "description": desc, "status": status,
			"risk_level": risk, "source": source, "incident_id": incidentID,
			"agent_run_id": agentRunID, "created_by": createdBy,
			"created_at": createdAt, "updated_at": updatedAt,
			"approved_at": approvedAt, "completed_at": completedAt,
		})
	}
	writeJSON(w, 200, items)
}

// ─── Get ───

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	proposalID, err := uuid.Parse(chi.URLParam(r, "remediationId"))
	if err != nil {
		writeErr(w, 400, "Invalid remediation ID")
		return
	}

	var id, title, desc, status, risk, source, createdBy string
	var incidentID, agentRunID, approvedBy, approvedAt, completedAt *string
	var approvalID *string
	var createdAt, updatedAt string

	err = h.pool.QueryRow(ctx, `
		SELECT id::text, title, description, status, risk_level, source,
		       incident_id::text, agent_run_id::text, created_by::text,
		       approval_id::text, approved_by::text,
		       created_at::text, updated_at::text, approved_at::text, completed_at::text
		FROM remediation_proposals WHERE id=$1 AND team_id=$2
	`, proposalID, teamID).Scan(&id, &title, &desc, &status, &risk, &source,
		&incidentID, &agentRunID, &createdBy, &approvalID, &approvedBy,
		&createdAt, &updatedAt, &approvedAt, &completedAt)
	if err != nil {
		writeErr(w, 404, "Remediation proposal not found")
		return
	}

	// Get steps
	rows, _ := h.pool.Query(ctx, `
		SELECT id::text, step_order, tool_name, risk_level, parameters::text,
		       status, continue_on_failure, error_message,
		       started_at::text, completed_at::text
		FROM remediation_steps WHERE proposal_id=$1 ORDER BY step_order
	`, proposalID)
	defer rows.Close()

	steps := []map[string]any{}
	for rows.Next() {
		var sID, tool, sRisk, sStatus string
		var sOrder int
		var params string
		var contOnFail bool
		var errMsg *string
		var startedAt, sCompletedAt *string
		rows.Scan(&sID, &sOrder, &tool, &sRisk, &params, &sStatus, &contOnFail, &errMsg, &startedAt, &sCompletedAt)
		steps = append(steps, map[string]any{
			"id": sID, "step_order": sOrder, "tool_name": tool,
			"risk_level": sRisk, "parameters": params, "status": sStatus,
			"continue_on_failure": contOnFail, "error_message": errMsg,
			"started_at": startedAt, "completed_at": sCompletedAt,
		})
	}

	writeJSON(w, 200, map[string]any{
		"id": id, "title": title, "description": desc, "status": status,
		"risk_level": risk, "source": source, "incident_id": incidentID,
		"agent_run_id": agentRunID, "created_by": createdBy,
		"approval_id": approvalID, "approved_by": approvedBy,
		"created_at": createdAt, "updated_at": updatedAt,
		"approved_at": approvedAt, "completed_at": completedAt,
		"steps": steps,
	})
}

// ─── Approve ───

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	proposalID, err := uuid.Parse(chi.URLParam(r, "remediationId"))
	if err != nil {
		writeErr(w, 400, "Invalid remediation ID")
		return
	}
	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	var status, riskLevel, source, createdBy string
	err = h.pool.QueryRow(ctx, `
		SELECT status, risk_level, source, created_by::text
		FROM remediation_proposals WHERE id=$1 AND team_id=$2
	`, proposalID, teamID).Scan(&status, &riskLevel, &source, &createdBy)
	if err != nil {
		writeErr(w, 404, "Remediation proposal not found")
		return
	}

	// Must be in draft or proposed state
	if status != "draft" && status != "proposed" {
		writeErr(w, 409, fmt.Sprintf("Cannot approve proposal in status: %s", status))
		return
	}

	// Guardrail 6: High/critical self-approval blocked
	createdUUID, _ := uuid.Parse(createdBy)
	isSelfApproval := userID == createdUUID
	if isSelfApproval && (riskLevel == "high" || riskLevel == "critical") {
		writeErr(w, 403, "Self-approval is not allowed for high/critical remediations")
		return
	}

	// Create approval_request for the remediation
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	approvalID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO approval_requests (id, team_id, action_type, action_target, risk_level, description,
		                                requested_by, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'approved', NOW() + interval '1 hour')
	`, approvalID, teamID, "remediation.approve", json.RawMessage(`{"proposal_id":"`+proposalID.String()+`"}`),
		riskLevel, fmt.Sprintf("Remediation approval: %s", proposalID), userID)
	if err != nil {
		writeErr(w, 500, "Failed to create approval")
		return
	}

	_, err = tx.Exec(ctx, `
		UPDATE remediation_proposals
		SET status='approved', approval_id=$1, approved_by=$2, approved_at=NOW(), updated_at=NOW()
		WHERE id=$3
	`, approvalID, userID, proposalID)
	if err != nil {
		writeErr(w, 500, "Failed to approve")
		return
	}

	// Audit + outbox
	auditMeta, _ := json.Marshal(map[string]any{
		"proposal_id": proposalID.String(),
		"approval_id": approvalID.String(),
		"risk_level":  riskLevel,
	})
	_ = audit.Write(ctx, tx, audit.Event{
		TeamID: &teamID, ActorID: userID, Action: "remediation.proposal.approved",
		EntityType: "remediation_proposal", EntityID: proposalID, NewValue: auditMeta,
	})
	_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType: "clarity.v1.remediation.proposal.approved",
		AggregateType: "remediation_proposal", AggregateID: proposalID.String(), Payload: auditMeta,
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{
		"id":          proposalID,
		"status":      "approved",
		"approval_id": approvalID,
	})
}

// ─── Execute ───

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	proposalID, err := uuid.Parse(chi.URLParam(r, "remediationId"))
	if err != nil {
		writeErr(w, 400, "Invalid remediation ID")
		return
	}
	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	// Load proposal
	var status, riskLevel, source string
	var approvalID *uuid.UUID
	err = h.pool.QueryRow(ctx, `
		SELECT status, risk_level, source, approval_id
		FROM remediation_proposals WHERE id=$1 AND team_id=$2
	`, proposalID, teamID).Scan(&status, &riskLevel, &source, &approvalID)
	if err != nil {
		writeErr(w, 404, "Remediation proposal not found")
		return
	}

	// Guardrail 11: Cancelled proposal cannot execute
	if status == "cancelled" {
		writeErr(w, 409, "Cannot execute cancelled proposal")
		return
	}

	// Guardrail 12: Completed proposal cannot execute again (idempotent replay ok)
	if status == "completed" || status == "executing" {
		writeJSON(w, 200, map[string]any{
			"id": proposalID, "status": status, "message": "already executed",
		})
		return
	}

	// Guardrail 4: Execution blocked without approval
	if status != "approved" {
		writeErr(w, 403, fmt.Sprintf("Proposal must be approved before execution (current: %s)", status))
		return
	}

	// Guardrail 5: High-risk MFA check
	if riskLevel == "high" || riskLevel == "critical" {
		if !h.hasRecentMFA(ctx, userID) {
			h.recordBlock(ctx, teamIDStr, teamID, userID, proposalID, "mfa_required")
			writeErr(w, 403, "Recent MFA verification required for high-risk remediation")
			return
		}
	}

	// Guardrail 2/14: Agent-created direct execution blocked — agent proposals are draft-only
	// Agent proposals should already be blocked by the status check (draft != approved),
	// but this is defense-in-depth.
	if source == "agent" && approvalID == nil {
		h.recordBlock(ctx, teamIDStr, teamID, userID, proposalID, "agent_direct_execution_blocked")
		writeErr(w, 403, "Agent-created proposals cannot execute directly")
		return
	}

	// Load steps
	rows, err := h.pool.Query(ctx, `
		SELECT id, step_order, tool_name, risk_level, parameters, continue_on_failure
		FROM remediation_steps WHERE proposal_id=$1 AND team_id=$2
		ORDER BY step_order
	`, proposalID, teamID)
	if err != nil {
		writeErr(w, 500, "Failed to load steps")
		return
	}

	type stepInfo struct {
		id               uuid.UUID
		order            int
		toolName         string
		riskLevel        string
		params           json.RawMessage
		continueOnFail   bool
	}
	var steps []stepInfo
	for rows.Next() {
		var s stepInfo
		rows.Scan(&s.id, &s.order, &s.toolName, &s.riskLevel, &s.params, &s.continueOnFail)
		steps = append(steps, s)
	}
	rows.Close()

	// Set proposal to executing
	h.pool.Exec(ctx, `UPDATE remediation_proposals SET status='executing', updated_at=NOW() WHERE id=$1`, proposalID)

	// Execute steps in order through Tool Gateway policy evaluator
	anyFailed := false
	var failReason string

	for _, step := range steps {
		// Check MFA for high-risk steps
		if step.riskLevel == "high" || step.riskLevel == "critical" {
			if !h.hasRecentMFA(ctx, userID) {
				h.markStepFailed(ctx, teamID, step.id, "mfa_required_for_step")
				anyFailed = true
				failReason = fmt.Sprintf("step %d: MFA required", step.order)
				if !step.continueOnFail {
					break
				}
				continue
			}
		}

		// Execute through Tool Gateway
		err := h.executeStep(ctx, teamID, userID, proposalID, step.id, step.toolName, step.riskLevel, step.params)
		if err != nil {
			anyFailed = true
			failReason = err.Error()
			if !step.continueOnFail {
				break
			}
		}
	}

	// Update proposal status based on aggregate step state
	finalStatus := "completed"
	if anyFailed {
		finalStatus = "failed"
	}

	_, err = h.pool.Exec(ctx, `
		UPDATE remediation_proposals SET status=$1, completed_at=NOW(), updated_at=NOW() WHERE id=$2
	`, finalStatus, proposalID)
	if err != nil {
		writeErr(w, 500, "Failed to update proposal status")
		return
	}

	// Audit + outbox for execution result
	execMeta, _ := json.Marshal(map[string]any{
		"proposal_id": proposalID.String(),
		"final_status": finalStatus,
		"step_count":  len(steps),
		"failed":      anyFailed,
		"fail_reason": failReason,
	})
	_, _ = h.pool.Exec(ctx, `
		INSERT INTO audit_logs (team_id, actor_id, actor_type, action, entity_type, entity_id, new_value, created_at)
		VALUES ($1, $2, 'user', $3, 'remediation_proposal', $4, $5, NOW())
	`, teamID, userID, fmt.Sprintf("remediation.proposal.%s", finalStatus), proposalID, execMeta)
	_, _ = h.pool.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_id, aggregate_type, event_type, payload, created_at)
		VALUES ($1, 'remediation_proposal', $2, $3, NOW())
	`, proposalID.String(), fmt.Sprintf("clarity.v1.remediation.proposal.%s", finalStatus), execMeta)

	writeJSON(w, 200, map[string]any{
		"id":          proposalID,
		"status":      finalStatus,
		"step_count":  len(steps),
		"any_failed":  anyFailed,
		"fail_reason": failReason,
	})
}

// executeStep runs a single remediation step through the Tool Gateway policy evaluator.
func (h *Handler) executeStep(ctx context.Context, teamID, userID, proposalID, stepID uuid.UUID, toolName, riskLevel string, params json.RawMessage) error {
	// Mark step executing
	h.pool.Exec(ctx, `UPDATE remediation_steps SET status='executing', started_at=NOW(), updated_at=NOW() WHERE id=$1`, stepID)

	// Use the agent PolicyEvaluator for enforcement
	// Create a synthetic intention for the tool execution
	// First, create or reuse an agent run for this remediation
	var agentID, runID, intentionID uuid.UUID

	// Find or create a system agent for remediation execution
	err := h.pool.QueryRow(ctx, `
		SELECT id FROM agent_identities WHERE team_id=$1 AND name='remediation-executor' AND deleted_at IS NULL LIMIT 1
	`, teamID).Scan(&agentID)
	if err != nil {
		// Create one
		err = h.pool.QueryRow(ctx, `
			INSERT INTO agent_identities (team_id, name, description, max_autonomy, created_by)
			VALUES ($1, 'remediation-executor', 'System agent for remediation step execution', 'A4', $2)
			RETURNING id
		`, teamID, userID).Scan(&agentID)
		if err != nil {
			h.markStepFailed(ctx, teamID, stepID, "failed_to_create_executor_agent")
			return fmt.Errorf("step execution: failed to create executor agent")
		}
	}

	// Create a run
	err = h.pool.QueryRow(ctx, `
		INSERT INTO agent_runs (team_id, agent_id, triggered_by, status)
		VALUES ($1, $2, $3, 'running')
		RETURNING id
	`, teamID, agentID, userID).Scan(&runID)
	if err != nil {
		h.markStepFailed(ctx, teamID, stepID, "failed_to_create_run")
		return fmt.Errorf("step execution: failed to create run")
	}

	// Create an intention
	err = h.pool.QueryRow(ctx, `
		INSERT INTO agent_intentions (agent_run_id, team_id, intention_type, tool_name, confidence, risk_level, autonomy_level, reasoning_summary, status)
		VALUES ($1, $2, 'remediation_step', $3, 1.0, $4, 'A3', 'remediation step execution', 'proposed')
		RETURNING id
	`, runID, teamID, toolName, riskLevel).Scan(&intentionID)
	if err != nil {
		h.markStepFailed(ctx, teamID, stepID, "failed_to_create_intention")
		return fmt.Errorf("step execution: failed to create intention")
	}

	// Run the policy evaluator
	pe := agent.NewPolicyEvaluator(h.pool)
	decision, _ := pe.Evaluate(ctx, agent.ToolRequest{
		AgentID:        agentID,
		RunID:          runID,
		IntentionID:    intentionID,
		TeamID:         teamID,
		UserID:         userID,
		ToolName:       toolName,
		AutonomyLevel:  "A3",
		ActionTarget:   params,
		IdempotencyKey: fmt.Sprintf("remediation-%s-step-%s", proposalID, stepID),
	})

	if decision.Denied() {
		h.markStepFailed(ctx, teamID, stepID, decision.Reason)
		return fmt.Errorf("step denied: %s", decision.Reason)
	}

	// Step succeeded — mark it
	h.pool.Exec(ctx, `UPDATE remediation_steps SET status='succeeded', completed_at=NOW(), updated_at=NOW() WHERE id=$1`, stepID)

	// Write effect result
	h.pool.Exec(ctx, `
		INSERT INTO agent_effect_results (intention_id, team_id, tool_name, status, result)
		VALUES ($1, $2, $3, 'succeeded', $4)
	`, intentionID, teamID, toolName, params)

	return nil
}

// ─── Cancel ───

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	proposalID, err := uuid.Parse(chi.URLParam(r, "remediationId"))
	if err != nil {
		writeErr(w, 400, "Invalid remediation ID")
		return
	}
	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	var status string
	err = h.pool.QueryRow(ctx, `SELECT status FROM remediation_proposals WHERE id=$1 AND team_id=$2`,
		proposalID, teamID).Scan(&status)
	if err != nil {
		writeErr(w, 404, "Remediation proposal not found")
		return
	}

	if status == "completed" || status == "executing" || status == "cancelled" {
		writeErr(w, 409, fmt.Sprintf("Cannot cancel proposal in status: %s", status))
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "DB error")
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE remediation_proposals SET status='cancelled', updated_at=NOW() WHERE id=$1`, proposalID)
	if err != nil {
		writeErr(w, 500, "Failed to cancel")
		return
	}

	auditMeta, _ := json.Marshal(map[string]any{"proposal_id": proposalID.String()})
	_ = audit.Write(ctx, tx, audit.Event{
		TeamID: &teamID, ActorID: userID, Action: "remediation.proposal.cancelled",
		EntityType: "remediation_proposal", EntityID: proposalID, NewValue: auditMeta,
	})
	_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType: "clarity.v1.remediation.proposal.cancelled",
		AggregateType: "remediation_proposal", AggregateID: proposalID.String(), Payload: auditMeta,
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Commit failed")
		return
	}

	writeJSON(w, 200, map[string]any{"id": proposalID, "status": "cancelled"})
}

// ─── Helpers ───

func (h *Handler) hasRecentMFA(ctx context.Context, userID uuid.UUID) bool {
	var recentMFA *time.Time
	h.pool.QueryRow(ctx,
		`SELECT MAX(recent_mfa_at) FROM user_sessions WHERE user_id=$1 AND revoked_at IS NULL`,
		userID).Scan(&recentMFA)
	if recentMFA == nil {
		return false
	}
	return time.Since(*recentMFA) < 5*time.Minute
}

func (h *Handler) recordBlock(ctx context.Context, teamIDStr string, tid, userID, proposalID uuid.UUID, reason string) {
	payload, _ := json.Marshal(map[string]any{"proposal_id": proposalID.String(), "reason": reason})
	_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &tid, ActorID: userID, Action: "remediation.proposal.blocked",
			EntityType: "remediation_proposal", EntityID: proposalID, NewValue: payload,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType: "clarity.v1.remediation.proposal.blocked",
			AggregateType: "remediation_proposal", AggregateID: proposalID.String(), Payload: payload,
		})
		return nil
	})
}

func (h *Handler) markStepFailed(ctx context.Context, tid, stepID uuid.UUID, reason string) {
	h.pool.Exec(ctx, `
		UPDATE remediation_steps SET status='failed', error_message=$1, completed_at=NOW(), updated_at=NOW() WHERE id=$2
	`, reason, stepID)
}

func validateRiskLevel(level string) error {
	switch level {
	case "low", "medium", "high", "critical":
		return nil
	default:
		return fmt.Errorf("invalid risk level: %s", level)
	}
}

// sensitiveParamRegex matches keys that look like secrets/tokens/passwords
var sensitiveParamRegex = regexp.MustCompile(`(?i)(token|secret|password|key|credential|api_key|auth|webhook_secret|mfa_code|recovery_code|proxmox_token)`)

func sanitizeParameters(raw json.RawMessage) json.RawMessage {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return json.RawMessage(`{}`)
	}
	sanitizeMapKeys(data)
	result, _ := json.Marshal(data)
	return result
}

func sanitizeMapKeys(m map[string]any) {
	for k, v := range m {
		if sensitiveParamRegex.MatchString(k) {
			m[k] = "[REDACTED]"
		}
		if nested, ok := v.(map[string]any); ok {
			sanitizeMapKeys(nested)
		}
	}
}

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
	writeJSON(w, status, map[string]string{"detail": msg})
}

// Ensure imports
var _ = strings.ToLower
