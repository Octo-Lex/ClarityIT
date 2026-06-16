package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ───

type KnowledgeItem struct {
	ID              string    `json:"id"`
	TeamID          string    `json:"team_id"`
	SourceType      string    `json:"source_type"`
	SourceID        string    `json:"source_id"`
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
	ContentText     string    `json:"content_text,omitempty"`
	ContentHash     *string   `json:"content_hash,omitempty"`
	Metadata        any       `json:"metadata"`
	Visibility      string     `json:"visibility"`
	IndexedAt       time.Time  `json:"indexed_at"`
	SourceUpdatedAt *time.Time `json:"source_updated_at,omitempty"`
	StaleAfter      *time.Time `json:"stale_after,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SearchResult struct {
	SourceType string  `json:"source_type"`
	SourceID   string  `json:"source_id"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
	Rank       float64 `json:"rank"`
	UpdatedAt  string  `json:"updated_at"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
}

type IndexRequest struct {
	SourceType      string `json:"source_type"`
	SourceID        string `json:"source_id"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	ContentText     string `json:"content_text"`
	Metadata        map[string]any `json:"metadata"`
	SourceUpdatedAt string `json:"source_updated_at"`
}

type IndexStatus struct {
	TotalItems  int            `json:"total_items"`
	ByType      map[string]int `json:"by_type"`
	StaleCount  int            `json:"stale_count"`
	LastIndexed *time.Time     `json:"last_indexed,omitempty"`
}

// ─── Handler ───

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// ─── Index (upsert a knowledge item) ───

func (h *Handler) Index(ctx context.Context, teamID string, req IndexRequest) (*KnowledgeItem, error) {
	tid, err := uuid.Parse(teamID)
	if err != nil {
		return nil, fmt.Errorf("invalid team ID")
	}
	sid, err := uuid.Parse(req.SourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid source ID")
	}

	metaJSON, _ := json.Marshal(req.Metadata)
	if req.Metadata == nil {
		metaJSON = []byte("{}")
	}

	var sourceUpdated *time.Time
	if req.SourceUpdatedAt != "" {
		t, err := time.Parse(time.RFC3339, req.SourceUpdatedAt)
		if err == nil {
			sourceUpdated = &t
		}
	}

	// Compute stale_after: 90 days from now
	staleAfter := time.Now().Add(90 * 24 * time.Hour)

	var item KnowledgeItem
	var metaBytes []byte
	err = h.pool.QueryRow(ctx, `
		INSERT INTO knowledge_items (team_id, source_type, source_id, title, summary, content_text, metadata, source_updated_at, stale_after, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, NOW())
		ON CONFLICT (team_id, source_type, source_id) DO UPDATE
		SET title = $4, summary = $5, content_text = $6, metadata = $7::jsonb,
		    source_updated_at = $8, stale_after = $9, indexed_at = NOW()
		RETURNING id, team_id::text, source_type, source_id::text, title, summary,
		          content_text, content_hash, metadata, visibility,
		          indexed_at, source_updated_at, stale_after, created_at, updated_at
	`, tid, req.SourceType, sid, req.Title, req.Summary, req.ContentText, metaJSON, sourceUpdated, staleAfter).Scan(
		&item.ID, &item.TeamID, &item.SourceType, &item.SourceID,
		&item.Title, &item.Summary, &item.ContentText, &item.ContentHash,
		&metaBytes, &item.Visibility, &item.IndexedAt, &item.SourceUpdatedAt,
		&item.StaleAfter, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to index knowledge item: %w", err)
	}
	item.Metadata = json.RawMessage(metaBytes)
	return &item, nil
}

// ─── Search ───

func (h *Handler) Search(ctx context.Context, teamID, query, sourceType string, limit, offset int) (*SearchResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Results: []SearchResult{}, Total: 0, Query: query}, nil
	}

	// Build tsquery
	tsquery := buildTSQuery(query)

	var rows pgx.Rows
	var err error
	if sourceType != "" && sourceType != "all" {
		rows, err = h.pool.Query(ctx, `
			SELECT source_type, source_id::text, title,
			       ts_headline('english', content_text, to_tsquery($3), 'MaxWords=35, MinWords=10') AS snippet,
			       ts_rank(search_vector, to_tsquery($3)) AS rank,
			       updated_at::text
			FROM knowledge_items
			WHERE team_id = $1::uuid
			  AND source_type = $2
			  AND search_vector @@ to_tsquery($3)
			ORDER BY rank DESC, updated_at DESC
			LIMIT $4 OFFSET $5
		`, teamID, sourceType, tsquery, limit, offset)
	} else {
		rows, err = h.pool.Query(ctx, `
			SELECT source_type, source_id::text, title,
			       ts_headline('english', content_text, to_tsquery($2), 'MaxWords=35, MinWords=10') AS snippet,
			       ts_rank(search_vector, to_tsquery($2)) AS rank,
			       updated_at::text
			FROM knowledge_items
			WHERE team_id = $1::uuid
			  AND search_vector @@ to_tsquery($2)
			ORDER BY rank DESC, updated_at DESC
			LIMIT $3 OFFSET $4
		`, teamID, tsquery, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.SourceType, &r.SourceID, &r.Title, &r.Snippet, &r.Rank, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, r)
	}

	// Get total count
	var total int
	if sourceType != "" && sourceType != "all" {
		err = h.pool.QueryRow(ctx, `
			SELECT count(*) FROM knowledge_items
			WHERE team_id = $1::uuid AND source_type = $2
			  AND search_vector @@ to_tsquery($3)
		`, teamID, sourceType, tsquery).Scan(&total)
	} else {
		err = h.pool.QueryRow(ctx, `
			SELECT count(*) FROM knowledge_items
			WHERE team_id = $1::uuid
			  AND search_vector @@ to_tsquery($2)
		`, teamID, tsquery).Scan(&total)
	}
	if err != nil {
		total = len(results)
	}

	return &SearchResponse{Results: results, Total: total, Query: query}, nil
}

// ─── Get knowledge item ───

func (h *Handler) Get(ctx context.Context, teamID, itemID string) (*KnowledgeItem, error) {
	var item KnowledgeItem
	var metaBytes []byte
	err := h.pool.QueryRow(ctx, `
		SELECT id, team_id::text, source_type, source_id::text, title, summary,
		       content_text, content_hash, metadata, visibility,
		       indexed_at, source_updated_at, stale_after, created_at, updated_at
		FROM knowledge_items
		WHERE id = $1::uuid AND team_id = $2::uuid
	`, itemID, teamID).Scan(
		&item.ID, &item.TeamID, &item.SourceType, &item.SourceID,
		&item.Title, &item.Summary, &item.ContentText, &item.ContentHash,
		&metaBytes, &item.Visibility, &item.IndexedAt, &item.SourceUpdatedAt,
		&item.StaleAfter, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	item.Metadata = json.RawMessage(metaBytes)
	return &item, nil
}

// ─── Index status ───

func (h *Handler) IndexStatus(ctx context.Context, teamID string) (*IndexStatus, error) {
	status := &IndexStatus{ByType: map[string]int{}}

	// Total count
	err := h.pool.QueryRow(ctx, `
		SELECT count(*) FROM knowledge_items WHERE team_id = $1::uuid
	`, teamID).Scan(&status.TotalItems)
	if err != nil {
		return nil, err
	}

	// Count by type
	rows, err := h.pool.Query(ctx, `
		SELECT source_type, count(*) FROM knowledge_items
		WHERE team_id = $1::uuid
		GROUP BY source_type
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var c int
		if err := rows.Scan(&st, &c); err != nil {
			return nil, err
		}
		status.ByType[st] = c
	}

	// Stale count
	err = h.pool.QueryRow(ctx, `
		SELECT count(*) FROM knowledge_items
		WHERE team_id = $1::uuid AND stale_after < NOW()
	`, teamID).Scan(&status.StaleCount)
	if err != nil {
		status.StaleCount = 0
	}

	// Last indexed
	var lastIdx *time.Time
	_ = h.pool.QueryRow(ctx, `
		SELECT max(indexed_at) FROM knowledge_items WHERE team_id = $1::uuid
	`, teamID).Scan(&lastIdx)
	status.LastIndexed = lastIdx

	return status, nil
}

// ─── Remove a knowledge item ───

func (h *Handler) Remove(ctx context.Context, teamID, sourceType, sourceID string) error {
	_, err := h.pool.Exec(ctx, `
		DELETE FROM knowledge_items
		WHERE team_id = $1::uuid AND source_type = $2 AND source_id = $3::uuid
	`, teamID, sourceType, sourceID)
	return err
}

// ─── HTTP Handlers ───

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	return r
}

// GET /api/teams/{teamId}/knowledge/search?q=&source_type=&limit=&offset=
func (h *Handler) SearchHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	q := r.URL.Query().Get("q")
	sourceType := r.URL.Query().Get("source_type")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	resp, err := h.Search(r.Context(), teamID, q, sourceType, limit, offset)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Search failed"})
		return
	}
	writeJSON(w, 200, resp)
}

// GET /api/teams/{teamId}/knowledge/{itemId}
func (h *Handler) GetHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	itemID := chi.URLParam(r, "itemId")

	item, err := h.Get(r.Context(), teamID, itemID)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "Knowledge item not found"})
		return
	}
	writeJSON(w, 200, item)
}

// GET /api/teams/{teamId}/knowledge/index-status
func (h *Handler) IndexStatusHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	status, err := h.IndexStatus(r.Context(), teamID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to get index status"})
		return
	}
	writeJSON(w, 200, status)
}

// POST /api/admin/knowledge/reindex
func (h *Handler) ReindexHTTP(w http.ResponseWriter, r *http.Request) {
	// Placeholder — actual indexing implemented in Track 2
	writeJSON(w, 200, map[string]string{"status": "reindex triggered"})
}

// GET /api/admin/knowledge/index-status
func (h *Handler) AdminIndexStatusHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := h.IndexStatus(r.Context(), "")
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed"})
		return
	}
	writeJSON(w, 200, status)
}

// ─── Helpers ───

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func parseIntDefault(s string, def int) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}

// buildTSQuery converts a plain-text query into a tsquery with & between terms.
func buildTSQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	var parts []string
	for _, t := range terms {
		// Escape single quotes for SQL safety
		t = strings.ReplaceAll(t, "'", " ")
		t = strings.TrimSpace(t)
		if t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " & ")
}

// ValidateSourceType checks that the source type is one of the allowed types.
func ValidateSourceType(st string) bool {
	switch st {
	case "artifact", "clarity_document", "meeting_summary", "status_report",
		"presentation", "template", "work_item", "incident", "project",
		"asset", "remediation", "approval", "context_node":
		return true
	}
	return false
}
