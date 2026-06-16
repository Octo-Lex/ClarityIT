package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ─── Status Report Types ───

type GenerateStatusReportRequest struct {
	Title           string   `json:"title"`
	ProjectID       *string  `json:"project_id"`
	PeriodStart     string   `json:"period_start"`
	PeriodEnd       string   `json:"period_end"`
	IncludeSections []string `json:"include_sections"`
}

var validSections = map[string]bool{
	"summary":       true,
	"milestones":    true,
	"risks":         true,
	"incidents":     true,
	"metrics":       true,
	"asset_actions": true,
	"remediations":  true,
}

const maxReportRangeDays = 366
const maxReportTitleLen = 200

// ─── Handler ───

func (h *Handler) GenerateStatusReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	var req GenerateStatusReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	// Validate title
	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, 400, "title is required")
		return
	}
	if len(req.Title) > maxReportTitleLen {
		writeErr(w, 400, fmt.Sprintf("title must be <= %d characters", maxReportTitleLen))
		return
	}

	// Validate date range
	if req.PeriodStart == "" || req.PeriodEnd == "" {
		writeErr(w, 400, "period_start and period_end are required")
		return
	}
	startDate, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		writeErr(w, 400, "period_start must be YYYY-MM-DD")
		return
	}
	endDate, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		writeErr(w, 400, "period_end must be YYYY-MM-DD")
		return
	}
	if startDate.After(endDate) {
		writeErr(w, 400, "period_start must be <= period_end")
		return
	}
	if endDate.Sub(startDate).Hours()/24 > maxReportRangeDays {
		writeErr(w, 400, fmt.Sprintf("date range cannot exceed %d days", maxReportRangeDays))
		return
	}

	// Validate sections
	if len(req.IncludeSections) == 0 {
		req.IncludeSections = []string{"summary", "incidents", "metrics"}
	}
	for _, s := range req.IncludeSections {
		if !validSections[s] {
			writeErr(w, 400, fmt.Sprintf("invalid section: %s", s))
			return
		}
	}

	// Validate project_id if provided
	if req.ProjectID != nil && *req.ProjectID != "" {
		pid, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			writeErr(w, 400, "Invalid project_id")
			return
		}
		var belongs bool
		err = h.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM objects WHERE id=$1 AND team_id=$2 AND object_type='project')",
			pid, teamID).Scan(&belongs)
		if err != nil || !belongs {
			writeErr(w, 400, "Project not found in this team")
			return
		}
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	// Generate markdown report
	markdown := h.buildStatusReportMarkdown(ctx, teamID, teamIDStr, &req, startDate, endDate)

	// Sanitize source_data — no raw sensitive data
	sourceData, _ := json.Marshal(map[string]any{
		"project_id":       req.ProjectID,
		"period_start":     req.PeriodStart,
		"period_end":       req.PeriodEnd,
		"include_sections": req.IncludeSections,
	})

	var artifactID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO artifacts (team_id, artifact_type, title, content_markdown,
			                       status, source_type, source_data, created_by, updated_by)
			VALUES ($1, 'status_report', $2, $3, 'draft', 'generated', $4, $5, $5)
			RETURNING id::text
		`, teamID, strings.TrimSpace(req.Title), markdown, sourceData, actorID).Scan(&id)
		if err != nil {
			return err
		}
		artifactID = id

		artID, _ := uuid.Parse(id)
		meta, _ := json.Marshal(map[string]any{
			"artifact_type": "status_report",
			"title":         req.Title,
			"period_start":  req.PeriodStart,
			"period_end":    req.PeriodEnd,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.status_report.generated",
			EntityType: "artifact", EntityID: artID, NewValue: meta,
		})
		_ = outbox.Write(ctx, tx, &teamIDStr, outbox.Event{
			EventType:     "clarity.v1.artifact.created",
			AggregateType: "artifact",
			AggregateID:   id,
			Payload:       meta,
		})
		return nil
	})
	if err != nil {
		writeErr(w, 500, "Failed to create status report")
		return
	}

	writeJSON(w, 201, map[string]any{
		"artifact_id":      artifactID,
		"artifact_type":    "status_report",
		"title":            req.Title,
		"status":           "draft",
		"content_markdown": markdown,
	})

	// v1.5 Knowledge index hook
	h.fireIndexHook(ctx, teamIDStr, "artifact", artifactID)
}

// ─── Report Builder ───

func (h *Handler) buildStatusReportMarkdown(ctx context.Context, teamID uuid.UUID, teamIDStr string, req *GenerateStatusReportRequest, startDate, endDate time.Time) string {
	var sb strings.Builder
	now := time.Now().UTC().Format("2006-01-02 15:04 MST")

	sb.WriteString(fmt.Sprintf("# %s\n\n", req.Title))
	sb.WriteString(fmt.Sprintf("**Period:** %s to %s\n\n", req.PeriodStart, req.PeriodEnd))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", now))

	for _, section := range req.IncludeSections {
		switch section {
		case "summary":
			h.writeSummarySection(ctx, &sb, teamID, startDate, endDate)
		case "milestones":
			h.writeMilestonesSection(ctx, &sb, teamID, startDate, endDate, req.ProjectID)
		case "risks":
			h.writeRisksSection(ctx, &sb, teamID, req.ProjectID)
		case "incidents":
			h.writeIncidentsSection(ctx, &sb, teamID, startDate, endDate)
		case "metrics":
			h.writeMetricsSection(ctx, &sb, teamID, startDate, endDate)
		case "asset_actions":
			h.writeAssetActionsSection(ctx, &sb, teamID, startDate, endDate)
		case "remediations":
			h.writeRemediationsSection(ctx, &sb, teamID, startDate, endDate)
		}
	}

	return sb.String()
}

func (h *Handler) writeSummarySection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, start, end time.Time) {
	sb.WriteString("## Summary\n\n")

	// Count work items in range
	var workItemCount int
	h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM objects WHERE team_id=$1 AND object_type='work_item' AND created_at >= $2 AND created_at < $3 + interval '1 day'`,
		teamID, start, end).Scan(&workItemCount)

	var incidentCount int
	h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incidents i JOIN objects o ON i.object_id=o.id WHERE o.team_id=$1 AND i.opened_at >= $2 AND i.opened_at < $3 + interval '1 day'`,
		teamID, start, end).Scan(&incidentCount)

	var openIncidents int
	h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incidents i JOIN objects o ON i.object_id=o.id WHERE o.team_id=$1 AND i.resolved_at IS NULL`,
		teamID).Scan(&openIncidents)

	sb.WriteString(fmt.Sprintf("- **Work Items Created:** %d\n", workItemCount))
	sb.WriteString(fmt.Sprintf("- **Incidents Opened:** %d\n", incidentCount))
	sb.WriteString(fmt.Sprintf("- **Currently Open Incidents:** %d\n\n", openIncidents))
}

func (h *Handler) writeMilestonesSection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, start, end time.Time, projectID *string) {
	sb.WriteString("## Milestones\n\n")

	query := `SELECT o.title, o.status, o.updated_at::text FROM objects o WHERE o.team_id=$1 AND o.object_type='milestone'`
	args := []any{teamID}
	if projectID != nil && *projectID != "" {
		query += ` AND o.metadata->>'project_id' = $2`
		args = append(args, *projectID)
	}
	query += ` ORDER BY o.updated_at DESC LIMIT 50`

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		sb.WriteString("_No milestones found._\n\n")
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var title, status, updated string
		rows.Scan(&title, &status, &updated)
		sb.WriteString(fmt.Sprintf("- [%s] %s _(updated %s)_\n", strings.ToUpper(status), title, updated[:10]))
		count++
	}
	if count == 0 {
		sb.WriteString("_No milestones found in this period._\n")
	}
	sb.WriteString("\n")
}

func (h *Handler) writeRisksSection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, projectID *string) {
	sb.WriteString("## Risks\n\n")

	query := `SELECT o.title, o.status, o.priority FROM objects o WHERE o.team_id=$1 AND o.object_type='risk'`
	args := []any{teamID}
	if projectID != nil && *projectID != "" {
		query += ` AND o.metadata->>'project_id' = $2`
		args = append(args, *projectID)
	}
	query += ` ORDER BY o.priority DESC LIMIT 50`

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		sb.WriteString("_No risks identified._\n\n")
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var title, status, priority string
		rows.Scan(&title, &status, &priority)
		sb.WriteString(fmt.Sprintf("- [%s/%s] %s\n", strings.ToUpper(status), strings.ToUpper(priority), title))
		count++
	}
	if count == 0 {
		sb.WriteString("_No risks identified._\n")
	}
	sb.WriteString("\n")
}

func (h *Handler) writeIncidentsSection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, start, end time.Time) {
	sb.WriteString("## Incidents\n\n")

	rows, err := h.pool.Query(ctx,
		`SELECT o.title, i.severity, i.opened_at::text, COALESCE(i.resolved_at::text, 'open')
		 FROM incidents i JOIN objects o ON i.object_id=o.id
		 WHERE o.team_id=$1 AND i.opened_at >= $2 AND i.opened_at < $3 + interval '1 day'
		 ORDER BY i.opened_at DESC`,
		teamID, start, end)
	if err != nil {
		sb.WriteString("_No incidents in this period._\n\n")
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var title, severity, opened, resolved string
		rows.Scan(&title, &severity, &opened, &resolved)
		// No raw incident body/impact text — only title + severity + dates
		sb.WriteString(fmt.Sprintf("- **%s** — %s — opened %s — %s\n",
			strings.ToUpper(severity), title, opened[:10],
			func() string { if resolved == "open" { return "OPEN" }; return "resolved " + resolved[:10] }()))
		count++
	}
	if count == 0 {
		sb.WriteString("_No incidents in this period._\n")
	}
	sb.WriteString("\n")
}

func (h *Handler) writeMetricsSection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, start, end time.Time) {
	sb.WriteString("## Metrics\n\n")

	// Work items by status
	rows, _ := h.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM objects WHERE team_id=$1 AND object_type='work_item'
		 AND created_at >= $2 AND created_at < $3 + interval '1 day' GROUP BY status`,
		teamID, start, end)
	defer rows.Close()

	sb.WriteString("### Work Items by Status\n\n")
	count := 0
	for rows.Next() {
		var status string
		var cnt int
		rows.Scan(&status, &cnt)
		sb.WriteString(fmt.Sprintf("- %s: %d\n", status, cnt))
		count++
	}
	if count == 0 {
		sb.WriteString("_No work items in this period._\n")
	}
	sb.WriteString("\n")
}

func (h *Handler) writeAssetActionsSection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, start, end time.Time) {
	sb.WriteString("## Asset Actions\n\n")

	// Count by status — no action_target or parameters
	rows, _ := h.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM asset_actions WHERE team_id=$1
		 AND created_at >= $2 AND created_at < $3 + interval '1 day' GROUP BY status`,
		teamID, start, end)
	defer rows.Close()

	sb.WriteString("### Actions by Status\n\n")
	count := 0
	for rows.Next() {
		var status string
		var cnt int
		rows.Scan(&status, &cnt)
		sb.WriteString(fmt.Sprintf("- %s: %d\n", status, cnt))
		count++
	}
	if count == 0 {
		sb.WriteString("_No asset actions in this period._\n")
	}
	sb.WriteString("\n")
}

func (h *Handler) writeRemediationsSection(ctx context.Context, sb *strings.Builder, teamID uuid.UUID, start, end time.Time) {
	sb.WriteString("## Remediations\n\n")

	// Count by status — no tool parameters or action targets
	rows, _ := h.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM remediation_proposals WHERE team_id=$1
		 AND created_at >= $2 AND created_at < $3 + interval '1 day' GROUP BY status`,
		teamID, start, end)
	defer rows.Close()

	sb.WriteString("### Remediations by Status\n\n")
	count := 0
	for rows.Next() {
		var status string
		var cnt int
		rows.Scan(&status, &cnt)
		sb.WriteString(fmt.Sprintf("- %s: %d\n", status, cnt))
		count++
	}
	if count == 0 {
		sb.WriteString("_No remediations in this period._\n")
	}
	sb.WriteString("\n")
}
