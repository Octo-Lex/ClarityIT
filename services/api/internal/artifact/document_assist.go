package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/iam"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─── v1.4 Track 3: Agent Assist for Documents ───

var validAssistModes = map[string]bool{
	"rewrite": true, "summarize": true, "expand": true,
	"make_concise": true, "make_executive": true, "make_technical": true,
	"draft_section": true, "create_outline": true,
	"extract_action_items": true, "improve_headings": true,
}

const (
	maxAssistTextLen   = 20000
	maxInstructionLen  = 2000
	assistTimeout      = 60 * time.Second
	maxAssistBodyBytes = 100000
)

// AssistRequest is the POST body for document-assist.
type AssistRequest struct {
	Mode         string `json:"mode"`
	BlockID      string `json:"block_id"`
	SelectedText string `json:"selected_text"`
	Instruction  string `json:"instruction"`
	DocumentType string `json:"document_type"`
	MaxWords     int    `json:"max_words"`
}

// AssistResponse is returned to the frontend.
type AssistResponse struct {
	SuggestedBlocks []map[string]any `json:"suggested_blocks"`
	Summary         string           `json:"summary"`
}

// workerAssistPayload is sent to the Python worker.
type workerAssistPayload struct {
	Mode         string `json:"mode"`
	SelectedText string `json:"selected_text"`
	Instruction  string `json:"instruction"`
	DocumentType string `json:"document_type"`
	MaxWords     int    `json:"max_words"`
}

// WorkerAssistConfig configures the Python worker document-assist endpoint.
type WorkerAssistConfig struct {
	URL   string
	Token string
}

// SetWorkerAssist configures the handler for document-assist calls.
func (h *Handler) SetWorkerAssist(cfg WorkerAssistConfig) {
	h.workerAssist = &cfg
}

// DocumentAssist handles agent-assisted document editing requests.
func (h *Handler) DocumentAssist(w http.ResponseWriter, r *http.Request) {
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

	_, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "Unauthorized")
		return
	}

	var req AssistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	// 1. Validate mode
	if !validAssistModes[req.Mode] {
		writeErr(w, 400, fmt.Sprintf("invalid mode: %q", req.Mode))
		return
	}

	// 2. Validate bounds
	if len(req.SelectedText) > maxAssistTextLen {
		writeErr(w, 400, fmt.Sprintf("selected_text exceeds %d chars", maxAssistTextLen))
		return
	}
	if len(req.Instruction) > maxInstructionLen {
		writeErr(w, 400, fmt.Sprintf("instruction exceeds %d chars", maxInstructionLen))
		return
	}
	if req.MaxWords == 0 {
		req.MaxWords = 300
	}
	if req.MaxWords < 20 || req.MaxWords > 2000 {
		writeErr(w, 400, "max_words must be between 20 and 2000")
		return
	}

	// 3. Verify artifact: team-scoped, document type, not archived
	var artifactType, status string
	var docJSON []byte
	err = h.pool.QueryRow(ctx, `
		SELECT a.artifact_type, a.status, d.document_json
		FROM artifacts a
		JOIN artifact_documents d ON d.artifact_id = a.id
		WHERE a.id = $1 AND a.team_id = $2
	`, artifactID, teamID).Scan(&artifactType, &status, &docJSON)
	if err != nil {
		writeErr(w, 404, "Document not found")
		return
	}

	if artifactType != "document" {
		writeErr(w, 400, "Artifact is not a document")
		return
	}
	if status == "archived" {
		writeErr(w, 403, "Archived documents cannot be edited")
		return
	}

	// 4. Validate block_id exists for block-based modes
	if req.BlockID != "" {
		var docData map[string]any
		if err := json.Unmarshal(docJSON, &docData); err == nil {
			blocks, ok := docData["blocks"].([]any)
			if ok {
				found := false
				for _, b := range blocks {
					if blk, ok := b.(map[string]any); ok {
						if id, _ := blk["id"].(string); id == req.BlockID {
							found = true
							break
						}
					}
				}
				if !found {
					writeErr(w, 400, fmt.Sprintf("block_id %q not found in document", req.BlockID))
					return
				}
			}
		}
	}

	// 5. Check worker assist is configured
	if h.workerAssist == nil || h.workerAssist.URL == "" {
		// Fallback: return a safe stub response if worker not configured
		writeErr(w, 503, "Document assist is not configured")
		return
	}

	// 6. Build payload for Python worker
	payload := workerAssistPayload{
		Mode:         req.Mode,
		SelectedText: req.SelectedText,
		Instruction:  req.Instruction,
		DocumentType: req.DocumentType,
		MaxWords:     req.MaxWords,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		writeErr(w, 500, "Failed to marshal assist payload")
		return
	}

	// 7. Call Python worker with timeout
	workerCtx, cancel := context.WithTimeout(ctx, assistTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(workerCtx, "POST",
		strings.TrimRight(h.workerAssist.URL, "/")+"/document-assist",
		bytes.NewReader(payloadBytes))
	if err != nil {
		writeErr(w, 500, "Failed to create worker request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.workerAssist.Token)

	client := &http.Client{Timeout: assistTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeErr(w, 504, "Document assist worker timed out or unreachable")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAssistBodyBytes))
	if err != nil {
		writeErr(w, 502, "Failed to read worker response")
		return
	}

	if resp.StatusCode != 200 {
		var errBody map[string]any
		if json.Unmarshal(body, &errBody) == nil {
			if msg, ok := errBody["error"].(string); ok {
				writeErr(w, 502, fmt.Sprintf("Worker error: %s", msg))
				return
			}
		}
		writeErr(w, 502, "Worker returned an error")
		return
	}

	// 8. Validate response structure
	var workerResp AssistResponse
	if err := json.Unmarshal(body, &workerResp); err != nil {
		writeErr(w, 502, "Malformed worker response")
		return
	}

	if len(workerResp.SuggestedBlocks) == 0 {
		writeErr(w, 502, "Worker returned no suggested blocks")
		return
	}

	// 9. Validate each suggested block using Track 1 validation
	for i, blk := range workerResp.SuggestedBlocks {
		validated, err := validateSuggestedBlock(blk)
		if err != nil {
			writeErr(w, 502, fmt.Sprintf("Worker returned invalid block %d: %s", i, err.Error()))
			return
		}
		// Assign ID if missing
		if _, hasID := validated["id"]; !hasID {
			validated["id"] = fmt.Sprintf("suggested_%d_%s", i, uuid.New().String()[:8])
		}
		workerResp.SuggestedBlocks[i] = validated
	}

	// 10. Return — NO persistence, NO side effects
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workerResp)
}

// validateSuggestedBlock validates a suggested block from the worker.
func validateSuggestedBlock(blk map[string]any) (map[string]any, error) {
	blkType, ok := blk["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing block type")
	}
	if !validBlockTypes[blkType] {
		return nil, fmt.Errorf("unknown block type: %s", blkType)
	}

	switch blkType {
	case "heading":
		var level float64
		if l, ok := blk["level"].(float64); ok {
			level = l
		} else if l, ok := blk["level"].(int); ok {
			level = float64(l)
		}
		if level < 1 || level > 6 {
			return nil, fmt.Errorf("heading level must be 1-6")
		}
		text, _ := blk["text"].(string)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("heading requires text")
		}
	case "paragraph":
		text, _ := blk["text"].(string)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("paragraph requires text")
		}
	case "bullets", "numbered_list":
		items, ok := blk["items"].([]any)
		if !ok || len(items) == 0 {
			return nil, fmt.Errorf("%s requires non-empty items", blkType)
		}
	case "quote":
		text, _ := blk["text"].(string)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("quote requires text")
		}
	case "callout":
		variant, _ := blk["variant"].(string)
		if !validCalloutVariants[variant] {
			return nil, fmt.Errorf("callout requires valid variant")
		}
		text, _ := blk["text"].(string)
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("callout requires text")
		}
	case "page_break":
		// ok
	}

	return blk, nil
}
