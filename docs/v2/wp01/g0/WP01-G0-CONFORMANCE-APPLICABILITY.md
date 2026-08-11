# WP-01 G0 — Kernel and Native Pattern Conformance Applicability

**Gate:** WP01-G0 — Plan/Contract Freeze  
**Authority:** `WP01-AUTH-2026-08-12`  
**Baseline:** `main@33d3802d93c6d3123d9377566f0f3b6fb1360ecb`  
**Status:** Frozen applicability matrix candidate

## 1. Purpose

This matrix prevents WP-01 from either omitting required Kernel proof or accidentally absorbing later-package scope. `Required` means RG-01 needs evidence. `Synthetic` means the property is proven with deterministic fake/no-op fixtures because `LIVE_PROVIDER_MUTATIONS=0`. `Deferred` means the higher-level contract remains valid but live implementation belongs to the named later package.

## 2. Kernel invariant applicability

| Kernel invariant | WP-01 disposition | Required evidence |
|---|---|---|
| K-01 PostgreSQL authoritative; transport derived | Required | A3/A4 projection rebuild and transport-loss/duplicate proof |
| K-02 typed truth records remain separate | Required | A2/A3/A6 schema/state and negative promotion tests |
| K-03 immutable proposed Operation Packet | Required | A5 deterministic packet digest/successor proof |
| K-04 PolicyDecision, ApprovalDecision, AuthorityGrant separate | Required | A5 separation and approval-not-execution negatives |
| K-05 exact grant binding / freshness / use | Required | A5 mismatch matrix |
| K-06 sole governed dispatch via Effect Broker | Required synthetic | A5 fake route + bypass negatives; no live provider |
| K-07 idempotency / one logical submission | Required synthetic | A5 concurrent duplicate proof |
| K-08 ambiguous submission is outcome_unknown | Required synthetic | A5/A6 ambiguity/reconciliation/no-blind-retry proof |
| K-09 provider output remains claim | Required synthetic | A6 provider-completed cannot verify/accept |
| K-10 independent exact VerificationSpec | Required synthetic | A6 pass/fail/inconclusive and invalid-input negatives |
| K-11 explicit human outcome acceptance | Required | A6 identified-human acceptance prerequisite |
| K-12 corrections/compensation are successors | Required synthetic | A6 correction/compensation lineage reconstruction |

## 3. Kernel KT scenarios

| Scenario | WP-01 disposition | Proof boundary |
|---|---|---|
| KT-01 happy path | Required synthetic | propose -> policy/approval -> grant -> fake submit -> fake receipt -> fresh fake Observation -> independent Verification -> identified-human acceptance |
| KT-02 packet modification/successor | Required | proposed bytes immutable; new packet/digest; old grant unusable |
| KT-03 stale baseline/resource | Required | blocks at fresh preflight before fake submission |
| KT-04 policy/approval/grant/subject/route/nonce failures | Required | complete AUTH mismatch matrix |
| KT-05 concurrent duplicate commands | Required synthetic | one attempt and at most one fake submission per logical key |
| KT-06 restart during polling | Required synthetic | durable fake provider-operation ref/checkpoint; resume without duplicate |
| KT-07 ambiguous submission | Required synthetic | `outcome_unknown`; reconcile; no blind new submission |
| KT-08 provider-completed/result mismatch | Required synthetic | ResultClaim persists; Verification cannot pass |
| KT-09 provider running/health failure | Required synthetic | provider state does not cause Accepted; verifier fails/inconclusive per spec |
| KT-10 verifier unavailable | Required synthetic | `inconclusive` unless exact spec defines deterministic failure |
| KT-11 cancellation boundary | Required synthetic | cancellation before/after submission boundary classified and manifested |
| KT-12 correction/compensation | Required synthetic | correction-required, compensation-required and compensated successor lineages |
| KT-13 Site Runtime disconnection | Deferred WP-04 | WP-01 proves only unavailable/unimplemented site route fails closed; no spool/journal/local polling claim |
| KT-14 secret scan | Required | no provider secret introduced; source/fixture/context/message/evidence scan |
| KT-15 evidence reconstruction | Required | all mandated terminal lineages reconstruct |
| KT-16 historical truth | Required | legacy claims create zero passed Verification/Accepted outcomes |

## 4. Native Pattern applicability

### P-02 — Identity

**WP-01 owns:** PC-ID-01..05 for implemented PrincipalRef/workspace/workload/source identity semantics.

Required proof includes typed principal categories, server-side workspace attribution, non-shared execution workload identity semantics, provenance, and reasoning-agent power negatives.

### P-03 — Typed Records / Truth

**WP-01 owns:** PC-TR-01..05.

Finding/Observation/proposal/PolicyDecision/ApprovalDecision/AuthorityGrant/Attempt/ProviderReceipt/ResultClaim/Verification/OutcomeDecision remain distinct and cannot be promoted by naming conventions or UI status.

### P-05 — Bounded Context / Context Overlay

**WP-01 owns:** PC-BC-01..09.

The controlling WP-01 contract is `ClarityIT_v2_Context_Overlay_Contract_v0.1.md`. RG-01 requires deterministic digest, scope/freshness provenance, monotonic tightening, anti-shadowing, topology limits, omission/gap evidence, secret exclusion and workspace isolation.

### P-06 — Idempotent Prepare/Intent

**WP-01 owns:** PC-IP-01..05 where kernel behavior can be proven through deterministic fake Prepare/route fixtures.

Live provider-specific prepare semantics remain WP-02+.

### P-08 — Adapter/Effect Broker boundary

**WP-01 owns:** PC-AD-01..05 at the kernel/fake-route contract level.

Required: Effect Broker sole dispatch, exact grant/resource/capability binding, idempotency, typed claims, no direct provider bypass. Live Proxmox conformance belongs to WP-02.

### P-09 — Credentialless reasoning / scoped secrets

**WP-01 owns:** PC-CS-01..05 as trust foundation: reasoning has no credentials; only secret references/entitlements may be represented; generic credential access is prohibited.

**Deferred:** live destination-bound credential resolution/injection PC-CS-06..09 to WP-02 central route and WP-04 Site Runtime as applicable.

WP-01 may implement entitlement schemas/policy evaluators but must not resolve/inject a real provider credential.

### P-10 — Independent Verification

**WP-01 owns:** PC-IV-01..05 through deterministic independent verifier fixtures.

Executor/provider/UI/agent results are invalid proof inputs unless the exact VerificationSpec explicitly treats a source-attributed observation as evidence; executor self-assertion alone is never sufficient.

### P-11 — Successor/Correction

**WP-01 owns:** PC-SC-01..05.

Required lineages include supersession, correction, safe-failure retry, unknown reconciliation and compensation, preserving original immutable records.

### P-18 — Workspace Isolation

**WP-01 owns foundation:** PC-WS-01 and the implemented trust/route-binding portion of PC-WS-02.

Isolation proof spans PostgreSQL, server APIs, events/messages, storage/evidence references, context/cache/search fixtures and background jobs. No cross-workspace provider execution exists in WP-01.

**Deferred:** deployment/production operational controls PC-WS-03..10 primarily to WP-10, except preservation of accepted WP-00 installation/upgrade controls.

## 5. Skeleton-only patterns

The following may be introduced only to the degree required by the WP-01 kernel contract and do not claim full pattern acceptance:

- P-01 Case skeleton;
- P-04 answer/source-reference skeleton;
- P-07 capability-registry skeleton.

Their complete product/provider conformance remains downstream.

## 6. Explicitly deferred package scope

| Capability | Deferred to | WP-01 boundary |
|---|---|---|
| live `compute.virtual_machine.start@1` / Proxmox mutation | WP-02 | no live provider mutation/credential in WP-01 |
| complete My Work / Case Workspace / Resource UX and pilot | WP-03 | only kernel/API/domain foundations as needed |
| Site Runtime, private-zone/offline journal/spool | WP-04 | unavailable site route fails closed only |
| reviewed knowledge/skills | WP-05 | reserve approved-knowledge authority class only |
| Signals/Routines | WP-06 | no event-triggered authority introduced |
| Projects/software-delivery context | WP-07 | no project authority introduced |
| multi-target execution | WP-08 | WP-01 single logical target semantics only |
| second provider / extension SDK | WP-09 | provider-neutral core contracts only |
| production hardening/rollout | WP-10 | development trust/custody boundary retained |

## 7. Evidence mapping

| Evidence | Conformance content |
|---|---|
| A1 | authority set, this applicability matrix, Context Overlay Contract identity |
| A2 | P-02/P-03/P-18 schema/principal/workspace foundation |
| A3 | typed states/provenance/concurrency |
| A4 | K-01 persistence/transport/replay/restart |
| A5 | P-06/P-08/P-09 authority/broker/idempotency/no-live-provider |
| A6 | P-10/P-11 KT lineage/Verification/outcome/evidence |
| A7 | P-05/P-09/P-18 context/trust/isolation |
| A8 | KT-16 compatibility/historical truth and fresh/P2/P3 regression |
| A9 | complete K/KT/Pattern crosswalk and final release identities |

## 8. RG-01 conformance rule

A scenario marked Required/Synthetic must be PASS or RG-01 is blocked. A deferred scenario cannot be claimed implemented by WP-01 and cannot be converted into a new RG-01 gate beyond the explicit fail-closed boundary stated here.

`LIVE_PROVIDER_MUTATIONS=0` remains mandatory evidence, not a limitation to be worked around.

## 9. Change control

Moving a deferred live capability into WP-01, weakening a required Kernel invariant, or declaring a required scenario not applicable changes the package acceptance boundary and requires governed successor authority rather than routine implementation discretion.
