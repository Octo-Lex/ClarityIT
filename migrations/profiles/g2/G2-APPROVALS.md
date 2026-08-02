# G2 — Decision Approvals (CI-Proven on Exact Head SHA, Executable Evidence Complete)

**Date:** 2 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)
**Commit:** `374f7be`
**CI:** [Push-triggered run on 374f7be](https://github.com/Octo-Lex/ClarityIT/actions/runs/30742211882) — `event: push`, `headSha: 374f7becb89d2a40a2715e1155df93474b61a240` (checked out directly via `refs/remotes/origin/wp00/g2-schema-decisions`, **not** a PR merge commit). All three jobs success.

> This record supersedes `759b790` (which pointed at `75766f1` / digest `fdaf5d90…`). One remaining acceptance condition — that the closed-world grant inventory is CI-enforced, not merely structurally correct — is closed in `374f7be`. The manifest digest is **unchanged** (`fdaf5d90…`), because the new comparison confirms the committed content exactly.

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Commit | `75766f1` (last content change) / `374f7be` (CI-enforcement harness) |
| Raw-byte SHA-256 | `fdaf5d90ae9d94811a3d01921a72d1b166e5432b07f5af0cfc939dd13429ac61` |
| Size | 293,691 bytes |

No in-band digest field. Digest and size unchanged from `75766f1`; `374f7be` modifies only the validation harness.

## CI evidence (push event on exact SHA — no merge commit, ON_ERROR_STOP=1)

- `GRANT-INV PASS: generated == committed (64 tables, 10 app functions, 81 extension excluded, 1 sequences, 1 schemas)` ← **new CI-enforcement step**
- `018 PASS: P1-canonical validated; raw-018 and 005-only divergences confirmed`
- `016 NEGATIVE PASS: corruption correctly detected` (transactional ROLLBACK proof)
- `016 PASS: all 7 canonical names, dual-grant collision, negative case validated`
- `029 PASS: fail-closed correctly rejected via assert_failure`
- `029 PASS: five-role posture validated with exact flags and membership options`
- `029 PASS: clarityit is non-superuser in production target`
- `R1 PASS: superuser clarityit correctly rejected` (message-bound to the superuser assertion)
- `R2 PASS: wrong clarityit_app flags correctly rejected` (message-bound)
- `R3 PASS: ADMIN TRUE delegation risk correctly rejected` (message-bound)
- `R4 PASS: partial posture correctly rejected` (message-bound)
- `R5 PASS: extraneous membership correctly rejected` (message-bound)
- `=== ALL G2 FIXTURES PASSED ===`

## Evidence defects closed in 75766f1

### B6 — R1 false-positive superuser proof
The original R1 created only `clarityit`, but `_reject_if_bad()` asserts role count (`n_roles = 5`) **before** the superuser assertion. The role-count ASSERT failed first and the EXCEPTION handler printed `R1 PASS` — it would still pass with the superuser check removed entirely.

**Fix:** R1 now seeds the complete correct 5-role posture with `clarityit` as the sole deviation (`SUPERUSER` via `ALTER ROLE`), isolating the superuser assertion. The handler binds to the specific superuser message via `SQLERRM`: if the rejection fires for any other reason, R1 FAILS. Applied uniformly — R2/R3/R4/R5 each bind to their specific assertion message.

**Proof the binding is real:** reverting R1 to the old single-role posture now reports `R1 PROVE-FAIL: rejected for wrong reason: ... expected exactly 5 application roles, got 1` — confirming the binding distinguishes the superuser rejection from a masked role-count rejection.

### B7 — per-signature function inventory was name-only
`application_functions[]` recorded only `schema` + `name`; the generator discovered functions as a name set. This cannot distinguish overloads and overstated machine-verifiable signature completeness.

**Fix:** each application-function grant now carries:
- `args` (the PostgreSQL argument-type list; `""` for zero-arg functions)
- `identity_signature` (`public.<name>(<argtypes>)` — PostgreSQL function identity)
- `public_revoke_sql` (`REVOKE EXECUTE ON FUNCTION <identity> FROM PUBLIC`)
- `grant_sql` (`GRANT EXECUTE ON FUNCTION <identity> TO clarityit_app`)

The generator partitions by `(name, args)` and hard-errors if any application function is overloaded (signature enumeration required). All 10 current functions are zero-argument, now represented as `public.<name>()`.

### B8 — closed-world grant inventory not CI-enforced
The per-object grant inventory (B4/B7) was structurally correct but not CI-enforced: neither `validate_g2.sh` nor `ci.yml` ran `generate_g2_grants.py` or compared its output to the committed manifest. A drift — a hand-edited manifest, a new migration adding an application function, or a stale generator — would pass silently.

**Fix:** Step 0 in `validate_g2.sh` regenerates the `target_grants` block from the manifest's own object inventory via `generate_g2_grants.py` and asserts it matches the committed manifest (canonical JSON comparison, `sort_keys=True` so dict ordering is not a false fail). It also prints the manifest SHA-256 + size so the receipt and artifact cannot silently diverge. Fail-closed (non-zero exit on any difference).

**Proven bidirectionally fail-closed:**
- privilege mutation (added `DELETE` to a table grant) → `GRANT-INV FAIL`, exit 1
- phantom object (extra application function in manifest) → `GRANT-INV FAIL`, exit 1

Both drifts restored; manifest digest unchanged at `fdaf5d90…` (293,691 bytes) — the comparison confirms the committed content exactly.

## Blockers closed in 98dd17a (prior commit, still in force)

- **B1** `clarityit_app` INHERIT (fixture aligned to manifest + decision)
- **B2** explicit ADMIN/INHERIT/SET membership options (all three validated on `pg_auth_members`)
- **B3** R1–R5 incorrect-role rejection profile (second disposable cluster)
- **B4** per-signature function revocation + owner-scoped default-privileges revoke (no `ALL FUNCTIONS`)
- **B5** push-triggered CI on the exact producing commit

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
- **Rejection cluster:** R1–R5 bad-posture rejection, each message-bound to its specific assertion

### Per-object grants inventory (closed-world, signature-scoped, CI-enforced)
Generated by `scripts/profile/generate_g2_grants.py` (re-derives the application function set from migrations each run; partitions by `(name, args)`). **CI-enforced**: Step 0 of `validate_g2.sh` regenerates the inventory from the committed manifest's object data and asserts byte-equality with the committed `target_grants` block (fail-closed).

| Object class | Count | Treatment |
|---|---|---|
| Tables | 64 | SELECT/INSERT/UPDATE to `clarityit_app` |
| Application functions | 10 | identity-scoped PUBLIC revoke + EXECUTE to `clarityit_app` (each with `public_revoke_sql` + `grant_sql`) |
| Extension functions | 81 | EXCLUDED from PUBLIC revoke (pgcrypto/citext/pg_trgm; managed by `CREATE EXTENSION`) |
| Sequences | 1 | USAGE/SELECT to `clarityit_app` |
| Schemas | 1 | USAGE to `clarityit_app` |
| Default privileges | — | owner-scoped PUBLIC revoke for future FUNCTIONS |

## Conditions

- Migrations 001–040 preserved byte-for-byte (verified: `git diff --stat origin/main..HEAD -- 'migrations/0[0-4][0-9]*.sql'` is empty).
- Reconciled baseline, migration runner, and corrective revisions NOT created in G2.

## Approvals

Architecture and Database must approve the exact target-manifest digest `fdaf5d90…`; Security reviews the 029 privilege decision. This record is **unsigned** — signatures are recorded separately by the named owners.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
