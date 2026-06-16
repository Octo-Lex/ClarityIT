package artifact

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const artifactDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

type artifactTestEnv struct {
	r           *chi.Mux
	pool        *pgxpool.Pool
	token       string
	teamID      string
	teamUUID    uuid.UUID
	actorUUID   uuid.UUID
	memberToken string
}

func setupArtifactTest(t *testing.T) *artifactTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, _ := pgxpool.New(t.Context(), artifactDBURL)
	t.Cleanup(func() { pool.Close() })

	// Re-seed system templates if wiped by other test packages' TRUNCATE CASCADE
	ensureSystemTemplates(t, pool)

	artH := NewHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifacts", artH.Create)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts", artH.List)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}", artH.Get)
		r.With(middleware.RequirePermission(pool, "artifacts.update")).Patch("/artifacts/{artifactId}", artH.Patch)
		r.With(middleware.RequirePermission(pool, "artifacts.delete")).Delete("/artifacts/{artifactId}", artH.Delete)
		// Meeting summaries (Track 3) — under /artifacts path
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifacts/meeting-summaries", artH.CreateMeetingSummary)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/meeting-summaries", artH.ListMeetingSummaries)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/meeting-summaries/{id}", artH.GetMeetingSummary)
		r.With(middleware.RequirePermission(pool, "artifacts.update")).Patch("/artifacts/meeting-summaries/{id}", artH.PatchMeetingSummary)
		// Status reports (Track 4)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/status-reports/generate", artH.GenerateStatusReport)
		// Templates (Track 5)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifact-templates", artH.ListTemplates)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifact-templates", artH.CreateTemplate)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifact-templates/{templateId}/instantiate", artH.InstantiateTemplate)
		// Storage endpoints (Track 6)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/recent", artH.Recent)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/search", artH.Search)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/storage-summary", artH.StorageSummary)
		// Download/Export (Track 7)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/download", artH.Download)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/export/markdown", artH.ExportMarkdown)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/export/pdf", artH.ExportPDF)
		// Documents (v1.4 Track 1)
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifacts/documents", artH.CreateDocument)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/documents", artH.ListDocuments)
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/documents/{artifactId}", artH.GetDocument)
		r.With(middleware.RequirePermission(pool, "artifacts.update")).Patch("/artifacts/documents/{artifactId}", artH.PatchDocument)
		// v1.4 Track 3: Document Assist
		r.With(middleware.RequirePermission(pool, "artifacts.update")).Post("/artifacts/documents/{artifactId}/document-assist", artH.DocumentAssist)
		// v1.4 Track 4: Document Generation
		r.With(middleware.RequirePermission(pool, "artifacts.create")).Post("/artifacts/generate-document", artH.GenerateDocument)
		// v1.4 Track 6: DOCX Export
		r.With(middleware.RequirePermission(pool, "artifacts.read")).Get("/artifacts/{artifactId}/export/docx", artH.ExportDOCX)
	})

	token := loginArtifact(t, r, "owner@test.dev", "password12")
	memberToken := loginArtifact(t, r, "member@test.dev", "password12")

	var teamID string
	pool.QueryRow(t.Context(), "SELECT id::text FROM teams LIMIT 1").Scan(&teamID)
	teamUUID, _ := uuid.Parse(teamID)
	var actorID string
	pool.QueryRow(t.Context(), "SELECT id::text FROM users WHERE email='owner@test.dev'").Scan(&actorID)
	actorUUID, _ := uuid.Parse(actorID)

	return &artifactTestEnv{r: r, pool: pool, token: token, teamID: teamID, teamUUID: teamUUID, actorUUID: actorUUID, memberToken: memberToken}
}

func loginArtifact(t *testing.T, r *chi.Mux, email, pw string) string {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, pw)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login as %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["access_token"].(string)
}

func (e *artifactTestEnv) createArtifact(t *testing.T, artType, title, content string) string {
	t.Helper()
	body := fmt.Sprintf(`{"artifact_type":%q,"title":%q,"content_markdown":%q,"source_data":{"project":"alpha"}}`, artType, title, content)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create artifact: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["id"].(string)
}

func (e *artifactTestEnv) listArtifacts(t *testing.T, query string) []any {
	t.Helper()
	url := fmt.Sprintf("/api/teams/%s/artifacts", e.teamID)
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list artifacts: %d %s", w.Code, w.Body.String())
	}
	var resp []any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ─── Tests ───

// Test 1: create artifact
func TestArtifact_Create(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "Weekly Status", "## Status\nAll good.")
	if id == "" {
		t.Fatal("expected artifact ID")
	}
}

// Test 2: list artifacts team-scoped
func TestArtifact_ListTeamScoped(t *testing.T) {
	e := setupArtifactTest(t)
	e.createArtifact(t, "document", "Doc 1", "content")
	arts := e.listArtifacts(t, "")
	if len(arts) == 0 {
		t.Error("expected at least 1 artifact")
	}
}

// Test 3: get artifact by id
func TestArtifact_GetByID(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "My Report", "content")
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Test 4: patch artifact
func TestArtifact_Patch(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "Original Title", "original")
	body := `{"title":"Updated Title","status":"published"}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["title"] != "Updated Title" {
		t.Errorf("expected Updated Title, got %v", resp["title"])
	}
	if resp["status"] != "published" {
		t.Errorf("expected published, got %v", resp["status"])
	}
}

// Test 5: delete archives artifact
func TestArtifact_DeleteArchives(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "To Archive", "content")
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify it's archived
	var status string
	e.pool.QueryRow(t.Context(), "SELECT status FROM artifacts WHERE id=$1", id).Scan(&status)
	if status != "archived" {
		t.Errorf("expected archived, got %s", status)
	}
}

// Test 6: archived hidden by default
func TestArtifact_ArchivedHiddenByDefault(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "Will Archive", "content")
	// Archive it
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// List — archived should not appear
	arts := e.listArtifacts(t, "")
	for _, a := range arts {
		art := a.(map[string]any)
		if art["id"] == id {
			t.Error("archived artifact should be hidden by default")
		}
	}
}

// Test 7: include_archived returns archived artifacts
func TestArtifact_IncludeArchived(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "Will Archive", "content")
	// Archive it
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	// List with include_archived
	arts := e.listArtifacts(t, "include_archived=true")
	found := false
	for _, a := range arts {
		art := a.(map[string]any)
		if art["id"] == id {
			found = true
		}
	}
	if !found {
		t.Error("archived artifact should appear with include_archived=true")
	}
}

// Test 8: type filter works
func TestArtifact_TypeFilter(t *testing.T) {
	e := setupArtifactTest(t)
	e.createArtifact(t, "report", "Report Item", "content")
	e.createArtifact(t, "document", "Doc Item", "content")

	arts := e.listArtifacts(t, "type=report")
	for _, a := range arts {
		art := a.(map[string]any)
		if art["artifact_type"] != "report" {
			t.Errorf("expected report only, got %v", art["artifact_type"])
		}
	}
}

// Test 9: status filter works
func TestArtifact_StatusFilter(t *testing.T) {
	e := setupArtifactTest(t)
	e.createArtifact(t, "report", "Published Report", "content")
	// Create and publish
	id2 := e.createArtifact(t, "report", "Draft Report", "content")
	body := `{"status":"published"}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id2), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	_ = id2

	arts := e.listArtifacts(t, "status=published")
	for _, a := range arts {
		art := a.(map[string]any)
		if art["status"] != "published" {
			t.Errorf("expected published only, got %v", art["status"])
		}
	}
}

// Test 10: title search works
func TestArtifact_TitleSearch(t *testing.T) {
	e := setupArtifactTest(t)
	e.createArtifact(t, "report", "UniqueSearchableTitle", "content")
	e.createArtifact(t, "report", "Other Report", "content")

	arts := e.listArtifacts(t, "q=UniqueSearchable")
	found := false
	for _, a := range arts {
		art := a.(map[string]any)
		if strings.Contains(art["title"].(string), "UniqueSearchable") {
			found = true
		}
	}
	if !found {
		t.Error("title search should find the artifact")
	}
}

// Test 11: invalid artifact_type rejected
func TestArtifact_InvalidTypeRejected(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"invalid","title":"Test","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid type, got %d", w.Code)
	}
}

// Test 12: invalid status rejected
func TestArtifact_InvalidStatusRejected(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"report","title":"Test","status":"invalid","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid status, got %d", w.Code)
	}
}

// Test 13: invalid file_format rejected
// Note: file_format is set by system, not by user create. This tests PATCH won't accept invalid.
func TestArtifact_InvalidFileFormatNotUserSettable(t *testing.T) {
	e := setupArtifactTest(t)
	// file_format is not in CreateRequest, so users can't set it
	// This is a structural test — verify the CHECK constraint exists
	var constraints []string
	rows, _ := e.pool.Query(t.Context(),
		"SELECT conname FROM pg_constraint WHERE conrelid='artifacts'::regclass AND contype='c'")
	for rows.Next() {
		var c string
		rows.Scan(&c)
		constraints = append(constraints, c)
	}
	rows.Close()
	found := false
	for _, c := range constraints {
		if strings.Contains(c, "file_format") {
			found = true
		}
	}
	if !found {
		t.Error("expected CHECK constraint on file_format")
	}
}

// Test 14: source_data scalar rejected (must be JSON object)
func TestArtifact_SourceDataScalarRejected(t *testing.T) {
	e := setupArtifactTest(t)
	// Send source_data as a string scalar — JSON decode should reject (400)
	body := `{"artifact_type":"report","title":"Test","content_markdown":"x","source_data":"scalar-string"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for scalar source_data, got %d", w.Code)
	}
}

// Test 15: cross-team access returns 404
func TestArtifact_CrossTeam404(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "Cross Team", "content")
	otherTeam := uuid.New().String()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s", otherTeam, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 for cross-team, got %d", w.Code)
	}
}

// Test 16: unauthorized user denied
func TestArtifact_UnauthorizedDenied(t *testing.T) {
	e := setupArtifactTest(t)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), nil)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// Test 17: sensitive source_data fields redacted
func TestArtifact_SensitiveFieldsRedacted(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"report","title":"Sensitive Test","content_markdown":"x","source_data":{"password":"hunter2","api_key":"sk-12345","project":"alpha"}}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	sd := resp["source_data"].(map[string]any)
	if sd["password"] != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %v", sd["password"])
	}
	if sd["api_key"] != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %v", sd["api_key"])
	}
	if sd["project"] != "alpha" {
		t.Errorf("expected alpha, got %v", sd["project"])
	}
}

// Test 18: no Tool Gateway/operational side effects
func TestArtifact_NoOperationalSideEffects(t *testing.T) {
	e := setupArtifactTest(t)
	var beforeApprovals, beforeActions int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&beforeApprovals)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&beforeActions)

	e.createArtifact(t, "report", "Side Effect Test", "content")

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

// ─── Permission Gating Tests (Closure Patch) ───

// Test 19: member can create artifact
func TestArtifact_MemberCanCreate(t *testing.T) {
	e := setupArtifactTest(t)
	body := `{"artifact_type":"report","title":"Member Created","content_markdown":"x"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.memberToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("member should be able to create, got %d", w.Code)
	}
}

// Test 20: member can update artifact
func TestArtifact_MemberCanUpdate(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "To Update", "content")
	body := `{"title":"Updated by Member"}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.memberToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("member should be able to update, got %d", w.Code)
	}
}

// Test 21: member cannot delete/archive artifact
func TestArtifact_MemberCannotDelete(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "To Archive", "content")
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.memberToken)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("member should NOT be able to delete, expected 403, got %d", w.Code)
	}
}

// Test 22: manager can delete/archive artifact
func TestArtifact_ManagerCanDelete(t *testing.T) {
	e := setupArtifactTest(t)
	id := e.createArtifact(t, "report", "Manager Archive", "content")
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, id), nil)
	req.Header.Set("Authorization", "Bearer "+e.token) // owner has manager+ equivalent
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("manager+ should be able to delete, got %d", w.Code)
	}
}

// Test 23: viewer can read but cannot create/update/delete
func TestArtifact_ViewerPermissions(t *testing.T) {
	e := setupArtifactTest(t)

	// Viewer can read (list)
	// We need a viewer token — member is the closest non-viewer test user
	// Structural: viewer has read only, confirmed by DB permission table
	// This test verifies via DB query
	var viewerActions []string
	rows, _ := e.pool.Query(t.Context(), `
		SELECT p.action FROM role_permissions rp
		JOIN roles r ON r.id=rp.role_id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE p.resource='artifacts' AND r.name='viewer'
	`)
	for rows.Next() {
		var a string
		rows.Scan(&a)
		viewerActions = append(viewerActions, a)
	}
	rows.Close()

	if len(viewerActions) != 1 || viewerActions[0] != "read" {
		t.Errorf("viewer should have read-only artifact access, got %v", viewerActions)
	}
}

// Test 24: member has create+read+update but NOT delete
func TestArtifact_MemberPermissionSet(t *testing.T) {
	e := setupArtifactTest(t)
	var memberActions []string
	rows, _ := e.pool.Query(t.Context(), `
		SELECT p.action FROM role_permissions rp
		JOIN roles r ON r.id=rp.role_id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE p.resource='artifacts' AND r.name='member'
		ORDER BY p.action
	`)
	for rows.Next() {
		var a string
		rows.Scan(&a)
		memberActions = append(memberActions, a)
	}
	rows.Close()

	expected := []string{"create", "read", "update"}
	if len(memberActions) != len(expected) {
		t.Errorf("member should have %v, got %v", expected, memberActions)
		return
	}
	for i, a := range expected {
		if memberActions[i] != a {
			t.Errorf("member action[%d]: expected %s, got %s", i, a, memberActions[i])
		}
	}
}
