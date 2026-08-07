package migration

// failpoints.go — typed failpoint injection for the apply executor. Failpoints
// are a typed enum + an injectable controller interface. Production wiring
// uses an inert implementation (no fault injection). Tests inject a controller
// that returns errors at specific points.
//
// Failpoints are NEVER activated through CLI flags, configuration files, or
// environment variables. The production binary has zero fault-injection surface
// (the inert controller is the only wiring).

import "context"

// Failpoint names the specific point in the apply sequence where a fault can be
// injected. Each corresponds to a real execution boundary.
type Failpoint string

const (
	FailAfterSecondProbe     Failpoint = "after_second_probe"
	FailAfterArtifactRoles   Failpoint = "after_artifact_roles"
	FailAfterArtifactPlatform Failpoint = "after_artifact_platform"
	FailAfterArtifactBaseline Failpoint = "after_artifact_baseline"
	FailAfterArtifactSeed    Failpoint = "after_artifact_seed"
	FailAfterAdoptionBody    Failpoint = "after_adoption_body"
	FailAfterTargetFingerprint Failpoint = "after_target_fingerprint"
	FailAfterRunInsert       Failpoint = "after_run_insert"
	FailAfterTargetReceipt   Failpoint = "after_target_receipt"
	FailAfterExecutionReceipt Failpoint = "after_execution_receipt"
	FailAfterEvidenceFingerprint Failpoint = "after_evidence_fingerprint"
	FailBeforeCommit         Failpoint = "before_commit"
)

// FailpointController is the test-only injection point. Hit returns a non-nil
// error to fail at the named failpoint, or nil to proceed. Production wiring
// uses InertFailpoints (always nil).
type FailpointController interface {
	Hit(ctx context.Context, fp Failpoint) error
}

// InertFailpoints is the production controller: always returns nil (no fault
// injection). It is the only controller the production binary uses.
type InertFailpoints struct{}

func (InertFailpoints) Hit(context.Context, Failpoint) error { return nil }

// ActiveFailpointController is the process-wide controller. Production =
// InertFailpoints. Tests set this to inject faults. It is never configurable
// via CLI/env/config.
var ActiveFailpointController FailpointController = InertFailpoints{}

// hitFailpoint is the internal helper the apply executor calls at each named
// boundary. It delegates to the active controller.
func hitFailpoint(ctx context.Context, fp Failpoint) error {
	return ActiveFailpointController.Hit(ctx, fp)
}
