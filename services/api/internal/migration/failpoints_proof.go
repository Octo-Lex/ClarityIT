//go:build proof

// Package migration — proof-build failpoint controller.
// This file is compiled ONLY when -tags proof is set. The production binary
// (ordinary `go build`) does NOT include MapFailpoint or any activatable fault-
// injection mechanism. The ActiveFailpointController in production is always
// InertFailpoints.

package migration

import "context"

// MapFailpoint is a test/proof controller that injects errors at specific
// failpoints via a map. It is ONLY available in the proof build (-tags proof).
// The production binary has no way to construct or activate it.
type MapFailpoint struct {
	Errors map[Failpoint]error
	Repeat bool
}

func (m *MapFailpoint) Hit(_ context.Context, fp Failpoint) error {
	if m.Errors == nil {
		return nil
	}
	if err, ok := m.Errors[fp]; ok {
		if !m.Repeat {
			delete(m.Errors, fp)
		}
		return err
	}
	return nil
}
