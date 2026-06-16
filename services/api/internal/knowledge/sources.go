package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Source Extractors ───
// Each function reads from a specific ClarityIT table and produces
// a SourceDocument suitable for indexing.

// ExtractClarityDocuments indexes artifact_documents (v1.4 ClarityDocs).
func ExtractClarityDocuments(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.id::text, a.team_id::text, a.title, a.status, a.updated_at,
		       d.document_type, d.document_json, d.word_count
		FROM artifacts a
		JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.team_id = $1::uuid
		  AND a.artifact_type = 'document'
		  AND a.status != 'archived'
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query clarity_documents: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title, status, docType string
		var updatedAt time.Time
		var docJSON []byte
		var wordCount int

		if err := rows.Scan(&id, &tid, &title, &status, &updatedAt, &docType, &docJSON, &wordCount); err != nil {
			continue
		}

		content := extractDocumentText(docJSON)

		docs = append(docs, SourceDocument{
			SourceType:      "clarity_document",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         fmt.Sprintf("%s — %d words", docType, wordCount),
			ContentText:     content,
			Metadata: map[string]any{
				"document_type": docType,
				"word_count":    wordCount,
				"status":        status,
			},
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractArtifacts indexes artifacts with content_markdown (non-document types).
func ExtractArtifacts(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, team_id::text, title, description, content_markdown,
		       artifact_type, status, updated_at
		FROM artifacts
		WHERE team_id = $1::uuid
		  AND artifact_type != 'document'
		  AND status != 'archived'
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query artifacts: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title, artType, status string
		var description, content *string
		var updatedAt time.Time

		if err := rows.Scan(&id, &tid, &title, &description, &content, &artType, &status, &updatedAt); err != nil {
			continue
		}

		title = title
		contentText := ""
		if content != nil {
			contentText = *content
		}
		summary := ""
		if description != nil {
			summary = *description
		}

		docs = append(docs, SourceDocument{
			SourceType:      "artifact",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         summary,
			ContentText:     contentText,
			Metadata: map[string]any{
				"artifact_type": artType,
				"status":        status,
			},
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractMeetingSummaries indexes meeting summary artifacts.
func ExtractMeetingSummaries(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.id::text, a.team_id::text, a.title, a.updated_at,
		       m.meeting_date, m.attendees, m.agenda_items, m.decisions, m.action_items
		FROM artifacts a
		JOIN artifact_meeting_data m ON m.artifact_id = a.id
		WHERE a.team_id = $1::uuid
		  AND a.artifact_type = 'meeting_summary'
		  AND a.status != 'archived'
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query meeting_summaries: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title string
		var updatedAt time.Time
		var meetingDate *time.Time
		var attendees, agenda, decisions, actions []byte

		if err := rows.Scan(&id, &tid, &title, &updatedAt, &meetingDate, &attendees, &agenda, &decisions, &actions); err != nil {
			continue
		}

		var sb strings.Builder
		sb.WriteString(title + "\n\n")
		sb.WriteString("Meeting Date: ")
		if meetingDate != nil {
			sb.WriteString(meetingDate.Format("2006-01-02"))
		}
		sb.WriteString("\n\n")

		appendJSONArray(&sb, "Attendees", attendees)
		appendJSONArray(&sb, "Agenda", agenda)
		appendJSONArray(&sb, "Decisions", decisions)
		appendJSONArray(&sb, "Action Items", actions)

		docs = append(docs, SourceDocument{
			SourceType:      "meeting_summary",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         "Meeting summary",
			ContentText:     sb.String(),
			Metadata:        map[string]any{"type": "meeting_summary"},
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractTemplates indexes artifact_templates.
func ExtractTemplates(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, team_id::text, name, description, content_markdown,
		       template_type, template_format, document_json, created_at
		FROM artifact_templates
		WHERE team_id = $1::uuid
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, name, tplType string
		var description, content *string
		var tplFormat *string
		var docJSON []byte
		var createdAt time.Time

		if err := rows.Scan(&id, &tid, &name, &description, &content, &tplType, &tplFormat, &docJSON, &createdAt); err != nil {
			continue
		}

		contentText := ""
		if content != nil {
			contentText = *content
		}
		if tplFormat != nil && *tplFormat == "document_json" && len(docJSON) > 0 {
			contentText = extractDocumentText(docJSON)
		}
		summary := ""
		if description != nil {
			summary = *description
		}

		docs = append(docs, SourceDocument{
			SourceType:      "template",
			SourceID:        id,
			TeamID:          tid,
			Title:           name,
			Summary:         summary,
			ContentText:     contentText,
			Metadata: map[string]any{
				"template_type":   tplType,
				"template_format": tplFormat,
			},
			SourceUpdatedAt: createdAt,
		})
	}
	return docs, nil
}

// ExtractWorkItems indexes work items (tasks, tickets).
func ExtractWorkItems(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT o.id::text, o.team_id::text, o.title, o.summary, o.status,
		       o.priority, o.updated_at, w.work_item_type
		FROM objects o
		JOIN work_items w ON w.object_id = o.id
		WHERE o.team_id = $1::uuid
		  AND o.deleted_at IS NULL
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query work_items: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title, status, wiType string
		var summary *string
		var priority *string
		var updatedAt time.Time

		if err := rows.Scan(&id, &tid, &title, &summary, &status, &priority, &updatedAt, &wiType); err != nil {
			continue
		}

		summaryText := ""
		if summary != nil {
			summaryText = *summary
		}
		priorityVal := ""
		if priority != nil {
			priorityVal = *priority
		}

		docs = append(docs, SourceDocument{
			SourceType:      "work_item",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         summaryText,
			ContentText:     fmt.Sprintf("%s\n\n%s", title, summaryText),
			Metadata: map[string]any{
				"work_item_type": wiType,
				"status":         status,
				"priority":       priorityVal,
			},
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractIncidents indexes incidents.
func ExtractIncidents(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT o.id::text, o.team_id::text, o.title, o.summary, o.status,
		       o.updated_at, i.severity, i.impact
		FROM objects o
		JOIN incidents i ON i.object_id = o.id
		WHERE o.team_id = $1::uuid
		  AND o.deleted_at IS NULL
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title, status, severity string
		var summary, impact *string
		var updatedAt time.Time

		if err := rows.Scan(&id, &tid, &title, &summary, &status, &updatedAt, &severity, &impact); err != nil {
			continue
		}

		summaryText := ""
		if summary != nil {
			summaryText = *summary
		}
		impactText := ""
		if impact != nil {
			impactText = *impact
		}

		docs = append(docs, SourceDocument{
			SourceType:      "incident",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         summaryText,
			ContentText:     fmt.Sprintf("Incident: %s\nSeverity: %s\nImpact: %s\n%s", title, severity, impactText, summaryText),
			Metadata: map[string]any{
				"severity": severity,
				"status":   status,
			},
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractAssets indexes assets.
func ExtractAssets(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT o.id::text, o.team_id::text, o.title, o.summary, o.status,
		       o.updated_at, a.asset_type, a.provider, a.hostname
		FROM objects o
		JOIN assets a ON a.object_id = o.id
		WHERE o.team_id = $1::uuid
		  AND o.deleted_at IS NULL
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query assets: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title, status, assetType string
		var summary, provider, hostname *string
		var updatedAt time.Time

		if err := rows.Scan(&id, &tid, &title, &summary, &status, &updatedAt, &assetType, &provider, &hostname); err != nil {
			continue
		}

		summaryText := ""
		if summary != nil {
			summaryText = *summary
		}
		contentParts := []string{title}
		if hostname != nil && *hostname != "" {
			contentParts = append(contentParts, "Hostname: "+*hostname)
		}

		meta := map[string]any{
			"asset_type": assetType,
			"status":     status,
		}
		if provider != nil {
			meta["provider"] = *provider
		}

		docs = append(docs, SourceDocument{
			SourceType:      "asset",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         summaryText,
			ContentText:     strings.Join(contentParts, "\n"),
			Metadata:        meta,
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractRemediations indexes remediation proposals.
func ExtractRemediations(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, team_id::text, title, status, updated_at
		FROM remediation_proposals
		WHERE team_id = $1::uuid
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query remediations: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, title, status string
		var updatedAt time.Time

		if err := rows.Scan(&id, &tid, &title, &status, &updatedAt); err != nil {
			continue
		}

		docs = append(docs, SourceDocument{
			SourceType:      "remediation",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         fmt.Sprintf("Remediation proposal — %s", status),
			ContentText:     title,
			Metadata:        map[string]any{"status": status},
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ExtractApprovals indexes approval requests (safely — no action_target payload).
func ExtractApprovals(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, team_id::text, action_type, status, created_at
		FROM approval_requests
		WHERE team_id = $1::uuid
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query approvals: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, actionType, status string
		var createdAt time.Time

		if err := rows.Scan(&id, &tid, &actionType, &status, &createdAt); err != nil {
			continue
		}

		// Intentionally NOT indexing action_target — may contain operational payloads
		docs = append(docs, SourceDocument{
			SourceType:      "approval",
			SourceID:        id,
			TeamID:          tid,
			Title:           fmt.Sprintf("Approval: %s", actionType),
			Summary:         fmt.Sprintf("Approval request — %s", status),
			ContentText:     fmt.Sprintf("Approval request for action type: %s. Status: %s.", actionType, status),
			Metadata: map[string]any{
				"action_type": actionType,
				"status":      status,
				// action_target intentionally omitted
			},
			SourceUpdatedAt: createdAt,
		})
	}
	return docs, nil
}

// ExtractContextNodes indexes context graph nodes.
func ExtractContextNodes(ctx context.Context, pool *pgxpool.Pool, teamID string) ([]SourceDocument, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, team_id::text, entity_type, entity_id::text,
		       source, properties, updated_at
		FROM context_nodes
		WHERE team_id = $1::uuid
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query context_nodes: %w", err)
	}
	defer rows.Close()

	var docs []SourceDocument
	for rows.Next() {
		var id, tid, entityType, entityID, source string
		var properties []byte
		var updatedAt time.Time

		if err := rows.Scan(&id, &tid, &entityType, &entityID, &source, &properties, &updatedAt); err != nil {
			continue
		}

		// Extract useful text from properties JSON
		var props map[string]any
		json.Unmarshal(properties, &props)

		title := fmt.Sprintf("%s: %s", entityType, entityID)
		if t, ok := props["title"].(string); ok && t != "" {
			title = t
		}
		summary := ""
		if s, ok := props["summary"].(string); ok {
			summary = s
		}
		contentText := title
		if desc, ok := props["description"].(string); ok {
			contentText += "\n" + desc
		}

		// Sanitize properties before storing as metadata
		meta := SanitizeMetadata(props)
		meta["entity_type"] = entityType
		meta["source"] = source

		docs = append(docs, SourceDocument{
			SourceType:      "context_node",
			SourceID:        id,
			TeamID:          tid,
			Title:           title,
			Summary:         summary,
			ContentText:     contentText,
			Metadata:        meta,
			SourceUpdatedAt: updatedAt,
		})
	}
	return docs, nil
}

// ─── Helpers ───

// extractDocumentText extracts plain text from a DocumentBlock JSON structure.
func extractDocumentText(docJSON []byte) string {
	if len(docJSON) == 0 {
		return ""
	}

	var doc struct {
		Title  string `json:"title"`
		Blocks []struct {
			Type    string   `json:"type"`
			Text    *string  `json:"text"`
			Level   *int     `json:"level"`
			Items   []string `json:"items"`
			Headers []string `json:"headers"`
			Rows    [][]string `json:"rows"`
		} `json:"blocks"`
	}

	if err := json.Unmarshal(docJSON, &doc); err != nil {
		return ""
	}

	var sb strings.Builder
	if doc.Title != "" {
		sb.WriteString(doc.Title + "\n\n")
	}

	for _, b := range doc.Blocks {
		switch b.Type {
		case "heading":
			if b.Text != nil {
				sb.WriteString(*b.Text + "\n\n")
			}
		case "paragraph":
			if b.Text != nil {
				sb.WriteString(*b.Text + "\n\n")
			}
		case "bullets", "numbered_list":
			for _, item := range b.Items {
				sb.WriteString("• " + item + "\n")
			}
			sb.WriteString("\n")
		case "table":
			for _, h := range b.Headers {
				sb.WriteString(h + "\t")
			}
			sb.WriteString("\n")
			for _, row := range b.Rows {
				for _, cell := range row {
					sb.WriteString(cell + "\t")
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		case "quote", "callout":
			if b.Text != nil {
				sb.WriteString(*b.Text + "\n\n")
			}
		case "page_break":
			sb.WriteString("\n---\n\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

// appendJSONArray writes a JSON array as bullet points into a string.Builder.
func appendJSONArray(sb *strings.Builder, label string, data []byte) {
	var items []string
	if err := json.Unmarshal(data, &items); err != nil || len(items) == 0 {
		return
	}
	sb.WriteString(label + ":\n")
	for _, item := range items {
		sb.WriteString("• " + item + "\n")
	}
	sb.WriteString("\n")
}
