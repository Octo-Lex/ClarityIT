# ClarityIT v2 — WP-01 Authoritative Kernel Foundation Plan

**Package:** WP-01 — Authoritative Kernel Foundation  
**Version:** 0.1  
**Status:** Authorized package plan; implementation authority activates when this plan is integrated to `main`  
**Authorization ID:** `WP01-AUTH-2026-08-12`  
**Authorization date:** 12 August 2026  
**Package baseline:** `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`  
**Entry condition:** WP-00 G6 accepted; AC-00-01 through AC-00-30 PASS; A1-A7 complete; unresolved Sev1/Sev2 = 0  
**Final gate:** RG-01 — Authoritative Kernel Foundation acceptance  

> **Package decision:** WP-01 establishes the canonical v2 domain, truth, authority, persistence, evidence, context, trust, isolation, and compatibility foundation alongside the stabilized v1 spine. It MUST NOT perform a live consequential provider mutation. All execution behavior in WP-01 is exercised through deterministic fake/no-op fixtures. The first real provider-neutral mutation remains WP-02.

---

## 1. Authority, purpose, and precedence

WP-01 is the first v2 feature-foundation package after the accepted WP-00 migration baseline. It converts the already-defined Product, Kernel, Compatibility, Architecture, Native Pattern, Trust, and Roadmap contracts into one bounded implementation package with explicit entry criteria, workstreams, evidence artifacts, acceptance criteria, failure scenarios, and exit authority.

This plan governs **package execution and RG-01 acceptance only**. It does not change higher-order product or semantic authorities. If an implementation convenience conflicts with a higher authority, the higher authority governs and WP-01 must stop or use an explicitly governed successor decision.

### 1.1 Bound authority set

| Priority | Authority | Bound repository identity | WP-01 use |
|---:|---|---|---|
| 1 | Product Definition v0.1 | `docs/v2/ClarityIT_v2_Product_Definition_v0.1.md`, blob `d44975d1557e8499c4e7613a5cd49115126266b0` | Product category, first-release scope, outcome language, acceptance boundary |
| 2 | Authoritative Execution Kernel v0.1 | `docs/v2/ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md`, blob `1153fb3bfadb1e603307354dc8b6e361eb44167d` | Highest execution-semantics authority; K-01 through K-12, canonical objects, state machines, persistence, verification, evidence |
| 3 | v1-to-v2 Compatibility and Migration v0.1 | `docs/v2/ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md`, blob `bdf179c677f283591842f5a52e41092a70e0b660` | Additive coexistence, historical truth, one-writer rule, compatibility/backfill constraints |
| 4 | Layered System Architecture | `docs/v2/ClarityIT-v2-Layered-System-Architecture.md`, blob `9d42a74b39e941509725c1c5dd42a87c9126b9e8` | Logical placement, read/write boundaries, route separation |
| 5 | Native Pattern Specification v0.1 | `docs/v2/ClarityIT_v2_Native_Pattern_Specification_v0.1.md`, blob `00ce72fab791e8b959549b4845d40b4a48954044` | P-02, P-03, P-05, P-06, P-08, P-09, P-10, P-11, P-18 plus P-01/P-04/P-07 skeletons |
| 6 | Environment Trust and Evidence Custody Profile v0.1 | `docs/v2/ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md`, blob `8a6d28d538fd0d5525114958329b0592829806a9` | Development trust placement, evidence handling, no-in-place production promotion |
| 7 | Delivery Roadmap v0.2 | `docs/v2/ClarityIT_v2_Delivery_Roadmap_v0.2.md`, blob `89911eb29972d813d75f22d98cf239d2b61784b6` | WP-01 scope, dependencies, RG-01 exit gate |
| 8 | WP-00 final evidence | `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f` | Accepted migration runner, governed DB target, blocking CI, historical-truth foundation |
| 9 | This plan | This file after integration | WP-01 implementation order, package gates, evidence and closure |

### 1.2 Frozen WP-00 inputs inherited by WP-01

WP-01 MUST preserve the accepted WP-00 foundation. The following are read-only package inputs unless a demonstrated defect requires a separately governed successor:

- WP-00 final integration: `e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`;
- governed target fingerprint: `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6`;
- baseline checksum: `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508`;
- composite installation SHA-256: `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084`;
- frozen P3 source fingerprint: `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`;
- frozen P2 v3.2 successor fingerprint: `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb`;
- historical P1/P2 v3.1 fingerprint: `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`, recognized/non-executable;
- P3 adoption artifact SHA-256: `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359`;
- PostgreSQL proof/runtime major: 16;
- existing required `main` checks: `Frontend (typecheck · test · build)`, `Worker (Python)`, `Backend (Go)`, `G5 Foundation Gate`.

No WP-01 change may silently redefine a WP-00 migration identity or weaken the blocking CI predicate.

---

## 2. Objective and package boundary

### 2.1 Objective

Introduce the canonical v2 domain and truth model alongside the stabilized v1 spine, with enough executable kernel behavior to prove state, authority, persistence, isolation, context composition, verification semantics, evidence reconstruction, and compatibility **without a live consequential provider mutation**.

At RG-01, ClarityIT must possess a trustworthy kernel foundation on which WP-02 can add the first real `compute.virtual_machine.start@1` provider path without inventing new authority, truth, persistence, or evidence semantics.

### 2.2 In scope

WP-01 SHALL implement or establish:

1. canonical v2 schemas for Cases, Resources, Provider Bindings, Observations, Operation Packets, Policy Decisions, Approval Decisions, Authority Grants, Execution Attempts, Provider Receipts/Result Claims as typed contract records, Verification Specs, Verifications, Verification Evidence, Outcome Decisions, Evidence Manifests, inbox/outbox, compatibility mappings, and required migration/backfill metadata;
2. the canonical Principal model for human, reasoning agent, service, policy, execution workload, and external source identities;
3. legal/illegal state machines, optimistic aggregate-version concurrency, successor lineage, unknown/inconclusive classifications, and immutable historical records;
4. deterministic Operation Packet canonicalization, digest/signature envelope contract, immutable proposal transition, and successor rules;
5. deterministic policy evaluation contracts, approval separation, grant lifecycle, separation of duties, replay protection, reservation/consumption semantics, and fail-closed scope validation;
6. an Effect Broker skeleton as the sole dispatch API with route/preflight/reservation/idempotent-attempt contracts and a deterministic fake/no-op execution route for tests only;
7. versioned verifier contracts and a deterministic fake/read-only verifier harness proving passed/failed/inconclusive semantics from fresh sealed inputs;
8. evidence-manifest generation and reconstruction for synthetic happy, blocked, failed, unknown, rejected, superseded, correction, and compensation-required lineages;
9. transactional outbox/inbox, aggregate concurrency, replay, duplicate delivery, lease-loss/restart semantics, and persist-before-publish behavior;
10. the Resource-aware bounded context and deterministic overlay composition contract, including monotonic tightening, anti-shadowing, provenance, screening states, omissions, topology limits, and deterministic composition digest;
11. trust-foundation schemas and policy evaluation for workload identity, secret references, route binding, signature/canonicalization, and destination-bound credential-broker entitlements without implementing a live credential-injecting connector;
12. additive v1/v2 coexistence with feature flags off, v1 read compatibility preserved, historical truth classification, safe backfill/mapping foundations, and explicit rejection of unsafe legacy execution where activated by WP-01 tests;
13. workspace isolation across database/API/event/storage-reference/cache/search fixtures and all WP-01 background processing;
14. blocking CI, reproducible fresh install and approved upgrade/adoption coverage for the new additive WP-01 revisions.

### 2.3 Explicit exclusions

WP-01 MUST NOT implement, enable, or claim acceptance for:

- a live provider mutation of any kind;
- a real Proxmox start request, provider credential, UPID workflow, provider polling, or provider reconciliation;
- `compute.virtual_machine.start@1` live adapter conformance; that is WP-02;
- provider-specific core packet fields or a provider-prefixed v2 authority model;
- Site Runtime, private-zone execution, local journal/spool, edge-gateway execution, or disconnected operation; that is WP-04;
- the complete My Work / Case Workspace / Resource product experience, live WebSocket product progress, accessibility acceptance, or controlled pilot; that is WP-03;
- reviewed knowledge publication, operational skills/playbooks, Signals/Routines, Project delivery integrations, multi-target execution, second provider, extension SDK, or production rollout;
- production cutover or production trust-domain promotion;
- arbitrary shell, SQL, SSH, Kubernetes, database, browser, desktop, Git, cloud, or generic authenticated HTTP mutation interfaces;
- destructive removal of legacy tables or historical records;
- migration of historical provider/agent/operator success claims into passed Verification or Accepted outcomes;
- a general credential broker that reasoning workers, browsers, playbooks, or routines can invoke.

### 2.4 Zero-provider-mutation rule

During WP-01:

- every execution-route fixture MUST be deterministic and test-only;
- provider credentials MUST NOT be present in CI, development fixtures, packets, logs, events, receipts, evidence, or test configuration;
- network egress to a managed-system mutation endpoint is not an acceptance dependency;
- the fake/no-op route may emit typed accepted/running/completed/failed/unknown receipts solely to exercise kernel state and evidence semantics;
- RG-01 evidence MUST explicitly state `LIVE_PROVIDER_MUTATIONS=0`.

A real consequential call is a package-boundary violation, not a shortcut to proving RG-01.

---

## 3. Canonical design decisions frozen by this plan

WP-01 implementation MUST preserve the following decisions unless a higher authority is formally revised:

1. **PostgreSQL is authoritative.** NATS transports committed commands/events and never creates product truth.
2. **Persist before publish.** Authoritative state, audit transition, and outbox row commit in one PostgreSQL transaction before publication.
3. **Deduplicate before applying.** Consumers persist inbox identity before/with side effects; duplicate delivery returns the recorded result.
4. **Typed truth levels remain separate.** Finding, Observation, proposal, policy decision, approval, grant, attempt, receipt, Result Claim, Verification, Outcome Decision, and evidence manifest cannot collapse into a generic success record.
5. **Operation Packets are immutable once proposed.** Material change creates a successor and invalidates bound authority as required.
6. **Approval is not authority.** ApprovalDecision and AuthorityGrant are separate records.
7. **Authority is exact and expiring.** Grant scope binds workspace, Case, packet digest, resource/version, capability/constraints, workload, route, policy revision, validity, nonce, and use count.
8. **Unknown is real.** Ambiguous submission becomes `outcome_unknown`; it is never coerced to success/failure and is never blindly resubmitted.
9. **Provider completion is a claim.** It cannot create Verification or Accepted.
10. **Verification is independent and versioned.** Exact VerificationSpec + fresh sealed inputs produce passed/failed/inconclusive.
11. **First-release acceptance is explicit.** An identified accountable human is required; Verification alone does not imply Accepted.
12. **Corrections are successors.** No prior packet, attempt, claim, Verification, or Outcome Decision is overwritten.
13. **Reasoning is credentialless and non-authoritative.** Agents may investigate/propose; they cannot approve, grant, dispatch directly, verify their own work, accept outcomes, or receive secret values.
14. **Effect Broker is the sole dispatch API.** Browser, Control API, compatibility path, agent, routine, playbook, Project, or extension cannot create a parallel consequential path.
15. **Workspace scope is mandatory server-side.** Missing or ambiguous workspace scope fails closed.
16. **Context is derived.** Context bundles, projections, caches, search, and summaries are rebuildable and cannot create authority.
17. **Overlay policy is monotonic.** Narrower/later overlays may tighten but never relax organization/workspace authority.
18. **Authoritative namespaces cannot be shadowed.** Personal/retrieved/prior-Case/generated content cannot replace policy, capabilities, Resource IDs/bindings, published knowledge versions, or current Observations.
19. **Historical truth is preserved.** Legacy success text/flags do not become Verification/Accepted.
20. **One writer per object family.** Additive coexistence is allowed; bidirectional authoritative dual-write is not.

---

## 4. Deliverable architecture

WP-01 SHALL deliver the following logical modules. Exact package/file placement may follow existing repository conventions, but the ownership boundaries are normative.

| Module | Owns / provides | Must not do |
|---|---|---|
| Domain kernel | Cases, Resources, packets, outcome decisions, transition validation, successor lineage | Call providers; issue execution claims |
| Principal/identity layer | Typed PrincipalRef, workload/source identity bindings, separation-of-duties inputs | Treat a UI session as execution workload |
| Authority service | PolicyDecision, approval requirements, ApprovalDecision validation, grant issuance/revocation | Use provider credentials; mark outcomes accepted |
| Effect Broker | Sole dispatch command, preflight contract, grant reservation, attempt/idempotency, dispatch record | Select/use a real provider in WP-01; mark Verified |
| Execution fixture | Deterministic fake/no-op route for kernel contract tests | External mutation; real secret resolution |
| Observation contract | Typed source-attributed baseline/result observations, freshness metadata | Treat stored observation as permanently fresh |
| Verification service | VerificationSpec, fresh-input evaluation, passed/failed/inconclusive, evidence sealing | Reuse execution success as proof |
| Evidence service | Append-only evidence manifest, digests, artifact refs, redaction metadata | Rewrite prior lineage |
| Messaging foundation | v2 envelopes, transactional outbox, inbox dedupe, replay/rebuild contracts | Create truth from NATS delivery |
| Context builder | Bounded Resource-aware bundles and deterministic overlays | Grant authority; leak cross-workspace/secret content |
| Compatibility layer | Additive mapping, read compatibility, historical classification, unsafe-write rejection contracts | Fall through to legacy unsafe execution |

---

## 5. Workstreams

### WS01-01 — Authority freeze, Context Overlay Contract, and implementation map

**Purpose:** turn the normative documents into executable WP-01 contracts before broad code changes.

Deliverables:

- repository-recorded `WP01-AUTH-2026-08-12` authority and exact baseline;
- this package plan integrated and listed in the v2 authority index;
- `ClarityIT_v2_Context_Overlay_Contract_v0.1.md` created and approved before RG-01, defining overlay order, authority classes, monotonic tightening, anti-shadowing, screening states, topology/size limits, deterministic composition digest, cache invalidation, omissions and evidence record;
- object/owner matrix for every WP-01 authoritative table and service;
- state-transition matrix and reason-code mapping;
- schema/API/event versioning policy for WP-01;
- applicability matrix for Kernel KT-01..KT-16 and Native Pattern conformance criteria;
- migration/additive revision design bound to the accepted WP-00 runner.

Exit evidence: **A1 — WP-01 Authority and Contract Manifest**.

### WS01-02 — Additive canonical schema, identity, Resource/Case skeleton

**Purpose:** establish the authoritative data model without changing live consequential behavior.

Deliverables:

- additive kernel/compat schema revisions through the accepted checksummed migration runner;
- Cases and Resource/ProviderBinding skeletons with workspace scope, identity, aggregate version and lineage fields;
- typed PrincipalRef and principal categories: human, reasoning, service, policy, execution workload, external source;
- Observations with source, observed/received times, fieldset, freshness, external revision and fingerprint;
- canonical tables for packet, policy/approval/grant, attempts, receipts/claims, VerificationSpec/Verification/evidence, outcomes and EvidenceManifest;
- inbox/outbox and compatibility idempotency/mapping foundations;
- constraints/indexes enforcing workspace and identity integrity;
- no destructive legacy DDL; feature flags remain off.

Exit evidence: **A2 — Canonical Schema and Principal Manifest**.

### WS01-03 — State machines, optimistic concurrency, typed provenance

**Purpose:** make illegal transitions impossible and history reconstructable.

Deliverables:

- packet states `draft -> proposed -> superseded|withdrawn|expired`;
- grant states `issued -> reserved -> consumed|released|revoked|expired` as allowed by the Kernel contract;
- attempt states including `created`, `preflight`, `dispatchable`, `submitting`, `submitted`, `running`, `provider_completed`, `provider_failed`, `blocked`, `cancelled`, `outcome_unknown`;
- Verification `pending -> running -> passed|failed|inconclusive`;
- OutcomeDecision `pending -> accepted|rejected|correction_required|compensation_required`;
- Case lifecycle projection over underlying typed records only;
- expected aggregate-version checks and conflict behavior;
- immutable successor/supersession lineage;
- typed record provenance and causal ordering independent of browser/message arrival order;
- projection rebuild from PostgreSQL without relying on NATS history.

Exit evidence: **A3 — Transition, Concurrency, and Provenance Evidence**.

### WS01-04 — Transactional messaging, inbox/outbox, replay and recovery

**Purpose:** prove PostgreSQL-owned truth under duplicate transport, restart and worker lease loss.

Deliverables:

- versioned v2 message envelopes with message ID, workspace, aggregate type/ID/version, correlation, causation, actor, times and payload digest;
- authoritative transition + audit + outbox atomic transaction;
- consumer inbox dedupe before/with state transition;
- duplicate delivery returns prior result without second aggregate transition;
- durable worker checkpoint fields needed by synthetic attempt/verification processing;
- replay/projector rebuild fixtures;
- lease-loss and restart tests proving lease state is coordination only;
- poison/unsupported schema version quarantine with reason codes, using the already-accepted durable disposition foundation.

Exit evidence: **A4 — Persistence, Messaging, and Recovery Evidence**.

### WS01-05 — Operation Packet, policy, approvals, grants and Effect Broker skeleton

**Purpose:** establish the sole authority and dispatch path without a live provider.

Deliverables:

- deterministic packet canonicalization and SHA-256 digest;
- signature envelope interface, key identifier and algorithm agility consistent with Kernel v0.1;
- proposal freeze and successor rules;
- deterministic PolicyDecision service contract;
- required-approval evaluation and ApprovalDecision binding to exact packet digest;
- separation-of-duties checks;
- AuthorityGrant issuance/revocation/expiry/reservation/use-limit lifecycle;
- exact scope validation for workspace, Case, packet, resource/version, capability/parameters, workload, route, policy revision, nonce and validity;
- deterministic idempotency key and uniqueness constraints;
- Effect Broker API as sole dispatch entry point;
- fake/no-op route and capability fixture for contract tests only;
- attempt + dispatch record + grant reservation atomicity;
- no external credential, route or provider call.

Exit evidence: **A5 — Packet, Authority, Broker and Idempotency Evidence**.

### WS01-06 — Verification, outcomes, evidence and successor recovery

**Purpose:** prove that claims cannot become truth without independent typed evaluation and accountable disposition.

Deliverables:

- immutable versioned VerificationSpec contract;
- fake read-only verifier harness with exact-spec, freshness, pass/fail/inconclusive and retry-evidence semantics;
- prohibition on executor flags, agent conclusions and UI state as verification inputs;
- explicit human OutcomeDecision requirement for first-release `Accepted`;
- synthetic provider-completed-but-mismatched and provider-completed-but-health-failed paths;
- correction/compensation successor relationships: `corrects`, `retries_after_safe_failure`, `reconciles_unknown`, `compensates`;
- blind retry rejection after ambiguous submission;
- EvidenceManifest generation for happy, blocked, rejected, failed, unknown, inconclusive, superseded and successor lineages;
- immutable artifact references/digests and redaction metadata;
- independent reviewer reconstruction fixture.

Exit evidence: **A6 — Verification, Outcome, Successor and Evidence Manifest Pack**.

### WS01-07 — Context overlays, trust foundation and anti-shadowing

**Purpose:** make reasoning context bounded, deterministic, attributable and unable to weaken authority.

Deliverables:

- Context Bundle identity/digest, exact Case/Resource scope, resource/binding versions, Observation refs/freshness, topology, ownership, health contracts, prior Case refs, allowed read capabilities, exclusions and builder version;
- ordered overlays: organization -> workspace -> Case/Resource -> role/task -> personal drafts;
- authority class, source identity, version/time, sensitivity, screening state and immutable digest per overlay entry;
- monotonic policy tightening;
- reserved authoritative namespaces and collision rejection/quarantine;
- deterministic composition digest with applied/rejected/omitted/conflict evidence;
- explicit topology depth, relation allowlist and target-count limits;
- workspace/access-scope/versioned cache keys and invalidation contract;
- `screened`, `quarantined`, `unscreened` handling;
- secret references allowed where policy permits; secret values prohibited;
- trust schemas for workload identities, route binding, secret references and credential-broker entitlements;
- broker entitlement **schema/policy evaluator only** in WP-01; no live credential injection or generic authenticated transport.

Exit evidence: **A7 — Context, Anti-Shadowing, Trust and Isolation Evidence**.

### WS01-08 — Compatibility, historical truth, backfill and one-writer coexistence

**Purpose:** introduce v2 semantics without inventing historical authority or breaking supported v1 reads.

Deliverables:

- additive coexistence with v1 remaining authoritative writer for object families not yet cut over;
- explicit one-writer ownership registry by object family;
- feature flags off by default for consequential v2 behavior;
- stable identity mappings where semantics are unchanged and new IDs for new semantic aggregates;
- historical classification preserving legacy provider/agent/operator claims;
- deterministic backfill/mapping fixtures that create **zero passed Verifications and zero Accepted outcomes** from historical rows;
- safe compatibility reads and truth classification;
- explicit fail-closed behavior for unsafe legacy execution paths when exercised by WP-01 compatibility tests;
- no destructive contract DDL;
- restartable/idempotent bounded backfill/checkpoint framework where WP-01 introduces backfillable objects;
- fresh-install and approved P2/P3 upgrade/adoption compatibility through the accepted migration runner.

Exit evidence: **A8 — Compatibility and Historical-Truth Evidence**.

### WS01-09 — Workspace isolation and security conformance

**Purpose:** prove server-side isolation and prohibited-power boundaries before any live provider route exists.

Deliverables:

- workspace constraints/query guards for all WP-01 authoritative records;
- cross-workspace API authorization negative tests;
- event/envelope workspace mismatch rejection;
- object-storage reference scope tests without exposing raw sensitive artifacts;
- cache and search-fixture workspace partition tests;
- background-job workspace attribution and rejection of ambiguous scope;
- agent/principal permission negative tests for approval, grant, broker, verification completion and outcome acceptance;
- secret scanning of source, fixtures, packets, messages, logs, receipts and evidence;
- no direct DB/kernel write path from reasoning/browser/compatibility code;
- no direct provider-call surface reachable from Control API or reasoning components.

Exit evidence is incorporated into A2, A5, A7 and final RG-01 release evidence.

### WS01-10 — RG-01 conformance, release evidence and acceptance

**Purpose:** prove the package as one coherent authoritative foundation.

Deliverables:

- complete AC-01 crosswalk;
- Kernel K-01 through K-12 invariant evidence;
- applicable KT-01 through KT-16 scenario evidence under deterministic fixtures;
- Native Pattern conformance evidence for WP-01-owned criteria;
- Context Overlay Contract final approved version;
- fresh install + P3 adoption/upgrade + P2 adoption/upgrade regression evidence;
- all four required `main` checks green plus any new WP-01 gate introduced by this plan and approved through normal repository governance;
- secret scan and workspace-isolation matrix;
- synthetic evidence-manifest reviewer reconstruction;
- zero unresolved Sev1/Sev2 kernel, migration, data-integrity, security, isolation, replay/idempotency, evidence or CI defects;
- **A9 — RG-01 Release Evidence Manifest**, binding A1-A8 and exact implementation/CI identities;
- RG-01 owner decisions: Architecture, Backend, Database, Security and Quality; Product participates where package behavior affects product semantics.

---

## 6. Package gates

WP-01 uses five internal execution gates followed by the Roadmap gate RG-01. Internal gates order work; only RG-01 authorizes WP-02 implementation.

| Gate | Decision | Minimum evidence | Exit authority |
|---|---|---|---|
| **WP01-G0 — Plan and Contract Freeze** | Is WP-01 executable without scope ambiguity? | Integrated plan, authority manifest, Context Overlay Contract draft structure, object/owner and test-applicability maps | Additive implementation may begin |
| **WP01-G1 — Canonical Schema Foundation** | Are canonical objects/principals/workspace constraints safely additive? | A2, fresh/install upgrade migration evidence, no destructive DDL, features off | Domain/state implementation may rely on schema |
| **WP01-G2 — State and Persistence Kernel** | Are transitions, concurrency, inbox/outbox and replay correct? | A3 + A4, legal/illegal matrices, duplicate/restart/lease-loss proof | Authority/broker and verifier flows may rely on persistence |
| **WP01-G3 — Authority and Dispatch Skeleton** | Are packet, policy, approvals, grants, broker and idempotency fail-closed? | A5, no-provider-mutation proof, direct-bypass negative tests | Synthetic execution lineage may be exercised end-to-end |
| **WP01-G4 — Verification, Context and Compatibility** | Are independent proof, successor lineage, overlays, isolation and historical truth coherent? | A6 + A7 + A8, anti-shadowing/isolation/backfill proof | Final RG-01 conformance may begin |
| **RG-01 — Authoritative Kernel Foundation** | Is the v2 kernel foundation semantically correct and safe for WP-02? | A1-A9, AC-01-01..AC-01-40 PASS, required owner decisions, blocking CI green, Sev1/Sev2=0 | WP-02 package may begin only under its own authorization |

A later internal-gate pass does not waive an earlier failed property. Gate evidence may be superseded only through an explicit successor record that preserves the original finding.

---

## 7. Acceptance criteria — AC-01

RG-01 is unconditional. Every criterion below must be PASS with repository-linked evidence.

### Authority and package integrity

- **AC-01-01:** WP-01 implementation is descended from accepted WP-00 `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f` and preserves all frozen WP-00 migration identities.
- **AC-01-02:** WP-01 introduces no live provider mutation, provider credential, Site Runtime execution or production cutover; release evidence states `LIVE_PROVIDER_MUTATIONS=0`.
- **AC-01-03:** The Context Overlay Contract is versioned, repository-approved and bound into A1/A9 before RG-01 acceptance.
- **AC-01-04:** All authoritative object/service ownership and prohibited-write boundaries are documented and enforced by API/module/database tests.

### Canonical schema, identity and workspace

- **AC-01-05:** Canonical Case, Resource/Binding, Observation, Packet, authority, attempt, claim, Verification, outcome, evidence, inbox/outbox and compatibility foundations exist as additive checksummed revisions.
- **AC-01-06:** Every authoritative aggregate and message carries workspace, schema version, actor/workload identity, correlation, causation and time; aggregates use monotonic aggregate version where applicable.
- **AC-01-07:** PrincipalRef supports human, reasoning, service, policy, execution workload and external source identities with explicit workspace/provenance.
- **AC-01-08:** Cross-workspace database/API/event/cache/search/storage-reference fixtures fail closed; ambiguous or missing workspace scope is rejected.
- **AC-01-09:** Shared or agent identities cannot invoke approval, grant issuance/reservation/consumption, direct dispatch, Verification completion or outcome acceptance outside their explicit principal permissions.

### Typed truth and state machines

- **AC-01-10:** Finding, Observation, proposal, PolicyDecision, ApprovalDecision, AuthorityGrant, Attempt, ProviderReceipt/ResultClaim, Verification and OutcomeDecision remain distinct typed records and cannot be promoted by changing a generic status field.
- **AC-01-11:** All legal packet/grant/attempt/Verification/outcome transitions pass and every specified illegal transition fails without partial authoritative mutation.
- **AC-01-12:** Expected aggregate-version conflicts reject concurrent stale writes and preserve the winning authoritative record.
- **AC-01-13:** Corrections/supersession create successor records; original authoritative bytes/digests remain reconstructable.
- **AC-01-14:** `outcome_unknown` and Verification `inconclusive` remain explicit states and are not presentation aliases for failed/succeeded.

### Persistence, messaging and recovery

- **AC-01-15:** Authoritative state, audit transition and outbox row commit atomically in PostgreSQL.
- **AC-01-16:** Duplicate message delivery produces one authoritative transition and one recorded inbox outcome per consumer.
- **AC-01-17:** Projection/timeline state rebuilds from PostgreSQL without NATS history and preserves typed record class/source and causal ordering.
- **AC-01-18:** Worker restart and lease loss resume from durable kernel state without duplicating logical work or rewriting authoritative history.
- **AC-01-19:** Unsupported/malformed envelope versions and poison events reach bounded durable terminal disposition with reason evidence; transport failure does not create product truth.

### Operation Packet and authority

- **AC-01-20:** Proposal canonicalization is deterministic; proposed packets are immutable and have a reproducible SHA-256 digest/signature envelope.
- **AC-01-21:** A material bound-field change creates a successor packet and prior approval/grant cannot authorize the successor.
- **AC-01-22:** PolicyDecision, ApprovalDecision and AuthorityGrant are separate records; approval alone cannot create an executable attempt.
- **AC-01-23:** Grant mismatch for workspace, Case, packet digest, resource/version, capability/parameters, workload, route, policy revision, validity, nonce or use count fails closed before synthetic submission.
- **AC-01-24:** Separation-of-duties conflicts are rejected deterministically and recorded with reason codes.
- **AC-01-25:** Concurrent duplicate broker commands create one ExecutionAttempt for one logical idempotency key and at most one synthetic submission.
- **AC-01-26:** Browser, reasoning worker, compatibility endpoint and Control API cannot call a provider mutation path directly; the Effect Broker is the sole dispatch API.

### Verification, outcome, evidence and successor behavior

- **AC-01-27:** Provider-completed synthetic receipts cannot directly create Verification or Accepted.
- **AC-01-28:** Verification uses the exact versioned VerificationSpec and fresh sealed inputs; pass, fail and inconclusive are independently reproducible.
- **AC-01-29:** Executor flags, agent conclusions and UI/projection state are rejected as Verification inputs.
- **AC-01-30:** First-release Accepted requires a passed Verification and an identified accountable human OutcomeDecision.
- **AC-01-31:** Ambiguous synthetic submission enters `outcome_unknown`; automatic resubmission is prohibited and reconciliation/successor semantics preserve the original lineage.
- **AC-01-32:** Evidence manifests reconstruct synthetic happy, blocked, rejected, failed, unknown, inconclusive, superseded and correction/compensation-required lineages with immutable digests and redaction metadata.

### Context, trust and secrets

- **AC-01-33:** Rebuilding a Context Bundle from the same input versions and builder version produces the same selected-reference/composition digest.
- **AC-01-34:** Overlay permutation, policy relaxation, deny removal and broader-scope substitution fail; later overlays can tighten but cannot weaken organization/workspace authority.
- **AC-01-35:** Personal drafts, retrieved content, prior Case prose and generated text cannot shadow authoritative namespaces or change authority class; rejected/quarantined collisions are evidenced.
- **AC-01-36:** Context topology expansion respects explicit depth, relationship and target-count limits; missing/stale/omitted/access-restricted context remains distinguishable.
- **AC-01-37:** Secret values are absent from reasoning context, packets, browser/client payloads, messages, logs, receipts, evidence, object metadata and search fixtures; only bounded secret references/entitlements may exist.

### Compatibility, migration and final quality

- **AC-01-38:** Historical backfill/classification creates zero passed Verifications and zero Accepted outcomes from v1 claims, provider task references or operator text; one-writer ownership is preserved.
- **AC-01-39:** Fresh install and approved P3/P2 upgrade/adoption paths remain reproducible under the accepted migration runner, and all required blocking CI contexts are green on the final RG-01 candidate.
- **AC-01-40:** A1-A9 reconstruct, an independent reviewer can reproduce the synthetic decision/evidence chain, and unresolved Sev1/Sev2 kernel/migration/data-integrity/security/isolation/idempotency/evidence/CI defects equal zero.

---

## 8. Mandatory scenario matrix

### 8.1 Kernel scenarios applicable in WP-01

The Kernel Specification's mandatory scenarios are exercised at the semantic/persistence layer through deterministic fixtures. Live-provider and Site-Runtime-specific conformance remains downstream.

| Kernel scenario | WP-01 disposition |
|---|---|
| KT-01 happy path | **Required synthetic:** propose -> approve -> grant -> broker fake submit -> synthetic provider completion -> fresh fake Observation -> independent fake Verification -> explicit human acceptance |
| KT-02 packet modification | **Required** |
| KT-03 stale baseline/resource version | **Required** |
| KT-04 policy/approval/grant/subject/route/nonce failures | **Required** |
| KT-05 concurrent duplicates | **Required** using fake route; one attempt/one synthetic submission |
| KT-06 restart during polling | **Required synthetic** using durable fake provider-operation reference |
| KT-07 ambiguous submission | **Required synthetic**; `outcome_unknown`, reconciliation, no automatic retry |
| KT-08 provider-completed/result mismatch | **Required synthetic** |
| KT-09 provider state good / health verification fails | **Required synthetic** |
| KT-10 verifier unavailable | **Required synthetic**; inconclusive unless exact spec says otherwise |
| KT-11 cancellation boundary | **Required synthetic** |
| KT-12 correction/compensation successor | **Required** |
| KT-13 Site Runtime disconnection | **Deferred to WP-04.** WP-01 only proves that an unavailable/unimplemented site route cannot create authority or dispatch. No local polling/spooling claim is made. |
| KT-14 secret scan | **Required**, with no real provider secret introduced |
| KT-15 reviewer reconstructs evidence | **Required synthetic** |
| KT-16 historical truth | **Required** |

### 8.2 Native Pattern criteria owned by WP-01

WP-01 must provide repository-linked conformance for:

- P-02: PC-ID-01 through PC-ID-05;
- P-03: PC-TR-01 through PC-TR-05;
- P-05: PC-BC-01 through PC-BC-09;
- P-06: PC-IP-01 through PC-IP-05, using a deterministic fake Prepare implementation where adapter behavior is required;
- P-08: PC-AD-01 through PC-AD-05 at kernel/fake-route level;
- P-09: PC-CS-01 through PC-CS-05 as trust-foundation requirements; destination-bound credential-injection conformance PC-CS-06 through PC-CS-09 is completed by WP-02/WP-04, although entitlement schema validation begins here;
- P-10: PC-IV-01 through PC-IV-05 using deterministic verifier fixtures;
- P-11: PC-SC-01 through PC-SC-05;
- P-18: WP-01 foundation subset of PC-WS-01/02 sufficient to prove workspace and route-scope isolation in implemented surfaces. Deployment-contract operations PC-WS-03 through PC-WS-10 remain WP-10 unless directly needed to preserve existing WP-00 install/upgrade proof.

P-01 Case domain skeleton, P-04 answer schema/source-reference skeleton and P-07 capability-registry skeleton may be introduced only to the extent required by WP-01 contracts. Their complete user/provider conformance remains WP-02/WP-03 or later.

---

## 9. Migration and database execution contract

### 9.1 Additive-only rule

WP-01 schema work is migration Phase 2 — **expand**. V1 behavior remains intact and consequential v2 features remain disabled by default.

- use the accepted Go migration runner and governed source-profile classification;
- create only forward, checksummed successor revisions;
- use explicit transactions where PostgreSQL permits;
- use advisory locking and accepted checksum/ledger/provenance semantics;
- do not edit any successful shared migration artifact;
- do not replay legacy `001-040`;
- do not drop rollback/audit/evidence/history structures;
- validate application schema-version ranges before writes;
- preserve P2/P3/fresh convergence of the WP-00 foundation.

### 9.2 One-writer rule

Each authoritative object family must have exactly one writer at every WP-01 stage. New v2 kernel objects may be v2-owned immediately because no equivalent v1 authoritative object exists. Existing v1 families remain v1-owned until a later recorded cutover package authorizes ownership transfer.

Shadow reads, backfill mappings and compatibility projections are derived. They cannot become a second authoritative writer.

### 9.3 Historical backfill safety

Where WP-01 creates mappings or synthetic historical classifications for tests/rehearsal:

- source table/key/profile/run provenance is mandatory;
- writes are idempotent and restartable;
- stable IDs are preserved only when semantic meaning is unchanged;
- new v2 semantic aggregates receive new IDs;
- legacy approvals do not create AuthorityGrants;
- legacy success/provider task/operator assessment does not create passed Verification or Accepted;
- ambiguous provider history remains `legacy_submitted_unverified` / `legacy_outcome_unknown` or the exact approved classification;
- sensitive values never enter evidence exports.

---

## 10. CI and evidence policy

### 10.1 Required checks

The accepted WP-00 required contexts remain mandatory throughout WP-01:

1. `Frontend (typecheck · test · build)`;
2. `Worker (Python)`;
3. `Backend (Go)`;
4. `G5 Foundation Gate`.

WP-01 MAY add a dedicated fail-closed `WP-01 Kernel Gate` once its stable fan-in is implemented. Adding that gate is an implementation decision inside WP-01, but it must not replace or weaken the four accepted WP-00 checks. If introduced, RG-01 evidence must bind its exact workflow/context identity and final green run.

### 10.2 Evidence hygiene

Repository/CI evidence may include schemas, fixture data, digests, sanitized manifests, state-transition transcripts, reason codes and generated test artifacts. It MUST NOT include:

- production dumps or raw P1/P2 sensitive rows;
- provider credentials or real secret values;
- reusable authenticated requests;
- tokens, password/MFA material or secret-manager values;
- raw evidence that violates the accepted custody profile.

Every release-evidence artifact must be secret-scanned before acceptance.

### 10.3 Required evidence artifacts

| ID | Artifact | Minimum contents |
|---|---|---|
| A1 | WP-01 Authority and Contract Manifest | authority set/digests, baseline, plan, ownership map, context-contract identity, applicability matrix |
| A2 | Canonical Schema and Principal Manifest | revisions/checksums, tables/constraints/indexes, principal/workspace model, fresh+upgrade results |
| A3 | Transition, Concurrency and Provenance Evidence | legal/illegal state matrix, optimistic conflict tests, successor immutability, projection rebuild |
| A4 | Persistence, Messaging and Recovery Evidence | atomic outbox, inbox dedupe, duplicate/replay/restart/lease-loss results |
| A5 | Packet, Authority, Broker and Idempotency Evidence | packet digest, policy/approval/grant separation, scope negatives, one-attempt proof, no-provider-mutation proof |
| A6 | Verification, Outcome, Successor and Evidence Pack | pass/fail/inconclusive, outcome requirements, unknown/reconcile, successor lineages, manifests |
| A7 | Context, Anti-Shadowing, Trust and Isolation Evidence | overlay digest, permutation/relaxation/collision negatives, topology limits, secret scan, workspace matrix |
| A8 | Compatibility and Historical-Truth Evidence | mapping/backfill classifications, zero promoted legacy truth, one-writer proof, P2/P3/fresh regression |
| A9 | RG-01 Release Evidence Manifest | exact accepted commits/runs/artifact digests, AC-01 crosswalk, defect disposition, approvals, A1-A8 digests |

---

## 11. Failure and recovery policy

WP-01 implementation must fail closed under bounded uncertainty and preserve the distinction between technical failure and authoritative truth.

Required classes:

- stale/changed baseline -> block before synthetic dispatch;
- packet digest/signature/schema mismatch -> block as integrity/security failure;
- policy denial/rejected approval -> no grant;
- expired/revoked/mismatched grant -> no dispatch;
- duplicate command -> existing attempt/result;
- pre-submit fake-route failure -> retry only under explicit safe classification;
- ambiguous fake submission -> `outcome_unknown`, no blind retry;
- provider-completed synthetic claim with mismatch -> Verification cannot pass;
- verifier unavailable/stale/missing -> inconclusive unless exact spec defines failure;
- aggregate-version conflict -> reject stale writer;
- worker restart/lease loss -> resume from durable records;
- unsupported message schema/poison event -> durable quarantine/dead-letter disposition;
- context critical-source gap -> visible degraded investigation only; cannot satisfy missing executable precondition;
- cross-workspace/context authority collision -> reject/quarantine and record evidence;
- compensation failure -> separate successor failure; original outcome unchanged.

No failure handler may manufacture success, erase an attempt, silently broaden authority, or mutate sealed historical records.

---

## 12. Defect, change, and blocker policy

### 12.1 Severity

RG-01 requires zero unresolved Sev1/Sev2 defects in:

- kernel truth/state transitions;
- migration/data integrity;
- workspace isolation/security;
- authority/replay/idempotency;
- transactional outbox/inbox/recovery;
- verification/evidence integrity;
- historical truth;
- blocking CI.

### 12.2 What is a blocker

A blocker requires a concrete observed problem plus evidence that directly prevents or invalidates a frozen acceptance property and cannot be handled by the accepted design.

Examples:

- required authoritative records cannot be made atomic under the accepted database model;
- additive schema cannot preserve an accepted WP-00 or v1 compatibility invariant;
- an authority binding required by Kernel v0.1 is internally contradictory;
- the deterministic context contract cannot preserve both authority precedence and required product semantics;
- a security/isolation failure cannot be corrected without changing the accepted authority model.

Routine code defects, test failures, migration syntax errors, schema naming choices, package layout, fixture design, internal API shape and bounded implementation uncertainty are **not** new authorization gates. Resolve them within this plan and record successor evidence where appropriate.

### 12.3 Change control

Do not silently change:

- Kernel K-01..K-12 semantics;
- the distinction among claims, Verification and Accepted;
- packet immutability;
- grant scope/binding requirements;
- Effect Broker sole-dispatch rule;
- workspace isolation;
- historical-truth classification;
- zero-live-provider-mutation WP-01 boundary;
- RG-01 acceptance criteria.

A demonstrated contradiction in a higher authority requires an explicit governed successor decision before implementing the semantic change.

---

## 13. Execution order

The default sequence is:

```text
WP01-G0 plan/contract freeze
  -> WS01-02 schema + principals
  -> WS01-03 states/concurrency
  -> WS01-04 persistence/messaging
  -> WP01-G1/G2
  -> WS01-05 packet/authority/broker skeleton
  -> WP01-G3
  -> WS01-06 verification/evidence/successors
  -> WS01-07 context/trust/anti-shadowing
  -> WS01-08 compatibility/historical truth
  -> WS01-09 isolation/security
  -> WP01-G4
  -> WS01-10 full conformance/evidence
  -> RG-01
```

Parallel work is permitted only where it cannot create conflicting authority or schema contracts. Documentation/test-fixture work may overlap implementation. WP-02 live adapter work may not merge into the release path before RG-01 acceptance.

---

## 14. RG-01 acceptance procedure

RG-01 may close only when:

1. AC-01-01 through AC-01-40 are individually evidenced as PASS;
2. Kernel K-01 through K-12 are evidenced without contradiction;
3. applicable KT scenarios and owned Native Pattern criteria pass;
4. A1 through A9 reconstruct from exact repository and CI identities;
5. fresh install plus approved P3 and P2 upgrade/adoption profiles remain reproducible;
6. the final candidate is green under all four accepted required contexts and any adopted WP-01 gate;
7. secret scan and workspace-isolation matrix pass;
8. historical backfill produces zero passed Verifications and zero Accepted outcomes from legacy claims;
9. synthetic happy, blocked, failed, unknown, inconclusive and successor evidence manifests reconstruct;
10. `LIVE_PROVIDER_MUTATIONS=0` is explicitly evidenced;
11. unresolved Sev1/Sev2 defects = 0;
12. Architecture, Backend, Database, Security and Quality record approval decisions; delegated role-based assessment must be identified transparently if used.

The final RG-01 receipt must bind exact implementation commits, migration checksums, schema range, Context Overlay Contract digest, CI runs, evidence-pack digests, AC crosswalk and defect disposition.

### 14.1 RG-01 decision

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

## 15. Post-acceptance boundary

RG-01 acceptance establishes the authoritative kernel foundation. It does **not** itself authorize WP-02.

After RG-01:

- WP-01 objects, state machines, authority semantics, messaging, verifier/evidence contracts, context-overlay contract and isolation behavior become frozen inputs to later packages;
- WP-02 may be separately authorized to implement `compute.virtual_machine.start@1`, the generic adapter conformance harness, destination-bound connector credential broker and first Proxmox VE live profile;
- WP-03 remains responsible for the complete Case/My Work product experience and Release R1 acceptance;
- WP-04+ remain separately gated.

No later package may weaken an accepted WP-01 authority, truth, verification, evidence, isolation or historical-truth invariant merely to simplify an adapter, UI, routine, project, runtime or extension.
