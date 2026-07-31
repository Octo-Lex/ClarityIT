package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const idempotencyTestDBURL = "postgres://clarityit:clarityit@postgres:5432/clarityit?sslmode=disable"

func TestRequestFingerprintCanonicalJSONAndRestoresBody(t *testing.T) {
	first := httptest.NewRequest(
		http.MethodPost,
		"/api/example?z=2&a=1",
		bytes.NewBufferString(`{"b":2,"a":1}`),
	)
	second := httptest.NewRequest(
		http.MethodPost,
		"/api/example?a=1&z=2",
		bytes.NewBufferString("{\n  \"a\": 1,\n  \"b\": 2\n}"),
	)

	firstFingerprint, err := requestFingerprint(first)
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}
	secondFingerprint, err := requestFingerprint(second)
	if err != nil {
		t.Fatalf("second fingerprint: %v", err)
	}
	if firstFingerprint != secondFingerprint {
		t.Fatalf("equivalent requests produced different fingerprints: %s != %s", firstFingerprint, secondFingerprint)
	}

	restored, err := io.ReadAll(first.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(restored) != `{"b":2,"a":1}` {
		t.Fatalf("request body was not restored: %q", restored)
	}
}

func TestIdempotencyRejectsOversizedBodyBeforeHandler(t *testing.T) {
	var calls atomic.Int32
	handler := Idempotency(IdempotencyConfig{
		Scope:  "anonymous",
		Expiry: time.Hour,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))

	body := strings.NewReader(strings.Repeat("x", int(maxIdempotencyRequestBodyBytes)+1))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.ContentLength = -1 // Exercise the bounded-read path for a streamed body.
	req.Header.Set("Idempotency-Key", "oversized-body")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request: got %d %s, want 413", recorder.Code, recorder.Body.String())
	}
	if calls.Load() != 0 {
		t.Fatalf("handler executed for oversized request; calls=%d", calls.Load())
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode oversized response: %v", err)
	}
	if response["detail"] != "Request body exceeds 1 MiB limit" {
		t.Fatalf("unexpected oversized response: %s", recorder.Body.String())
	}
}

func TestIdempotencyReplayAndFingerprintConflict(t *testing.T) {
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, idempotencyTestDBURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(pool.Close)

	scopeID := uuid.NewString()
	key := "middleware-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DELETE FROM idempotency_keys
			WHERE scope_type = 'user' AND scope_id = $1 AND key = $2
		`, scopeID, key)
	})

	var calls atomic.Int32
	handler := Idempotency(IdempotencyConfig{
		Pool:   pool,
		Scope:  "user",
		Expiry: time.Hour,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"call":%d}`, call)
	}))

	serve := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/example?b=2&a=1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		req = req.WithContext(context.WithValue(
			req.Context(),
			ClaimsKey,
			&iam.TokenClaims{UserID: scopeID},
		))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}

	first := serve(`{"name":"example","enabled":true}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first request: %d %s", first.Code, first.Body.String())
	}

	replay := serve("{\n  \"enabled\": true,\n  \"name\": \"example\"\n}")
	if replay.Code != http.StatusCreated {
		t.Fatalf("replay: %d %s", replay.Code, replay.Body.String())
	}
	var firstBody, replayBody map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &replayBody); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if !reflect.DeepEqual(replayBody, firstBody) {
		t.Fatalf("replay body mismatch: %v != %v", replayBody, firstBody)
	}
	if calls.Load() != 1 {
		t.Fatalf("handler executed %d times; want 1", calls.Load())
	}

	conflict := serve(`{"name":"different","enabled":true}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("fingerprint conflict: %d %s", conflict.Code, conflict.Body.String())
	}
	var conflictBody map[string]string
	if err := json.Unmarshal(conflict.Body.Bytes(), &conflictBody); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	if conflictBody["detail"] != "Idempotency key reused with different request" {
		t.Fatalf("unexpected conflict response: %s", conflict.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("handler executed after conflict; calls=%d", calls.Load())
	}

	var fingerprint string
	if err := pool.QueryRow(ctx, `
		SELECT request_fingerprint
		FROM idempotency_keys
		WHERE scope_type = 'user' AND scope_id = $1 AND key = $2
	`, scopeID, key).Scan(&fingerprint); err != nil {
		t.Fatalf("read stored fingerprint: %v", err)
	}
	if fingerprint == "" {
		t.Fatal("stored request fingerprint is empty")
	}
}
