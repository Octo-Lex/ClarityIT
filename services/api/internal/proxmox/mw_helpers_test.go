package proxmox

import (
	"context"
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


type mwTestEnv struct {
	r       *chi.Mux
	pool    *pgxpool.Pool
	token   string
	handler *MutationWindowHandler
	ctx     context.Context
	cfg     *config.Config
}

func setupMutationWindowTest(t *testing.T) *mwTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:            "test-secret",
		HMACKey:              "test-hmac-key",
		AccessTokenTTL:       15 * 60 * 1e9,
		RefreshTokenTTL:      7 * 24 * 3600 * 1e9,
		ProxmoxMutationEnabled: true,
	}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })

	// Clean any existing windows from prior test runs
	pool.Exec(t.Context(), `UPDATE proxmox_mutation_windows SET status='closed' WHERE status='open'`)

	handler := NewMutationWindowHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Route("/api/admin/proxmox/mutation-window", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/", handler.OpenWindow)
		r.Get("/", handler.GetActiveWindow)
		r.Post("/{windowId}/close", handler.CloseWindow)
	})

	token := mwLogin(t, r)

	return &mwTestEnv{r: r, pool: pool, token: token, handler: handler, ctx: t.Context(), cfg: cfg}
}

func mwLogin(t *testing.T, r *chi.Mux) string {
	t.Helper()
	body := `{"email":"owner@test.dev","password":"password12"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	return resp["access_token"].(string)
}

func getTestUserID(t *testing.T) string {
	t.Helper()
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()
	var id string
	pool.QueryRow(t.Context(),
		"SELECT id::text FROM users WHERE email='owner@test.dev'").Scan(&id)
	return id
}
