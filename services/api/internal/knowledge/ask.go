package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// ─── Ask Clarity Types ───

type AskRequest struct {
	Question    string   `json:"question"`
	SourceTypes []string `json:"source_types"`
	MaxSources  int      `json:"max_sources"`
}

type AskSourceCard struct {
	SourceType      string `json:"source_type"`
	SourceID        string `json:"source_id"`
	KnowledgeItemID string `json:"knowledge_item_id"`
	ChunkID         string `json:"chunk_id"`
	Title           string `json:"title"`
	Snippet         string `json:"snippet"`
}

type AskResponse struct {
	Answer     string          `json:"answer"`
	Sources    []AskSourceCard `json:"sources"`
	Confidence string          `json:"confidence"`
	MissingInfo []string       `json:"missing_info"`
}

// Worker ask types (internal)
type workerAskSource struct {
	SourceKey  string `json:"source_key"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Title      string `json:"title"`
	Snippet    string `json:"snippet"`
}

type workerAskRequest struct {
	Question string              `json:"question"`
	Sources  []workerAskSource   `json:"sources"`
}

type workerAskResponse struct {
	Answer      string   `json:"answer"`
	Citations   []string `json:"citations"`
	Confidence  string   `json:"confidence"`
	MissingInfo []string `json:"missing_info"`
}

// WorkerConfig configures the Python worker knowledge-ask endpoint.
type WorkerConfig struct {
	URL   string
	Token string
}

// ─── Constants ───

const (
	askMinQuestion    = 5
	askMaxQuestion    = 1000
	askDefaultSources = 8
	askMinSources     = 1
	askMaxSources     = 12
	askMaxSnippetLen  = 4000
	askMaxWorkerBytes = 100_000
	askTimeoutSeconds = 60
	askMaxSourceTypes = 10
)

var validConfidence = map[string]bool{
	"low": true, "medium": true, "high": true,
}

var forbiddenWorkerFields = []string{
	"chain_of_thought", "thinking", "internal_reasoning",
	"tool_calls", "action", "mutation", "execute",
}

// ─── Ask Handler ───

func (h *Handler) AskHTTP(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Invalid JSON body"})
		return
	}

	// Validate question
	req.Question = strings.TrimSpace(req.Question)
	if len(req.Question) < askMinQuestion {
		writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("Question must be at least %d characters", askMinQuestion)})
		return
	}
	if len(req.Question) > askMaxQuestion {
		writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("Question must be at most %d characters", askMaxQuestion)})
		return
	}

	// Validate max_sources
	if req.MaxSources <= 0 {
		req.MaxSources = askDefaultSources
	}
	if req.MaxSources < askMinSources {
		req.MaxSources = askMinSources
	}
	if req.MaxSources > askMaxSources {
		req.MaxSources = askMaxSources
	}

	// Validate source_types
	if len(req.SourceTypes) > askMaxSourceTypes {
		writeJSON(w, 400, map[string]string{"error": "Too many source_types"})
		return
	}

	// Retrieve matching knowledge_chunks via FTS
	sources, err := h.retrieveSources(r.Context(), teamID, req.Question, req.SourceTypes, req.MaxSources)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "Failed to retrieve knowledge"})
		return
	}

	// No sources found — safe response without calling Python
	if len(sources) == 0 {
		writeJSON(w, 200, AskResponse{
			Answer:      "There was not enough indexed knowledge to answer this question.",
			Sources:     []AskSourceCard{},
			Confidence:  "low",
			MissingInfo: []string{"No matching indexed knowledge was found."},
		})
		return
	}

	// Build worker request
	workerSources := make([]workerAskSource, len(sources))
	sourceKeyMap := make(map[string]int) // source_key -> index in sources
	for i, s := range sources {
		key := fmt.Sprintf("src-%d", i)
		workerSources[i] = workerAskSource{
			SourceKey:  key,
			SourceType: s.SourceType,
			SourceID:   s.SourceID,
			Title:      s.Title,
			Snippet:    truncate(s.Snippet, askMaxSnippetLen),
		}
		sourceKeyMap[key] = i
	}

	// Call Python worker
	workerReq := workerAskRequest{
		Question: req.Question,
		Sources:  workerSources,
	}

	workerResp, rawBytes, err := h.callWorker(r.Context(), workerReq)
	if err != nil {
		// Safe degradation on worker failure
		writeJSON(w, 200, AskResponse{
			Answer:      "The knowledge assistant is temporarily unavailable. Please try again later.",
			Sources:     sources,
			Confidence:  "low",
			MissingInfo: []string{"The AI assistant could not generate an answer at this time."},
		})
		return
	}

	// Validate worker response
	if err := validateWorkerResponse(workerResp, rawBytes, sourceKeyMap); err != nil {
		writeJSON(w, 200, AskResponse{
			Answer:      "The knowledge assistant could not produce a reliable answer. Please try rephrasing your question.",
			Sources:     sources,
			Confidence:  "low",
			MissingInfo: []string{"The AI assistant response did not pass validation."},
		})
		return
	}

	// Build citation-filtered source cards
	citedIndices := make(map[int]bool)
	for _, cite := range workerResp.Citations {
		if idx, ok := sourceKeyMap[cite]; ok {
			citedIndices[idx] = true
		}
	}

	citedSources := make([]AskSourceCard, 0, len(sources))
	for i, s := range sources {
		if citedIndices[i] || len(workerResp.Citations) == 0 {
			// Include source if cited, or if no citations given (include all retrieved)
			citedSources = append(citedSources, s)
		}
	}
	if len(citedSources) == 0 {
		// Always include at least the retrieved sources
		citedSources = sources
	}

	missingInfo := workerResp.MissingInfo
	if missingInfo == nil {
		missingInfo = []string{}
	}

	writeJSON(w, 200, AskResponse{
		Answer:      truncate(workerResp.Answer, 50000),
		Sources:     citedSources,
		Confidence:  workerResp.Confidence,
		MissingInfo: missingInfo,
	})
}

// retrieveSources retrieves matching knowledge_chunks via PostgreSQL FTS
func (h *Handler) retrieveSources(ctx context.Context, teamID, query string, sourceTypes []string, maxSources int) ([]AskSourceCard, error) {
	tsquery := buildTSQuery(query)
	if tsquery == "" || tsquery == "()" {
		return []AskSourceCard{}, nil
	}

	// Build query with optional source_type filter
	args := []any{teamID, tsquery}
	argIdx := 3

	query_ := `
		SELECT kc.id::text, ki.source_type, ki.source_id::text, ki.id::text,
		       ki.title,
		       ts_headline('english', kc.content_text, to_tsquery($2), 'MaxWords=35, MinWords=10') AS snippet
		FROM knowledge_chunks kc
		JOIN knowledge_items ki ON ki.id = kc.knowledge_item_id
		WHERE ki.team_id = $1::uuid
		  AND kc.search_vector @@ to_tsquery($2)
	`

	if len(sourceTypes) > 0 {
		placeholders := make([]string, len(sourceTypes))
		for i, st := range sourceTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, st)
			argIdx++
		}
		query_ += fmt.Sprintf(" AND ki.source_type IN (%s)", strings.Join(placeholders, ", "))
	}

	args = append(args, maxSources)
	query_ += fmt.Sprintf(" ORDER BY ts_rank(kc.search_vector, to_tsquery($2)) DESC LIMIT $%d", argIdx)

	rows, err := h.pool.Query(ctx, query_, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := []AskSourceCard{}
	for rows.Next() {
		var s AskSourceCard
		if err := rows.Scan(&s.ChunkID, &s.SourceType, &s.SourceID, &s.KnowledgeItemID, &s.Title, &s.Snippet); err != nil {
			continue
		}
		sources = append(sources, s)
	}

	return sources, nil
}

// callWorker sends the ask request to the Python worker
func (h *Handler) callWorker(ctx context.Context, req workerAskRequest) (*workerAskResponse, []byte, error) {
	if h.worker == nil || h.worker.URL == "" {
		return nil, nil, fmt.Errorf("worker not configured")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeoutSeconds*time.Second)
	defer cancel()

	url := strings.TrimRight(h.worker.URL, "/") + "/knowledge-ask"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if h.worker.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.worker.Token)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("worker returned status %d", resp.StatusCode)
	}

	// Cap response size
	limitedReader := io.LimitReader(resp.Body, askMaxWorkerBytes)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, nil, err
	}

	var workerResp workerAskResponse
	if err := json.Unmarshal(respBody, &workerResp); err != nil {
		return nil, nil, err
	}

	return &workerResp, respBody, nil
}

// validateWorkerResponse validates the Python worker response using raw bytes
func validateWorkerResponse(resp *workerAskResponse, rawBytes []byte, sourceKeyMap map[string]int) error {
	// Check forbidden fields in raw JSON response bytes
	rawJSON := string(rawBytes)
	for _, field := range forbiddenWorkerFields {
		if strings.Contains(rawJSON, "\""+field+"\"") {
			return fmt.Errorf("forbidden field in response: %s", field)
		}
	}

	// Validate confidence
	if !validConfidence[resp.Confidence] {
		return fmt.Errorf("invalid confidence: %s", resp.Confidence)
	}

	// Validate citations are in retrieved set
	for _, cite := range resp.Citations {
		if _, ok := sourceKeyMap[cite]; !ok {
			return fmt.Errorf("citation not in retrieved set: %s", cite)
		}
	}

	// Validate answer length
	if len(resp.Answer) > 50000 {
		return fmt.Errorf("answer exceeds 50000 chars")
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// SetWorker configures the Python worker for knowledge-ask calls.
func (h *Handler) SetWorker(cfg WorkerConfig) {
	h.worker = &cfg
}
