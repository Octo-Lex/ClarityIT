# G2 Decision — Migration 016 (Normalize Permissions) — Corrected v2

## Conflict

Migration 009 seeds seven `.edit` permissions. Migration 016 renames only three, then asserts none remain. Four survive; fresh install fails.

## Decision

**Adopt canonical `update` permissions.** Rename ALL 7 `.edit` → `.update`.

### Collision handling

1. **If only legacy `.edit` exists:** simple `UPDATE name = replace(name, '.edit', '.update')` and update `action` consistently.
2. **If both `.edit` and `.update` exist (collision):**
   - Define the canonical `.update` row as the survivor
   - Union legacy grants into canonical via `INSERT INTO role_permissions ... ON CONFLICT (role_id, permission_id) DO NOTHING`
   - Delete legacy grant rows from `role_permissions`
   - Delete legacy permission rows from `permissions`
3. **If neither exists:** assertion fails — baseline is corrupt (negative case)

### Grant preservation proof

The fixture (`fixtures/016-permissions.sql`) tests:
- **Legacy-only role:** had `.edit`, retains `.update` after rename (ID-stable)
- **Canonical-only role:** had `.update`, retains `.update` after collision resolution
- **Dual-grant role:** had both `.edit` and `.update` — union via `ON CONFLICT DO NOTHING` prevents PK violation; legacy row deleted; role retains exactly one grant on the canonical row

### Negative case

A corruption case where neither legacy nor canonical exists is caught by the final assertion requiring all 7 canonical names to exist.

## Permission rename map

| Legacy | Canonical |
|---|---|
| `work.items.edit.own` | `work.items.update.own` |
| `work.items.edit.any` | `work.items.update.any` |
| `projects.edit` | `projects.update` |
| `incidents.edit.own` | `incidents.update.own` |
| `incidents.edit.any` | `incidents.update.any` |
| `docs.edit.own` | `docs.update.own` |
| `docs.edit.any` | `docs.update.any` |
