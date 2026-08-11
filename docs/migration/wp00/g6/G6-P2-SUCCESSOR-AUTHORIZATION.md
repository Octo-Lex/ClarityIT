# WP-00 G6 — P1/P2 v3.2 Source-Identity Successor Authorization

**Authorization ID:** `G6-P2-SUCCESSOR-AUTH-2026-08-11`  
**Status:** **AUTHORIZED BY CLIENT · BOUNDED CORRECTIVE SUCCESSOR**  
**Authorization date:** 2026-08-11  
**Parent G6 authorization:** `G6-AUTH-2026-08-11`  
**Starting baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce` plus preserved G6 diagnostic evidence on `wp00/g6-acceptance`

## 1. Demonstrated defect

The immutable G1 custody artifact `manifest-p2a.json` (version `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1`, SHA-256 `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb`) records:

- `profiler_version = 3.1.0-p1p2`;
- source fingerprint `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`.

The accepted G4/G5 runner uses source profiler `3.2.0-p1p2`. Against the same approved P2 restore, v3.2 computes candidate fingerprint:

`57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`

The sanitized diagnostic reports that, after removal of the normal fingerprint-excluded fields, the v3.1 and v3.2 stable manifests differ at exactly one JSON path: `profiler_version`. The runner therefore correctly fails closed with `SOURCE_PROFILE_UNKNOWN` before DDL.

## 2. Authorization decision

The client separately authorizes the narrow corrective successor required to reconcile this source-profile version drift and resume the frozen G6 P2 rehearsal.

This authorization permits only:

1. reproduce the v3.2 P1/P2 candidate identity using the exact accepted v3.2 profiler and the approved P2 backup lineage;
2. require deterministic repeat evidence before freezing the successor identity;
3. preserve `89b7792d...` as immutable historical G1/v3.1 custody identity and provenance evidence;
4. establish the reproducible v3.2 identity as the supported P1/P2 classification identity through a governed successor record;
5. update only the runner source-profile classification/constants/tests necessary to recognize that successor;
6. keep the historical v3.1 identity recognized but non-executable;
7. preserve all unknown/drifted-source fail-closed behavior and all G4/G5 regression properties;
8. run the existing adoption contract against the real restored P2 only if its existing fail-closed structural/precondition checks accept the P2 state without changing adoption SQL or migration semantics;
9. rerun the full G6 P2 restore → classify → apply → restart/reapply → verify rehearsal and bind sanitized evidence;
10. resume G6 acceptance only if the corrective successor and the full P2 rehearsal pass.

## 3. Successor-freeze precondition

The candidate `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` is **observed, not yet frozen**, until deterministic v3.2 repeat evidence is recorded.

Before the runner allowlist is changed, evidence must show at minimum:

- approved backup SHA-256 `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2`;
- PostgreSQL 16 isolated restore with the approved source role posture;
- exact accepted `capture_schema.py` blob `731324aabbe049dc5278f3cedc49bf8980c5f5e5`;
- two unchanged v3.2 captures producing the same fingerprint;
- both captures produce candidate `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`;
- no structural difference from the immutable P2a stable manifest except `profiler_version`.

If any of those assertions fails, stop and keep the candidate unfrozen.

## 4. Historical identity disposition

`89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` remains:

- frozen historical G1 custody identity;
- attributed to profiler `3.1.0-p1p2` by the immutable custody artifact;
- recognized for provenance/diagnostics;
- **non-executable** by the supported v3.2 runner.

No signed G1 object, custody object, or historical digest may be rewritten.

## 5. Corrective implementation boundary

The intended implementation is classifier-only unless live evidence demonstrates that the accepted adoption contract cannot safely consume the real P2 source.

Permitted code changes are limited to source-profile identity constants, classification mapping, stable diagnostics, and tests/evidence directly required for the successor.

The following are not authorized without another explicit decision:

- modification of `0001_adopt_p3.sql` or other frozen migration SQL;
- new migration/backfill semantics;
- legacy `001-040` replay;
- manual DDL or business-data edits;
- weakening checksum, advisory-lock, transactional, verification, privilege, or evidence-hygiene controls;
- provider, Site Runtime, execution-kernel, or WP-01+ work;
- production migration/cutover.

If the accepted adoption artifact rejects the real P2 for a structural/precondition reason after the successor classifier is proven, that is a demonstrated blocker requiring a new governed decision rather than an expansion under this authorization.

## 6. Required regression evidence

Before integration/acceptance of the corrective successor:

- source-classification unit/matrix tests pass;
- historical v3.1 identity remains recognized/non-executable;
- v3.2 successor is executable only through the existing approved-source adoption path;
- unknown/drifted sources remain blocked before DDL;
- G4 11-row evidence matrix remains green;
- G5 Foundation Gate remains green;
- ordinary required Frontend, Worker, and Backend checks remain green;
- the real P2 rehearsal reaches the governed target without legacy replay or manual correction.

## 7. Stop boundary

This authorization does not accept the successor identity, G6, or WP-00 by itself. It authorizes the bounded correction and evidence needed to reach a decision.

If the successor evidence or P2 rehearsal fails, record the observed blocker and stop. Conditional or partial G6 acceptance remains prohibited.
