package presenton

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds Presenton runtime config.
type Config struct {
	Enabled      bool
	URL          string
	AdminUser    string
	AdminPass    string
	Timeout      time.Duration
	MaxFileBytes int64
}

// Handler for Presenton endpoints.
type Handler struct {
	pool   *pgxpool.Pool
	client Client
	s3     storage.S3Client
	bucket string
	cfg    Config
}

func NewHandler(pool *pgxpool.Pool, client Client, s3 storage.S3Client, bucket string, cfg Config) *Handler {
	return &Handler{pool: pool, client: client, s3: s3, bucket: bucket, cfg: cfg}
}

// SetClient allows injecting a client after construction (for testing).
func (h *Handler) SetClient(c Client) { h.client = c }

// ─── Status Endpoint ───

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled {
		writeJSON(w, 200, map[string]any{
			"enabled":   false,
			"reachable": false,
			"message":   "Presenton integration is disabled.",
		})
		return
	}

	// Check reachability with a short timeout GET
	reachable := false
	if h.client != nil {
		// Try a lightweight generate call with empty content — if Presenton is up,
		// it will respond (even with an error). We just need to know it's alive.
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_, err := h.client.Generate(ctx, GenerateRequest{
			Content:   "ping",
			NumSlides: 1,
			ExportAs:  "pptx",
		})
		// If it returned any response (even error), it's reachable
		if err == nil || !strings.Contains(err.Error(), "connection refused") {
			reachable = true
		}
	}

	msg := "Presenton is enabled and reachable."
	if !reachable {
		msg = "Presenton is enabled but not reachable."
	}

	writeJSON(w, 200, map[string]any{
		"enabled":   true,
		"reachable": reachable,
		"message":   msg,
	})
}

// ─── Generate Endpoint ───

type GeneratePresentationRequest struct {
	Title        string `json:"title"`
	Content      string `json:"content"`
	NumSlides    int    `json:"num_slides"`
	Template     string `json:"template"`
	Tone         string `json:"tone"`
	Language     string `json:"language"`
	ExportAs     string `json:"export_as"`
	Instructions string `json:"instructions"`
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamIDStr := chi.URLParam(r, "teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		writeErr(w, 400, "Invalid team ID")
		return
	}

	// Check enabled
	if !h.cfg.Enabled {
		writeErr(w, 503, "Presenton integration is disabled")
		return
	}

	// Parse + validate request
	var req GeneratePresentationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "Invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, 400, "title is required")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeErr(w, 400, "content is required")
		return
	}
	if req.NumSlides < 1 || req.NumSlides > 30 {
		writeErr(w, 400, "num_slides must be between 1 and 30")
		return
	}
	if req.ExportAs != "pptx" && req.ExportAs != "pdf" {
		writeErr(w, 400, "export_as must be 'pptx' or 'pdf'")
		return
	}
	if req.Template == "" {
		req.Template = "default"
	}
	if req.Tone == "" {
		req.Tone = "professional"
	}
	if req.Language == "" {
		req.Language = "en"
	}

	// Get actor
	cl, ok := iam.GetClaims(r)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}
	actorID, _ := uuid.Parse(cl.UserID)

	// Call Presenton Generate
	genReq := GenerateRequest{
		Content:      req.Content,
		NumSlides:    req.NumSlides,
		Template:     req.Template,
		Tone:         req.Tone,
		Language:     req.Language,
		ExportAs:     req.ExportAs,
		Instructions: req.Instructions,
	}

	genCtx, cancel := context.WithTimeout(ctx, h.cfg.Timeout)
	defer cancel()

	genResp, err := h.client.Generate(genCtx, genReq)
	if err != nil {
		writeErr(w, 503, "Presentation generation service unavailable")
		return
	}

	// Download the generated file
	dlCtx, dlCancel := context.WithTimeout(ctx, h.cfg.Timeout)
	defer dlCancel()

	fileBytes, contentType, err := h.client.DownloadFile(dlCtx, genResp.PresentationID, req.ExportAs)
	if err != nil {
		writeErr(w, 502, "Failed to download generated presentation")
		return
	}

	// Validate content type
	validContentTypes := map[string]bool{
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"application/pdf": true,
		"application/octet-stream": true, // Presenton may not set content-type
		"application/vnd.openxmlformats-officedocument.presentationml.template": true,
	}
	if !validContentTypes[contentType] {
		writeErr(w, 400, fmt.Sprintf("Unsupported content type: %s", contentType))
		return
	}

	// Enforce max file size
	if int64(len(fileBytes)) > h.cfg.MaxFileBytes {
		writeErr(w, 413, fmt.Sprintf("Generated file exceeds %d byte limit", h.cfg.MaxFileBytes))
		return
	}

	// Compute SHA256 of file bytes
	hash := sha256.Sum256(fileBytes)
	sha := hex.EncodeToString(hash[:])

	// Store file in MinIO — must succeed before creating artifact
	if h.s3 == nil {
		writeErr(w, 503, "Storage service unavailable")
		return
	}
	storageKey := fmt.Sprintf("teams/%s/artifacts/%s.%s", teamIDStr, uuid.New().String(), req.ExportAs)
	if err := h.s3.PutObject(ctx, h.bucket, storageKey, fileBytes, contentType); err != nil {
		writeErr(w, 500, "Failed to store presentation file")
		return
	}

	// Create storage_object + artifact in a transaction
	var artifactID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Create storage_object
		var storageObjID string
		s3Err := tx.QueryRow(ctx, `
			INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, retention_policy, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, 'none', 'default', $7)
			RETURNING id::text
		`, teamID, h.bucket, storageKey, contentType, len(fileBytes), sha, actorID).Scan(&storageObjID)
		if s3Err != nil {
			return fmt.Errorf("create storage_object: %w", s3Err)
		}

		// Create artifact record
		sourceData, _ := json.Marshal(map[string]any{
			"template":  req.Template,
			"tone":      req.Tone,
			"language":  req.Language,
			"num_slides": req.NumSlides,
		})

		err := tx.QueryRow(ctx, `
			INSERT INTO artifacts (team_id, artifact_type, title, description, content_markdown,
			                       status, source_type, source_data, storage_object_id, file_format,
			                       created_by, updated_by)
			VALUES ($1, 'presentation', $2, '', NULL, 'published', 'generated', $3, $4, $5, $6, $6)
			RETURNING id::text
		`, teamID, strings.TrimSpace(req.Title), sourceData, storageObjID, req.ExportAs, actorID).Scan(&artifactID)
		if err != nil {
			return fmt.Errorf("create artifact: %w", err)
		}

		// Audit — no content/prompts in audit
		artID, _ := uuid.Parse(artifactID)
		meta, _ := json.Marshal(map[string]any{
			"artifact_type": "presentation",
			"title":         req.Title,
			"file_format":   req.ExportAs,
			"source":        "generated",
		})
		_ = audit.Write(ctx, tx, audit.Event{
			TeamID: &teamID, ActorID: actorID, Action: "artifact.generated",
			EntityType: "artifact", EntityID: artID, NewValue: meta,
		})

		return nil
	})
	if err != nil {
		writeErr(w, 500, "Failed to create artifact record")
		return
	}

	writeJSON(w, 201, map[string]any{
		"artifact_id":         artifactID,
		"artifact_type":       "presentation",
		"title":               req.Title,
		"file_format":         req.ExportAs,
		"download_available":  true,
	})
}

func writeJSON(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, s int, m string) {
	writeJSON(w, s, map[string]string{"detail": m})
}
