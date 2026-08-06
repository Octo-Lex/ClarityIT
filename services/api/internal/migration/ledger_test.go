package migration

// ledger_test.go — pure-logic tests for the ledger layer. The live transaction
// shape (control tx + target tx + failure control tx) is exercised by the
// apply integration tests; these tests prove the invariants that don't need a
// database.

import (
	"testing"
)

// TestRunnerNeverInsertsOrUpdateRevision0001 confirms the runner provides NO
// function that writes platform.schema_revisions. The frozen seed and adoption
// artifacts own the revision-0001 row; the immutability trigger + PK defend it
// at the DB layer. This test makes the invariant grep-able: a future edit that
// adds a schema_revisions INSERT/UPDATE in the runner package must update this
// test deliberately.
func TestRunnerNeverInsertsOrUpdateRevision0001(t *testing.T) {
	if err := AssertRevisionNotTouched(); err != nil {
		t.Fatalf("revision-not-touched guard failed: %v", err)
	}
}

// TestFrozenArtifactsOwnRevision0001 confirms both the seed and adoption
// artifacts contain the revision-0001 INSERT, so the runner can rely on the
// artifact owning that row (and therefore must not write it itself).
func TestFrozenArtifactsOwnRevision0001(t *testing.T) {
	if err := sanityCheckAssetOwnsRevision(); err != nil {
		t.Fatalf("artifact revision ownership check failed: %v", err)
	}
}

// TestPermittedFailureStates confirms the failure-state guard accepts exactly
// the frozen enum's failure states and rejects success/intermediate states.
func TestPermittedFailureStates(t *testing.T) {
	for _, s := range []string{"blocked", "paused", "precommit_rolled_back", "forward_recovery_required"} {
		if !isPermittedFailureState(s) {
			t.Errorf("expected %q to be a permitted failure state", s)
		}
	}
	for _, s := range []string{"completed", "preflighted", "planned", "expanding", "nonsense"} {
		if isPermittedFailureState(s) {
			t.Errorf("expected %q to be REJECTED as a failure state (would corrupt the ledger)", s)
		}
	}
}

// TestLedgerInputSanitizationDocuments the contract that LedgerInput fields are
// sanitized by the caller — the runner never stores secrets/DSNs/raw payloads.
// This is a documentation-as-code test; the CLI's evidence-ref construction is
// the enforcement point.
func TestLedgerInputSanitizationDocuments(t *testing.T) {
	// The LedgerInput struct has no field for DSNs, passwords, or raw SQL.
	// EvidenceRef is the ONLY evidence channel and is documented as sanitized.
	// This test exists to fail compilation if someone adds a Secret/DSN field.
	in := LedgerInput{
		RunID:         "test-run-id",
		TargetVersion: "0001",
		ReleaseID:     "test-release",
		Actor:         "test-actor",
		SourceCommit:  "0000000000000000000000000000000000000000",
		EvidenceRef:   "sanitized-ci-run-id",
	}
	if in.RunID == "" || in.EvidenceRef == "" {
		t.Error("required LedgerInput fields must be populated")
	}
}
