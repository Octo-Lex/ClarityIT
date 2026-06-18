package artifact

import (
	"fmt"
	"strings"
)

// ─── v1.4 Track 6: Document Rendering Pipeline ───
//
// Renders native document_json blocks to Markdown and PDF text.
// DOCX rendering is in docx_export.go.

// renderBlocksToMarkdown converts document blocks to a markdown string.
func renderBlocksToMarkdown(blocks []DocumentBlock) string {
	var sb strings.Builder
	for _, blk := range blocks {
		switch blk.Type {
		case "heading":
			if blk.Level != nil {
				sb.WriteString(strings.Repeat("#", *blk.Level))
				sb.WriteString(" ")
			}
			if blk.Text != nil {
				sb.WriteString(*blk.Text)
			}
			sb.WriteString("\n\n")
		case "paragraph":
			if blk.Text != nil {
				sb.WriteString(*blk.Text)
			}
			sb.WriteString("\n\n")
		case "bullets":
			for _, item := range blk.Items {
				sb.WriteString("- ")
				sb.WriteString(item)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		case "numbered_list":
			for i, item := range blk.Items {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
			}
			sb.WriteString("\n")
		case "table":
			if len(blk.Headers) > 0 {
				sb.WriteString("| ")
				sb.WriteString(strings.Join(blk.Headers, " | "))
				sb.WriteString(" |\n")
				sb.WriteString("|")
				for range blk.Headers {
					sb.WriteString(" --- |")
				}
				sb.WriteString("\n")
				for _, row := range blk.Rows {
					sb.WriteString("| ")
					sb.WriteString(strings.Join(row, " | "))
					sb.WriteString(" |\n")
				}
				sb.WriteString("\n")
			}
		case "quote":
			if blk.Text != nil {
				sb.WriteString("> ")
				sb.WriteString(*blk.Text)
				sb.WriteString("\n\n")
			}
		case "callout":
			variant := ""
			if blk.Variant != nil {
				variant = *blk.Variant
			}
			if blk.Text != nil {
				sb.WriteString(fmt.Sprintf("> **[%s]** %s\n\n", strings.ToUpper(variant), *blk.Text))
			}
		case "page_break":
			sb.WriteString("\n<!-- PAGE BREAK -->\n\n")
		}
	}
	return sb.String()
}

// renderBlocksToPDFLines converts document blocks to lines suitable for PDF rendering.
// Returns title and body markdown.
func renderBlocksToPDFText(title string, blocks []DocumentBlock) string {
	return renderBlocksToMarkdown(blocks)
}
