# WP-00 G6 — Terminal Closure Authorization

**Authorization ID:** `G6-TERMINAL-CLOSURE-AUTH-2026-08-11`  
**Status:** **AUTHORIZED BY CLIENT · EXECUTE TO ACCEPTANCE OR ONE DEMONSTRATED BLOCKER**  
**Date:** 2026-08-11  
**Starting G6 baseline:** accepted G5 integration `dc366eadede4556615dd5d3977c35cceae43dcce`

## Decision

The client explicitly authorizes the complete terminal G6 closure package. This supersedes the need for further routine micro-authorizations inside this bounded corrective package. Routine implementation choices, deterministic IDs, artifact naming, code organization, CI mechanics, evidence formatting, and reversible proof steps are implementation matters and do not require separate approval.

## Authorized corrective scope

The package may:

1. Preserve historical G1 P1/P2 fingerprint `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` as immutable profiler-v3.1 evidence and recognized/non-executable history.
2. Establish and freeze the reproducible profiler-v3.2 P2 successor fingerprint `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` after two independent clean restored-P2 captures reproduce it.
3. Add a truthful P2-specific executable adoption path and deterministic P2 source-profile identity.
4. Add a new generated P2-specific adoption artifact while preserving `0001_adopt_p3.sql` and its P3 semantics byte-for-byte.
5. Update packaging, classification, source-profile ledger mapping, runner dispatch, tests, and evidence required for the P2-specific path.
6. Preserve legacy `001-040` as non-selectable/non-executable provenance only.
7. Preserve the accepted fresh-install and P3 paths and all G4/G5 fail-closed properties.
8. Run the complete G4/G5 regression suite and all required repository checks.
9. Run the exact-integrated real-P2 WS6 rehearsal: backup verification, clean restore, v3.2 classification, P2 adoption/apply, restart, duplication-free reapply, verify, failure/recovery, truth/security/provenance review, and sanitized evidence.
10. Produce A7, close AC-00-01 through AC-00-30 if supported by evidence, record the required G6 role decisions, update completion authority, dispose issue #1, and accept WP-00 if and only if every frozen criterion passes.

## Frozen boundaries

The package must not:

- rewrite or delete signed G1-G5 evidence;
- reinterpret `89b7792d...` as a v3.2 fingerprint;
- modify the frozen P3 adoption artifact to masquerade as P2;
- replay legacy migrations `001-040`;
- use manual DDL/data repair to make P2 pass;
- alter the approved governed target fingerprint `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6`;
- weaken unknown/drifted-source rejection, checksum, lock, verification, provenance, or CI controls;
- perform provider mutation, production cutover, Site Runtime work, or WP-01+ implementation.

## Closure rule

Execute continuously through the bounded package. Stop only when either:

1. all frozen G6/WP-00 criteria pass and the final acceptance receipt is integrated; or
2. a concrete observed problem with evidence directly prevents the accepted result or proves it unsafe/invalid.

No new acceptance gates are to be invented after implementation begins.
