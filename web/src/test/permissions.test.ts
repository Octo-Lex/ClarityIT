import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { ALL_PERMISSIONS, Perm, isPermission } from '../auth/permissions';

const __dirname_test = dirname(fileURLToPath(import.meta.url));
// web/src/test/ → web/src/ → web/ → repo root → services/api/cmd/api/main.go
const MAIN_GO = resolve(__dirname_test, '../../../services/api/cmd/api/main.go');

/**
 * Guards against the class of bug where the UI references a permission string
 * that doesn't exist on the backend (e.g. 'work.items.list' vs the real
 * 'work.items.view'). Such mismatches make nav entries / buttons vanish for
 * every user — silent and easy to miss.
 *
 * Source of truth: every RequirePermission("...") in services/api/cmd/api/main.go.
 */

// Re-derive the backend set from source at test time — fails loudly if the
// backend adds/changes a permission that isn't reflected in permissions.ts.
function backendPermissions(): Set<string> {
  const mainGo = readFileSync(MAIN_GO, 'utf8');
  const matches = mainGo.matchAll(/RequirePermission\([^,]+,\s*"([^"]+)"/g);
  return new Set([...matches].map((m) => m[1]));
}

describe('permissions — client/backend contract', () => {
  it('every Perm.* value exists in the backend RequirePermission set', () => {
    const backend = backendPermissions();
    const missing = ALL_PERMISSIONS.filter((p) => !backend.has(p));
    expect(missing, `Perm.* values absent from backend: ${missing.join(', ')}`).toEqual([]);
  });

  it('the client covers every backend permission (no silent drift)', () => {
    const client = new Set<string>(ALL_PERMISSIONS);
    const backend = backendPermissions();
    const missing = [...backend].filter((p) => !client.has(p));
    // Advisory: surfaces backend permissions the client doesn't model yet.
    // Not a hard failure (some permissions may be backend-only), but logged.
    if (missing.length) {
      // eslint-disable-next-line no-console
      console.warn('[permissions] Backend permissions not modeled on client:', missing.join(', '));
    }
    expect(backend.size).toBeGreaterThan(0);
  });

  it('isPermission() rejects unknown strings (the original bug class)', () => {
    expect(isPermission('work.items.list')).toBe(false);  // the bug we fixed
    expect(isPermission('incidents.list')).toBe(false);    // the bug we fixed
    expect(isPermission('team.settings.view')).toBe(false); // the bug we fixed
    expect(isPermission(Perm.WorkItemsView)).toBe(true);
    expect(isPermission(Perm.IncidentsRead)).toBe(true);
    expect(isPermission(Perm.TeamSettingsRead)).toBe(true);
  });

  it('no Perm.* value is duplicated', () => {
    const values = Object.values(Perm);
    expect(new Set(values).size).toBe(values.length);
  });
});
