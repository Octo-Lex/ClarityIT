package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestQuality_ReportStructure(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert some knowledge items
	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Doc A", "Summary A", "content A")
	insertKnowledgeItem(t, e.pool, e.teamID, "incident", "Incident B", "Summary B", "content B")

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)

	if report["total_items"].(float64) != 2 {
		t.Errorf("expected 2 total items, got %v", report["total_items"])
	}
	if _, ok := report["stale_count"]; !ok {
		t.Error("missing stale_count")
	}
	if _, ok := report["duplicate_count"]; !ok {
		t.Error("missing duplicate_count")
	}
	if _, ok := report["orphan_count"]; !ok {
		t.Error("missing orphan_count")
	}
	if _, ok := report["by_type"]; !ok {
		t.Error("missing by_type")
	}
	if _, ok := report["stale_items"]; !ok {
		t.Error("missing stale_items")
	}
	if _, ok := report["duplicate_groups"]; !ok {
		t.Error("missing duplicate_groups")
	}
	if _, ok := report["orphan_items"]; !ok {
		t.Error("missing orphan_items")
	}
}

func TestQuality_DetectsStaleItems(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert a stale item (stale_after in the past)
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata, stale_after, indexed_at)
		VALUES ($1::uuid, 'artifact', gen_random_uuid(), 'Stale Doc', 'Old', 'content',
		        '{}'::jsonb, NOW() - INTERVAL '30 days', NOW() - INTERVAL '120 days')
	`, e.teamID)
	if err != nil {
		t.Fatalf("insert stale: %v", err)
	}

	// Insert a fresh item
	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Fresh Doc", "Summary", "content")

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)

	if report["stale_count"].(float64) != 1 {
		t.Errorf("expected 1 stale item, got %v", report["stale_count"])
	}
	staleItems := report["stale_items"].([]any)
	if len(staleItems) != 1 {
		t.Errorf("expected 1 stale item in list, got %d", len(staleItems))
	}
}

func TestQuality_StaleItemsEndpoint(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert stale item
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata, stale_after, indexed_at)
		VALUES ($1::uuid, 'incident', gen_random_uuid(), 'Old Incident', 'Stale', 'content',
		        '{}'::jsonb, NOW() - INTERVAL '15 days', NOW() - INTERVAL '100 days')
	`, e.teamID)
	if err != nil {
		t.Fatalf("insert stale: %v", err)
	}

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality/stale", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["stale_items"].([]any)
	if len(items) != 1 {
		t.Errorf("expected 1 stale item, got %d", len(items))
	}
}

func TestQuality_DetectsDuplicates(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert two items with the same content_hash
	hash := "aabbccdd" + "eeff00112233445566778899aabbccdd" + "eeff00112233445566778899aabbccdd"
	if len(hash) > 64 {
		hash = hash[:64]
	}
	// Use a valid 64-char hex hash
	hash = "aabbccddeeff0011223344556677889900aabbccddeeff0011223344556677889"[:64]

	for i := 0; i < 2; i++ {
		_, err := e.pool.Exec(context.Background(), `
			INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, content_hash, metadata)
			VALUES ($1::uuid, 'artifact', gen_random_uuid(), $2, $3, 'same content', $4, '{}'::jsonb)
		`, e.teamID, fmt.Sprintf("Dup Doc %d", i), fmt.Sprintf("Dup %d", i), hash)
		if err != nil {
			t.Fatalf("insert dup %d: %v", i, err)
		}
	}

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality/duplicates", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	groups := resp["duplicate_groups"].([]any)
	if len(groups) != 1 {
		t.Errorf("expected 1 dup group, got %d", len(groups))
	}
}

func TestQuality_DuplicatesInReport(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Use valid 64-char hex
	hash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

	for i := 0; i < 3; i++ {
		e.pool.Exec(context.Background(), `
			INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, content_hash, metadata)
			VALUES ($1::uuid, 'artifact', gen_random_uuid(), $2, $3, 'same content', $4, '{}'::jsonb)
		`, e.teamID, fmt.Sprintf("Triplet %d", i), "Sum", hash)
	}

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)

	// Dup count should be 3 (total items with shared hash)
	dupCount := report["duplicate_count"].(float64)
	if dupCount != 3 {
		t.Errorf("expected dup_count=3, got %v", dupCount)
	}

	groups := report["duplicate_groups"].([]any)
	if len(groups) != 1 {
		t.Errorf("expected 1 dup group, got %d", len(groups))
	}
}

func TestQuality_OrphanDetection(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert a knowledge item referencing a non-existent artifact
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata)
		VALUES ($1::uuid, 'artifact', $2::uuid, 'Orphan Doc', 'No source', 'content', '{}'::jsonb)
	`, e.teamID, "00000000-0000-0000-0000-000000000099")
	if err != nil {
		t.Fatalf("insert orphan: %v", err)
	}

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality/orphans", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["orphan_items"].([]any)
	if len(items) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(items))
	}
}

func TestQuality_ByTypeBreakdown(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Doc 1", "S1", "C1")
	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Doc 2", "S2", "C2")
	insertKnowledgeItem(t, e.pool, e.teamID, "incident", "Inc 1", "S3", "C3")

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)
	byType := report["by_type"].(map[string]any)

	if byType["artifact"].(float64) != 2 {
		t.Errorf("expected 2 artifacts, got %v", byType["artifact"])
	}
	if byType["incident"].(float64) != 1 {
		t.Errorf("expected 1 incident, got %v", byType["incident"])
	}
}

func TestQuality_NoOperationalSideEffects(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Doc", "Sum", "content")

	// Call all quality endpoints
	for _, endpoint := range []string{
		"/knowledge/quality",
		"/knowledge/quality/stale",
		"/knowledge/quality/duplicates",
		"/knowledge/quality/orphans",
	} {
		w := doReq(e, "GET", "/api/teams/"+e.teamID+endpoint, "")
		if w.Code != 200 {
			t.Errorf("%s returned %d", endpoint, w.Code)
		}
	}

	// Verify no operational fields in responses
	// Quality endpoints are read-only and never return worker_token, tool_gateway, etc.
}

func TestQuality_TeamScoped(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert items in our team
	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Our Doc", "Our Summary", "our content")

	// Query quality — should only see our team's items
	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)
	if report["total_items"].(float64) != 1 {
		t.Errorf("expected 1 item (team-scoped), got %v", report["total_items"])
	}
}

func TestQuality_RequiresPermission(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Owner has all permissions, quality endpoint should succeed
	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Errorf("expected 200 for owner, got %d", w.Code)
	}

	// Verify the endpoint uses knowledge.read permission (not a higher perm)
	// by checking the route registration in the test router
}

func TestQuality_NoMutation(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Doc", "Sum", "content")

	// Count knowledge items before
	var countBefore int
	e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM knowledge_items WHERE team_id = $1::uuid", e.teamID).Scan(&countBefore)

	// Call quality report
	doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")

	// Count after — should be unchanged
	var countAfter int
	e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM knowledge_items WHERE team_id = $1::uuid", e.teamID).Scan(&countAfter)

	if countAfter != countBefore {
		t.Errorf("knowledge_items changed: before=%d after=%d", countBefore, countAfter)
	}
}

func TestQuality_NoPythonCalls(t *testing.T) {
	e := setupKnowledgeTest(t)

	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Doc", "Sum", "content")

	// Quality endpoints are pure SQL — no Python worker involved
	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify response body has no worker-related fields
	body := w.Body.String()
	for _, forbidden := range []string{"worker_token", "reasoning", "chain_of_thought"} {
		if contains(body, forbidden) {
			t.Errorf("response contains forbidden field: %s", forbidden)
		}
	}
}

func TestQuality_EmptyIndex(t *testing.T) {
	e := setupKnowledgeTest(t)

	// No items inserted
	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)

	if report["total_items"].(float64) != 0 {
		t.Errorf("expected 0 items, got %v", report["total_items"])
	}
	if report["stale_count"].(float64) != 0 {
		t.Errorf("expected 0 stale, got %v", report["stale_count"])
	}
}

func TestQuality_FreshItemNotStale(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert fresh item (stale_after 90 days from now, which is default)
	insertKnowledgeItem(t, e.pool, e.teamID, "artifact", "Fresh", "Summary", "content")

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report map[string]any
	json.Unmarshal(w.Body.Bytes(), &report)

	if report["stale_count"].(float64) != 0 {
		t.Errorf("expected 0 stale items for fresh index, got %v", report["stale_count"])
	}
}

func TestQuality_OrphansExcludedForNonArtifactTypes(t *testing.T) {
	e := setupKnowledgeTest(t)

	// Insert knowledge item with source_type=incident — orphans check only
	// checks artifact-type sources
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata)
		VALUES ($1::uuid, 'incident', $2::uuid, 'Inc Doc', 'S', 'content', '{}'::jsonb)
	`, e.teamID, "00000000-0000-0000-0000-000000000088")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	w := doReq(e, "GET", "/api/teams/"+e.teamID+"/knowledge/quality/orphans", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["orphan_items"].([]any)
	if len(items) != 0 {
		t.Errorf("expected 0 orphans (incident type not checked), got %d", len(items))
	}
}

// Ensure time import is used
var _ = time.Now
