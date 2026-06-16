package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─── v1.4 Track 4: Internal Document Generation ───

var validGenerateTones = map[string]bool{
	"technical": true,
	"executive": true,
	"casual":    true,
	"formal":    true,
}

const (
	maxGenerateTitleLen   = 200
	maxGeneratePromptLen  = 2000
	maxGenerateSections   = 20
	maxSectionNameLen     = 100
	generateTimeout       = 90 * time.Second
	maxGenerateBodyBytes  = 200_000
)

// GenerateDocumentRequest is the POST body for generate-document.
type GenerateDocumentRequest struct {
	Title        string   `json:"title"`
	DocumentType string   `json:"document_type"`
	Prompt       string   `json:"prompt"`
	Tone         string   `json:"tone"`
	Sections     []string `json:"sections"`
}

// GenerateDocumentResponse is returned to the frontend.
type GenerateDocumentResponse struct {
	ArtifactID    string       `json:"artifact_id"`
	ArtifactType  string       `json:"artifact_type"`
	DocumentType  string       `json:"document_type"`
	Title         string       `json:"title"`
	Status        string       `json:"status"`
	SchemaVersion int          `json:"schema_version"`
	WordCount     int          `json:"word_count"`
	DocumentJSON  DocumentJSON `json:"document_json"`
}

// workerGeneratePayload is sent to the Python worker /document-generate endpoint.
type workerGeneratePayload struct {
	Title        string   `json:"title"`
	DocumentType string   `json:"document_type"`
	Prompt       string   `json:"prompt"`
	Tone         string   `json:"tone"`
	Sections     []string `json:"sections"`
}

// workerGenerateResponse is what the Python worker returns.
type workerGenerateResponse struct {
	DocumentJSON json.RawMessage `json:"document_json"`
	Summary      string          `json:"summary"`
}

// GenerateDocument handles POST /api/teams/{teamId}/artifacts/generate-document.
func (h *Handler) GenerateDocument(w http.ResponseWriter, r *http.Request) {
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

	// Limit request body
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxGenerateBodyBytes))
	if err != nil {
		writeErr(w, 400, "Failed to read request body")
		return
	}

	var req GenerateDocumentRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	// ─── Input validation ───
	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, 400, "title is required")
		return
	}
	if len(req.Title) > maxGenerateTitleLen {
		writeErr(w, 400, fmt.Sprintf("title exceeds %d chars", maxGenerateTitleLen))
		return
	}

	if !validDocumentTypes[req.DocumentType] {
		writeErr(w, 400, fmt.Sprintf("invalid document_type: %q", req.DocumentType))
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		writeErr(w, 400, "prompt is required")
		return
	}
	if len(req.Prompt) > maxGeneratePromptLen {
		writeErr(w, 400, fmt.Sprintf("prompt exceeds %d chars", maxGeneratePromptLen))
		return
	}

	if req.Tone == "" {
		req.Tone = "technical"
	}
	if !validGenerateTones[req.Tone] {
		writeErr(w, 400, fmt.Sprintf("invalid tone: %q", req.Tone))
		return
	}

	if len(req.Sections) > maxGenerateSections {
		writeErr(w, 400, fmt.Sprintf("sections exceeds maximum of %d", maxGenerateSections))
		return
	}
	for i, s := range req.Sections {
		if strings.TrimSpace(s) == "" {
			writeErr(w, 400, fmt.Sprintf("section %d is empty", i))
			return
		}
		if len(s) > maxSectionNameLen {
			writeErr(w, 400, fmt.Sprintf("section %d exceeds %d chars", i, maxSectionNameLen))
			return
		}
	}

	// ─── Check worker is configured ───
	if h.workerAssist == nil || h.workerAssist.URL == "" {
		writeErr(w, 503, "Document generation is not configured")
		return
	}

	// ─── Call Python worker /document-generate ───
	workerPayload := workerGeneratePayload{
		Title:        req.Title,
		DocumentType: req.DocumentType,
		Prompt:       req.Prompt,
		Tone:         req.Tone,
		Sections:     req.Sections,
	}

	payloadBytes, err := json.Marshal(workerPayload)
	if err != nil {
		writeErr(w, 500, "Failed to marshal generation payload")
		return
	}

	workerCtx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(workerCtx, "POST",
		strings.TrimRight(h.workerAssist.URL, "/")+"/document-generate",
		bytes.NewReader(payloadBytes))
	if err != nil {
		writeErr(w, 500, "Failed to create worker request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.workerAssist.Token)

	client := &http.Client{Timeout: generateTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeErr(w, 504, "Document generation worker timed out or unreachable")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxGenerateBodyBytes))
	if err != nil {
		writeErr(w, 502, "Failed to read worker response")
		return
	}

	if resp.StatusCode != 200 {
		var errBody map[string]any
		if json.Unmarshal(respBody, &errBody) == nil {
			if msg, ok := errBody["error"].(string); ok {
				writeErr(w, 502, fmt.Sprintf("Worker error: %s", msg))
				return
			}
		}
		writeErr(w, 502, "Worker returned an error")
		return
	}

	// ─── Parse and validate worker response ───
	var workerResp workerGenerateResponse
	if err := json.Unmarshal(respBody, &workerResp); err != nil {
		writeErr(w, 502, "Malformed worker response")
		return
	}

	// Parse the returned document_json
	var docJSON DocumentJSON
	if err := json.Unmarshal(workerResp.DocumentJSON, &docJSON); err != nil {
		writeErr(w, 502, "Malformed document_json from worker")
		return
	}

	// Force schema_version = 1 and ensure metadata matches request
	docJSON.SchemaVersion = 1
	docJSON.DocumentType = req.DocumentType
	docJSON.Title = req.Title

	// Validate blocks using Track 1 validation
	if err := validateBlocks(docJSON.Blocks); err != nil {
		writeErr(w, 502, fmt.Sprintf("Worker returned invalid blocks: %s", err.Error()))
		return
	}

	// Recompute word count server-side
	wordCount := computeWordCount(docJSON.Blocks)

	// Marshal final document JSON
	docJSONBytes, err := json.Marshal(docJSON)
	if err != nil {
		writeErr(w, 500, "Failed to marshal document JSON")
		return
	}
	if len(docJSONBytes) > maxDocumentJSON {
		writeErr(w, 502, "Generated document exceeds maximum size")
		return
	}

	// ─── Create artifact + document in transaction ───
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "Failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	artifactID := uuid.New()

	// source_data records metadata but NOT raw prompt
	sourceData, _ := json.Marshal(map[string]any{
		"document_type": req.DocumentType,
		"tone":          req.Tone,
		"sections":      req.Sections,
		"generation":    "worker",
	})

	var createdArtID, createdAt, updatedAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO artifacts (id, team_id, artifact_type, title, description, status, source_type, source_data, created_by, updated_by)
		VALUES ($1, $2, 'document', $3, '', 'draft', 'generated', $4, $5, $5)
		RETURNING id::text, created_at::text, updated_at::text
	`, artifactID, teamID, req.Title, sourceData, actorID).Scan(&createdArtID, &createdAt, &updatedAt)
	if err != nil {
		writeErr(w, 500, "Failed to create artifact")
		return
	}

	var docCreatedAt, docUpdatedAt string
	_, err = tx.Exec(ctx, `
		INSERT INTO artifact_documents (artifact_id, document_type, document_json, schema_version, word_count)
		VALUES ($1, $2, $3, 1, $4)
	`, artifactID, req.DocumentType, docJSONBytes, wordCount)
	if err != nil {
		writeErr(w, 500, "Failed to create document row")
		return
	}
	_ = docCreatedAt
	_ = docUpdatedAt

	// Audit
	teamUUID := teamID
	newVal, _ := json.Marshal(map[string]any{
		"artifact_type":  "document",
		"document_type": req.DocumentType,
		"title":         req.Title,
		"word_count":    wordCount,
		"source_type":   "generated",
		"block_count":   len(docJSON.Blocks),
	})
	audit.Write(ctx, tx, audit.Event{
		Action:     "document.generated",
		EntityType: "artifact",
		EntityID:   artifactID,
		ActorID:    actorID,
		TeamID:     &teamUUID,
		NewValue:   newVal,
	})

	// Outbox event
	outboxPayload, _ := json.Marshal(map[string]any{
		"artifact_id":   createdArtID,
		"team_id":       teamIDStr,
		"document_type": req.DocumentType,
		"title":         req.Title,
		"word_count":    wordCount,
		"source_type":   "generated",
	})
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.artifact.document.generated",
		AggregateType: "artifact",
		AggregateID:   artifactID.String(),
		Payload:       outboxPayload,
	})

	// v1.4 Track 7: Create initial version with source=generated
	createDocumentVersion(ctx, tx, artifactID, teamID, docJSONBytes, wordCount,
		VersionSourceGenerated, "Generated document", &actorID)

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Failed to commit transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(GenerateDocumentResponse{
		ArtifactID:    createdArtID,
		ArtifactType:  "document",
		DocumentType:  req.DocumentType,
		Title:         req.Title,
		Status:        "draft",
		SchemaVersion: 1,
		WordCount:     wordCount,
		DocumentJSON:  docJSON,
	})

	// v1.5 Knowledge index hook
	h.fireIndexHook(ctx, teamIDStr, "clarity_document", createdArtID)
}
