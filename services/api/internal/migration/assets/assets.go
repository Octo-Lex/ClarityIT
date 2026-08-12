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
	AssetRolesBootstrap AssetName = "0000_roles.sql"
	AssetPlatformSchema AssetName = "0000_platform.sql"
	AssetBaseline       AssetName = "0001_reconciled.sql"
	AssetSeed           AssetName = "0001_seed.sql"

	// Approved WP-00 adoption artifacts.
	AssetAdoptP3 AssetName = "0001_adopt_p3.sql"
	AssetAdoptP2 AssetName = "0001_adopt_p2.sql"

	// WP-01 forward revisions. These are applied only after exact accepted
	// revision-0001 ancestry/foundation verification.
	AssetForward0002 AssetName = "0002_kernel_foundation.sql"
	AssetForward0003 AssetName = "0003_kernel_integrity_hardening.sql"
	AssetForward0004 AssetName = "0004_packet_immutability_barrier.sql"
	AssetForward0005 AssetName = "0005_lineage_and_message_integrity.sql"

	// Identity manifests (read-only inputs; never executed as SQL).
	AssetG3A4Manifest    AssetName = "G3-A4-MANIFEST.json"
	AssetControlManifest AssetName = "CONTROL-SCHEMA-MANIFEST.json"
	AssetG2Manifest      AssetName = "TARGET-SCHEMA-MANIFEST.json"

	// Detached checksum inventories (provenance; never executed).
	AssetV2Checksums     AssetName = "v2-SHA256SUMS"
	AssetLegacyChecksums AssetName = "legacy-SHA256SUMS"
)

var AllAssets = []AssetName{
	AssetRolesBootstrap, AssetPlatformSchema, AssetBaseline, AssetSeed, AssetAdoptP3, AssetAdoptP2,
	AssetForward0002, AssetForward0003, AssetForward0004, AssetForward0005,
	AssetG3A4Manifest, AssetControlManifest, AssetG2Manifest,
	AssetV2Checksums, AssetLegacyChecksums,
}

var FreshInstallChain = []AssetName{
	AssetRolesBootstrap,
	AssetPlatformSchema,
	AssetBaseline,
	AssetSeed,
}

var AdoptionChain = []AssetName{AssetAdoptP3}
var P2AdoptionChain = []AssetName{AssetAdoptP2}

// ForwardChain is the only ordered post-0001 revision sequence in WP-01 G1.
// G1 current/accepted state is the complete atomic chain through 0005. Earlier
// forward revisions are immutable ancestry, not accepted persisted checkpoints.
var ForwardChain = []AssetName{
	AssetForward0002,
	AssetForward0003,
	AssetForward0004,
	AssetForward0005,
}

func Bytes(name AssetName) ([]byte, error) {
	b, err := v2FS.ReadFile("v2/" + string(name))
	if err != nil {
		return nil, fmt.Errorf("asset %s: %w", name, err)
	}
	return b, nil
}

func MustBytes(name AssetName) []byte {
	b, err := Bytes(name)
	if err != nil {
		panic(err)
	}
	return b
}

func SHA256(name AssetName) (string, error) {
	b, err := Bytes(name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

var ErrDigestMismatch = errors.New("embedded asset digest mismatch")
