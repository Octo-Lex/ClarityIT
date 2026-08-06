package migration

// package.go — deterministic SQL transformation for the frozen G3 artifacts.
//
// The runner owns the transaction. Each frozen artifact's outer BEGIN/COMMIT
// must be removed so the runner can wrap the body in its own transaction
// (together with the advisory lock, ledger writes, and verification). The only
// psql-meta construct in the frozen artifacts is `\set ON_ERROR_STOP on` (one
// per file); the adoption artifact additionally has one psql-variable
// interpolation `:'g3_source_commit'` on a set_config line that the runner
// executes separately as a parameterized statement.
//
// Contract (per G4 authorization):
//   - Input ONLY from embedded frozen assets. Never the filesystem. Never env
//     or command substitution.
//   - REJECT unsupported psql meta-commands rather than silently stripping.
//   - An explicit allowlist of supported transformations; anything else errors.
//   - Deterministic output (same bytes in -> same bytes out).
//   - Original embedded digest retained alongside the transformed digest.
//   - No silent removal of unrecognized syntax.
//
// Allowed transformations (the only ones the frozen package contract requires):
//   1. remove exactly one `\set ON_ERROR_STOP on` line (verbatim);
//   2. remove exactly one outer `BEGIN;` (the first statement);
//   3. remove exactly one outer `COMMIT;` (the last statement);
//   4. for the adoption artifact only: remove the single
//      `SELECT set_config('g3.source_commit', :'g3_source_commit', true);` line
//      (the runner executes set_config separately, parameterized).
//
// Any other psql meta-command (`\i`, `\include`, `\copy`, `\set <var>`,
// `\!`, `\if`, `\connect`, etc.) is REJECTED with ErrUnsupportedPsqlMeta.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/clarityit/api/internal/migration/assets"
)

// ErrUnsupportedPsqlMeta is returned when the transformer encounters a psql
// meta-command that is not in the explicit allowlist.
var ErrUnsupportedPsqlMeta = errors.New("unsupported psql meta-command in frozen artifact")

// ErrTransformInvariant is returned when an artifact's psql-meta/BEGIN/COMMIT
// counts differ from the frozen package contract (e.g. a second BEGIN appeared).
var ErrTransformInvariant = errors.New("frozen artifact transform invariant violated")

// TransformedScript is the deterministic output of transforming a frozen
// artifact for runner-owned execution. Both digests are retained so the
// transformation is auditable: SourceSHA256 ties to the embedded/frozen
// identity; TransformedSHA256 ties to the executable bytes.
type TransformedScript struct {
	Asset            assets.AssetName
	SourceSHA256     string // SHA-256 of the immutable embedded bytes
	TransformedSHA256 string // SHA-256 of the transformed executable body
	Body             []byte // the parameter-free, BEGIN/COMMIT-stripped body
	// NeedsSetConfig is true for the adoption artifact: the runner must execute
	// SELECT set_config('g3.source_commit',$1,true) separately before the body.
	NeedsSetConfig bool
}

// psqlMetaLine matches any line whose first non-whitespace token is a psql
// backslash command (letter OR punctuation like `!`). Used to REJECT everything
// not explicitly allowed. `\set ON_ERROR_STOP on` is the sole allowed command.
var psqlMetaLine = regexp.MustCompile(`^\s*\\[A-Za-z!]`)

// allowedSetLine matches the single allowed \set line.
var allowedSetLine = regexp.MustCompile(`^\s*\\set\s+ON_ERROR_STOP\s+on\s*$`)

// setConfigLine matches the adoption artifact's runtime-bind line. The runner
// removes it and executes the equivalent parameterized statement separately.
var setConfigLine = regexp.MustCompile(`^\s*SELECT\s+set_config\('g3\.source_commit',\s*:'g3_source_commit',\s*true\);\s*$`)

// Transform produces the deterministic executable body for a frozen artifact.
// It reads only from the embedded bytes. Any deviation from the frozen package
// contract (unexpected meta-command, wrong BEGIN/COMMIT count, a second
// set_config line) is an error, never a silent strip.
func Transform(name assets.AssetName) (TransformedScript, error) {
	src, err := assets.Bytes(name)
	if err != nil {
		return TransformedScript{}, err
	}
	srcSum := sha256.Sum256(src)
	srcHex := hex.EncodeToString(srcSum[:])

	out := TransformedScript{Asset: name, SourceSHA256: srcHex}

	// Only the adoption artifact is permitted to have (and must have) the
	// set_config line. Every other artifact must NOT contain it.
	expectSetConfig := name == assets.AssetAdoptP3

	setCount := 0
	setConfigCount := 0

	var kept []string
	for _, raw := range splitLines(src) {
		// 1. psql meta-commands: allow only the exact `\set ON_ERROR_STOP on`.
		if psqlMetaLine.MatchString(raw) {
			if allowedSetLine.MatchString(raw) {
				setCount++
				continue // drop
			}
			return out, fmt.Errorf("%w: %q in %s", ErrUnsupportedPsqlMeta, strings.TrimSpace(raw), name)
		}
		// 2. set_config line (adoption only).
		if setConfigLine.MatchString(raw) {
			setConfigCount++
			if !expectSetConfig {
				return out, fmt.Errorf("%w: unexpected set_config line in %s", ErrTransformInvariant, name)
			}
			continue // drop; runner executes parameterized separately
		}
		kept = append(kept, raw)
	}

	// 3. Invariant checks on counts.
	if setCount != 1 {
		return out, fmt.Errorf("%w: %s has %d \\set ON_ERROR_STOP lines (want 1)", ErrTransformInvariant, name, setCount)
	}
	if expectSetConfig && setConfigCount != 1 {
		return out, fmt.Errorf("%w: %s has %d set_config lines (want 1)", ErrTransformInvariant, name, setConfigCount)
	}
	if !expectSetConfig && setConfigCount != 0 {
		return out, fmt.Errorf("%w: %s has unexpected set_config line", ErrTransformInvariant, name)
	}

	// 4. Strip exactly one outer BEGIN; and one outer COMMIT;. The frozen
	//    artifacts place BEGIN; after a header comment block (not necessarily
	//    the first line) and COMMIT; as the last statement. PL/pgSQL function
	//    bodies contain bare `BEGIN` (no semicolon) which must NOT be touched.
	//    Find the first standalone `BEGIN;` and the last standalone `COMMIT;`.
	beginIdx := -1
	commitIdx := -1
	for i, line := range kept {
		if beginIdx == -1 && isBeginLine(line) {
			beginIdx = i
		}
		if isCommitLine(line) {
			commitIdx = i // keep updating; last one wins
		}
	}
	if beginIdx == -1 {
		return out, fmt.Errorf("%w: %s has no outer BEGIN;", ErrTransformInvariant, name)
	}
	if commitIdx == -1 {
		return out, fmt.Errorf("%w: %s has no outer COMMIT;", ErrTransformInvariant, name)
	}
	if commitIdx <= beginIdx {
		return out, fmt.Errorf("%w: %s COMMIT; precedes BEGIN;", ErrTransformInvariant, name)
	}
	// Verify exactly one outer BEGIN; (the first). A second standalone BEGIN;
	// later would indicate a nested tx the runner doesn't own.
	beginCount := 0
	for _, line := range kept {
		if isBeginLine(line) {
			beginCount++
		}
	}
	if beginCount != 1 {
		return out, fmt.Errorf("%w: %s has %d outer BEGIN; (want 1)", ErrTransformInvariant, name, beginCount)
	}
	// Remove the BEGIN; and COMMIT; lines.
	stripped := append(append([]string{}, kept[:beginIdx]...), kept[beginIdx+1:commitIdx]...)
	stripped = append(stripped, kept[commitIdx+1:]...)

	// Re-join. Preserve original line endings (we split on \n; rejoin with \n).
	// sql_bytes canonicalizes with rstrip + single trailing newline. The frozen
	// files already end with exactly one newline; after stripping the final
	// COMMIT we re-add one so the body is a complete SQL text.
	bodyStr := strings.Join(stripped, "\n")
	bodyStr = strings.TrimRight(bodyStr, "\n") + "\n"
	out.Body = []byte(bodyStr)
	tSum := sha256.Sum256(out.Body)
	out.TransformedSHA256 = hex.EncodeToString(tSum[:])
	out.NeedsSetConfig = expectSetConfig
	return out, nil
}

// splitLines splits on \n without dropping empty lines (preserves structure).
func splitLines(b []byte) []string {
	s := string(b)
	// Normalize CRLF -> LF if any (frozen artifacts are LF, but be defensive).
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// trimBlankLines removes leading and trailing empty/whitespace-only lines.
func trimBlankLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}

func isBeginLine(s string) bool {
	return strings.TrimSpace(s) == "BEGIN;"
}

func isCommitLine(s string) bool {
	return strings.TrimSpace(s) == "COMMIT;"
}
