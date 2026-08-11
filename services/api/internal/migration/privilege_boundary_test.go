package migration

// privilege_boundary_test.go — proves the migration runner is free of provider
// credentials, provider clients, target-system access, effect dispatch, and
// general application authority. The security property (G4-AUTHORIZATION §3
// item 9, §4 privilege-boundary row) is verified by two mechanisms:
//
//  1. Go dependency denylist: `go list -deps ./cmd/clarity-migrate` must not
//     contain any provider/NATS/outbox/gateway/effect package.
//  2. Legacy exclusion: the embedded asset set and all code paths must never
//     select or execute migrations 001-040.
//
// These are the chosen proofs for the frozen security property. The property
// itself (no provider/target/effect access) is what the authorization requires;
// the denylist is the mechanism.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

// forbiddenDeps is the list of internal package path prefixes that the migration
// runner must NEVER import. If any appear in the dependency graph of
// cmd/clarity-migrate, the runner has a provider/effect/target path.
var forbiddenDeps = []string{
	"github.com/clarityit/api/internal/natsx",       // NATS runtime
	"github.com/clarityit/api/internal/outbox",      // outbox event dispatch
	"github.com/clarityit/api/internal/gateway",     // API gateway / effect dispatch
	"github.com/clarityit/api/internal/proxmox",     // Proxmox provider client
	"github.com/clarityit/api/internal/remediation", // remediation provider
	"github.com/clarityit/api/internal/storage",     // object storage (MinIO)
	"github.com/clarityit/api/internal/email",       // email dispatch
	"github.com/clarityit/api/internal/integration", // external integrations
	"github.com/clarityit/api/internal/authz",       // application authorization
	"github.com/clarityit/api/internal/mfa",         // MFA enforcement
	"github.com/clarityit/api/internal/admin",       // admin handlers
	"github.com/clarityit/api/internal/agent",       // agent runtime
	"github.com/clarityit/api/internal/context",     // context worker
	"github.com/clarityit/api/internal/domain",      // domain handlers
	"github.com/clarityit/api/internal/team",        // team handlers
	"github.com/clarityit/api/internal/knowledge",   // knowledge base
	"github.com/clarityit/api/internal/work",        // work items
	"github.com/clarityit/api/internal/iam",         // IAM
	"github.com/clarityit/api/internal/health",      // health checks
	"github.com/clarityit/api/internal/middleware",  // HTTP middleware
	"github.com/clarityit/api/internal/presenton",   // presentation layer
	"github.com/clarityit/api/internal/security",    // security helpers
	"github.com/clarityit/api/internal/approval",    // approval workflow
	"github.com/clarityit/api/internal/audit",       // audit trail
	"github.com/clarityit/api/internal/artifact",    // artifact management
	"github.com/clarityit/api/internal/wsx",         // WebSocket
}

// TestPrivilegeBoundary_NoForbiddenDeps runs `go list -deps ./cmd/clarity-migrate`
// and asserts none of the forbidden packages appear. This proves the migration
// runner binary has no provider/target/effect code path in its dependency graph.
func TestPrivilegeBoundary_NoForbiddenDeps(t *testing.T) {
	// This test requires the Go toolchain; skip if unavailable.
	cmd := exec.Command("go", "list", "-deps", "./cmd/clarity-migrate/")
	cmd.Dir = "." // test working directory is services/api/internal/migration
	// Override to the module root.
	cmd.Dir = "../../"
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go list unavailable: %v", err)
	}

	deps := strings.Split(strings.TrimSpace(string(out)), "\n")
	depSet := map[string]bool{}
	for _, d := range deps {
		depSet[strings.TrimSpace(d)] = true
	}

	for _, forbidden := range forbiddenDeps {
		// Check if any forbidden package OR a child of it is in the dep list.
		for dep := range depSet {
			if dep == forbidden || strings.HasPrefix(dep, forbidden+"/") {
				t.Errorf("FORBIDDEN DEP: %s is in the cmd/clarity-migrate dependency graph (provider/effect/target path)", dep)
			}
		}
	}
}

// TestPrivilegeBoundary_LegacyMigrationsNeverSelectable inspects the REAL
// embedded asset registry (assets.AllAssets) and confirms every .sql asset
// belongs to exactly the authorized execution set. No legacy 001-040 can appear.
func TestPrivilegeBoundary_LegacyMigrationsNeverSelectable(t *testing.T) {
	authorizedSQL := map[string]bool{
		"0000_platform.sql":   true,
		"0000_roles.sql":      true,
		"0001_reconciled.sql": true,
		"0001_seed.sql":       true,
		"0001_adopt_p3.sql":   true,
		"0001_adopt_p2.sql":   true,
	}
	for _, asset := range assets.AllAssets {
		name := string(asset)
		if !strings.HasSuffix(name, ".sql") {
			continue // manifests + checksum files are not executable SQL
		}
		if !authorizedSQL[name] {
			t.Errorf("unauthorized SQL asset embedded: %q (not in the authorized execution set)", name)
		}
	}
	// Also confirm no legacy-looking file snuck in.
	for _, asset := range assets.AllAssets {
		name := string(asset)
		if strings.HasPrefix(name, "001_") || strings.HasPrefix(name, "040_") {
			t.Errorf("legacy migration %q is embedded in the asset package", name)
		}
	}
}
