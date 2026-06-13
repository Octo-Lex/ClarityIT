package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

type Handler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewHandler(pool *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{pool: pool, cfg: cfg}
}

// ─── Users ───

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.pool.Query(ctx, `
		SELECT u.id, u.name, u.email, u.is_active, u.created_at, u.last_login_at,
		       COALESCE(json_agg(pr.name) FILTER (WHERE pr.name IS NOT NULL), '[]')
		FROM users u
		LEFT JOIN user_platform_roles upr ON upr.user_id = u.id AND upr.revoked_at IS NULL
		LEFT JOIN platform_roles pr ON pr.id = upr.platform_role_id
		WHERE u.deleted_at IS NULL
		GROUP BY u.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load users")
		return
	}
	defer rows.Close()

	type User struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Email        string   `json:"email"`
		IsActive     bool     `json:"is_active"`
		PlatformRoles []string `json:"platform_roles"`
		CreatedAt    string   `json:"created_at"`
		LastLoginAt  *string  `json:"last_login_at"`
	}
	var users []User
	for rows.Next() {
		var u User
		var roles string
		rows.Scan(&u.ID, &u.Name, &u.Email, &u.IsActive, &u.CreatedAt, &u.LastLoginAt, &roles)
		json.Unmarshal([]byte(roles), &u.PlatformRoles)
		users = append(users, u)
	}
	if users == nil {
		users = []User{}
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var name, email string
	var isActive bool
	err = h.pool.QueryRow(ctx, `
		SELECT name, email, is_active FROM users WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&name, &email, &isActive)
	if err != nil {
		writeErr(w, http.StatusNotFound, "User not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id": userID, "name": name, "email": email, "is_active": isActive,
	})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Name     *string `json:"name"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		if req.Name != nil {
			if _, err := tx.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", *req.Name, userID); err != nil {
				return err
			}
		}
		if req.IsActive != nil {
			if !*req.IsActive {
				if _, err := tx.Exec(ctx, "UPDATE users SET deactivated_at = NOW() WHERE id = $1", userID); err != nil {
					return err
				}
			} else {
				if _, err := tx.Exec(ctx, "UPDATE users SET deactivated_at = NULL WHERE id = $1", userID); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, "UPDATE users SET is_active = $1 WHERE id = $2", *req.IsActive, userID); err != nil {
				return err
			}
		}

		newValue := iam.MergeAuditMeta(ctx, map[string]any{
			"name": req.Name, "is_active": req.IsActive,
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID:    actorID,
			ActorType:  "user",
			Action:     "platform.user.updated",
			EntityType: "user",
			EntityID:   userID,
			NewValue:   newValue,
			Summary:    "Platform user updated",
		})

		outbox.Write(ctx, tx, nil, outbox.Event{
			EventType:     "clarity.v1.platform.user.updated",
			AggregateType: "user",
			AggregateID:   userID.String(),
			Payload:       newValue,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User updated"})
}

// ─── Teams ───

func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug, COALESCE(t.description, ''),
		       (SELECT COUNT(*) FROM team_memberships WHERE team_id = t.id) as member_count,
		       t.created_at
		FROM teams t WHERE t.deleted_at IS NULL
		ORDER BY t.name
	`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load teams")
		return
	}
	defer rows.Close()

	type Team struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		MemberCount int    `json:"member_count"`
		CreatedAt   string `json:"created_at"`
	}
	var teams []Team
	for rows.Next() {
		var t Team
		rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Description, &t.MemberCount, &t.CreatedAt)
		teams = append(teams, t)
	}
	if teams == nil {
		teams = []Team{}
	}
	writeJSON(w, http.StatusOK, teams)
}

// ─── Audit ───

func (h *Handler) ListAudit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query params
	q := r.URL.Query()
	limit := 100
	action := q.Get("action")
	entityType := q.Get("entity_type")

	query := `
		SELECT event_id, actor_id, actor_type, action, entity_type, entity_id,
		       change_summary, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []any{}
	argN := 1

	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argN)
		args = append(args, action)
		argN++
	}
	if entityType != "" {
		query += fmt.Sprintf(" AND entity_type = $%d", argN)
		args = append(args, entityType)
		argN++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to load audit")
		return
	}
	defer rows.Close()

	type AuditEntry struct {
		EventID   string `json:"event_id"`
		ActorID   string `json:"actor_id"`
		ActorType string `json:"actor_type"`
		Action    string `json:"action"`
		EntityType string `json:"entity_type"`
		EntityID  string `json:"entity_id"`
		Summary   string `json:"summary"`
		CreatedAt string `json:"created_at"`
	}
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		rows.Scan(&e.EventID, &e.ActorID, &e.ActorType, &e.Action, &e.EntityType, &e.EntityID, &e.Summary, &e.CreatedAt)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// ─── Settings ───

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	// Platform settings — for now, return bootstrap status
	ctx := r.Context()
	var isLocked bool
	var lockedAt *string
	h.pool.QueryRow(ctx, "SELECT is_locked, locked_at FROM bootstrap_lock").Scan(&isLocked, &lockedAt)

	writeJSON(w, http.StatusOK, map[string]any{
		"bootstrapped": isLocked,
		"locked_at":    lockedAt,
		"version":      "0.4.0",
	})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	// Platform settings updates — placeholder for now
	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	_ = claims

	writeJSON(w, http.StatusOK, map[string]string{"message": "Settings updated"})
}

// ─── Helpers ───

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}
