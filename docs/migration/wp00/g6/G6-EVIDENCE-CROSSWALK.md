# WP-00 G6 — AC-00 Evidence Crosswalk

**Authorization:** `G6-AUTH-2026-08-11`  
**Corrective authorization:** `G6-P2-SUCCESSOR-AUTH-2026-08-11`  
**Terminal closure:** `G6-TERMINAL-CLOSURE-AUTH-2026-08-11`  
**G6 starting baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce`  
**G6 integrated implementation tip:** `0d0d842c088284d54abe7fd56df9d6ebf63a7e66`  
**G6 final receipt tip:** `b67d63720aa3fc2231d2d221d06ccb58d7fc09a0`  
**Status:** **G6 ACCEPTED · WP-00 ACCEPTED**  
**Date:** 2026-08-11

This crosswalk is the final G6 evidence record. All 30 criteria are PASS. The corrective successor (`57c2b645…`) was frozen, the P2 adoption artifact was generated, and the real P2 rehearsal converged to `9881c93e…` without manual correction.

## 1. Bound inputs

| Input | Bound identity |
|---|---|
| Accepted G5 baseline | `dc366eadede4556615dd5d3977c35cceae43dcce` |
| G5 implementation squash | `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| G4 implementation/proof tip | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| G4 Linux proof | Actions run `31336112238` — 11/11 PASS |
| G6 implementation tip (column-order fix) | `0d0d842c088284d54abe7fd56df9d6ebf63a7e66` |
| G6 final receipt tip | `b67d63720aa3fc2231d2d221d06ccb58d7fc09a0` |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Historical G1 P1/P2 source fingerprint (v3.1) | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` — recognized, non-executable |
| Frozen v3.2 P1/P2 successor fingerprint | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` — executable |
| Approved P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| Approved P2 operational backup reference | `opbak-20260731-173628` |
| Approved P2 operational backup SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| P2a custody manifest version ID | `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1` |
| P2a custody manifest SHA-256 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |
| Accepted v3.2 profiler Git blob | `731324aabbe049dc5278f3cedc49bf8980c5f5e5` |
| PostgreSQL proof image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` |

Sensitive P1/P2 bytes remain outside Git and ordinary CI. This record contains references, digests, and sanitized diagnostic facts only.

## 2. P2 rehearsal summary

The approved backup was recovered from custody and its SHA-256 matched exactly. Two independent clean PostgreSQL 16.14 restores produced deterministic v3.2 captures of `57c2b645…`. The P2 adoption artifact (`0001_adopt_p2.sql`, SHA `78af6a7a…`) was generated from the unchanged P3 artifact through count-checked transformations. The real P2 rehearsal applied the artifact through the supported Go runner and converged to `9881c93e…`. A PostgreSQL restart confirmed persistence: the governed fingerprint remained `9881c93e…`.

The profiler-version erratum (`89b7792d…` v3.1 → `57c2b645…` v3.2) and the column-order canonicalization erratum (governed projection sorts signed product-table columns by name) are recorded in `G6-APPROVALS.md`.

## 3. AC-00-01 through AC-00-30

| Criterion | G6 state | Evidence |
|---|---|---|
| AC-00-01 | **PASS** | WP-00 source/freeze authority and repository history bind the historical source and revision starting baseline. |
| AC-00-02 | **PASS** | G0/G1 freeze and deployed-artifact reconciliation preserved; no executable source/binary mismatch demonstrated. |
| AC-00-03 | **PASS** | Durable context-worker poison-event disposition implemented; G5 `Backend (Go)` retained context-worker tests. |
| AC-00-04 | **PASS** | No file-copy-only semantic delta outside the frozen source identified. |
| AC-00-05 | **PASS** | v3.2 successor fingerprint `57c2b645…` deterministically reproduced by two independent restores using accepted profiler blob `731324aabbe0…`. |
| AC-00-06 | **PASS** | G1 profile evidence structurally valid; profiler-version provenance contradiction resolved by governed successor. |
| AC-00-07 | **PASS** | Approved P2 backup recovered from immutable custody, digest-verified, restored cleanly in isolation. |
| AC-00-08 | **PASS** | P3 deterministic fixture and golden fingerprint accepted and continuously validated by CI. |
| AC-00-09 | **PASS** | G4/G5 negative-profile proof and G6 P2 rehearsal demonstrate fail-closed rejection before DDL. |
| AC-00-10 | **PASS** | G2 signed target decisions cover 016/018/029. |
| AC-00-11 | **PASS** | Legacy 001-040 checksum inventory immutable and audited by G5. |
| AC-00-12 | **PASS** | G4-10/G5 prove runner cannot select legacy 001-040; no legacy replay in P2 rehearsal. |
| AC-00-13 | **PASS** | G4-01/G5 prove deterministic fresh-install A/B convergence to `9881c93e…`. |
| AC-00-14 | **PASS** | Real P2 restore → v3.2 fingerprint `57c2b645…` → PathAdoptP2 → apply → governed `9881c93e…`. No manual correction. |
| AC-00-15 | **PASS** | G2/G3 target manifest and G4 privilege-boundary proof verify explicit ownership/grants. |
| AC-00-16 | **PASS** | G4-05 advisory-lock contention proof. |
| AC-00-17 | **PASS** | G4 ledger evidence binds immutable successful-revision metadata. |
| AC-00-18 | **PASS** | G4-04 checksum-mutation rejection. |
| AC-00-19 | **PASS** | G4-06 transactional failure/rerun proof. |
| AC-00-20 | **PASS** | G4-07 proves non-transactional path absent; no unproven non-transactional step exists. |
| AC-00-21 | **PASS** | G4-08 verify-mode drift/revision/fingerprint coverage. |
| AC-00-22 | **PASS** | G4-09 privilege boundary proves no provider credential/target/effect path. |
| AC-00-23 | **PASS** | `Backend (Go)` blocking; no `continue-on-error`/non-blocking designation. |
| AC-00-24 | **PASS** | G5 ruleset requires Frontend, Worker, Backend, G5 Foundation Gate; all green on `b67d637`. |
| AC-00-25 | **PASS** | Fresh install, P3 adoption, and P2 adoption all converge to `9881c93e…` from clean PG16 instances. |
| AC-00-26 | **PASS** | G5 historical-truth fixture: 5/5, zero authoritative promotions. |
| AC-00-27 | **PASS** | Required Backend suite covers bounded poison-event retry, durable terminal disposition, replay/operator visibility. |
| AC-00-28 | **PASS** | G4/G5 evidence-hygiene and artifact audit passed; G6 P2 evidence supplied only sanitized references/digests. |
| AC-00-29 | **PASS** | A7 release manifest committed (see section 5); all A1-A6 evidence reconstructed without rewriting G1. |
| AC-00-30 | **PASS** | Zero unresolved Sev1/Sev2 defects. Profiler-version drift and column-order divergence resolved as governed errata. |

## 4. WS6 execution state

| WS6 item | State | Notes |
|---|---|---|
| WS6-01 full release-artifact rehearsal | **PASS** | P2 backup → restore → v3.2 fingerprint → PathAdoptP2 → apply → `9881c93e…` → restart → `9881c93e…`. No legacy replay, no manual DDL. |
| WS6-02 failure/recovery rehearsal | **PASS** | Pre-DDL rejection directly demonstrated on P2 (before v3.2 freeze). G4/G5 preserved transactional failure, lock contention, verification drift, and rerun properties. Post-apply restart/recovery proven: governed fingerprint persisted through restart. |
| WS6-03 historical-truth confirmation | **PASS** | G5 required historical-truth job passed and remains blocking. |
| WS6-04 final foundation review | **PASS** | Profiler-version erratum and column-order canonicalization recorded as governed implementation errata. All frozen G1-G5 identities preserved. |
| WS6-05 release decision | **PASS** | A7 committed; 7 role-based approval decisions recorded in G6-APPROVALS.md. |

## 5. A1-A7 reconstruction

| Artifact | Evidence |
|---|---|
| A1 — Source freeze | G0/G1: repository history, P1/P2/P3 custody manifests, profiler v3.1 and v3.2 fingerprints. |
| A2 — Schema decisions | G2: signed target manifest (SHA `1f6e3142…`), decisions 016/018/029, five-role posture. |
| A3 — Reconciled baseline | G3: producing commit `570a0ec…`, signed tip `97f83e4…`, baseline checksum `1021adef…`, composite `8af2c9f5…`. |
| A4 — Go migration runner | G4: squash `f769cd3…`, Linux proof run `31336112238`, 11/11 evidence rows. CLI, plan/status/verify, PathAdoptP3, PathAdoptP2, privilege denylist, proof-tagged failpoints. |
| A5 — Blocking CI matrix | G5: integrated `dc366ea…`, required Frontend + Worker + Backend + G5 Foundation Gate on `main`. |
| A6 — P2 successor correction | G6: v3.2 fingerprint `57c2b645…` frozen; P2 artifact `0001_adopt_p2.sql`; column-order canonicalization; profiler-version erratum recorded. |
| A7 — WP-00 release manifest | Integrated tip `0d0d842…`, final receipt `b67d637…`, governed target `9881c93e…` proven from fresh install, P3 adoption, and P2 adoption. Issue #1 closed. |

## 6. Sev1/Sev2 defect disposition

| Defect | Severity | Status | Resolution |
|---|---|---|---|
| Profiler-version drift (v3.1 custody vs v3.2 runner) | Sev2 | **Resolved** | Governed successor: `57c2b645…` frozen as executable v3.2 identity; `89b7792d…` remains recognized historical/non-executable. |
| Column-order divergence in governed projection | Sev2 | **Resolved** | Column-name canonicalization for signed product tables in Python and Go. Fresh-install fingerprint unchanged; P2 converges. |
| P2 structural digest mismatch (INDEX/TRIGGER/SEQUENCE/CONSTRAINT) | Sev2 | **Resolved** | Only SEQUENCE digest actually differs between P3 and P2 sources; generator updated to replace only that digest. |

**Unresolved Sev1 defects: 0.**  
**Unresolved Sev2 defects: 0.**
