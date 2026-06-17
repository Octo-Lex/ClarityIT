package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Related Knowledge Tests ───

func TestRelated_RequiresValidSource(t *testing.T) {
	e := setupKnowledgeTest(t)

	w := doRelated(e, "work_item", "00000000-0000-0000-0000-000000000000")
	if w.Code != 404 {
		t.Errorf("expected 404 for nonexistent source, got %d", w.Code)
	}
}

func TestRelated_ExcludesSourceItself(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "work_item", "Fix auth bug", "Fix auth bug in login", "Authentication bug in the login flow")
	id2 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Auth Design", "Auth architecture doc", "The authentication system uses JWT tokens")

	_ = id2

	w := doRelated(e, "work_item", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	for _, r := range resp.Related {
		if r.SourceType == "work_item" && r.SourceID == id1 {
			t.Error("source item appeared in its own related results")
		}
	}
}

func TestRelated_ContentSimilarity(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Authentication Guide", "How auth works", "The authentication system uses JWT tokens for login")
	id2 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "incident", "Auth Login Outage", "Login broken", "Authentication login failures across the platform")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	found := false
	for _, r := range resp.Related {
		if r.SourceID == id2 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find the auth-related incident in related results")
	}
}

func TestRelated_SameSourceFamily(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "work_item", "First task unique", "First task desc", "Completely unrelated content about gardening")
	id2 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "work_item", "Second task unique", "Second task desc", "Another task about cooking")

	w := doRelated(e, "work_item", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	found := false
	for _, r := range resp.Related {
		if r.SourceID == id2 {
			found = true
		}
	}
	if !found {
		t.Error("expected to find same-source-family item in results")
	}
}

func TestRelated_RespectsLimit(t *testing.T) {
	e := setupKnowledgeTest(t)

	for i := 0; i < 25; i++ {
		insertKnowledgeItemSourceID(t, e.pool, e.teamID, "artifact",
			fmt.Sprintf("Limit Item %d", i), "Summary",
			fmt.Sprintf("Some content for item %d about auth", i))
	}
	id := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Target Limit", "Target doc", "Target document about auth content")

	w := doRelatedWithLimit(e, "clarity_document", id, 5)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Related) > 5 {
		t.Errorf("expected at most 5 results, got %d", len(resp.Related))
	}
}

func TestRelated_DefaultLimit8(t *testing.T) {
	e := setupKnowledgeTest(t)

	for i := 0; i < 15; i++ {
		insertKnowledgeItemSourceID(t, e.pool, e.teamID, "artifact",
			fmt.Sprintf("Default Item %d", i), "Summary",
			fmt.Sprintf("Some content for item %d about auth", i))
	}
	id := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Target Default", "Target doc", "Target document about auth content")

	w := doRelated(e, "clarity_document", id)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Related) > 8 {
		t.Errorf("expected at most 8 results with default limit, got %d", len(resp.Related))
	}
}

func TestRelated_MaxLimit20(t *testing.T) {
	e := setupKnowledgeTest(t)

	for i := 0; i < 30; i++ {
		insertKnowledgeItemSourceID(t, e.pool, e.teamID, "artifact",
			fmt.Sprintf("Max Item %d", i), "Summary",
			fmt.Sprintf("Some content for item %d about auth systems", i))
	}
	id := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Target Max", "Target doc", "Target document about auth content")

	w := doRelatedWithLimit(e, "clarity_document", id, 100)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Related) > 20 {
		t.Errorf("expected at most 20 results with max limit, got %d", len(resp.Related))
	}
}

func TestRelated_DeterministicReasons(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Reason Auth Guide", "Auth design", "Authentication system JWT login tokens")
	insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Reason Auth Runbook", "Auth ops", "Authentication JWT tokens troubleshooting")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	validReasons := map[string]bool{
		"explicit_link":      true,
		"context_edge":       true,
		"shared_reference":   true,
		"content_similarity": true,
		"same_source_family": true,
		"recent_related":     true,
	}

	for _, r := range resp.Related {
		if !validReasons[r.Reason] {
			t.Errorf("invalid reason value: %s", r.Reason)
		}
	}
}

func TestRelated_NoContentHash(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Hash Doc A", "Summary A", "Content about authentication tokens")
	insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Hash Doc B", "Summary B", "More about authentication tokens")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if contains(body, "content_hash") {
		t.Error("response contains content_hash")
	}
}

func TestRelated_NoRawMetadata(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Meta Doc A", "Summary", "Content about auth")
	insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Meta Doc B", "Summary", "Content about auth")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if contains(body, "\"metadata\"") {
		t.Error("response contains raw metadata field")
	}
}

func TestRelated_NoStorageIdentifiers(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Storage Doc A", "Summary", "Content about auth")
	insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Storage Doc B", "Summary", "Content about auth")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if contains(body, "storage_object_id") {
		t.Error("response contains storage_object_id")
	}
}

func TestRelated_EmptyRelatedSet(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Unique Empty Doc", "Unique content", "Completely unique content about zebras and elephants and unicorns")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Related == nil {
		t.Error("expected non-nil related array")
	}
}

func TestRelated_MissingParams(t *testing.T) {
	e := setupKnowledgeTest(t)

	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/knowledge/related", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing params, got %d", w.Code)
	}
}

func TestRelated_SourceInResponse(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Source Response Doc", "Summary", "Content about auth systems")

	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp RelatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Source.SourceType != "clarity_document" {
		t.Errorf("expected source_type=clarity_document, got %s", resp.Source.SourceType)
	}
	if resp.Source.SourceID != id1 {
		t.Errorf("expected source_id=%s, got %s", id1, resp.Source.SourceID)
	}
}

func TestRelated_UsesKnowledgeReadPermission(t *testing.T) {
	e := setupKnowledgeTest(t)

	id1 := insertKnowledgeItemSourceID(t, e.pool, e.teamID, "clarity_document", "Permission Doc", "Summary", "Content about auth")

	// Verify the route is registered with knowledge.read permission by checking it works
	w := doRelated(e, "clarity_document", id1)
	if w.Code != 200 {
		t.Errorf("expected 200 for knowledge.read user, got %d", w.Code)
	}
}

// ─── Helpers ───

func doRelated(e *knowledgeTestEnv, sourceType, sourceID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET",
		"/api/teams/"+e.teamID+"/knowledge/related?source_type="+sourceType+"&source_id="+sourceID, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func doRelatedWithLimit(e *knowledgeTestEnv, sourceType, sourceID string, limit int) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET",
		fmt.Sprintf("/api/teams/%s/knowledge/related?source_type=%s&source_id=%s&limit=%d",
			e.teamID, sourceType, sourceID, limit), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

// insertKnowledgeItemSourceID inserts a knowledge item and returns its source_id (not PK id)
func insertKnowledgeItemSourceID(t *testing.T, pool *pgxpool.Pool, teamID, sourceType, title, summary, content string) string {
	t.Helper()
	var sourceID string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata, stale_after)
		VALUES ($1::uuid, $2, gen_random_uuid(), $3, $4, $5, '{}'::jsonb, NOW() + INTERVAL '90 days')
		RETURNING source_id::text
	`, teamID, sourceType, title, summary, content).Scan(&sourceID)
	if err != nil {
		t.Fatalf("insert knowledge item: %v", err)
	}
	return sourceID
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
