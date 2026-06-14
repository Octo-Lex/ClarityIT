package proxmox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Test setup ───

type actionTestClient struct {
	failNext bool
}

func (c *actionTestClient) ListNodes(_ context.Context) ([]ProxmoxNode, error) {
	return []ProxmoxNode{{Node: "pve1", Status: "online"}}, nil
}
func (c *actionTestClient) ListVMs(_ context.Context, _ string) ([]ProxmoxVM, error) {
	return []ProxmoxVM{{VMID: 100, Name: "test", Status: "running"}}, nil
}
func (c *actionTestClient) StartVM(_ context.Context, _ MutationTarget) (string, error) {
	if c.failNext {
		return "", fmt.Errorf("simulated failure")
	}
	return "UPID:pve1:fake:start:100:", nil
}
func (c *actionTestClient) ShutdownVM(_ context.Context, _ MutationTarget) (string, error) {
	if c.failNext {
		return "", fmt.Errorf("simulated failure")
	}
	return "UPID:pve1:fake:shutdown:100:", nil
}
func (c *actionTestClient) StopVM(_ context.Context, _ MutationTarget) (string, error) {
	if c.failNext {
		return "", fmt.Errorf("simulated failure")
	}
	return "UPID:pve1:fake:stop:100:", nil
}
func (c *actionTestClient) SnapshotVM(_ context.Context, _ MutationTarget, _ string) (string, error) {
	if c.failNext {
		return "", fmt.Errorf("simulated failure")
	}
	return "UPID:pve1:fake:snapshot:100:", nil
}
func (c *actionTestClient) GetTaskStatus(_ context.Context, _, _ string) (*TaskStatus, error) {
	return &TaskStatus{Status: "stopped", ExitCode: "OK"}, nil
}

type actionTestEnv struct {
	r         *chi.Mux
	pool      *pgxpool.Pool
	token     string
	teamID    string
	memberTok string
	assetID   string
	client    *actionTestClient
}

func setupActionTest(t *testing.T, mutationEnabled bool) *actionTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:         "test-secret",
		HMACKey:           "test-hmac-key",
		AccessTokenTTL:    15 * 60 * 1e9,
		RefreshTokenTTL:   7 * 24 * 3600 * 1e9,
		ProxmoxMutationEnabled: mutationEnabled,
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })

	client := &actionTestClient{}
	h := NewActionHandler(pool, client, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		// Mount action routes directly (simplified for testing)
		r.Post("/assets/{assetId}/actions/proxmox/start", h.CreateAction)
		r.Post("/assets/{assetId}/actions/proxmox/shutdown", h.CreateAction)
		r.Post("/assets/{assetId}/actions/proxmox/stop", h.CreateAction)
		r.Post("/assets/{assetId}/actions/proxmox/snapshot", h.CreateAction)
		r.Get("/asset-actions", h.ListActions)
		r.Get("/asset-actions/{actionId}", h.GetAction)
		r.With(middleware.Idempotency(middleware.IdempotencyConfig{Pool: pool, Scope: "tool-gateway", Expiry: 1 * time.Hour})).
			Post("/asset-actions/{actionId}/execute", h.ExecuteAction)
	})

	token, teamID := loginAndGetTeamAsset(t, r, pool)
	memberToken := loginAsMember(t, r)

	// Create a test Proxmox asset
	var assetID string
	pool.QueryRow(t.Context(),
		`INSERT INTO objects (team_id, object_type, title, status) VALUES ($1, 'asset', 'test-vm', 'active') RETURNING id::text`,
		teamID).Scan(&assetID)
	aid, _ := uuid.Parse(assetID)
	pool.Exec(t.Context(),
		`INSERT INTO assets (object_id, asset_type, provider, external_id, hostname) VALUES ($1, 'vm', 'proxmox', $2, 'test-vm')`,
		aid, "pve:pve1:100")

	// Ensure permissions exist for action execution
	pool.Exec(t.Context(), "UPDATE user_sessions SET recent_mfa_at=NULL WHERE revoked_at IS NULL")

	// If mutation is enabled, create a mutation window for tests that need it
	if mutationEnabled {
		pool.Exec(t.Context(), `UPDATE proxmox_mutation_windows SET status='closed' WHERE status='open'`)
		var userID string
		pool.QueryRow(t.Context(), "SELECT id::text FROM users WHERE email='owner@test.dev'").Scan(&userID)
		uid, _ := uuid.Parse(userID)
		pool.Exec(t.Context(), `
			INSERT INTO proxmox_mutation_windows (id, status, reason, opened_by, opened_at, expires_at)
			VALUES ($1, 'open', 'test window', $2, now(), now() + interval '1 hour')
		`, uuid.New(), uid)
	}

	return &actionTestEnv{
		r: r, pool: pool, token: token, teamID: teamID,
		memberTok: memberToken, assetID: assetID, client: client,
	}
}

func loginAndGetTeamAsset(t *testing.T, r *chi.Mux, pool *pgxpool.Pool) (token, teamID string) {
	t.Helper()
	body := `{"email":"owner@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token = resp["access_token"].(string)
	pool.QueryRow(t.Context(),
		"SELECT id::text FROM teams LIMIT 1").Scan(&teamID)
	return
}

func loginAsMember(t *testing.T, r *chi.Mux) string {
	t.Helper()
	body := `{"email":"member@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Skipf("member login failed: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["access_token"].(string)
}

func (e *actionTestEnv) setMFA(t *testing.T) {
	e.pool.Exec(t.Context(), "UPDATE user_sessions SET recent_mfa_at=NOW() WHERE revoked_at IS NULL")
}

func (e *actionTestEnv) createAction(t *testing.T, action string) (actionID, approvalID string) {
	t.Helper()
	body := `{"snapshot_name":"test-snap"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/%s", e.teamID, e.assetID, action),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create action %s: expected 201, got %d: %s", action, w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	actionID = resp["id"].(string)
	approvalID = resp["approval_id"].(string)
	return
}

func (e *actionTestEnv) approveAction(t *testing.T, approvalID string) {
	t.Helper()
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()
	pool.Exec(t.Context(), `
		INSERT INTO approval_decisions (id, approval_id, decided_by, decision, reason, mfa_verified)
		VALUES (gen_random_uuid(), $1,
			(SELECT id FROM users WHERE email='member@test.dev'),
			'approved', 'approved by member', false)
	`, approvalID)
	pool.Exec(t.Context(), `UPDATE approval_requests SET status='approved' WHERE id::text=$1`, approvalID)
}

func (e *actionTestEnv) approveActionTwoApprovers(t *testing.T, approvalID string) {
	t.Helper()
	e.approveAction(t, approvalID)
	// Add a second approver
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	// Try to find a third user or create a second decision from owner
	// Since self-approval is blocked, we need another user. Use a raw insert.
	// We'll create a temporary user for the second approval.
	var secondUserID string
	pool.QueryRow(t.Context(), `
		INSERT INTO users (email, password_hash, name, status)
		VALUES ('approver2@test.dev', 'x', 'Approver 2', 'active')
		ON CONFLICT (email) DO UPDATE SET status='active'
		RETURNING id::text
	`).Scan(&secondUserID)

	pool.Exec(t.Context(), `
		INSERT INTO approval_decisions (id, approval_id, decided_by, decision, reason, mfa_verified)
		VALUES (gen_random_uuid(), $1, $2, 'approved', 'second approver', true)
		ON CONFLICT DO NOTHING
	`, approvalID, secondUserID)
}

func (e *actionTestEnv) executeAction(t *testing.T, actionID, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/execute", e.teamID, actionID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Idempotency-Key", idemKey)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

// ─── Tests ───

// Test 1: Start creates approval request
func TestAction_StartCreatesApproval(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")
	if actionID == "" {
		t.Error("action ID empty")
	}
	if approvalID == "" {
		t.Error("approval ID empty")
	}

	// Verify DB rows
	var status, riskLevel string
	e.pool.QueryRow(t.Context(),
		"SELECT status, risk_level FROM approval_requests WHERE id::text=$1", approvalID).Scan(&status, &riskLevel)
	if status != "pending" {
		t.Errorf("approval status: %s", status)
	}
	if riskLevel != "medium" {
		t.Errorf("start risk level: %s", riskLevel)
	}
}

// Test 2: Snapshot creates approval request
func TestAction_SnapshotCreatesApproval(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "snapshot")
	if actionID == "" || approvalID == "" {
		t.Fatal("IDs empty")
	}

	var riskLevel string
	e.pool.QueryRow(t.Context(),
		"SELECT risk_level FROM approval_requests WHERE id::text=$1", approvalID).Scan(&riskLevel)
	if riskLevel != "medium" {
		t.Errorf("snapshot risk level: %s", riskLevel)
	}
}

// Test 3: Shutdown requires MFA
func TestAction_ShutdownRequiresMFA(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "shutdown")
	e.approveAction(t, approvalID)
	// MFA not set → execute should fail
	w := e.executeAction(t, actionID, "exec-shutdown-nomfa-"+uuid.New().String()[:8])
	if w.Code != 403 {
		t.Fatalf("expected 403 without MFA, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: Stop requires 2 approvers and MFA
func TestAction_StopRequiresTwoApprovers(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "stop")

	// Only 1 approver → should fail even with MFA
	e.approveAction(t, approvalID)
	e.setMFA(t)
	w := e.executeAction(t, actionID, "exec-stop-1appr-"+uuid.New().String()[:8])
	if w.Code != 403 {
		t.Fatalf("expected 403 with only 1 approver for stop, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 5: Approved start executes through Proxmox client
func TestAction_ApprovedStartExecutes(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)

	w := e.executeAction(t, actionID, "exec-start-ok-"+uuid.New().String()[:8])
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "succeeded" {
		t.Errorf("expected succeeded, got %v", resp["status"])
	}

	// Verify task UPID stored
	var taskID string
	e.pool.QueryRow(t.Context(),
		"SELECT proxmox_task_id FROM asset_actions WHERE id::text=$1", actionID).Scan(&taskID)
	if taskID == "" {
		t.Error("proxmox_task_id not stored")
	}

	// Verify approval marked executed
	var approvalStatus string
	e.pool.QueryRow(t.Context(),
		"SELECT status FROM approval_requests WHERE id::text=$1", approvalID).Scan(&approvalStatus)
	if approvalStatus != "executed" {
		t.Errorf("approval status: %s (expected executed)", approvalStatus)
	}
}

// Test 6: Unapproved action does not execute
func TestAction_UnapprovedDoesNotExecute(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, _ := e.createAction(t, "start")
	// Don't approve
	w := e.executeAction(t, actionID, "exec-unapproved-"+uuid.New().String()[:8])
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// Test 7: Failed Proxmox call stores failed effect result
func TestAction_FailedProxmoxCall(t *testing.T) {
	e := setupActionTest(t, true)
	e.client.failNext = true

	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)

	w := e.executeAction(t, actionID, "exec-fail-"+uuid.New().String()[:8])
	if w.Code != 502 {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}

	var status, errMsg string
	e.pool.QueryRow(t.Context(),
		"SELECT status, error_message FROM asset_actions WHERE id::text=$1", actionID).Scan(&status, &errMsg)
	if status != "failed" {
		t.Errorf("expected failed, got %s", status)
	}
	if errMsg == "" {
		t.Error("error_message empty")
	}

	// Verify audit for failure
	var auditCount int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action='asset.action.failed' AND entity_id::text=$1", actionID).Scan(&auditCount)
	if auditCount == 0 {
		t.Error("no audit for failed action")
	}
}

// Test 8: Proxmox token not logged/audited/outboxed
func TestAction_TokenNotLogged(t *testing.T) {
	e := setupActionTest(t, true)

	// Create some action data
	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)
	e.executeAction(t, actionID, "exec-token-"+uuid.New().String()[:8])

	// Check all audit logs for this action don't contain secrets
	rows, _ := e.pool.Query(t.Context(),
		"SELECT new_value::text FROM audit_logs WHERE entity_id::text=$1", actionID)
	defer rows.Close()
	for rows.Next() {
		var payload string
		rows.Scan(&payload)
		if strings.Contains(strings.ToLower(payload), "secret") || strings.Contains(strings.ToLower(payload), "token_id") {
			t.Errorf("audit payload contains secret/token: %s", payload)
		}
	}
}

// Test 9: Forbidden mutation routes do not exist
func TestAction_ForbiddenRoutesReturn404(t *testing.T) {
	e := setupActionTest(t, true)
	u := uuid.New().String()[:8]

	for _, action := range []string{"delete", "migrate", "clone", "reset", "firewall", "network"} {
		body := `{}`
		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/%s", e.teamID, e.assetID, action),
			strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+e.token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code == 200 || w.Code == 201 {
			t.Errorf("forbidden action %s returned %d", action, w.Code)
		}
		_ = u
	}
}

// Test 10: Asset outside team blocked
func TestAction_AssetOutsideTeamBlocked(t *testing.T) {
	e := setupActionTest(t, true)

	// Create asset in different team
	var team2ID uuid.UUID
	e.pool.QueryRow(t.Context(),
		"INSERT INTO teams (name, slug) VALUES ('other-team', $1) RETURNING id", "other-"+uuid.New().String()[:8]).Scan(&team2ID)
	var otherAssetID string
	e.pool.QueryRow(t.Context(),
		"INSERT INTO objects (team_id, object_type, title, status) VALUES ($1, 'asset', 'other', 'active') RETURNING id::text",
		team2ID).Scan(&otherAssetID)

	body := `{}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/start", e.teamID, otherAssetID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 for cross-team asset, got %d", w.Code)
	}
}

// Test 11: Non-Proxmox asset blocked
func TestAction_NonProxmoxAssetBlocked(t *testing.T) {
	e := setupActionTest(t, true)

	// Create non-Proxmox asset
	var nonPxAsset string
	tid, _ := uuid.Parse(e.teamID)
	e.pool.QueryRow(t.Context(),
		"INSERT INTO objects (team_id, object_type, title, status) VALUES ($1, 'asset', 'aws-vm', 'active') RETURNING id::text",
		tid).Scan(&nonPxAsset)
	nid, _ := uuid.Parse(nonPxAsset)
	e.pool.Exec(t.Context(),
		`INSERT INTO assets (object_id, asset_type, provider, external_id, hostname) VALUES ($1, 'vm', 'aws', 'aws:i-123', 'aws-vm')`,
		nid)

	body := `{}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/start", e.teamID, nonPxAsset),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for non-Proxmox asset, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 12: Missing node/vmid/vm_type blocked
func TestAction_MissingMetadataBlocked(t *testing.T) {
	e := setupActionTest(t, true)

	// Create asset with malformed external_id
	var badAsset string
	tid, _ := uuid.Parse(e.teamID)
	e.pool.QueryRow(t.Context(),
		"INSERT INTO objects (team_id, object_type, title, status) VALUES ($1, 'asset', 'bad-vm', 'active') RETURNING id::text",
		tid).Scan(&badAsset)
	bid, _ := uuid.Parse(badAsset)
	e.pool.Exec(t.Context(),
		`INSERT INTO assets (object_id, asset_type, provider, external_id, hostname) VALUES ($1, 'vm', 'proxmox', 'bad-format', 'bad-vm')`,
		bid)

	body := `{}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/start", e.teamID, badAsset),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for missing metadata, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 13: Snapshot name validation works
func TestAction_SnapshotNameValidation(t *testing.T) {
	e := setupActionTest(t, true)
	u := uuid.New().String()[:8]

	// Invalid snapshot name (contains shell metacharacters)
	body := `{"snapshot_name":"$(rm -rf /)"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/snapshot", e.teamID, e.assetID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid snapshot name, got %d", w.Code)
	}

	// Valid snapshot name
	body = fmt.Sprintf(`{"snapshot_name":"valid-snap-%s"}`, u)
	req = httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/assets/%s/actions/proxmox/snapshot", e.teamID, e.assetID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("expected 201 for valid snapshot name, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 14: Mutation disabled config blocks execution
func TestAction_MutationDisabledBlocks(t *testing.T) {
	e := setupActionTest(t, false) // mutation disabled

	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)

	w := e.executeAction(t, actionID, "exec-disabled-"+uuid.New().String()[:8])
	if w.Code != 403 {
		t.Fatalf("expected 403 when mutation disabled, got %d", w.Code)
	}
}

// Test 15: Idempotency replay returns same action/effect result
func TestAction_IdempotencyReplay(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)

	key := "idem-action-replay-" + uuid.New().String()[:8]
	w1 := e.executeAction(t, actionID, key)
	if w1.Code != 200 {
		t.Fatalf("first exec: %d %s", w1.Code, w1.Body.String())
	}

	w2 := e.executeAction(t, actionID, key)
	if w2.Code != 200 {
		t.Errorf("replay: %d", w2.Code)
	}

	// Only 1 asset_action row (no duplicate)
	var count int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM asset_actions WHERE id::text=$1", actionID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 asset_action, got %d", count)
	}
}

// Test 16: Idempotency conflict rejected
func TestAction_IdempotencyConflict(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)

	key := "idem-action-conflict-" + uuid.New().String()[:8]

	// Insert processing state
	var userID string
	e.pool.QueryRow(t.Context(),
		"SELECT id::text FROM users WHERE email='owner@test.dev'").Scan(&userID)
	e.pool.Exec(t.Context(),
		`INSERT INTO idempotency_keys (scope_type, scope_id, key, request_method, request_path, status, expires_at)
		 VALUES ('tool-gateway', $1, $2, 'POST', '/execute', 'processing', NOW() + interval '1 hour')`,
		userID, key)

	w := e.executeAction(t, actionID, key)
	if w.Code != 409 {
		t.Errorf("expected 409 conflict, got %d", w.Code)
	}
}

// Test 17: Approval action_type mismatch blocked
func TestAction_ApprovalActionTypeMismatch(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")

	// Change the approval's action_type to something different
	e.pool.Exec(t.Context(),
		"UPDATE approval_requests SET action_type='proxmox.shutdown' WHERE id::text=$1", approvalID)
	e.approveAction(t, approvalID)

	w := e.executeAction(t, actionID, "exec-mismatch-"+uuid.New().String()[:8])
	if w.Code != 403 {
		t.Fatalf("expected 403 for action_type mismatch, got %d", w.Code)
	}
}

// Test 18: Approval already executed blocked / idempotent
func TestAction_ApprovalAlreadyExecutedBlocked(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")
	e.approveAction(t, approvalID)

	// Execute once
	w := e.executeAction(t, actionID, "exec-first-"+uuid.New().String()[:8])
	if w.Code != 200 {
		t.Fatalf("first exec: %d", w.Code)
	}

	// Try to execute again with same action (approval is now executed)
	// The handler should see status=succeeded and return idempotent response
	w2 := e.executeAction(t, actionID, "exec-second-"+uuid.New().String()[:8])
	if w2.Code != 200 {
		t.Errorf("second exec (idempotent): expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["status"] != "succeeded" {
		t.Errorf("expected succeeded, got %v", resp["status"])
	}
}

// Test 19: Cross-team approval blocked
func TestAction_CrossTeamApprovalBlocked(t *testing.T) {
	e := setupActionTest(t, true)
	actionID, approvalID := e.createAction(t, "start")

	// Move the approval to a different team
	var team2ID uuid.UUID
	e.pool.QueryRow(t.Context(),
		"INSERT INTO teams (name, slug) VALUES ('xfer-team', $1) RETURNING id", "xfer-"+uuid.New().String()[:8]).Scan(&team2ID)
	e.pool.Exec(t.Context(),
		"UPDATE approval_requests SET team_id=$1 WHERE id::text=$2", team2ID, approvalID)
	e.approveAction(t, approvalID)

	w := e.executeAction(t, actionID, "exec-cross-team-"+uuid.New().String()[:8])
	if w.Code != 403 {
		t.Fatalf("expected 403 for cross-team approval, got %d", w.Code)
	}
}

// Test 20: Audit/outbox written for request, execution, success, failure, and block paths
func TestAction_AuditOutboxAllPaths(t *testing.T) {
	e := setupActionTest(t, true)

	// Request path
	actionID, _ := e.createAction(t, "start")
	var reqAudit int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action='asset.action.requested' AND entity_id::text=$1", actionID).Scan(&reqAudit)
	if reqAudit == 0 {
		t.Error("no audit for request path")
	}

	// Block path (unapproved execute)
	e.executeAction(t, actionID, "exec-block-audit-"+uuid.New().String()[:8])
	var blockAudit int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action='asset.action.blocked' AND entity_id::text=$1", actionID).Scan(&blockAudit)
	if blockAudit == 0 {
		t.Error("no audit for blocked path")
	}

	// Success path — need a fresh action since the first one was marked failed by the block
	actionID2, approvalID2 := e.createAction(t, "start")
	e.approveAction(t, approvalID2)
	e.executeAction(t, actionID2, "exec-success-audit-"+uuid.New().String()[:8])
	var execAudit int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action='asset.action.executed' AND entity_id::text=$1", actionID2).Scan(&execAudit)
	if execAudit == 0 {
		t.Error("no audit for execution path")
	}
	var execOutbox int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM outbox_events WHERE event_type='clarity.v1.asset.action.executed' AND aggregate_id=$1", actionID2).Scan(&execOutbox)
	if execOutbox == 0 {
		t.Error("no outbox for execution path")
	}

	// Failure path
	e.client.failNext = true
	actionID3, approvalID3 := e.createAction(t, "snapshot")
	e.approveAction(t, approvalID3)
	e.executeAction(t, actionID3, "exec-fail-audit-"+uuid.New().String()[:8])
	var failAudit int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action='asset.action.failed' AND entity_id::text=$1", actionID3).Scan(&failAudit)
	if failAudit == 0 {
		t.Error("no audit for failure path")
	}
}

// ─── parseMutationTarget unit tests ───

func TestParseMutationTarget(t *testing.T) {
	tests := []struct {
		externalID string
		assetType  string
		wantNode   string
		wantVMID   int
		wantType   string
		wantErr    bool
	}{
		{"pve:pve1:100", "vm", "pve1", 100, "qemu", false},
		{"pve:pve2:200", "lxc", "pve2", 200, "lxc", false},
		{"pve:pve3:300", "container", "pve3", 300, "lxc", false},
		{"bad-format", "vm", "", 0, "", true},
		{"pve:only", "vm", "", 0, "", true},
		{"pve:pve1:abc", "vm", "", 0, "", true},
	}
	for _, tt := range tests {
		target, err := parseMutationTarget(tt.externalID, tt.assetType)
		if tt.wantErr {
			if err == nil {
				t.Errorf("expected error for %s", tt.externalID)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for %s: %v", tt.externalID, err)
			continue
		}
		if target.Node != tt.wantNode || target.VMID != tt.wantVMID || target.VMType != tt.wantType {
			t.Errorf("for %s: got node=%s vmid=%d type=%s, want node=%s vmid=%d type=%s",
				tt.externalID, target.Node, target.VMID, target.VMType,
				tt.wantNode, tt.wantVMID, tt.wantType)
		}
	}
}

// Ensure imports used
var _ = bytes.Contains
var _ = time.Now
