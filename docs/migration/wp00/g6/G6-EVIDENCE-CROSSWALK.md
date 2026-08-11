# WP-00 G6 — AC-00 Evidence Crosswalk

**Authorization:** `G6-AUTH-2026-08-11`  
**G6 starting baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce`  
**Status:** **G6 ACTIVE · BLOCKED — P2 SOURCE-PROFILE SUCCESSOR DECISION REQUIRED**  
**Date:** 2026-08-11

This crosswalk is a G6 working evidence record. It does not itself accept WP-00. A criterion is marked `PASS-PRESERVED` only where prior signed/accepted evidence still establishes the property and G6 has found no contradictory evidence that invalidates that property. `PENDING-FINAL` requires final-commit or A7 closure evidence. `BLOCKED-P2` means the mandatory approved-P2 release path cannot currently complete.

## 1. Bound inputs

| Input | Bound identity |
|---|---|
| Accepted G5 baseline | `dc366eadede4556615dd5d3977c35cceae43dcce` |
| G5 implementation squash | `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| G4 implementation/proof tip | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| G4 Linux proof | Actions run `31336112238` — 11/11 PASS |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Historical G1 P1/P2 source fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| Fresh accepted-profiler v3.2 P2 observation | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` |
| Approved P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| Approved P2 operational backup reference | `opbak-20260731-173628` |
| Approved P2 operational backup SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| P2a custody manifest version ID | `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1` |
| P2a custody manifest SHA-256 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |
| G1 custody manifest version | `c7b86b9b-18fc-4504-942c-fe26ccc83f07` |
| G1 custody manifest SHA-256 | `75ce4f08f4afdc7cedff0e1d977026ec5b590f8822caefdc079905ed678130c1` |
| PostgreSQL proof image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` |

Sensitive P1/P2 bytes remain outside Git and ordinary CI. This record contains references, digests, and sanitized diagnostic facts only.

## 2. G6 P2 diagnostic finding

The approved backup was recovered from custody and its SHA-256 matched exactly. A clean PostgreSQL 16.14 restore completed without manual DDL/data correction.

The immutable P2a custody manifest then established that the historical P1/P2 fingerprint `89b7792d...` was produced with profiler `3.1.0-p1p2`, despite repository G1 prose stating `3.2.0-p1p2`.

A fresh capture of the same restored database with the accepted `3.2.0-p1p2` profiler produced `57c2b645...`. After removing the accepted fingerprint-excluded fields, the stable manifests differed at exactly one path: `profiler_version`. The reported schema/catalog, roles/grants, extensions, migration state, and fingerprinted PostgreSQL settings were otherwise identical.

The accepted Go runner therefore correctly returns `SOURCE_PROFILE_UNKNOWN` before DDL because its v3.2 profiler computes `57c2b645...` while the executable allowlist contains historical identity `89b7792d...`.

See `G6-P2-PROFILER-DRIFT-DIAGNOSTIC.md`.

## 3. AC-00-01 through AC-00-30

| Criterion | G6 state | Evidence / remaining action |
|---|---|---|
| AC-00-01 | PASS-PRESERVED | WP-00 source/freeze authority and repository history bind the historical source and revision starting baseline. The profiler-version metadata contradiction is now explicitly carried as successor evidence rather than silently rewriting G1. |
| AC-00-02 | PASS-PRESERVED | G0/G1 freeze and deployed-artifact reconciliation remain preserved; no executable source/binary mismatch was demonstrated by the P2 diagnostic. |
| AC-00-03 | PASS-PRESERVED | Durable context-worker poison-event disposition is implemented; G5 required `Backend (Go)` retained the context-worker tests. |
| AC-00-04 | PASS-PRESERVED | No later evidence identifies a file-copy-only semantic delta outside the frozen source. |
| AC-00-05 | PASS-PRESERVED | Immutable G1 P2a/P2b evidence established deterministic captures under the historical profiler. G6 additionally reproduced the historical P2a digest and isolated the version-only successor difference. |
| AC-00-06 | PASS-PRESERVED | G1 profile evidence remains structurally valid; G6 found the profiler-version metadata/provenance contradiction but no schema/catalog divergence in the restored P2 source. |
| AC-00-07 | PASS-PRESERVED | Approved P2 backup was recovered from immutable custody, digest-verified, and restored cleanly in isolation. |
| AC-00-08 | PASS-PRESERVED | P3 deterministic fixture and golden fingerprint accepted and continuously validated by CI. |
| AC-00-09 | PASS-PRESERVED | G4/G5 negative-profile proof and the live G6 P2 attempt both demonstrate fail-closed rejection before DDL for an unrecognized fingerprint. |
| AC-00-10 | PASS-PRESERVED | G2 signed target decisions cover 016/018/029. |
| AC-00-11 | PASS-PRESERVED | Legacy 001-040 checksum inventory remains immutable and is audited in G5. |
| AC-00-12 | PASS-PRESERVED | G4-10/G5 prove the supported runner cannot select legacy 001-040; no legacy replay occurred in the blocked P2 rehearsal. |
| AC-00-13 | PASS-PRESERVED | G4-01/G5 database matrix prove deterministic fresh-install A/B convergence to the governed target. |
| AC-00-14 | **BLOCKED-P2** | The real approved P2 restore is now available and structurally reproduced, but the supported v3.2 runner rejects it as `SOURCE_PROFILE_UNKNOWN`. A governed successor source-profile decision/correction is required, followed by a clean full P2 adoption/apply/restart/verify rehearsal with no manual DDL/data edits. |
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
| AC-00-25 | PASS-PRESERVED | G4/G5 fresh install and P3 adoption use clean PostgreSQL instances and deterministic fingerprints; the G6 P2 diagnostic also used a clean isolated restore. |
| AC-00-26 | PASS-PRESERVED | G5 historical-truth fixture: 5/5, zero authoritative promotions. |
| AC-00-27 | PASS-PRESERVED | Required Backend suite covers bounded poison-event retry, durable terminal disposition, and replay/operator visibility. |
| AC-00-28 | PASS-PRESERVED | G4/G5 evidence-hygiene and artifact audit passed; the G6 diagnostic supplied only sanitized references/digests. Final A7 must repeat release-evidence hygiene review. |
| AC-00-29 | **PENDING-FINAL** | A7 does not yet exist. Final reconstruction must include the historical v3.1 identity, v3.2 successor decision, corrective implementation/proof, and all prior A1-A6 evidence without rewriting G1. |
| AC-00-30 | **BLOCKED-DEFECT** | A demonstrated P1/P2 source-profile identity/provenance defect is unresolved. G6 cannot pass until the successor decision is authorized, implemented, reproven, and the final Sev-1/Sev-2 defect disposition records no remaining qualifying defect. |

## 4. WS6 execution state

| WS6 item | State | Notes |
|---|---|---|
| WS6-01 full release-artifact rehearsal | **BLOCKED-P2** | Backup retrieval/restore now passes. Runner preflight blocks on v3.1 historical identity versus v3.2 accepted-profiler identity. |
| WS6-02 failure/recovery rehearsal | PARTIALLY PRESERVED | Pre-DDL rejection is directly demonstrated on P2; transactional failure, lock contention, verification drift and rerun properties remain preserved by G4/G5. Full P2 post-apply restart/recovery remains pending after the corrective successor. |
| WS6-03 historical-truth confirmation | PASS-PRESERVED | G5 required historical-truth job passed and remains blocking. |
| WS6-04 final foundation review | **BLOCKED-DEFECT** | Final review found the G1 profiler-version provenance contradiction and executable P2 classification defect. |
| WS6-05 release decision | PENDING | A7 and final decisions cannot be issued until the corrective successor is complete and the P2 rehearsal passes. |

## 5. Demonstrated blocker

**Observed problem:** the immutable G1 P2a custody manifest uses profiler `3.1.0-p1p2` and fingerprint `89b7792d...`; the accepted G4/G5 runner uses profiler `3.2.0-p1p2` and computes `57c2b645...` for the same restored database. The stable manifests differ only at `profiler_version`.

**Direct acceptance impact:** the supported runner rejects the mandatory restored P2 source before DDL, so AC-00-14 and WS6-01 cannot complete. AC-00-30 also remains blocked by the unresolved foundation identity/provenance defect.

**Current safe behavior:** the runner fails closed, executes no legacy migration, performs no DDL, and does not misclassify the source.

**Required next governance action:** separately authorize a bounded P1/P2 source-profile successor correction. The preferred direction is to preserve `89b7792d...` as historical G1/v3.1 evidence, establish `57c2b645...` as the v3.2 successor identity backed by immutable custody comparison/repeat captures, update only the supported classification mechanism, retain all negative-profile/G4/G5 safeguards, then rerun the complete clean P2 rehearsal.

No such corrective implementation is authorized by this evidence record.

Until that successor decision is authorized, implemented, and reproven, **G6 remains BLOCKED and WP-00 is not accepted**.
