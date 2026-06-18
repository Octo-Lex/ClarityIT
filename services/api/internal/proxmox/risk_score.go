package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ───

type RiskFactor struct {
	Factor      string `json:"factor"`
	Label       string `json:"label"`
	Score       int    `json:"score"`
	Explanation string `json:"explanation"`
}

type RiskScoreInputs struct {
	RecentIncidentWindowDays int    `json:"recent_incident_window_days"`
	MutationWindowActive     bool   `json:"mutation_window_active"`
	ChangeWindowStatus       string `json:"change_window_status"`
}

type RiskScoreResponse struct {
	AssetID         string        `json:"asset_id"`
	ActionType      string        `json:"action_type"`
	RiskScore       int           `json:"risk_score"`
	RiskLevel       string        `json:"risk_level"`
	AdvisoryOnly    bool          `json:"advisory_only"`
	Factors         []RiskFactor  `json:"factors"`
	MitigationNotes []string      `json:"mitigation_notes"`
	Inputs          RiskScoreInputs `json:"inputs"`
}

// ─── Handler ───

type RiskScoreHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewRiskScoreHandler(pool *pgxpool.Pool, cfg *config.Config) *RiskScoreHandler {
	return &RiskScoreHandler{pool: pool, cfg: cfg}
}

func (h *RiskScoreHandler) GetRiskScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeRiskErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}
	assetID, err := uuid.Parse(chi.URLParam(r, "assetId"))
	if err != nil {
		writeRiskErr(w, http.StatusBadRequest, "Invalid asset ID")
		return
	}

	actionType := r.URL.Query().Get("action")
	if actionType == "" {
		writeRiskErr(w, http.StatusBadRequest, "action query parameter is required")
		return
	}

	// Validate action is in the allowlist
	actionInfo, ok := allowedActions[actionType]
	if !ok {
		writeRiskErr(w, http.StatusBadRequest,
			fmt.Sprintf("Unsupported action: %s. Allowed: proxmox.start, proxmox.shutdown, proxmox.stop, proxmox.snapshot", actionType))
		return
	}

	// Fetch asset metadata (team-scoped)
	var hostname, assetType, provider, priority string
	var metadataBytes []byte
	err = h.pool.QueryRow(ctx, `
		SELECT COALESCE(a.hostname, o.title), COALESCE(a.asset_type, ''), COALESCE(a.provider, ''),
		       COALESCE(o.priority, 'none'), COALESCE(o.metadata, '{}'::jsonb)
		FROM objects o
		JOIN assets a ON a.object_id = o.id
		WHERE o.id = $1 AND o.team_id = $2 AND o.deleted_at IS NULL
	`, assetID, teamID).Scan(&hostname, &assetType, &provider, &priority, &metadataBytes)

	if err != nil {
		writeRiskErr(w, http.StatusNotFound, "Asset not found in team scope")
		return
	}

	// Parse asset metadata
	var assetMeta map[string]any
	if len(metadataBytes) > 0 {
		json.Unmarshal(metadataBytes, &assetMeta)
	}

	// Compute risk score from factors
	score, factors, mitigationNotes, inputs := computeRiskScore(ctx, h.pool, h.cfg, teamID, assetID, actionType, actionInfo, priority, assetMeta)

	// Build response
	resp := RiskScoreResponse{
		AssetID:         assetID.String(),
		ActionType:      actionType,
		RiskScore:       score,
		RiskLevel:       scoreToLevel(score),
		AdvisoryOnly:    true,
		Factors:         factors,
		MitigationNotes: mitigationNotes,
		Inputs:          inputs,
	}

	writeJSON(w, http.StatusOK, resp)
}

// ─── Risk Scoring Logic ───

func computeRiskScore(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *config.Config,
	teamID uuid.UUID,
	assetID uuid.UUID,
	actionType string,
	actionInfo struct {
		riskLevel    string
		requiresMFA  bool
		minApprovers int
	},
	assetPriority string,
	assetMeta map[string]any,
) (int, []RiskFactor, []string, RiskScoreInputs) {
	var factors []RiskFactor
	var mitigations []string
	totalScore := 0

	// ─── Factor 1: Action Type Risk ───
	actionScores := map[string]int{
		"proxmox.start":    15,
		"proxmox.snapshot": 20,
		"proxmox.shutdown": 30,
		"proxmox.stop":     40,
	}
	actionLabels := map[string]string{
		"proxmox.start":    "Start is a low-impact action.",
		"proxmox.snapshot": "Snapshot is a read-only operation with minimal risk.",
		"proxmox.shutdown": "Shutdown is a high-risk action.",
		"proxmox.stop":     "Force stop is a critical-risk action — data loss possible.",
	}
	actionScore := actionScores[actionType]
	totalScore += actionScore
	factors = append(factors, RiskFactor{
		Factor:      "action_type",
		Label:       "Action type risk",
		Score:       actionScore,
		Explanation: actionLabels[actionType],
	})

	// ─── Factor 2: Asset Criticality ───
	criticalityScore := 5 // base
	criticalityExplanation := "Standard asset priority."
	switch assetPriority {
	case "critical":
		criticalityScore = 20
		criticalityExplanation = "Asset has critical priority."
	case "high":
		criticalityScore = 15
		criticalityExplanation = "Asset has high priority."
	case "medium":
		criticalityScore = 8
		criticalityExplanation = "Asset has medium priority."
	case "low", "none":
		criticalityScore = 3
		criticalityExplanation = "Asset has low/standard priority."
	}
	// Check metadata for additional criticality hints
	if tags, ok := assetMeta["tags"].([]any); ok {
		for _, tag := range tags {
			if s, ok := tag.(string); ok {
				if s == "production" || s == "prod" {
					criticalityScore += 5
					criticalityExplanation += " Tagged as production."
				}
				if s == "database" || s == "db" {
					criticalityScore += 5
					criticalityExplanation += " Tagged as database."
				}
			}
		}
	}
	totalScore += criticalityScore
	factors = append(factors, RiskFactor{
		Factor:      "asset_criticality",
		Label:       "Asset criticality",
		Score:       criticalityScore,
		Explanation: criticalityExplanation,
	})

	// ─── Factor 3: Recent Incidents ───
	windowDays := 7
	recentIncidentScore := 0
	var incidentCount int
	var highSeverityCount int

	pool.QueryRow(ctx, `
		SELECT COUNT(*), 0
		FROM objects o
		JOIN incidents i ON i.object_id = o.id
		WHERE o.team_id = $1 AND o.deleted_at IS NULL
		  AND i.opened_at >= NOW() - INTERVAL '%d days'
	`, teamID).Scan(&incidentCount, &highSeverityCount) // fallback; actual query below

	// Better query that also checks context_edges for asset-linked incidents
	incidentRows, _ := pool.Query(ctx, `
		SELECT i.severity
		FROM objects o
		JOIN incidents i ON i.object_id = o.id
		WHERE o.team_id = $1 AND o.deleted_at IS NULL
		  AND i.opened_at >= NOW() - INTERVAL '7 days'
		  AND (
		    o.metadata->>'asset_id' = $2
		    OR EXISTS (
		      SELECT 1 FROM context_edges ce
		      JOIN context_nodes cn_from ON cn_from.id = ce.from_node_id
		      JOIN context_nodes cn_to ON cn_to.id = ce.to_node_id
		      WHERE ce.team_id = $1
		        AND cn_from.entity_type = 'incident'
		        AND cn_from.entity_id = o.id
		        AND cn_to.entity_type = 'asset'
		        AND cn_to.entity_id = $3::uuid
		    )
		  )
	`, teamID, assetID.String(), assetID)
	if incidentRows != nil {
		for incidentRows.Next() {
			var sev string
			incidentRows.Scan(&sev)
			recentIncidentScore += 5
			if sev == "sev1" || sev == "sev2" {
				highSeverityCount++
				recentIncidentScore += 5
			}
		}
		incidentRows.Close()
	}

	if recentIncidentScore > 25 {
		recentIncidentScore = 25
	}

	recentExplanation := "No recent incidents linked to this asset."
	if incidentCount > 0 {
		recentExplanation = fmt.Sprintf("%d incidents linked to this asset in the last %d days.", incidentCount, windowDays)
		if highSeverityCount > 0 {
			recentExplanation += fmt.Sprintf(" %d were high-severity.", highSeverityCount)
		}
	}

	totalScore += recentIncidentScore
	factors = append(factors, RiskFactor{
		Factor:      "recent_incidents",
		Label:       "Recent incidents",
		Score:       recentIncidentScore,
		Explanation: recentExplanation,
	})

	// ─── Factor 4: Blast Radius ───
	// Count how many other assets/services depend on this asset via context_edges
	var dependentCount int
	pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT cn_from.entity_id)
		FROM context_edges ce
		JOIN context_nodes cn_from ON cn_from.id = ce.from_node_id
		JOIN context_nodes cn_to ON cn_to.id = ce.to_node_id
		WHERE ce.team_id = $1
		  AND cn_to.entity_type = 'asset' AND cn_to.entity_id = $2::uuid
		  AND cn_from.entity_type IN ('asset', 'service')
	`, teamID, assetID).Scan(&dependentCount)

	blastRadiusScore := 0
	blastExplanation := "No known dependencies on this asset."
	if dependentCount > 0 {
		blastRadiusScore = dependentCount * 5
		if blastRadiusScore > 20 {
			blastRadiusScore = 20
		}
		blastExplanation = fmt.Sprintf("%d assets/services depend on this asset.", dependentCount)
	}

	totalScore += blastRadiusScore
	factors = append(factors, RiskFactor{
		Factor:      "blast_radius",
		Label:       "Blast radius",
		Score:       blastRadiusScore,
		Explanation: blastExplanation,
	})

	// ─── Factor 5: Time of Day ───
	hour := time.Now().UTC().Hour()
	timeScore := 0
	timeExplanation := "Change during normal business hours."

	if hour >= 22 || hour < 6 {
		timeScore = 10
		timeExplanation = "Change during off-hours (night) — reduced staff availability."
	} else if hour >= 18 || hour < 8 {
		timeScore = 5
		timeExplanation = "Change outside core business hours."
	}

	totalScore += timeScore
	factors = append(factors, RiskFactor{
		Factor:      "time_of_day",
		Label:       "Time of day",
		Score:       timeScore,
		Explanation: timeExplanation,
	})

	// ─── Factor 6: Mutation Window Status ───
	mutationWindowActive := HasActiveMutationWindow(ctx, pool)
	featureFlagEnabled := cfg.ProxmoxMutationEnabled
	changeWindowStatus := "closed"
	if mutationWindowActive {
		changeWindowStatus = "active"
	} else if !featureFlagEnabled {
		changeWindowStatus = "disabled"
	}

	mwScore := 0
	mwExplanation := "No active mutation window."
	if !mutationWindowActive {
		mwScore = 5
		mwExplanation = "No active mutation window — change would require opening one."
	}
	if !featureFlagEnabled {
		mwScore += 5
		mwExplanation += " Feature flag is disabled."
	}

	totalScore += mwScore
	factors = append(factors, RiskFactor{
		Factor:      "mutation_window_status",
		Label:       "Mutation window status",
		Score:       mwScore,
		Explanation: mwExplanation,
	})

	// ─── Factor 7: Past Outcomes (v1.2 Track 5 feedback signal) ───
	var pastOutcomeCount int
	var failedOutcomeCount int
	pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(CASE WHEN outcome_status IN ('failed', 'partially_successful') THEN 1 END)
		FROM action_outcomes ao
		JOIN asset_actions aa ON aa.id = ao.asset_action_id
		WHERE ao.team_id = $1 AND aa.asset_id = $2::uuid
	`, teamID, assetID).Scan(&pastOutcomeCount, &failedOutcomeCount)

	pastOutcomeScore := 0
	pastOutcomeExplanation := "No past action outcomes for this asset."
	if pastOutcomeCount > 0 {
		pastOutcomeScore = pastOutcomeCount * 3
		if pastOutcomeScore > 10 {
			pastOutcomeScore = 10
		}
		pastOutcomeExplanation = fmt.Sprintf("%d past action outcome(s) recorded for this asset", pastOutcomeCount)
		if failedOutcomeCount > 0 {
			pastOutcomeExplanation += fmt.Sprintf(" (%d failed/partial)", failedOutcomeCount)
			pastOutcomeScore += 5
		}
	}

	totalScore += pastOutcomeScore
	factors = append(factors, RiskFactor{
		Factor:      "past_outcomes",
		Label:       "Past outcomes",
		Score:       pastOutcomeScore,
		Explanation: pastOutcomeExplanation,
	})

	// ─── Mitigation Notes ───
	if actionType == "proxmox.shutdown" || actionType == "proxmox.stop" {
		mitigations = append(mitigations, "Confirm service owner before shutdown/stop.")
	}
	if !mutationWindowActive {
		mitigations = append(mitigations, "Open an active mutation window before proceeding.")
	}
	if recentIncidentScore > 10 {
		mitigations = append(mitigations, "Asset has recent incidents — verify stability before change.")
	}
	if blastRadiusScore > 10 {
		mitigations = append(mitigations, "Multiple dependencies detected — coordinate with dependent service owners.")
	}
	if assetPriority == "critical" || assetPriority == "high" {
		mitigations = append(mitigations, "Verify backup freshness before high-risk change.")
	}
	if len(mitigations) == 0 {
		mitigations = append(mitigations, "Standard change procedures apply.")
	}

	// Clamp total score 0-100
	if totalScore > 100 {
		totalScore = 100
	}
	if totalScore < 0 {
		totalScore = 0
	}

	inputs := RiskScoreInputs{
		RecentIncidentWindowDays: windowDays,
		MutationWindowActive:     mutationWindowActive,
		ChangeWindowStatus:       changeWindowStatus,
	}

	return totalScore, factors, mitigations, inputs
}

// ─── Helpers ───

func scoreToLevel(score int) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 50:
		return "high"
	case score >= 25:
		return "medium"
	default:
		return "low"
	}
}

// computeRiskScoreSummary is used by dry-run to include a compact risk score
func computeRiskScoreSummary(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *config.Config,
	teamID uuid.UUID,
	assetID uuid.UUID,
	actionType string,
) map[string]any {
	actionInfo, ok := allowedActions[actionType]
	if !ok {
		return map[string]any{
			"score":         0,
			"level":         "unknown",
			"advisory_only": true,
			"top_factors":   []string{},
		}
	}

	// Fetch asset priority
	var priority string
	var metadataBytes []byte
	pool.QueryRow(ctx, `
		SELECT COALESCE(o.priority, 'none'), COALESCE(o.metadata, '{}'::jsonb)
		FROM objects o JOIN assets a ON a.object_id = o.id
		WHERE o.id = $1 AND o.team_id = $2 AND o.deleted_at IS NULL
	`, assetID, teamID).Scan(&priority, &metadataBytes)

	var assetMeta map[string]any
	if len(metadataBytes) > 0 {
		json.Unmarshal(metadataBytes, &assetMeta)
	}

	score, factors, _, _ := computeRiskScore(ctx, pool, cfg, teamID, assetID, actionType, actionInfo, priority, assetMeta)

	// Get top 3 factors by score
	topFactors := []string{}
	// Sort factors by score descending (simple selection)
	for i := 0; i < len(factors) && len(topFactors) < 3; i++ {
		maxIdx := i
		for j := i + 1; j < len(factors); j++ {
			if factors[j].Score > factors[maxIdx].Score {
				maxIdx = j
			}
		}
		if maxIdx != i {
			factors[i], factors[maxIdx] = factors[maxIdx], factors[i]
		}
		if factors[i].Score > 0 {
			topFactors = append(topFactors, factors[i].Factor)
		}
	}

	return map[string]any{
		"score":         score,
		"level":         scoreToLevel(score),
		"advisory_only": true,
		"top_factors":   topFactors,
	}
}

func writeRiskErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

// Suppress unused import warnings for types used in dry-run integration
var _ = audit.Write
var _ = outbox.Write
var _ = database.WithTx
