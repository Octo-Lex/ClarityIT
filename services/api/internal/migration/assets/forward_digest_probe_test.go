package assets

import "testing"

// TestForwardDigestProbe is a bounded G1 packaging probe. It intentionally
// fails once to surface the SHA-256 of the exact embedded bytes. The next G1
// commit replaces this probe with permanent frozen-digest assertions.
func TestForwardDigestProbe(t *testing.T) {
	for _, name := range ForwardChain {
		digest, err := SHA256(name)
		if err != nil {
			t.Fatalf("FORWARD_DIGEST_ERROR %s: %v", name, err)
		}
		t.Errorf("FORWARD_DIGEST %s=%s", name, digest)
	}
}
