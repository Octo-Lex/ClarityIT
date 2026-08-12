package assets

import "testing"

// One-shot bounded packaging probe. The next commit freezes this SHA-256 in the
// production catalog and removes this intentional failure.
func TestForward0005DigestProbe(t *testing.T) {
	digest, err := SHA256(AssetForward0005)
	if err != nil {
		t.Fatalf("FORWARD_0005_DIGEST_ERROR: %v", err)
	}
	t.Errorf("FORWARD_0005_SHA256=%s", digest)
}
