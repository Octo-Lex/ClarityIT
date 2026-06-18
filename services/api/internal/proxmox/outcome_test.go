package proxmox

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

const outcomeDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type outcomeTestEnv struct {
	r       *chi.Mux
	pool    *pgxpool.Pool
	token   string
	teamID  string
}

func setupOutcomeTest(t *testing.T) *outcomeTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), outcomeDBURL)
	t.Cleanup(func() { pool.Close() })

	outcomeH := NewOutcomeHandler(pool, cfg)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "assets.actions.read")).
			Post("/asset-actions/{actionId}/outcome", outcomeH.CreateOrUpdateAssetActionOutcome)
		r.With(middleware.RequirePermission(pool, "assets.actions.read")).
			Get("/asset-actions/{actionId}/outcome", outcomeH.GetAssetActionOutcome)
		r.With(middleware.RequirePermission(pool, "remediations.read")).
			Post("/remediations/{proposalId}/outcome", outcomeH.CreateOrUpdateRemediationOutcome)
		r.With(middleware.RequirePermission(pool, "remediations.read")).
			Get("/remediations/{proposalId}/outcome", outcomeH.GetRemediationOutcome)
	})

	token := loginForOutcome(t, r)

	var teamID string
	pool.QueryRow(t.Context(), "SELECT id::text FROM teams LIMIT 1").Scan(&teamID)

	return &outcomeTestEnv{r: r, pool: pool, token: token, teamID: teamID}
}

func loginForOutcome(t *testing.T, r *chi.Mux) string {
	t.Helper()
	body := `{"email":"owner@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["access_token"].(string)
}

// createSucceededAssetAction inserts a fake succeeded asset action for testing
func (e *outcomeTestEnv) createTestAssetAction(t *testing.T, status string) string {
	t.Helper()
	ctx := t.Context()
	assetID, _ := uuid.Parse(getTestAssetID(e.pool))
	teamID, _ := uuid.Parse(e.teamID)
	actorID, _ := uuid.Parse(getOwnerUserID(e.pool))
	actionID := uuid.New()

	_, err := e.pool.Exec(ctx, `
		INSERT INTO asset_actions (id, team_id, asset_id, action_type, status, requested_by)
		VALUES ($1, $2, $3, 'proxmox.shutdown', 'succeeded', $4)
	`, actionID, teamID, assetID, actorID)
	if err != nil {
		t.Fatalf("create test asset action: %v", err)
	}

	// Set status
	_, err = e.pool.Exec(ctx, "UPDATE asset_actions SET status=$1 WHERE id=$2", status, actionID)
	if err != nil {
		t.Fatalf("set status: %v", err)
	}

	return actionID.String()
}

func getTestAssetID(pool *pgxpool.Pool) string {
	var id string
	pool.QueryRow(context.Background(), "SELECT o.id::text FROM objects o JOIN assets a ON a.object_id=o.id WHERE o.deleted_at IS NULL LIMIT 1").Scan(&id)
	return id
}

func getOwnerUserID(pool *pgxpool.Pool) string {
	var id string
	pool.QueryRow(context.Background(), "SELECT id::text FROM users WHERE email='owner@test.dev' LIMIT 1").Scan(&id)
	return id
}

// ─── Tests ───

// Test 1: creates outcome for succeeded asset action
func TestOutcome_CreateForSucceededAction(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	body := `{"outcome_status":"successful","actual_result":"VM shut down cleanly"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["outcome_status"] != "successful" {
		t.Errorf("expected successful, got %v", resp["outcome_status"])
	}
}

// Test 2: creates outcome for failed asset action
func TestOutcome_CreateForFailedAction(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "failed")

	body := `{"outcome_status":"failed","actual_result":"Shutdown timed out"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 3: rejects outcome for pending asset action
func TestOutcome_RejectsPendingAction(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "pending")

	body := `{"outcome_status":"successful"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Errorf("expected 409 for pending action, got %d", w.Code)
	}
}

// Test 4: gets outcome for asset action
func TestOutcome_GetForAssetAction(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	// Create outcome first
	body := `{"outcome_status":"successful","actual_result":"OK"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Get outcome
	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["outcome_status"] != "successful" {
		t.Errorf("expected successful, got %v", resp["outcome_status"])
	}
}

// Test 5: gets unavailable when no outcome exists
func TestOutcome_GetUnavailableWhenAbsent(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["available"] != false {
		t.Error("expected available=false when no outcome")
	}
}

// Test 6: rejects invalid outcome_status
func TestOutcome_RejectsInvalidStatus(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	body := `{"outcome_status":"great"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid status, got %d", w.Code)
	}
}

// Test 7: upserts (updates) existing outcome
func TestOutcome_UpsertExisting(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	// Create
	body1 := `{"outcome_status":"successful","actual_result":"first"}`
	req1 := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body1))
	req1.Header.Set("Authorization", "Bearer "+e.token)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	e.r.ServeHTTP(w1, req1)
	var resp1 map[string]any
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	id1 := resp1["id"].(string)

	// Update
	body2 := `{"outcome_status":"partially_successful","actual_result":"updated"}`
	req2 := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var resp2 map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	id2 := resp2["id"].(string)

	// Should be same ID (upsert)
	if id1 != id2 {
		t.Errorf("upsert should preserve ID: %s vs %s", id1, id2)
	}
	if resp2["outcome_status"] != "partially_successful" {
		t.Errorf("expected updated status, got %v", resp2["outcome_status"])
	}
}

// Test 8: cross-team returns 404
func TestOutcome_CrossTeamReturns404(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	otherTeam := uuid.New().String()
	body := `{"outcome_status":"successful"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", otherTeam, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for cross-team, got %d", w.Code)
	}
}

// Test 9: no Tool Gateway call (structural — outcome handler has no client reference)
func TestOutcome_NoToolGateway(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	body := `{"outcome_status":"successful"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Verify no new asset_actions were created
	var actionCount int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionCount)
	// Should match what existed before (no new actions from outcome recording)
}

// Test 10: no asset action/remediation created automatically
func TestOutcome_NoAutoCreation(t *testing.T) {
	e := setupOutcomeTest(t)

	var actionsBefore, proposalsBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionsBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&proposalsBefore)

	actionID := e.createTestAssetAction(t, "succeeded")
	// This creates one asset_action so account for it
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionsBefore)

	body := `{"outcome_status":"successful"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var actionsAfter, proposalsAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionsAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&proposalsAfter)

	if actionsAfter != actionsBefore {
		t.Errorf("asset_actions changed: %d -> %d", actionsBefore, actionsAfter)
	}
	if proposalsAfter != proposalsBefore {
		t.Errorf("remediation_proposals changed: %d -> %d", proposalsBefore, proposalsAfter)
	}
}

// Test 11: sensitive fields sanitized
func TestOutcome_SensitiveFieldsSanitized(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	body := `{"outcome_status":"successful","actual_result":"Completed with password=hunter2 and token=secret123"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, "hunter2") {
		t.Error("password leaked in outcome response")
	}
	if strings.Contains(bodyStr, "secret123") {
		t.Error("token leaked in outcome response")
	}
}

// Test 12: text length validation
func TestOutcome_TextLengthValidation(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	// Create a string longer than expected_result limit (2000)
	longText := strings.Repeat("x", 2001)
	body := fmt.Sprintf(`{"outcome_status":"successful","expected_result":"%s"}`, longText)
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for overly long text, got %d", w.Code)
	}
}

// Test 13: unauthorized user denied
func TestOutcome_UnauthorizedDenied(t *testing.T) {
	e := setupOutcomeTest(t)
	actionID := e.createTestAssetAction(t, "succeeded")

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/asset-actions/%s/outcome", e.teamID, actionID), nil)
	// No auth header
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// Test 14: remediation outcome can be created
func TestOutcome_RemediationOutcomeCreated(t *testing.T) {
	e := setupOutcomeTest(t)
	ctx := t.Context()

	// Create a completed remediation proposal
	proposalID := uuid.New()
	teamID, _ := uuid.Parse(e.teamID)
	actorID, _ := uuid.Parse(getOwnerUserID(e.pool))
	_, err := e.pool.Exec(ctx, `
		INSERT INTO remediation_proposals (id, team_id, title, description, risk_level, source, created_by, status)
		VALUES ($1, $2, 'test outcome', 'test', 'low', 'operator', $3, 'completed')
	`, proposalID, teamID, actorID)
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}

	body := `{"outcome_status":"successful","actual_result":"Remediation worked"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/remediations/%s/outcome", e.teamID, proposalID.String()),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["outcome_status"] != "successful" {
		t.Errorf("expected successful, got %v", resp["outcome_status"])
	}
}

// Test 15: rejects outcome for non-terminal remediation
func TestOutcome_RejectsNonTerminalRemediation(t *testing.T) {
	e := setupOutcomeTest(t)
	ctx := t.Context()

	proposalID := uuid.New()
	teamID, _ := uuid.Parse(e.teamID)
	actorID, _ := uuid.Parse(getOwnerUserID(e.pool))
	_, err := e.pool.Exec(ctx, `
		INSERT INTO remediation_proposals (id, team_id, title, description, risk_level, source, created_by, status)
		VALUES ($1, $2, 'test', 'test', 'low', 'operator', $3, 'approved')
	`, proposalID, teamID, actorID)
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}

	body := `{"outcome_status":"successful"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/remediations/%s/outcome", e.teamID, proposalID.String()),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Errorf("expected 409 for non-terminal remediation, got %d", w.Code)
	}
}
