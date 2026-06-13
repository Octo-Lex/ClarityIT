package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Test Setup ───

// policyTestEnv holds the shared state for policy tests.
type policyTestEnv struct {
	r       *chi.Mux
	pool    *pgxpool.Pool
	token   string
	teamID  string
	userID  string
}

func setupPolicyTest(t *testing.T) *policyTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:      "test-secret",
		HMACKey:        "test-hmac-key",
		AccessTokenTTL: 15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)

	// Get user ID
	var userID string
	pool.QueryRow(t.Context(),
		"SELECT id::text FROM users WHERE email=$1", testEmail).Scan(&userID)

	// Insert test tools (idempotent)
	for _, tool := range []struct {
		name, risk string
		appr, mfa  bool
	}{
		{"test.high_risk", "high", true, true},
		{"test.critical", "critical", true, true},
		{"test.medium", "medium", true, false},
	} {
		pool.Exec(t.Context(),
			`INSERT INTO tool_registry (tool_name, display_name, description, risk_level, requires_approval, requires_mfa)
			 VALUES ($1, $1, 'test tool', $2, $3, $4) ON CONFLICT (tool_name) DO NOTHING`,
			tool.name, tool.risk, tool.appr, tool.mfa)
	}

	// Clear MFA state
	pool.Exec(t.Context(), "UPDATE user_sessions SET recent_mfa_at=NULL WHERE revoked_at IS NULL")

	return &policyTestEnv{r: r, pool: pool, token: token, teamID: teamID, userID: userID}
}

func (e *policyTestEnv) setMFA(t *testing.T) {
	e.pool.Exec(t.Context(),
		"UPDATE user_sessions SET recent_mfa_at=NOW() WHERE revoked_at IS NULL")
}

// createPolicyPipeline creates agent + grant + run + intention via HTTP.
func (e *policyTestEnv) createPipeline(t *testing.T, agentMax, tool, grantMax string, appr, mfa bool, riskLevel, autLevel string) (agentID, runID, intID string) {
	t.Helper()
	u := uniq()

	// Create agent
	w := doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/agents", e.teamID), e.token,
		map[string]string{"name": "PolTest-" + u, "max_autonomy": agentMax}, "pol-a-"+u)
	if w.Code != 201 {
		t.Fatalf("create agent: %d %s", w.Code, w.Body.String())
	}
	var aR map[string]any
	json.Unmarshal(w.Body.Bytes(), &aR)
	agentID = aR["id"].(string)

	// Create grant
	if tool != "" {
		doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/agents/%s/grants", e.teamID, agentID), e.token,
			map[string]any{"tool_name": tool, "max_autonomy_level": grantMax, "requires_approval": appr, "requires_mfa": mfa}, "pol-g-"+u)
	}

	// Create run
	w = doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs", e.teamID), e.token,
		map[string]string{"agent_id": agentID}, "pol-r-"+u)
	if w.Code != 201 {
		t.Fatalf("create run: %d %s", w.Code, w.Body.String())
	}
	var rR map[string]any
	json.Unmarshal(w.Body.Bytes(), &rR)
	runID = rR["id"].(string)

	// Create intention
	if riskLevel == "" {
		riskLevel = "low"
	}
	if autLevel == "" {
		autLevel = "A3"
	}
	w = doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs/%s/intentions", e.teamID, runID), e.token,
		map[string]any{
			"intention_type":    "test",
			"requested_tool":    tool,
			"confidence":        0.9,
			"risk_level":        riskLevel,
			"autonomy_level":    autLevel,
			"reasoning_summary": "test policy execution",
		}, "pol-i-"+u)
	if w.Code != 201 {
		t.Fatalf("create intention: %d %s", w.Code, w.Body.String())
	}
	var iR map[string]any
	json.Unmarshal(w.Body.Bytes(), &iR)
	intID = iR["id"].(string)
	return
}

// createApprovalDirect inserts an approval request directly via SQL.
func (e *policyTestEnv) createApprovalDirect(t *testing.T, actionType, riskLevel string, target json.RawMessage, status string) string {
	t.Helper()
	if target == nil {
		target = json.RawMessage(`{}`)
	}
	if status == "" {
		status = "approved"
	}
	var id string
	err := e.pool.QueryRow(t.Context(),
		`INSERT INTO approval_requests (team_id, action_type, action_target, risk_level, description, requested_by, status, expires_at)
		 VALUES ($1, $2, $3, $4, 'test approval', $5, $6, NOW() + interval '1 hour')
		 RETURNING id::text`,
		e.teamID, actionType, target, riskLevel, e.userID, status).Scan(&id)
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	return id
}

// executeTool posts to the tool gateway.
func (e *policyTestEnv) execute(t *testing.T, agentID, runID, intID, tool, aut string, extra map[string]any, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{
		"agent_id":       agentID,
		"run_id":         runID,
		"intention_id":   intID,
		"tool_name":      tool,
		"autonomy_level": aut,
	}
	for k, v := range extra {
		body[k] = v
	}
	return doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", e.teamID), e.token, body, idemKey)
}

// ─── Tests ───

// Test 1: A4 blocked without approval
func TestPolicy_A4BlockedWithoutApproval(t *testing.T) {
	e := setupPolicyTest(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4", nil, "pol-noappr-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "approval_required" {
		t.Errorf("expected approval_required, got %v", resp["detail"])
	}

	// Verify effect result stored
	var status string
	e.pool.QueryRow(t.Context(),
		"SELECT status FROM agent_effect_results WHERE intention_id=$1", intID).Scan(&status)
	if status != "blocked" {
		t.Errorf("expected blocked effect, got %s", status)
	}
}

// Test 2: A4 blocked without recent MFA
func TestPolicy_A4BlockedWithoutMFA(t *testing.T) {
	e := setupPolicyTest(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	// Create approval but DON'T set MFA
	approvalID := e.createApprovalDirect(t, "test.high_risk", "high", json.RawMessage(`{"vmid":"100"}`), "approved")

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "100"}},
		"pol-nomfa-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "mfa_required" {
		t.Errorf("expected mfa_required, got %v", resp["detail"])
	}
}

// Test 3: A4 executes after approval + MFA + grant + policy
func TestPolicy_A4ExecutesWithApprovalAndMFA(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	approvalID := e.createApprovalDirect(t, "test.high_risk", "high", json.RawMessage(`{"vmid":"100"}`), "approved")

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "100"}},
		"pol-ok-"+u)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "succeeded" {
		t.Errorf("expected succeeded, got %v", resp["status"])
	}

	// Verify approval marked executed in same transaction
	var approvalStatus string
	e.pool.QueryRow(t.Context(),
		"SELECT status FROM approval_requests WHERE id::text=$1", approvalID).Scan(&approvalStatus)
	if approvalStatus != "executed" {
		t.Errorf("expected approval executed, got %s", approvalStatus)
	}

	// Verify effect result has approval_id linked
	var linkedApproval *string
	e.pool.QueryRow(t.Context(),
		"SELECT approval_id::text FROM agent_effect_results WHERE intention_id=$1", intID).Scan(&linkedApproval)
	if linkedApproval == nil || *linkedApproval != approvalID {
		t.Errorf("expected approval_id linked, got %v", linkedApproval)
	}
}

// Test 4: A4 denied if target outside team
func TestPolicy_A4DeniedTargetOutsideTeam(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	approvalID := e.createApprovalDirect(t, "test.high_risk", "high", json.RawMessage(`{"vmid":"100"}`), "approved")

	// Create object in a different team
	var team2ID uuid.UUID
	e.pool.QueryRow(t.Context(),
		"INSERT INTO teams (name, slug) VALUES ('test-team-x', $1) RETURNING id", "slug-"+u).Scan(&team2ID)
	var objID string
	e.pool.QueryRow(t.Context(),
		"INSERT INTO objects (team_id, object_type, title, status) VALUES ($1, 'vm', 'test', 'active') RETURNING id::text",
		team2ID).Scan(&objID)

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{
			"approval_id":       approvalID,
			"action_target":     map[string]string{"vmid": "100"},
			"target_object_id":  objID,
		},
		"pol-tgt-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "target_not_in_team" {
		t.Errorf("expected target_not_in_team, got %v", resp["detail"])
	}
}

// Test 5: A4 denied if grant scope mismatch (no grant for requested tool)
func TestPolicy_GrantScopeMismatch(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	// Agent has grant for objects.add_comment but tries test.high_risk
	agentID, runID, intID := e.createPipeline(t, "A4", "objects.add_comment", "A4", false, false, "low", "A3")
	u := uniq()

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A3", nil, "pol-scope-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "no_active_grant" {
		t.Errorf("expected no_active_grant, got %v", resp["detail"])
	}
}

// Test 6: A5 disabled hardcoded rejection
func TestPolicy_A5DisabledHardcoded(t *testing.T) {
	e := setupPolicyTest(t)
	// Create an A5 agent and grant (even though A5 should be rejected)
	agentID, runID, intID := e.createPipeline(t, "A5", "objects.add_comment", "A5", false, false, "low", "A5")
	u := uniq()

	w := e.execute(t, agentID, runID, intID, "objects.add_comment", "A5", nil, "pol-a5-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "a5_disabled" {
		t.Errorf("expected a5_disabled, got %v", resp["detail"])
	}
}

// Test 7: Python worker cannot execute directly / no direct route exists
func TestPolicy_NoDirectExecutionRoute(t *testing.T) {
	e := setupPolicyTest(t)

	// Verify no /api/workers/execute route
	w := doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/workers/execute", e.teamID), e.token,
		map[string]string{"tool_name": "test"}, "pol-wkr-"+uniq())
	if w.Code != 404 {
		t.Errorf("expected 404 for workers/execute, got %d", w.Code)
	}

	// Verify tool-gateway requires auth
	w = doReq(e.r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", e.teamID), "",
		map[string]string{"tool_name": "test"}, "")
	if w.Code != 401 {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}

// Test 8: Effect result stored for allowed path
func TestPolicy_EffectResultStored(t *testing.T) {
	e := setupPolicyTest(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "objects.add_comment", "A4", false, false, "low", "A3")
	u := uniq()

	w := e.execute(t, agentID, runID, intID, "objects.add_comment", "A3", nil, "pol-eff-"+u)
	if w.Code != 200 {
		t.Fatalf("execute: %d %s", w.Code, w.Body.String())
	}

	var count int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM agent_effect_results WHERE intention_id=$1 AND status='succeeded'", intID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 succeeded effect, got %d", count)
	}
}

// Test 9: Audit/outbox written for all paths (blocked and allowed)
func TestPolicy_AuditOutboxForAllPaths(t *testing.T) {
	e := setupPolicyTest(t)

	// Blocked path
	agentID1, runID1, intID1 := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u1 := uniq()
	e.execute(t, agentID1, runID1, intID1, "test.high_risk", "A4", nil, "pol-aud-b-"+u1)

	var blockedAudit, blockedOutbox int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE entity_id=$1 AND action LIKE 'agent.tool.execution_%'", intID1).Scan(&blockedAudit)
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM outbox_events WHERE aggregate_id=$1 AND event_type LIKE 'clarity.v1.agent.tool.execution_%'", intID1).Scan(&blockedOutbox)
	if blockedAudit == 0 {
		t.Error("no audit for blocked path")
	}
	if blockedOutbox == 0 {
		t.Error("no outbox for blocked path")
	}

	// Allowed path
	agentID2, runID2, intID2 := e.createPipeline(t, "A4", "objects.add_comment", "A4", false, false, "low", "A3")
	u2 := uniq()
	e.execute(t, agentID2, runID2, intID2, "objects.add_comment", "A3", nil, "pol-aud-a-"+u2)

	var okAudit, okOutbox int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE entity_id=$1 AND action='agent.tool.execution_succeeded'", intID2).Scan(&okAudit)
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM outbox_events WHERE aggregate_id=$1 AND event_type='clarity.v1.agent.tool.execution_succeeded'", intID2).Scan(&okOutbox)
	if okAudit == 0 {
		t.Error("no audit for allowed path")
	}
	if okOutbox == 0 {
		t.Error("no outbox for allowed path")
	}
}

// Test 10: Approval action_type mismatch blocked
func TestPolicy_ApprovalActionTypeMismatch(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	// Create approval for a DIFFERENT tool
	approvalID := e.createApprovalDirect(t, "test.medium", "high", json.RawMessage(`{"vmid":"100"}`), "approved")

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "100"}},
		"pol-at-mismatch-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "approval_action_type_mismatch" {
		t.Errorf("expected approval_action_type_mismatch, got %v", resp["detail"])
	}
}

// Test 11: Approval target mismatch blocked
func TestPolicy_ApprovalTargetMismatch(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	// Approval for vmid=100, execute with vmid=200
	approvalID := e.createApprovalDirect(t, "test.high_risk", "high", json.RawMessage(`{"vmid":"100"}`), "approved")

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "200"}},
		"pol-tgt-mismatch-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "approval_target_mismatch" {
		t.Errorf("expected approval_target_mismatch, got %v", resp["detail"])
	}
}

// Test 12: Expired approval blocked
func TestPolicy_ExpiredApprovalBlocked(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	approvalID := e.createApprovalDirect(t, "test.high_risk", "high", json.RawMessage(`{"vmid":"100"}`), "approved")
	// Expire it
	e.pool.Exec(t.Context(),
		"UPDATE approval_requests SET expires_at=NOW() - interval '1 hour' WHERE id::text=$1", approvalID)

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "100"}},
		"pol-exp-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "approval_expired" {
		t.Errorf("expected approval_expired, got %v", resp["detail"])
	}
}

// Test 13: Cross-team approval blocked
func TestPolicy_CrossTeamApprovalBlocked(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	// Create a second team and an approval in it
	var team2ID uuid.UUID
	e.pool.QueryRow(t.Context(),
		"INSERT INTO teams (name, slug) VALUES ('test-team-cross', $1) RETURNING id", "cross-"+u).Scan(&team2ID)

	var approvalID string
	e.pool.QueryRow(t.Context(),
		`INSERT INTO approval_requests (team_id, action_type, action_target, risk_level, description, requested_by, status, expires_at)
		 VALUES ($1, 'test.high_risk', '{"vmid":"100"}'::jsonb, 'high', 'test', $2, 'approved', NOW() + interval '1 hour')
		 RETURNING id::text`, team2ID, e.userID).Scan(&approvalID)

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "100"}},
		"pol-cross-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "approval_wrong_team" {
		t.Errorf("expected approval_wrong_team, got %v", resp["detail"])
	}
}

// Test 14: Already-executed approval blocked
func TestPolicy_AlreadyExecutedApprovalBlocked(t *testing.T) {
	e := setupPolicyTest(t)
	e.setMFA(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	approvalID := e.createApprovalDirect(t, "test.high_risk", "high", json.RawMessage(`{"vmid":"100"}`), "approved")
	// Mark as already executed
	e.pool.Exec(t.Context(),
		"UPDATE approval_requests SET status='executed', executed_at=NOW() WHERE id::text=$1", approvalID)

	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"approval_id": approvalID, "action_target": map[string]string{"vmid": "100"}},
		"pol-used-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "approval_already_executed" {
		t.Errorf("expected approval_already_executed, got %v", resp["detail"])
	}
}

// Test 15: Unknown tool denied before grant lookup
func TestPolicy_UnknownToolDeniedBeforeGrantLookup(t *testing.T) {
	e := setupPolicyTest(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "objects.add_comment", "A4", false, false, "low", "A3")
	u := uniq()

	// Execute with a tool that doesn't exist in the registry
	w := e.execute(t, agentID, runID, intID, "nonexistent.tool", "A3", nil, "pol-unk-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "tool_not_registered" {
		t.Errorf("expected tool_not_registered, got %v", resp["detail"])
	}

	// Verify the check happened at check #3 (before grant lookup)
	var reason string
	e.pool.QueryRow(t.Context(),
		"SELECT (result->>'reason')::text FROM agent_effect_results WHERE intention_id=$1", intID).Scan(&reason)
	if reason != "tool_not_registered" {
		t.Errorf("effect result reason: %s", reason)
	}
}

// Test 16: Idempotency replay returns same effect result
func TestPolicy_IdempotencyReplayReturnsSameEffect(t *testing.T) {
	e := setupPolicyTest(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "objects.add_comment", "A4", false, false, "low", "A3")
	key := "pol-idem-replay-" + uniq()

	w1 := e.execute(t, agentID, runID, intID, "objects.add_comment", "A3", nil, key)
	if w1.Code != 200 {
		t.Fatalf("first exec: %d %s", w1.Code, w1.Body.String())
	}

	w2 := e.execute(t, agentID, runID, intID, "objects.add_comment", "A3", nil, key)
	if w2.Code != 200 {
		t.Fatalf("replay: %d", w2.Code)
	}

	// Same response content (compare parsed JSON, not raw bytes — key order may differ)
	var resp1, resp2 map[string]any
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp1["status"] != resp2["status"] || resp1["tool"] != resp2["tool"] {
		t.Errorf("replay response mismatch:\n%s\n%s", w1.Body.String(), w2.Body.String())
	}

	// Only 1 effect result
	var count int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM agent_effect_results WHERE intention_id=$1", intID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 effect result, got %d", count)
	}
}

// Test 17: Idempotency conflict rejected
func TestPolicy_IdempotencyConflictRejected(t *testing.T) {
	e := setupPolicyTest(t)
	u := uniq()

	// Insert a 'processing' idempotency key directly to simulate concurrent request
	key := "pol-idem-conflict-" + u
	e.pool.Exec(t.Context(),
		`INSERT INTO idempotency_keys (scope_type, scope_id, key, request_method, request_path, status, expires_at)
		 VALUES ('tool-gateway', $1, $2, 'POST', '/api/teams/' || $3 || '/tool-gateway/execute', 'processing', NOW() + interval '1 hour')`,
		e.userID, key, e.teamID)

	// Try to use the same key → should get 409 conflict
	agentID, runID, intID := e.createPipeline(t, "A4", "objects.add_comment", "A4", false, false, "low", "A3")
	w := e.execute(t, agentID, runID, intID, "objects.add_comment", "A3", nil, key)
	if w.Code != 409 {
		t.Errorf("expected 409 conflict, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 18: Denied/block payload redaction verified
func TestPolicy_DeniedPayloadRedaction(t *testing.T) {
	e := setupPolicyTest(t)
	agentID, runID, intID := e.createPipeline(t, "A4", "test.high_risk", "A4", false, false, "high", "A4")
	u := uniq()

	// Execute with sensitive data in action_target — should be blocked (no approval)
	sensitiveTarget := map[string]any{
		"vmid":     "100",
		"token":    "super-secret-12345",
		"password": "hidden-password",
		"api_key":  "key-abc-def",
	}
	w := e.execute(t, agentID, runID, intID, "test.high_risk", "A4",
		map[string]any{"action_target": sensitiveTarget}, "pol-redact-"+u)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// Check audit doesn't contain raw secrets
	var auditPayload string
	rows, _ := e.pool.Query(t.Context(),
		"SELECT new_value::text FROM audit_logs WHERE entity_id=$1 ORDER BY created_at DESC LIMIT 1", intID)
	for rows.Next() {
		rows.Scan(&auditPayload)
	}
	rows.Close()

	if bytes.Contains([]byte(auditPayload), []byte("super-secret-12345")) {
		t.Error("raw token in audit log")
	}
	if bytes.Contains([]byte(auditPayload), []byte("hidden-password")) {
		t.Error("raw password in audit log")
	}
	if bytes.Contains([]byte(auditPayload), []byte("key-abc-def")) {
		t.Error("raw api_key in audit log")
	}

	// Check outbox doesn't contain raw secrets
	var outboxPayload string
	rows, _ = e.pool.Query(t.Context(),
		"SELECT payload::text FROM outbox_events WHERE aggregate_id=$1 ORDER BY created_at DESC LIMIT 1", intID)
	for rows.Next() {
		rows.Scan(&outboxPayload)
	}
	rows.Close()

	if bytes.Contains([]byte(outboxPayload), []byte("super-secret-12345")) {
		t.Error("raw token in outbox")
	}

	// Check effect result doesn't contain raw secrets
	var effectPayload string
	e.pool.QueryRow(t.Context(),
		"SELECT result::text FROM agent_effect_results WHERE intention_id=$1", intID).Scan(&effectPayload)
	if bytes.Contains([]byte(effectPayload), []byte("super-secret-12345")) {
		t.Error("raw token in effect result")
	}
}

// ─── Bonus: Direct PolicyEvaluator tests ───

// Verify A5 rejection at the evaluator level (no HTTP overhead)
func TestPolicy_EvaluatorA5Hardcoded(t *testing.T) {
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	pe := NewPolicyEvaluator(pool)
	decision, _ := pe.Evaluate(t.Context(), ToolRequest{
		AgentID:       uuid.New(),
		RunID:         uuid.New(),
		IntentionID:   uuid.New(),
		TeamID:        uuid.New(),
		UserID:        uuid.New(),
		ToolName:      "any.tool",
		AutonomyLevel: "A5",
	})

	if decision.Outcome != OutcomeBlockedPolicy {
		t.Errorf("expected blocked_policy, got %s", decision.Outcome)
	}
	if decision.Reason != "a5_disabled" {
		t.Errorf("expected a5_disabled, got %s", decision.Reason)
	}
}

// Verify matchActionTarget correctness
func TestPolicy_MatchActionTarget(t *testing.T) {
	tests := []struct {
		name     string
		stored   string
		req      string
		expected bool
	}{
		{"both empty", ``, ``, true},
		{"identical", `{"vmid":"100"}`, `{"vmid":"100"}`, true},
		{"different value", `{"vmid":"100"}`, `{"vmid":"200"}`, false},
		{"missing key", `{"vmid":"100","node":"pve"}`, `{"vmid":"100"}`, false},
		{"extra key", `{"vmid":"100"}`, `{"vmid":"100","node":"pve"}`, false},
		{"both empty objects", `{}`, `{}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchActionTarget(
				json.RawMessage(tt.stored),
				json.RawMessage(tt.req),
			)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Ensure imports
var _ time.Duration
var _ = middleware.IdempotencyConfig{}
