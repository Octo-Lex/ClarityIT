package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Chunk data ───

type ChunkData struct {
	ChunkIndex    int
	Heading       string
	ContentText   string
	TokenEstimate int
}

// ─── Source document (unified representation from any source) ───

type SourceDocument struct {
	SourceType      string
	SourceID        string
	TeamID          string
	Title           string
	Summary         string
	ContentText     string
	Metadata        map[string]any
	SourceUpdatedAt time.Time
	// Chunks override — if nil, chunks are auto-generated from ContentText
	Chunks []ChunkData
}

// ─── Indexer ───

const (
	MaxItemContent = 100_000 // chars
	MaxChunkSize   = 4_000   // chars
	MaxChunks      = 100
	MaxMetadataKB  = 32 * 1024
)

type Indexer struct {
	pool *pgxpool.Pool
}

func NewIndexer(pool *pgxpool.Pool) *Indexer {
	return &Indexer{pool: pool}
}

// IndexSource indexes a single SourceDocument into the knowledge index.
// - Computes content_hash
// - Skips if content_hash is unchanged
// - Upserts knowledge_items
// - Replaces chunks transactionally when content changed
func (ix *Indexer) IndexSource(ctx context.Context, doc SourceDocument) error {
	// Sanitize before anything
	content := SanitizeContent(doc.ContentText)
	if len(content) > MaxItemContent {
		content = content[:MaxItemContent]
	}
	title := SanitizeContent(doc.Title)
	summary := SanitizeContent(doc.Summary)
	meta := SanitizeMetadata(doc.Metadata)
	if meta == nil {
		meta = map[string]any{}
	}

	contentHash := ComputeContentHash(title + "\n" + summary + "\n" + content)

	// Check if existing item has same content_hash
	var existingHash *string
	err := ix.pool.QueryRow(ctx, `
		SELECT content_hash FROM knowledge_items
		WHERE team_id = $1::uuid AND source_type = $2 AND source_id = $3::uuid
	`, doc.TeamID, doc.SourceType, doc.SourceID).Scan(&existingHash)

	if err == nil && existingHash != nil && *existingHash == contentHash {
		// Content unchanged — skip
		return nil
	}

	// Marshal metadata
	metaJSON, _ := json.Marshal(meta)
	if len(metaJSON) > MaxMetadataKB {
		metaJSON = []byte("{}")
	}

	// Generate chunks if not provided
	chunks := doc.Chunks
	if chunks == nil {
		chunks = ChunkContent(content)
	}

	// Begin transaction for atomic chunk replacement
	tx, err := ix.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert knowledge item
	var sourceUpdated *time.Time
	if !doc.SourceUpdatedAt.IsZero() {
		sourceUpdated = &doc.SourceUpdatedAt
	}
	staleAfter := time.Now().Add(90 * 24 * time.Hour)

	_, err = tx.Exec(ctx, `
		INSERT INTO knowledge_items
			(team_id, source_type, source_id, title, summary, content_text,
			 content_hash, metadata, source_updated_at, stale_after, indexed_at)
		VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6, $7, $8::jsonb, $9, $10, NOW())
		ON CONFLICT (team_id, source_type, source_id) DO UPDATE
		SET title = $4, summary = $5, content_text = $6, content_hash = $7,
		    metadata = $8::jsonb, source_updated_at = $9, stale_after = $10,
		    indexed_at = NOW()
	`, doc.TeamID, doc.SourceType, doc.SourceID, title, summary, content,
		contentHash, metaJSON, sourceUpdated, staleAfter)
	if err != nil {
		return fmt.Errorf("upsert knowledge_item: %w", err)
	}

	// Delete old chunks
	_, err = tx.Exec(ctx, `
		DELETE FROM knowledge_chunks
		WHERE knowledge_item_id = (
			SELECT id FROM knowledge_items
			WHERE team_id = $1::uuid AND source_type = $2 AND source_id = $3::uuid
		)
	`, doc.TeamID, doc.SourceType, doc.SourceID)
	if err != nil {
		return fmt.Errorf("delete old chunks: %w", err)
	}

	// Insert new chunks
	for _, c := range chunks {
		if c.ChunkIndex >= MaxChunks {
			break
		}
		chunkText := SanitizeContent(c.ContentText)
		if len(chunkText) > MaxChunkSize {
			chunkText = chunkText[:MaxChunkSize]
		}
		chunkHash := ComputeContentHash(chunkText)
		if strings.TrimSpace(chunkText) == "" {
			continue
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO knowledge_chunks
				(knowledge_item_id, team_id, chunk_index, heading, content_text,
				 content_hash, token_estimate, metadata)
			VALUES (
				(SELECT id FROM knowledge_items
				 WHERE team_id = $1::uuid AND source_type = $2 AND source_id = $3::uuid),
				$1::uuid, $4, $5, $6, $7, $8, '{}'::jsonb
			)
		`, doc.TeamID, doc.SourceType, doc.SourceID,
			c.ChunkIndex, c.Heading, chunkText, chunkHash, c.TokenEstimate)
		if err != nil {
			return fmt.Errorf("insert chunk %d: %w", c.ChunkIndex, err)
		}
	}

	return tx.Commit(ctx)
}

// RemoveFromIndex removes a knowledge item and its chunks.
func (ix *Indexer) RemoveFromIndex(ctx context.Context, teamID, sourceType, sourceID string) error {
	_, err := ix.pool.Exec(ctx, `
		DELETE FROM knowledge_items
		WHERE team_id = $1::uuid AND source_type = $2 AND source_id = $3::uuid
	`, teamID, sourceType, sourceID)
	return err
}

// ─── Chunking ───

// ChunkContent splits content into chunks by paragraphs.
// Each chunk is approximately MaxChunkSize chars.
func ChunkContent(content string) []ChunkData {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	// Split by double newlines (paragraph breaks)
	paragraphs := strings.Split(content, "\n\n")

	var chunks []ChunkData
	var current strings.Builder
	currentHeading := ""
	chunkIdx := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Check if this paragraph is a heading
		heading := ""
		if strings.HasPrefix(para, "#") {
			heading = strings.TrimLeft(para, "# ")
		}

		// If adding this paragraph would exceed max chunk size, flush current
		if current.Len() > 0 && current.Len()+len(para)+2 > MaxChunkSize {
			chunks = append(chunks, ChunkData{
				ChunkIndex:    chunkIdx,
				Heading:       currentHeading,
				ContentText:   current.String(),
				TokenEstimate: estimateTokens(current.String()),
			})
			chunkIdx++
			current.Reset()
			currentHeading = heading
		}

		// If the paragraph itself exceeds max chunk size, split it by sentences
		if len(para) > MaxChunkSize {
			for len(para) > 0 {
				splitAt := MaxChunkSize
				// Try to split at sentence boundary
				for i := splitAt; i > MaxChunkSize/2; i-- {
					if i < len(para) && (para[i] == '.' || para[i] == '!' || para[i] == '?') {
					splitAt = i + 1
					break
					}
				}
				if splitAt > len(para) {
					splitAt = len(para)
				}
				chunkText := para[:splitAt]
				para = para[splitAt:]
				chunks = append(chunks, ChunkData{
					ChunkIndex:    chunkIdx,
					Heading:       currentHeading,
					ContentText:   strings.TrimSpace(chunkText),
					TokenEstimate: estimateTokens(chunkText),
				})
				chunkIdx++
			}
			continue
		}

		if current.Len() == 0 {
			currentHeading = heading
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	// Flush remaining
	if current.Len() > 0 {
		chunks = append(chunks, ChunkData{
			ChunkIndex:    chunkIdx,
			Heading:       currentHeading,
			ContentText:   current.String(),
			TokenEstimate: estimateTokens(current.String()),
		})
	}

	return chunks
}

// estimateTokens returns an approximate token count (rough: words * 1.3).
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	est := int(float64(words) * 1.3)
	if est < 1 {
		est = 1
	}
	return est
}
