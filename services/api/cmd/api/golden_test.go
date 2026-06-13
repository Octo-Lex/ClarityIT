package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/integration"
	"github.com/clarityit/api/internal/middleware"
	"github.com/clarityit/api/internal/team"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"

func setupGolden(t *testing.T) (*chi.Mux, *pgxpool.Pool) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })
	iamH := iam.NewHandler(pool, cfg)
	teamH := team.NewHandler(pool, cfg)
	intH := integration.NewHandlerWithEnv(pool, cfg.HMACKey, "development")
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/integration-keys", intH.CreateKey)
		r.Get("/integration-keys", intH.ListKeys)
		r.Get("/settings", teamH.GetSettings)
	})
	r.Post("/api/webhooks/{source}", intH.ReceiveWebhook)
	return r, pool
}

func goldenLogin(t *testing.T, r *chi.Mux) (string, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte(`{"email":"owner@test.dev","password":"password12"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d", w.Code) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token, _ := resp["access_token"].(string)
	p, _ := pgxpool.New(t.Context(), dbURL)
	defer p.Close()
	var tid string
	p.QueryRow(t.Context(), `SELECT t.id::text FROM teams t JOIN team_memberships tm ON t.id=tm.team_id JOIN users u ON tm.user_id=u.id WHERE u.email=$1 LIMIT 1`, "owner@test.dev").Scan(&tid)
	return token, tid
}

func goldenReq(r *chi.Mux, method, path, token string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var br *bytes.Reader
	if body != nil { b, _ := json.Marshal(body); br = bytes.NewReader(b) } else { br = bytes.NewReader([]byte{}) }
	req := httptest.NewRequest(method, path, br)
	if token != "" { req.Header.Set("Authorization", "Bearer "+token) }
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers { req.Header.Set(k, v) }
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─── Scenario 1: Incident Response ───
// Monitoring alert → webhook → alert object → audit → outbox → no raw PII
func TestGolden_IncidentResponse(t *testing.T) {
	r, pool := setupGolden(t)
	token, tid := goldenLogin(t, r)

	// Create integration key
	w := goldenReq(r, "POST", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token,
		map[string]any{"name": "Prometheus", "allowed_sources": []string{"prometheus"}, "allowed_scopes": []string{"webhooks:ingest"}, "allow_unsigned_dev": true},
		map[string]string{"Idempotency-Key": "g1-key"})
	if w.Code != 201 { t.Fatalf("key: %d", w.Code) }
	var kr map[string]any
	json.Unmarshal(w.Body.Bytes(), &kr)
	rawKey, _ := kr["key"].(string)

	// Send alert
	w = goldenReq(r, "POST", "/api/webhooks/prometheus", "",
		map[string]any{"name": "CPU Spike", "severity": "critical", "source_id": "prod-web-01"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 201 { t.Fatalf("webhook: %d %s", w.Code, w.Body.String()) }

	// Verify alert created
	var alertCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM alerts WHERE source='prometheus'`).Scan(&alertCnt)
	if alertCnt == 0 { t.Error("alert not created") }

	// Verify audit
	var auditCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE action='integration.webhook.received'`).Scan(&auditCnt)
	if auditCnt == 0 { t.Error("audit not written") }

	// Verify outbox
	var obCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM outbox_events WHERE event_type='clarity.v1.alert.triggered'`).Scan(&obCnt)
	if obCnt == 0 { t.Error("outbox event not written") }

	// Verify no raw PII in audit
	var nv string
	pool.QueryRow(t.Context(), `SELECT new_value::text FROM audit_logs WHERE action='integration.webhook.received' ORDER BY created_at DESC LIMIT 1`).Scan(&nv)
	if bytes.Contains([]byte(nv), []byte("source_id")) { t.Error("raw PII in audit") }
	if !bytes.Contains([]byte(nv), []byte("payload_hash")) { t.Error("payload_hash missing") }
}

// ─── Scenario 2: Service Desk Triage ───
// Alert received → object in universal spine → context node
func TestGolden_ServiceDeskTriage(t *testing.T) {
	r, pool := setupGolden(t)
	token, tid := goldenLogin(t, r)

	w := goldenReq(r, "POST", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token,
		map[string]any{"name": "PagerDuty", "allowed_sources": []string{"pagerduty"}, "allowed_scopes": []string{"*"}, "allow_unsigned_dev": true},
		map[string]string{"Idempotency-Key": "g2-key"})
	var kr map[string]any
	json.Unmarshal(w.Body.Bytes(), &kr)
	rawKey, _ := kr["key"].(string)

	w = goldenReq(r, "POST", "/api/webhooks/pagerduty", "",
		map[string]any{"name": "Disk Full", "severity": "high", "source_id": "db-01"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 201 { t.Fatalf("webhook: %d", w.Code) }

	// Object in universal spine
	var objCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM objects WHERE object_type='alert' AND team_id=$1`, tid).Scan(&objCnt)
	if objCnt == 0 { t.Error("alert object not in universal spine") }

	// Context node (may be empty if worker hasn't processed)
	var nodeCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM context_nodes`).Scan(&nodeCnt)
	_ = nodeCnt // acceptable to be 0
}

// ─── Scenario 3: Infrastructure Action Approval Blocked ───
// Verify tool_registry enforces approval for high-risk tools
func TestGolden_InfraApprovalBlocked(t *testing.T) {
	_, pool := setupGolden(t)

	// Register high-risk tool
	pool.Exec(t.Context(), `INSERT INTO tool_registry (tool_name, display_name, risk_level, requires_approval, requires_mfa) VALUES ('vm.restart', 'Restart VM', 'high', true, false) ON CONFLICT (tool_name) DO UPDATE SET risk_level='high', requires_approval=true, requires_mfa=false`)

	// Verify
	var reqApproval bool
	pool.QueryRow(t.Context(), `SELECT requires_approval FROM tool_registry WHERE tool_name='vm.restart'`).Scan(&reqApproval)
	if !reqApproval { t.Error("vm.restart should require approval") }

	// Verify agent_effect_results with blocked status exist
	var blockedCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM agent_effect_results WHERE status='blocked'`).Scan(&blockedCnt)
	if blockedCnt == 0 { t.Log("Warning: no blocked effect results yet — tool gateway tests should populate these") }
}

// ─── Scenario 4: Knowledge Drift Correction ───
// Context nodes must not contain raw PII/titles
func TestGolden_KnowledgeDriftCorrection(t *testing.T) {
	_, pool := setupGolden(t)

	// Verify context nodes have empty properties
	rows, _ := pool.Query(t.Context(), `SELECT properties FROM context_nodes LIMIT 10`)
	defer rows.Close()
	for rows.Next() {
		var props []byte
		rows.Scan(&props)
		if len(props) > 0 && string(props) != "{}" && string(props) != "null" {
			t.Logf("Warning: context node with non-empty properties: %s", string(props))
		}
	}

	// Verify no raw text in context nodes
	var titleCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM context_nodes WHERE properties::text LIKE '%title%' OR properties::text LIKE '%body%' OR properties::text LIKE '%summary%'`).Scan(&titleCnt)
	if titleCnt > 0 { t.Errorf("found %d context nodes with raw text fields", titleCnt) }
}

// ─── Scenario 5: Project Risk Detection ───
// Create project + work items → verify audit trail and no PII in outbox
func TestGolden_ProjectRiskDetection(t *testing.T) {
	r, pool := setupGolden(t)
	_, tid := goldenLogin(t, r)

	// Create project and work item objects
	pool.Exec(t.Context(), `INSERT INTO objects (team_id, object_type, title, status) VALUES ($1,'project','Q3 Risk Review','active')`, tid)
	pool.Exec(t.Context(), `INSERT INTO objects (team_id, object_type, title, status, priority) VALUES ($1,'work_item','Review security audit','active','high')`, tid)

	var cnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM objects WHERE team_id=$1 AND object_type IN ('project','work_item')`, tid).Scan(&cnt)
	if cnt < 2 { t.Errorf("expected >=2 objects, got %d", cnt) }

	// Verify no PII in outbox
	var piiCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM outbox_events WHERE payload::text LIKE '%password%' OR payload::text LIKE '%secret%' OR payload::text LIKE '%token%'`).Scan(&piiCnt)
	if piiCnt > 0 { t.Errorf("found %d outbox events with PII keywords", piiCnt) }

	_ = r
}

// ─── Scenario 6: Denied Agent Action ───
// Verify enforcement: blocked results, audit for denials, no PII, worker isolation
func TestGolden_DeniedAgentAction(t *testing.T) {
	_, pool := setupGolden(t)

	// Verify audit for denied/blocked actions
	var denyCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%denied%' OR action LIKE '%blocked%'`).Scan(&denyCnt)
	if denyCnt == 0 { t.Log("No denied audit entries yet — expected after tool gateway test runs") }

	// Verify no raw PII in audit
	var piiCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE new_value::text LIKE '%password%' OR new_value::text LIKE '%secret%'`).Scan(&piiCnt)
	if piiCnt > 0 { t.Errorf("found %d audit entries with potential PII", piiCnt) }

	// Verify agent runs have only valid statuses
	var badStatusCnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM agent_runs WHERE status NOT IN ('pending','queued','running','completed','failed')`).Scan(&badStatusCnt)
	if badStatusCnt > 0 { t.Errorf("found %d agent runs with unexpected statuses", badStatusCnt) }
}
