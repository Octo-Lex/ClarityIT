package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MutationWindowHandler manages Proxmox mutation change-windows.
type MutationWindowHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewMutationWindowHandler(pool *pgxpool.Pool, cfg *config.Config) *MutationWindowHandler {
	return &MutationWindowHandler{pool: pool, cfg: cfg}
}

const (
	maxWindowMinutes = 60
	minWindowMinutes = 1
)

// Routes returns the chi router for mutation window endpoints.
// These are platform-admin endpoints mounted under /api/admin.
func (h *MutationWindowHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.OpenWindow)
	r.Get("/", h.GetActiveWindow)
	r.Post("/{windowId}/close", h.CloseWindow)
	return r
}

// ─── Open Mutation Window ───

func (h *MutationWindowHandler) OpenWindow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	var req struct {
		Reason          string `json:"reason"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid request body")
		return
	}

	if req.Reason == "" {
		writeErr(w, 400, "reason is required")
		return
	}
	if len(req.Reason) > 500 {
		writeErr(w, 400, "reason must be at most 500 characters")
		return
	}
	if req.DurationMinutes < minWindowMinutes {
		writeErr(w, 400, fmt.Sprintf("duration_minutes must be at least %d", minWindowMinutes))
		return
	}
	if req.DurationMinutes > maxWindowMinutes {
		writeErr(w, 400, fmt.Sprintf("duration_minutes must be at most %d", maxWindowMinutes))
		return
	}

	// Check for existing active window
	var existingID string
	err := h.pool.QueryRow(ctx,
		`SELECT id::text FROM proxmox_mutation_windows WHERE status='open' LIMIT 1`).Scan(&existingID)
	if err == nil {
		writeErr(w, 409, fmt.Sprintf("An active mutation window already exists (id=%s). Close it first.", existingID))
		return
	}

	// Create window
	windowID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(time.Duration(req.DurationMinutes) * time.Minute)

	_, err = h.pool.Exec(ctx, `
		INSERT INTO proxmox_mutation_windows (id, status, reason, opened_by, opened_at, expires_at)
		VALUES ($1, 'open', $2, $3, $4, $5)
	`, windowID, req.Reason, userID, now, expiresAt)
	if err != nil {
		writeErr(w, 500, "Failed to create mutation window")
		return
	}

	// Audit + outbox via transaction
	auditMeta, _ := json.Marshal(map[string]any{
		"window_id":        windowID.String(),
		"reason":           req.Reason,
		"duration_minutes": req.DurationMinutes,
		"expires_at":       expiresAt.Format(time.RFC3339),
	})
	platformStr := "platform"
	_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		_ = audit.Write(ctx, tx, audit.Event{
			ActorID:    userID,
			Action:     "proxmox.mutation_window.opened",
			EntityType: "proxmox_mutation_window",
			EntityID:   windowID,
			NewValue:   auditMeta,
			Summary:    fmt.Sprintf("Mutation window opened: %s (%d min)", req.Reason, req.DurationMinutes),
		})
		_ = outbox.Write(ctx, tx, &platformStr, outbox.Event{
			EventType:     "clarity.v1.proxmox.mutation_window.opened",
			AggregateType: "proxmox_mutation_window",
			AggregateID:   windowID.String(),
			Payload:       auditMeta,
		})
		return nil
	})

	writeJSON(w, 201, map[string]any{
		"id":               windowID,
		"status":           "open",
		"reason":           req.Reason,
		"opened_by":        userID,
		"opened_at":        now,
		"expires_at":       expiresAt,
		"duration_minutes": req.DurationMinutes,
	})
}

// ─── Get Active Window ───

func (h *MutationWindowHandler) GetActiveWindow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Auto-expire any windows that are past their expiry
	h.expireStaleWindows(ctx)

	var id, status, reason, openedBy string
	var openedAt, expiresAt time.Time
	var closedAt *time.Time
	var closedBy *string
	var closeReason *string

	err := h.pool.QueryRow(ctx, `
		SELECT id::text, status, reason, opened_by::text, opened_at, expires_at,
		       closed_by::text, closed_at, close_reason
		FROM proxmox_mutation_windows
		WHERE status = 'open'
		ORDER BY opened_at DESC LIMIT 1
	`).Scan(&id, &status, &reason, &openedBy, &openedAt, &expiresAt, &closedBy, &closedAt, &closeReason)
	if err != nil {
		writeJSON(w, 200, map[string]any{
			"active":              false,
			"mutation_enabled":    h.cfg.ProxmoxMutationEnabled,
			"window":              nil,
		})
		return
	}

	writeJSON(w, 200, map[string]any{
		"active":           true,
		"mutation_enabled": h.cfg.ProxmoxMutationEnabled,
		"window": map[string]any{
			"id":          id,
			"status":      status,
			"reason":      reason,
			"opened_by":   openedBy,
			"opened_at":   openedAt,
			"expires_at":  expiresAt,
			"closed_by":   closedBy,
			"closed_at":   closedAt,
			"close_reason": closeReason,
		},
	})
}

// ─── Close Mutation Window ───

func (h *MutationWindowHandler) CloseWindow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	windowID, err := uuid.Parse(chi.URLParam(r, "windowId"))
	if err != nil {
		writeErr(w, 400, "Invalid window ID")
		return
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	userID, _ := uuid.Parse(cl.UserID)

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Check window exists and is open
	var status string
	err = h.pool.QueryRow(ctx,
		`SELECT status FROM proxmox_mutation_windows WHERE id=$1`, windowID).Scan(&status)
	if err != nil {
		writeErr(w, 404, "Mutation window not found")
		return
	}
	if status != "open" {
		writeErr(w, 409, fmt.Sprintf("Window is already %s", status))
		return
	}

	// Close it
	now := time.Now()
	_, err = h.pool.Exec(ctx, `
		UPDATE proxmox_mutation_windows
		SET status='closed', closed_by=$1, closed_at=$2, close_reason=$3, updated_at=$4
		WHERE id=$5 AND status='open'
	`, userID, now, req.Reason, now, windowID)
	if err != nil {
		writeErr(w, 500, "Failed to close mutation window")
		return
	}

	// Audit + outbox via transaction
	auditMeta, _ := json.Marshal(map[string]any{
		"window_id":    windowID.String(),
		"closed_by":    userID.String(),
		"close_reason": req.Reason,
	})
	platformStr := "platform"
	_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		_ = audit.Write(ctx, tx, audit.Event{
			ActorID:    userID,
			Action:     "proxmox.mutation_window.closed",
			EntityType: "proxmox_mutation_window",
			EntityID:   windowID,
			NewValue:   auditMeta,
			Summary:    "Mutation window closed manually",
		})
		_ = outbox.Write(ctx, tx, &platformStr, outbox.Event{
			EventType:     "clarity.v1.proxmox.mutation_window.closed",
			AggregateType: "proxmox_mutation_window",
			AggregateID:   windowID.String(),
			Payload:       auditMeta,
		})
		return nil
	})

	writeJSON(w, 200, map[string]any{
		"id":           windowID,
		"status":       "closed",
		"closed_by":    userID,
		"closed_at":    now,
		"close_reason": req.Reason,
	})
}

// ─── HasActiveMutationWindow checks if there's an open, non-expired window.
// Called by the action handler before executing any Proxmox mutation.
func HasActiveMutationWindow(ctx context.Context, pool *pgxpool.Pool) bool {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM proxmox_mutation_windows WHERE status='open' AND expires_at > now()`).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// ─── expireStaleWindows marks any open windows past their expiry as expired.
func (h *MutationWindowHandler) expireStaleWindows(ctx context.Context) {
	rows, err := h.pool.Query(ctx,
		`SELECT id::text, opened_by::text FROM proxmox_mutation_windows
		 WHERE status='open' AND expires_at <= now()`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, openedBy string
		rows.Scan(&id, &openedBy)

		wid, _ := uuid.Parse(id)
		uid, _ := uuid.Parse(openedBy)

		h.pool.Exec(ctx,
			`UPDATE proxmox_mutation_windows SET status='expired', updated_at=now() WHERE id=$1 AND status='open'`,
			wid)

		meta, _ := json.Marshal(map[string]string{"window_id": id})
		platformStr := "platform"
		_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
			_ = audit.Write(ctx, tx, audit.Event{
				ActorID:    uid,
				Action:     "proxmox.mutation_window.expired",
				EntityType: "proxmox_mutation_window",
				EntityID:   wid,
				NewValue:   meta,
				Summary:    "Mutation window expired automatically",
			})
			_ = outbox.Write(ctx, tx, &platformStr, outbox.Event{
				EventType:     "clarity.v1.proxmox.mutation_window.expired",
				AggregateType: "proxmox_mutation_window",
				AggregateID:   id,
				Payload:       meta,
			})
			return nil
		})
	}
}
