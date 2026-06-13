package iam

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAuthFlow tests the complete auth lifecycle.
// Requires a running PostgreSQL with migrations applied.
// Set TEST_DATABASE_URL to point to your test database.
func TestAuthFlow(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9, // 15 minutes in nanoseconds
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}

	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Connect DB: %v", err)
	}
	defer pool.Close()

	handler := NewHandler(pool, cfg)

	t.Run("Bootstrap", func(t *testing.T) {
		// Reset bootstrap lock for test (disable trigger temporarily)
		pool.Exec(ctx, "ALTER TABLE bootstrap_lock DISABLE TRIGGER trg_bootstrap_lock")
		pool.Exec(ctx, "UPDATE bootstrap_lock SET is_locked = FALSE, locked_by_user_id = NULL, locked_at = NULL WHERE id = 1")
		pool.Exec(ctx, "ALTER TABLE bootstrap_lock ENABLE TRIGGER trg_bootstrap_lock")
		pool.Exec(ctx, "TRUNCATE users, teams, team_memberships, user_platform_roles, user_sessions, refresh_tokens, audit_logs, outbox_events, password_reset_tokens, invitations, team_access_grants, integration_api_keys CASCADE")

		body, _ := json.Marshal(map[string]string{
			"name": "Test Owner", "email": "owner@test.dev", "password": "password12",
			"team_name": "Test Team",
		})
		req := httptest.NewRequest("POST", "/api/bootstrap", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Bootstrap(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Bootstrap: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["access_token"] == nil {
			t.Error("Bootstrap: no access_token")
		}
		if resp["refresh_token"] == nil {
			t.Error("Bootstrap: no refresh_token")
		}
	})

	t.Run("Bootstrap_Locked", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "Hacker", "email": "hacker@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/bootstrap", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Bootstrap(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Bootstrap locked: expected 409, got %d", w.Code)
		}
	})

	t.Run("Register", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "Test Member", "email": "member@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Register: expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Register_Duplicate_Email", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "Duplicate", "email": "member@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Duplicate register: expected 409, got %d", w.Code)
		}
	})

	t.Run("Login_Success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email": "owner@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Login: expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Login_Wrong_Password", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email": "owner@test.dev", "password": "wrongpass1",
		})
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Wrong password: expected 401, got %d", w.Code)
		}
	})

	t.Run("Login_Nonexistent_User", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email": "nobody@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Nonexistent user: expected 401, got %d", w.Code)
		}
	})

	t.Run("Refresh_Rotation", func(t *testing.T) {
		// Login first
		body, _ := json.Marshal(map[string]string{
			"email": "member@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		var loginResp map[string]any
		json.NewDecoder(w.Body).Decode(&loginResp)
		refreshToken := loginResp["refresh_token"].(string)

		// Refresh
		refreshBody, _ := json.Marshal(map[string]string{
			"refresh_token": refreshToken,
		})
		req2 := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(refreshBody))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handler.Refresh(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Refresh: expected 200, got %d: %s", w2.Code, w2.Body.String())
		}

		var refreshResp map[string]any
		json.NewDecoder(w2.Body).Decode(&refreshResp)
		if refreshResp["access_token"] == nil {
			t.Error("Refresh: no new access_token")
		}
		if refreshResp["refresh_token"] == nil {
			t.Error("Refresh: no new refresh_token")
		}
	})

	t.Run("Refresh_Reuse_Detection", func(t *testing.T) {
		// Login
		body, _ := json.Marshal(map[string]string{
			"email": "member@test.dev", "password": "password12",
		})
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		var loginResp map[string]any
		json.NewDecoder(w.Body).Decode(&loginResp)
		oldRefresh := loginResp["refresh_token"].(string)

		// Rotate
		rb, _ := json.Marshal(map[string]string{"refresh_token": oldRefresh})
		req2 := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(rb))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handler.Refresh(w2, req2)

		if w2.Code != http.StatusOK {
			t.Fatalf("First refresh: expected 200, got %d", w2.Code)
		}

		// Reuse old token
		req3 := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(rb))
		req3.Header.Set("Content-Type", "application/json")
		w3 := httptest.NewRecorder()
		handler.Refresh(w3, req3)

		if w3.Code != http.StatusUnauthorized {
			t.Errorf("Reuse detection: expected 401, got %d", w3.Code)
		}

		var errResp map[string]any
		json.NewDecoder(w3.Body).Decode(&errResp)
		if errResp["detail"] != "Token reuse detected, session revoked" {
			t.Errorf("Reuse message: got %v", errResp["detail"])
		}
	})

	t.Run("Audit_Written", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs").Scan(&count)
		if err != nil {
			t.Fatalf("Audit count: %v", err)
		}
		if count == 0 {
			t.Error("No audit events written")
		}
		t.Logf("Audit events: %d", count)
	})

	t.Run("Outbox_Written", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events").Scan(&count)
		if err != nil {
			t.Fatalf("Outbox count: %v", err)
		}
		if count == 0 {
			t.Error("No outbox events written")
		}
		t.Logf("Outbox events: %d", count)
	})

	t.Run("PII_Redacted", func(t *testing.T) {
		// Verify no raw email or IP in audit logs
		var rawEmailCount int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs WHERE change_summary LIKE '%@test.dev%'").Scan(&rawEmailCount)
		if rawEmailCount > 0 {
			t.Errorf("Raw email found in audit logs: %d rows", rawEmailCount)
		}

		// Verify ip_hmac is HMAC'd (64 char hex), not raw IP
		var ipHMAC string
		pool.QueryRow(ctx, "SELECT ip_hmac FROM audit_logs WHERE ip_hmac IS NOT NULL LIMIT 1").Scan(&ipHMAC)
		if ipHMAC != "" && len(ipHMAC) != 64 {
			t.Errorf("IP not HMAC'd: length=%d value=%s", len(ipHMAC), ipHMAC)
		}
	})

	t.Run("Password_Reset_Flow", func(t *testing.T) {
		// Request reset
		body, _ := json.Marshal(map[string]string{"email": "member@test.dev"})
		req := httptest.NewRequest("POST", "/api/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ForgotPassword(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Forgot password: expected 200, got %d", w.Code)
		}

		// Get token from DB
		var tokenHash string
		pool.QueryRow(ctx, "SELECT token_hash FROM password_reset_tokens WHERE used_at IS NULL ORDER BY created_at DESC LIMIT 1").Scan(&tokenHash)

		// We can't easily test the full reset because we don't have the raw token,
		// but we can verify the token was created
		if tokenHash == "" {
			t.Error("No reset token created")
		}
	})

	t.Run("Event_Payload_Shape", func(t *testing.T) {
		// Verify outbox events have required fields in payload
		rows, err := pool.Query(ctx, "SELECT event_type, payload FROM outbox_events")
		if err != nil {
			t.Fatalf("Query outbox: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var eventType, payload string
			rows.Scan(&eventType, &payload)
			if eventType == "" {
				t.Error("Empty event_type in outbox")
			}
			if payload == "" || payload == "null" {
				t.Errorf("Empty payload for event %s", eventType)
			}
			// Verify it's valid JSON
			var parsed any
			if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
				t.Errorf("Invalid JSON payload for %s: %v", eventType, err)
			}
		}
	})

	t.Run("Platform_Role_Separation", func(t *testing.T) {
		// Verify owner has platform_owner in user_platform_roles
		var count int
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_platform_roles upr
			JOIN platform_roles pr ON pr.id = upr.platform_role_id
			JOIN users u ON u.id = upr.user_id
			WHERE pr.name = 'platform_owner' AND u.email = 'owner@test.dev'
		`).Scan(&count)
		if count != 1 {
			t.Errorf("Owner should have platform_owner role, got %d", count)
		}

		// Verify member does NOT have platform_owner
		pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_platform_roles upr
			JOIN platform_roles pr ON pr.id = upr.platform_role_id
			JOIN users u ON u.id = upr.user_id
			WHERE pr.name = 'platform_owner' AND u.email = 'member@test.dev'
		`).Scan(&count)
		if count != 0 {
			t.Errorf("Member should NOT have platform_owner role, got %d", count)
		}
	})

	t.Run("Role_Via_Role_ID", func(t *testing.T) {
		// Verify team_memberships uses role_id, not a role string column
		var roleName string
		err := pool.QueryRow(ctx, `
			SELECT r.name FROM team_memberships tm
			JOIN roles r ON r.id = tm.role_id
			JOIN users u ON u.id = tm.user_id
			WHERE u.email = 'owner@test.dev'
		`).Scan(&roleName)
		if err != nil {
			t.Errorf("Membership role lookup failed: %v", err)
		}
		if roleName != "owner" {
			t.Errorf("Owner should have 'owner' role, got '%s'", roleName)
		}
	})
}
