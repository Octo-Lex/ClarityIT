# G2 — Decision Approvals (CI-Proven on Exact Head SHA, Executable Evidence Complete)

**Date:** 2 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)
**Commit:** `98dd17a`
**CI:** [Push-triggered run on 98dd17a](https://github.com/Octo-Lex/ClarityIT/actions/runs/30736738036) — `event: push`, `headSha: 98dd17a8f6d44e2bea071027666332217b688af7` (checked out directly, **not** a PR merge commit). All three jobs success.

> This record supersedes `c24e997` (which pointed at `fc0bfd5` / digest `ab0dfa8f…`). Five signature-readiness blockers identified in review of `fc0bfd5` are closed in `98dd17a`; the manifest digest changed accordingly.

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Commit | `98dd17a` |
| Raw-byte SHA-256 | `43e4437ba1cad6126a7fa07272be2d21132017191ce43f434f2caaaf61527d14` |
| Size | 290,693 bytes |

No in-band digest field.

## CI evidence (push event on exact SHA — no merge commit, ON_ERROR_STOP=1)

- `018 PASS: P1-canonical validated; raw-018 and 005-only divergences confirmed`
- `016 NEGATIVE PASS: corruption correctly detected` (transactional ROLLBACK proof)
- `016 PASS: all 7 canonical names, dual-grant collision, negative case validated`
- `029 PASS: fail-closed correctly rejected via assert_failure`
- `029 PASS: five-role posture validated with exact flags and membership options` (all three PG16 options: ADMIN/INHERIT/SET)
- `029 PASS: clarityit is non-superuser in production target`
- `R1 PASS: superuser clarityit correctly rejected`
- `R2 PASS: wrong clarityit_app flags correctly rejected`
- `R3 PASS: ADMIN TRUE delegation risk correctly rejected`
- `R4 PASS: partial posture correctly rejected`
- `R5 PASS: extraneous membership correctly rejected`
- `=== ALL G2 FIXTURES PASSED ===`

## Blockers closed in 98dd17a

### B1 — `clarityit_app` INHERIT contradiction
The fixture created `clarityit_app NOINHERIT` while the manifest and DECISION-029 declared `INHERIT`. Fixture now creates and validates `clarityit_app` with `rolinherit=true`, aligned to manifest + decision.

### B2 — Explicit, fully-validated membership options
PostgreSQL 16 stores `admin_option`, `inherit_option`, **and** `set_option` independently in `pg_auth_members`; any omitted option defaults to `TRUE`. The prior `WITH ADMIN OPTION` grant silently granted `set_option=TRUE` and `inherit_option=TRUE`, and only `admin_option` was validated. Now all three options are stated explicitly on every GRANT and validated on all three columns:

| Membership | ADMIN | INHERIT | SET |
|---|---|---|---|
| `clarityit` → `clarityit_app` | FALSE | TRUE | FALSE |
| `clarityit_migrator` → `clarityit_owner` | FALSE | FALSE | TRUE |

`SET` (not `ADMIN`) authorizes `SET ROLE` in PG16. `ADMIN FALSE` on the migrator prevents delegating owner membership. Manifest `target_memberships` carries explicit `admin_option`/`inherit_option`/`set_option` booleans and the exact SQL.

### B3 — Incorrect-role rejection profile
New `029-rejection-profile.sql` runs on a second disposable cluster and proves the validator **rejects** five bad postures (each in `BEGIN…ROLLBACK`):
- **R1** pre-existing superuser `clarityit`
- **R2** wrong `clarityit_app` flags (`NOINHERIT`)
- **R3** `ADMIN TRUE` on migrator→owner (delegation risk)
- **R4** partial posture (3 of 5 roles)
- **R5** extraneous membership (`clarityit_admin` wrongly in `clarityit_app`)

Harness step 3d confirms all five `Rx PASS` notices appear.

### B4 — Contradictory function revocation policy
The manifest prescribed `REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC` while excluding 81 extension functions — `ALL FUNCTIONS` includes those and would break operator classes/casts. Now:
- Per-signature `REVOKE EXECUTE ON FUNCTION <each of 10 application functions> FROM PUBLIC` (enumerated in `application_functions[].public_revoke`)
- `ALTER DEFAULT PRIVILEGES FOR ROLE clarityit_owner REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC` for future functions (owner-scoped; does not affect extension functions)
- Zero aggregate grant patterns remain.

### B5 — CI on the exact head SHA
Added `wp00/g2-schema-decisions` to the workflow `push` trigger. Run `30736738036` is `event: push`, checks out `98dd17a` directly (checkout log: `fetch --depth=1 origin +98dd17a…` → `Checking out the ref 98dd17a…`) — no merge commit.

## Executable proof summary

### 018 (isolated schemas g2_p1, g2_018, g2_005)
- **g2_p1**: 16 P1-specific assertions (max_autonomy no default, metadata absent, 10 absent FKs, CASCADE, trigger, column names, FK counts) — all PASS
- **g2_018**: 7 divergence assertions — confirm 018 ≠ P1
- **g2_005**: 7 failure assertions — confirm 005 ≠ P1

### 016 (isolated schema g2_016)
- All 7 `.edit` permissions seeded; simple rename, dual-grant collision, and transactional negative proof
- **Negative proof:** pre-test checksum → delete canonical pair in `BEGIN` → ASSERT fails (SQLSTATE P0004 / `assert_failure`) → `ROLLBACK` → byte-equality with pre-test state

### 029 (two disposable PostgreSQL 16 clusters)
- **Bootstrap cluster:** real-query readiness probe (twice), pre-check of 0 roles, assert-vs-connection discrimination in 3a, five-role bootstrap with exact flags + all-three-options memberships, non-superuser `clarityit`
- **Rejection cluster:** R1–R5 bad-posture rejection

### Per-object grants inventory (closed-world)
Generated by `scripts/profile/generate_g2_grants.py` (re-derives the application function set from migrations each run):

| Object class | Count | Treatment |
|---|---|---|
| Tables | 64 | SELECT/INSERT/UPDATE to `clarityit_app` |
| Application functions | 10 | per-signature PUBLIC revoke + EXECUTE to `clarityit_app` |
| Extension functions | 81 | EXCLUDED from PUBLIC revoke (pgcrypto/citext/pg_trgm; managed by `CREATE EXTENSION`) |
| Sequences | 1 | USAGE/SELECT to `clarityit_app` |
| Schemas | 1 | USAGE to `clarityit_app` |
| Default privileges | — | owner-scoped PUBLIC revoke for future FUNCTIONS |

## Conditions

- Migrations 001–040 preserved byte-for-byte (verified: `git diff --stat origin/main..HEAD -- 'migrations/0[0-4][0-9]*.sql'` is empty).
- Reconciled baseline, migration runner, and corrective revisions NOT created in G2.

## Approvals

Architecture and Database must approve the exact target-manifest digest `43e4437b…`; Security reviews the 029 privilege decision. This record is **unsigned** — signatures are recorded separately by the named owners.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
