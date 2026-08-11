# WP-00 G6 — AC-00 Evidence Crosswalk

**Authorization:** `G6-AUTH-2026-08-11`  
**Corrective authorization:** `G6-P2-SUCCESSOR-AUTH-2026-08-11`  
**G6 starting baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce`  
**Status:** **G6 ACTIVE · P2 SUCCESSOR CORRECTION AUTHORIZED · REPEAT EVIDENCE REQUIRED**  
**Date:** 2026-08-11

This crosswalk is a G6 working evidence record. It does not itself accept WP-00. A criterion is marked `PASS-PRESERVED` only where prior signed/accepted evidence still establishes the property and G6 has found no contradictory evidence that invalidates that property. `PENDING-FINAL` requires final-commit or A7 closure evidence. `BLOCKED-CORRECTIVE` identifies a demonstrated defect that must be corrected under the separately authorized bounded successor before acceptance can continue.

## 1. Bound inputs

| Input | Bound identity |
|---|---|
| Accepted G5 baseline | `dc366eadede4556615dd5d3977c35cceae43dcce` |
| G5 implementation squash | `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| G4 implementation/proof tip | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| G4 Linux proof | Actions run `31336112238` — 11/11 PASS |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Historical G1 P1/P2 source fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` — immutable custody identity, profiler v3.1.0 |
| Candidate v3.2 P1/P2 successor fingerprint | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` — observed, not frozen until repeat evidence passes |
| Approved P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| Approved P2 operational backup reference | `opbak-20260731-173628` |
| Approved P2 operational backup SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| P2a custody manifest version ID | `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1` |
| P2a custody manifest SHA-256 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |
| Accepted v3.2 profiler Git blob | `731324aabbe049dc5278f3cedc49bf8980c5f5e5` |
| PostgreSQL proof image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` |

Sensitive P1/P2 bytes remain outside Git and ordinary CI. This record contains references, digests, and sanitized diagnostic facts only.

## 2. G6 P2 diagnostic finding

The approved backup was recovered from custody and its SHA-256 matched exactly. A clean PostgreSQL 16.14 restore completed without manual DDL/data correction.

The immutable P2a custody manifest established that historical P1/P2 fingerprint `89b7792d...` was produced with profiler `3.1.0-p1p2`, despite repository G1 prose stating `3.2.0-p1p2`.

A fresh capture of the same restored database with the accepted `3.2.0-p1p2` profiler produced candidate `57c2b645...`. After removing the accepted fingerprint-excluded fields, the stable manifests differed at exactly one path: `profiler_version`. The reported schema/catalog, roles/grants, extensions, migration state, and fingerprinted PostgreSQL settings were otherwise identical.

The accepted Go runner therefore correctly returns `SOURCE_PROFILE_UNKNOWN` before DDL because its v3.2 profiler computes `57c2b645...` while the historical recognized identity is `89b7792d...`.

See `G6-P2-PROFILER-DRIFT-DIAGNOSTIC.md` and `G6-P2-SUCCESSOR-AUTHORIZATION.md`.

## 3. AC-00-01 through AC-00-30

| Criterion | G6 state | Evidence / remaining action |
|---|---|---|
| AC-00-01 | PASS-PRESERVED | WP-00 source/freeze authority and repository history bind the historical source and revision starting baseline. The profiler-version metadata contradiction is carried as successor evidence rather than silently rewriting G1. |
| AC-00-02 | PASS-PRESERVED | G0/G1 freeze and deployed-artifact reconciliation remain preserved; no executable source/binary mismatch was demonstrated by the P2 diagnostic. |
| AC-00-03 | PASS-PRESERVED | Durable context-worker poison-event disposition is implemented; G5 required `Backend (Go)` retained the context-worker tests. |
| AC-00-04 | PASS-PRESERVED | No later evidence identifies a file-copy-only semantic delta outside the frozen source. |
| AC-00-05 | PASS-PRESERVED | Immutable G1 P2a/P2b evidence established deterministic captures under the historical profiler. The v3.2 successor requires its own deterministic repeat evidence before freezing. |
| AC-00-06 | PASS-PRESERVED | G1 profile evidence remains structurally valid; G6 found the profiler-version metadata/provenance contradiction but no schema/catalog divergence in the restored P2 source. |
| AC-00-07 | PASS-PRESERVED | Approved P2 backup was recovered from immutable custody, digest-verified, and restored cleanly in isolation. |
| AC-00-08 | PASS-PRESERVED | P3 deterministic fixture and golden fingerprint accepted and continuously validated by CI. |
| AC-00-09 | PASS-PRESERVED | G4/G5 negative-profile proof and the live G6 P2 attempt both demonstrate fail-closed rejection before DDL for an unrecognized fingerprint. |
| AC-00-10 | PASS-PRESERVED | G2 signed target decisions cover 016/018/029. |
| AC-00-11 | PASS-PRESERVED | Legacy 001-040 checksum inventory remains immutable and is audited in G5. |
| AC-00-12 | PASS-PRESERVED | G4-10/G5 prove the supported runner cannot select legacy 001-040; no legacy replay occurred in the blocked P2 rehearsal. |
| AC-00-13 | PASS-PRESERVED | G4-01/G5 database matrix prove deterministic fresh-install A/B convergence to the governed target. |
| AC-00-14 | **BLOCKED-CORRECTIVE** | Real P2 restore is available and structurally reproduced, but the accepted v3.2 runner cannot classify the historical v3.1 identity. Corrective successor `57c2b645...` is authorized but must be repeat-verified/frozen before classifier change; then the full P2 adoption/restart/verify rehearsal must pass with no manual correction. |
| AC-00-15 | PASS-PRESERVED | G2/G3 target manifest and G4 privilege-boundary proof verify explicit ownership/grants. |
| AC-00-16 | PASS-PRESERVED | G4-05 advisory-lock contention proof. |
| AC-00-17 | PASS-PRESERVED | G4 ledger evidence binds immutable successful-revision metadata. |
| AC-00-18 | PASS-PRESERVED | G4-04 checksum-mutation rejection. |
| AC-00-19 | PASS-PRESERVED | G4-06 transactional failure/rerun proof. |
| AC-00-20 | PASS-PRESERVED | G4-07 proves the non-transactional path is absent; no unproven non-transactional step exists. |
| AC-00-21 | PASS-PRESERVED | G4-08 verify-mode drift/revision/fingerprint coverage. |
| AC-00-22 | PASS-PRESERVED | G4-09 privilege boundary plus accepted runner scope prove no provider credential/target/effect path. |
| AC-00-23 | PASS-PRESERVED | `Backend (Go)` is blocking and contains no `continue-on-error`/non-blocking designation. |
| AC-00-24 | **PENDING-FINAL** | G5 ruleset requires Frontend, Worker, Backend, and G5 Foundation Gate; final G6 integration commit must itself be green under that ruleset. |
| AC-00-25 | PASS-PRESERVED | G4/G5 fresh install and P3 adoption use clean PostgreSQL instances and deterministic fingerprints. G6 P2 successor freeze additionally requires deterministic v3.2 repeat evidence from the approved backup lineage. |
| AC-00-26 | PASS-PRESERVED | G5 historical-truth fixture: 5/5, zero authoritative promotions. |
| AC-00-27 | PASS-PRESERVED | Required Backend suite covers bounded poison-event retry, durable terminal disposition, and replay/operator visibility. |
| AC-00-28 | PASS-PRESERVED | G4/G5 evidence-hygiene and artifact audit passed; G6 P2 diagnostics supplied only sanitized references/digests. Final A7 must repeat release-evidence hygiene review. |
| AC-00-29 | **PENDING-FINAL** | A7 does not yet exist. Final reconstruction must include the historical v3.1 identity, v3.2 successor decision, corrective implementation/proof, and all prior A1-A6 evidence without rewriting G1. |
| AC-00-30 | **PENDING-FINAL** | The demonstrated P2 classifier identity defect is unresolved. Final Sev-1/Sev-2 defect review cannot close until the authorized successor correction and P2 rehearsal complete successfully. |

## 4. WS6 execution state

| WS6 item | State | Notes |
|---|---|---|
| WS6-01 full release-artifact rehearsal | **BLOCKED-CORRECTIVE** | Backup retrieval/restore pass. Rehearsal stopped correctly at preflight because the v3.2 runner computed `57c2b645...` and only historical v3.1 `89b7792d...` is recognized for P1/P2. Resume after successor freeze/classifier correction. |
| WS6-02 failure/recovery rehearsal | PARTIALLY PRESERVED | Pre-DDL rejection is directly demonstrated on P2; transactional failure, lock contention, verification drift and rerun properties remain preserved by G4/G5. Full P2 post-apply restart/recovery remains pending. |
| WS6-03 historical-truth confirmation | PASS-PRESERVED | G5 required historical-truth job passed and remains blocking. |
| WS6-04 final foundation review | **CORRECTIVE IN PROGRESS** | Immutable custody evidence established the G1 profiler-version provenance contradiction. Historical evidence remains immutable and the correction is handled by governed successor. |
| WS6-05 release decision | PENDING | A7 and final decisions cannot be issued until the corrective successor and full P2 rehearsal close. |

## 5. Authorized corrective path

`G6-P2-SUCCESSOR-AUTH-2026-08-11` authorizes a bounded successor:

- preserve `89b7792d...` as historical G1/v3.1 recognized/non-executable identity;
- repeat-verify and freeze the v3.2 successor only if the approved P2 lineage deterministically reproduces `57c2b645...`;
- update only source-classification constants/mapping/tests necessary to recognize the frozen v3.2 successor;
- do not alter frozen adoption SQL or migration semantics;
- if the existing adoption contract rejects real P2 after classification, stop for a new governed decision;
- preserve all G4/G5 regression and fail-closed properties;
- rerun the complete P2 adoption/restart/verify rehearsal.

### Immediate evidence prerequisite

Before changing the runner allowlist, obtain deterministic repeat evidence using exact accepted profiler blob `731324aabbe049dc5278f3cedc49bf8980c5f5e5` against the approved restored P2 source. Two unchanged v3.2 captures must both equal:

`57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`

Until that evidence is recorded, the candidate remains observed but unfrozen and the classifier must not change.

Until the corrective successor and full P2 rehearsal pass, **G6 remains active but BLOCKED and WP-00 is not accepted**.
