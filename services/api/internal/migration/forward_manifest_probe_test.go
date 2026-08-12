package migration

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestForwardManifestProbe is the one-shot G1 live PostgreSQL rehearsal used to
// derive the deterministic target-manifest identity from the exact forward
// package. It intentionally fails after a successful apply so CI captures the
// digest. The immediately following commit freezes the digest and replaces this
// probe with permanent convergence tests.
func TestForwardManifestProbe(t *testing.T) {
	const port = 56241
	ctx := context.Background()
	bootstrapPool := applyTestPool(t, "wp01-g1-forward-manifest", port)
	base := Apply(ctx, bootstrapPool, ApplyOptions{
		Actor: "wp01-g1-stage-a-test", ReleaseID: "wp01-g1-probe", EvidenceRef: "sanitized-wp01-g1-probe",
	})
	if base.Err != nil {
		t.Fatalf("Stage A fresh 0001: %v", base.Err)
	}
	if base.GovernedFingerprint != GovernedTargetFingerprint {
		t.Fatalf("Stage A governed fp=%s want=%s", base.GovernedFingerprint, GovernedTargetFingerprint)
	}

	// The production role contract intentionally stores no password in SQL.
	// Provision a fixture-only test password, then execute Stage B as the actual
	// least-privilege clarityit_migrator role which has SET-only membership in
	// clarityit_owner.
	admin, err := pgx.Connect(ctx, applyDSN(port))
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	if _, err := admin.Exec(ctx, `ALTER ROLE clarityit_migrator PASSWORD 'wp01-g1-test-only'`); err != nil {
		admin.Close(ctx)
		t.Fatalf("provision fixture migrator auth: %v", err)
	}
	admin.Close(ctx)

	migratorDSN := fmt.Sprintf("postgres://clarityit_migrator:wp01-g1-test-only@localhost:%d/clarityit?sslmode=disable", port)
	migratorPool, err := pgxpool.New(ctx, migratorDSN)
	if err != nil {
		t.Fatalf("migrator pool: %v", err)
	}
	defer migratorPool.Close()

	forward := ApplyForward(ctx, migratorPool, ApplyOptions{
		Actor: "clarityit_migrator@wp01-g1-test", ReleaseID: "wp01-g1-probe", EvidenceRef: "sanitized-wp01-g1-probe",
	})
	if forward.Err != nil {
		t.Fatalf("Stage B forward apply: %v", forward.Err)
	}

	inspectConn, err := pgx.Connect(ctx, applyDSN(port))
	if err != nil {
		t.Fatalf("inspect connect: %v", err)
	}
	defer inspectConn.Close(ctx)
	ins, err := InspectForward(ctx, inspectConn)
	if err != nil {
		t.Fatalf("InspectForward: %v", err)
	}
	if !ins.Current || ins.CurrentVersion != ForwardTargetVersion {
		t.Fatalf("forward current=%v version=%s", ins.Current, ins.CurrentVersion)
	}
	if ins.PackageDigest != ForwardPackageSHA256 {
		t.Fatalf("package=%s want=%s", ins.PackageDigest, ForwardPackageSHA256)
	}

	t.Errorf("FORWARD_TARGET_MANIFEST_SHA256=%s FORWARD_PACKAGE_SHA256=%s", ins.ManifestDigest, ins.PackageDigest)
}
