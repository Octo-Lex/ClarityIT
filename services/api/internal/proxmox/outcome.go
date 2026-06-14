package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ───

type OutcomeRequest struct {
	ExpectedResult          *string `json:"expected_result"`
	ActualResult            *string `json:"actual_result"`
	OperatorFeedback        *string `json:"operator_feedback"`
	OutcomeStatus           string  `json:"outcome_status"`
	FollowUpRecommendation  *string `json:"follow_up_recommendation"`
}

type OutcomeResponse struct {
	ID                     string  `json:"id"`
	AssetActionID          *string `json:"asset_action_id,omitempty"`
	RemediationProposalID  *string `json:"remediation_proposal_id,omitempty"`
	ExpectedResult         *string `json:"expected_result"`
	ActualResult           *string `json:"actual_result"`
	OperatorFeedback       *string `json:"operator_feedback"`
	OutcomeStatus          string  `json:"outcome_status"`
	FollowUpRecommendation *string `json:"follow_up_recommendation"`
	CreatedBy              string  `json:"created_by"`
	CreatedAt              string  `json:"created_at"`
	UpdatedAt              string  `json:"updated_at"`
}

var validOutcomeStatuses = map[string]bool{
	"successful":            true,
	"partially_successful": true,
	"failed":               true,
	"inconclusive":         true,
}

// Sensitive patterns to strip from outcome text fields
var outcomeSensitiveRegex = regexp.MustCompile(`(?i)(password|secret|token|api_key|credential)[\s]*[=:]\s*[^\s,;\]}"]*`)
var outcomeStandaloneRegex = regexp.MustCompile(`(?i)\b(password|secret|token|api_key|credential)\b`)

// ─── Handler ───

type OutcomeHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewOutcomeHandler(pool *pgxpool.Pool, cfg *config.Config) *OutcomeHandler {
	return &OutcomeHandler{pool: pool, cfg: cfg}
}

// CreateOrUpdateAssetActionOutcome creates or updates the outcome for an asset action
func (h *OutcomeHandler) CreateOrUpdateAssetActionOutcome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, "Invalid action ID")
		return
	}

	// Verify the asset action exists, is team-scoped, and is in a terminal status
	var actionStatus string
	err = h.pool.QueryRow(ctx, `
		SELECT status FROM asset_actions WHERE id=$1 AND team_id=$2
	`, actionID, teamID).Scan(&actionStatus)
	if err != nil {
		writeOutcomeErr(w, http.StatusNotFound, "Asset action not found in team scope")
		return
	}

	// Only completed/succeeded/failed actions may receive outcomes
	terminalStatuses := map[string]bool{"succeeded": true, "failed": true, "completed": true}
	if !terminalStatuses[actionStatus] {
		writeOutcomeErr(w, http.StatusConflict,
			fmt.Sprintf("Asset action must be in a terminal status (succeeded/failed/completed) to record an outcome. Current: %s", actionStatus))
		return
	}

	// Parse and validate request
	var req OutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if !validOutcomeStatuses[req.OutcomeStatus] {
		writeOutcomeErr(w, http.StatusBadRequest,
			"outcome_status must be one of: successful, partially_successful, failed, inconclusive")
		return
	}
	if err := validateOutcomeLengths(&req); err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize text fields
	sanitizeOutcomeFields(&req)

	// Get actor ID from claims
	cl, ok := iam.GetClaims(r)
	var actorID uuid.UUID
	if ok {
		actorID, _ = uuid.Parse(cl.UserID)
	}

	// Upsert outcome (unique index on asset_action_id)
	var outcomeID uuid.UUID
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Check if outcome already exists
		var existingID uuid.UUID
		err := tx.QueryRow(ctx,
			"SELECT id FROM action_outcomes WHERE asset_action_id=$1 AND team_id=$2",
			actionID, teamID).Scan(&existingID)

		if err == nil {
			// Update existing
			_, err = tx.Exec(ctx, `
				UPDATE action_outcomes
				SET expected_result=$3, actual_result=$4, operator_feedback=$5,
				    outcome_status=$6, follow_up_recommendation=$7, updated_at=NOW()
				WHERE id=$1 AND team_id=$2
			`, existingID, teamID,
				req.ExpectedResult, req.ActualResult, req.OperatorFeedback,
				req.OutcomeStatus, req.FollowUpRecommendation)
			outcomeID = existingID
		} else {
			// Insert new
			err = tx.QueryRow(ctx, `
				INSERT INTO action_outcomes (team_id, asset_action_id, expected_result, actual_result,
				    operator_feedback, outcome_status, follow_up_recommendation, created_by)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING id
			`, teamID, actionID,
				req.ExpectedResult, req.ActualResult, req.OperatorFeedback,
				req.OutcomeStatus, req.FollowUpRecommendation, actorID).Scan(&outcomeID)
		}
		if err != nil {
			return err
		}

		// Audit event (outcome-recorded, not execution)
		meta, _ := json.Marshal(map[string]any{
			"action_id":     actionID.String(),
			"outcome_status": req.OutcomeStatus,
			"is_update":     err == nil && existingID != uuid.Nil,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID,
			Action:      "outcome.recorded",
			EntityType:  "asset_action",
			EntityID:    actionID,
			NewValue:    meta,
		})
		teamIDStr := teamID.String()
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.outcome.recorded",
			AggregateType: "asset_action",
			AggregateID:   actionID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeOutcomeErr(w, http.StatusInternalServerError, "Failed to record outcome")
		return
	}

	// Return the outcome
	h.returnAssetActionOutcome(ctx, w, teamID, outcomeID)
}

// GetAssetActionOutcome returns the outcome for an asset action
func (h *OutcomeHandler) GetAssetActionOutcome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	actionID, _ := uuid.Parse(chi.URLParam(r, "actionId"))

	// Verify action exists and is team-scoped
	var actionStatus string
	err := h.pool.QueryRow(ctx, `
		SELECT status FROM asset_actions WHERE id=$1 AND team_id=$2
	`, actionID, teamID).Scan(&actionStatus)
	if err != nil {
		writeOutcomeErr(w, http.StatusNotFound, "Asset action not found in team scope")
		return
	}

	// Check if outcome exists
	var outcomeID uuid.UUID
	err = h.pool.QueryRow(ctx,
		"SELECT id FROM action_outcomes WHERE asset_action_id=$1 AND team_id=$2",
		actionID, teamID).Scan(&outcomeID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"available":      false,
			"action_id":      actionID.String(),
			"action_status":  actionStatus,
		})
		return
	}

	h.returnAssetActionOutcome(ctx, w, teamID, outcomeID)
}

// CreateOrUpdateRemediationOutcome creates or updates the outcome for a remediation proposal
func (h *OutcomeHandler) CreateOrUpdateRemediationOutcome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	proposalID, err := uuid.Parse(chi.URLParam(r, "proposalId"))
	if err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, "Invalid proposal ID")
		return
	}

	// Verify the remediation exists, is team-scoped, and is in a terminal status
	var proposalStatus string
	err = h.pool.QueryRow(ctx, `
		SELECT status FROM remediation_proposals WHERE id=$1 AND team_id=$2
	`, proposalID, teamID).Scan(&proposalStatus)
	if err != nil {
		writeOutcomeErr(w, http.StatusNotFound, "Remediation proposal not found in team scope")
		return
	}

	// Only completed/failed/cancelled remediations may receive outcomes
	remediationTerminalStatuses := map[string]bool{
		"completed": true, "failed": true, "cancelled": true,
		"executed": true, "succeeded": true,
	}
	if !remediationTerminalStatuses[proposalStatus] {
		writeOutcomeErr(w, http.StatusConflict,
			fmt.Sprintf("Remediation must be in a terminal status to record an outcome. Current: %s", proposalStatus))
		return
	}

	// Parse and validate request
	var req OutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if !validOutcomeStatuses[req.OutcomeStatus] {
		writeOutcomeErr(w, http.StatusBadRequest,
			"outcome_status must be one of: successful, partially_successful, failed, inconclusive")
		return
	}
	if err := validateOutcomeLengths(&req); err != nil {
		writeOutcomeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	sanitizeOutcomeFields(&req)

	cl, ok := iam.GetClaims(r)
	var actorID uuid.UUID
	if ok {
		actorID, _ = uuid.Parse(cl.UserID)
	}

	var outcomeID uuid.UUID
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var existingID uuid.UUID
		err := tx.QueryRow(ctx,
			"SELECT id FROM action_outcomes WHERE remediation_proposal_id=$1 AND team_id=$2",
			proposalID, teamID).Scan(&existingID)

		if err == nil {
			_, err = tx.Exec(ctx, `
				UPDATE action_outcomes
				SET expected_result=$3, actual_result=$4, operator_feedback=$5,
				    outcome_status=$6, follow_up_recommendation=$7, updated_at=NOW()
				WHERE id=$1 AND team_id=$2
			`, existingID, teamID,
				req.ExpectedResult, req.ActualResult, req.OperatorFeedback,
				req.OutcomeStatus, req.FollowUpRecommendation)
			outcomeID = existingID
		} else {
			err = tx.QueryRow(ctx, `
				INSERT INTO action_outcomes (team_id, remediation_proposal_id, expected_result, actual_result,
				    operator_feedback, outcome_status, follow_up_recommendation, created_by)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING id
			`, teamID, proposalID,
				req.ExpectedResult, req.ActualResult, req.OperatorFeedback,
				req.OutcomeStatus, req.FollowUpRecommendation, actorID).Scan(&outcomeID)
		}
		if err != nil {
			return err
		}

		meta, _ := json.Marshal(map[string]any{
			"proposal_id":    proposalID.String(),
			"outcome_status": req.OutcomeStatus,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID,
			Action:     "outcome.recorded",
			EntityType: "remediation_proposal",
			EntityID:   proposalID,
			NewValue:   meta,
		})
		teamIDStr := teamID.String()
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.outcome.recorded",
			AggregateType: "remediation_proposal",
			AggregateID:   proposalID.String(),
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeOutcomeErr(w, http.StatusInternalServerError, "Failed to record outcome")
		return
	}

	h.returnRemediationOutcome(ctx, w, teamID, outcomeID)
}

// GetRemediationOutcome returns the outcome for a remediation proposal
func (h *OutcomeHandler) GetRemediationOutcome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, _ := uuid.Parse(chi.URLParam(r, "teamId"))
	proposalID, _ := uuid.Parse(chi.URLParam(r, "proposalId"))

	var proposalStatus string
	err := h.pool.QueryRow(ctx, `
		SELECT status FROM remediation_proposals WHERE id=$1 AND team_id=$2
	`, proposalID, teamID).Scan(&proposalStatus)
	if err != nil {
		writeOutcomeErr(w, http.StatusNotFound, "Remediation proposal not found in team scope")
		return
	}

	var outcomeID uuid.UUID
	err = h.pool.QueryRow(ctx,
		"SELECT id FROM action_outcomes WHERE remediation_proposal_id=$1 AND team_id=$2",
		proposalID, teamID).Scan(&outcomeID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"available":       false,
			"proposal_id":     proposalID.String(),
			"proposal_status": proposalStatus,
		})
		return
	}

	h.returnRemediationOutcome(ctx, w, teamID, outcomeID)
}

// ─── Helpers ───

func (h *OutcomeHandler) returnAssetActionOutcome(ctx context.Context, w http.ResponseWriter, teamID, outcomeID uuid.UUID) {
	resp, err := fetchOutcome(ctx, h.pool, teamID, outcomeID)
	if err != nil {
		writeOutcomeErr(w, http.StatusInternalServerError, "Failed to fetch outcome")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *OutcomeHandler) returnRemediationOutcome(ctx context.Context, w http.ResponseWriter, teamID, outcomeID uuid.UUID) {
	resp, err := fetchOutcome(ctx, h.pool, teamID, outcomeID)
	if err != nil {
		writeOutcomeErr(w, http.StatusInternalServerError, "Failed to fetch outcome")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func fetchOutcome(ctx context.Context, pool *pgxpool.Pool, teamID, outcomeID uuid.UUID) (*OutcomeResponse, error) {
	var resp OutcomeResponse
	var assetActionID, remediationID *uuid.UUID
	var expected, actual, feedback, followUp *string
	var createdBy uuid.UUID

	err := pool.QueryRow(ctx, `
		SELECT id::text, asset_action_id, remediation_proposal_id,
		       expected_result, actual_result, operator_feedback,
		       outcome_status, follow_up_recommendation,
		       created_by, created_at::text, updated_at::text
		FROM action_outcomes WHERE id=$1 AND team_id=$2
	`, outcomeID, teamID).Scan(
		&resp.ID, &assetActionID, &remediationID,
		&expected, &actual, &feedback,
		&resp.OutcomeStatus, &followUp,
		&createdBy, &resp.CreatedAt, &resp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	resp.ExpectedResult = expected
	resp.ActualResult = actual
	resp.OperatorFeedback = feedback
	resp.FollowUpRecommendation = followUp
	resp.CreatedBy = createdBy.String()

	if assetActionID != nil {
		s := assetActionID.String()
		resp.AssetActionID = &s
	}
	if remediationID != nil {
		s := remediationID.String()
		resp.RemediationProposalID = &s
	}

	return &resp, nil
}

func validateOutcomeLengths(req *OutcomeRequest) error {
	limits := map[*string]int{
		req.ExpectedResult:         2000,
		req.ActualResult:           4000,
		req.OperatorFeedback:       4000,
		req.FollowUpRecommendation: 2000,
	}
	names := map[*string]string{
		req.ExpectedResult:         "expected_result",
		req.ActualResult:           "actual_result",
		req.OperatorFeedback:       "operator_feedback",
		req.FollowUpRecommendation: "follow_up_recommendation",
	}
	for field, limit := range limits {
		if field != nil && len(*field) > limit {
			return fmt.Errorf("%s must be %d characters or less", names[field], limit)
		}
	}
	return nil
}

func sanitizeOutcomeFields(req *OutcomeRequest) {
	fields := []*string{req.ExpectedResult, req.ActualResult, req.OperatorFeedback, req.FollowUpRecommendation}
	for _, f := range fields {
		if f != nil {
			*f = outcomeSensitiveRegex.ReplaceAllString(*f, "[REDACTED]")
			*f = outcomeStandaloneRegex.ReplaceAllString(*f, "[REDACTED]")
		}
	}
}

func writeOutcomeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

// outcome.go is in package proxmox — uses existing imports from action_handler.go
// All imports are resolved at the package level.
