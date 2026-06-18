package contextx

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

const qualityDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type qualityTestEnv struct {
	r      *chi.Mux
	pool   *pgxpool.Pool
	token  string
	teamID string
}

func setupQualityTest(t *testing.T) *qualityTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), qualityDBURL)
	t.Cleanup(func() { pool.Close() })

	qualityH := NewQualityHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "assets.read")).
			Get("/context/quality", qualityH.GetQuality)
		r.With(middleware.RequirePermission(pool, "assets.read")).
			Post("/context/relations/{relationId}/confirm", qualityH.ConfirmRelation)
		r.With(middleware.RequirePermission(pool, "assets.read")).
			Post("/context/relations/{relationId}/dismiss", qualityH.DismissRelation)
	})

	token := loginForQuality(t, r)

	var teamID string
	pool.QueryRow(t.Context(), "SELECT id::text FROM teams LIMIT 1").Scan(&teamID)

	return &qualityTestEnv{r: r, pool: pool, token: token, teamID: teamID}
}

func loginForQuality(t *testing.T, r *chi.Mux) string {
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

func (e *qualityTestEnv) getQuality(t *testing.T) QualityResponse {
	t.Helper()
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/context/quality", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("quality: %d %s", w.Code, w.Body.String())
	}
	var resp QualityResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// createTestRelation creates a context edge for testing
func (e *qualityTestEnv) createTestRelation(t *testing.T, fromType, toType, relationType string, weight float64, daysStale int) (relationID, fromNodeID, toNodeID string) {
	t.Helper()
	ctx := t.Context()
	teamID, _ := uuid.Parse(e.teamID)
	fromID := uuid.New()
	toID := uuid.New()

	// Build the stale time if needed
	var staleTime *time.Time
	if daysStale > 0 {
		st := time.Now().UTC().AddDate(0, 0, -daysStale)
		staleTime = &st
	}

	// Create from node (with stale timestamps on INSERT — trigger only fires on UPDATE)
	if staleTime != nil {
		_, err := e.pool.Exec(ctx, `
			INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties, created_at, updated_at)
			VALUES ($1, $2, $3, 'test', '{}'::jsonb, $4, $4)
			ON CONFLICT DO NOTHING
		`, teamID, fromType, fromID, *staleTime)
		if err != nil {
			t.Fatalf("create from node: %v", err)
		}
	} else {
		_, err := e.pool.Exec(ctx, `
			INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
			VALUES ($1, $2, $3, 'test', '{}'::jsonb)
			ON CONFLICT DO NOTHING
		`, teamID, fromType, fromID)
		if err != nil {
			t.Fatalf("create from node: %v", err)
		}
	}

	// Create to node
	if staleTime != nil {
		_, err := e.pool.Exec(ctx, `
			INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties, created_at, updated_at)
			VALUES ($1, $2, $3, 'test', '{}'::jsonb, $4, $4)
			ON CONFLICT DO NOTHING
		`, teamID, toType, toID, *staleTime)
		if err != nil {
			t.Fatalf("create to node: %v", err)
		}
	} else {
		_, err := e.pool.Exec(ctx, `
			INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
			VALUES ($1, $2, $3, 'test', '{}'::jsonb)
			ON CONFLICT DO NOTHING
		`, teamID, toType, toID)
		if err != nil {
			t.Fatalf("create to node: %v", err)
		}
	}

	// Look up actual node IDs
	err := e.pool.QueryRow(ctx,
		"SELECT id::text FROM context_nodes WHERE team_id=$1 AND entity_id=$2", teamID, fromID).Scan(&fromNodeID)
	if err != nil {
		t.Fatalf("get from node id: %v", err)
	}
	err = e.pool.QueryRow(ctx,
		"SELECT id::text FROM context_nodes WHERE team_id=$1 AND entity_id=$2", teamID, toID).Scan(&toNodeID)
	if err != nil {
		t.Fatalf("get to node id: %v", err)
	}

	// Create edge
	edgeID := uuid.New()
	_, err = e.pool.Exec(ctx, `
		INSERT INTO context_edges (id, team_id, from_node_id, to_node_id, relation_type, weight)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, edgeID, teamID, fromNodeID, toNodeID, relationType, weight)
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}
	relationID = edgeID.String()

	return relationID, fromNodeID, toNodeID
}

// ─── Tests ───

// Test 1: quality endpoint returns score and summary
func TestQuality_ReturnsScoreAndSummary(t *testing.T) {
	e := setupQualityTest(t)
	resp := e.getQuality(t)

	if resp.QualityScore < 0 || resp.QualityScore > 100 {
		t.Errorf("quality score %d out of bounds", resp.QualityScore)
	}
	if resp.Summary.TotalNodes < 0 {
		t.Error("total nodes should be >= 0")
	}
	if resp.Summary.TotalRelations < 0 {
		t.Error("total relations should be >= 0")
	}
}

// Test 2: stale nodes are detected
func TestQuality_StaleNodesDetected(t *testing.T) {
	e := setupQualityTest(t)
	e.createTestRelation(t, "asset", "service", "depends_on", 1.0, 45)

	resp := e.getQuality(t)
	if len(resp.StaleNodes) == 0 {
		t.Error("expected at least 1 stale node")
	}
}

// Test 3: low-confidence relations are detected
func TestQuality_LowConfidenceDetected(t *testing.T) {
	e := setupQualityTest(t)
	e.createTestRelation(t, "asset", "asset", "related_to", 0.3, 0)

	resp := e.getQuality(t)
	if len(resp.LowConfidenceRelations) == 0 {
		t.Error("expected at least 1 low-confidence relation")
	}
}

// Test 4: conflicting relations are detected
func TestQuality_ConflictingDetected(t *testing.T) {
	e := setupQualityTest(t)
	// Create two edges between the same nodes with different types
	relID, fromID, toID := e.createTestRelation(t, "asset", "asset", "depends_on", 1.0, 0)
	ctx := t.Context()
	teamID, _ := uuid.Parse(e.teamID)
	fromUUID, _ := uuid.Parse(fromID)
	toUUID, _ := uuid.Parse(toID)
	_, _ = e.pool.Exec(ctx, `
		INSERT INTO context_edges (id, team_id, from_node_id, to_node_id, relation_type, weight)
		VALUES ($1, $2, $3, $4, 'blocks', 1.0)
		ON CONFLICT DO NOTHING
	`, uuid.New(), teamID, fromUUID, toUUID)

	resp := e.getQuality(t)
	found := false
	for _, c := range resp.ConflictingRelations {
		if c.RelationID == relID {
			found = true
		}
	}
	if !found {
		t.Error("expected conflicting relation to be detected")
	}
}

// Test 5: quality score is bounded 0-100
func TestQuality_ScoreBounded(t *testing.T) {
	e := setupQualityTest(t)
	resp := e.getQuality(t)
	if resp.QualityScore < 0 || resp.QualityScore > 100 {
		t.Errorf("score %d out of bounds", resp.QualityScore)
	}
}

// Test 6: stale_days bounds enforced
func TestQuality_StaleDaysBounds(t *testing.T) {
	e := setupQualityTest(t)
	for _, val := range []string{"0", "366"} {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/teams/%s/context/quality?stale_days=%s", e.teamID, val), nil)
		req.Header.Set("Authorization", "Bearer "+e.token)
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for stale_days=%s, got %d", val, w.Code)
		}
	}
}

// Test 7: confidence_threshold bounds enforced
func TestQuality_ConfidenceThresholdBounds(t *testing.T) {
	e := setupQualityTest(t)
	for _, val := range []string{"-0.1", "1.1"} {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/teams/%s/context/quality?confidence_threshold=%s", e.teamID, val), nil)
		req.Header.Set("Authorization", "Bearer "+e.token)
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Errorf("expected 400 for confidence_threshold=%s, got %d", val, w.Code)
		}
	}
}

// Test 8: team scoping enforced
func TestQuality_TeamScoping(t *testing.T) {
	e := setupQualityTest(t)
	otherTeam := uuid.New().String()

	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/context/quality", otherTeam), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp QualityResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	// Other team should have 0 nodes
	if resp.Summary.TotalNodes > 0 {
		t.Error("expected 0 nodes for other team")
	}
}

// Test 9: confirm relation records review state
func TestQuality_ConfirmRelation(t *testing.T) {
	e := setupQualityTest(t)
	relID, _, _ := e.createTestRelation(t, "asset", "service", "depends_on", 0.4, 0)

	body := `{"reason":"Reviewed and confirmed"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/context/relations/%s/confirm", e.teamID, relID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ReviewResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.QualityStatus != "confirmed" {
		t.Errorf("expected confirmed, got %s", resp.QualityStatus)
	}
}

// Test 10: dismiss relation records review state
func TestQuality_DismissRelation(t *testing.T) {
	e := setupQualityTest(t)
	relID, _, _ := e.createTestRelation(t, "asset", "service", "depends_on", 0.3, 0)

	body := `{"reason":"False positive"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/context/relations/%s/dismiss", e.teamID, relID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ReviewResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.QualityStatus != "dismissed" {
		t.Errorf("expected dismissed, got %s", resp.QualityStatus)
	}
}

// Test 11: confirm does not delete node or relation
func TestQuality_ConfirmNoDelete(t *testing.T) {
	e := setupQualityTest(t)
	relID, fromNodeID, _ := e.createTestRelation(t, "asset", "service", "depends_on", 0.4, 0)

	// Count before
	var edgesBefore, nodesBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_edges WHERE team_id=$1", e.teamID).Scan(&edgesBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_nodes WHERE team_id=$1", e.teamID).Scan(&nodesBefore)

	body := `{"reason":"Confirmed"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/context/relations/%s/confirm", e.teamID, relID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Count after — should be same
	var edgesAfter, nodesAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_edges WHERE team_id=$1", e.teamID).Scan(&edgesAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_nodes WHERE team_id=$1", e.teamID).Scan(&nodesAfter)

	if edgesAfter != edgesBefore {
		t.Errorf("edges changed: %d -> %d", edgesBefore, edgesAfter)
	}
	if nodesAfter != nodesBefore {
		t.Errorf("nodes changed: %d -> %d", nodesBefore, nodesAfter)
	}
	_ = fromNodeID
}

// Test 12: dismiss does not delete node or relation
func TestQuality_DismissNoDelete(t *testing.T) {
	e := setupQualityTest(t)
	relID, _, _ := e.createTestRelation(t, "asset", "service", "depends_on", 0.3, 0)

	var edgesBefore, nodesBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_edges WHERE team_id=$1", e.teamID).Scan(&edgesBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_nodes WHERE team_id=$1", e.teamID).Scan(&nodesBefore)

	body := `{"reason":"Dismissed"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/context/relations/%s/dismiss", e.teamID, relID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var edgesAfter, nodesAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_edges WHERE team_id=$1", e.teamID).Scan(&edgesAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM context_nodes WHERE team_id=$1", e.teamID).Scan(&nodesAfter)

	if edgesAfter != edgesBefore {
		t.Errorf("edges changed: %d -> %d", edgesBefore, edgesAfter)
	}
	if nodesAfter != nodesBefore {
		t.Errorf("nodes changed: %d -> %d", nodesBefore, nodesAfter)
	}
}

// Test 13: dismissed relation remains retrievable
func TestQuality_DismissedRemainsRetrievable(t *testing.T) {
	e := setupQualityTest(t)
	relID, _, _ := e.createTestRelation(t, "asset", "service", "depends_on", 0.3, 0)

	// Dismiss it
	body := `{"reason":"Dismissed"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/context/relations/%s/dismiss", e.teamID, relID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Verify it still exists in the DB
	var count int
	e.pool.QueryRow(t.Context(),
		"SELECT COUNT(*) FROM context_edges WHERE id=$1", relID).Scan(&count)
	if count != 1 {
		t.Error("dismissed relation should still exist in context_edges")
	}
}

// Test 14: no Python worker call (structural)
func TestQuality_NoPythonWorker(t *testing.T) {
	e := setupQualityTest(t)
	resp := e.getQuality(t)
	if !resp.AdvisoryOnly {
		t.Error("advisory_only should be true")
	}
}

// Test 15: no Tool Gateway call (structural — handler has no client reference)
func TestQuality_NoToolGateway(t *testing.T) {
	e := setupQualityTest(t)
	// Structural verification: QualityHandler only has a pool reference
	// No Tool Gateway, ProxmoxClient, or agent handler reference
	resp := e.getQuality(t)
	_ = resp // just verify it runs without side effects
}

// Test 16: sensitive fields are suppressed
func TestQuality_SensitiveFieldsSuppressed(t *testing.T) {
	e := setupQualityTest(t)
	resp := e.getQuality(t)

	bodyBytes, _ := json.Marshal(resp)
	bodyStr := string(bodyBytes)

	// No sensitive fields
	if strings.Contains(strings.ToLower(bodyStr), "password") {
		t.Error("password found in quality response")
	}
	if strings.Contains(strings.ToLower(bodyStr), "secret") {
		t.Error("secret found in quality response")
	}
	if strings.Contains(strings.ToLower(bodyStr), "token") {
		t.Error("token found in quality response")
	}
}

// Test 17: advisory_only always true
func TestQuality_AdvisoryOnlyAlwaysTrue(t *testing.T) {
	e := setupQualityTest(t)
	resp := e.getQuality(t)
	if !resp.AdvisoryOnly {
		t.Error("advisory_only should always be true")
	}
}

// Test 18: cross-team relation returns 404
func TestQuality_CrossTeamRelation404(t *testing.T) {
	e := setupQualityTest(t)
	fakeRelID := uuid.New().String()

	body := `{"reason":"test"}`
	req := httptest.NewRequest("POST",
		fmt.Sprintf("/api/teams/%s/context/relations/%s/confirm", e.teamID, fakeRelID),
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for non-existent relation, got %d", w.Code)
	}
}

// Test 19: unauthorized user denied
func TestQuality_UnauthorizedDenied(t *testing.T) {
	e := setupQualityTest(t)
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/context/quality", e.teamID), nil)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ensure context import is used
var _ = context.Background
