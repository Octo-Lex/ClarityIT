# ClarityIT v2 — WP-01 Authoritative Kernel Foundation Plan

**Package:** WP-01 — Authoritative Kernel Foundation  
**Version:** 0.1  
**Status:** Authorized package plan; implementation authority activates when this plan is integrated to `main`  
**Authorization ID:** `WP01-AUTH-2026-08-12`  
**Authorization date:** 12 August 2026 (project local time, UTC+03:00)  
**Package baseline:** `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`  
**Entry condition:** WP-00 G6 accepted; AC-00-01 through AC-00-30 PASS; A1-A7 complete; unresolved Sev1/Sev2 = 0; closure recorded by [`docs/migration/wp00/g6/G6-EVIDENCE-CROSSWALK.md`](../migration/wp00/g6/G6-EVIDENCE-CROSSWALK.md) and [`docs/migration/wp00/g6/G6-APPROVALS.md`](../migration/wp00/g6/G6-APPROVALS.md)  
**Final gate:** RG-01 — Authoritative Kernel Foundation acceptance

> **Package decision:** WP-01 establishes the canonical v2 domain, truth, authority, persistence, evidence, context, trust, isolation, and compatibility foundation alongside the stabilized v1 spine. It MUST NOT perform a live consequential provider mutation. Execution semantics are proven with deterministic fake/no-op fixtures only. The first real provider-neutral mutation remains WP-02.

---

## 1. Authority and precedence

WP-01 is the first v2 feature-foundation package after the accepted WP-00 migration baseline. It converts the Product, Kernel, Compatibility, Architecture, Native Pattern, Trust, and Roadmap contracts into a bounded implementation package with explicit workstreams, intermediate gates, evidence artifacts, acceptance criteria, failure scenarios, and RG-01 closure.

This plan does not modify higher semantic authorities. If implementation convenience conflicts with them, the higher authority governs.

### 1.1 Bound authority set

`WP01-AUTH-2026-08-12` ratifies the exact authority set below for WP-01. The durable authorization record is [`wp01/WP01-AUTHORIZATION.md`](wp01/WP01-AUTHORIZATION.md).

| Priority | Authority | Exact repository identity | WP-01 use |
|---:|---|---|---|
| 1 | Product Definition v0.1 | `ClarityIT_v2_Product_Definition_v0.1.md`, blob `d44975d1557e8499c4e7613a5cd49115126266b0` | Product scope and first-release semantics |
| 2 | Authoritative Execution Kernel v0.1 | `ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md`, blob `1153fb3bfadb1e603307354dc8b6e361eb44167d` | Highest execution-semantics authority; K-01..K-12, canonical objects, states, persistence, verification and evidence |
| 3 | v1-to-v2 Compatibility and Migration v0.1 | `ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md`, blob `bdf179c677f283591842f5a52e41092a70e0b660` | Additive coexistence, one-writer rule, historical truth and migration safety |
| 4 | Layered System Architecture | `ClarityIT-v2-Layered-System-Architecture.md`, blob `9d42a74b39e941509725c1c5dd42a87c9126b9e8` | Logical placement and read/write boundaries |
| 5 | Native Pattern Specification v0.1 | `ClarityIT_v2_Native_Pattern_Specification_v0.1.md`, blob `00ce72fab791e8b959549b4845d40b4a48954044` | WP-01-owned P-02, P-03, P-05, P-06, P-08, P-09, P-10, P-11, P-18 and required skeletons |
| 6 | Environment Trust and Evidence Custody Profile v0.1 | `ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md`, blob `8a6d28d538fd0d5525114958329b0592829806a9` | Development trust/custody placement and production non-promotion boundary |
| 7 | Delivery Roadmap v0.2 | `ClarityIT_v2_Delivery_Roadmap_v0.2.md`, blob `89911eb29972d813d75f22d98cf239d2b61784b6` | WP-01 scope, RG-01 and sequencing |
| 8 | WP-00 final evidence | `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`; [`G6-EVIDENCE-CROSSWALK.md`](../migration/wp00/g6/G6-EVIDENCE-CROSSWALK.md); [`G6-APPROVALS.md`](../migration/wp00/g6/G6-APPROVALS.md) | Accepted migration/CI foundation: AC-00 30/30 PASS, A1-A7 complete, Sev1/Sev2=0 |
| 9 | This plan | This file after integration | WP-01 execution and closure contract |

### 1.2 Frozen WP-00 inputs

WP-01 MUST preserve these accepted inputs unless a demonstrated defect requires a separately governed successor:

- WP-00 final integration `e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`;
- governed target fingerprint `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6`;
- baseline checksum `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508`;
- composite installation SHA-256 `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084`;
- P3 source `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`;
- executable P2 v3.2 successor `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`;
- historical P1/P2 v3.1 identity `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`, recognized/non-executable;
- P3 adoption artifact SHA-256 `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359`;
- PostgreSQL major 16;
- required `main` checks: `Frontend (typecheck · test · build)`, `Worker (Python)`, `Backend (Go)`, `G5 Foundation Gate`.

No WP-01 change may silently redefine a WP-00 identity or weaken its blocking CI predicate.

---

## 2. Objective and package boundary

### 2.1 Objective

Introduce the canonical v2 domain and truth model beside the stabilized v1 spine, with enough executable kernel behavior to prove state, authority, persistence, isolation, context composition, verification semantics, evidence reconstruction, and compatibility **without a live consequential provider mutation**.

RG-01 must leave a kernel foundation on which WP-02 can add `compute.virtual_machine.start@1` without inventing new truth, authority, persistence, verification, evidence, isolation, or replay semantics.

### 2.2 In scope

WP-01 SHALL implement or establish:

1. canonical v2 schemas for Cases, Resources, Provider Bindings, Observations, Operation Packets, Policy Decisions, Approval Decisions, Authority Grants, Execution Attempts, Provider Receipts/Result Claims as contract records, Verification Specs, Verifications, Verification Evidence, Outcome Decisions, Evidence Manifests, inbox/outbox and required compatibility/mapping records;
2. PrincipalRef for human, reasoning agent, service, policy, execution workload and external source identities;
3. legal/illegal state machines, optimistic aggregate concurrency, successor lineage and explicit unknown/inconclusive states;
4. deterministic Operation Packet canonicalization, digest/signature envelope, proposal freeze and successor semantics;
5. policy evaluation, approval separation, grant lifecycle, separation of duties, replay protection and exact scope validation;
6. an Effect Broker skeleton as the sole dispatch API, with deterministic fake/no-op route/capability fixtures for tests only;
7. versioned verifier contracts plus deterministic read-only verifier fixtures proving passed/failed/inconclusive semantics;
8. evidence manifests for synthetic happy, blocked, rejected, failed, cancelled, unknown, inconclusive, superseded, compensation-required, compensated and successor lineages;
9. transactional outbox/inbox, replay, duplicate delivery, restart and lease-loss semantics;
10. Resource-aware bounded context and deterministic overlay composition with monotonic tightening, anti-shadowing, provenance, screening, omissions and topology limits;
11. trust schemas/policy evaluation for workload identity, route binding, secret references and destination-bound credential-broker entitlements, without a live credential-injecting connector;
12. additive v1/v2 coexistence with v1 read compatibility, historical truth classification, one-writer ownership and safe mapping/backfill foundations;
13. workspace isolation across database/API/event/storage-reference/cache/search fixtures and background processing;
14. fresh install and approved P2/P3 adoption/upgrade compatibility under the accepted WP-00 runner and blocking CI.

### 2.3 Explicit exclusions

WP-01 MUST NOT implement, enable, or claim acceptance for:

- any live provider mutation;
- any real Proxmox request, provider credential, UPID workflow, provider polling or live reconciliation;
- live `compute.virtual_machine.start@1` adapter conformance; that is WP-02;
- provider-specific core packet semantics;
- Site Runtime, private-zone execution, edge-gateway execution, offline journals/spools or disconnected execution; that is WP-04;
- complete My Work/Case Workspace/Resource UX, live WebSocket product progress, accessibility acceptance or controlled pilot; that is WP-03;
- reviewed knowledge/skills, Signals/Routines, Project delivery integration, multi-target execution, second provider, extension SDK or production rollout;
- production cutover or production trust-domain promotion;
- arbitrary shell, SQL, SSH, Kubernetes, database, browser, desktop, Git, cloud or generic authenticated-HTTP mutation interfaces;
- destructive removal of legacy tables/history;
- promotion of legacy provider/agent/operator success claims to passed Verification or Accepted;
- a generic credential broker callable by agents, browsers, playbooks or routines.

### 2.4 Zero-live-provider rule

During WP-01:

- every execution route is deterministic and test-only;
- provider credentials are absent from fixtures/configuration/evidence;
- no acceptance test depends on managed-system mutation egress;
- fake routes may emit typed accepted/running/completed/failed/unknown receipts only to exercise kernel semantics;
- RG-01 evidence MUST state `LIVE_PROVIDER_MUTATIONS=0`.

A real consequential provider call is a package-boundary violation.

---

## 3. Frozen semantic decisions

WP-01 implementation MUST preserve:

1. PostgreSQL is authoritative; NATS is transport only.
2. Authoritative transition + audit + outbox commit atomically before publish.
3. Consumers deduplicate through inbox before/with state transition.
4. Typed truth levels remain separate; no generic `succeeded` promotion.
5. Operation Packets become immutable when proposed; material edits create successors.
6. PolicyDecision, ApprovalDecision and AuthorityGrant remain separate.
7. Grants bind exact workspace, Case, packet digest, resource/version, capability/constraints, workload, route, policy revision, time, nonce and use limit.
8. `outcome_unknown` is a real state and forbids blind resubmission.
9. Provider completion is a claim, not Verification or Accepted.
10. Verification requires exact versioned spec and fresh sealed inputs.
11. First-release Accepted requires explicit identified human OutcomeDecision after passed Verification.
12. Corrections/compensation are successor lineages; prior records are not overwritten.
13. Reasoning is credentialless and non-authoritative.
14. Effect Broker is the sole dispatch API.
15. Workspace scope is mandatory server-side and fails closed if missing/ambiguous.
16. Context/projections/caches/search are derived and rebuildable.
17. Overlay policy is monotonic: narrower layers may tighten but never relax organization/workspace authority.
18. Personal/retrieved/prior-Case/generated content cannot shadow authoritative namespaces.
19. Historical truth remains weaker than current verified truth unless current evidence establishes otherwise.
20. Each object family has exactly one authoritative writer at a time; bidirectional authoritative dual-write is prohibited.

---

## 4. Logical module ownership

| Module | Owns / provides | Must not do |
|---|---|---|
| Domain kernel | Cases, Resources, packets, outcome decisions, transitions, successor lineage | Call providers or create execution claims |
| Principal/identity layer | PrincipalRef and workload/source identity bindings | Treat a UI session as execution workload |
| Authority service | Policy decisions, approval requirements, grant issue/revoke | Use provider credentials or accept outcomes |
| Effect Broker | Sole dispatch command, preflight contract, reservation, attempt/idempotency, dispatch record | Call a real provider in WP-01 or mark Verified |
| Execution fixture | Deterministic fake/no-op route for contract tests | External mutation or real secret resolution |
| Observation contract | Typed source-attributed observations and freshness metadata | Treat stored observation as permanently fresh |
| Verification service | VerificationSpec and passed/failed/inconclusive evaluation | Reuse execution success as proof |
| Evidence service | Append-only manifests, digests, artifact refs, redaction metadata | Rewrite lineage |
| Messaging foundation | v2 envelopes, outbox, inbox, replay/rebuild | Create truth from message delivery |
| Context builder | Bounded context and deterministic overlays | Grant authority or leak cross-workspace/secret data |
| Compatibility layer | Mapping, read compatibility, historical classification, unsafe-write rejection contracts | Fall through to legacy unsafe execution |

---

## 5. Workstreams and evidence

### WS01-01 — Authority freeze, Context Overlay Contract and implementation map

Deliver:

- this plan and `WP01-AUTH-2026-08-12` integrated;
- `ClarityIT_v2_Context_Overlay_Contract_v0.1.md` approved before RG-01, covering overlay order, authority classes, monotonic tightening, anti-shadowing, screening states, limits, deterministic composition digest, invalidation and evidence;
- object/owner/prohibited-write matrix;
- state-transition and reason-code applicability maps;
- KT and Native Pattern applicability matrix;
- additive migration design bound to the WP-00 runner.

Evidence: **A1 — WP-01 Authority and Contract Manifest**.

### WS01-02 — Canonical schema, identity, Case/Resource skeleton

Deliver:

- additive checksummed kernel/compat revisions;
- Case and Resource/ProviderBinding skeletons with workspace, identity, version and lineage;
- typed PrincipalRef categories;
- Observation contract;
- tables for packet, policy/approval/grant, attempts, receipts/claims, VerificationSpec/Verification/evidence, outcomes and EvidenceManifest;
- inbox/outbox and compatibility idempotency/mapping foundations;
- explicit constraints/indexes for workspace and identity integrity;
- feature flags off and no destructive legacy DDL.

Evidence: **A2 — Canonical Schema and Principal Manifest**.

### WS01-03 — State machines, optimistic concurrency and typed provenance

Deliver exactly the Kernel states and transitions:

- Operation Packet states: `draft`, `proposed`, `superseded`, `withdrawn`, `expired`;
- AuthorityGrant states: `issued`, `reserved`, `consumed`, `revoked`, `expired`; a permitted reservation release transitions `reserved -> issued`, it does **not** invent a `released` state;
- ExecutionAttempt states: `created`, `preflight`, `dispatchable`, `submitting`, `submitted`, `running`, `provider_completed`, `provider_failed`, `blocked`, `cancelled`, `outcome_unknown`;
- Verification: `pending -> running -> passed|failed|inconclusive`;
- OutcomeDecision: `pending -> accepted|rejected|correction_required|compensation_required`;
- Case lifecycle as a rebuildable projection only;
- expected aggregate-version conflicts;
- immutable successor/supersession lineage;
- causal/provenance ordering independent of browser/message arrival.

Evidence: **A3 — Transition, Concurrency and Provenance Evidence**.

### WS01-04 — Transactional messaging, replay and recovery

Deliver:

- versioned v2 envelopes with message/workspace/aggregate/version/correlation/causation/actor/time/payload-digest fields;
- authoritative transition + audit + outbox atomicity;
- inbox dedupe before/with state transition;
- duplicate replay returning recorded result;
- durable synthetic worker checkpoints;
- projection rebuild fixtures;
- restart and lease-loss proof;
- unsupported-version/poison-message quarantine with reason evidence, using accepted durable-disposition foundations.

Evidence: **A4 — Persistence, Messaging and Recovery Evidence**.

### WS01-05 — Packet, authority and Effect Broker skeleton

Deliver:

- deterministic packet canonicalization/SHA-256/signature-envelope interface;
- immutable proposal and successor behavior;
- deterministic PolicyDecision contract;
- ApprovalDecision binding to exact packet digest;
- separation-of-duties enforcement;
- grant issue/reserve/consume/revoke/expire lifecycle and permitted `reserved -> issued` release semantics only when provider submission provably did not occur and policy permits retry;
- exact grant scope validation;
- deterministic logical idempotency key/uniqueness;
- Effect Broker as sole dispatch entry;
- deterministic fake/no-op route/capability fixture;
- atomic reservation + attempt + dispatch record;
- no external credential/provider call.

Evidence: **A5 — Packet, Authority, Broker and Idempotency Evidence**.

### WS01-06 — Verification, outcomes, evidence and successors

Deliver:

- immutable versioned VerificationSpec;
- read-only fake verifier with exact-spec/freshness/pass/fail/inconclusive/retry-evidence semantics;
- rejection of executor flags, agent conclusions and UI/projection state as Verification inputs;
- explicit human acceptance requirement;
- synthetic result-state mismatch and health-failure paths;
- successor relations `corrects`, `retries_after_safe_failure`, `reconciles_unknown`, `compensates`;
- blind retry rejection after ambiguous submission;
- EvidenceManifests for happy, blocked, rejected, failed, cancelled, unknown, inconclusive, superseded, compensation-required, compensated and successor lineages;
- immutable artifact refs/digests/redaction metadata;
- independent reviewer reconstruction fixture.

Evidence: **A6 — Verification, Outcome, Successor and Evidence Pack**.

### WS01-07 — Context overlays, trust foundation and anti-shadowing

Deliver:

- Context Bundle ID/digest, exact scope/versions/Observation refs/topology/ownership/health/prior-Case/read-capability/exclusion/builder metadata;
- ordered overlays: organization -> workspace -> Case/Resource -> role/task -> personal draft;
- source identity, authority class, version/time, sensitivity, screening state and digest per entry;
- monotonic policy tightening;
- reserved authoritative namespaces and collision reject/quarantine behavior;
- deterministic composition record covering applied/rejected/omitted/conflicting entries;
- topology depth/relation/target-count limits;
- workspace/access/version keyed cache and invalidation contract;
- `screened`, `quarantined`, `unscreened` handling;
- secret references only, never secret values;
- workload identity, route, secret-reference and credential-entitlement schemas/policy evaluator only; no live credential injection.

Evidence: **A7 — Context, Anti-Shadowing, Trust and Isolation Evidence**.

### WS01-08 — Compatibility, historical truth and one-writer coexistence

Deliver:

- additive coexistence with explicit writer ownership by object family;
- consequential v2 features disabled by default;
- stable identity mapping only when semantics are unchanged; new semantic aggregates receive new IDs;
- historical classification preserving legacy claims;
- backfill/mapping fixtures creating **zero passed Verifications and zero Accepted outcomes** from legacy rows;
- safe compatibility reads/truth classification;
- fail-closed unsafe legacy execution behavior in WP-01 tests;
- no destructive contract DDL;
- bounded idempotent/restartable mapping/checkpoint behavior where used;
- fresh/P2/P3 migration regression through the accepted runner.

Evidence: **A8 — Compatibility and Historical-Truth Evidence**.

### WS01-09 — Workspace isolation and security conformance

Deliver:

- workspace constraints/query guards for all WP-01 records;
- cross-workspace API/event/cache/search/object-reference negatives;
- background-job workspace attribution;
- agent/principal negatives for approval/grant/dispatch/Verification/outcome powers;
- secret scanning of source/fixtures/packets/messages/logs/receipts/evidence;
- no direct kernel-table write from reasoning/browser/compatibility code;
- no provider-call surface reachable from Control API or reasoning code.

Evidence is bound into A2/A5/A7 and the final release manifest.

### WS01-10 — RG-01 conformance and release evidence

Deliver:

- complete AC-01 crosswalk;
- K-01..K-12 evidence;
- applicable KT and Native Pattern conformance evidence;
- final Context Overlay Contract;
- fresh/P2/P3 migration regression;
- required CI green;
- secret/isolation evidence;
- synthetic lineage reconstruction including cancelled, compensation-required and compensated paths;
- zero unresolved Sev1/Sev2 defects;
- **A9 — RG-01 Release Evidence Manifest**, binding A1-A8 and exact commits/runs/digests;
- Architecture, Backend, Database, Security and Quality decisions; Product participates where product semantics are affected.

---

## 6. Package gates

| Gate | Decision | Minimum evidence | Exit authority |
|---|---|---|---|
| **WP01-G0 — Plan/Contract Freeze** | Is the package executable without scope ambiguity? | Integrated plan/authorization, A1 foundation, Context Overlay Contract structure, object/owner/test applicability maps | Additive implementation may begin |
| **WP01-G1 — Canonical Schema Foundation** | Are objects/principals/workspace constraints safely additive? | A2, fresh/install upgrade migration proof, no destructive DDL, features off | Domain/state logic may rely on schema |
| **WP01-G2 — State and Persistence Kernel** | Are transitions, concurrency, outbox/inbox and replay correct? | A3+A4 | Authority/broker/verifier work may rely on persistence |
| **WP01-G3 — Authority/Dispatch Skeleton** | Are packet, policy, approvals, grants, broker and idempotency fail-closed? | A5, direct-bypass negatives, no-provider-mutation proof | Synthetic lineages may execute end-to-end |
| **WP01-G4 — Verification/Context/Compatibility** | Are independent proof, successors, overlays, isolation and historical truth coherent? | A6+A7+A8 | Final RG-01 conformance may begin |
| **RG-01 — Authoritative Kernel Foundation** | Is the kernel foundation safe for a separately authorized WP-02? | A1-A9, AC-01-01..40 PASS, owner decisions, blocking CI green, Sev1/Sev2=0 | WP-01 accepted; WP-02 remains separately authorized |

A later internal-gate pass cannot waive an earlier failed property.

---

## 7. Acceptance criteria — AC-01

RG-01 is unconditional. Every criterion below must be evidenced PASS.

### Authority and package integrity

- **AC-01-01:** implementation descends from accepted WP-00 `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f` and preserves frozen WP-00 identities.
- **AC-01-02:** no live provider mutation, provider credential, Site Runtime execution or production cutover occurs; `LIVE_PROVIDER_MUTATIONS=0` is evidenced.
- **AC-01-03:** Context Overlay Contract is versioned, approved and bound into A1/A9.
- **AC-01-04:** object/service ownership and prohibited-write boundaries are documented and enforced by tests.

### Schema, identity and workspace

- **AC-01-05:** canonical Case, Resource/Binding, Observation, Packet, authority, attempt, claim, Verification, outcome, evidence, inbox/outbox and compatibility foundations exist as additive checksummed revisions.
- **AC-01-06:** every authoritative aggregate/message carries workspace, schema version, actor/workload, correlation, causation and time; applicable aggregates use monotonic aggregate version.
- **AC-01-07:** PrincipalRef supports human, reasoning, service, policy, execution workload and external source identities with workspace/provenance.
- **AC-01-08:** cross-workspace DB/API/event/cache/search/storage-reference fixtures fail closed; missing/ambiguous workspace scope is rejected.
- **AC-01-09:** reasoning/shared identities cannot exercise approval, grant, direct dispatch, Verification completion or acceptance powers outside explicit permissions.

### Typed truth and state

- **AC-01-10:** Finding, Observation, proposal, PolicyDecision, ApprovalDecision, AuthorityGrant, Attempt, ProviderReceipt/ResultClaim, Verification and OutcomeDecision remain distinct typed records.
- **AC-01-11:** all specified legal state transitions pass and illegal transitions fail without partial authoritative mutation.
- **AC-01-12:** expected aggregate-version conflict rejects stale concurrent writes.
- **AC-01-13:** correction/supersession creates successors while original bytes/digests remain reconstructable.
- **AC-01-14:** `outcome_unknown` and `inconclusive` remain explicit non-success/non-failure states.

### Persistence and messaging

- **AC-01-15:** authoritative state, audit and outbox commit atomically in PostgreSQL.
- **AC-01-16:** duplicate delivery produces one authoritative transition and one inbox result per consumer.
- **AC-01-17:** projections/timelines rebuild from PostgreSQL without NATS history while preserving typed class/source and causal ordering.
- **AC-01-18:** restart and lease loss resume from durable state without duplicating logical work or rewriting history.
- **AC-01-19:** unsupported/malformed messages and poison events reach bounded durable terminal disposition; transport failure cannot create product truth.

### Packet and authority

- **AC-01-20:** packet canonicalization is deterministic; proposal is immutable with reproducible SHA-256/signature envelope.
- **AC-01-21:** material bound-field change creates a successor and prior approval/grant cannot authorize it.
- **AC-01-22:** PolicyDecision, ApprovalDecision and AuthorityGrant remain separate; approval alone cannot create an executable attempt.
- **AC-01-23:** mismatch of workspace, Case, packet, resource/version, capability/parameters, workload, route, policy revision, time, nonce or use limit blocks before synthetic submission.
- **AC-01-24:** separation-of-duties conflicts are rejected deterministically and evidenced.
- **AC-01-25:** concurrent duplicate broker commands create one attempt and at most one synthetic submission per logical idempotency key.
- **AC-01-26:** browser, reasoning worker, compatibility path and Control API cannot bypass the Effect Broker to a provider mutation path.

### Verification, outcome and evidence

- **AC-01-27:** provider-completed synthetic receipts cannot directly create Verification or Accepted.
- **AC-01-28:** Verification uses exact spec version and fresh sealed inputs; pass/fail/inconclusive are reproducible.
- **AC-01-29:** executor flags, agent conclusions and UI/projection state are rejected as Verification inputs.
- **AC-01-30:** Accepted requires passed Verification and identified accountable human OutcomeDecision.
- **AC-01-31:** ambiguous synthetic submission enters `outcome_unknown`; no blind automatic resubmission occurs.
- **AC-01-32:** evidence manifests reconstruct happy, blocked, rejected, failed, cancelled, unknown, inconclusive, superseded, compensation-required, compensated and successor lineages with immutable digests/redaction metadata.

### Context, trust and secrets

- **AC-01-33:** identical Context Bundle input versions + builder version reproduce the same selected-reference/composition digest.
- **AC-01-34:** overlay permutation, policy relaxation, deny removal and broader-scope substitution fail; later overlays only tighten.
- **AC-01-35:** personal/retrieved/prior-Case/generated content cannot shadow authoritative namespaces or change authority class; collisions are evidenced.
- **AC-01-36:** topology expansion enforces depth/relation/target-count limits; missing/stale/omitted/access-restricted context remains distinguishable.
- **AC-01-37:** secret values are absent from reasoning context, packets, client payloads, messages, logs, receipts, evidence, object metadata and search fixtures.

### Compatibility and release quality

- **AC-01-38:** historical mapping/backfill creates zero passed Verifications and zero Accepted outcomes from v1 claims; one-writer ownership is preserved.
- **AC-01-39:** fresh install and approved P3/P2 adoption/upgrade paths remain reproducible under the accepted runner and required blocking CI is green on the final candidate.
- **AC-01-40:** A1-A9 reconstruct, an independent reviewer reproduces the synthetic decision/evidence chain, and unresolved Sev1/Sev2 kernel/migration/data-integrity/security/isolation/idempotency/evidence/CI defects equal zero.

---

## 8. Mandatory scenario applicability

### 8.1 Kernel KT matrix

| Scenario | WP-01 disposition |
|---|---|
| KT-01 happy path | **Required synthetic:** propose -> approve -> grant -> fake submit -> synthetic provider completion -> fresh fake Observation -> independent fake Verification -> human accept |
| KT-02 packet modification/successor | Required |
| KT-03 stale baseline/resource version | Required |
| KT-04 policy/approval/grant/subject/route/nonce failures | Required |
| KT-05 concurrent duplicates | Required via fake route |
| KT-06 restart during polling | Required via durable fake provider-operation reference |
| KT-07 ambiguous submission | Required; `outcome_unknown`, reconciliation, no auto-retry |
| KT-08 provider-completed/result mismatch | Required synthetic |
| KT-09 provider running/health Verification fails | Required synthetic |
| KT-10 verifier unavailable | Required synthetic; inconclusive unless exact spec defines failure |
| KT-11 cancellation boundary | Required synthetic |
| KT-12 correction/compensation successor | Required, including compensation-required and compensated lineage evidence |
| KT-13 Site Runtime disconnection | **Deferred to WP-04.** WP-01 only proves an unavailable/unimplemented site route cannot authorize/dispatch; it makes no local polling/spool claim. |
| KT-14 secret scan | Required; no real provider secret is introduced |
| KT-15 evidence reconstruction | Required synthetic across every required terminal lineage |
| KT-16 historical truth | Required |

### 8.2 Native Pattern criteria

WP-01 owns repository-linked conformance for:

- P-02: PC-ID-01..05;
- P-03: PC-TR-01..05;
- P-05: PC-BC-01..09;
- P-06: PC-IP-01..05 using fake Prepare where adapter behavior is required;
- P-08: PC-AD-01..05 at kernel/fake-route level;
- P-09: PC-CS-01..05 as trust foundation. Credential-injection conformance PC-CS-06..09 completes in WP-02/WP-04, though entitlement schema validation begins here;
- P-10: PC-IV-01..05 using deterministic verifier fixtures;
- P-11: PC-SC-01..05;
- P-18: WP-01 foundation subset of PC-WS-01/02 for implemented surfaces. Deployment-contract operations PC-WS-03..10 remain WP-10 except preservation of existing WP-00 install/upgrade proof.

P-01 Case skeleton, P-04 answer/source-reference skeleton and P-07 capability-registry skeleton may be introduced only as required by WP-01 contracts. Full user/provider conformance remains downstream.

---

## 9. Migration/database contract

WP-01 is Compatibility Specification migration **Phase 2 — expand**:

- use the accepted Go migration runner/source-profile classification;
- create forward checksummed successor revisions only;
- use advisory lock, checksum and ledger/provenance semantics already accepted;
- use explicit transactions where PostgreSQL permits;
- never edit a successful shared migration artifact;
- never replay legacy `001-040`;
- never drop structures required for rollback/audit/evidence/history;
- enforce allowed application schema-version range before writes;
- preserve fresh/P2/P3 WP-00 convergence.

### One-writer rule

Each object family has exactly one authoritative writer. New v2-only kernel objects may be v2-owned immediately. Existing v1 families remain v1-owned until a later recorded cutover package transfers ownership. Shadow reads, mappings and compatibility projections are derived, not second writers.

### Historical mapping safety

Where WP-01 creates mappings/classifications:

- source table/key/profile/run provenance is mandatory;
- writes are idempotent/restartable;
- stable IDs are preserved only where meaning is unchanged;
- new semantic aggregates get new IDs;
- legacy approvals do not create AuthorityGrants;
- legacy success/task/operator text does not create passed Verification or Accepted;
- ambiguous provider history remains the exact approved weaker classification;
- sensitive values never enter evidence exports.

---

## 10. CI and evidence policy

### 10.1 Required checks

The WP-00 required contexts remain mandatory:

1. `Frontend (typecheck · test · build)`;
2. `Worker (Python)`;
3. `Backend (Go)`;
4. `G5 Foundation Gate`.

WP-01 MAY add a fail-closed `WP-01 Kernel Gate` once stable. It may not replace/weaken the four accepted checks. If adopted, RG-01 binds its exact context/workflow and green final run.

### 10.2 Evidence artifacts

| ID | Artifact | Minimum contents |
|---|---|---|
| A1 | Authority and Contract Manifest | authority identities, baseline, plan, ownership map, context-contract identity, applicability matrix |
| A2 | Canonical Schema and Principal Manifest | revisions/checksums, tables/constraints/indexes, principals/workspaces, fresh+upgrade results |
| A3 | Transition/Concurrency/Provenance Evidence | legal/illegal matrix, conflicts, successor immutability, projection rebuild |
| A4 | Persistence/Messaging/Recovery Evidence | outbox atomicity, inbox dedupe, replay/restart/lease-loss |
| A5 | Packet/Authority/Broker/Idempotency Evidence | packet digest, decision separation, scope negatives, one-attempt proof, `LIVE_PROVIDER_MUTATIONS=0` |
| A6 | Verification/Outcome/Successor/Evidence Pack | pass/fail/inconclusive, human acceptance, unknown/reconcile, cancelled, compensation-required, compensated, successor lineages and manifests |
| A7 | Context/Anti-Shadowing/Trust/Isolation Evidence | overlay digest, relaxation/collision negatives, limits, secret scan, isolation matrix |
| A8 | Compatibility/Historical-Truth Evidence | mappings, zero promoted legacy truth, one-writer proof, P2/P3/fresh regression |
| A9 | RG-01 Release Evidence Manifest | exact commits/runs/digests, AC crosswalk, A1-A8 digests, defect disposition, approvals |

### 10.3 Evidence hygiene

Do not commit production dumps/raw sensitive P1/P2 rows, provider credentials, real secret values, reusable authenticated requests, password/MFA/token material or secret-manager values. Every release-evidence artifact is secret-scanned.

---

## 11. Failure/recovery requirements

WP-01 must fail closed for:

- stale/changed baseline -> block before synthetic dispatch;
- packet digest/signature/schema mismatch -> integrity/security block;
- policy denial/rejected approval -> no grant;
- expired/revoked/mismatched grant -> no dispatch;
- duplicate command -> existing attempt/result;
- pre-submit fake-route failure -> retry only when safe classification proves no submission;
- ambiguous fake submission -> `outcome_unknown`, no blind retry;
- synthetic provider completion + result mismatch -> Verification cannot pass;
- verifier unavailable/stale/missing -> inconclusive unless exact spec says failure;
- aggregate-version conflict -> reject stale writer;
- restart/lease loss -> resume from durable records;
- unsupported envelope/poison event -> durable quarantine/disposition;
- context critical-source gap -> visible degraded investigation only, not executable-precondition satisfaction;
- cross-workspace/authority namespace collision -> reject/quarantine with evidence;
- compensation failure -> separate successor failure, original unchanged.

No recovery may manufacture success, erase an attempt, broaden authority, or mutate sealed history.

---

## 12. Defect and change control

RG-01 requires zero unresolved Sev1/Sev2 defects in kernel truth/state, migration/data integrity, workspace isolation/security, authority/replay/idempotency, outbox/inbox/recovery, verification/evidence, historical truth and blocking CI.

A blocker requires a concrete observed problem plus evidence that directly prevents/invalidates a frozen acceptance property and cannot be handled by the accepted design.

Routine code/test/migration errors, schema/package naming, fixture design, internal API shape and bounded implementation uncertainty are not new authorization gates.

Do not silently change:

- K-01..K-12;
- claim/Verification/Accepted separation;
- packet immutability;
- grant binding requirements;
- Effect Broker sole-dispatch rule;
- workspace isolation;
- historical-truth classification;
- zero-live-provider WP-01 boundary;
- AC-01 criteria.

A demonstrated higher-authority contradiction requires a governed successor before semantic change.

---

## 13. Execution order

```text
WP01-G0 plan/contract freeze
  -> WS01-02 schema/principals
  -> WS01-03 states/concurrency
  -> WS01-04 persistence/messaging
  -> WP01-G1/G2
  -> WS01-05 packet/authority/broker
  -> WP01-G3
  -> WS01-06 verification/evidence/successors
  -> WS01-07 context/trust/anti-shadowing
  -> WS01-08 compatibility/historical truth
  -> WS01-09 isolation/security
  -> WP01-G4
  -> WS01-10 full conformance/evidence
  -> RG-01
```

Parallel work is permitted only where it cannot create conflicting authority/schema contracts. WP-02 live adapter work may not merge into the release path before RG-01.

---

## 14. RG-01 acceptance

RG-01 closes only when:

1. AC-01-01..40 are individually PASS;
2. K-01..K-12 are evidenced without contradiction;
3. applicable KT and Native Pattern criteria pass;
4. A1-A9 reconstruct from exact repository/CI identities;
5. fresh install + approved P3/P2 adoption/upgrade remain reproducible;
6. final candidate is green under all four accepted required contexts and any adopted WP-01 gate;
7. secret scan and workspace-isolation matrix pass;
8. historical mapping creates zero passed Verifications and zero Accepted from legacy claims;
9. synthetic happy, blocked, rejected, failed, cancelled, unknown, inconclusive, superseded, compensation-required, compensated and successor manifests reconstruct;
10. `LIVE_PROVIDER_MUTATIONS=0` is evidenced;
11. unresolved Sev1/Sev2 = 0;
12. Architecture, Backend, Database, Security and Quality record approval; delegated role-based assessment is identified transparently if used.

The final receipt binds implementation commits, migration checksums, schema range, Context Overlay Contract digest, CI runs, evidence digests, AC crosswalk and defect disposition.

Only these terminal decisions are valid:

```text
RG-01=ACCEPTED
WP-01=ACCEPTED
```

or

```text
RG-01=BLOCKED
WP-01=NOT_ACCEPTED
```

Conditional acceptance is not permitted.

---

## 15. Post-RG-01 boundary

RG-01 acceptance freezes WP-01 truth, authority, persistence, verifier/evidence, context-overlay and isolation contracts as downstream inputs. It does **not** authorize WP-02 by itself.

WP-02, if separately authorized, may implement the first real `compute.virtual_machine.start@1` adapter path, connector-side destination-bound credential broker and Proxmox VE conformance without weakening accepted WP-01 semantics. WP-03 owns the complete Case/My Work product experience and R1 acceptance. WP-04+ remain separately gated.
