package artifact

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// uniqueStorageKey generates a unique bucket+key pair to avoid UNIQUE constraint violations
// from leftover data in previous test runs.
func uniqueStorageKey(t *testing.T) (bucket, key string) {
	t.Helper()
	u := uuid.New().String()
	return "test-bucket-" + u[:8], fmt.Sprintf("teams/test/artifacts/%s.pptx", u)
}

func TestStorage_IncludeFilesReturnsFileMetadata(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"Inline Doc","content_markdown":"Hello"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	bucket, key := uniqueStorageKey(t)
	var storageID string
	err := e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 123456, 'abc123', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)
	if err != nil {
		t.Fatalf("INSERT storage_objects: %v", err)
	}

	fileBody := `{"artifact_type":"presentation","title":"PPTX Artifact","content_markdown":""}`
	fileReq := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(fileBody))
	fileReq.Header.Set("Authorization", "Bearer "+e.token)
	fileReq.Header.Set("Content-Type", "application/json")
	fileW := httptest.NewRecorder()
	e.r.ServeHTTP(fileW, fileReq)
	var fileArt map[string]any
	json.Unmarshal(fileW.Body.Bytes(), &fileArt)
	artID := fileArt["id"].(string)

	_, err = e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1, file_format = 'pptx' WHERE id = $2", storageID, artID)
	if err != nil {
		t.Fatalf("UPDATE artifacts: %v", err)
	}

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts?include_files=true", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)

	if strings.Contains(w2.Body.String(), bucket) {
		t.Error("bucket name leaked")
	}
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var list []map[string]any
	json.Unmarshal(w2.Body.Bytes(), &list)
	var foundFileMeta bool
	for _, a := range list {
		if a["id"] == artID {
			meta, exists := a["file_metadata"]
			if !exists {
				t.Error("expected file_metadata on file artifact")
				continue
			}
			metaMap, ok := meta.(map[string]any)
			if !ok {
				t.Error("file_metadata is not an object")
				continue
			}
			if metaMap["file_size"] == nil {
				t.Error("expected file_size in metadata")
			}
			if metaMap["file_format"] == nil {
				t.Error("expected file_format in metadata")
			}
			if metaMap["download_available"] != false {
				t.Error("expected download_available=false")
			}
			foundFileMeta = true
		}
	}
	if !foundFileMeta {
		t.Error("file artifact not found in list or no file_metadata")
	}
}

func TestStorage_IncludeFilesOmitsBucketAndKeys(t *testing.T) {
	e := setupArtifactTest(t)
	bucket, key := uniqueStorageKey(t)
	var storageID string
	e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 9999, 'abc123', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)

	body := `{"artifact_type":"presentation","title":"Linked Artifact","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1, file_format = 'pptx' WHERE id = $2", storageID, artID)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts?include_files=true", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)

	bodyStr := w2.Body.String()
	if strings.Contains(bodyStr, bucket) {
		t.Error("bucket name leaked in include_files response")
	}
	if strings.Contains(bodyStr, key) {
		t.Error("object key leaked in include_files response")
	}
	if strings.Contains(bodyStr, "object_key") {
		t.Error("object_key field present in include_files response")
	}
}

func TestStorage_RecentReturnsUpdatedAtDesc(t *testing.T) {
	e := setupArtifactTest(t)
	body1 := `{"artifact_type":"document","title":"First","content_markdown":"1"}`
	req1 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body1))
	req1.Header.Set("Authorization", "Bearer "+e.token)
	req1.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req1)

	body2 := `{"artifact_type":"document","title":"Second","content_markdown":"2"}`
	req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+e.token)
	req2.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req2)

	e.pool.Exec(t.Context(), "UPDATE artifacts SET updated_at = NOW() + interval '1 hour' WHERE title = 'Second'")

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/recent", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) < 2 {
		t.Fatalf("expected at least 2, got %d", len(list))
	}
	if list[0]["title"] != "Second" {
		t.Errorf("expected Second first (most recent), got %v", list[0]["title"])
	}
}

func TestStorage_RecentLimitIs20(t *testing.T) {
	e := setupArtifactTest(t)
	for i := 0; i < 25; i++ {
		body := fmt.Sprintf(`{"artifact_type":"document","title":"Art-%d","content_markdown":"x"}`, i)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+e.token)
		req.Header.Set("Content-Type", "application/json")
		e.r.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/recent", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) > 20 {
		t.Errorf("expected max 20, got %d", len(list))
	}
}

func TestStorage_RecentExcludesArchivedByDefault(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"ToArchiveRecent","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	delReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, artID), nil)
	delReq.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), delReq)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/recent", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var list []map[string]any
	json.Unmarshal(w2.Body.Bytes(), &list)
	for _, a := range list {
		if a["title"] == "ToArchiveRecent" {
			t.Error("archived artifact appeared in recent without include_archived")
		}
	}
}

func TestStorage_RecentIncludeArchived(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"ArchivedOneRecent","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	delReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, artID), nil)
	delReq.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), delReq)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/recent?include_archived=true", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var list []map[string]any
	json.Unmarshal(w2.Body.Bytes(), &list)
	found := false
	for _, a := range list {
		if a["title"] == "ArchivedOneRecent" {
			found = true
		}
	}
	if !found {
		t.Error("archived artifact not found with include_archived=true")
	}
}

func TestStorage_SearchMatchesTitle(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"UniqueSearchTitleX9","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/search?q=UniqueSearchTitleX9", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req2)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Error("search by title returned empty")
	}
}

func TestStorage_SearchMatchesDescription(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"DocDescSearch","description":"SpecialDescTextX7","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/search?q=SpecialDescTextX7", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req2)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Error("search by description returned empty")
	}
}

func TestStorage_SearchMatchesContent(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"DocContentSearch","content_markdown":"RareContentKeywordX5"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/search?q=RareContentKeywordX5", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req2)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Error("search by content returned empty")
	}
}

func TestStorage_SearchTeamScoped(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"TeamScopedSearchX3","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest("GET", "/api/teams/00000000-0000-0000-0000-000000000000/artifacts/search?q=TeamScopedSearchX3", nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req2)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Error("cross-team search returned results")
	}
}

func TestStorage_SearchQueryLengthEnforced(t *testing.T) {
	e := setupArtifactTest(t)
	longQuery := strings.Repeat("a", 201)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/search?q=%s", e.teamID, longQuery), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for long query, got %d", w.Code)
	}
}

func TestStorage_SearchEmptyQuery(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/search", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 for empty query, got %d", w.Code)
	}
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Error("expected empty list for empty query")
	}
}

func TestStorage_SummaryCountsCorrectly(t *testing.T) {
	e := setupArtifactTest(t)
	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"artifact_type":"document","title":"SummaryInline-%d","content_markdown":"x"}`, i)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+e.token)
		req.Header.Set("Content-Type", "application/json")
		e.r.ServeHTTP(httptest.NewRecorder(), req)
	}

	bucket, key := uniqueStorageKey(t)
	var storageID string
	err := e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 5000, 'abc', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)
	if err != nil {
		t.Fatalf("INSERT storage: %v", err)
	}

	fileBody := `{"artifact_type":"presentation","title":"SummaryFileArt","content_markdown":""}`
	fileReq := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(fileBody))
	fileReq.Header.Set("Authorization", "Bearer "+e.token)
	fileReq.Header.Set("Content-Type", "application/json")
	fileW := httptest.NewRecorder()
	e.r.ServeHTTP(fileW, fileReq)
	var fileArt map[string]any
	json.Unmarshal(fileW.Body.Bytes(), &fileArt)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1, file_format = 'pptx' WHERE id = $2", storageID, fileArt["id"])

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/storage-summary", e.teamID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var summary map[string]any
	json.Unmarshal(w.Body.Bytes(), &summary)
	if summary["file_artifacts"].(float64) < 1 {
		t.Error("expected at least 1 file artifact")
	}
	if summary["total_file_size_bytes"].(float64) < 5000 {
		t.Error("expected total file size >= 5000")
	}
	byFormat, ok := summary["by_format"].(map[string]any)
	if !ok || byFormat["pptx"] == nil {
		t.Error("expected pptx in by_format")
	}
}

func TestStorage_SummaryExcludesArchived(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"document","title":"SummaryArchived","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	delReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, resp["id"]), nil)
	delReq.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), delReq)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/storage-summary", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var summary map[string]any
	json.Unmarshal(w2.Body.Bytes(), &summary)
	totalArtifacts := int(summary["total_artifacts"].(float64))

	body2 := `{"artifact_type":"document","title":"SummaryActive","content_markdown":"x"}`
	req3 := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body2))
	req3.Header.Set("Authorization", "Bearer "+e.token)
	req3.Header.Set("Content-Type", "application/json")
	e.r.ServeHTTP(httptest.NewRecorder(), req3)

	req4 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/storage-summary", e.teamID), nil)
	req4.Header.Set("Authorization", "Bearer "+e.token)
	w4 := httptest.NewRecorder()
	e.r.ServeHTTP(w4, req4)
	var summary2 map[string]any
	json.Unmarshal(w4.Body.Bytes(), &summary2)
	if int(summary2["total_artifacts"].(float64)) != totalArtifacts+1 {
		t.Error("summary should increase by 1 after creating non-archived artifact")
	}
}

func TestStorage_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/recent", e.teamID), nil)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 && w.Code != 403 {
		t.Errorf("expected 401/403, got %d", w.Code)
	}
}

func TestStorage_NoPresignedURL(t *testing.T) {
	e := setupArtifactTest(t)
	bucket, key := uniqueStorageKey(t)
	var storageID string
	e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 1000, 'abc', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)

	fileBody := `{"artifact_type":"presentation","title":"NoPresignArt","content_markdown":""}`
	fileReq := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(fileBody))
	fileReq.Header.Set("Authorization", "Bearer "+e.token)
	fileReq.Header.Set("Content-Type", "application/json")
	fileW := httptest.NewRecorder()
	e.r.ServeHTTP(fileW, fileReq)
	var fileArt map[string]any
	json.Unmarshal(fileW.Body.Bytes(), &fileArt)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1, file_format = 'pdf' WHERE id = $2", storageID, fileArt["id"])

	endpoints := []string{
		fmt.Sprintf("/api/teams/%s/artifacts?include_files=true", e.teamID),
		fmt.Sprintf("/api/teams/%s/artifacts/recent", e.teamID),
		fmt.Sprintf("/api/teams/%s/artifacts/storage-summary", e.teamID),
	}
	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		req.Header.Set("Authorization", "Bearer "+e.token)
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		bodyStr := w.Body.String()
		if strings.Contains(bodyStr, "presigned") || strings.Contains(bodyStr, "download_url") || strings.Contains(bodyStr, "X-Amz") {
			t.Errorf("presigned URL or download link found in %s response", ep)
		}
	}
}

func TestStorage_MinioKeyPatternSafe(t *testing.T) {
	e := setupArtifactTest(t)
	uid := uuid.New().String()
	expectedKey := fmt.Sprintf("teams/%s/artifacts/%s.pdf", e.teamID, uid)
	var objKey string
	err := e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, 'b-' || $2, $3, 'application/octet-stream', 1000, 'abc', 'provider_managed', $4)
		RETURNING object_key
	`, e.teamUUID, uid, expectedKey, e.actorUUID).Scan(&objKey)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	if !strings.HasPrefix(objKey, "teams/") || !strings.Contains(objKey, "/artifacts/") {
		t.Errorf("MinIO key does not follow safe pattern: %s", objKey)
	}

	body := `{"artifact_type":"presentation","title":"PatternTestArt","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	var storageID string
	e.pool.QueryRow(t.Context(), "SELECT id::text FROM storage_objects WHERE object_key = $1", objKey).Scan(&storageID)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1 WHERE id = $2", storageID, resp["id"])

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts?include_files=true", e.teamID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if strings.Contains(w2.Body.String(), objKey) {
		t.Error("MinIO object key exposed in include_files response")
	}
}
