package migration

// provenance.go — build-bound provenance identities. The producing commit is
// embedded at build time via -ldflags "-X ...ProducingCommit=<40-hex-SHA>"; it
// is NOT accepted from CLI input. The production CLI must not expose
// --producing-commit. Tests inject via ProducingCommitForTest (test-only).
//
// Authoritative database identities (session_user, current_user) are read from
// the connection. A caller-supplied actor label (OS user, CI actor, operator)
// may be recorded separately but is labeled as caller-supplied, not authoritative.

import (
	"errors"
	"regexp"
)

// ProducingCommit is the build-bound Git commit SHA (40 lowercase hex). It is
// set via -ldflags at build time. The zero value ("") means "not built with
// provenance" — Apply rejects this in production wiring (tests override via
// ProducingCommitForTest).
var ProducingCommit = ""

// ProducingCommitForTest is the test-only injection point. When non-empty, it
// overrides ProducingCommit for tests. Production never sets this.
var ProducingCommitForTest = ""

// ErrProvenanceInvalid is returned when the producing commit is empty,
// malformed, all-zero, or otherwise untrustworthy.
var ErrProvenanceInvalid = errors.New("producing commit is invalid (must be a 40-char lowercase hex SHA, not all-zero)")

var commitSHARe = regexp.MustCompile(`^[0-9a-f]{40}$`)

const allZeroCommit = "0000000000000000000000000000000000000000"

// ResolveProducingCommit returns the effective producing commit (test injection
// takes precedence, then the build-bound value). Returns ErrProvenanceInvalid
// if the resolved value is empty, malformed, or all-zero.
func ResolveProducingCommit() (string, error) {
	c := ProducingCommit
	if ProducingCommitForTest != "" {
		c = ProducingCommitForTest
	}
	if err := ValidateProducingCommit(c); err != nil {
		return "", err
	}
	return c, nil
}

// ValidateProducingCommit rejects empty, malformed, or all-zero commits.
func ValidateProducingCommit(c string) error {
	if c == "" {
		return ErrProvenanceInvalid
	}
	if c == allZeroCommit {
		return ErrProvenanceInvalid
	}
	if !commitSHARe.MatchString(c) {
		return ErrProvenanceInvalid
	}
	return nil
}
