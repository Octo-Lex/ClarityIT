# ClarityIT v2 Project Completion Authority

**Status:** Current authoritative project handoff
**Snapshot date:** 4 August 2026
**Integrated baseline:** `wp00/g2-schema-decisions` at `211c0ee1abeab0626472a2502d35de13eb9db080`
**Current completed boundary:** WP-00 G3 closed, signed, and integrated
**Next gate:** G4, only after separate explicit authorization

## 1. Purpose

This file is the durable entry point for continuing and completing ClarityIT v2.
It records current progress, controlling authorities, frozen identities, remaining
gates, sequencing constraints, external-evidence boundaries, and the required
start procedure for a later work session.

Chat history is not a project authority. A decision that changes scope,
semantics, a gate, an identity, or an authorization boundary must be recorded in
the repository with its evidence before later work may depend on it.

This file governs **status and continuation**. It does not override product,
execution, migration, architecture, security, evidence, or signed-receipt
semantics. When a conflict exists, use the hierarchy below.

## 2. Authority hierarchy

| Priority | Authority | Governs | Current status |
|---:|---|---|---|
| 1 | [Product Definition v0.1](ClarityIT_v2_Product_Definition_v0.1.md) | Product category, users, value, scope, first-release outcome, and acceptance | Draft product authority |
| 2 | [Authoritative Execution Kernel v0.1](ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md) | Truth, authority, execution, verification, outcomes, and evidence | Draft normative engineering authority |
| 3 | [v1-to-v2 Compatibility and Migration v0.1](ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md) | Source profiles, coexistence, migration, cutover, rollback, and historical truth | Draft normative migration authority |
| 4 | [Layered System Architecture suite](ClarityIT-v2-Layered-System-Architecture.md) | Logical component placement, physical dispatch, evidence sealing, trust/deployment boundaries, Signals/Routines, persistence, and routes | Target architecture baseline; draft overlays remain explicitly proposed |
| 5 | [Native Pattern Specification v0.1](ClarityIT_v2_Native_Pattern_Specification_v0.1.md) | Reusable experience and orchestration patterns that must conform to priorities 1-4 | Draft normative specification; not yet approved |
| 6 | [Environment Trust and Evidence Custody Profile v0.1](ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md) | Development placement, production trust topology, evidence custody, and no-in-place promotion | Adopted for development; normative production exit criteria |
| 7 | [WP-00 Plan v0.1](ClarityIT_v2_WP-00_Migration_Baseline_and_CI_Stabilization_Plan_v0.1.md) | Formal migration-foundation package, G0-G6, AC-00-01 through AC-00-30 | Active formal package; complete through G3 |
| 8 | [Delivery Roadmap v0.2](ClarityIT_v2_Delivery_Roadmap_v0.2.md) | Proposed WP-01 through WP-10 sequence, RG-01 through RG-10, and R1-R5 packaging | Draft planning authority; later package names are not yet approved plans |
| 9 | This file | Current completion ledger, next permitted gate, and continuity rules | Current handoff authority |

Higher-priority semantic authorities prevail over lower-priority planning or
status text. Signed receipts prevail over intermediate report wording for the
specific artifact and decision they bind. A later status record may explain an
older record, but must not rewrite historical evidence.

### Architecture source and render status

Diagram version 0.2 replaces the prior layered PNG as the implementation
reference. Its committed Mermaid sources and version-stamped renders are:

| View | Authority status |
|---|---|
| [Layered System Architecture](ClarityIT-v2-Layered-System-Architecture.md) | Executive target-state overview; Product v0.1 + Kernel v0.1 baseline |
| [Authoritative Operation Sequence](ClarityIT-v2-Authoritative-Operation-Sequence.md) | Normative companion for the sole outbox dispatch, evidence sealing, verification, and outcome-decision boundaries |
| [Trust and Deployment Topology](ClarityIT-v2-Trust-and-Deployment-Topology.md) | Target-state trust companion; environment placement governed by the adopted trust/custody profile |
| [Signals and Routines](ClarityIT-v2-Signals-and-Routines.md) | Proposed P-15 companion; not an approved implementation contract |

Dashed P-05, P-09, P-12, and P-15 elements remain proposed under the unapproved
Native Pattern Specification. Their inclusion documents the intended extension
boundary without granting implementation authority or changing the G3/G4 gate.

## 3. Current integrated state

G3 is complete, signed, and integrated. The integration contains an honest,
non-destructive repair for the repository's squash-only pull-request setting:

| Item | Exact value |
|---|---|
| G2 signed commit | `f04f94faad0105d1c3274e9c7974d44f936a0d28` |
| G3 producing implementation | `570a0ec7e31087d1dd6db22e14935e21e7481cf6` |
| Exact-SHA Linux proof | GitHub Actions run `30900328914` — pass |
| G3 signed tip | `97f83e4ac0609994b64493c7a8b2b76208545bb1` |
| PR #12 squash commit | `4677e104d4a81a3c21dd30f19054a2d79abe0c72` |
| Ancestry-repair merge | `211c0ee1abeab0626472a2502d35de13eb9db080` |
| Integrated target | `wp00/g2-schema-decisions` at `211c0ee1abeab0626472a2502d35de13eb9db080` |
| Recovery branch | `wp00/g3-reconciled-baseline-recovery` at `97f83e4ac0609994b64493c7a8b2b76208545bb1` |
| Pull request | PR #12 — closed and merged |
| Recorded G3 approvals | Architecture: Archy / AR; Database: Domty / DO; both `APPROVE`, 2026-08-04 |

The squash commit and signed tip have the same tree. The two-parent
`211c0ee1...` merge has the squash commit as first parent and the signed tip as
second parent, changes no content, and makes the signed tip an ancestor of the
integrated target. The squash operation remains part of the historical record.

The controlling G3 receipt is
[`docs/migration/wp00/g3/G3-APPROVALS.md`](../migration/wp00/g3/G3-APPROVALS.md).
Do not edit that signed receipt merely to restate current status.

## 4. Frozen identities

These identities are unchanged by the integration and by documentation-only
continuity updates:

| Identity | Value |
|---|---|
| Product manifest blob SHA-256 | `1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63` |
| Product manifest blob size | `284064` bytes |
| Control manifest SHA-256 | `3fd65e917ded8b7d59a1f42051b69f41e4b5c24f583f9524deaccdfdfb1add66` |
| Composite installation SHA-256 | `8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084` |
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| P3 adoption artifact SHA-256 | `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359` |
| P3 golden source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| P1/P2 source fingerprint | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| PostgreSQL target | Major version 16; database `clarityit` |

Any change to a frozen artifact requires a new governed successor decision and
new evidence. A convenient recomputation or an equivalent-looking schema does
not supersede a signed identity.

## 5. WP-00 gate ledger

WP-00 gates are sequential authority boundaries. Passing a later implementation
test does not skip an earlier decision, and code presence does not close a gate.

| Gate | Status | Controlling repository evidence | Continuation rule |
|---|---|---|---|
| G0 — source freeze | Complete prerequisite | Repository history and WP-00 source baseline | Preserve; do not reopen implicitly |
| G1 — profiles and restore | **Closed**, 2026-08-01 | [`migrations/profiles/G1-APPROVALS.md`](../../migrations/profiles/G1-APPROVALS.md), closure commit `0dd21d8` | Development-only trust exception remains in force |
| G2 — schema decisions and target | **Closed**, 2026-08-02 | [`migrations/profiles/g2/G2-APPROVALS.md`](../../migrations/profiles/g2/G2-APPROVALS.md), signed commit `f04f94f...` | Target manifest is a read-only G3 input |
| G3 — reconciled baseline | **Closed, signed, integrated**, 2026-08-04 | [G3 approval receipt](../migration/wp00/g3/G3-APPROVALS.md), signed tip `97f83e4...`, integrated target `211c0ee1...` | Preserve identities and ancestry |
| G4 — Go migration runner | **Not started; not authorized** | WP-00 WS4/G4 contract | Requires separate explicit authorization and a package-specific plan |
| G5 — blocking CI matrix | **Not started** | WP-00 WS5/G5 contract | Begins only after G4 acceptance; existing green CI does not by itself pass G5 |
| G6 — WP-00 acceptance | **Not started** | WP-00 AC-00-01 through AC-00-30 and signed A11 evidence | No conditional or partial acceptance |

The immediate technical sequence is G4, then G5, then G6. This record does not
authorize any of them. It also does not authorize provider mutation, Site
Runtime, host agents, additional adapters, broader UI work, production cutover,
or WP-01 through WP-10 implementation.

## 6. Position against the final architecture

| Plane | Delivered and proven | Still required |
|---|---|---|
| 1. Experience | Product surfaces and first-release acceptance are specified | My Work, Case Workspace, Resources, live progress, accessibility, and pilot acceptance |
| 2. Authoritative control | Domain and database contracts are frozen through G3 | Control API, IAM isolation, Authority, Effect Broker, ingress, and gateways |
| 3. Intelligence and processing | Reason/Observe/Execute/Verify responsibilities and invariants are specified | Operational workers, independent verifier, retry/reconciliation, and failure proof |
| 4. Data and evidence | PostgreSQL target, fresh/adopted convergence, drift checks, atomicity, provenance, and G3 signatures | Runner, blocking CI, NATS transport, immutable object evidence, search, and later migration phases |
| 5. Trust services | Development custody exception and production exit topology are specified; G1 evidence controls passed for development | Runtime workload identity, secret brokering, policy distribution, trust-boundary enforcement, and fresh production trust domains |
| 6. Target | Provider-neutral and Proxmox boundaries are specified | Generic adapter, live Proxmox conformance, native enforcement, managed-system execution, and later Site Runtime |
| 7. Operational sources | Telemetry and independent health roles are specified | Source integrations and end-to-end observation/health evidence |

The project therefore has a signed authoritative database foundation, not a
delivered seven-plane runtime or an accepted product release.

## 7. Project completion path

Only WP-00 is currently a formal approved work package. The following names and
boundaries are the draft completion route defined by the Delivery Roadmap; each
later package requires its own approval and package plan before implementation.

| Release | Work packages | Required outcome | Current status |
|---|---|---|---|
| Foundation | WP-00, WP-01 | Reproducible migration foundation and authoritative kernel foundation | WP-00 through G3 only |
| R1 — Verified VM Recovery | WP-02, WP-03 | One governed `compute.virtual_machine.start` lineage on one enrolled non-production VM, independently verified and human-accepted | Not started |
| R2 — Private and Reusable Operations | WP-04, WP-05 | Private-zone execution plus reviewed knowledge and skills | Not started; blocked by R1 |
| R3 — Evented and Project Work | WP-06, WP-07 | Governed Signals, Routines, Projects, and software-delivery contexts | Not started |
| R4 — Controlled Scale and Extensibility | WP-08, WP-09 | Per-target multi-target truth plus second-provider/extension conformance | Not started |
| R5 — Production Rollout | WP-10 | Operable, secure, supportable production deployment | Not started |

Project completion means all of the following are true and recorded:

1. WP-00 reaches signed G6 with AC-00-01 through AC-00-30 evidenced.
2. Every approved successor package passes its entry gate and recorded release
   gate in dependency order through RG-10.
3. Release R5 is accepted against an executable deployment contract, migration
   and recovery evidence, security and accessibility criteria, service-level
   objectives, support readiness, and independent production trust/custody
   topology.
4. Production is provisioned fresh. CT 150 identities, root keys, service
   identities, storage credentials, and evidence keys are not promoted or
   reused.
5. Every consequential operation continues to use the sole governed dispatch,
   independent verification, and explicit outcome-decision model.
6. Deferred capabilities remain excluded unless an approved successor authority
   assigns them a typed capability, work package, and acceptance gate.

## 8. Historical-record precedence

Some evidence files deliberately preserve the state at the moment they were
created. Interpret them as follows:

- Blank approval rows or “awaiting approval” text in G1 capture/custody working
  records are historical. The later signed
  [`G1-APPROVALS.md`](../../migrations/profiles/G1-APPROVALS.md) is the G1 closure
  authority.
- Earlier G2 receipts and manifest identities are superseded by the signed
  [`G2-APPROVALS.md`](../../migrations/profiles/g2/G2-APPROVALS.md) identity
  `1f6e3142...` / `284064` bytes.
- The G3 receipt is bound to producing commit `570a0ec...`, Linux run
  `30900328914`, and the identities in section 4. PR #12's squash and subsequent
  no-content ancestry repair do not change those bytes.
- Pull-request state and branch heads are operational metadata. Commit IDs,
  signed receipts, artifact digests, and recorded ancestry are the durable
  authority.

## 9. Repository and external-state boundary

With the authority set indexed here, the repository contains the specifications,
delivery sequence, implementation state, gate criteria, frozen identities, and
continuation rules needed to plan the remainder of the project. Current chat
memory is not required to decide what is complete or what comes next.

The following remain external by design and must be represented in the
repository by immutable references, digests, receipts, and decisions rather
than copied into source control:

- raw P1/P2 manifests, database backups, restore logs, and sensitive evidence;
- production credentials, identity material, KES/KMS roots, evidence keys, and
  provider access;
- live infrastructure state, environment health, and provider observations; and
- future human approvals, release decisions, and operational go/no-go actions
  until their signed records are committed.

No credential, production hostname, customer data, secret, or raw production
dump may be added to make the repository appear self-contained.

## 10. Required start procedure for the next session

1. Fetch remote references and identify the commit that contains this file.
2. Read this file, the authority index, and the controlling specification for
   the requested gate.
3. Verify the G3 signed tip remains an ancestor of the integrated target and the
   frozen identities still match the signed receipts.
4. Confirm explicit authorization for the next gate. In the present state, that
   means G4 only; silence or a general request to continue is not authorization
   for G5, G6, provider mutation, or a later work package.
5. Create a normal forward branch and preserve all existing evidence and signed
   history. Do not force-push, rewrite, or delete recovery references.
6. Update this completion ledger in the same integration that changes a gate or
   project status. Record exact commits, CI runs, artifact digests, approvals,
   remaining blockers, and the next permitted action.

The completion ledger must never claim a gate because code exists, a CI job is
green outside the required boundary, a provider returned success, or a chat
participant stated approval. The gate's specified evidence and recorded decision
are required.
