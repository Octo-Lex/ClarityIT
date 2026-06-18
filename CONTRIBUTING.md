# Contributing to ClarityIT

Thank you for your interest in contributing. This document covers the basics.

## Development Setup

```bash
# Backend (Go) — requires Docker for the test database
make test

# Frontend (web/) — Node 20+
cd web && npm install
npm run dev        # dev server (configure VITE_API_URL to point at your API)
npm test           # vitest unit/component tests
npm run build      # production build
npm run typecheck  # tsc -b (full, incl. tests)
npm run typecheck:app  # tsc app-only (stricter gate)
```

## Pull Requests

1. Branch from `main` (not `master` — the repo was renamed for publication).
2. Keep PRs focused; one logical change per PR.
3. Ensure the following pass before requesting review:
   - `npm test` (frontend) — no regressions
   - `npm run typecheck:app` — zero app type errors
   - `npm run build` — production build succeeds
   - `make test` (backend) — if you touched Go
4. Preserve `data-testid` attributes — they are part of the testing contract and E2E selectors depend on them.
5. Do not introduce raw permission-string literals; use the `Perm.*` enum from `web/src/auth/permissions.ts`.
6. Do not swallow errors silently (`.catch(() => {})`); surface them as loading/error states.

## Commit Messages

Follow the existing convention: `type(scope): summary` (e.g. `feat(web-track3): incidents migration`).

## Code Style

- **Frontend:** React + TypeScript + TanStack Query + the shadcn design system. No `any` on API signatures. Use the unified token system (`var(--card)`, `StatusBadge`), not hardcoded Tailwind colors or `slate-`/`dark:` variants.
- **Backend:** Go modules; see `docs/adr/` for architectural decisions.

## Security

See [SECURITY.md](SECURITY.md). Never commit real credentials, production hostnames, or customer data. If you discover a secret in history, rotate it immediately and notify the maintainers.
