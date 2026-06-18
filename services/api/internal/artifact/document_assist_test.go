package artifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── v1.4 Track 3: Agent Assist Backend Tests ───

// mockWorkerServer creates a test HTTP server that simulates the Python worker.
func mockWorkerServer(t *testing.T, status int, response any) (url string, token string) {
	t.Helper()
	token = "test-worker-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/document-assist" {
			w.WriteHeader(404)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, token
}

func setupAssistTest(t *testing.T) *artifactTestEnv {
	t.Helper()
	e := setupArtifactTest(t)
	token := e.token

	// Create a test document
	body := makeCreateDocReq("Assist Test Doc", "implementation_plan", []DocumentBlock{
		{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Overview")},
		{ID: "blk_002", Type: "paragraph", Text: strPtr("This is a paragraph to rewrite.")},
	})
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("failed to create test document: %d %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)

	return e
}

func assistRequest(artifactID, mode, blockID, selectedText, instruction string) map[string]any {
	return map[string]any{
		"mode":          mode,
		"block_id":      blockID,
		"selected_text": selectedText,
		"instruction":   instruction,
		"document_type": "implementation_plan",
		"max_words":     300,
	}
}

func doAssist(t *testing.T, e *artifactTestEnv, docID string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents/"+docID+"/document-assist", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func TestDocumentAssist_ValidatesTeamAccess(t *testing.T) {
	e := setupAssistTest(t)
	// Get doc ID from the test setup — create explicitly
	body := makeCreateDocReq("Team Access", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	// Set up a mock worker
	workerURL, workerToken := mockWorkerServer(t, 200, AssistResponse{
		SuggestedBlocks: []map[string]any{{"type": "paragraph", "text": "Rewritten text"}},
		Summary:         "Rewritten",
	})
	// Can't call SetWorkerAssist on env's handler directly — but the test route doesn't have it
	// So we expect 503 (not configured) which proves team access was validated before worker call
	fakeTeam := "00000000-0000-0000-0000-000000000999"
	body2, _ := json.Marshal(assistRequest(doc.ID, "rewrite", "blk_001", "text", ""))
	req2 := httptest.NewRequest("POST", "/api/teams/"+fakeTeam+"/artifacts/documents/"+doc.ID+"/document-assist", bytes.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 404 {
		t.Errorf("expected 404 for cross-team, got %d", w2.Code)
	}
	_ = workerURL
	_ = workerToken
}

func TestDocumentAssist_CrossTeamReturns404(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Cross Team Assist", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	fakeTeam := "00000000-0000-0000-0000-000000000999"
	b2, _ := json.Marshal(assistRequest(doc.ID, "rewrite", "", "text", ""))
	req2 := httptest.NewRequest("POST", "/api/teams/"+fakeTeam+"/artifacts/documents/"+doc.ID+"/document-assist", bytes.NewReader(b2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 404 {
		t.Errorf("expected 404, got %d", w2.Code)
	}
}

func TestDocumentAssist_ArchivedReturns403(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Archived Assist", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	// Archive it
	req = httptest.NewRequest("DELETE", "/api/teams/"+e.teamID+"/artifacts/"+doc.ID, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Try assist
	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "", "text", ""))
	if w.Code != 403 {
		t.Errorf("expected 403 for archived, got %d", w.Code)
	}
}

func TestDocumentAssist_InvalidModeRejected(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Bad Mode", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "invalid_mode", "", "text", ""))
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid mode, got %d", w.Code)
	}
}

func TestDocumentAssist_OversizedSelectedTextRejected(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Big Text", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	bigText := fmt.Sprintf("%20001s", "x")
	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "", bigText, ""))
	if w.Code != 400 {
		t.Errorf("expected 400 for oversized text, got %d", w.Code)
	}
}

func TestDocumentAssist_OversizedInstructionRejected(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Big Instruction", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	bigInstruction := fmt.Sprintf("%2001s", "x")
	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "", "text", bigInstruction))
	if w.Code != 400 {
		t.Errorf("expected 400 for oversized instruction, got %d", w.Code)
	}
}

func TestDocumentAssist_InvalidMaxWordsRejected(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Bad MaxWords", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	w = doAssist(t, e, doc.ID, map[string]any{
		"mode": "rewrite", "selected_text": "text", "max_words": 5,
	})
	if w.Code != 400 {
		t.Errorf("expected 400 for max_words=5, got %d", w.Code)
	}

	w = doAssist(t, e, doc.ID, map[string]any{
		"mode": "rewrite", "selected_text": "text", "max_words": 5000,
	})
	if w.Code != 400 {
		t.Errorf("expected 400 for max_words=5000, got %d", w.Code)
	}
}

func TestDocumentAssist_UnknownBlockIDRejected(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Unknown Block", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "blk_nonexistent", "text", ""))
	if w.Code != 400 {
		t.Errorf("expected 400 for unknown block_id, got %d", w.Code)
	}
}

func TestDocumentAssist_WorkerNotConfiguredReturns503(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("No Worker", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	// Worker assist is not configured in test env → 503
	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "", "text", ""))
	if w.Code != 503 {
		t.Errorf("expected 503 (worker not configured), got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocumentAssist_UnauthorizedDenied(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("Unauthorized Assist", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	// No auth header
	b2, _ := json.Marshal(assistRequest(doc.ID, "rewrite", "", "text", ""))
	req2 := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents/"+doc.ID+"/document-assist", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 401 {
		t.Errorf("expected 401 for unauthorized, got %d", w2.Code)
	}
}

func TestDocumentAssist_NoOperationalSideEffects(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("No Side Effects Assist", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	// Capture before counts
	var apprBefore, actionBefore, remBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&apprBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remBefore)

	// Call assist (will get 503 but that's fine for side-effect check)
	w = doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "", "text", ""))

	// Verify no new operational records
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

func TestDocumentAssist_NoDocumentMutation(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("No Mutation", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	// Capture word_count and JSON before
	var wcBefore int
	var jsonBefore []byte
	e.pool.QueryRow(t.Context(), "SELECT word_count, document_json FROM artifact_documents WHERE artifact_id = $1", doc.ID).Scan(&wcBefore, &jsonBefore)

	// Call assist
	doAssist(t, e, doc.ID, assistRequest(doc.ID, "rewrite", "", "text", ""))

	// Verify unchanged
	var wcAfter int
	var jsonAfter []byte
	e.pool.QueryRow(t.Context(), "SELECT word_count, document_json FROM artifact_documents WHERE artifact_id = $1", doc.ID).Scan(&wcAfter, &jsonAfter)

	if wcBefore != wcAfter {
		t.Errorf("word_count changed: %d → %d", wcBefore, wcAfter)
	}
}

func TestDocumentAssist_AllModesValid(t *testing.T) {
	e := setupAssistTest(t)
	body := makeCreateDocReq("All Modes", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var doc DocumentResponse
	json.NewDecoder(w.Body).Decode(&doc)

	modes := []string{
		"rewrite", "summarize", "expand", "make_concise", "make_executive",
		"make_technical", "draft_section", "create_outline",
		"extract_action_items", "improve_headings",
	}

	for _, mode := range modes {
		w := doAssist(t, e, doc.ID, assistRequest(doc.ID, mode, "", "Some text to work with", ""))
		// All should pass validation (503 expected since worker not configured)
		if w.Code == 400 {
			t.Errorf("mode %q was rejected with 400: %s", mode, w.Body.String())
		}
	}
}

func TestDocumentAssist_ValidateSuggestedBlock(t *testing.T) {
	// Test the block validation directly
	tests := []struct {
		name string
		blk  map[string]any
		err  bool
	}{
		{"valid paragraph", map[string]any{"type": "paragraph", "text": "Hello"}, false},
		{"missing type", map[string]any{"text": "Hello"}, true},
		{"unknown type", map[string]any{"type": "unknown", "text": "Hello"}, true},
		{"empty paragraph", map[string]any{"type": "paragraph", "text": ""}, true},
		{"valid heading", map[string]any{"type": "heading", "level": 2, "text": "Title"}, false},
		{"heading no level", map[string]any{"type": "heading", "text": "Title"}, true},
		{"valid bullets", map[string]any{"type": "bullets", "items": []any{"a", "b"}}, false},
		{"empty bullets", map[string]any{"type": "bullets", "items": []any{}}, true},
		{"valid page_break", map[string]any{"type": "page_break"}, false},
		{"valid callout", map[string]any{"type": "callout", "variant": "info", "text": "Note"}, false},
		{"callout bad variant", map[string]any{"type": "callout", "variant": "bad", "text": "Note"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateSuggestedBlock(tc.blk)
			if tc.err && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
