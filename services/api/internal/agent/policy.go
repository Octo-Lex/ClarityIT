package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Decision Outcomes ───

const (
	OutcomeAllowed         = "allowed"
	OutcomeDenied          = "denied"
	OutcomeBlockedApproval = "blocked_approval_required"
	OutcomeBlockedMFA      = "blocked_mfa_required"
	OutcomeBlockedPolicy   = "blocked_policy"
	OutcomeBlockedRisk     = "blocked_risk"
	OutcomeBlockedScope    = "blocked_scope"
	OutcomeBlockedGrant    = "blocked_grant"
	OutcomeExecuted        = "executed"
	OutcomeFailed          = "failed"
)

// ToolRequest carries all context needed for policy evaluation.
type ToolRequest struct {
	AgentID        uuid.UUID
	RunID          uuid.UUID
	IntentionID    uuid.UUID
	TeamID         uuid.UUID
	UserID         uuid.UUID
	ToolName       string
	AutonomyLevel  string
	ApprovalID     *uuid.UUID
	ActionTarget   json.RawMessage
	TargetObjectID *uuid.UUID
	IdempotencyKey string
}

// Decision is the output of the 13-check policy chain.
type Decision struct {
	Outcome    string
	Reason     string
	Check      int // 0 = all passed, 1-13 = which check failed
	ApprovalID *uuid.UUID
	ToolName   string
}

// PolicyEvaluator runs the 13-check enforcement chain.
type PolicyEvaluator struct {
	pool *pgxpool.Pool
}

func NewPolicyEvaluator(pool *pgxpool.Pool) *PolicyEvaluator {
	return &PolicyEvaluator{pool: pool}
}

// Evaluate runs the 13-check chain in exact order and returns a Decision.
// The handler uses the Decision to record effects, audit, and outbox.
func (pe *PolicyEvaluator) Evaluate(ctx context.Context, req ToolRequest) (*Decision, error) {
	// Pre-check: A5 hardcoded rejection — fail closed before any DB lookup.
	// Guardrail 10: A5 must fail closed even if a grant claims A5.
	if req.AutonomyLevel == "A5" {
		return &Decision{
			Outcome:  OutcomeBlockedPolicy,
			Reason:   "a5_disabled",
			Check:    0,
			ToolName: req.ToolName,
		}, nil
	}

	// Check 1: Agent active
	var agentStatus, agentMax string
	err := pe.pool.QueryRow(ctx,
		`SELECT status, max_autonomy FROM agent_identities
		 WHERE id=$1 AND team_id=$2 AND deleted_at IS NULL`,
		req.AgentID, req.TeamID).Scan(&agentStatus, &agentMax)
	if err != nil {
		return &Decision{Outcome: OutcomeDenied, Reason: "agent_not_found", Check: 1, ToolName: req.ToolName}, nil
	}
	if agentStatus != "active" {
		return &Decision{Outcome: OutcomeDenied, Reason: "agent_disabled", Check: 1, ToolName: req.ToolName}, nil
	}

	// Check 2: Agent run active
	var runStatus string
	err = pe.pool.QueryRow(ctx,
		`SELECT status FROM agent_runs WHERE id=$1 AND team_id=$2`,
		req.RunID, req.TeamID).Scan(&runStatus)
	if err != nil || (runStatus != "pending" && runStatus != "running") {
		return &Decision{Outcome: OutcomeDenied, Reason: "run_not_active", Check: 2, ToolName: req.ToolName}, nil
	}

	// Check 3: Tool registered — unknown tools denied BEFORE grant lookup.
	// Guardrail 9: Unknown tools must be denied before grant lookup.
	var toolRisk string
	var toolRequiresApproval, toolRequiresMFA bool
	err = pe.pool.QueryRow(ctx,
		`SELECT risk_level, requires_approval, requires_mfa FROM tool_registry
		 WHERE tool_name=$1 AND is_active`,
		req.ToolName).Scan(&toolRisk, &toolRequiresApproval, &toolRequiresMFA)
	if err != nil {
		return &Decision{Outcome: OutcomeDenied, Reason: "tool_not_registered", Check: 3, ToolName: req.ToolName}, nil
	}

	// Check 4: Tool grant active
	var grantMax string
	var grantRequiresApproval, grantRequiresMFA bool
	var grantExpiresAt *time.Time
	err = pe.pool.QueryRow(ctx,
		`SELECT max_autonomy_level, requires_approval, requires_mfa, expires_at
		 FROM agent_tool_grants
		 WHERE agent_id=$1 AND tool_name=$2 AND team_id=$3 AND revoked_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`,
		req.AgentID, req.ToolName, req.TeamID).Scan(
		&grantMax, &grantRequiresApproval, &grantRequiresMFA, &grantExpiresAt)
	if err != nil {
		return &Decision{Outcome: OutcomeBlockedGrant, Reason: "no_active_grant", Check: 4, ToolName: req.ToolName}, nil
	}
	if grantExpiresAt != nil && grantExpiresAt.Before(time.Now()) {
		return &Decision{Outcome: OutcomeBlockedGrant, Reason: "grant_expired", Check: 4, ToolName: req.ToolName}, nil
	}

	// Check 5: Grant scope matches target.
	// For v1.0, grants are per-tool-name. Since the grant lookup above used
	// the requested tool_name, the scope is inherently correct. This check
	// exists for future scoped grants (per-target, per-namespace).
	// → Always passes for current schema.

	// Check 6: Requested autonomy <= agent max
	requestedRank := autRank(req.AutonomyLevel)
	if requestedRank > autRank(agentMax) {
		return &Decision{Outcome: OutcomeBlockedPolicy, Reason: "autonomy_exceeds_agent", Check: 6, ToolName: req.ToolName}, nil
	}

	// Check 7: Requested autonomy <= grant max
	if requestedRank > autRank(grantMax) {
		return &Decision{Outcome: OutcomeBlockedPolicy, Reason: "autonomy_exceeds_grant", Check: 7, ToolName: req.ToolName}, nil
	}

	// Check 8: Risk level allowed.
	// A0-A3 cannot execute high/critical risk tools.
	// A4 can execute any risk (subject to approval + MFA in subsequent checks).
	if (toolRisk == "high" || toolRisk == "critical") && requestedRank < 4 {
		return &Decision{Outcome: OutcomeBlockedRisk, Reason: "risk_exceeds_autonomy", Check: 8, ToolName: req.ToolName}, nil
	}

	// Check 9: Approval required → satisfied by approval_requests.
	needsApproval := toolRequiresApproval || grantRequiresApproval ||
		toolRisk == "medium" || toolRisk == "high" || toolRisk == "critical"
	var linkedApproval *uuid.UUID
	if needsApproval {
		if req.ApprovalID == nil {
			return &Decision{
				Outcome:  OutcomeBlockedApproval,
				Reason:   "approval_required",
				Check:    9,
				ToolName: req.ToolName,
			}, nil
		}
		approvalDecision := pe.verifyApproval(ctx, *req.ApprovalID, req.TeamID, req.ToolName, req.ActionTarget)
		if approvalDecision != nil {
			approvalDecision.Check = 9
			return approvalDecision, nil
		}
		linkedApproval = req.ApprovalID
	}

	// Check 10: Recent MFA required → satisfied by sessions.recent_mfa_at.
	needsMFA := toolRequiresMFA || grantRequiresMFA ||
		toolRisk == "high" || toolRisk == "critical"
	if needsMFA {
		if !pe.hasRecentMFA(ctx, req.UserID) {
			return &Decision{
				Outcome:  OutcomeBlockedMFA,
				Reason:   "mfa_required",
				Check:    10,
				ToolName: req.ToolName,
			}, nil
		}
	}

	// Check 11: Team permission satisfied (defense-in-depth).
	hasPermission := pe.checkTeamPermission(ctx, req.UserID, req.TeamID, "agents.tools.execute")
	if !hasPermission {
		return &Decision{
			Outcome:  OutcomeDenied,
			Reason:   "permission_denied",
			Check:    11,
			ToolName: req.ToolName,
		}, nil
	}

	// Check 12: Idempotency key valid.
	if req.IdempotencyKey == "" {
		return &Decision{
			Outcome:  OutcomeDenied,
			Reason:   "idempotency_key_required",
			Check:    12,
			ToolName: req.ToolName,
		}, nil
	}

	// Check 13: Target belongs to team.
	if req.TargetObjectID != nil {
		var belongs bool
		err := pe.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM objects WHERE id=$1 AND team_id=$2)`,
			*req.TargetObjectID, req.TeamID).Scan(&belongs)
		if err != nil || !belongs {
			return &Decision{
				Outcome:  OutcomeDenied,
				Reason:   "target_not_in_team",
				Check:    13,
				ToolName: req.ToolName,
			}, nil
		}
	}

	// All 13 checks passed.
	return &Decision{
		Outcome:    OutcomeAllowed,
		Reason:     "all_checks_passed",
		Check:      0,
		ApprovalID: linkedApproval,
		ToolName:   req.ToolName,
	}, nil
}

// verifyApproval runs the 10 additional guardrails for approval validation.
// Returns nil if the approval is valid. Returns a Decision if invalid.
func (pe *PolicyEvaluator) verifyApproval(
	ctx context.Context,
	approvalID, teamID uuid.UUID,
	toolName string,
	actionTarget json.RawMessage,
) *Decision {
	var status, approvalActionType string
	var approvalTarget json.RawMessage
	var approvalTeamID uuid.UUID
	var expiresAt time.Time
	var executedAt *time.Time

	err := pe.pool.QueryRow(ctx,
		`SELECT status, action_type, action_target, team_id, expires_at, executed_at
		 FROM approval_requests WHERE id=$1`,
		approvalID).Scan(&status, &approvalActionType, &approvalTarget,
		&approvalTeamID, &expiresAt, &executedAt)
	if err != nil {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_not_found", ToolName: toolName}
	}

	// Guardrail 5: Cross-team approval blocked.
	if approvalTeamID != teamID {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_wrong_team", ToolName: toolName}
	}

	// Guardrail 6: Already-executed approval blocked (checked before status
	// so we get the more specific reason when status='executed' and executed_at is set).
	if executedAt != nil {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_already_executed", ToolName: toolName}
	}

	// Guardrail 4: Status must be 'approved'.
	if status != "approved" {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_not_approved", ToolName: toolName}
	}

	// Guardrail 3: Expired approval blocked.
	if time.Now().After(expiresAt) {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_expired", ToolName: toolName}
	}

	// Guardrail 1: action_type must match requested tool.
	if approvalActionType != toolName {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_action_type_mismatch", ToolName: toolName}
	}

	// Guardrail 2: action_target must match requested target.
	if !matchActionTarget(approvalTarget, actionTarget) {
		return &Decision{Outcome: OutcomeBlockedApproval, Reason: "approval_target_mismatch", ToolName: toolName}
	}

	return nil // approval is valid
}

// hasRecentMFA checks if the user has a session with MFA within the last 5 minutes.
func (pe *PolicyEvaluator) hasRecentMFA(ctx context.Context, userID uuid.UUID) bool {
	var recentMFA *time.Time
	pe.pool.QueryRow(ctx,
		`SELECT MAX(recent_mfa_at) FROM user_sessions
		 WHERE user_id=$1 AND revoked_at IS NULL`,
		userID).Scan(&recentMFA)
	if recentMFA == nil {
		return false
	}
	return time.Since(*recentMFA) < 5*time.Minute
}

// checkTeamPermission verifies the user has a specific permission in the team.
func (pe *PolicyEvaluator) checkTeamPermission(ctx context.Context, userID, teamID uuid.UUID, permissionName string) bool {
	var exists bool
	err := pe.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM team_memberships tm
			JOIN roles r ON r.id = tm.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE tm.user_id=$1 AND tm.team_id=$2 AND p.name=$3
		)`,
		userID, teamID, permissionName).Scan(&exists)
	return err == nil && exists
}

// matchActionTarget performs deep equality comparison of two JSON action targets.
// Sensitive fields are already redacted in the stored version, so only
// non-sensitive identity fields (vmid, node, etc.) participate in matching.
func matchActionTarget(stored, requested json.RawMessage) bool {
	if len(stored) == 0 && len(requested) == 0 {
		return true
	}
	if len(stored) == 0 || len(requested) == 0 {
		return false
	}
	var s, r interface{}
	if err := json.Unmarshal(stored, &s); err != nil {
		return false
	}
	if err := json.Unmarshal(requested, &r); err != nil {
		return false
	}
	return reflect.DeepEqual(s, r)
}

// Denied returns true if the outcome is a denial or block (not allowed/executed).
func (d *Decision) Denied() bool {
	return d.Outcome != OutcomeAllowed && d.Outcome != OutcomeExecuted
}

// HTTPStatus returns the appropriate HTTP status code for a decision.
func (d *Decision) HTTPStatus() int {
	switch d.Outcome {
	case OutcomeAllowed, OutcomeExecuted:
		return 200
	case OutcomeFailed:
		return 500
	default:
		return 403
	}
}

// EffectStatus maps the decision outcome to agent_effect_results.status.
func (d *Decision) EffectStatus() string {
	switch d.Outcome {
	case OutcomeAllowed, OutcomeExecuted:
		return "succeeded"
	case OutcomeFailed:
		return "failed"
	case OutcomeDenied:
		return "denied"
	default:
		return "blocked"
	}
}

// String returns a human-readable description of the decision.
func (d *Decision) String() string {
	if d.Check == 0 {
		return fmt.Sprintf("allowed (%s)", d.Reason)
	}
	return fmt.Sprintf("%s at check %d: %s", d.Outcome, d.Check, d.Reason)
}
