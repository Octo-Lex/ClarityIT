package migration

import (
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

// The backend CI container mounts services/api, not the repository root, so
// source-file equality is evidenced in A2 by the exact Git blob identities used
// to create the embedded paths. This hermetic runtime test proves the properties
// the binary itself relies on: exact embedded SHA-256s and ordered package identity.
func TestForwardAssetsMatchFrozenDigests(t *testing.T) {
	for _, rev := range []struct {
		asset assets.AssetName
		want  string
	}{
		{assets.AssetForward0002, Forward0002SHA256},
		{assets.AssetForward0003, Forward0003SHA256},
		{assets.AssetForward0004, Forward0004SHA256},
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
