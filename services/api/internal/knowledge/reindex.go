package knowledge

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Reindex ───

type ReindexResult struct {
	TotalSources  int            `json:"total_sources"`
	Indexed       int            `json:"indexed"`
	Skipped       int            `json:"skipped"`
	Errors        int            `json:"errors"`
	ByType        map[string]int `json:"by_type"`
	Duration      string         `json:"duration"`
	TeamID        string         `json:"team_id,omitempty"`
}

// ReindexAll reindexes all supported sources for a team.
// If teamID is empty, reindexes all teams.
func ReindexAll(ctx context.Context, pool *pgxpool.Pool, teamID string) (*ReindexResult, error) {
	start := time.Now()
	result := &ReindexResult{ByType: map[string]int{}}
	ix := NewIndexer(pool)

	// Get team IDs to process
	var teamIDs []string
	if teamID != "" {
		teamIDs = []string{teamID}
	} else {
		rows, err := pool.Query(ctx, "SELECT id::text FROM teams")
		if err != nil {
			return nil, fmt.Errorf("query teams: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			teamIDs = append(teamIDs, id)
		}
	}

	// Extractors for each source type
	type extractor struct {
		sourceType string
		fn         func(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error)
	}
	extractors := []extractor{
		{"clarity_document", ExtractClarityDocuments},
		{"artifact", ExtractArtifacts},
		{"meeting_summary", ExtractMeetingSummaries},
		{"template", ExtractTemplates},
		{"work_item", ExtractWorkItems},
		{"incident", ExtractIncidents},
		{"asset", ExtractAssets},
		{"remediation", ExtractRemediations},
		{"approval", ExtractApprovals},
		{"context_node", ExtractContextNodes},
	}

	for _, tid := range teamIDs {
		for _, ex := range extractors {
			docs, err := ex.fn(ctx, pool, tid)
			if err != nil {
				result.Errors++
				continue
			}

			for _, doc := range docs {
				result.TotalSources++
				err := ix.IndexSource(ctx, doc)
				if err != nil {
					result.Errors++
					continue
				}
				result.Indexed++
				result.ByType[ex.sourceType]++
			}
		}
	}

	result.Duration = time.Since(start).String()
	result.TeamID = teamID
	return result, nil
}

// AdminReindexHTTP handles POST /api/admin/knowledge/reindex
func (h *Handler) AdminReindexHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := r.URL.Query().Get("team_id")

	result, err := ReindexAll(r.Context(), h.pool, teamID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Reindex failed: " + err.Error()})
		return
	}
	writeJSON(w, 200, result)
}

// AdminIndexStatusAllHTTP handles GET /api/admin/knowledge/index-status
func (h *Handler) AdminIndexStatusAllHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := &IndexStatus{ByType: map[string]int{}}

	// Total count across all teams
	err := h.pool.QueryRow(ctx, `SELECT count(*) FROM knowledge_items`).Scan(&status.TotalItems)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed"})
		return
	}

	// Count by type
	rows, err := h.pool.Query(ctx, `SELECT source_type, count(*) FROM knowledge_items GROUP BY source_type`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var st string
			var c int
			rows.Scan(&st, &c)
			status.ByType[st] = c
		}
	}

	// Stale count
	_ = h.pool.QueryRow(ctx, `SELECT count(*) FROM knowledge_items WHERE stale_after < NOW()`).Scan(&status.StaleCount)

	// Chunk count
	var totalChunks int
	_ = h.pool.QueryRow(ctx, `SELECT count(*) FROM knowledge_chunks`).Scan(&totalChunks)
	status.ByType["__chunks_total__"] = totalChunks

	// Last indexed
	var lastIdx *time.Time
	_ = h.pool.QueryRow(ctx, `SELECT max(indexed_at) FROM knowledge_items`).Scan(&lastIdx)
	status.LastIndexed = lastIdx

	writeJSON(w, 200, status)
}
