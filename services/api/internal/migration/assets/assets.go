// Package assets embeds the frozen WP-00 migration artifacts plus approved
// post-0001 forward revisions into the migration runner binary. Repository-root
// artifacts under migrations/ are the source of truth; embedded copies are
// byte-for-byte mirrors and divergence is a packaging failure.
//
// Legacy migration SQL (migrations/legacy/v1/001-040/) is never embedded — only
// the legacy checksum inventory is present as provenance evidence. Tests assert
// that no legacy SQL is selectable or executable.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
)

//go:embed v2/*.sql v2/*.json v2/v2-SHA256SUMS v2/legacy-SHA256SUMS
var v2FS embed.FS

// AssetName identifies an embedded migration artifact by a stable short name.
type AssetName string

const (
	// Fresh-install chain (executed in this order by the accepted WP-00 stage).
	AssetRolesBootstrap AssetName = "0000_roles.sql"      // privileged: five-role posture
	AssetPlatformSchema AssetName = "0000_platform.sql"   // platform control schema (4 tables)
	AssetBaseline       AssetName = "0001_reconciled.sql" // 64 product tables + 10 app functions
	AssetSeed           AssetName = "0001_seed.sql"       // canonical permissions + revision 0001 row

	// Approved WP-00 adoption artifacts.
	AssetAdoptP3 AssetName = "0001_adopt_p3.sql"
	AssetAdoptP2 AssetName = "0001_adopt_p2.sql"

	// WP-01 forward revisions. These are applied only after exact accepted
	// revision 0001 ancestry/foundation verification.
	AssetForward0002 AssetName = "0002_kernel_foundation.sql"
	AssetForward0003 AssetName = "0003_kernel_integrity_hardening.sql"

	// Identity manifests (read-only inputs; never executed as SQL).
	AssetG3A4Manifest    AssetName = "G3-A4-MANIFEST.json"
	AssetControlManifest AssetName = "CONTROL-SCHEMA-MANIFEST.json"
	AssetG2Manifest      AssetName = "TARGET-SCHEMA-MANIFEST.json"

	// Detached checksum inventories (provenance; never executed).
	AssetV2Checksums     AssetName = "v2-SHA256SUMS"
	AssetLegacyChecksums AssetName = "legacy-SHA256SUMS"
)

// AllAssets enumerates every embedded asset. Order is stable and is the single
// place that proves legacy SQL (001-040) is not embedded.
var AllAssets = []AssetName{
	AssetRolesBootstrap, AssetPlatformSchema, AssetBaseline, AssetSeed, AssetAdoptP3, AssetAdoptP2,
	AssetForward0002, AssetForward0003,
	AssetG3A4Manifest, AssetControlManifest, AssetG2Manifest,
	AssetV2Checksums, AssetLegacyChecksums,
}

// FreshInstallChain is the exact execution order for a fresh WP-00 install.
var FreshInstallChain = []AssetName{
	AssetRolesBootstrap,
	AssetPlatformSchema,
	AssetBaseline,
	AssetSeed,
}

// AdoptionChain is the execution chain for approved P3 adoption.
var AdoptionChain = []AssetName{
	AssetAdoptP3,
}

// P2AdoptionChain is the execution chain for approved P2 adoption.
var P2AdoptionChain = []AssetName{
	AssetAdoptP2,
}

// ForwardChain is the only ordered post-0001 revision sequence in WP-01 G1.
// G1 current/accepted state is the complete chain through 0003; 0002 alone is
// an intermediate revision, not an accepted WP-01 target.
var ForwardChain = []AssetName{
	AssetForward0002,
	AssetForward0003,
}

// Bytes returns the immutable raw embedded bytes for an asset.
func Bytes(name AssetName) ([]byte, error) {
	b, err := v2FS.ReadFile("v2/" + string(name))
	if err != nil {
		return nil, fmt.Errorf("asset %s: %w", name, err)
	}
	return b, nil
}

// MustBytes is Bytes without the error (panics on a missing embedded asset,
// which is a packaging defect rather than a runtime database condition).
func MustBytes(name AssetName) []byte {
	b, err := Bytes(name)
	if err != nil {
		panic(err)
	}
	return b
}

// SHA256 returns the lowercase hex SHA-256 of the asset's embedded bytes.
func SHA256(name AssetName) (string, error) {
	b, err := Bytes(name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ErrDigestMismatch is returned by Verify when an asset's embedded bytes do not
// hash to its frozen digest.
var ErrDigestMismatch = errors.New("embedded asset digest mismatch")
