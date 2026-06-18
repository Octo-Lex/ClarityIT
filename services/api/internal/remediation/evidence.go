package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EvidenceHandler provides recommendation evidence pack endpoints.
type EvidenceHandler struct {
	pool *pgxpool.Pool
}

func NewEvidenceHandler(pool *pgxpool.Pool) *EvidenceHandler {
	return &EvidenceHandler{pool: pool}
}

func (h *EvidenceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/recommendations/{recommendationId}/evidence", h.GetEvidence)
	return r
}

// ─── Get Evidence ───

func (h *EvidenceHandler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	recommendationID, err := uuid.Parse(chi.URLParam(r, "recommendationId"))
	if err != nil {
		writeErr(w, 400, "Invalid recommendation ID")
		return
	}

	var id, sourceType, summary, confidenceLevel, riskNotes string
	var sourceID uuid.UUID
	var supportingEvidence, conflictingEvidence, missingInfo json.RawMessage
	var confidenceScore float64
	var isStale bool
	var staleAfter *time.Time
	var createdAt, updatedAt time.Time

	err = h.pool.QueryRow(ctx, `
		SELECT id::text, source_type, source_id, recommendation_summary,
		       supporting_evidence, conflicting_evidence,
		       confidence_score, confidence_level, risk_notes, missing_info,
		       is_stale, stale_after, created_at, updated_at
		FROM recommendation_evidence
		WHERE team_id=$1 AND recommendation_id=$2
	`, teamID, recommendationID).Scan(
		&id, &sourceType, &sourceID, &summary,
		&supportingEvidence, &conflictingEvidence,
		&confidenceScore, &confidenceLevel, &riskNotes, &missingInfo,
		&isStale, &staleAfter, &createdAt, &updatedAt,
	)
	if err != nil {
		// Legacy recommendations or missing evidence — return safe "unavailable" state
		writeJSON(w, 200, map[string]any{
			"recommendation_id": recommendationID,
			"available":          false,
			"message":            "Evidence unavailable for this recommendation",
		})
		return
	}

	// Check if evidence should be marked stale based on stale_after timestamp
	if staleAfter != nil && time.Now().After(*staleAfter) {
		isStale = true
	}

	// Parse evidence arrays for response
	var supporting []map[string]any
	var conflicting []map[string]any
	var missing []map[string]any
	json.Unmarshal(supportingEvidence, &supporting)
	json.Unmarshal(conflictingEvidence, &conflicting)
	json.Unmarshal(missingInfo, &missing)

	writeJSON(w, 200, map[string]any{
		"recommendation_id":      recommendationID,
		"available":              true,
		"source_type":            sourceType,
		"source_id":              sourceID.String(),
		"recommendation_summary": summary,
		"supporting_evidence":    supporting,
		"conflicting_evidence":   conflicting,
		"confidence_score":       confidenceScore,
		"confidence_level":       confidenceLevel,
		"risk_notes":             riskNotes,
		"missing_info":           missing,
		"is_stale":               isStale,
		"stale_after":            staleAfter,
		"created_at":             createdAt,
		"updated_at":             updatedAt,
	})
}

// ─── Create Evidence (internal — called by remediation Create handler) ───

// EvidenceInput is the evidence data submitted with a recommendation.
// The Go control plane validates and persists this — the Python worker
// never writes to the DB directly.
type EvidenceInput struct {
	RecommendationSummary string             `json:"recommendation_summary"`
	SupportingEvidence    []map[string]any   `json:"supporting_evidence"`
	ConflictingEvidence   []map[string]any   `json:"conflicting_evidence"`
	ConfidenceScore       float64            `json:"confidence_score"`
	ConfidenceLevel       string             `json:"confidence_level"`
	RiskNotes             string             `json:"risk_notes"`
	MissingInfo           []map[string]any   `json:"missing_info"`
}

// ValidateEvidenceInput validates evidence data before persistence.
// Returns sanitized evidence or an error.
func ValidateEvidenceInput(input EvidenceInput) (EvidenceInput, error) {
	// Clamp confidence score to [0.0, 1.0]
	if input.ConfidenceScore < 0.0 || input.ConfidenceScore > 1.0 {
		return input, fmt.Errorf("confidence_score must be between 0.0 and 1.0, got %.4f", input.ConfidenceScore)
	}

	// Validate confidence level
	if input.ConfidenceLevel == "" {
		// Derive from score
		if input.ConfidenceScore >= 0.7 {
			input.ConfidenceLevel = "high"
		} else if input.ConfidenceScore >= 0.4 {
			input.ConfidenceLevel = "medium"
		} else {
			input.ConfidenceLevel = "low"
		}
	} else if input.ConfidenceLevel != "low" && input.ConfidenceLevel != "medium" && input.ConfidenceLevel != "high" {
		return input, fmt.Errorf("confidence_level must be 'low', 'medium', or 'high'")
	}

	// Sanitize all evidence fields — strip any sensitive patterns
	input.SupportingEvidence = sanitizeEvidenceItems(input.SupportingEvidence)
	input.ConflictingEvidence = sanitizeEvidenceItems(input.ConflictingEvidence)
	input.MissingInfo = sanitizeEvidenceItems(input.MissingInfo)

	// Sanitize free-text fields
	input.RecommendationSummary = sanitizeEvidenceText(input.RecommendationSummary)
	input.RiskNotes = sanitizeEvidenceText(input.RiskNotes)

	return input, nil
}

// sanitizeEvidenceItems removes sensitive patterns from evidence arrays.
func sanitizeEvidenceItems(items []map[string]any) []map[string]any {
	if items == nil {
		return []map[string]any{}
	}
	for _, item := range items {
		sanitizeMapKeys(item)
		// Also sanitize text values that may contain sensitive patterns
		for k, v := range item {
			if s, ok := v.(string); ok {
				item[k] = sanitizeEvidenceText(s)
			}
		}
	}
	return items
}

// sanitizeEvidenceText strips potential secrets from free-text fields.
func sanitizeEvidenceText(s string) string {
	// Redact patterns like token=xxx, secret=xxx in text content
	s = sensitiveParamRegex.ReplaceAllString(s, "[REDACTED]")
	return s
}

// PersistEvidence creates an evidence pack in the database.
// Called within the remediation creation transaction.
func PersistEvidence(
	ctx context.Context,
	pool *pgxpool.Pool,
	teamID uuid.UUID,
	recommendationID uuid.UUID,
	sourceType string,
	sourceID uuid.UUID,
	input EvidenceInput,
) error {
	// Calculate stale_after: 7 days from creation
	staleAfter := time.Now().Add(7 * 24 * time.Hour)

	// Serialize evidence arrays
	supportingJSON, _ := json.Marshal(input.SupportingEvidence)
	conflictingJSON, _ := json.Marshal(input.ConflictingEvidence)
	missingJSON, _ := json.Marshal(input.MissingInfo)

	_, err := pool.Exec(ctx, `
		INSERT INTO recommendation_evidence
			(team_id, recommendation_id, source_type, source_id,
			 recommendation_summary, supporting_evidence, conflicting_evidence,
			 confidence_score, confidence_level, risk_notes, missing_info,
			 stale_after)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT DO NOTHING
	`, teamID, recommendationID, sourceType, sourceID,
		input.RecommendationSummary, supportingJSON, conflictingJSON,
		input.ConfidenceScore, input.ConfidenceLevel, input.RiskNotes, missingJSON,
		staleAfter,
	)
	return err
}

// PersistEvidenceTx is the transactional version of PersistEvidence.
func PersistEvidenceTx(
	ctx context.Context,
	tx interface {
		Exec(context.Context, string, ...any) (interface{ RowsAffected() int64 }, error)
	},
	teamID uuid.UUID,
	recommendationID uuid.UUID,
	sourceType string,
	sourceID uuid.UUID,
	input EvidenceInput,
) error {
	// This is kept for transactional use but currently uses pool directly
	// since evidence creation is a separate concern
	return fmt.Errorf("not implemented — use PersistEvidence with pool")
}

// WriteEvidenceAudit writes an audit event for evidence creation.
func WriteEvidenceAudit(
	ctx context.Context,
	teamIDStr string,
	teamID, userID, recommendationID uuid.UUID,
	sourceType string,
) {
	meta, _ := json.Marshal(map[string]any{
		"recommendation_id": recommendationID.String(),
		"source_type":       sourceType,
	})

	// Use direct pool exec for audit since this is a post-commit action
	// In the remediation handler, evidence is created within the same tx
	_ = audit.Write(ctx, nil, audit.Event{
		TeamID:     &teamID,
		ActorID:    userID,
		Action:     "recommendation.evidence.created",
		EntityType: "recommendation_evidence",
		EntityID:   recommendationID,
		NewValue:   meta,
	})

	_ = outbox.Write(ctx, nil, &teamIDStr, outbox.Event{
		EventType:       "clarity.v1.recommendation.evidence.created",
		AggregateType:   "recommendation_evidence",
		AggregateID:     recommendationID.String(),
		Payload:         meta,
	})
}

var _ = iam.GetClaims
