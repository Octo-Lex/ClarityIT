// Package assets embeds the frozen G3 migration artifacts into the G4 runner
// binary. Repository-root artifacts under migrations/ are the source of truth;
// the copies embedded here are byte-for-byte refreshes produced by gen_assets.sh
// (run from the repo root). A divergence test (assets_test.go) fails if any
// embedded copy diverges from its repo original or from a frozen digest.
//
// The package embeds ONLY the G3 v2 artifacts and provenance inventories. It
// never embeds legacy migration SQL (migrations/legacy/v1/001-040/) — only the
// legacy checksum inventory (migrations/legacy/v1/SHA256SUMS), which is
// provenance evidence, not executable SQL. A test asserts no legacy SQL is
// embedded and that no exported API can select it.
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

// AssetName identifies a frozen embedded artifact by a stable short name.
type AssetName string

const (
	// Fresh-install chain (executed in this order by the runner).
	AssetRolesBootstrap AssetName = "0000_roles.sql"      // privileged: five-role posture
	AssetPlatformSchema AssetName = "0000_platform.sql"   // platform control schema (4 tables)
	AssetBaseline       AssetName = "0001_reconciled.sql" // 64 product tables + 10 app functions
	AssetSeed           AssetName = "0001_seed.sql"       // 7 canonical permissions + revision 0001 row

	// Approved P3 adoption (self-contained: brings its own platform ledger).
	AssetAdoptP3 AssetName = "0001_adopt_p3.sql"

	// Approved P2 adoption (G6 successor; self-contained like P3).
	AssetAdoptP2 AssetName = "0001_adopt_p2.sql"

	// Identity manifests (read-only inputs; never executed as SQL).
	AssetG3A4Manifest    AssetName = "G3-A4-MANIFEST.json"
	AssetControlManifest AssetName = "CONTROL-SCHEMA-MANIFEST.json"
	AssetG2Manifest      AssetName = "TARGET-SCHEMA-MANIFEST.json"

	// Detached checksum inventories (provenance; never executed).
	AssetV2Checksums     AssetName = "v2-SHA256SUMS"
	AssetLegacyChecksums AssetName = "legacy-SHA256SUMS"
)

// AllAssets enumerates every embedded asset. Order is stable and is the single
// place that proves the legacy SQL (001-040) is not embedded.
var AllAssets = []AssetName{
	AssetRolesBootstrap, AssetPlatformSchema, AssetBaseline, AssetSeed, AssetAdoptP3, AssetAdoptP2,
	AssetG3A4Manifest, AssetControlManifest, AssetG2Manifest,
	AssetV2Checksums, AssetLegacyChecksums,
}

// FreshInstallChain is the exact execution order for a fresh install. The
// runner owns the transaction and strips each artifact's outer BEGIN/COMMIT.
var FreshInstallChain = []AssetName{
	AssetRolesBootstrap,
	AssetPlatformSchema,
	AssetBaseline,
	AssetSeed,
}

// AdoptionChain is the execution chain for approved P3 adoption. The adoption
// artifact is self-contained (it recreates the platform ledger itself).
var AdoptionChain = []AssetName{
	AssetAdoptP3,
}

// P2AdoptionChain is the execution chain for approved P2 adoption (G6 successor).
var P2AdoptionChain = []AssetName{
	AssetAdoptP2,
}

// Bytes returns the immutable raw embedded bytes for an asset.
func Bytes(name AssetName) ([]byte, error) {
	b, err := v2FS.ReadFile("v2/" + string(name))
	if err != nil {
		return nil, fmt.Errorf("asset %s: %w", name, err)
	}
	return b, nil
}

// MustBytes is Bytes without the error (panics on a missing frozen asset, which
// is a compile-time packaging defect, not a runtime condition).
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
