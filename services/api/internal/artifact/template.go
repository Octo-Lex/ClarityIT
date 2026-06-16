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
	ID              string          `json:"id"`
	TeamID          *string         `json:"team_id"`
	TemplateType    string          `json:"template_type"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	ContentMarkdown string          `json:"content_markdown"`
	Metadata        any             `json:"metadata"`
	IsSystem        bool            `json:"is_system"`
	CreatedBy       *string         `json:"created_by"`
	CreatedAt       string          `json:"created_at"`
	TemplateFormat  string          `json:"template_format"`
	DocumentJSON    json.RawMessage `json:"document_json,omitempty"`
	SchemaVersion   *int            `json:"schema_version,omitempty"`
}

type CreateTemplateRequest struct {
	TemplateType    string          `json:"template_type"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	ContentMarkdown string          `json:"content_markdown"`
	Metadata        map[string]any  `json:"metadata"`
	TemplateFormat  string          `json:"template_format"`
	DocumentJSON    json.RawMessage `json:"document_json"`
	SchemaVersion   *int            `json:"schema_version"`
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
		       metadata, is_system, created_by::text, created_at::text,
		       template_format, document_json, schema_version
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
		var id, templateType, name, createdAt string
		var description *string
		var content *string
		var teamIDVal, createdBy *string
		var isSystem bool
		var metadata []byte
		var templateFormat string
		var docJSON []byte
		var schemaVersion *int

		if err := rows.Scan(&id, &teamIDVal, &templateType, &name, &description, &content,
			&metadata, &isSystem, &createdBy, &createdAt,
			&templateFormat, &docJSON, &schemaVersion); err != nil {
			continue
		}

		var meta any
		json.Unmarshal(metadata, &meta)

		entry := map[string]any{
			"id":               id,
			"team_id":          teamIDVal,
			"template_type":    templateType,
			"name":             name,
			"description":      "",
			"metadata":         meta,
			"is_system":        isSystem,
			"created_by":       createdBy,
			"template_format":  templateFormat,
		}
		if content != nil {
			entry["content_markdown"] = *content
		} else {
			entry["content_markdown"] = ""
		}
		if description != nil {
			entry["description"] = *description
		}
		if docJSON != nil {
			var docData any
			json.Unmarshal(docJSON, &docData)
			entry["document_json"] = docData
		}
		if schemaVersion != nil {
			entry["schema_version"] = *schemaVersion
		}
		out = append(out, entry)
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
	// Sanitize metadata
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	metaJSON, _ := json.Marshal(req.Metadata)

	// v1.4 Track 5: Determine template format
	if req.TemplateFormat == "" {
		req.TemplateFormat = "markdown"
	}
	if req.TemplateFormat != "markdown" && req.TemplateFormat != "document_json" {
		writeErr(w, 400, "invalid template_format: must be 'markdown' or 'document_json'")
		return
	}

	// Validate based on format
	var docJSONBytes []byte
	if req.TemplateFormat == "document_json" {
		if len(req.DocumentJSON) == 0 {
			writeErr(w, 400, "document_json is required for document_json templates")
			return
		}
		// Validate document_json structure
		var docJSON DocumentJSON
		if err := json.Unmarshal(req.DocumentJSON, &docJSON); err != nil {
			writeErr(w, 400, "invalid document_json: must be a JSON object")
			return
		}
		if req.SchemaVersion != nil && *req.SchemaVersion != 1 {
			writeErr(w, 400, "schema_version must be 1")
			return
		}
		// Force schema_version = 1
		if docJSON.SchemaVersion == 0 {
			docJSON.SchemaVersion = 1
		}
		if docJSON.SchemaVersion != 1 {
			writeErr(w, 400, "schema_version must be 1")
			return
		}
		if err := validateBlocks(docJSON.Blocks); err != nil {
			writeErr(w, 400, fmt.Sprintf("invalid document_json: %s", err.Error()))
			return
		}
		docJSONBytes, _ = json.Marshal(docJSON)
		sv := 1
		req.SchemaVersion = &sv
	} else {
		// markdown format validation
		if strings.TrimSpace(req.ContentMarkdown) == "" {
			writeErr(w, 400, "content_markdown is required for markdown templates")
			return
		}
		if len(req.ContentMarkdown) > maxTemplateContentLen {
			writeErr(w, 400, "content_markdown exceeds length limit")
			return
		}
	}
	if !validTemplateTypes[req.TemplateType] {
		writeErr(w, 400, "invalid template_type")
		return
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var templateID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var contentMD *string
		if req.TemplateFormat == "markdown" {
			contentMD = &req.ContentMarkdown
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO artifact_templates (team_id, template_type, name, description, content_markdown, document_json, schema_version, template_format, metadata, is_system, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10)
			RETURNING id::text
		`, teamID, req.TemplateType, strings.TrimSpace(req.Name), req.Description,
			contentMD, docJSONBytes, req.SchemaVersion, req.TemplateFormat, metaJSON, actorID).Scan(&templateID)
		return err
	})
	if err != nil {
		writeErr(w, 500, "Failed to create template")
		return
	}

	writeJSON(w, 201, map[string]any{
		"id":               templateID,
		"team_id":          teamIDStr,
		"template_type":    req.TemplateType,
		"name":             req.Name,
		"description":      req.Description,
		"content_markdown": req.ContentMarkdown,
		"metadata":         req.Metadata,
		"is_system":        false,
		"template_format":  req.TemplateFormat,
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
	var templateType, name string
	var contentMarkdown *string
	var templateFormat string
	var docJSONBytes []byte
	err = h.pool.QueryRow(ctx, `
		SELECT template_type, name, content_markdown, template_format, document_json
		FROM artifact_templates
		WHERE id = $1 AND (team_id IS NULL OR team_id = $2)
	`, templateID, teamID).Scan(&templateType, &name, &contentMarkdown, &templateFormat, &docJSONBytes)
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
	isDocTemplate := templateFormat == "document_json" && len(docJSONBytes) > 0

	if isDocTemplate {
		// v1.4 Track 5: Instantiate document_json template → create artifact + artifact_documents
		var docJSON DocumentJSON
		if err := json.Unmarshal(docJSONBytes, &docJSON); err != nil {
			writeErr(w, 500, "Template document_json is malformed")
			return
		}
		docJSON.SchemaVersion = 1
		if strings.TrimSpace(title) != "" {
			docJSON.Title = title
		}

		if err := validateBlocks(docJSON.Blocks); err != nil {
			writeErr(w, 500, fmt.Sprintf("Template has invalid blocks: %s", err.Error()))
			return
		}

		wordCount := computeWordCount(docJSON.Blocks)
		finalDocJSON, _ := json.Marshal(docJSON)

		sourceDataStr := fmt.Sprintf(`{"template_id":"%s","template_format":"document_json"}`, templateIDStr)

		err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
			artID := uuid.New()
			err := tx.QueryRow(ctx, `
				INSERT INTO artifacts (id, team_id, artifact_type, title, description, status, source_type, source_data, created_by, updated_by)
				VALUES ($1, $2, 'document', $3, $4, $5, 'template', $6, $7, $7)
				RETURNING id::text
			`, artID, teamID, strings.TrimSpace(title), req.Description, status, sourceDataStr, actorID).Scan(&artifactID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO artifact_documents (artifact_id, document_type, document_json, schema_version, word_count)
				VALUES ($1, $2, $3, 1, $4)
			`, artID, docJSON.DocumentType, finalDocJSON, wordCount)
			if err != nil {
				return err
			}
			meta, _ := json.Marshal(map[string]any{
				"artifact_type":  "document",
				"document_type": docJSON.DocumentType,
				"title":         title,
				"word_count":    wordCount,
				"source":        "template",
			})
			_ = audit.Write(ctx, tx, audit.Event{
				TeamID: &teamID, ActorID: actorID, Action: "document.instantiated",
				EntityType: "artifact", EntityID: artID, NewValue: meta,
			})
			_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
				EventType:     "clarity.v1.artifact.document.created",
				AggregateType: "artifact",
				AggregateID:   artID.String(),
				Payload:       meta,
			})
			// v1.4 Track 7: Create initial version with source=template
			createDocumentVersion(ctx, tx, artID, teamID, finalDocJSON, wordCount,
				VersionSourceTemplate, "Created from template", &actorID)
			return nil
		})
		if err != nil {
			writeErr(w, 500, "Failed to create document from template")
			return
		}

		writeJSON(w, 201, map[string]any{
			"artifact_id":    artifactID,
			"artifact_type":  "document",
			"document_type": docJSON.DocumentType,
			"title":          title,
			"status":         status,
			"schema_version": 1,
			"word_count":     wordCount,
			"source_type":    "template",
		})
		return
	}
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
