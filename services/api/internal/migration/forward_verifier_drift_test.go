package migration

import (
	"context"
	"strings"
	"testing"
)

// TestForwardG1VerifierDrift proves the frozen post-forward verifier fails closed
// for every drift category explicitly required by the G1 migration contract:
// object, constraint, index, grant, and successful-revision history.
func TestForwardG1VerifierDrift(t *testing.T) {
	binaryPath, _ := buildForwardTestCLI(t)

	cases := []struct {
		name     string
		port     int
		mutation string
	}{
		{
			name:     "missing_object",
			port:     56245,
			mutation: `DROP TABLE kernel.case_resource_refs`,
		},
		{
			name:     "missing_constraint",
			port:     56246,
			mutation: `ALTER TABLE kernel.principal_refs DROP CONSTRAINT principal_refs_uuid_v7`,
		},
		{
			name:     "missing_index",
			port:     56247,
			mutation: `DROP INDEX kernel.outbox_messages_pending_idx`,
		},
		{
			name:     "grant_drift",
			port:     56248,
			mutation: `REVOKE UPDATE (published_at) ON kernel.outbox_messages FROM clarityit_app`,
		},
		{
			name:     "revision_drift",
			port:     56249,
			mutation: `UPDATE platform.schema_revisions SET checksum = repeat('0', 64) WHERE version = '0005'`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			container := "wp01-g1-verifier-" + strings.ReplaceAll(tc.name, "_", "-")
			ctx := context.Background()
			pool := applyTestPool(t, container, tc.port)

			base := Apply(ctx, pool, ApplyOptions{
				Actor:       "wp01-g1-stage-a-verifier-test",
				ReleaseID:   "wp01-g1-verifier-drift",
				EvidenceRef: "sanitized-wp01-g1-verifier-drift",
			})
			if base.Err != nil || base.GovernedFingerprint != GovernedTargetFingerprint {
				t.Fatalf("Stage A foundation: err=%v fp=%s", base.Err, base.GovernedFingerprint)
			}

			installForwardTestCLI(t, container, binaryPath)
			forwardOut := runForwardTestCLI(t, container, "forward",
				"-actor", "clarityit_migrator@wp01-g1-verifier-test",
				"-release", "wp01-g1-verifier-drift",
				"-evidence", "sanitized-wp01-g1-verifier-drift",
			)
			if !strings.Contains(forwardOut, ForwardTargetManifestSHA256) &&
				!strings.Contains(forwardOut, ForwardTargetVersion) {
				t.Fatalf("forward did not reach frozen target: %s", forwardOut)
			}

			mutationSQL := "SET ROLE clarityit_owner; " + tc.mutation + ";"
			out, err := execCommandCombined(
				"docker", "exec", "-u", "postgres", container,
				"psql", "-U", "clarityit_migrator", "-d", "clarityit",
				"-v", "ON_ERROR_STOP=1", "-c", mutationSQL,
			)
			if err != nil {
				t.Fatalf("apply %s mutation: %v: %s", tc.name, err, strings.TrimSpace(out))
			}

			verifyOut, verifyErr := runForwardTestCLIRaw(container, "verify")
			if verifyErr == nil {
				t.Fatalf("%s verifier unexpectedly succeeded: %s", tc.name, verifyOut)
			}
			if !strings.Contains(verifyOut, `"status":"blocked"`) {
				t.Fatalf("%s verifier did not fail closed: err=%v out=%s", tc.name, verifyErr, verifyOut)
			}
			if strings.Contains(verifyOut, `"status":"verified"`) {
				t.Fatalf("%s drift produced false verification: %s", tc.name, verifyOut)
			}

			t.Logf("%s drift rejected on disposable PG16 port %d", tc.name, tc.port)
		})
	}
}
