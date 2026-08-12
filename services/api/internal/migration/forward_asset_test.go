package migration

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

func TestForwardAssetsMatchFrozenDigestsAndRepositorySources(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", "..", ".."))
	for _, rev := range []struct {
		asset assets.AssetName
		path  string
		want  string
	}{
		{assets.AssetForward0002, "migrations/v2/forward/0002_kernel_foundation.sql", Forward0002SHA256},
		{assets.AssetForward0003, "migrations/v2/forward/0003_kernel_integrity_hardening.sql", Forward0003SHA256},
		{assets.AssetForward0004, "migrations/v2/forward/0004_packet_immutability_barrier.sql", Forward0004SHA256},
	} {
		embedded, err := assets.Bytes(rev.asset)
		if err != nil {
			t.Fatalf("embedded %s: %v", rev.asset, err)
		}
		got, err := assets.SHA256(rev.asset)
		if err != nil {
			t.Fatalf("sha %s: %v", rev.asset, err)
		}
		if got != rev.want {
			t.Fatalf("%s digest got=%s want=%s", rev.asset, got, rev.want)
		}
		source, err := os.ReadFile(filepath.Join(repoRoot, rev.path))
		if err != nil {
			t.Fatalf("read source %s: %v", rev.path, err)
		}
		if !bytes.Equal(source, embedded) {
			t.Fatalf("source/embed divergence for %s", rev.asset)
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
