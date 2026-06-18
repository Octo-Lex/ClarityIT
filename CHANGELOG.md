# Changelog

All notable changes to ClarityIT are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/).

Detailed release notes live in [`docs/releases/`](docs/releases/).

## [Unreleased] — Web UI Rebuild

### Added
- **React Query foundation** — TanStack Query v5 with a hierarchical query-key
  factory covering all 201 backend routes; WebSocket events drive precise
  cache invalidation (replaces the global version-counter).
- **Typed API client** — all 201 endpoints typed (no `any`); `api/types.ts` is
  the authoritative source. Dual error-envelope handling, `If-Match` for
  document optimistic concurrency, `AbortSignal` cancellation.
- **Centralized permission contract** — `auth/permissions.ts` (80 backend
  strings) with a test that reads the Go router to prevent drift.
- **Design system** — consolidated to the shadcn semantic-token model (light +
  dark); new primitives (dropdown-menu, tabs, tooltip, skeleton, avatar, sheet,
  status-badge, label, command palette, sonner toasts).
- **Global UX scaffolding** — `ErrorBoundary`, `Toaster` + `notify` helpers,
  `PageState` (loading/error/empty), ⌘K command palette, rebuilt `AppLayout`
  (collapsible sidebar, mobile slide-over, theme toggle).
- **Test harness** — MSW server, `renderWithProviders`, typed fixtures,
  coverage gate (v8), `typecheck:app`, jsdom polyfills.
- **Real Kanban** — `@dnd-kit` drag-and-drop board with optimistic-concurrency
  status moves.
- **72 new tests** across all migrated pages.

### Fixed
- **Dead 409 conflict modal** — `updateDocument` now sends `If-Match`; the
  document conflict modal is reachable for the first time.
- **Silent error swallowing** — `.catch(() => {})` eliminated across migrated
  pages; errors surface as recoverable states + toasts.
- **Broken dark theme** — `--bg` (near-black) no longer fights `--card`.
- **Wrong permission strings** — 3 nav entries that vanished for all users
  (`work.items.list` → `work.items.view`, etc.).
- **Production build** — `vite build` now succeeds (was blocked by `tsc` errors
  + CSS token conflicts).
- **Permission-string drift** — `auth/permissions.ts` + a test that validates
  against the backend source.

### Changed
- Unified all 16 knowledge files to the design system (zero `slate-`/`dark:`
  classes remain).
- Migrated Dashboard, Queue, Board, Objects, Incidents, Auth, Team, Agents,
  Admin, and Knowledge to React Query.

## [v1.5.0] — 2026-06-17 — Knowledge Productivity

See [`docs/releases/v1.5.0.md`](docs/releases/v1.5.0.md).

### Added
- Knowledge index (`knowledge_items`/`knowledge_chunks`) + 12 source indexers
  + sanitizer (strips secrets/chain-of-thought, SHA-256 hashing).
- Unified `/knowledge` search (PostgreSQL FTS + `ts_headline` snippets).
- Related-knowledge panels (5 deterministic ranking signals).
- "Ask Clarity" source-grounded Q&A → Python `/knowledge-ask` endpoint.
- Collections + saved answers.
- Quality dashboard (stale/duplicates/orphans).
- Migrations 039–040; 1,236 tests passing (772 Go + 355 frontend + 66 Python + 43 E2E).

## [v1.4.0] — Document Productivity

Native block-based document editor, version history, export (MD/PDF/DOCX).

## [v1.3.0] — Team Productivity

Artifacts, meeting summaries, status reports, presentations, template library.

## [v1.2.0] — Operational Intelligence

Remediation proposals, recommendation evidence, context-graph quality, change-risk
scoring, approval policy simulation, agent recommendation evaluation, post-action
outcomes, dry-run previews.

## [v1.1.0] — Operational Refinement

Approvals, MFA (TOTP + WebAuthn), asset actions, Proxmox integration.

## [v1.0.0] — Initial Release

Core IAM, team-scoped objects, work items, incidents, agent runtime (ESAA),
Tool Gateway, event outbox, WebSocket realtime.
