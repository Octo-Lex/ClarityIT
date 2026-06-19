# Design Spec: ClarityIT Design System Restyle

**Date:** 2026-06-19
**Status:** Approved (visual direction validated via rendered mockups)
**Approach:** B — Token swap + legacy-class purge

## 1. Problem

The previous rebuild restyled the shell (Dashboard, AppLayout, shadcn UI primitives)
but left the feature pages on the old design. Three forms of legacy styling remain
in the codebase, which is why the old design "crept in":

| Anti-pattern | Count | Where |
|---|---|---|
| Global legacy classes (`.card`, `.btn-secondary`, `.btn-danger`, `.badge-green/yellow/red/blue/gray`, `.error-msg`) | 93 usages | 28 files |
| Inline CSS-variable incantations (`bg-[var(--card)] border border-[var(--border)] rounded-lg p-4`) | 269 usages | 27 files |
| Raw color utilities (`slate-N`, `gray-N`, `dark:`) | 13 usages | 4 files |

Any restyle that leaves these in place repeats the original failure: two design
systems running in parallel. This spec eliminates the legacy system entirely and
replaces it with a single token-driven one.

## 2. Design language (approved)

Validated against rendered mockups (`docs/design-dark.png`, `docs/design-light.png`,
source `docs/design-mockup.html`). Decisions:

- **Density:** Balanced (~14px base)
- **Neutrals:** Cool neutral (achromatic, Linear-style)
- **Accent:** Linear violet, **toned down** (chroma ~0.13 light / ~0.11 dark)
- **Theme:** Dark-first; both token sets defined; default `.dark` on `<html>`
- **Depth:** Border-led (1px hairline), minimal shadow (hover-only)
- **Typography:** Geist Sans (UI) + Geist Mono (numbers/code/IDs) — already installed
- **Status:** Tinted-background badges with dot, not solid-fill pills

The palette is intentionally restrained — structure and typography lead, color is
a quiet accent.

## 3. Token model (`web/src/index.css`)

Single source of truth. No raw colors anywhere outside this file.

### Light (secondary theme)

```
--background:   oklch(1 0 0)
--surface:      oklch(0.985 0.002 250)     /* new: raised card bg */
--foreground:   oklch(0.18 0.005 250)
--muted:        oklch(0.96 0.002 250)
--muted-foreground: oklch(0.50 0.012 250)
--border:       oklch(0.91 0.003 250)
--input:        oklch(0.93 0.003 250)
--primary:      oklch(0.50 0.13 277)
--primary-foreground: oklch(0.99 0 0)
--ring:         oklch(0.50 0.13 277)
--destructive:  oklch(0.54 0.15 27)
--success:      oklch(0.52 0.09 160)
--warning:      oklch(0.62 0.10 70)
--info:         oklch(0.55 0.09 240)
```

### Dark (default — `.dark` on `<html>`)

```
--background:   oklch(0.155 0.005 257)
--surface:      oklch(0.195 0.006 257)
--foreground:   oklch(0.95 0.002 257)
--muted:        oklch(0.25 0.006 257)
--muted-foreground: oklch(0.63 0.012 257)
--border:       oklch(1 0 0 / 9%)
--input:        oklch(1 0 0 / 14%)
--primary:      oklch(0.62 0.11 277)
--primary-foreground: oklch(0.16 0.01 257)
--ring:         oklch(0.62 0.11 277)
--destructive:  oklch(0.62 0.14 25)
--success:      oklch(0.66 0.10 162)
--warning:      oklch(0.74 0.10 75)
--info:         oklch(0.68 0.09 240)
```

Plus foreground companion tokens for each status (`--success-foreground`,
`--warning-foreground`, `--info-foreground`, `--destructive-foreground`).

### Removed (deletion is the enforcement)

- Legacy alias block (lines ~9–19): `--bg`, `--card`, `--text`, `--text-muted`,
  `--danger`, `--primary-hover`, `--success` (raw), `--warning` (raw)
- Legacy global classes (lines ~89–124): `.card`, `.btn-secondary`, `.btn-danger`,
  `.badge`, `.badge-green/yellow/red/blue/gray`, `.error-msg`
- Global element overrides: `body { font-family: -apple-system... }`,
  `input/select/textarea { ... }`, `button { ... }` (lines ~58–98) — these are
  restated by shadcn primitives and cause specificity conflicts. Primitives own
  element styling; pages must not.

### `--surface` is new

Introduces a distinct raised-card background vs page canvas. `Card` primitive
migrates from `bg-card` → `bg-surface`. This is the one token-shape change;
called out because it affects the `Card` component, not just CSS values.

## 4. Typography

- Remove `body { font-family: -apple-system... }` override so `@theme inline`'s
  `--font-sans: 'Geist Variable'` actually applies app-wide.
- Add `--font-mono: 'Geist Mono'` and `@theme inline { --font-mono: var(--font-mono) }`.
- Apply `font-mono` utility to: KPI values, IDs, timestamps, code blocks.
- Type scale (balanced density):
  - page title: `text-xl font-semibold tracking-tight` (~20px)
  - card title: `text-base font-semibold` (~15px)
  - body: `text-sm` (14px) default
  - caption/label: `text-xs font-medium` (12px), uppercase + tracking for section labels

## 5. Component changes

### 5.1 `Card` (`components/ui/card.tsx`)
- `bg-card` → `bg-surface` (new `--surface` token).
- Keep `ring-1 ring-foreground/10` (border-led). No static shadow.
- **Add hover variant** for interactive cards: `hover:shadow-sm hover:ring-foreground/15`
  via an opt-in `interactive` prop. Default cards stay flat.

### 5.2 `Badge` (`components/ui/badge.tsx`) — the one new component work
Add semantic status variants. Current variants (`default`, `secondary`,
`destructive`, `outline`, `ghost`, `link`) stay; **add**:
- `success` — `bg-success/12 text-success` + leading dot
- `warning` — `bg-warning/12 text-warning` + leading dot
- `info` — `bg-info/12 text-info` + leading dot

Tinted background (10–12% via `color-mix`), not solid fill. Status badges render
a 6px leading dot in the badge's text color via a `dot?: boolean` prop on `Badge`
(defaults `false`). Status variants set `dot` semantics in their docs but the prop
is explicit so non-status badges can opt in if ever needed. This replaces
`.badge-green/yellow/red/blue/gray`.

### 5.3 `Button` (`components/ui/button.tsx`)
No structural change. Inherits new `--primary` / `--destructive` automatically.
Verify `secondary` reads correctly against `--surface` after the token swap.

### 5.4 `Input` / `Textarea` / `Select` (`components/ui/input.tsx` etc.)
Already exist as primitives but underused — the feature pages hand-roll
`bg-[var(--card)] border ... rounded px-2 py-1` inputs instead. Migration replaces
these with the primitives. No primitive change needed beyond verifying the focus
ring uses the new `--ring`/`--primary`.

## 6. Migration plan

Three independent tracks; each is a complete, testable unit. The token swap (T1)
uplifts everything using primitives immediately; T2 and T3 remove the legacy
systems so regression is impossible.

### T1 — Token model + type fix (`index.css`, `Card`, `Badge`)
T1 is split into two ordered sub-steps to avoid temporarily breaking feature
pages (which still use legacy classes until T2 runs):

- **T1a — swap + extend (non-breaking):** swap token values to §3 (both themes),
  add `--surface`, fix typography (§4), add `Badge` status variants + dot
  (§5.2), migrate `Card` to `--surface`. Legacy aliases and global classes
  REMAIN in place. Everything using primitives uplifts immediately; feature
  pages keep working because the legacy classes still resolve.
- **T1b — delete legacy (enforcement, runs LAST):** remove the legacy alias
  block, global classes, and element overrides (§3 "Removed"). This is safe
  only after T2 + T3 + T4 have removed all usages. The drift test (T5) gates
  it: T1b cannot land until the test is green.

**Gate for T1a:** `npm run build` succeeds; Dashboard + AppLayout render
correctly against new tokens (visual check). No feature file is broken.

### T2 — Legacy global-class purge (93 usages / 28 files)
Map each global class to its primitive:
- `.card` / `bg-[var(--card)] border ... rounded-lg` → `<Card>` (or `<div className="rounded-lg border border-border bg-surface p-4">` where Card's flex layout is wrong for the slot)
- `.btn-secondary` → `<Button variant="secondary">` (or `variant="outline"`)
- `.btn-danger` → `<Button variant="destructive">`
- `.badge-green` → `<Badge variant="success">`
- `.badge-yellow` → `<Badge variant="warning">`
- `.badge-red` → `<Badge variant="destructive">`
- `.badge-blue` → `<Badge variant="info">`
- `.badge-gray` → `<Badge variant="secondary">` or `muted`
- `.error-msg` → `<p className="text-sm text-destructive">`

Group by feature directory; each file is a small isolated change. Verify after
each directory.

### T3 — Inline CSS-var incantation purge (269 usages / 27 files)
Replace `bg-[var(--card)] border border-[var(--border)] rounded-lg p-4` and the
`var(--text-muted)` / `var(--danger)` references with token utilities:
- `bg-[var(--card)]` → `bg-surface` (or `bg-card` where appropriate)
- `border-[var(--border)]` → `border-border`
- `text-[var(--text-muted)]` → `text-muted-foreground`
- `text-[var(--danger)]` → `text-destructive`
- `hover:bg-[var(--border)]` → `hover:bg-muted`
- `bg-[var(--primary)]` → `bg-primary`
- The modal containers (`bg-[var(--bg-card)] ... rounded-lg p-6`) → `<Card>` or a
  dedicated `Modal`/`Dialog` surface (verify shadcn `Dialog` exists; if not, use
  Card-styled div).

### T4 — Raw color utility purge (13 usages / 4 files)
- `slate-N`, `gray-N`, `zinc-N` → semantic tokens (`text-muted-foreground`,
  `bg-muted`, `border-border`)
- `dark:` variants → rely on the `.dark` token swap instead

### T5 — Drift-detection test (enforcement)
Add a test (e.g. `src/test/no-legacy-styles.test.ts`) that fails if any source
file under `src/` contains:
- `className=...card`, `btn-secondary`, `btn-danger`, `badge-green`,
  `badge-yellow`, `badge-red`, `badge-blue`, `badge-gray`, `error-msg`
- `bg-[var(--card)]`, `bg-[var(--bg-card)]`, `var(--text-muted)`, `var(--danger)`
- `slate-`, `gray-`, `zinc-` raw utilities (allow in `index.css` and
  `components/ui/` only)

This makes the "crept in" regression structurally impossible — a repeat of the
original failure becomes a failing test.

### T6 — Dark-first default
Apply `class="dark"` to `<html>` by default (in `index.html` or the root render),
with a theme toggle (localStorage-persisted) for light. Verify every page reads
correctly in both themes.

## 7. Testing

- **Unit/component:** existing 427 tests must stay green. New `Badge` variants
  get render tests. New drift-detection test (T5).
- **Visual:** re-render the mockup HTML against the final `index.css` tokens and
  confirm parity with the approved PNGs. Spot-check 3-4 real pages (Dashboard,
  ArtifactsPage, AdminOps, DocumentEditor) in both themes via Playwright screenshots.
- **E2E:** existing 11 smoke tests must stay green; selectors are data-testid-based
  and unaffected by class changes.
- **Build:** `npm run build` clean; `npm run typecheck:app` clean.

## 8. Out of scope

- No new pages or features.
- No backend changes.
- No content/copy changes.
- The migration bug cluster (#1) and backend CI are tracked separately.
- Information architecture / navigation restructuring is not part of this — only
  the visual layer and the component primitives feature pages consume.

## 9. Risks

- **T1b deletes classes before all usages are migrated** → mitigated by ordering
  (T2/T3/T4 complete before T1b) and the drift test catching stragglers.
- **`--surface` introduction** changes `Card` background subtly → verified in T1a
  visual gate before proceeding.
- **Dark-first default** may surprise existing users → a visible theme toggle is
  part of T6; localStorage remembers the choice.
- **269-file incantation purge is mechanical but voluminous** → group by
  directory, verify after each, rely on type-check + drift test as backstops.

## 10. Success criteria

1. Zero legacy global classes in `src/` (drift test green).
2. Zero `bg-[var(--card)]` / `var(--text-muted)` / `var(--danger)` incantations in `src/`.
3. Zero raw `slate/gray/zinc` utilities in `src/` (outside `index.css` + `ui/`).
4. App renders correctly dark-first; light toggle works everywhere.
5. All 427 unit tests + 11 E2E tests green; build + typecheck clean.
6. A side-by-side of the approved mockup vs a real Dashboard screenshot matches.
