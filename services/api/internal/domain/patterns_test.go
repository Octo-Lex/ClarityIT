package domain

import (
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

const patternsDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type patternsTestEnv struct {
	r       *chi.Mux
	pool    *pgxpool.Pool
	token   string
	teamID  string
}

func setupPatternsTest(t *testing.T) *patternsTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), patternsDBURL)
	t.Cleanup(func() { pool.Close() })

	domainH := NewHandler(pool, cfg)
	patternsH := NewPatternsHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "incidents.create")).
			Post("/incidents", domainH.CreateIncident)
		r.With(middleware.RequirePermission(pool, "incidents.read")).
			Get("/incidents/patterns", patternsH.GetPatterns)
	})

	token := loginForPatterns(t, r)

	var teamID string
	pool.QueryRow(t.Context(), "SELECT id::text FROM teams LIMIT 1").Scan(&teamID)

	return &patternsTestEnv{r: r, pool: pool, token: token, teamID: teamID}
}

func loginForPatterns(t *testing.T, r *chi.Mux) string {
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

// createTestIncident inserts an incident with the given parameters
func (e *patternsTestEnv) createTestIncident(t *testing.T, title, severity, assetID string, daysAgo int) string {
	t.Helper()
	ctx := context.Background()

	metadata := "{}"
	if assetID != "" {
		metadata = fmt.Sprintf(`{"asset_id":"%s"}`, assetID)
	}

	// Create via API (to trigger proper event creation)
	body := fmt.Sprintf(`{"title":"%s","severity":"%s"}`, title, severity)
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/incidents", e.teamID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "pattern-test-"+uuid.New().String())
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("create incident: %d %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	incID := resp["id"].(string)

	// Update metadata and opened_at directly for test control
	_, err := e.pool.Exec(ctx, `
		UPDATE objects SET metadata = $1::jsonb WHERE id = $2
	`, metadata, incID)
	if err != nil {
		t.Fatalf("update metadata: %v", err)
	}

	// Set opened_at to control time window
	openedAt := time.Now().UTC().AddDate(0, 0, -daysAgo)
	_, err = e.pool.Exec(ctx, `
		UPDATE incidents SET opened_at = $1 WHERE object_id = $2
	`, openedAt, incID)
	if err != nil {
		t.Fatalf("update opened_at: %v", err)
	}

	return incID
}

// ─── Tests ───

// Test 1: recurring_asset pattern detected
func TestPatterns_RecurringAsset(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	e.createTestIncident(t, "Disk full", "sev2", assetID, 1)
	e.createTestIncident(t, "Disk full again", "sev3", assetID, 2)
	e.createTestIncident(t, "Memory issue", "sev2", assetID, 3)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	found := false
	for _, p := range resp.Patterns {
		if p.PatternType == "recurring_asset" && len(p.AssetIDs) > 0 && p.AssetIDs[0] == assetID {
			found = true
			if p.OccurrenceCount < 3 {
				t.Errorf("expected 3 occurrences, got %d", p.OccurrenceCount)
			}
		}
	}
	if !found {
		t.Error("recurring_asset pattern not detected")
	}
}

// Test 2: recurring_symptom pattern detected
func TestPatterns_RecurringSymptom(t *testing.T) {
	e := setupPatternsTest(t)

	e.createTestIncident(t, "Database connection timeout", "sev3", "", 1)
	e.createTestIncident(t, "Database connection timeout", "sev3", "", 2)
	e.createTestIncident(t, "Database connection timeout", "sev2", "", 3)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	found := false
	for _, p := range resp.Patterns {
		if p.PatternType == "recurring_symptom" && p.OccurrenceCount >= 3 {
			found = true
		}
	}
	if !found {
		t.Error("recurring_symptom pattern not detected")
	}
}

// Test 3: cluster pattern detected
func TestPatterns_Cluster(t *testing.T) {
	e := setupPatternsTest(t)

	// Create incidents with same severity in quick succession
	e.createTestIncident(t, "Service down A", "sev1", "", 1)
	e.createTestIncident(t, "Service down B", "sev1", "", 1)
	e.createTestIncident(t, "Service down C", "sev1", "", 1)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	found := false
	for _, p := range resp.Patterns {
		if p.PatternType == "cluster" {
			found = true
			break
		}
	}
	if !found {
		t.Error("cluster pattern not detected")
	}
}

// Test 4: noisy_asset pattern detected
func TestPatterns_NoisyAsset(t *testing.T) {
	e := setupPatternsTest(t)

	assetA := uuid.New().String()
	assetB := uuid.New().String()

	// Asset A gets many incidents
	e.createTestIncident(t, "Issue 1", "sev3", assetA, 1)
	e.createTestIncident(t, "Issue 2", "sev3", assetA, 2)
	e.createTestIncident(t, "Issue 3", "sev3", assetA, 3)
	e.createTestIncident(t, "Issue 4", "sev3", assetA, 4)
	e.createTestIncident(t, "Issue 5", "sev3", assetA, 5)

	// Asset B gets one
	e.createTestIncident(t, "Issue B", "sev3", assetB, 1)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	found := false
	for _, p := range resp.Patterns {
		if p.PatternType == "noisy_asset" && len(p.AssetIDs) > 0 && p.AssetIDs[0] == assetA {
			found = true
		}
	}
	if !found {
		t.Error("noisy_asset pattern not detected")
	}
}

// Test 5: no patterns returns empty list
func TestPatterns_EmptyList(t *testing.T) {
	e := setupPatternsTest(t)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns?window_days=1", uuid.New().String()), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(resp.Patterns))
	}
}

// Test 6: confidence bounded 0.0–1.0
func TestPatterns_ConfidenceBounded(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	for i := 0; i < 5; i++ {
		e.createTestIncident(t, fmt.Sprintf("Recurring %d", i), "sev2", assetID, i+1)
	}

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	for _, p := range resp.Patterns {
		if p.Confidence < 0.0 || p.Confidence > 1.0 {
			t.Errorf("confidence %f is out of bounds for pattern %s", p.Confidence, p.PatternType)
		}
	}
}

// Test 7: stable pattern_id for same input
func TestPatterns_StablePatternID(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	e.createTestIncident(t, "Stable 1", "sev2", assetID, 1)
	e.createTestIncident(t, "Stable 2", "sev2", assetID, 2)

	// First request
	req1 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req1.Header.Set("Authorization", "Bearer "+e.token)
	w1 := httptest.NewRecorder()
	e.r.ServeHTTP(w1, req1)
	var resp1 PatternsResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)

	// Second request (same data)
	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var resp2 PatternsResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	// Compare pattern IDs
	if len(resp1.Patterns) != len(resp2.Patterns) {
		t.Fatalf("pattern count mismatch: %d vs %d", len(resp1.Patterns), len(resp2.Patterns))
	}

	idSet1 := make(map[string]bool)
	for _, p := range resp1.Patterns {
		idSet1[p.PatternID] = true
	}

	for _, p := range resp2.Patterns {
		if !idSet1[p.PatternID] {
			t.Errorf("pattern_id %s not stable across requests", p.PatternID)
		}
	}
}

// Test 8: team scoping enforced
func TestPatterns_TeamScoping(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	e.createTestIncident(t, "Team A incident", "sev2", assetID, 1)
	e.createTestIncident(t, "Team A incident", "sev2", assetID, 2)

	// Query with a different team ID
	otherTeam := uuid.New().String()
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", otherTeam), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Patterns) != 0 {
		t.Errorf("expected 0 patterns from other team, got %d", len(resp.Patterns))
	}
}

// Test 9: invalid window_days rejected
func TestPatterns_InvalidWindowDays(t *testing.T) {
	e := setupPatternsTest(t)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns?window_days=0", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for window_days=0, got %d", w.Code)
	}

	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns?window_days=91", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)

	if w2.Code != 400 {
		t.Errorf("expected 400 for window_days=91, got %d", w2.Code)
	}
}

// Test 10: invalid min_occurrences rejected
func TestPatterns_InvalidMinOccurrences(t *testing.T) {
	e := setupPatternsTest(t)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns?min_occurrences=1", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for min_occurrences=1, got %d", w.Code)
	}

	req2 := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns?min_occurrences=21", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)

	if w2.Code != 400 {
		t.Errorf("expected 400 for min_occurrences=21, got %d", w2.Code)
	}
}

// Test 11: endpoint does not mutate incident records
func TestPatterns_NoMutation(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	id1 := e.createTestIncident(t, "Mut test 1", "sev2", assetID, 1)
	id2 := e.createTestIncident(t, "Mut test 2", "sev2", assetID, 2)

	// Snapshot the records
	var title1, status1, sev1 string
	e.pool.QueryRow(t.Context(),
		"SELECT o.title, o.status, i.severity FROM objects o JOIN incidents i ON i.object_id=o.id WHERE o.id=$1",
		id1).Scan(&title1, &status1, &sev1)

	var title2, status2, sev2 string
	e.pool.QueryRow(t.Context(),
		"SELECT o.title, o.status, i.severity FROM objects o JOIN incidents i ON i.object_id=o.id WHERE o.id=$1",
		id2).Scan(&title2, &status2, &sev2)

	// Call patterns endpoint
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Verify records unchanged
	var title1After, status1After, sev1After string
	e.pool.QueryRow(t.Context(),
		"SELECT o.title, o.status, i.severity FROM objects o JOIN incidents i ON i.object_id=o.id WHERE o.id=$1",
		id1).Scan(&title1After, &status1After, &sev1After)

	if title1 != title1After || status1 != status1After || sev1 != sev1After {
		t.Error("incident 1 was mutated by patterns endpoint")
	}

	var title2After, status2After, sev2After string
	e.pool.QueryRow(t.Context(),
		"SELECT o.title, o.status, i.severity FROM objects o JOIN incidents i ON i.object_id=o.id WHERE o.id=$1",
		id2).Scan(&title2After, &status2After, &sev2After)

	if title2 != title2After || status2 != status2After || sev2 != sev2After {
		t.Error("incident 2 was mutated by patterns endpoint")
	}
}

// Test 12: raw incident body not returned
func TestPatterns_NoRawIncidentBody(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	e.createTestIncident(t, "Body test with secret password=hunter2", "sev2", assetID, 1)
	e.createTestIncident(t, "Body test with secret password=hunter2", "sev2", assetID, 2)

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	body := w.Body.String()
	if strings.Contains(body, "hunter2") {
		t.Error("raw incident body content leaked into patterns response")
	}
	if strings.Contains(body, "\"summary\"") {
		// Summary field should not be in the response
		t.Error("raw incident summary field leaked into patterns response")
	}
}

// Test 13: advisory_only flag always true
func TestPatterns_AdvisoryOnlyAlwaysTrue(t *testing.T) {
	e := setupPatternsTest(t)
	assetID := uuid.New().String()

	for i := 0; i < 3; i++ {
		e.createTestIncident(t, fmt.Sprintf("Advisory %d", i), "sev2", assetID, i+1)
	}

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/incidents/patterns", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp PatternsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	for _, p := range resp.Patterns {
		if !p.AdvisoryOnly {
			t.Errorf("pattern %s should have advisory_only=true", p.PatternType)
		}
	}
}
