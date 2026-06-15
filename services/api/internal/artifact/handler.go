package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/authz"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ───

type Artifact struct {
	ID              string    `json:"id"`
	TeamID          string    `json:"team_id"`
	ArtifactType    string    `json:"artifact_type"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ContentMarkdown string    `json:"content_markdown"`
	Status          string    `json:"status"`
	SourceType      string    `json:"source_type"`
	SourceData      any       `json:"source_data"`
	StorageObjectID *string   `json:"storage_object_id"`
	FileFormat      *string   `json:"file_format"`
	CreatedBy       string    `json:"created_by"`
	UpdatedBy       string    `json:"updated_by"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

type CreateRequest struct {
	ArtifactType    string         `json:"artifact_type"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	ContentMarkdown string         `json:"content_markdown"`
	Status          string         `json:"status"`
	SourceType      string         `json:"source_type"`
	SourceData      map[string]any `json:"source_data"`
}

type UpdateRequest struct {
	Title           *string        `json:"title"`
	Description     *string        `json:"description"`
	ContentMarkdown *string        `json:"content_markdown"`
	Status          *string        `json:"status"`
	SourceData      map[string]any `json:"source_data"`
}

var validTypes = map[string]bool{
	"document": true, "report": true, "presentation": true,
	"meeting_summary": true, "status_report": true,
	"decision_memo": true, "training_deck": true,
}

var validStatuses = map[string]bool{
	"draft": true, "published": true, "archived": true,
}

var validFileFormats = map[string]bool{
	"pptx": true, "pdf": true, "md": true,
}

const maxMarkdownLen = 500000 // 500KB

// Sensitive field patterns to suppress in source_data
var sensitiveKeys = map[string]bool{
	"password": true, "secret": true, "token": true, "api_key": true,
	"credential": true, "private_key": true, "access_token": true,
	"refresh_token": true,
}

// ─── Handler ───

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	if req.Title == "" || strings.TrimSpace(req.Title) == "" {
		writeErr(w, 400, "title is required")
		return
	}
	if !validTypes[req.ArtifactType] {
		writeErr(w, 400, "invalid artifact_type")
		return
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if !validStatuses[req.Status] {
		writeErr(w, 400, "invalid status")
		return
	}
	if len(req.ContentMarkdown) > maxMarkdownLen {
		writeErr(w, 400, fmt.Sprintf("content_markdown exceeds %d character limit", maxMarkdownLen))
		return
	}

	// Sanitize source_data
	sanitizedData := sanitizeSourceData(req.SourceData)
	sourceDataJSON, err := json.Marshal(sanitizedData)
	if err != nil {
		sourceDataJSON = []byte("{}")
	}

	var cl *iam.TokenClaims
	if c, ok := iam.GetClaims(r); ok {
		cl = c
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var art Artifact
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var id, createdBy, createdAt, updatedAt string
		err := tx.QueryRow(ctx, `
			INSERT INTO artifacts (team_id, artifact_type, title, description, content_markdown,
			                       status, source_type, source_data, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
			RETURNING id::text, created_by::text, created_at::text, updated_at::text
		`, teamID, req.ArtifactType, strings.TrimSpace(req.Title), req.Description,
			req.ContentMarkdown, req.Status, req.SourceType, sourceDataJSON, actorID,
		).Scan(&id, &createdBy, &createdAt, &updatedAt)
		if err != nil {
			return err
		}

		art = Artifact{
			ID: id, TeamID: teamIDStr, ArtifactType: req.ArtifactType,
			Title: req.Title, Description: req.Description, ContentMarkdown: req.ContentMarkdown,
			Status: req.Status, SourceType: req.SourceType, SourceData: sanitizedData,
			CreatedBy: createdBy, CreatedAt: createdAt, UpdatedAt: updatedAt,
		}

		// Audit
		artID, _ := uuid.Parse(id)
		meta, _ := json.Marshal(map[string]any{
			"artifact_type": req.ArtifactType,
			"title":         req.Title,
			"status":        req.Status,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.created",
			EntityType: "artifact", EntityID: artID, NewValue: meta,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.artifact.created",
			AggregateType: "artifact",
			AggregateID:   id,
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, 500, "Failed to create artifact")
		return
	}

	writeJSON(w, 201, art)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	// Parse query params
	artifactType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	includeFiles := r.URL.Query().Get("include_files") == "true"

	// If include_files, use the file-metadata join path
	if includeFiles {
		h.listWithFiles(ctx, w, teamID, teamIDStr, artifactType, status, search, includeArchived)
		return
	}

	// Build query
	qb := strings.Builder{}
	args := []any{teamID}
	argIdx := 2

	qb.WriteString(`SELECT id::text, artifact_type, title, description, content_markdown,
		status, source_type, source_data,
		storage_object_id::text, file_format,
		created_by::text, updated_by::text,
		created_at::text, updated_at::text
		FROM artifacts WHERE team_id = $1`)

	if !includeArchived {
		if status == "" {
			qb.WriteString(fmt.Sprintf(" AND status != 'archived'"))
		}
	}

	if artifactType != "" {
		if !validTypes[artifactType] {
			writeErr(w, 400, "invalid type filter")
			return
		}
		qb.WriteString(fmt.Sprintf(" AND artifact_type = $%d", argIdx))
		args = append(args, artifactType)
		argIdx++
	}

	if status != "" {
		if !validStatuses[status] {
			writeErr(w, 400, "invalid status filter")
			return
		}
		qb.WriteString(fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		qb.WriteString(fmt.Sprintf(" AND title ILIKE $%d", argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	qb.WriteString(" ORDER BY updated_at DESC LIMIT 100")

	rows, err := h.pool.Query(ctx, qb.String(), args...)
	if err != nil {
		writeErr(w, 500, "Failed to query artifacts")
		return
	}
	defer rows.Close()

	artifacts := []Artifact{}
	for rows.Next() {
		art, err := scanArtifact(rows)
		if err != nil {
			continue
		}
		artifacts = append(artifacts, art)
	}

	writeJSON(w, 200, artifacts)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	art, err := h.getArtifact(ctx, teamID, artifactID)
	if err != nil {
		writeErr(w, 404, "Artifact not found")
		return
	}

	writeJSON(w, 200, art)
}

func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	// Validate
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		writeErr(w, 400, "title cannot be empty")
		return
	}
	if req.Status != nil && !validStatuses[*req.Status] {
		writeErr(w, 400, "invalid status")
		return
	}
	if req.ContentMarkdown != nil && len(*req.ContentMarkdown) > maxMarkdownLen {
		writeErr(w, 400, fmt.Sprintf("content_markdown exceeds %d character limit", maxMarkdownLen))
		return
	}

	var cl *iam.TokenClaims
	if c, ok := iam.GetClaims(r); ok {
		cl = c
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var art Artifact
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Verify exists + team scoped
		var exists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM artifacts WHERE id=$1 AND team_id=$2)", artifactID, teamID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("not found")
		}

		// Build dynamic update
		setParts := []string{}
		args := []any{}
		argIdx := 1

		if req.Title != nil {
			setParts = append(setParts, fmt.Sprintf("title = $%d", argIdx))
			args = append(args, strings.TrimSpace(*req.Title))
			argIdx++
		}
		if req.Description != nil {
			setParts = append(setParts, fmt.Sprintf("description = $%d", argIdx))
			args = append(args, *req.Description)
			argIdx++
		}
		if req.ContentMarkdown != nil {
			setParts = append(setParts, fmt.Sprintf("content_markdown = $%d", argIdx))
			args = append(args, *req.ContentMarkdown)
			argIdx++
		}
		if req.Status != nil {
			setParts = append(setParts, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, *req.Status)
			argIdx++
		}
		if req.SourceData != nil {
			sdJSON, _ := json.Marshal(sanitizeSourceData(req.SourceData))
			setParts = append(setParts, fmt.Sprintf("source_data = $%d", argIdx))
			args = append(args, sdJSON)
			argIdx++
		}

		setParts = append(setParts, fmt.Sprintf("updated_by = $%d", argIdx))
		args = append(args, actorID)
		argIdx++

		if len(setParts) == 0 {
			return fmt.Errorf("no fields to update")
		}

		args = append(args, artifactID, teamID)
		query := fmt.Sprintf("UPDATE artifacts SET %s WHERE id = $%d AND team_id = $%d",
			strings.Join(setParts, ", "), argIdx, argIdx+1)

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}

		// Audit
		meta, _ := json.Marshal(map[string]any{
			"artifact_id": artifactID.String(),
			"updated_by":  actorID.String(),
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.updated",
			EntityType: "artifact", EntityID: artifactID, NewValue: meta,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.artifact.updated",
			AggregateType: "artifact",
			AggregateID:   artifactID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		if err.Error() == "not found" {
			writeErr(w, 404, "Artifact not found")
		} else if err.Error() == "no fields to update" {
			writeErr(w, 400, "No fields to update")
		} else {
			writeErr(w, 500, "Failed to update artifact")
		}
		return
	}

	art, _ = h.getArtifact(ctx, teamID, artifactID)
	writeJSON(w, 200, art)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	var cl *iam.TokenClaims
	if c, ok := iam.GetClaims(r); ok {
		cl = c
	}
	actorID, _ := uuid.Parse(cl.UserID)

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var exists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM artifacts WHERE id=$1 AND team_id=$2)", artifactID, teamID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("not found")
		}

		// Archive, don't physically delete
		_, err = tx.Exec(ctx, "UPDATE artifacts SET status='archived', updated_by=$1 WHERE id=$2 AND team_id=$3",
			actorID, artifactID, teamID)
		if err != nil {
			return err
		}

		// Audit
		meta, _ := json.Marshal(map[string]any{
			"artifact_id": artifactID.String(),
			"archived_by": actorID.String(),
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.archived",
			EntityType: "artifact", EntityID: artifactID, NewValue: meta,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.artifact.archived",
			AggregateType: "artifact",
			AggregateID:   artifactID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		if err.Error() == "not found" {
			writeErr(w, 404, "Artifact not found")
		} else {
			writeErr(w, 500, "Failed to archive artifact")
		}
		return
	}

	writeJSON(w, 200, map[string]any{"id": artifactID.String(), "status": "archived"})
}

// ─── Helpers ───

func (h *Handler) getArtifact(ctx context.Context, teamID, artifactID uuid.UUID) (Artifact, error) {
	var art Artifact
	var sourceData []byte
	var storageObjID, fileFmt *string

	row := h.pool.QueryRow(ctx, `
		SELECT id::text, artifact_type, title, description, content_markdown,
		       status, source_type, source_data,
		       storage_object_id::text, file_format,
		       created_by::text, updated_by::text,
		       created_at::text, updated_at::text
		FROM artifacts WHERE id=$1 AND team_id=$2
	`, artifactID, teamID)

	err := row.Scan(&art.ID, &art.ArtifactType, &art.Title, &art.Description,
		&art.ContentMarkdown, &art.Status, &art.SourceType, &sourceData,
		&storageObjID, &fileFmt, &art.CreatedBy, &art.UpdatedBy,
		&art.CreatedAt, &art.UpdatedAt)
	if err != nil {
		return art, err
	}

	var sd any
	json.Unmarshal(sourceData, &sd)
	art.SourceData = sanitizeSourceData(sd)
	art.StorageObjectID = derefString(storageObjID)
	art.FileFormat = fileFmt
	art.TeamID = teamID.String()

	return art, nil
}

func scanArtifact(rows pgx.Rows) (Artifact, error) {
	var art Artifact
	var sourceData []byte
	var storageObjID, fileFmt *string

	err := rows.Scan(&art.ID, &art.ArtifactType, &art.Title, &art.Description,
		&art.ContentMarkdown, &art.Status, &art.SourceType, &sourceData,
		&storageObjID, &fileFmt, &art.CreatedBy, &art.UpdatedBy,
		&art.CreatedAt, &art.UpdatedAt)
	if err != nil {
		return art, err
	}

	var sd any
	json.Unmarshal(sourceData, &sd)
	art.SourceData = sanitizeSourceData(sd)
	art.StorageObjectID = derefString(storageObjID)
	art.FileFormat = fileFmt

	return art, nil
}

func derefString(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func sanitizeSourceData(data any) map[string]any {
	if data == nil {
		return map[string]any{}
	}

	// Handle json.RawMessage from DB scan
	switch v := data.(type) {
	case map[string]any:
		return sanitizeMap(v)
	case []byte:
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return map[string]any{}
		}
		return sanitizeMap(m)
	default:
		return map[string]any{}
	}
}

func sanitizeMap(m map[string]any) map[string]any {
	sanitized := make(map[string]any)
	for k, v := range m {
		if isSensitiveKey(k) {
			sanitized[k] = "[REDACTED]"
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			sanitized[k] = sanitizeMap(val)
		case string:
			sanitized[k] = val
		default:
			sanitized[k] = v
		}
	}
	return sanitized
}

func isSensitiveKey(key string) bool {
	return sensitiveKeys[strings.ToLower(key)]
}

// ─── HTTP Helpers ───

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// Ensure imports
var _ = authz.Bypass{}
var _ = time.Now
