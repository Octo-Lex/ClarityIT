# G2 Decision — Migration 016 (Normalize Permissions) — Corrected

## Conflict

Migration 009 seeds seven `.edit` permissions. Migration 016 renames only three (`work.items.edit.own/.any`, `projects.edit`) to `.update`, then asserts `NOT EXISTS ... LIKE '%.edit%'`. Four remain: `incidents.edit.own/.any`, `docs.edit.own/.any`. Fresh install fails.

## P1 evidence

P1 permissions table schema has columns `id`, `name`, `description`, `resource`, `action`, `risk_level`, `created_at`. Unique constraint on `name`. The application code uses `.update` permission names exclusively.

## Decision

**Adopt canonical `update` permissions.** The reconciled baseline will:

### 1. Rename all 7 `.edit` → `.update`

```
work.items.edit.own  → work.items.update.own   (016 handles)
work.items.edit.any  → work.items.update.any   (016 handles)
projects.edit        → projects.update          (016 handles)
incidents.edit.own   → incidents.update.own     (NEW)
incidents.edit.any   → incidents.update.any     (NEW)
docs.edit.own        → docs.update.own          (NEW)
docs.edit.any        → docs.update.any          (NEW)
```

### 2. Collision handling (corrected)

An ID-stable rename works **only** when the canonical row is absent. If both the `.edit` and `.update` rows exist (e.g., from dual-seeding or manual insertion), the resolution is:

1. **Define the canonical `.update` row as the survivor.**
2. **Repoint all `role_permissions`** from the legacy `.edit` row's ID to the canonical `.update` row's ID:
   ```sql
   UPDATE role_permissions rp SET permission_id = canon.id
   FROM permissions legacy, permissions canon
   WHERE rp.permission_id = legacy.id
     AND legacy.name LIKE '%.edit%'
     AND canon.name = replace(legacy.name, '.edit', '.update')
     AND canon.id <> legacy.id;
   ```
3. **Union the role grants** — any role that had either the `.edit` or `.update` permission retains access via the canonical row.
4. **Remove the legacy `.edit` row:**
   ```sql
   DELETE FROM permissions WHERE name LIKE '%.edit%';
   ```
5. **Assert zero `.edit` remain:**
   ```sql
   ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%');
   ```

### 3. No-mutation failure proof

If the canonical `.update` row does not exist and the legacy `.edit` row does, the rename is a simple UPDATE. If both exist, the collision resolution above applies. If neither exists, the assertion fails — the baseline is corrupt.

## Proof: no legacy `.edit` permission remains

```sql
SELECT count(*) FROM permissions WHERE name LIKE '%.edit%';
-- Expected: 0
```
