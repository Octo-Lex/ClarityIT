package artifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── v1.4 Track 1: Native Document Artifact Tests ───

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// makeDocJSON creates a valid document JSON for testing.
func makeDocJSON(title, docType string, blocks []DocumentBlock) DocumentJSON {
	return DocumentJSON{
		SchemaVersion: 1,
		Title:         title,
		DocumentType:  docType,
		Blocks:        blocks,
	}
}

func basicBlocks() []DocumentBlock {
	return []DocumentBlock{
		{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Overview")},
		{ID: "blk_002", Type: "paragraph", Text: strPtr("This document describes the implementation approach.")},
	}
}

func makeCreateDocReq(title, docType string, blocks []DocumentBlock) map[string]any {
	return map[string]any{
		"title":         title,
		"document_type": docType,
		"document_json": map[string]any{
			"schema_version": 1,
			"title":          title,
			"document_type":  docType,
			"blocks":         blocks,
		},
	}
}

// blockToMap converts a DocumentBlock to a map for JSON encoding in tests.
func blockToMap(blk DocumentBlock) map[string]any {
	m := map[string]any{
		"id":   blk.ID,
		"type": blk.Type,
	}
	if blk.Level != nil {
		m["level"] = *blk.Level
	}
	if blk.Text != nil {
		m["text"] = *blk.Text
	}
	if blk.Items != nil {
		m["items"] = blk.Items
	}
	if blk.Headers != nil {
		m["headers"] = blk.Headers
	}
	if blk.Rows != nil {
		// Convert [][]string to [][]any for JSON
		rows := make([][]any, len(blk.Rows))
		for i, row := range blk.Rows {
			r := make([]any, len(row))
			for j, cell := range row {
				r[j] = cell
			}
			rows[i] = r
		}
		m["rows"] = rows
	}
	if blk.Variant != nil {
		m["variant"] = *blk.Variant
	}
	return m
}

func TestDocument_Create(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	body := makeCreateDocReq("Implementation Plan", "implementation_plan",
		[]DocumentBlock{
			{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Overview")},
			{ID: "blk_002", Type: "paragraph", Text: strPtr("This is a test document with several words in it.")},
			{ID: "blk_003", Type: "bullets", Items: []string{"First item", "Second item"}},
		},
	)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ArtifactType != "document" {
		t.Errorf("expected artifact_type=document, got %s", resp.ArtifactType)
	}
	if resp.DocumentType != "implementation_plan" {
		t.Errorf("expected document_type=implementation_plan, got %s", resp.DocumentType)
	}
	if resp.SchemaVersion != 1 {
		t.Errorf("expected schema_version=1, got %d", resp.SchemaVersion)
	}
	if resp.WordCount == 0 {
		t.Error("expected non-zero word_count")
	}
	if resp.ID == "" {
		t.Error("expected non-empty artifact ID")
	}
	if resp.Title != "Implementation Plan" {
		t.Errorf("expected title 'Implementation Plan', got %s", resp.Title)
	}
	if resp.Status != "draft" {
		t.Errorf("expected status=draft, got %s", resp.Status)
	}
}

func TestDocument_CreateCreatesArtifactAndDocumentRows(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	body := makeCreateDocReq("Test Doc", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Verify artifact row exists
	var artifactType string
	err := e.pool.QueryRow(req.Context(),
		"SELECT artifact_type FROM artifacts WHERE id = $1", resp.ID).Scan(&artifactType)
	if err != nil {
		t.Fatalf("artifact row not found: %v", err)
	}
	if artifactType != "document" {
		t.Errorf("expected artifact_type=document, got %s", artifactType)
	}

	// Verify artifact_documents row exists
	var docType string
	err = e.pool.QueryRow(req.Context(),
		"SELECT document_type FROM artifact_documents WHERE artifact_id = $1", resp.ID).Scan(&docType)
	if err != nil {
		t.Fatalf("artifact_documents row not found: %v", err)
	}
	if docType != "general_document" {
		t.Errorf("expected document_type=general_document, got %s", docType)
	}
}

func TestDocument_WordCountComputedServerSide(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	blocks := []DocumentBlock{
		{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Three Word Title")},
		{ID: "blk_002", Type: "paragraph", Text: strPtr("Exactly five words here now.")},
	}
	body := makeCreateDocReq("Word Count Test", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// "Three Word Title" = 3 words + "Exactly five words here now." = 5 words = 8 total
	if resp.WordCount != 8 {
		t.Errorf("expected word_count=8, got %d", resp.WordCount)
	}
}

func TestDocument_Get(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create a document first
	body := makeCreateDocReq("Get Test Doc", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Now GET it
	req = httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/artifacts/documents/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, resp.ID)
	}
	if resp.Title != "Get Test Doc" {
		t.Errorf("expected title 'Get Test Doc', got %s", resp.Title)
	}
}

func TestDocument_List(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create two documents
	for _, title := range []string{"Doc One", "Doc Two"} {
		body := makeCreateDocReq(title, "general_document", basicBlocks())
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
		}
	}

	// List documents
	req := httptest.NewRequest("GET", "/api/teams/"+e.teamID+"/artifacts/documents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var docs []DocumentResponse
	json.NewDecoder(w.Body).Decode(&docs)

	if len(docs) < 2 {
		t.Errorf("expected at least 2 documents, got %d", len(docs))
	}

	// Verify all returned items are document type
	for _, doc := range docs {
		if doc.ArtifactType != "document" {
			t.Errorf("expected artifact_type=document, got %s", doc.ArtifactType)
		}
	}
}

func TestDocument_PatchUpdatesTitle(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create
	body := makeCreateDocReq("Original Title", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Patch title
	patchBody, _ := json.Marshal(map[string]any{
		"title": "Updated Title",
	})
	req = httptest.NewRequest("PATCH", "/api/teams/"+e.teamID+"/artifacts/documents/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", resp.Title)
	}
}

func TestDocument_PatchUpdatesDocumentJSON(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create
	body := makeCreateDocReq("JSON Update Test", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Patch with new blocks
	newBlocks := map[string]any{
		"schema_version": 1,
		"title":          "JSON Update Test",
		"document_type":  "general_document",
		"blocks": []map[string]any{
			{"id": "blk_new1", "type": "heading", "level": 1, "text": "Updated Heading"},
			{"id": "blk_new2", "type": "paragraph", "text": "Updated paragraph text with more words than before."},
		},
	}
	patchBody, _ := json.Marshal(map[string]any{
		"document_json": newBlocks,
	})
	req = httptest.NewRequest("PATCH", "/api/teams/"+e.teamID+"/artifacts/documents/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.WordCount == created.WordCount {
		t.Error("expected word_count to change after JSON update")
	}
}

func TestDocument_PatchRecomputesWordCount(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create with 2 blocks
	blocks := []DocumentBlock{
		{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Original Heading")},
		{ID: "blk_002", Type: "paragraph", Text: strPtr("Short text.")},
	}
	body := makeCreateDocReq("Word Count Recompute", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Patch with more text
	newBlocks := map[string]any{
		"schema_version": 1,
		"title":          "Word Count Recompute",
		"document_type":  "general_document",
		"blocks": []map[string]any{
			{"id": "blk_001", "type": "heading", "level": 1, "text": "Updated Heading With More Words Now"},
			{"id": "blk_002", "type": "paragraph", "text": "This paragraph has significantly more words than the original short text did."},
		},
	}
	patchBody, _ := json.Marshal(map[string]any{
		"document_json": newBlocks,
	})
	req = httptest.NewRequest("PATCH", "/api/teams/"+e.teamID+"/artifacts/documents/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.WordCount <= created.WordCount {
		t.Errorf("expected word_count to increase: was %d, now %d", created.WordCount, resp.WordCount)
	}
}

func TestDocument_RejectsInvalidDocumentType(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	body := makeCreateDocReq("Bad Type Doc", "invalid_type", basicBlocks())
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_RejectsSchemaVersionNot1(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Send schema_version=2 in the document_json
	body := map[string]any{
		"title":         "Bad Schema Doc",
		"document_type": "general_document",
		"document_json": map[string]any{
			"schema_version": 2,
			"title":          "Bad Schema Doc",
			"document_type":  "general_document",
			"blocks":         basicBlocks(),
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for schema_version=2, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_RejectsUnknownBlockType(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	blocks := []DocumentBlock{
		{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Title")},
		{ID: "blk_002", Type: "unknown_type", Text: strPtr("Unknown block")},
	}
	body := makeCreateDocReq("Unknown Block Doc", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for unknown block type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_RejectsMalformedHeading(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// heading with missing level
	blocks := []DocumentBlock{
		{ID: "blk_001", Type: "heading", Text: strPtr("No level")},
	}
	body := makeCreateDocReq("Bad Heading Doc", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for malformed heading, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_RejectsMalformedTable(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// table with row that has wrong number of columns
	blocks := []DocumentBlock{
		{ID: "blk_001", Type: "heading", Level: intPtr(1), Text: strPtr("Table Test")},
		{
			ID:      "blk_002",
			Type:    "table",
			Headers: []string{"A", "B"},
			Rows:    [][]string{{"1"}}, // only 1 cell, should be 2
		},
	}
	body := makeCreateDocReq("Bad Table Doc", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for malformed table, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_RejectsDuplicateBlockIDs(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	blocks := []DocumentBlock{
		{ID: "blk_dup", Type: "heading", Level: intPtr(1), Text: strPtr("First")},
		{ID: "blk_dup", Type: "paragraph", Text: strPtr("Duplicate ID")},
	}
	body := makeCreateDocReq("Dup ID Doc", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for duplicate block IDs, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_RejectsOversizedDocument(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create 501 blocks (max is 500)
	var blocks []DocumentBlock
	for i := 0; i < 501; i++ {
		blocks = append(blocks, DocumentBlock{
			ID:   fmt.Sprintf("blk_%03d", i),
			Type: "paragraph",
			Text: strPtr("x"),
		})
	}
	body := makeCreateDocReq("Oversized Doc", "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for oversized document, got %d", w.Code)
	}
}

func TestDocument_CrossTeamGet404(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create in team A
	body := makeCreateDocReq("Cross Team Doc", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Try to GET from a different team UUID
	fakeTeamID := "00000000-0000-0000-0000-000000000999"
	req = httptest.NewRequest("GET", "/api/teams/"+fakeTeamID+"/artifacts/documents/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for cross-team GET, got %d", w.Code)
	}
}

func TestDocument_CrossTeamPatch404(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create in team A
	body := makeCreateDocReq("Cross Team Patch", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Try to PATCH from a different team
	fakeTeamID := "00000000-0000-0000-0000-000000000999"
	patchBody, _ := json.Marshal(map[string]any{"title": "Hacked"})
	req = httptest.NewRequest("PATCH", "/api/teams/"+fakeTeamID+"/artifacts/documents/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404 for cross-team PATCH, got %d", w.Code)
	}
}

func TestDocument_ArchivedPatchReturns403(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Create
	body := makeCreateDocReq("To Archive", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var created DocumentResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Archive via existing DELETE endpoint
	req = httptest.NewRequest("DELETE", "/api/teams/"+e.teamID+"/artifacts/"+created.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// Try to PATCH the archived document
	patchBody, _ := json.Marshal(map[string]any{"title": "Should Fail"})
	req = httptest.NewRequest("PATCH", "/api/teams/"+e.teamID+"/artifacts/documents/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for archived PATCH, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)

	body := makeCreateDocReq("Unauthorized Doc", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for unauthorized request, got %d", w.Code)
	}
}

func TestDocument_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Capture counts BEFORE creating the document
	var approvalBefore, actionBefore, remBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&approvalBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remBefore)

	// Create a document
	body := makeCreateDocReq("No Side Effects", "general_document", basicBlocks())
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Verify no NEW approval_requests created
	var approvalAfter int
	e.pool.QueryRow(req.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&approvalAfter)
	if approvalAfter != approvalBefore {
		t.Errorf("expected %d approval_requests (unchanged), got %d", approvalBefore, approvalAfter)
	}

	// Verify no NEW asset_actions created
	var actionAfter int
	e.pool.QueryRow(req.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionAfter)
	if actionAfter != actionBefore {
		t.Errorf("expected %d asset_actions (unchanged), got %d", actionBefore, actionAfter)
	}

	// Verify no NEW remediation_proposals created
	var remAfter int
	e.pool.QueryRow(req.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remAfter)
	if remAfter != remBefore {
		t.Errorf("expected %d remediation_proposals (unchanged), got %d", remBefore, remAfter)
	}

	// Verify last_exported_storage_object_id is NULL
	var lastExported *string
	e.pool.QueryRow(req.Context(),
		"SELECT last_exported_storage_object_id::text FROM artifact_documents WHERE artifact_id = $1",
		resp.ID).Scan(&lastExported)
	if lastExported != nil {
		t.Errorf("expected NULL last_exported_storage_object_id, got %s", *lastExported)
	}
}

func TestDocument_AllBlockTypesValid(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	// Test all 8 block types in one document
	blocks := []DocumentBlock{
		{ID: "blk_h", Type: "heading", Level: intPtr(2), Text: strPtr("Heading Text")},
		{ID: "blk_p", Type: "paragraph", Text: strPtr("Paragraph text here.")},
		{ID: "blk_b", Type: "bullets", Items: []string{"Bullet one", "Bullet two"}},
		{ID: "blk_n", Type: "numbered_list", Items: []string{"First", "Second", "Third"}},
		{ID: "blk_t", Type: "table", Headers: []string{"Col A", "Col B"}, Rows: [][]string{{"1", "2"}, {"3", "4"}}},
		{ID: "blk_q", Type: "quote", Text: strPtr("A quoted passage.")},
		{ID: "blk_c", Type: "callout", Variant: strPtr("info"), Text: strPtr("Note this information.")},
		{ID: "blk_pb", Type: "page_break"},
	}
	body := makeCreateDocReq("All Block Types", "architecture_doc", blocks)
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201 for all block types, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocument_AllDocumentTypesValid(t *testing.T) {
	e := setupArtifactTest(t)
	token := e.token

	validTypes := []string{
		"general_document", "decision_memo", "implementation_plan",
		"incident_summary", "training_doc", "architecture_doc",
		"project_report", "status_report", "meeting_summary", "executive_brief",
	}

	for _, dt := range validTypes {
		body := makeCreateDocReq("Type Test: "+dt, dt, basicBlocks())
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)

		if w.Code != 201 {
			t.Errorf("document_type %q: expected 201, got %d: %s", dt, w.Code, w.Body.String())
		}
	}
}

// Ensure imports are used
var _ = http.MethodPost
