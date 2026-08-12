package migration

import (
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

// Source/embed byte equality is recorded in A2 through exact Git blob identities.
// This hermetic binary test independently asserts every embedded forward SHA-256
// and the ordered package identity.
func TestForwardAssetsMatchFrozenDigests(t *testing.T) {
	for _, rev := range []struct {
		asset assets.AssetName
		want  string
	}{
		{assets.AssetForward0002, Forward0002SHA256},
		{assets.AssetForward0003, Forward0003SHA256},
		{assets.AssetForward0004, Forward0004SHA256},
		{assets.AssetForward0005, Forward0005SHA256},
	} {
		if _, err := assets.Bytes(rev.asset); err != nil {
			t.Fatalf("embedded %s: %v", rev.asset, err)
		}
		got, err := assets.SHA256(rev.asset)
		if err != nil {
			t.Fatalf("sha %s: %v", rev.asset, err)
		}
		if got != rev.want {
			t.Fatalf("%s digest got=%s want=%s", rev.asset, got, rev.want)
		}
	}
}

func TestForwardPackageDigestFrozen(t *testing.T) {
	cat, err := ForwardCatalog()
	if err != nil {
		t.Fatalf("ForwardCatalog: %v", err)
	}
	got := ForwardPackageDigest(cat)
	if got != ForwardPackageSHA256 {
		t.Fatalf("forward package digest got=%s want=%s", got, ForwardPackageSHA256)
	}
}
