package migration

// apply.go — the version-0001 executor. Implements the verified transaction
// shape (proven by the feasibility tests) with the three corrections:
//
//  1. Physical-connection DESTRUCTION after every apply (the P3 adoption lands
//     the session on a renamed superuser; NOLOGIN doesn't de-privilege the
//     existing session, so the connection must never return to the pool).
//  2. SET ROLE NONE (not RESET ROLE) at fresh-artifact boundaries; assert
//     current_user=session_user after each.
//  3. Runtime actor + duration mapped to a runner.execution_receipt
//     reconciliation row (revision 0001 is artifact-owned and immutable).
//
// Shape:
//
//	acquire dedicated physical connection
//	acquire specific session advisory lock
//	verify embedded identities
//	run full preflight
//	select fresh or P3 path
//	begin target transaction
//	re-run full preflight probe; require complete decision-state equality
//	bind controlled runtime metadata (set_config for P3; SET ROLE NONE for fresh)
//	[ddl_started = true immediately before the first frozen body]
//	execute transformed frozen bodies (RESET-equivalent boundaries for fresh)
//	fully drain + close every MultiResultReader
//	verify artifact-owned revision 0001 exactly
//	compute governed target fingerprint; require 9881c93e...
//	insert completed migration_run
//	append target-fingerprint reconciliation
//	append runner.execution_receipt reconciliation
//	before-commit failpoint (proof build only)
//	commit
//	unlock exact advisory key once
//	destroy physical connection (Hijack + close; never Release)
//
// On any failure: roll back the target tx; while still holding the session lock,
// record a sanitized failure diagnostic in EXTERNAL evidence (the fresh/P3
// version-0001 paths cannot have a durable migration_runs row after rollback
// because the platform schema is created inside the rolled-back tx); unlock;
// destroy the physical connection.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplyOptions is the sanitized input to Apply. No secrets, DSNs, or raw
// payloads. The producing commit is BUILD-BOUND (ldflags), not accepted here;
// callers cannot supply it. The Actor is a caller-supplied label (OS user, CI
// identity) recorded separately from the authoritative session_user/current_user.
type ApplyOptions struct {
	Actor       string // CALLER-SUPPLIED label (OS user, CI actor); NOT authoritative
	ReleaseID   string // compiled release identifier (ldflags)
	EvidenceRef string // sanitized immutable reference (CI run id)
	// PreflightVerifier is the packaging verifier (production = real embedded;
	// tests can inject). Defaults to the real verifier when nil.
	PreflightVerifier PackageVerifier
}

// ApplyResult is the outcome of an apply attempt. On success, the governed
// fingerprint matched the frozen target. On failure, Err carries the cause and
// the diagnostic is sanitized.
type ApplyResult struct {
	Class              Class
	Path               Path
	Code               ReasonCode
	GovernedFingerprint string
	RunID              string
	StartedAt          time.Time
	CompletedAt        time.Time
	ExecutionMs        int64
	DDLStarted         bool // true only when DDL was actually submitted (past preflight + into target tx)
	Err                error
}

// Apply executes the version-0001 migration on a dedicated physical connection
// acquired from the pool. The connection is DESTROYED (never released) after
// every attempt — success or failure — because the P3 adoption leaves the
// session on a renamed superuser identity.
func Apply(ctx context.Context, pool *pgxpool.Pool, opts ApplyOptions) ApplyResult {
	res := ApplyResult{StartedAt: time.Now()}
	verifier := opts.PreflightVerifier
	if verifier == nil {
		verifier = defaultPackageVerifier{}
	}

	// Resolve the build-bound producing commit (NOT CLI-supplied).
	producingCommit, err := ResolveProducingCommit()
	if err != nil {
		res.Err = fmt.Errorf("provenance: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// Acquire a dedicated physical connection.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		res.Err = fmt.Errorf("acquire connection: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// IMPORTANT: this connection is DESTROYED at the end regardless of outcome.
	// We use Hijack to take ownership so Release cannot return it to the pool.
	rawConn := conn.Hijack()
	defer func() {
		// Physical destruction: close the hijacked connection. This ends the
		// backend session, releasing any residual advisory lock and discarding
		// any mutated session/role state (critical after P3 adoption).
		rawConn.Close(context.Background())
	}()

	// Acquire the session advisory lock on the hijacked connection.
	var lockState LockState
	lockState.Conn = nil // rawConn is *pgx.Conn, not *pgxpool.Conn; unlock uses it directly
	if err := acquireLockOnConn(ctx, rawConn, &lockState); err != nil {
		res.Err = fmt.Errorf("acquire lock: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	defer func() {
		// Unlock exactly once on the hijacked connection. The connection is
		// destroyed after, which would release the lock anyway, but explicit
		// unlock is the documented contract and asserts the key was held.
		_ = releaseLockOnConn(context.Background(), &lockState)
	}()

	// Verify embedded identities + run full preflight.
	pf, err := PreflightWithVerifier(ctx, rawConn, verifier)
	if err != nil || pf.Path == PathBlock {
		code := pf.Code
		if err != nil {
			code = CodePackagingMismatch
		}
		res.Err = fmt.Errorf("preflight blocked: %s (ddl_started=%v)", code, false)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	res.Class = pf.Class
	res.Path = pf.Path

	// Short-circuit no-op: a governed-current DB needs no mutation. This is
	// correct ONLY when ALL of these hold:
	//   - frozen target fingerprint matches;
	//   - successful revision 0001 is exact;
	//   - no contradictory revision exists;
	//   - no active or interrupted migration run exists;
	//   - packaged identities still verify (VerifyAll passed in preflight).
	// If an active/interrupted run exists, return a restart/recovery diagnostic
	// rather than silently no-op'ing. The no-op path performs NO ledger or
	// reconciliation write (per the G4 no-op contract).
	if pf.Path == PathNoOp {
		// Check for an active/interrupted run (a non-terminal state indicates a
		// prior apply crashed mid-flight and requires restart reconciliation).
		var activeRun int
		_ = rawConn.QueryRow(ctx, `SELECT count(*) FROM platform.migration_runs WHERE state NOT IN ('completed','blocked','precommit_rolled_back')`).Scan(&activeRun)
		if activeRun > 0 {
			res.Err = fmt.Errorf("governed-current DB has %d active/interrupted migration run(s); restart reconciliation required (not a silent no-op)", activeRun)
			res.Class = ClassUnknownDrifted
			res.Path = PathBlock
			res.Code = CodeRestartRequired
			res.CompletedAt = time.Now()
			res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
			return res
		}
		// Clean no-op: no ledger write, no reconciliation write.
		res.GovernedFingerprint = pf.GovernedFingerprint
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// Generate run_id in Go.
	runID := newRunID()
	res.RunID = runID

	// Begin the target transaction.
	tx, err := rawConn.Begin(ctx)
	if err != nil {
		res.Err = fmt.Errorf("begin target tx: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback(ctx)
		}
	}()

	// Re-run the SAME complete probe inside the target tx and require complete
	// decision-state equality with the original preflight. probeAll populates
	// the structural fields; the fingerprints must be recomputed here too (they
	// are decision-relevant, not merely derived).
	rep, err := probeAll(ctx, tx)
	if err != nil {
		res.Err = fmt.Errorf("re-probe: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// Compute the fingerprints identically to Preflight (read-only on the tx).
	if !rep.Fresh {
		if cap, err := profilerCaptureAdapter(ctx, tx); err == nil {
			if fp, err := profilerFingerprintAdapter(cap); err == nil {
				rep.SourceFingerprint = fp
			}
		}
		if gfp, ok, _ := tryGovernedFingerprint(ctx, tx); ok {
			rep.GovernedFingerprint = gfp
		}
	}
	reClass, rePath, reCode := Classify(rep)
	// Failpoint: after second probe.
	if err := hitFailpoint(ctx, FailAfterSecondProbe); err != nil {
		res.Err = fmt.Errorf("after-second-probe failpoint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	if reClass != pf.Class || rePath != pf.Path || reCode != pf.Code ||
		rep.SourceFingerprint != pf.Probe.SourceFingerprint ||
		rep.GovernedFingerprint != pf.Probe.GovernedFingerprint ||
		rep.DatabaseName != pf.Probe.DatabaseName ||
		rep.PGMajor != pf.Probe.PGMajor {
		res.Err = fmt.Errorf("preflight decision-state changed between probes (race/contention)")
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// ddl_started flips to true immediately before the first frozen body is
	// submitted. Role normalization + metadata binding happen BEFORE this point
	// and are not DDL. This is the real DDL-start state the CLI emits.
	res.DDLStarted = true

	// Execute the path-specific artifact chain.
	switch pf.Path {
	case PathInstall:
		if err := execFreshChain(ctx, tx); err != nil {
			res.Err = fmt.Errorf("fresh chain: %w", err)
			res.CompletedAt = time.Now()
			res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
			return res
		}
	case PathAdopt:
		if err := execAdoption(ctx, tx, producingCommit); err != nil {
			res.Err = fmt.Errorf("adoption: %w", err)
			res.CompletedAt = time.Now()
			res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
			return res
		}
	default:
		res.Err = fmt.Errorf("unsupported path %q for apply", pf.Path)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// Verify artifact-owned revision 0001 exactly.
	if err := verifyRevision0001(ctx, tx); err != nil {
		res.Err = fmt.Errorf("verify revision: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// Compute governed target fingerprint; require 9881c93e...
	signed, _ := loadSignedG2()
	control, _ := loadControl()
	cap, err := governedCaptureLocal(ctx, tx, signed, control)
	if err != nil {
		res.Err = fmt.Errorf("governed capture: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	fp, err := governedFingerprintLocal(cap)
	if err != nil {
		res.Err = fmt.Errorf("governed fingerprint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	res.GovernedFingerprint = fp
	if fp != GovernedTargetFingerprint {
		res.Err = fmt.Errorf("governed fingerprint mismatch: got %s want %s", fp, GovernedTargetFingerprint)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// Failpoint: after target-fingerprint verification.
	if err := hitFailpoint(ctx, FailAfterTargetFingerprint); err != nil {
		res.Err = fmt.Errorf("after-target-fingerprint failpoint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// Insert completed migration_run + the two reconciliation rows (all inside
	// the target tx so a verification/evidence failure rolls back everything).
	if _, err := tx.Exec(ctx, `
		INSERT INTO platform.migration_runs (run_id, source_profile_id, target_version, state, started_at, completed_at, release_id, evidence_ref)
		VALUES ($1, NULLIF($2,''), $3, 'completed', $4, now(), $5, NULLIF($6,''))`,
		runID, profileIDForPath(pf.Path), "0001", res.StartedAt, opts.ReleaseID, opts.EvidenceRef); err != nil {
		res.Err = fmt.Errorf("insert migration_runs: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// Failpoint: after migration-run insert.
	if err := hitFailpoint(ctx, FailAfterRunInsert); err != nil {
		res.Err = fmt.Errorf("after-run-insert failpoint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// Target-fingerprint reconciliation.
	if err := AppendReconciliation(ctx, tx, runID, "governed.target_fingerprint", "target",
		map[string]any{"fingerprint": GovernedTargetFingerprint},
		map[string]any{"fingerprint": fp},
		"pass", opts.EvidenceRef); err != nil {
		res.Err = fmt.Errorf("append target-fp reconciliation: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// Failpoint: after target-fingerprint reconciliation insert.
	if err := hitFailpoint(ctx, FailAfterTargetReceipt); err != nil {
		res.Err = fmt.Errorf("after-target-receipt failpoint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}
	// Runner execution receipt (actor + duration + producing commit + digests).
	res.CompletedAt = time.Now()
	res.ExecutionMs = res.CompletedAt.Sub(res.StartedAt).Milliseconds()
	if err := AppendExecutionReceipt(ctx, tx, ExecutionReceipt{
		RunID:             runID,
		ReleaseID:         opts.ReleaseID,
		PackageDigest:     mustCompositeDigest(),
		TargetFingerprint: fp,
		TargetVersion:     "0001",
		Actor:             opts.Actor,
		Path:              string(pf.Path),
		StartedAt:         res.StartedAt,
		CompletedAt:       res.CompletedAt,
		ExecutionMs:       res.ExecutionMs,
		ProducingCommit:   producingCommit,
		OriginalDigests:   collectOriginalDigests(pf.Path),
		TransformedDigests: collectTransformedDigests(pf.Path),
		EvidenceRef:       opts.EvidenceRef,
	}); err != nil {
		res.Err = fmt.Errorf("append execution receipt: %w", err)
		return res
	}
	// Failpoint: after execution-receipt insert.
	if err := hitFailpoint(ctx, FailAfterExecutionReceipt); err != nil {
		res.Err = fmt.Errorf("after-execution-receipt failpoint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// Recheck the governed fingerprint AFTER the evidence inserts. The migration_
	// runs + reconciliation rows are governed objects, but the fingerprint is
	// structural (no row content), so they must NOT alter it. This second check
	// catches unexpected role, privilege, or catalog side effects introduced
	// while recording evidence — if the fingerprint drifted, roll back everything.
	cap2, err := governedCaptureLocal(ctx, tx, signed, control)
	if err != nil {
		res.Err = fmt.Errorf("post-evidence governed capture: %w", err)
		return res
	}
	fp2, err := governedFingerprintLocal(cap2)
	if err != nil {
		res.Err = fmt.Errorf("post-evidence governed fingerprint: %w", err)
		return res
	}
	if fp2 != GovernedTargetFingerprint {
		res.Err = fmt.Errorf("post-evidence governed fingerprint drift: %s != %s (evidence inserts must not alter the frozen target)", fp2, GovernedTargetFingerprint)
		return res
	}
	// Failpoint: after evidence-fingerprint recheck.
	if err := hitFailpoint(ctx, FailAfterEvidenceFingerprint); err != nil {
		res.Err = fmt.Errorf("after-evidence-fingerprint failpoint: %w", err)
		res.CompletedAt = time.Now()
		res.ExecutionMs = time.Since(res.StartedAt).Milliseconds()
		return res
	}

	// before-commit failpoint.
	if err := hitFailpoint(ctx, FailBeforeCommit); err != nil {
		res.Err = fmt.Errorf("before-commit failpoint: %w", err)
		return res
	}

	// Commit.
	if err := tx.Commit(ctx); err != nil {
		res.Err = fmt.Errorf("commit: %w", err)
		return res
	}
	committed = true
	return res
}

// execFreshChain executes the fresh-install chain with SET ROLE NONE boundary
// normalization before each artifact body. ddl_started is implicitly true
// (the caller has already committed to DDL by invoking the executor).
func execFreshChain(ctx context.Context, tx pgx.Tx) error {
	// Map each artifact to its failpoint (hit after the artifact body executes).
	artifactFailpoints := map[assets.AssetName]Failpoint{
		assets.AssetRolesBootstrap: FailAfterArtifactRoles,
		assets.AssetPlatformSchema: FailAfterArtifactPlatform,
		assets.AssetBaseline:       FailAfterArtifactBaseline,
		assets.AssetSeed:           FailAfterArtifactSeed,
	}
	for _, name := range assets.FreshInstallChain {
		// SET ROLE NONE: deterministically restore current_user=session_user,
		// emulating separate-session application (the frozen-identity derivation).
		if _, err := tx.Exec(ctx, "SET ROLE NONE"); err != nil {
			return fmt.Errorf("set role none before %s: %w", name, err)
		}
		// Assert boundary normalization.
		var eq bool
		if err := tx.QueryRow(ctx, "SELECT current_user = session_user").Scan(&eq); err != nil {
			return fmt.Errorf("boundary check before %s: %w", name, err)
		}
		if !eq {
			return fmt.Errorf("boundary normalization failed before %s: current_user != session_user", name)
		}
		ts, err := Transform(name)
		if err != nil {
			return fmt.Errorf("transform %s: %w", name, err)
		}
		if err := execSimpleProtocolDrained(ctx, tx, string(ts.Body)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
		// Per-artifact failpoint (hit after the artifact body executes).
		if fp, ok := artifactFailpoints[name]; ok {
			if err := hitFailpoint(ctx, fp); err != nil {
				return fmt.Errorf("%s failpoint: %w", fp, err)
			}
		}
	}
	return nil
}

// execAdoption executes the P3 adoption artifact. The producing commit is bound
// via parameterized set_config (extended protocol) BEFORE the body; the body
// (set_config line removed by Transform) runs via simple protocol.
func execAdoption(ctx context.Context, tx pgx.Tx, producingCommit string) error {
	if producingCommit == "" {
		return errors.New("adoption requires a producing commit (ldflags-bound)")
	}
	// Bind the runtime producing commit (extended protocol, parameterized).
	if _, err := tx.Exec(ctx, `SELECT set_config('g3.source_commit', $1, true)`, producingCommit); err != nil {
		return fmt.Errorf("set_config producing commit: %w", err)
	}
	ts, err := Transform(assets.AssetAdoptP3)
	if err != nil {
		return fmt.Errorf("transform adoption: %w", err)
	}
	if err := execSimpleProtocolDrained(ctx, tx, string(ts.Body)); err != nil {
		return fmt.Errorf("exec adoption body: %w", err)
	}
	// Failpoint: after adoption body.
	if err := hitFailpoint(ctx, FailAfterAdoptionBody); err != nil {
		return fmt.Errorf("after-adoption-body failpoint: %w", err)
	}
	return nil
}

// execSimpleProtocolDrained executes multi-statement SQL via the raw simple
// query protocol (PgConn.Exec) and fully drains + closes the MultiResultReader
// before returning. Both result-level and close errors are failures.
func execSimpleProtocolDrained(ctx context.Context, tx pgx.Tx, sql string) error {
	mrr := tx.Conn().PgConn().Exec(ctx, sql)
	for mrr.NextResult() {
		// drain every result
	}
	if err := mrr.Close(); err != nil {
		return fmt.Errorf("simple protocol exec (drain/close): %w", err)
	}
	return nil
}

// verifyRevision0001 confirms the artifact-owned revision 0001 row exists with
// the exact frozen checksum and success=true. The runner does NOT insert/update
// this row — the frozen artifact owns it.
func verifyRevision0001(ctx context.Context, tx pgx.Tx) error {
	var count int
	var checksum string
	var success bool
	err := tx.QueryRow(ctx, `
		SELECT count(*),
		       (SELECT checksum FROM platform.schema_revisions WHERE version='0001' LIMIT 1),
		       (SELECT success FROM platform.schema_revisions WHERE version='0001' LIMIT 1)
		FROM platform.schema_revisions WHERE version='0001'`).Scan(&count, &checksum, &success)
	if err != nil {
		return fmt.Errorf("query revision 0001: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("revision 0001 count=%d (want 1, artifact-owned)", count)
	}
	if checksum != BaselineChecksum {
		return fmt.Errorf("revision 0001 checksum=%s want frozen %s", checksum, BaselineChecksum)
	}
	if !success {
		return errors.New("revision 0001 success=false (want true)")
	}
	return nil
}

// governedCaptureLocal/governedFingerprintLocal are thin wrappers so apply.go
// doesn't import the fingerprint package directly (keeps the dependency graph
// clean for the privilege-boundary denylist).
func governedCaptureLocal(ctx context.Context, q pgx.Tx, signed *fingerprintSigned, control *fingerprintControl) (map[string]any, error) {
	return fpGovernedCapture(ctx, q, signed, control)
}
func governedFingerprintLocal(cap map[string]any) (string, error) { return fpGovernedFingerprint(cap) }

// profileIDForPath returns the source-profile id for the ledger. P3 has a
// deterministic profile id; fresh-install has none (empty).
func profileIDForPath(p Path) string {
	if p == PathAdopt {
		return P3ProfileID
	}
	return ""
}

func mustCompositeDigest() string {
	d, err := CompositeDigest()
	if err != nil {
		return ""
	}
	return d
}

func collectOriginalDigests(p Path) map[string]string {
	chain := chainForPath(p)
	out := map[string]string{}
	for _, name := range chain {
		if d, ok := FrozenDigest[name]; ok {
			out[string(name)] = d
		}
	}
	return out
}

func collectTransformedDigests(p Path) map[string]string {
	chain := chainForPath(p)
	out := map[string]string{}
	for _, name := range chain {
		if ts, err := Transform(name); err == nil {
			out[string(name)] = ts.TransformedSHA256
		}
	}
	return out
}

func chainForPath(p Path) []assets.AssetName {
	if p == PathAdopt {
		return assets.AdoptionChain
	}
	return assets.FreshInstallChain
}
