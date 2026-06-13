package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterWithinLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		HMACKey:     "test-key",
		MaxRequests: 5,
		Window:      1 * time.Minute,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/webhooks/test", nil)
		req.Header.Set("X-ClarityIT-Integration-Key", "clarity_abc123456789")
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: want 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimiterOverLimitReturns429(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		HMACKey:     "test-key",
		MaxRequests: 3,
		Window:      1 * time.Minute,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Exhaust limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/webhooks/test", nil)
		req.Header.Set("X-ClarityIT-Integration-Key", "clarity_abc123456789")
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("setup request %d: want 200, got %d", i, w.Code)
		}
	}

	// Next request should be 429
	req := httptest.NewRequest("POST", "/api/webhooks/test", nil)
	req.Header.Set("X-ClarityIT-Integration-Key", "clarity_abc123456789")
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("want 429, got %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["detail"] != "rate limit exceeded" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestRateLimiterDifferentKeysSeparate(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		HMACKey:     "test-key",
		MaxRequests: 1,
		Window:      1 * time.Minute,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// First key
	req1 := httptest.NewRequest("POST", "/api/webhooks/test", nil)
	req1.Header.Set("X-ClarityIT-Integration-Key", "clarity_abc123456789")
	req1.RemoteAddr = "10.0.0.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != 200 { t.Fatalf("key1 first: want 200, got %d", w1.Code) }

	// Same key should be limited
	req2 := httptest.NewRequest("POST", "/api/webhooks/test", nil)
	req2.Header.Set("X-ClarityIT-Integration-Key", "clarity_abc123456789")
	req2.RemoteAddr = "10.0.0.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != 429 { t.Errorf("key1 second: want 429, got %d", w2.Code) }

	// Different key AND different source should pass
	req3 := httptest.NewRequest("POST", "/api/webhooks/different-source", nil)
	req3.Header.Set("X-ClarityIT-Integration-Key", "clarity_xyz987654321")
	req3.RemoteAddr = "10.0.0.2:54321"
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Code != 200 { t.Errorf("key2 first: want 200, got %d", w3.Code) }
}

func TestRateLimiterNoRawIPInState(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		HMACKey:     "test-key",
		MaxRequests: 100,
		Window:      1 * time.Minute,
	})

	req := httptest.NewRequest("POST", "/api/webhooks/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rl.allow(req)

	// Verify no raw IP stored in entries
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k := range rl.entries {
		if k == "10.0.0.1:12345" || k == "10.0.0.1" {
			t.Errorf("raw IP found in rate limit entries: %s", k)
		}
	}
}
