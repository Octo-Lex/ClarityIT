//go:build !proof

// Package migration — production failpoint implementation. The ordinary build
// compiles this file. It contains NO controller interface, NO mutable global,
// NO injection dispatch. The hitFailpoint function is inert (always returns nil)
// and the compiler can inline it as a no-op.

package migration

import "context"

// hitFailpoint is the inert production implementation. It always returns nil.
// There is no controller to replace, no global to assign, and no way to activate
// fault injection in the production binary.
func hitFailpoint(_ context.Context, _ Failpoint) error { return nil }
