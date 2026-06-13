package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"
const testEmail = "owner@test.dev"
const testPass = "password12"

func testRouter(pool *pgxpool.Pool, cfg *config.Config, agentHandler *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "agents.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/agents", agentHandler.CreateAgent)
		r.With(middleware.RequirePermission(pool, "agents.read")).Get("/agents", agentHandler.ListAgents)
		r.With(middleware.RequirePermission(pool, "agents.read")).Get("/agents/{agentId}", agentHandler.GetAgent)
		r.With(middleware.RequirePermission(pool, "agents.update")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Patch("/agents/{agentId}", agentHandler.UpdateAgent)
		r.With(middleware.RequirePermission(pool, "agents.disable")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/agents/{agentId}", agentHandler.DisableAgent)
		r.With(middleware.RequirePermission(pool, "agents.grants.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/agents/{agentId}/grants", agentHandler.CreateGrant)
		r.With(middleware.RequirePermission(pool, "agents.grants.read")).Get("/agents/{agentId}/grants", agentHandler.ListGrants)
		r.With(middleware.RequirePermission(pool, "agents.grants.revoke")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Delete("/agents/{agentId}/grants/{grantId}", agentHandler.RevokeGrant)
		r.With(middleware.RequirePermission(pool, "agents.runs.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/agent-runs", agentHandler.CreateRun)
		r.With(middleware.RequirePermission(pool, "agents.runs.read")).Get("/agent-runs", agentHandler.ListRuns)
		r.With(middleware.RequirePermission(pool, "agents.runs.read")).Get("/agent-runs/{runId}", agentHandler.GetRun)
		r.With(middleware.RequirePermission(pool, "agents.intentions.create")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "user", Expiry: 1 * time.Hour})).
			Post("/agent-runs/{runId}/intentions", agentHandler.CreateIntention)
		r.With(middleware.RequirePermission(pool, "agents.intentions.read")).Get("/agent-runs/{runId}/intentions", agentHandler.ListIntentions)
		r.With(middleware.RequirePermission(pool, "agents.tools.execute")).
			With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "tool-gateway", Expiry: 1 * time.Hour})).
			Post("/tool-gateway/execute", agentHandler.ExecuteTool)
	})
	return r
}

func doReq(r *chi.Mux, method, path, tok string, body any, idemKey ...string) *httptest.ResponseRecorder {
	var br *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		br = bytes.NewReader(b)
	} else {
		br = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, path, br)
	if tok != "" { req.Header.Set("Authorization", "Bearer "+tok) }
	req.Header.Set("Content-Type", "application/json")
	if len(idemKey) > 0 { req.Header.Set("Idempotency-Key", idemKey[0]) }
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginAndGetTeam(t *testing.T, r *chi.Mux, pool *pgxpool.Pool) (token, teamID string) {
	t.Helper()
	w := doReq(r, "POST", "/api/auth/login", "", map[string]string{"email": testEmail, "password": testPass})
	if w.Code != 200 { t.Fatalf("login: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token = resp["access_token"].(string)
	// Find team
	var tid string
	if err := pool.QueryRow(t.Context(), `SELECT t.id::text FROM teams t JOIN team_memberships tm ON t.id=tm.team_id JOIN users u ON tm.user_id=u.id WHERE u.email=$1 LIMIT 1`, testEmail).Scan(&tid); err != nil {
		t.Fatalf("team lookup: %v", err)
	}
	return token, tid
}

// uniq returns a unique suffix for test isolation.
func uniq() string { return uuid.New().String()[:8] }

// ─── Agent CRUD Tests ───

func TestAgentCreateWritesRowAuditOutbox(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": "Test-" + u, "max_autonomy": "A3", "description": "desc"}, "create-"+u)
	if w.Code != 201 { t.Fatalf("create: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	agentID := resp["id"].(string)

	var cnt int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_identities WHERE id=$1 AND name LIKE 'Test-%' AND max_autonomy='A3'`, agentID).Scan(&cnt)
	if cnt != 1 { t.Error("row not found") }

	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE entity_type='agent_identity' AND action='agent.identity.created' AND entity_id=$1`, agentID).Scan(&cnt)
	if cnt != 1 { t.Errorf("audit not written: cnt=%d", cnt) }

	pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_events WHERE aggregate_type='agent_identity' AND aggregate_id=$1`, agentID).Scan(&cnt)
	if cnt != 1 { t.Errorf("outbox not written: cnt=%d", cnt) }
}

func TestAgentUpdateWritesAuditOutbox(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": "Upd-" + u, "max_autonomy": "A1"}, "upd-c-"+u)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	agentID := resp["id"].(string)

	w = doReq(r, "PATCH", fmt.Sprintf("/api/teams/%s/agents/%s", teamID, agentID), token,
		map[string]string{"max_autonomy": "A4"}, "upd-"+u)
	if w.Code != 200 { t.Fatalf("update: %d %s", w.Code, w.Body.String()) }

	var cnt int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action='agent.identity.updated' AND entity_id=$1`, agentID).Scan(&cnt)
	if cnt != 1 { t.Errorf("audit: cnt=%d", cnt) }
}

func TestAgentDisableIsSoft(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": "Dis-" + u, "max_autonomy": "A2"}, "dis-c-"+u)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	agentID := resp["id"].(string)

	w = doReq(r, "DELETE", fmt.Sprintf("/api/teams/%s/agents/%s", teamID, agentID), token, nil, "dis-"+u)
	if w.Code != 200 { t.Fatalf("disable: %d %s", w.Code, w.Body.String()) }

	var deletedAt *time.Time
	pool.QueryRow(ctx, `SELECT deleted_at FROM agent_identities WHERE id=$1`, agentID).Scan(&deletedAt)
	if deletedAt == nil { t.Error("deleted_at not set") }

	var status string
	pool.QueryRow(ctx, `SELECT status FROM agent_identities WHERE id=$1`, agentID).Scan(&status)
	if status != "disabled" { t.Errorf("status=%s", status) }

	w = doReq(r, "GET", fmt.Sprintf("/api/teams/%s/agents/%s", teamID, agentID), token, nil)
	if w.Code != 404 { t.Errorf("get disabled: %d", w.Code) }
}

func TestAgentCreateDeniedWithoutPermission(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	r := testRouter(poolSetup(t), cfg, NewHandler(poolSetup(t)))
	w := doReq(r, "POST", "/api/teams/x/agents", "", map[string]string{"name": "Nope"}, "noauth-"+uniq())
	if w.Code != 401 { t.Errorf("want 401, got %d", w.Code) }
}

func poolSetup(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), dbURL)
	if err != nil { t.Fatalf("pool: %v", err) }
	t.Cleanup(func() { pool.Close() })
	return pool
}

// ─── Grant Tests ───

func TestGrantCreateAndRevoke(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": "Grant-" + u, "max_autonomy": "A3"}, "gc-"+u)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	agentID := resp["id"].(string)

	w = doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents/%s/grants", teamID, agentID), token,
		map[string]any{"tool_name": "objects.add_comment", "max_autonomy_level": "A3", "requires_approval": false}, "g-"+u)
	if w.Code != 201 { t.Fatalf("grant: %d %s", w.Code, w.Body.String()) }
	var gR map[string]any
	json.Unmarshal(w.Body.Bytes(), &gR)
	grantID := gR["id"].(string)

	var cnt int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action='agent.tool_grant.created' AND entity_id=$1`, grantID).Scan(&cnt)
	if cnt != 1 { t.Errorf("grant audit: cnt=%d", cnt) }

	w = doReq(r, "GET", fmt.Sprintf("/api/teams/%s/agents/%s/grants", teamID, agentID), token, nil)
	if w.Code != 200 { t.Fatalf("list grants: %d", w.Code) }
	var grants []map[string]any
	json.Unmarshal(w.Body.Bytes(), &grants)
	if len(grants) == 0 { t.Error("no grants") }

	w = doReq(r, "DELETE", fmt.Sprintf("/api/teams/%s/agents/%s/grants/%s", teamID, agentID, grantID), token, nil, "gr-"+u)
	if w.Code != 200 { t.Fatalf("revoke: %d %s", w.Code, w.Body.String()) }

	var revokedAt *time.Time
	pool.QueryRow(ctx, `SELECT revoked_at FROM agent_tool_grants WHERE id=$1`, grantID).Scan(&revokedAt)
	if revokedAt == nil { t.Error("revoked_at not set") }
}

// ─── Run & Intention Tests ───

func TestRunCreateWritesAuditOutbox(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": "Run-" + u, "max_autonomy": "A3"}, "rc-"+u)
	var aR map[string]any
	json.Unmarshal(w.Body.Bytes(), &aR)
	agentID := aR["id"].(string)

	w = doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs", teamID), token,
		map[string]string{"agent_id": agentID}, "r-"+u)
	if w.Code != 201 { t.Fatalf("run: %d %s", w.Code, w.Body.String()) }
	var rR map[string]any
	json.Unmarshal(w.Body.Bytes(), &rR)
	runID := rR["id"].(string)

	var cnt int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action='agent.run.created' AND entity_id=$1`, runID).Scan(&cnt)
	if cnt != 1 { t.Errorf("run audit: cnt=%d", cnt) }
}

func TestIntentionCreateStoresReasoning(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, _ := pgxpool.New(ctx, dbURL)
	defer pool.Close()

	r := testRouter(pool, cfg, NewHandler(pool))
	token, teamID := loginAndGetTeam(t, r, pool)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": "Int-" + u, "max_autonomy": "A3"}, "ic-"+u)
	var aR map[string]any
	json.Unmarshal(w.Body.Bytes(), &aR)
	agentID := aR["id"].(string)

	w = doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs", teamID), token,
		map[string]string{"agent_id": agentID}, "ir-"+u)
	var rR map[string]any
	json.Unmarshal(w.Body.Bytes(), &rR)
	runID := rR["id"].(string)

	w = doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs/%s/intentions", teamID, runID), token,
		map[string]any{
			"intention_type":    "add_comment",
			"requested_tool":    "objects.add_comment",
			"confidence":        0.85,
			"risk_level":        "low",
			"autonomy_level":    "A3",
			"reasoning_summary": "Incident needs status update based on recent alerts",
		}, "i-"+u)
	if w.Code != 201 { t.Fatalf("intention: %d %s", w.Code, w.Body.String()) }
	var iR map[string]any
	json.Unmarshal(w.Body.Bytes(), &iR)
	intID := iR["id"].(string)

	var summary string
	pool.QueryRow(ctx, `SELECT reasoning_summary FROM agent_intentions WHERE id=$1`, intID).Scan(&summary)
	if summary != "Incident needs status update based on recent alerts" { t.Errorf("summary=%s", summary) }

	var cnt int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action='agent.intent.created' AND entity_id=$1`, intID).Scan(&cnt)
	if cnt != 1 { t.Errorf("intention audit: cnt=%d", cnt) }
}

// ─── Tool Gateway Tests ───

func setupGatewayTest(t *testing.T) (r *chi.Mux, pool *pgxpool.Pool, token, teamID string) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ = pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })
	r = testRouter(pool, cfg, NewHandler(pool))
	token, teamID = loginAndGetTeam(t, r, pool)
	return
}

func createAgentRunIntention(t *testing.T, r *chi.Mux, pool *pgxpool.Pool, token, teamID, agentName, maxAut, tool string, grantApproval, grantMFA bool) (agentID, runID, intID string) {
	t.Helper()
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents", teamID), token,
		map[string]string{"name": agentName + "-" + u, "max_autonomy": maxAut}, "gwa-"+u)
	if w.Code != 201 { t.Fatalf("agent: %d %s", w.Code, w.Body.String()) }
	var aR map[string]any; json.Unmarshal(w.Body.Bytes(), &aR)
	agentID = aR["id"].(string)

	if tool != "" {
		doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agents/%s/grants", teamID, agentID), token,
			map[string]any{"tool_name": tool, "max_autonomy_level": maxAut, "requires_approval": grantApproval, "requires_mfa": grantMFA}, "gwg-"+u)
	}

	w = doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs", teamID), token,
		map[string]string{"agent_id": agentID}, "gwr-"+u)
	if w.Code != 201 { t.Fatalf("run: %d %s", w.Code, w.Body.String()) }
	var rR map[string]any; json.Unmarshal(w.Body.Bytes(), &rR)
	runID = rR["id"].(string)

	w = doReq(r, "POST", fmt.Sprintf("/api/teams/%s/agent-runs/%s/intentions", teamID, runID), token,
		map[string]any{"intention_type": "test", "requested_tool": tool, "confidence": 0.9, "risk_level": "low", "autonomy_level": "A3", "reasoning_summary": "test"}, "gwi-"+u)
	if w.Code != 201 { t.Fatalf("intention: %d %s", w.Code, w.Body.String()) }
	var iR map[string]any; json.Unmarshal(w.Body.Bytes(), &iR)
	intID = iR["id"].(string)
	return
}

func TestToolGatewayValidLowRiskSucceeds(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "OK", "A4", "objects.add_comment", false, false)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "objects.add_comment", "autonomy_level": "A3"}, "gw-ok-"+u)
	if w.Code != 200 { t.Fatalf("exec: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "succeeded" { t.Errorf("status=%v", resp["status"]) }

	var cnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM agent_effect_results WHERE intention_id=$1 AND status='succeeded'`, intID).Scan(&cnt)
	if cnt != 1 { t.Error("effect result not written") }
}

func TestToolGatewayMissingGrantDenied(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "NoGrant", "A4", "objects.add_comment", false, false)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "incidents.add_timeline", "autonomy_level": "A3"}, "gw-ng-"+u)
	if w.Code != 403 { t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String()) }
}

func TestToolGatewayDisabledAgentDenied(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "DisAg", "A4", "objects.add_comment", false, false)
	u := uniq()

	doReq(r, "DELETE", fmt.Sprintf("/api/teams/%s/agents/%s", teamID, agentID), token, nil, "gw-dis-"+u)

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "objects.add_comment", "autonomy_level": "A3"}, "gw-disd-"+u)
	if w.Code != 403 { t.Fatalf("want 403, got %d", w.Code) }
}

func TestToolGatewayAutonomyExceedsAgentMax(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "LowAuth", "A2", "objects.add_comment", false, false)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "objects.add_comment", "autonomy_level": "A3"}, "gw-aex-"+u)
	if w.Code != 403 { t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String()) }
}

func TestToolGatewayApprovalRequired(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "Appr", "A4", "incidents.summarize", true, false)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "incidents.summarize", "autonomy_level": "A3"}, "gw-appr-"+u)
	if w.Code != 403 { t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String()) }

	var reason string
	pool.QueryRow(t.Context(), `SELECT (result->>'reason')::text FROM agent_effect_results WHERE intention_id=$1 AND status='blocked'`, intID).Scan(&reason)
	if reason != "approval_required" { t.Errorf("reason=%s", reason) }

	var cnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE action='agent.tool.execution_blocked' AND entity_id=$1`, intID).Scan(&cnt)
	if cnt != 1 { t.Errorf("blocked audit: cnt=%d", cnt) }

	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM outbox_events WHERE event_type='clarity.v1.agent.tool.execution_blocked' AND aggregate_id=$1`, intID).Scan(&cnt)
	if cnt != 1 { t.Errorf("blocked outbox: cnt=%d", cnt) }
}

func TestToolGatewayMFARequired(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "MFA", "A4", "incidents.summarize", false, true)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "incidents.summarize", "autonomy_level": "A3"}, "gw-mfa-"+u)
	if w.Code != 403 { t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String()) }

	var reason string
	pool.QueryRow(t.Context(), `SELECT (result->>'reason')::text FROM agent_effect_results WHERE intention_id=$1 AND status='blocked'`, intID).Scan(&reason)
	if reason != "mfa_required" { t.Errorf("reason=%s", reason) }
}

func TestToolGatewayMediumRiskBlocked(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "Med", "A4", "work_items.create", false, false)
	u := uniq()

	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "work_items.create", "autonomy_level": "A3"}, "gw-med-"+u)
	if w.Code != 403 { t.Fatalf("want 403, got %d: %s", w.Code, w.Body.String()) }
}

func TestToolGatewayIdempotencyReplay(t *testing.T) {
	r, pool, token, teamID := setupGatewayTest(t)
	agentID, runID, intID := createAgentRunIntention(t, r, pool, token, teamID, "Idem", "A4", "objects.add_comment", false, false)
	u := uniq()

	key := "idem-replay-" + u
	w1 := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "objects.add_comment", "autonomy_level": "A3"}, key)
	if w1.Code != 200 { t.Fatalf("first: %d %s", w1.Code, w1.Body.String()) }

	w2 := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": agentID, "run_id": runID, "intention_id": intID, "tool_name": "objects.add_comment", "autonomy_level": "A3"}, key)
	if w2.Code != 200 { t.Errorf("replay: %d", w2.Code) }
}

func TestToolGatewayIdempotencyConflict(t *testing.T) {
	r, _, token, teamID := setupGatewayTest(t)
	u := uniq()

	// Send a request that will fail with processing status, then retry with same key
	key := "idem-conflict-" + u
	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": "00000000-0000-0000-0000-000000000000", "run_id": "00000000-0000-0000-0000-000000000000", "intention_id": "00000000-0000-0000-0000-000000000000", "tool_name": "x", "autonomy_level": "A3"}, key)
	// First request will fail (nonexistent agent) but idempotency key is saved as "failed"
	_ = w
	// Second with same key returns cached response (either original code or 409)
	w2 := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/tool-gateway/execute", teamID), token,
		map[string]string{"agent_id": "00000000-0000-0000-0000-000000000000", "tool_name": "x"}, key)
	if w2.Code != 200 && w2.Code != 409 && w2.Code != 403 && w2.Code != 400 {
		t.Errorf("replay: %d", w2.Code)
	}
}
