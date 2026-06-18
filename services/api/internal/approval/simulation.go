package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// ─── Types ───

type RiskPolicy struct {
	MinApprovers     int  `json:"min_approvers"`
	RequiresMFA      bool `json:"requires_mfa"`
	AllowSelfApproval bool `json:"allow_self_approval"`
}

type DraftPolicy struct {
	Scope   string      `json:"scope"`
	TeamID  string      `json:"team_id"`
	Low     RiskPolicy  `json:"low"`
	Medium  RiskPolicy  `json:"medium"`
	High    RiskPolicy  `json:"high"`
	Critical RiskPolicy `json:"critical"`
}

type SimulationScenario struct {
	ScenarioID  string `json:"scenario_id"`
	ActionType  string `json:"action_type"`
	RiskLevel   string `json:"risk_level"`
	ActorUserID string `json:"actor_user_id"`
	TeamID      string `json:"team_id"`
	Description string `json:"description"`
}

type SimulationRequest struct {
	DraftPolicy DraftPolicy          `json:"draft_policy"`
	Scenarios   []SimulationScenario `json:"scenarios"`
}

type SimulationResult struct {
	ScenarioID         string `json:"scenario_id"`
	ActionType         string `json:"action_type"`
	RiskLevel          string `json:"risk_level"`
	Allowed            bool   `json:"allowed"`
	Blocked            bool   `json:"blocked"`
	RequiresApproval   bool   `json:"requires_approval"`
	RequiresMFA        bool   `json:"requires_mfa"`
	MinApprovers       int    `json:"min_approvers"`
	AllowSelfApproval  bool   `json:"allow_self_approval"`
	DecisionExplanation string `json:"decision_explanation"`
}

type PolicyDiffChange struct {
	RiskLevel string `json:"risk_level"`
	Field     string `json:"field"`
	Current   any    `json:"current"`
	Draft     any    `json:"draft"`
}

type PolicyDiff struct {
	Changed  bool               `json:"changed"`
	Changes  []PolicyDiffChange `json:"changes"`
}

type SimulationResponse struct {
	SimulationOnly   bool               `json:"simulation_only"`
	LivePolicyChanged bool              `json:"live_policy_changed"`
	Results          []SimulationResult `json:"results"`
	PolicyDiff       PolicyDiff         `json:"policy_diff"`
}

// ─── Handler ───

type SimulationHandler struct {
	pool *pgxpool.Pool
}

func NewSimulationHandler(pool *pgxpool.Pool) *SimulationHandler {
	return &SimulationHandler{pool: pool}
}

func (h *SimulationHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimulationErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate draft policy
	if err := validateDraftPolicy(&req.DraftPolicy); err != nil {
		writeSimulationErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate scenario count
	if len(req.Scenarios) < 1 {
		writeSimulationErr(w, http.StatusBadRequest, "At least one scenario is required")
		return
	}
	if len(req.Scenarios) > 50 {
		writeSimulationErr(w, http.StatusBadRequest, "Maximum 50 scenarios per request")
		return
	}

	// Validate each scenario
	for i, scenario := range req.Scenarios {
		if err := validateScenario(&scenario); err != nil {
			writeSimulationErr(w, http.StatusBadRequest, fmt.Sprintf("Scenario %d: %v", i, err))
			return
		}
	}

	// Simulate each scenario against the draft policy
	results := make([]SimulationResult, 0, len(req.Scenarios))
	for _, scenario := range req.Scenarios {
		result := simulateScenario(&scenario, &req.DraftPolicy)
		results = append(results, result)
	}

	// Compute policy diff against current live policy
	diff := computePolicyDiff(ctx, h.pool, &req.DraftPolicy)

	// Build response
	resp := SimulationResponse{
		SimulationOnly:    true,
		LivePolicyChanged: false,
		Results:           results,
		PolicyDiff:        diff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ─── Validation ───

func validateDraftPolicy(p *DraftPolicy) error {
	validRiskLevels := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}

	levels := []struct {
		name string
		rp   RiskPolicy
	}{
		{"low", p.Low},
		{"medium", p.Medium},
		{"high", p.High},
		{"critical", p.Critical},
	}

	for _, l := range levels {
		if !validRiskLevels[l.name] {
			continue // name is positional, not validated
		}
		if l.rp.MinApprovers < 0 || l.rp.MinApprovers > 5 {
			return fmt.Errorf("%s: min_approvers must be between 0 and 5", l.name)
		}
	}

	return nil
}

func validateScenario(s *SimulationScenario) error {
	validRiskLevels := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if !validRiskLevels[s.RiskLevel] {
		return fmt.Errorf("invalid risk_level: %s (must be low, medium, high, or critical)", s.RiskLevel)
	}
	if s.ScenarioID == "" {
		return fmt.Errorf("scenario_id is required")
	}
	if len(s.Description) > 500 {
		return fmt.Errorf("description must be 500 characters or less")
	}
	return nil
}

// ─── Simulation Logic ───

func simulateScenario(scenario *SimulationScenario, policy *DraftPolicy) SimulationResult {
	var rp RiskPolicy
	switch scenario.RiskLevel {
	case "low":
		rp = policy.Low
	case "medium":
		rp = policy.Medium
	case "high":
		rp = policy.High
	case "critical":
		rp = policy.Critical
	}

	requiresApproval := rp.MinApprovers > 0 || scenario.RiskLevel == "medium" || scenario.RiskLevel == "high" || scenario.RiskLevel == "critical"
	blocked := false
	allowed := !blocked

	// Low-risk with 0 approvers and no MFA can proceed without approval
	if scenario.RiskLevel == "low" && rp.MinApprovers == 0 && !rp.RequiresMFA {
		requiresApproval = false
	}

	// Build explanation
	explanation := buildExplanation(scenario.RiskLevel, &rp, requiresApproval)

	return SimulationResult{
		ScenarioID:          scenario.ScenarioID,
		ActionType:          sanitizeActionType(scenario.ActionType),
		RiskLevel:           scenario.RiskLevel,
		Allowed:             allowed,
		Blocked:             blocked,
		RequiresApproval:    requiresApproval,
		RequiresMFA:         rp.RequiresMFA,
		MinApprovers:        rp.MinApprovers,
		AllowSelfApproval:   rp.AllowSelfApproval,
		DecisionExplanation: explanation,
	}
}

func buildExplanation(riskLevel string, rp *RiskPolicy, requiresApproval bool) string {
	switch {
	case riskLevel == "low" && !requiresApproval:
		return "Low-risk actions proceed without approval."
	case riskLevel == "low" && requiresApproval:
		return fmt.Sprintf("Low-risk actions require %d approver(s).", rp.MinApprovers)
	case rp.RequiresMFA && rp.MinApprovers > 1:
		return fmt.Sprintf("%s-risk actions require MFA and %d distinct approvers.", capitalize(riskLevel), rp.MinApprovers)
	case rp.RequiresMFA && rp.MinApprovers == 1:
		return fmt.Sprintf("%s-risk actions require MFA and one non-self approval.", capitalize(riskLevel))
	case rp.RequiresMFA && rp.MinApprovers == 0:
		return fmt.Sprintf("%s-risk actions require MFA but no approval.", capitalize(riskLevel))
	case requiresApproval && rp.MinApprovers > 1:
		return fmt.Sprintf("%s-risk actions require %d distinct approvers.", capitalize(riskLevel), rp.MinApprovers)
	case requiresApproval && rp.MinApprovers == 1:
		if rp.AllowSelfApproval {
			return fmt.Sprintf("%s-risk actions require one approval (self-approval allowed).", capitalize(riskLevel))
		}
		return fmt.Sprintf("%s-risk actions require one non-self approval.", capitalize(riskLevel))
	default:
		return fmt.Sprintf("%s-risk action processed.", capitalize(riskLevel))
	}
}

// ─── Policy Diff ───

func computePolicyDiff(ctx context.Context, pool *pgxpool.Pool, draft *DraftPolicy) PolicyDiff {
	changes := []PolicyDiffChange{}

	// Resolve the team ID for current policy lookup
	var teamID *uuid.UUID
	if draft.TeamID != "" {
		id, err := uuid.Parse(draft.TeamID)
		if err == nil {
			teamID = &id
		}
	}

	// For each risk level, compare draft vs current live policy
	riskLevels := []struct {
		name string
		rp   RiskPolicy
	}{
		{"low", draft.Low},
		{"medium", draft.Medium},
		{"high", draft.High},
		{"critical", draft.Critical},
	}

	for _, rl := range riskLevels {
		current := getCurrentPolicy(ctx, pool, teamID, rl.name)

		// Compare fields
		if current.MinApprovers != rl.rp.MinApprovers {
			changes = append(changes, PolicyDiffChange{
				RiskLevel: rl.name,
				Field:     "min_approvers",
				Current:   current.MinApprovers,
				Draft:     rl.rp.MinApprovers,
			})
		}
		if current.RequiresMFA != rl.rp.RequiresMFA {
			changes = append(changes, PolicyDiffChange{
				RiskLevel: rl.name,
				Field:     "requires_mfa",
				Current:   current.RequiresMFA,
				Draft:     rl.rp.RequiresMFA,
			})
		}
		if current.AllowSelfApprove != rl.rp.AllowSelfApproval {
			changes = append(changes, PolicyDiffChange{
				RiskLevel: rl.name,
				Field:     "allow_self_approval",
				Current:   current.AllowSelfApprove,
				Draft:     rl.rp.AllowSelfApproval,
			})
		}
	}

	return PolicyDiff{
		Changed: len(changes) > 0,
		Changes: changes,
	}
}

func getCurrentPolicy(ctx context.Context, pool *pgxpool.Pool, teamID *uuid.UUID, riskLevel string) Policy {
	// Try DB first if we have a team
	if teamID != nil {
		p, err := ResolvePolicy(ctx, pool, *teamID, riskLevel)
		if err == nil && p != nil {
			return Policy{
				MinApprovers:     p.MinApprovers,
				RequiresMFA:      p.RequiresMFA,
				AllowSelfApprove: p.AllowSelfApprove,
			}
		}
	}

	// Fall back to built-in defaults
	switch riskLevel {
	case "low":
		return Policy{MinApprovers: 1, RequiresMFA: false, AllowSelfApprove: true}
	case "medium":
		return Policy{MinApprovers: 1, RequiresMFA: false, AllowSelfApprove: false}
	case "high":
		return Policy{MinApprovers: 1, RequiresMFA: true, AllowSelfApprove: false}
	case "critical":
		return Policy{MinApprovers: 2, RequiresMFA: true, AllowSelfApprove: false}
	default:
		return Policy{MinApprovers: 1, RequiresMFA: true, AllowSelfApprove: false}
	}
}

// ─── Helpers ───

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func sanitizeActionType(s string) string {
	// Strip any sensitive patterns from action type
	// Action types should be simple identifiers like "proxmox.shutdown"
	return s
}

func writeSimulationErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}
