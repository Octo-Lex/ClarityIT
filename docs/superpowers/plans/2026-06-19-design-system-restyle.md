# ClarityIT Design System Restyle — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy dual design system (shadcn primitives + global `.card`/`.btn-*`/`.badge-*` classes + inline `bg-[var(--card)]` incantations) with a single token-driven system in the approved Linear-violet / cool-neutral / dark-first / border-led / Geist language, then guard it with a drift test so the regression can't recur.

**Architecture:** Approach B from the approved spec (`docs/superpowers/specs/2026-06-19-design-system-restyle.md`). Token swap in `index.css` uplifts everything using primitives; three purge tracks migrate the 93 global-class usages (28 files), 269 inline CSS-var incantations (27 files), and 13 raw `slate/gray/dark:` utilities (4 files) onto the existing primitives; a drift-detection test enforces the new state; the legacy classes are then deleted as final enforcement.

**Tech Stack:** React 19, TypeScript, Tailwind 4, shadcn/ui (base-ui primitives), Geist / Geist Mono, vitest + @testing-library, Playwright.

**Two spec corrections discovered during planning (use these, not the spec text):**
1. **StatusBadge already exists** (`src/components/ui/status-badge.tsx`) with success/warning/danger/info/neutral tones + `dot` prop. The migration target for `.badge-green/yellow/red/blue/gray` is this existing component — NOT new variants on `Badge`. Do not add status variants to `Badge`; route status badges to `StatusBadge`.
2. **ThemeProvider already exists** (`src/components/theme/ThemeProvider.tsx`) with light/dark/system + localStorage + toggle, and the toggle is already wired into `AppLayout`. The only change for "dark-first" is the default from `'system'` → `'dark'`. No new toggle UI.

---

## File Structure

**Modified:**
- `web/src/index.css` — token model (both themes), typography, `--surface` token. Legacy classes/aliases DELETED in the final task.
- `web/src/components/ui/card.tsx` — `bg-card` → `bg-surface`; add `interactive` prop.
- `web/src/components/theme/ThemeProvider.tsx` — default theme `'system'` → `'dark'`.
- 27 feature files under `web/src/features/` — purge inline `bg-[var(--card)]` / `var(--text-muted)` / `var(--danger)` incantations.
- 28 feature files (overlap) — purge `.card` / `.btn-secondary` / `.btn-danger` / `.badge-*` / `.error-msg` classes.
- 4 files — purge raw `slate-N`/`gray-N`/`dark:` utilities.

**Created:**
- `web/src/test/no-legacy-styles.test.ts` — drift-detection guard.

**Migration target map (single source of truth — every task refers to this):**

| Legacy pattern | Replace with |
|---|---|
| `className="card p-4"` (div) | `<Card className="p-4">` or `<div className="rounded-xl border border-border bg-surface p-4">` |
| `className="card ..."` where flex/children don't fit Card | `<div className="rounded-xl border border-border bg-surface ...">` |
| `.btn-secondary` | `<Button variant="secondary">` or `<Button variant="outline">` |
| `.btn-danger` | `<Button variant="destructive">` |
| `.badge-green` | `<StatusBadge tone="success">` |
| `.badge-yellow` | `<StatusBadge tone="warning">` |
| `.badge-red` | `<StatusBadge tone="danger">` |
| `.badge-blue` | `<StatusBadge tone="info">` |
| `.badge-gray` | `<StatusBadge tone="neutral">` |
| `.error-msg` | `<p className="text-sm text-destructive">` |
| `bg-[var(--card)]` | `bg-surface` |
| `bg-[var(--bg-card)]` | `bg-surface` |
| `bg-[var(--primary)]` | `bg-primary` |
| `border-[var(--border)]` / `border border-[var(--border)]` | `border-border` |
| `text-[var(--text-muted)]` | `text-muted-foreground` |
| `text-[var(--danger)]` | `text-destructive` |
| `hover:bg-[var(--border)]` | `hover:bg-muted` |
| `slate-N` / `gray-N` / `zinc-N` | nearest semantic token (`text-muted-foreground` / `bg-muted` / `border-border`) |

---

## Task 1: T1a — Token model swap (light + dark)

**Files:**
- Modify: `web/src/index.css:8-56` (light `:root`), `web/src/index.css:177-218` (`.dark`)

- [ ] **Step 1: Replace the light `:root` token block**

Open `web/src/index.css`. Replace the entire `:root { ... }` block (lines ~8–56, from `:root {` through the closing `}` before `body {`) with:

```css
:root {
  /* Cool-neutral light theme (secondary). Dark is the default — see .dark below. */
  --background: oklch(1 0 0);
  --surface: oklch(0.985 0.002 250);
  --foreground: oklch(0.18 0.005 250);
  --card: var(--surface);
  --card-foreground: oklch(0.18 0.005 250);
  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.18 0.005 250);
  --primary: oklch(0.50 0.13 277);
  --primary-foreground: oklch(0.99 0 0);
  --secondary: oklch(0.96 0.002 250);
  --secondary-foreground: oklch(0.28 0.008 250);
  --muted: oklch(0.96 0.002 250);
  --muted-foreground: oklch(0.50 0.012 250);
  --accent: oklch(0.96 0.002 250);
  --accent-foreground: oklch(0.28 0.008 250);
  --destructive: oklch(0.54 0.15 277);
  --destructive-foreground: oklch(0.99 0 0);
  --success: oklch(0.52 0.09 160);
  --success-foreground: oklch(0.99 0 0);
  --warning: oklch(0.62 0.10 70);
  --warning-foreground: oklch(0.18 0.005 250);
  --info: oklch(0.55 0.09 240);
  --info-foreground: oklch(0.99 0 0);
  --input: oklch(0.91 0.003 250);
  --ring: oklch(0.50 0.13 277);
  --border: oklch(0.91 0.003 250);
  --chart-1: oklch(0.50 0.13 277);
  --chart-2: oklch(0.52 0.09 160);
  --chart-3: oklch(0.62 0.10 70);
  --chart-4: oklch(0.54 0.15 27);
  --chart-5: oklch(0.55 0.09 240);
  --radius: 0.625rem;
  --sidebar: var(--surface);
  --sidebar-foreground: oklch(0.18 0.005 250);
  --sidebar-primary: oklch(0.50 0.13 277);
  --sidebar-primary-foreground: oklch(0.99 0 0);
  --sidebar-accent: oklch(0.96 0.002 250);
  --sidebar-accent-foreground: oklch(0.28 0.008 250);
  --sidebar-border: oklch(0.91 0.003 250);
  --sidebar-ring: oklch(0.50 0.13 277);
}
```

Keep the existing `@import` lines at the top unchanged.

- [ ] **Step 2: Replace the `.dark` token block**

Replace the entire `.dark { ... }` block (lines ~177–218) with:

```css
.dark {
  --background: oklch(0.155 0.005 257);
  --surface: oklch(0.195 0.006 257);
  --foreground: oklch(0.95 0.002 257);
  --card: var(--surface);
  --card-foreground: oklch(0.95 0.002 257);
  --popover: oklch(0.195 0.006 257);
  --popover-foreground: oklch(0.95 0.002 257);
  --primary: oklch(0.62 0.11 277);
  --primary-foreground: oklch(0.16 0.01 257);
  --secondary: oklch(0.25 0.006 257);
  --secondary-foreground: oklch(0.95 0.002 257);
  --muted: oklch(0.25 0.006 257);
  --muted-foreground: oklch(0.63 0.012 257);
  --accent: oklch(0.25 0.006 257);
  --accent-foreground: oklch(0.95 0.002 257);
  --destructive: oklch(0.62 0.14 25);
  --destructive-foreground: oklch(0.95 0.002 257);
  --success: oklch(0.66 0.10 162);
  --success-foreground: oklch(0.16 0.01 257);
  --warning: oklch(0.74 0.10 75);
  --warning-foreground: oklch(0.16 0.01 257);
  --info: oklch(0.68 0.09 240);
  --info-foreground: oklch(0.16 0.01 257);
  --border: oklch(1 0 0 / 9%);
  --input: oklch(1 0 0 / 14%);
  --ring: oklch(0.62 0.11 277);
  --chart-1: oklch(0.62 0.11 277);
  --chart-2: oklch(0.66 0.10 162);
  --chart-3: oklch(0.74 0.10 75);
  --chart-4: oklch(0.62 0.14 25);
  --chart-5: oklch(0.68 0.09 240);
  --sidebar: var(--surface);
  --sidebar-foreground: oklch(0.95 0.002 257);
  --sidebar-primary: oklch(0.62 0.11 277);
  --sidebar-primary-foreground: oklch(0.95 0.002 257);
  --sidebar-accent: oklch(0.25 0.006 257);
  --sidebar-accent-foreground: oklch(0.95 0.002 257);
  --sidebar-border: oklch(1 0 0 / 9%);
  --sidebar-ring: oklch(0.62 0.11 277);
}
```

Note: `--card` is kept as an alias of `--surface` so the existing `bg-card` utility keeps working during migration. This avoids a mid-migration break.

- [ ] **Step 3: Verify build + typecheck still pass**

Run: `cd web && npm run build && npm run typecheck:app`
Expected: build succeeds, typecheck clean. (Feature pages still resolve because legacy classes are NOT yet deleted.)

- [ ] **Step 4: Visual spot-check the Dashboard**

Run the dev server, open the app, navigate to the Dashboard, confirm it renders with the new violet/cool-neutral tokens. This is a manual gate — the primitives (`Card`, `Button`) now use the new palette.

- [ ] **Step 5: Commit**

```bash
git add web/src/index.css
git commit -m "style(tokens): swap to cool-neutral/violet palette, add --surface

Light + dark token sets replaced with the approved Linear-violet /
cool-neutral language (toned down). Introduces --surface as the raised-
card background distinct from page canvas; --card kept as alias during
migration. Legacy classes/aliases retained until purge tracks complete."
```

---

## Task 2: T1a — Typography fix (Geist actually applies)

**Files:**
- Modify: `web/src/index.css` — remove the `body { font-family: -apple-system... }` override (currently around lines 58–63) and the global `input/select/textarea` / `button` overrides (lines ~65–98).

- [ ] **Step 1: Fix the body font**

Find the `body { ... }` rule (around line 58). It currently sets `font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;`. Replace the entire `body { ... }` block with:

```css
body {
  background: var(--background);
  color: var(--foreground);
  margin: 0;
}
```

The `@theme inline { --font-sans: 'Geist Variable', sans-serif; }` block (already present later in the file) now actually applies via the `@layer base { html { @apply font-sans; } }` rule.

- [ ] **Step 2: Add --font-mono to the theme**

In the `@theme inline { ... }` block, add `--font-mono` so the `font-mono` utility maps to Geist Mono. Find the line `--font-sans: 'Geist Variable', sans-serif;` and add immediately after it:

```css
  --font-mono: 'Geist Mono', ui-monospace, monospace;
```

- [ ] **Step 3: Do NOT yet delete the global element overrides**

The `input, select, textarea { ... }` and `button { ... }` and `.btn-secondary`/`.btn-danger`/`.card`/`.badge*`/`.error-msg` blocks (lines ~65–124) reference `--card`/`--border`/`--danger`/`--text-muted` legacy aliases. These aliases were removed in Task 1 — but those class blocks still need to resolve for the 28 feature files until T2/T3 purge them. **Re-add minimal legacy aliases** at the top of the `:root` block (after the new tokens) to keep them resolving during migration:

```css
  /* Legacy aliases — removed in T1b after purge tracks complete */
  --bg: var(--background);
  --text: var(--foreground);
  --text-muted: var(--muted-foreground);
  --danger: var(--destructive);
  --primary-hover: var(--primary);
```

(`--card` and `--border` already exist; `--success`/`--warning` already exist as real tokens.)

- [ ] **Step 4: Verify build + dev server**

Run: `cd web && npm run build`
Expected: build succeeds.
Open dev server, confirm Geist is now the active font (compare to before — system fonts vs geometric Geist is visible).

- [ ] **Step 5: Commit**

```bash
git add web/src/index.css
git commit -m "style(type): apply Geist app-wide, add Geist Mono, restore legacy aliases

Remove the body font-family override that was masking Geist. Add
--font-mono for code/number utility. Re-add minimal legacy token aliases
so the not-yet-purged legacy classes keep resolving during migration."
```

---

## Task 3: T1a — Card `interactive` prop + `--surface` migration

**Files:**
- Modify: `web/src/components/ui/card.tsx`

- [ ] **Step 1: Add the `interactive` prop**

In `card.tsx`, the `Card` function signature is currently:
```tsx
function Card({ className, size = "default", ...props }: React.ComponentProps<"div"> & { size?: "default" | "sm" }) {
```

Change it to add `interactive`:
```tsx
function Card({ className, size = "default", interactive = false, ...props }: React.ComponentProps<"div"> & { size?: "default" | "sm"; interactive?: boolean }) {
```

- [ ] **Step 2: Switch bg-card → bg-surface and apply hover when interactive**

In the `cn(...)` call inside `Card`, find `"group/card flex flex-col gap-(--card-spacing) overflow-hidden rounded-xl bg-card py-(--card-spacing)..."`. Change `bg-card` to `bg-surface`. Then append the interactive hover treatment by adding to the `cn` argument list (after the existing className merge):

```tsx
      className,
      interactive && "hover:shadow-sm hover:ring-foreground/15 transition-shadow cursor-pointer"
```

The full updated return:
```tsx
  return (
    <div
      data-slot="card"
      data-size={size}
      className={cn(
        "group/card flex flex-col gap-(--card-spacing) overflow-hidden rounded-xl bg-surface py-(--card-spacing) text-sm text-card-foreground ring-1 ring-foreground/10 [--card-spacing:--spacing(4)] has-data-[slot=card-footer]:pb-0 has-[>img:first-child]:pt-0 data-[size=sm]:[--card-spacing:--spacing(3)] data-[size=sm]:has-data-[slot=card-footer]:pb-0 *:[img:first-child]:rounded-t-xl *:[img:last-child]:rounded-b-xl",
        className,
        interactive && "hover:shadow-sm hover:ring-foreground/15 transition-shadow cursor-pointer"
      )}
      {...props}
    />
  )
```

- [ ] **Step 3: Verify typecheck + tests**

Run: `cd web && npm run typecheck:app && npm test`
Expected: typecheck clean, 429/429 tests pass. (No test references `bg-card` directly, and the new prop is optional with a default.)

- [ ] **Step 4: Commit**

```bash
git add web/src/components/ui/card.tsx
git commit -m "style(card): use --surface bg, add interactive hover variant"
```

---

## Task 4: T6 — Dark-first default

**Files:**
- Modify: `web/src/components/theme/ThemeProvider.tsx:32`

- [ ] **Step 1: Change the default theme**

In `ThemeProvider.tsx`, find:
```tsx
  const [theme, setThemeState] = useState<Theme>(() => {
    if (typeof window === 'undefined') return 'system';
    return (localStorage.getItem(THEME_KEY) as Theme) || 'system';
  });
```

Change the two `'system'` fallbacks to `'dark'`:
```tsx
  const [theme, setThemeState] = useState<Theme>(() => {
    if (typeof window === 'undefined') return 'dark';
    return (localStorage.getItem(THEME_KEY) as Theme) || 'dark';
  });
```

Users who previously toggled keep their choice (localStorage); new users land in dark.

- [ ] **Step 2: Verify**

Run: `cd web && npm run build`
Open dev server in a fresh browser profile (no localStorage) — confirm it loads in dark mode. Toggle (already in the user menu) → light works.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/theme/ThemeProvider.tsx
git commit -m "feat(theme): default to dark mode for new users"
```

---

## Task 5: T5 — Drift-detection test (write FIRST, before purges)

This test is written before the purges so it can guide them. It starts in a "known-violations" state and we shrink the allowlist to zero as each file is migrated.

**Files:**
- Create: `web/src/test/no-legacy-styles.test.ts`

- [ ] **Step 1: Write the test**

Create `web/src/test/no-legacy-styles.test.ts` with exactly this content:

```typescript
import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { resolve, dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const srcRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');

// Patterns that signal legacy styling. Each is a [name, regex] pair.
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

// Files exempt from the check (token definitions / primitive layer).
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
```

- [ ] **Step 2: Run it — expect failure listing all current violations**

Run: `cd web && npx vitest run src/test/no-legacy-styles.test.ts`
Expected: FAIL with a list of files (this is the work manifest for T2/T3/T4). Save the output — it's the punch list.

- [ ] **Step 3: Commit (test is intentionally failing — do NOT commit yet)**

Do not commit until T2/T3/T4 bring violations to zero. The test is the guide. Proceed to Task 6.

---

## Task 6: T2 + T3 + T4 — Purge legacy patterns file-by-file

This is the bulk of the work. Run the drift test after **each file or small group** to watch the violation count drop. Order: alphabetical by feature directory. Each file is a self-contained edit.

**For EVERY file below, the procedure is the same:**

- [ ] **Step 1: Open the file and apply the migration target map** (from the File Structure section above) to every match. Use the exact replacements. When a `<div className="card p-4">` wraps content that fits a card, prefer `<Card className="p-4">...</Card>`; when the div has custom flex/grid that Card's flex-col layout would break, use `<div className="rounded-xl border border-border bg-surface p-4">`.

- [ ] **Step 2: Re-run the drift test** — `cd web && npx vitest run src/test/no-legacy-styles.test.ts 2>&1 | grep -c "FAIL\|✓\|violations"` and confirm that file no longer appears.

- [ ] **Step 3: When a whole feature directory is clean, run the full suite** — `cd web && npm run typecheck:app && npm test` — and commit.

**File list (process in this order, grouping commits by directory):**

Admin directory:
- [ ] `web/src/features/admin/AdminOps.tsx` (10 `.card` + several `var(--text-muted)`/`var(--danger)`)
- [ ] `web/src/features/admin/AdminMetrics.tsx` (`bg-[var(--card)]` ×8)
- [ ] `web/src/features/admin/AdminApprovals.tsx` (`bg-[var(--card)]` ×6)
- [ ] `web/src/features/admin/AdminAssetActions.tsx` (×5)
- [ ] `web/src/features/admin/AgentEvaluationPanel.tsx` (×3 `.card` + ×13 incantations)
- [ ] `web/src/features/admin/ContextQualityPanel.tsx` (×6 `.card` + incantations)
- [ ] `web/src/features/admin/PolicySimulationPanel.tsx` (`.badge-yellow` + ×6)
- [ ] Commit: `style(admin): purge legacy card/badge/var incantations`

Account directory:
- [ ] `web/src/features/account/SecurityPage.tsx` (×7, 2 `.card`-like sections)
- [ ] Commit: `style(account): purge legacy styling`

Artifacts directory (largest cluster):
- [ ] `web/src/features/artifacts/ArtifactsPage.tsx` (`.card`, `.btn-secondary`, `.badge-gray`, `.badge-blue`)
- [ ] `web/src/features/artifacts/DocumentBlockEditor.tsx` (×15 `bg-[var(--card)]`)
- [ ] `web/src/features/artifacts/DocumentEditorPage.tsx` (×12)
- [ ] `web/src/features/artifacts/DocumentGenerateModal.tsx` (×14)
- [ ] `web/src/features/artifacts/AgentAssistPanel.tsx` (×11)
- [ ] `web/src/features/artifacts/ArtifactEditor.tsx` (`bg-[var(--bg-card)]`, raw `slate-`)
- [ ] `web/src/features/artifacts/MeetingSummaryEditor.tsx` (`bg-[var(--bg-card)]`)
- [ ] `web/src/features/artifacts/PresentationModal.tsx` (`bg-[var(--bg-card)]`)
- [ ] `web/src/features/artifacts/StatusReportModal.tsx` (`bg-[var(--bg-card)]`, raw `slate-`)
- [ ] `web/src/features/artifacts/TemplateGallery.tsx` (`.card`, raw `slate-`, `dark:`)
- [ ] `web/src/features/artifacts/VersionHistoryDrawer.tsx` (×10)
- [ ] `web/src/features/artifacts/DocumentToolbar.tsx` (×3)
- [ ] Commit: `style(artifacts): purge legacy card/var/slate incantations`

Knowledge directory:
- [ ] `web/src/features/knowledge/AskClarityPanel.tsx`
- [ ] `web/src/features/knowledge/SavedKnowledgeAnswersPage.tsx` (×6, raw `slate-`/`dark:`)
- [ ] Commit: `style(knowledge): purge legacy styling`

Remaining directories (incidents, assets, remediation, shared):
- [ ] `web/src/features/incidents/PatternCards.tsx`
- [ ] `web/src/features/incidents/RemediationPanel.tsx`
- [ ] `web/src/features/assets/AssetActions.tsx` (`.badge-gray` ×2, ×23 incantations)
- [ ] `web/src/features/remediation/EvidencePanel.tsx` (×11)
- [ ] `web/src/features/shared/OutcomePanel.tsx` (×15)
- [ ] `web/src/test/track4-risk-score.test.tsx` (`.badge-gray`)
- [ ] Commit: `style(features): purge legacy styling from incidents/assets/remediation/shared`

**After the last file:**
- [ ] **Step 4: Confirm drift test is green** — `cd web && npx vitest run src/test/no-legacy-styles.test.ts`
Expected: PASS, 0 violations.
- [ ] **Step 5: Commit the drift test itself**

```bash
git add web/src/test/no-legacy-styles.test.ts
git commit -m "test(drift): add legacy-styling guard, now green

Fails if any source file reintroduces legacy classes, inline var()
incantations, or raw slate/gray/zinc utilities. Makes the 'old design
crept in' regression structurally impossible."
```

---

## Task 7: T1b — Delete legacy classes + aliases (final enforcement)

**Files:**
- Modify: `web/src/index.css`

**Gate:** Only do this AFTER Task 6's drift test is green. If any usage remained, this deletion would unstyle those elements.

- [ ] **Step 1: Delete the legacy global class blocks**

In `web/src/index.css`, delete these blocks entirely (they're between the `body` rule and the `@theme inline` block):
- `.btn-secondary { ... }`
- `.btn-danger { ... }`
- `.card { ... }`
- `.badge { ... }`
- `.badge-green`, `.badge-yellow`, `.badge-red`, `.badge-blue`, `.badge-gray { ... }`
- `.error-msg { ... }`

Also delete the global `input, select, textarea { ... }` and `button { ... }` element override blocks — the shadcn primitives (`Input`, `Button`) own element styling and these cause specificity conflicts.

- [ ] **Step 2: Delete the legacy token aliases**

Delete the minimal aliases added in Task 2 Step 3:
```css
  --bg: var(--background);
  --text: var(--foreground);
  --text-muted: var(--muted-foreground);
  --danger: var(--destructive);
  --primary-hover: var(--primary);
```

- [ ] **Step 3: Verify everything**

Run: `cd web && npm run typecheck:app && npm test && npm run build`
Expected: typecheck clean, 429/429 tests pass, build succeeds. (If anything fails, a file still references a deleted class — the drift test should have caught it, but double-check.)

- [ ] **Step 4: Confirm the drift test is still green** (now it's the only thing preventing reintroduction)

Run: `cd web && npx vitest run src/test/no-legacy-styles.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/index.css
git commit -m "style(css): delete legacy global classes + token aliases

Final enforcement step. With all usages migrated to primitives and the
drift test green, the parallel legacy class system is removed entirely.
Shadcn primitives now own all element styling; no more specificity
conflicts from global input/button overrides."
```

---

## Task 8: Visual parity verification + deploy

**Files:** none modified (verification only)

- [ ] **Step 1: Screenshot the real Dashboard in both themes**

Use Playwright (Chrome headless, as in the mockup capture) to screenshot:
- `http://localhost:5173/` (dev server) in dark mode
- same in light mode

Save to `docs/design-dark-real.png` and `docs/design-light-real.png`.

- [ ] **Step 2: Compare against the approved mockups**

Open `docs/design-dark.png` (approved) vs `docs/design-dark-real.png` side by side. Confirm:
- Violet primary buttons/rings match the toned-down mockup
- Cards are border-led hairlines, no heavy shadow
- Status badges are tinted-background + dot, not solid pills
- Geist font is active (geometric, not system)
- Numbers are monospaced

If any drift, adjust tokens in `index.css` and re-screenshot.

- [ ] **Step 3: Spot-check 3 feature pages in both themes**

Manually verify (dev server): ArtifactsPage, AdminOps, DocumentEditorPage. Confirm no broken layouts, no missing backgrounds, status badges render correctly.

- [ ] **Step 4: Run the E2E suite**

Run: `E2E_API_URL=http://localhost:8765 npx playwright test e2e/smoke.spec.ts`
Expected: 11/11 pass. (Selectors are data-testid-based, unaffected by class changes.)

- [ ] **Step 5: Deploy to production**

Sync the rebuilt `web/` to `192.168.3.20:/opt/clarityit/web/` and rebuild the container (same process as the prior deploy). Verify health at `http://192.168.3.20:3000`.

- [ ] **Step 6: Push and verify CI green**

```bash
git push origin main
```
Watch the frontend CI job — confirm typecheck + tests + build all green.

---

## Self-Review Notes

**Spec coverage check:**
- §3 Token model → Task 1 + Task 7 ✓
- §4 Typography → Task 2 ✓
- §5.1 Card → Task 3 ✓
- §5.2 Badge — CORRECTED: spec proposed new Badge variants; planning found StatusBadge already exists. Route status badges to StatusBadge (used directly in Task 6 per the map). ✓
- §5.3 Button — no change needed; inherits new tokens automatically ✓
- §5.4 Input/Textarea/Select — migrated in Task 6 (replaced inline `bg-[var(--card)] border ...` with `<Input>` where it fits, or token utilities otherwise) ✓
- §6 T1a/T1b/T2/T3/T4/T5/T6 → Tasks 1–7 ✓
- §7 Testing → every task verifies; Task 8 does E2E + visual ✓
- §10 Success criteria → Task 8 Step 2 + drift test green ✓

**Two spec corrections applied** (StatusBadge exists; ThemeProvider exists) — both reduce scope vs the spec. Plan is internally consistent.

**Type consistency:** `--surface` is introduced in Task 1, consumed by Card in Task 3, used as `bg-surface` throughout Task 6. `StatusBadge` `tone` prop values (success/warning/danger/info/neutral) match the migration map. `interactive` prop added in Task 3 is boolean, defaults false.
