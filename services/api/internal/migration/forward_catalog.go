package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
)

const (
	ForwardTargetVersion = "0005"
	// Binds exact ordered version/name/checksum tuples for the complete G1 batch.
	// The target-manifest identity is frozen only after a live PostgreSQL rehearsal
	// of this exact package succeeds.
	ForwardPackageSHA256        = "9f19b9c94f5d9e978a5cf815d0971788a6d744b5a0b91d6ba5dab1f67135f1ef"
	ForwardTargetManifestSHA256 = ""
)

var (
	ErrForwardHistory      = errors.New("forward revision history is inconsistent")
	ErrForwardPackaging    = errors.New("forward package identity mismatch")
	ErrForwardManifest     = errors.New("forward target manifest mismatch")
	ErrForwardIntermediate = errors.New("intermediate forward revision state is not an accepted G1 checkpoint")
	ErrForwardFoundation   = errors.New("WP-00 foundation does not match the accepted pre-forward identity")
)

type ForwardRevision struct {
	Version  string
	Name     string
	Asset    assets.AssetName
	Checksum string
}

type forwardLedgerRow struct {
	Version  string
	Name     string
	Checksum string
	Success  bool
}

type ForwardInspection struct {
	CurrentVersion string
	Current        bool
	FoundationOnly bool
	ManifestDigest string
	PackageDigest  string
}

func ForwardCatalog() ([]ForwardRevision, error) {
	specs := []struct {
		version string
		name    string
		asset   assets.AssetName
	}{
		{"0002", "wp01-kernel-foundation", assets.AssetForward0002},
		{"0003", "wp01-kernel-integrity-hardening", assets.AssetForward0003},
		{"0004", "wp01-packet-immutability-barrier", assets.AssetForward0004},
		{"0005", "wp01-lineage-and-message-integrity", assets.AssetForward0005},
	}
	out := make([]ForwardRevision, 0, len(specs))
	for _, s := range specs {
		want, ok := FrozenDigest[s.asset]
		if !ok || want == "" {
			return nil, fmt.Errorf("%w: no frozen SHA-256 for %s", ErrForwardPackaging, s.asset)
		}
		got, err := assets.SHA256(s.asset)
		if err != nil {
			return nil, fmt.Errorf("%w: hash %s: %v", ErrForwardPackaging, s.asset, err)
		}
		if got != want {
			return nil, fmt.Errorf("%w: %s embedded=%s frozen=%s", ErrForwardPackaging, s.asset, got, want)
		}
		out = append(out, ForwardRevision{Version: s.version, Name: s.name, Asset: s.asset, Checksum: want})
	}
	if err := validateForwardCatalog(out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateForwardCatalog(cat []ForwardRevision) error {
	if len(cat) != 4 {
		return fmt.Errorf("%w: expected 4 G1 revisions, got %d", ErrForwardPackaging, len(cat))
	}
	for i, r := range cat {
		want := fmt.Sprintf("%04d", i+2)
		if r.Version != want || r.Name == "" || r.Checksum == "" {
			return fmt.Errorf("%w: invalid catalog entry %d version=%q", ErrForwardPackaging, i, r.Version)
		}
	}
	return nil
}

func ForwardPackageDigest(cat []ForwardRevision) string {
	h := sha256.New()
	h.Write([]byte("clarityit-wp01-forward-package-v1\x00"))
	for _, r := range cat {
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00", r.Version, r.Name, r.Checksum)
	}
	return hex.EncodeToString(h.Sum(nil))
}

type forwardQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readForwardHistory(ctx context.Context, q forwardQueryer) ([]forwardLedgerRow, error) {
	rows, err := q.Query(ctx, `SELECT version, name, checksum, success FROM platform.schema_revisions ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read revision history: %w", err)
	}
	defer rows.Close()
	var out []forwardLedgerRow
	for rows.Next() {
		var r forwardLedgerRow
		if err := rows.Scan(&r.Version, &r.Name, &r.Checksum, &r.Success); err != nil {
			return nil, fmt.Errorf("scan revision history: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ApplyForward commits 0002..0005 atomically, so only exact foundation or exact
// complete G1 target are legitimate persisted histories.
func validateForwardHistory(rows []forwardLedgerRow, cat []ForwardRevision) (string, error) {
	if len(rows) == 0 || rows[0].Version != "0001" || rows[0].Checksum != BaselineChecksum || !rows[0].Success {
		return "", fmt.Errorf("%w: exact successful 0001 ancestry required", ErrForwardHistory)
	}
	if len(rows) == 1 {
		return "foundation", nil
	}
	if len(rows) != 1+len(cat) {
		return "", fmt.Errorf("%w: got %d rows; expected 1 or %d", ErrForwardIntermediate, len(rows), 1+len(cat))
	}
	for i, expected := range cat {
		got := rows[i+1]
		if got.Version != expected.Version || got.Name != expected.Name || got.Checksum != expected.Checksum || !got.Success {
			return "", fmt.Errorf("%w: %s mismatch", ErrForwardHistory, expected.Version)
		}
	}
	return "current", nil
}

func validateForwardSQL(body []byte) error {
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "\\") {
			return fmt.Errorf("psql meta-command forbidden: %q", line)
		}
		u := strings.ToUpper(line)
		if u == "BEGIN;" || u == "COMMIT;" {
			return fmt.Errorf("runner-owned transaction boundary found: %s", line)
		}
	}
	return nil
}
