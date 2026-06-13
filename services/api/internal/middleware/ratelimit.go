package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RateLimiterConfig configures in-memory rate limiting.
type RateLimiterConfig struct {
	HMACKey       string
	MaxRequests   int
	Window        time.Duration
	Pool          *pgxpool.Pool
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

// RateLimiter provides per-key-prefix, per-source, per-remote-ip-HMAC rate limiting.
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateEntry
	config   RateLimiterConfig
}

func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateEntry),
		config:  cfg,
	}
	// Periodic cleanup
	go func() {
		for range time.Tick(cfg.Window) {
			rl.mu.Lock()
			now := time.Now()
			for k, v := range rl.entries {
				if now.After(v.resetAt) {
					delete(rl.entries, k)
				}
			}
		rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			w.Write([]byte(`{"detail":"rate limit exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(r *http.Request) bool {
	now := time.Now()
	keys := rl.keys(r)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for _, k := range keys {
		entry, ok := rl.entries[k]
		if !ok || now.After(entry.resetAt) {
			rl.entries[k] = &rateEntry{count: 1, resetAt: now.Add(rl.config.Window)}
			continue
		}
		if entry.count >= rl.config.MaxRequests {
			return false
		}
		entry.count++
	}
	return true
}

func (rl *RateLimiter) keys(r *http.Request) []string {
	keys := make([]string, 0, 3)

	// Per key prefix
	if rawKey := r.Header.Get("X-ClarityIT-Integration-Key"); len(rawKey) >= 12 {
		keys = append(keys, "prefix:"+rawKey[:12])
	}

	// Per source (URL param)
	source := r.URL.Path // simplified; chi extracts params later
	keys = append(keys, "source:"+source)

	// Per remote IP HMAC (never store raw IP)
	if r.RemoteAddr != "" {
		mac := hmac.New(sha256.New, []byte(rl.config.HMACKey))
		mac.Write([]byte(r.RemoteAddr))
		ipHash := hex.EncodeToString(mac.Sum(nil))[:16]
		keys = append(keys, "ip:"+ipHash)
	}

	return keys
}

// WriteRateLimitAudit persists a rate-limit security event.
func WriteRateLimitAudit(ctx context.Context, pool *pgxpool.Pool, hmacKey, remoteAddr, path string) {
	ipHash := ""
	if remoteAddr != "" {
		mac := hmac.New(sha256.New, []byte(hmacKey))
		mac.Write([]byte(remoteAddr))
		ipHash = hex.EncodeToString(mac.Sum(nil))[:16]
	}
	// Best-effort audit — don't fail the request if audit fails
	pool.Exec(ctx, `INSERT INTO audit_logs (id, action, entity_type, new_value) VALUES (gen_random_uuid(), $1, 'security', $2)`,
		"security.rate_limited",
		fmt.Sprintf(`{"ip_hash":"%s","path":"%s"}`, ipHash, path),
	)
}
