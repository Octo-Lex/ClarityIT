package domain

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// ─── Helpers ───

func (h *Handler) getClaims(r *http.Request) (*iam.TokenClaims, bool) {
	return iam.GetClaims(r)
}

func strPtr(s string) *string { return &s }

func bodyHash(body string) string {
	h := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%x", h[:])
}

// requireOwnOrAny checks whether the actor has either the .any permission (always satisfies)
// or the .own permission (satisfies only when isOwner is true).
// Returns an error if neither condition is met, and writes a permission-denied audit event.
func (h *Handler) requireOwnOrAny(ctx context.Context, actorID uuid.UUID, teamID uuid.UUID, ownPerm, anyPerm string, isOwner bool) error {
	// Check .any first — it always satisfies
	var hasAny bool
	err := h.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM team_memberships tm
			JOIN role_permissions rp ON rp.role_id = tm.role_id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE tm.user_id = $1 AND tm.team_id = $2 AND p.name = $3
		)
	`, actorID, teamID, anyPerm).Scan(&hasAny)
	if err == nil && hasAny {
		return nil
	}

	// Check .own — only if the actor is the owner
	if isOwner {
		var hasOwn bool
		err := h.pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM team_memberships tm
				JOIN role_permissions rp ON rp.role_id = tm.role_id
				JOIN permissions p ON p.id = rp.permission_id
				WHERE tm.user_id = $1 AND tm.team_id = $2 AND p.name = $3
			)
		`, actorID, teamID, ownPerm).Scan(&hasOwn)
		if err == nil && hasOwn {
			return nil
		}
	}

	// Denied — write audit
	permChecked := ownPerm
	if !isOwner {
		permChecked = anyPerm
	}
	h.writeDomainPermissionDenied(ctx, actorID, teamID, permChecked)

	return fmt.Errorf("permission denied")
}

// writeDomainPermissionDenied writes a sanitized identity.permission.denied audit event.
func (h *Handler) writeDomainPermissionDenied(ctx context.Context, actorID, teamID uuid.UUID, permission string) {
	meta, _ := json.Marshal(map[string]string{
		"permission_checked": permission,
		"team_id":           teamID.String(),
		"denial_reason":     "insufficient_role_permissions",
	})
	summary := fmt.Sprintf("Permission denied: %s", permission)
	h.pool.Exec(ctx, `
		INSERT INTO audit_logs (event_id, actor_id, actor_type, action, entity_type, entity_id, old_value, new_value, change_summary)
		VALUES ($1, $2, 'user', 'identity.permission.denied', 'permission', $2, '{}', $3, $4)
	`, uuid.New().String(), actorID, meta, summary)
}

// ─── Objects CRUD ───

func (h *Handler) CreateObject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		ObjectType string          `json:"object_type"`
		Title      string          `json:"title"`
		Summary    string          `json:"summary"`
		Status     string          `json:"status"`
		Priority   string          `json:"priority"`
		OwnerID    *string         `json:"owner_id"`
		Metadata   json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title == "" {
		writeErr(w, http.StatusBadRequest, "Title is required")
		return
	}
	if req.Status == "" {
		req.Status = "open"
	}
	if req.Priority == "" {
		req.Priority = "none"
	}

	var ownerID *uuid.UUID
	if req.OwnerID != nil && *req.OwnerID != "" {
		oid, _ := uuid.Parse(*req.OwnerID)
		ownerID = &oid
	}

	if req.Metadata == nil {
		req.Metadata = json.RawMessage(`{}`)
	}

	var objectID uuid.UUID
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO objects (team_id, object_type, title, summary, status, priority, owner_user_id, created_by, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, teamID, req.ObjectType, req.Title, req.Summary, req.Status, req.Priority, ownerID, actorID, req.Metadata).Scan(&id)
		if err != nil {
			return err
		}
		objectID = id

		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"object_type": req.ObjectType, "title_sha": bodyHash(req.Title),
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID: actorID, ActorType: "user", TeamID: &teamID,
			Action: "object.created", EntityType: "object", EntityID: objectID,
			NewValue: meta, Summary: "Object created",
		})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType: "clarity.v1.object.created", AggregateType: "object", AggregateID: objectID.String(),
			Payload: meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": objectID})
}

func (h *Handler) ListObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	objectType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	query := `SELECT id, object_type, title, summary, status, priority, owner_user_id, created_by, version, created_at, updated_at
		FROM objects WHERE team_id = $1 AND deleted_at IS NULL`
	args := []any{teamID}
	n := 2

	if objectType != "" {
		query += fmt.Sprintf(" AND object_type = $%d", n)
		args = append(args, objectType)
		n++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, status)
		n++
	}
	query += " ORDER BY created_at DESC LIMIT 200"

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to list objects")
		return
	}
	defer rows.Close()

	type Obj struct {
		ID         string  `json:"id"`
		ObjectType string  `json:"object_type"`
		Title      string  `json:"title"`
		Summary    string  `json:"summary"`
		Status     string  `json:"status"`
		Priority   string  `json:"priority"`
		OwnerID    *string `json:"owner_id"`
		CreatedBy  *string `json:"created_by"`
		Version    int     `json:"version"`
		CreatedAt  string  `json:"created_at"`
		UpdatedAt  string  `json:"updated_at"`
	}
	var objects []Obj
	for rows.Next() {
		var o Obj
		rows.Scan(&o.ID, &o.ObjectType, &o.Title, &o.Summary, &o.Status, &o.Priority, &o.OwnerID, &o.CreatedBy, &o.Version, &o.CreatedAt, &o.UpdatedAt)
		objects = append(objects, o)
	}
	if objects == nil {
		objects = []Obj{}
	}
	writeJSON(w, http.StatusOK, objects)
}

func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, err := uuid.Parse(chi.URLParam(r, "objectId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid object ID")
		return
	}

	var obj struct {
		ID         string          `json:"id"`
		TeamID     string          `json:"team_id"`
		ObjectType string          `json:"object_type"`
		Title      string          `json:"title"`
		Summary    string          `json:"summary"`
		Status     string          `json:"status"`
		Priority   string          `json:"priority"`
		OwnerID    *string         `json:"owner_id"`
		CreatedBy  *string         `json:"created_by"`
		Version    int             `json:"version"`
		Metadata   json.RawMessage `json:"metadata"`
		CreatedAt  string          `json:"created_at"`
		UpdatedAt  string          `json:"updated_at"`
	}
	err = h.pool.QueryRow(ctx, `
		SELECT id, team_id, object_type, title, COALESCE(summary,''), status, priority,
		       owner_user_id, created_by, version, COALESCE(metadata,'{}'), created_at, updated_at
		FROM objects WHERE id = $1 AND deleted_at IS NULL
	`, objectID).Scan(&obj.ID, &obj.TeamID, &obj.ObjectType, &obj.Title, &obj.Summary, &obj.Status, &obj.Priority,
		&obj.OwnerID, &obj.CreatedBy, &obj.Version, &obj.Metadata, &obj.CreatedAt, &obj.UpdatedAt)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Object not found")
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (h *Handler) UpdateObject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, err := uuid.Parse(chi.URLParam(r, "objectId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid object ID")
		return
	}
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title    *string `json:"title"`
		Summary  *string `json:"summary"`
		Status   *string `json:"status"`
		Priority *string `json:"priority"`
		OwnerID  *string `json:"owner_id"`
		Version  int     `json:"expected_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		var oldStatus, oldPriority string
		err := tx.QueryRow(ctx, "SELECT team_id, status, COALESCE(priority,'none') FROM objects WHERE id = $1 AND deleted_at IS NULL", objectID).Scan(&teamID, &oldStatus, &oldPriority)
		if err != nil {
			return fmt.Errorf("object not found")
		}

		sets := []string{}
		args := []any{objectID}
		n := 2
		if req.Title != nil {
			sets = append(sets, fmt.Sprintf("title = $%d", n))
			args = append(args, *req.Title)
			n++
		}
		if req.Summary != nil {
			sets = append(sets, fmt.Sprintf("summary = $%d", n))
			args = append(args, *req.Summary)
			n++
		}
		if req.Status != nil {
			sets = append(sets, fmt.Sprintf("status = $%d", n))
			args = append(args, *req.Status)
			n++
		}
		if req.Priority != nil {
			sets = append(sets, fmt.Sprintf("priority = $%d", n))
			args = append(args, *req.Priority)
			n++
		}
		if req.OwnerID != nil {
			sets = append(sets, fmt.Sprintf("owner_user_id = $%d", n))
			if *req.OwnerID != "" {
				oid, _ := uuid.Parse(*req.OwnerID)
				args = append(args, oid)
			} else {
				args = append(args, nil)
			}
			n++
		}

		if len(sets) == 0 {
			return fmt.Errorf("no fields to update")
		}

		sets = append(sets, "version = version + 1")
		query := fmt.Sprintf("UPDATE objects SET %s WHERE id = $1 AND version = %d AND deleted_at IS NULL", strings.Join(sets, ", "), req.Version)
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("CONFLICT: stale version")
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"old_status": oldStatus, "new_status": req.Status,
		})
		audit.Write(ctx, tx, audit.Event{
			ActorID: actorID, ActorType: "user", TeamID: &teamID,
			Action: "object.updated", EntityType: "object", EntityID: objectID,
			NewValue: meta, Summary: "Object updated",
		})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType: "clarity.v1.object.updated", AggregateType: "object", AggregateID: objectID.String(),
			Payload: meta,
		})
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "CONFLICT") {
			writeErr(w, http.StatusConflict, "Stale version — object was modified by another request")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Object updated"})
}

func (h *Handler) DeleteObject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, err := uuid.Parse(chi.URLParam(r, "objectId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid object ID")
		return
	}
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		err := tx.QueryRow(ctx, "SELECT team_id FROM objects WHERE id = $1 AND deleted_at IS NULL", objectID).Scan(&teamID)
		if err != nil {
			return fmt.Errorf("object not found")
		}
		_, err = tx.Exec(ctx, "UPDATE objects SET deleted_at = NOW() WHERE id = $1", objectID)
		if err != nil {
			return err
		}
		meta := iam.MergeAuditMeta(ctx, map[string]any{"object_id": objectID.String()})
		audit.Write(ctx, tx, audit.Event{
			ActorID: actorID, ActorType: "user", TeamID: &teamID,
			Action: "object.deleted", EntityType: "object", EntityID: objectID,
			NewValue: meta, Summary: "Object soft-deleted",
		})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{
			EventType: "clarity.v1.object.deleted", AggregateType: "object", AggregateID: objectID.String(),
			Payload: meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Object deleted"})
}

// ─── Object Links ───

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	fromID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		ToID         string `json:"to_object_id"`
		RelationType string `json:"relation_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	toID, _ := uuid.Parse(req.ToID)

	var linkID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var count int
		tx.QueryRow(ctx, `SELECT COUNT(*) FROM objects WHERE id IN ($1, $2) AND team_id = $3 AND deleted_at IS NULL`, fromID, toID, teamID).Scan(&count)
		if count != 2 {
			return fmt.Errorf("both objects must exist in the same team")
		}
		return tx.QueryRow(ctx, `
			INSERT INTO object_links (team_id, from_object_id, to_object_id, relation_type, created_by)
			VALUES ($1, $2, $3, $4, $5) RETURNING id
		`, teamID, fromID, toID, req.RelationType, actorID).Scan(&linkID)
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	meta := iam.MergeAuditMeta(ctx, map[string]any{"from": fromID.String(), "to": toID.String(), "relation": req.RelationType})
	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "object.linked", EntityType: "link", EntityID: linkID, NewValue: meta, Summary: "Objects linked"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.object.linked", AggregateType: "link", AggregateID: linkID.String(), Payload: meta})
		return nil
	})

	writeJSON(w, http.StatusOK, map[string]any{"id": linkID})
}

func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	rows, err := h.pool.Query(ctx, `SELECT id, from_object_id, to_object_id, relation_type, created_at FROM object_links WHERE (from_object_id = $1 OR to_object_id = $1) ORDER BY created_at`, objectID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed")
		return
	}
	defer rows.Close()
	type Link struct {
		ID       string `json:"id"`
		FromID   string `json:"from_object_id"`
		ToID     string `json:"to_object_id"`
		Relation string `json:"relation_type"`
		Created  string `json:"created_at"`
	}
	var links []Link
	for rows.Next() {
		var l Link
		rows.Scan(&l.ID, &l.FromID, &l.ToID, &l.Relation, &l.Created)
		links = append(links, l)
	}
	if links == nil {
		links = []Link{}
	}
	writeJSON(w, http.StatusOK, links)
}

func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	linkID, _ := uuid.Parse(chi.URLParam(r, "linkId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT team_id FROM object_links WHERE id = $1", linkID).Scan(&teamID); err != nil {
			return fmt.Errorf("link not found")
		}
		if _, err := tx.Exec(ctx, "DELETE FROM object_links WHERE id = $1", linkID); err != nil {
			return err
		}
		meta := iam.MergeAuditMeta(ctx, map[string]any{"link_id": linkID.String()})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "object.unlinked", EntityType: "link", EntityID: linkID, NewValue: meta, Summary: "Link removed"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.object.unlinked", AggregateType: "link", AggregateID: linkID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Link removed"})
}

// ─── Comments ───

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}

	var commentID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO object_comments (team_id, object_id, author_id, body_markdown)
			VALUES ($1, $2, $3, $4) RETURNING id
		`, teamID, objectID, actorID, req.Body).Scan(&commentID)
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"comment_id": commentID.String(), "object_id": objectID.String(),
			"body_sha256": bodyHash(req.Body), "body_length": len(req.Body),
		})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "object.commented", EntityType: "comment", EntityID: commentID, NewValue: meta, Summary: "Comment added"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.object.commented", AggregateType: "comment", AggregateID: commentID.String(), Payload: meta})
		return nil
	})

	writeJSON(w, http.StatusOK, map[string]any{"id": commentID})
}

func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	rows, err := h.pool.Query(ctx, `SELECT id, author_id, body_markdown, created_at, updated_at FROM object_comments WHERE object_id = $1 ORDER BY created_at`, objectID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed")
		return
	}
	defer rows.Close()
	type C struct {
		ID        string  `json:"id"`
		AuthorID  string  `json:"author_id"`
		Body      string  `json:"body"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt *string `json:"updated_at"`
	}
	var comments []C
	for rows.Next() {
		var c C
		rows.Scan(&c.ID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
		comments = append(comments, c)
	}
	if comments == nil {
		comments = []C{}
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID, _ := uuid.Parse(chi.URLParam(r, "commentId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}

	var teamID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var authorID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT author_id, team_id FROM object_comments WHERE id = $1", commentID).Scan(&authorID, &teamID); err != nil {
			return fmt.Errorf("comment not found")
		}

		// Own-vs-any permission check
		isOwner := authorID == actorID
		if err := h.requireOwnOrAny(ctx, actorID, teamID, "objects.comments.update.own", "objects.comments.update.any", isOwner); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, "UPDATE object_comments SET body_markdown = $1, updated_at = NOW() WHERE id = $2", req.Body, commentID)
		return err
	})
	if err != nil {
		if err.Error() == "permission denied" {
			writeErr(w, http.StatusForbidden, "Not authorized to update this comment")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		meta := iam.MergeAuditMeta(ctx, map[string]any{"comment_id": commentID.String(), "body_sha256": bodyHash(req.Body)})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "object.comment_updated", EntityType: "comment", EntityID: commentID, NewValue: meta, Summary: "Comment updated"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.object.comment_updated", AggregateType: "comment", AggregateID: commentID.String(), Payload: meta})
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Comment updated"})
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID, _ := uuid.Parse(chi.URLParam(r, "commentId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var teamID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var authorID uuid.UUID
		if err := tx.QueryRow(ctx, "SELECT author_id, team_id FROM object_comments WHERE id = $1", commentID).Scan(&authorID, &teamID); err != nil {
			return fmt.Errorf("comment not found")
		}

		// Own-vs-any permission check
		isOwner := authorID == actorID
		if err := h.requireOwnOrAny(ctx, actorID, teamID, "objects.comments.delete.own", "objects.comments.delete.any", isOwner); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, "DELETE FROM object_comments WHERE id = $1", commentID)
		return err
	})
	if err != nil {
		if err.Error() == "permission denied" {
			writeErr(w, http.StatusForbidden, "Not authorized to delete this comment")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		meta := iam.MergeAuditMeta(ctx, map[string]any{"comment_id": commentID.String()})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "object.comment_deleted", EntityType: "comment", EntityID: commentID, NewValue: meta, Summary: "Comment deleted"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.object.comment_deleted", AggregateType: "comment", AggregateID: commentID.String(), Payload: meta})
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Comment deleted"})
}

// ─── Work Items ───

func (h *Handler) CreateWorkItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title      string  `json:"title"`
		Summary    string  `json:"summary"`
		Status     string  `json:"status"`
		Priority   string  `json:"priority"`
		WorkType   string  `json:"work_item_type"`
		OwnerID    *string `json:"owner_id"`
		AssigneeID *string `json:"assignee_id"`
		ProjectID  *string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}
	if req.Status == "" {
		req.Status = "open"
	}
	if req.Priority == "" {
		req.Priority = "none"
	}
	if req.WorkType == "" {
		req.WorkType = "task"
	}

	var objectID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var ownerUUID, assigneeUUID, projectUUID *uuid.UUID
		if req.OwnerID != nil && *req.OwnerID != "" {
			oid, _ := uuid.Parse(*req.OwnerID)
			ownerUUID = &oid
		}
		if req.AssigneeID != nil && *req.AssigneeID != "" {
			aid, _ := uuid.Parse(*req.AssigneeID)
			assigneeUUID = &aid
		}
		if req.ProjectID != nil && *req.ProjectID != "" {
			pid, _ := uuid.Parse(*req.ProjectID)
			projectUUID = &pid
		}

		if err := tx.QueryRow(ctx, `
			INSERT INTO objects (team_id, object_type, title, summary, status, priority, owner_user_id, created_by)
			VALUES ($1, 'work_item', $2, $3, $4, $5, $6, $7) RETURNING id
		`, teamID, req.Title, req.Summary, req.Status, req.Priority, ownerUUID, actorID).Scan(&objectID); err != nil {
			return err
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO work_items (object_id, work_item_type, assignee_user_id, project_id)
			VALUES ($1, $2, $3, $4)
		`, objectID, req.WorkType, assigneeUUID, projectUUID)
		if err != nil {
			return err
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{
			"object_id": objectID.String(), "work_item_type": req.WorkType,
			"title_sha": bodyHash(req.Title),
		})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "work.item.created", EntityType: "work_item", EntityID: objectID, NewValue: meta, Summary: "Work item created"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.work.item.created", AggregateType: "work_item", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": objectID})
}

func (h *Handler) ListWorkItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	status := r.URL.Query().Get("status")

	query := `SELECT o.id, o.title, COALESCE(o.summary,''), o.status, o.priority, o.owner_user_id, o.version,
	          w.work_item_type, w.assignee_user_id, w.project_id, o.created_at
	          FROM objects o JOIN work_items w ON w.object_id = o.id
	          WHERE o.team_id = $1 AND o.deleted_at IS NULL`
	args := []any{teamID}
	if status != "" {
		query += " AND o.status = $2"
		args = append(args, status)
	}
	query += " ORDER BY o.created_at DESC LIMIT 200"

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed")
		return
	}
	defer rows.Close()

	type WI struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Summary    string  `json:"summary"`
		Status     string  `json:"status"`
		Priority   string  `json:"priority"`
		OwnerID    *string `json:"owner_id"`
		Version    int     `json:"version"`
		WorkType   string  `json:"work_item_type"`
		AssigneeID *string `json:"assignee_id"`
		ProjectID  *string `json:"project_id"`
		CreatedAt  string  `json:"created_at"`
	}
	var items []WI
	for rows.Next() {
		var w WI
		rows.Scan(&w.ID, &w.Title, &w.Summary, &w.Status, &w.Priority, &w.OwnerID, &w.Version, &w.WorkType, &w.AssigneeID, &w.ProjectID, &w.CreatedAt)
		items = append(items, w)
	}
	if items == nil {
		items = []WI{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetWorkItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	var wi struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Summary    string  `json:"summary"`
		Status     string  `json:"status"`
		Priority   string  `json:"priority"`
		OwnerID    *string `json:"owner_id"`
		Version    int     `json:"version"`
		WorkType   string  `json:"work_item_type"`
		AssigneeID *string `json:"assignee_id"`
		ProjectID  *string `json:"project_id"`
		CreatedAt  string  `json:"created_at"`
	}
	err := h.pool.QueryRow(ctx, `
		SELECT o.id, o.title, COALESCE(o.summary,''), o.status, o.priority, o.owner_user_id, o.version,
		       w.work_item_type, w.assignee_user_id, w.project_id, o.created_at
		FROM objects o JOIN work_items w ON w.object_id = o.id
		WHERE o.id = $1 AND o.deleted_at IS NULL
	`, objectID).Scan(&wi.ID, &wi.Title, &wi.Summary, &wi.Status, &wi.Priority, &wi.OwnerID, &wi.Version, &wi.WorkType, &wi.AssigneeID, &wi.ProjectID, &wi.CreatedAt)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Work item not found")
		return
	}
	writeJSON(w, http.StatusOK, wi)
}

func (h *Handler) UpdateWorkItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title    *string `json:"title"`
		Summary  *string `json:"summary"`
		Status   *string `json:"status"`
		Priority *string `json:"priority"`
		OwnerID  *string `json:"owner_id"`
		Version  int     `json:"expected_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		var oldStatus, ownerID, assigneeID *string
		err := tx.QueryRow(ctx, `
			SELECT o.team_id, o.status, o.owner_user_id::text, w.assignee_user_id::text
			FROM objects o JOIN work_items w ON w.object_id = o.id
			WHERE o.id = $1 AND o.deleted_at IS NULL
		`, objectID).Scan(&teamID, &oldStatus, &ownerID, &assigneeID)
		if err != nil {
			return fmt.Errorf("not found")
		}

		// Own-vs-any: owner/creator/assignee counts as "own"
		isOwner := (ownerID != nil && *ownerID == actorID.String()) ||
			(assigneeID != nil && *assigneeID == actorID.String()) ||
			true // creator check: objects.created_by
		// Also check created_by
		var createdBy *string
		tx.QueryRow(ctx, "SELECT created_by::text FROM objects WHERE id = $1", objectID).Scan(&createdBy)
		isOwner = isOwner || (createdBy != nil && *createdBy == actorID.String())

		if err := h.requireOwnOrAny(ctx, actorID, teamID, "work.items.update.own", "work.items.update.any", isOwner); err != nil {
			return err
		}

		sets := []string{}
		args := []any{objectID}
		n := 2
		if req.Title != nil {
			sets = append(sets, fmt.Sprintf("title = $%d", n))
			args = append(args, *req.Title)
			n++
		}
		if req.Summary != nil {
			sets = append(sets, fmt.Sprintf("summary = $%d", n))
			args = append(args, *req.Summary)
			n++
		}
		if req.Status != nil {
			sets = append(sets, fmt.Sprintf("status = $%d", n))
			args = append(args, *req.Status)
			n++
		}
		if req.Priority != nil {
			sets = append(sets, fmt.Sprintf("priority = $%d", n))
			args = append(args, *req.Priority)
			n++
		}
		if len(sets) == 0 {
			return fmt.Errorf("no fields to update")
		}
		sets = append(sets, "version = version + 1")

		query := fmt.Sprintf("UPDATE objects SET %s WHERE id = $1 AND version = %d AND deleted_at IS NULL", strings.Join(sets, ", "), req.Version)
		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("CONFLICT: stale version")
		}

		// Status change event
		if req.Status != nil && *req.Status != *oldStatus {
			meta := iam.MergeAuditMeta(ctx, map[string]any{"old_status": *oldStatus, "new_status": *req.Status})
			outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.work.item.status_changed", AggregateType: "work_item", AggregateID: objectID.String(), Payload: meta})
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{"old_status": oldStatus})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "work.item.updated", EntityType: "work_item", EntityID: objectID, NewValue: meta, Summary: "Work item updated"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.work.item.updated", AggregateType: "work_item", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		if err.Error() == "permission denied" {
			writeErr(w, http.StatusForbidden, "Not authorized to update this work item")
		} else if strings.Contains(err.Error(), "CONFLICT") {
			writeErr(w, http.StatusConflict, "Stale version")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Work item updated"})
}

func (h *Handler) DeleteWorkItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		var ownerID, assigneeID, createdBy *string
		err := tx.QueryRow(ctx, `
			SELECT o.team_id, o.owner_user_id::text, w.assignee_user_id::text, o.created_by::text
			FROM objects o JOIN work_items w ON w.object_id = o.id
			WHERE o.id = $1 AND o.deleted_at IS NULL
		`, objectID).Scan(&teamID, &ownerID, &assigneeID, &createdBy)
		if err != nil {
			return fmt.Errorf("not found")
		}

		isOwner := (ownerID != nil && *ownerID == actorID.String()) ||
			(assigneeID != nil && *assigneeID == actorID.String()) ||
			(createdBy != nil && *createdBy == actorID.String())

		if err := h.requireOwnOrAny(ctx, actorID, teamID, "work.items.delete.own", "work.items.delete.any", isOwner); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, "UPDATE objects SET deleted_at = NOW() WHERE id = $1", objectID)
		if err != nil {
			return err
		}
		meta := iam.MergeAuditMeta(ctx, map[string]any{"object_id": objectID.String()})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "work.item.deleted", EntityType: "work_item", EntityID: objectID, NewValue: meta, Summary: "Work item soft-deleted"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.work.item.deleted", AggregateType: "work_item", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		if err.Error() == "permission denied" {
			writeErr(w, http.StatusForbidden, "Not authorized to delete this work item")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Work item deleted"})
}

func (h *Handler) BoardView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))

	rows, err := h.pool.Query(ctx, `
		SELECT o.status, o.id, o.title, o.priority, w.work_item_type, o.owner_user_id
		FROM objects o JOIN work_items w ON w.object_id = o.id
		WHERE o.team_id = $1 AND o.deleted_at IS NULL
		ORDER BY o.priority, o.created_at
	`, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed")
		return
	}
	defer rows.Close()

	board := map[string][]map[string]any{}
	for rows.Next() {
		var status, id, title, priority, workType string
		var ownerID *string
		rows.Scan(&status, &id, &title, &priority, &workType, &ownerID)
		board[status] = append(board[status], map[string]any{
			"id": id, "title": title, "priority": priority, "work_item_type": workType, "owner_id": ownerID,
		})
	}
	if board == nil {
		board = map[string][]map[string]any{}
	}
	writeJSON(w, http.StatusOK, board)
}

// ─── Incidents ───

func (h *Handler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title    string  `json:"title"`
		Summary  string  `json:"summary"`
		Severity string  `json:"severity"`
		Impact   string  `json:"impact"`
		OwnerID  *string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}
	if req.Severity == "" {
		req.Severity = "sev3"
	}

	var objectID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var ownerUUID *uuid.UUID
		if req.OwnerID != nil && *req.OwnerID != "" {
			oid, _ := uuid.Parse(*req.OwnerID)
			ownerUUID = &oid
		}

		if err := tx.QueryRow(ctx, `
			INSERT INTO objects (team_id, object_type, title, summary, status, priority, owner_user_id, created_by)
			VALUES ($1, 'incident', $2, $3, 'open', 'critical', $4, $5) RETURNING id
		`, teamID, req.Title, req.Summary, ownerUUID, actorID).Scan(&objectID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `INSERT INTO work_items (object_id, work_item_type) VALUES ($1, 'incident')`, objectID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `INSERT INTO incidents (object_id, severity, impact) VALUES ($1, $2, $3)`, objectID, req.Severity, req.Impact); err != nil {
			return err
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{"object_id": objectID.String(), "severity": req.Severity})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "incident.opened", EntityType: "incident", EntityID: objectID, NewValue: meta, Summary: "Incident opened"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.incident.opened", AggregateType: "incident", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": objectID})
}

func (h *Handler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	rows, err := h.pool.Query(ctx, `
		SELECT o.id, o.title, o.status, i.severity, i.resolved_at, o.created_at
		FROM objects o JOIN incidents i ON i.object_id = o.id
		WHERE o.team_id = $1 AND o.deleted_at IS NULL ORDER BY o.created_at DESC
	`, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed")
		return
	}
	defer rows.Close()
	type Inc struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Status     string  `json:"status"`
		Severity   string  `json:"severity"`
		ResolvedAt *string `json:"resolved_at"`
		CreatedAt  string  `json:"created_at"`
	}
	var incidents []Inc
	for rows.Next() {
		var i Inc
		rows.Scan(&i.ID, &i.Title, &i.Status, &i.Severity, &i.ResolvedAt, &i.CreatedAt)
		incidents = append(incidents, i)
	}
	if incidents == nil {
		incidents = []Inc{}
	}
	writeJSON(w, http.StatusOK, incidents)
}

func (h *Handler) GetIncident(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	var inc struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Summary    string  `json:"summary"`
		Status     string  `json:"status"`
		Severity   string  `json:"severity"`
		Impact     string  `json:"impact"`
		ResolvedAt *string `json:"resolved_at"`
		CreatedAt  string  `json:"created_at"`
		Version    int     `json:"version"`
	}
	err := h.pool.QueryRow(ctx, `
		SELECT o.id, o.title, COALESCE(o.summary,''), o.status, i.severity, i.impact, i.resolved_at, o.created_at, o.version
		FROM objects o JOIN incidents i ON i.object_id = o.id
		WHERE o.id = $1 AND o.deleted_at IS NULL
	`, objectID).Scan(&inc.ID, &inc.Title, &inc.Summary, &inc.Status, &inc.Severity, &inc.Impact, &inc.ResolvedAt, &inc.CreatedAt, &inc.Version)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Incident not found")
		return
	}
	writeJSON(w, http.StatusOK, inc)
}

func (h *Handler) UpdateIncident(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title    *string `json:"title"`
		Summary  *string `json:"summary"`
		Status   *string `json:"status"`
		Severity *string `json:"severity"`
		Version  int     `json:"expected_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		var oldStatus, oldSeverity string
		err := tx.QueryRow(ctx, "SELECT team_id, status FROM objects WHERE id = $1 AND deleted_at IS NULL", objectID).Scan(&teamID, &oldStatus)
		if err != nil {
			return fmt.Errorf("not found")
		}
		if req.Severity != nil {
			tx.QueryRow(ctx, "SELECT severity FROM incidents WHERE object_id = $1", objectID).Scan(&oldSeverity)
		}

		sets := []string{}
		args := []any{objectID}
		n := 2
		if req.Title != nil {
			sets = append(sets, fmt.Sprintf("title = $%d", n))
			args = append(args, *req.Title)
			n++
		}
		if req.Summary != nil {
			sets = append(sets, fmt.Sprintf("summary = $%d", n))
			args = append(args, *req.Summary)
			n++
		}
		if req.Status != nil {
			sets = append(sets, fmt.Sprintf("status = $%d", n))
			args = append(args, *req.Status)
			n++
		}
		if len(sets) == 0 && req.Severity == nil {
			return fmt.Errorf("no fields")
		}
		if len(sets) > 0 {
			sets = append(sets, "version = version + 1")
			tag, err := tx.Exec(ctx, fmt.Sprintf("UPDATE objects SET %s WHERE id = $1 AND version = %d AND deleted_at IS NULL", strings.Join(sets, ", "), req.Version), args...)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("CONFLICT")
			}
		} else {
			var v int
			if err := tx.QueryRow(ctx, "SELECT version FROM objects WHERE id = $1 AND deleted_at IS NULL", objectID).Scan(&v); err != nil {
				return fmt.Errorf("not found")
			}
			if v != req.Version {
				return fmt.Errorf("CONFLICT")
			}
		}

		if req.Status != nil && *req.Status == "resolved" && oldStatus != "resolved" {
			tx.Exec(ctx, "UPDATE incidents SET resolved_at = NOW() WHERE object_id = $1", objectID)
			outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.incident.resolved", AggregateType: "incident", AggregateID: objectID.String(), Payload: json.RawMessage(`{"status":"resolved"}`)})
		}

		if req.Severity != nil && *req.Severity != oldSeverity {
			tx.Exec(ctx, "UPDATE incidents SET severity = $1 WHERE object_id = $2", *req.Severity, objectID)
			outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.incident.severity_changed", AggregateType: "incident", AggregateID: objectID.String(), Payload: json.RawMessage(fmt.Sprintf(`{"old_severity":"%s","new_severity":"%s"}`, oldSeverity, *req.Severity))})
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "incident.updated", EntityType: "incident", EntityID: objectID, NewValue: meta, Summary: "Incident updated"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.incident.updated", AggregateType: "incident", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "CONFLICT") {
			writeErr(w, http.StatusConflict, "Stale version")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Incident updated"})
}

func (h *Handler) AddTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}

	var commentID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO object_comments (team_id, object_id, author_id, body_markdown)
			VALUES ($1, $2, $3, $4) RETURNING id
		`, teamID, objectID, actorID, req.Body).Scan(&commentID)
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		meta := iam.MergeAuditMeta(ctx, map[string]any{"comment_id": commentID.String(), "object_id": objectID.String(), "body_sha256": bodyHash(req.Body)})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "incident.timeline_added", EntityType: "comment", EntityID: commentID, NewValue: meta, Summary: "Timeline entry added"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.incident.timeline_added", AggregateType: "incident", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]any{"id": commentID})
}

// ─── Projects ───

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title    string          `json:"title"`
		Summary  string          `json:"summary"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}
	if req.Metadata == nil {
		req.Metadata = json.RawMessage(`{}`)
	}

	var objectID uuid.UUID
	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO objects (team_id, object_type, title, summary, status, priority, created_by, metadata)
			VALUES ($1, 'project', $2, $3, 'active', 'none', $4, $5) RETURNING id
		`, teamID, req.Title, req.Summary, actorID, req.Metadata).Scan(&objectID); err != nil {
			return err
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{"object_id": objectID.String()})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "project.created", EntityType: "project", EntityID: objectID, NewValue: meta, Summary: "Project created"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.project.created", AggregateType: "project", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": objectID})
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	rows, err := h.pool.Query(ctx, `SELECT id, title, COALESCE(summary,''), status, version, created_at FROM objects WHERE team_id = $1 AND object_type = 'project' AND deleted_at IS NULL ORDER BY created_at DESC`, teamID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed")
		return
	}
	defer rows.Close()
	type P struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Summary   string `json:"summary"`
		Status    string `json:"status"`
		Version   int    `json:"version"`
		CreatedAt string `json:"created_at"`
	}
	var projects []P
	for rows.Next() {
		var p P
		rows.Scan(&p.ID, &p.Title, &p.Summary, &p.Status, &p.Version, &p.CreatedAt)
		projects = append(projects, p)
	}
	if projects == nil {
		projects = []P{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) { h.GetObject(w, r) }

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectID, _ := uuid.Parse(chi.URLParam(r, "objectId"))
	claims, ok := h.getClaims(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	actorID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Title    *string         `json:"title"`
		Summary  *string         `json:"summary"`
		Status   *string         `json:"status"`
		Version  int             `json:"expected_version"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid body")
		return
	}

	err := database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var teamID uuid.UUID
		err := tx.QueryRow(ctx, "SELECT team_id FROM objects WHERE id = $1 AND object_type = 'project' AND deleted_at IS NULL", objectID).Scan(&teamID)
		if err != nil {
			return fmt.Errorf("project not found")
		}

		sets := []string{}
		args := []any{objectID}
		n := 2
		if req.Title != nil {
			sets = append(sets, fmt.Sprintf("title = $%d", n))
			args = append(args, *req.Title)
			n++
		}
		if req.Summary != nil {
			sets = append(sets, fmt.Sprintf("summary = $%d", n))
			args = append(args, *req.Summary)
			n++
		}
		if req.Status != nil {
			sets = append(sets, fmt.Sprintf("status = $%d", n))
			args = append(args, *req.Status)
			n++
		}
		if req.Metadata != nil {
			sets = append(sets, fmt.Sprintf("metadata = $%d", n))
			args = append(args, req.Metadata)
			n++
		}
		if len(sets) == 0 {
			return fmt.Errorf("no fields")
		}
		sets = append(sets, "version = version + 1")
		tag, err := tx.Exec(ctx, fmt.Sprintf("UPDATE objects SET %s WHERE id = $1 AND version = %d AND deleted_at IS NULL", strings.Join(sets, ", "), req.Version), args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("CONFLICT")
		}

		meta := iam.MergeAuditMeta(ctx, map[string]any{})
		audit.Write(ctx, tx, audit.Event{ActorID: actorID, ActorType: "user", TeamID: &teamID, Action: "project.updated", EntityType: "project", EntityID: objectID, NewValue: meta, Summary: "Project updated"})
		outbox.Write(ctx, tx, strPtr(teamID.String()), outbox.Event{EventType: "clarity.v1.project.updated", AggregateType: "project", AggregateID: objectID.String(), Payload: meta})
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "CONFLICT") {
			writeErr(w, http.StatusConflict, "Stale version")
		} else {
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Project updated"})
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) { h.DeleteObject(w, r) }

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
