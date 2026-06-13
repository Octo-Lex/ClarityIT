package integration

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"

func testSetup(t *testing.T, env string) (*Handler, *chi.Mux, string, string) {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testDBURL)
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { pool.Close() })

	hmacKey := "test-hmac-key-for-integration-tests"
	h := NewHandlerWithEnv(pool, hmacKey, env)

	cfg := &config.Config{JWTSecret: "test-jwt-secret-32chars-minimum!!", HMACKey: hmacKey, AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Mount("/api/teams/{teamId}/integration-keys", h.KeyRoutes())
	r.Post("/api/webhooks/{source}", h.ReceiveWebhook)

	// Login to get token
	loginBody := `{"email":"owner@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login failed: %d %s", w.Code, w.Body.String()) }
	var loginResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token, _ := loginResp["access_token"].(string)

	// Get team ID dynamically
	var teamID string
	pool.QueryRow(t.Context(), `SELECT id::text FROM teams LIMIT 1`).Scan(&teamID)

	return h, r, token, teamID
}

func createTestKey(t *testing.T, r *chi.Mux, token, teamID string, allowUnsignedDev bool) (rawKey, signingSecret string) {
	t.Helper()
	body := fmt.Sprintf(`{"name":"test-key","allowed_sources":["grafana"],"allowed_scopes":["webhooks:ingest"],"allow_unsigned_dev":%v}`, allowUnsignedDev)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/integration-keys", teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 { t.Fatalf("create key: %d %s", w.Code, w.Body.String()) }
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	rawKey, _ = resp["key"].(string)
	signingSecret, _ = resp["signing_secret"].(string)
	return
}

func signPayload(signingSecret, timestamp string, body []byte) string {
	signingString := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(signingString))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWebhookValidSignature(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	rawKey, _ := createTestKey(t, r, token, teamID, true)

	payload := []byte(`{"name":"High CPU","severity":"critical","source_id":"node-1"}`)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	sig := signPayload(rawKey, timestamp, payload)

	req := httptest.NewRequest("POST", "/api/webhooks/grafana", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("X-ClarityIT-Signature", sig)
	req.Header.Set("X-ClarityIT-Timestamp", timestamp)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 { t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String()) }
}

func TestWebhookInvalidSignature(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	rawKey, _ := createTestKey(t, r, token, teamID, false)

	payload := []byte(`{"name":"High CPU","severity":"critical","source_id":"node-1"}`)
	timestamp := time.Now().UTC().Format(time.RFC3339)

	req := httptest.NewRequest("POST", "/api/webhooks/grafana", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("X-ClarityIT-Signature", "v1=invalidsignature")
	req.Header.Set("X-ClarityIT-Timestamp", timestamp)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("expected 401 for invalid signature, got %d", w.Code) }
}

func TestWebhookOldTimestampRejected(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	rawKey, _ := createTestKey(t, r, token, teamID, false)

	payload := []byte(`{"name":"High CPU","severity":"critical","source_id":"node-1"}`)
	oldTimestamp := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	sig := signPayload(rawKey, oldTimestamp, payload)

	req := httptest.NewRequest("POST", "/api/webhooks/grafana", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("X-ClarityIT-Signature", sig)
	req.Header.Set("X-ClarityIT-Timestamp", oldTimestamp)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("expected 401 for old timestamp, got %d", w.Code) }
}

func TestWebhookMissingSignatureRejectedInProduction(t *testing.T) {
	_, r, token, teamID := testSetup(t, "production")
	rawKey, _ := createTestKey(t, r, token, teamID, true) // even with allow_unsigned_dev

	payload := []byte(`{"name":"High CPU","severity":"critical","source_id":"node-1"}`)
	req := httptest.NewRequest("POST", "/api/webhooks/grafana", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("expected 401 for missing signature in production, got %d", w.Code) }
}

func TestWebhookUnsignedAllowedInDevWithFlag(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	rawKey, _ := createTestKey(t, r, token, teamID, true)

	payload := []byte(`{"name":"High CPU","severity":"critical","source_id":"node-1"}`)
	req := httptest.NewRequest("POST", "/api/webhooks/grafana", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 { t.Errorf("expected 201 for unsigned dev webhook, got %d: %s", w.Code, w.Body.String()) }
}

// ─── Key Management Tests ───

func TestKeyCreateReturnsRawKeyAndSigningSecret(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")

	body := `{"name":"sig-test-key","allowed_sources":["grafana"],"allowed_scopes":["webhooks:ingest"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/integration-keys", teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 { t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String()) }

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["key"]; !ok { t.Error("missing key in response") }
	if _, ok := resp["signing_secret"]; !ok { t.Error("missing signing_secret in response") }
	if !strings.HasPrefix(resp["key"].(string), "clarity_") { t.Error("key should have clarity_ prefix") }
	if !strings.HasPrefix(resp["signing_secret"].(string), "clss_") { t.Error("signing_secret should have clss_ prefix") }
}

func TestKeyListDoesNotContainRawKey(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	createTestKey(t, r, token, teamID, false)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/integration-keys", teamID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200, got %d", w.Code) }

	body := w.Body.String()
	// Check that full raw keys (44+ chars) don't appear — prefixes (12 chars) are expected
	// The key_prefix field shows "clarity_xxxxxxxx" which is safe
	if strings.Contains(body, "clss_") { t.Error("raw signing_secret appeared in list response") }
	// Verify key field is NOT in the response (only prefix is)
	var listResp []map[string]any
	json.Unmarshal([]byte(body), &listResp)
	for _, item := range listResp {
		if _, ok := item["key"]; ok { t.Error("raw key field appeared in list response") }
		if _, ok := item["signing_secret"]; ok { t.Error("signing_secret field appeared in list response") }
	}
}

func TestKeyCreateRequiresAuth(t *testing.T) {
	_, r, _, teamID := testSetup(t, "development")

	body := `{"name":"no-auth","allowed_sources":["grafana"],"allowed_scopes":["webhooks:ingest"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/integration-keys", teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("expected 401, got %d", w.Code) }
}

func TestRevokeKey(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")

	body := `{"name":"revoke-test","allowed_sources":["grafana"],"allowed_scopes":["webhooks:ingest"]}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/integration-keys", teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 { t.Fatalf("create: %d %s", w.Code, w.Body.String()) }
	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	keyID, _ := createResp["id"].(string)

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/integration-keys/%s", teamID, keyID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("revoke: expected 200, got %d", w.Code) }
}

func TestMissingKeyHeaderReturns401(t *testing.T) {
	_, r, _, _ := testSetup(t, "development")

	payload := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/api/webhooks/grafana", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("expected 401, got %d", w.Code) }
}

func TestInvalidKeyReturns401(t *testing.T) {
	_, r, _, _ := testSetup(t, "development")

	payload := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/api/webhooks/grafana", strings.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", "clarity_invalidkey123")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("expected 401, got %d", w.Code) }
}

func TestWrongSourceRejected(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	rawKey, _ := createTestKey(t, r, token, teamID, true)

	payload := []byte(`{"name":"test","severity":"warning","source_id":"x"}`)
	req := httptest.NewRequest("POST", "/api/webhooks/pagerduty", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 { t.Errorf("expected 403 for wrong source, got %d", w.Code) }
}

// ─── Signature-specific Tests ───

func TestWebhookSignatureNotStoredInAudit(t *testing.T) {
	_, r, token, teamID := testSetup(t, "development")
	rawKey, _ := createTestKey(t, r, token, teamID, true)

	payload := []byte(`{"name":"Audit Test","severity":"warning","source_id":"node-2"}`)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	sig := signPayload(rawKey, timestamp, payload)

	req := httptest.NewRequest("POST", "/api/webhooks/grafana", bytes.NewReader(payload))
	req.Header.Set("X-ClarityIT-Integration-Key", rawKey)
	req.Header.Set("X-ClarityIT-Signature", sig)
	req.Header.Set("X-ClarityIT-Timestamp", timestamp)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 { t.Fatalf("expected 201, got %d", w.Code) }

	// Check audit logs don't contain the raw signature
	pool, _ := pgxpool.New(t.Context(), testDBURL)
	defer pool.Close()
	var metaStr string
	pool.QueryRow(t.Context(), `SELECT new_value::text FROM audit_logs WHERE action='integration.webhook.received' ORDER BY created_at DESC LIMIT 1`).Scan(&metaStr)
	if strings.Contains(metaStr, sig) { t.Error("signature appeared in audit log") }
	if strings.Contains(metaStr, rawKey) { t.Error("integration key appeared in audit log") }
	if !strings.Contains(metaStr, "payload_hash") { t.Error("expected payload_hash in audit log") }
}
