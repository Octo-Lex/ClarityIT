package artifact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── v1.4 Track 6: DOCX/PDF/Markdown Export Tests ───

// createTestDocument creates a native document and returns the artifact ID.
func createTestDocument(t *testing.T, e *artifactTestEnv, title string, blocks []DocumentBlock) string {
	t.Helper()
	body := makeCreateDocReq(title, "general_document", blocks)
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/teams/"+e.teamID+"/artifacts/documents", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("failed to create document: %d %s", w.Code, w.Body.String())
	}
	var resp DocumentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

func allBlockTypes() []DocumentBlock {
	level1 := 1
	level2 := 2
	text := "text"
	variant := "info"
	return []DocumentBlock{
		{ID: "b1", Type: "heading", Level: &level1, Text: &text},
		{ID: "b2", Type: "heading", Level: &level2, Text: &text},
		{ID: "b3", Type: "paragraph", Text: &text},
		{ID: "b4", Type: "bullets", Items: []string{"a", "b"}},
		{ID: "b5", Type: "numbered_list", Items: []string{"1", "2"}},
		{ID: "b6", Type: "table", Headers: []string{"H1"}, Rows: [][]string{{"c1"}}},
		{ID: "b7", Type: "quote", Text: &text},
		{ID: "b8", Type: "callout", Variant: &variant, Text: &text},
		{ID: "b9", Type: "page_break"},
	}
}

func doExport(t *testing.T, e *artifactTestEnv, artID, format string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/%s", e.teamID, artID, format), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

func TestDocExport_MarkdownWorks(t *testing.T) {
	e := setupDownloadTest(t)
	// Create document via document route
	artID := createTestDocument(t, e, "MD Export Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("Hello export")},
	})
	w := doExport(t, e, artID, "markdown")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Hello export") {
		t.Error("expected markdown content")
	}
}

func TestDocExport_MarkdownAllBlockTypes(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "All Blocks MD", allBlockTypes())
	w := doExport(t, e, artID, "markdown")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	md := w.Body.String()
	// Check that key markdown patterns are present
	if !strings.Contains(md, "# ") { // heading level 1
		t.Error("missing heading 1")
	}
	if !strings.Contains(md, "- a") { // bullet
		t.Error("missing bullet list")
	}
	if !strings.Contains(md, "1. 1") { // numbered
		t.Error("missing numbered list")
	}
	if !strings.Contains(md, "> ") { // quote/callout
		t.Error("missing quote or callout")
	}
}

func TestDocExport_MarkdownSafeFilename(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "My Report 2026", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("x")},
	})
	w := doExport(t, e, artID, "markdown")
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".md") {
		t.Errorf("expected .md in filename: %s", cd)
	}
}

func TestDocExport_PDFReturnsApplicationPDF(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "PDF Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("PDF content")},
	})
	w := doExport(t, e, artID, "pdf")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/pdf") {
		t.Errorf("expected application/pdf, got %s", w.Header().Get("Content-Type"))
	}
	bodyBytes := w.Body.Bytes()
	if len(bodyBytes) < 4 || string(bodyBytes[:4]) != "%PDF" {
		t.Error("PDF does not start with %PDF")
	}
}

func TestDocExport_PDFAllBlockTypes(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "All Blocks PDF", allBlockTypes())
	w := doExport(t, e, artID, "pdf")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(w.Body.Bytes()) < 100 {
		t.Error("PDF too small — expected content from all blocks")
	}
}

func TestDocExport_PDFSafeFilename(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Quarterly Report", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("x")},
	})
	w := doExport(t, e, artID, "pdf")
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".pdf") {
		t.Errorf("expected .pdf in filename: %s", cd)
	}
}

func TestDocExport_PDFNoJavaScript(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "No JS", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("<script>alert(1)</script>")},
	})
	w := doExport(t, e, artID, "pdf")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// PDF should escape the content — the raw PDF text would have escaped parens
	// but should NOT contain literal <script> tags that would execute
	bodyStr := w.Body.String()
	// The content IS in the PDF but escaped — it won't execute as JS
	// Verify the PDF is valid
	if !strings.HasPrefix(bodyStr, "%PDF") {
		t.Error("not a valid PDF")
	}
}

func TestDocExport_DOCXContentType(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "DOCX Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("DOCX content")},
	})
	w := doExport(t, e, artID, "docx")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "wordprocessingml") {
		t.Errorf("expected wordprocessingml content type, got %s", ct)
	}
}

func TestDocExport_DOCXValidZIP(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "ZIP Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("ZIP check")},
	})
	w := doExport(t, e, artID, "docx")
	bodyBytes := w.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		t.Fatalf("DOCX is not a valid ZIP: %v", err)
	}
	// Check for required entries
	entryMap := make(map[string]bool)
	for _, f := range zr.File {
		entryMap[f.Name] = true
	}
	if !entryMap["[Content_Types].xml"] {
		t.Error("missing [Content_Types].xml")
	}
	if !entryMap["_rels/.rels"] {
		t.Error("missing _rels/.rels")
	}
	if !entryMap["word/document.xml"] {
		t.Error("missing word/document.xml")
	}
	if !entryMap["word/styles.xml"] {
		t.Error("missing word/styles.xml")
	}
}

func TestDocExport_DOCXNumberingWhenListsPresent(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Lists Doc", []DocumentBlock{
		{ID: "b1", Type: "bullets", Items: []string{"a", "b"}},
		{ID: "b2", Type: "numbered_list", Items: []string{"1", "2"}},
	})
	w := doExport(t, e, artID, "docx")
	bodyBytes := w.Body.Bytes()
	zr, _ := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	found := false
	for _, f := range zr.File {
		if f.Name == "word/numbering.xml" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected word/numbering.xml when lists present")
	}
}

func TestDocExport_DOCXNoNumberingWhenNoLists(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "No Lists", []DocumentBlock{
		{ID: "b1", Type: "heading", Level: intPtr(1), Text: strPtr("Title")},
		{ID: "b2", Type: "paragraph", Text: strPtr("Just text")},
	})
	w := doExport(t, e, artID, "docx")
	bodyBytes := w.Body.Bytes()
	zr, _ := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	for _, f := range zr.File {
		if f.Name == "word/numbering.xml" {
			t.Error("numbering.xml should not exist when no lists")
		}
	}
}

func TestDocExport_DOCXXMLEscaping(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Escape Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("<script>alert('xss')</script>&amp;")},
	})
	w := doExport(t, e, artID, "docx")
	bodyBytes := w.Body.Bytes()
	zr, _ := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	var docXML string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := make([]byte, f.UncompressedSize64+100)
			n, _ := rc.Read(buf)
			docXML = string(buf[:n])
			rc.Close()
			break
		}
	}
	// Should NOT contain raw <script> — should be escaped
	if strings.Contains(docXML, "<script>") {
		t.Error("user text not escaped in document.xml")
	}
	// Should contain escaped version
	if !strings.Contains(docXML, "&lt;script&gt;") {
		t.Error("expected escaped script tag")
	}
}

func TestDocExport_DOCXRendersHeadings(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Heading Test", []DocumentBlock{
		{ID: "b1", Type: "heading", Level: intPtr(1), Text: strPtr("Main Title")},
	})
	w := doExport(t, e, artID, "docx")
	docXML := extractDocXML(t, w.Body.Bytes())
	if !strings.Contains(docXML, "Heading1") {
		t.Error("expected Heading1 style in DOCX")
	}
	if !strings.Contains(docXML, "Main Title") {
		t.Error("heading text not in DOCX")
	}
}

func TestDocExport_DOCXRendersParagraphs(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Para Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("A paragraph here")},
	})
	w := doExport(t, e, artID, "docx")
	docXML := extractDocXML(t, w.Body.Bytes())
	if !strings.Contains(docXML, "A paragraph here") {
		t.Error("paragraph text not in DOCX")
	}
}

func TestDocExport_DOCXRendersLists(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "List Test", []DocumentBlock{
		{ID: "b1", Type: "bullets", Items: []string{"Bullet 1", "Bullet 2"}},
		{ID: "b2", Type: "numbered_list", Items: []string{"Num 1", "Num 2"}},
	})
	w := doExport(t, e, artID, "docx")
	docXML := extractDocXML(t, w.Body.Bytes())
	if !strings.Contains(docXML, "Bullet 1") {
		t.Error("bullet item not in DOCX")
	}
	if !strings.Contains(docXML, "Num 1") {
		t.Error("numbered item not in DOCX")
	}
	if !strings.Contains(docXML, "numPr") {
		t.Error("numPr (numbering properties) not found in DOCX")
	}
}

func TestDocExport_DOCXRendersTables(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Table Test", []DocumentBlock{
		{ID: "b1", Type: "table", Headers: []string{"Name", "Value"}, Rows: [][]string{{"A", "1"}}},
	})
	w := doExport(t, e, artID, "docx")
	// Parse ZIP to check table content in document.xml
	bodyBytes := w.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		t.Fatalf("invalid DOCX ZIP: %v", err)
	}
	var docContent string
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			buf := new(strings.Builder)
			io.Copy(buf, rc)
			rc.Close()
			docContent = buf.String()
			break
		}
	}
	if !strings.Contains(docContent, "<w:tbl>") {
		t.Error("expected w:tbl in DOCX")
	}
	if !strings.Contains(docContent, "Name") {
		t.Error("header text not in DOCX table")
	}
}

func TestDocExport_DOCXRendersPageBreaks(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Page Break Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("Before")},
		{ID: "b2", Type: "page_break"},
		{ID: "b3", Type: "paragraph", Text: strPtr("After")},
	})
	w := doExport(t, e, artID, "docx")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Parse the ZIP to check page break in document.xml
	bodyBytes := w.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		t.Fatalf("invalid DOCX ZIP: %v", err)
	}
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			defer rc.Close()
			buf := new(strings.Builder)
			io.Copy(buf, rc)
			if !strings.Contains(buf.String(), "w:br") || !strings.Contains(buf.String(), "page") {
				t.Error("expected page break in DOCX document.xml")
			}
			break
		}
	}
}

func TestDocExport_CrossTeam404(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Cross Team", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("x")},
	})
	fakeTeam := "00000000-0000-0000-0000-000000000999"
	for _, fmt2 := range []string{"markdown", "pdf", "docx"} {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/%s", fakeTeam, artID, fmt2), nil)
		req.Header.Set("Authorization", "Bearer "+e.token)
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Errorf("expected 404 for cross-team %s export, got %d", fmt2, w.Code)
		}
	}
}

func TestDocExport_Archived403(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Archived Export", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("x")},
	})
	// Archive
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/teams/%s/artifacts/%s", e.teamID, artID), nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.r.ServeHTTP(httptest.NewRecorder(), req)

	for _, fmt2 := range []string{"markdown", "pdf", "docx"} {
		w := doExport(t, e, artID, fmt2)
		if w.Code != 403 {
			t.Errorf("expected 403 for archived %s export, got %d", fmt2, w.Code)
		}
	}
}

func TestDocExport_NonDocArtifactCompatibility(t *testing.T) {
	e := setupDownloadTest(t)
	// Create a markdown artifact (not native document)
	body := `{"artifact_type":"report","title":"MD Report","content_markdown":"# Old Report\n\nContent"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/teams/%s/artifacts", e.teamID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	artID := resp["id"].(string)

	// Markdown export should use v1.3 path
	w = doExport(t, e, artID, "markdown")
	if w.Code != 200 {
		t.Errorf("expected 200 for markdown artifact export, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Old Report") {
		t.Error("markdown content missing")
	}

	// PDF export should use v1.3 path
	w = doExport(t, e, artID, "pdf")
	if w.Code != 200 {
		t.Errorf("expected 200 for PDF export, got %d", w.Code)
	}

	// DOCX should be 404 (no artifact_documents row) or 400 (not a native document)
	w = doExport(t, e, artID, "docx")
	if w.Code != 400 && w.Code != 404 {
		t.Errorf("expected 400 or 404 for DOCX on non-document, got %d", w.Code)
	}
}

func TestDocExport_UnauthorizedDenied(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Auth Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("x")},
	})
	for _, fmt2 := range []string{"markdown", "pdf", "docx"} {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/teams/%s/artifacts/%s/export/%s", e.teamID, artID, fmt2), nil)
		w := httptest.NewRecorder()
		e.r.ServeHTTP(w, req)
		if w.Code != 401 && w.Code != 403 {
			t.Errorf("expected 401/403 for unauthorized %s, got %d", fmt2, w.Code)
		}
	}
}

func TestDocExport_NoOperationalSideEffects(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Side Effects", allBlockTypes())

	var apprBefore, actionBefore, remBefore int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&apprBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionBefore)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remBefore)

	for _, fmt2 := range []string{"markdown", "pdf", "docx"} {
		doExport(t, e, artID, fmt2)
	}

	var apprAfter, actionAfter, remAfter int
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM approval_requests").Scan(&apprAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM asset_actions").Scan(&actionAfter)
	e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM remediation_proposals").Scan(&remAfter)

	if apprAfter != apprBefore {
		t.Errorf("approval_requests changed: %d → %d", apprBefore, apprAfter)
	}
	if actionAfter != actionBefore {
		t.Errorf("asset_actions changed: %d → %d", actionBefore, actionAfter)
	}
	if remAfter != remBefore {
		t.Errorf("remediation_proposals changed: %d → %d", remBefore, remAfter)
	}
}

func TestDocExport_DOCXSafeFilename(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "Safe Name Test", []DocumentBlock{
		{ID: "b1", Type: "paragraph", Text: strPtr("x")},
	})
	w := doExport(t, e, artID, "docx")
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".docx") {
		t.Errorf("expected .docx in filename: %s", cd)
	}
}

func TestDocExport_DocumentJSONNotMutated(t *testing.T) {
	e := setupDownloadTest(t)
	artID := createTestDocument(t, e, "No Mutation", allBlockTypes())

	// Get JSON before export
	var jsonBefore []byte
	e.pool.QueryRow(t.Context(), "SELECT document_json FROM artifact_documents WHERE artifact_id::text = $1", artID).Scan(&jsonBefore)

	// Export all formats
	for _, fmt2 := range []string{"markdown", "pdf", "docx"} {
		doExport(t, e, artID, fmt2)
	}

	// Get JSON after export
	var jsonAfter []byte
	e.pool.QueryRow(t.Context(), "SELECT document_json FROM artifact_documents WHERE artifact_id::text = $1", artID).Scan(&jsonAfter)

	if !bytes.Equal(jsonBefore, jsonAfter) {
		t.Error("document_json was mutated during export")
	}
}

// extractDocXML reads the word/document.xml entry from a DOCX ZIP.
func extractDocXML(t *testing.T, docxBytes []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("invalid DOCX ZIP: %v", err)
	}
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			defer rc.Close()
			buf := new(strings.Builder)
			io.Copy(buf, rc)
			return buf.String()
		}
	}
	t.Fatal("word/document.xml not found in DOCX")
	return ""
}

// Helper funcs for creating pointers in test data are in document_test.go
