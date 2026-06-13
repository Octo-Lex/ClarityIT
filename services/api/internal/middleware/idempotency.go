package middleware

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyConfig controls idempotency behavior for mutations.
type IdempotencyConfig struct {
	Pool   *pgxpool.Pool
	Scope  string // "user" or "system"
	Expiry time.Duration
}

// Idempotency reserves and validates idempotency keys for mutating requests.
// Returns a chi-compatible middleware function.
// Usage: pass an Idempotency-Key header. If the key was already used, returns the cached response.
// If no header is provided, the request passes through without idempotency tracking.
func Idempotency(cfg IdempotencyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				// No key = pass through without tracking
				next.ServeHTTP(w, r)
				return
			}

			claims, _ := r.Context().Value(ClaimsKey).(*iam.TokenClaims)
			scopeID := "anonymous"
			if claims != nil {
				scopeID = claims.UserID
			}

			// Check if key already used
			var status string
			var responseCode int
			var responseBody string
			err := cfg.Pool.QueryRow(r.Context(), `
				SELECT status, response_code, response_body FROM idempotency_keys
				WHERE scope_type = $1 AND scope_id = $2 AND key = $3
			`, cfg.Scope, scopeID, key).Scan(&status, &responseCode, &responseBody)

			if err == nil {
				// Key exists
				if status == "completed" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(responseCode)
					w.Write([]byte(responseBody))
					return
				}
				if status == "processing" {
					writeErr(w, http.StatusConflict, "Request already in progress")
					return
				}
			}

			// Reserve the key
			_, err = cfg.Pool.Exec(r.Context(), `
				INSERT INTO idempotency_keys (scope_type, scope_id, key, request_method, request_path, status, expires_at)
				VALUES ($1, $2, $3, $4, $5, 'processing', NOW() + $6)
			`, cfg.Scope, scopeID, key, r.Method, r.URL.Path, cfg.Expiry)

			if err != nil {
				// Unique violation — another request beat us
				writeErr(w, http.StatusConflict, "Duplicate request")
				return
			}

			// Execute the handler
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)

			// Store the response
			finalStatus := "completed"
			if rec.Code >= 500 {
				finalStatus = "failed"
			}
			cfg.Pool.Exec(r.Context(), `
				UPDATE idempotency_keys
				SET status = $1, response_code = $2, response_body = $3, completed_at = NOW()
				WHERE scope_type = $4 AND scope_id = $5 AND key = $6
			`, finalStatus, rec.Code, rec.Body.String(), cfg.Scope, scopeID, key)

			// Forward the response
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			w.Write(rec.Body.Bytes())
		})
	}
}
