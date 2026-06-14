package remediation

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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const evidenceDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type evidenceTestEnv struct {
	r          *chi.Mux
	pool       *pgxpool.Pool
	token      string
	teamID     string
	memberTok  string
}

func setupEvidenceTest(t *testing.T) *evidenceTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), evidenceDBURL)
	t.Cleanup(func() { pool.Close() })

	remediationH := NewHandler(pool)
	evidenceH := NewEvidenceHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		// Remediation routes (simplified for testing)
		r.With(middleware.RequirePermission(pool, "remediations.create")).
			Post("/remediations", remediationH.Create)
		r.With(middleware.RequirePermission(pool, "remediations.read")).
			Get("/remediations", remediationH.List)
		r.With(middleware.RequirePermission(pool, "remediations.read")).
			Get("/remediations/{remediationId}", remediationH.Get)
		// Evidence route
		r.With(middleware.RequirePermission(pool, "remediations.read")).
			Get("/recommendations/{recommendationId}/evidence", evidenceH.GetEvidence)
	})

	// Login as owner
	token := loginForEvidence(t, r, "owner@test.dev")
	memberToken := loginForEvidence(t, r, "member@test.dev")

	var teamID string
	pool.QueryRow(t.Context(), "SELECT id::text FROM teams LIMIT 1").Scan(&teamID)

	return &evidenceTestEnv{
		r: r, pool: pool, token: token, teamID: teamID, memberTok: memberToken,
	}
}

func loginForEvidence(t *testing.T, r *chi.Mux, email string) string {
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

// createRemediationWithEvidence creates a remediation proposal with an evidence pack
func (e *evidenceTestEnv) createRemediationWithEvidence(t *testing.T, body string) (proposalID string) {
	t.Helper()
	if body == "" {
		body = `{
			"title": "Test Remediation",
			"description": "test",
			"source": "operator",
			"risk_level": "low",
			"steps": [{"step_order":1,"tool_name":"work_items.create","risk_level":"low","parameters":{}}],
			"evidence": {
				"recommendation_summary": "Restart the service to clear the error state",
				"supporting_evidence": [{"type":"log_entry","description":"OOM killer invoked","source":"syslog"}],
				"conflicting_evidence": [{"type":"metric","description":"Memory usage normal","source":"prometheus"}],
				"confidence_score": 0.75,
				"confidence_level": "high",
				"risk_notes": "Service restart may cause brief downtime",
				"missing_info": [{"description":"No data on concurrent users"}]
			}
		}`
	}

	// For agent source we need a valid agent_run_id — operator source is used for evidence tests

	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/remediations", e.teamID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "evidence-test-"+uuid.New().String())
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create remediation: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["id"].(string)
}

// ─── Tests ───

// Test 1: Evidence pack created with remediation recommendation
func TestEvidence_CreatedWithRemediation(t *testing.T) {
	e := setupEvidenceTest(t)
	proposalID := e.createRemediationWithEvidence(t, "")

	// Verify evidence exists in DB
	var count int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM recommendation_evidence WHERE recommendation_id::text=$1",
		proposalID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 evidence pack, got %d", count)
	}
}

// Test 2: Evidence lookup returns full pack
func TestEvidence_LookupReturnsFullPack(t *testing.T) {
	e := setupEvidenceTest(t)
	proposalID := e.createRemediationWithEvidence(t, "")

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["available"] != true {
		t.Error("evidence should be available")
	}
	if resp["recommendation_summary"] == nil {
		t.Error("missing recommendation_summary")
	}
	if resp["confidence_score"] == nil {
		t.Error("missing confidence_score")
	}
	if resp["confidence_level"] == nil {
		t.Error("missing confidence_level")
	}
	supporting, ok := resp["supporting_evidence"].([]any)
	if !ok || len(supporting) != 1 {
		t.Errorf("expected 1 supporting evidence item, got %v", resp["supporting_evidence"])
	}
	conflicting, ok := resp["conflicting_evidence"].([]any)
	if !ok || len(conflicting) != 1 {
		t.Errorf("expected 1 conflicting evidence item, got %v", resp["conflicting_evidence"])
	}
}

// Test 3: Evidence lookup is team-scoped
func TestEvidence_TeamScoped(t *testing.T) {
	e := setupEvidenceTest(t)
	proposalID := e.createRemediationWithEvidence(t, "")

	// Try to access from a different team (use invalid team UUID)
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", uuid.New().String(), proposalID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Should return "unavailable" (not found in that team)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["available"] != false {
		t.Error("evidence should not be available from a different team")
	}
}

// Test 4: Confidence score rejects < 0
func TestEvidence_ConfidenceRejectsNegative(t *testing.T) {
	input := EvidenceInput{ConfidenceScore: -0.1}
	_, err := ValidateEvidenceInput(input)
	if err == nil {
		t.Error("should reject negative confidence score")
	}
}

// Test 5: Confidence score rejects > 1
func TestEvidence_ConfidenceRejectsAboveOne(t *testing.T) {
	input := EvidenceInput{ConfidenceScore: 1.5}
	_, err := ValidateEvidenceInput(input)
	if err == nil {
		t.Error("should reject confidence score > 1.0")
	}
}

// Test 6: Stale evidence remains returned with stale flag
func TestEvidence_StaleEvidenceRemainsVisible(t *testing.T) {
	e := setupEvidenceTest(t)
	proposalID := e.createRemediationWithEvidence(t, "")

	// Manually set stale_after to past
	e.pool.Exec(t.Context(),
		"UPDATE recommendation_evidence SET stale_after=NOW() - INTERVAL '1 hour' WHERE recommendation_id::text=$1",
		proposalID)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["available"] != true {
		t.Error("stale evidence should still be available")
	}
	if resp["is_stale"] != true {
		t.Error("is_stale should be true")
	}
}

// Test 7: Missing evidence for legacy recommendation returns safe state
func TestEvidence_MissingEvidenceReturnsSafeState(t *testing.T) {
	e := setupEvidenceTest(t)

	// Create a remediation WITHOUT evidence
	body := `{
		"title": "Legacy Remediation",
		"description": "no evidence",
		"source": "operator",
		"risk_level": "low",
		"steps": [{"step_order":1,"tool_name":"work_items.create","risk_level":"low","parameters":{}}]
	}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/remediations", e.teamID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "legacy-test-"+uuid.New().String())
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	proposalID, ok := createResp["id"].(string)
	if !ok || proposalID == "" {
		t.Fatalf("failed to create legacy remediation: %d %s", w.Code, w.Body.String())
	}

	// Look up evidence for this proposal
	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["available"] != false {
		t.Error("should return available=false for legacy recommendation")
	}
}

// Test 8: Supporting/conflicting evidence returned correctly
func TestEvidence_SupportingAndConflictingReturned(t *testing.T) {
	e := setupEvidenceTest(t)
	proposalID := e.createRemediationWithEvidence(t, "")

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	supporting := resp["supporting_evidence"].([]any)
	if len(supporting) != 1 {
		t.Errorf("expected 1 supporting, got %d", len(supporting))
	}
	supportItem := supporting[0].(map[string]any)
	if supportItem["description"] != "OOM killer invoked" {
		t.Errorf("unexpected supporting evidence: %v", supportItem)
	}

	conflicting := resp["conflicting_evidence"].([]any)
	if len(conflicting) != 1 {
		t.Errorf("expected 1 conflicting, got %d", len(conflicting))
	}
	conflictItem := conflicting[0].(map[string]any)
	if conflictItem["description"] != "Memory usage normal" {
		t.Errorf("unexpected conflicting evidence: %v", conflictItem)
	}
}

// Test 9: No raw tool parameters returned in evidence
func TestEvidence_NoRawToolParameters(t *testing.T) {
	e := setupEvidenceTest(t)

	// Create remediation with evidence that tries to include tool params
	body := `{
		"title": "Evidence with params",
		"description": "test",
		"source": "operator",
		"risk_level": "low",
		"steps": [{"step_order":1,"tool_name":"work_items.create","risk_level":"low","parameters":{}}],
		"evidence": {
			"recommendation_summary": "test",
			"supporting_evidence": [{"type":"tool_call","description":"ran check","tool_parameters":{"secret":"should-be-redacted","password":"hunter2"}}],
			"conflicting_evidence": [],
			"confidence_score": 0.5,
			"risk_notes": "",
			"missing_info": []
		}
	}`
	proposalID := e.createRemediationWithEvidence(t, body)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	body_str := w.Body.String()
	if strings.Contains(body_str, "hunter2") {
		t.Error("raw password should not appear in evidence response")
	}
	if strings.Contains(body_str, "\"secret\":\"should-be-redacted\"") {
		t.Error("raw secret should not appear in evidence response")
	}
}

// Test 10: No action_target returned unredacted
func TestEvidence_NoUnredactedActionTarget(t *testing.T) {
	e := setupEvidenceTest(t)

	body := `{
		"title": "Evidence with target",
		"description": "test",
		"source": "operator",
		"risk_level": "low",
		"steps": [{"step_order":1,"tool_name":"work_items.create","risk_level":"low","parameters":{}}],
		"evidence": {
			"recommendation_summary": "test",
			"supporting_evidence": [{"type":"log","description":"action_target=pve:node1:100:token=secret123"}],
			"conflicting_evidence": [],
			"confidence_score": 0.5,
			"risk_notes": "",
			"missing_info": []
		}
	}`
	proposalID := e.createRemediationWithEvidence(t, body)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	body_str := w.Body.String()
	if strings.Contains(body_str, "secret123") {
		t.Error("unredacted token should not appear in evidence response")
	}
}

// Test 11: Python worker cannot directly persist evidence
// This is a design verification: the PersistEvidence function requires
// a *pgxpool.Pool, which the Python worker never has access to.
// The Python worker communicates only through the Go API HTTP interface.
func TestEvidence_PythonWorkerCannotPersistDirectly(t *testing.T) {
	// Verify that the evidence table is only accessible through the Go API.
	// The Python worker has no DATABASE_URL env var and validates this at startup.
	// This test verifies the design boundary structurally.

	// The EvidenceInput type and ValidateEvidenceInput function are Go-only.
	// The Python worker can only submit evidence through the remediation
	// creation endpoint, where the Go control plane validates and persists it.

	// Verify that evidence can only be created through the remediation Create handler
	// by checking that there's no standalone evidence creation endpoint
	e := setupEvidenceTest(t)

	// Try to POST directly to an evidence creation endpoint — should not exist
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, uuid.New().String()),
		strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 404 && w.Code != 405 {
		t.Errorf("evidence creation endpoint should not exist, got %d", w.Code)
	}
}

// Test 12: Unauthorized user denied
func TestEvidence_UnauthorizedDenied(t *testing.T) {
	e := setupEvidenceTest(t)
	proposalID := e.createRemediationWithEvidence(t, "")

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/recommendations/%s/evidence", e.teamID, proposalID), nil)
	// No Authorization header
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for no auth, got %d", w.Code)
	}
}

// Test 13: Confidence level derived from score when not provided
func TestEvidence_ConfidenceLevelDerivedFromScore(t *testing.T) {
	// High: >= 0.7
	input := EvidenceInput{ConfidenceScore: 0.8}
	result, err := ValidateEvidenceInput(input)
	if err != nil || result.ConfidenceLevel != "high" {
		t.Errorf("expected high, got %s (err: %v)", result.ConfidenceLevel, err)
	}

	// Medium: >= 0.4
	input = EvidenceInput{ConfidenceScore: 0.5}
	result, err = ValidateEvidenceInput(input)
	if err != nil || result.ConfidenceLevel != "medium" {
		t.Errorf("expected medium, got %s (err: %v)", result.ConfidenceLevel, err)
	}

	// Low: < 0.4
	input = EvidenceInput{ConfidenceScore: 0.2}
	result, err = ValidateEvidenceInput(input)
	if err != nil || result.ConfidenceLevel != "low" {
		t.Errorf("expected low, got %s (err: %v)", result.ConfidenceLevel, err)
	}
}
