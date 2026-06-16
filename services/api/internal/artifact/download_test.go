package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// mockS3ForDownload implements storage.S3Client for download tests.
type mockS3ForDownload struct {
	presignedURL string
	presignErr   error
	putErr       error
	called       bool
}

func (m *mockS3ForDownload) PutObject(_ context.Context, _, _ string, _ []byte, _ string) error {
	return m.putErr
}

func (m *mockS3ForDownload) GetPresignedURL(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	m.called = true
	if m.presignErr != nil {
		return "", m.presignErr
	}
	if m.presignedURL != "" {
		return m.presignedURL, nil
	}
	return "https://minio.local/test-bucket/file.pptx?X-Amz-Signature=abc", nil
}

func setupDownloadTest(t *testing.T) *artifactTestEnv {
	e := setupArtifactTest(t)
	// Re-register download/export routes on the existing router
	// The setupArtifactTest already created the router with the artifact routes.
	// We need to rebuild it with S3 support.
	pool := e.pool
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}

	artH := NewHandler(pool)
	artH.SetStorage(&mockS3ForDownload{}, "test-bucket")
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifacts", artH.Create)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts", artH.List)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}", artH.Get)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/download", artH.Download)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/export/markdown", artH.ExportMarkdown)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/export/pdf", artH.ExportPDF)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/export/docx", artH.ExportDOCX)
		// v1.4 Track 7: Version History
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/documents/{artifactId}/versions", artH.ListVersions)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/documents/{artifactId}/versions/{versionId}", artH.GetVersion)
		r.With(middleware.RequirePermission(pool, "artifacts.update")).Post("/artifacts/documents/{artifactId}/versions/{versionId}/restore", artH.RestoreVersion)
		r.With(middleware.RequirePermission(pool, "artifacts.delete")).Delete("/artifacts/{artifactId}", artH.Delete)
		// v1.4 Track 1: Document routes (for native document export tests)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifacts/documents", artH.CreateDocument)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/documents/{artifactId}", artH.GetDocument)
	})

	e.r = r
	return e
}

func TestDownload_FileBackedReturnsPresignedURL(t *testing.T) {
	e := setupDownloadTest(t)

	body := `{"artifact_type":"presentation","title":"My Deck","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	bucket := "test-bucket-" + uuid.New().String()[:8]
	key := fmt.Sprintf("teams/%s/artifacts/%s.pptx", e.teamID, uuid.New().String())
	var storageID string
	e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 5000, 'abc', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1, file_format = 'pptx' WHERE id = $2", storageID, artID)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/download", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var dl map[string]any
	json.Unmarshal(w2.Body.Bytes(), &dl)
	if dl["download_url"] == nil || dl["download_url"] == "" {
		t.Error("expected download_url in response")
	}
}

func TestDownload_ExpiryMax900Seconds(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"presentation","title":"Expiry Test","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	bucket := "b-" + uuid.New().String()[:8]
	key := "k-" + uuid.New().String()
	var storageID string
	e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 1000, 'abc', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1 WHERE id = $2", storageID, artID)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/download", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	var dl map[string]any
	json.Unmarshal(w2.Body.Bytes(), &dl)
	if dl["expires_in_seconds"].(float64) > 900 {
		t.Errorf("expiry > 900: %v", dl["expires_in_seconds"])
	}
	if dl["expires_in_seconds"].(float64) != 900 {
		t.Errorf("expected exactly 900, got %v", dl["expires_in_seconds"])
	}
}

func TestDownload_OmitsBucketAndObjectKey(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"presentation","title":"Leak Test","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	leakyBucket := "secret-bucket-" + uuid.New().String()[:8]
	leakyKey := "secret/key/path-" + uuid.New().String() + ".pptx"
	var storageID string
	e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 1000, 'abc', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, leakyBucket, leakyKey, e.actorUUID).Scan(&storageID)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1 WHERE id = $2", storageID, artID)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/download", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	bodyStr := w2.Body.String()
	if strings.Contains(bodyStr, leakyBucket) {
		t.Error("bucket name leaked in download response")
	}
	if strings.Contains(bodyStr, leakyKey) {
		t.Error("object key leaked in download response")
	}
	if strings.Contains(bodyStr, "object_key") {
		t.Error("object_key field in download response")
	}
}

func TestDownload_InlineArtifactRejected(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"document","title":"Inline Only","content_markdown":"some content"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/download", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Errorf("expected 400 for inline download, got %d", w2.Code)
	}
}

func TestDownload_CrossTeam404(t *testing.T) {
	e := setupDownloadTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/00000000-0000-0000-0000-000000000000/artifacts/00000000-0000-0000-0000-000000000001/download"), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDownload_ArchivedDenied(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"presentation","title":"Archived DL","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	bucket := "b-" + uuid.New().String()[:8]
	key := "k-" + uuid.New().String()
	var storageID string
	e.pool.QueryRow(t.Context(), `
		INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, created_by)
		VALUES ($1, $2, $3, 'application/octet-stream', 1000, 'abc', 'provider_managed', $4)
		RETURNING id::text
	`, e.teamUUID, bucket, key, e.actorUUID).Scan(&storageID)
	e.pool.Exec(t.Context(), "UPDATE artifacts SET storage_object_id = $1 WHERE id = $2", storageID, artID)

	// Archive
	delReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, artID), nil)
	delReq.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), delReq)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/download", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 403 {
		t.Errorf("expected 403 for archived download, got %d", w2.Code)
	}
}

func TestExport_MarkdownReturnsContent(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"document","title":"MD Export","content_markdown":"# Hello World\n\nThis is content."}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/markdown", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "Hello World") {
		t.Error("markdown content not in export")
	}
	if !strings.Contains(w2.Header().Get("Content-Type"), "text/markdown") {
		t.Error("expected text/markdown content type")
	}
}

func TestExport_MarkdownSafeFilename(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"document","title":"Report 2026-Q2 Sales","content_markdown":"content"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/markdown", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	cd := w2.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".md") {
		t.Errorf("expected .md in filename: %s", cd)
	}
}

func TestExport_MarkdownDeniedWhenEmpty(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"presentation","title":"Empty MD","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/markdown", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Errorf("expected 400 for empty content export, got %d", w2.Code)
	}
}

func TestExport_PDFReturnsApplicationPDF(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"document","title":"PDF Export","content_markdown":"# Hello\n\nWorld"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/pdf", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Header().Get("Content-Type"), "application/pdf") {
		t.Errorf("expected application/pdf, got %s", w2.Header().Get("Content-Type"))
	}
	bodyBytes := w2.Body.Bytes()
	if len(bodyBytes) < 4 || string(bodyBytes[:4]) != "%PDF" {
		t.Error("PDF does not start with %PDF")
	}
}

func TestExport_PDFSafeFilename(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"document","title":"Report Quarterly","content_markdown":"content"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/pdf", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	cd := w2.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".pdf") {
		t.Errorf("expected .pdf in filename: %s", cd)
	}
}

func TestExport_PDFDeniedWhenEmpty(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"presentation","title":"Empty PDF","content_markdown":""}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/pdf", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Errorf("expected 400 for empty content PDF export, got %d", w2.Code)
	}
}

func TestDownload_UnauthorizedDenied(t *testing.T) {
	e := setupDownloadTest(t)
	body := `{"artifact_type":"document","title":"Test","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/download", e.teamID, artID), nil)
	w2 := httptest.NewRecorder()
	e.r.ServeHTTP(w2, req2)
	if w2.Code != 401 && w2.Code != 403 {
		t.Errorf("expected 401/403, got %d", w2.Code)
	}
}

func TestDownload_NoOperationalSideEffects(t *testing.T) {
	e := setupDownloadTest(t)
	var beforeApprovals, beforeActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&beforeApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&beforeActions)

	body := `{"artifact_type":"document","title":"No Side Effects","content_markdown":"# Test"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	// Export markdown
	req2 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/markdown", e.teamID, artID), nil)
	req2.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req2)

	// Export PDF
	req3 := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/pdf", e.teamID, artID), nil)
	req3.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req3)

	var afterApprovals, afterActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&afterApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&afterActions)
	if afterApprovals != beforeApprovals {
		t.Errorf("approval_requests changed: %d → %d", beforeApprovals, afterApprovals)
	}
	if afterActions != beforeActions {
		t.Errorf("asset_actions changed: %d → %d", beforeActions, afterActions)
	}
}
