package proxmox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbURL = "postgres://clarityit:clarityit@192.168.3.20:5432/clarityit?sslmode=disable"

type testClient struct{}

func (f *testClient) ListNodes(_ context.Context) ([]ProxmoxNode, error) {
	return []ProxmoxNode{{Node: "pve1", Status: "online", CPU: 0.3, Mem: 8e9, MaxMem: 32e9}}, nil
}
func (f *testClient) ListVMs(_ context.Context, _ string) ([]ProxmoxVM, error) {
	return []ProxmoxVM{{VMID: 100, Name: "test-vm", Status: "running", CPU: 0.1, Mem: 2e9, MaxMem: 4e9}}, nil
}

func testSetup(t *testing.T) (*chi.Mux, *pgxpool.Pool) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test-secret", HMACKey: "test-hmac-key", AccessTokenTTL: 15 * 60 * 1e9, RefreshTokenTTL: 7 * 24 * 3600 * 1e9}
	pool, _ := pgxpool.New(t.Context(), dbURL)
	t.Cleanup(func() { pool.Close() })

	h := NewHandler(pool, &testClient{})
	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iam.NewHandler(pool, cfg).Login)
	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Get("/integrations/proxmox/status", h.Status)
		r.Post("/integrations/proxmox/sync", h.Sync)
	})
	return r, pool
}

func doReq(r *chi.Mux, method, path, tok string, body any) *httptest.ResponseRecorder {
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
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func login(t *testing.T, r *chi.Mux) (string, string) {
	t.Helper()
	w := doReq(r, "POST", "/api/auth/login", "", map[string]string{"email": "owner@test.dev", "password": "password12"})
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	token, _ := resp["access_token"].(string)
	var tid string
	pool, _ := pgxpool.New(t.Context(), dbURL)
	defer pool.Close()
	pool.QueryRow(t.Context(), `SELECT t.id::text FROM teams t JOIN team_memberships tm ON t.id=tm.team_id JOIN users u ON tm.user_id=u.id WHERE u.email=$1 LIMIT 1`, "owner@test.dev").Scan(&tid)
	return token, tid
}

func TestProxmoxStatus(t *testing.T) {
	r, _ := testSetup(t)
	token, tid := login(t, r)
	w := doReq(r, "GET", fmt.Sprintf("/api/teams/%s/integrations/proxmox/status", tid), token, nil)
	if w.Code != 200 {
		t.Fatalf("status: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["connected"] != true {
		t.Error("expected connected=true")
	}
}

func TestProxmoxSync(t *testing.T) {
	r, pool := testSetup(t)
	token, tid := login(t, r)
	w := doReq(r, "POST", fmt.Sprintf("/api/teams/%s/integrations/proxmox/sync", tid), token, nil)
	if w.Code != 200 {
		t.Fatalf("sync: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["synced"] == nil {
		t.Error("expected synced count")
	}
	if resp["nodes"] == nil {
		t.Error("expected nodes count")
	}

	// Verify assets in DB
	var cnt int
	pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM assets`).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected assets after sync")
	}
}

func TestStatusRequiresAuth(t *testing.T) {
	r, _ := testSetup(t)
	w := doReq(r, "GET", "/api/teams/00000000-0000-0000-0000-000000000000/integrations/proxmox/status", "", nil)
	if w.Code != 401 {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestSyncRequiresAuth(t *testing.T) {
	r, _ := testSetup(t)
	w := doReq(r, "POST", "/api/teams/00000000-0000-0000-0000-000000000000/integrations/proxmox/sync", "", nil)
	if w.Code != 401 {
		t.Errorf("want 401, got %d", w.Code)
	}
}
