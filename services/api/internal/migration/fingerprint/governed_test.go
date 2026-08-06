package fingerprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/clarityit/api/internal/migration/assets"
	"github.com/jackc/pgx/v5"
)

// testDSN is the isolated PG16 fixture used for fingerprint proof. Override
// with CLARITY_G4_TEST_DSN; defaults to the local Docker fixture container.
func testDSN() string {
	if d := os.Getenv("CLARITY_G4_TEST_DSN"); d != "" {
		return d
	}
	return "postgres://postgres:postgres@localhost:55432/clarityit?sslmode=disable"
}

// skipIfNoDB skips the test if the fixture database is not reachable. These
// tests require a fresh-install PG16 (apply the full G3 chain first).
func skipIfNoDB(t *testing.T) pgxQuerier {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("fixture DB not reachable at %s: %v (apply the fresh-install chain first)", testDSN(), err)
	}
	// Enforce read-only at the session level for every fingerprint test.
	if _, err := conn.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		conn.Close(context.Background())
		t.Skipf("could not set read-only: %v", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

// TestGovernedFingerprintFreshInstallReproducesFrozenTarget is the load-bearing
// fingerprint gate: against a fresh-install PG16 (full G3 chain applied), the
// Go governed-fingerprint port must reproduce the frozen target digest
// 9881c93e… . If this fails, the catalog extraction or canonicalization diverges
// from the Python oracle.
func TestGovernedFingerprintFreshInstallReproducesFrozenTarget(t *testing.T) {
	q := skipIfNoDB(t)

	signed, err := loadSignedG2Manifest()
	if err != nil {
		t.Fatalf("load signed G2 manifest: %v", err)
	}
	control, err := loadControlManifest()
	if err != nil {
		t.Fatalf("load control manifest: %v", err)
	}

	ctx := context.Background()
	cap, err := GovernedCapture(ctx, q, signed, control)
	if err != nil {
		t.Fatalf("GovernedCapture: %v", err)
	}
	fp, err := GovernedFingerprint(cap)
	if err != nil {
		t.Fatalf("GovernedFingerprint: %v", err)
	}
	const frozen = "9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6"
	if fp != frozen {
		// Dump the projection for diffing against the Python oracle fixture.
		debugWriteProjection(t, cap)
		t.Fatalf("governed fingerprint mismatch:\n  got    %s\n  frozen %s\n(projection written to testdata for diffing)", fp, frozen)
	}
	t.Logf("governed fingerprint OK: %s", fp)
}

// TestRolesDigestMatchesFrozen confirms the roles_digest projection field
// matches the frozen value recorded in the G3 manifest adoption block
// (2273a104…). This isolates the roles/memberships layer from the full
// projection.
func TestRolesDigestMatchesFrozen(t *testing.T) {
	q := skipIfNoDB(t)
	ctx := context.Background()
	roles, err := queryRoles(ctx, q)
	if err != nil {
		t.Fatalf("queryRoles: %v", err)
	}
	memberships, err := queryMemberships(ctx, q)
	if err != nil {
		t.Fatalf("queryMemberships: %v", err)
	}
	got := rolesDigest(roles, memberships)
	const frozen = "2273a104fa6145ebe699ffc570da41941d49df4584ee2b093f323ce8d5a0a7c3"
	if got != frozen {
		t.Fatalf("roles_digest mismatch:\n  got    %s\n  frozen %s", got, frozen)
	}
}

func loadSignedG2Manifest() (*SignedG2Manifest, error) {
	raw, err := assets.Bytes(assets.AssetG2Manifest)
	if err != nil {
		return nil, err
	}
	var s SignedG2Manifest
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("unmarshal G2 manifest: %w", err)
	}
	return &s, nil
}

func loadControlManifest() (*ControlManifestFunctions, error) {
	raw, err := assets.Bytes(assets.AssetControlManifest)
	if err != nil {
		return nil, err
	}
	var c ControlManifestFunctions
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("unmarshal control manifest: %w", err)
	}
	return &c, nil
}

func debugWriteProjection(t *testing.T, cap map[string]any) {
	t.Helper()
	b, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile("testdata/governed_capture_GO_debug.json", b, 0644); err != nil {
		t.Logf("could not write debug projection: %v", err)
	}
}
