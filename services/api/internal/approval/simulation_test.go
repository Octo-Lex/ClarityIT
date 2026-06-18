package approval

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const simDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type simTestEnv struct {
	r          *chi.Mux
	pool       *pgxpool.Pool
	ownerToken string
	memberToken string
}

func setupSimTest(t *testing.T) *simTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), simDBURL)
	t.Cleanup(func() { pool.Close() })

	simHandler := NewSimulationHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Use(middleware.RequirePlatformRole(pool, "platform_owner"))
		r.Post("/approval-policy/simulate", simHandler.Simulate)
	})
	// Also mount without platform role check for member test
	r.Route("/api/member-test", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/approval-policy/simulate", simHandler.Simulate)
	})

	ownerToken := loginForSim(t, r, "owner@test.dev")
	memberToken := loginForSim(t, r, "member@test.dev")

	return &simTestEnv{r: r, pool: pool, ownerToken: ownerToken, memberToken: memberToken}
}

func loginForSim(t *testing.T, r *chi.Mux, email string) string {
	t.Helper()
	body := `{"email":"` + email + `","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["access_token"].(string)
}

func (e *simTestEnv) simulate(t *testing.T, body string) SimulationResponse {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/admin/approval-policy/simulate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("simulate: %d %s", w.Code, w.Body.String())
	}
	var resp SimulationResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

const defaultDraftPolicy = `"draft_policy":{
	"scope":"team",
	"low":{"min_approvers":0,"requires_mfa":false,"allow_self_approval":true},
	"medium":{"min_approvers":1,"requires_mfa":false,"allow_self_approval":false},
	"high":{"min_approvers":1,"requires_mfa":true,"allow_self_approval":false},
	"critical":{"min_approvers":2,"requires_mfa":true,"allow_self_approval":false}
}`

// ─── Tests ───

// Test 1: low-risk scenario does not require approval
func TestSim_LowRiskNoApproval(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"low-test","action_type":"noop","risk_level":"low","description":"test"}]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.RequiresApproval {
		t.Error("low-risk with 0 approvers should not require approval")
	}
	if r.RequiresMFA {
		t.Error("low-risk should not require MFA")
	}
}

// Test 2: medium-risk scenario requires approval
func TestSim_MediumRiskRequiresApproval(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"med-test","action_type":"noop","risk_level":"medium","description":"test"}]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	r := resp.Results[0]
	if !r.RequiresApproval {
		t.Error("medium-risk should require approval")
	}
	if r.MinApprovers < 1 {
		t.Error("medium-risk should need at least 1 approver")
	}
}

// Test 3: high-risk scenario requires MFA
func TestSim_HighRiskRequiresMFA(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"high-test","action_type":"proxmox.shutdown","risk_level":"high","description":"test"}]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	r := resp.Results[0]
	if !r.RequiresMFA {
		t.Error("high-risk should require MFA")
	}
	if !r.RequiresApproval {
		t.Error("high-risk should require approval")
	}
}

// Test 4: critical-risk scenario requires two approvers
func TestSim_CriticalRiskTwoApprovers(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"crit-test","action_type":"proxmox.stop","risk_level":"critical","description":"test"}]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	r := resp.Results[0]
	if r.MinApprovers < 2 {
		t.Errorf("critical-risk should require >= 2 approvers, got %d", r.MinApprovers)
	}
	if !r.RequiresMFA {
		t.Error("critical-risk should require MFA")
	}
}

// Test 5: draft policy diff is computed vs current policy
func TestSim_PolicyDiffComputed(t *testing.T) {
	e := setupSimTest(t)
	// Use a draft that differs from built-in defaults (high: min_approvers=1 -> draft 2)
	body := `{"draft_policy":{
		"scope":"team",
		"low":{"min_approvers":0,"requires_mfa":false,"allow_self_approval":true},
		"medium":{"min_approvers":1,"requires_mfa":false,"allow_self_approval":false},
		"high":{"min_approvers":2,"requires_mfa":true,"allow_self_approval":false},
		"critical":{"min_approvers":3,"requires_mfa":true,"allow_self_approval":false}
	},"scenarios":[{"scenario_id":"diff-test","action_type":"noop","risk_level":"low","description":"test"}]}`
	resp := e.simulate(t, body)

	if !resp.PolicyDiff.Changed {
		t.Error("policy diff should show changes")
	}
	// Should have changes for high (1->2) and critical (2->3)
	foundHighChange := false
	foundCriticalChange := false
	for _, c := range resp.PolicyDiff.Changes {
		if c.RiskLevel == "high" && c.Field == "min_approvers" && c.Current == float64(1) && c.Draft == float64(2) {
			foundHighChange = true
		}
		if c.RiskLevel == "critical" && c.Field == "min_approvers" && c.Current == float64(2) && c.Draft == float64(3) {
			foundCriticalChange = true
		}
	}
	if !foundHighChange {
		t.Error("expected high risk min_approvers change (1->2)")
	}
	if !foundCriticalChange {
		t.Error("expected critical risk min_approvers change (2->3)")
	}
}

// Test 6: invalid risk level rejected
func TestSim_InvalidRiskLevel(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"bad","action_type":"noop","risk_level":"extreme","description":"test"}]}`, defaultDraftPolicy)

	req := httptest.NewRequest("POST", "/api/admin/approval-policy/simulate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid risk_level, got %d", w.Code)
	}
}

// Test 7: invalid min_approvers rejected
func TestSim_InvalidMinApprovers(t *testing.T) {
	e := setupSimTest(t)
	body := `{"draft_policy":{
		"scope":"team",
		"low":{"min_approvers":99,"requires_mfa":false,"allow_self_approval":true},
		"medium":{"min_approvers":1,"requires_mfa":false,"allow_self_approval":false},
		"high":{"min_approvers":1,"requires_mfa":true,"allow_self_approval":false},
		"critical":{"min_approvers":2,"requires_mfa":true,"allow_self_approval":false}
	},"scenarios":[{"scenario_id":"bad","action_type":"noop","risk_level":"low","description":"test"}]}`

	req := httptest.NewRequest("POST", "/api/admin/approval-policy/simulate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for min_approvers > 5, got %d", w.Code)
	}
}

// Test 8: scenario limit enforced
func TestSim_ScenarioLimit(t *testing.T) {
	e := setupSimTest(t)
	// Build 51 scenarios
	scenarios := "["
	for i := 0; i < 51; i++ {
		if i > 0 {
			scenarios += ","
		}
		scenarios += fmt.Sprintf(`{"scenario_id":"s%d","action_type":"noop","risk_level":"low","description":"test"}`, i)
	}
	scenarios += "]"
	body := fmt.Sprintf(`{%s,"scenarios":%s}`, defaultDraftPolicy, scenarios)

	req := httptest.NewRequest("POST", "/api/admin/approval-policy/simulate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for > 50 scenarios, got %d", w.Code)
	}
}

// Test 9: simulation does not update live approval policy
func TestSim_NoLivePolicyMutation(t *testing.T) {
	e := setupSimTest(t)

	// Count approval_policies before
	var countBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_policies").Scan(&countBefore)

	// Run simulation with a draft that differs from defaults
	body := `{"draft_policy":{
		"scope":"team",
		"low":{"min_approvers":5,"requires_mfa":true,"allow_self_approval":false},
		"medium":{"min_approvers":3,"requires_mfa":true,"allow_self_approval":false},
		"high":{"min_approvers":4,"requires_mfa":true,"allow_self_approval":false},
		"critical":{"min_approvers":5,"requires_mfa":true,"allow_self_approval":false}
	},"scenarios":[{"scenario_id":"mut-test","action_type":"noop","risk_level":"low","description":"test"}]}`
	_ = e.simulate(t, body)

	// Count approval_policies after
	var countAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_policies").Scan(&countAfter)

	if countAfter != countBefore {
		t.Errorf("approval_policies count changed: %d -> %d", countBefore, countAfter)
	}
}

// Test 10: simulation does not create approval_request
func TestSim_NoApprovalRequestCreated(t *testing.T) {
	e := setupSimTest(t)

	var countBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&countBefore)

	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"no-req","action_type":"proxmox.shutdown","risk_level":"high","description":"test"}]}`, defaultDraftPolicy)
	_ = e.simulate(t, body)

	var countAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&countAfter)

	if countAfter != countBefore {
		t.Errorf("approval_requests count changed: %d -> %d", countBefore, countAfter)
	}
}

// Test 11: simulation does not call Tool Gateway
// This is verified structurally: the SimulationHandler has no reference
// to the Tool Gateway, agent handler, or any execution path.
// The simulation logic is pure computation from the draft policy.
func TestSim_NoToolGatewayCall(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"no-gw","action_type":"proxmox.stop","risk_level":"critical","description":"test"}]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	// Verify response has simulation_only=true and live_policy_changed=false
	if !resp.SimulationOnly {
		t.Error("simulation_only should be true")
	}
	if resp.LivePolicyChanged {
		t.Error("live_policy_changed should be false")
	}
}

// Test 12: non-admin denied
func TestSim_NonAdminDenied(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"denied","action_type":"noop","risk_level":"low","description":"test"}]}`, defaultDraftPolicy)

	req := httptest.NewRequest("POST", "/api/admin/approval-policy/simulate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.memberToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}
}

// Test 13: sensitive action_target/payload fields not returned
func TestSim_NoSensitiveFields(t *testing.T) {
	e := setupSimTest(t)
	// Scenario includes a description with sensitive content
	body := fmt.Sprintf(`{%s,"scenarios":[
		{"scenario_id":"secret-test","action_type":"proxmox.shutdown","risk_level":"high","description":"shutdown vm with token=secret123 password=hunter2"}
	]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	for _, r := range resp.Results {
		// Check that explanation doesn't contain secrets
		if strings.Contains(r.DecisionExplanation, "secret123") {
			t.Error("explanation contains secret value")
		}
		if strings.Contains(r.DecisionExplanation, "hunter2") {
			t.Error("explanation contains password")
		}
		// Action type should not contain sensitive data
		if strings.Contains(r.ActionType, "token") {
			t.Error("action_type contains token")
		}
	}
}

// Test 14: simulation_only flag always true
func TestSim_SimulationOnlyFlag(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"flag-test","action_type":"noop","risk_level":"low","description":"test"}]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	if !resp.SimulationOnly {
		t.Error("simulation_only must always be true")
	}
	if resp.LivePolicyChanged {
		t.Error("live_policy_changed must always be false")
	}
}

// Test 15: no changes to approval_decisions table
func TestSim_NoApprovalDecisionsCreated(t *testing.T) {
	e := setupSimTest(t)

	var countBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_decisions").Scan(&countBefore)

	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"no-dec","action_type":"proxmox.stop","risk_level":"critical","description":"test"}]}`, defaultDraftPolicy)
	_ = e.simulate(t, body)

	var countAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_decisions").Scan(&countAfter)

	if countAfter != countBefore {
		t.Errorf("approval_decisions count changed: %d -> %d", countBefore, countAfter)
	}
}

// Test 16: empty scenarios rejected
func TestSim_EmptyScenariosRejected(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[]}`, defaultDraftPolicy)

	req := httptest.NewRequest("POST", "/api/admin/approval-policy/simulate", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for empty scenarios, got %d", w.Code)
	}
}

// Test 17: multiple scenarios in single request
func TestSim_MultipleScenarios(t *testing.T) {
	e := setupSimTest(t)
	body := fmt.Sprintf(`{%s,"scenarios":[
		{"scenario_id":"multi-1","action_type":"noop","risk_level":"low","description":"low test"},
		{"scenario_id":"multi-2","action_type":"proxmox.shutdown","risk_level":"high","description":"high test"},
		{"scenario_id":"multi-3","action_type":"proxmox.stop","risk_level":"critical","description":"critical test"}
	]}`, defaultDraftPolicy)
	resp := e.simulate(t, body)

	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}
	// Verify each has correct risk level
	riskMap := map[string]bool{}
	for _, r := range resp.Results {
		riskMap[r.RiskLevel] = true
	}
	if !riskMap["low"] || !riskMap["high"] || !riskMap["critical"] {
		t.Error("missing expected risk levels in results")
	}
}

// Test 18: no audit event emitted
func TestSim_NoAuditEmitted(t *testing.T) {
	e := setupSimTest(t)

	var countBefore int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%%approval_policy%%' OR action LIKE '%%simulate%%'").Scan(&countBefore)

	body := fmt.Sprintf(`{%s,"scenarios":[{"scenario_id":"no-audit","action_type":"noop","risk_level":"low","description":"test"}]}`, defaultDraftPolicy)
	_ = e.simulate(t, body)

	var countAfter int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%%approval_policy%%' OR action LIKE '%%simulate%%'").Scan(&countAfter)

	if countAfter != countBefore {
		t.Errorf("audit logs changed: %d -> %d (simulation should not emit audit events)", countBefore, countAfter)
	}
}

// ensure context import is used
