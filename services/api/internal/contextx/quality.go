package contextx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ───

type QualitySummary struct {
	TotalNodes             int `json:"total_nodes"`
	TotalRelations         int `json:"total_relations"`
	StaleNodes             int `json:"stale_nodes"`
	LowConfidenceRelations int `json:"low_confidence_relations"`
	ConflictingRelations   int `json:"conflicting_relations"`
	ConfirmedRelations     int `json:"confirmed_relations"`
	DismissedRelations     int `json:"dismissed_relations"`
}

type StaleNodeInfo struct {
	NodeID       string `json:"node_id"`
	NodeType     string `json:"node_type"`
	Label        string `json:"label"`
	LastUpdatedAt string `json:"last_updated_at"`
	DaysStale    int    `json:"days_stale"`
	Reason       string `json:"reason"`
}

type LowConfidenceRelation struct {
	RelationID    string  `json:"relation_id"`
	SourceNodeID  string  `json:"source_node_id"`
	TargetNodeID  string  `json:"target_node_id"`
	RelationType  string  `json:"relation_type"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason"`
}

type ConflictingRelation struct {
	RelationID     string `json:"relation_id"`
	SourceNodeID   string `json:"source_node_id"`
	TargetNodeID   string `json:"target_node_id"`
	RelationType   string `json:"relation_type"`
	ConflictReason string `json:"conflict_reason"`
}

type QualityResponse struct {
	QualityScore          int                     `json:"quality_score"`
	AdvisoryOnly          bool                    `json:"advisory_only"`
	Summary               QualitySummary          `json:"summary"`
	StaleNodes            []StaleNodeInfo         `json:"stale_nodes"`
	LowConfidenceRelations []LowConfidenceRelation `json:"low_confidence_relations"`
	ConflictingRelations  []ConflictingRelation   `json:"conflicting_relations"`
}

type ReviewRequest struct {
	Reason string `json:"reason"`
}

type ReviewResponse struct {
	RelationID    string `json:"relation_id"`
	QualityStatus string `json:"quality_status"`
	ReviewedBy    string `json:"reviewed_by"`
	ReviewedAt    string `json:"reviewed_at"`
}

// ─── Handler ───

type QualityHandler struct {
	pool *pgxpool.Pool
}

func NewQualityHandler(pool *pgxpool.Pool) *QualityHandler {
	return &QualityHandler{pool: pool}
}

func (h *QualityHandler) GetQuality(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeQualityErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	// Parse query parameters
	staleDays := 30
	confidenceThreshold := 0.60

	if sd := r.URL.Query().Get("stale_days"); sd != "" {
		var n int
		if _, err := fmt.Sscanf(sd, "%d", &n); err == nil && n >= 1 && n <= 365 {
			staleDays = n
		} else {
			writeQualityErr(w, http.StatusBadRequest, "stale_days must be between 1 and 365")
			return
		}
	}

	if ct := r.URL.Query().Get("confidence_threshold"); ct != "" {
		var f float64
		if _, err := fmt.Sscanf(ct, "%f", &f); err == nil && f >= 0.0 && f <= 1.0 {
			confidenceThreshold = f
		} else {
			writeQualityErr(w, http.StatusBadRequest, "confidence_threshold must be between 0.0 and 1.0")
			return
		}
	}

	// Compute quality report
	resp := computeQualityReport(ctx, h.pool, teamID, staleDays, confidenceThreshold)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ConfirmRelation marks a relation as confirmed by an operator
func (h *QualityHandler) ConfirmRelation(w http.ResponseWriter, r *http.Request) {
	h.reviewRelation(w, r, "confirmed")
}

// DismissRelation marks a relation as dismissed by an operator
func (h *QualityHandler) DismissRelation(w http.ResponseWriter, r *http.Request) {
	h.reviewRelation(w, r, "dismissed")
}

func (h *QualityHandler) reviewRelation(w http.ResponseWriter, r *http.Request, status string) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeQualityErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	relationID, err := uuid.Parse(chi.URLParam(r, "relationId"))
	if err != nil {
		writeQualityErr(w, http.StatusBadRequest, "Invalid relation ID")
		return
	}

	// Verify the relation exists and is team-scoped
	var exists bool
	err = h.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM context_edges WHERE id=$1 AND team_id=$2)",
		relationID, teamID).Scan(&exists)
	if err != nil || !exists {
		writeQualityErr(w, http.StatusNotFound, "Relation not found in team scope")
		return
	}

	// Parse reason
	var req ReviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Get actor
	var actorID uuid.UUID
	if cl, ok := iam.GetClaims(r); ok {
		actorID, _ = uuid.Parse(cl.UserID)
	}

	// Upsert review (unique index on relation_id)
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Delete any existing review for this relation
		_, _ = tx.Exec(ctx,
			"DELETE FROM context_relation_reviews WHERE relation_id=$1 AND team_id=$2",
			relationID, teamID)

		// Insert new review
		_, err := tx.Exec(ctx, `
			INSERT INTO context_relation_reviews (team_id, relation_id, quality_status, reason, reviewed_by)
			VALUES ($1, $2, $3, $4, $5)
		`, teamID, relationID, status, req.Reason, actorID)
		if err != nil {
			return err
		}

		// Audit (advisory review event, not a mutation event)
		meta, _ := json.Marshal(map[string]any{
			"relation_id":   relationID.String(),
			"quality_status": status,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID,
			Action:     "context.relation.reviewed",
			EntityType: "context_edge",
			EntityID:   relationID,
			NewValue:   meta,
		})
		teamIDStr := teamID.String()
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.context.relation.reviewed",
			AggregateType: "context_edge",
			AggregateID:   relationID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeQualityErr(w, http.StatusInternalServerError, "Failed to record review")
		return
	}

	// Return review response
	resp := ReviewResponse{
		RelationID:    relationID.String(),
		QualityStatus: status,
		ReviewedBy:    actorID.String(),
		ReviewedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

// ─── Quality Computation ───

func computeQualityReport(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, staleDays int, confidenceThreshold float64) QualityResponse {
	// Count total nodes
	var totalNodes int
	pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM context_nodes WHERE team_id=$1", teamID).Scan(&totalNodes)

	// Count total relations
	var totalRelations int
	pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM context_edges WHERE team_id=$1", teamID).Scan(&totalRelations)

	// Count confirmed/dismissed
	var confirmed, dismissed int
	pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM context_relation_reviews WHERE team_id=$1 AND quality_status='confirmed'", teamID).Scan(&confirmed)
	pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM context_relation_reviews WHERE team_id=$1 AND quality_status='dismissed'", teamID).Scan(&dismissed)

	// Detect stale nodes
	staleNodes := detectStaleNodes(ctx, pool, teamID, staleDays)

	// Detect low-confidence relations
	lowConfRels := detectLowConfidenceRelations(ctx, pool, teamID, confidenceThreshold)

	// Detect conflicting relations
	conflictingRels := detectConflictingRelations(ctx, pool, teamID)

	// Compute quality score
	summary := QualitySummary{
		TotalNodes:             totalNodes,
		TotalRelations:         totalRelations,
		StaleNodes:             len(staleNodes),
		LowConfidenceRelations: len(lowConfRels),
		ConflictingRelations:   len(conflictingRels),
		ConfirmedRelations:     confirmed,
		DismissedRelations:     dismissed,
	}

	score := computeQualityScore(summary, totalNodes, totalRelations)

	// Ensure lists are not nil
	if staleNodes == nil {
		staleNodes = []StaleNodeInfo{}
	}
	if lowConfRels == nil {
		lowConfRels = []LowConfidenceRelation{}
	}
	if conflictingRels == nil {
		conflictingRels = []ConflictingRelation{}
	}

	return QualityResponse{
		QualityScore:          score,
		AdvisoryOnly:          true,
		Summary:               summary,
		StaleNodes:            staleNodes,
		LowConfidenceRelations: lowConfRels,
		ConflictingRelations:  conflictingRels,
	}
}

func detectStaleNodes(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, staleDays int) []StaleNodeInfo {
	staleThreshold := time.Now().UTC().AddDate(0, 0, -staleDays)

	rows, err := pool.Query(ctx, `
		SELECT id::text, entity_type,
		       COALESCE(properties->>'name', properties->>'title', entity_type || ':' || entity_id::text, entity_type) as label,
		       updated_at,
		       EXTRACT(EPOCH FROM (NOW() - updated_at))::int / 86400 as days_stale
		FROM context_nodes
		WHERE team_id=$1 AND updated_at < $2
		ORDER BY updated_at ASC
		LIMIT 100
	`, teamID, staleThreshold)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var nodes []StaleNodeInfo
	for rows.Next() {
		var n StaleNodeInfo
		var lastUpdated time.Time
		if err := rows.Scan(&n.NodeID, &n.NodeType, &n.Label, &lastUpdated, &n.DaysStale); err == nil {
			n.LastUpdatedAt = lastUpdated.Format(time.RFC3339)
			n.Reason = fmt.Sprintf("Node has not been updated in more than %d days.", staleDays)
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func detectLowConfidenceRelations(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, threshold float64) []LowConfidenceRelation {
	rows, err := pool.Query(ctx, `
		SELECT ce.id::text, ce.from_node_id::text, ce.to_node_id::text, ce.relation_type,
		       CAST(ce.weight AS DOUBLE PRECISION)
		FROM context_edges ce
		WHERE ce.team_id=$1 AND CAST(ce.weight AS DOUBLE PRECISION) < $2
		  AND NOT EXISTS (
		    SELECT 1 FROM context_relation_reviews crr
		    WHERE crr.relation_id = ce.id AND crr.quality_status = 'dismissed'
		  )
		ORDER BY ce.weight ASC
		LIMIT 100
	`, teamID, threshold)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var rels []LowConfidenceRelation
	for rows.Next() {
		var r LowConfidenceRelation
		if err := rows.Scan(&r.RelationID, &r.SourceNodeID, &r.TargetNodeID, &r.RelationType, &r.Confidence); err == nil {
			r.Reason = "Relation confidence is below threshold."
			rels = append(rels, r)
		}
	}
	return rels
}

func detectConflictingRelations(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID) []ConflictingRelation {
	// Find pairs of nodes with multiple different relation types between them
	rows, err := pool.Query(ctx, `
		WITH relation_counts AS (
			SELECT from_node_id, to_node_id,
			       COUNT(DISTINCT relation_type) as type_count,
			       array_agg(DISTINCT relation_type) as types
			FROM context_edges
			WHERE team_id=$1
			GROUP BY from_node_id, to_node_id
			HAVING COUNT(DISTINCT relation_type) > 1
		)
		SELECT ce.id::text, ce.from_node_id::text, ce.to_node_id::text,
		       ce.relation_type, rc.types::text
		FROM context_edges ce
		JOIN relation_counts rc ON rc.from_node_id = ce.from_node_id AND rc.to_node_id = ce.to_node_id
		WHERE ce.team_id=$1
		  AND NOT EXISTS (
		    SELECT 1 FROM context_relation_reviews crr
		    WHERE crr.relation_id = ce.id AND crr.quality_status = 'dismissed'
		  )
		LIMIT 50
	`, teamID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var rels []ConflictingRelation
	for rows.Next() {
		var r ConflictingRelation
		var typesStr string
		if err := rows.Scan(&r.RelationID, &r.SourceNodeID, &r.TargetNodeID, &r.RelationType, &typesStr); err == nil {
			r.ConflictReason = "Multiple contradictory relation types exist between the same nodes."
			rels = append(rels, r)
		}
	}
	return rels
}

func computeQualityScore(summary QualitySummary, totalNodes, totalRelations int) int {
	if totalNodes == 0 {
		return 100 // empty graph is perfect
	}

	// Start from 100 and subtract penalties
	score := 100

	// Stale nodes penalty: up to -20
	if totalNodes > 0 {
		staleRatio := float64(summary.StaleNodes) / float64(totalNodes)
		score -= int(staleRatio * 20)
	}

	// Low-confidence relations penalty: up to -20
	if totalRelations > 0 {
		lowConfRatio := float64(summary.LowConfidenceRelations) / float64(totalRelations)
		score -= int(lowConfRatio * 20)
	}

	// Conflicting relations penalty: up to -15
	if totalRelations > 0 {
		conflictRatio := float64(summary.ConflictingRelations) / float64(totalRelations)
		score -= int(conflictRatio * 15)
	}

	// Confirmed bonus: up to +5 (capped at 100)
	if totalRelations > 0 {
		confirmedRatio := float64(summary.ConfirmedRelations) / float64(totalRelations)
		score += int(confirmedRatio * 5)
	}

	// Clamp 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// ─── Helpers ───

func writeQualityErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
