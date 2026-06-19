import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

/**
 * Regression guard for the dead-call class of bug.
 *
 * `request()` and `mutation()` in client.ts prepend `BASE = '/api'` to every
 * path. Any path argument that itself starts with `/api/...` therefore resolves
 * to `/api/api/...` and 404s in production. This silently broke several admin
 * pages (AdminMetrics, PolicySimulation, AgentEvaluation) for a long time
 * because the calls don't crash — they just never reach the backend.
 *
 * This test scans client.ts source for that pattern and fails fast.
 */
const clientSrc = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../api/client.ts'),
  'utf8',
);

describe('client.ts path hygiene', () => {
  it('no path argument double-prefixes /api', () => {
    // Match string literals that start with "/api/" — these get BASE prepended.
    // Allow the BASE declaration itself and any "/api/..." inside a comment.
    const lines = clientSrc.split('\n');
    const offenders: string[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      // Skip the BASE constant definition.
      if (/const\s+BASE\s*=/.test(line)) continue;
      // Skip comment-only lines.
      if (/^\s*(\/\/|\*|\/\*)/.test(line)) continue;

      // Find string literals (single or double quoted, and template literals)
      // containing a path that starts with /api/.
      const stringLiterals = [
        ...line.matchAll(/['"`](\/api\/[^'"`]+)['"`]/g),
      ];
      for (const m of stringLiterals) {
        offenders.push(`line ${i + 1}: ${m[1]} — request()/mutation() prepend /api`);
      }
    }

    expect(offenders, offenders.join('\n')).toEqual([]);
  });

  it('teamPath arguments do not duplicate a leading /teams segment', () => {
    // teamPath() already prepends `/teams/{tid}`. A leading `/teams` in its
    // argument would produce `/teams/{tid}/teams/...`.
    const lines = clientSrc.split('\n');
    const offenders: string[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (/^\s*(\/\/|\*|\/\*)/.test(line)) continue;
      const teamPathCalls = [...line.matchAll(/teamPath\(\s*['"``]([^'"`]+)['"`]/g)];
      for (const m of teamPathCalls) {
        if (m[1].startsWith('/teams')) {
          offenders.push(`line ${i + 1}: teamPath('${m[1]}') — teamPath already prepends /teams`);
        }
      }
    }

    expect(offenders, offenders.join('\n')).toEqual([]);
  });
});
