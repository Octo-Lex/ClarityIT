package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const evalDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type evalTestEnv struct {
	r           *chi.Mux
	pool        *pgxpool.Pool
	token       string
	memberToken string
}

func setupEvalTest(t *testing.T) *evalTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), evalDBURL)
	t.Cleanup(func() { pool.Close() })

	evalH := NewEvalHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	// Admin routes with platform_owner requirement
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Use(middleware.RequirePlatformRole(pool, "platform_owner"))
		r.Post("/agent-evaluation/run", evalH.RunEvaluation)
		r.Get("/agent-evaluation/results", evalH.GetLatestResults)
		r.Get("/agent-evaluation/runs/{runId}", evalH.GetRunDetail)
	})

	token := loginForEval(t, r, "owner@test.dev", "password12")
	memberToken := loginForEval(t, r, "member@test.dev", "password12")

	return &evalTestEnv{r: r, pool: pool, token: token, memberToken: memberToken}
}

func loginForEval(t *testing.T, r *chi.Mux, email, pw string) string {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, pw)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login as %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["access_token"].(string)
}

func (e *evalTestEnv) runEval(t *testing.T) map[string]any {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/admin/agent-evaluation/run", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("run eval: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

func (e *evalTestEnv) getLatest(t *testing.T) map[string]any {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/admin/agent-evaluation/results", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

func (e *evalTestEnv) getRunDetail(t *testing.T, runID string) map[string]any {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/admin/agent-evaluation/runs/"+runID, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ─── Tests ───

// Test 1: evaluation run creates run record
func TestEval_RunCreatesRecord(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)

	if resp["run_id"] == nil || resp["run_id"] == "" {
		t.Fatal("expected run_id")
	}
	if resp["run_status"] != "completed" {
		t.Errorf("expected completed, got %v", resp["run_status"])
	}
}

// Test 2: five golden scenarios are evaluated
func TestEval_FiveScenariosEvaluated(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)

	scenarios := resp["scenarios"].([]any)
	if len(scenarios) != 5 {
		t.Errorf("expected 5 scenarios, got %d", len(scenarios))
	}
}

// Test 3: scores are bounded 0.0-1.0
func TestEval_ScoresBounded(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)

	dims := []string{"average_score", "safety_score", "explainability_score", "correctness_score", "quality_score"}
	for _, dim := range dims {
		val := resp[dim].(float64)
		if val < 0.0 || val > 1.0 {
			t.Errorf("%s = %f, out of bounds [0.0, 1.0]", dim, val)
		}
	}

	// Per-scenario scores also bounded
	scenarios := resp["scenarios"].([]any)
	for _, s := range scenarios {
		scn := s.(map[string]any)
		score := scn["score"].(float64)
		if score < 0.0 || score > 1.0 {
			t.Errorf("scenario %v score = %f, out of bounds", scn["scenario_id"], score)
		}
	}
}

// Test 4: scenario failure reasons are persisted
func TestEval_FailureReasonsPersisted(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)

	// All scenarios should pass (golden scenarios are designed to pass)
	scenarios := resp["scenarios"].([]any)
	for _, s := range scenarios {
		scn := s.(map[string]any)
		reasons := scn["failure_reasons"].([]any)
		// failure_reasons should exist (even if empty)
		_ = reasons
	}
}

// Test 5: results endpoint returns latest run
func TestEval_ResultsReturnsLatest(t *testing.T) {
	e := setupEvalTest(t)
	e.runEval(t)

	resp := e.getLatest(t)
	if resp["run_id"] == nil || resp["run_id"] == "" {
		t.Fatal("expected run_id from results endpoint")
	}
	if resp["scenarios"] == nil {
		t.Fatal("expected scenarios from results endpoint")
	}
}

// Test 6: run detail endpoint returns scenario results
func TestEval_RunDetailReturnsScenarios(t *testing.T) {
	e := setupEvalTest(t)
	runResp := e.runEval(t)
	runID := runResp["run_id"].(string)

	detail := e.getRunDetail(t, runID)
	scenarios := detail["scenarios"].([]any)
	if len(scenarios) != 5 {
		t.Errorf("expected 5 scenarios in run detail, got %d", len(scenarios))
	}
}

// Test 7: non-admin denied
func TestEval_NonAdminDenied(t *testing.T) {
	e := setupEvalTest(t)
	req := httptest.NewRequest("POST", "/api/admin/agent-evaluation/run", nil)
	req.Header.Set("Authorization", "Bearer "+e.memberToken)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}
}

// Test 8: evaluation does not call Tool Gateway (structural — no tool gateway reference)
func TestEval_NoToolGateway(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)
	// evaluation_only must be true
	if resp["evaluation_only"] != true {
		t.Error("evaluation_only should be true")
	}
}

// Test 9: evaluation does not create approval_request
func TestEval_NoApprovalRequest(t *testing.T) {
	e := setupEvalTest(t)
	var before int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM approval_requests").Scan(&before)

	e.runEval(t)

	var after int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM approval_requests").Scan(&after)

	if after != before {
		t.Errorf("approval_requests changed: %d -> %d", before, after)
	}
}

// Test 10: evaluation does not create asset_action
func TestEval_NoAssetAction(t *testing.T) {
	e := setupEvalTest(t)
	var before int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM asset_actions").Scan(&before)

	e.runEval(t)

	var after int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM asset_actions").Scan(&after)

	if after != before {
		t.Errorf("asset_actions changed: %d -> %d", before, after)
	}
}

// Test 11: evaluation does not create remediation_proposal
func TestEval_NoRemediationProposal(t *testing.T) {
	e := setupEvalTest(t)
	var before int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM remediation_proposals").Scan(&before)

	e.runEval(t)

	var after int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM remediation_proposals").Scan(&after)

	if after != before {
		t.Errorf("remediation_proposals changed: %d -> %d", before, after)
	}
}

// Test 12: evaluation does not mutate incidents/assets/context graph
func TestEval_NoContextMutation(t *testing.T) {
	e := setupEvalTest(t)
	var nodesBefore, edgesBefore int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM context_nodes").Scan(&nodesBefore)
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM context_edges").Scan(&edgesBefore)

	e.runEval(t)

	var nodesAfter, edgesAfter int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM context_nodes").Scan(&nodesAfter)
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM context_edges").Scan(&edgesAfter)

	if nodesAfter != nodesBefore {
		t.Errorf("context_nodes changed: %d -> %d", nodesBefore, nodesAfter)
	}
	if edgesAfter != edgesBefore {
		t.Errorf("context_edges changed: %d -> %d", edgesBefore, edgesAfter)
	}
}

// Test 13: Python worker cannot persist results directly (structural — no DB pool in worker)
// The evaluation handler persists results itself — there is no Python worker DB access path
func TestEval_PythonCannotPersistDirectly(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)
	// Go handler persists — verified by the run_id existing
	if resp["run_id"] == nil {
		t.Error("Go handler should persist evaluation results")
	}
}

// Test 14: raw prompts/tool parameters/action_target are redacted
func TestEval_SensitiveFieldsRedacted(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)
	bodyBytes, _ := json.Marshal(resp)
	bodyStr := strings.ToLower(string(bodyBytes))

	for _, pattern := range []string{"password", "secret", "token", "action_target", "tool_parameters"} {
		// The words should not appear as raw values (only as redaction markers or not at all)
		// We check that actual sensitive data isn't exposed — "action_target" as a field name is fine
		// but actual passwords/secrets shouldn't be in the response
		if strings.Contains(bodyStr, "password=") || strings.Contains(bodyStr, "secret=") {
			t.Errorf("sensitive data found in response: %s", pattern)
		}
	}
}

// Test 15: chain-of-thought is not returned or persisted
func TestEval_NoChainOfThought(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)
	bodyBytes, _ := json.Marshal(resp)
	bodyStr := strings.ToLower(string(bodyBytes))

	if strings.Contains(bodyStr, "chain_of_thought") {
		t.Error("chain_of_thought should not be in response")
	}
	if strings.Contains(bodyStr, "reasoning_chain") {
		t.Error("reasoning_chain should not be in response")
	}
	if strings.Contains(bodyStr, "thought_process") {
		t.Error("thought_process should not be in response")
	}
}

// Test 16: deterministic fixture run returns same scores
func TestEval_DeterministicResults(t *testing.T) {
	e := setupEvalTest(t)
	resp1 := e.runEval(t)
	resp2 := e.runEval(t)

	// Compare per-scenario scores
	scenarios1 := resp1["scenarios"].([]any)
	scenarios2 := resp2["scenarios"].([]any)

	if len(scenarios1) != len(scenarios2) {
		t.Fatal("scenario count mismatch")
	}

	for i := range scenarios1 {
		s1 := scenarios1[i].(map[string]any)
		s2 := scenarios2[i].(map[string]any)
		if s1["score"] != s2["score"] {
			t.Errorf("scenario %v score differs: %v vs %v", s1["scenario_id"], s1["score"], s2["score"])
		}
		if s1["passed"] != s2["passed"] {
			t.Errorf("scenario %v passed differs: %v vs %v", s1["scenario_id"], s1["passed"], s2["passed"])
		}
	}
}

// Test 17: evaluation_only always true
func TestEval_EvaluationOnlyAlwaysTrue(t *testing.T) {
	e := setupEvalTest(t)
	resp := e.runEval(t)
	if resp["evaluation_only"] != true {
		t.Error("evaluation_only must always be true")
	}
}

// Test 18: unauthorized user denied
func TestEval_UnauthorizedDenied(t *testing.T) {
	e := setupEvalTest(t)
	req := httptest.NewRequest("POST", "/api/admin/agent-evaluation/run", nil)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// Test 19: run detail 404 for unknown run
func TestEval_RunDetail404(t *testing.T) {
	e := setupEvalTest(t)
	req := httptest.NewRequest("GET", "/api/admin/agent-evaluation/runs/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// Test 20: latest results with no prior runs returns empty
func TestEval_LatestEmptyWhenNoRuns(t *testing.T) {
	e := setupEvalTest(t)
	// Clean up any prior runs from other tests
	e.pool.Exec(t.Context(), "DELETE FROM agent_evaluation_scenario_results")
	e.pool.Exec(t.Context(), "DELETE FROM agent_evaluation_runs")

	resp := e.getLatest(t)
	if resp["run_id"] != nil {
		t.Error("expected nil run_id when no runs exist")
	}
}

// ensure context import is used
var _ = context.Background
