package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// RedactedFields are field names whose values must never appear in logs.
var RedactedFields = map[string]bool{
	"password": true, "token": true, "authorization": true, "cookie": true,
	"api_key": true, "secret": true, "jwt_secret": true, "hmac_key": true,
	"refresh_token": true, "access_token": true, "smtp_password": true,
	"proxmox_secret": true, "minio_secret_key": true, "webhook_key": true,
	"integration_key": true, "x-clarityit-integration-key": true,
}

// Slog is a structured JSON logger.
type Slog struct {
	service string
	version string
	mu      sync.Mutex
	output  io.Writer
}

var defaultLog *Slog

func InitLogger(service, version string) {
	defaultLog = &Slog{
		service: service,
		version: version,
		output:  os.Stdout,
	}
	log.SetOutput(io.Discard) // suppress default log
}

// Entry is a structured log entry.
type Entry struct {
	Timestamp     string `json:"timestamp"`
	Level         string `json:"level"`
	Service       string `json:"service"`
	Version       string `json:"version"`
	RequestID     string `json:"request_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	TeamID        string `json:"team_id,omitempty"`
	ActorType     string `json:"actor_type,omitempty"`
	EventType     string `json:"event_type,omitempty"`
	Message       string `json:"message"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	Error         string `json:"error,omitempty"`
}

func (s *Slog) log(level string, msg string, fields map[string]any) {
	e := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   s.service,
		Version:   s.version,
		Message:   msg,
	}
	for k, v := range fields {
		if RedactedFields[strings.ToLower(k)] {
			continue // skip redacted fields
		}
		switch k {
		case "request_id":
			e.RequestID, _ = v.(string)
		case "correlation_id":
			e.CorrelationID, _ = v.(string)
		case "team_id":
			e.TeamID, _ = v.(string)
		case "actor_type":
			e.ActorType, _ = v.(string)
		case "event_type":
			e.EventType, _ = v.(string)
		case "duration_ms":
			e.DurationMs, _ = v.(int64)
		case "error_code":
			e.ErrorCode, _ = v.(string)
		case "error":
			e.Error, _ = v.(string)
		}
	}
	b, _ := json.Marshal(e)
	s.mu.Lock()
	s.output.Write(b)
	s.output.Write([]byte("\n"))
	s.mu.Unlock()
}

func Info(msg string, fields map[string]any) {
	if defaultLog != nil {
		defaultLog.log("info", msg, fields)
	}
}

func Warn(msg string, fields map[string]any) {
	if defaultLog != nil {
		defaultLog.log("warn", msg, fields)
	}
}

func Error(msg string, fields map[string]any) {
	if defaultLog != nil {
		defaultLog.log("error", msg, fields)
	}
}

// RedactPayload removes redacted fields from a map before logging.
func RedactPayload(payload map[string]any) map[string]any {
	safe := make(map[string]any)
	for k, v := range payload {
		if RedactedFields[strings.ToLower(k)] {
			safe[k] = "[REDACTED]"
		} else {
			safe[k] = v
		}
	}
	return safe
}
