# G2 — Decision Approvals (Corrected)

**Date:** 1 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**Target manifest SHA-256:** `41e5c14e4d2275bc250ce7c9d4c186bb87a8d3c169421607c852f646b0584724`

> **This digest is computed from the committed manifest bytes excluding the
> `manifest_sha256` field itself.** It is a detached digest — the manifest does
> not contain its own hash. Verify with:
> ```bash
> python3 -c "import json,hashlib; m=json.load(open('TARGET-SCHEMA-MANIFEST.json')); s={k:v for k,v in m.items() if k!='manifest_sha256'}; print(hashlib.sha256(json.dumps(s,sort_keys=True,ensure_ascii=True,separators=(',',':')).encode()).hexdigest())"
> ```

## Decisions

| Migration | Decision | Key resolution |
|---|---|---|
| 016 | Rename ALL 7 `.edit` → `.update`; collision: canonical survives, repoint `role_permissions`, delete legacy; assert zero `.edit` remain | `fixtures/016-permissions.sql` |
| 018 | Adopt complete P1 agent shape (not 018 verbatim): no `metadata`, no `max_autonomy` default, `status` defaults to `created`, no `team_id`/`created_by` FKs, trigger present; skip 005; 005-only and raw-018 both fail | `fixtures/018-agent-schema.sql` |
| 029 | Bootstrap `clarityit_app` (NOLOGIN) + `clarityit` (LOGIN, member) + `clarityit_owner` + `clarityit_admin`; separate privileged bootstrap from app migration; missing role = fail-closed; least-privilege posture with no superuser in production target | `fixtures/029-role-bootstrap.sql` |

## Corrections from initial G2 (v1)

1. **Target manifest** now contains full function bodies, complete post-decision role posture (`clarityit_app`, `clarityit`, `clarityit_owner`, `clarityit_admin`), memberships, and ownership specification.
2. **Decision 018** corrected: P1 ≠ 018 verbatim. Six P1-specific divergences from 018 documented. Adopt P1 shape, not 018.
3. **Decision 016** corrected: collision handling explicitly defined (canonical `.update` survives, repoint `role_permissions`, delete legacy).
4. **Decision 029** corrected: PostgreSQL raises error on GRANT to nonexistent role (not silent). Least-privilege posture with 4 roles, explicit flags, membership constraints. `clarityit` as superuser is development-exception only.
5. **Manifest digest** corrected to `41e5c14e…` (matches committed bytes). Fixtures and validation SQL committed.

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
