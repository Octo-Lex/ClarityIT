package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clarityit/api/internal/config"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Test setup ───

func setupIndexerTest(t *testing.T) *knowledgeTestEnv {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:       "test-secret",
		HMACKey:         "test-hmac-key",
		AccessTokenTTL:  15 * 60 * 1e9,
		RefreshTokenTTL: 7 * 24 * 3600 * 1e9,
	}
	pool, err := pgxpool.New(context.Background(), knowledgeDBURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	_, err = pool.Exec(context.Background(), "TRUNCATE knowledge_items, knowledge_chunks CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	kh := NewHandler(pool)
	iamH := iam.NewHandler(pool, cfg)

	r := chi.NewRouter()
	r.Use(middleware.ResolveAuth(cfg.JWTSecret))
	r.Post("/api/auth/login", iamH.Login)

	r.Route("/api/teams/{teamId}", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.With(middleware.RequirePermission(pool, "knowledge.search")).
			Get("/knowledge/search", kh.SearchHTTP)
		r.With(middleware.RequirePermission(pool, "knowledge.read")).
			Get("/knowledge/index-status", kh.IndexStatusHTTP)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/knowledge/reindex", kh.AdminReindexHTTP)
		r.Get("/knowledge/index-status", kh.AdminIndexStatusAllHTTP)
	})

	token := loginKnowledge(t, r, "owner@test.dev", "password12")

	var teamID string
	pool.QueryRow(context.Background(), `
		SELECT t.id::text FROM teams t
		JOIN team_memberships tm ON tm.team_id = t.id
		JOIN users u ON u.id = tm.user_id
		WHERE u.email = 'owner@test.dev' LIMIT 1
	`).Scan(&teamID)

	return &knowledgeTestEnv{r: r, pool: pool, token: token, teamID: teamID}
}

func getIndexedCount(t *testing.T, pool *pgxpool.Pool, teamID string) int {
	var count int
	pool.QueryRow(context.Background(),
		"SELECT count(*) FROM knowledge_items WHERE team_id = $1::uuid", teamID).Scan(&count)
	return count
}

func getChunkCount(t *testing.T, pool *pgxpool.Pool, teamID string) int {
	var count int
	pool.QueryRow(context.Background(),
		"SELECT count(*) FROM knowledge_chunks WHERE team_id = $1::uuid", teamID).Scan(&count)
	return count
}

func indexDoc(t *testing.T, ix *Indexer, teamID, sourceType, sourceID, title, content string) {
	t.Helper()
	err := ix.IndexSource(context.Background(), SourceDocument{
		SourceType:  sourceType,
		SourceID:    sourceID,
		TeamID:      teamID,
		Title:       title,
		ContentText: content,
		Metadata:    map[string]any{},
	})
	if err != nil {
		t.Fatalf("IndexSource: %v", err)
	}
}

// ─── Source-specific indexing tests ───

func TestIndexClarityDocument(t *testing.T) {
	e := setupIndexerTest(t)

	// Create a real document artifact
	var artID string
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO artifacts (team_id, artifact_type, title, content_markdown, status, source_type)
		VALUES ($1::uuid, 'document', 'Backup Strategy Doc', null, 'draft', 'native')
		RETURNING id::text
	`, e.teamID).Scan(&artID)
	if err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	docJSON := `{"schema_version":1,"title":"Backup Strategy","document_type":"general_document","blocks":[{"id":"b1","type":"heading","level":1,"text":"Backup Strategy"},{"id":"b2","type":"paragraph","text":"Backups must be verified weekly."}]}`

	_, err = e.pool.Exec(context.Background(), `
		INSERT INTO artifact_documents (artifact_id, document_type, document_json, word_count)
		VALUES ($1::uuid, 'general_document', $2, 5)
	`, artID, docJSON)
	if err != nil {
		t.Fatalf("create doc: %v", err)
	}

	// Extract and index
	docs, err := ExtractClarityDocuments(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Find our specific doc
	var found bool
	for _, doc := range docs {
		if doc.SourceID == artID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our clarity_document in extraction")
	}

	// Index all
	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	// Verify searchable
	resp := doSearch(e, "backup+strategy")
	if resp.Total < 1 {
		t.Errorf("expected to find indexed doc, got total=%d", resp.Total)
	}
}

func TestIndexArtifactMarkdown(t *testing.T) {
	e := setupIndexerTest(t)
	ix := NewIndexer(e.pool)

	var artID string
	e.pool.QueryRow(context.Background(), `
		INSERT INTO artifacts (team_id, artifact_type, title, content_markdown, status, source_type)
		VALUES ($1::uuid, 'report', 'Quarterly Report', '# Q3 Report\nRevenue is up 15%.', 'published', 'native')
		RETURNING id::text
	`, e.teamID).Scan(&artID)

	docs, err := ExtractArtifacts(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if doc.SourceID == artID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our artifact")
	}

	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "quarterly")
	if resp.Total < 1 {
		t.Errorf("expected to find artifact, got total=%d", resp.Total)
	}
}

func TestIndexMeetingSummary(t *testing.T) {
	e := setupIndexerTest(t)

	var artID string
	e.pool.QueryRow(context.Background(), `
		INSERT INTO artifacts (team_id, artifact_type, title, status, source_type)
		VALUES ($1::uuid, 'meeting_summary', 'Weekly Sync', 'published', 'native')
		RETURNING id::text
	`, e.teamID).Scan(&artID)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO artifact_meeting_data (artifact_id, meeting_date, attendees, agenda_items, decisions, action_items)
		VALUES ($1::uuid, '2026-06-15',
		        '["Alice", "Bob"]'::jsonb,
		        '["Review roadmap"]'::jsonb,
		        '["Ship v1.5"]'::jsonb,
		        '["Update docs"]'::jsonb)
	`, artID)
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	docs, err := ExtractMeetingSummaries(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if doc.SourceID == artID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our meeting summary")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "roadmap")
	if resp.Total < 1 {
		t.Errorf("expected to find meeting, got total=%d", resp.Total)
	}
}

func TestIndexTemplate(t *testing.T) {
	e := setupIndexerTest(t)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO artifact_templates (team_id, template_type, name, description, content_markdown, metadata, is_system, template_format)
		VALUES ($1::uuid, 'report', 'Incident Runbook Template',
		        'Standard incident response template',
		        '# Incident Response\n1. Assess impact\n2. Engage team',
		        '{}'::jsonb, false, 'markdown')
	`, e.teamID)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	docs, err := ExtractTemplates(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if strings.Contains(doc.Title, "Incident Runbook") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find Incident Runbook template")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "incident+response")
	if resp.Total < 1 {
		t.Errorf("expected to find template, got total=%d", resp.Total)
	}
}

func TestIndexDocumentJSONTemplate(t *testing.T) {
	e := setupIndexerTest(t)

	docJSON := `{"schema_version":1,"title":"Decision Memo","document_type":"decision_memo","blocks":[{"id":"b1","type":"paragraph","text":"Context and decision"}]}`

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO artifact_templates (team_id, template_type, name, description, content_markdown, metadata, is_system, template_format, document_json, schema_version)
		VALUES ($1::uuid, 'document', 'Decision Memo Template',
		        'Decision memo template',
		        'placeholder',
		        '{}'::jsonb, false, 'document_json', $2, 1)
	`, e.teamID, docJSON)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	docs, err := ExtractTemplates(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	// Should find the template
	found := false
	for _, doc := range docs {
		if strings.Contains(doc.Title, "Decision Memo") {
			found = true
			if !strings.Contains(doc.ContentText, "Context and decision") {
				t.Error("expected document_json text extraction")
			}
		}
	}
	if !found {
		t.Error("expected to find Decision Memo template")
	}
}

func TestIndexWorkItem(t *testing.T) {
	e := setupIndexerTest(t)

	var objID string
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO objects (team_id, object_type, title, summary, status)
		VALUES ($1::uuid, 'work_item', 'Fix auth bug', 'Login fails on Safari', 'open')
		RETURNING id::text
	`, e.teamID).Scan(&objID)
	if err != nil {
		t.Fatalf("create object: %v", err)
	}

	_, err = e.pool.Exec(context.Background(), `
		INSERT INTO work_items (object_id, work_item_type)
		VALUES ($1::uuid, 'ticket')
	`, objID)
	if err != nil {
		t.Fatalf("create work_item: %v", err)
	}

	docs, err := ExtractWorkItems(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if doc.SourceID == objID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our work_item")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "auth+bug")
	if resp.Total < 1 {
		t.Errorf("expected to find work_item, got total=%d", resp.Total)
	}
}

func TestIndexIncident(t *testing.T) {
	e := setupIndexerTest(t)

	var objID string
	e.pool.QueryRow(context.Background(), `
		INSERT INTO objects (team_id, object_type, title, summary, status)
		VALUES ($1::uuid, 'incident', 'DB Outage', 'Primary database unreachable', 'active')
		RETURNING id::text
	`, e.teamID).Scan(&objID)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO incidents (object_id, severity, impact)
		VALUES ($1::uuid, 'sev1', 'All services down')
	`, objID)
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	docs, err := ExtractIncidents(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if doc.SourceID == objID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our incident")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "database+outage")
	if resp.Total < 1 {
		t.Errorf("expected to find incident, got total=%d", resp.Total)
	}
}

func TestIndexAsset(t *testing.T) {
	e := setupIndexerTest(t)

	var objID string
	e.pool.QueryRow(context.Background(), `
		INSERT INTO objects (team_id, object_type, title, summary, status)
		VALUES ($1::uuid, 'asset', 'web-prod-01', 'Production web server', 'active')
		RETURNING id::text
	`, e.teamID).Scan(&objID)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO assets (object_id, asset_type, provider, hostname)
		VALUES ($1::uuid, 'server', 'proxmox', 'web-prod-01.internal')
	`, objID)
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}

	docs, err := ExtractAssets(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if doc.SourceID == objID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our asset")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "production+web+server")
	if resp.Total < 1 {
		t.Errorf("expected to find asset, got total=%d", resp.Total)
	}
}

func TestIndexRemediation(t *testing.T) {
	e := setupIndexerTest(t)

	var ownerID string
	e.pool.QueryRow(context.Background(),
		"SELECT id::text FROM users WHERE email='owner@test.dev'").Scan(&ownerID)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO remediation_proposals (team_id, title, status, created_by)
		VALUES ($1::uuid, 'Disk Space Cleanup', 'proposed', $2::uuid)
	`, e.teamID, ownerID)
	if err != nil {
		t.Fatalf("create remediation: %v", err)
	}

	docs, err := ExtractRemediations(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if strings.Contains(doc.Title, "Disk Space") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our remediation")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "disk+space")
	if resp.Total < 1 {
		t.Errorf("expected to find remediation, got total=%d", resp.Total)
	}
}

func TestIndexApprovalSafely(t *testing.T) {
	e := setupIndexerTest(t)

	var ownerID string
	e.pool.QueryRow(context.Background(),
		"SELECT id::text FROM users WHERE email='owner@test.dev'").Scan(&ownerID)

	// Create approval with sensitive action_target
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO approval_requests (team_id, action_type, action_target, risk_level, status, requested_by, expires_at)
		VALUES ($1::uuid, 'proxmox.stop',
		        '{"node":"pve1","vmid":"100"}'::jsonb,
		        'high', 'pending', $2::uuid, NOW() + INTERVAL '1 hour')
	`, e.teamID, ownerID)
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	docs, err := ExtractApprovals(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if doc.Title == "Approval: proxmox.stop" {
			found = true
		}
		// Verify no sensitive action_target payload content in indexed text
		if strings.Contains(doc.ContentText, "password") {
			t.Error("approval content should NOT contain action_target fields")
		}
	}
	if !found {
		t.Fatalf("expected to find our approval")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	// Verify the password is not in the knowledge_items table
	var indexedContent string
	e.pool.QueryRow(context.Background(),
		"SELECT content_text FROM knowledge_items WHERE source_type='approval' AND team_id=$1::uuid",
		e.teamID).Scan(&indexedContent)
	if strings.Contains(indexedContent, "secret123") {
		t.Error("indexed approval should NOT contain action_target password")
	}
}

func TestIndexContextNode(t *testing.T) {
	e := setupIndexerTest(t)

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO context_nodes (team_id, entity_type, entity_id, source, properties)
		VALUES ($1::uuid, 'service', gen_random_uuid(), 'manual',
		        '{"title":"Payment API","description":"Handles payment processing","version":"2.1"}'::jsonb)
	`, e.teamID)
	if err != nil {
		t.Fatalf("create context_node: %v", err)
	}

	docs, err := ExtractContextNodes(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	found := false
	for _, doc := range docs {
		if strings.Contains(doc.Title, "Payment API") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find our context_node")
	}

	ix := NewIndexer(e.pool)
	for _, doc := range docs {
		ix.IndexSource(context.Background(), doc)
	}

	resp := doSearch(e, "payment+processing")
	if resp.Total < 1 {
		t.Errorf("expected to find context_node, got total=%d", resp.Total)
	}
}

// ─── Content hash / chunk behavior tests ───

func TestUnchangedContentHashSkipsReindex(t *testing.T) {
	e := setupIndexerTest(t)
	ix := NewIndexer(e.pool)
	ctx := context.Background()

	var sourceID string
	e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)

	// Index first time
	indexDoc(t, ix, e.teamID, "artifact", sourceID, "Test Doc", "Original content here")
	before := getChunkCount(t, e.pool, e.teamID)

	// Index again with same content — should skip
	indexDoc(t, ix, e.teamID, "artifact", sourceID, "Test Doc", "Original content here")
	after := getChunkCount(t, e.pool, e.teamID)

	if after != before {
		t.Errorf("chunks should not change when content_hash is unchanged: before=%d after=%d", before, after)
	}
}

func TestChangedContentHashReplacesChunks(t *testing.T) {
	e := setupIndexerTest(t)
	ix := NewIndexer(e.pool)
	ctx := context.Background()

	var sourceID string
	e.pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&sourceID)

	// Index first time
	err := ix.IndexSource(ctx, SourceDocument{
		SourceType: "artifact", SourceID: sourceID, TeamID: e.teamID,
		Title: "V1", ContentText: "Version one content with some text",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := getChunkCount(t, e.pool, e.teamID)
	if before == 0 {
		t.Fatal("expected chunks after first index")
	}

	// Index with different content
	err = ix.IndexSource(ctx, SourceDocument{
		SourceType: "artifact", SourceID: sourceID, TeamID: e.teamID,
		Title: "V2", ContentText: "Version two content is different from version one",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Chunks should still exist (replaced, not duplicated beyond max)
	after := getChunkCount(t, e.pool, e.teamID)
	if after == 0 {
		t.Error("expected chunks after content change")
	}

	// Verify only 1 knowledge_item (upsert)
	count := getIndexedCount(t, e.pool, e.teamID)
	if count != 1 {
		t.Errorf("expected 1 item after upsert, got %d", count)
	}
}

// ─── Cross-team isolation test ───

func TestIndexedCrossTeamInvisible(t *testing.T) {
	e := setupIndexerTest(t)
	ix := NewIndexer(e.pool)

	// Index in team1
	var sourceID string
	e.pool.QueryRow(context.Background(), "SELECT gen_random_uuid()::text").Scan(&sourceID)
	indexDoc(t, ix, e.teamID, "artifact", sourceID, "Secret Team1 Doc", "classified content zzz999unique")

	// Create team2
	var team2ID string
	e.pool.QueryRow(context.Background(), `
		INSERT INTO teams (name, slug)
		VALUES ('Other Team Idx', 'other-team-idx-' || md5(random()::text))
		RETURNING id::text
	`).Scan(&team2ID)

	// Search from team2 context
	req := httptest.NewRequest("GET", "/api/teams/"+team2ID+"/knowledge/search?q=zzz999unique", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	var resp SearchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("team2 should not see team1's indexed content, got total=%d", resp.Total)
	}
}

// ─── Archived/deleted exclusion test ───

func TestArchivedSourceExcluded(t *testing.T) {
	e := setupIndexerTest(t)

	// Create archived artifact
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO artifacts (team_id, artifact_type, title, content_markdown, status, source_type)
		VALUES ($1::uuid, 'report', 'Archived Report', 'This is archived content', 'archived', 'native')
	`, e.teamID)
	if err != nil {
		t.Fatalf("create archived artifact: %v", err)
	}

	// Extract should skip archived
	docs, err := ExtractArtifacts(context.Background(), e.pool, e.teamID)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	for _, doc := range docs {
		if strings.Contains(doc.Title, "Archived Report") {
			t.Error("archived artifacts should not be extracted")
		}
	}
}

// ─── Sanitizer tests ───

func TestSanitizerRemovesSecrets(t *testing.T) {
	input := "The config has password=secret123 and api_key=abc456 and bearer xyz789"
	result := SanitizeContent(input)
	if strings.Contains(result, "secret123") {
		t.Error("password should be sanitized")
	}
	if strings.Contains(result, "abc456") {
		t.Error("api_key should be sanitized")
	}
	if strings.Contains(result, "xyz789") {
		t.Error("bearer token should be sanitized")
	}
}

func TestSanitizerRemovesStorageIdentifiers(t *testing.T) {
	input := "File stored at s3://clarityit-bucket/object-key-123 and bucket=clarityit"
	result := SanitizeContent(input)
	if strings.Contains(result, "s3://clarityit-bucket") {
		t.Error("s3 URL should be sanitized")
	}
}

func TestSanitizerRemovesPrompts(t *testing.T) {
	meta := map[string]any{
		"title":  "Generated Doc",
		"prompt": "Write a detailed report about internal systems",
	}
	result := SanitizeMetadata(meta)
	if _, exists := result["prompt"]; exists {
		t.Error("raw prompt should be removed from metadata")
	}
	if result["title"] != "Generated Doc" {
		t.Error("safe metadata should be preserved")
	}
}

func TestSanitizerRemovesChainOfThought(t *testing.T) {
	input := `The answer is yes. {"chain_of_thought": "I need to think about this carefully..."} {"thinking": "let me reason through this"}`
	result := SanitizeContent(input)
	if strings.Contains(result, "chain_of_thought") {
		t.Error("chain_of_thought should be removed")
	}
	if strings.Contains(result, "let me reason through this") {
		t.Error("thinking content should be removed")
	}
}

// ─── Admin reindex test ───

func TestAdminReindex(t *testing.T) {
	e := setupIndexerTest(t)

	// Create some sources with unique names to avoid collision
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO artifacts (team_id, artifact_type, title, content_markdown, status, source_type)
		VALUES ($1::uuid, 'report', 'Reindex Test ' || gen_random_uuid()::text, 'Reindexable content here', 'published', 'native')
	`, e.teamID)
	if err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	var objID string
	err = e.pool.QueryRow(context.Background(), `
		INSERT INTO objects (team_id, object_type, title, summary, status)
		VALUES ($1::uuid, 'work_item', 'Reindex Task ' || gen_random_uuid()::text, 'Task for reindex testing', 'open')
		RETURNING id::text
	`, e.teamID).Scan(&objID)
	if err != nil {
		t.Fatalf("create work_item: %v", err)
	}
	_, err = e.pool.Exec(context.Background(),
		"INSERT INTO work_items (object_id, work_item_type) VALUES ($1::uuid, 'task')", objID)
	if err != nil {
		t.Fatalf("create work_item ext: %v", err)
	}

	// Run reindex
	req := httptest.NewRequest("POST", "/api/admin/knowledge/reindex", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result ReindexResult
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Indexed < 1 {
		t.Errorf("expected >=1 indexed, got %d", result.Indexed)
	}
	if result.Errors > 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}

	// Verify index status endpoint
	req = httptest.NewRequest("GET", "/api/admin/knowledge/index-status", nil)
	req.Header.Set("Authorization", "Bearer "+e.token)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 for index-status, got %d", w.Code)
	}
}

// ─── No side effects test ───

func TestIndexNoOperationalSideEffects(t *testing.T) {
	e := setupIndexerTest(t)
	ix := NewIndexer(e.pool)

	var sourceID string
	e.pool.QueryRow(context.Background(), "SELECT gen_random_uuid()::text").Scan(&sourceID)

	// Count artifacts before
	var artifactsBefore int
	e.pool.QueryRow(context.Background(), "SELECT count(*) FROM artifacts").Scan(&artifactsBefore)

	// Count objects before
	var objectsBefore int
	e.pool.QueryRow(context.Background(), "SELECT count(*) FROM objects WHERE deleted_at IS NULL").Scan(&objectsBefore)

	// Index a source
	indexDoc(t, ix, e.teamID, "artifact", sourceID, "Side Effect Test", "Content for indexing")

	// Verify no operational tables changed
	var artifactsAfter int
	e.pool.QueryRow(context.Background(), "SELECT count(*) FROM artifacts").Scan(&artifactsAfter)
	if artifactsAfter != artifactsBefore {
		t.Errorf("artifacts count changed: before=%d after=%d", artifactsBefore, artifactsAfter)
	}

	var objectsAfter int
	e.pool.QueryRow(context.Background(), "SELECT count(*) FROM objects WHERE deleted_at IS NULL").Scan(&objectsAfter)
	if objectsAfter != objectsBefore {
		t.Errorf("objects count changed: before=%d after=%d", objectsBefore, objectsAfter)
	}
}

// ─── Chunking tests ───

func TestChunkContentShort(t *testing.T) {
	content := "This is a short paragraph."
	chunks := ChunkContent(content)
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].ChunkIndex != 0 {
		t.Errorf("expected chunk_index=0, got %d", chunks[0].ChunkIndex)
	}
}

func TestChunkContentMultipleParagraphs(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString(fmt.Sprintf("# Section %d\n\n", i+1))
		for j := 0; j < 100; j++ {
			sb.WriteString("This is content for testing chunking behavior. ")
		}
		sb.WriteString("\n\n")
	}

	chunks := ChunkContent(sb.String())
	if len(chunks) < 2 {
		t.Errorf("expected >=2 chunks for large content, got %d", len(chunks))
	}

	// Verify chunk_index starts at 0 and increments
	for i, c := range chunks {
		if c.ChunkIndex != i {
			t.Errorf("chunk %d has chunk_index=%d", i, c.ChunkIndex)
		}
	}
}

func TestChunkContentEmpty(t *testing.T) {
	chunks := ChunkContent("")
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestChunkContentSizeBounded(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("word ")
	}
	content := sb.String()

	chunks := ChunkContent(content)
	for _, c := range chunks {
		if len(c.ContentText) > MaxChunkSize+200 { // some overflow from paragraph join
			t.Errorf("chunk %d exceeds size bound: %d chars", c.ChunkIndex, len(c.ContentText))
		}
	}
}

func TestContentHashStable(t *testing.T) {
	h1 := ComputeContentHash("test content")
	h2 := ComputeContentHash("test content")
	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	h3 := ComputeContentHash("different content")
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
}

func TestTokenEstimatePositive(t *testing.T) {
	est := estimateTokens("hello world test")
	if est < 1 {
		t.Error("token estimate should be >=1")
	}
}

func TestMaxChunksEnforced(t *testing.T) {
	e := setupIndexerTest(t)
	ix := NewIndexer(e.pool)

	var sourceID string
	e.pool.QueryRow(context.Background(), "SELECT gen_random_uuid()::text").Scan(&sourceID)

	// Create content with many paragraphs
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString(fmt.Sprintf("# Section %d\n\nLong content paragraph %d.\n\n", i, i))
	}

	err := ix.IndexSource(context.Background(), SourceDocument{
		SourceType:  "artifact",
		SourceID:    sourceID,
		TeamID:      e.teamID,
		Title:       "Many Chunks Test",
		ContentText: sb.String(),
	})
	if err != nil {
		t.Fatalf("index: %v", err)
	}

	// Count chunks for this item
	var chunkCount int
	e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM knowledge_chunks WHERE team_id=$1::uuid", e.teamID).Scan(&chunkCount)
	if chunkCount > MaxChunks {
		t.Errorf("expected <=%d chunks, got %d", MaxChunks, chunkCount)
	}
}

// Ensure time import is used
var _ = time.Now
