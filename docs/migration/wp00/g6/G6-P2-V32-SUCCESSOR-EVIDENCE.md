# WP-00 G6 — P2 v3.2 Successor Repeat Evidence

**Authority:** `G6-TERMINAL-CLOSURE-AUTH-2026-08-11`  
**Status:** **T1 COMPLETE · SUCCESSOR IDENTITY FROZEN FOR TERMINAL PACKAGE**  
**Evidence source:** sanitized external custody-host rehearsal reported by the authorized local assistant  

## Bound source

- Approved backup reference: `opbak-20260731-173628`
- Approved backup SHA-256: `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2`
- Historical immutable G1/P2 v3.1 fingerprint: `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
- Accepted profiler blob: `731324aabbe049dc5278f3cedc49bf8980c5f5e5`
- Profiler version: `3.2.0-p1p2`
- Frozen v3.2 successor fingerprint: `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`

## Repeat result

Two independent clean PostgreSQL 16 restores from the approved backup reproduced the same v3.2 source fingerprint:

```text
P2_V32_CAPTURE_A=57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb
P2_V32_CAPTURE_B=57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb
```

The prior custody comparison established that the historical v3.1 stable manifest and fresh v3.2 stable manifest differ in the fingerprinted JSON at exactly one path: `profiler_version`; schema/catalog, roles/grants, extensions, migration state, and fingerprinted PostgreSQL settings otherwise match.

Under `G6-TERMINAL-CLOSURE-AUTH-2026-08-11`, this repeat evidence freezes `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` as the executable P2 v3.2 successor identity for the terminal corrective package.

`89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` remains immutable historical v3.1 evidence and recognized/non-executable history.

## Additional P2 source observations for T2

The authorized clean-restored P2 inspection additionally observed:

- four legacy permission names requiring the already-approved Decision-016 reconciliation: `docs.edit.any`, `docs.edit.own`, `incidents.edit.any`, `incidents.edit.own`;
- `pg_trgm` absent on the P2 source while it is required by the governed G2 target;
- source bootstrap role `clarityit` is the source superuser and owns installed source extensions;
- no `platform` schema exists before adoption.

These are implementation inputs for the already-authorized P2-specific adoption path, not new acceptance criteria.
