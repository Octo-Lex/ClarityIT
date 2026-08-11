# WP-00 G6 — Terminal Execution Package

**Authority:** `G6-TERMINAL-CLOSURE-AUTH-2026-08-11`  
**Status:** **EXECUTE CONTINUOUSLY TO G6 CLOSURE OR ONE DEMONSTRATED BLOCKER**  
**Date:** 2026-08-11

This is the single remaining implementation/rehearsal package for WP-00 G6. Routine implementation choices inside this scope do not require further authorization.

## 1. Frozen identities

- Accepted G5 baseline: `dc366eadede4556615dd5d3977c35cceae43dcce`
- Historical G1 P1/P2 v3.1 fingerprint: `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
- Candidate P2 v3.2 successor: `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`
- Deterministic P2 successor profile UUID: `7b5b8b87-3467-5fd5-9bac-3dbcdd858178`
- Historical source commit: `b9d15877583e84c45ab5478dfaa6087966926fc5`
- P3 source fingerprint: `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`
- Governed target: `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6`
- Baseline checksum: `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508`
- Approved P2 backup: `opbak-20260731-173628`
- Backup SHA-256: `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2`
- PostgreSQL proof image: `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382`
- P3 adoption artifact SHA-256 must remain: `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359`
- Terminal authorization commit: `20cb1336828b50ebf4abf462d91a5684d22895ad`

`89b7792d...` remains recognized historical v3.1 evidence and non-executable. It must never be relabeled as v3.2.

## 2. Phase T1 — freeze the v3.2 successor

On the custody host, perform two independent clean PostgreSQL 16 restores from the exact approved backup. Do not reuse a volume/database between captures.

For each restore:

1. verify backup SHA-256;
2. restore without manual DDL/data correction;
3. use the exact accepted `scripts/profile/capture_schema.py` blob `731324aabbe049dc5278f3cedc49bf8980c5f5e5` (`3.2.0-p1p2`);
4. capture twice if needed to establish determinism;
5. retain sanitized manifest digests only.

Required:

```text
P2_V32_CAPTURE_A=57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb
P2_V32_CAPTURE_B=57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb
```

Also verify the v3.1 custody manifest and fresh v3.2 manifest differ in the fingerprinted stable JSON only at `profiler_version`.

If either clean restore produces another fingerprint, stop with a concrete manifest-diff report. Otherwise the v3.2 successor is frozen by this terminal authorization and repeat evidence.

## 3. Phase T2 — implement P2-specific adoption

Work on `wp00/g6-acceptance` or a child implementation branch based on its latest authorized tip.

### 3.1 Preserve existing paths

Do not alter the bytes or semantics of:

- `migrations/v2/adoption/0001_adopt_p3.sql`;
- P3 profile ID/fingerprint/path;
- fresh-install artifacts;
- legacy `001-040` archive.

### 3.2 Source classification

Implement an explicit P2 path, preferably `PathAdoptP2 = "adopt_p2"`.

Classification must be exactly:

- `57c2b645...` -> approved executable P2 path;
- `cedf689d...` -> existing P3 path unchanged;
- `89b7792d...` -> recognized historical/non-executable with stable diagnostic;
- any other non-governed non-empty fingerprint -> fail closed before DDL.

### 3.3 Deterministic P2 adoption artifact

Generate a new `0001_adopt_p2.sql` from the deterministic generator; do not hand-maintain a fork of the P3 SQL.

The P2 artifact may reuse the existing signed structural digest/preflight logic where it is valid for the P1/P2 canonical schema, but must record truthful P2 provenance and must not pretend P2 is P3.

The source profile row must bind:

- profile ID `7b5b8b87-3467-5fd5-9bac-3dbcdd858178`;
- schema fingerprint `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`;
- source commit `b9d15877583e84c45ab5478dfaa6087966926fc5`;
- approval reference `G6-TERMINAL-CLOSURE-AUTH-2026-08-11` or its repository authorization record;
- PostgreSQL major 16 and the approved extensions;
- the same signed target role/membership digest convention already used by the generated adoption artifacts.

The revision row must remain version `0001`, checksum `1021adef...`, and use a truthful P2 revision name such as `adopt-p2-v32`.

### 3.4 Approved product-row reconciliation only

The P1/P2 schema is already the signed canonical G2 product shape. Do not recreate product tables or backfill arbitrary data.

The only permitted source business-row reconciliation is the already-signed G2 Decision-016 permission normalization:

- legacy-only `.edit` -> rename to `.update`, preserving row ID/grants;
- canonical-only -> preserve canonical row;
- collision -> union `role_permissions`, remove legacy grants/row, preserve canonical;
- neither legacy nor canonical -> fail closed;
- after reconciliation all seven canonical names must exist and no relevant legacy `.edit` row may remain.

Do not introduce any other product-row mutation without a demonstrated blocker that proves an existing signed G2 reconciliation is insufficient.

### 3.5 Role transition

Use the signed G2 Decision-029 target role posture. Validate the actual clean-restored P2 source posture before mutation. The P2 artifact must consume the real approved source posture; do not manually manufacture a P3 fixture posture before running it.

### 3.6 Packaging and runner dispatch

Add the P2 artifact as its own embedded executable asset and bind it to its own frozen SHA-256 / successor package evidence without changing the frozen P3 artifact digest.

Update:

- asset registry / embedding;
- package verification;
- preflight mapping;
- apply dispatch;
- source-profile ledger mapping;
- execution-receipt original/transformed digests;
- deterministic generation/verification scripts.

The existing G3 composite identity must not be silently redefined. Record the P2 successor artifact/package identity separately.

## 4. Phase T3 — tests before PR

Add tests proving at minimum:

1. v3.2 P2 fingerprint -> P2 executable path;
2. historical v3.1 `89b...` remains non-executable;
3. P3 fingerprint/path unchanged;
4. unknown fingerprint blocks before DDL;
5. P2 source profile ledger uses P2 UUID/fingerprint, never P3 identity;
6. P2 permission reconciliation passes legacy-only/canonical-only/collision cases and blocks missing-both;
7. P2 does not select/embed legacy `001-040`;
8. P3 adoption artifact SHA remains exactly `a89ab852...`;
9. packaging checksum mutation fails closed;
10. fresh-install and P3 adoption still converge to `9881c93e...`.

Run Go unit/integration tests and generator verification locally where practical.

## 5. Phase T4 — implementation PR

Open one corrective implementation PR to `main` from the terminal G6 implementation branch.

Before merge, require all four repository contexts:

- `Frontend (typecheck · test · build)`
- `Worker (Python)`
- `Backend (Go)`
- `G5 Foundation Gate`

Also require the G4 11-row regression matrix to remain 11/11 PASS.

Do not merge on a failing required context.

After all checks pass, merge normally. Record the exact integrated main SHA. That SHA becomes the only acceptable binary source for the final P2 rehearsal.

## 6. Phase T5 — exact-integrated real P2 rehearsal

On the custody host, build `clarity-migrate` from the exact integrated corrective main SHA with the producing commit bound.

Using a new clean PostgreSQL 16 instance:

```text
approved backup digest
-> restore
-> v3.2 fingerprint 57c2b645...
-> preflight approved P2 / PathAdoptP2
-> apply
-> no legacy replay
-> no manual DDL/data correction
-> truthful P2 source profile ledger
-> governed target 9881c93e...
```

Then restart PostgreSQL and run:

```text
status
-> reapply
-> no duplicate revision
-> verify
-> target still 9881c93e...
```

Required ledger assertions:

- exactly one successful revision `0001`;
- checksum `1021adef...`;
- P2 source profile ID `7b5b8b87-3467-5fd5-9bac-3dbcdd858178` exists exactly once;
- migration run references that P2 profile ID;
- P3 profile ID/fingerprint is not falsely recorded for the P2 run.

## 7. Phase T6 — WS6 failure/recovery

Use the accepted proof harness plus disposable P2 rehearsal instances.

Prove:

- before-DDL failure -> zero mutation;
- transactional failure -> no partial state and clean rerun;
- advisory-lock contention -> only one governed writer;
- verification failure -> no false success;
- post-commit ambiguity -> after a successful commit, simulate loss of caller result/process context, then reconnect/reapply and prove governed-current/no duplicate revision/no duplicate source-profile row.

Do not invent production-only failpoints if the same property can be proven by disconnect/restart/reapply evidence.

## 8. Phase T7 — regression / security / truth / provenance

Against the exact final candidate:

- G4 matrix = 11/11 PASS;
- historical-truth validator = PASS with zero authoritative legacy promotions;
- ownership/grant/constraint/index verification = PASS;
- legacy execution reachability = none;
- package/artifact checksums = PASS;
- evidence/secret scan = clean;
- P2 provenance = truthful;
- P3 artifact/path = unchanged.

## 9. Phase T8 — final evidence

Produce sanitized evidence only. Do not commit the raw P2 dump or sensitive manifests.

At minimum produce:

- two-capture v3.2 successor evidence + digests;
- corrective implementation/package identities;
- exact-integrated binary SHA/source commit;
- P2 restore/apply/restart/verify transcript summary;
- failure/recovery results;
- final G4/G5 CI results;
- secret scan;
- defect disposition;
- A1-A7 reconstruction;
- AC-00-01 through AC-00-30 crosswalk.

## 10. Phase T9 — G6 final receipt

Create one evidence-only final PR containing:

- `G6-APPROVALS.md`;
- final `G6-EVIDENCE-CROSSWALK.md`;
- A7 WP-00 release evidence manifest;
- profiler-version historical erratum/successor record;
- final Sev1/Sev2 defect disposition;
- `PROJECT-COMPLETION-AUTHORITY.md` update.

Record required G6 decisions for Product, Architecture, Operations, Security, and Quality. If one delegated assessor fills multiple roles, say so explicitly rather than implying independent people.

All four required repository contexts must pass on this final receipt PR.

## 11. Closure

When and only when:

- AC-00-01 through AC-00-30 are PASS;
- real P2 reaches `9881c93e...` without manual correction;
- restart/recovery passes;
- G4 11/11 and G5 blocking checks pass;
- A1-A7 reconstruct;
- unresolved Sev1/Sev2 migration/data-integrity/security/retry/CI defects = 0;
- required G6 decisions are recorded;

then merge the final receipt, close issue #1 if its fresh-install/CI claims are demonstrably resolved, record `G6 = ACCEPTED`, `WP-00 = ACCEPTED`, and stop. Do not start WP-01+.

## Required terminal report

Return exactly:

### Done

Implementation PR/merge SHA, exact-integrated rehearsal SHA, final receipt PR/merge SHA.

### Verified

Two v3.2 captures; P2 classification; apply; restart/reapply; final target; P2 ledger provenance; G4 11/11; G5 checks; historical truth; secret scan; A1-A7; AC-00-01..30; Sev1/Sev2 count.

### Blockers

`None` if closure succeeded; otherwise only one or more concrete observed problems with evidence and direct acceptance impact.

### Decision

Either:

```text
G6=ACCEPTED
WP-00=ACCEPTED
```

or:

```text
G6=BLOCKED
WP-00=NOT_ACCEPTED
```
