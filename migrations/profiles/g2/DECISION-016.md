# G2 Decision — Migration 016 (Normalize Permissions)

## Conflict

Migration 016 renames three permissions (`work.items.edit.own`, `work.items.edit.any`, `projects.edit`) to their `.update` canonical forms, then asserts `NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%')`.

Migration 009 seeds seven `.edit` permissions. 016 renames only three. Four remain: `incidents.edit.own`, `incidents.edit.any`, `docs.edit.own`, `docs.edit.any`. The assertion fails on fresh install.

The P0 CI fixture patches 016 by replacing the renames with a broader UPDATE that catches all `.edit` permissions.

## P1 evidence

P1's permissions table schema (7 columns: `id`, `name`, `description`, `resource`, `action`, `risk_level`, `created_at`) contains no `.edit` permissions in production data. The application code and all current queries use `.update` permission names. Production was migrated past this conflict (likely by manual intervention or a prior fix not captured in the migration chain).

## Decision

**Adopt canonical `update` permissions.** The reconciled baseline will:

1. Rename ALL `.edit` permissions to `.update` (not just the three in 016):
   - `incidents.edit.own` → `incidents.update.own`
   - `incidents.edit.any` → `incidents.update.any`
   - `docs.edit.own` → `docs.update.own`
   - `docs.edit.any` → `docs.update.any`
   - (plus the three 016 already handles: `work.items.edit.own/.any`, `projects.edit`)

2. Preserve the union of existing role grants. The `role_permissions` table references permission IDs; as long as the renames preserve IDs (UPDATE on the `name` column only), all existing role grants are automatically preserved.

3. Collapse `incidents.edit.own/.any` and `docs.edit.own/.any` collisions deterministically by renaming them to `incidents.update.own/.any` and `docs.update.own/.any` respectively — matching the pattern already established by 016 for `work.items`.

4. The assertion `NOT EXISTS ... LIKE '%.edit%'` passes because all seven seeded `.edit` permissions have been renamed to `.update`.

## Proof: no legacy `.edit` permission remains

After the reconciled baseline applies:
```sql
SELECT count(*) FROM permissions WHERE name LIKE '%.edit%';
-- Expected: 0
```

## Fixtures

A sanitized fixture (`fixtures/016-permissions.sql`) will reproduce all seven `.edit` permissions from 009, apply the complete rename, and assert the result is zero — proving the resolution works end-to-end.
