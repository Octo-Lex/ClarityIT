package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"
const testEmail = "owner@test.dev"
const testPass = "password12"

func testSetup(t *testing.T) (*chi.Mux, *pgxpool.Pool) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })

	h := NewHandler(pool)
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/integration-keys", h.CreateKey)
		r.Get("/integration-keys", h.ListKeys)
		r.Delete("/integration-keys/{keyId}", h.RevokeKey)
	})
	r.Post("/api/webhooks/{source}", h.ReceiveWebhook)
	return r, pool
}

func doReq(r *chi.Mux, method, path, tok string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var br *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		br = bytes.NewReader(b)
	} else {
		br = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, path, br)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginGetTeam(t *testing.T, r *chi.Mux) (string, string) {
	t.Helper()
	w := doReq(r, "POST", "/api/auth/login", "", map[string]string{"email": testEmail, "password": testPass}, nil)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token, _ := resp["access_token"].(string)
	var tid string
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()
	pool.QueryRow(t.Context(), `SELECT t.id::text FROM teams t JOIN team_memberships tm ON t.id=tm.team_id JOIN users u ON tm.user_id=u.id WHERE u.email=$1 LIMIT 1`, testEmail).Scan(&tid)
	return token, tid
}

func createTestKey(t *testing.T, r *chi.Mux, token, tid string, name string, sources, scopes []string) (rawKey, keyID string) {
	t.Helper()
	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token,
		map[string]any{"name": name, "allowed_sources": sources, "allowed_scopes": scopes},
		map[string]string{"Idempotency-Key": uuid.New().String()})
	if w.Code != 201 {
		t.Fatalf("create key: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	rawKey, _ = resp["key"].(string)
	keyID, _ = resp["id"].(string)
	return
}

// ─── Tests ───

func TestKeyCreateReturnsRawKeyOnce(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, keyID := createTestKey(t, r, token, tid, "Test Key", []string{"grafana"}, []string{"webhooks:ingest"})
	if rawKey == "" {
		t.Error("raw key not returned")
	}
	if keyID == "" {
		t.Error("key ID not returned")
	}
	if len(rawKey) < 20 {
		t.Error("key too short")
	}
}

func TestKeyListDoesNotContainRawKey(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "List Test", []string{"*"}, []string{"*"})

	w := doReq(r, "GET", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token, nil, nil)
	if w.Code != 200 {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Raw key must not appear in list response
	if bytes.Contains([]byte(body), []byte(rawKey)) {
		t.Error("raw key leaked in list response")
	}
}

func TestKeyHashStoredNotRaw(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "Hash Test", []string{"*"}, []string{"*"})

	// Compute hash
	h := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(h[:])

	// Verify hash exists in DB
	var count int
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM integration_api_keys WHERE key_hash=$1`, hash).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 key with hash, got %d", count)
	}
}

func TestRevokeKey(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	_, keyID := createTestKey(t, r, token, tid, "Revoke Test", []string{"grafana"}, []string{"webhooks:ingest"})

	w := doReq(r, "DELETE", fmt.Sprintf("/api/teams/%s/integration-keys/%s", tid, keyID), token, nil,
		map[string]string{"Idempotency-Key": uuid.New().String()})
	if w.Code != 200 {
		t.Fatalf("revoke: %d %s", w.Code, w.Body.String())
	}
}

func TestRevokedKeyRejected(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, keyID := createTestKey(t, r, token, tid, "Revoke Reject", []string{"grafana"}, []string{"webhooks:ingest"})

	// Revoke
	doReq(r, "DELETE", fmt.Sprintf("/api/teams/%s/integration-keys/%s", tid, keyID), token, nil,
		map[string]string{"Idempotency-Key": uuid.New().String()})

	// Use revoked key
	w := doReq(r, "POST", "/api/webhooks/grafana", "", map[string]any{"name": "Test"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 401 {
		t.Errorf("want 401 for revoked key, got %d", w.Code)
	}
}

func TestWrongSourceRejected(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "Source Test", []string{"grafana"}, []string{"webhooks:ingest"})

	w := doReq(r, "POST", "/api/webhooks/prometheus", "", map[string]any{"name": "Test"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 403 {
		t.Errorf("want 403 for wrong source, got %d", w.Code)
	}
}

func TestWildcardSourceAccepted(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "Wildcard Test", []string{"*"}, []string{"webhooks:ingest"})

	w := doReq(r, "POST", "/api/webhooks/anything", "", map[string]any{"name": "Test Alert"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 201 {
		t.Errorf("want 201 for wildcard source, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMissingScopeRejected(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "No Scope", []string{"grafana"}, []string{"read_only"})

	w := doReq(r, "POST", "/api/webhooks/grafana", "", map[string]any{"name": "Test"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 403 {
		t.Errorf("want 403 for missing scope, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidWebhookCreatesAlert(t *testing.T) {
	r, pool := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "Alert Test", []string{"grafana"}, []string{"webhooks:ingest"})

	w := doReq(r, "POST", "/api/webhooks/grafana", "",
		map[string]any{"name": "High CPU", "severity": "critical", "source_id": "node-1"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 201 {
		t.Fatalf("webhook: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "created" {
		t.Error("expected status=created")
	}

	// Verify alert object in DB
	var cnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM alerts WHERE source='grafana' AND severity='critical'`).Scan(&cnt)
	if cnt == 0 {
		t.Error("alert not created in DB")
	}

	// Verify audit written
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM audit_logs WHERE action='integration.webhook.received'`).Scan(&cnt)
	if cnt == 0 {
		t.Error("audit not written")
	}

	// Verify outbox event
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM outbox_events WHERE event_type='clarity.v1.integration.webhook.received'`).Scan(&cnt)
	if cnt == 0 {
		t.Error("outbox event not written")
	}
}

func TestPIINotInAudit(t *testing.T) {
	r, pool := testSetup(t)
	token, tid := loginGetTeam(t, r)
	rawKey, _ := createTestKey(t, r, token, tid, "PII Test", []string{"grafana"}, []string{"webhooks:ingest"})

	w := doReq(r, "POST", "/api/webhooks/grafana", "",
		map[string]any{"name": "Alert", "severity": "low", "source_id": "srv-1", "secret_field": "should_not_appear"},
		map[string]string{"X-ClarityIT-Integration-Key": rawKey})
	if w.Code != 201 {
		t.Fatalf("webhook: %d", w.Code)
	}

	// Check audit doesn't contain raw secret
	var nv string
	pool.QueryRow(t.Context(), `SELECT new_value::text FROM audit_logs WHERE action='integration.webhook.received' ORDER BY created_at DESC LIMIT 1`).Scan(&nv)
	if nv == "" {
		t.Fatal("no audit row")
	}
	if bytes.Contains([]byte(nv), []byte("should_not_appear")) {
		t.Error("audit contains raw payload data — PII leak")
	}
	if !bytes.Contains([]byte(nv), []byte("payload_hash")) {
		t.Error("audit missing payload_hash")
	}
}

func TestNoKeyHeaderReturns401(t *testing.T) {
	r, _ := testSetup(t)
	w := doReq(r, "POST", "/api/webhooks/grafana", "", map[string]any{"name": "Test"}, nil)
	if w.Code != 401 {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestInvalidKeyReturns401(t *testing.T) {
	r, _ := testSetup(t)
	w := doReq(r, "POST", "/api/webhooks/grafana", "", map[string]any{"name": "Test"},
		map[string]string{"X-ClarityIT-Integration-Key": "clarity_invalidkey123456789"})
	if w.Code != 401 {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestKeyCreateRequiresAuth(t *testing.T) {
	r, _ := testSetup(t)
	w := doReq(r, "POST", "/api/teams/00000000-0000-0000-0000-000000000000/integration-keys", "",
		map[string]any{"name": "NoAuth"}, nil)
	if w.Code != 401 {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestKeyCreateRequiresAllFields(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token,
		map[string]any{"name": ""}, map[string]string{"Idempotency-Key": uuid.New().String()})
	if w.Code != 400 {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestKeyCreateDuplicatesAllowed(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := loginGetTeam(t, r)
	body := map[string]any{"name": "Dup Key", "allowed_sources": []string{"test"}, "allowed_scopes": []string{"*"}}
	w1 := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token, body, map[string]string{"Idempotency-Key": uuid.New().String()})
	w2 := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/integration-keys", tid), token, body, map[string]string{"Idempotency-Key": uuid.New().String()})
	if w1.Code != 201 || w2.Code != 201 { t.Fatalf("both should succeed: %d %d", w1.Code, w2.Code) }
	var r1, r2 map[string]any
	json.Unmarshal(w1.Body.Bytes(), &r1)
	json.Unmarshal(w2.Body.Bytes(), &r2)
	if r1["key"] == r2["key"] { t.Error("different keys should have different raw values") }
}
