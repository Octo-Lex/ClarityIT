# WP-00 G4 — Go Migration Runner Authorization and Package Plan

**Authorization ID:** `G4-AUTH-2026-08-05`
**Status:** **AUTHORIZED TO IMPLEMENT · NOT ACCEPTED**
**Authorization date:** 2026-08-05
**Authorized baseline:** `wp00/g2-schema-decisions@ac7222737e14796174ed78420f1f388e6c21170b`
**Implementation branch:** `wp00/g4-migration-runner`

## 1. Decision

G4 implementation may begin from the exact authorized baseline above. This is a
bounded authorization for WP-00 WS4 and AC-00-16 through AC-00-22 only. It does
not close G4, infer acceptance from code or CI, or authorize G5 or G6.

The project authority requested one delegated assessment across the required
roles. The decisions below are therefore one transparent role-based assessment,
not independent human attestations.

| Assumed role | Decision | Bound responsibility |
|---|---|---|
| Product | **AUTHORIZE** | Preserve WP-00 scope and exclusions; no product-feature expansion |
| Architecture | **AUTHORIZE** | Preserve the signed G2/G3 authority hierarchy, identities, and no-legacy-replay rule |
| Database | **AUTHORIZE** | Own runner database behavior, source-profile checks, locking, ledger, adoption, restart, and verification |
| Backend | **AUTHORIZE** | Implement the Go command, packaged migration bytes, deterministic diagnostics, and tests |
| Operations | **AUTHORIZE** | Development and isolated-test execution only; no production mutation or cutover |
| Security | **AUTHORIZE** | No provider credentials, target-system access, raw production data, or secret-bearing evidence |
| Quality | **AUTHORIZE** | Require the G4 test/evidence matrix; do not claim the separately governed G5 blocking-CI gate |

## 2. Frozen inputs

The runner must consume the following inputs without changing their bytes or
meaning:

| Input | Frozen value |
|---|---|
| G3 signed tip | `97f83e4ac0609994b64493c7a8b2b76208545bb1` |
| G3 producing implementation | `570a0ec7e31087d1dd6db22e14935e21e7481cf6` |
| Product manifest blob SHA-256 | `1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63` |
| Control manifest SHA-256 | `3fd65e917ded8b7d59a1f42051b69f41e4b5c24f583f9524deaccdfdfb1add66` |
| Composite installation SHA-256 | `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084` |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| P3 adoption artifact SHA-256 | `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359` |
| P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| P1/P2 source fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| Runtime target | PostgreSQL 16; database `clarityit` |
| Legacy migrations | `migrations/legacy/v1/001-040/`; immutable provenance, never selectable by the runner |

Any required change to a frozen input stops G4 and requires a governed successor
decision with new evidence.

## 3. Authorized implementation scope

The G4 branch may change only the runner, its migration packaging, its direct
tests/evidence, the Makefile migration entry point, and G4 documentation:

1. Add a Go migration command under `services/api/cmd/` and a bounded internal
   migration package under `services/api/internal/`.
2. Package the exact G3 bootstrap, baseline, seed, adoption, manifests, and any
   approved forward-revision bytes into the release artifact. Runtime selection
   must be checksum-bound and deterministic.
3. Provide `plan`, `apply`, `status`, and read-only `verify` behavior with stable
   machine-readable diagnostics and non-zero failure exits.
4. Validate source profile, application compatibility, PostgreSQL major,
   required extensions and roles, evidence reference, and every packaged
   checksum before DDL.
5. Acquire a PostgreSQL advisory lock before any migration-control or target
   mutation and reject concurrent writers deterministically.
6. Execute transactional revisions atomically. Any unavoidable
   non-transactional step requires explicit preconditions, postconditions,
   restart state, and failpoint evidence.
7. Record immutable revision, checksum, source commit, actor, run, duration,
   result, target fingerprint, and sanitized evidence reference.
8. Replace the Makefile's direct `psql` file loop with the supported Go runner
   only after the replacement passes the full G4 matrix.
9. Keep the runner free of provider credentials, provider clients,
   target-system access, effect dispatch, and general application authority.

The implementation may add a local G4 proof workflow for exact-commit evidence.
Making the complete backend matrix blocking or changing branch protection is G5
and remains unauthorized.

## 4. Required G4 evidence matrix

The exact implementation tip must prove all of the following from clean,
isolated PostgreSQL 16 instances:

| Evidence | Required result |
|---|---|
| Fresh install A/B | Two independent installs are byte- and fingerprint-equivalent to the frozen G3 target |
| Approved P3 adoption | Adoption converges to the governed target without legacy replay or manual edits |
| Unknown/drifted source | Fails before DDL with a stable diagnostic |
| Packaged checksum mutation | Rejected before execution; a succeeded checksum is immutable |
| Advisory-lock contention | Exactly one writer proceeds; competitors fail deterministically without partial state |
| Transaction failure and rerun | No partial mutation; restart resumes without duplicate application |
| Non-transactional failpoints | Explicit pre/post state and restart behavior are proven, or the path is absent |
| Verify mode | Detects missing/extra objects, constraint/index/grant drift, revision mismatch, and fingerprint mismatch without mutation |
| Privilege boundary | Bootstrap and application roles remain separated; runner cannot invoke a product effect |
| Legacy exclusion | No supported command can select or execute migrations `001-040` |
| Evidence hygiene | Logs and artifacts contain no credentials, raw production data, or sensitive P1/P2 content |

Unit tests alone do not pass G4. The receipt must bind the implementation commit,
packaged artifact digests, PostgreSQL image identity, test commands, run IDs,
result markers, and frozen identities.

## 5. Commit and review contract

1. Start `wp00/g4-migration-runner` from the integrated commit containing this
   authorization.
2. Keep the authorization/plan commit separate from implementation evidence.
3. Produce an implementation commit containing runner code and the complete G4
   test matrix.
4. Run the matrix against that exact commit on Linux and retain sanitized logs
   and immutable references.
5. Add a receipt-only `docs/migration/wp00/g4/G4-APPROVALS.md` commit. It must
   identify the producing commit, evidence, frozen identities, and explicit
   Database and Backend decisions.
6. G4 may be marked **accepted** only when both Database and Backend approval
   rows are present and every required matrix result passes. A role-based review
   must be labeled accurately; it must not be presented as independent human
   signatures when it is not.
7. Update `docs/v2/PROJECT-COMPLETION-AUTHORITY.md` in the same integration that
   closes or blocks G4.

## 6. Stop conditions and exclusions

Stop and report without expanding scope if any frozen identity changes, the
runner would replay legacy SQL, a source profile is unknown, DDL could begin
before preflight, locking or restart behavior is ambiguous, evidence contains
sensitive material, or the target cannot be reproduced from packaged bytes.

This authorization does **not** authorize:

- G5 blocking-CI or branch-protection changes;
- G6 WP-00 acceptance;
- provider mutation, production migration, cutover, rollback, or deployment;
- Site Runtime, host agents, typed provider adapters, NATS runtime work, or UI;
- WP-01 through WP-10; or
- modification, deletion, force-push, or rewriting of signed G1-G3 evidence or
  recovery references.

## 7. Current decision

**G4 is authorized to start and remains unaccepted.** The next permitted action
is implementation planning and runner work on `wp00/g4-migration-runner` within
this document. G5 remains blocked until a separately recorded G4 acceptance.
