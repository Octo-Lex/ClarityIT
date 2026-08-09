//go:build proof

// Package migration — proof-build failpoint controller. This file is compiled
// ONLY when -tags proof is set. The production binary does NOT include any
// controller interface, mutable global, or injection dispatch — only the inert
// hitFailpoint from failpoints_prod.go.

package migration

import (
	"context"
)

// FailpointController is the proof-only injection interface. It does not exist
// in the production binary.
type FailpointController interface {
	Hit(ctx context.Context, fp Failpoint) error
}

// InertFailpoints is the default proof controller (matches production behavior).
type InertFailpoints struct{}

func (InertFailpoints) Hit(context.Context, Failpoint) error { return nil }

// ActiveFailpointController is the proof-only mutable controller. Tests set
// this to inject faults. It does not exist in the production binary.
var ActiveFailpointController FailpointController = InertFailpoints{}

// hitFailpoint dispatches to the active proof controller. In production this
// symbol is the inert version from failpoints_prod.go.
func hitFailpoint(ctx context.Context, fp Failpoint) error {
	return ActiveFailpointController.Hit(ctx, fp)
}

// MapFailpoint is a test/proof controller that injects errors at specific
// failpoints via a map. Only available with -tags proof.
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
