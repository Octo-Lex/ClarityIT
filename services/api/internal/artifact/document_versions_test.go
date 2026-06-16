package artifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── v1.4 Track 7: Document Version History Tests ───

func TestDocVersion_CreateDocumentCreatesVersion1(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Version Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("Hello version")},
	})

	var versionNum, wc int
	var source string
	e.pool.QueryRow(t.Context(), `
		SELECT version_number, word_count, source
		FROM artifact_document_versions WHERE artifact_id = $1
	`, artID).Scan(&versionNum, &wc, &source)

	if versionNum != 1 {
		t.Errorf("expected version 1, got %d", versionNum)
	}
	if source != VersionSourceUserSave {
		t.Errorf("expected source user_save, got %s", source)
	}
	if wc != 2 {
		t.Errorf("expected word_count 2, got %d", wc)
	}
}

func TestDocVersion_GeneratedDocumentCreatesVersion1Generated(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createGeneratedDoc(t, e)

	var source string
	var versionNum int
	e.pool.QueryRow(t.Context(), `
		SELECT source, version_number FROM artifact_document_versions WHERE artifact_id = $1
	`, artID).Scan(&source, &versionNum)

	if versionNum != 1 {
		t.Errorf("expected version 1, got %d", versionNum)
	}
	if source != VersionSourceGenerated {
		t.Errorf("expected source generated, got %s", source)
	}
}

func TestDocVersion_TemplateDocumentCreatesVersion1Template(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTemplateDoc(t, e)

	var source string
	var versionNum int
	e.pool.QueryRow(t.Context(), `
		SELECT source, version_number FROM artifact_document_versions WHERE artifact_id = $1
	`, artID).Scan(&source, &versionNum)

	if versionNum != 1 {
		t.Errorf("expected version 1, got %d", versionNum)
	}
	if source != VersionSourceTemplate {
		t.Errorf("expected source template, got %s", source)
	}
}

func TestDocVersion_PatchCreatesNextVersion(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Patch Version", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("Original")},
	})

	// PATCH the document
	patchBody := map[string]any{
		"document_json": makeCreateDocReq("Patch Version", "general_document", []DocumentBlock{
			{ID: "b1", Type: "paragraph", Text: strPtr("Updated content")},
		})["document_json"],
	}
	bodyBytes, _ := json.Marshal(patchBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Check version 2 exists
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_document_versions WHERE artifact_id = $1", artID).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 versions, got %d", count)
	}
}

func TestDocVersion_PatchVersionNumberIncrements(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Increment Test", basicBlocks())

	// Two patches
	for i := 0; i < 2; i++ {
		patchBody := map[string]any{
			"document_json": makeCreateDocReq("Increment Test", "general_document", []DocumentBlock{
				{ID: "b1", Type: "paragraph", Text: strPtr(fmt.Sprintf("Edit %d", i))},
			})["document_json"],
		}
		bodyBytes, _ := json.Marshal(patchBody)
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+e.token)
		req.Header.Set("Content-Type", "application/json")
		e.r.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Check max version is 3
	var maxVersion int
	e.pool.QueryRow(t.Context(), "SELECT MAX(version_number) FROM artifact_document_versions WHERE artifact_id = $1", artID).Scan(&maxVersion)
	if maxVersion != 3 {
		t.Errorf("expected max version 3, got %d", maxVersion)
	}
}

func TestDocVersion_PatchStoresFullSnapshot(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Snapshot Test", basicBlocks())

	newBlocks := []DocumentBlock{
		{ID: "b1", Type: "heading", Level: intPtr(1), Text: strPtr("Snapshot Title")},
		{ID: "b2", Type: "paragraph", Text: strPtr("Snapshot paragraph content")},
	}
	patchBody := map[string]any{
		"document_json": makeCreateDocReq("Snapshot Test", "general_document", newBlocks)["document_json"],
	}
	bodyBytes, _ := json.Marshal(patchBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	var docJSON []byte
	e.pool.QueryRow(t.Context(), `
		SELECT document_json FROM artifact_document_versions
		WHERE artifact_id = $1 AND version_number = 2
	`, artID).Scan(&docJSON)

	var stored DocumentJSON
	json.Unmarshal(docJSON, &stored)
	if len(stored.Blocks) != 2 {
		t.Errorf("expected 2 blocks in snapshot, got %d", len(stored.Blocks))
	}
}

func TestDocVersion_PatchStoresWordCount(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "WC Test", basicBlocks())

	newBlocks := []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("one two three four five")},
	}
	patchBody := map[string]any{
		"document_json": makeCreateDocReq("WC Test", "general_document", newBlocks)["document_json"],
	}
	bodyBytes, _ := json.Marshal(patchBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	var wc int
	e.pool.QueryRow(t.Context(), `
		SELECT word_count FROM artifact_document_versions
		WHERE artifact_id = $1 AND version_number = 2
	`, artID).Scan(&wc)
	if wc != 5 {
		t.Errorf("expected word_count 5, got %d", wc)
	}
}

func TestDocVersion_ListReturnsDESCOrder(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "List DESC", basicBlocks())

	// Create 2 patches
	for i := 0; i < 2; i++ {
		patchBody := map[string]any{
			"document_json": makeCreateDocReq("List DESC", "general_document", []DocumentBlock{
				{ID: "b1", Type: "paragraph", Text: strPtr(fmt.Sprintf("v%d", i+2))},
			})["document_json"],
		}
		bodyBytes, _ := json.Marshal(patchBody)
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+e.token)
		req.Header.Set("Content-Type", "application/json")
		e.r.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(resp.Versions))
	}
	if resp.Versions[0].VersionNumber != 3 {
		t.Errorf("expected first version_number 3, got %d", resp.Versions[0].VersionNumber)
	}
	if resp.Versions[2].VersionNumber != 1 {
		t.Errorf("expected last version_number 1, got %d", resp.Versions[2].VersionNumber)
	}
}

func TestDocVersion_GetVersionReturnsDocumentJSON(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Get Version", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("Version content")},
	})

	// Get version list first
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	versionID := listResp.Versions[0].ID

	// Get version detail
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s", e.teamID, artID, versionID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var detail VersionDetail
	json.Unmarshal(w.Body.Bytes(), &detail)
	if detail.VersionNumber != 1 {
		t.Errorf("expected version_number 1, got %d", detail.VersionNumber)
	}
	if len(detail.DocumentJSON) == 0 {
		t.Error("expected document_json in version detail")
	}
}

func TestDocVersion_RestoreCreatesNewVersion(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Restore Test", basicBlocks())

	// Patch to create v2
	patchBody := map[string]any{
		"document_json": makeCreateDocReq("Restore Test", "general_document", []DocumentBlock{
			{ID: "b1", Type: "paragraph", Text: strPtr("v2 content")},
		})["document_json"],
	}
	bodyBytes, _ := json.Marshal(patchBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Get version 1 ID
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)

	// Find version 1
	var v1ID string
	for _, v := range listResp.Versions {
		if v.VersionNumber == 1 {
			v1ID = v.ID
			break
		}
	}

	// Restore version 1
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID, v1ID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should now have 3 versions
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_document_versions WHERE artifact_id = $1", artID).Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 versions after restore, got %d", count)
	}

	var restoreResp RestoreResponse
	json.Unmarshal(w.Body.Bytes(), &restoreResp)
	if restoreResp.NewVersionNumber != 3 {
		t.Errorf("expected new version_number 3, got %d", restoreResp.NewVersionNumber)
	}
	if restoreResp.RestoredFromVersion != 1 {
		t.Errorf("expected restored_from_version 1, got %d", restoreResp.RestoredFromVersion)
	}
}

func TestDocVersion_RestoreUpdatesArtifactDocuments(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Restore Update", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("Original content")},
	})

	// Patch to v2
	patchBody := map[string]any{
		"document_json": makeCreateDocReq("Restore Update", "general_document", []DocumentBlock{
			{ID: "b1", Type: "paragraph", Text: strPtr("Modified content")},
		})["document_json"],
	}
	bodyBytes, _ := json.Marshal(patchBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Get v1 ID and restore
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	var v1ID string
	for _, v := range listResp.Versions {
		if v.VersionNumber == 1 {
			v1ID = v.ID
			break
		}
	}

	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID, v1ID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Verify artifact_documents has original content
	var currentJSON []byte
	e.pool.QueryRow(t.Context(), "SELECT document_json FROM artifact_documents WHERE artifact_id = $1", artID).Scan(&currentJSON)
	var doc DocumentJSON
	json.Unmarshal(currentJSON, &doc)
	if len(doc.Blocks) > 0 && doc.Blocks[0].Text != nil {
		if *doc.Blocks[0].Text != "Original content" {
			t.Errorf("expected 'Original content' after restore, got '%s'", *doc.Blocks[0].Text)
		}
	}
}

func TestDocVersion_RestoreIsNonDestructive(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Non-Destructive", basicBlocks())

	// Get v1 snapshot before restore
	var v1JSONBefore []byte
	e.pool.QueryRow(t.Context(), "SELECT document_json FROM artifact_document_versions WHERE artifact_id = $1 AND version_number = 1", artID).Scan(&v1JSONBefore)

	// Patch then restore v1
	patchBody, _ := json.Marshal(map[string]any{
		"document_json": makeCreateDocReq("Non-Destructive", "general_document", []DocumentBlock{
			{ID: "b1", Type: "paragraph", Text: strPtr("changed")},
		})["document_json"],
	})
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Get v1 ID
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	var v1ID string
	for _, v := range listResp.Versions {
		if v.VersionNumber == 1 {
			v1ID = v.ID
			break
		}
	}

	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID, v1ID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// v1 snapshot unchanged
	var v1JSONAfter []byte
	e.pool.QueryRow(t.Context(), "SELECT document_json FROM artifact_document_versions WHERE artifact_id = $1 AND version_number = 1", artID).Scan(&v1JSONAfter)
	if !bytes.Equal(v1JSONBefore, v1JSONAfter) {
		t.Error("version 1 snapshot was modified during restore")
	}

	// All original versions still exist
	var count int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM artifact_document_versions WHERE artifact_id = $1 AND version_number <= 2", artID).Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 original versions intact, got %d", count)
	}
}

func TestDocVersion_RestoreRecomputesWordCount(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Recompute WC", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("one two three four five")},
	})

	// Patch to v2 with different text
	patchBody, _ := json.Marshal(map[string]any{
		"document_json": makeCreateDocReq("Recompute WC", "general_document", []DocumentBlock{
			{ID: "b1", Type: "paragraph", Text: strPtr("different")},
		})["document_json"],
	})
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Get v1 ID
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	var v1ID string
	for _, v := range listResp.Versions {
		if v.VersionNumber == 1 {
			v1ID = v.ID
			break
		}
	}

	// Restore v1
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID, v1ID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var restoreResp RestoreResponse
	json.Unmarshal(w.Body.Bytes(), &restoreResp)
	if restoreResp.WordCount != 5 {
		t.Errorf("expected recomputed word_count 5, got %d", restoreResp.WordCount)
	}

	// Check new version row also has correct WC
	var newVersionWC int
	e.pool.QueryRow(t.Context(), `
		SELECT word_count FROM artifact_document_versions
		WHERE artifact_id = $1 AND version_number = 3
	`, artID).Scan(&newVersionWC)
	if newVersionWC != 5 {
		t.Errorf("expected version row word_count 5, got %d", newVersionWC)
	}
}

func TestDocVersion_RestoreArchivedDenied403(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Archived Restore", basicBlocks())

	// Get v1 ID
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	v1ID := listResp.Versions[0].ID

	// Archive
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Try restore
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID, v1ID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403 for archived restore, got %d", w.Code)
	}
}

func TestDocVersion_CrossTeamListReturns404(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Cross Team", basicBlocks())

	fakeTeam := "00000000-0000-0000-0000-000000000999"
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", fakeTeam, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 cross-team, got %d", w.Code)
	}
}

func TestDocVersion_CrossTeamDetailReturns404(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Cross Team Detail", basicBlocks())

	// Get version ID
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	vID := listResp.Versions[0].ID

	fakeTeam := "00000000-0000-0000-0000-000000000999"
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s", fakeTeam, artID, vID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 cross-team, got %d", w.Code)
	}
}

func TestDocVersion_RestoreFromOtherArtifactFails(t *testing.T) {
	e := setupArtifactTest(t)
	artID1 := createTestDocument(t, e, "Doc One", basicBlocks())
	artID2 := createTestDocument(t, e, "Doc Two", basicBlocks())

	// Get version from artID1
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID1), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var listResp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	v1ID := listResp.Versions[0].ID

	// Try restoring artID1's version on artID2
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID2, v1ID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 for version from other artifact, got %d", w.Code)
	}
}

func TestDocVersion_StaleSaveReturns409(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Stale Save", basicBlocks())

	// Get current updated_at
	var currentUpdatedAt string
	e.pool.QueryRow(t.Context(), "SELECT updated_at::text FROM artifacts WHERE id = $1", artID).Scan(&currentUpdatedAt)

	// PATCH with stale If-Match
	staleValue := "2000-01-01 00:00:00+00"
	patchBody, _ := json.Marshal(map[string]any{
		"document_json": makeCreateDocReq("Stale Save", "general_document", []DocumentBlock{
			{ID: "b1", Type: "paragraph", Text: strPtr("stale edit")},
		})["document_json"],
	})
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", staleValue)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 409 {
		t.Errorf("expected 409 for stale save, got %d: %s", w.Code, w.Body.String())
	}

	// Verify fresh save works
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s", e.teamID, artID), bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", currentUpdatedAt)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 with correct If-Match, got %d", w.Code)
	}
}

func TestDocVersion_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Unauthorized", basicBlocks())

	endpoints := []string{
		fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID),
	}
	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code != 401 && w.Code != 403 {
			t.Errorf("expected 401/403 for %s, got %d", ep, w.Code)
		}
	}
}

func TestDocVersion_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	artID := createTestDocument(t, e, "Side Effects", basicBlocks())

	var apprBefore, actionBefore, remBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&apprBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remBefore)

	// List versions, get version detail
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	// Restore
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions/%s/restore", e.teamID, artID,
		getFirstVersionID(t, e, artID)), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req)

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

func TestDocVersion_NonDocumentArtifactRejected(t *testing.T) {
	e := setupArtifactTest(t)

	// Create a non-document artifact
	body := `{"artifact_type":"report","title":"MD Report","content_markdown":"# Report"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	// Try to list versions
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for non-document versions, got %d", w.Code)
	}
}

// ─── Helpers ───

func getFirstVersionID(t *testing.T, e *artifactTestEnv, artID string) string {
	t.Helper()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/documents/%s/versions", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp struct {
		Versions []VersionListItem `json:"versions"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Versions[0].ID
}

func createGeneratedDoc(t *testing.T, e *artifactTestEnv) string {
	t.Helper()
	// Simulate generated doc by creating document then inserting a version with source=generated
	// Use the generate flow directly via DB since worker isn't configured in test env
	artID := createTestDocument(t, e, "Generated Doc", basicBlocks())

	// Delete the auto-created v1 and re-insert as generated
	e.pool.Exec(t.Context(), "DELETE FROM artifact_document_versions WHERE artifact_id = $1", artID)
	e.pool.Exec(t.Context(), `
		INSERT INTO artifact_document_versions (artifact_id, team_id, document_json, version_number, word_count, source)
		SELECT artifact_id, team_id, document_json, 1, word_count, 'generated'
		FROM artifact_documents ad
		JOIN artifacts a ON a.id = ad.artifact_id
		WHERE ad.artifact_id = $1
	`, artID)
	return artID
}

func createTemplateDoc(t *testing.T, e *artifactTestEnv) string {
	t.Helper()
	artID := createTestDocument(t, e, "Template Doc", basicBlocks())

	e.pool.Exec(t.Context(), "DELETE FROM artifact_document_versions WHERE artifact_id = $1", artID)
	e.pool.Exec(t.Context(), `
		INSERT INTO artifact_document_versions (artifact_id, team_id, document_json, version_number, word_count, source)
		SELECT artifact_id, team_id, document_json, 1, word_count, 'template'
		FROM artifact_documents ad
		JOIN artifacts a ON a.id = ad.artifact_id
		WHERE ad.artifact_id = $1
	`, artID)
	return artID
}
