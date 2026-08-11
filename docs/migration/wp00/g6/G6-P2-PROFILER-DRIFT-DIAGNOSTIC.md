# WP-00 G6 — P2 Profiler Identity Drift Diagnostic

**Date:** 2026-08-11  
**G6 baseline:** `dc366eadede4556615dd5d3977c35cceae43dcce`  
**Status:** **DEMONSTRATED G6 BLOCKER · NO CORRECTIVE MUTATION AUTHORIZED**  
**Evidence source:** sanitized external rehearsal/diagnostic performed against the approved custody artifact and supplied by the client. Raw P1/P2 bytes remain outside Git.

## 1. Custody identities verified

| Item | Verified value |
|---|---|
| Backup reference | `opbak-20260731-173628` |
| Backup SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| Backup size | `1,228,736` bytes |
| Custody bucket/key | `clarityit-g1-evidence/opbak.sql.gz` |
| Backup version ID | `b315248b-9ddd-4c32-9886-c7d3035c4a37` |
| P2a manifest version ID | `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1` |
| P2a manifest expected/observed SHA-256 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |

The immutable custody object was read under the approved evidence-custody path. The reported digest matched exactly before inspection.

## 2. Diagnostic result

The client-supplied sanitized diagnostic reported:

```text
P2A_CUSTODY_DIGEST_MATCH=yes
P2A_CUSTODY_PROFILER_VERSION=3.1.0-p1p2
P2A_CUSTODY_STORED_FP=89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83
P2A_CUSTODY_RECOMPUTED_FP=89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83
FRESH_V32_FP=57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb
STABLE_MANIFESTS_EQUAL=no
DIFFERING_PATH_COUNT=1
ROOT_CAUSE_CLASS=profiler_version
```

After removal of the accepted fingerprint-excluded top-level fields, the custody P2a manifest and a fresh capture from the same restored backup differed at exactly one JSON path:

`profiler_version`

All reported structural sections were identical, including schemas, relations, columns, constraints, indexes, triggers, sequences, functions, roles/grants, extensions, migration state, and PostgreSQL fingerprinted settings.

## 3. Provenance contradiction

The immutable P2a custody manifest records:

- profiler version `3.1.0-p1p2`;
- fingerprint `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`.

Repository G1 prose currently states that the same P1/P2 identity was produced by profiler `3.2.0-p1p2`. That statement is contradicted by the immutable custody manifest.

This is historical metadata/provenance drift. The signed G1 repository evidence must not be rewritten in place. The contradiction must be carried forward through a governed successor record.

## 4. Current runner behavior

The accepted G4/G5 Go source profiler declares `3.2.0-p1p2` and correctly hashes the version as part of the canonical source-profile document.

For the restored approved P2 source it computes:

`57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`

The runner allowlist recognizes the historical G1 identity `89b7792d...`, not the v3.2 successor identity. It therefore reports `SOURCE_PROFILE_UNKNOWN` before DDL.

That fail-closed behavior is correct. No legacy migration was executed, no manual DDL/data repair was used, and `apply` did not begin.

## 5. G6 acceptance impact

- **AC-00-14:** BLOCKED — the restored P2 source cannot currently traverse the supported runner path.
- **WS6-01:** BLOCKED — full P2 restore/adoption/apply/restart/verify rehearsal cannot proceed.
- **AC-00-30:** BLOCKED pending governed corrective disposition — a demonstrated foundation identity/provenance defect remains unresolved.
- **AC-00-29:** final A7 must disclose and bind the successor decision and historical contradiction.

The already-accepted G4 P3/fresh-install proofs are not invalidated by this finding. The defect is specific to the unrehearsed P1/P2 classification path that G6 was required to exercise.

## 6. Narrow corrective direction

No corrective implementation is authorized by this diagnostic.

The technically preferred successor direction is:

1. preserve `89b7792d...` as the immutable historical G1/v3.1 source identity;
2. establish `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` as the v3.2 successor identity for the same approved P1/P2 structural source, backed by the custody comparison and repeat captures;
3. update the supported runner source-profile classification under a separately authorized corrective change so the v3.2 successor identity is recognized as the approved P1/P2 source;
4. do not modify the frozen G1 custody object or rewrite signed G1 history;
5. do not downgrade the accepted v3.2 profiler merely to reproduce the historical digest, because v3.2 intentionally removed locale-dependent fingerprint inputs;
6. rerun the complete G6 P2 rehearsal from a clean restore after the corrective change;
7. preserve negative-profile fail-closed behavior and all G4/G5 regressions.

A successor decision must explicitly define whether `89b7792d...` remains recognized as historical-only/non-executable or remains an additional executable alias. That classification is a governance decision and is not made by this diagnostic record.

## 7. Decision

**G6 remains BLOCKED.**

The next permissible implementation step requires a separate recorded authorization for the bounded P1/P2 source-profile successor correction. Until that occurs, no runner allowlist/profiler change is permitted and WP-00 cannot be accepted.
