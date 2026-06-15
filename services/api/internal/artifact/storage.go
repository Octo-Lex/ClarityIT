package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─── Track 6: Artifact Storage and Recent Files ───

const maxSearchQueryLen = 200
const recentLimit = 20

// FileMetadata holds safe file info for include_files responses
type FileMetadata struct {
	FileSize          *int64 `json:"file_size"`
	FileFormat        *string `json:"file_format"`
	DownloadAvailable bool   `json:"download_available"`
	StorageObjectID   *string `json:"storage_object_id"`
}

// listWithFiles returns artifacts with safe file metadata joined from storage_objects.
// No bucket names, object keys, or filesystem paths are exposed.
func (h *Handler) listWithFiles(ctx context.Context, w http.ResponseWriter,
	teamID uuid.UUID, teamIDStr, artifactType, status, search string, includeArchived bool) {

	var query string
	var args []any
	argIdx := 1

	args = append(args, teamID)

	selectCols := `a.id::text, a.artifact_type, a.title, a.description, a.content_markdown,
		a.status, a.source_type, a.source_data,
		a.storage_object_id::text, a.file_format,
		a.created_by::text, a.updated_by::text,
		a.created_at::text, a.updated_at::text, s.size_bytes`

	query = fmt.Sprintf(`SELECT %s FROM artifacts a LEFT JOIN storage_objects s ON a.storage_object_id = s.id WHERE a.team_id = $1`, selectCols)

	if !includeArchived {
		if status == "" {
			query += ` AND a.status != 'archived'`
		}
	}

	if artifactType != "" {
		if !validTypes[artifactType] {
			writeErr(w, 400, "invalid type filter")
			return
		}
		argIdx++
		query += fmt.Sprintf(" AND a.artifact_type = $%d", argIdx)
		args = append(args, artifactType)
	}

	if status != "" {
		if !validStatuses[status] {
			writeErr(w, 400, "invalid status filter")
			return
		}
		argIdx++
		query += fmt.Sprintf(" AND a.status = $%d", argIdx)
		args = append(args, status)
	}

	if search != "" {
		argIdx++
		query += fmt.Sprintf(" AND a.title ILIKE $%d", argIdx)
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY a.updated_at DESC LIMIT 100"

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, 500, "Failed to query artifacts")
		return
	}
	defer rows.Close()

	results := []map[string]any{}
	for rows.Next() {
		var id, artType, title, desc, content, stat, srcType, createdAt, updatedAt, createdBy, updatedBy string
		var sourceData []byte
		var storageObjID, fileFmt *string
		var sizeBytes *int64

		err := rows.Scan(&id, &artType, &title, &desc, &content,
			&stat, &srcType, &sourceData,
			&storageObjID, &fileFmt, &createdBy, &updatedBy,
			&createdAt, &updatedAt, &sizeBytes)
		if err != nil {
			continue
		}

		var sd any
		json.Unmarshal(sourceData, &sd)

		row := map[string]any{
			"id":               id,
			"artifact_type":    artType,
			"title":            title,
			"description":      desc,
			"content_markdown": content,
			"status":           stat,
			"source_type":      srcType,
			"source_data":      sanitizeSourceData(sd),
			"created_by":       createdBy,
			"updated_by":       updatedBy,
			"created_at":       createdAt,
			"updated_at":       updatedAt,
		}

		if storageObjID != nil && *storageObjID != "" {
			row["file_metadata"] = FileMetadata{
				FileSize:          sizeBytes,
				FileFormat:        fileFmt,
				DownloadAvailable: false,
				StorageObjectID:   storageObjID,
			}
		}
		results = append(results, row)
	}

	writeJSON(w, 200, results)
}

// Recent returns last 20 modified artifacts, excluding archived by default.
func (h *Handler) Recent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"

	query := `
		SELECT id::text, artifact_type, title, description, content_markdown,
		       status, source_type, source_data,
		       storage_object_id::text, file_format,
		       created_by::text, updated_by::text,
		       created_at::text, updated_at::text
		FROM artifacts WHERE team_id = $1
	`
	args := []any{teamID}

	if !includeArchived {
		query += ` AND status != 'archived'`
	}
	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT %d", recentLimit)

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		writeErr(w, 500, "Failed to query recent artifacts")
		return
	}
	defer rows.Close()

	artifacts := []Artifact{}
	for rows.Next() {
		art, err := scanArtifact(rows)
		if err != nil {
			continue
		}
		art.TeamID = teamIDStr
		artifacts = append(artifacts, art)
	}

	writeJSON(w, 200, artifacts)
}

// Search searches title, description, and content_markdown.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, 200, []Artifact{})
		return
	}
	if len(query) > maxSearchQueryLen {
		writeErr(w, 400, fmt.Sprintf("search query exceeds %d character limit", maxSearchQueryLen))
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"
	pattern := "%" + query + "%"

	sqlQuery := `
		SELECT id::text, artifact_type, title, description, content_markdown,
		       status, source_type, source_data,
		       storage_object_id::text, file_format,
		       created_by::text, updated_by::text,
		       created_at::text, updated_at::text
		FROM artifacts WHERE team_id = $1
		  AND (title ILIKE $2 OR description ILIKE $2 OR content_markdown ILIKE $2)
	`
	args := []any{teamID, pattern}

	if !includeArchived {
		sqlQuery += ` AND status != 'archived'`
	}
	sqlQuery += " ORDER BY updated_at DESC LIMIT 50"

	rows, err := h.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		writeErr(w, 500, "Failed to search artifacts")
		return
	}
	defer rows.Close()

	artifacts := []Artifact{}
	for rows.Next() {
		art, err := scanArtifact(rows)
		if err != nil {
			continue
		}
		art.TeamID = teamIDStr
		artifacts = append(artifacts, art)
	}

	writeJSON(w, 200, artifacts)
}

// StorageSummary returns aggregate storage statistics for the team.
func (h *Handler) StorageSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"

	archivedClause := ""
	if !includeArchived {
		archivedClause = " AND a.status != 'archived'"
	}

	// Total artifacts
	var totalArtifacts int
	err = h.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM artifacts a WHERE a.team_id = $1%s", archivedClause),
		teamID).Scan(&totalArtifacts)
	if err != nil {
		writeErr(w, 500, "Failed to compute storage summary")
		return
	}

	// File artifacts (with storage_object_id)
	var fileArtifacts int
	err = h.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM artifacts a WHERE a.team_id = $1 AND a.storage_object_id IS NOT NULL%s", archivedClause),
		teamID).Scan(&fileArtifacts)
	if err != nil {
		writeErr(w, 500, "Failed to compute storage summary")
		return
	}

	// Inline artifacts (without storage_object_id)
	inlineArtifacts := totalArtifacts - fileArtifacts

	// Total file size
	var totalFileSize *int64
	err = h.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT COALESCE(SUM(s.size_bytes), 0) FROM artifacts a
			LEFT JOIN storage_objects s ON a.storage_object_id = s.id
			WHERE a.team_id = $1 AND a.storage_object_id IS NOT NULL%s`, archivedClause),
		teamID).Scan(&totalFileSize)
	if err != nil {
		writeErr(w, 500, "Failed to compute storage summary")
		return
	}

	totalFileSizeVal := int64(0)
	if totalFileSize != nil {
		totalFileSizeVal = *totalFileSize
	}

	// By format
	rows, err := h.pool.Query(ctx,
		fmt.Sprintf(`SELECT COALESCE(a.file_format, 'md') as fmt, COUNT(*) as cnt
			FROM artifacts a
			WHERE a.team_id = $1 AND a.storage_object_id IS NOT NULL%s
			GROUP BY COALESCE(a.file_format, 'md')`, archivedClause),
		teamID)
	if err != nil {
		writeErr(w, 500, "Failed to compute storage summary")
		return
	}
	defer rows.Close()

	byFormat := map[string]int{}
	for rows.Next() {
		var format string
		var count int
		if err := rows.Scan(&format, &count); err != nil {
			continue
		}
		byFormat[format] = count
	}

	writeJSON(w, 200, map[string]any{
		"total_artifacts":       totalArtifacts,
		"file_artifacts":        fileArtifacts,
		"inline_artifacts":      inlineArtifacts,
		"total_file_size_bytes": totalFileSizeVal,
		"by_format":             byFormat,
	})
}

// Suppress unused import warning
var _ = context.Background
