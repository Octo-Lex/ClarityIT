package migration

// adapters.go — small adapters that keep apply.go from importing the fingerprint
// package directly (preserving the clean dependency graph for the privilege-
// boundary denylist). These are thin aliases/wrappers.

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/clarityit/api/internal/migration/fingerprint"
	"github.com/jackc/pgx/v5"
)

// fingerprintSigned / fingerprintControl are aliases for the fingerprint
// package's manifest types, so apply.go doesn't import fingerprint directly.
type fingerprintSigned = fingerprint.SignedG2Manifest
type fingerprintControl = fingerprint.ControlManifestFunctions

// fpGovernedCapture / fpGovernedFingerprint delegate to the fingerprint package.
// pgx.Tx satisfies fingerprint.pgxQuerier (both implement Query/QueryRow).
func fpGovernedCapture(ctx context.Context, q pgx.Tx, signed *fingerprintSigned, control *fingerprintControl) (map[string]any, error) {
	return fingerprint.GovernedCapture(ctx, q, signed, control)
}
func fpGovernedFingerprint(cap map[string]any) (string, error) {
	return fingerprint.GovernedFingerprint(cap)
}

// profilerCaptureAdapter / profilerFingerprintAdapter delegate to the
// fingerprint package's source-profiler functions (used by the re-probe).
func profilerCaptureAdapter(ctx context.Context, q pgx.Tx) (map[string]any, error) {
	return fingerprint.ProfilerCapture(ctx, q)
}
func profilerFingerprintAdapter(cap map[string]any) (string, error) {
	return fingerprint.ProfilerFingerprint(cap)
}

// newRunID generates a v4 UUID string for migration_runs.run_id. Uses crypto/rand
// so the id is unpredictable (avoids collision/overlap across runs).
func newRunID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read failing is catastrophic; fall back to a timestamp-based id.
		return fmt.Sprintf("00000000-0000-4000-8000-%012x", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
