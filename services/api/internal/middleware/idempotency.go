package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyConfig controls idempotency behavior for mutations.
type IdempotencyConfig struct {
	Pool   *pgxpool.Pool
	Scope  string // "user" or "system"
	Expiry time.Duration
}

const maxIdempotencyRequestBodyBytes int64 = 1 << 20

var errIdempotencyRequestBodyTooLarge = errors.New("idempotency request body too large")

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

			fingerprint, err := requestFingerprint(r)
			if err != nil {
				if errors.Is(err, errIdempotencyRequestBodyTooLarge) {
					writeErr(w, http.StatusRequestEntityTooLarge, "Request body exceeds 1 MiB limit")
					return
				}
				writeErr(w, http.StatusBadRequest, "Unable to fingerprint request")
				return
			}

			var reserved bool
			err = cfg.Pool.QueryRow(r.Context(), `
				INSERT INTO idempotency_keys (
					scope_type, scope_id, key, request_method, request_path,
					request_fingerprint, status, expires_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, 'processing', NOW() + $7)
				ON CONFLICT (scope_type, scope_id, key) DO NOTHING
				RETURNING TRUE
			`, cfg.Scope, scopeID, key, strings.ToUpper(r.Method), r.URL.Path, fingerprint, cfg.Expiry).Scan(&reserved)

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				writeErr(w, http.StatusInternalServerError, "Unable to reserve idempotency key")
				return
			}

			if !reserved {
				var (
					existingMethod      string
					existingPath        string
					existingFingerprint *string
					status              string
					responseCode        *int
					responseBody        *string
				)
				err = cfg.Pool.QueryRow(r.Context(), `
					SELECT request_method, request_path, request_fingerprint,
					       status, response_code, response_body
					FROM idempotency_keys
					WHERE scope_type = $1 AND scope_id = $2 AND key = $3
				`, cfg.Scope, scopeID, key).Scan(
					&existingMethod,
					&existingPath,
					&existingFingerprint,
					&status,
					&responseCode,
					&responseBody,
				)
				if err != nil {
					writeErr(w, http.StatusInternalServerError, "Unable to read idempotency key")
					return
				}

				if existingFingerprint == nil ||
					existingMethod != strings.ToUpper(r.Method) ||
					existingPath != r.URL.Path ||
					*existingFingerprint != fingerprint {
					writeErr(w, http.StatusConflict, "Idempotency key reused with different request")
					return
				}

				switch status {
				case "completed":
					if responseCode == nil || responseBody == nil {
						writeErr(w, http.StatusInternalServerError, "Stored idempotency response is incomplete")
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(*responseCode)
					_, _ = w.Write([]byte(*responseBody))
					return
				case "processing":
					writeErr(w, http.StatusConflict, "Request already in progress")
					return
				default:
					writeErr(w, http.StatusConflict, "Previous request with this idempotency key failed")
					return
				}
			}

			// Execute the handler only after the key and fingerprint are durably reserved.
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)

			// Store the response before returning it to the caller.
			finalStatus := "completed"
			if rec.Code >= 500 {
				finalStatus = "failed"
			}
			if _, err := cfg.Pool.Exec(r.Context(), `
				UPDATE idempotency_keys
				SET status = $1, response_code = $2, response_body = $3, completed_at = NOW()
				WHERE scope_type = $4 AND scope_id = $5 AND key = $6
			`, finalStatus, rec.Code, rec.Body.String(), cfg.Scope, scopeID, key); err != nil {
				writeErr(w, http.StatusInternalServerError, "Unable to persist idempotency response")
				return
			}

			// Forward the response
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			_, _ = w.Write(rec.Body.Bytes())
		})
	}
}

// requestFingerprint returns a stable digest over the request semantics used by
// the idempotency contract. JSON bodies and query maps are canonicalized so
// insignificant object-key order and JSON whitespace do not change the digest.
// The request body is restored before returning. Reads are capped so an
// idempotency key cannot turn an unbounded request body into unbounded memory.
func requestFingerprint(r *http.Request) (string, error) {
	var bodyBytes []byte
	if r.Body != nil {
		if r.ContentLength > maxIdempotencyRequestBodyBytes {
			return "", errIdempotencyRequestBodyTooLarge
		}

		var err error
		bodyBytes, err = io.ReadAll(io.LimitReader(r.Body, maxIdempotencyRequestBodyBytes+1))
		if err != nil {
			return "", err
		}
		if int64(len(bodyBytes)) > maxIdempotencyRequestBodyBytes {
			return "", errIdempotencyRequestBodyTooLarge
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	body := any(map[string]any{})
	if trimmed := bytes.TrimSpace(bodyBytes); len(trimmed) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()

		var decoded any
		if err := decoder.Decode(&decoded); err == nil {
			var trailing any
			if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
				body = decoded
			} else {
				body = map[string]string{
					"raw_base64": base64.StdEncoding.EncodeToString(bodyBytes),
				}
			}
		} else {
			body = map[string]string{
				"raw_base64": base64.StdEncoding.EncodeToString(bodyBytes),
			}
		}
	}

	canonical, err := json.Marshal(struct {
		Method string              `json:"method"`
		Path   string              `json:"path"`
		Query  map[string][]string `json:"query"`
		Body   any                 `json:"body"`
	}{
		Method: strings.ToUpper(r.Method),
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Body:   body,
	})
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
