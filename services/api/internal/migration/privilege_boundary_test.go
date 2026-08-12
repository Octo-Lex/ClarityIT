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
// Authorized WP-01 forward SQL is additive migration content in the same
// migration package; it does not weaken either security property.

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/clarityit/api/internal/migration/assets"
)

var forbiddenDeps = []string{
	"github.com/clarityit/api/internal/natsx",
	"github.com/clarityit/api/internal/outbox",
	"github.com/clarityit/api/internal/gateway",
	"github.com/clarityit/api/internal/proxmox",
	"github.com/clarityit/api/internal/remediation",
	"github.com/clarityit/api/internal/storage",
	"github.com/clarityit/api/internal/email",
	"github.com/clarityit/api/internal/integration",
	"github.com/clarityit/api/internal/authz",
	"github.com/clarityit/api/internal/mfa",
	"github.com/clarityit/api/internal/admin",
	"github.com/clarityit/api/internal/agent",
	"github.com/clarityit/api/internal/context",
	"github.com/clarityit/api/internal/domain",
	"github.com/clarityit/api/internal/team",
	"github.com/clarityit/api/internal/knowledge",
	"github.com/clarityit/api/internal/work",
	"github.com/clarityit/api/internal/iam",
	"github.com/clarityit/api/internal/health",
	"github.com/clarityit/api/internal/middleware",
	"github.com/clarityit/api/internal/presenton",
	"github.com/clarityit/api/internal/security",
	"github.com/clarityit/api/internal/approval",
	"github.com/clarityit/api/internal/audit",
	"github.com/clarityit/api/internal/artifact",
	"github.com/clarityit/api/internal/wsx",
}

func TestPrivilegeBoundary_NoForbiddenDeps(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./cmd/clarity-migrate/")
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
		for dep := range depSet {
			if dep == forbidden || strings.HasPrefix(dep, forbidden+"/") {
				t.Errorf("FORBIDDEN DEP: %s is in the cmd/clarity-migrate dependency graph (provider/effect/target path)", dep)
			}
		}
	}
}

// TestPrivilegeBoundary_LegacyMigrationsNeverSelectable inspects the real
// embedded registry and requires every SQL file to belong to the exact approved
// WP-00 set or explicit WP-01 ForwardChain. Historical v1 001-040 remain absent.
func TestPrivilegeBoundary_LegacyMigrationsNeverSelectable(t *testing.T) {
	authorizedSQL := authorizedEmbeddedSQLNames()
	for _, asset := range assets.AllAssets {
		name := string(asset)
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if isLegacyV1SQLName(name) {
			t.Errorf("legacy v1 migration %q is embedded in the asset package", name)
		}
		if !authorizedSQL[name] {
			t.Errorf("unauthorized SQL asset embedded: %q (not in explicit execution set)", name)
		}
	}
}
