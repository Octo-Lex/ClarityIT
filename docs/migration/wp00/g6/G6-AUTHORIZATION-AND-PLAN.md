# WP-00 G6 — Final Acceptance Authorization and Package Plan

**Authorization ID:** `G6-AUTH-2026-08-11`  
**Status:** **AUTHORIZED BY CLIENT · EXECUTION BLOCKED UNTIL G5 ACCEPTANCE**  
**Authorization date:** 2026-08-11  
**Authorization-recording baseline:** `main@a0be44780aa0f486bd6fb1d5fd5d87d26de09001`  
**Implementation / evidence branch after activation:** `wp00/g6-acceptance`

## 1. Decision

The client has explicitly and separately authorized WP-00 G6.

This authorization is durable now. It removes the need for another G6 authorization decision after G5 closes, but it does **not** waive the sequential gate prerequisite. Under the governing WP-00 plan, G5 must be accepted before G6 acceptance work may depend on G5 as an approved input.

Accordingly:

- G6 is **authorized**;
- G6 is **not started** for execution purposes while G5 remains unaccepted;
- no G6 evidence may treat `main@a0be44780aa0f486bd6fb1d5fd5d87d26de09001` as a G5-accepted baseline merely because the G5 workflow implementation is present;
- once the G5 acceptance receipt and authority update are integrated, G6 becomes active automatically under this authorization and may begin from that exact accepted G5 integration tip;
- G6 remains unaccepted until its own frozen criteria pass and the final WP-00 acceptance record is integrated.

This record is authorization only. It does not manufacture missing G5 evidence, configure repository rules, close G5, close issue #1, or accept WP-00.

## 2. Frozen prerequisite

The prerequisite is **accepted G5**, not merely implemented G5.

At the time this authorization is recorded:

| Item | Current observed state |
|---|---|
| G5 workflow implementation | Integrated on `main` as `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| G5 authorization | Recorded by `G5-AUTH-2026-08-10` |
| `docs/migration/wp00/g5/G5-APPROVALS.md` | Not present on `main` |
| G5 final authority update | Not integrated |
| G5 accepted prerequisite | **NOT YET SATISFIED** |

The activation baseline for G6 therefore remains intentionally unresolved until G5 closes. When G5 is accepted, its exact integrated acceptance SHA becomes the sole G6 starting baseline and must be recorded in the G6 receipt/evidence manifest.

## 3. Frozen G6 objective

G6 is WP-00 WS6 — final rehearsal, review, evidence consolidation, and acceptance decision.

The governing WP-00 plan defines the WS6 work as:

1. **WS6-01 — full release-artifact rehearsal.** Rehearse the complete P2 restore, fingerprint, adoption, forward apply, restart, verify, and evidence flow using only release artifacts and the approved runbook.
2. **WS6-02 — failure/recovery rehearsal.** Inject failures before DDL, during a transactional revision, after a committed revision, during lock contention, and during verification; prove the required recovery behavior.
3. **WS6-03 — historical-truth confirmation.** Run the historical-truth fixture and prove synthetic success, operator outcome text, and provider submission references remain weaker classifications rather than authoritative v2 truth.
4. **WS6-04 — final foundation review.** Complete schema, identity, constraint, grant, checksum, secret-scan, and artifact-provenance review.
5. **WS6-05 — release decision.** Publish the WP-00 release evidence manifest, record approval or block, and close issue #1 only if its fresh-install and CI claims are demonstrably resolved.

G6 is an acceptance/evidence gate. It is not an authorization for new product, migration-runner, provider, execution-kernel, Site Runtime, adapter, NATS runtime, host-agent, or UI feature implementation.

## 4. Frozen acceptance rule

WP-00 is accepted only when all of the following are true:

- **AC-00-01 through AC-00-30 are individually evidenced** against the final accepted foundation state;
- every required G6 owner signs the decision;
- no unresolved severity 1 or severity 2 migration, data-integrity, security, retry, or CI foundation defect remains open;
- the final evidence manifest binds the exact accepted G5 baseline and the exact G6 evidence state;
- the acceptance is unconditional.

Conditional or partial WP-00 acceptance is prohibited. If any criterion is not satisfied, G6 records **BLOCKED** with the failed criterion, owner, corrective work, and preserved evidence.

## 5. Evidence contract after activation

Once G5 is accepted, G6 evidence must be bound at minimum to:

- exact accepted G5 integration SHA;
- G5 acceptance receipt and required-status enforcement evidence;
- frozen G1-G4 identities and signed receipts without rewriting them;
- release/runbook identity used for the WS6 rehearsal;
- P2 restore evidence and approved P1/P2 source identity;
- migration CLI producing identity;
- PostgreSQL 16 image identity;
- fresh-install and adoption fingerprints;
- restart/rollback/lock-contention/verification failure results;
- historical-truth fixture result;
- schema, ownership, role, membership, grant, constraint, checksum, and provenance review results;
- secret/evidence-hygiene result;
- AC-00-01 through AC-00-30 cross-reference to concrete evidence;
- severity 1/2 defect disposition;
- issue #1 disposition;
- explicit required-owner decisions;
- final `ACCEPTED` or `BLOCKED` decision.

Evidence must be sanitized and must not place sensitive P1/P2 bytes, credentials, or production secrets into ordinary repository history or CI artifacts.

## 6. Execution order after G5 acceptance

When the G5 prerequisite is satisfied, execute G6 in this order:

1. Re-read the exact G5 acceptance receipt and `PROJECT-COMPLETION-AUTHORITY.md`; bind the accepted G5 integration SHA.
2. Start `wp00/g6-acceptance` from that exact accepted G5 tip.
3. Build an AC-00-01 through AC-00-30 evidence crosswalk from existing signed evidence; do not reopen a closed gate unless current evidence demonstrates an actual defect.
4. Run WS6-01 using only the release artifacts and approved runbook.
5. Run WS6-02 failure/recovery cases and retain deterministic pass/fail markers.
6. Run WS6-03 historical-truth confirmation.
7. Run WS6-04 final schema/security/provenance review.
8. Inspect open severity 1/2 foundation defects and issue #1 against the proven final state.
9. Publish the WP-00 release evidence manifest and final G6 receipt.
10. Record the required owner decisions.
11. Update `docs/v2/PROJECT-COMPLETION-AUTHORITY.md` in the same final acceptance integration.
12. Accept WP-00 only if every frozen criterion passes; otherwise record G6 as blocked and stop.

## 7. Stop conditions

Stop and report G6 as blocked rather than expanding scope if any of the following occurs:

- G5 is not accepted when G6 execution is about to begin;
- any AC-00 criterion lacks evidence or fails;
- a release-artifact-only rehearsal requires manual database correction or an unapproved artifact;
- a failure/recovery case leaves duplicate, partial, drifted, or unverifiable state;
- historical truth is promoted beyond its frozen weak classification;
- final schema/identity/grant/checksum/provenance review detects unexplained drift;
- sensitive evidence or credentials would have to enter ordinary CI/repository history;
- a severity 1 or 2 foundation defect remains unresolved;
- a frozen signed G1-G4 input would need mutation rather than a governed successor decision.

Routine uncertainty is not a blocker; use bounded reversible evidence gathering. Do not invent new acceptance gates.

## 8. Explicit exclusions

This G6 authorization does **not** authorize:

- bypassing or deeming G5 accepted without its frozen receipt/enforcement closure;
- provider mutation or provider credentials;
- production migration, cutover, rollback, or deployment;
- Site Runtime, host agents, generic/provider adapters, NATS execution-runtime work, or broader UI implementation;
- WP-01 through WP-10 implementation;
- modification, deletion, force-push, or rewriting of signed G1-G4 evidence or recovery references;
- conditional or partial WP-00 acceptance.

The WP-00 plan states that G6 acceptance is the prerequisite for successor implementation packages. G6 acceptance itself does not silently authorize those packages; their separate approved contracts still govern their implementation.

## 9. Current decision

**G6 is authorized by the client.**

**Execution is blocked solely on the sequential prerequisite: G5 must first be accepted and integrated.**

Once that prerequisite is satisfied, this authorization activates automatically and the next permitted action is bounded WS6/G6 acceptance work from the exact accepted G5 integration tip.
