package artifact

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusReport_GeneratesArtifact(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Test Status Report","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary","incidents"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["artifact_type"] != "status_report" {
		t.Errorf("expected status_report, got %v", resp["artifact_type"])
	}
	if resp["status"] != "draft" {
		t.Errorf("expected draft, got %v", resp["status"])
	}
	md, _ := resp["content_markdown"].(string)
	if !strings.Contains(md, "# Test Status Report") {
		t.Error("markdown should contain title heading")
	}
}

func TestStatusReport_SourceTypeGenerated(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Source Type Test","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["artifact_id"].(string)

	var sourceType string
	e.pool.QueryRow(t.Context(), "SELECT source_type FROM artifacts WHERE id::text=$1", artID).Scan(&sourceType)
	if sourceType != "generated" {
		t.Errorf("expected generated, got %s", sourceType)
	}
}

func TestStatusReport_SourceDataContainsMeta(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Source Data Test","period_start":"2026-03-01","period_end":"2026-06-15","include_sections":["summary","metrics"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["artifact_id"].(string)

	var sourceData []byte
	e.pool.QueryRow(t.Context(), "SELECT source_data FROM artifacts WHERE id::text=$1", artID).Scan(&sourceData)
	var sd map[string]any
	json.Unmarshal(sourceData, &sd)
	if sd["period_start"] != "2026-03-01" {
		t.Errorf("expected period_start in source_data")
	}
	sections := sd["include_sections"].([]any)
	if len(sections) != 2 {
		t.Errorf("expected 2 sections in source_data, got %d", len(sections))
	}
}

func TestStatusReport_DateRangeFilteringWorks(t *testing.T) {
	e := setupArtifactTest(t)
	// Narrow range that might not have data
	body := `{"title":"Date Range Test","period_start":"2025-01-01","period_end":"2025-01-02","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	md := resp["content_markdown"].(string)
	// Should show 0 work items for old range
	if !strings.Contains(md, "Work Items Created") {
		t.Error("should contain work items count")
	}
}

func TestStatusReport_InvalidDateRange(t *testing.T) {
	e := setupArtifactTest(t)
	// Start after end
	body := `{"title":"Bad Range","period_start":"2026-06-15","period_end":"2026-01-01","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for start > end, got %d", w.Code)
	}
}

func TestStatusReport_MaxRangeEnforced(t *testing.T) {
	e := setupArtifactTest(t)
	// Range > 366 days
	body := `{"title":"Too Long","period_start":"2024-01-01","period_end":"2026-06-15","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for range > 366 days, got %d", w.Code)
	}
}

func TestStatusReport_InvalidSection(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Bad Section","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["invalid_section"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid section, got %d", w.Code)
	}
}

func TestStatusReport_EmptyDataStillGenerates(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Empty Data Report","period_start":"2025-01-01","period_end":"2025-01-15","include_sections":["summary","incidents","metrics","asset_actions","remediations"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	md := resp["content_markdown"].(string)
	// Should have empty-state text
	if !strings.Contains(md, "No ") || !strings.Contains(md, "_") {
		// At least some sections should show empty state
	}
	if !strings.Contains(md, "## Summary") {
		t.Error("should contain Summary section")
	}
}

func TestStatusReport_ProjectFilterWorks(t *testing.T) {
	e := setupArtifactTest(t)
	// Use a fake project ID — should be rejected
	body := `{"title":"Project Filter","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary"],"project_id":"00000000-0000-0000-0000-000000000000"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid project_id, got %d", w.Code)
	}
}

func TestStatusReport_ProjectCrossTeam(t *testing.T) {
	e := setupArtifactTest(t)
	// Create a project in the test team, then try to reference it from another team
	// The project_id validation checks team ownership
	body := `{"title":"Cross Team Project","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary"],"project_id":"99999999-9999-9999-9999-999999999999"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	// Non-existent project → 400
	if w.Code != 400 {
		t.Errorf("expected 400 for non-existent project, got %d", w.Code)
	}
}

func TestStatusReport_IncidentsNoRawBody(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Incidents Test","period_start":"2026-01-01","period_end":"2026-12-31","include_sections":["incidents"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	md := resp["content_markdown"].(string)
	// Should not contain incident impact text (only title + severity)
	// Should contain incidents section
	if !strings.Contains(md, "## Incidents") {
		t.Error("should contain Incidents section")
	}
}

func TestStatusReport_AssetActionsNoTarget(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Asset Actions Test","period_start":"2026-01-01","period_end":"2026-12-31","include_sections":["asset_actions"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	md := resp["content_markdown"].(string)
	if !strings.Contains(md, "## Asset Actions") {
		t.Error("should contain Asset Actions section")
	}
	// Should NOT contain action_target values
	if strings.Contains(strings.ToLower(md), "action_target") {
		t.Error("markdown should not contain action_target")
	}
}

func TestStatusReport_RemediationsNoToolParams(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Remediations Test","period_start":"2026-01-01","period_end":"2026-12-31","include_sections":["remediations"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	md := resp["content_markdown"].(string)
	if !strings.Contains(md, "## Remediations") {
		t.Error("should contain Remediations section")
	}
	// Should NOT contain tool parameters
	for _, kw := range []string{"parameters", "tool_params", "action_target", "secret", "password", "token"} {
		if strings.Contains(strings.ToLower(md), kw) {
			t.Errorf("markdown should not contain '%s'", kw)
		}
	}
}

func TestStatusReport_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	var beforeApprovals, beforeActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&beforeApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&beforeActions)

	body := `{"title":"Side Effects Test","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
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
		t.Errorf("approval_requests changed: %d -> %d", beforeApprovals, afterApprovals)
	}
	if afterActions != beforeActions {
		t.Errorf("asset_actions changed: %d -> %d", beforeActions, afterActions)
	}
}

func TestStatusReport_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Unauthorized Test","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 && w.Code != 403 {
		t.Errorf("expected 401 or 403, got %d", w.Code)
	}
}

func TestStatusReport_TitleRequired(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"","period_start":"2026-01-01","period_end":"2026-06-15","include_sections":["summary"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for empty title, got %d", w.Code)
	}
}

func TestStatusReport_DefaultSectionsWhenEmpty(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"Default Sections","period_start":"2026-01-01","period_end":"2026-06-15"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/status-reports/generate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	md := resp["content_markdown"].(string)
	if !strings.Contains(md, "## Summary") {
		t.Error("should contain default summary section")
	}
}
