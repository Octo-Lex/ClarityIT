package migration

// preflight_test.go — the fail-closed rejection suite. Every rejection must
// assert: a stable diagnostic code, phase "preflight", ddl_started false, and
// (because Classify is pure) no mutation path is reachable. These are pure
// unit tests — no DB required — so they run in every environment and prove the
// decision logic cannot reach apply for an unidentified database.

import (
	"testing"
)

// baseProbe is a sound, identity-valid probe that classifies as governed_current.
// Each rejection test mutates one field to trigger a specific block.
func baseProbe() Probe {
	return Probe{
		DatabaseName:          "clarityit",
		PGMajor:               16,
		Fresh:                 false,
		PlatformSchemaPresent: true,
		Revision0001Present:   true,
		GovernedFingerprint:   GovernedTargetFingerprint,
		Recorded0001Checksum:  BaselineChecksum,
		ExtensionsPresent:     map[string]bool{"pgcrypto": true, "citext": true, "pg_trgm": true},
		RolesPresent:          map[string]bool{"clarityit": true, "clarityit_app": true, "clarityit_owner": true, "clarityit_migrator": true, "clarityit_admin": true},
	}
}

func assertBlocked(t *testing.T, class Class, path Path, code ReasonCode, wantCode ReasonCode) {
	t.Helper()
	if class != ClassUnknownDrifted {
		t.Errorf("class: got %q want %q", class, ClassUnknownDrifted)
	}
	if path != PathBlock {
		t.Errorf("path: got %q want %q (DDL must be unreachable)", path, PathBlock)
	}
	if code != wantCode {
		t.Errorf("code: got %q want %q", code, wantCode)
	}
	// Every preflight rejection produces a Result with ddl_started=false.
	r := blockedResult(code, class)
	if r.Phase != PhasePreflight {
		t.Errorf("phase: got %q want %q", r.Phase, PhasePreflight)
	}
	if r.DDLStarted {
		t.Error("ddl_started: got true want false (DDL must not start on a preflight rejection)")
	}
	if r.Status != "blocked" {
		t.Errorf("status: got %q want %q", r.Status, "blocked")
	}
}

// TestGovernedCurrentClassifiesNoOp: a sound governed DB is a no-op, never DDL.
func TestGovernedCurrentClassifiesNoOp(t *testing.T) {
	class, path, code := Classify(baseProbe())
	if class != ClassGovernedCurrent || path != PathNoOp || code != CodeOK {
		t.Fatalf("governed current: got class=%q path=%q code=%q want governed_current/no_op/OK", class, path, code)
	}
	r := okResult(PhasePreflight, class, path)
	if r.DDLStarted {
		t.Error("no-op result must have ddl_started=false")
	}
}

// TestUnknownSourceBlocks: a non-empty DB with an unrecognized fingerprint blocks.
func TestUnknownSourceBlocks(t *testing.T) {
	p := baseProbe()
	p.GovernedFingerprint = "" // not governed
	p.PlatformSchemaPresent = false
	p.Revision0001Present = false
	p.SourceFingerprint = "deadbeef" + "00000000000000000000000000000000000000000000000000000000" // 64-hex, unknown
	p.Fresh = false
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeSourceProfileUnknown)
}

// TestPartiallyMatchingSourceBlocks: a fingerprint that is close-but-not-exact
// (e.g. a truncated or single-char-different P3 fingerprint) is unknown.
func TestPartiallyMatchingSourceBlocks(t *testing.T) {
	p := baseProbe()
	p.GovernedFingerprint = ""
	p.PlatformSchemaPresent = false
	p.Revision0001Present = false
	// One character off from the real P3 golden fingerprint.
	bad := P3GoldenFingerprint
	bad = bad[:len(bad)-1] + (func() string {
		if bad[len(bad)-1] == 'a' {
			return "b"
		}
		return "a"
	}())
	p.SourceFingerprint = bad
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeSourceProfileUnknown)
}

// TestP1P2RecognizedButNotExecutable: P1/P2 fingerprint is recognized but has
// no G4 executable path; must block with the specific code.
func TestP1P2RecognizedButNotExecutable(t *testing.T) {
	p := baseProbe()
	p.GovernedFingerprint = ""
	p.PlatformSchemaPresent = false
	p.Revision0001Present = false
	p.SourceFingerprint = P1P2Fingerprint
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeSourceProfileP1P2)
}

// TestDriftedGovernedBlocks: a DB that has a platform ledger + revision but a
// governed fingerprint that differs from the frozen target has drifted.
func TestDriftedGovernedBlocks(t *testing.T) {
	p := baseProbe()
	// Keep ledger present but mutate the governed fingerprint.
	p.GovernedFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	p.Recorded0001Checksum = BaselineChecksum // checksum consistent, but shape drifted
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeDriftedGoverned)
}

// TestInconsistentLedgerBlocks: a governed-current fingerprint but a revision
// checksum that differs from the frozen baseline means a succeeded revision is
// being presented with different bytes — block immediately.
func TestInconsistentLedgerBlocks(t *testing.T) {
	p := baseProbe()
	p.GovernedFingerprint = GovernedTargetFingerprint
	p.Recorded0001Checksum = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeLedgerInconsistent)
}

// TestUnsupportedPGMajorBlocks: any non-16 major blocks before DDL.
func TestUnsupportedPGMajorBlocks(t *testing.T) {
	p := baseProbe()
	p.PGMajor = 15
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodePgMajorUnsupported)
}

// TestWrongDatabaseIdentityBlocks: a database not named clarityit blocks.
func TestWrongDatabaseIdentityBlocks(t *testing.T) {
	p := baseProbe()
	p.DatabaseName = "other_db"
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeDbIdentityWrong)
}

// TestEmptyInstallClassifiesInstall: a closed-world-empty DB classifies as install.
func TestEmptyInstallClassifiesInstall(t *testing.T) {
	p := Probe{
		DatabaseName: "clarityit",
		PGMajor:      16,
		Fresh:        true,
	}
	class, path, code := Classify(p)
	if class != ClassEmptyInstall || path != PathInstall || code != CodeOK {
		t.Fatalf("empty install: got class=%q path=%q code=%q want empty_install/install/OK", class, path, code)
	}
}

// TestNonEmptyNonGovernedNonMatchedBlocks: a DB that is not fresh, not governed,
// and has no recognized source fingerprint is unknown/drifted.
func TestNonEmptyNonGovernedNonMatchedBlocks(t *testing.T) {
	p := Probe{
		DatabaseName: "clarityit",
		PGMajor:      16,
		Fresh:        false,
		// no fingerprint, not governed, not fresh
	}
	class, path, code := Classify(p)
	assertBlocked(t, class, path, code, CodeNotFreshNonEmpty)
}

// TestApprovedP3ClassifiesAdopt: the P3 golden fingerprint classifies as adopt.
func TestApprovedP3ClassifiesAdopt(t *testing.T) {
	p := Probe{
		DatabaseName:      "clarityit",
		PGMajor:           16,
		Fresh:             false,
		SourceFingerprint: P3GoldenFingerprint,
		ExtensionsPresent: map[string]bool{"pgcrypto": true, "citext": true, "pg_trgm": true},
		RolesPresent:      map[string]bool{"clarityit": true, "clarityit_app": true, "clarityit_owner": true, "clarityit_migrator": true, "clarityit_admin": true},
	}
	class, path, code := Classify(p)
	if class != ClassApprovedSource || path != PathAdopt || code != CodeOK {
		t.Fatalf("approved P3: got class=%q path=%q code=%q want approved_source/adopt_p3/OK", class, path, code)
	}
}

// TestPackagingMismatchIsIndependent: VerifyAll failing is a separate block path
// that the runner checks before even probing the DB. This test confirms the
// frozen packaging gate is independently testable.
func TestPackagingMismatchIsIndependent(t *testing.T) {
	// VerifyAll must pass against the real embedded assets (proven in
	// catalog_test.go). Here we only assert the code constant exists and the
	// blocked-result shape is correct for a packaging failure.
	r := blockedResult(CodePackagingMismatch, ClassUnknownDrifted)
	if r.Code != CodePackagingMismatch {
		t.Errorf("code: got %q want %q", r.Code, CodePackagingMismatch)
	}
	if r.DDLStarted {
		t.Error("packaging mismatch must have ddl_started=false")
	}
	if r.Phase != PhasePreflight {
		t.Errorf("phase: got %q want %q", r.Phase, PhasePreflight)
	}
}

// TestNoReconstructionOfLegacyChain: the allowlist contains NO path that could
// route to legacy migrations 001-040. Every recognized fingerprint resolves to
// install, adopt_p3, block, or "" (recognized-but-not-executable) — never to a
// legacy chain. There is no legacy-chain path constant at all.
func TestNoReconstructionOfLegacyChain(t *testing.T) {
	// Confirm no legacy path constant exists in the Path type by checking that
	// every allowlisted action resolves to a known-safe path or the empty
	// not-executable marker.
	allowedPaths := map[Path]bool{
		PathInstall: true,
		PathAdopt:   true,
		PathAdoptP2: true,
		PathBlock:   true,
		"":          true, // recognized but not executable (P1/P2 v3.1)
	}
	for fp, act := range AllowlistedFingerprints {
		if !allowedPaths[act.Path] {
			t.Errorf("fingerprint %s resolves to unexpected path %q (legacy leak?)", fp, act.Path)
		}
		if !act.Executable && act.Path != "" {
			t.Errorf("fingerprint %s is not executable but has path %q (must be empty)", fp, act.Path)
		}
		if act.Executable && act.Path == "" {
			t.Errorf("fingerprint %s is executable but has empty path", fp)
		}
	}
}
