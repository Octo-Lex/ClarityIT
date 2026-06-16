package artifact

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── v1.4 Track 5: Native Document Template Tests ───

func TestDocTemplate_StructuredSystemTemplatesSeeded(t *testing.T) {
	e := setupArtifactTest(t)
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_templates WHERE is_system = true AND template_format = 'document_json'").Scan(&count)
	if count < 7 {
		t.Errorf("expected at least 7 system document_json templates, got %d", count)
	}
}

func TestDocTemplate_MarkdownTemplatesPreserved(t *testing.T) {
	e := setupArtifactTest(t)
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_templates WHERE is_system = true AND template_format = 'markdown'").Scan(&count)
	if count < 6 {
		t.Errorf("expected at least 6 system markdown templates, got %d", count)
	}
}

func TestDocTemplate_ListReturnsBothFormats(t *testing.T) {
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
	hasMarkdown := false
	hasDocJSON := false
	for _, tmpl := range list {
		if tmpl["template_format"] == "markdown" {
			hasMarkdown = true
		}
		if tmpl["template_format"] == "document_json" {
			hasDocJSON = true
		}
	}
	if !hasMarkdown {
		t.Error("expected markdown templates in list")
	}
	if !hasDocJSON {
		t.Error("expected document_json templates in list")
	}
}

func TestDocTemplate_ListHasTemplateFormatField(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	for _, tmpl := range list {
		if _, ok := tmpl["template_format"]; !ok {
			t.Error("template missing template_format field")
			break
		}
	}
}

func TestDocTemplate_ListTypeFilterStillWorks(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifact-templates?type=document", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	for _, tmpl := range list {
		if tmpl["template_type"] != "document" {
			t.Errorf("expected all templates to be type=document, found %v", tmpl["template_type"])
			break
		}
	}
}

func TestDocTemplate_CreateCustomDocJSONTemplate(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{
		"template_type": "document",
		"name": "Custom Doc Template",
		"description": "A test structured template",
		"template_format": "document_json",
		"schema_version": 1,
		"document_json": {
			"schema_version": 1,
			"title": "Custom Doc",
			"document_type": "general_document",
			"blocks": [
				{"id": "b1", "type": "heading", "level": 1, "text": "Title"},
				{"id": "b2", "type": "paragraph", "text": "Body text."}
			]
		}
	}`
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
	if resp["template_format"] != "document_json" {
		t.Errorf("expected template_format=document_json, got %v", resp["template_format"])
	}
}

func TestDocTemplate_RejectInvalidTemplateFormat(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"template_type":"document","name":"Bad","template_format":"html","content_markdown":"# hi"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDocTemplate_RejectDocJSONScalar(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"template_type":"document","name":"Scalar","template_format":"document_json","document_json":"not an object"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for scalar document_json, got %d", w.Code)
	}
}

func TestDocTemplate_RejectBadSchemaVersion(t *testing.T) {
	e := setupArtifactTest(t)
	sv := 99
	bodyMap := map[string]any{
		"template_type":   "document",
		"name":            "Bad SV",
		"template_format": "document_json",
		"schema_version":  sv,
		"document_json": map[string]any{
			"schema_version": 1,
			"title":          "T",
			"document_type":  "general_document",
			"blocks":         []map[string]any{{"id": "b1", "type": "paragraph", "text": "hi"}},
		},
	}
	bodyBytes, _ := json.Marshal(bodyMap)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(string(bodyBytes)))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for schema_version=99, got %d", w.Code)
	}
}

func TestDocTemplate_RejectInvalidDocBlock(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{
		"template_type": "document",
		"name": "Bad Block",
		"template_format": "document_json",
		"document_json": {
			"schema_version": 1,
			"title": "T",
			"document_type": "general_document",
			"blocks": [
				{"id": "b1", "type": "unknown_type", "text": "bad"}
			]
		}
	}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid block, got %d", w.Code)
	}
}

func TestDocTemplate_RejectAPICreatedSystemTemplate(t *testing.T) {
	e := setupArtifactTest(t)
	// The API always creates with is_system=false, so system templates can't be created via API
	body := `{"template_type":"document","name":"Fake System","content_markdown":"# hi","is_system":true}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	// is_system should always be false from API
	if resp["is_system"] == true {
		t.Error("API should not create system templates")
	}
}

func TestDocTemplate_InstantiateMarkdownPreservesBehavior(t *testing.T) {
	e := setupArtifactTest(t)
	// Create a markdown template
	body := `{"template_type":"document","name":"MD Test","content_markdown":"# Hello World"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var tmpl map[string]any
	json.Unmarshal(w.Body.Bytes(), &tmpl)

	// Instantiate it
	req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/%s/instantiate", e.teamID, tmpl["id"]), strings.NewReader(`{}`))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["content_markdown"] != "# Hello World" {
		t.Errorf("expected content_markdown preserved, got %v", resp["content_markdown"])
	}
}

func TestDocTemplate_InstantiateDocJSONCreatesArtifact(t *testing.T) {
	e := setupArtifactTest(t)
	// Use a seeded system document template
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000010/instantiate", e.teamID), strings.NewReader(`{"title":"My Decision"}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["artifact_type"] != "document" {
		t.Errorf("expected artifact_type=document, got %v", resp["artifact_type"])
	}
	if resp["source_type"] != "template" {
		t.Errorf("expected source_type=template, got %v", resp["source_type"])
	}
}

func TestDocTemplate_InstantiateCreatesArtifactDocumentsRow(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000011/instantiate", e.teamID), strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artifactID := resp["artifact_id"]

	// Verify artifact_documents row exists
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_documents WHERE artifact_id::text = $1", artifactID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 artifact_documents row, got %d", count)
	}
}

func TestDocTemplate_InstantiateComputesWordCount(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000010/instantiate", e.teamID), strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	wc, ok := resp["word_count"]
	if !ok || wc.(float64) <= 0 {
		t.Errorf("expected word_count > 0, got %v", wc)
	}
}

func TestDocTemplate_InstantiateSetsSourceTypeTemplate(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000010/instantiate", e.teamID), strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artifactID := resp["artifact_id"]

	var sourceType string
	e.pool.QueryRow(t.Context(), "SELECT source_type FROM artifacts WHERE id::text = $1", artifactID).Scan(&sourceType)
	if sourceType != "template" {
		t.Errorf("expected source_type=template, got %s", sourceType)
	}
}

func TestDocTemplate_SourceDataExcludesRawDocJSON(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000010/instantiate", e.teamID), strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artifactID := resp["artifact_id"]

	var sourceData []byte
	e.pool.QueryRow(t.Context(), "SELECT source_data FROM artifacts WHERE id::text = $1", artifactID).Scan(&sourceData)
	var sd map[string]any
	json.Unmarshal(sourceData, &sd)
	// Should have template_id and template_format but NOT document_json
	if _, hasDocJSON := sd["document_json"]; hasDocJSON {
		t.Error("source_data should not contain raw document_json")
	}
	if _, hasTemplateID := sd["template_id"]; !hasTemplateID {
		t.Error("source_data should contain template_id")
	}
}

func TestDocTemplate_CrossTeamCustomTemplate404(t *testing.T) {
	e := setupArtifactTest(t)
	// Create a team template
	body := `{"template_type":"document","name":"Team Only","content_markdown":"# hi"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var tmpl map[string]any
	json.Unmarshal(w.Body.Bytes(), &tmpl)

	// Try to access from a different (fake) team
	fakeTeam := "00000000-0000-0000-0000-000000000999"
	req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/%s/instantiate", fakeTeam, tmpl["id"]), strings.NewReader(`{}`))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 404 {
		t.Errorf("expected 404 for cross-team template, got %d", w2.Code)
	}
}

func TestDocTemplate_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000010/instantiate", e.teamID), strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDocTemplate_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	var apprBefore, actionBefore, remBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&apprBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remBefore)

	// Instantiate a document template
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifact-templates/a0000000-0000-0000-0000-000000000010/instantiate", e.teamID), strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var apprAfter, actionAfter, remAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&apprAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remAfter)

	if apprAfter != apprBefore {
		t.Errorf("approval_requests changed: %d → %d", apprBefore, apprAfter)
	}
	if actionAfter != actionBefore {
		t.Errorf("asset_actions changed: %d → %d", actionBefore, actionAfter)
	}
	if remAfter != remBefore {
		t.Errorf("remediation_proposals changed: %d → %d", remBefore, remAfter)
	}
}
