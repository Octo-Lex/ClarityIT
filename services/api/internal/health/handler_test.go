package health

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"

func TestDeepHealth(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, "test-build")
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.With(middleware.RequireAuth).Get("/api/health/deep", h.Deep)

	// Login to get token
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"owner@test.dev","password":"password12"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d", w.Code) }
	var loginResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token, _ := loginResp["access_token"].(string)

	// Deep health
	req = httptest.NewRequest("GET", "/api/health/deep", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("deep health: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	pg, _ := resp["postgres"].(map[string]any)
	if pg["status"] != "up" { t.Error("expected postgres up") }
	if resp["build"] != "test-build" { t.Error("expected build") }
	if resp["outbox"] == nil { t.Error("expected outbox in health") }
	if resp["agent_runs"] == nil { t.Error("expected agent_runs in health") }
	if resp["timestamp"] == nil { t.Error("expected timestamp") }
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
