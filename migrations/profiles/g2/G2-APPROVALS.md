# G2 — Decision Approvals (CI-Proven, Executable Evidence Complete)

**Date:** 1 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)
**Commit:** `b1cb33e`
**CI:** [Run on b1cb33e](https://github.com/Octo-Lex/ClarityIT/actions/runs/30716800094) — success, including G2 isolated fixtures

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Commit | `b1cb33e` |
| Raw-byte SHA-256 | `781365c9043ae68c07cc0d9abf0a7d1d9d1c701b954e645bcfa19f5edf4ee92d` |
| Size | 259,157 bytes |

No in-band digest field.

## CI evidence (ON_ERROR_STOP=1 — no false passes)

- `018 PASS: P1-canonical validated; raw-018 and 005-only divergences confirmed`
- `016 PASS: all 7 canonical names, dual-grant collision, negative case validated`
- `029 PASS: five-role posture validated with exact flags and memberships`
- `=== ALL G2 FIXTURES PASSED ===`

## Executable proof summary

### 018 (isolated schemas g2_p1, g2_018, g2_005)
- **g2_p1**: 16 P1-specific assertions (max_autonomy no default, metadata absent, 10 absent FKs, CASCADE, trigger, column names, FK counts) — all PASS
- **g2_018**: 7 divergence assertions (metadata present, DEFAULT 'A0', FKs present, no CASCADE, no trigger, 14 FKs) — all confirm 018 ≠ P1
- **g2_005**: 7 failure assertions (max_autonomy_level not max_autonomy, result_payload not result, no tool_registry, no description/deleted_at) — all confirm 005 ≠ P1

### 016 (isolated schema g2_016)
- Seeds all 7 `.edit` permissions with valid resource/action columns
- role-legacy-only (7 grants), role-canonical-only (1 grant), role-dual-grant (holds both legacy AND canonical simultaneously)
- Collision: `INSERT ON CONFLICT DO NOTHING` unions to canonical; legacy deleted; dual-grant role retains exactly 1
- Negative: unrelated permission proves canonical-count assertion catches corruption
- Asserts all 7 canonical `.update` names exist exactly once after each case

### 029 (ON_ERROR_STOP=1, proper cleanup)
- Pre-mutation fail-closed: asserts roles absent before bootstrap
- Creates 5 roles with exact flags (NOINHERIT on owner/migrator/admin)
- Validates `admin_option` on memberships (INHERIT for clarityit→app, ADMIN for migrator→owner)
- Validates absence of unauthorized memberships (admin not in app, owner not in app)
- GRANT succeeds because clarityit_app exists
- REVOKE before DROP ROLE (no dependency errors)
- Superuser rejection documented as production-only assertion

## Remaining known gap

**Manifest per-object grants inventory** uses aggregate patterns (`ALL TABLES`, `ALL FUNCTIONS`) instead of per-object enumeration. This is acknowledged as incomplete and will be addressed in a follow-up commit. The executable fixtures prove the role posture and decision logic; the per-object grant inventory is documentation precision work.

## Conditions

- Migrations 001–040 preserved byte-for-byte.
- Reconciled baseline, migration runner, and corrective revisions NOT created in G2.

## Approvals

Architecture and Database must approve the exact target-manifest digest `781365c9…`; Security reviews the 029 privilege decision.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
