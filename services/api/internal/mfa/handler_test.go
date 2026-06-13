package mfa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

func testSetup(t *testing.T) (*pgxpool.Pool, *chi.Mux, string) {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-jwt-secret",
		HMACKey:         "test-hmac-key-at-least-32-characters",
		MFAKey:          "test-mfa-key-at-least-32-characters!!",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
		Env:             "development",
	}

	pool, err := pgxpool.New(t.Context(), testDBURL)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Clean up MFA tables for test user
	pool.Exec(t.Context(), "DELETE FROM mfa_recovery_codes WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev')")
	pool.Exec(t.Context(), "DELETE FROM mfa_challenges WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev')")
	pool.Exec(t.Context(), "DELETE FROM user_mfa_factors WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev')")
	// Reset recent_mfa_at
	pool.Exec(t.Context(), "UPDATE user_sessions SET recent_mfa_at=NULL WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev')")

	h, err := NewHandler(pool, cfg)
	if err != nil {
		t.Fatalf("create MFA handler: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))

	// Mock auth: login as owner@test.dev
	r.Post("/api/auth/login", mockLogin(pool, cfg))
	r.Route("/api/auth/mfa", func(sr chi.Router) {
		sr.Use(middleware.RequireAuth)
		sr.Mount("/", h.Routes())
	})

	// Login to get token
	token := doLogin(t, r)

	return pool, r, token
}

func mockLogin(pool *pgxpool.Pool, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var userID string
		var name string
		err := pool.QueryRow(r.Context(),
			"SELECT id::text, name FROM users WHERE email=$1", req.Email).Scan(&userID, &name)
		if err != nil {
			writeErr(w, 401, "Invalid credentials")
			return
		}

		// Get team context
		var teamID string
		pool.QueryRow(r.Context(),
			"SELECT team_id::text FROM team_memberships WHERE user_id=$1 LIMIT 1", userID).Scan(&teamID)

		token, _ := issueTestToken(cfg.JWTSecret, userID, req.Email, name, teamID)

		// Create session
		pool.Exec(r.Context(), `
			INSERT INTO user_sessions (id, user_id, expires_at)
			VALUES (gen_random_uuid(), $1, NOW() + interval '7 days')
		`, userID)

		writeJSON(w, 200, map[string]any{
			"access_token": token,
			"token_type":   "bearer",
		})
	}
}

func doLogin(t *testing.T, r *chi.Mux) string {
	body := `{"email":"owner@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["access_token"].(string)
}

// Test 1: Enroll TOTP
func TestEnrollTOTP(t *testing.T) {
	_, r, token := testSetup(t)

	req := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["provisioning_uri"] == nil {
		t.Error("missing provisioning_uri")
	}
	if resp["secret"] == nil {
		t.Error("missing secret")
	}
	uri, _ := resp["provisioning_uri"].(string)
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Error("provisioning URI should start with otpauth://totp/")
	}
}

// Test 2: Verify valid TOTP code activates factor
func TestVerifyEnrollmentValidCode(t *testing.T) {
	pool, r, token := testSetup(t)

	// Enroll first
	req := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("enroll failed: %d", w.Code)
	}

	// Get the secret from enrollment response
	var enrollResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &enrollResp)
	secretB32, _ := enrollResp["secret"].(string)

	// Decode secret to generate valid TOTP
	encoder := newBase32Decoder()
	secret, err := encoder(secretB32)
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}

	// Generate valid TOTP code
	code := GenerateTOTP(secret, time.Now())

	// Verify enrollment
	verifyBody := fmt.Sprintf(`{"code":"%s"}`, code)
	req = httptest.NewRequest("POST", "/api/auth/mfa/totp/verify-enrollment", strings.NewReader(verifyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	codes, ok := resp["recovery_codes"].([]any)
	if !ok || len(codes) != recoveryCodeCount {
		t.Errorf("expected %d recovery codes, got %v", recoveryCodeCount, codes)
	}

	// Verify factor is active in DB
	var status string
	pool.QueryRow(t.Context(),
		"SELECT status FROM user_mfa_factors WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev') AND status='active'").
		Scan(&status)
	if status != "active" {
		t.Errorf("factor should be active, got %s", status)
	}
}

// Test 3: Reject invalid TOTP code
func TestRejectInvalidTOTP(t *testing.T) {
	_, r, token := testSetup(t)

	// Enroll
	req := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("enroll failed: %d", w.Code)
	}

	// Try invalid code
	verifyBody := `{"code":"000000"}`
	req = httptest.NewRequest("POST", "/api/auth/mfa/totp/verify-enrollment", strings.NewReader(verifyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for invalid code, got %d", w.Code)
	}
}

// Test 4: Recent MFA required for high-risk action
func TestRecentMFARequired(t *testing.T) {
	pool, r, token := testSetup(t)
	_ = pool

	// Create a mock high-risk endpoint protected by RequireRecentMFA
	protected := chi.NewRouter()
	protected.Use(middleware.ResolveAuth("test-jwt-secret"))
	protected.Use(middleware.RequireAuth)
	protected.With(RequireRecentMFA(pool)).Post("/high-risk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"message": "executed"})
	})

	// Without recent MFA — should be blocked
	req := httptest.NewRequest("POST", "/high-risk", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 without recent MFA, got %d", w.Code)
	}

	// Now do full MFA verification
	enrollReq := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	enrollReq.Header.Set("Authorization", "Bearer "+token)
	enrollW := httptest.NewRecorder()
	r.ServeHTTP(enrollW, enrollReq)

	var enrollResp map[string]any
	json.Unmarshal(enrollW.Body.Bytes(), &enrollResp)
	secretB32, _ := enrollResp["secret"].(string)
	secret, _ := newBase32Decoder()(secretB32)

	verifyBody := fmt.Sprintf(`{"code":"%s"}`, GenerateTOTP(secret, time.Now()))
	verifyReq := httptest.NewRequest("POST", "/api/auth/mfa/totp/verify-enrollment", strings.NewReader(verifyBody))
	verifyReq.Header.Set("Authorization", "Bearer "+token)
	verifyReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), verifyReq)

	// Create MFA challenge
	chReq := httptest.NewRequest("POST", "/api/auth/mfa/challenge", nil)
	chReq.Header.Set("Authorization", "Bearer "+token)
	chW := httptest.NewRecorder()
	r.ServeHTTP(chW, chReq)
	if chW.Code != 200 {
		t.Fatalf("challenge failed: %d %s", chW.Code, chW.Body.String())
	}
	var chResp map[string]any
	json.Unmarshal(chW.Body.Bytes(), &chResp)
	challengeID, _ := chResp["challenge_id"].(string)

	// Verify challenge with TOTP
	verifyChBody := fmt.Sprintf(`{"challenge_id":"%s","code":"%s"}`, challengeID, GenerateTOTP(secret, time.Now()))
	verifyChReq := httptest.NewRequest("POST", "/api/auth/mfa/verify", strings.NewReader(verifyChBody))
	verifyChReq.Header.Set("Authorization", "Bearer "+token)
	verifyChReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), verifyChReq)

	// Now high-risk should succeed
	req = httptest.NewRequest("POST", "/high-risk", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 with recent MFA, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 5: Expired MFA window rejected
func TestExpiredMFAWindow(t *testing.T) {
	pool, _, token := testSetup(t)

	// Manually set recent_mfa_at to past
	pool.Exec(t.Context(), `
		UPDATE user_sessions SET recent_mfa_at=NOW() - interval '10 minutes'
		WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev')
	`)

	protected := chi.NewRouter()
	protected.Use(middleware.ResolveAuth("test-jwt-secret"))
	protected.Use(middleware.RequireAuth)
	protected.With(RequireRecentMFA(pool)).Post("/high-risk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"message": "executed"})
	})

	req := httptest.NewRequest("POST", "/high-risk", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for expired MFA, got %d", w.Code)
	}
}

// Test 6: Recovery code single-use
func TestRecoveryCodeSingleUse(t *testing.T) {
	_, r, token := testSetup(t)

	// Enroll + verify to get recovery codes
	req := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var enrollResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &enrollResp)
	secretB32, _ := enrollResp["secret"].(string)
	secret, _ := newBase32Decoder()(secretB32)

	verifyBody := fmt.Sprintf(`{"code":"%s"}`, GenerateTOTP(secret, time.Now()))
	req = httptest.NewRequest("POST", "/api/auth/mfa/totp/verify-enrollment", strings.NewReader(verifyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var verifyResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &verifyResp)
	recoveryCodes, _ := verifyResp["recovery_codes"].([]any)
	if len(recoveryCodes) == 0 {
		t.Fatal("no recovery codes returned")
	}
	firstCode, _ := recoveryCodes[0].(string)

	// Create challenge
	chReq := httptest.NewRequest("POST", "/api/auth/mfa/challenge", nil)
	chReq.Header.Set("Authorization", "Bearer "+token)
	chW := httptest.NewRecorder()
	r.ServeHTTP(chW, chReq)
	var chResp map[string]any
	json.Unmarshal(chW.Body.Bytes(), &chResp)
	challengeID, _ := chResp["challenge_id"].(string)

	// First use — should succeed
	rcBody := fmt.Sprintf(`{"challenge_id":"%s","recovery_code":"%s"}`, challengeID, firstCode)
	rcReq := httptest.NewRequest("POST", "/api/auth/mfa/verify", strings.NewReader(rcBody))
	rcReq.Header.Set("Authorization", "Bearer "+token)
	rcReq.Header.Set("Content-Type", "application/json")
	rcW := httptest.NewRecorder()
	r.ServeHTTP(rcW, rcReq)
	if rcW.Code != 200 {
		t.Fatalf("first recovery code use should succeed: %d %s", rcW.Code, rcW.Body.String())
	}

	// Create new challenge for second attempt
	chReq = httptest.NewRequest("POST", "/api/auth/mfa/challenge", nil)
	chReq.Header.Set("Authorization", "Bearer "+token)
	chW = httptest.NewRecorder()
	r.ServeHTTP(chW, chReq)
	json.Unmarshal(chW.Body.Bytes(), &chResp)
	challengeID2, _ := chResp["challenge_id"].(string)

	// Second use of same code — should fail
	rcBody = fmt.Sprintf(`{"challenge_id":"%s","recovery_code":"%s"}`, challengeID2, firstCode)
	rcReq = httptest.NewRequest("POST", "/api/auth/mfa/verify", strings.NewReader(rcBody))
	rcReq.Header.Set("Authorization", "Bearer "+token)
	rcReq.Header.Set("Content-Type", "application/json")
	rcW = httptest.NewRecorder()
	r.ServeHTTP(rcW, rcReq)

	if rcW.Code != 401 {
		t.Errorf("reused recovery code should fail, got %d", rcW.Code)
	}
}

// Test 7: MFA secret not in audit/outbox/logs
func TestMFASecretNotInAudit(t *testing.T) {
	pool, r, token := testSetup(t)

	// Enroll
	req := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("enroll failed: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	secret, _ := resp["secret"].(string)

	// Check audit logs don't contain the raw secret
	var auditValues []string
	rows, _ := pool.Query(t.Context(),
		"SELECT COALESCE(new_value::text,''), COALESCE(old_value::text,''), change_summary FROM audit_logs WHERE action LIKE 'mfa.%'")
	defer rows.Close()
	for rows.Next() {
		var nv, ov, summary string
		rows.Scan(&nv, &ov, &summary)
		auditValues = append(auditValues, nv, ov, summary)
	}

	for _, v := range auditValues {
		if strings.Contains(v, secret) {
			t.Errorf("raw TOTP secret found in audit log: %s", v)
		}
	}

	// Check outbox events
	var outboxPayloads []string
	rows2, _ := pool.Query(t.Context(),
		"SELECT payload::text FROM outbox_events WHERE event_type LIKE '%mfa.%'")
	defer rows2.Close()
	for rows2.Next() {
		var p string
		rows2.Scan(&p)
		outboxPayloads = append(outboxPayloads, p)
	}

	for _, p := range outboxPayloads {
		if strings.Contains(p, secret) {
			t.Errorf("raw TOTP secret found in outbox: %s", p)
		}
	}
}

// Test 8: Disable MFA requires recent MFA
func TestDisableMFARequiresRecentMFA(t *testing.T) {
	pool, r, token := testSetup(t)

	// Enroll + verify (full flow to activate factor)
	req := httptest.NewRequest("POST", "/api/auth/mfa/totp/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var enrollResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &enrollResp)
	secretB32, _ := enrollResp["secret"].(string)
	secret, _ := newBase32Decoder()(secretB32)

	verifyBody := fmt.Sprintf(`{"code":"%s"}`, GenerateTOTP(secret, time.Now()))
	req = httptest.NewRequest("POST", "/api/auth/mfa/totp/verify-enrollment", strings.NewReader(verifyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Get factor ID
	var factorID string
	pool.QueryRow(t.Context(),
		"SELECT id::text FROM user_mfa_factors WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev') AND status='active'").
		Scan(&factorID)

	// Reset recent_mfa_at to nil (simulate no recent MFA)
	pool.Exec(t.Context(), "UPDATE user_sessions SET recent_mfa_at=NULL WHERE user_id IN (SELECT id FROM users WHERE email='owner@test.dev')")

	// Try to disable without recent MFA — should fail
	req = httptest.NewRequest("DELETE", "/api/auth/mfa/factors/"+factorID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403 for disable without recent MFA, got %d: %s", w.Code, w.Body.String())
	}

	// Now verify MFA and try again
	chReq := httptest.NewRequest("POST", "/api/auth/mfa/challenge", nil)
	chReq.Header.Set("Authorization", "Bearer "+token)
	chW := httptest.NewRecorder()
	r.ServeHTTP(chW, chReq)
	var chResp map[string]any
	json.Unmarshal(chW.Body.Bytes(), &chResp)
	challengeID, _ := chResp["challenge_id"].(string)

	verifyChBody := fmt.Sprintf(`{"challenge_id":"%s","code":"%s"}`, challengeID, GenerateTOTP(secret, time.Now()))
	verifyChReq := httptest.NewRequest("POST", "/api/auth/mfa/verify", strings.NewReader(verifyChBody))
	verifyChReq.Header.Set("Authorization", "Bearer "+token)
	verifyChReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), verifyChReq)

	// Now disable should work
	req = httptest.NewRequest("DELETE", "/api/auth/mfa/factors/"+factorID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for disable with recent MFA, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Helpers ───

func issueTestToken(secret, userID, email, name, teamID string) (string, error) {
	// Use the iam package's token issuer
	return issueTokenHelper(secret, userID, email, name, teamID)
}

// newBase32Decoder returns a function that decodes base32 (no padding) strings.
func newBase32Decoder() func(string) ([]byte, error) {
	return func(s string) ([]byte, error) {
		// Use Go's base32 no-padding decoder
		return base32Decode(s)
	}
}

// Keep the context import
var _ = context.Background
