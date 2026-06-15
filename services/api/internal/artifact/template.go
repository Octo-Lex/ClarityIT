package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ─── Template Types ───

type ArtifactTemplate struct {
	ID              string `json:"id"`
	TeamID          *string `json:"team_id"`
	TemplateType    string `json:"template_type"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ContentMarkdown string `json:"content_markdown"`
	Metadata        any    `json:"metadata"`
	IsSystem        bool   `json:"is_system"`
	CreatedBy       *string `json:"created_by"`
	CreatedAt       string `json:"created_at"`
}

type CreateTemplateRequest struct {
	TemplateType    string         `json:"template_type"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	ContentMarkdown string         `json:"content_markdown"`
	Metadata        map[string]any `json:"metadata"`
}

type InstantiateRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      *string `json:"status"`
}

var validTemplateTypes = map[string]bool{
	"document": true, "report": true, "meeting_summary": true,
	"status_report": true, "decision_memo": true,
	"training_deck": true, "presentation": true,
}

const maxTemplateNameLen = 200
const maxTemplateContentLen = 500000

// ─── Template Handler ───

func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	typeFilter := r.URL.Query().Get("type")

	query := `
		SELECT id::text, team_id::text, template_type, name, description, content_markdown,
		       metadata, is_system, created_by::text, created_at::text
		FROM artifact_templates
		WHERE (team_id IS NULL OR team_id = $1)
	`
	args := []any{teamID}
	argIdx := 2

	if typeFilter != "" {
		query += fmt.Sprintf(" AND template_type = $%d", argIdx)
		args = append(args, typeFilter)
		argIdx++
	}
	query += " ORDER BY is_system DESC, name ASC"

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, 500, "Failed to list templates")
		return
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id, templateType, name, content, createdAt string
		var description string
		var teamIDVal, createdBy *string
		var isSystem bool
		var metadata []byte

		if err := rows.Scan(&id, &teamIDVal, &templateType, &name, &description, &content,
			&metadata, &isSystem, &createdBy, &createdAt); err != nil {
			continue
		}

		var meta any
		json.Unmarshal(metadata, &meta)

		out = append(out, map[string]any{
			"id":               id,
			"team_id":          teamIDVal,
			"template_type":    templateType,
			"name":             name,
			"description":      description,
			"content_markdown": content,
			"metadata":         meta,
			"is_system":        isSystem,
			"created_by":       createdBy,
			"created_at":       createdAt,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, 200, out)
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, 400, "name is required")
		return
	}
	if len(req.Name) > maxTemplateNameLen {
		writeErr(w, 400, "name exceeds length limit")
		return
	}
	if strings.TrimSpace(req.ContentMarkdown) == "" {
		writeErr(w, 400, "content_markdown is required")
		return
	}
	if len(req.ContentMarkdown) > maxTemplateContentLen {
		writeErr(w, 400, "content_markdown exceeds length limit")
		return
	}
	if !validTemplateTypes[req.TemplateType] {
		writeErr(w, 400, "invalid template_type")
		return
	}

	// Sanitize metadata
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	metaJSON, _ := json.Marshal(req.Metadata)

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var templateID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO artifact_templates (team_id, template_type, name, description, content_markdown, metadata, is_system, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, false, $7)
			RETURNING id::text
		`, teamID, req.TemplateType, strings.TrimSpace(req.Name), req.Description,
			req.ContentMarkdown, metaJSON, actorID).Scan(&templateID)
		return err
	})
	if err != nil {
		writeErr(w, 500, "Failed to create template")
		return
	}

	writeJSON(w, 201, map[string]any{
		"id":              templateID,
		"team_id":         teamIDStr,
		"template_type":   req.TemplateType,
		"name":            req.Name,
		"description":     req.Description,
		"content_markdown": req.ContentMarkdown,
		"metadata":        req.Metadata,
		"is_system":       false,
	})
}

func (h *Handler) InstantiateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	templateIDStr := chi.URLParam(r, "templateId")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid template ID")
		return
	}

	// Look up template (must be system or belong to this team)
	var templateType, name, contentMarkdown string
	err = h.pool.QueryRow(ctx, `
		SELECT template_type, name, content_markdown
		FROM artifact_templates
		WHERE id = $1 AND (team_id IS NULL OR team_id = $2)
	`, templateID, teamID).Scan(&templateType, &name, &contentMarkdown)
	if err != nil {
		writeErr(w, 404, "Template not found")
		return
	}

	var req InstantiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body — use template name as default title
		req = InstantiateRequest{}
	}

	title := req.Title
	if strings.TrimSpace(title) == "" {
		title = name
	}
	status := "draft"
	if req.Status != nil && validStatuses[*req.Status] {
		status = *req.Status
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var artifactID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO artifacts (team_id, artifact_type, title, description, content_markdown,
			                       status, source_type, source_data, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, 'template', $7, $8, $8)
			RETURNING id::text
		`, teamID, templateType, strings.TrimSpace(title), req.Description,
			contentMarkdown, status, fmt.Sprintf(`{"template_id":"%s"}`, templateIDStr),
			actorID).Scan(&artifactID)
		if err != nil {
			return err
		}

		artID, _ := uuid.Parse(artifactID)
		meta, _ := json.Marshal(map[string]any{
			"artifact_type": templateType,
			"title":         title,
			"source":        "template",
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.instantiated",
			EntityType: "artifact", EntityID: artID, NewValue: meta,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.artifact.created",
			AggregateType: "artifact",
			AggregateID:   artifactID,
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, 500, "Failed to create artifact from template")
		return
	}

	writeJSON(w, 201, map[string]any{
		"artifact_id":      artifactID,
		"artifact_type":    templateType,
		"title":            title,
		"status":           status,
		"content_markdown": contentMarkdown,
	})
}
