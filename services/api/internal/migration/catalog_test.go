package migration

import (
	"strings"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

// authorizedEmbeddedSQLNames is the exact executable SQL package boundary:
// accepted WP-00 assets plus the explicit WP-01 ForwardChain. It intentionally
// does not infer authorization from a numeric filename.
func authorizedEmbeddedSQLNames() map[string]bool {
	allowed := map[string]bool{
		string(assets.AssetRolesBootstrap): true,
		string(assets.AssetPlatformSchema): true,
		string(assets.AssetBaseline):       true,
		string(assets.AssetSeed):           true,
		string(assets.AssetAdoptP3):        true,
		string(assets.AssetAdoptP2):        true,
	}
	for _, asset := range assets.ForwardChain {
		allowed[string(asset)] = true
	}
	return allowed
}

// isLegacyV1SQLName recognizes the historical numbered migration naming shape
// 001_*.sql through 040_*.sql. WP-01 forward files use four-digit revisions
// (0002_*.sql etc.) and are therefore never confused with the legacy series.
func isLegacyV1SQLName(name string) bool {
	if len(name) < 8 || name[3] != '_' || !strings.HasSuffix(name, ".sql") {
		return false
	}
	if name[0] < '0' || name[0] > '9' || name[1] < '0' || name[1] > '9' || name[2] < '0' || name[2] > '9' {
		return false
	}
	n := int(name[0]-'0')*100 + int(name[1]-'0')*10 + int(name[2]-'0')
	return n >= 1 && n <= 40
}

// TestCompositeDigestReproducesFrozenIdentity is the load-bearing packaging
// gate: the Go composite-digest port must reproduce the frozen installation
// SHA-256 from the embedded bytes. If this fails, the embed boundary is wrong,
// the port is wrong, or a frozen artifact drifted — all are G4 stops.
func TestCompositeDigestReproducesFrozenIdentity(t *testing.T) {
	got, err := CompositeDigest()
	if err != nil {
		t.Fatalf("CompositeDigest: %v", err)
	}
	if got != CompositeInstallationSHA256 {
		t.Fatalf("composite digest mismatch:\n  got    %s\n  frozen %s", got, CompositeInstallationSHA256)
	}
}

// TestVerifyAllPasses confirms every embedded asset hashes to its frozen digest
// and the composite matches. This is the preflight packaging check.
func TestVerifyAllPasses(t *testing.T) {
	res, err := VerifyAll()
	if err != nil {
		t.Fatalf("VerifyAll returned error: %v\nmismatches: %v\ncompositeOK=%v composite=%s",
			err, res.Mismatches, res.CompositeOK, res.Composite)
	}
	if !res.CompositeOK {
		t.Fatalf("composite not OK: got %s want %s", res.Composite, CompositeInstallationSHA256)
	}
	if len(res.Mismatches) > 0 {
		t.Fatalf("unexpected mismatches: %v", res.Mismatches)
	}
}

// TestPerAssetDigestsMatchFrozen checks each individually-listed asset against
// its frozen digest for a precise failure message.
func TestPerAssetDigestsMatchFrozen(t *testing.T) {
	for name, want := range FrozenDigest {
		got, err := assets.SHA256(assets.AssetName(name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != want {
			t.Errorf("asset %s:\n  got    %s\n  frozen %s", name, got, want)
		}
	}
}

// TestLegacySQLNotEmbedded proves no historical legacy migration SQL (001-040)
// is present in the embed boundary. Authorized post-0001 WP-01 forward SQL is
// explicitly enumerated via ForwardChain and does not weaken this guarantee.
func TestLegacySQLNotEmbedded(t *testing.T) {
	allowed := authorizedEmbeddedSQLNames()
	for _, asset := range assets.AllAssets {
		name := string(asset)
		if isLegacyV1SQLName(name) {
			t.Errorf("legacy v1 migration embedded: %s", name)
		}
		if strings.HasSuffix(name, ".sql") && !allowed[name] {
			t.Errorf("SQL asset embedded outside explicit execution set: %s", name)
		}
	}
	// Also confirm the legacy checksum inventory is provenance only, never SQL.
	inv, err := assets.Bytes(assets.AssetLegacyChecksums)
	if err != nil {
		t.Fatalf("legacy inventory missing: %v", err)
	}
	if !strings.Contains(string(inv), "MUST NOT be selected for execution") {
		t.Error("legacy checksum inventory missing its non-execution marker")
	}
	if strings.Contains(string(inv), "CREATE TABLE") {
		t.Error("legacy checksum inventory unexpectedly contains DDL")
	}
}
