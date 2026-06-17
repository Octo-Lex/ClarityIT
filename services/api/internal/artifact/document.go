package artifact

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─── v1.4 Track 1: Native Document Artifact Model ───

// Document block validation bounds
const (
	maxBlocks       = 500
	maxTextLen      = 20000
	maxListItems    = 200
	maxTableCols    = 20
	maxTableRows    = 200
	maxDocumentJSON = 5_000_000 // 5MB JSONB
)

var validDocumentTypes = map[string]bool{
	"general_document":    true,
	"decision_memo":       true,
	"implementation_plan": true,
	"incident_summary":    true,
	"training_doc":        true,
	"architecture_doc":    true,
	"project_report":      true,
	"status_report":       true,
	"meeting_summary":     true,
	"executive_brief":     true,
}

var validBlockTypes = map[string]bool{
	"heading":       true,
	"paragraph":     true,
	"bullets":       true,
	"numbered_list": true,
	"table":         true,
	"quote":         true,
	"callout":       true,
	"page_break":    true,
}

var validCalloutVariants = map[string]bool{
	"info":     true,
	"warning":  true,
	"success":  true,
	"error":    true,
	"note":     true,
	"tip":      true,
}

// DocumentBlock represents a single block in the document model.
type DocumentBlock struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Level    *int           `json:"level,omitempty"`
	Text     *string        `json:"text,omitempty"`
	Items    []string       `json:"items,omitempty"`
	Headers  []string       `json:"headers,omitempty"`
	Rows     [][]string     `json:"rows,omitempty"`
	Variant  *string        `json:"variant,omitempty"`
}

// DocumentJSON is the canonical JSONB document model.
type DocumentJSON struct {
	SchemaVersion int              `json:"schema_version"`
	Title         string           `json:"title"`
	DocumentType  string           `json:"document_type"`
	Blocks        []DocumentBlock  `json:"blocks"`
}

// CreateDocumentRequest is the POST body for creating a native document.
type CreateDocumentRequest struct {
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	DocumentType  string        `json:"document_type"`
	DocumentJSON  DocumentJSON  `json:"document_json"`
	Status        string        `json:"status"`
}

// UpdateDocumentRequest is the PATCH body for updating a native document.
type UpdateDocumentRequest struct {
	Title        *string       `json:"title"`
	Description  *string       `json:"description"`
	DocumentType *string       `json:"document_type"`
	DocumentJSON *DocumentJSON `json:"document_json"`
}

// DocumentResponse is the API response for a native document.
type DocumentResponse struct {
	Artifact
	DocumentType            string       `json:"document_type"`
	DocumentJSON            DocumentJSON `json:"document_json"`
	SchemaVersion           int          `json:"schema_version"`
	WordCount               int          `json:"word_count"`
	LastExportedStorageID   *string      `json:"last_exported_storage_object_id,omitempty"`
}

// ─── Block Validation ───

func validateBlocks(blocks []DocumentBlock) error {
	if len(blocks) == 0 {
		return fmt.Errorf("document must have at least one block")
	}
	if len(blocks) > maxBlocks {
		return fmt.Errorf("document exceeds maximum of %d blocks", maxBlocks)
	}

	seenIDs := make(map[string]bool)
	for i, blk := range blocks {
		if blk.ID == "" {
			return fmt.Errorf("block %d: id is required", i)
		}
		if seenIDs[blk.ID] {
			return fmt.Errorf("block %d: duplicate block id %q", i, blk.ID)
		}
		seenIDs[blk.ID] = true

		if !validBlockTypes[blk.Type] {
			return fmt.Errorf("block %d: unknown block type %q", i, blk.Type)
		}

		switch blk.Type {
		case "heading":
			if blk.Level == nil {
				return fmt.Errorf("block %d: heading requires level", i)
			}
			if *blk.Level < 1 || *blk.Level > 6 {
				return fmt.Errorf("block %d: heading level must be 1-6", i)
			}
			if blk.Text == nil || strings.TrimSpace(*blk.Text) == "" {
				return fmt.Errorf("block %d: heading requires non-empty text", i)
			}
			if len(*blk.Text) > maxTextLen {
				return fmt.Errorf("block %d: heading text exceeds %d chars", i, maxTextLen)
			}
		case "paragraph":
			if blk.Text == nil || strings.TrimSpace(*blk.Text) == "" {
				return fmt.Errorf("block %d: paragraph requires non-empty text", i)
			}
			if len(*blk.Text) > maxTextLen {
				return fmt.Errorf("block %d: paragraph text exceeds %d chars", i, maxTextLen)
			}
		case "bullets", "numbered_list":
			if len(blk.Items) == 0 {
				return fmt.Errorf("block %d: %s requires non-empty items array", i, blk.Type)
			}
			if len(blk.Items) > maxListItems {
				return fmt.Errorf("block %d: %s exceeds %d items", i, blk.Type, maxListItems)
			}
			for j, item := range blk.Items {
				if strings.TrimSpace(item) == "" {
					return fmt.Errorf("block %d: item %d is empty", i, j)
				}
				if len(item) > maxTextLen {
					return fmt.Errorf("block %d: item %d exceeds %d chars", i, j, maxTextLen)
				}
			}
		case "table":
			if len(blk.Headers) == 0 {
				return fmt.Errorf("block %d: table requires non-empty headers array", i)
			}
			if len(blk.Headers) > maxTableCols {
				return fmt.Errorf("block %d: table exceeds %d columns", i, maxTableCols)
			}
			if len(blk.Rows) > maxTableRows {
				return fmt.Errorf("block %d: table exceeds %d rows", i, maxTableRows)
			}
			for j, h := range blk.Headers {
				if len(h) > maxTextLen {
					return fmt.Errorf("block %d: header %d exceeds %d chars", i, j, maxTextLen)
				}
			}
			for j, row := range blk.Rows {
				if len(row) != len(blk.Headers) {
					return fmt.Errorf("block %d: row %d has %d cells, expected %d", i, j, len(row), len(blk.Headers))
				}
				for _, cell := range row {
					if len(cell) > maxTextLen {
						return fmt.Errorf("block %d: cell in row %d exceeds %d chars", i, j, maxTextLen)
					}
				}
			}
		case "quote":
			if blk.Text == nil || strings.TrimSpace(*blk.Text) == "" {
				return fmt.Errorf("block %d: quote requires non-empty text", i)
			}
			if len(*blk.Text) > maxTextLen {
				return fmt.Errorf("block %d: quote text exceeds %d chars", i, maxTextLen)
			}
		case "callout":
			if blk.Variant == nil || !validCalloutVariants[*blk.Variant] {
				return fmt.Errorf("block %d: callout requires valid variant (info, warning, success, error, note, tip)", i)
			}
			if blk.Text == nil || strings.TrimSpace(*blk.Text) == "" {
				return fmt.Errorf("block %d: callout requires non-empty text", i)
			}
			if len(*blk.Text) > maxTextLen {
				return fmt.Errorf("block %d: callout text exceeds %d chars", i, maxTextLen)
			}
		case "page_break":
			// page_break has no additional fields
		}
	}
	return nil
}

// computeWordCount counts words from text-bearing blocks.
func computeWordCount(blocks []DocumentBlock) int {
	count := 0
	countText := func(s string) {
		count += len(strings.Fields(s))
	}
	for _, blk := range blocks {
		switch blk.Type {
		case "heading", "paragraph", "quote", "callout":
			if blk.Text != nil {
				countText(*blk.Text)
			}
		case "bullets", "numbered_list":
			for _, item := range blk.Items {
				countText(item)
			}
		case "table":
			for _, h := range blk.Headers {
				countText(h)
			}
			for _, row := range blk.Rows {
				for _, cell := range row {
					countText(cell)
				}
			}
		}
	}
	return count
}

// validateDocumentJSON validates the full document JSON structure.
func validateDocumentJSON(doc *DocumentJSON, docType string) error {
	if doc.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1")
	}
	if !validDocumentTypes[docType] {
		return fmt.Errorf("invalid document_type: %q", docType)
	}
	return validateBlocks(doc.Blocks)
}

// validateIncomingDocumentJSON validates the schema_version from the request BEFORE overwriting.
func validateIncomingSchemaVersion(doc *DocumentJSON) error {
	if doc.SchemaVersion != 0 && doc.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1")
	}
	return nil
}

// ─── Handlers ───

// CreateDocument creates a new native structured document artifact.
func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "Unauthorized")
		return
	}
	actorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeErr(w, 401, "Invalid actor")
		return
	}

	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, 400, "title is required")
		return
	}

	if !validDocumentTypes[req.DocumentType] {
		writeErr(w, 400, fmt.Sprintf("invalid document_type: %q", req.DocumentType))
		return
	}

	if req.Status == "" {
		req.Status = "draft"
	}
	if !validStatuses[req.Status] {
		writeErr(w, 400, "invalid status")
		return
	}

	// Set document JSON metadata
	if err := validateIncomingSchemaVersion(&req.DocumentJSON); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	req.DocumentJSON.SchemaVersion = 1
	req.DocumentJSON.DocumentType = req.DocumentType
	if req.DocumentJSON.Title == "" {
		req.DocumentJSON.Title = req.Title
	}

	if err := validateDocumentJSON(&req.DocumentJSON, req.DocumentType); err != nil {
		writeErr(w, 400, err.Error())
		return
	}

	wordCount := computeWordCount(req.DocumentJSON.Blocks)

	// Marshal document JSON
	docJSONBytes, err := json.Marshal(req.DocumentJSON)
	if err != nil {
		writeErr(w, 500, "Failed to marshal document JSON")
		return
	}
	if len(docJSONBytes) > maxDocumentJSON {
		writeErr(w, 400, fmt.Sprintf("document JSON exceeds %d bytes", maxDocumentJSON))
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "Failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	artifactID := uuid.New()

	// Create artifacts row
	var createdArtID, createdAt, updatedAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO artifacts (id, team_id, artifact_type, title, description, status, source_type, source_data, created_by, updated_by)
		VALUES ($1, $2, 'document', $3, $4, $5, 'native', '{}', $6, $6)
		RETURNING id::text, created_at::text, updated_at::text
	`, artifactID, teamID, req.Title, req.Description, req.Status, actorID).Scan(&createdArtID, &createdAt, &updatedAt)
	if err != nil {
		writeErr(w, 500, "Failed to create artifact")
		return
	}

	// Create artifact_documents row
	var docCreatedAt, docUpdatedAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO artifact_documents (artifact_id, document_type, document_json, schema_version, word_count)
		VALUES ($1, $2, $3, 1, $4)
		RETURNING created_at::text, updated_at::text
	`, artifactID, req.DocumentType, docJSONBytes, wordCount).Scan(&docCreatedAt, &docUpdatedAt)
	if err != nil {
		writeErr(w, 500, "Failed to create document")
		return
	}

	// Audit
	teamUUID := teamID
	newVal, _ := json.Marshal(map[string]any{
		"artifact_type":  "document",
		"document_type": req.DocumentType,
		"title":         req.Title,
		"word_count":   wordCount,
		"block_count":  len(req.DocumentJSON.Blocks),
	})
	audit.Write(ctx, tx, audit.Event{
		Action:     "document.created",
		EntityType: "artifact",
		EntityID:   artifactID,
		ActorID:    actorID,
		TeamID:     &teamUUID,
		NewValue:   newVal,
	})

	// Outbox event
	payload, _ := json.Marshal(map[string]any{
		"artifact_id":   createdArtID,
		"team_id":       teamIDStr,
		"document_type": req.DocumentType,
		"title":         req.Title,
		"word_count":    wordCount,
	})
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.artifact.document.created",
		AggregateType: "artifact",
		AggregateID:   artifactID.String(),
		Payload:       payload,
	})

	// v1.4 Track 7: Create initial version snapshot
	createDocumentVersion(ctx, tx, artifactID, teamID, docJSONBytes, wordCount,
		VersionSourceUserSave, "Initial version", &actorID)

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Failed to commit transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(DocumentResponse{
		Artifact: Artifact{
			ID:           createdArtID,
			TeamID:       teamIDStr,
			ArtifactType: "document",
			Title:        req.Title,
			Description:  req.Description,
			Status:       req.Status,
			SourceType:   "native",
			CreatedBy:    claims.UserID,
			UpdatedBy:    claims.UserID,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		},
		DocumentType:  req.DocumentType,
		DocumentJSON:  req.DocumentJSON,
		SchemaVersion: 1,
		WordCount:     wordCount,
	})

	// v1.5 Knowledge index hook
	h.fireIndexHook(ctx, teamIDStr, "clarity_document", createdArtID)
}

// ListDocuments returns document-type artifacts for a team.
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"

	query := `
		SELECT a.id::text, a.team_id::text, a.artifact_type, a.title, a.description,
		       a.content_markdown, a.status, a.source_type, a.source_data,
		       a.storage_object_id::text, a.file_format,
		       a.created_by::text, a.updated_by::text,
		       a.created_at::text, a.updated_at::text,
		       d.document_type, d.document_json, d.schema_version, d.word_count,
		       d.last_exported_storage_object_id::text
		FROM artifacts a
		JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.team_id = $1 AND a.artifact_type = 'document'
	`
	args := []any{teamID}
	if !includeArchived {
		query += " AND a.status != 'archived'"
	}
	query += " ORDER BY a.updated_at DESC LIMIT 100"

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, 500, "Failed to query documents")
		return
	}
	defer rows.Close()

	var docs []DocumentResponse
	for rows.Next() {
		var doc DocumentResponse
		var contentMD, storageID, fileFmt, lastExported *string
		var sourceDataRaw any

		var description *string
		err := rows.Scan(
			&doc.ID, &doc.TeamID, &doc.ArtifactType, &doc.Title, &description,
			&contentMD, &doc.Status, &doc.SourceType, &sourceDataRaw,
			&storageID, &fileFmt,
			&doc.CreatedBy, &doc.UpdatedBy,
			&doc.CreatedAt, &doc.UpdatedAt,
			&doc.DocumentType, &doc.DocumentJSON, &doc.SchemaVersion, &doc.WordCount,
			&lastExported,
		)
		if err != nil {
			writeErr(w, 500, "Failed to scan document")
			return
		}
		if description != nil {
			doc.Description = *description
		}

		docs = append(docs, doc)
	}

	if docs == nil {
		docs = []DocumentResponse{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

// GetDocument returns a single document by artifact ID.
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
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

	var doc DocumentResponse
	var contentMD *string
	var sourceDataRaw any
	var storageID, fileFmt, lastExported *string
	var description *string

	err = h.pool.QueryRow(ctx, `
		SELECT a.id::text, a.team_id::text, a.artifact_type, a.title, a.description,
		       a.content_markdown, a.status, a.source_type, a.source_data,
		       a.storage_object_id::text, a.file_format,
		       a.created_by::text, a.updated_by::text,
		       a.created_at::text, a.updated_at::text,
		       d.document_type, d.document_json, d.schema_version, d.word_count,
		       d.last_exported_storage_object_id::text
		FROM artifacts a
		JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.id = $1 AND a.team_id = $2 AND a.artifact_type = 'document'
	`, artifactID, teamID).Scan(
		&doc.ID, &doc.TeamID, &doc.ArtifactType, &doc.Title, &description,
		&contentMD, &doc.Status, &doc.SourceType, &sourceDataRaw,
		&storageID, &fileFmt,
		&doc.CreatedBy, &doc.UpdatedBy,
		&doc.CreatedAt, &doc.UpdatedAt,
		&doc.DocumentType, &doc.DocumentJSON, &doc.SchemaVersion, &doc.WordCount,
		&lastExported,
	)
	if err != nil {
		writeErr(w, 404, "Document not found")
		return
	}
	if description != nil {
		doc.Description = *description
	}

	_ = contentMD
	_ = sourceDataRaw
	_ = storageID
	_ = fileFmt
	_ = lastExported

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// PatchDocument updates a native document's fields and/or document JSON.
func (h *Handler) PatchDocument(w http.ResponseWriter, r *http.Request) {
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

	claims, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "Unauthorized")
		return
	}
	actorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeErr(w, 401, "Invalid actor")
		return
	}

	var req UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	// v1.4 Track 7: Save conflict detection (Option A)
	// Client must send If-Match header with the updated_at value they last saw.
	// If it doesn't match the current DB value, return 409.
	ifMatch := r.Header.Get("If-Match")

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "Failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	// Check artifact exists, is team-scoped, and not archived
	var status, title, description string
	var currentUpdatedAt string
	err = tx.QueryRow(ctx, `
		SELECT status, title, COALESCE(description, ''), updated_at::text
		FROM artifacts WHERE id = $1 AND team_id = $2 AND artifact_type = 'document'
	`, artifactID, teamID).Scan(&status, &title, &description, &currentUpdatedAt)
	if err != nil {
		writeErr(w, 404, "Document not found")
		return
	}
	if status == "archived" {
		writeErr(w, 403, "Archived documents cannot be updated")
		return
	}
	// Save conflict check
	if ifMatch != "" && ifMatch != currentUpdatedAt && ifMatch != `"`+currentUpdatedAt+`"` {
		writeErr(w, 409, "Document was modified by another user. Please refresh and try again.")
		return
	}

	// Apply title/description updates to artifacts table
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			writeErr(w, 400, "title cannot be empty")
			return
		}
		title = *req.Title
	}
	if req.Description != nil {
		description = *req.Description
	}

	var newWordCount int
	var docJSONToStore []byte

	// Determine document_type and document_json to update
	currentDocType := ""
	if req.DocumentType != nil {
		if !validDocumentTypes[*req.DocumentType] {
			writeErr(w, 400, fmt.Sprintf("invalid document_type: %q", *req.DocumentType))
			return
		}
		currentDocType = *req.DocumentType
	}

	if req.DocumentJSON != nil {
		// Validate the new document JSON
		docType := currentDocType
		if docType == "" {
			// Use existing type
			err = tx.QueryRow(ctx, `SELECT document_type FROM artifact_documents WHERE artifact_id = $1`, artifactID).Scan(&docType)
			if err != nil {
				writeErr(w, 500, "Failed to get document type")
				return
			}
		}
		req.DocumentJSON.SchemaVersion = 1
		req.DocumentJSON.DocumentType = docType
		if req.DocumentJSON.Title == "" {
			req.DocumentJSON.Title = title
		}
		if err := validateDocumentJSON(req.DocumentJSON, docType); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		newWordCount = computeWordCount(req.DocumentJSON.Blocks)
		docJSONToStore, err = json.Marshal(req.DocumentJSON)
		if err != nil {
			writeErr(w, 500, "Failed to marshal document JSON")
			return
		}
		if len(docJSONToStore) > maxDocumentJSON {
			writeErr(w, 400, fmt.Sprintf("document JSON exceeds %d bytes", maxDocumentJSON))
			return
		}
	}

	// Update artifacts table
	_, err = tx.Exec(ctx, `
		UPDATE artifacts SET title = $3, description = $4, updated_by = $5, updated_at = NOW()
		WHERE id = $1 AND team_id = $2
	`, artifactID, teamID, title, description, claims.UserID)
	if err != nil {
		writeErr(w, 500, "Failed to update artifact")
		return
	}

	// Update artifact_documents if needed
	if req.DocumentJSON != nil || req.DocumentType != nil {
		if docJSONToStore != nil {
			if currentDocType != "" {
				_, err = tx.Exec(ctx, `
					UPDATE artifact_documents
					SET document_json = $2, word_count = $3, document_type = $4
					WHERE artifact_id = $1
				`, artifactID, docJSONToStore, newWordCount, currentDocType)
			} else {
				_, err = tx.Exec(ctx, `
					UPDATE artifact_documents
					SET document_json = $2, word_count = $3
					WHERE artifact_id = $1
				`, artifactID, docJSONToStore, newWordCount)
			}
		} else if currentDocType != "" {
			_, err = tx.Exec(ctx, `
				UPDATE artifact_documents
				SET document_type = $2
				WHERE artifact_id = $1
			`, artifactID, currentDocType)
		}
		if err != nil {
			writeErr(w, 500, "Failed to update document")
			return
		}
	}

	// Get updated values for response
	var respDocType, respCreatedAt, respUpdatedAt string
	var respDocJSON DocumentJSON
	var respSchemaVersion int
	err = tx.QueryRow(ctx, `
		SELECT d.document_type, d.document_json, d.schema_version, d.word_count,
		       a.created_at::text, a.updated_at::text
		FROM artifacts a
		JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.id = $1
	`, artifactID).Scan(&respDocType, &respDocJSON, &respSchemaVersion, &newWordCount, &respCreatedAt, &respUpdatedAt)
	if err != nil {
		writeErr(w, 500, "Failed to fetch updated document")
		return
	}

	// Audit for patch
	newVal, _ := json.Marshal(map[string]any{
		"title":        title,
		"word_count":  newWordCount,
		"json_updated": req.DocumentJSON != nil,
	})
	audit.Write(ctx, tx, audit.Event{
		Action:     "document.updated",
		EntityType: "artifact",
		EntityID:   artifactID,
		ActorID:    actorID,
		TeamID:     &teamID,
		NewValue:   newVal,
	})

	// v1.4 Track 7: Create version snapshot if document_json was updated
	if req.DocumentJSON != nil {
		source := VersionSourceUserSave
		if r.Header.Get("X-Version-Source") == "agent_assisted_edit" {
			source = VersionSourceAgentAssistedEdit
		}
		versionJSON, _ := json.Marshal(req.DocumentJSON)
		createDocumentVersion(ctx, tx, artifactID, teamID, versionJSON,
			computeWordCount(req.DocumentJSON.Blocks), source, "", &actorID)
	}

	// Outbox event for patch
	patchPayload, _ := json.Marshal(map[string]any{
		"artifact_id": artifactID.String(),
		"team_id":     teamIDStr,
		"title":       title,
		"word_count":  newWordCount,
	})
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.artifact.document.updated",
		AggregateType: "artifact",
		AggregateID:   artifactID.String(),
		Payload:       patchPayload,
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Failed to commit transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DocumentResponse{
		Artifact: Artifact{
			ID:           artifactID.String(),
			TeamID:       teamIDStr,
			ArtifactType: "document",
			Title:        title,
			Description:  description,
			Status:       status,
			SourceType:   "native",
			CreatedBy:    claims.UserID,
			UpdatedBy:    claims.UserID,
			CreatedAt:    respCreatedAt,
			UpdatedAt:    respUpdatedAt,
		},
		DocumentType:  respDocType,
		DocumentJSON:  respDocJSON,
		SchemaVersion: respSchemaVersion,
		WordCount:     newWordCount,
	})

	// v1.5 Knowledge index hook
	h.fireIndexHook(ctx, teamIDStr, "clarity_document", artifactID.String())
}


