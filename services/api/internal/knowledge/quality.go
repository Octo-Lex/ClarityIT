package knowledge

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ─── Types ───

type QualityReport struct {
	TeamID       string         `json:"team_id"`
	TotalItems   int            `json:"total_items"`
	StaleCount   int            `json:"stale_count"`
	DupCount     int            `json:"duplicate_count"`
	OrphanCount  int            `json:"orphan_count"`
	ByType       map[string]int `json:"by_type"`
	StaleItems   []QualityItem  `json:"stale_items"`
	DupGroups    []DupGroup     `json:"duplicate_groups"`
	OrphanItems  []QualityItem  `json:"orphan_items"`
	GeneratedAt  string         `json:"generated_at"`
}

type QualityItem struct {
	KnowledgeItemID string `json:"knowledge_item_id"`
	SourceType      string `json:"source_type"`
	SourceID        string `json:"source_id"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	IndexedAt       string `json:"indexed_at"`
	StaleAfter      string `json:"stale_after,omitempty"`
	DaysStale       int    `json:"days_stale,omitempty"`
}

type DupGroup struct {
	ContentHash string        `json:"content_hash"`
	Count       int           `json:"count"`
	Items       []QualityItem `json:"items"`
}

// GET /api/teams/{teamId}/knowledge/quality
func (h *Handler) QualityReportHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	report := QualityReport{
		TeamID:      teamID,
		ByType:      map[string]int{},
		StaleItems:  []QualityItem{},
		DupGroups:   []DupGroup{},
		OrphanItems: []QualityItem{},
		GeneratedAt: "now()",
	}

	// Total items + by type
	rows, err := h.pool.Query(r.Context(), `
		SELECT source_type, count(*) FROM knowledge_items
		WHERE team_id = $1::uuid
		GROUP BY source_type
	`, teamID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to get quality report"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var st string
		var c int
		if err := rows.Scan(&st, &c); err != nil {
			writeJSON(w, 500, map[string]string{"error": "Failed to scan quality report"})
			return
		}
		report.ByType[st] = c
		report.TotalItems += c
	}

	// Stale count
	h.pool.QueryRow(r.Context(), `
		SELECT count(*) FROM knowledge_items
		WHERE team_id = $1::uuid AND stale_after IS NOT NULL AND stale_after < NOW()
	`, teamID).Scan(&report.StaleCount)

	// Stale items (top 20)
	staleRows, err := h.pool.Query(r.Context(), `
		SELECT id::text, source_type, source_id::text, title, COALESCE(summary, ''),
		       indexed_at::text, stale_after::text,
		       EXTRACT(DAY FROM NOW() - stale_after)::int AS days_stale
		FROM knowledge_items
		WHERE team_id = $1::uuid AND stale_after IS NOT NULL AND stale_after < NOW()
		ORDER BY stale_after ASC
		LIMIT 20
	`, teamID)
	if err == nil {
		defer staleRows.Close()
		for staleRows.Next() {
			var item QualityItem
			if err := staleRows.Scan(&item.KnowledgeItemID, &item.SourceType, &item.SourceID,
				&item.Title, &item.Summary, &item.IndexedAt, &item.StaleAfter, &item.DaysStale); err == nil {
				report.StaleItems = append(report.StaleItems, item)
			}
		}
	}

	// Duplicate groups (same content_hash, count > 1)
	dupRows, err := h.pool.Query(r.Context(), `
		SELECT content_hash, count(*) AS cnt
		FROM knowledge_items
		WHERE team_id = $1::uuid AND content_hash IS NOT NULL
		GROUP BY content_hash
		HAVING count(*) > 1
		ORDER BY cnt DESC
		LIMIT 10
	`, teamID)
	if err == nil {
		for dupRows.Next() {
			var hash string
			var cnt int
			if err := dupRows.Scan(&hash, &cnt); err != nil {
				continue
			}
			report.DupCount += cnt

			// Get items in this dup group
			itemRows, err := h.pool.Query(r.Context(), `
				SELECT id::text, source_type, source_id::text, title, COALESCE(summary, ''),
				       indexed_at::text
				FROM knowledge_items
				WHERE team_id = $1::uuid AND content_hash = $2
				ORDER BY indexed_at DESC
				LIMIT 10
			`, teamID, hash)
			if err != nil {
				continue
			}
			var items []QualityItem
			for itemRows.Next() {
				var item QualityItem
				if err := itemRows.Scan(&item.KnowledgeItemID, &item.SourceType, &item.SourceID,
					&item.Title, &item.Summary, &item.IndexedAt); err == nil {
					items = append(items, item)
				}
			}
			itemRows.Close()
			report.DupGroups = append(report.DupGroups, DupGroup{
				ContentHash: hash[:12] + "…",
				Count:       cnt,
				Items:       items,
			})
		}
		dupRows.Close()
	}

	// Orphan detection (v1): knowledge items whose source has been deleted.
	// v1 scope: checks the artifacts table for artifact/clarity_document/
	// status_report/meeting_summary/presentation types — these share the
	// artifacts table via typed extension tables (ADR-013).
	// Future versions should extend orphan detection to:
	//   - work_item / incident / project / asset → objects table
	//   - remediation → remediations table
	//   - approval → approvals table
	//   - context_node → context_nodes table
	//   - template → artifact_templates table
	// For v1.5.0, artifact-orphan detection covers the highest-risk case
	// (documents and artifacts deleted while knowledge index retains them).
	orphanRows, err := h.pool.Query(r.Context(), `
		SELECT ki.id::text, ki.source_type, ki.source_id::text, ki.title, COALESCE(ki.summary, ''),
		       ki.indexed_at::text
		FROM knowledge_items ki
		WHERE ki.team_id = $1::uuid
		  AND ki.source_type IN ('artifact', 'clarity_document', 'meeting_summary', 'status_report', 'presentation')
		  AND NOT EXISTS (
		    SELECT 1 FROM artifacts a
		    WHERE a.id = ki.source_id AND a.team_id = ki.team_id
		  )
		ORDER BY ki.indexed_at ASC
		LIMIT 20
	`, teamID)
	if err == nil {
		defer orphanRows.Close()
		for orphanRows.Next() {
			var item QualityItem
			if err := orphanRows.Scan(&item.KnowledgeItemID, &item.SourceType, &item.SourceID,
				&item.Title, &item.Summary, &item.IndexedAt); err == nil {
				report.OrphanItems = append(report.OrphanItems, item)
			}
		}
	}
	report.OrphanCount = len(report.OrphanItems)

	writeJSON(w, 200, report)
}

// GET /api/teams/{teamId}/knowledge/quality/stale
func (h *Handler) StaleItemsHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	if limit > 200 {
		limit = 200
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id::text, source_type, source_id::text, title, COALESCE(summary, ''),
		       indexed_at::text, stale_after::text,
		       EXTRACT(DAY FROM NOW() - stale_after)::int AS days_stale
		FROM knowledge_items
		WHERE team_id = $1::uuid AND stale_after IS NOT NULL AND stale_after < NOW()
		ORDER BY stale_after ASC
		LIMIT $2
	`, teamID, limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to get stale items"})
		return
	}
	defer rows.Close()

	items := []QualityItem{}
	for rows.Next() {
		var item QualityItem
		if err := rows.Scan(&item.KnowledgeItemID, &item.SourceType, &item.SourceID,
			&item.Title, &item.Summary, &item.IndexedAt, &item.StaleAfter, &item.DaysStale); err != nil {
			writeJSON(w, 500, map[string]string{"error": "Failed to scan stale items"})
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"stale_items": items, "count": len(items)})
}

// GET /api/teams/{teamId}/knowledge/quality/duplicates
func (h *Handler) DuplicateItemsHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	rows, err := h.pool.Query(r.Context(), `
		SELECT content_hash, count(*) AS cnt
		FROM knowledge_items
		WHERE team_id = $1::uuid AND content_hash IS NOT NULL
		GROUP BY content_hash
		HAVING count(*) > 1
		ORDER BY cnt DESC
		LIMIT 20
	`, teamID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to get duplicates"})
		return
	}
	defer rows.Close()

	groups := []DupGroup{}
	for rows.Next() {
		var hash string
		var cnt int
		if err := rows.Scan(&hash, &cnt); err != nil {
			continue
		}
		groups = append(groups, DupGroup{
			ContentHash: hash[:12] + "…",
			Count:       cnt,
			Items:       []QualityItem{},
		})
	}
	writeJSON(w, 200, map[string]any{"duplicate_groups": groups, "count": len(groups)})
}

// GET /api/teams/{teamId}/knowledge/quality/orphans
func (h *Handler) OrphanItemsHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	if limit > 200 {
		limit = 200
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT ki.id::text, ki.source_type, ki.source_id::text, ki.title, COALESCE(ki.summary, ''),
		       ki.indexed_at::text
		FROM knowledge_items ki
		WHERE ki.team_id = $1::uuid
		  AND ki.source_type IN ('artifact', 'clarity_document', 'meeting_summary', 'status_report', 'presentation')
		  AND NOT EXISTS (
		    SELECT 1 FROM artifacts a
		    WHERE a.id = ki.source_id AND a.team_id = ki.team_id
		  )
		ORDER BY ki.indexed_at ASC
		LIMIT $2
	`, teamID, limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to get orphans"})
		return
	}
	defer rows.Close()

	items := []QualityItem{}
	for rows.Next() {
		var item QualityItem
		if err := rows.Scan(&item.KnowledgeItemID, &item.SourceType, &item.SourceID,
			&item.Title, &item.Summary, &item.IndexedAt); err != nil {
			writeJSON(w, 500, map[string]string{"error": "Failed to scan orphans"})
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"orphan_items": items, "count": len(items)})
}
