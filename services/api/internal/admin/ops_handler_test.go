package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if w.Code != 200 { t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String()) }

	// Should return array
	body := w.Body.String()
	if !strings.HasPrefix(body, "[") {
		t.Error("expected array response")
	}
}

// TestOpsDeadLettersNullableAggregateID verifies a dead-letter row with
// aggregate_id=NULL (as the context-worker records for unparseable messages)
// renders correctly as JSON null, not an HTTP 500 from a Scan error.
func TestOpsDeadLettersNullableAggregateID(t *testing.T) {
	r, token := opsTestSetup(t)
	pool, _ := pgxpool.New(t.Context(), opsDBURL)
	defer pool.Close()

	// Insert a dead-letter row with nullable aggregate_id/type and last_error.
	etag := fmt.Sprintf("dlq-null-agg-%d", time.Now().UnixNano())
	_, err := pool.Exec(t.Context(), `
		INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload, status, dead_lettered_at, attempts, last_error, next_attempt_at, purge_after)
		VALUES ($1, NULL, NULL, '{}'::jsonb, 'dead_letter', NOW(), 10, NULL, NOW(), NOW() + INTERVAL '30 days')
	`, etag)
	if err != nil {
		t.Fatalf("seed dead-letter: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE event_type=$1", etag) })

	req := httptest.NewRequest("GET", "/api/admin/ops/dead-letters?limit=200", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var items []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var found bool
	for _, it := range items {
		if it["event_type"] == etag {
			found = true
			// NULL must render as JSON null, not "" or an omitted field.
			if _, present := it["aggregate_id"]; !present {
				t.Errorf("aggregate_id key missing; want present with null value")
			} else if it["aggregate_id"] != nil {
				t.Errorf("aggregate_id = %v; want null", it["aggregate_id"])
			}
			if it["aggregate_type"] != nil {
				t.Errorf("aggregate_type = %v; want null", it["aggregate_type"])
			}
			if it["error_message"] != nil {
				t.Errorf("error_message = %v; want null", it["error_message"])
			}
		}
	}
	if !found {
		t.Errorf("seeded nullable dead-letter row %q not found in response", etag)
	}
}

// TestOpsOutboxNullableColumns verifies a normal pending outbox row (which
// commonly has last_error=NULL) renders without a 500 once Scan errors are
// checked. Before the fix, scanning NULL last_error into a non-pointer string
// would error.
func TestOpsOutboxNullableColumns(t *testing.T) {
	r, token := opsTestSetup(t)
	pool, _ := pgxpool.New(t.Context(), opsDBURL)
	defer pool.Close()

	// Insert a normal pending row: last_error=NULL (the common case).
	etag := fmt.Sprintf("pending-null-err-%d", time.Now().UnixNano())
	_, err := pool.Exec(t.Context(), `
		INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload, status, attempts, last_error, next_attempt_at, purge_after)
		VALUES ($1, 'object', '00000000-0000-0000-0000-000000000001', '{}'::jsonb, 'pending', 0, NULL, NOW(), NOW() + INTERVAL '7 days')
	`, etag)
	if err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE event_type=$1", etag) })

	req := httptest.NewRequest("GET", "/api/admin/ops/outbox?limit=200", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var events []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var found bool
	for _, e := range events {
		if e["event_type"] == etag {
			found = true
			if e["error_message"] != nil {
				t.Errorf("error_message = %v; want null for pending row", e["error_message"])
			}
		}
	}
	if !found {
		t.Errorf("seeded pending row %q not found in outbox response", etag)
	}
}

// TestOpsDeadLettersLegacyNullableTypeAndError covers a legacy dead-letter row
// with BOTH aggregate_type and last_error NULL, ensuring the reader handles
// every nullable column without a 500.
func TestOpsDeadLettersLegacyNullableTypeAndError(t *testing.T) {
	r, token := opsTestSetup(t)
	pool, _ := pgxpool.New(t.Context(), opsDBURL)
	defer pool.Close()

	etag := fmt.Sprintf("legacy-null-type-%d", time.Now().UnixNano())
	_, err := pool.Exec(t.Context(), `
		INSERT INTO outbox_events (event_type, aggregate_type, aggregate_id, payload, status, dead_lettered_at, attempts, last_error, next_attempt_at, purge_after)
		VALUES ($1, NULL, NULL, '{}'::jsonb, 'dead_letter', NOW(), 7, NULL, NOW(), NOW() + INTERVAL '30 days')
	`, etag)
	if err != nil {
		t.Fatalf("seed legacy: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM outbox_events WHERE event_type=$1", etag) })

	req := httptest.NewRequest("GET", "/api/admin/ops/dead-letters?limit=200", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, etag) {
		t.Errorf("legacy row %q not in response", etag)
	}
}
