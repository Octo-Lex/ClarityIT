package migration

// g6_p2_test.go — T3 required tests for the P2 adoption path.
// These are pure-logic tests verifying classification, asset integrity,
// and provenance without requiring a live database.

import (
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

// 1. v3.2 P2 fingerprint → P2 executable path
func TestG6_P2V32_ClassifiesAsAdoptP2(t *testing.T) {
	p := Probe{DatabaseName: "clarityit", PGMajor: 16, SourceFingerprint: P2SuccessorFingerprint}
	class, path, code := Classify(p)
	if class != ClassApprovedSource || path != PathAdoptP2 || code != CodeOK {
		t.Fatalf("P2 v3.2: got class=%q path=%q code=%q want approved_source/adopt_p2/OK", class, path, code)
	}
}

// 2. Historical v3.1 89b7792d... remains non-executable
func TestG6_HistoricalV31_RemainsNonExecutable(t *testing.T) {
	p := Probe{DatabaseName: "clarityit", PGMajor: 16, SourceFingerprint: P1P2Fingerprint}
	class, path, code := Classify(p)
	if class != ClassUnknownDrifted || path != PathBlock || code != CodeSourceProfileP1P2 {
		t.Fatalf("v3.1 historical: got class=%q path=%q code=%q want unknown_drifted/block/P1P2_NOT_EXECUTABLE", class, path, code)
	}
}

// 3. P3 fingerprint/path unchanged
func TestG6_P3PathUnchanged(t *testing.T) {
	p := Probe{DatabaseName: "clarityit", PGMajor: 16, SourceFingerprint: P3GoldenFingerprint}
	class, path, code := Classify(p)
	if class != ClassApprovedSource || path != PathAdopt || code != CodeOK {
		t.Fatalf("P3: got class=%q path=%q code=%q want approved_source/adopt_p3/OK", class, path, code)
	}
}

// 4. Unknown fingerprint blocks before DDL
func TestG6_UnknownFingerprintBlocks(t *testing.T) {
	p := Probe{DatabaseName: "clarityit", PGMajor: 16, SourceFingerprint: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}
	class, path, code := Classify(p)
	if path != PathBlock || code != CodeSourceProfileUnknown {
		t.Fatalf("unknown: got path=%q code=%q want block/SOURCE_PROFILE_UNKNOWN", path, code)
	}
	_ = class
}

// 5. P2 ledger uses P2 UUID/fingerprint, never P3 identity
func TestG6_P2ProfileID_DistinctFromP3(t *testing.T) {
	if P2ProfileID == P3ProfileID {
		t.Fatal("P2 and P3 profile IDs must be distinct")
	}
	if P2SuccessorFingerprint == P3GoldenFingerprint {
		t.Fatal("P2 and P3 fingerprints must be distinct")
	}
	if pid := profileIDForPath(PathAdoptP2); pid != P2ProfileID {
		t.Errorf("P2 profileID: got %q want %q", pid, P2ProfileID)
	}
	if pid := profileIDForPath(PathAdopt); pid != P3ProfileID {
		t.Errorf("P3 profileID: got %q want %q", pid, P3ProfileID)
	}
}

// 6. P2 does not select/embed historical legacy 001-040. Authorized WP-01
// forward revisions are explicitly separate and must not be mistaken for v1.
func TestG6_P2LegacyExclusion(t *testing.T) {
	allowed := authorizedEmbeddedSQLNames()
	for _, asset := range assets.AllAssets {
		name := string(asset)
		if isLegacyV1SQLName(name) {
			t.Errorf("legacy v1 asset embedded: %s", name)
		}
		if len(name) >= 4 && name[len(name)-4:] == ".sql" && !allowed[name] {
			t.Errorf("SQL asset outside explicit execution set: %s", name)
		}
	}
}

// 7. P3 adoption artifact SHA unchanged
func TestG6_P3ArtifactSHAUnchanged(t *testing.T) {
	got, err := assets.SHA256(assets.AssetAdoptP3)
	if err != nil {
		t.Fatalf("P3 SHA: %v", err)
	}
	if got != P3AdoptionArtifactSHA256 {
		t.Errorf("P3 artifact SHA: got %s want %s", got, P3AdoptionArtifactSHA256)
	}
}

// 8. P2 artifact SHA matches frozen identity
func TestG6_P2ArtifactSHA(t *testing.T) {
	got, err := assets.SHA256(assets.AssetAdoptP2)
	if err != nil {
		t.Fatalf("P2 SHA: %v", err)
	}
	if got != P2AdoptionArtifactSHA256 {
		t.Errorf("P2 artifact SHA: got %s want %s", got, P2AdoptionArtifactSHA256)
	}
}

// 9. Packaging checksum mutation fails closed (already tested by existing
// TestVerifyAllPasses, but confirm P2 is in the frozen digest map)
func TestG6_P2InFrozenDigestMap(t *testing.T) {
	if _, ok := FrozenDigest[assets.AssetAdoptP2]; !ok {
		t.Error("P2 artifact not in FrozenDigest map")
	}
}

// 10. Transform produces deterministic output for P2 artifact
func TestG6_P2TransformDeterministic(t *testing.T) {
	ts1, err := Transform(assets.AssetAdoptP2)
	if err != nil {
		t.Fatalf("transform P2: %v", err)
	}
	ts2, err := Transform(assets.AssetAdoptP2)
	if err != nil {
		t.Fatalf("transform P2 (2nd): %v", err)
	}
	if ts1.TransformedSHA256 != ts2.TransformedSHA256 {
		t.Error("P2 transform is non-deterministic")
	}
	if !ts1.NeedsSetConfig {
		t.Error("P2 artifact should need set_config (binds g3.source_commit)")
	}
	if psqlMetaLine.Match(ts1.Body) {
		t.Error("P2 transformed body contains psql meta-command")
	}
	if setConfigLine.Match(ts1.Body) {
		t.Error("P2 transformed body contains set_config line")
	}
	for _, line := range linesOf(ts1.Body) {
		if isBeginLine(line) || isCommitLine(line) {
			t.Errorf("P2 body contains outer BEGIN/COMMIT: %q", line)
		}
	}
}

func linesOf(b []byte) []string { return split("\n", string(b)) }

func split(sep, s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			out = append(out, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	out = append(out, s[start:])
	return out
}
