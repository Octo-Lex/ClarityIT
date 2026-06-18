package proxmox

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

const riskDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type riskTestEnv struct {
	r       *chi.Mux
	pool    *pgxpool.Pool
	token   string
	teamID  string
	assetID string
}

func setupRiskTest(t *testing.T) *riskTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), riskDBURL)
	t.Cleanup(func() { pool.Close() })

	riskHandler := NewRiskScoreHandler(pool, cfg)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "assets.read")).
			Get("/assets/{assetId}/risk-score", riskHandler.GetRiskScore)
	})

	token := loginForRisk(t, r)

	var teamID, assetID string
	pool.QueryRow(t.Context(),
		"SELECT o.team_id::text, o.id::text FROM objects o JOIN assets a ON a.object_id=o.id WHERE o.deleted_at IS NULL LIMIT 1").Scan(&teamID, &assetID)

	return &riskTestEnv{r: r, pool: pool, token: token, teamID: teamID, assetID: assetID}
}

func loginForRisk(t *testing.T, r *chi.Mux) string {
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

func (e *riskTestEnv) getRiskScore(t *testing.T, action string) RiskScoreResponse {
	t.Helper()
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/assets/%s/risk-score?action=%s", e.teamID, e.assetID, action), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("risk score: %d %s", w.Code, w.Body.String())
	}
	var resp RiskScoreResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ─── Tests ───

// Test 1: start action returns lower risk than shutdown
func TestRiskScore_StartLowerThanShutdown(t *testing.T) {
	e := setupRiskTest(t)
	startScore := e.getRiskScore(t, "proxmox.start")
	shutdownScore := e.getRiskScore(t, "proxmox.shutdown")

	if startScore.RiskScore >= shutdownScore.RiskScore {
		t.Errorf("start (%d) should be lower than shutdown (%d)", startScore.RiskScore, shutdownScore.RiskScore)
	}
}

// Test 2: stop action can produce critical risk
func TestRiskScore_StopCanProduceCritical(t *testing.T) {
	e := setupRiskTest(t)
	stopScore := e.getRiskScore(t, "proxmox.stop")

	if stopScore.RiskLevel != "high" && stopScore.RiskLevel != "critical" {
		t.Errorf("stop should produce high or critical risk, got %s (%d)", stopScore.RiskLevel, stopScore.RiskScore)
	}
}

// Test 3: score is bounded 0-100
func TestRiskScore_BoundedZeroToHundred(t *testing.T) {
	e := setupRiskTest(t)
	actions := []string{"proxmox.start", "proxmox.shutdown", "proxmox.stop", "proxmox.snapshot"}
	for _, action := range actions {
		score := e.getRiskScore(t, action)
		if score.RiskScore < 0 || score.RiskScore > 100 {
			t.Errorf("%s score %d out of bounds [0, 100]", action, score.RiskScore)
		}
	}
}

// Test 4: risk level thresholds correct
func TestRiskScore_LevelThresholds(t *testing.T) {
	// Verify the mapping function directly
	tests := []struct {
		score    int
		expected string
	}{
		{0, "low"},
		{24, "low"},
		{25, "medium"},
		{49, "medium"},
		{50, "high"},
		{79, "high"},
		{80, "critical"},
		{100, "critical"},
	}
	for _, tc := range tests {
		level := scoreToLevel(tc.score)
		if level != tc.expected {
			t.Errorf("score %d: expected %s, got %s", tc.score, tc.expected, level)
		}
	}
}

// Test 5: factors include all required types
func TestRiskScore_AllFactorsPresent(t *testing.T) {
	e := setupRiskTest(t)
	score := e.getRiskScore(t, "proxmox.shutdown")

	factorNames := make(map[string]bool)
	for _, f := range score.Factors {
		factorNames[f.Factor] = true
	}

	required := []string{"action_type", "asset_criticality", "recent_incidents", "blast_radius", "time_of_day", "mutation_window_status"}
	for _, req := range required {
		if !factorNames[req] {
			t.Errorf("missing required factor: %s", req)
		}
	}
}

// Test 6: advisory_only is always true
func TestRiskScore_AdvisoryOnlyAlwaysTrue(t *testing.T) {
	e := setupRiskTest(t)
	actions := []string{"proxmox.start", "proxmox.shutdown", "proxmox.stop", "proxmox.snapshot"}
	for _, action := range actions {
		score := e.getRiskScore(t, action)
		if !score.AdvisoryOnly {
			t.Errorf("%s: advisory_only should be true", action)
		}
	}
}

// Test 7: invalid action rejected
func TestRiskScore_InvalidActionRejected(t *testing.T) {
	e := setupRiskTest(t)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/assets/%s/risk-score?action=proxmox.delete", e.teamID, e.assetID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid action, got %d", w.Code)
	}
}

// Test 8: team scoping enforced
func TestRiskScore_TeamScoping(t *testing.T) {
	e := setupRiskTest(t)

	// Use a different team ID
	otherTeam := uuid.New().String()
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/assets/%s/risk-score?action=proxmox.shutdown", otherTeam, e.assetID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for asset from different team, got %d", w.Code)
	}
}

// Test 9: no raw incident body returned
func TestRiskScore_NoRawIncidentBody(t *testing.T) {
	e := setupRiskTest(t)
	score := e.getRiskScore(t, "proxmox.shutdown")

	bodyBytes, _ := json.Marshal(score)
	bodyStr := string(bodyBytes)

	// No raw incident fields should be present
	if strings.Contains(bodyStr, "\"summary\"") {
		t.Error("raw incident summary in risk score response")
	}
	if strings.Contains(bodyStr, "\"title\"") {
		t.Error("raw incident title in risk score response")
	}
	if strings.Contains(bodyStr, "\"description\"") {
		t.Error("raw incident description in risk score response")
	}
}

// Test 10: no audit/outbox emission
func TestRiskScore_NoAuditEmission(t *testing.T) {
	e := setupRiskTest(t)

	var countBefore int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%%risk_score%%'").Scan(&countBefore)

	_ = e.getRiskScore(t, "proxmox.shutdown")

	var countAfter int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%%risk_score%%'").Scan(&countAfter)

	if countAfter != countBefore {
		t.Errorf("audit logs changed: %d -> %d", countBefore, countAfter)
	}
}

// Test 11: no sensitive metadata in response
func TestRiskScore_NoSensitiveMetadata(t *testing.T) {
	e := setupRiskTest(t)
	score := e.getRiskScore(t, "proxmox.stop")

	bodyBytes, _ := json.Marshal(score)
	bodyStr := string(bodyBytes)

	sensitive := []string{"token", "secret", "password", "credential", "api_key"}
	for _, s := range sensitive {
		if strings.Contains(strings.ToLower(bodyStr), s) {
			t.Errorf("sensitive word '%s' found in risk score response", s)
		}
	}
}

// Test 12: mitigation notes are present
func TestRiskScore_MitigationNotesPresent(t *testing.T) {
	e := setupRiskTest(t)
	score := e.getRiskScore(t, "proxmox.shutdown")

	if len(score.MitigationNotes) == 0 {
		t.Error("mitigation notes should not be empty")
	}
}

// Test 13: inputs section populated
func TestRiskScore_InputsPopulated(t *testing.T) {
	e := setupRiskTest(t)
	score := e.getRiskScore(t, "proxmox.shutdown")

	if score.Inputs.RecentIncidentWindowDays != 7 {
		t.Errorf("expected 7 day window, got %d", score.Inputs.RecentIncidentWindowDays)
	}
	// Mutation window should be reported
	if score.Inputs.ChangeWindowStatus == "" {
		t.Error("change window status should be populated")
	}
}

// Test 14: snapshot action is lower risk than stop
func TestRiskScore_SnapshotLowerThanStop(t *testing.T) {
	e := setupRiskTest(t)
	snapScore := e.getRiskScore(t, "proxmox.snapshot")
	stopScore := e.getRiskScore(t, "proxmox.stop")

	if snapScore.RiskScore >= stopScore.RiskScore {
		t.Errorf("snapshot (%d) should be lower than stop (%d)", snapScore.RiskScore, stopScore.RiskScore)
	}
}

// Test 15: unauthorized user denied
func TestRiskScore_UnauthorizedDenied(t *testing.T) {
	e := setupRiskTest(t)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/assets/%s/risk-score?action=proxmox.shutdown", e.teamID, e.assetID), nil)
	// No Authorization header
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for no auth, got %d", w.Code)
	}
}

// Test 16: missing action parameter rejected
func TestRiskScore_MissingActionRejected(t *testing.T) {
	e := setupRiskTest(t)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/assets/%s/risk-score", e.teamID, e.assetID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing action, got %d", w.Code)
	}
}
