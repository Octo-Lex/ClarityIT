package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"

// fakeS3 stores objects in memory
type fakeS3 struct {
	objects map[string][]byte
}

func newFakeS3() *fakeS3 { return &fakeS3{objects: make(map[string][]byte)} }

func (f *fakeS3) PutObject(_ context.Context, _, key string, data []byte, _ string) error {
	f.objects[key] = data
	return nil
}

func (f *fakeS3) GetPresignedURL(_ context.Context, _, key string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("http://minio:9000/fake-bucket/%s?expires=%v", key, expiry), nil
}

func testSetup(t *testing.T) (*chi.Mux, *pgxpool.Pool, *fakeS3) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })

	s3 := newFakeS3()
	h := NewHandler(pool, s3, "test-bucket")
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/objects/{objectId}/attachments", h.Upload)
		r.Get("/objects/{objectId}/attachments", h.List)
		r.Get("/objects/{objectId}/attachments/{attachmentId}/download-url", h.DownloadURL)
	})
	return r, pool, s3
}

func loginGetTeam(t *testing.T, r *chi.Mux) (string, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte(`{"email":"owner@test.dev","password":"password12"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d", w.Code) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token, _ := resp["access_token"].(string)
	var tid string
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()
	pool.QueryRow(t.Context(), `SELECT t.id::text FROM teams t JOIN team_memberships tm ON t.id=tm.team_id JOIN users u ON tm.user_id=u.id WHERE u.email=$1 LIMIT 1`, "owner@test.dev").Scan(&tid)
	return token, tid
}

func createObject(t *testing.T, pool *pgxpool.Pool, teamID string) string {
	t.Helper()
	var id string
	pool.QueryRow(t.Context(), `INSERT INTO objects (team_id, object_type, title, status) VALUES ($1,'incident','Test Object','active') RETURNING id::text`, teamID).Scan(&id)
	return id
}

func uploadFile(r *chi.Mux, token, teamID, objectID, filename, content string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/objects/%s/attachments", teamID, objectID), &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUploadStoresMetadata(t *testing.T) {
	r, pool, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	oid := createObject(t, pool, tid)

	w := uploadFile(r, token, tid, oid, "test.txt", "hello world")
	if w.Code != 201 { t.Fatalf("upload: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["sha256"] == nil { t.Error("sha256 missing") }
	if resp["size"] == nil { t.Error("size missing") }
	if resp["storage_object_id"] == nil { t.Error("storage_object_id missing") }

	sha256Val, _ := resp["sha256"].(string)

	// Verify storage_objects row exists
	var cnt int
	err := pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM storage_objects WHERE sha256=$1`, sha256Val).Scan(&cnt)
	if err != nil { t.Fatalf("query error: %v", err) }
	if cnt == 0 { t.Errorf("storage_objects row not created (sha256=%s)", sha256Val) }

	// Verify object_storage_refs row exists
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM object_storage_refs WHERE object_id=$1`, oid).Scan(&cnt)
	if cnt != 1 { t.Error("object_storage_refs row not created") }
}

func TestUploadStoresObjectInS3(t *testing.T) {
	r, pool, s3 := testSetup(t)
	token, tid := loginGetTeam(t, r)
	oid := createObject(t, pool, tid)

	w := uploadFile(r, token, tid, oid, "doc.pdf", "PDF content here")
	if w.Code != 201 { t.Fatalf("upload: %d", w.Code) }

	if len(s3.objects) == 0 { t.Error("no objects stored in S3") }
	for k, v := range s3.objects {
		if v == nil { t.Errorf("S3 object %s has nil data", k) }
		if string(v) != "PDF content here" { t.Errorf("S3 data mismatch: got %q", string(v)) }
	}
}

func TestAuditExcludesBytes(t *testing.T) {
	r, pool, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	oid := createObject(t, pool, tid)

	w := uploadFile(r, token, tid, oid, "secret.txt", "SECRET DATA 12345")
	if w.Code != 201 { t.Fatalf("upload: %d", w.Code) }

	// Check audit doesn't contain file bytes
	var nv string
	pool.QueryRow(t.Context(), `SELECT new_value::text FROM audit_logs WHERE action='object.attachment.created' ORDER BY created_at DESC LIMIT 1`).Scan(&nv)
	if nv == "" { t.Fatal("no audit row") }
	if bytes.Contains([]byte(nv), []byte("SECRET DATA 12345")) {
		t.Error("audit contains file bytes — data leak")
	}
	if !bytes.Contains([]byte(nv), []byte("sha256")) {
		t.Error("audit missing sha256")
	}
}

func TestDownloadURLHasExpiry(t *testing.T) {
	r, pool, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	oid := createObject(t, pool, tid)

	w := uploadFile(r, token, tid, oid, "file.txt", "content")
	if w.Code != 201 { t.Fatalf("upload: %d", w.Code) }
	var uploadResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &uploadResp)
	attachmentID := uploadResp["id"].(string)

	// Get download URL
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/objects/%s/attachments/%s/download-url", tid, oid, attachmentID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("download URL: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	url, _ := resp["url"].(string)
	if url == "" { t.Error("no URL returned") }
	if resp["expires_in"] == nil { t.Error("no expires_in") }
	expiresIn := int(resp["expires_in"].(float64))
	if expiresIn != 900 { t.Errorf("expected 900s expiry, got %d", expiresIn) }
}

func TestCrossTeamAccessDenied(t *testing.T) {
	r, pool, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	oid := createObject(t, pool, tid)

	w := uploadFile(r, token, tid, oid, "file.txt", "private data")
	if w.Code != 201 { t.Fatalf("upload: %d", w.Code) }

	// Try accessing from different team (use a different team ID)
	fakeTeamID := uuid.New().String()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/objects/%s/attachments", fakeTeamID, oid), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Should return empty list (not the other team's attachments)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 { t.Error("cross-team attachment data leaked") }
}

func TestListAttachments(t *testing.T) {
	r, pool, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	oid := createObject(t, pool, tid)

	uploadFile(r, token, tid, oid, "file1.txt", "content1")
	uploadFile(r, token, tid, oid, "file2.txt", "content2")

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/objects/%s/attachments", tid, oid), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("list: %d %s", w.Code, w.Body.String()) }
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 2 { t.Errorf("expected 2 attachments, got %d", len(list)) }
}

func TestUploadRequiresAuth(t *testing.T) {
	r, _, _ := testSetup(t)
	req := httptest.NewRequest("POST", "/api/teams/00000000-0000-0000-0000-000000000000/objects/00000000-0000-0000-0000-000000000000/attachments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("want 401, got %d", w.Code) }
}
