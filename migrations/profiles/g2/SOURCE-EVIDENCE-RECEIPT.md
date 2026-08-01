# G2 — Source Evidence Receipt

**Date:** 1 August 2026
**G2 branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**P1 version ID:** `1fd353ec-2258-4cd5-af57-fdbafc2c7f3a`
**P1 SHA-256:** `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` ✅ verified
**P2a version ID:** `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1`
**P2a SHA-256:** `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` ✅ verified
**P1 fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
**Profiler:** v3.1.0-p1p2 (P1 captured before v3.2.0 bump; fingerprint is version-independent)

## P1 schema facts used for G2 decisions

### P1 agent table shape (the 018 canonical candidate)

**agent_identities** (11 columns):
- `max_autonomy TEXT NOT NULL` (NOT `max_autonomy_level`)
- CHECK constraint: `agent_identities_max_autonomy_level_check` — note: constraint name retains `_level` suffix even though column is `max_autonomy`
- `description TEXT DEFAULT ''`
- `deleted_at TIMESTAMPTZ` (soft delete)

**agent_effect_results** (9 columns):
- `result JSONB NOT NULL DEFAULT '{}'` (NOT `result_payload`)
- CHECK constraint: `agent_effect_results_result_payload_check` — note: constraint name retains `_payload` suffix even though column is `result`

**tool_registry** (7 columns):
- `display_name`, `description`, `risk_level`, `requires_approval`, `requires_mfa`, `is_active`

### P1 permissions table (the 016 authority)

- Columns: `id`, `name`, `description`, `resource`, `action`, `risk_level`, `created_at`
- Unique constraint on `name` (`permissions_name_key`)
- No `.edit` permissions exist in the column-level schema (permission *rows* are data, not schema)
- The application code uses `.update` permission names (confirmed by the CI P0 fixture which patches 016)

### P1 roles and grants (the 029 authority)

- Single role: `clarityit` (superuser, canlogin, createdb, createrole, replication, bypassrls, inherit)
- No memberships (no secondary roles)
- No default privileges configured
- All 65 owned objects owned by `clarityit`
- `clarityit_app` role does NOT exist in P1 (confirmed: 029 grants to a nonexistent role)
- No RLS policies or RLS state in P1

### P1 recommendation_evidence table (029 target)

- Exists in P1 with 16 columns (added by migration 029's CREATE TABLE)
- The `GRANT ... TO clarityit_app` in migration 029 fails silently because the role doesn't exist

## Verification

P1 and P2a retrieved from MinIO evidence bucket via `evidence-verifier` identity.
SHA-256 hashes match the custody receipt exactly. P1 is the upgrade-shape authority.
