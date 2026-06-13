package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clarityit/api/internal/audit"
	"github.com/clarityit/api/internal/database"
	"github.com/clarityit/api/internal/iam"
	"github.com/clarityit/api/internal/outbox"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// S3Client is a minimal interface for S3-compatible storage.
type S3Client interface {
	PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error
	GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
}

// Handler for object attachments
type Handler struct {
	pool   *pgxpool.Pool
	s3     S3Client
	bucket string
}

func NewHandler(pool *pgxpool.Pool, s3 S3Client, bucket string) *Handler {
	return &Handler{pool: pool, s3: s3, bucket: bucket}
}

func claims(r *http.Request) (*iam.TokenClaims, bool) { return iam.GetClaims(r) }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Upload)
	r.Get("/", h.List)
	r.Get("/{attachmentId}/download-url", h.DownloadURL)
	return r
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	objectID := chi.URLParam(r, "objectId")
	cl, ok := claims(r)
	if !ok { writeErr(w, 401, "unauthorized"); return }
	actorID, _ := uuid.Parse(cl.UserID)
	tid, _ := uuid.Parse(teamID)
	oid, _ := uuid.Parse(objectID)

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024) // 50MB
	file, header, err := r.FormFile("file")
	if err != nil { writeErr(w, 400, "File required"); return }
	defer file.Close()

	// Read file bytes
	data := make([]byte, header.Size)
	n, _ := file.Read(data)
	data = data[:n]

	// Compute SHA256
	hash := sha256.Sum256(data)
	sha := hex.EncodeToString(hash[:])
	contentType := header.Header.Get("Content-Type")
	if contentType == "" { contentType = "application/octet-stream" }

	// Generate storage key
	storageKey := fmt.Sprintf("teams/%s/objects/%s/%s", teamID, objectID, uuid.New().String())

	// Upload to S3/MinIO
	if h.s3 != nil {
		if err := h.s3.PutObject(ctx, h.bucket, storageKey, data, contentType); err != nil {
			writeErr(w, 500, "Storage upload failed"); return
		}
	}

	var storageObjID, refID string
	err = database.WithTx(ctx, h.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Create storage_object
		if err := tx.QueryRow(ctx, `INSERT INTO storage_objects (team_id, bucket, object_key, content_type, size_bytes, sha256, encryption_status, retention_policy, created_by) VALUES ($1,$2,$3,$4,$5,$6,'none','default',$7) RETURNING id::text`, tid, h.bucket, storageKey, contentType, len(data), sha, actorID).Scan(&storageObjID); err != nil { return err }

		// Create object_storage_ref
		soid, _ := uuid.Parse(storageObjID)
		if err := tx.QueryRow(ctx, `INSERT INTO object_storage_refs (team_id, object_id, storage_object_id, ref_type, created_by) VALUES ($1,$2,$3,'attachment',$4) RETURNING id::text`, tid, oid, soid, actorID).Scan(&refID); err != nil { return err }

		// Audit — metadata only, no bytes
		meta, _ := json.Marshal(map[string]any{"object_id": objectID, "storage_object_id": storageObjID, "sha256": sha, "size": len(data), "content_type": contentType})
		_ = audit.Write(ctx, tx, audit.Event{TeamID: &tid, ActorID: actorID, Action: "object.attachment.created", EntityType: "object_storage_ref", EntityID: soid, NewValue: meta})
		_ = outbox.Write(ctx, tx, &teamID, outbox.Event{EventType: "clarity.v1.object.attachment.created", AggregateType: "object_storage_ref", AggregateID: refID, Payload: meta})
		return nil
	})
	if err != nil { writeErr(w, 500, "Failed to store attachment"); return }
	writeJSON(w, 201, map[string]any{"id": refID, "storage_object_id": storageObjID, "sha256": sha, "size": len(data)})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	objectID := chi.URLParam(r, "objectId")

	rows, _ := h.pool.Query(ctx, `SELECT r.id::text, s.bucket, s.object_key, s.content_type, s.size_bytes, s.sha256, s.created_at FROM object_storage_refs r JOIN storage_objects s ON r.storage_object_id=s.id WHERE r.object_id=$1 AND r.team_id=$2 ORDER BY r.created_at DESC`, objectID, teamID)
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var id, bucket, key, ct, sha string; var size int; var c time.Time
		rows.Scan(&id, &bucket, &key, &ct, &size, &sha, &c)
		out = append(out, map[string]any{"id": id, "content_type": ct, "size": size, "sha256": sha, "created_at": c})
	}
	if out == nil { out = []map[string]any{} }
	writeJSON(w, 200, out)
}

func (h *Handler) DownloadURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := chi.URLParam(r, "teamId")
	refID := chi.URLParam(r, "attachmentId")

	var bucket, key string
	err := h.pool.QueryRow(ctx, `SELECT s.bucket, s.object_key FROM object_storage_refs r JOIN storage_objects s ON r.storage_object_id=s.id WHERE r.id=$1 AND r.team_id=$2`, refID, teamID).Scan(&bucket, &key)
	if err != nil { writeErr(w, 404, "Attachment not found"); return }

	if h.s3 == nil {
		writeErr(w, 501, "Storage not configured"); return
	}

	url, err := h.s3.GetPresignedURL(ctx, bucket, key, 15*time.Minute)
	if err != nil { writeErr(w, 500, "Failed to generate URL"); return }
	writeJSON(w, 200, map[string]any{"url": url, "expires_in": 900})
}

func writeJSON(w http.ResponseWriter, s int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(s); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, s int, m string) { writeJSON(w, s, map[string]string{"detail": m}) }
