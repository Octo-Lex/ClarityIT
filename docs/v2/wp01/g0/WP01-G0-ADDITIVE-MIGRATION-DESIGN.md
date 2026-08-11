# WP-01 G0 — Additive Migration Design

**Gate:** WP01-G0 — Plan/Contract Freeze  
**Authority:** `WP01-AUTH-2026-08-12`  
**Baseline:** `main@33d3802d93c6d3123d9377566f0f3b6fb1360ecb`  
**Compatibility phase:** Phase 2 — Expand  
**Status:** Frozen G1 implementation design candidate

## 1. Decision

WP-01 introduces additive kernel/compatibility schema as **forward revisions after the accepted WP-00 revision `0001`**.

The existing accepted Go migration package remains the only migration implementation path. WP-01 will add the smallest generic forward-series stage needed to apply immutable post-`0001` revisions. It will not create a parallel migration runner, edit any successful `0001` artifact, replay legacy `001-040`, or reinterpret a WP-00 source/target identity.

## 2. Observed current runner boundary

At the WP-01 package baseline, `services/api/internal/migration/apply.go` explicitly describes itself as the **version-0001 executor**. It:

- classifies fresh, P3 and P2 source paths;
- executes the frozen fresh/adoption `0001` artifact chain;
- verifies exactly one artifact-owned revision `0001` with the frozen checksum;
- requires the frozen governed WP-00 target fingerprint `9881c93e...`;
- records a migration run with target version `0001`;
- then commits.

There is no generic post-`0001` revision series in the accepted path. Therefore G1 requires a bounded forward-series extension rather than pretending the existing executor already supports WP-01 revisions.

## 3. Frozen WP-00 boundary

The following remain unchanged:

- fresh source classification;
- P3 source fingerprint/classification/adoption path;
- P2 v3.2 source fingerprint/classification/adoption path;
- historical v3.1 P1/P2 recognized/non-executable behavior;
- unknown/drifted source fail-closed behavior;
- all version-`0001` artifact bytes and checksums;
- P3 adoption SHA-256 `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359`;
- baseline checksum `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508`;
- governed WP-00 target fingerprint `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` as the **pre-forward foundation identity**;
- advisory-lock safety and package-verification principles;
- legacy `001-040` exclusion.

WP-01 must not call a post-WP01 database `9881c93e...`; that fingerprint identifies the accepted WP-00 structural foundation before forward revisions.

## 4. Target execution shape

The G1 implementation shall evolve the migration command into two ordered stages:

```text
Stage A — establish/verify WP-00 foundation
  source preflight
  -> fresh/P3/P2 version-0001 path as already accepted, OR
     existing exact version-0001 foundation verification
  -> require WP-00 governed foundation identity 9881c93e...

Stage B — forward revision series
  verify packaged forward revision catalog
  -> inspect platform.schema_revisions
  -> reject checksum/history contradictions
  -> apply each pending immutable revision in ascending version order
  -> verify exact successful revision row/checksum
  -> verify WP-01 target invariants/schema manifest
  -> record run/evidence/provenance
```

For a database already beyond `0001`, Stage A validates that its revision ancestry begins with the exact accepted `0001` identity; it must not require the current post-forward structure to equal the pre-forward WP-00 fingerprint.

## 5. Forward revision namespace

WP-01 forward revisions start **after `0001`**. The first implemented revision should be `0002` unless implementation evidence proves an existing reserved version conflicts.

Revision names are monotonically ordered fixed-width numeric strings (`0002`, `0003`, ...). Each revision has:

- version;
- immutable SQL/body bytes;
- SHA-256 checksum;
- semantic name;
- producing source commit/build provenance as applicable;
- transaction classification;
- required predecessor/version range;
- optional verification manifest/invariant identifier.

A successful revision's version/checksum pair is immutable. Changed bytes under an already-successful version are a hard failure.

## 6. Artifact location and embedding

G1 shall introduce a forward-revision location under `migrations/v2/` (recommended `migrations/v2/forward/`) and embed/register those artifacts through the **existing `services/api/internal/migration` package**.

The exact Go internal type names may be chosen during G1, but these semantics are fixed:

- deterministic ordered catalog;
- one authoritative embedded byte source per revision;
- SHA-256 verification before any DDL;
- duplicate version rejected;
- missing predecessor/gap rejected unless an explicitly governed future policy permits it;
- registry/catalog mutation covered by tests.

Do not copy the forward SQL into a second independently maintained executor.

## 7. Transaction semantics

WP-01 schema revisions are expected to be transactional PostgreSQL DDL unless a concrete required operation proves otherwise.

For each transactional forward revision:

1. hold the accepted migration advisory lock;
2. verify package/catalog/checksum state;
3. begin transaction;
4. recheck applicable preconditions/revision state;
5. execute exactly one pending revision body or a deterministic bounded batch only if atomic-series behavior is explicitly proven;
6. insert/update the revision ledger as defined by the forward runner contract;
7. verify revision-specific structural invariants;
8. write run/evidence records in the same transaction where their schema permits;
9. commit;
10. continue to the next pending revision.

A failure rolls back that revision without partially marking it successful. Rerun must not duplicate objects/data.

If a future WP-01 operation is demonstrably nontransactional, it requires explicit pre/post/restart semantics and evidence under the accepted migration contract; G0 does not authorize an implicit nontransactional escape hatch.

## 8. Revision ledger semantics

`platform.schema_revisions` remains the authoritative successful-revision ledger.

Forward support must preserve the exact `0001` row and add later rows rather than rewriting it.

At minimum, each successful forward row must retain the existing ledger's required identity fields and provide an unambiguous version/checksum/success/source provenance. If the existing table lacks a WP-01-required provenance field, an additive compatible mechanism (for example reconciliation/execution receipt data) may carry it; do not destructively alter historical `0001` semantics.

Changed checksum for an already-successful version is rejected before DDL.

## 9. Source-profile semantics after `0001`

P2/P3 source profiles are **adoption provenance for reaching the WP-00 foundation**. They are not new source profiles for every forward revision.

A forward migration run shall preserve the original adoption/foundation provenance and record the exact current revision ancestry/package identity. It must not relabel a P2 database as P3 or assign the frozen P2/P3 source fingerprint to the post-forward schema.

Fresh/P2/P3 paths must converge to the same post-WP01 revision state despite having different pre-`0001` histories.

## 10. Pre-forward foundation proof

For a database with only successful `0001`, Stage B may rely on the accepted WP-00 foundation verification:

- exact successful revision `0001` checksum;
- no contradictory revision history;
- governed foundation fingerprint `9881c93e...`;
- no active/interrupted migration run requiring reconciliation;
- package verification passes.

For a database that already contains successful WP-01 forward revisions, the runner shall validate revision ancestry/checksums and the current WP-01 manifest/invariants rather than attempting to recompute `9881c93e...` over the evolved schema.

## 11. WP-01 target identity

WP-01 needs a new deterministic **post-forward schema manifest/invariant identity** produced by G1 evidence. It is not frozen in G0 because the exact additive SQL has not yet been generated/reviewed.

G1 shall freeze, at minimum:

- exact forward revision artifact SHA-256 values;
- ordered catalog/package digest;
- resulting canonical schema/invariant manifest digest;
- allowed application schema-version range;
- exact revision ledger expectation.

The post-WP01 identity must be reproduced from:

1. fresh install -> `0001` -> forward series;
2. approved P3 adoption -> `0001` -> forward series;
3. approved P2 adoption -> `0001` -> forward series.

All three must converge without manual DDL/data correction.

## 12. G1 additive schema rules

WP-01 Phase-2 expand revisions may:

- create new schemas/tables/types/indexes/constraints/functions needed by the kernel;
- add nullable/default-safe columns where compatibility requires them;
- add new roles/grants only under the signed role/trust model and least privilege;
- seed immutable definitions/reference rows where semantics require deterministic IDs;
- add compatibility mapping/checkpoint structures;
- add inbox/outbox/kernel evidence structures.

They must not:

- drop legacy tables/columns required by v1/rollback/history;
- rename/remove v1 contract fields in place;
- backfill legacy success into Verification/Accepted;
- enable live provider mutation;
- replay old migration directory `001-040`;
- modify successful `0001` bytes;
- make v2 the writer for a v1-owned family without a separately recorded cutover decision;
- embed production data/secrets.

## 13. Feature and writer activation

Consequential v2 behavior remains disabled by default during WP-01 migration expansion.

New v2-only object families may be v2-owned immediately. Existing v1 object families remain v1-owned. The writer-ownership registry/mapping evidence must make this explicit.

Schema presence is not cutover authority.

## 14. Application schema-version compatibility

G1 shall define a machine-readable allowed schema-version range for WP-01 binaries and enforce it before authoritative writes.

At minimum:

- binary older than required schema must fail closed for unsupported writes;
- binary newer than database supported range must fail closed;
- read-only diagnostics may remain available where safe;
- compatibility must not silently use an unsafe legacy execution path.

Exact version-range values are frozen with the implemented forward catalog in G1/A2.

## 15. Forward runner minimum code scope

G1 is authorized to make the smallest changes within the existing migration package needed for:

- forward artifact embedding/catalog;
- ordered pending-revision planning;
- package/checksum verification;
- revision-history validation;
- transactional forward execution;
- exact ledger/provenance recording;
- current-schema verification;
- status/plan/verify output for post-`0001` revisions;
- proof failpoints/tests where needed.

Existing source classification and `0001` execution code should remain behaviorally stable. Refactoring it is permitted only when tests prove identical accepted behavior and is not required merely for style.

## 16. Required G1 migration tests

G1 must prove at least:

1. fresh -> accepted `0001` -> all forward revisions -> target manifest;
2. P3 -> unchanged P3 adoption -> forward revisions -> same target manifest;
3. P2 -> unchanged P2 adoption -> forward revisions -> same target manifest;
4. historical v3.1 remains non-executable at initial adoption;
5. unknown pre-`0001` source still blocks before DDL;
6. exact `0001` bytes/checksums remain unchanged;
7. forward package checksum mutation blocks before DDL;
8. successful forward revision checksum mutation is rejected;
9. transaction failure leaves no partial forward schema/revision success;
10. rerun after rollback converges without duplicates;
11. advisory lock prevents concurrent forward runners;
12. a fully current forward database is a clean no-op;
13. a gap/unknown revision/contradictory revision blocks;
14. post-forward verifier detects missing object/constraint/index/grant/revision drift required by A2;
15. legacy `001-040` are never selected/embedded;
16. required WP-00 G4/G5 regressions remain green.

## 17. Evidence

A2 shall bind:

- runner implementation commit;
- all forward artifact digests;
- ordered package/catalog digest;
- fresh/P2/P3 transcripts;
- revision ledger rows;
- target manifest digest;
- transaction/rerun/lock/checksum negatives;
- schema-version range;
- secret scan;
- inherited WP-00 regression runs.

## 18. Failure classification

Routine absence of forward-series support in the WP-00 runner is **not a blocker**; it is the bounded extension authorized here.

A blocker would require evidence that implementing a safe forward series inside the accepted runner necessarily changes a frozen WP-00 identity/behavior or violates a higher semantic authority and cannot be isolated as described.

## 19. Change control

The following require a governed successor rather than routine G1 discretion:

- editing successful `0001` artifact bytes;
- redefining `9881c93e...` as a different structure;
- introducing a second/parallel authoritative migration runner;
- allowing checksum drift for successful revisions;
- destructive v1 contract removal/cutover in WP-01;
- removing fresh/P2/P3 convergence as an acceptance property.
