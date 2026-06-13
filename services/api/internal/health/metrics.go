package health

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

// Metrics tracks platform counters for Prometheus exposition.
type Metrics struct {
	HTTPRequestsTotal         atomic.Int64
	HTTPRequestDurationMs     atomic.Int64 // sum, for average calculation
	OutboxPendingCount        atomic.Int64
	OutboxDeadLetterCount     atomic.Int64
	OutboxPublishTotal        atomic.Int64
	OutboxPublishFailedTotal  atomic.Int64
	ContextIngestTotal        atomic.Int64
	ContextIngestFailedTotal  atomic.Int64
	WebhookReceivedTotal      atomic.Int64
	WebhookRejectedTotal      atomic.Int64
	WebhookRateLimitedTotal   atomic.Int64
	AgentToolBlockedTotal     atomic.Int64
	AgentToolDeniedTotal      atomic.Int64
	AgentToolSucceededTotal   atomic.Int64
	WSConnectionsActive       atomic.Int64
}

// Global metrics instance
var M = &Metrics{}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var sb strings.Builder

	writeCounter := func(name, help string, val int64) {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, help))
		sb.WriteString(fmt.Sprintf("# TYPE %s counter\n", name))
		sb.WriteString(fmt.Sprintf("%s %d\n", name, val))
	}

	writeGauge := func(name, help string, val int64) {
		sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, help))
		sb.WriteString(fmt.Sprintf("# TYPE %s gauge\n", name))
		sb.WriteString(fmt.Sprintf("%s %d\n", name, val))
	}

	writeCounter("clarity_http_requests_total", "Total HTTP requests processed", M.HTTPRequestsTotal.Load())
	writeCounter("clarity_http_request_duration_ms_total", "Total request duration in milliseconds", M.HTTPRequestDurationMs.Load())
	writeGauge("clarity_outbox_pending_count", "Current outbox events pending processing", M.OutboxPendingCount.Load())
	writeGauge("clarity_outbox_dead_letter_count", "Current dead-letter events", M.OutboxDeadLetterCount.Load())
	writeCounter("clarity_outbox_publish_total", "Total outbox events published to NATS", M.OutboxPublishTotal.Load())
	writeCounter("clarity_outbox_publish_failed_total", "Total outbox publish failures", M.OutboxPublishFailedTotal.Load())
	writeCounter("clarity_context_ingest_total", "Total context events ingested", M.ContextIngestTotal.Load())
	writeCounter("clarity_context_ingest_failed_total", "Total context ingest failures", M.ContextIngestFailedTotal.Load())
	writeCounter("clarity_webhook_received_total", "Total webhooks received", M.WebhookReceivedTotal.Load())
	writeCounter("clarity_webhook_rejected_total", "Total webhooks rejected", M.WebhookRejectedTotal.Load())
	writeCounter("clarity_webhook_rate_limited_total", "Total webhooks rate limited", M.WebhookRateLimitedTotal.Load())
	writeCounter("clarity_agent_tool_blocked_total", "Total agent tool executions blocked", M.AgentToolBlockedTotal.Load())
	writeCounter("clarity_agent_tool_denied_total", "Total agent tool executions denied", M.AgentToolDeniedTotal.Load())
	writeCounter("clarity_agent_tool_succeeded_total", "Total agent tool executions succeeded", M.AgentToolSucceededTotal.Load())
	writeGauge("clarity_ws_connections_active", "Active WebSocket connections", M.WSConnectionsActive.Load())

	w.Write([]byte(sb.String()))
}
