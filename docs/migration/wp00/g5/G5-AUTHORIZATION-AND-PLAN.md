# WP-00 G5 — Blocking CI Matrix Authorization and Package Plan

**Authorization ID:** `G5-AUTH-2026-08-10`  
**Status:** **AUTHORIZED TO IMPLEMENT · NOT ACCEPTED**  
**Authorization date:** 2026-08-10  
**Authorized baseline:** `main@ecb0ea48eb67bc07371b72e11517a77ad802d465`  
**Implementation branch:** `wp00/g5-blocking-ci`

## 1. Decision

G5 implementation is authorized from the exact G4-accepted baseline above. The authorization record must be integrated before implementation begins; the implementation branch starts from that integrated authorization commit, preserving the authorized baseline as its parent state.

This is a bounded authorization for WP-00 WS5 and the existing WP-00 CI requirements only. It does not close G5, infer acceptance from workflow presence or a single green run, authorize G6, or authorize any provider, Site Runtime, execution-kernel, or WP-01+ work.

The project authority requested one delegated assessment across the required roles. The decisions below are therefore one transparent role-based authorization, not independent human attestations.

| Assumed role | Decision | Bound responsibility |
|---|---|---|
| Backend | **AUTHORIZE** | Implement deterministic blocking CI integration without changing accepted G4 runner semantics |
| Quality | **AUTHORIZE** | Require the complete WP-00 foundation matrix and fail-closed aggregation |
| Security | **AUTHORIZE** | Preserve secret/evidence hygiene and security-relevant fail-closed checks |

## 2. Frozen prerequisite and inputs

G4 is the accepted prerequisite and must not be reopened without demonstrated defect evidence.

| Input | Frozen value |
|---|---|
| G4 acceptance tip | `ecb0ea48eb67bc07371b72e11517a77ad802d465` |
| G4 exact integrated implementation/proof tip | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| PR #16 implementation squash | `f769cd3815ea08194b56c267cfa3b30fb7a12fd9` |
| G4 Linux proof run | `31336112238` — 11/11 PASS |
| PostgreSQL image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` |
| Product manifest blob SHA-256 | `1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63` |
| Composite installation SHA-256 | `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084` |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| P1/P2 source fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| Baseline SQL checksum | `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508` |

G5 may consume and verify these values but must not redefine, regenerate, or silently replace them.

## 3. G5 objective

G5 converts the accepted WP-00 foundation into a mandatory, fail-closed CI merge contract.

A change targeting `main` must not satisfy the G5 foundation gate while any required backend, migration, restart/locking, poison-event, historical-truth, or artifact/evidence check is failing or bypassed.

G5 is CI-governance work. It is not another migration-runner feature phase.

## 4. Frozen required CI classes

The existing WP-00 contract defines eight required CI check classes. G5 does not add a ninth acceptance criterion.

| Check class | Required result |
|---|---|
| `backend-static` | gofmt/vet/build and migration CLI build succeed on the exact candidate SHA; no tolerated failure |
| `db-fresh` | clean PostgreSQL 16 fresh installs converge reproducibly; supported reapply/no-op behavior; no legacy replay |
| `db-adopt-p3` | approved P3 source adopts through the Go runner and converges to the governed target without manual SQL or legacy replay |
| `db-negative` | unknown/drifted source, checksum mutation, and required fail-closed preflight cases reject before prohibited mutation |
| `db-restart-lock` | one writer, advisory-lock contention, transactional rollback/rerun, no duplicate revision or partial state; non-transactional path explicit or absent |
| `context-dead-letter` | poison-event retry is bounded and terminal disposition remains durably visible before terminal acknowledgement |
| `backend-integration` | existing blocking Go application suite and its P0 test fixture remain green; P0 is not migration authority |
| `artifact-audit` | frozen/checksummed inputs, receipt binding, sanitized evidence, and secret hygiene pass |

The accepted G4 11-row matrix remains a required regression oracle inside G5 evidence; G5 does not replace or reinterpret it.

## 5. Historical-truth safeguard

Before G5 can close, blocking CI must preserve the existing WP-00 classification contract:

- synthetic legacy success maps at most to `legacy_unverified`;
- provider task identity without terminal proof maps at most to `legacy_submitted_unverified`;
- ambiguous/executing legacy action maps to `legacy_outcome_unknown`;
- legacy approval is evidence only, never an AuthorityGrant;
- operator outcome text is annotation/claim only, never Verification or Accepted outcome.

G5 may add or strengthen only the bounded fixture/test necessary to prove this already-existing WP-00 requirement. G5 performs no historical backfill.

## 6. Authorized implementation scope

Once this authorization is integrated, the G5 implementation branch may change only:

1. GitHub Actions workflow configuration necessary to implement the blocking WP-00 foundation gate.
2. Narrow CI scripts supporting those workflows.
3. Test/fixture code required to satisfy an already-existing WP-00 CI scenario.
4. Repository required-status configuration necessary to make the accepted G5 result mandatory for `main`.
5. G5 evidence and receipt documentation.
6. `docs/v2/PROJECT-COMPLETION-AUTHORITY.md` only when G5 is accepted or blocked.

Preferred mechanism: add `.github/workflows/g5-foundation.yml`, preserving `.github/workflows/g4-proof.yml` as historical G4 proof infrastructure. The new workflow should expose one final fan-in context, `G5 Foundation Gate`, while the existing `Backend (Go)` application gate remains separately required.

The intended effective merge predicate is:

```text
Backend (Go) == SUCCESS
AND
G5 Foundation Gate == SUCCESS
```

This mechanism may be adjusted if GitHub requires a technically equivalent implementation, but the frozen property is the fail-closed conjunction of the required WP-00 checks.

## 7. Repository enforcement boundary

G5 authorizes only the minimum repository rule change necessary to require the defined foundation status contexts for `main` after the exact integrated G5 candidate has passed.

G5 does not authorize unrelated repository-hardening controls such as:

- mandatory reviewer counts;
- CODEOWNERS enforcement;
- signed-commit requirements;
- linear-history requirements;
- deployment approvals;
- unrelated branch restrictions.

If GitHub cannot enforce the required statuses with the available repository capabilities, that is a G5 blocker to report; do not substitute unrelated governance.

## 8. Evidence and exact-commit contract

The exact G5 candidate must record and prove:

- exact Git SHA;
- workflow identity/revision;
- Ubuntu runner identity;
- Go and Python versions;
- PostgreSQL image digest;
- migration CLI producing SHA;
- all eight required CI classes;
- all eleven G4 regression rows;
- historical-truth safeguard result;
- artifact/evidence hygiene result;
- final aggregation result;
- evidence artifact IDs and digests where retained.

A machine-readable summary such as `g5-results.json` may be added as an evidence mechanism. It is not a new acceptance criterion.

## 9. Commit, integration, and review contract

1. Integrate this authorization record before G5 implementation begins.
2. Start `wp00/g5-blocking-ci` from the integrated authorization commit descended directly from the authorized baseline.
3. Keep authorization history separate from implementation evidence.
4. Implement the G5 workflow and only bounded supporting scripts/tests; do not alter G4 receipt/evidence.
5. Prove the complete G5 suite on the exact candidate SHA before making repository required-status changes.
6. Integrate the workflow implementation and capture the resulting exact `main` SHA.
7. Run ordinary CI and the G5 foundation gate against that exact integrated `main` SHA.
8. Only after both pass, configure the minimum repository required-status enforcement for `Backend (Go)` and `G5 Foundation Gate` (or a technically equivalent frozen conjunction).
9. Verify the repository reports the required contexts as enforced.
10. Create a receipt-only `docs/migration/wp00/g5/G5-APPROVALS.md` change binding implementation/integration SHA, proof runs, workflow identity, PostgreSQL image, evidence artifacts, required-check results, G4 regression result, enforcement state, and explicit Backend/Quality/Security decisions.
11. Update `docs/v2/PROJECT-COMPLETION-AUTHORITY.md` in the same acceptance integration.
12. G5 is accepted only when every frozen requirement passes and the receipt/authority integration is complete.

## 10. Stop conditions and blockers

Stop and report without expanding scope if any of the following is observed:

- a required G5 check fails on the exact candidate or integrated SHA;
- a required path can fail while its required status remains green;
- `continue-on-error` or equivalent tolerated failure exists on a required foundation path;
- an accepted G4 behavior is demonstrated to be incorrect;
- sensitive P1/P2 bytes or credentials enter ordinary CI/evidence;
- historical truth can be promoted beyond the frozen classification contract;
- GitHub cannot enforce the required foundation statuses;
- a frozen G1-G4 input would need modification.

Uncertainty, stylistic CI refactoring, broader branch governance, or desire for additional feature coverage is not a blocker.

## 11. Explicit exclusions

This authorization does **not** authorize:

- G6 WP-00 acceptance or G6 evidence decision;
- provider mutation or provider credentials;
- production migration, cutover, rollback, or deployment;
- Site Runtime, host agents, adapters, NATS execution-runtime work, or UI feature work;
- WP-01 through WP-10 implementation;
- migration-runner feature expansion unrelated to a demonstrated G5 blocker;
- modification, deletion, force-push, or rewriting of signed G1-G4 evidence or recovery references.

## 12. Current decision

**G5 is authorized to implement and remains unaccepted.**

The next permitted action after this authorization is integrated is bounded G5 implementation on `wp00/g5-blocking-ci`.

G6 remains **not started and unauthorized**. Successful G5 closure will make G6 separately governable; it will not authorize G6 automatically.
