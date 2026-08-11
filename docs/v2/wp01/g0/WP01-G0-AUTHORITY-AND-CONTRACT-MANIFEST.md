# WP-01 G0 — A1 Authority and Contract Manifest

**Evidence ID:** A1 — WP-01 Authority and Contract Manifest  
**Gate:** WP01-G0 — Plan/Contract Freeze  
**Authorization:** `WP01-AUTH-2026-08-12`  
**Package integration baseline:** `33d3802d93c6d3123d9377566f0f3b6fb1360ecb`  
**Status:** **G0 CANDIDATE — READY FOR REVIEW/INTEGRATION**  

## 1. Purpose

A1 binds the exact WP-01 authorities and the contract artifacts that implementation may rely upon. It freezes scope and semantic boundaries before additive schema/code work begins.

No provider mutation, provider credential use, Site Runtime execution or WP-02 implementation is included in G0.

## 2. Package activation

WP-01 package-plan PR #35 integrated as:

`33d3802d93c6d3123d9377566f0f3b6fb1360ecb`

Under the integrated authorization, that event activated WP-01 implementation authority through RG-01.

Exact integrated control artifacts:

| Artifact | Blob SHA |
|---|---|
| `docs/v2/ClarityIT_v2_WP-01_Authoritative_Kernel_Foundation_Plan_v0.1.md` | `ada6352ad940ab8d781aa152daf8e290bd73e149` |
| `docs/v2/wp01/WP01-AUTHORIZATION.md` | `ff3d6b1671419b440492186725b315075362779f` |

The authorization date is 12 August 2026 project-local time (UTC+03:00).

## 3. Bound semantic authority set

| Priority | Authority | Exact blob/identity | WP-01 use |
|---:|---|---|---|
| 1 | Product Definition v0.1 | `d44975d1557e8499c4e7613a5cd49115126266b0` | product scope/first-release semantics |
| 2 | Authoritative Execution Kernel v0.1 | `1153fb3bfadb1e603307354dc8b6e361eb44167d` | highest execution-semantics authority |
| 3 | v1-to-v2 Compatibility and Migration v0.1 | `bdf179c677f283591842f5a52e41092a70e0b660` | expand/coexistence/historical truth/one-writer |
| 4 | Layered System Architecture | `9d42a74b39e941509725c1c5dd42a87c9126b9e8` | logical placement/read-write boundaries |
| 5 | Native Pattern Specification v0.1 | `00ce72fab791e8b959549b4845d40b4a48954044` | WP-01-owned patterns/skeletons |
| 6 | Environment Trust and Evidence Custody Profile v0.1 | `8a6d28d538fd0d5525114958329b0592829806a9` | development trust/custody/non-promotion |
| 7 | Delivery Roadmap v0.2 | `89911eb29972d813d75f22d98cf239d2b61784b6` | WP-01/RG-01 boundary and sequence |
| 8 | WP-00 final evidence | `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f` | immutable migration/CI foundation |
| 9 | WP-01 plan | blob `ada6352ad940ab8d781aa152daf8e290bd73e149` | AC-01/workstreams/internal gates/evidence |

Higher semantic authority prevails over lower planning convenience.

## 4. Frozen WP-00 inherited identities

| Identity | Frozen value |
|---|---|
| Final WP-00 evidence integration | `e13c8b734b39afb32ff5e3e4a7281543f33d8a1f` |
| Governed pre-forward foundation fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Baseline revision `0001` checksum | `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508` |
| Composite installation SHA-256 | `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084` |
| P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| P2 v3.2 executable source | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` |
| Historical P1/P2 v3.1 | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` — recognized/non-executable |
| P3 adoption artifact | `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359` |
| PostgreSQL | major 16 |

WP-01 consumes these as foundation identities; it does not rewrite them.

## 5. G0 contract set

The following exact branch blobs form the G0 contract candidate:

| Artifact | Blob SHA | Purpose |
|---|---|---|
| `docs/v2/ClarityIT_v2_Context_Overlay_Contract_v0.1.md` | `6f789efa183b9c93f75dbcdbc2cb8bf0ff7e7fea` | deterministic bounded context, overlays, anti-shadowing, screening, digest, isolation |
| `docs/v2/wp01/g0/WP01-G0-OBJECT-OWNERSHIP-AND-PROHIBITED-WRITES.md` | `57d86c75305d440efb8aed5f0bf99d701decebb1` | single authoritative owner per object family; deny bypass writers |
| `docs/v2/wp01/g0/WP01-G0-STATE-AND-REASON-CODE-APPLICABILITY.md` | `feb520eb0eb8fc754d0315900634ad3111da423f` | exact Kernel states/transitions/reason families; no invented transition edges |
| `docs/v2/wp01/g0/WP01-G0-CONFORMANCE-APPLICABILITY.md` | `387dfcd08146ea5957f34afa8bdaede495f93e61` | K/KT/Native Pattern required vs deferred scope |
| `docs/v2/wp01/g0/WP01-G0-ADDITIVE-MIGRATION-DESIGN.md` | `83081c702caa8e80c26e260436d122d8e92a0115` | Phase-2 expand and post-0001 forward-series design |

This A1 file is the sixth G0 artifact and binds the set after its own integration.

## 6. Runner observation bound into migration design

At the package baseline:

- `services/api/internal/migration/apply.go` blob `6443f46ece84c2e9691b8ca5ab8910a70d10f5f9` explicitly identifies itself as the **version-0001 executor**;
- it verifies artifact-owned revision `0001` exactly;
- it requires the WP-00 governed target before completing the `0001` run;
- the current accepted path has no generic post-`0001` forward revision series.

G0 therefore authorizes G1 to add the **smallest forward-series extension inside the existing migration package**, while leaving `0001` classification/artifacts/identities behaviorally unchanged. A parallel runner is prohibited.

## 7. Frozen WP-01 semantic decisions

Implementation SHALL preserve:

1. PostgreSQL authoritative; NATS/transport derived.
2. authoritative transition + audit + outbox atomic before publish.
3. inbox dedupe before/with consumer state transition.
4. typed truth levels remain distinct.
5. proposed Operation Packets immutable; material change creates successor.
6. PolicyDecision, ApprovalDecision and AuthorityGrant remain separate.
7. exact grant binding to workspace/Case/packet/resource/version/capability/workload/route/policy/time/nonce/use.
8. `outcome_unknown` and Verification `inconclusive` remain first-class states.
9. provider/executor result remains a claim.
10. Verification requires exact versioned spec and independent allowed evidence.
11. first-release Accepted requires identified accountable human decision after passed Verification.
12. corrections/compensation are successor lineages.
13. reasoning is credentialless and non-authoritative.
14. Effect Broker is sole dispatch API.
15. workspace scope is mandatory server-side.
16. context/projections/caches/search are derived and rebuildable.
17. overlay policy only tightens authority.
18. lower-authority content cannot shadow reserved authoritative namespaces.
19. historical truth remains weaker unless current evidence establishes stronger truth.
20. exactly one authoritative writer exists per object family.
21. WP-01 performs zero live provider mutations.

## 8. G0 scope boundary

G0 contains contracts/evidence only. It does not create kernel schema, activate feature flags, call providers, add provider credentials, alter the migration database, or start WP-02.

After G0 integration, G1 may implement:

- additive canonical kernel/compatibility schema;
- minimal forward migration-series support inside the accepted runner;
- PrincipalRef/workspace constraints;
- canonical object tables and inbox/outbox/mapping foundations;
- A2 migration/schema evidence.

## 9. Required CI inherited unchanged

All WP-01 integrations retain:

1. `Frontend (typecheck · test · build)`;
2. `Worker (Python)`;
3. `Backend (Go)`;
4. `G5 Foundation Gate`.

G0 does not weaken or replace them.

## 10. G0 acceptance check

WP01-G0 is ready to close when:

- plan and authorization are integrated;
- this A1 set reconstructs;
- Context Overlay Contract is coherent with Kernel/Native Pattern authority;
- ownership/prohibited-write matrix has one authoritative writer per family;
- state/reason map contains no invented weakening state or transition edge;
- conformance map requires every WP-01 Kernel scenario and clearly defers later live scope;
- additive migration design preserves all frozen WP-00 identities and uses no parallel runner;
- inherited CI is green on the final G0 candidate;
- no unresolved review finding demonstrates a semantic contradiction.

On integration, the decision is:

```text
WP01-G0=PASS
A1=FROZEN
NEXT=WP01-G1
```

No additional client authorization is required for G1.
