package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplyForward performs only WP-01 Stage B. The supplied pool must connect as
// an identity permitted to SET ROLE clarityit_owner (normally
// clarityit_migrator). Stage A bootstrap/adoption credentials are intentionally
// not reused or inferred here.
func ApplyForward(ctx context.Context, pool *pgxpool.Pool, opts ApplyOptions) ApplyResult {
	res := ApplyResult{StartedAt: time.Now(), Path: Path("forward")}
	finish := func(err error) ApplyResult {
		res.Err = err
		res.CompletedAt = time.Now()
		res.ExecutionMs = res.CompletedAt.Sub(res.StartedAt).Milliseconds()
		return res
	}

	producingCommit, err := ResolveProducingCommit()
	if err != nil {
		return finish(fmt.Errorf("forward provenance: %w", err))
	}
	if _, err := VerifyAll(); err != nil {
		return finish(fmt.Errorf("forward package verify: %w", err))
	}
	cat, err := ForwardCatalog()
	if err != nil {
		return finish(err)
	}
	packageDigest := ForwardPackageDigest(cat)
	if ForwardPackageSHA256 != "" && packageDigest != ForwardPackageSHA256 {
		return finish(fmt.Errorf("%w: package=%s frozen=%s", ErrForwardPackaging, packageDigest, ForwardPackageSHA256))
	}

	conn, err := AcquirePinnedConn(ctx, pool)
	if err != nil {
		return finish(err)
	}
	defer conn.Release()
	var lockState LockState
	if err := AcquireMigrationLock(ctx, conn, &lockState); err != nil {
		res.Code = LockDiagnosticCode(err)
		return finish(err)
	}
	defer func() { _ = ReleaseMigrationLock(context.Background(), &lockState) }()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return finish(fmt.Errorf("forward begin: %w", err))
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()

	if _, err := tx.Exec(ctx, "SET LOCAL ROLE clarityit_owner"); err != nil {
		return finish(fmt.Errorf("forward privilege boundary: SET ROLE clarityit_owner: %w", err))
	}
	var ownerRole bool
	if err := tx.QueryRow(ctx, `SELECT current_user='clarityit_owner'`).Scan(&ownerRole); err != nil || !ownerRole {
		return finish(fmt.Errorf("forward privilege boundary: current_user is not clarityit_owner"))
	}

	history, err := readForwardHistory(ctx, tx)
	if err != nil {
		return finish(err)
	}
	state, err := validateForwardHistory(history, cat)
	if err != nil {
		return finish(err)
	}

	if state == "current" {
		if _, err := verifyForwardTargetTx(ctx, tx); err != nil {
			return finish(err)
		}
		res.Path = PathNoOp
		res.Code = CodeOK
		res.Class = ClassGovernedCurrent
		_ = tx.Rollback(ctx)
		res.CompletedAt = time.Now()
		res.ExecutionMs = res.CompletedAt.Sub(res.StartedAt).Milliseconds()
		return res
	}

	// For the only accepted pre-forward state, independently prove the exact
	// WP-00 foundation before submitting the first forward DDL statement.
	gfp, ok, err := tryGovernedFingerprint(ctx, tx)
	if err != nil || !ok || gfp != GovernedTargetFingerprint {
		if err != nil {
			return finish(fmt.Errorf("%w: governed capture: %v", ErrForwardFoundation, err))
		}
		return finish(fmt.Errorf("%w: computed=%s expected=%s", ErrForwardFoundation, gfp, GovernedTargetFingerprint))
	}

	// One atomic batch: a failure in any revision rolls back all 0002+ changes
	// and all >=0002 ledger rows, returning the DB to exact 0001.
	for _, rev := range cat {
		body, err := assets.Bytes(rev.Asset)
		if err != nil {
			return finish(fmt.Errorf("forward asset %s: %w", rev.Version, err))
		}
		if err := validateForwardSQL(body); err != nil {
			return finish(fmt.Errorf("forward SQL %s: %w", rev.Version, err))
		}
		if !res.DDLStarted {
			res.DDLStarted = true
		}
		started := time.Now()
		if err := execSimpleProtocolDrained(ctx, tx, string(body)); err != nil {
			return finish(fmt.Errorf("forward exec %s: %w", rev.Version, err))
		}
		executionMS := time.Since(started).Milliseconds()
		if _, err := tx.Exec(ctx, `
			INSERT INTO platform.schema_revisions
				(version,name,checksum,source_commit,applied_at,applied_by,execution_ms,success)
			VALUES ($1,$2,$3,$4,now(),session_user,$5,true)`,
			rev.Version, rev.Name, rev.Checksum, producingCommit, executionMS); err != nil {
			return finish(fmt.Errorf("forward ledger %s: %w", rev.Version, err))
		}
	}

	manifestDigest, err := verifyForwardTargetTx(ctx, tx)
	if err != nil {
		return finish(err)
	}
	if ForwardTargetManifestSHA256 != "" && manifestDigest != ForwardTargetManifestSHA256 {
		return finish(fmt.Errorf("%w: computed=%s frozen=%s", ErrForwardManifest, manifestDigest, ForwardTargetManifestSHA256))
	}

	runID := newRunID()
	res.RunID = runID
	sourceProfile := foundationSourceProfile(ctx, tx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO platform.migration_runs
			(run_id,source_profile_id,target_version,state,started_at,completed_at,release_id,evidence_ref)
		VALUES ($1,NULLIF($2,''),$3,'completed',$4,now(),$5,NULLIF($6,''))`,
		runID, sourceProfile, ForwardTargetVersion, res.StartedAt, opts.ReleaseID, opts.EvidenceRef); err != nil {
		return finish(fmt.Errorf("forward migration_run: %w", err))
	}
	if err := AppendReconciliation(ctx, tx, runID, "wp01.forward_manifest", "kernel+compat",
		map[string]any{"target_version": ForwardTargetVersion, "package_digest": packageDigest, "manifest_digest": ForwardTargetManifestSHA256},
		map[string]any{"target_version": ForwardTargetVersion, "package_digest": packageDigest, "manifest_digest": manifestDigest},
		"pass", opts.EvidenceRef); err != nil {
		return finish(fmt.Errorf("forward reconciliation: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return finish(fmt.Errorf("forward commit: %w", err))
	}
	committed = true
	res.Code = CodeOK
	res.Class = ClassGovernedCurrent
	return finish(nil)
}

// foundationSourceProfile preserves Stage-A adoption provenance instead of
// relabelling a post-forward DB as P2/P3. Fresh installs legitimately return
// empty because their 0001 run has no adoption source profile.
func foundationSourceProfile(ctx context.Context, tx pgx.Tx) string {
	var profile string
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(source_profile_id,'')
		FROM platform.migration_runs
		WHERE target_version='0001' AND state='completed'
		ORDER BY completed_at DESC NULLS LAST, started_at DESC
		LIMIT 1`).Scan(&profile)
	return profile
}
