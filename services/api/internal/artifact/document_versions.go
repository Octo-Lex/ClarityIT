package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─── v1.4 Track 7: Document Version History ───

// VersionSource constants
const (
	VersionSourceUserSave          = "user_save"
	VersionSourceAgentAssistedEdit = "agent_assisted_edit"
	VersionSourceGenerated         = "generated"
	VersionSourceTemplate          = "template"
	VersionSourceRestore           = "restore"
)

// createDocumentVersion creates a version snapshot inside an existing transaction.
// version_number is computed atomically via MAX+1.
func createDocumentVersion(ctx context.Context, tx pgx.Tx, artifactID, teamID uuid.UUID,
	docJSON []byte, wordCount int, source, changeSummary string, createdBy *uuid.UUID) (int, error) {

	var versionNum int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_number), 0) + 1
		FROM artifact_document_versions
		WHERE artifact_id = $1
	`, artifactID).Scan(&versionNum)
	if err != nil {
		return 0, fmt.Errorf("failed to compute version number: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO artifact_document_versions (artifact_id, team_id, document_json, version_number, word_count, change_summary, source, created_by)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8)
	`, artifactID, teamID, docJSON, versionNum, wordCount, changeSummary, source, createdBy)
	if err != nil {
		return 0, fmt.Errorf("failed to insert version: %w", err)
	}

	return versionNum, nil
}

// ─── Version Response Types ───

type VersionListItem struct {
	ID            string `json:"id"`
	VersionNumber int    `json:"version_number"`
	WordCount     int    `json:"word_count"`
	Source        string `json:"source"`
	ChangeSummary string `json:"change_summary,omitempty"`
	CreatedBy     string `json:"created_by,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type VersionDetail struct {
	ID            string          `json:"id"`
	VersionNumber int             `json:"version_number"`
	DocumentJSON  json.RawMessage `json:"document_json"`
	WordCount     int             `json:"word_count"`
	Source        string          `json:"source"`
	ChangeSummary string          `json:"change_summary,omitempty"`
	CreatedAt     string          `json:"created_at"`
}

type RestoreResponse struct {
	ArtifactID          string          `json:"artifact_id"`
	RestoredFromVersion int             `json:"restored_from_version"`
	NewVersionNumber    int             `json:"new_version_number"`
	DocumentJSON        json.RawMessage `json:"document_json"`
	WordCount           int             `json:"word_count"`
}

// ListVersions returns all versions of a document in DESC order.
func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
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

	// Verify artifact exists, team-scoped, is a document
	var artifactType string
	err = h.pool.QueryRow(ctx, `
		SELECT artifact_type FROM artifacts WHERE id = $1 AND team_id = $2
	`, artifactID, teamID).Scan(&artifactType)
	if err != nil {
		writeErr(w, 404, "Artifact not found")
		return
	}
	if artifactType != "document" {
		writeErr(w, 400, "Versions are only available for documents")
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT id::text, version_number, word_count, source,
		       COALESCE(change_summary, ''), COALESCE(created_by::text, ''), created_at::text
		FROM artifact_document_versions
		WHERE artifact_id = $1 AND team_id = $2
		ORDER BY version_number DESC
	`, artifactID, teamID)
	if err != nil {
		writeErr(w, 500, "Failed to query versions")
		return
	}
	defer rows.Close()

	versions := []VersionListItem{}
	for rows.Next() {
		var v VersionListItem
		if err := rows.Scan(&v.ID, &v.VersionNumber, &v.WordCount, &v.Source,
			&v.ChangeSummary, &v.CreatedBy, &v.CreatedAt); err != nil {
			writeErr(w, 500, "Failed to scan version")
			return
		}
		versions = append(versions, v)
	}

	writeJSON(w, 200, map[string]any{"versions": versions})
}

// GetVersion returns a specific version with full document_json.
func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
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
	versionID, err := uuid.Parse(chi.URLParam(r, "versionId"))
	if err != nil {
		writeErr(w, 400, "Invalid version ID")
		return
	}

	var v VersionDetail
	err = h.pool.QueryRow(ctx, `
		SELECT id::text, version_number, document_json, word_count,
		       source, COALESCE(change_summary, ''), created_at::text
		FROM artifact_document_versions
		WHERE id = $1 AND artifact_id = $2 AND team_id = $3
	`, versionID, artifactID, teamID).Scan(
		&v.ID, &v.VersionNumber, &v.DocumentJSON, &v.WordCount,
		&v.Source, &v.ChangeSummary, &v.CreatedAt,
	)
	if err != nil {
		writeErr(w, 404, "Version not found")
		return
	}

	writeJSON(w, 200, v)
}

// RestoreVersion restores a document to a previous version.
// Non-destructive: creates a new version with the old content.
func (h *Handler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
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
	versionID, err := uuid.Parse(chi.URLParam(r, "versionId"))
	if err != nil {
		writeErr(w, 400, "Invalid version ID")
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

	// Fetch the version to restore (team + artifact scoped)
	var versionNum int
	var versionJSON []byte
	var versionWordCount int
	err = h.pool.QueryRow(ctx, `
		SELECT version_number, document_json, word_count
		FROM artifact_document_versions
		WHERE id = $1 AND artifact_id = $2 AND team_id = $3
	`, versionID, artifactID, teamID).Scan(&versionNum, &versionJSON, &versionWordCount)
	if err != nil {
		writeErr(w, 404, "Version not found")
		return
	}

	// Validate stored document_json before applying
	var docJSON DocumentJSON
	if err := json.Unmarshal(versionJSON, &docJSON); err != nil {
		writeErr(w, 500, "Stored document_json is malformed")
		return
	}
	if err := validateDocumentJSON(&docJSON, docJSON.DocumentType); err != nil {
		writeErr(w, 500, fmt.Sprintf("Stored document_json is invalid: %s", err.Error()))
		return
	}

	// Recompute word count server-side (don't trust stored value)
	recomputedWC := computeWordCount(docJSON.Blocks)

	// Marshal final JSON
	finalJSON, err := json.Marshal(docJSON)
	if err != nil {
		writeErr(w, 500, "Failed to marshal document JSON")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeErr(w, 500, "Failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	// Check artifact status (not archived)
	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM artifacts WHERE id = $1 AND team_id = $2
	`, artifactID, teamID).Scan(&status)
	if err != nil {
		writeErr(w, 404, "Artifact not found")
		return
	}
	if status == "archived" {
		writeErr(w, 403, "Archived documents cannot be restored")
		return
	}

	// Update artifact_documents to restored content
	_, err = tx.Exec(ctx, `
		UPDATE artifact_documents
		SET document_json = $2, word_count = $3, updated_at = NOW()
		WHERE artifact_id = $1
	`, artifactID, finalJSON, recomputedWC)
	if err != nil {
		writeErr(w, 500, "Failed to restore document content")
		return
	}

	// Update artifacts updated_at and updated_by
	_, err = tx.Exec(ctx, `
		UPDATE artifacts SET updated_at = NOW(), updated_by = $3
		WHERE id = $1 AND team_id = $2
	`, artifactID, teamID, actorID)
	if err != nil {
		writeErr(w, 500, "Failed to update artifact")
		return
	}

	// Create new version row with source=restore (non-destructive)
	changeSummary := fmt.Sprintf("Restored from version %d", versionNum)
	newVersionNum, err := createDocumentVersion(ctx, tx, artifactID, teamID,
		finalJSON, recomputedWC, VersionSourceRestore, changeSummary, &actorID)
	if err != nil {
		writeErr(w, 500, "Failed to create restore version")
		return
	}

	// Audit
	teamUUID := teamID
	auditVal, _ := json.Marshal(map[string]any{
		"restored_from_version": versionNum,
		"new_version_number":    newVersionNum,
		"word_count":            recomputedWC,
	})
	audit.Write(ctx, tx, audit.Event{
		Action:     "document.restored",
		EntityType: "artifact",
		EntityID:   artifactID,
		ActorID:    actorID,
		TeamID:     &teamUUID,
		NewValue:   auditVal,
	})

	// Outbox
	restorePayload, _ := json.Marshal(map[string]any{
		"artifact_id":           artifactID.String(),
		"team_id":               teamIDStr,
		"restored_from_version": versionNum,
		"new_version_number":    newVersionNum,
	})
	outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
		EventType:     "clarity.v1.artifact.document.restored",
		AggregateType: "artifact",
		AggregateID:   artifactID.String(),
		Payload:       restorePayload,
	})

	if err := tx.Commit(ctx); err != nil {
		writeErr(w, 500, "Failed to commit restore")
		return
	}

	writeJSON(w, 200, RestoreResponse{
		ArtifactID:          artifactID.String(),
		RestoredFromVersion: versionNum,
		NewVersionNumber:    newVersionNum,
		DocumentJSON:        finalJSON,
		WordCount:           recomputedWC,
	})
}
