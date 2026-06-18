package artifact

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── Meeting Summary Tests ───

func TestMeeting_Create(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{
		"title": "Weekly Platform Sync",
		"description": "Internal team meeting",
		"meeting_date": "2026-06-15",
		"duration_minutes": 45,
		"attendees": [{"name": "Sara", "role": "Engineer"}],
		"agenda_items": [{"title": "Release status", "notes": "Reviewed v1.3 progress"}],
		"decisions": [{"text": "Proceed with Track 3", "decided_by": "Team"}],
		"action_items": [{"text": "Draft test plan", "assignee": "Ali", "due_date": "2026-06-18", "status": "open"}]
	}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["artifact"].(map[string]any)["artifact_type"] != "meeting_summary" {
		t.Errorf("expected meeting_summary type")
	}
}

func TestMeeting_CreatesArtifactRow(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Test Meeting DB","meeting_date":"2026-06-15"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["artifact"].(map[string]any)["id"].(string)

	var artType string
	e.pool.QueryRow(t.Context(), "SELECT artifact_type FROM artifacts WHERE id::text = $1", artID).Scan(&artType)
	if artType != "meeting_summary" {
		t.Errorf("expected meeting_summary, got %s", artType)
	}
}

func TestMeeting_CreatesMeetingDataRow(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Test Meeting Data Row","meeting_date":"2026-06-15","duration_minutes":30}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["artifact"].(map[string]any)["id"].(string)

	var dur *int
	e.pool.QueryRow(t.Context(), "SELECT duration_minutes FROM artifact_meeting_data WHERE artifact_id::text = $1", artID).Scan(&dur)
	if dur == nil || *dur != 30 {
		t.Errorf("expected duration_minutes=30, got %v", dur)
	}
}

func TestMeeting_GetReturnsStructuredData(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Get Test Meeting","meeting_date":"2026-06-15","attendees":[{"name":"Ali"}],"action_items":[{"text":"Follow up","assignee":"Ali","status":"open"}]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	artID := createResp["artifact"].(map[string]any)["id"].(string)

	// GET it
	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries/%s", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["title"] != "Get Test Meeting" {
		t.Errorf("wrong title")
	}
	attendees := resp["attendees"].([]any)
	if len(attendees) != 1 {
		t.Errorf("expected 1 attendee, got %d", len(attendees))
	}
}

func TestMeeting_ListReturnsOnlyMeetingSummaries(t *testing.T) {
	e := setupArtifactTest(t)
	// Create a regular artifact
	e.createArtifact(t, "document", "Regular Doc", "content")
	// Create a meeting summary
	body := `{"title":"List Test Meeting","meeting_date":"2026-06-15"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// List meeting summaries
	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var list []any
	json.Unmarshal(w2.Body.Bytes(), &list)
	for _, item := range list {
		m := item.(map[string]any)
		if m["artifact_type"] != "meeting_summary" {
			t.Errorf("list should only contain meeting_summary type")
		}
	}
}

func TestMeeting_PatchUpdatesArtifactFields(t *testing.T) {
	e := setupArtifactTest(t)
	// Create
	body := `{"title":"Patch Test Meeting","meeting_date":"2026-06-15"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	artID := createResp["artifact"].(map[string]any)["id"].(string)

	// Patch title
	patchBody := `{"title":"Patched Title"}`
	req2 := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries/%s", e.teamID, artID), strings.NewReader(patchBody))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["title"] != "Patched Title" {
		t.Errorf("expected patched title, got %v", resp["title"])
	}
}

func TestMeeting_PatchUpdatesStructuredData(t *testing.T) {
	e := setupArtifactTest(t)
	// Create
	body := `{"title":"Patch Data Test","meeting_date":"2026-06-15","attendees":[]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	artID := createResp["artifact"].(map[string]any)["id"].(string)

	// Patch attendees
	patchBody := `{"attendees":[{"name":"New Attendee"}]}`
	req2 := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries/%s", e.teamID, artID), strings.NewReader(patchBody))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	attendees := resp["attendees"].([]any)
	if len(attendees) != 1 {
		t.Errorf("expected 1 attendee after patch, got %d", len(attendees))
	}
}

func TestMeeting_RejectsNonArrayAttendees(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Bad Attendees","attendees":"not-an-array"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	// Go will unmarshal string into []any as []any containing one string — the handler
	// should handle this. The DB constraint will catch it.
	// If the field is a string, json.Unmarshal into []any will error
	if w.Code != 400 && w.Code != 500 {
		t.Errorf("expected 400 or 500 for non-array attendees, got %d", w.Code)
	}
}

func TestMeeting_RejectsInvalidDuration(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Bad Duration","duration_minutes":-5}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for negative duration, got %d", w.Code)
	}

	// Also test > 1440
	body2 := `{"title":"Bad Duration 2","duration_minutes":1500}`
	req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Errorf("expected 400 for duration > 1440, got %d", w2.Code)
	}
}

func TestMeeting_CrossTeamAccess404(t *testing.T) {
	e := setupArtifactTest(t)
	// Create a meeting in the test team
	body := `{"title":"Cross Team Test","meeting_date":"2026-06-15"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	artID := createResp["artifact"].(map[string]any)["id"].(string)

	// Try to GET from a different team
	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/00000000-0000-0000-0000-000000000000/artifacts/meeting-summaries/%s", artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 404 {
		t.Errorf("expected 404 for cross-team access, got %d", w2.Code)
	}
}

func TestMeeting_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Unauthorized Test"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 && w.Code != 403 {
		t.Errorf("expected 401 or 403, got %d", w.Code)
	}
}

func TestMeeting_NoWorkItemsCreated(t *testing.T) {
	e := setupArtifactTest(t)
	var beforeCount int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM objects WHERE object_type='work_item'").Scan(&beforeCount)

	body := `{"title":"Work Item Test","action_items":[{"text":"Create work item","assignee":"Ali"}]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var afterCount int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM objects WHERE object_type='work_item'").Scan(&afterCount)
	if afterCount != beforeCount {
		t.Errorf("work_items changed: %d -> %d", beforeCount, afterCount)
	}
}

func TestMeeting_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	var beforeApprovals, beforeActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&beforeApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&beforeActions)

	body := `{"title":"Side Effects Test","action_items":[{"text":"Do something"}]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var afterApprovals, afterActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&afterApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&afterActions)
	if afterApprovals != beforeApprovals {
		t.Errorf("approval_requests changed")
	}
	if afterActions != beforeActions {
		t.Errorf("asset_actions changed")
	}
}

func TestMeeting_SensitiveFieldsSuppressed(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Sensitive Test","attendees":[{"name":"Ali","password":"hunter2"}],"action_items":[{"text":"Send api_key=sk-123 to team"}]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/meeting-summaries", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	respBody := w.Body.String()
	if strings.Contains(respBody, "hunter2") {
		t.Error("response should not contain raw password value")
	}
	if strings.Contains(respBody, "sk-123") {
		t.Error("response should not contain raw api_key value")
	}
	if !strings.Contains(respBody, "[REDACTED]") {
		t.Error("response should contain [REDACTED] for sensitive fields")
	}
}
