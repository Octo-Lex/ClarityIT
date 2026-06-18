package artifact

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ─── v1.4 Track 6: DOCX (OOXML) Export ───
//
// Generates a valid DOCX file directly in Go using archive/zip and encoding/xml.
// No external DOCX library. No SuperDoc. No LibreOffice.
// All user content is XML-escaped.

// escapeXML escapes special XML characters in user content.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// buildDOCX generates a DOCX ZIP from document blocks.
func buildDOCX(title string, blocks []DocumentBlock) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Detect if lists are present (for numbering.xml)
	hasBullets := false
	hasNumbered := false
	for _, blk := range blocks {
		if blk.Type == "bullets" {
			hasBullets = true
		}
		if blk.Type == "numbered_list" {
			hasNumbered = true
		}
	}

	// [Content_Types].xml
	if err := writeZipFile(zw, "[Content_Types].xml", contentTypesXML()); err != nil {
		return nil, err
	}

	// _rels/.rels
	if err := writeZipFile(zw, "_rels/.rels", relsXML()); err != nil {
		return nil, err
	}

	// word/document.xml
	if err := writeZipFile(zw, "word/document.xml", documentXML(title, blocks)); err != nil {
		return nil, err
	}

	// word/styles.xml
	if err := writeZipFile(zw, "word/styles.xml", stylesXML()); err != nil {
		return nil, err
	}

	// word/numbering.xml (only when lists present)
	if hasBullets || hasNumbered {
		if err := writeZipFile(zw, "word/numbering.xml", numberingXML(hasBullets, hasNumbered)); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func writeZipFile(zw *zip.Writer, name string, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, content)
	return err
}

func contentTypesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
</Types>`
}

func relsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
}

func stylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults>
    <w:rPrDefault>
      <w:rPr>
        <w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:cs="Calibri"/>
        <w:sz w:val="22"/>
      </w:rPr>
    </w:rPrDefault>
    <w:pPrDefault>
      <w:pPr>
        <w:spacing w:after="160" w:line="259" w:lineRule="auto"/>
      </w:pPr>
    </w:pPrDefault>
  </w:docDefaults>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:pPr><w:keepNext/><w:spacing w:before="240" w:after="60"/><w:outlineLvl w:val="0"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="32"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:pPr><w:keepNext/><w:spacing w:before="200" w:after="40"/><w:outlineLvl w:val="1"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="26"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading3">
    <w:name w:val="heading 3"/>
    <w:pPr><w:keepNext/><w:spacing w:before="160" w:after="20"/><w:outlineLvl w:val="2"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="24"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Quote">
    <w:name w:val="Quote"/>
    <w:pPr><w:spacing w:before="80" w:after="80"/><w:ind w:left="720"/></w:pPr>
    <w:rPr><w:i/><w:color w:val="666666"/></w:rPr>
  </w:style>
</w:styles>`
}

func numberingXML(hasBullets, hasNumbered bool) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
`)
	// Bullet abstract numbering
	if hasBullets {
		sb.WriteString(`
  <w:abstractNum w:abstractNumId="0">
    <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="&#8226;"/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:lvl>
  </w:abstractNum>
`)
	}
	// Numbered abstract numbering
	if hasNumbered {
		id := 0
		if hasBullets {
			id = 1
		}
		sb.WriteString(fmt.Sprintf(`
  <w:abstractNum w:abstractNumId="%d">
    <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%%1."/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:lvl>
  </w:abstractNum>
`, id))
	}

	// Map abstract → concrete
	idx := 0
	if hasBullets {
		sb.WriteString(fmt.Sprintf(`  <w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
`))
		idx = 1
	}
	if hasNumbered {
		absId := 0
		if hasBullets {
			absId = 1
		}
		numId := 1
		if hasBullets {
			numId = 2
		}
		_ = absId
		sb.WriteString(fmt.Sprintf(`  <w:num w:numId="%d"><w:abstractNumId w:val="%d"/></w:num>
`, numId, absId))
	}
	_ = idx

	sb.WriteString(`</w:numbering>`)
	return sb.String()
}

func documentXML(title string, blocks []DocumentBlock) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
`)

	// Title as first heading
	if title != "" {
		sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, escapeXML(title)))
	}

	bulletNumId := 0
	numberedNumId := 0
	if hasBlockType(blocks, "bullets") {
		bulletNumId = 1
	}
	if hasBlockType(blocks, "numbered_list") {
		if bulletNumId > 0 {
			numberedNumId = 2
		} else {
			numberedNumId = 1
		}
	}

	for _, blk := range blocks {
		switch blk.Type {
		case "heading":
			level := 2
			if blk.Level != nil {
				level = *blk.Level
			}
			styleName := "Heading2"
			if level <= 1 {
				styleName = "Heading1"
			} else if level == 3 {
				styleName = "Heading3"
			} else if level >= 4 {
				styleName = "Heading3"
			}
			text := ""
			if blk.Text != nil {
				text = *blk.Text
			}
			sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:pStyle w:val="%s"/></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, styleName, escapeXML(text)))

		case "paragraph":
			text := ""
			if blk.Text != nil {
				text = *blk.Text
			}
			sb.WriteString(fmt.Sprintf(`    <w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, escapeXML(text)))

		case "bullets":
			for _, item := range blk.Items {
				sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="%d"/></w:numPr></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, bulletNumId, escapeXML(item)))
			}

		case "numbered_list":
			for _, item := range blk.Items {
				sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="%d"/></w:numPr></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, numberedNumId, escapeXML(item)))
			}

		case "table":
			writeDocxTable(&sb, blk)

		case "quote":
			text := ""
			if blk.Text != nil {
				text = *blk.Text
			}
			sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:pStyle w:val="Quote"/></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, escapeXML(text)))

		case "callout":
			variant := "info"
			if blk.Variant != nil {
				variant = *blk.Variant
			}
			text := ""
			if blk.Text != nil {
				text = *blk.Text
			}
			label := fmt.Sprintf("[%s] %s", strings.ToUpper(variant), text)
			sb.WriteString(fmt.Sprintf(`    <w:p><w:pPr><w:shd w:val="clear" w:color="auto" w:fill="F0F0F0"/><w:spacing w:before="80" w:after="80"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p>
`, escapeXML(label)))

		case "page_break":
			sb.WriteString(`    <w:p><w:r><w:br w:type="page"/></w:r></w:p>
`)
		}
	}

	sb.WriteString(`  </w:body>
</w:document>`)
	return sb.String()
}

func writeDocxTable(sb *strings.Builder, blk DocumentBlock) {
	if len(blk.Headers) == 0 {
		return
	}
	sb.WriteString(`    <w:tbl>
      <w:tblPr>
        <w:tblStyle w:val="TableGrid"/>
        <w:tblW w:w="5000" w:type="pct"/>
      </w:tblPr>
`)
	// Header row
	sb.WriteString("      <w:tr>\n")
	for _, h := range blk.Headers {
		sb.WriteString(fmt.Sprintf(`        <w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>
`, escapeXML(h)))
	}
	sb.WriteString("      </w:tr>\n")
	// Data rows
	for _, row := range blk.Rows {
		sb.WriteString("      <w:tr>\n")
		for i, cell := range row {
			if i >= len(blk.Headers) {
				break
			}
			sb.WriteString(fmt.Sprintf(`        <w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr><w:p><w:r><w:t xml:space="preserve">%s</w:t></w:r></w:p></w:tc>
`, escapeXML(cell)))
		}
		sb.WriteString("      </w:tr>\n")
	}
	sb.WriteString("    </w:tbl>\n")
}

func hasBlockType(blocks []DocumentBlock, btype string) bool {
	for _, b := range blocks {
		if b.Type == btype {
			return true
		}
	}
	return false
}
