package migration

import "context"

// executor.go — the preflight/execution boundary. The Executor interface is the
// single seam between classification (read-only preflight) and any target-schema
// mutation. A preflight rejection must never cross it.
//
// The executor spy (SpyExecutor) records invocations so the zero-mutation matrix
// can assert executor invocation count == 0 for every rejection — the primary
// proof that the application never ATTEMPTED execution, complementing the
// PostgreSQL-level snapshots which prove no DDL was committed.

// Executor is the target-mutation boundary. Apply is the only legitimate caller.
// A blocked preflight must never reach it.
type Executor interface {
	// Execute runs the frozen transformed artifact chain on the given transaction
	// for the classified path. It is the ONLY path that may mutate the target
	// schema.
	Execute(ctx context.Context, pre PreflightResult) error
}

// SpyExecutor wraps a real Executor and records every invocation. Tests use it
// to assert that a rejection never crossed into execution. InvocationCount==0 is
// the primary no-attempt proof.
type SpyExecutor struct {
	Inner           Executor
	InvocationCount int
	LastPreflight   *PreflightResult
}

// Execute records the invocation and delegates to the inner executor (if any).
func (s *SpyExecutor) Execute(ctx context.Context, pre PreflightResult) error {
	s.InvocationCount++
	p := pre
	s.LastPreflight = &p
	if s.Inner == nil {
		return nil
	}
	return s.Inner.Execute(ctx, pre)
}

// NoExecutor is an Executor whose Execute always returns an error. Used as the
// spy's inner when tests want to prove the spy was never called (any call would
// surface as an error). The default nil-inner spy is sufficient for count checks.
type NoExecutor struct{}

func (NoExecutor) Execute(ctx context.Context, pre PreflightResult) error {
	return ErrExecutorShouldNotBeReached
}

// ErrExecutorShouldNotBeReached is returned by NoExecutor — it should never be
// observed because a blocked preflight must never invoke the executor.
var ErrExecutorShouldNotBeReached = newExecutorErr("executor reached on a blocked preflight (this is a classification bug)")

type executorErr struct{ msg string }

func (e *executorErr) Error() string { return e.msg }

func newExecutorErr(msg string) error { return &executorErr{msg: msg} }
