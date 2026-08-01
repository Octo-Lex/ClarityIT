# G2 — Decision Approvals (Final)

**Date:** 1 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)
**Commit:** `a1024f7`
**CI:** [Run 30712334705](https://github.com/Octo-Lex/ClarityIT/actions/runs/30712334705) — success, including G2 fixtures

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Commit | `a1024f7` |
| Raw-byte SHA-256 | `83e05335301da5a0a5db5247115573c86864b06f6e0c570d1df84ed693b8c3c7` |
| Size | 256,809 bytes |

The manifest contains no in-band digest field. The digest above is the raw committed-file SHA-256.

## CI evidence

All G2 fixtures executed and passed:
- `018 PASS: canonical P1 agent shape validated`
- `016 PASS: permission normalization validated (simple rename + collision + negative)`
- `029 PASS: role bootstrap validated`
- `=== ALL G2 FIXTURES PASSED ===`

## Decisions

| Migration | Decision | Fixture |
|---|---|---|
| 016 | Rename ALL 7 `.edit` → `.update`; collision: canonical `.update` survives, union via `ON CONFLICT DO NOTHING`, delete legacy; negative case; assert zero `.edit` + all 7 canonical exist | `fixtures/016-permissions.sql` |
| 018 | Adopt complete P1 shape: 14 REFERENCES in 018 (10 absent from P1); 4 default differences; CASCADE difference; metadata absent; trigger present. Both 005-only and raw-018 fail. P0-shared assertions active; P1-only documented | `fixtures/018-agent-schema.sql` |
| 029 | 5-role model (`clarityit_app`, `clarityit`, `clarityit_owner`, `clarityit_migrator`, `clarityit_admin`); pre-mutation fail-closed; ownership caveat; ADMIN option; PUBLIC EXECUTE revocation; sequence privileges | `fixtures/029-role-bootstrap.sql` |

## Conditions

- Migrations 001–040 preserved byte-for-byte.
- Reconciled baseline, migration runner, and corrective revisions NOT created in G2.
- All decisions evidence-bound to P1 (version `1fd353ec…`, SHA-256 `0f81cf93…`).

## Approvals

Architecture and Database must approve the exact target-manifest digest `83e05335…`; Security reviews the 029 privilege decision.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
