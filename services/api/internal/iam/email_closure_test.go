package iam

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestForgotPasswordDevModeReturnsPreview(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
		EmailMode:       "dev",
		Env:             "development",
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, cfg)
	r := chi.NewRouter()
	r.Post("/api/auth/forgot-password", h.ForgotPassword)

	body := `{"email":"owner@test.dev"}`
	req := httptest.NewRequest("POST", "/api/auth/forgot-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Dev mode should return dev_preview
	if _, ok := resp["dev_preview"]; !ok {
		t.Error("dev mode should return dev_preview field")
	}
	if _, ok := resp["_dev_notice"]; !ok {
		t.Error("dev mode should return _dev_notice field")
	}
}

func TestForgotPasswordNonExistentEmailReturnsSuccess(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
		EmailMode:       "dev",
		Env:             "development",
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, cfg)
	r := chi.NewRouter()
	r.Post("/api/auth/forgot-password", h.ForgotPassword)

	body := `{"email":"nonexistent@test.dev"}`
	req := httptest.NewRequest("POST", "/api/auth/forgot-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 even for non-existent email, got %d", w.Code)
	}
	// Should NOT return dev_preview for non-existent email
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["dev_preview"]; ok {
		t.Error("should not return dev_preview for non-existent email")
	}
}

func TestForgotPasswordDisabledModeReturnsGeneric(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
		EmailMode:       "disabled",
		Env:             "production",
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, cfg)
	r := chi.NewRouter()
	r.Post("/api/auth/forgot-password", h.ForgotPassword)

	body := `{"email":"owner@test.dev"}`
	req := httptest.NewRequest("POST", "/api/auth/forgot-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should not have dev_preview in disabled mode
	if _, ok := resp["dev_preview"]; ok {
		t.Error("disabled mode should not return dev_preview")
	}
	// Should have generic success message
	msg, _ := resp["message"].(string)
	if !strings.Contains(msg, "reset link has been sent") {
		t.Errorf("expected generic success message, got: %s", msg)
	}
}

func TestResetPasswordTokenNotInAudit(t *testing.T) {
	dbURL := "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
		EmailMode:       "dev",
		Env:             "development",
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()

	h := NewHandler(pool, cfg)
	r := chi.NewRouter()
	r.Post("/api/auth/forgot-password", h.ForgotPassword)
	r.Post("/api/auth/reset-password", h.ResetPassword)

	// Request reset
	body := `{"email":"owner@test.dev"}`
	req := httptest.NewRequest("POST", "/api/auth/forgot-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("forgot-password: %d %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	devPreview, _ := resp["dev_preview"].(string)
	token := strings.TrimPrefix(devPreview, "/reset-password?token=")
	if token == "" {
		t.Fatal("no token in dev_preview")
	}

	// Check audit log doesn't contain the raw token
	var metaStr string
	pool.QueryRow(t.Context(),
		`SELECT new_value::text FROM audit_logs WHERE action='identity.password_reset.requested' ORDER BY created_at DESC LIMIT 1`).
		Scan(&metaStr)

	if strings.Contains(metaStr, token) {
		t.Error("raw reset token appeared in audit log")
	}

	// Reset password using the token
	resetBody := `{"token":"` + token + `","password":"newpassword123"}`
	req = httptest.NewRequest("POST", "/api/auth/reset-password", strings.NewReader(resetBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("reset-password: %d %s", w.Code, w.Body.String())
	}
}

// Helper to avoid unused import warning
var _ = bytes.NewReader
