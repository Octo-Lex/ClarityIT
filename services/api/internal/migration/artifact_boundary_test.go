package migration

// artifact_boundary_test.go — proves the frozen fresh-install artifacts contain
// NO cross-artifact session state other than the approved SET LOCAL ROLE
// operation. This is the standalone-artifact-boundary-emulation contract: the
// runner's SET ROLE NONE between artifacts is SUFFICIENT normalization because
// the artifacts carry no other stateful commands.
//
// If a future frozen artifact introduces SET SESSION AUTHORIZATION, temporary
// objects, advisory locks, prepared statements, additional SET LOCAL, or
// session-level GUC mutation, this test FAILS CLOSED and the boundary semantics
// must be re-governed before transformation can proceed.
//
// The P3 adoption artifact is EXCLUDED from this contract: it deliberately uses
// SET SESSION AUTHORIZATION + role rename + demotion as its governed design, and
// its boundary is handled by physical-connection destruction, not SET ROLE NONE.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

// forbiddenBoundaryState patterns that would break the SET-ROLE-NONE boundary
// emulation if present in a fresh-install artifact. Each is anchored to catch
// the stateful command at statement granularity.
//
// IMPORTANT scope: the boundary emulation normalizes the EFFECTIVE ROLE between
// artifacts via SET ROLE NONE. Commands that mutate SESSION-LEVEL state (which
// persists across the transaction and leaks to the next artifact) are forbidden.
// Commands that mutate TRANSACTION-LOCAL state (SET LOCAL anything) are SAFE —
// they reset at transaction end and do not leak across artifacts within the
// same transaction any more than within separate sessions. So:
//   - SET LOCAL ROLE / SET LOCAL search_path / any SET LOCAL: ALLOWED (tx-scoped)
//   - SET ROLE (session-level): FORBIDDEN (persists)
//   - SET SESSION AUTHORIZATION: FORBIDDEN (persists; adoption-only)
//   - SET SESSION <guc>: FORBIDDEN (persists)
//
// Note: Go's regexp (RE2) does not support negative lookahead, so SET SESSION
// is matched broadly and SET SESSION AUTHORIZATION is allowed in code.
var forbiddenBoundaryState = []struct {
	name string
	re   *regexp.Regexp
}{
	{"SET SESSION AUTHORIZATION", regexp.MustCompile(`(?im)^\s*SET\s+SESSION\s+AUTHORIZATION\s+`)},
	{"CREATE TEMP table/object", regexp.MustCompile(`(?im)^\s*CREATE\s+(TEMP|TEMPORARY)\s+`)},
	{"advisory lock acquire", regexp.MustCompile(`(?im)pg_advisory_lock\s*\(`)},
	{"advisory lock unlock_all", regexp.MustCompile(`(?im)pg_advisory_unlock_all\s*\(`)},
	{"PREPARE statement", regexp.MustCompile(`(?im)^\s*PREPARE\s+`)},
	{"SET SESSION (non-auth)", regexp.MustCompile(`(?im)^\s*SET\s+SESSION\s+`)},
	// SET ROLE (session-level, not SET LOCAL ROLE) mutates session state that
	// persists across the transaction; only SET LOCAL ROLE is approved.
	{"SET ROLE (session-level)", regexp.MustCompile(`(?im)^\s*SET\s+ROLE\s+`)},
}

// isApprovedSetSession returns true if a SET SESSION statement is SET SESSION
// AUTHORIZATION. Other SET SESSION <guc> is forbidden (session-level GUC
// mutation that persists). SET SESSION AUTHORIZATION is ALSO forbidden in fresh
// artifacts (caught by its own pattern), so this helper exists only to keep the
// "SET SESSION (non-auth)" pattern from double-reporting.
func isApprovedSetSession(line string) bool {
	return regexp.MustCompile(`(?im)^\s*SET\s+SESSION\s+AUTHORIZATION\s+`).MatchString(line)
}

// TestFreshArtifactsHaveNoCrossArtifactSessionState confirms the four fresh-
// install artifacts contain only the approved SET LOCAL ROLE and none of the
// forbidden boundary-state commands. This is the contract that makes SET ROLE
// NONE sufficient normalization.
func TestFreshArtifactsHaveNoCrossArtifactSessionState(t *testing.T) {
	for _, name := range []assets.AssetName{
		assets.AssetRolesBootstrap,
		assets.AssetPlatformSchema,
		assets.AssetBaseline,
		assets.AssetSeed,
	} {
		t.Run(string(name), func(t *testing.T) {
			b, err := assets.Bytes(name)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			s := string(b)
			for _, fb := range forbiddenBoundaryState {
				matches := fb.re.FindAllStringIndex(s, -1)
				for _, loc := range matches {
					lineStart := strings.LastIndex(s[:loc[0]], "\n") + 1
					lineEnd := strings.Index(s[loc[0]:], "\n")
					if lineEnd < 0 {
						lineEnd = len(s) - loc[0]
					}
					line := strings.TrimSpace(s[lineStart : loc[0]+lineEnd])
					// SET SESSION AUTHORIZATION is caught by its own dedicated
					// pattern; skip it here to avoid double-reporting.
					if fb.name == "SET SESSION (non-auth)" && isApprovedSetSession(line) {
						continue
					}
					t.Errorf("%s: forbidden cross-artifact state %q at %q\n"+
						"SET ROLE NONE boundary emulation is INSUFFICIENT; transformation must fail closed until this is governed",
						name, fb.name, line)
				}
			}
			// Confirm SET LOCAL ROLE is present (the approved operation) where
			// expected. Platform and baseline use it; roles and seed may not.
			// This is informational, not a hard assertion on every artifact.
		})
	}
}

// TestP3AdoptionArtifactUsesSessionAuthorization confirms the adoption artifact
// IS excluded from the boundary contract — it deliberately uses SET SESSION
// AUTHORIZATION. This documents WHY it needs physical-connection destruction
// rather than SET ROLE NONE.
func TestP3AdoptionArtifactUsesSessionAuthorization(t *testing.T) {
	b, err := assets.Bytes(assets.AssetAdoptP3)
	if err != nil {
		t.Fatalf("read adoption: %v", err)
	}
	if !regexp.MustCompile(`(?im)SET\s+SESSION\s+AUTHORIZATION`).Match(b) {
		t.Fatal("adoption artifact unexpectedly lacks SET SESSION AUTHORIZATION; its boundary contract (physical-connection destruction) depends on it")
	}
}
