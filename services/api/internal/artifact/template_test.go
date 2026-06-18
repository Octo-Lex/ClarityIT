package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ensureSystemTemplates re-seeds system templates if they've been wiped by
// other test packages' TRUNCATE ... CASCADE (iam/team/domain packages truncate
// the teams table which cascades to artifact_templates via FK).
func ensureSystemTemplates(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var count int
	pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM artifact_templates WHERE is_system = true").Scan(&count)
	if count >= 13 {
		return
	}
	// Re-seed all 6 markdown system templates
	templates := []struct {
		id, ttype, name, desc, content string
	}{
		{"a0000000-0000-0000-0000-000000000001", "status_report", "Weekly Status Report", "Standard weekly status", "# Weekly Status"},
		{"a0000000-0000-0000-0000-000000000002", "meeting_summary", "Meeting Summary", "Standard meeting", "# Meeting"},
		{"a0000000-0000-0000-0000-000000000003", "decision_memo", "Decision Memo", "Decision template", "# Decision"},
		{"a0000000-0000-0000-0000-000000000004", "report", "Incident Summary", "Post-incident", "# Incident"},
		{"a0000000-0000-0000-0000-000000000005", "document", "Architecture Walkthrough", "Architecture doc", "# Architecture"},
		{"a0000000-0000-0000-0000-000000000006", "training_deck", "Training Deck Prompt", "Training template", "# Training"},
	}
	for _, tmpl := range templates {
		pool.Exec(context.Background(), `
			INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, template_format, schema_version, metadata, is_system)
			VALUES ($1, NULL, $2, $3, $4, $5, 'markdown', NULL, '{}'::jsonb, true)
			ON CONFLICT (id) DO NOTHING
		`, tmpl.id, tmpl.ttype, tmpl.name, tmpl.desc, tmpl.content)
	}
	// Re-seed 7 document_json system templates
	docTemplates := []struct {
		id, name, desc, docType, docJSON string
	}{
		{"a0000000-0000-0000-0000-000000000010", "Decision Memo (Structured)", "Structured decision memo.", "decision_memo",
			`{"schema_version":1,"title":"Decision Memo","document_type":"decision_memo","blocks":[{"id":"b1","type":"heading","level":1,"text":"Decision"},{"id":"b2","type":"paragraph","text":"Context"}]}`},
		{"a0000000-0000-0000-0000-000000000011", "Implementation Plan (Structured)", "Structured implementation plan.", "implementation_plan",
			`{"schema_version":1,"title":"Implementation Plan","document_type":"implementation_plan","blocks":[{"id":"b1","type":"heading","level":1,"text":"Plan"},{"id":"b2","type":"paragraph","text":"Overview"}]}`},
		{"a0000000-0000-0000-0000-000000000012", "Incident Summary (Structured)", "Structured post-incident summary.", "incident_summary",
			`{"schema_version":1,"title":"Incident Summary","document_type":"incident_summary","blocks":[{"id":"b1","type":"heading","level":1,"text":"Incident"},{"id":"b2","type":"paragraph","text":"Summary"}]}`},
		{"a0000000-0000-0000-0000-000000000013", "Architecture Document (Structured)", "Structured architecture document.", "architecture_doc",
			`{"schema_version":1,"title":"Architecture Document","document_type":"architecture_doc","blocks":[{"id":"b1","type":"heading","level":1,"text":"Architecture"},{"id":"b2","type":"paragraph","text":"Overview"}]}`},
		{"a0000000-0000-0000-0000-000000000014", "Training Document (Structured)", "Structured training document.", "training_doc",
			`{"schema_version":1,"title":"Training Document","document_type":"training_doc","blocks":[{"id":"b1","type":"heading","level":1,"text":"Training"},{"id":"b2","type":"paragraph","text":"Introduction"}]}`},
		{"a0000000-0000-0000-0000-000000000015", "Project Report (Structured)", "Structured project report.", "project_report",
			`{"schema_version":1,"title":"Project Report","document_type":"project_report","blocks":[{"id":"b1","type":"heading","level":1,"text":"Report"},{"id":"b2","type":"paragraph","text":"Summary"}]}`},
		{"a0000000-0000-0000-0000-000000000016", "Executive Brief (Structured)", "Structured executive brief.", "executive_brief",
			`{"schema_version":1,"title":"Executive Brief","document_type":"executive_brief","blocks":[{"id":"b1","type":"heading","level":1,"text":"Brief"},{"id":"b2","type":"paragraph","text":"Summary"}]}`},
	}
	for _, dt := range docTemplates {
		if ct, err := pool.Exec(context.Background(), `
			INSERT INTO artifact_templates (id, team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system)
			VALUES ($1, NULL, 'document', $2, $3, NULL, $4::jsonb, 1, 'document_json', jsonb_build_object('doc_type', $5::text), true)
			ON CONFLICT (id) DO NOTHING
		`, dt.id, dt.name, dt.desc, dt.docJSON, dt.docType); err != nil {
			t.Logf("WARNING: failed to seed document_json template %s: %v (rows=%s)", dt.id, err, ct.String())
		}
	}
}

func TestTemplate_SystemTemplatesSeeded(t *testing.T) {
	e := setupArtifactTest(t)
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_templates WHERE is_system = true").Scan(&count)
	if count < 6 {
		t.Errorf("expected at least 6 system templates, got %d", count)
	}
}

func TestTemplate_ListReturnsSystemTemplates(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	found := false
	for _, tmpl := range list {
		if tmpl["is_system"] == true {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one system template in list")
	}
}

func TestTemplate_ListReturnsTeamTemplates(t *testing.T) {
	e := setupArtifactTest(t)
	// Create a team template first
	body := `{"template_type":"document","name":"Team Template","content_markdown":"# Hello"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201 creating team template, got %d", w.Code)
	}

	// List
	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var list []map[string]any
	json.Unmarshal(w2.Body.Bytes(), &list)
	found := false
	for _, tmpl := range list {
		if tmpl["name"] == "Team Template" {
			found = true
			if tmpl["is_system"] != false {
				t.Error("team template should not be system")
			}
		}
	}
	if !found {
		t.Error("team template not found in list")
	}
}

func TestTemplate_TypeFilterWorks(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifact-templates?type=status_report", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	for _, tmpl := range list {
		if tmpl["template_type"] != "status_report" {
			t.Errorf("filter failed: got type %v", tmpl["template_type"])
		}
	}
}

func TestTemplate_CreateCustomTeamTemplate(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"template_type":"decision_memo","name":"Custom Decision","description":"My template","content_markdown":"# Decision: [Title]"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["is_system"] != false {
		t.Error("custom template must not be system")
	}
}

func TestTemplate_RejectsInvalidType(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"template_type":"invalid_type","name":"Bad Type","content_markdown":"content"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTemplate_RejectsScalarMetadata(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"template_type":"document","name":"Bad Meta","content_markdown":"content","metadata":"scalar-string"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	// metadata is decoded into map[string]any — string scalar will fail decode
	if w.Code != 400 && w.Code != 500 {
		t.Errorf("expected 400 or 500 for scalar metadata, got %d", w.Code)
	}
}

func TestTemplate_InstantiateSystemTemplate(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"My Status Report"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000001/instantiate", e.teamID), strings.NewReader(body))
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
	if resp["title"] != "My Status Report" {
		t.Errorf("expected custom title, got %v", resp["title"])
	}
}

func TestTemplate_InstantiateTeamTemplate(t *testing.T) {
	e := setupArtifactTest(t)
	// Create team template
	body := `{"template_type":"report","name":"Quarterly Report Template","content_markdown":"# Quarterly Report\n\n## Q[Q] [Year]"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	templateID := createResp["id"].(string)

	// Instantiate it
	body2 := `{"title":"Q2 2026 Report"}`
	req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/%s/instantiate", e.teamID, templateID), strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Fatalf("expected 201, got %d", w2.Code)
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["artifact_type"] != "report" {
		t.Errorf("expected report type, got %v", resp["artifact_type"])
	}
}

func TestTemplate_InstantiatePreservesType(t *testing.T) {
	e := setupArtifactTest(t)
	// Instantiate decision_memo template
	body := `{}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000003/instantiate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["artifact_type"] != "decision_memo" {
		t.Errorf("expected decision_memo, got %v", resp["artifact_type"])
	}
}

func TestTemplate_InstantiateAllowsTitleOverride(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"title":"My Custom Title"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000002/instantiate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["title"] != "My Custom Title" {
		t.Errorf("expected overridden title, got %v", resp["title"])
	}
}

func TestTemplate_InstantiateDefaultsToTemplateName(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000005/instantiate", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["title"] != "Architecture Walkthrough" {
		t.Errorf("expected template name as title, got %v", resp["title"])
	}
}

func TestTemplate_CrossTeamAccess404(t *testing.T) {
	e := setupArtifactTest(t)
	// Create team template
	body := `{"template_type":"document","name":"Team Secret Template","content_markdown":"# Secret"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	templateID := createResp["id"].(string)

	// Try to instantiate from a different team
	body2 := `{}`
	req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/00000000-0000-0000-0000-000000000000/artifact-templates/%s/instantiate", templateID), strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 404 {
		t.Errorf("expected 404 for cross-team template access, got %d", w2.Code)
	}
}

func TestTemplate_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), nil)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 && w.Code != 403 {
		t.Errorf("expected 401 or 403, got %d", w.Code)
	}
}

func TestTemplate_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	var beforeApprovals, beforeActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&beforeApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&beforeActions)

	body := `{"title":"Side Effects Test"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000001/instantiate", e.teamID), strings.NewReader(body))
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

func TestTemplate_NameRequired(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"template_type":"document","name":"","content_markdown":"content"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}
