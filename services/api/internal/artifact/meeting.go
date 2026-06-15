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

// ─── Meeting Summary Types ───

type MeetingSummary struct {
	Artifact        Artifact   `json:"artifact"`
	MeetingDate     *string    `json:"meeting_date"`
	DurationMinutes *int       `json:"duration_minutes"`
	Attendees       []any      `json:"attendees"`
	AgendaItems     []any      `json:"agenda_items"`
	Decisions       []any      `json:"decisions"`
	ActionItems     []any      `json:"action_items"`
}

type CreateMeetingRequest struct {
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	ContentMarkdown string  `json:"content_markdown"`
	MeetingDate     *string `json:"meeting_date"`
	DurationMinutes *int    `json:"duration_minutes"`
	Attendees       []any   `json:"attendees"`
	AgendaItems     []any   `json:"agenda_items"`
	Decisions       []any   `json:"decisions"`
	ActionItems     []any   `json:"action_items"`
}

type PatchMeetingRequest struct {
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	ContentMarkdown *string  `json:"content_markdown"`
	Status          *string  `json:"status"`
	MeetingDate     *string  `json:"meeting_date"`
	DurationMinutes *int     `json:"duration_minutes"`
	Attendees       []any    `json:"attendees"`
	AgendaItems     []any    `json:"agenda_items"`
	Decisions       []any    `json:"decisions"`
	ActionItems     []any    `json:"action_items"`
}

// ─── Meeting Summary Handler ───

func (h *Handler) CreateMeetingSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	var req CreateMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, 400, "title is required")
		return
	}
	if req.DurationMinutes != nil && (*req.DurationMinutes < 0 || *req.DurationMinutes > 1440) {
		writeErr(w, 400, "duration_minutes must be between 0 and 1440")
		return
	}
	// Validate array bounds
	if len(req.Attendees) > 100 {
		writeErr(w, 400, "attendees cannot exceed 100 entries")
		return
	}
	if len(req.AgendaItems) > 100 {
		writeErr(w, 400, "agenda_items cannot exceed 100 entries")
		return
	}
	if len(req.Decisions) > 100 {
		writeErr(w, 400, "decisions cannot exceed 100 entries")
		return
	}
	if len(req.ActionItems) > 200 {
		writeErr(w, 400, "action_items cannot exceed 200 entries")
		return
	}

	// Sanitize arrays for sensitive fields
	attendees := sanitizeMeetingArray(req.Attendees)
	agendaItems := sanitizeMeetingArray(req.AgendaItems)
	decisions := sanitizeMeetingArray(req.Decisions)
	actionItems := sanitizeMeetingArray(req.ActionItems)

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	var artifactID string
	var createdAt, updatedAt string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Create artifact
		var id, createdBy string
		err := tx.QueryRow(ctx, `
			INSERT INTO artifacts (team_id, artifact_type, title, description, content_markdown,
			                       status, source_type, source_data, created_by, updated_by)
			VALUES ($1, 'meeting_summary', $2, $3, $4, 'published', 'manual', '[]'::jsonb, $5, $5)
			RETURNING id::text, created_by::text, created_at::text, updated_at::text
		`, teamID, strings.TrimSpace(req.Title), req.Description, req.ContentMarkdown, actorID,
		).Scan(&id, &createdBy, &createdAt, &updatedAt)
		if err != nil {
			return err
		}
		artifactID = id

		// Create meeting data
		attJSON, _ := json.Marshal(attendees)
		agendaJSON, _ := json.Marshal(agendaItems)
		decJSON, _ := json.Marshal(decisions)
		actJSON, _ := json.Marshal(actionItems)

		var meetingDate *time.Time
		if req.MeetingDate != nil && *req.MeetingDate != "" {
			t, err := time.Parse("2006-01-02", *req.MeetingDate)
			if err != nil {
				return fmt.Errorf("invalid meeting_date format: %w", err)
			}
			meetingDate = &t
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO artifact_meeting_data (artifact_id, meeting_date, attendees, agenda_items, decisions, action_items, duration_minutes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, meetingDate, attJSON, agendaJSON, decJSON, actJSON, req.DurationMinutes)
		if err != nil {
			return err
		}

		// Audit
		artID, _ := uuid.Parse(id)
		meta, _ := json.Marshal(map[string]any{
			"artifact_type": "meeting_summary",
			"title":         req.Title,
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.meeting_summary.created",
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
		writeErr(w, 500, "Failed to create meeting summary")
		return
	}

	// Return full meeting summary
	ms := MeetingSummary{
		Artifact: Artifact{
			ID: artifactID, TeamID: teamIDStr, ArtifactType: "meeting_summary",
			Title: req.Title, Description: req.Description, ContentMarkdown: req.ContentMarkdown,
			Status: "published", SourceType: "manual",
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		MeetingDate:     req.MeetingDate,
		DurationMinutes: req.DurationMinutes,
		Attendees:       attendees,
		AgendaItems:     agendaItems,
		Decisions:       decisions,
		ActionItems:     actionItems,
	}
	writeJSON(w, 201, ms)
}

func (h *Handler) ListMeetingSummaries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")

	rows, err := h.pool.Query(ctx, `
		SELECT a.id::text, a.team_id::text, a.title, a.description, a.content_markdown,
		       a.status, a.source_type, a.created_by::text, a.created_at::text, a.updated_at::text,
		       m.meeting_date::text, m.duration_minutes, m.attendees, m.agenda_items, m.decisions, m.action_items
		FROM artifacts a
		JOIN artifact_meeting_data m ON a.id = m.artifact_id
		WHERE a.team_id = $1 AND a.artifact_type = 'meeting_summary' AND a.status != 'archived'
		ORDER BY COALESCE(m.meeting_date, a.created_at) DESC
	`, teamID)
	if err != nil {
		writeErr(w, 500, "Failed to list meeting summaries")
		return
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id, team, title, desc, content, status, source, createdBy, createdAt, updatedAt string
		var meetingDate *string
		var duration *int
		var attendees, agenda, decisions, actions []byte

		if err := rows.Scan(&id, &team, &title, &desc, &content, &status, &source, &createdBy, &createdAt, &updatedAt,
			&meetingDate, &duration, &attendees, &agenda, &decisions, &actions); err != nil {
			continue
		}

		var att, agi, dec, act any
		json.Unmarshal(attendees, &att)
		json.Unmarshal(agenda, &agi)
		json.Unmarshal(decisions, &dec)
		json.Unmarshal(actions, &act)

		out = append(out, map[string]any{
			"id":              id,
			"artifact_type":   "meeting_summary",
			"title":           title,
			"description":     desc,
			"content_markdown": content,
			"status":          status,
			"meeting_date":     meetingDate,
			"duration_minutes": duration,
			"attendees":        att,
			"agenda_items":     agi,
			"decisions":        dec,
			"action_items":     act,
			"created_at":       createdAt,
			"updated_at":       updatedAt,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, 200, out)
}

func (h *Handler) GetMeetingSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	artifactID := chi.URLParam(r, "id")

	var id, title, desc, content, status string
	var meetingDate *string
	var duration *int
	var attendees, agenda, decisions, actions []byte
	var createdAt, updatedAt string

	err := h.pool.QueryRow(ctx, `
		SELECT a.id::text, a.title, a.description, a.content_markdown, a.status,
		       a.created_at::text, a.updated_at::text,
		       m.meeting_date::text, m.duration_minutes, m.attendees, m.agenda_items, m.decisions, m.action_items
		FROM artifacts a
		JOIN artifact_meeting_data m ON a.id = m.artifact_id
		WHERE a.id = $1 AND a.team_id = $2 AND a.artifact_type = 'meeting_summary'
	`, artifactID, teamID).Scan(&id, &title, &desc, &content, &status, &createdAt, &updatedAt,
		&meetingDate, &duration, &attendees, &agenda, &decisions, &actions)
	if err != nil {
		writeErr(w, 404, "Meeting summary not found")
		return
	}

	var att, agi, dec, act any
	json.Unmarshal(attendees, &att)
	json.Unmarshal(agenda, &agi)
	json.Unmarshal(decisions, &dec)
	json.Unmarshal(actions, &act)

	writeJSON(w, 200, map[string]any{
		"id":               id,
		"artifact_type":    "meeting_summary",
		"title":            title,
		"description":      desc,
		"content_markdown": content,
		"status":           status,
		"meeting_date":      meetingDate,
		"duration_minutes":  duration,
		"attendees":         att,
		"agenda_items":      agi,
		"decisions":         dec,
		"action_items":      act,
		"created_at":        createdAt,
		"updated_at":        updatedAt,
	})
}

func (h *Handler) PatchMeetingSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, _ := uuid.Parse(teamIDStr)
	artifactID := chi.URLParam(r, "id")

	// Verify ownership
	var existingType string
	err := h.pool.QueryRow(ctx,
		"SELECT artifact_type FROM artifacts WHERE id = $1 AND team_id = $2",
		artifactID, teamID).Scan(&existingType)
	if err != nil || existingType != "meeting_summary" {
		writeErr(w, 404, "Meeting summary not found")
		return
	}

	var req PatchMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}
	if req.DurationMinutes != nil && (*req.DurationMinutes < 0 || *req.DurationMinutes > 1440) {
		writeErr(w, 400, "duration_minutes must be between 0 and 1440")
		return
	}

	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	// Build updates
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.ContentMarkdown != nil {
		setClauses = append(setClauses, fmt.Sprintf("content_markdown = $%d", argIdx))
		args = append(args, *req.ContentMarkdown)
		argIdx++
	}
	if req.Status != nil && validStatuses[*req.Status] {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	setClauses = append(setClauses, fmt.Sprintf("updated_by = $%d", argIdx))
	args = append(args, actorID)
	argIdx++

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, artifactID, teamID)

	updatedArtFields := len(setClauses) // track if we have art updates

	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		if len(setClauses) > 0 {
			query := fmt.Sprintf("UPDATE artifacts SET %s WHERE id = $%d AND team_id = $%d",
				strings.Join(setClauses, ", "), argIdx, argIdx+1)
			if _, err := tx.Exec(ctx, query, args...); err != nil {
				return err
			}
		}
		_ = updatedArtFields

		// Update meeting data
		mSetClauses := []string{}
		mArgs := []any{}
		mIdx := 1

		if req.MeetingDate != nil {
			if *req.MeetingDate == "" {
				mSetClauses = append(mSetClauses, fmt.Sprintf("meeting_date = NULL"))
			} else {
				t, err := time.Parse("2006-01-02", *req.MeetingDate)
				if err != nil {
					return fmt.Errorf("invalid meeting_date format")
				}
				mSetClauses = append(mSetClauses, fmt.Sprintf("meeting_date = $%d", mIdx))
				mArgs = append(mArgs, t)
				mIdx++
			}
		}
		if req.DurationMinutes != nil {
			mSetClauses = append(mSetClauses, fmt.Sprintf("duration_minutes = $%d", mIdx))
			mArgs = append(mArgs, *req.DurationMinutes)
			mIdx++
		}
		if req.Attendees != nil {
			sanitized := sanitizeMeetingArray(req.Attendees)
			j, _ := json.Marshal(sanitized)
			mSetClauses = append(mSetClauses, fmt.Sprintf("attendees = $%d", mIdx))
			mArgs = append(mArgs, j)
			mIdx++
		}
		if req.AgendaItems != nil {
			sanitized := sanitizeMeetingArray(req.AgendaItems)
			j, _ := json.Marshal(sanitized)
			mSetClauses = append(mSetClauses, fmt.Sprintf("agenda_items = $%d", mIdx))
			mArgs = append(mArgs, j)
			mIdx++
		}
		if req.Decisions != nil {
			sanitized := sanitizeMeetingArray(req.Decisions)
			j, _ := json.Marshal(sanitized)
			mSetClauses = append(mSetClauses, fmt.Sprintf("decisions = $%d", mIdx))
			mArgs = append(mArgs, j)
			mIdx++
		}
		if req.ActionItems != nil {
			sanitized := sanitizeMeetingArray(req.ActionItems)
			j, _ := json.Marshal(sanitized)
			mSetClauses = append(mSetClauses, fmt.Sprintf("action_items = $%d", mIdx))
			mArgs = append(mArgs, j)
			mIdx++
		}

		if len(mSetClauses) > 0 {
			mArgs = append(mArgs, artifactID)
			query := fmt.Sprintf("UPDATE artifact_meeting_data SET %s WHERE artifact_id = $%d",
				strings.Join(mSetClauses, ", "), mIdx)
			if _, err := tx.Exec(ctx, query, mArgs...); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		writeErr(w, 500, "Failed to update meeting summary")
		return
	}

	// Return updated summary
	h.GetMeetingSummary(w, r)
}

// ─── Helpers ───

// sanitizeMeetingArray sanitizes each element of a meeting JSON array.
func sanitizeMeetingArray(arr []any) []any {
	if arr == nil {
		return []any{}
	}
	result := make([]any, len(arr))
	for i, v := range arr {
		result[i] = sanitizeMeetingValue(v)
	}
	return result
}

// sanitizeMeetingValue recursively sanitizes a meeting data value.
func sanitizeMeetingValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, mv := range val {
			if isSensitiveKey(k) {
				result[k] = "[REDACTED]"
			} else {
				result[k] = sanitizeMeetingValue(mv)
			}
		}
		return result
	case []any:
		for i, item := range val {
			val[i] = sanitizeMeetingValue(item)
		}
		return val
	case string:
		return sanitizeSensitiveString(val)
	default:
		return v
	}
}

// sanitizeSensitiveString checks for key=value patterns in strings.
func sanitizeSensitiveString(s string) string {
	lower := strings.ToLower(s)
	for _, pattern := range []string{"password=", "secret=", "token=", "api_key=", "credential="} {
		if idx := strings.Index(lower, pattern); idx >= 0 {
			end := idx + len(pattern)
			// Find end of value (space, comma, quote, or end of string)
			for end < len(s) && s[end] != ' ' && s[end] != ',' && s[end] != '"' && s[end] != '\'' {
				end++
			}
			return s[:idx+len(pattern)] + "[REDACTED]" + s[end:]
		}
	}
	return s
}

// isSensitiveKey is defined in handler.go
