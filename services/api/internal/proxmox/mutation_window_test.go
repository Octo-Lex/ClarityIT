package proxmox

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestMutationWindow_OpenWithReason verifies a window can be opened with valid reason and duration.
func TestMutationWindow_OpenWithReason(t *testing.T) {
	env := setupMutationWindowTest(t)

	body := `{"reason":"Track 2 validation","duration_minutes":15}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "open" {
		t.Errorf("expected status 'open', got %v", resp["status"])
	}
	if resp["reason"] != "Track 2 validation" {
		t.Errorf("expected reason 'Track 2 validation', got %v", resp["reason"])
	}
	// Cleanup
	env.pool.Exec(env.ctx, `UPDATE proxmox_mutation_windows SET status='closed' WHERE status='open'`)
}

// TestMutationWindow_RejectMissingReason verifies window creation fails without a reason.
func TestMutationWindow_RejectMissingReason(t *testing.T) {
	env := setupMutationWindowTest(t)

	body := `{"duration_minutes":15}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestMutationWindow_RejectDurationAboveMax verifies duration above 60 is rejected.
func TestMutationWindow_RejectDurationAboveMax(t *testing.T) {
	env := setupMutationWindowTest(t)

	body := `{"reason":"too long","duration_minutes":61}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestMutationWindow_RejectDurationBelowMin verifies duration below 1 is rejected.
func TestMutationWindow_RejectDurationBelowMin(t *testing.T) {
	env := setupMutationWindowTest(t)

	body := `{"reason":"too short","duration_minutes":0}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestMutationWindow_OnlyOneActiveWindow verifies only one active window at a time.
func TestMutationWindow_OnlyOneActiveWindow(t *testing.T) {
	env := setupMutationWindowTest(t)

	// Open first window
	body := `{"reason":"first window","duration_minutes":10}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201 for first window, got %d: %s", w.Code, w.Body.String())
	}

	// Try second window
	body2 := `{"reason":"second window","duration_minutes":5}`
	req2 := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+env.token)
	w2 := httptest.NewRecorder()
	env.r.ServeHTTP(w2, req2)

	if w2.Code != 409 {
		t.Fatalf("expected 409 for duplicate window, got %d: %s", w2.Code, w2.Body.String())
	}

	// Cleanup
	env.pool.Exec(env.ctx, `UPDATE proxmox_mutation_windows SET status='closed' WHERE status='open'`)
}

// TestMutationWindow_GetActiveWindow verifies GET returns the active window.
func TestMutationWindow_GetActiveWindow(t *testing.T) {
	env := setupMutationWindowTest(t)

	// Open a window
	body := `{"reason":"get test","duration_minutes":10}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var openResp map[string]any
	json.NewDecoder(w.Body).Decode(&openResp)

	// GET active window
	getReq := httptest.NewRequest("GET", "/api/admin/proxmox/mutation-window", nil)
	getReq.Header.Set("Authorization", "Bearer "+env.token)
	getW := httptest.NewRecorder()
	env.r.ServeHTTP(getW, getReq)

	if getW.Code != 200 {
		t.Fatalf("expected 200, got %d", getW.Code)
	}

	var resp map[string]any
	json.NewDecoder(getW.Body).Decode(&resp)
	if resp["active"] != true {
		t.Errorf("expected active=true, got %v", resp["active"])
	}

	// Cleanup
	env.pool.Exec(env.ctx, `UPDATE proxmox_mutation_windows SET status='closed' WHERE status='open'`)
}

// TestMutationWindow_ManualClose verifies manual close blocks mutation.
func TestMutationWindow_ManualClose(t *testing.T) {
	env := setupMutationWindowTest(t)

	// Open
	body := `{"reason":"close test","duration_minutes":10}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	windowID := resp["id"].(string)

	// Close
	closeBody := `{"reason":"done with mutation"}`
	closeReq := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/proxmox/mutation-window/%s/close", windowID), strings.NewReader(closeBody))
	closeReq.Header.Set("Content-Type", "application/json")
	closeReq.Header.Set("Authorization", "Bearer "+env.token)
	closeW := httptest.NewRecorder()
	env.r.ServeHTTP(closeW, closeReq)

	if closeW.Code != 200 {
		t.Fatalf("expected 200 on close, got %d: %s", closeW.Code, closeW.Body.String())
	}

	// Verify no active window
	if HasActiveMutationWindow(env.ctx, env.pool) {
		t.Error("expected no active window after close")
	}
}

// TestMutationWindow_ExpiredBlocksMutation verifies expired window blocks mutation.
func TestMutationWindow_ExpiredBlocksMutation(t *testing.T) {
	env := setupMutationWindowTest(t)

	userID := getTestUserID(t)
	_, err := env.pool.Exec(env.ctx, `
		INSERT INTO proxmox_mutation_windows (id, status, reason, opened_by, opened_at, expires_at)
		VALUES ($1, 'open', 'expired test', $2, now(), now() - interval '1 minute')
	`, uuid.New(), userID)
	if err != nil {
		t.Fatal(err)
	}

	// Run expire
	env.handler.expireStaleWindows(env.ctx)

	// Verify no active window
	if HasActiveMutationWindow(env.ctx, env.pool) {
		t.Error("expected no active window after expiry")
	}
}

// TestMutationWindow_AutoExpiry verifies auto-expiry marks status expired.
func TestMutationWindow_AutoExpiry(t *testing.T) {
	env := setupMutationWindowTest(t)

	userID := getTestUserID(t)
	windowID := uuid.New()

	_, err := env.pool.Exec(env.ctx, `
		INSERT INTO proxmox_mutation_windows (id, status, reason, opened_by, opened_at, expires_at)
		VALUES ($1, 'open', 'auto-expire test', $2, now() - interval '5 minutes', now() - interval '1 minute')
	`, windowID, userID)
	if err != nil {
		t.Fatal(err)
	}

	env.handler.expireStaleWindows(env.ctx)

	var status string
	env.pool.QueryRow(env.ctx,
		`SELECT status FROM proxmox_mutation_windows WHERE id=$1`, windowID).Scan(&status)

	if status != "expired" {
		t.Errorf("expected status 'expired', got '%s'", status)
	}
}

// TestMutationWindow_AuditOnOpen verifies audit is written on window open.
func TestMutationWindow_AuditOnOpen(t *testing.T) {
	env := setupMutationWindowTest(t)

	body := `{"reason":"audit test","duration_minutes":5}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	windowID := resp["id"].(string)

	var auditCount int
	env.pool.QueryRow(env.ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE action='proxmox.mutation_window.opened' AND entity_id::text=$1`,
		windowID).Scan(&auditCount)

	if auditCount == 0 {
		t.Error("expected audit entry for window open")
	}

	// Cleanup
	env.pool.Exec(env.ctx, `UPDATE proxmox_mutation_windows SET status='closed' WHERE status='open'`)
}

// TestMutationWindow_AuditOnClose verifies audit is written on window close.
func TestMutationWindow_AuditOnClose(t *testing.T) {
	env := setupMutationWindowTest(t)

	// Open
	body := `{"reason":"audit close test","duration_minutes":5}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	windowID := resp["id"].(string)

	// Close
	closeBody := `{"reason":"done"}`
	closeReq := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/proxmox/mutation-window/%s/close", windowID), strings.NewReader(closeBody))
	closeReq.Header.Set("Content-Type", "application/json")
	closeReq.Header.Set("Authorization", "Bearer "+env.token)
	closeW := httptest.NewRecorder()
	env.r.ServeHTTP(closeW, closeReq)

	var auditCount int
	env.pool.QueryRow(env.ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE action='proxmox.mutation_window.closed' AND entity_id::text=$1`,
		windowID).Scan(&auditCount)

	if auditCount == 0 {
		t.Error("expected audit entry for window close")
	}
}

// TestMutationWindow_NoSecretsInState verifies no secrets in window data.
func TestMutationWindow_NoSecretsInState(t *testing.T) {
	env := setupMutationWindowTest(t)

	body := `{"reason":"no-leak-check","duration_minutes":5}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	respJSON, _ := json.Marshal(resp)
	respStr := string(respJSON)

	for _, secret := range []string{"token", "secret", "password", "credential", "api_key"} {
		if strings.Contains(strings.ToLower(respStr), secret) {
			t.Errorf("response contains '%s': %s", secret, respStr)
		}
	}

	// Cleanup
	if id, ok := resp["id"].(string); ok {
		env.pool.Exec(env.ctx, `UPDATE proxmox_mutation_windows SET status='closed' WHERE id=$1`, id)
	}
}

// TestMutationWindow_ReasonTooLong verifies reason length is bounded.
func TestMutationWindow_ReasonTooLong(t *testing.T) {
	env := setupMutationWindowTest(t)

	longReason := strings.Repeat("a", 501)
	body := fmt.Sprintf(`{"reason":"%s","duration_minutes":5}`, longReason)
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400 for long reason, got %d", w.Code)
	}
}

// TestMutationWindow_CloseNonExistent verifies closing a non-existent window returns 404.
func TestMutationWindow_CloseNonExistent(t *testing.T) {
	env := setupMutationWindowTest(t)

	fakeID := uuid.New().String()
	body := `{"reason":"nope"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/proxmox/mutation-window/%s/close", fakeID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// TestMutationWindow_CloseAlreadyClosed verifies closing an already-closed window returns 409.
func TestMutationWindow_CloseAlreadyClosed(t *testing.T) {
	env := setupMutationWindowTest(t)

	// Open
	body := `{"reason":"double close","duration_minutes":5}`
	req := httptest.NewRequest("POST", "/api/admin/proxmox/mutation-window", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.r.ServeHTTP(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	windowID := resp["id"].(string)

	// Close once
	closeBody := `{"reason":"first close"}`
	closeReq := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/proxmox/mutation-window/%s/close", windowID), strings.NewReader(closeBody))
	closeReq.Header.Set("Content-Type", "application/json")
	closeReq.Header.Set("Authorization", "Bearer "+env.token)
	closeW := httptest.NewRecorder()
	env.r.ServeHTTP(closeW, closeReq)

	// Close again
	closeReq2 := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/proxmox/mutation-window/%s/close", windowID), strings.NewReader(closeBody))
	closeReq2.Header.Set("Content-Type", "application/json")
	closeReq2.Header.Set("Authorization", "Bearer "+env.token)
	closeW2 := httptest.NewRecorder()
	env.r.ServeHTTP(closeW2, closeReq2)

	if closeW2.Code != 409 {
		t.Fatalf("expected 409 for double close, got %d", closeW2.Code)
	}
}
