import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { resolve, dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const srcRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');

/**
 * Drift-detection guard for the design-system restyle.
 *
 * The previous rebuild left two parallel design systems running: the shadcn
 * primitives AND a legacy layer of global classes (.card, .btn-*,
 * .badge-green/yellow/red/blue/gray, .error-msg) plus inline CSS-variable
 * incantations (bg-[var(--card)], var(--text-muted), var(--danger)) and raw
 * slate/gray/zinc utilities. That's how the "old design crept in."
 *
 * This test fails if any source file reintroduces those legacy patterns. The
 * primitive layer (index.css + components/ui/*) is exempt because it owns the
 * token definitions and base components.
 */
const LEGACY_PATTERNS: [string, RegExp][] = [
  ['legacy .card class', /\bclassName="[^"]*\bcard\b[^"]*"/],
  ['legacy .btn-secondary', /\bclassName="[^"]*\bbtn-secondary\b/],
  ['legacy .btn-danger', /\bclassName="[^"]*\bbtn-danger\b/],
  ['legacy .badge-green/yellow/red/blue/gray', /\bclassName="[^"]*\bbadge-(green|yellow|red|blue|gray)\b/],
  ['legacy .error-msg', /\bclassName="[^"]*\berror-msg\b/],
  ['inline bg-[var(--card)]', /bg-\[var\(--card\)\]/],
  ['inline bg-[var(--bg-card)]', /bg-\[var\(--bg-card\)\]/],
  ['inline var(--text-muted)', /var\(--text-muted\)/],
  ['inline var(--danger)', /var\(--danger\)/],
  ['raw slate-N utility', /\b(slate|gray|zinc)-\d{2,3}\b/],
];

const EXEMPT = new Set<string>([
  'index.css',
  'components/ui/card.tsx',
  'components/ui/status-badge.tsx',
  'components/ui/badge.tsx',
  'components/ui/button.tsx',
  'components/ui/input.tsx',
]);

function walk(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    if (name === 'node_modules' || name.startsWith('.')) continue;
    const p = join(dir, name);
    const st = statSync(p);
    if (st.isDirectory()) walk(p, out);
    else if (/\.(tsx?|css)$/.test(name)) out.push(p);
  }
  return out;
}

describe('legacy styling drift detection', () => {
  const files = walk(srcRoot);
  const violations: string[] = [];

  for (const file of files) {
    const rel = relative(srcRoot, file).replace(/\\/g, '/');
    if (EXEMPT.has(rel)) continue;
    const src = readFileSync(file, 'utf8');
    for (const [name, re] of LEGACY_PATTERNS) {
      if (re.test(src)) violations.push(`${rel}: ${name}`);
    }
  }

  it('source contains no legacy styling patterns', () => {
    expect(violations, violations.slice(0, 50).join('\n')).toEqual([]);
  });
});
