package canonicalize

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFrozenVectors is the second hard gate: the Go canonical serializer must
// reproduce the Python-produced golden bytes for every Phase 0 positive vector.
// It walks testdata/ (excluding negative/), loads each .input.json with
// UseNumber so integers stay exact, canonicalizes, and asserts byte-equality
// with the sibling .expected.bytes.
func TestFrozenVectors(t *testing.T) {
	td := "testdata"
	entries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	ran := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".input.json") {
			continue
		}
		base := strings.TrimSuffix(name, ".input.json")
		t.Run(base, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(td, name))
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			var v any
			if err := dec.Decode(&v); err != nil {
				t.Fatalf("decode input: %v", err)
			}
			got, err := Marshal(v)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			want, err := os.ReadFile(filepath.Join(td, base+".expected.bytes"))
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("mismatch for %s:\n  got   %q\n  want  %q", base, got, want)
			}
			if bytes.HasSuffix(got, []byte("\n")) {
				t.Errorf("%s: output has trailing newline", base)
			}
		})
		ran++
	}
	if ran == 0 {
		t.Fatal("no positive vectors found")
	}
	t.Logf("ran %d positive vectors", ran)
}

// TestRejectsFloats confirms the serializer refuses float values. This is the
// no-fallback rule: a float in a frozen projection is a G4 stop, not a
// serialization workaround.
func TestRejectsFloats(t *testing.T) {
	for _, v := range []any{float32(1.5), float64(1.5)} {
		if _, err := Marshal(v); err == nil {
			t.Errorf("Marshal(%T) unexpectedly succeeded; want ErrFloatUnsupported", v)
		}
	}
}

// TestNegativeFloatVectorsAreNotMatched documents that the negative vectors
// under testdata/negative/ are informational only. The serializer does NOT
// attempt to match Python float formatting. This test loads each negative
// vector and asserts that the serializer either rejects the value (float) or —
// for bare-scalar integer negatives — matches. Currently all three negatives
// are floats, so all must be rejected.
func TestNegativeFloatVectorsAreNotMatched(t *testing.T) {
	td := filepath.Join("testdata", "negative")
	entries, err := os.ReadDir(td)
	if err != nil {
		t.Fatalf("read negative testdata: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".input.json") {
			continue
		}
		base := strings.TrimSuffix(name, ".input.json")
		t.Run(base, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(td, name))
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.UseNumber()
			var v any
			if err := dec.Decode(&v); err != nil {
				t.Fatalf("decode input: %v", err)
			}
			_, err = Marshal(v)
			// All three negative vectors are floats; the serializer must reject.
			if err == nil {
				t.Errorf("%s: float value was not rejected (negative vectors must not be matched)", base)
			}
		})
	}
}
