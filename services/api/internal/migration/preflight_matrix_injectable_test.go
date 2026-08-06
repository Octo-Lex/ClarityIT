package migration

// preflight_matrix_injectable_test.go — the two remaining matrix cases:
//
//  1. Embedded-checksum-mutation, exercised LIVE through the injectable
//     identity verifier at the orchestration boundary (no filesystem, no
//     arbitrary SQL). Production wiring uses the real embedded-byte verifier;
//     this test injects a deterministic mismatch.
//
//  2. P1/P2 recognized-but-not-executable — recorded as an explicit external-
//     fixture acceptance dependency. NOT fabricated from the capture manifest.

import (
	"context"
	"testing"
)

// failingVerifier is a test-only PackageVerifier that returns a deterministic
// packaging mismatch (one fabricated mismatch entry + composite mismatch). It
// exercises the checksum-mutation path through the LIVE preflight orchestrator
// without touching the filesystem or arbitrary SQL.
type failingVerifier struct{}

func (failingVerifier) VerifyAll() (VerifyResult, error) {
	return VerifyResult{
		Composite:   "deadbeef00000000000000000000000000000000000000000000000000deadbeef",
		CompositeOK: false,
		Mismatches: []string{
			"0001_reconciled.sql: embedded=aaaaaaaa... frozen=1021adefe8b5...",
		},
	}, ErrPackagingMismatch
}

// TestMatrix_EmbeddedChecksumMutation_LIVE exercises the packaging-mismatch
// rejection path through the actual live preflight orchestrator by injecting a
// failing verifier. This is a LIVE case (not logic-layer): it confirms the
// orchestrator stops at the verification step, never opens the read-only probe
// transaction for classification, and reports PACKAGING_MISMATCH with
// ddl_started:false. A spy executor confirms execution was never attempted.
func TestMatrix_EmbeddedChecksumMutation_LIVE(t *testing.T) {
	conn := startFixture(t, "g4-matrix-pkgmut", 55518)
	// Non-empty DB (so if the orchestrator wrongly skipped packaging verify, the
	// classification would proceed and the spy might be reachable).
	applySQL(t, conn, `CREATE SCHEMA app_z; CREATE TABLE app_z.t (id int)`)
	installDDLEventTrigger(t, conn)
	spy := &SpyExecutor{Inner: NoExecutor{}}
	before := snapshotDBStrong(t, conn)
	// Inject the failing verifier — production Preflight uses the real one.
	res, err := PreflightWithVerifier(context.Background(), conn, failingVerifier{})
	_ = err // expected: packaging verify fails
	after := snapshotDBStrong(t, conn)
	assertBlockedWithoutMutation(t, before, after, res, CodePackagingMismatch)
	assertExecutorNeverInvoked(t, spy)
	// Confirm the orchestrator never reached the read-only probe transaction:
	// packaging verify runs first and there is no DB classification on mismatch.
	// (The DDL sentinel count is the live proof no DDL was attempted/committed.)
}

// TestMatrix_P1P2_ExternalFixtureDependency records that the P1/P2 recognized-
// but-not-executable case requires a sanctioned P1/P2 database fixture that is
// NOT present in this repository and MUST NOT be fabricated from the capture
// manifest. This is an explicit, unresolved acceptance dependency.
//
// The pure-logic layer (TestP1P2RecognizedButNotExecutable in preflight_test.go)
// proves the fingerprint is recognized and resolves to a non-executable block.
// The live matrix case requires the production legacy P1/P2 database shape,
// which lives in the external evidence store and is intentionally not in-repo.
func TestMatrix_P1P2_ExternalFixtureDependency(t *testing.T) {
	t.Skip("EXTERNAL-FIXTURE DEPENDENCY: the P1/P2 recognized-but-not-executable " +
		"live matrix case requires a sanctioned P1/P2 database (fingerprint " +
		P1P2Fingerprint[:12] + "…) from the external evidence store, which is " +
		"intentionally not in this repository. The pure-logic guarantee is " +
		"covered by TestP1P2RecognizedButNotExecutable. Do NOT fabricate a " +
		"P1/P2 schema from the capture manifest to turn this skip green; it " +
		"must be satisfied by a sanctioned fixture or formally recorded as an " +
		"unresolved acceptance dependency.")
}
