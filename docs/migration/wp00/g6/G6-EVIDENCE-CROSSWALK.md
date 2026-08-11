# WP-00 G6 — AC-00 Evidence Crosswalk

**Authorization:** `G6-AUTH-2026-08-11`  
**G6 starting baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce`  
**Status:** **G6 ACTIVE · EVIDENCE REVIEW IN PROGRESS · P2 REHEARSAL BLOCKED ON CUSTODY ACCESS**  
**Date:** 2026-08-11

This crosswalk is a G6 working evidence record. It does not itself accept WP-00. A criterion is marked `PASS-PRESERVED` only where prior signed/accepted evidence already establishes the property and G6 has found no contradictory evidence. `PENDING-FINAL` requires final-commit or A7 closure evidence. `BLOCKED-P2` requires the approved P2 restore artifact and cannot be substituted with P3.

## 1. Bound inputs

| Input | Bound identity |
|---|---|
| Accepted G5 baseline | `dc366eadede4556615dd5d3977c35cceae43dcce` |
| G5 implementation squash | `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| G4 implementation/proof tip | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| G4 Linux proof | Actions run `31336112238` — 11/11 PASS |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Approved P1/P2 source fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| Approved P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| Approved P2 operational backup reference | `opbak-20260731-173628` |
| Approved P2 operational backup SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| G1 custody manifest version | `c7b86b9b-18fc-4504-942c-fe26ccc83f07` |
| G1 custody manifest SHA-256 | `75ce4f08f4afdc7cedff0e1d977026ec5b590f8822caefdc079905ed678130c1` |
| PostgreSQL proof image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` |

Sensitive P1/P2 bytes remain outside Git and ordinary CI. This record contains references and digests only.

## 2. AC-00-01 through AC-00-30

| Criterion | G6 state | Evidence / remaining action |
|---|---|---|
| AC-00-01 | PASS-PRESERVED | WP-00 source/freeze authority and repository history bind the historical source and revision starting baseline; no contradictory G6 evidence observed. |
| AC-00-02 | PASS-PRESERVED | G0/G1 freeze and deployed-artifact reconciliation were prerequisites to the closed foundation chain; preserve existing freeze evidence for A1 reconstruction. |
| AC-00-03 | PASS-PRESERVED | Durable context-worker poison-event disposition is implemented; G5 required `Backend (Go)` retained the context-worker tests. |
| AC-00-04 | PASS-PRESERVED | G0 closed with source/deployment reconciliation; no later evidence identifies a file-copy-only semantic delta outside the frozen source. |
| AC-00-05 | PASS-PRESERVED | G1 closed on deterministic profiler/fingerprint evidence. |
| AC-00-06 | PASS-PRESERVED | G1 approved P1/P2/P3 profile pack and role/grant/source identities. |
| AC-00-07 | PASS-PRESERVED | G1 records P2 as an isolated restore-derived approved profile; the release rehearsal must reuse the approved P2 lineage rather than synthesize a replacement. |
| AC-00-08 | PASS-PRESERVED | P3 deterministic fixture and golden fingerprint accepted and continuously validated by CI. |
| AC-00-09 | PASS-PRESERVED | G4/G5 negative-profile proof rejects unknown/drifted source before prohibited DDL. |
| AC-00-10 | PASS-PRESERVED | G2 signed target decisions cover 016/018/029. |
| AC-00-11 | PASS-PRESERVED | Legacy 001-040 checksum inventory remains immutable and is audited in G5. |
| AC-00-12 | PASS-PRESERVED | G4-10 and G5 regression prove the supported runner cannot select legacy 001-040. |
| AC-00-13 | PASS-PRESERVED | G4-01/G5 database matrix prove deterministic fresh-install A/B convergence to the governed target. |
| AC-00-14 | **BLOCKED-P2** | P3 adoption is proven, but WS6 requires the approved restored-P2 release-artifact rehearsal with no manual DDL/data correction. The approved backup bytes are not accessible in this execution environment. |
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
| AC-00-25 | PASS-PRESERVED | G4/G5 fresh install and P3 adoption use clean PostgreSQL instances and deterministic fingerprints. |
| AC-00-26 | PASS-PRESERVED | G5 historical-truth fixture: 5/5, zero authoritative promotions. |
| AC-00-27 | PASS-PRESERVED | Required Backend suite covers bounded poison-event retry, durable terminal disposition, and replay/operator visibility. |
| AC-00-28 | PASS-PRESERVED | G4/G5 evidence-hygiene and artifact audit passed; no sensitive P1/P2 bytes were committed. Final A7 must repeat release-evidence hygiene review. |
| AC-00-29 | **PENDING-FINAL** | A7 does not yet exist. Final reviewer reconstruction must bind A1-A7 and verify all recorded digests. |
| AC-00-30 | **PENDING-FINAL** | Final Sev-1/Sev-2 foundation defect review and issue #1 disposition remain to be recorded after the P2 rehearsal. |

## 3. WS6 execution state

| WS6 item | State | Notes |
|---|---|---|
| WS6-01 full release-artifact rehearsal | **BLOCKED-P2** | Requires approved backup `opbak-20260731-173628`; no MinIO/S3/object-storage connector or backup bytes are available in this session. |
| WS6-02 failure/recovery rehearsal | PARTIALLY PRESERVED | Pre-DDL, transactional failure, lock contention, verification drift and rerun properties are preserved by G4/G5. Post-commit behavior and complete release-artifact rehearsal must be confirmed in the P2 run. |
| WS6-03 historical-truth confirmation | PASS-PRESERVED | G5 required historical-truth job passed and remains blocking. Re-run may be included in final G6 evidence without changing semantics. |
| WS6-04 final foundation review | IN PROGRESS | Frozen identities and G5 enforcement reviewed; final P2 result, release artifact provenance, final secret/evidence scan and defect disposition remain. |
| WS6-05 release decision | PENDING | A7 and required G6 decisions cannot be issued until blocked/pending criteria close. |

## 4. Demonstrated blocker

**Observed problem:** the approved P2 backup is an external custody artifact and is not available through the GitHub connector, local runtime, or any installable MinIO/S3/object-storage connector in this session.

**Direct acceptance impact:** WS6-01 and the final P2 form of AC-00-14 cannot be executed or evidenced. P3 is not an acceptable substitute for the mandatory P2 rehearsal.

**Required corrective input:** provide an execution path to the approved backup in an isolated PostgreSQL 16 rehearsal environment, or execute the approved G6 P2 run externally and provide sanitized, digest-bound results sufficient to verify:

1. backup identity `opbak-20260731-173628` / SHA-256 `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2`;
2. isolated restore success;
3. restored source fingerprint `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`;
4. supported Go-runner adoption/forward apply without legacy replay or manual DDL/data edits;
5. restart/recovery behavior;
6. final verify result and governed target `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6`;
7. sanitized evidence digests and no-secret result.

Until that evidence exists, **G6 remains active but BLOCKED and WP-00 is not accepted**.
