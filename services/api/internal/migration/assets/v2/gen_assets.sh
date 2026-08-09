#!/usr/bin/env bash
# Deterministic asset refresh for the G4 runner's embed boundary.
#
# The frozen G3 artifacts live at the repository root under migrations/.
# Go's //go:embed cannot reach outside the module root (services/api/), so the
# runner embeds byte-for-byte copies under services/api/internal/migration/assets/v2/.
#
# Repository-root artifacts remain the source of truth. This script refreshes the
# module-internal copies from them. A CI divergence check (see assets_test.go and
# the g4-proof workflow) re-runs this and fails if any byte differs.
#
# Run from the repository root:
#     bash services/api/internal/migration/assets/v2/gen_assets.sh
#
# Output: 10 files under services/api/internal/migration/assets/v2/, no legacy SQL.
set -euo pipefail

# Resolve the repo root by walking up from this script's directory until both
# the module root (services/api/go.mod) and the migration tree (migrations/v2)
# exist. Robust to CWD and to MSYS path quirks.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$SCRIPT_DIR"
while [ "$REPO_ROOT" != "/" ] && [ -n "$REPO_ROOT" ]; do
  if [ -f "$REPO_ROOT/services/api/go.mod" ] && [ -d "$REPO_ROOT/migrations/v2" ]; then
    break
  fi
  REPO_ROOT="$(dirname "$REPO_ROOT")"
done
DEST="$SCRIPT_DIR"

# Guard: confirm we found a real repo root.
[ -f "$REPO_ROOT/services/api/go.mod" ] || { echo "STOP: repo root not found from $SCRIPT_DIR" >&2; exit 64; }
[ -d "$REPO_ROOT/migrations/v2" ]       || { echo "STOP: migrations/v2 missing at $REPO_ROOT" >&2; exit 64; }

# (src relative to repo root, dest filename)
ARTIFACTS=(
  "migrations/v2/bootstrap/0000_platform.sql|0000_platform.sql"
  "migrations/v2/bootstrap/0000_roles.sql|0000_roles.sql"
  "migrations/v2/baseline/0001_reconciled.sql|0001_reconciled.sql"
  "migrations/v2/baseline/0001_seed.sql|0001_seed.sql"
  "migrations/v2/adoption/0001_adopt_p3.sql|0001_adopt_p3.sql"
  "migrations/v2/manifests/G3-A4-MANIFEST.json|G3-A4-MANIFEST.json"
  "migrations/v2/manifests/CONTROL-SCHEMA-MANIFEST.json|CONTROL-SCHEMA-MANIFEST.json"
  "migrations/v2/manifests/SHA256SUMS|v2-SHA256SUMS"
  "migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json|TARGET-SCHEMA-MANIFEST.json"
  "migrations/legacy/v1/SHA256SUMS|legacy-SHA256SUMS"
)

for pair in "${ARTIFACTS[@]}"; do
  src="${pair%%|*}"
  dst="${pair##*|}"
  if [ ! -f "$REPO_ROOT/$src" ]; then
    echo "STOP: source artifact missing: $src" >&2
    exit 64
  fi
  cp -- "$REPO_ROOT/$src" "$DEST/$dst"
done

echo "Refreshed ${#ARTIFACTS[@]} asset copies under $DEST (relative to $REPO_ROOT)."
