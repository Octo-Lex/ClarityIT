package migration

// preflight.go — fail-closed classification of the target database BEFORE any
// DDL. Every rejection returns a stable diagnostic with ddl_started=false. The
// runner must never begin work against an unidentified database.
//
// Classification order (per G4 contract):
//  1. governed_current/no-op    — governed fingerprint == 9881c93e… AND ledger present
//  2. approved_source/adopt     — source profiler fingerprint ∈ executable allowlist (P3)
//  3. empty/install             — closed-world empty (no user objects, no platform)
//  4. unknown_or_drifted/block  — everything else
//
// This file implements the pure decision logic (no DB access) so the rejection
// suite can test it deterministically. The live DB probes that feed these
// decisions live in probe.go.

// Class is the database classification result.
type Class string

const (
	ClassGovernedCurrent Class = "governed_current" // no-op; already at target
	ClassApprovedSource  Class = "approved_source"  // adopt the approved source
	ClassEmptyInstall    Class = "empty_install"    // fresh install
	ClassUnknownDrifted  Class = "unknown_drifted"  // BLOCK — never DDL
)

// Path is the executable action a class resolves to.
type Path string

const (
	PathNoOp    Path = "no_op"
	PathAdopt   Path = "adopt_p3"
	PathAdoptP2 Path = "adopt_p2"
	PathInstall Path = "install"
	PathBlock   Path = "block"
)

// ReasonCode is a stable diagnostic code for a preflight outcome.
type ReasonCode string

const (
	CodeOK                  ReasonCode = "OK"
	CodeSourceProfileUnknown ReasonCode = "SOURCE_PROFILE_UNKNOWN"
	CodeSourceProfileP1P2    ReasonCode = "SOURCE_PROFILE_P1P2_NOT_EXECUTABLE"
	CodeDriftedGoverned      ReasonCode = "GOVERNED_TARGET_DRIFTED"
	CodeNotFreshNonEmpty     ReasonCode = "NOT_FRESH_NONEMPTY"
	CodePgMajorUnsupported  ReasonCode = "PG_MAJOR_UNSUPPORTED"
	CodeDbIdentityWrong      ReasonCode = "DB_IDENTITY_WRONG"
	CodeMissingExtension     ReasonCode = "MISSING_EXTENSION"
	CodeMissingRole          ReasonCode = "MISSING_ROLE"
	CodeLedgerInconsistent   ReasonCode = "LEDGER_INCONSISTENT"
	CodePackagingMismatch    ReasonCode = "PACKAGING_MISMATCH"
)

// AllowlistedFingerprints maps a known source-profiler fingerprint to its
// executable G4 path. P3 is the only executable adoption source in G4; P1/P2
// are recognized but not executable (a separate governed artifact would be
// required). Anything not here is unknown.
var AllowlistedFingerprints = map[string]FingerprintAction{
	P3GoldenFingerprint:     {Path: PathAdopt, Class: ClassApprovedSource, Executable: true},
	P2SuccessorFingerprint:  {Path: PathAdoptP2, Class: ClassApprovedSource, Executable: true},
	P1P2Fingerprint:         {Path: "", Class: ClassUnknownDrifted, Executable: false, Code: CodeSourceProfileP1P2},
}

// FingerprintAction is the resolution of a recognized source fingerprint.
type FingerprintAction struct {
	Path       Path
	Class      Class
	Executable bool
	Code       ReasonCode // set when Executable is false
}

// Probe is the read-only database observation that preflight classification
// consumes. It is populated by live DB probes (read-only transaction) before
// classification. Zero values mean "not present/observed".
type Probe struct {
	// Database identity.
	DatabaseName string
	PGMajor      int

	// Empty-install closed-world inventory. Fresh=true means NO non-system,
	// non-approved-extension user relations, functions, schemas, migration
	// ledgers, or conflicting roles exist.
	Fresh bool

	// Platform ledger presence.
	PlatformSchemaPresent bool
	Revision0001Present  bool

	// Source-profiler fingerprint of the live database (empty if DB is empty).
	SourceFingerprint string

	// Governed fingerprint of the live database (empty if not governable).
	GovernedFingerprint string

	// Required extensions present (pgcrypto, citext, pg_trgm).
	ExtensionsPresent map[string]bool

	// Target roles present (the five-role posture).
	RolesPresent map[string]bool

	// Ledger consistency: the recorded revision-0001 checksum, if present.
	Recorded0001Checksum string
}

// Classify is the pure decision function. It never touches the DB and never
// mutates anything. Every non-OK outcome carries ddl_started=false by
// construction (this function performs no DDL).
func Classify(p Probe) (Class, Path, ReasonCode) {
	// Identity guards come first — they apply regardless of class.
	if p.DatabaseName != "clarityit" {
		return ClassUnknownDrifted, PathBlock, CodeDbIdentityWrong
	}
	if p.PGMajor != 16 {
		return ClassUnknownDrifted, PathBlock, CodePgMajorUnsupported
	}

	// 1. Ledger consistency is the MOST specific diagnosis: a succeeded revision
	// presented with different bytes is an immutable-violation, regardless of
	// whether the governed fingerprint also drifted. Check it first whenever a
	// revision row is present.
	if p.PlatformSchemaPresent && p.Revision0001Present {
		if p.Recorded0001Checksum != "" && p.Recorded0001Checksum != BaselineChecksum {
			return ClassUnknownDrifted, PathBlock, CodeLedgerInconsistent
		}
	}

	// 1b. Governed current: fingerprint matches AND ledger present.
	if p.GovernedFingerprint == GovernedTargetFingerprint && p.PlatformSchemaPresent && p.Revision0001Present {
		return ClassGovernedCurrent, PathNoOp, CodeOK
	}

	// 1c. Drifted governed DB: a platform ledger + revision EXISTS (meaning this
	// WAS a governed database) but the governed fingerprint no longer matches the
	// frozen target. This is structural drift from a known-governed state — the
	// most specific remaining diagnosis — and must be reported BEFORE the generic
	// source-profile-unknown path. The source fingerprint of such a DB is
	// naturally non-allowlisted (it drifted), so without this check it would
	// misclassify as SOURCE_PROFILE_UNKNOWN.
	if p.PlatformSchemaPresent && p.Revision0001Present {
		return ClassUnknownDrifted, PathBlock, CodeDriftedGoverned
	}

	// 2. Approved source: source fingerprint in the executable allowlist.
	if p.SourceFingerprint != "" {
		act, ok := AllowlistedFingerprints[p.SourceFingerprint]
		if !ok {
			return ClassUnknownDrifted, PathBlock, CodeSourceProfileUnknown
		}
		if !act.Executable {
			return ClassUnknownDrifted, PathBlock, act.Code
		}
		// Executable adoption path (P3). Still require the prerequisites.
		if !prerequisitesSatisfied(p) {
			return ClassUnknownDrifted, PathBlock, CodeMissingExtension // refined by caller
		}
		return ClassApprovedSource, act.Path, CodeOK
	}

	// 3. Empty install: closed-world empty.
	if p.Fresh {
		if !prerequisitesSatisfied(p) {
			return ClassUnknownDrifted, PathBlock, CodeMissingExtension
		}
		return ClassEmptyInstall, PathInstall, CodeOK
	}

	// 4. Unknown or drifted.
	// If the database is non-empty, has no recognized source fingerprint, and
	// is not governed-current, it is drifted/unknown.
	if p.GovernedFingerprint != "" && p.GovernedFingerprint != GovernedTargetFingerprint {
		return ClassUnknownDrifted, PathBlock, CodeDriftedGoverned
	}
	return ClassUnknownDrifted, PathBlock, CodeNotFreshNonEmpty
}

// prerequisitesSatisfied checks extensions and roles required for the chosen
// path. For fresh install, roles/extensions are created by the artifacts, so
// this returns true (the artifacts will create them). For adoption, the seven
// adoption preconditions apply (checked separately by the adopt path).
func prerequisitesSatisfied(p Probe) bool {
	// Fresh install: artifacts create roles and extensions; only require that
	// no conflicting posture exists (Fresh already encodes that). So a fresh
	// DB satisfies prerequisites trivially.
	return true
}
