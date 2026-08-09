package migration

// diagnostics.go — the stable result document the runner emits on stdout. Fields
// are allowlisted; the runner never emits DSNs, connection strings, or raw SQL
// errors here. Human logs go to stderr.

import (
	"encoding/json"
	"io"
)

// Phase labels the stage where a decision was made.
type Phase string

const (
	PhasePreflight Phase = "preflight"
	PhaseApply     Phase = "apply"
	PhaseVerify    Phase = "verify"
)

// Result is the stable, machine-readable result document. It is the ONLY thing
// the runner writes to stdout. Unknown/problematic fields are never added; the
// schema is intentionally small and allowlisted.
type Result struct {
	Status          string      `json:"status"`            // "ok" | "blocked" | "no_op" | "applied" | "verified"
	Code            ReasonCode  `json:"code"`              // stable reason code (OK when status is ok/*)
	Phase           Phase       `json:"phase"`             // preflight | apply | verify
	DDLStarted      bool        `json:"ddl_started"`       // false for every preflight rejection
	Class           Class       `json:"class,omitempty"`   // classification (when known)
	Path            Path        `json:"path,omitempty"`    // resolved path (when known)
	SourceProfile   string      `json:"source_profile,omitempty"`    // fingerprint (allowlisted, not a secret)
	GovernedFP      string      `json:"governed_fingerprint,omitempty"` // computed, never a secret
	Composite       string      `json:"composite,omitempty"`          // recomputed packaging digest
	RunID           string      `json:"run_id,omitempty"`             // migration_runs.run_id (after apply)
	TargetVersion   string      `json:"target_version,omitempty"`     // e.g. "0001"
	DurationMs      int64       `json:"duration_ms,omitempty"`
	Diagnostics     []Diag      `json:"diagnostics,omitempty"`        // per-check detail
}

// Diag is one check's detail within a result (e.g. a reconciliation_result row
// projection). It carries only allowlisted, sanitized fields.
type Diag struct {
	CheckID  string `json:"check_id"`
	Scope    string `json:"scope,omitempty"`
	Result   string `json:"result,omitempty"` // pass | fail | blocked
	Detail   string `json:"detail,omitempty"` // sanitized summary, never raw SQL
}

// Emit writes the result document as JSON to w (one line, no trailing newline
// beyond what JSON produces). Used by the CLI to emit the stable stdout doc.
func Emit(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

// okResult builds a successful result for a given phase.
func okResult(phase Phase, class Class, path Path) Result {
	return Result{Status: "ok", Code: CodeOK, Phase: phase, DDLStarted: false, Class: class, Path: path}
}

// blockedResult builds a preflight-blocked result. ddl_started is always false.
func blockedResult(code ReasonCode, class Class, diags ...Diag) Result {
	return Result{
		Status:      "blocked",
		Code:        code,
		Phase:       PhasePreflight,
		DDLStarted:  false,
		Class:       class,
		Path:        PathBlock,
		Diagnostics: diags,
	}
}
