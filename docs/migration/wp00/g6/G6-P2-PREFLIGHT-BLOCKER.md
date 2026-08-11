# WP-00 G6 — P2 Preflight Blocker Record

**Authorization:** `G6-AUTH-2026-08-11`  
**G6 starting baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce`  
**Status:** **BLOCKED — SOURCE PROFILE RECONCILIATION REQUIRED BEFORE MUTATION**  
**Date:** 2026-08-11

This is a sanitized G6 working evidence record. It does not alter or supersede any frozen G1-G5 identity and does not accept WP-00.

## 1. Custody retrieval and restore evidence reported by the rehearsal operator

The external G6 rehearsal reported:

- approved backup reference `opbak-20260731-173628` found in the G1 custody store;
- exact backup SHA-256 observed: `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2`;
- byte size: `1228736`;
- immutable custody object version: `b315248b-9ddd-4c32-9886-c7d3035c4a37`;
- clean isolated PostgreSQL 16.14 restore completed without manual DDL or data correction;
- no legacy migration was executed;
- supported runner stopped before DDL with `SOURCE_PROFILE_UNKNOWN`.

The raw backup, restored database, manifests, credentials, host addresses, and sensitive P1/P2 contents remain outside Git.

## 2. Observed source-profile conflict

The external rehearsal reported two different source fingerprints from the same restored backup state:

| Observation | Reported fingerprint |
|---|---|
| Rehearsal capture using a historical v3.1 profiler checkout | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| Current accepted v3.2 profiler / Go runner | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` |

The Go runner therefore correctly failed closed before mutation because `57c2b645...` is not an approved source-profile identity.

## 3. Contradiction with signed G1 repository evidence

The local explanation that the frozen `89b7792d...` identity was created by profiler v3.1 is **not supported by the signed G1 repository record**.

The G1 `RESTORE-PROOF.md` and `CAPTURE-REPORT.md` both state that:

- profiler version was `3.2.0-p1p2`;
- P1, P2a, and P2b all produced `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`;
- P2a custody manifest SHA-256 is `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb`;
- P2a immutable object version is `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1`.

The exact `scripts/profile/capture_schema.py` blob at the G1 closure head and at accepted G5 is the same Git blob:

`731324aabbe049dc5278f3cedc49bf8980c5f5e5`

and declares `PROFILER_VERSION = "3.2.0-p1p2"`.

Commit `2db092e0e020fd8260c23612001156c51d583d40` changed the profiler from v3.1 to v3.2 by removing locale/environment settings from the hashed PostgreSQL settings and bumping the version. That commit predates the G1 closure records that explicitly bind the P1/P2 identity to v3.2.

Therefore no runner or allowlist change is authorized from the current evidence. The immutable G1 custody manifest must be compared directly with a fresh v3.2 capture from the restored approved backup.

## 4. Required bounded diagnostic

Retrieve the exact immutable G1 P2a manifest from custody:

- object key: `manifest-p2a.json` (custody paths may include a prefix);
- version ID: `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1`;
- expected byte size: `250235`;
- expected SHA-256: `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb`.

Compare it against a fresh manifest produced by the exact accepted v3.2 profiler blob `731324aabbe049dc5278f3cedc49bf8980c5f5e5` from the clean restored approved backup.

The diagnostic must report only sanitized metadata and structural difference paths/digests, not raw sensitive manifest values.

Required assertions:

1. custody P2a bytes match the frozen SHA-256;
2. custody P2a `profiler_version` value;
3. custody P2a stored fingerprint and independently recomputed fingerprint;
4. fresh v3.2 stored/recomputed fingerprint;
5. canonical stable-manifest digest for each capture;
6. top-level and nested difference paths between the two stable manifests, with values represented only by type/length/SHA-256 where sensitive;
7. whether the difference is profiler-version metadata only, role/grant posture, PostgreSQL settings, extension state, catalog structure, or another concrete field set.

## 5. Decision boundary

Do **not**:

- update the frozen P1/P2 allowlist identity;
- modify signed G1 evidence;
- modify accepted G4 runner semantics;
- manually repair the restored database;
- execute any migration after `SOURCE_PROFILE_UNKNOWN`.

If the custody P2a manifest itself proves v3.2 + `89b7792d...`, the current runner/live capture mismatch must be diagnosed and corrected through a governed successor if it is an accepted-runner defect.

If the custody manifest instead proves that the signed G1 prose incorrectly attributes a v3.1 capture to v3.2, that is a signed-evidence contradiction requiring a governed successor decision; the frozen record must not be silently rewritten.

**Current G6 decision:** `BLOCKED` pending exact custody-manifest reconciliation. AC-00-14 remains unsatisfied and AC-00-30 cannot close while this foundation discrepancy is unresolved.
