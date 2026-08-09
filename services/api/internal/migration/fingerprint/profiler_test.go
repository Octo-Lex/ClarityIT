package fingerprint

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// p3TestDSN is the approved-P3 fixture DB (pinned 16.14-alpine image,
// POSTGRES_USER=clarityit, schema+seed applied). The source fingerprint of
// this DB must equal the frozen P3 golden cedf689d…
func p3TestDSN() string {
	if d := os.Getenv("CLARITY_G4_P3_DSN"); d != "" {
		return d
	}
	return "postgres://clarityit:clarityit@localhost:55433/clarityit?sslmode=disable"
}

// skipIfNoP3 skips if the P3 fixture DB is not reachable.
func skipIfNoP3(t *testing.T) pgxQuerier {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, p3TestDSN())
	if err != nil {
		t.Skipf("P3 fixture DB not reachable at %s: %v (build P3 fixture with pinned image + clarityit user)", p3TestDSN(), err)
	}
	if _, err := conn.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		conn.Close(ctx)
		t.Skipf("could not set read-only: %v", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

// TestProfilerFingerprintP3ReproducesFrozenGolden is the load-bearing source-
// profiler gate: against the approved P3 fixture, the Go source-profiler port
// must reproduce the frozen P3 golden cedf689d… exactly. If this fails, the
// catalog extraction or canonicalization diverges from the Python oracle.
func TestProfilerFingerprintP3ReproducesFrozenGolden(t *testing.T) {
	q := skipIfNoP3(t)
	ctx := context.Background()
	cap, err := ProfilerCapture(ctx, q)
	if err != nil {
		t.Fatalf("ProfilerCapture: %v", err)
	}
	fp, err := ProfilerFingerprint(cap)
	if err != nil {
		t.Fatalf("ProfilerFingerprint: %v", err)
	}
	const frozen = "cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b"
	if fp != frozen {
		// Debug: dump the Go projection for diffing against the Python golden.
		debugWriteProfilerProjection(t, cap)
		t.Fatalf("P3 source fingerprint mismatch:\n  got    %s\n  frozen %s\n(projection written to testdata for diffing)", fp, frozen)
	}
	t.Logf("P3 source fingerprint OK: %s", fp)
}

func debugWriteProfilerProjection(t *testing.T, cap map[string]any) {
	t.Helper()
	import_json_dump(t, cap, "testdata/profiler_capture_P3_GO_debug.json")
}

// import_json_dump writes the projection as pretty JSON for diffing against
// the Python oracle fixture on failure.
func import_json_dump(t *testing.T, cap map[string]any, dest string) {
	t.Helper()
	b, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		t.Logf("debug dump marshal: %v", err)
		return
	}
	if err := os.WriteFile(dest, b, 0644); err != nil {
		t.Logf("debug dump write: %v", err)
	}
}
