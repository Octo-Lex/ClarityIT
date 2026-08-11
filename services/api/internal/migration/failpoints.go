package migration

// failpoints.go — the typed Failpoint enum ONLY. No controller interface, no
// mutable global, no injection dispatch. The production and proof builds both
// compile this file. The actual hitFailpoint implementation lives in
// failpoints_prod.go (inert, always nil) or failpoints_proof.go (active).

// Failpoint names the specific point in the apply sequence where a fault can be
// injected. Each corresponds to a real execution boundary.
type Failpoint string

const (
	FailAfterSecondProbe         Failpoint = "after_second_probe"
	FailAfterArtifactRoles       Failpoint = "after_artifact_roles"
	FailAfterArtifactPlatform    Failpoint = "after_artifact_platform"
	FailAfterArtifactBaseline    Failpoint = "after_artifact_baseline"
	FailAfterArtifactSeed        Failpoint = "after_artifact_seed"
	FailAfterAdoptionBody        Failpoint = "after_adoption_body"
	FailAfterTargetFingerprint   Failpoint = "after_target_fingerprint"
	FailAfterRunInsert           Failpoint = "after_run_insert"
	FailAfterTargetReceipt       Failpoint = "after_target_receipt"
	FailAfterExecutionReceipt    Failpoint = "after_execution_receipt"
	FailAfterEvidenceFingerprint Failpoint = "after_evidence_fingerprint"
	FailBeforeCommit             Failpoint = "before_commit"
)
