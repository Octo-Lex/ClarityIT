package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ─── Track 7: Lightweight Sharing/Download Flow ───

const presignedExpirySeconds = 900 // 15 minutes

// Download returns a presigned MinIO URL for file-backed artifacts.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	if h.s3 == nil {
		writeErr(w, 503, "Storage not configured")
		return
	}

	// Look up artifact, verify team-scoped and has storage_object_id
	var storageObjID, fileFmt, bucket, objectKey *string
	var status string
	err = h.pool.QueryRow(ctx, `
		SELECT a.storage_object_id::text, a.file_format, a.status,
		       s.bucket, s.object_key
		FROM artifacts a
		LEFT JOIN storage_objects s ON a.storage_object_id = s.id
		WHERE a.id = $1 AND a.team_id = $2
	`, artifactID, teamID).Scan(&storageObjID, &fileFmt, &status, &bucket, &objectKey)
	if err != nil {
		writeErr(w, 404, "Artifact not found")
		return
	}

	if status == "archived" {
		writeErr(w, 403, "Archived artifacts cannot be downloaded")
		return
	}

	if storageObjID == nil || *storageObjID == "" {
		writeErr(w, 400, "This artifact has no file to download")
		return
	}

	if bucket == nil || objectKey == nil {
		writeErr(w, 500, "Storage object metadata missing")
		return
	}

	presignedURL, err := h.s3.GetPresignedURL(ctx, *bucket, *objectKey, time.Duration(presignedExpirySeconds)*time.Second)
	if err != nil {
		writeErr(w, 500, "Failed to generate download URL")
		return
	}

	// Audit
	cl, ok := iam.GetClaims(r)
	if ok {
		actorID, _ := uuid.Parse(cl.UserID)
		_ = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
			meta, _ := json.Marshal(map[string]any{
				"artifact_id": artifactID.String(),
				"action":      "download",
			})
			_ = audit.Write(ctx, tx, audit.Event{
				TeamID: &teamID, ActorID: actorID, Action: "artifact.download",
				EntityType: "artifact", EntityID: artifactID, NewValue: meta,
			})
			return nil
		})
	}

	format := ""
	if fileFmt != nil {
		format = *fileFmt
	}

	writeJSON(w, 200, map[string]any{
		"download_url":       presignedURL,
		"expires_in_seconds": presignedExpirySeconds,
		"file_format":        format,
	})
}

// ExportMarkdown returns content_markdown as a .md file download.
// For native documents (artifact_type=document), renders document_json to markdown.
// For other artifacts, returns content_markdown directly (v1.3 behavior).
func (h *Handler) ExportMarkdown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	var title, content, status, artifactType string
	var docJSON []byte
	err = h.pool.QueryRow(ctx, `
		SELECT a.title, COALESCE(a.content_markdown, ''), a.status, a.artifact_type,
		       d.document_json
		FROM artifacts a
		LEFT JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.id = $1 AND a.team_id = $2
	`, artifactID, teamID).Scan(&title, &content, &status, &artifactType, &docJSON)
	if err != nil {
		writeErr(w, 404, "Artifact not found")
		return
	}

	if status == "archived" {
		writeErr(w, 403, "Archived artifacts cannot be exported")
		return
	}

	// v1.4 Track 6: Native document export from document_json
	if artifactType == "document" && len(docJSON) > 0 {
		var doc DocumentJSON
		if err := json.Unmarshal(docJSON, &doc); err != nil {
			writeErr(w, 400, "Invalid document_json")
			return
		}
		if doc.SchemaVersion != 1 {
			writeErr(w, 400, "Unsupported schema_version")
			return
		}
		mdContent := renderBlocksToMarkdown(doc.Blocks)
		if strings.TrimSpace(mdContent) == "" {
			writeErr(w, 400, "Document has no content to export")
			return
		}
		filename := sanitizeFilename(title) + ".md"
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.WriteHeader(200)
		w.Write([]byte(mdContent))
		return
	}

	// v1.3 fallback: markdown-based artifacts
	if strings.TrimSpace(content) == "" {
		writeErr(w, 400, "This artifact has no markdown content to export")
		return
	}

	filename := sanitizeFilename(title) + ".md"
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(200)
	w.Write([]byte(content))
}

// ExportPDF returns a simple server-side PDF generated from markdown content.
// For native documents (artifact_type=document), renders document_json to PDF.
// Uses only Go standard library — no external rendering, no JS execution.
func (h *Handler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	var title, content, status, artifactType string
	var docJSON []byte
	err = h.pool.QueryRow(ctx, `
		SELECT a.title, COALESCE(a.content_markdown, ''), a.status, a.artifact_type,
		       d.document_json
		FROM artifacts a
		LEFT JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.id = $1 AND a.team_id = $2
	`, artifactID, teamID).Scan(&title, &content, &status, &artifactType, &docJSON)
	if err != nil {
		writeErr(w, 404, "Artifact not found")
		return
	}

	if status == "archived" {
		writeErr(w, 403, "Archived artifacts cannot be exported")
		return
	}

	// v1.4 Track 6: Native document export from document_json
	if artifactType == "document" && len(docJSON) > 0 {
		var doc DocumentJSON
		if err := json.Unmarshal(docJSON, &doc); err != nil {
			writeErr(w, 400, "Invalid document_json")
			return
		}
		if doc.SchemaVersion != 1 {
			writeErr(w, 400, "Unsupported schema_version")
			return
		}
		mdContent := renderBlocksToMarkdown(doc.Blocks)
		if strings.TrimSpace(mdContent) == "" {
			writeErr(w, 400, "Document has no content to export")
			return
		}
		pdfBytes, err := buildMinimalPDF(title, mdContent)
		if err != nil {
			writeErr(w, 500, "Failed to generate PDF")
			return
		}
		filename := sanitizeFilename(title) + ".pdf"
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.WriteHeader(200)
		w.Write(pdfBytes)
		return
	}

	// v1.3 fallback: markdown-based artifacts
	if strings.TrimSpace(content) == "" {
		writeErr(w, 400, "This artifact has no markdown content to export")
		return
	}

	pdfBytes, err := buildMinimalPDF(title, content)
	if err != nil {
		writeErr(w, 500, "Failed to generate PDF")
		return
	}

	filename := sanitizeFilename(title) + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(200)
	w.Write(pdfBytes)
}

// ExportDOCX exports a native ClarityDocs document to DOCX (OOXML) format.
// Only available for native documents (artifact_type=document).
// v1.4 Track 6.
func (h *Handler) ExportDOCX(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		writeErr(w, 400, "Invalid artifact ID")
		return
	}

	var title, status, artifactType string
	var docJSON []byte
	err = h.pool.QueryRow(ctx, `
		SELECT a.title, a.status, a.artifact_type, d.document_json
		FROM artifacts a
		JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.id = $1 AND a.team_id = $2
	`, artifactID, teamID).Scan(&title, &status, &artifactType, &docJSON)
	if err != nil {
		writeErr(w, 404, "Document not found")
		return
	}

	if artifactType != "document" {
		writeErr(w, 400, "DOCX export is only available for native documents")
		return
	}

	if status == "archived" {
		writeErr(w, 403, "Archived artifacts cannot be exported")
		return
	}

	var doc DocumentJSON
	if err := json.Unmarshal(docJSON, &doc); err != nil {
		writeErr(w, 400, "Invalid document_json")
		return
	}
	if doc.SchemaVersion != 1 {
		writeErr(w, 400, "Unsupported schema_version")
		return
	}

	docxBytes, err := buildDOCX(title, doc.Blocks)
	if err != nil {
		writeErr(w, 500, "Failed to generate DOCX")
		return
	}

	filename := sanitizeFilename(title) + ".docx"
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(200)
	w.Write(docxBytes)
}

// ─── Helpers ───

var unsafeFilenameChars = regexp.MustCompile(`[^\w\-. ]`)

func sanitizeFilename(name string) string {
	safe := unsafeFilenameChars.ReplaceAllString(name, "")
	safe = strings.TrimSpace(safe)
	if safe == "" {
		safe = "artifact"
	}
	if len(safe) > 100 {
		safe = safe[:100]
	}
	return safe
}

// buildMinimalPDF constructs a raw PDF 1.4 document from markdown text.
// No external libraries, no JS execution.
// Recognizes markdown headings (#, ##, ###) for font sizing.
func buildMinimalPDF(title, markdown string) ([]byte, error) {
	lines := strings.Split(markdown, "\n")
	var objects []string

	objects = append(objects, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	objects = append(objects, "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Build content stream
	var content strings.Builder
	content.WriteString("BT\n")

	marginLeft := 50.0
	yPos := 750.0
	lineHeight := 14.0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			yPos -= lineHeight * 0.5
			continue
		}

		fontSize := 12.0
		fontName := "F1"

		if strings.HasPrefix(trimmed, "### ") {
			fontSize = 13.0
			trimmed = strings.TrimPrefix(trimmed, "### ")
		} else if strings.HasPrefix(trimmed, "## ") {
			fontSize = 15.0
			trimmed = strings.TrimPrefix(trimmed, "## ")
		} else if strings.HasPrefix(trimmed, "# ") {
			fontSize = 18.0
			trimmed = strings.TrimPrefix(trimmed, "# ")
		}

		pdfText := stripMarkdownForPDF(trimmed)
		if pdfText == "" {
			continue
		}

		// Simple word-wrap at 90 chars
		if len(pdfText) > 90 {
			words := strings.Fields(pdfText)
			var current string
			for _, w := range words {
				if len(current)+len(w)+1 > 90 {
					if current != "" {
						escaped := escapePDFString(current)
						content.WriteString(fmt.Sprintf("/%s %.0f Tf\n", fontName, fontSize))
						content.WriteString(fmt.Sprintf("1 0 0 1 %.0f %.0f Tm\n", marginLeft, yPos))
						content.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
						yPos -= lineHeight
						current = w
					}
				} else {
					if current == "" {
						current = w
					} else {
						current += " " + w
					}
				}
			}
			if current != "" {
				escaped := escapePDFString(current)
				content.WriteString(fmt.Sprintf("/%s %.0f Tf\n", fontName, fontSize))
				content.WriteString(fmt.Sprintf("1 0 0 1 %.0f %.0f Tm\n", marginLeft, yPos))
				content.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
			}
		} else {
			escaped := escapePDFString(pdfText)
			content.WriteString(fmt.Sprintf("/%s %.0f Tf\n", fontName, fontSize))
			content.WriteString(fmt.Sprintf("1 0 0 1 %.0f %.0f Tm\n", marginLeft, yPos))
			content.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
		}
		yPos -= lineHeight
	}
	content.WriteString("ET\n")

	contentStr := content.String()

	objects = append(objects, fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n"))
	objects = append(objects, fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", len(contentStr), contentStr))
	objects = append(objects, "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	var pdf strings.Builder
	pdf.WriteString("%PDF-1.4\n")
	offset := len("%PDF-1.4\n")
	var offsets []int

	for _, obj := range objects {
		offsets = append(offsets, offset)
		pdf.WriteString(obj)
		offset += len(obj)
	}

	xrefOffset := offset
	pdf.WriteString("xref\n")
	pdf.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	pdf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}

	pdf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	pdf.WriteString("startxref\n")
	pdf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	pdf.WriteString("%%EOF\n")

	return []byte(pdf.String()), nil
}

func escapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func stripMarkdownForPDF(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "~~", "")

	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimPrefix(s, "* ")
	s = strings.TrimPrefix(s, "+ ")

	// Remove [text](url) → keep text
	linkRe := regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	s = linkRe.ReplaceAllString(s, "$1")

	s = strings.TrimPrefix(s, "> ")
	return strings.TrimSpace(s)
}

var _ = time.Now
