package team

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func invitationTestSetup(t *testing.T, env string) (*chi.Mux, string, string) {
	t.Helper()
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
		EmailMode:       "dev",
		Env:             env,
	}
	if env == "production" {
		cfg.EmailMode = "smtp"
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })

	iamH := iam.NewHandler(pool, cfg)
	teamH := NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/invitations", teamH.CreateInvitation)
	})

	// Login
	loginBody := `{"email":"owner@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login: %d %s", w.Code, w.Body.String()) }
	var loginResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	token, _ := loginResp["access_token"].(string)

	// Get team ID
	var teamID string
	pool.QueryRow(t.Context(), `SELECT id::text FROM teams LIMIT 1`).Scan(&teamID)

	return r, token, teamID
}

func TestInvitationDevReturnsDevPreview(t *testing.T) {
	r, token, teamID := invitationTestSetup(t, "development")

	// Get role ID
	pool, _ := pgxpool.New(t.Context(), "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable")
	defer pool.Close()
	var roleID string
	pool.QueryRow(t.Context(), `SELECT id::text FROM roles WHERE name='member' LIMIT 1`).Scan(&roleID)

	body := fmt.Sprintf(`{"email":"invite-dev@test.dev","role_id":"%s"}`, roleID)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/invitations", teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "inv-dev-test-v9")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["dev_preview"]; !ok {
		t.Error("dev invitation should return dev_preview")
	}
	if _, ok := resp["_dev_notice"]; !ok {
		t.Error("dev invitation should return _dev_notice")
	}
}

func TestInvitationProductionDoesNotReturnToken(t *testing.T) {
	r, token, teamID := invitationTestSetup(t, "production")

	pool, _ := pgxpool.New(t.Context(), "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable")
	defer pool.Close()
	var roleID string
	pool.QueryRow(t.Context(), `SELECT id::text FROM roles WHERE name='member' LIMIT 1`).Scan(&roleID)

	body := fmt.Sprintf(`{"email":"invite-prod@test.dev","role_id":"%s"}`, roleID)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/invitations", teamID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "inv-prod-test-v9")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	// Production must NOT return raw token
	if _, ok := resp["token"]; ok {
		t.Error("production invitation should NOT return token")
	}
	if _, ok := resp["dev_preview"]; ok {
		t.Error("production invitation should NOT return dev_preview")
	}
	// Should have message about email
	msg, _ := resp["message"].(string)
	if !strings.Contains(strings.ToLower(msg), "sent") {
		t.Errorf("production invitation should say email was sent, got: %s", msg)
	}
}
