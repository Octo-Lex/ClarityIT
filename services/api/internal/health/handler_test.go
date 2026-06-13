package health

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

func TestDeepHealth(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, "test-build")
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.With(middleware.RequireAuth).Get("/api/health/deep", h.Deep)

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"owner@test.dev","password":"password12"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d", w.Code) }
	var loginResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token, _ := loginResp["access_token"].(string)

	req = httptest.NewRequest("GET", "/api/health/deep", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("deep health: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	pg, _ := resp["postgres"].(map[string]any)
	if pg["status"] != "up" { t.Error("expected postgres up") }
	if resp["build"] == nil { t.Error("expected build") }
	if resp["outbox"] == nil { t.Error("expected outbox in health") }
	if resp["workers"] == nil { t.Error("expected workers in health") }
	if resp["uptime"] == nil { t.Error("expected uptime") }
	// NATS/Redis/MinIO should be present (even if "unknown")
	if resp["nats"] == nil { t.Error("expected nats in health") }
	if resp["redis"] == nil { t.Error("expected redis in health") }
	if resp["minio"] == nil { t.Error("expected minio in health") }
}

func TestMetricsEndpoint(t *testing.T) {
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, "test")
	r := chi.NewRouter()
	r.Get("/metrics", h.Metrics)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("metrics: %d", w.Code) }
	body := w.Body.String()
	if !strings.Contains(body, "clarity_http_requests_total") {
		t.Error("missing http_requests_total metric")
	}
	if !strings.Contains(body, "clarity_outbox_pending_count") {
		t.Error("missing outbox_pending_count metric")
	}
	if !strings.Contains(body, "clarity_webhook_received_total") {
		t.Error("missing webhook_received_total metric")
	}
	if !strings.Contains(body, "clarity_agent_tool_blocked_total") {
		t.Error("missing agent_tool_blocked_total metric")
	}
}

func TestDeepHealthRequiresAuth(t *testing.T) {
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, "test")
	r := chi.NewRouter()
	r.With(middleware.ResolveAuth("secret")).With(middleware.RequireAuth).Get("/api/health/deep", h.Deep)

	req := httptest.NewRequest("GET", "/api/health/deep", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("want 401, got %d", w.Code) }
}

// TestMinIOHealthChecker checks that the MinIO health checker can be instantiated.
func TestMinIOHealthChecker(t *testing.T) {
	m := NewMinIOHealthChecker("minio:9000", false)
	if m == nil { t.Fatal("expected non-nil checker") }
	if m.endpoint != "minio:9000" { t.Error("wrong endpoint") }
}

// TestS3BucketCheckerInterface verifies MinIOHealthChecker implements S3BucketChecker.
func TestS3BucketCheckerInterface(t *testing.T) {
	var _ S3BucketChecker = (*MinIOHealthChecker)(nil)
}

// TestDeepHealthWithDeps tests that the handler works with dependency clients.
func TestDeepHealthWithDeps(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	// Create handler with nil deps — should report "unknown" for each
	h := NewHandlerWithDeps(pool, "0.8.0", "", nil, nil, nil, "")
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.With(middleware.RequireAuth).Get("/api/health/deep", h.Deep)

	// Login
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"owner@test.dev","password":"password12"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var loginResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token, _ := loginResp["access_token"].(string)

	// Deep health
	req = httptest.NewRequest("GET", "/api/health/deep", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("deep health: %d", w.Code) }

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	nats, _ := resp["nats"].(map[string]any)
	if nats["status"] != "unknown" { t.Errorf("expected nats unknown with nil client, got %v", nats["status"]) }

	redis, _ := resp["redis"].(map[string]any)
	if redis["status"] != "unknown" { t.Errorf("expected redis unknown with nil client, got %v", redis["status"]) }

	minio, _ := resp["minio"].(map[string]any)
	if minio["status"] != "unknown" { t.Errorf("expected minio unknown with nil client, got %v", minio["status"]) }
}

// stubS3Checker is a test double for S3BucketChecker.
type stubS3Checker struct {
	err error
}

func (s *stubS3Checker) HeadBucket(ctx context.Context, bucket string) error {
	return s.err
}

func TestCheckMinIOWithStub(t *testing.T) {
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	// Stub returns nil — should be "up"
	h := &Handler{pool: pool, s3: &stubS3Checker{err: nil}, bucket: "test"}
	health := h.checkMinIO(t.Context())
	if health.Status != "up" { t.Errorf("expected up, got %s", health.Status) }

	// Stub returns error — should be "down"
	h2 := &Handler{pool: pool, s3: &stubS3Checker{err: context.DeadlineExceeded}, bucket: "test"}
	health2 := h2.checkMinIO(t.Context())
	if health2.Status != "down" { t.Errorf("expected down, got %s", health2.Status) }

	// No bucket — should be "unknown"
	h3 := &Handler{pool: pool, s3: &stubS3Checker{}, bucket: ""}
	health3 := h3.checkMinIO(t.Context())
	if health3.Status != "unknown" { t.Errorf("expected unknown, got %s", health3.Status) }
}

// Prevent unused import
var _ = time.Now
