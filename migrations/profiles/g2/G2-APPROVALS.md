# G2 — Decision Approvals (Final, CI-Proven)

**Date:** 1 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)
**Commit:** `4bf055a`
**CI:** [Run 30716145690](https://github.com/Octo-Lex/ClarityIT/actions/runs/30716145690) — success, including G2 isolated fixtures

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Commit | `4bf055a` |
| Raw-byte SHA-256 | `781365c9043ae68c07cc0d9abf0a7d1d9d1c701b954e645bcfa19f5edf4ee92d` |
| Size | 259,157 bytes |

No in-band digest field. The digest above is the raw committed-file SHA-256.

## CI evidence

All G2 isolated fixtures executed and passed:
- `018 PASS: P1-canonical validated; raw-018 divergences confirmed`
- `016 PASS: collision + negative cases validated in isolated schema`
- `029 PASS: five-role posture validated`
- `=== ALL G2 FIXTURES PASSED ===`

## Decisions

| Migration | Decision | Fixture |
|---|---|---|
| 016 | Rename ALL 7 `.edit` → `.update`; collision: canonical survives, union via `ON CONFLICT DO NOTHING`, delete legacy; negative case; assert zero `.edit` | `fixtures/016-permissions.sql` (isolated `g2_016` schema) |
| 018 | Adopt complete P1 shape: 16 P1 assertions in isolated `g2_p1` schema (4 defaults, metadata, 10 absent FKs, CASCADE, trigger, column names, FK counts); raw-018 in `g2_018` schema confirms all 7 divergences; 005-only fails | `fixtures/018-agent-p1-validation.sql` |
| 029 | 5-role posture (`clarityit_app`, `clarityit`, `clarityit_owner`, `clarityit_migrator`, `clarityit_admin`); pre-mutation fail-closed; exact flags; explicit INHERIT/ADMIN options; ownership rules; PUBLIC EXECUTE; sequence privileges | `fixtures/029-role-bootstrap.sql` |

## Conditions

- Migrations 001–040 preserved byte-for-byte.
- Reconciled baseline, migration runner, and corrective revisions NOT created in G2.
- All decisions evidence-bound to P1 (version `1fd353ec…`, SHA-256 `0f81cf93…`).

## Approvals

Architecture and Database must approve the exact target-manifest digest `781365c9…`; Security reviews the 029 privilege decision.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
