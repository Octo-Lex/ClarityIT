package knowledge

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

const knowledgeDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

// ─── Test environment ───

type knowledgeTestEnv struct {
	r      *chi.Mux
	pool   *pgxpool.Pool
	token  string
	teamID string
}

func setupKnowledgeTest(t *testing.T) *knowledgeTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, err := pgxpool.New(context.Background(), knowledgeDBURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	_, err = pool.Exec(context.Background(), "TRUNCATE knowledge_items, knowledge_chunks CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	kh := NewHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)

	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "knowledge.search")).
			Get("/knowledge/search", kh.SearchHTTP)
		r.With(middleware.RequirePermission(pool, "knowledge.read")).
			Get("/knowledge/index-status", kh.IndexStatusHTTP)
		r.With(middleware.RequirePermission(pool, "knowledge.read")).
			Get("/knowledge/{itemId}", kh.GetHTTP)
	})

	token := loginKnowledge(t, r, "owner@test.dev", "password12")

	var teamID string
	pool.QueryRow(context.Background(), `
		SELECT t.id::text FROM teams t
		JOIN team_memberships tm ON tm.team_id = t.id
		JOIN users u ON u.id = tm.user_id
		WHERE u.email = 'owner@test.dev'
		LIMIT 1
	`).Scan(&teamID)

	return &knowledgeTestEnv{
		r:      r,
		pool:   pool,
		token:  token,
		teamID: teamID,
	}
}

func loginKnowledge(t *testing.T, r *chi.Mux, email, pw string) string {
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

func insertKnowledgeItem(t *testing.T, pool *pgxpool.Pool, teamID, sourceType, title, summary, content string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata)
		VALUES ($1::uuid, $2, gen_random_uuid(), $3, $4, $5, '{}'::jsonb)
		RETURNING id::text
	`, teamID, sourceType, title, summary, content).Scan(&id)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	return id
}

func doSearch(e *knowledgeTestEnv, query string) SearchResponse {
	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/search?q="+query, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp SearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ─── Tests ───

func TestIndexAndSearch(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Backup Verification Runbook",
		"How to verify backups",
		"This document describes the backup verification process. Backups must be tested weekly.")
	insertKnowledgeItem(t, e.pool, e.teamID, "incident",
		"Database Outage Postmortem",
		"Sev1 incident from disk failure",
		"On June 15th the primary database experienced a disk failure leading to a 45 minute outage.")
	insertKnowledgeItem(t, e.pool, e.teamID, "artifact",
		"Q3 Architecture Decision Record",
		"Decisions about microservices migration",
		"We decided to adopt a modular monolith pattern instead of microservices for the initial release.")

	// Test 1: Search for "backup"
	resp := doSearch(e, "backup")
	if resp.Total < 1 {
		t.Fatalf("expected >=1 result for 'backup', got %d", resp.Total)
	}
	if !strings.Contains(resp.Results[0].Title, "Backup") {
		t.Errorf("expected title to contain 'Backup', got %q", resp.Results[0].Title)
	}

	// Test 2: Search for "database outage"
	resp = doSearch(e, "database+outage")
	if resp.Total < 1 {
		t.Fatalf("expected >=1 result for 'database outage', got %d", resp.Total)
	}
	if !strings.Contains(resp.Results[0].Title, "Database Outage") {
		t.Errorf("expected 'Database Outage', got %q", resp.Results[0].Title)
	}

	// Test 3: Search for "microservices"
	resp = doSearch(e, "microservices")
	if resp.Total < 1 {
		t.Fatalf("expected >=1 result for 'microservices', got %d", resp.Total)
	}
	if !strings.Contains(resp.Results[0].Title, "Architecture Decision") {
		t.Errorf("expected 'Architecture Decision', got %q", resp.Results[0].Title)
	}
}

func TestSearchWithSourceTypeFilter(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Backup Guide", "Backup procedures", "Weekly backup verification required.")
	insertKnowledgeItem(t, e.pool, e.teamID, "incident",
		"Backup Failure", "Backup system failed", "The backup system failed due to disk space.")

	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/search?q=backup&source_type=incident", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp SearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	for _, r := range resp.Results {
		if r.SourceType != "incident" {
			t.Errorf("expected source_type=incident, got %s", r.SourceType)
		}
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	e := setupKnowledgeTest(t)

	resp := doSearch(e, "")
	if resp.Total != 0 || len(resp.Results) != 0 {
		t.Errorf("expected empty results for empty query, got total=%d results=%d", resp.Total, len(resp.Results))
	}
}

func TestSearchNoResults(t *testing.T) {
	e := setupKnowledgeTest(t)

	resp := doSearch(e, "zzznonexistentxyz123")
	if resp.Total != 0 || len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got total=%d", resp.Total)
	}
}

func TestIndexStatus(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Status Test Doc", "Summary", "Content")

	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/index-status", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var status IndexStatus
	json.Unmarshal(w.Body.Bytes(), &status)
	if status.TotalItems < 1 {
		t.Errorf("expected >=1 total items, got %d", status.TotalItems)
	}
	if status.ByType["clarity_document"] < 1 {
		t.Errorf("expected >=1 clarity_document, got %d", status.ByType["clarity_document"])
	}
}

func TestGetKnowledgeItem(t *testing.T) {
	e := setupKnowledgeTest(t)

	itemID := insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Get Test", "Summary", "Content body")

	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/"+itemID, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var item KnowledgeItem
	json.Unmarshal(w.Body.Bytes(), &item)
	if item.Title != "Get Test" {
		t.Errorf("expected 'Get Test', got %q", item.Title)
	}
	if item.Visibility != "team" {
		t.Errorf("expected visibility=team, got %s", item.Visibility)
	}
}

func TestGetKnowledgeItemNotFound(t *testing.T) {
	e := setupKnowledgeTest(t)

	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/00000000-0000-0000-0000-000000000099", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetKnowledgeItemCrossTeam(t *testing.T) {
	e := setupKnowledgeTest(t)

	itemID := insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Cross Team Test", "Summary", "Content")

	var team2ID string
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO teams (name, slug)
		VALUES ('Other Team KB', 'other-team-kb-' || md5(random()::text))
		RETURNING id::text
	`).Scan(&team2ID)
	if err != nil {
		t.Fatalf("create team2: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/teams/"+team2ID+"/knowledge/"+itemID, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 for cross-team access, got %d", w.Code)
	}
}

func TestTeamIsolation(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Team Isolation Test", "Summary", "unique searchable content zzztop")

	var team2ID string
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO teams (name, slug)
		VALUES ('Isolation Team KB2', 'iso-team-kb2-' || md5(random()::text))
		RETURNING id::text
	`).Scan(&team2ID)
	if err != nil {
		t.Fatalf("create team2: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/teams/"+team2ID+"/knowledge/search?q=zzztop", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp SearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("team2 should not see team1's knowledge, got total=%d", resp.Total)
	}
}

func TestIndexFunctionDirectly(t *testing.T) {
	e := setupKnowledgeTest(t)
	h := NewHandler(e.pool)
	ctx := context.Background()

	var sourceID string
	err := e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)
	if err != nil {
		t.Fatal(err)
	}

	item, err := h.Index(ctx, e.teamID, IndexRequest{
		SourceType:  "artifact",
		SourceID:    sourceID,
		Title:       "Direct Index Test",
		Summary:     "Indexed via function",
		ContentText: "This content was indexed directly via the Index function.",
		Metadata:    map[string]any{"doc_type": "test"},
	})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if item.Title != "Direct Index Test" {
		t.Errorf("expected 'Direct Index Test', got %q", item.Title)
	}

	// Upsert
	item2, err := h.Index(ctx, e.teamID, IndexRequest{
		SourceType:  "artifact",
		SourceID:    sourceID,
		Title:       "Updated Title",
		Summary:     "Updated summary",
		ContentText: "Updated content text.",
		Metadata:    map[string]any{"doc_type": "test"},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if item.ID != item2.ID {
		t.Errorf("expected same ID after upsert, got %s vs %s", item.ID, item2.ID)
	}

	var count int
	e.pool.QueryRow(ctx, `
		SELECT count(*) FROM knowledge_items
		WHERE team_id = $1::uuid AND source_type = 'artifact' AND source_id = $2::uuid
	`, e.teamID, sourceID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 item after upsert, got %d", count)
	}
}

func TestIndexUpsertSameContent(t *testing.T) {
	e := setupKnowledgeTest(t)
	h := NewHandler(e.pool)
	ctx := context.Background()

	var sourceID string
	e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)

	h.Index(ctx, e.teamID, IndexRequest{
		SourceType: "clarity_document", SourceID: sourceID,
		Title: "Hash Test", ContentText: "Same content",
	})
	h.Index(ctx, e.teamID, IndexRequest{
		SourceType: "clarity_document", SourceID: sourceID,
		Title: "Hash Test", ContentText: "Same content",
	})

	var count int
	e.pool.QueryRow(ctx, `
		SELECT count(*) FROM knowledge_items
		WHERE team_id = $1::uuid AND source_type = 'clarity_document'
	`, e.teamID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 item after same-content upsert, got %d", count)
	}
}

func TestSearchRanking(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"PostgreSQL Backup Strategy", "Backup guide",
		"postgresql backup strategy verification weekly procedures detailed")
	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"General Notes", "Various notes",
		"backup is mentioned once here")

	resp := doSearch(e, "backup")
	if len(resp.Results) < 2 {
		t.Fatalf("expected >=2 results, got %d", len(resp.Results))
	}
	if resp.Results[0].Rank < resp.Results[1].Rank {
		t.Errorf("expected DESC ranking, got rank[0]=%f < rank[1]=%f", resp.Results[0].Rank, resp.Results[1].Rank)
	}
}

func TestSearchWithLimitOffset(t *testing.T) {
	e := setupKnowledgeTest(t)

	for i := 0; i < 5; i++ {
		insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
			fmt.Sprintf("Pagination Test Doc %d", i),
			"Summary",
			fmt.Sprintf("common keyword paginationtest content number %d", i))
	}

	// Test limit
	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/search?q=paginationtest&limit=2", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp SearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}

	// Test offset
	req = httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/search?q=paginationtest&limit=2&offset=2", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
}

func TestSearchSnippetGeneration(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Snippet Test", "Summary",
		"This is a long document that discusses incident response procedures in detail. The procedures include backup verification and database recovery.")

	resp := doSearch(e, "backup+verification")
	if len(resp.Results) < 1 {
		t.Fatalf("expected >=1 result")
	}
	if resp.Results[0].Snippet == "" {
		t.Error("expected non-empty snippet")
	}
}

func TestRemoveKnowledgeItem(t *testing.T) {
	e := setupKnowledgeTest(t)
	h := NewHandler(e.pool)
	ctx := context.Background()

	var sourceID string
	e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)

	h.Index(ctx, e.teamID, IndexRequest{
		SourceType: "artifact", SourceID: sourceID,
		Title: "To Be Removed", ContentText: "This will be deleted",
	})

	resp := doSearch(e, "deleted")
	if resp.Total < 1 {
		t.Fatalf("expected >=1 result before remove")
	}

	h.Remove(ctx, e.teamID, "artifact", sourceID)

	resp = doSearch(e, "deleted")
	if resp.Total != 0 {
		t.Errorf("expected 0 results after remove, got %d", resp.Total)
	}
}

func TestValidateSourceType(t *testing.T) {
	validTypes := []string{
		"artifact", "clarity_document", "meeting_summary", "status_report",
		"presentation", "template", "work_item", "incident", "project",
		"asset", "remediation", "approval", "context_node",
	}
	for _, st := range validTypes {
		if !ValidateSourceType(st) {
			t.Errorf("expected valid: %s", st)
		}
	}
	invalidTypes := []string{"", "invalid", "user", "session", "token", "secret", "password"}
	for _, st := range invalidTypes {
		if ValidateSourceType(st) {
			t.Errorf("expected invalid: %s", st)
		}
	}
}

func TestStaleDetection(t *testing.T) {
	e := setupKnowledgeTest(t)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata, stale_after)
		VALUES ($1::uuid, 'clarity_document', gen_random_uuid(),
		        'Old Document', 'Old summary', 'Old content',
		        '{}'::jsonb, NOW() - INTERVAL '1 day')
	`, e.teamID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.pool.Exec(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata, stale_after)
		VALUES ($1::uuid, 'clarity_document', gen_random_uuid(),
		        'Fresh Document', 'Fresh summary', 'Fresh content',
		        '{}'::jsonb, NOW() + INTERVAL '90 days')
	`, e.teamID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/index-status", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var status IndexStatus
	json.Unmarshal(w.Body.Bytes(), &status)
	if status.StaleCount < 1 {
		t.Errorf("expected >=1 stale item, got %d", status.StaleCount)
	}
}

func TestSearchVectorTrigger(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "clarity_document",
		"Trigger Test", "Testing auto tsvector",
		"The search vector trigger should populate this automatically.")

	resp := doSearch(e, "trigger+populate+automatically")
	if resp.Total < 1 {
		t.Errorf("trigger should auto-populate search_vector, got total=%d", resp.Total)
	}
}

func TestIndexWithMetadata(t *testing.T) {
	e := setupKnowledgeTest(t)
	h := NewHandler(e.pool)
	ctx := context.Background()

	var sourceID string
	e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)

	item, err := h.Index(ctx, e.teamID, IndexRequest{
		SourceType:  "incident",
		SourceID:    sourceID,
		Title:       "Metadata Test",
		ContentText: "Content with metadata",
		Metadata: map[string]any{
			"severity":    "sev2",
			"status":      "resolved",
			"object_type": "incident",
		},
	})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if item.Metadata == nil {
		t.Error("expected non-nil metadata")
	}
}

func TestSourceUpdatedAndStaleAfter(t *testing.T) {
	e := setupKnowledgeTest(t)
	h := NewHandler(e.pool)
	ctx := context.Background()

	var sourceID string
	e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)

	now := time.Now().Format(time.RFC3339)
	item, err := h.Index(ctx, e.teamID, IndexRequest{
		SourceType:      "clarity_document",
		SourceID:        sourceID,
		Title:           "Timestamp Test",
		ContentText:     "Content with timestamps",
		SourceUpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if item.StaleAfter.IsZero() {
		t.Error("stale_after should be set")
	}
	if item.SourceUpdatedAt.IsZero() {
		t.Error("source_updated_at should be set")
	}
}

// Ensure uuid import is used
var _ = uuid.New
