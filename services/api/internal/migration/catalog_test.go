package migration

import (
	"strings"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

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

// TestLegacySQLNotEmbedded proves no legacy migration SQL (001-040) is present
// in the embed boundary. Only the legacy checksum *inventory* (provenance) is
// embedded. This is the "complete exclusion of legacy migration replay"
// guarantee at the packaging layer.
func TestLegacySQLNotEmbedded(t *testing.T) {
	for _, name := range assets.AllAssets {
		s := string(name)
		// Reject any embedded file that looks like a numbered legacy migration.
		if len(s) >= 3 && s[0] >= '0' && s[0] <= '9' {
			// Allowed numbered assets are the G3 v2 files: 0000_platform,
			// 0000_roles, 0001_reconciled, 0001_seed, 0001_adopt_p3.
			// Legacy files are 001..040 with descriptive names like
			// 001_core_extensions.sql. Assert the only numbered embeds are G3.
			switch s {
			case "0000_platform.sql", "0000_roles.sql",
				"0001_reconciled.sql", "0001_seed.sql", "0001_adopt_p3.sql":
				// allowed
			default:
				t.Errorf("legacy-looking asset embedded: %s", s)
			}
		}
	}
	// Also confirm the legacy checksum inventory is the ONLY legacy artifact
	// embedded, and that it is an inventory (text), not SQL.
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
