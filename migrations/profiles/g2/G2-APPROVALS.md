# G2 — Decision Approvals

**Date:** 1 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**Target manifest SHA-256:** `cb79f6742a135c153459c90f2a4c401916c1a29ffa8fee73685b245bc650fbc9`

## Decisions

| Migration | Decision | Key resolution |
|---|---|---|
| 016 | Adopt canonical `update` permissions; rename ALL 7 `.edit` perms; preserve role grants via ID-stable UPDATE | No `.edit` remains |
| 018 | Adopt P1 agent shape (018): `max_autonomy`, `result`, `tool_registry`; skip 005 agent tables; 005-only fails | No dual-column or IF NOT EXISTS |
| 029 | Create `clarityit_app` (NOLOGIN group) in bootstrap; `clarityit` (LOGIN, INHERIT, member); missing role = fail-closed | Separate bootstrap from app migration |

## Conditions

- Migrations 001–040 preserved byte-for-byte as historical provenance.
- Reconciled baseline, migration runner, and corrective revisions are NOT created during G2.
- All decisions are evidence-bound to P1 (version `1fd353ec…`, SHA-256 `0f81cf93…`).

## Approvals

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
