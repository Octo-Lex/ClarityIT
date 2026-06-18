package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// ─── Types ───

type Collection struct {
	ID          string  `json:"id"`
	TeamID      string  `json:"team_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CreatedBy   *string `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	ArchivedAt  *string `json:"archived_at,omitempty"`
	ItemCount   int     `json:"item_count"`
}

type CollectionItem struct {
	ID              string  `json:"id"`
	CollectionID    string  `json:"collection_id"`
	TeamID          string  `json:"team_id"`
	SourceType      string  `json:"source_type"`
	SourceID        string  `json:"source_id"`
	KnowledgeItemID *string `json:"knowledge_item_id,omitempty"`
	Title           string  `json:"title,omitempty"`
	Summary         string  `json:"summary,omitempty"`
	Note            *string `json:"note,omitempty"`
	AddedBy         *string `json:"added_by"`
	AddedAt         string  `json:"added_at"`
}

type CollectionDetail struct {
	Collection
	Items []CollectionItem `json:"items"`
}

type SavedAnswer struct {
	ID           string          `json:"id"`
	TeamID       string          `json:"team_id"`
	CollectionID *string         `json:"collection_id,omitempty"`
	Question     string          `json:"question"`
	Answer       string          `json:"answer"`
	Confidence   string          `json:"confidence"`
	Sources      json.RawMessage `json:"sources"`
	CreatedBy    *string         `json:"created_by"`
	CreatedAt    string          `json:"created_at"`
}

// ─── Collections CRUD ───

// GET /api/teams/{teamId}/knowledge/collections
func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	includeArchived := r.URL.Query().Get("include_archived") == "true"

	var rows pgx.Rows
	var err error
	if includeArchived {
		rows, err = h.pool.Query(r.Context(), `
			SELECT c.id::text, c.team_id::text, c.name, COALESCE(c.description, ''),
			       c.created_by::text, c.created_at::text, c.updated_at::text,
			       c.archived_at::text,
			       (SELECT count(*) FROM knowledge_collection_items i WHERE i.collection_id = c.id) AS item_count
			FROM knowledge_collections c
			WHERE c.team_id = $1::uuid
			ORDER BY c.archived_at NULLS FIRST, c.updated_at DESC
		`, teamID)
	} else {
		rows, err = h.pool.Query(r.Context(), `
			SELECT c.id::text, c.team_id::text, c.name, COALESCE(c.description, ''),
			       c.created_by::text, c.created_at::text, c.updated_at::text,
			       c.archived_at::text,
			       (SELECT count(*) FROM knowledge_collection_items i WHERE i.collection_id = c.id) AS item_count
			FROM knowledge_collections c
			WHERE c.team_id = $1::uuid AND c.archived_at IS NULL
			ORDER BY c.updated_at DESC
		`, teamID)
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to list collections"})
		return
	}
	defer rows.Close()

	collections := []Collection{}
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.TeamID, &c.Name, &c.Description,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.ArchivedAt, &c.ItemCount); err != nil {
			writeJSON(w, 500, map[string]string{"error": "Failed to scan collection"})
			return
		}
		collections = append(collections, c)
	}
	writeJSON(w, 200, map[string]any{"collections": collections})
}

// POST /api/teams/{teamId}/knowledge/collections
func (h *Handler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	userID := r.Context().Value("user_id").(string)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if len(name) == 0 || len(name) > 200 {
		writeJSON(w, 400, map[string]string{"error": "Collection name must be 1-200 characters"})
		return
	}

	var c Collection
	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO knowledge_collections (team_id, name, description, created_by)
		VALUES ($1::uuid, $2, $3, $4::uuid)
		RETURNING id::text, team_id::text, name, COALESCE(description, ''),
		          created_by::text, created_at::text, updated_at::text, archived_at::text
	`, teamID, name, desc, userID).Scan(
		&c.ID, &c.TeamID, &c.Name, &c.Description,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.ArchivedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uq_active_collection_name_team") {
			writeJSON(w, 409, map[string]string{"error": "A collection with this name already exists"})
			return
		}
		writeJSON(w, 500, map[string]string{"error": "Failed to create collection"})
		return
	}
	writeJSON(w, 201, c)
}

// GET /api/teams/{teamId}/knowledge/collections/{collectionId}
func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	collectionID := chi.URLParam(r, "collectionId")

	var c Collection
	var archivedAt *string
	err := h.pool.QueryRow(r.Context(), `
		SELECT c.id::text, c.team_id::text, c.name, COALESCE(c.description, ''),
		       c.created_by::text, c.created_at::text, c.updated_at::text,
		       c.archived_at::text,
		       (SELECT count(*) FROM knowledge_collection_items i WHERE i.collection_id = c.id)
		FROM knowledge_collections c
		WHERE c.id = $1::uuid AND c.team_id = $2::uuid
	`, collectionID, teamID).Scan(
		&c.ID, &c.TeamID, &c.Name, &c.Description,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &archivedAt, &c.ItemCount,
	)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "Collection not found"})
		return
	}
	c.ArchivedAt = archivedAt

	// Get items
	items, err := h.getCollectionItems(r.Context(), collectionID, teamID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to load items"})
		return
	}
	if items == nil {
		items = []CollectionItem{}
	}

	writeJSON(w, 200, CollectionDetail{Collection: c, Items: items})
}

// PATCH /api/teams/{teamId}/knowledge/collections/{collectionId}
func (h *Handler) PatchCollection(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	collectionID := chi.URLParam(r, "collectionId")

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	// Build dynamic update
	var setParts []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if len(name) == 0 || len(name) > 200 {
			writeJSON(w, 400, map[string]string{"error": "Collection name must be 1-200 characters"})
			return
		}
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, name)
		argIdx++
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if len(desc) > 2000 {
			writeJSON(w, 400, map[string]string{"error": "Description must be at most 2000 characters"})
			return
		}
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, desc)
		argIdx++
	}

	if len(setParts) == 0 {
		writeJSON(w, 400, map[string]string{"error": "No fields to update"})
		return
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, "NOW()")
	argIdx++

	// WHERE clause args
	args = append(args, collectionID, teamID)

	query := fmt.Sprintf(`
		UPDATE knowledge_collections SET %s
		WHERE id = $%d::uuid AND team_id = $%d::uuid AND archived_at IS NULL
		RETURNING id::text, team_id::text, name, COALESCE(description, ''),
		          created_by::text, created_at::text, updated_at::text, archived_at::text
	`, strings.Join(setParts, ", "), argIdx, argIdx+1)

	var c Collection
	var archivedAt *string
	err := h.pool.QueryRow(r.Context(), query, args...).Scan(
		&c.ID, &c.TeamID, &c.Name, &c.Description,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &archivedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uq_active_collection_name_team") {
			writeJSON(w, 409, map[string]string{"error": "A collection with this name already exists"})
			return
		}
		writeJSON(w, 404, map[string]string{"error": "Collection not found or archived"})
		return
	}
	c.ArchivedAt = archivedAt
	writeJSON(w, 200, c)
}

// DELETE /api/teams/{teamId}/knowledge/collections/{collectionId}
func (h *Handler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	collectionID := chi.URLParam(r, "collectionId")

	// Soft delete — set archived_at
	var archivedAtStr string
	err := h.pool.QueryRow(r.Context(), `
		UPDATE knowledge_collections
		SET archived_at = NOW(), updated_at = NOW()
		WHERE id = $1::uuid AND team_id = $2::uuid AND archived_at IS NULL
		RETURNING archived_at::text
	`, collectionID, teamID).Scan(&archivedAtStr)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "Collection not found or already archived"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "archived"})
}

// ─── Collection Items ───

// POST /api/teams/{teamId}/knowledge/collections/{collectionId}/items
func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	collectionID := chi.URLParam(r, "collectionId")
	userID := r.Context().Value("user_id").(string)

	var req struct {
		SourceType      string  `json:"source_type"`
		SourceID        string  `json:"source_id"`
		KnowledgeItemID *string `json:"knowledge_item_id"`
		Note            *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	if !ValidateSourceType(req.SourceType) {
		writeJSON(w, 400, map[string]string{"error": "Invalid source type"})
		return
	}

	sourceID := strings.TrimSpace(req.SourceID)
	if len(sourceID) == 0 || len(sourceID) > 255 {
		writeJSON(w, 400, map[string]string{"error": "Source ID must be 1-255 characters"})
		return
	}

	// Validate note bounds
	if req.Note != nil && len(*req.Note) > 1000 {
		writeJSON(w, 400, map[string]string{"error": "Note must be at most 1000 characters"})
		return
	}

	// Verify collection exists and belongs to team, not archived
	var exists bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM knowledge_collections
			WHERE id = $1::uuid AND team_id = $2::uuid AND archived_at IS NULL
		)
	`, collectionID, teamID).Scan(&exists)
	if err != nil || !exists {
		writeJSON(w, 404, map[string]string{"error": "Collection not found"})
		return
	}

	// If knowledge_item_id supplied, verify it belongs to same team
	if req.KnowledgeItemID != nil {
		var kiExists bool
		err := h.pool.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM knowledge_items
				WHERE id = $1::uuid AND team_id = $2::uuid
			)
		`, *req.KnowledgeItemID, teamID).Scan(&kiExists)
		if err != nil || !kiExists {
			writeJSON(w, 400, map[string]string{"error": "Knowledge item not found in this team"})
			return
		}
	}

	// Insert item — handle duplicate idempotently (ON CONFLICT DO NOTHING, return existing)
	var item CollectionItem
	var kiID *string
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO knowledge_collection_items (collection_id, team_id, source_type, source_id, knowledge_item_id, note, added_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, $6, $7::uuid)
		ON CONFLICT (collection_id, source_type, source_id) DO NOTHING
		RETURNING id::text, collection_id::text, team_id::text, source_type, source_id,
		          knowledge_item_id::text, note, added_by::text, added_at::text
	`, collectionID, teamID, req.SourceType, sourceID, req.KnowledgeItemID, req.Note, userID).Scan(
		&item.ID, &item.CollectionID, &item.TeamID, &item.SourceType, &item.SourceID,
		&kiID, &item.Note, &item.AddedBy, &item.AddedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Duplicate — fetch existing item
			err = h.pool.QueryRow(r.Context(), `
				SELECT id::text, collection_id::text, team_id::text, source_type, source_id,
				       knowledge_item_id::text, note, added_by::text, added_at::text
				FROM knowledge_collection_items
				WHERE collection_id = $1::uuid AND source_type = $2 AND source_id = $3
			`, collectionID, req.SourceType, sourceID).Scan(
				&item.ID, &item.CollectionID, &item.TeamID, &item.SourceType, &item.SourceID,
				&kiID, &item.Note, &item.AddedBy, &item.AddedAt,
			)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "Failed to fetch existing item"})
				return
			}
			// Return 200 with a flag that it was a duplicate
			writeJSON(w, 200, map[string]any{"item": item, "duplicate": true})
			return
		}
		writeJSON(w, 500, map[string]string{"error": "Failed to add item"})
		return
	}
	item.KnowledgeItemID = kiID
	writeJSON(w, 201, map[string]any{"item": item})
}

// DELETE /api/teams/{teamId}/knowledge/collections/{collectionId}/items/{itemId}
func (h *Handler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	collectionID := chi.URLParam(r, "collectionId")
	itemID := chi.URLParam(r, "itemId")

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM knowledge_collection_items
		WHERE id = $1::uuid AND collection_id = $2::uuid AND team_id = $3::uuid
	`, itemID, collectionID, teamID)
	_ = tag
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to remove item"})
		return
	}

	// Check with a separate count query since tag rows affected may be unreliable
	var count int
	h.pool.QueryRow(r.Context(), `
		SELECT count(*) FROM knowledge_collection_items
		WHERE id = $1::uuid AND team_id = $2::uuid
	`, collectionID, teamID).Scan(&count)

	writeJSON(w, 200, map[string]string{"status": "removed"})
}

// ─── Saved Answers ───

// POST /api/teams/{teamId}/knowledge/saved-answers
func (h *Handler) SaveAnswer(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	userID := r.Context().Value("user_id").(string)

	var req struct {
		Question     string          `json:"question"`
		Answer       string          `json:"answer"`
		Confidence   string          `json:"confidence"`
		Sources      json.RawMessage `json:"sources"`
		CollectionID *string         `json:"collection_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate
	question := strings.TrimSpace(req.Question)
	if len(question) == 0 || len(question) > 1000 {
		writeJSON(w, 400, map[string]string{"error": "Question must be 1-1000 characters"})
		return
	}

	answer := strings.TrimSpace(req.Answer)
	if len(answer) == 0 || len(answer) > 50000 {
		writeJSON(w, 400, map[string]string{"error": "Answer must be 1-50000 characters"})
		return
	}

	if req.Confidence != "low" && req.Confidence != "medium" && req.Confidence != "high" {
		writeJSON(w, 400, map[string]string{"error": "Confidence must be low, medium, or high"})
		return
	}

	// Strip forbidden fields from sources JSON
	sourcesBytes := stripForbiddenFromJSON(req.Sources)

	// Validate collection_id if supplied
	if req.CollectionID != nil {
		var exists bool
		err := h.pool.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM knowledge_collections
				WHERE id = $1::uuid AND team_id = $2::uuid AND archived_at IS NULL
			)
		`, *req.CollectionID, teamID).Scan(&exists)
		if err != nil || !exists {
			writeJSON(w, 400, map[string]string{"error": "Collection not found"})
			return
		}
	}

	var sa SavedAnswer
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO saved_knowledge_answers (team_id, collection_id, question, answer, confidence, sources, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, $7::uuid)
		RETURNING id::text, team_id::text, collection_id::text, question, answer, confidence,
		          sources, created_by::text, created_at::text
	`, teamID, req.CollectionID, question, answer, req.Confidence, sourcesBytes, userID).Scan(
		&sa.ID, &sa.TeamID, &sa.CollectionID, &sa.Question, &sa.Answer,
		&sa.Confidence, &sa.Sources, &sa.CreatedBy, &sa.CreatedAt,
	)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to save answer"})
		return
	}
	writeJSON(w, 201, sa)
}

// GET /api/teams/{teamId}/knowledge/saved-answers
func (h *Handler) ListSavedAnswers(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	if limit > 100 {
		limit = 100
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id::text, team_id::text, collection_id::text, question, answer, confidence,
		       sources, created_by::text, created_at::text
		FROM saved_knowledge_answers
		WHERE team_id = $1::uuid
		ORDER BY created_at DESC
		LIMIT $2
	`, teamID, limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to list saved answers"})
		return
	}
	defer rows.Close()

	answers := []SavedAnswer{}
	for rows.Next() {
		var sa SavedAnswer
		if err := rows.Scan(&sa.ID, &sa.TeamID, &sa.CollectionID, &sa.Question,
			&sa.Answer, &sa.Confidence, &sa.Sources, &sa.CreatedBy, &sa.CreatedAt); err != nil {
			writeJSON(w, 500, map[string]string{"error": "Failed to scan saved answer"})
			return
		}
		answers = append(answers, sa)
	}
	writeJSON(w, 200, map[string]any{"answers": answers})
}

// GET /api/teams/{teamId}/knowledge/saved-answers/{answerId}
func (h *Handler) GetSavedAnswer(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	answerID := chi.URLParam(r, "answerId")

	var sa SavedAnswer
	err := h.pool.QueryRow(r.Context(), `
		SELECT id::text, team_id::text, collection_id::text, question, answer, confidence,
		       sources, created_by::text, created_at::text
		FROM saved_knowledge_answers
		WHERE id = $1::uuid AND team_id = $2::uuid
	`, answerID, teamID).Scan(
		&sa.ID, &sa.TeamID, &sa.CollectionID, &sa.Question,
		&sa.Answer, &sa.Confidence, &sa.Sources, &sa.CreatedBy, &sa.CreatedAt,
	)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "Saved answer not found"})
		return
	}
	writeJSON(w, 200, sa)
}

// DELETE /api/teams/{teamId}/knowledge/saved-answers/{answerId}
func (h *Handler) DeleteSavedAnswer(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	answerID := chi.URLParam(r, "answerId")

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM saved_knowledge_answers
		WHERE id = $1::uuid AND team_id = $2::uuid
	`, answerID, teamID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to delete saved answer"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, 404, map[string]string{"error": "Saved answer not found"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// ─── Helpers ───

func (h *Handler) getCollectionItems(ctx context.Context, collectionID, teamID string) ([]CollectionItem, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT i.id::text, i.collection_id::text, i.team_id::text, i.source_type, i.source_id,
		       i.knowledge_item_id::text,
		       COALESCE(ki.title, ''), COALESCE(ki.summary, ''),
		       i.note, i.added_by::text, i.added_at::text
		FROM knowledge_collection_items i
		LEFT JOIN knowledge_items ki ON ki.id = i.knowledge_item_id
		WHERE i.collection_id = $1::uuid AND i.team_id = $2::uuid
		ORDER BY i.added_at DESC
	`, collectionID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CollectionItem{}
	for rows.Next() {
		var item CollectionItem
		var kiID *string
		if err := rows.Scan(&item.ID, &item.CollectionID, &item.TeamID,
			&item.SourceType, &item.SourceID, &kiID,
			&item.Title, &item.Summary, &item.Note, &item.AddedBy, &item.AddedAt); err != nil {
			return nil, err
		}
		item.KnowledgeItemID = kiID
		items = append(items, item)
	}
	return items, nil
}

// forbiddenAnswerFields are fields that must never appear in saved answer sources
var forbiddenAnswerFields = []string{
	"chain_of_thought", "thinking", "internal_reasoning",
	"tool_calls", "action", "mutation", "execute",
	"prompt", "raw_prompt",
}

// stripForbiddenFromJSON removes known forbidden keys from a JSON object or array
func stripForbiddenFromJSON(raw json.RawMessage) []byte {
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return []byte("[]")
	}

	switch v := parsed.(type) {
	case []any:
		for i, item := range v {
			if m, ok := item.(map[string]any); ok {
				v[i] = stripMapKeys(m)
			}
		}
		result, _ := json.Marshal(v)
		return result
	case map[string]any:
		stripped := stripMapKeys(v)
		result, _ := json.Marshal(stripped)
		return result
	default:
		return []byte("[]")
	}
}

func stripMapKeys(m map[string]any) map[string]any {
	for _, field := range forbiddenAnswerFields {
		delete(m, field)
	}
	return m
}
