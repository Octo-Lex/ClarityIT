# ClarityIT v2 Project Completion Authority

**Status:** Current authoritative project handoff  
**Snapshot date:** 12 August 2026  
**Accepted foundation baseline:** `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`  
**Current completed boundary:** **WP-00 ACCEPTED — G0 through G6 closed**  
**Current authorized package:** **WP-01 — Authoritative Kernel Foundation**  
**Authorization:** `WP01-AUTH-2026-08-12`  
**Current implementation boundary:** provider-mutation-free authoritative kernel foundation  
**Final WP-01 gate:** RG-01  
**Later packages:** WP-02+ require separate authorization

## 1. Purpose

This file is the durable entry point for continuing ClarityIT v2. It records the current completed boundary, controlling authorities, frozen inputs, active package, blocked scope, and the required continuation procedure.

Chat history is not project authority. A decision that changes scope, semantics, a gate, an identity, or authorization must be recorded in the repository before later work depends on it.

This file governs **status and continuation**. It does not override higher product, kernel, migration, architecture, security, evidence, or signed-receipt semantics.

---

## 2. Authority hierarchy

| Priority | Authority | Governs | Current WP-01 status |
|---:|---|---|---|
| 1 | [Product Definition v0.1](ClarityIT_v2_Product_Definition_v0.1.md) | Product category, users, value, first-release outcome and scope | Ratified for WP-01 by `WP01-AUTH-2026-08-12` |
| 2 | [Authoritative Execution Kernel v0.1](ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md) | Truth, authority, execution, verification, outcomes and evidence | Highest WP-01 execution-semantics authority |
| 3 | [v1-to-v2 Compatibility and Migration v0.1](ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md) | Coexistence, migration, historical truth and one-writer sequencing | Ratified for WP-01 |
| 4 | [Layered System Architecture](ClarityIT-v2-Layered-System-Architecture.md) | Logical placement, trust boundaries, persistence and routes | WP-01 architecture baseline where consistent with higher authorities |
| 5 | [Native Pattern Specification v0.1](ClarityIT_v2_Native_Pattern_Specification_v0.1.md) | Reusable native patterns | Ratified for WP-01-owned patterns/skeletons only |
| 6 | [Environment Trust and Evidence Custody Profile v0.1](ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md) | Development trust/custody and production non-promotion boundary | Existing adopted profile remains in force |
| 7 | [WP-01 Authorization](wp01/WP01-AUTHORIZATION.md) | Exact authority-set ratification and activation boundary | `WP01-AUTH-2026-08-12` |
| 8 | [WP-01 Plan](ClarityIT_v2_WP-01_Authoritative_Kernel_Foundation_Plan_v0.1.md) | Workstreams, gates, AC-01, evidence and RG-01 closure | Formal active package plan after integration |
| 9 | [Delivery Roadmap v0.2](ClarityIT_v2_Delivery_Roadmap_v0.2.md) | WP-01–WP-10 order and RG-01–RG-10 | WP-01 boundary active; WP-02+ remain separately gated |
| 10 | This file | Current status and continuation | Current handoff authority |

Higher semantic authorities prevail over package planning. Signed historical receipts remain authoritative for the decisions and identities they bind.

---

## 3. WP-00 closure — immutable foundation

WP-00 is closed and MUST NOT be reopened by routine WP-01 work.

Final closure:

- final evidence integration: `e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`;
- G6 approval receipt integration: `b67d63720aa3fc2231d2d221d06ccb58d7fc09a0`;
- exact integrated P2 rehearsal implementation: `0d0d842c088284d54abe7fd56df9d6ebf63a7e66`;
- AC-00-01 through AC-00-30: **30/30 PASS**;
- A1-A7: complete;
- unresolved Sev1/Sev2: 0;
- issue #1: closed as completed;
- G0 through G6: closed.

### 3.1 Frozen WP-00 identities inherited by WP-01

| Identity | Value |
|---|---|
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Baseline checksum | `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508` |
| Composite installation SHA-256 | `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084` |
| P3 source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| P2 executable v3.2 fingerprint | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` |
| Historical P1/P2 v3.1 fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` — recognized/non-executable |
| P3 adoption artifact SHA-256 | `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359` |
| PostgreSQL target | Major 16; database `clarityit` |

Any modification to a frozen WP-00 artifact/identity requires a demonstrated defect and separately governed successor decision. WP-01 normally consumes these as read-only foundation inputs.

### 3.2 Required CI inherited from WP-00

The `main` merge predicate retains all four accepted contexts:

1. `Frontend (typecheck · test · build)`;
2. `Worker (Python)`;
3. `Backend (Go)`;
4. `G5 Foundation Gate`.

WP-01 may add a kernel-specific gate but may not replace or weaken these checks.

---

## 4. WP-01 authorization and scope

`WP01-AUTH-2026-08-12` authorizes the package defined by the integrated WP-01 plan through RG-01. Routine implementation decisions inside that plan do not require new authorization.

The active objective is:

> Introduce the canonical v2 domain and truth model alongside the stabilized v1 spine, proving authority, state machines, transactional persistence, independent verification semantics, evidence reconstruction, deterministic bounded context, trust foundations, compatibility/historical truth and workspace isolation **without a live consequential provider mutation**.

### 4.1 Non-negotiable WP-01 boundary

WP-01 is **provider-mutation-free**.

Required final evidence:

```text
LIVE_PROVIDER_MUTATIONS=0
```

Permitted:

- additive kernel/compat schema;
- canonical objects/principals/states;
- transactional outbox/inbox;
- immutable packets;
- policy/approval/grant semantics;
- Effect Broker skeleton;
- deterministic fake/no-op route fixtures;
- deterministic read-only verifier fixtures;
- evidence manifests;
- Context Overlay Contract and implementation;
- secret-reference/entitlement schemas and trust-policy evaluation;
- v1 read compatibility, mapping and historical classification;
- workspace/isolation/security tests.

Not permitted in WP-01:

- a real Proxmox/provider call or provider credential;
- Site Runtime/private-zone execution;
- production cutover;
- complete WP-03 Case/My Work UX and live pilot;
- reviewed skills/knowledge, Signals/Routines, Project delivery integration;
- multi-target execution, second provider or extension SDK;
- any WP-02+ implementation merged as part of WP-01.

The first real provider-neutral `compute.virtual_machine.start@1` mutation remains WP-02 after RG-01 and separate authorization.

---

## 5. WP-01 gate ledger

| Gate | State | Purpose | Continuation |
|---|---|---|---|
| WP01-G0 — Plan/Contract Freeze | **Authorized / active on plan integration** | Bind authority, plan, Context Overlay Contract structure, ownership/test maps | Additive implementation |
| WP01-G1 — Canonical Schema Foundation | Pending | Canonical objects/principals/workspace constraints safely additive | State/persistence work may rely on schema |
| WP01-G2 — State and Persistence Kernel | Pending | Legal/illegal states, concurrency, atomic outbox/inbox, replay/restart | Authority/verifier flows may rely on persistence |
| WP01-G3 — Authority/Dispatch Skeleton | Pending | Packet/policy/approval/grant/broker/idempotency fail closed | Synthetic end-to-end lineages |
| WP01-G4 — Verification/Context/Compatibility | Pending | Verification, successors, overlays, isolation, historical truth coherent | Final RG-01 conformance |
| RG-01 — Authoritative Kernel Foundation | Pending | AC-01-01..40, A1-A9, required approvals, blocking CI, Sev1/Sev2=0 | WP-01 accepted; WP-02 still separately authorized |

No later gate can waive an earlier failed property.

---

## 6. WP-01 required evidence

| Evidence | Purpose |
|---|---|
| A1 — Authority and Contract Manifest | Exact authorities, baseline, ownership, applicability, Context Overlay Contract identity |
| A2 — Canonical Schema and Principal Manifest | Revisions/checksums, objects/constraints, principals/workspaces, fresh/upgrade proof |
| A3 — Transition/Concurrency/Provenance Evidence | Legal/illegal states, conflict handling, successors, projection rebuild |
| A4 — Persistence/Messaging/Recovery Evidence | Outbox atomicity, inbox dedupe, replay/restart/lease loss |
| A5 — Packet/Authority/Broker/Idempotency Evidence | Packet digest, decision separation, grant negatives, one-attempt and no-provider proof |
| A6 — Verification/Outcome/Successor/Evidence Pack | Pass/fail/inconclusive, human acceptance, unknown/reconcile, successors, manifests |
| A7 — Context/Anti-Shadowing/Trust/Isolation Evidence | Overlay digest, monotonic policy, collisions, limits, secret/isolation matrix |
| A8 — Compatibility/Historical-Truth Evidence | Mapping/backfill, zero promoted legacy truth, one-writer and P2/P3/fresh regression |
| A9 — RG-01 Release Evidence Manifest | Exact implementation/CI/evidence identities, AC crosswalk, defects, approvals |

The formal definitions and AC-01-01 through AC-01-40 are in the WP-01 plan.

---

## 7. Controlling semantic invariants

Every WP-01 session must preserve:

- PostgreSQL is authoritative; NATS is transport only.
- State + audit + outbox persist atomically before publication.
- Inbox dedupe precedes or joins consumer state mutation.
- Operation Packets are immutable after proposal.
- PolicyDecision, ApprovalDecision and AuthorityGrant remain separate.
- Grants bind exact packet/resource/capability/workload/route/policy/time/nonce/use scope.
- `outcome_unknown` and Verification `inconclusive` are legitimate states.
- Provider/executor claims cannot create Verification or Accepted.
- Verification uses exact versioned specs and fresh evidence.
- First-release acceptance requires an identified accountable human.
- Correction/compensation create successors and preserve original history.
- Reasoning agents are credentialless and cannot issue authority or write execution/verification/outcome truth.
- Effect Broker is the sole dispatch API.
- Workspace isolation is server-side across all implemented surfaces.
- Context is derived; overlay policy can tighten but never relax authority.
- Personal/retrieved/prior-Case/generated content cannot shadow authoritative namespaces.
- Historical v1 success is not upgraded to current verified truth.
- One authoritative writer exists per object family at every coexistence stage.

---

## 8. Continuation procedure

A later session continuing WP-01 must:

1. read this file;
2. read `wp01/WP01-AUTHORIZATION.md`;
3. read `ClarityIT_v2_WP-01_Authoritative_Kernel_Foundation_Plan_v0.1.md`;
4. verify current `main` descends from the accepted WP-00 baseline and the integrated WP-01 plan;
5. identify the current WP01-G* gate and its unmet acceptance evidence;
6. preserve all frozen WP-00 identities and required CI contexts;
7. execute only the current WP-01 scope;
8. stop only for a concrete demonstrated blocker that the accepted design cannot safely handle.

Routine implementation uncertainty is resolved through bounded reversible tests, not new planning gates.

### Required status reporting

Use:

```text
Done
Verified
Blockers
Non-blocking follow-up
Decision
```

Do not report a planning or context-window limitation as a technical blocker.

---

## 9. Final WP-01 closure condition

WP-01 closes only when:

- AC-01-01 through AC-01-40 are PASS;
- Kernel K-01 through K-12 are evidenced;
- applicable Kernel/Native Pattern conformance scenarios pass;
- A1-A9 reconstruct;
- fresh/P3/P2 install/adoption/upgrade paths remain reproducible;
- all required blocking CI is green;
- historical mapping creates zero passed Verifications and zero Accepted outcomes from legacy claims;
- `LIVE_PROVIDER_MUTATIONS=0`;
- unresolved Sev1/Sev2 = 0;
- required Architecture, Backend, Database, Security and Quality decisions are recorded.

Only these decisions are valid:

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

## 10. Roadmap boundary after RG-01

RG-01 freezes the authoritative kernel foundation as an input to later packages. It does not itself authorize WP-02.

A separately authorized WP-02 may then implement the first live provider-neutral VM-start vertical slice, destination-bound connector credential broker and Proxmox VE conformance. WP-03 completes the Case experience and R1 pilot. WP-04+ remain separately gated.
