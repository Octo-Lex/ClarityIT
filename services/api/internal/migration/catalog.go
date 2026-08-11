package migration

// catalog.go — frozen identities, embedded asset registry, and packaging
// self-verification. This is the single source of truth for every SHA-256 the
// G4 runner is bound to. It does not import the database; it is pure data +
// crypto so it can run at startup before any connection is opened.
//
// Frozen identities are transcribed verbatim from G4-AUTHORIZATION-AND-PLAN.md
// section 2. Any mismatch between these constants and the embedded bytes (or
// the live database) is a G4 stop condition.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/clarityit/api/internal/migration/assets"
)

// Frozen identities — G4-AUTHORIZATION-AND-PLAN.md §2.
const (
	// Signed G3 tip (the authority chain; not the producing implementation).
	G3SignedTip = "97f83e4ac0609994b64493c7a8b2b76208545bb1"

	// G3 producing implementation commit (the revision source_commit for the
	// governed fresh-install path is the frozen G2 commit below; the producing
	// implementation commit is recorded separately in reconciliation evidence).
	G3ProducingImplementation = "570a0ec7e31087d1dd6db22e14935e21e7481cf6"

	// Frozen G2 commit (the source_commit recorded in the fresh-install
	// revision 0001 row by the seed artifact).
	G2FrozenCommit = "f04f94faad0105d1c3274e9c7974d44f936a0d28"

	// Product manifest blob SHA-256 (284064 bytes). G2 product identity.
	G2ManifestBlobSHA256 = "1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63"

	// Control manifest SHA-256. Platform control-schema identity (separate
	// from G2 product identity).
	ControlManifestSHA256 = "3fd65e917ded8b7d59a1f42051b69f41e4b5c24f583f9524deaccdfdfb1add66"

	// Composite installation SHA-256. Recomputed from the 6 labeled components
	// by CompositeDigest; must match exactly.
	CompositeInstallationSHA256 = "8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084"

	// Governed target fingerprint — the convergence identity that both
	// fresh-install and P3-adopted databases must reach.
	GovernedTargetFingerprint = "9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6"

	// P3 adoption artifact SHA-256 (migrations/v2/adoption/0001_adopt_p3.sql).
	P3AdoptionArtifactSHA256 = "a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359"

	// P3 golden source fingerprint (the G1-approved source-profile allowlist
	// entry; the only fingerprint executable through adoption in G4).
	P3GoldenFingerprint = "cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b"

	// P1/P2 source fingerprint (historical v3.1). Recognized but NOT executable;
	// the v3.2 successor is executable through the G6 P2 adoption path.
	P1P2Fingerprint = "89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83"

	// P1/P2 source fingerprint (v3.2 successor). The executable P2 adoption
	// fingerprint established by G6-TERMINAL-CLOSURE-AUTH-2026-08-11.
	P2SuccessorFingerprint = "57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb"

	// P2 adoption artifact SHA-256 (migrations/v2/adoption/0001_adopt_p2.sql).
	P2AdoptionArtifactSHA256 = "d98f67cde22de734a791dfb81a82c40727a510ee6fd5333a213b89bb8672b70c"

	// Baseline SQL checksum — the revision-0001 checksum recorded by both the
	// seed artifact and the adoption artifact (they converge on this value).
	BaselineChecksum = "1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508"

	// Per-file frozen digests for the embedded SQL assets (G3-A4-MANIFEST
	// components + detached SHA256SUMS). Used by VerifyAll.
	PlatformSQLSHA256 = "3e3ace0d2bbb9378c31f012bf85e56a8657ffec37622d6216bbdff882768fbec"
	RolesSQLSHA256    = "a6fb24da88c1d265d9b511745198d333dfe357027343310e07206f415af53163"
	SeedSQLSHA256     = "afc923bd77172810dfbea9ab21c5e4244050e51b52686a25203356764a78d990"

	// P3 adoption profile identifiers (UUIDv5, deterministic; transcribed from
	// the G3-A4 manifest adoption block).
	P3ProfileID = "7c5cb0b9-1fb4-540d-9433-f0196ff6f7bb"

	// P2 successor profile UUID (from G6-TERMINAL-CLOSURE-AUTH-2026-08-11).
	P2ProfileID = "7b5b8b87-3467-5fd5-9bac-3dbcdd858178"

	// G1 approval reference (the commit that approved the P1/P2/P3 profile pack).
	G1ApprovalRef = "3b4a6fdeb35473e5f73ca74bafa479bd2648fb10"

	// P3 fixture source commit (where the P3 schema.sql/seed.sql originate).
	P3SourceCommit = "29c4cdcb4c7bd9f13209f5627b55f4fabbd08a33"
)

// FrozenDigest maps an embedded asset to its frozen SHA-256. Assets not listed
// here (the manifests that *contain* digests, and the detached checksum files)
// are verified structurally rather than against a single frozen digest.
var FrozenDigest = map[assets.AssetName]string{
	assets.AssetBaseline:        BaselineChecksum,
	assets.AssetSeed:            SeedSQLSHA256,
	assets.AssetPlatformSchema:  PlatformSQLSHA256,
	assets.AssetRolesBootstrap:  RolesSQLSHA256,
	assets.AssetAdoptP3:         P3AdoptionArtifactSHA256,
	assets.AssetAdoptP2:         P2AdoptionArtifactSHA256,
	assets.AssetG2Manifest:      G2ManifestBlobSHA256,
	assets.AssetControlManifest: ControlManifestSHA256,
}

// Composite domain and component labels — transcribed verbatim from
// generate_g3.py::composite_digest. The Go port must produce identical bytes.
const (
	compositeDomain    = "clarityit-g3-composite-v1\x00"
	lblProductManifest = "product_manifest_blob_sha256"
	lblControlManifest = "control_manifest"
	lblBaselineSQL     = "baseline_sql"
	lblSeedSQL         = "seed_sql"
	lblRolesBootstrap  = "role_bootstrap_sql"
	lblLegacyChecksums = "legacy_checksum_inventory"
)

// CompositeDigest reproduces generate_g3.py::composite_digest byte-for-byte:
//
//	SHA-256( "clarityit-g3-composite-v1\0"
//	         + Σ components[ uint32be(label_len) | label | uint64be(data_len) | data ] )
//
// Component 1's data is the 64-char lowercase hex SHA-256 string (NOT the raw
// 32 bytes, NOT the manifest file bytes). Components 2-6 are the file bytes.
// The order is fixed.
func CompositeDigest() (string, error) {
	h := sha256.New()
	h.Write([]byte(compositeDomain))

	controlBytes, err := assets.Bytes(assets.AssetControlManifest)
	if err != nil {
		return "", err
	}
	baselineBytes, err := assets.Bytes(assets.AssetBaseline)
	if err != nil {
		return "", err
	}
	seedBytes, err := assets.Bytes(assets.AssetSeed)
	if err != nil {
		return "", err
	}
	rolesBytes, err := assets.Bytes(assets.AssetRolesBootstrap)
	if err != nil {
		return "", err
	}
	legacyBytes, err := assets.Bytes(assets.AssetLegacyChecksums)
	if err != nil {
		return "", err
	}

	// Component 1: the literal hex string of the G2 manifest blob SHA-256.
	writeComponent(h, lblProductManifest, []byte(G2ManifestBlobSHA256))
	writeComponent(h, lblControlManifest, controlBytes)
	writeComponent(h, lblBaselineSQL, baselineBytes)
	writeComponent(h, lblSeedSQL, seedBytes)
	writeComponent(h, lblRolesBootstrap, rolesBytes)
	writeComponent(h, lblLegacyChecksums, legacyBytes)

	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeComponent(h sha256Writer, label string, data []byte) {
	var b4 [4]byte
	var b8 [8]byte
	labelBytes := []byte(label)
	binary.BigEndian.PutUint32(b4[:], uint32(len(labelBytes)))
	h.Write(b4[:])
	h.Write(labelBytes)
	binary.BigEndian.PutUint64(b8[:], uint64(len(data)))
	h.Write(b8[:])
	h.Write(data)
}

type sha256Writer interface {
	Write([]byte) (int, error)
}

// VerifyResult is the outcome of packaging self-verification.
type VerifyResult struct {
	Composite   string            // recomputed composite digest
	PerAsset    map[string]string // asset name -> recomputed SHA-256
	CompositeOK bool
	Mismatches  []string // asset names whose embedded bytes diverge from frozen
}

// VerifyAll recomputes every per-file SHA-256 and the composite digest from the
// embedded bytes and reports any divergence from the frozen identities. This
// runs before any DDL; a non-empty Mismatches list or CompositeOK==false is a
// hard stop.
func VerifyAll() (VerifyResult, error) {
	res := VerifyResult{PerAsset: map[string]string{}}

	for _, name := range assets.AllAssets {
		got, err := assets.SHA256(name)
		if err != nil {
			return res, fmt.Errorf("verify %s: %w", name, err)
		}
		res.PerAsset[string(name)] = got
		if want, ok := FrozenDigest[name]; ok && got != want {
			res.Mismatches = append(res.Mismatches,
				fmt.Sprintf("%s: embedded=%s frozen=%s", name, got, want))
		}
	}

	comp, err := CompositeDigest()
	if err != nil {
		return res, err
	}
	res.Composite = comp
	res.CompositeOK = comp == CompositeInstallationSHA256

	if len(res.Mismatches) > 0 || !res.CompositeOK {
		return res, ErrPackagingMismatch
	}
	return res, nil
}

// ErrPackagingMismatch indicates embedded asset bytes diverge from frozen
// identities. The runner must not proceed to DDL when this is non-nil.
var ErrPackagingMismatch = errors.New("packaging mismatch: embedded assets diverge from frozen identities")
