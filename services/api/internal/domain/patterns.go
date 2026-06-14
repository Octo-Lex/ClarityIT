package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Types ───

type Pattern struct {
	PatternID          string            `json:"pattern_id"`
	PatternType        string            `json:"pattern_type"`
	PatternDescription string            `json:"pattern_description"`
	Confidence         float64           `json:"confidence"`
	IncidentIDs        []string          `json:"incident_ids"`
	AssetIDs           []string          `json:"asset_ids,omitempty"`
	AffectedAssets     []AffectedAsset   `json:"affected_assets,omitempty"`
	SeverityMix        map[string]int    `json:"severity_mix,omitempty"`
	FirstSeen          string            `json:"first_seen"`
	LastSeen           string            `json:"last_seen"`
	OccurrenceCount    int               `json:"occurrence_count"`
	AdvisoryOnly       bool              `json:"advisory_only"`
}

type AffectedAsset struct {
	AssetID  string `json:"asset_id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type PatternsResponse struct {
	Patterns []Pattern `json:"patterns"`
}

// incidentRow is the internal representation of a fetched incident for pattern analysis
type incidentRow struct {
	ObjectID     string
	TeamID       string
	Title        string
	Status       string
	Priority     string
	Summary      string
	Severity     string
	Impact       string
	Metadata     map[string]any
	OpenedAt     time.Time
	ResolvedAt   *time.Time
	// Derived asset linkage (filled by asset resolution)
	LinkedAssetIDs []string
}

// ─── Handler ───

type PatternsHandler struct {
	pool *pgxpool.Pool
}

func NewPatternsHandler(pool *pgxpool.Pool) *PatternsHandler {
	return &PatternsHandler{pool: pool}
}

func (h *PatternsHandler) GetPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Invalid team ID")
		return
	}

	// Parse and validate query parameters
	windowDays := 7
	minOccurrences := 2

	if wd := r.URL.Query().Get("window_days"); wd != "" {
		if n, err := parseIntParam(wd); err == nil && n >= 1 && n <= 90 {
			windowDays = n
		} else {
			writeErr(w, http.StatusBadRequest, "window_days must be between 1 and 90")
			return
		}
	}

	if mo := r.URL.Query().Get("min_occurrences"); mo != "" {
		if n, err := parseIntParam(mo); err == nil && n >= 2 && n <= 20 {
			minOccurrences = n
		} else {
			writeErr(w, http.StatusBadRequest, "min_occurrences must be between 2 and 20")
			return
		}
	}

	// Fetch incidents within the window (read-only)
	incidents, err := fetchIncidentsForPatterns(ctx, h.pool, teamID, windowDays)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Failed to fetch incidents")
		return
	}

	// If not enough incidents, return empty list
	if len(incidents) < minOccurrences {
		writeJSON(w, http.StatusOK, PatternsResponse{Patterns: []Pattern{}})
		return
	}

	// Resolve asset links for incidents
	resolveAssetLinks(ctx, h.pool, teamID, incidents)

	// Detect all pattern types
	var patterns []Pattern
	patterns = append(patterns, detectRecurringAsset(incidents, minOccurrences)...)
	patterns = append(patterns, detectRecurringSymptom(incidents, minOccurrences)...)
	patterns = append(patterns, detectCluster(incidents, minOccurrences)...)
	patterns = append(patterns, detectNoisyAsset(incidents, teamID, minOccurrences)...)

	// v1.2 Track 5: Repeated failed outcomes signal
	patterns = append(patterns, detectRepeatedFailedOutcomes(ctx, h.pool, teamID, incidents, minOccurrences)...)

	// Deduplicate: the same set of incidents could trigger multiple pattern types
	// That's fine — each pattern type provides a different lens

	// Sort by confidence descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	if patterns == nil {
		patterns = []Pattern{}
	}

	writeJSON(w, http.StatusOK, PatternsResponse{Patterns: patterns})
}

// ─── Data Fetching ───

func fetchIncidentsForPatterns(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, windowDays int) ([]incidentRow, error) {
	since := time.Now().UTC().AddDate(0, 0, -windowDays)

	rows, err := pool.Query(ctx, `
		SELECT o.id::text, o.team_id::text, o.title, o.status, COALESCE(o.priority, ''),
		       COALESCE(o.summary, ''), i.severity, COALESCE(i.impact, ''),
		       COALESCE(o.metadata, '{}'::jsonb), i.opened_at,
		       i.resolved_at
		FROM objects o
		JOIN incidents i ON i.object_id = o.id
		WHERE o.team_id = $1 AND o.deleted_at IS NULL AND i.opened_at >= $2
		ORDER BY i.opened_at DESC
	`, teamID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []incidentRow
	for rows.Next() {
		var inc incidentRow
		var openedAt time.Time
		var resolvedAt *time.Time
		var metadataBytes []byte

		if err := rows.Scan(&inc.ObjectID, &inc.TeamID, &inc.Title, &inc.Status,
			&inc.Priority, &inc.Summary, &inc.Severity, &inc.Impact,
			&metadataBytes, &openedAt, &resolvedAt); err != nil {
			return nil, err
		}

		inc.OpenedAt = openedAt
		inc.ResolvedAt = resolvedAt
		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &inc.Metadata)
		}
		if inc.Metadata == nil {
			inc.Metadata = map[string]any{}
		}

		incidents = append(incidents, inc)
	}

	return incidents, nil
}

// resolveAssetLinks attempts to find asset links for each incident via multiple strategies:
// 1. metadata->>'asset_id'
// 2. context_edges from incident node to asset node
// 3. affected_service_id matching assets.service_id
func resolveAssetLinks(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, incidents []incidentRow) {
	if len(incidents) == 0 {
		return
	}

	// Build a map of incident object IDs for batch query
	incIDs := make([]string, len(incidents))
	for i, inc := range incidents {
		incIDs[i] = inc.ObjectID
	}

	// Strategy 1: metadata asset references
	for i := range incidents {
		inc := &incidents[i]
		if assetID, ok := inc.Metadata["asset_id"].(string); ok && assetID != "" {
			inc.LinkedAssetIDs = append(inc.LinkedAssetIDs, assetID)
		}
		if assetIDs, ok := inc.Metadata["asset_ids"].([]any); ok {
			for _, aid := range assetIDs {
				if s, ok := aid.(string); ok && s != "" {
					inc.LinkedAssetIDs = append(inc.LinkedAssetIDs, s)
				}
			}
		}
	}

	// Strategy 2: context_edges
	// Find context nodes for incidents and check edges to asset nodes
	edgeRows, err := pool.Query(ctx, `
		SELECT cn_from.entity_id::text as inc_id, cn_to.entity_id::text as asset_id
		FROM context_edges ce
		JOIN context_nodes cn_from ON cn_from.id = ce.from_node_id
		JOIN context_nodes cn_to ON cn_to.id = ce.to_node_id
		WHERE ce.team_id = $1
		  AND cn_from.entity_type = 'incident'
		  AND cn_to.entity_type = 'asset'
	`, teamID)
	if err == nil {
		defer edgeRows.Close()
		incMap := make(map[string]*incidentRow)
		for i := range incidents {
			incMap[incidents[i].ObjectID] = &incidents[i]
		}
		for edgeRows.Next() {
			var incID, assetID string
			if err := edgeRows.Scan(&incID, &assetID); err == nil {
				if inc, ok := incMap[incID]; ok {
					inc.LinkedAssetIDs = append(inc.LinkedAssetIDs, assetID)
				}
			}
		}
	}

	// Deduplicate linked asset IDs per incident
	for i := range incidents {
		incidents[i].LinkedAssetIDs = uniqueStrings(incidents[i].LinkedAssetIDs)
	}
}

// ─── Pattern Detectors ───

// detectRecurringAsset finds assets with multiple incidents in the window
func detectRecurringAsset(incidents []incidentRow, minOccurrences int) []Pattern {
	assetIncidents := make(map[string][]incidentRow)

	for _, inc := range incidents {
		for _, aid := range inc.LinkedAssetIDs {
			assetIncidents[aid] = append(assetIncidents[aid], inc)
		}
	}

	var patterns []Pattern
	for assetID, incs := range assetIncidents {
		if len(incs) < minOccurrences {
			continue
		}

		incIDs := make([]string, len(incs))
		severityMix := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
		for i, inc := range incs {
			incIDs[i] = inc.ObjectID
			mixInto(severityMix, inc.Severity, inc.Priority)
		}

		confidence := computeConfidence(len(incs), minOccurrences)
		first, last := timeRange(incs)

		patterns = append(patterns, Pattern{
			PatternID:          stablePatternID("recurring_asset", assetID, incIDs),
			PatternType:        "recurring_asset",
			PatternDescription: fmt.Sprintf("Asset %s has %d incidents in the analysis window.", assetID, len(incs)),
			Confidence:         confidence,
			IncidentIDs:        incIDs,
			AssetIDs:           []string{assetID},
			AffectedAssets:     []AffectedAsset{{AssetID: assetID}},
			SeverityMix:        severityMix,
			FirstSeen:          first.Format(time.RFC3339),
			LastSeen:           last.Format(time.RFC3339),
			OccurrenceCount:    len(incs),
			AdvisoryOnly:       true,
		})
	}

	return patterns
}

// detectRecurringSymptom finds incidents sharing normalized title keywords
func detectRecurringSymptom(incidents []incidentRow, minOccurrences int) []Pattern {
	// Group by normalized title keywords
	symptomGroups := make(map[string][]incidentRow)

	for _, inc := range incidents {
		key := normalizeSymptom(inc.Title)
		if key == "" {
			continue
		}
		symptomGroups[key] = append(symptomGroups[key], inc)
	}

	var patterns []Pattern
	for symptom, incs := range symptomGroups {
		if len(incs) < minOccurrences {
			continue
		}

		incIDs := make([]string, len(incs))
		severityMix := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
		for i, inc := range incs {
			incIDs[i] = inc.ObjectID
			mixInto(severityMix, inc.Severity, inc.Priority)
		}

		confidence := computeConfidence(len(incs), minOccurrences)
		first, last := timeRange(incs)

		// Sanitize symptom description to avoid leaking potential secrets from titles
		safeSymptom := sanitizeSymptomDisplay(symptom)

		patterns = append(patterns, Pattern{
			PatternID:          stablePatternID("recurring_symptom", symptom, incIDs),
			PatternType:        "recurring_symptom",
			PatternDescription: fmt.Sprintf("%d incidents share a common symptom pattern.", len(incs)),
			Confidence:         confidence,
			IncidentIDs:        incIDs,
			SeverityMix:        severityMix,
			FirstSeen:          first.Format(time.RFC3339),
			LastSeen:           last.Format(time.RFC3339),
			OccurrenceCount:    len(incs),
			AdvisoryOnly:       true,
		})
		_ = safeSymptom // symptom is used internally for grouping, not exposed raw
	}

	return patterns
}

// detectCluster finds incidents close in time with shared characteristics
func detectCluster(incidents []incidentRow, minOccurrences int) []Pattern {
	if len(incidents) < minOccurrences {
		return nil
	}

	// Sort by opened_at ascending
	sorted := make([]incidentRow, len(incidents))
	copy(sorted, incidents)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OpenedAt.Before(sorted[j].OpenedAt)
	})

	// Sliding window: group incidents within 1 hour of each other
	var clusters [][]incidentRow
	var currentCluster []incidentRow

	for i, inc := range sorted {
		if i == 0 {
			currentCluster = []incidentRow{inc}
			continue
		}
		gap := inc.OpenedAt.Sub(sorted[i-1].OpenedAt)
		if gap <= time.Hour {
			currentCluster = append(currentCluster, inc)
		} else {
			if len(currentCluster) >= minOccurrences {
				clusters = append(clusters, currentCluster)
			}
			currentCluster = []incidentRow{inc}
		}
	}
	if len(currentCluster) >= minOccurrences {
		clusters = append(clusters, currentCluster)
	}

	var patterns []Pattern
	for _, cluster := range clusters {
		// Check for shared characteristics
		sharedSeverity := countSharedSeverity(cluster)
		sharedAssets := countSharedAssets(cluster)

		// Only report as cluster if there's some shared characteristic
		if sharedSeverity < 2 && sharedAssets < 2 {
			continue
		}

		incIDs := make([]string, len(cluster))
		severityMix := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
		for i, inc := range cluster {
			incIDs[i] = inc.ObjectID
			mixInto(severityMix, inc.Severity, inc.Priority)
		}

		confidence := computeConfidence(len(cluster), minOccurrences) * 0.9 // slightly lower for clusters
		first, last := timeRange(cluster)

		patterns = append(patterns, Pattern{
			PatternID:          stablePatternID("cluster", "", incIDs),
			PatternType:        "cluster",
			PatternDescription: fmt.Sprintf("%d incidents occurred in close succession within the window.", len(cluster)),
			Confidence:         confidence,
			IncidentIDs:        incIDs,
			SeverityMix:        severityMix,
			FirstSeen:          first.Format(time.RFC3339),
			LastSeen:           last.Format(time.RFC3339),
			OccurrenceCount:    len(cluster),
			AdvisoryOnly:       true,
		})
	}

	return patterns
}

// detectNoisyAsset finds assets with incident counts above the team baseline
func detectNoisyAsset(incidents []incidentRow, teamID uuid.UUID, minOccurrences int) []Pattern {
	assetIncidentCount := make(map[string]int)

	for _, inc := range incidents {
		for _, aid := range inc.LinkedAssetIDs {
			assetIncidentCount[aid]++
		}
	}

	if len(assetIncidentCount) == 0 {
		return nil
	}

	// Compute baseline (average incidents per asset)
	total := 0
	for _, count := range assetIncidentCount {
		total += count
	}
	avg := float64(total) / float64(len(assetIncidentCount))
	threshold := avg * 1.5 // noisy = 1.5x the average (must also meet minOccurrences)

	var patterns []Pattern
	for assetID, count := range assetIncidentCount {
		if float64(count) < threshold || count < minOccurrences {
			continue
		}

		// Collect incidents for this asset
		var incs []incidentRow
		for _, inc := range incidents {
			for _, aid := range inc.LinkedAssetIDs {
				if aid == assetID {
					incs = append(incs, inc)
					break
				}
			}
		}

		incIDs := make([]string, len(incs))
		severityMix := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
		for i, inc := range incs {
			incIDs[i] = inc.ObjectID
			mixInto(severityMix, inc.Severity, inc.Priority)
		}

		confidence := computeConfidence(count, minOccurrences)
		first, last := timeRange(incs)

		patterns = append(patterns, Pattern{
			PatternID:          stablePatternID("noisy_asset", assetID, incIDs),
			PatternType:        "noisy_asset",
			PatternDescription: fmt.Sprintf("Asset %s has %d incidents — above the team baseline.", assetID, count),
			Confidence:         confidence,
			IncidentIDs:        incIDs,
			AssetIDs:           []string{assetID},
			AffectedAssets:     []AffectedAsset{{AssetID: assetID}},
			SeverityMix:        severityMix,
			FirstSeen:          first.Format(time.RFC3339),
			LastSeen:           last.Format(time.RFC3339),
			OccurrenceCount:    count,
			AdvisoryOnly:       true,
		})
	}

	return patterns
}

// ─── Helpers ───

func parseIntParam(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// normalizeSymptom extracts a normalized keyword set from an incident title
func normalizeSymptom(title string) string {
	if title == "" {
		return ""
	}
	// Lowercase, strip non-alphanumeric, collapse spaces
	lower := strings.ToLower(title)
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	cleaned := reg.ReplaceAllString(lower, "")
	words := strings.Fields(cleaned)

	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "on": true, "in": true,
		"at": true, "to": true, "for": true, "of": true, "is": true,
		"was": true, "has": true, "with": true, "and": true, "or": true,
	}

	var meaningful []string
	for _, w := range words {
		if !stopWords[w] && len(w) >= 3 {
			meaningful = append(meaningful, w)
		}
	}

	if len(meaningful) == 0 {
		return ""
	}

	// Sort for stability
	sort.Strings(meaningful)
	return strings.Join(meaningful, " ")
}

// stablePatternID generates a deterministic ID for the same input set
func stablePatternID(patternType, key string, incidentIDs []string) string {
	sorted := make([]string, len(incidentIDs))
	copy(sorted, incidentIDs)
	sort.Strings(sorted)
	h := sha256.Sum256([]byte(patternType + ":" + key + ":" + strings.Join(sorted, ",")))
	return hex.EncodeToString(h[:16])
}

// computeConfidence maps occurrence count to a 0.0–1.0 confidence score
func computeConfidence(count, minOccurrences int) float64 {
	if minOccurrences <= 0 {
		minOccurrences = 2
	}
	// Base confidence from occurrences: 2 = 0.5, 3 = 0.7, 4 = 0.8, 5+ = 0.9+
	base := 0.3 + 0.2*float64(count-minOccurrences+1)
	if base > 0.95 {
		base = 0.95
	}
	return clampConfidence(base)
}

func clampConfidence(c float64) float64 {
	if c < 0.0 {
		return 0.0
	}
	if c > 1.0 {
		return 1.0
	}
	return c
}

// mixInto accumulates severity/priority into the severity mix
func mixInto(mix map[string]int, severity, priority string) {
	// Map severity (sev1-sev5) to standard buckets
	switch severity {
	case "sev1":
		mix["critical"]++
	case "sev2":
		mix["high"]++
	case "sev3":
		mix["medium"]++
	case "sev4", "sev5":
		mix["low"]++
	default:
		// Fall back to priority
		switch strings.ToLower(priority) {
		case "critical":
			mix["critical"]++
		case "high":
			mix["high"]++
		case "medium":
			mix["medium"]++
		case "low":
			mix["low"]++
		}
	}
}

// timeRange returns the earliest and latest opened_at from a set of incidents
func timeRange(incs []incidentRow) (time.Time, time.Time) {
	if len(incs) == 0 {
		return time.Time{}, time.Time{}
	}
	first := incs[0].OpenedAt
	last := incs[0].OpenedAt
	for _, inc := range incs[1:] {
		if inc.OpenedAt.Before(first) {
			first = inc.OpenedAt
		}
		if inc.OpenedAt.After(last) {
			last = inc.OpenedAt
		}
	}
	return first, last
}

func countSharedSeverity(incs []incidentRow) int {
	if len(incs) == 0 {
		return 0
	}
	sev := incs[0].Severity
	count := 0
	for _, inc := range incs {
		if inc.Severity == sev {
			count++
		}
	}
	return count
}

func countSharedAssets(incs []incidentRow) int {
	assetMap := make(map[string]int)
	for _, inc := range incs {
		for _, aid := range inc.LinkedAssetIDs {
			assetMap[aid]++
		}
	}
	maxShared := 0
	for _, count := range assetMap {
		if count > maxShared {
			maxShared = count
		}
	}
	return maxShared
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// sensitiveWordRegex matches common secret-like patterns in text
var sensitiveWordRegex = regexp.MustCompile(`(?i)(password|secret|token|key|credential|api_key|auth)`)

// sanitizeSymptomDisplay removes sensitive-looking keywords from symptom text
func sanitizeSymptomDisplay(s string) string {
	return sensitiveWordRegex.ReplaceAllString(s, "[filtered]")
}

// detectRepeatedFailedOutcomes finds assets with repeated failed/partial outcomes (v1.2 Track 5)
func detectRepeatedFailedOutcomes(ctx context.Context, pool *pgxpool.Pool, teamID uuid.UUID, incidents []incidentRow, minOccurrences int) []Pattern {
	// Query for assets with repeated failed outcomes
	rows, err := pool.Query(ctx, `
		SELECT aa.asset_id::text, COUNT(*) as fail_count
		FROM action_outcomes ao
		JOIN asset_actions aa ON aa.id = ao.asset_action_id
		WHERE ao.team_id = $1
		  AND ao.outcome_status IN ('failed', 'partially_successful')
		GROUP BY aa.asset_id
		HAVING COUNT(*) >= $2
	`, teamID, minOccurrences)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var assetID string
		var failCount int
		if err := rows.Scan(&assetID, &failCount); err != nil {
			continue
		}

		// Get the incident IDs from the window that are linked to this asset
		var incIDs []string
		for _, inc := range incidents {
			for _, aid := range inc.LinkedAssetIDs {
				if aid == assetID {
					incIDs = append(incIDs, inc.ObjectID)
					break
				}
			}
		}
		if len(incIDs) == 0 {
			incIDs = []string{}
		}

		confidence := computeConfidence(failCount, minOccurrences) * 0.85 // slightly lower for outcomes-based
		first, last := timeRange(incidents)

		patterns = append(patterns, Pattern{
			PatternID:          stablePatternID("repeated_failed_outcomes", assetID, incIDs),
			PatternType:        "repeated_failed_outcomes",
			PatternDescription: fmt.Sprintf("Asset %s has %d failed or partially successful action outcomes.", assetID, failCount),
			Confidence:         confidence,
			IncidentIDs:        incIDs,
			AssetIDs:           []string{assetID},
			FirstSeen:          first.Format(time.RFC3339),
			LastSeen:           last.Format(time.RFC3339),
			OccurrenceCount:    failCount,
			AdvisoryOnly:       true,
		})
	}

	return patterns
}
