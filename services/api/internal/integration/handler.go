package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

func claims(r *http.Request) (*iam.TokenClaims, bool) { return iam.GetClaims(r) }

// generateKey creates a random API key with prefix
func generateKey() (raw, prefix, hash string) {
	b := make([]byte, 32)
	rand.Read(b)
	raw = "clarity_" + hex.EncodeToString(b)
	prefix = raw[:12]
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return
}

// ResolveIntegrationKey looks up an integration key by hash, returns (teamID, keyID, allowedSources, allowedScopes, error)
func ResolveIntegrationKey(ctx context.Context, pool *pgxpool.Pool, rawKey string) (teamID, keyID string, sources, scopes []string, err error) {
	h := sha256.Sum256([]byte(rawKey))
	hash := hex.EncodeToString(h[:])
	err = pool.QueryRow(ctx, `SELECT id::text, team_id::text, allowed_sources, allowed_scopes FROM integration_api_keys WHERE key_hash=$1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())`, hash).Scan(&keyID, &teamID, &sources, &scopes)
	return
}

// ─── Routes ───

func (h *Handler) KeyRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.CreateKey)
	r.Get("/", h.ListKeys)
	r.Delete("/{keyId}", h.RevokeKey)
	return r
}

// ─── Create Key ───

func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	cl, ok := claims(r)
	if !ok { writeErr(w, 401, "unauthorized"); return }
	actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID)

	var req struct {
		Name           string     `json:"name"`
		AllowedSources []string   `json:"allowed_sources"`
		AllowedScopes  []string   `json:"allowed_scopes"`
		ExpiresAt      *time.Time `json:"expires_at"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" || len(req.AllowedSources) == 0 || len(req.AllowedScopes) == 0 {
		writeErr(w, 400, "name, allowed_sources, and allowed_scopes are required")
		return
	}

	raw, prefix, hash := generateKey()
	var id string

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO integration_api_keys (team_id, name, key_hash, key_prefix, allowed_sources, allowed_scopes, expires_at, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id::text`, tid, req.Name, hash, prefix, req.AllowedSources, req.AllowedScopes, req.ExpiresAt, actorID).Scan(&id); err != nil { return err }
		meta, _ := json.Marshal(map[string]any{"name": req.Name, "prefix": prefix})
		eid, _ := uuid.Parse(id)
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "integration.key.created", EntityType: "integration_api_key", EntityID: eid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.integration.key.created", AggregateType: "integration_api_key", AggregateID: id, Payload: meta})
	})
	if err != nil { writeErr(w, 500, "Failed to create key"); return }
	writeJSON(w, 201, map[string]any{"id": id, "key": raw, "prefix": prefix, "name": req.Name})
}

// ─── List Keys ───

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, _ := h.pool.Query(ctx, `SELECT id::text, name, key_prefix, allowed_sources, allowed_scopes, expires_at, revoked_at, created_at FROM integration_api_keys WHERE team_id=$1 ORDER BY created_at DESC`, chi.URLParam(r, "teamId"))
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, n, p string; var src, sc []string; var exp, rev *time.Time; var c time.Time
		rows.Scan(&id, &n, &p, &src, &sc, &exp, &rev, &c)
		out = append(out, map[string]any{"id": id, "name": n, "prefix": p, "allowed_sources": src, "allowed_scopes": sc, "expires_at": exp, "revoked_at": rev, "created_at": c})
	}
	if out == nil { out = []map[string]any{} }
	writeJSON(w, 200, out)
}

// ─── Revoke Key ───

func (h *Handler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId"); keyID := chi.URLParam(r, "keyId")
	cl, _ := claims(r); actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID); kid, _ := uuid.Parse(keyID)

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE integration_api_keys SET revoked_at=now() WHERE id=$1 AND team_id=$2 AND revoked_at IS NULL`, kid, tid)
		if err != nil { return err }
		if tag.RowsAffected() == 0 { return fmt.Errorf("not found") }
		meta, _ := json.Marshal(map[string]any{"key_id": keyID})
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "integration.key.revoked", EntityType: "integration_api_key", EntityID: kid, NewValue: meta})
		return outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.integration.key.revoked", AggregateType: "integration_api_key", AggregateID: keyID, Payload: meta})
	})
	if err != nil { writeErr(w, 404, "Key not found"); return }
	writeJSON(w, 200, map[string]any{"message": "revoked"})
}

// ─── Webhook Receiver ───

func (h *Handler) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	source := chi.URLParam(r, "source")

	// Auth: X-ClarityIT-Integration-Key header
	rawKey := r.Header.Get("X-ClarityIT-Integration-Key")
	if rawKey == "" {
		writeErr(w, 401, "Missing integration key"); return
	}

	teamID, _, sources, scopes, err := ResolveIntegrationKey(ctx, h.pool, rawKey)
	if err != nil { writeErr(w, 401, "Invalid integration key"); return }

	// Verify source
	sourceAllowed := false
	for _, s := range sources {
		if s == source || s == "*" { sourceAllowed = true; break }
	}
	if !sourceAllowed { writeErr(w, 403, "Source not allowed"); return }

	// Verify scope
	scopeOK := false
	for _, s := range scopes {
		if s == "webhooks:ingest" || s == "alerts:create" || s == "*" { scopeOK = true; break }
	}
	if !scopeOK { writeErr(w, 403, "Missing required scope"); return }

	// Validate payload size
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024) // 256KB max
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErr(w, 400, "Invalid JSON payload"); return
	}

	// Sanitize: extract only known fields
	alertName, _ := payload["name"].(string)
	alertSeverity, _ := payload["severity"].(string)
	if alertSeverity == "" { alertSeverity = "warning" }
	alertSourceID, _ := payload["source_id"].(string)
	alertFingerprint := fmt.Sprintf("%s:%s:%s", source, alertSourceID, alertName)

	// Hash the full payload for audit (never store raw payload)
	payloadBytes, _ := json.Marshal(payload)
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(payloadBytes))

	tid, _ := uuid.Parse(teamID)

	// Create alert object
	var objectID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Create object
		if err := tx.QueryRow(ctx, `INSERT INTO objects (team_id, object_type, title, status, priority) VALUES ($1,'alert',$2,'active','medium') RETURNING id::text`, tid, alertName).Scan(&objectID); err != nil { return err }

		// Create alert row (upsert on fingerprint — deduplicate)
		oid, _ := uuid.Parse(objectID)
		if _, err := tx.Exec(ctx, `INSERT INTO alerts (object_id, source, source_alert_id, severity, fingerprint, first_seen_at) VALUES ($1,$2,$3,$4,$5,now()) ON CONFLICT (source, fingerprint) DO UPDATE SET last_seen_at=now(), severity=EXCLUDED.severity`, oid, source, alertSourceID, alertSeverity, alertFingerprint); err != nil { return err }

		// Audit (sanitized — no raw payload)
		meta, _ := json.Marshal(map[string]any{"source": source, "object_id": objectID, "severity": alertSeverity, "payload_hash": payloadHash})
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, Action: "integration.webhook.received", EntityType: "alert", EntityID: oid, NewValue: meta})
		_ = outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.integration.webhook.received", AggregateType: "alert", AggregateID: objectID, Payload: meta})
		_ = outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.alert.triggered", AggregateType: "alert", AggregateID: objectID, Payload: meta})
		return nil
	})
	if err != nil { writeErr(w, 500, fmt.Sprintf("Failed to process webhook: %v", err)); return }
	writeJSON(w, 201, map[string]any{"id": objectID, "status": "created"})
}

func writeJSON(w http.ResponseWriter, s int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(s); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, s int, m string) { writeJSON(w, s, map[string]string{"detail": m}) }
