package admin

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/health"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const opsDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

func opsTestSetup(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), opsDBURL)
	t.Cleanup(func() { pool.Close() })

	hc := health.NewHandler(pool, "test")
	opsH := NewOpsHandler(pool, hc)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Use(middleware.RequirePlatformRole(pool, "platform_owner"))
		r.Get("/ops/summary", opsH.Summary)
		r.Get("/ops/outbox", opsH.Outbox)
		r.Get("/ops/dead-letters", opsH.DeadLetters)
		r.Get("/ops/workers", opsH.Workers)
		r.Get("/ops/webhooks/rejections", opsH.WebhookRejections)
		r.Get("/ops/agent-blocks", opsH.AgentBlocks)
	})

	// Login as owner
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"owner@test.dev","password":"password12"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token, _ := resp["access_token"].(string)

	return r, token
}

func TestOpsSummary(t *testing.T) {
	r, token := opsTestSetup(t)

	req := httptest.NewRequest("GET", "/api/admin/ops/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String()) }

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	// Must contain expected fields
	required := []string{"outbox_pending", "dead_letters", "agent_runs_pending", "webhook_rejections_24h", "agent_blocks_24h", "total_users", "total_teams", "integration_keys_active"}
	for _, key := range required {
		if _, ok := resp[key]; !ok {
			t.Errorf("missing field: %s", key)
		}
	}
}

func TestOpsSummaryRequiresPlatformOwner(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), opsDBURL)
	defer pool.Close()

	hc := health.NewHandler(pool, "test")
	opsH := NewOpsHandler(pool, hc)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Use(middleware.RequirePlatformRole(pool, "platform_owner"))
		r.Get("/ops/summary", opsH.Summary)
	})

	// Login as member (non-owner)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"member@test.dev","password":"password12"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d", w.Code) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token, _ := resp["access_token"].(string)

	// Try to access ops
	req = httptest.NewRequest("GET", "/api/admin/ops/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 { t.Errorf("expected 403 for non-platform-owner, got %d", w.Code) }
}

func TestOpsDeadLettersRedactPayload(t *testing.T) {
	r, token := opsTestSetup(t)

	req := httptest.NewRequest("GET", "/api/admin/ops/dead-letters", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }

	body := w.Body.String()
	// Must not contain raw payload field
	if strings.Contains(body, "\"payload\"") {
		t.Error("dead letter response contains raw payload field")
	}
}

func TestOpsWebhookRejectionsRedactKey(t *testing.T) {
	r, token := opsTestSetup(t)

	req := httptest.NewRequest("GET", "/api/admin/ops/webhooks/rejections", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }

	body := w.Body.String()
	// Must not contain raw keys or payloads
	if strings.Contains(body, "clarity_") {
		t.Error("webhook rejections contain raw integration key prefix")
	}
	if strings.Contains(body, "\"new_value\"") {
		t.Error("webhook rejections contain new_value field")
	}
}

func TestOpsWorkers(t *testing.T) {
	r, token := opsTestSetup(t)

	req := httptest.NewRequest("GET", "/api/admin/ops/workers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }

	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) < 3 {
		t.Errorf("expected at least 3 workers, got %d", len(resp))
	}
	// Each worker must have name + status
	for _, w := range resp {
		if _, ok := w["name"]; !ok { t.Error("worker missing name") }
		if _, ok := w["status"]; !ok { t.Error("worker missing status") }
	}
}

func TestOpsOutbox(t *testing.T) {
	r, token := opsTestSetup(t)

	req := httptest.NewRequest("GET", "/api/admin/ops/outbox", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }

	// Should return array (possibly empty)
	body := w.Body.String()
	if !strings.HasPrefix(body, "[") {
		t.Error("expected array response")
	}
}

func TestOpsAgentBlocks(t *testing.T) {
	r, token := opsTestSetup(t)

	req := httptest.NewRequest("GET", "/api/admin/ops/agent-blocks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }

	// Should return array
	body := w.Body.String()
	if !strings.HasPrefix(body, "[") {
		t.Error("expected array response")
	}
}
