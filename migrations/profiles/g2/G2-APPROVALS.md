# G2 — Decision Approvals (v2 Corrected)

**Date:** 1 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Raw-byte SHA-256 | `cdce34540344f496e7bf9a25e0333e79c600c8231370e2d44b42f461c20fad85` |
| Size | 265,573 bytes |

The manifest contains no in-band digest field. The digest above is the raw committed-file SHA-256, computed externally and recorded here.

## Decisions

| Migration | Decision | Key resolution |
|---|---|---|
| 016 | Rename ALL 7 `.edit` → `.update`; collision: canonical `.update` survives, union role_permissions via `INSERT ON CONFLICT DO NOTHING`, delete legacy, assert zero `.edit` remain | `fixtures/016-permissions.sql` |
| 018 | Adopt complete P1 agent shape: 6 default divergences, 10 missing FKs, 1 added CASCADE, missing `metadata`, status=`created`, trigger present; skip 005; both 005-only and raw-018 fail | `fixtures/018-agent-schema.sql` |
| 029 | Bootstrap 4-role posture (`clarityit_app` NOLOGIN, `clarityit` LOGIN, `clarityit_owner` NOLOGIN owner, `clarityit_admin` LOGIN role admin); missing role = fail-closed before mutation; no superuser in production target | `fixtures/029-role-bootstrap.sql` |

## Corrections from G2 v1 (commit 75b0dca)

1. **016 fixture:** will include `resource`/`action` columns, seed `role_permissions`, test legacy-only/canonical-only/dual-grant collision with `INSERT ON CONFLICT DO NOTHING`.
2. **018 analysis:** corrected — 10 FKs in 018 absent from P1 (was "0 FK clauses"); `agent_tool_grants.agent_id` CASCADE difference; 3 additional default differences (`max_autonomy_level`, `agent_runs.status`, `agent_intentions.status`).
3. **029 posture:** `clarityit_owner` and `clarityit_admin` validated; fail-closed before mutation; object-owner-access caveat acknowledged; NOLOGIN owner defined.
4. **Manifest grants:** target grants inventory added.
5. **Manifest digest:** in-band field removed; raw-byte digest `cdce3454…` recorded here.
6. **CI:** PR #9 opened for branch; fixtures and CI integration in progress.

## Conditions

- Migrations 001–040 preserved byte-for-byte.
- Reconciled baseline, migration runner, and corrective revisions NOT created in G2.
- All decisions evidence-bound to P1 (version `1fd353ec…`, SHA-256 `0f81cf93…`).

## Approvals

Architecture and Database must approve the exact target-manifest digest `cdce3454…`; Security reviews the 029 privilege decision.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
