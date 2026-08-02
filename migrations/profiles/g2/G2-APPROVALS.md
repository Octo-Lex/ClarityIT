# G2 — Decision Approvals (CI-Proven on Exact Head SHA, Executable Evidence Complete)

**Date:** 2 August 2026
**Branch:** `wp00/g2-schema-decisions` (stacked from `0dd21d8`)
**PR:** [#9](https://github.com/Octo-Lex/ClarityIT/pull/9) (DRAFT)
**Commit:** `4eea00b` (carries the regenerated checksum + receipt identity for the B11-corrected manifest)
**CI:** [Push-triggered run on 4eea00b](https://github.com/Octo-Lex/ClarityIT/actions/runs/30753928693) — `event: push`, `headSha: 4eea00bbcb53e953a76f991df033cdd31d14d11f` (checked out directly via `refs/remotes/origin/wp00/g2-schema-decisions`, **not** a PR merge commit). All three jobs success.

> This record supersedes `f974e57`. B11 (`5d67e3e`) corrected a security-significant internal contradiction: the manifest's `decision_029.resolution` prose said `clarityit_migrator` has "ADMIN on owner," conflicting with its own `target_memberships` (`admin_option: false, set_option: true`), DECISION-029, and the R3 fixture (which rejects `ADMIN TRUE`). ADMIN delegates membership; SET permits `SET ROLE`. The manifest bytes changed, so the identity is now `1f6e3142…` / 284,064 bytes (superseding `ace036c2…` / 283,888). B11 also corrected the PG16 omitted-option rationale (ADMIN→FALSE, SET→TRUE, INHERIT→from member's `rolinherit`).

## Target manifest identity (detached)

| Property | Value |
|---|---|
| File | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Commit | `5d67e3e` (last content change — B11 authority-semantics correction) |
| **Committed Git blob SHA-256** | `1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63` |
| **Committed Git blob size** | 284,064 bytes |
| Line endings (blob) | LF |
| Detached checksum file | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.sha256` (CI-asserted by Step 0) |

No in-band digest field. The blob digest is authoritative; the CRLF working-tree digest is platform-specific and not the artifact CI tests. `.gitattributes` pins `migrations/profiles/g2/**` to `eol=lf`. (B11 superseded the prior `ace036c2…` / 283,888 identity — that manifest contained a contradictory `ADMIN on owner` in `decision_029.resolution`.)

### Machine-readable receipt identity (CI-bound)

The fenced block below is the single source of truth for the target-manifest identity cited by this receipt. `validate_g2.sh` Step 0 parses it and asserts the digest and size match BOTH the detached checksum file AND the committed Git blob. A receipt-only edit to these values fails CI; a blob/checksum change without a matching receipt edit also fails CI.

```g2-receipt-identity
manifest_path: migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json
manifest_blob_sha256: 1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63
manifest_blob_size: 284064
```

## CI evidence (push event on exact SHA — no merge commit, ON_ERROR_STOP=1)

- `GRANT-INV PASS: generated == committed (64 tables, 10 app functions, 81 extension excluded, 1 sequences, 1 schemas)`
- `BLOB-DIGEST PASS: committed blob sha256 1f6e3142… (284064 bytes) == checksum file`
- `RECEIPT-BIND PASS: receipt == checksum == committed blob (sha256 1f6e3142…, 284064 bytes)`
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

**Fix:** Step 0 in `validate_g2.sh` regenerates the `target_grants` block from the manifest's own object inventory via `generate_g2_grants.py` and asserts it matches the committed manifest (canonical JSON comparison, `sort_keys=True` so dict ordering is not a false fail). Fail-closed (non-zero exit on any difference).

**Proven bidirectionally fail-closed:**
- privilege mutation (added `DELETE` to a table grant) → `GRANT-INV FAIL`, exit 1
- phantom object (extra application function in manifest) → `GRANT-INV FAIL`, exit 1

### B9 — receipt cited the wrong bytes (CRLF working-tree, not the committed blob)
Prior receipts cited SHA-256 `fdaf5d90…` / 293,691 bytes — that is the **Windows CRLF working-tree representation**, not the repository artifact. The committed Git blob is LF: SHA-256 `ace036c2…` / 283,888 bytes. The 9,803-byte difference is exactly the CRLF expansion. CI (Linux, LF working tree) computed the correct blob digest, but Step 0 only *printed* it without asserting it, so the divergence went unnoticed across four receipts (`c24e997`, `e3397a1`, `759b790`, `ec2f38d`).

**Fix (three parts):**
1. `.gitattributes` pins `migrations/profiles/g2/**` to `eol=lf`, so the working-tree bytes equal the committed blob bytes on every platform.
2. A detached checksum file (`TARGET-SCHEMA-MANIFEST.sha256`) records the authoritative blob digest + size. `regenerate_g2_checksum.py` recomputes it from `git cat-file blob HEAD:<path>` (the repository artifact, not the working-tree file) and warns if the working tree has uncommitted edits.
3. Step 0 now ASSERTS the committed blob's SHA-256 and size against the checksum file (fail-closed), not merely prints them.

**Proven fail-closed in both failure modes:**
- checksum citing the CRLF digest (`fdaf5d90…`) → `BLOB-DIGEST FAIL`, exit 1
- manifest committed but checksum not regenerated → `BLOB-DIGEST FAIL`, exit 1

> Note: B9 bound the *checksum file* to the *committed blob*, but did not yet bind the *receipt* to either. A receipt-only digest edit would still pass B9's checks. That binding is closed in B10 below.

### B10 — receipt not CI-bound
B9's `BLOB-DIGEST` check compared the committed blob only to the detached checksum file; `validate_g2.sh` never read `G2-APPROVALS.md`. Therefore changing only the receipt's digest or size still produced `BLOB-DIGEST PASS` — the claim that "a receipt mismatch fails CI" was false. (The `8884d99` green run also predates the final receipt commit `14ef412`, so the receipt itself had not been CI-tested against the binding.)

**Fix:** `G2-APPROVALS.md` now carries a machine-readable ```g2-receipt-identity fenced block (the single source of truth for the cited identity). Step 0 parses it and asserts three-way agreement: **receipt == checksum file == committed blob**. The committed blob is read via `git cat-file blob HEAD:<path>` (the repository artifact, not the working-tree file).

**Proven fail-closed:** editing only the receipt's `manifest_blob_sha256` field → `RECEIPT-BIND FAIL: receipt identity != blob/checksum (three-way mismatch)`, exit 1, with all three values printed for diagnosis.

### B11 — manifest authority semantics contradicted its own structured fields
The manifest's `decision_029.resolution` prose described `clarityit_migrator` as having "ADMIN on owner." This contradicted the manifest's own `target_memberships` (`admin_option: false, set_option: true`), DECISION-029, and the R3 fixture (which rejects `ADMIN TRUE`). PostgreSQL defines `ADMIN` as membership-delegation authority and `SET` as `SET ROLE` permission — conflating them is security-significant.

Separately, DECISION-029.md and the 029 fixture comment stated every omitted `GRANT` option defaults to `TRUE`. PostgreSQL 16 differs: `ADMIN` defaults `FALSE`, `SET` defaults `TRUE`, and `INHERIT` defaults from the member role's `rolinherit` attribute (empirically verified: no-option GRANT on a `NOINHERIT` member yields `INHERIT=FALSE`).

**Fix:** `decision_029.resolution` corrected to "SET on owner / ADMIN FALSE" with a pointer to `target_memberships` as authoritative. Membership rationales corrected. DECISION-029.md and the 029 fixture comment corrected to state the actual PG16 defaults. The executable grants/memberships SQL and the R3 rejection of `ADMIN TRUE` were already correct; only the prose/rationale was wrong. Manifest bytes changed → identity is now `1f6e3142…` / 284,064.

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

Architecture and Database must approve the exact target-manifest blob digest `1f6e3142…` (284,064 bytes, LF — the committed Git artifact, CI-asserted against `TARGET-SCHEMA-MANIFEST.sha256`); Security reviews the 029 privilege decision. This record is **unsigned** — signatures are recorded separately by the named owners.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Database | | | | |
| Security | | | | |
