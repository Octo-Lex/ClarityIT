package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogIncludesRequestID(t *testing.T) {
	var buf bytes.Buffer
	s := &Slog{service: "test", version: "0.8.0", output: &buf}
	s.log("info", "test message", map[string]any{"request_id": "req-123"})

	var entry map[string]any
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["request_id"] != "req-123" {
		t.Errorf("expected request_id=req-123, got %v", entry["request_id"])
	}
}

func TestLogRedactsAuthorizationHeader(t *testing.T) {
	var buf bytes.Buffer
	s := &Slog{service: "test", version: "0.8.0", output: &buf}
	s.log("info", "request", map[string]any{"authorization": "Bearer secret-token"})

	output := buf.String()
	if strings.Contains(output, "secret-token") {
		t.Error("authorization header value appeared in log output")
	}
}

func TestLogRedactsWebhookKey(t *testing.T) {
	var buf bytes.Buffer
	s := &Slog{service: "test", version: "0.8.0", output: &buf}
	s.log("info", "webhook", map[string]any{"x-clarityit-integration-key": "clarity_abc123"})

	output := buf.String()
	if strings.Contains(output, "clarity_abc123") {
		t.Error("integration key appeared in log output")
	}
}

func TestLogRedactsPassword(t *testing.T) {
	var buf bytes.Buffer
	s := &Slog{service: "test", version: "0.8.0", output: &buf}
	s.log("info", "login", map[string]any{"password": "supersecret"})

	output := buf.String()
	if strings.Contains(output, "supersecret") {
		t.Error("password appeared in log output")
	}
}

func TestRedactPayload(t *testing.T) {
	payload := map[string]any{
		"password":  "secret123",
		"username":  "alice",
		"api_key":   "key-abc",
		"source":    "grafana",
	}
	safe := RedactPayload(payload)
	if safe["password"] != "[REDACTED]" {
		t.Error("password not redacted")
	}
	if safe["api_key"] != "[REDACTED]" {
		t.Error("api_key not redacted")
	}
	if safe["username"] != "alice" {
		t.Error("username was incorrectly redacted")
	}
	if safe["source"] != "grafana" {
		t.Error("source was incorrectly redacted")
	}
}

func TestLogIncludesServiceAndVersion(t *testing.T) {
	var buf bytes.Buffer
	s := &Slog{service: "clarityit-api", version: "0.8.0", output: &buf}
	s.log("info", "startup", nil)

	var entry map[string]any
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["service"] != "clarityit-api" {
		t.Errorf("expected service=clarityit-api, got %v", entry["service"])
	}
	if entry["version"] != "0.8.0" {
		t.Errorf("expected version=0.8.0, got %v", entry["version"])
	}
}

func TestLogIncludesCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	s := &Slog{service: "test", version: "0.8.0", output: &buf}
	s.log("info", "outbox", map[string]any{"correlation_id": "corr-456"})

	var entry map[string]any
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["correlation_id"] != "corr-456" {
		t.Errorf("expected correlation_id=corr-456, got %v", entry["correlation_id"])
	}
}
