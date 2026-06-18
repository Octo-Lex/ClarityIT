package knowledge

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
)

// ─── Related Knowledge Types ───

type RelatedRequest struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
}

type RelatedItem struct {
	SourceType string  `json:"source_type"`
	SourceID   string  `json:"source_id"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary"`
	Snippet    string  `json:"snippet"`
	Rank       float64 `json:"rank"`
	Reason     string  `json:"reason"`
	UpdatedAt  string  `json:"updated_at"`
}

type RelatedResponse struct {
	Source  RelatedRequest `json:"source"`
	Related []RelatedItem  `json:"related"`
}

// ─── Related Knowledge Handler ───

// Related returns related knowledge items for a given source item.
// It uses deterministic ranking signals: explicit links, context edges,
// shared references, content similarity (FTS), same source family, and recency.
func (h *Handler) Related(ctx context.Context, teamID, sourceType, sourceID string, limit int) (*RelatedResponse, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}

	// Validate source exists in knowledge_items for this team
	var sourceTitle string
	err := h.pool.QueryRow(ctx, `
		SELECT title FROM knowledge_items
		WHERE team_id = $1::uuid AND source_type = $2 AND source_id::text = $3
	`, teamID, sourceType, sourceID).Scan(&sourceTitle)
	if err != nil {
		return nil, fmt.Errorf("source item not found")
	}

	// Gather candidates from all signals
	type candidate struct {
		RelatedItem
		priority int // lower = higher priority (first reason wins)
	}
	candidates := make(map[string]*candidate) // key = source_type + "|" + source_id

	// ─── Signal 1: Explicit object_links ───
	// If the source is an object (work_item, incident, asset), check object_links
	if sourceType == "work_item" || sourceType == "incident" || sourceType == "asset" {
		rows, err := h.pool.Query(ctx, `
			SELECT ki.source_type, ki.source_id::text, ki.title, ki.summary,
			       ki.updated_at::text
			FROM object_links ol
			JOIN knowledge_items ki ON ki.team_id = $1::uuid
			  AND ki.source_type IN ('work_item', 'incident', 'asset', 'clarity_document', 'artifact')
			WHERE ol.team_id = $1::uuid
			  AND (
			    (ol.from_object_id::text = $3 AND ki.source_id::text = ol.to_object_id::text)
			    OR
			    (ol.to_object_id::text = $3 AND ki.source_id::text = ol.from_object_id::text)
			  )
			  AND NOT (ki.source_type = $2 AND ki.source_id::text = $3)
			LIMIT $4
		`, teamID, sourceType, sourceID, limit)
		if err == nil {
			for rows.Next() {
				var item RelatedItem
				item.Reason = "explicit_link"
				item.Rank = 0.9
				if err := rows.Scan(&item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.UpdatedAt); err == nil {
					key := item.SourceType + "|" + item.SourceID
					if _, exists := candidates[key]; !exists {
						candidates[key] = &candidate{RelatedItem: item, priority: 1}
					}
				}
			}
			rows.Close()
		}
	}

	// ─── Signal 2: Context edges ───
	// Check if source has a context_node, find linked context_nodes
	rows, err := h.pool.Query(ctx, `
		SELECT ki.source_type, ki.source_id::text, ki.title, ki.summary,
		       ki.updated_at::text
		FROM context_edges ce
		JOIN context_nodes src_cn ON src_cn.id = ce.from_node_id
		  AND src_cn.team_id = $1::uuid
		  AND src_cn.entity_type = $2
		  AND src_cn.entity_id::text = $3
		JOIN context_nodes dst_cn ON dst_cn.id = ce.to_node_id
		  AND dst_cn.team_id = $1::uuid
		JOIN knowledge_items ki ON ki.team_id = $1::uuid
		  AND ki.source_type = dst_cn.entity_type
		  AND ki.source_id::text = dst_cn.entity_id::text
		WHERE ce.team_id = $1::uuid
		  AND NOT (ki.source_type = $2 AND ki.source_id::text = $3)
		LIMIT $4
	`, teamID, sourceType, sourceID, limit)
	if err == nil {
		for rows.Next() {
			var item RelatedItem
			item.Reason = "context_edge"
			item.Rank = 0.8
			if err := rows.Scan(&item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.UpdatedAt); err == nil {
				key := item.SourceType + "|" + item.SourceID
				if _, exists := candidates[key]; !exists {
					candidates[key] = &candidate{RelatedItem: item, priority: 2}
				}
			}
		}
		rows.Close()
	}
	// Also check reverse direction (to_node -> from_node)
	rows2, err := h.pool.Query(ctx, `
		SELECT ki.source_type, ki.source_id::text, ki.title, ki.summary,
		       ki.updated_at::text
		FROM context_edges ce
		JOIN context_nodes src_cn ON src_cn.id = ce.to_node_id
		  AND src_cn.team_id = $1::uuid
		  AND src_cn.entity_type = $2
		  AND src_cn.entity_id::text = $3
		JOIN context_nodes dst_cn ON dst_cn.id = ce.from_node_id
		  AND dst_cn.team_id = $1::uuid
		JOIN knowledge_items ki ON ki.team_id = $1::uuid
		  AND ki.source_type = dst_cn.entity_type
		  AND ki.source_id::text = dst_cn.entity_id::text
		WHERE ce.team_id = $1::uuid
		  AND NOT (ki.source_type = $2 AND ki.source_id::text = $3)
		LIMIT $4
	`, teamID, sourceType, sourceID, limit)
	if err == nil {
		for rows2.Next() {
			var item RelatedItem
			item.Reason = "context_edge"
			item.Rank = 0.75
			if err := rows2.Scan(&item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.UpdatedAt); err == nil {
				key := item.SourceType + "|" + item.SourceID
				if _, exists := candidates[key]; !exists {
					candidates[key] = &candidate{RelatedItem: item, priority: 2}
				}
			}
		}
		rows2.Close()
	}

	// ─── Signal 3: Content similarity via FTS ───
	// Use the source item's title as the search query
	tsquery := buildTSQuery(sourceTitle)
	if tsquery != "" && tsquery != "()" {
		rows, err := h.pool.Query(ctx, `
			SELECT ki.source_type, ki.source_id::text, ki.title, ki.summary,
			       ts_rank(ki.search_vector, to_tsquery($4)) AS rank,
			       ki.updated_at::text
			FROM knowledge_items ki
			WHERE ki.team_id = $1::uuid
			  AND ki.search_vector @@ to_tsquery($4)
			  AND NOT (ki.source_type = $2 AND ki.source_id::text = $3)
			ORDER BY rank DESC, ki.updated_at DESC
			LIMIT $5
		`, teamID, sourceType, sourceID, tsquery, limit*2)
		if err == nil {
			for rows.Next() {
				var item RelatedItem
				item.Reason = "content_similarity"
				if err := rows.Scan(&item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.Rank, &item.UpdatedAt); err == nil {
					// Scale rank down for content similarity
					item.Rank = item.Rank * 0.5
					key := item.SourceType + "|" + item.SourceID
					if _, exists := candidates[key]; !exists {
						candidates[key] = &candidate{RelatedItem: item, priority: 3}
					}
				}
			}
			rows.Close()
		}
	}

	// ─── Signal 4: Same source family ───
	// Items with the same source_type (excluding the source itself)
	rows, err = h.pool.Query(ctx, `
		SELECT ki.source_id::text, ki.title, ki.summary,
		       ki.updated_at::text
		FROM knowledge_items ki
		WHERE ki.team_id = $1::uuid
		  AND ki.source_type = $2
		  AND ki.source_id::text != $3
		ORDER BY ki.updated_at DESC
		LIMIT $4
	`, teamID, sourceType, sourceID, limit)
	if err == nil {
		for rows.Next() {
			var item RelatedItem
			item.SourceType = sourceType
			item.Reason = "same_source_family"
			item.Rank = 0.3
			if err := rows.Scan(&item.SourceID, &item.Title, &item.Summary, &item.UpdatedAt); err == nil {
				key := item.SourceType + "|" + item.SourceID
				if _, exists := candidates[key]; !exists {
					candidates[key] = &candidate{RelatedItem: item, priority: 4}
				}
			}
		}
		rows.Close()
	}

	// ─── Signal 5: Recent related (fill remaining slots) ───
	currentCount := len(candidates)
	if currentCount < limit {
		rows, err := h.pool.Query(ctx, `
			SELECT ki.source_type, ki.source_id::text, ki.title, ki.summary,
			       ki.updated_at::text
			FROM knowledge_items ki
			WHERE ki.team_id = $1::uuid
			  AND NOT (ki.source_type = $2 AND ki.source_id::text = $3)
			ORDER BY ki.updated_at DESC
			LIMIT $4
		`, teamID, sourceType, sourceID, limit)
		if err == nil {
			for rows.Next() {
				var item RelatedItem
				item.Reason = "recent_related"
				item.Rank = 0.1
				if err := rows.Scan(&item.SourceType, &item.SourceID, &item.Title, &item.Summary, &item.UpdatedAt); err == nil {
					key := item.SourceType + "|" + item.SourceID
					if _, exists := candidates[key]; !exists {
						candidates[key] = &candidate{RelatedItem: item, priority: 5}
					}
				}
			}
			rows.Close()
		}
	}

	// Convert map to slice, sort by priority then rank
	result := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, *c)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].priority != result[j].priority {
			return result[i].priority < result[j].priority
		}
		return result[i].Rank > result[j].Rank
	})

	// Apply limit
	if len(result) > limit {
		result = result[:limit]
	}

	// Build response
	related := make([]RelatedItem, 0, len(result))
	for _, c := range result {
		related = append(related, c.RelatedItem)
	}

	return &RelatedResponse{
		Source:  RelatedRequest{SourceType: sourceType, SourceID: sourceID},
		Related: related,
	}, nil
}

// ─── HTTP Handler ───

// GET /api/teams/{teamId}/knowledge/related?source_type=&source_id=&limit=
func (h *Handler) RelatedHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	sourceType := r.URL.Query().Get("source_type")
	sourceID := r.URL.Query().Get("source_id")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 8)

	if sourceType == "" || sourceID == "" {
		writeJSON(w, 400, map[string]string{"error": "source_type and source_id are required"})
		return
	}

	resp, err := h.Related(r.Context(), teamID, sourceType, sourceID, limit)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "Source item not found"})
		return
	}
	writeJSON(w, 200, resp)
}
