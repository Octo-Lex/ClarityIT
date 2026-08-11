# ClarityIT v2 Project Completion Authority

**Status:** Current authoritative project handoff
**Snapshot date:** 11 August 2026
**Integrated baseline:** G5 acceptance integration (this change), descended from exact-main proof baseline `main@d39c44fe942a786be43c1931f4047bf6a57df36e`
**Current completed boundary:** WP-00 G5 accepted, signed, and enforced; required frontend + worker + `Backend (Go)` + `G5 Foundation Gate` merge predicate active for `main`
**Current authorized activity:** G6 final WP-00 acceptance under `G6-AUTH-2026-08-11`
**Next gate:** G6 — execute WS6 from the accepted G5 integration boundary; G6 remains unaccepted until AC-00-01 through AC-00-30 pass

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
| 7 | [WP-00 Plan v0.1](ClarityIT_v2_WP-00_Migration_Baseline_and_CI_Stabilization_Plan_v0.1.md) | Formal migration-foundation package, G0-G6, AC-00-01 through AC-00-30 | Active formal package; complete through G5 |
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
boundary without granting implementation authority or changing the WP-00 gate.

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

PR #13 separately integrated the completion authority and corrected architecture
suite without changing any G3 byte or identity:

| Item | Exact value |
|---|---|
| PR #13 source commit | `ed5e1fe2497eea9335f069920608e0d28b68b6e1` |
| PR #13 render commit | `60c64893708f530c812567d6e817e8222dfa3b4b` |
| PR #13 squash commit | `91fe4919e82044573241f24fdf619f0aef26bc84` |
| Provenance ancestry bridge | `ac7222737e14796174ed78420f1f388e6c21170b` |
| Current authority integration root | `wp00/g2-schema-decisions@ac7222737e14796174ed78420f1f388e6c21170b` |
| Role-based review | GitHub review `4859164706`; one delegated Product/Architecture/Security/Delivery assessment, not four independent attestations |

The squash, render tip, and bridge have the same tree. The bridge has the squash
as first parent and the render tip as second parent, so both the source and
render provenance commits are ancestors. The G3 signed tip and recovery branch
remain unchanged.

### G4/G5 integrated completion chain

| Item | Exact value |
|---|---|
| G4 implementation squash | `f769cd3815ea08194b56c267cfa3b30fb7a12fd9` |
| G4 exact integrated implementation/proof tip | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| G4 acceptance receipt integration | `ecb0ea48eb67bc07371b72e11517a77ad802d465` |
| G5 authorization integration | `ea231810ba3b858a78cdb25850ab3e0fd407a3f1` |
| G5 implementation squash | `a0be44780aa0f486bd6fb1d5fd5d87d26de09001` |
| G5 exact-main proof baseline | `d39c44fe942a786be43c1931f4047bf6a57df36e` |
| G5 exact-main G5 workflow | `WP-00 G5 Foundation Gate` run #11 — success |
| G5 exact-main ordinary CI | `CI` run #136 — success |
| Required-status ruleset | ID `20672081`, `WP-00 G5 Required Checks`, active on default branch, no bypass actors |
| Required contexts | `Frontend (typecheck · test · build)` AND `Worker (Python)` AND `Backend (Go)` AND `G5 Foundation Gate` |
| G5 receipt | [`docs/migration/wp00/g5/G5-APPROVALS.md`](../migration/wp00/g5/G5-APPROVALS.md) |
| G6 authorization | [`docs/migration/wp00/g6/G6-AUTHORIZATION-AND-PLAN.md`](../migration/wp00/g6/G6-AUTHORIZATION-AND-PLAN.md), `G6-AUTH-2026-08-11` |

G5 acceptance does not modify a frozen G1-G4 identity. It establishes the
required fail-closed merge predicate for the accepted foundation. The G6
authorization was recorded before G5 closure but explicitly remained inactive
until accepted G5; this G5 acceptance integration satisfies that prerequisite.

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
| G3 — reconciled baseline | **Closed, signed, integrated**, 2026-08-04 | [G3 approval receipt](../migration/wp00/g3/G3-APPROVALS.md), signed tip `97f83e4...`, authority integration root `ac722273...` | Preserve identities and ancestry |
| G4 — Go migration runner | **Accepted, signed, integrated**, 2026-08-10 | [G4 approval receipt](../migration/wp00/g4/G4-APPROVALS.md), implementation squash `f769cd3815ea08194b56c267cfa3b30fb7a12fd9`, authority tip `b31a7c5cd0ba132cb179db5751e8e2b8f339639f`, Linux CI run `31336112238` | Preserve accepted runner and proof identities |
| G5 — blocking CI matrix | **Accepted, signed, enforced**, 2026-08-11 | [G5 approval receipt](../migration/wp00/g5/G5-APPROVALS.md), implementation squash `a0be44780aa0f486bd6fb1d5fd5d87d26de09001`, exact-main proof `d39c44fe942a786be43c1931f4047bf6a57df36e`, active ruleset ID `20672081` | G6 authorization is now active; preserve required frontend + worker + `Backend (Go)` + `G5 Foundation Gate` conjunction |
| G6 — WP-00 acceptance | **Authorized, active, not accepted** | [G6 authorization](../migration/wp00/g6/G6-AUTHORIZATION-AND-PLAN.md), WP-00 AC-00-01 through AC-00-30 and A1-A7 evidence | Execute WS6 only; no conditional or partial acceptance |

The immediate technical sequence is now G6 final WP-00 acceptance. G6 is
already separately authorized by `G6-AUTH-2026-08-11`; accepted G5 activates
that authority. No additional G6 authorization is required. G6 does not
authorize provider mutation, Site Runtime, host agents, adapters, broader UI
work, production cutover, or WP-01 through WP-10 implementation.

## 6. Position against the final architecture

| Plane | Delivered and proven | Still required |
|---|---|---|
| 1. Experience | Product surfaces and first-release acceptance are specified | My Work, Case Workspace, Resources, live progress, accessibility, and pilot acceptance |
| 2. Authoritative control | Domain and database contracts are frozen through G3 | Control API, IAM isolation, Authority, Effect Broker, ingress, and gateways |
| 3. Intelligence and processing | Reason/Observe/Execute/Verify responsibilities and invariants are specified | Operational workers, independent verifier, retry/reconciliation, and failure proof |
| 4. Data and evidence | PostgreSQL target, fresh/adopted convergence, drift checks, atomicity, provenance, Go migration runner, and blocking G5 CI enforcement are proven | G6 P2 release rehearsal and final A1-A7 WP-00 acceptance evidence; NATS transport, immutable object evidence, search, and later migration phases remain downstream |
| 5. Trust services | Development custody exception and production exit topology are specified; G1 evidence controls passed for development | Runtime workload identity, secret brokering, policy distribution, trust-boundary enforcement, and fresh production trust domains |
| 6. Target | Provider-neutral and Proxmox boundaries are specified | Generic adapter, live Proxmox conformance, native enforcement, managed-system execution, and later Site Runtime |
| 7. Operational sources | Telemetry and independent health roles are specified | Source integrations and end-to-end observation/health evidence |

The project therefore has a signed authoritative database/migration foundation
and enforced foundation CI, not a delivered seven-plane runtime or an accepted
product release.

## 7. Project completion path

Only WP-00 is currently a formal approved work package. The following names and
boundaries are the draft completion route defined by the Delivery Roadmap; each
later package requires its own approval and package plan before implementation.

| Release | Work packages | Required outcome | Current status |
|---|---|---|---|
| Foundation | WP-00, WP-01 | Reproducible migration foundation and authoritative kernel foundation | WP-00 G5 accepted; G6 authorized and active; WP-01 not authorized |
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
- The G4 receipt is bound to the accepted G4 implementation/proof chain and
  Linux run `31336112238`; later G5 CI governance does not modify that evidence.
- The G5 receipt binds the G5 implementation, exact-main proof baseline,
  required-status ruleset, and role-based decisions. Ruleset/UI evidence is
  preserved as sanitized metadata and does not replace the workflow evidence.
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
2. Read this file, the authority index, the G5 acceptance receipt, the G6
   authorization, and the WP-00 plan.
3. Verify the frozen G1-G4 identities remain unchanged, the G5 receipt is
   integrated, and the repository still requires `Frontend (typecheck · test · build)`,
   `Worker (Python)`, `Backend (Go)`, and `G5 Foundation Gate` for `main`.
4. Bind the accepted G5 integration tip as the G6 starting baseline. Execute
   only the already-authorized WS6/G6 work: AC-00-01 through AC-00-30 evidence
   crosswalk, P2 release-artifact rehearsal, recovery/failure rehearsal,
   historical-truth confirmation, final schema/security/provenance review,
   A7 release manifest, issue #1 disposition, and required G6 decisions.
5. Create a normal forward branch (`wp00/g6-acceptance`) from the accepted G5
   integration tip and preserve all existing evidence and signed history. Do
   not force-push, rewrite, or delete recovery references.
6. Update this completion ledger in the same integration that changes G6 or
   project status. Record exact commits, CI runs, artifact digests, approvals,
   remaining blockers, and the next permitted action.

The completion ledger must never claim a gate because code exists, a CI job is
green outside the required boundary, a provider returned success, or a chat
participant stated approval. The gate's specified evidence and recorded decision
are required.
