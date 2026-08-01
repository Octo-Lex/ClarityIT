# ClarityIT v2 Delivery Roadmap

**Migration-first sequence from authoritative foundation to controlled operational scale**

**Version:** 0.2  
**Status:** Draft for product, architecture, and engineering approval  
**Date:** 1 August 2026  
**Roadmap basis:** Product Definition v0.1, Authoritative Execution Kernel Specification v0.1, v1-to-v2 Compatibility and Migration Specification v0.1, Layered System Architecture, and Native Pattern Specification v0.1

> **Roadmap decision:** WP-00 remains the hard prerequisite. The first product release remains one provider-neutral `compute.virtual_machine.start` lineage on one enrolled non-production virtual machine. Every later pattern is sequenced behind evidence that the authoritative kernel, provider adapter, independent verifier, and Case experience work end to end.

## 1. Roadmap authority and use

This roadmap converts the current Product Definition delivery sequence into implementation work packages with explicit pattern ownership, entry criteria, deliverables, release evidence, and exit gates. It does not change the Product Definition, execution kernel, migration specification, or first-release scope.

Only WP-00 is already a formally named v2 delivery package. WP-01 through WP-10 are proposed package names and boundaries. They become authoritative only after approval and creation of their package plans.

Calendar dates are intentionally omitted until team capacity, WP-00 exit evidence, and pilot-environment readiness are known. Dependency gates, not forecast dates, control sequencing.

## 2. Non-negotiable sequencing rules

1. WP-00 completes under its existing G6 acceptance boundary before feature work from later packages is merged into the release path.
2. WP-01 establishes canonical objects, invariants, persistence, identity, authority, claims, verification, evidence, and isolation before a live provider mutation.
3. WP-02 proves the generic capability and provider conformance through the sole broker dispatch path.
4. WP-03 completes the product experience and controlled non-production pilot; this creates Release R1.
5. Private access, knowledge, routines, projects, multi-target execution, and provider expansion are follow-on releases, not parallel first-release scope.
6. A new mutation path cannot be introduced by a routine, Project integration, playbook, Site Runtime, extension, or UI shortcut. It must begin as a versioned typed capability and satisfy the kernel.
7. Multi-target execution cannot start before the single-target release is accepted. A second provider cannot weaken the first adapter's contract.
8. Each package must leave blocking CI green, preserve migration compatibility, and add evidence-backed failure tests before exit.

## 3. Release and work-package overview

| Release | Work packages | User-visible outcome | Hard release boundary |
|---|---|---|---|
| Foundation | WP-00, WP-01 | Reproducible platform and authoritative kernel foundation | No provider mutation required |
| R1 - Verified VM Recovery | WP-02, WP-03 | One operator can safely start one enrolled non-production VM and accept independently verified recovery | All product and kernel first-release criteria pass live |
| R2 - Private and Reusable Operations | WP-04, WP-05 | The same governed model reaches one private zone and uses reviewed knowledge and skills | Site Runtime and knowledge governance pass |
| R3 - Evented and Project Work | WP-06, WP-07 | Signals, schedules, and delivery contexts create accountable Cases and reviewed work | No direct mutation outside the kernel |
| R4 - Controlled Scale and Extensibility | WP-08, WP-09 | Bounded multi-target work and a second provider/extension contract | Per-target truth and cross-provider conformance pass |
| R5 - Production Rollout | WP-10 | Operable, supportable, secure wider deployment | Production readiness and go/no-go evidence pass |

## 4. Pattern-to-roadmap map

| Pattern | Foundation | First complete implementation | Later maturation |
|---|---|---|---|
| P-01 Governed Case Workspace | WP-01 domain skeleton | WP-03 | WP-06, WP-07 |
| P-02 Identifiable Human-Agent Participation | WP-01 | WP-03 UX proof | WP-05 skills, WP-07 Projects |
| P-03 Typed Work Record and Provenance Timeline | WP-01 | WP-03 | Every later package |
| P-04 Evidence-Backed Operational Answer | WP-01 schema | WP-03 basic | WP-05 reviewed retrieval |
| P-05 Resource-Aware Bounded Context | WP-01 ordered overlay and anti-shadowing contract | WP-03 | WP-05, WP-07 |
| P-06 Intent-to-Immutable Operation Packet | WP-01 | WP-02 | Every later capability |
| P-07 Typed Capability and Adapter Boundary | WP-01 registry skeleton | WP-02 | WP-09 SDK and second provider |
| P-08 Scoped Authority and Trusted Dispatch | WP-01 | WP-02 | WP-04, WP-08, WP-09 |
| P-09 Credentialless Reasoning and Executor-Held Secrets | WP-01 entitlement schema | WP-02 destination-bound broker | WP-04 local broker; WP-09 SDK conformance |
| P-10 Independent Verification and Human Acceptance | WP-01 | WP-02 and WP-03 | Every later capability |
| P-11 Successor Correction and Compensation | WP-01 | WP-03 | WP-08 partial recovery |
| P-12 Deterministic Site Runtime and Private Access | Architecture only | WP-04 runtime profile and migration refusal | WP-09 route SDK |
| P-13 Reviewed Operational Knowledge | Retained foundation | WP-05 | WP-06, WP-07 |
| P-14 Versioned Operational Skill and Playbook | Schema concepts only | WP-05 signed scope-owned packages | WP-06 routines |
| P-15 Signal-Triggered Routine and Exception Case | Signal ingestion retained | WP-06 Routine Principal, Fire Key, and destination binding | Production tuning in WP-10 |
| P-16 Project and Software-Delivery Context Binding | Existing project objects retained | WP-07 | Future typed delivery effects |
| P-17 Controlled Multi-Target Execution | Prohibited before R1 | WP-08 | WP-09 cross-provider sets |
| P-18 Workspace Isolation and Sovereign Deployment | WP-00 and WP-01 | Exercised in every gate | WP-10 executable deployment contract and production proof |

## 5. Work-package specifications

### WP-00 - Migration Baseline and CI Stabilization

**Status:** Existing formal package.  
**Objective:** Establish a reproducible source, database, restore, migration, and CI foundation before v2 feature work.

#### Entry

Current v1 repository and deployed-state discrepancy, broken legacy migration chain, non-blocking backend CI, and unresolved source schema profiles.

#### Required scope

- freeze the exact source and deployed artifact basis;
- capture and approve production and restored-backup source profiles;
- reconcile schema fingerprints and live-code deltas;
- prove backup restoration;
- establish the reconciled baseline and immutable checksummed migration runner;
- remove backend `continue-on-error` and make required suites blocking;
- implement the WP-00 evidence, restore, reconciliation, and acceptance plan already defined.

#### Explicit exclusions

No new Case experience, Site Runtime, operational skill, routine, Project integration, multi-target execution, second provider, or extension SDK belongs in WP-00.

#### Exit

WP-00 exits only under its existing G6 acceptance boundary. Its evidence becomes the input to WP-01; pilot or pattern implementation cannot substitute for missing baseline, restore, or CI evidence.

### WP-01 - Authoritative Kernel Foundation

**Objective:** Introduce the canonical v2 domain and truth model alongside the stabilized v1 spine without performing a live consequential provider mutation.

**Pattern ownership:** P-02, P-03, P-05, P-06, P-08, P-09, P-10, P-11, and P-18; skeleton for P-01, P-04, and P-07.

#### Entry criteria

- WP-00 G6 accepted.
- Product Definition, Kernel Specification, Migration Specification, Layered Architecture, Native Pattern Specification, and this roadmap approved as one authority set.
- No unresolved schema-profile or CI blocker.

#### Deliverables

1. **Canonical schema:** Cases, Resources, provider bindings, Observations, Operation Packets, policy and approval decisions, grants, attempts, receipts, claims, Verification Specs, Verifications, Outcome Decisions, evidence manifests, v2 inbox/outbox, and migration mappings.
2. **Principal model:** human, reasoning, service, policy, execution workload, and source identities with workspace scope and provenance.
3. **State machines:** legal and illegal transitions, aggregate-version concurrency, successor lineage, and explicit unknown/inconclusive states.
4. **Authority service:** deterministic policy evaluation, decision separation, grant lifecycle, replay protection, and separation of duties.
5. **Effect Broker skeleton:** sole dispatch API, route resolution contract, preflight, reservation, idempotent attempt creation, and no-op/fake adapter for contract tests only.
6. **Verifier contract:** versioned spec, fresh input rules, pass/fail/inconclusive, and evidence sealing.
7. **Context overlay contract:** bounded Resource-aware context bundle with ordered organization, workspace, Case/Resource, role/task, and personal-draft overlays; provenance, authority classes, screening states, monotonic policy tightening, authoritative-namespace anti-shadowing, topology limits, exclusions, and deterministic composition digest.
8. **Trust controls:** workload identity contract, secret references, route binding, signature and canonicalization libraries, plus the schema and policy evaluator for destination-bound credential-broker entitlements.
9. **Compatibility foundation:** additive schema, feature flags off, v1 read compatibility, and explicit rejection of unsafe legacy execution after the later cutover.

#### Acceptance gate RG-01

- all kernel invariants and legal/illegal transition tests pass;
- authoritative state, audit, and outbox commit atomically;
- inbox deduplication, replay, lease loss, and restart tests pass;
- no agent, browser, or compatibility path can issue authority or dispatch directly;
- overlay permutation, policy-relaxation, deny-removal, cross-workspace, and authoritative-content shadowing tests fail closed and reconstruct the composition digest;
- workspace isolation and secret scanning pass across database, API, events, storage references, caches, and search fixtures;
- historical backfill creates zero passed Verifications and zero Accepted outcomes;
- evidence manifests reconstruct synthetic happy, blocked, failed, unknown, and successor lineages;
- blocking CI remains green on fresh install and approved upgrade profiles.

### WP-02 - Governed Compute Vertical Slice

**Objective:** Implement the first real provider-neutral capability and its initial provider conformance profile through the authoritative kernel.

**Pattern ownership:** complete P-06, P-07, P-08, P-09, and the machine side of P-10.

#### Scope

- `compute.virtual_machine` Resource and Observation schema;
- `compute.virtual_machine.start@1` capability registry entry;
- generic adapter contract and conformance harness;
- initial Proxmox VE adapter profile: exact binding, read, prepare, start submit, UPID persistence, polling, result read, and reconciliation;
- least-privilege credential with observe/start/task-read only;
- adapter-internal credential broker with workload, workspace, route, target, capability, packet, grant, attempt, HTTPS host/port, method, normalized-path, header, redirect, validity, use-limit, and sanitization binding;
- central connector route only for the first live slice;
- HTTPS verifier profile with TLS validation, 5-second probe timeout, three consecutive 2xx results, and 120-second window;
- complete failure taxonomy and sanitized receipts;
- controlled non-production environment and deterministic fixtures.

#### Explicit exclusions

No VM stop, restart, reset, snapshot, resize, migrate, delete, container, production target, multi-target, Site Runtime production route, SSH, Kubernetes, database, browser, desktop, or Git mutation.

#### Acceptance gate RG-02

- generic compute contract contains no provider-specific core field;
- adapter contract tests cover success, rejection, authentication, authorization, pre-send transport failure, ambiguous send, timeout, duplicate delivery, restart, polling, cancellation semantics, result mismatch, and reconciliation;
- one logical idempotency key creates at most one provider start operation;
- UPID is stored only as provider operation identity and claim evidence;
- provider completion cannot directly create Verification or Accepted;
- credential-broker conformance rejects every workload, route, target, packet, host, port, method, path, header, redirect, expiry, use-limit, and revocation mismatch before credential injection;
- broker audit evidence explains each permitted call without retaining a secret or replayable authenticated request;
- no credential appears in agent context, client payload, packet, message, log, receipt, or evidence export;
- controlled live adapter tests pass against one enrolled non-production VM;
- CI and migration compatibility remain green.

### WP-03 - Case Experience and Verified Pilot

**Objective:** Deliver the outcome-centered product experience and accept the first release through a complete live lineage.

**Pattern ownership:** complete P-01, P-02, P-03, P-04 basic form, P-05 first-release context, the human side of P-10, and the first P-11 correction path.

#### Scope

- My Work views for active Cases, approvals, agent-prepared work, execution uncertainty, verification failures, and outcomes awaiting acceptance;
- Case creation from My Work or a Resource with objective, owner, target, and success criteria;
- Case Workspace sections for findings, source references, immutable packet, policy, approval, grant, execution, Verification, outcome, and evidence;
- Resource detail with stable identity, provider binding, ownership, current Observations, capabilities, and health contract;
- source-attributed answer artifacts and bounded context inspection;
- inspection of applied context overlays, authority classes, rejected collisions, omissions, freshness, and screening or quarantine state;
- generic effect and redacted provider translation preview;
- live WebSocket progress from committed events only;
- human acceptance, rejection, correction, and separately authorized compensation proposal;
- evidence export and reviewer reconstruction;
- keyboard, focus, contrast, status-not-by-color-alone, and clear-error accessibility requirements.

#### Acceptance gate RG-03 / Release R1

- all 59 Product Definition first-release criteria and applicable kernel tests pass;
- live happy path proves baseline -> packet -> policy -> approval -> grant -> attempt -> provider completion -> result Observation -> HTTPS Verification -> human acceptance;
- stale baseline, denial, expiry, duplicate command, provider failure, ambiguous result, state mismatch, verifier failure, restart, and correction scenarios pass;
- personal drafts, retrieved content, prior Case prose, and generated text cannot shadow policy, capabilities, Resources, bindings, approved knowledge, or current Observations in the live Case path;
- Submitted, provider completed, observed, Verified, and Accepted are distinct in APIs, state, and UI;
- an independent reviewer reconstructs the complete lineage from the evidence manifest;
- fresh install, approved upgrade, and controlled pilot run are repeatable;
- no severity-one or severity-two defect remains open;
- product and engineering representatives record Release R1 acceptance.

### WP-04 - Site Runtime and Private Access

**Objective:** Execute the already-proven capability through one private network zone without changing product, packet, authority, or verification semantics.

**Pattern ownership:** P-12 and local maturation of P-08, P-09, and P-18.

#### Scope

- signed Go Site Runtime and Remote-Site Edge Gateway protocol;
- attested workload identity and outbound-initiated mTLS;
- signed route/adapter inventory, policy bundle, and compatibility handshake;
- signed Runtime Capability Profile covering protocol and adapter versions, durable state, encrypted journal and spools, identity, local policy, secret resolution, egress grade, backup/restore, blob staging, clock, resource budgets, upgrade, rollback, and offline guarantees;
- packet and route minimum-profile evaluation plus a signed Migration Gap Report for route or runtime movement;
- local packet/grant/target validation and short-lived local credential resolution;
- local destination-bound credential broker conforming to the central entitlement and sanitization contract;
- encrypted append-only journal, receipt/Observation spool, sequence/acknowledgment, reconnect, and deduplication;
- disconnect-before-dispatch deny and disconnect-after-submit polling/reconciliation behavior;
- signed staged upgrade and rollback to last trusted version;
- one private-network route for the existing VM-start capability;
- constrained private-access adapter research for the next slice; no general terminal product.

#### Acceptance gate RG-04 / Release R2A

- the same approved packet can use an authorized site route with no capability change;
- route change triggers required policy re-evaluation and new grant;
- invalid identity, signature, nonce, route, target, expiry, clock, policy, or version blocks locally;
- missing, stale, unsigned, identity-mismatched, incompatible, or weaker Runtime Capability Profiles block dispatch;
- migration is refused when the destination weakens required authority, isolation, durability, secret, egress, verification, evidence, or recovery capabilities, and in-flight tests preserve single ownership and sequence continuity;
- offline journal survives restart and replays centrally exactly once;
- disconnect scenarios preserve true unknown and submitted states;
- runtime binary contains no model runtime or autonomous planner;
- compromised runtime revocation and upgrade rollback pass;
- one private-zone live pilot produces a complete central evidence lineage.

### WP-05 - Reviewed Knowledge and Operational Skills

**Objective:** Make resolved experience reusable through reviewed knowledge and versioned playbooks without turning history into authority.

**Pattern ownership:** complete P-13 and P-14; mature P-04 and P-05.

#### Scope

- knowledge candidate, review, approval, publication, supersession, retirement, and review-date lifecycle;
- source manifests, applicability, exclusions, sensitivity, ownership, version digest, and access scope;
- workspace-safe PostgreSQL full-text and vector retrieval with authority and freshness labels;
- conflict, stale, missing-source, and access-limited behavior;
- signed scope-owned operational skill package with manifest, inputs, steps, capability refs and approved ceiling, preconditions, stop conditions, authority profile, evidence, verifier, correction guidance, dependencies, file allowlist, signature, promotion lineage, and materializer compatibility;
- explicit `draft -> review_pending -> reviewed -> published -> superseded | archived` lifecycle and organization-promotion review;
- deterministic isolated materialization with exact package, dependency, selected-file, materializer, and resulting tree digests;
- skill conformance fixtures and review workflow;
- Case-to-knowledge and Case-to-skill candidate submission;
- one reviewed VM-recovery knowledge item and one reviewed VM-recovery playbook instantiated through the existing kernel.

#### Explicit exclusions

No automatic publication, skill marketplace, arbitrary code execution, standing grant, or capability creation from prose.

#### Acceptance gate RG-05 / Release R2B

- unreviewed Case content never appears as approved knowledge;
- retrieval isolation, sensitivity, exact version, source, conflict, and retirement tests pass;
- playbook instantiation creates current packets and authority; it reuses no historical approval or grant;
- only published, correctly scoped, signed packages materialize; promotion creates a new reviewed signature and workspace packages cannot shadow organization packages;
- identical signed inputs produce the same materialized tree without floating dependencies, mutable references, undeclared files, or undeclared network access;
- secret and undeclared-input tests pass;
- one reviewed playbook completes the R1 workflow with the same execution and Verification evidence;
- knowledge and skill history remains reconstructable after supersession and retirement.

### WP-06 - Signals and Routines

**Objective:** Turn alerts, schedules, requests, and webhooks into accountable recurring work and exception Cases.

**Pattern ownership:** P-15, using P-13 and P-14.

#### Scope

- Signal schema, source authentication, normalization, Resource resolution, deduplication, correlation, and payload digest;
- Routine definition, trigger version, owner, dedicated least-privilege Routine Principal, target rule, quiet window, concurrency, catch-up behavior, exception condition, and review date;
- versioned person-directed and notification destination bindings with verified identity, purpose, consent or approved policy basis, suppression, opt-out, quiet-window, and revocation controls;
- deterministic Fire Key and durable atomic firing ledger covering trigger occurrence, destination binding, and resolved target result;
- routine health, missed schedule, source outage, storm control, and operator intervention;
- read-only checks and normal-completion confirmation;
- exception Case creation/advancement and reviewed playbook instantiation;
- packet drafting for consequential steps, with the normal authority path preserved;
- routine evidence and operational dashboards.

#### Acceptance gate RG-06 / Release R3A

- duplicates and storms obey dedupe, correlation, rate, and concurrency budgets;
- replay, retry, lease-loss, restart, and catch-up tests produce one logical fire, Case, notification, packet draft, and playbook instance for each Fire Key;
- Routine Principals never inherit owner permissions, hold provider credentials, approve work, or issue grants;
- unverified, revoked, suppressed, non-consenting, cross-workspace, or purpose-mismatched destinations fail before delivery, and a destination change creates a successor Routine version;
- ambiguous targets cannot create a consequential packet;
- schedules and webhooks have no provider mutation or grant interface;
- missed-run and source-outage policies are visible and testable;
- an exception routine opens one Case with source-attributed Signals and, after normal authority, completes the verified slice;
- disabling or superseding a routine prevents new instances without erasing history.

### WP-07 - Projects and Software-Delivery Contexts

**Objective:** Organize longer-lived objectives and software-delivery evidence while retaining Case ownership for governed effects.

**Pattern ownership:** P-16; mature P-01, P-02, P-04, and P-05.

#### Scope

- Project aggregate, milestones, dependencies, owners, participants, linked Work Items and Cases;
- external delivery-context bindings for repository, branch, commit, review, build, environment, and release identities;
- source-attributed observations of external delivery state;
- revision-bound repository context bundles for reasoning and review;
- Project views for decisions, evidence, blockers, releases, and intervention needs;
- explicit boundary that Project metadata, branch state, review state, or pipeline status creates no execution authority;
- read-only repository and delivery-system integrations first.

#### Explicit exclusions

No custom source-control hosting, general chat product, unrestricted coding agent, or Git/deployment mutation path in this package.

#### Acceptance gate RG-07 / Release R3B

- every external reference preserves provider identity and observed immutable revision;
- force-push, rebase, rename, deletion, and permission-loss scenarios preserve historical evidence;
- repository context obeys workspace, path, and commit scope;
- a Project can organize multiple Cases without merging their authority or outcomes;
- external review and pipeline state cannot create a grant, Verification, or Accepted outcome.

### WP-08 - Controlled Multi-Target Execution

**Objective:** Extend one proven capability to a small bounded target set while preserving per-target truth and failure isolation.

**Pattern ownership:** P-17; scale P-06 through P-11 and P-12 where applicable.

#### Entry criteria

- Release R1 accepted and stable through an agreed observation period.
- WP-04 site-route conformance accepted if multi-zone targets are included.
- No unresolved severity-one or severity-two execution, idempotency, verification, or evidence defect.

#### Scope

- immutable target-set snapshot from exact Resource IDs and binding versions;
- policy-defined initial target limit, one capability, one provider, and non-production environment;
- batch plan with concurrency, rate, zone, maintenance, failure, and stop budgets;
- per-target authority, attempt, provider operation, Observation, Verification, outcome, and successor lineage;
- derived aggregate progress with partial, unknown, stopped, and completed classifications;
- restart, lease loss, threshold stop, changed baseline, partial failure, and correction flows;
- operator intervention and evidence export at batch and target levels.

#### Acceptance gate RG-08 / Release R4A

- dynamic selectors resolve and freeze before approval;
- target-count, concurrency, rate, zone, and failure budgets are enforced server-side;
- each target receives at most one logical provider submission under duplicate/restart tests;
- aggregate status cannot overwrite or obscure per-target state;
- partial failure and unknown results produce correct successor choices;
- a controlled live batch passes with complete per-target evidence.

### WP-09 - Multi-Provider Expansion and Extension SDK

**Objective:** Prove that provider neutrality is real and expose a controlled extension boundary after the first adapter and scale semantics are stable.

**Pattern ownership:** mature P-07, P-08, P-09, P-10, P-12, and P-18.

#### Scope

- one second provider profile for an already-specified capability before any new broad capability family;
- cross-provider generic contract and golden conformance suite;
- signed extension manifest and SDK for adapter, observer, verifier, connector, context-source, and UI-extension roles as approved;
- version negotiation, compatibility window, route declarations, permission manifest, secret references, evidence hooks, health, and upgrade behavior;
- Runtime Capability Profile requirements and downgrade refusal for connector and Site Runtime extensions;
- credential-broker entitlement hooks that prevent an extension from exposing generic authenticated transport or broadening destination scope;
- deployment clause declarations distinguishing runtime-enforced, validation-only, and reserved extension obligations;
- sandbox or process isolation, workload identity, resource budgets, and revocation;
- extension registry and operator approval lifecycle;
- documentation and test harness that prevent authority, database, and credential bypass.

#### Acceptance gate RG-09 / Release R4B

- the second provider implements the same product capability without core packet or Case changes;
- all cross-provider failure and reconciliation fixtures pass;
- extensions cannot write kernel tables, issue grants, access secret values outside their workload scope, or publish unpersisted truth;
- connector and runtime extensions pass destination-bound credential-broker and Runtime Capability Profile conformance, including downgrade and migration-gap refusal;
- extension clause-status tests reject unsupported runtime-enforcement claims;
- signed versioning, revocation, upgrade, rollback, and compatibility tests pass;
- workspace and route isolation pass for concurrent providers and extensions;
- operator documentation and conformance evidence are complete.

### WP-10 - Production Hardening and Wider Rollout

**Objective:** Convert the accepted non-production capabilities into an operable, supportable, secure production service without widening semantic scope implicitly.

**Pattern ownership:** production proof for P-18 and operational proof for every released pattern.

#### Scope

- SLOs, error budgets, capacity and queue limits, disaster recovery, backup/restore, key rotation, retention, legal hold, and evidence export operations;
- production deployment topology, upgrade rings, rollback posture, compatibility, and support runbooks;
- version-controlled executable Deployment Contract Directory binding product and deployment-tool versions, configuration schema, migration range, policy and extension descriptors, runtime profiles, secret-reference routes, and immutable artifact/image digests;
- deterministic `check`, read-only `doctor`, reviewable `plan`, controlled idempotent `up`, and live `verify-drift` operations with persisted evidence and explicit exit classifications;
- immutable pre-change deployment and rollback manifests, database migration position, backup or snapshot references, forward-recovery instructions, and decision record;
- deployment-clause register using `ENFORCED`, `VALIDATED-ONLY`, and `RESERVED`, with evidence that prevents status inflation;
- threat model, penetration testing, supply-chain verification, secret scanning, policy review, and workspace isolation at scale;
- operational dashboards for Cases, execution unknowns, verifier health, route health, routine health, stale knowledge, failed projections, and evidence sealing;
- accessibility, performance, internationalization readiness, admin and support controls;
- controlled production enrollment policy and staged rollout;
- final contract decisions for legacy write paths only after retention and rollback criteria are satisfied.

#### Acceptance gate RG-10 / Release R5

- SLO, load, soak, failover, restore, key-rotation, retention, and incident exercises pass;
- no unresolved severity-one or severity-two defect or unowned operational risk remains;
- independent security, privacy, accessibility, migration, and evidence reviews pass;
- release artifacts include signatures, SBOM, provenance, migration range, and configuration schema;
- fresh-install, upgrade, rollback, forward-recovery, and drift exercises reproduce the declared manifests, preserve authoritative history and workspace isolation, and truthfully classify every deployment clause;
- deployment tooling cannot issue operational grants, call managed-system capabilities, or manufacture Verification, and deployment success remains distinct from operational outcome proof;
- production go/no-go is recorded by product, engineering, operations, security, database, and quality owners;
- rollout begins with a small named resource scope and explicit rollback/forward-recovery posture.

## 6. Cross-package dependency matrix

| Package | Depends on | May overlap with | Must not begin before |
|---|---|---|---|
| WP-00 | None | Documentation only | Current package authority |
| WP-01 | WP-00 | UX research; non-implementing design | WP-00 G6 |
| WP-02 | WP-01 core contracts | WP-03 design only | RG-01 |
| WP-03 | WP-02 live backend | Final WP-02 defect closure | Stable end-to-end backend path |
| WP-04 | R1 | WP-05 design | RG-03 / R1 acceptance |
| WP-05 | R1 | WP-04 after contracts stabilize | RG-03 |
| WP-06 | WP-05 | WP-07 read-only integration | RG-05 |
| WP-07 | R1; P-05 context | WP-06 | RG-03 |
| WP-08 | R1; WP-04 if private/multi-zone | WP-09 design only | RG-03 and observation period |
| WP-09 | WP-02 conformance; preferably WP-08 | WP-10 planning | One stable provider and extension threat model |
| WP-10 | All packages selected for production | Contract planning | Required release gates accepted |

## 7. Decision gates and evidence owners

| Gate | Decision | Minimum evidence | Required sign-off |
|---|---|---|---|
| WP-00 G6 | Baseline and CI are reliable | Existing WP-00 evidence pack | Existing WP-00 authorities |
| RG-01 | Kernel foundation is semantically correct | Transition, persistence, isolation, context-overlay, anti-shadowing, migration, evidence tests | Architecture, backend, database, security, quality |
| RG-02 | Provider-neutral compute execution is conformant | Adapter, live provider, idempotency, destination-bound credential-broker, secret, verifier evidence | Architecture, backend, operations, security, quality |
| RG-03 | First product release is accepted | All product/kernel criteria and live Case evidence | Product and engineering; operations/security participate |
| RG-04 | Site Runtime is trusted | Identity, Runtime Capability Profile, migration-gap refusal, credential broker, offline, local-policy, upgrade, private-pilot evidence | Architecture, operations, security, quality |
| RG-05 | Knowledge and skills are governable | Review lifecycle, retrieval isolation, package signature/scope/promotion, deterministic materialization, skill conformance | Product, operations, security, quality |
| RG-06 | Routines cannot bypass governance | Routine Principal, Fire Key/replay, destination consent, delivery, Signal/routine failure and exception Case evidence | Product, operations, security, quality |
| RG-07 | Project contexts preserve authority boundaries | Revision, binding, context, and non-authority tests | Product, engineering, security, quality |
| RG-08 | Multi-target work preserves per-target truth | Budget, partial failure, restart, live batch evidence | Product, architecture, operations, security, quality |
| RG-09 | Extensibility preserves kernel invariants | Second-provider, SDK, runtime-profile, credential-broker, and clause-status conformance evidence | Architecture, engineering, security, quality |
| RG-10 | Production service is operable and safe | Executable deployment contract, clause truth, drift, rollback/forward recovery, SLO, DR, security, accessibility, migration, support evidence | Product, engineering, operations, security, database, quality |

## 8. Roadmap metrics

Progress is measured by proof, not feature count.

| Dimension | Metric | Required interpretation |
|---|---|---|
| Authority | Consequential provider calls outside broker | Must remain zero |
| Idempotency | Duplicate logical submissions | Must remain zero in conformance scenarios |
| Truth | Provider completion promoted directly to Verification/Accepted | Must remain zero |
| Migration | Legacy passed Verifications or Accepted outcomes created | Must remain zero |
| Evidence | Terminal/superseded lineages with sealed reconstructable manifest | 100% |
| Identity | Unattributed authoritative transitions | Must remain zero |
| Secrets | Credential findings in prohibited surfaces | Must remain zero |
| Isolation | Cross-workspace access findings | Must remain zero |
| Context | Policy relaxations or authoritative-namespace shadowing accepted through overlays | Must remain zero |
| Verification | Passed results without exact spec and fresh sealed inputs | Must remain zero |
| Reliability | Unknown outcomes automatically resubmitted | Must remain zero |
| Knowledge | Published items without source, owner, reviewer, scope, and digest | Must remain zero |
| Skills | Consequential steps executed without current packet and grant | Must remain zero |
| Skills | Published packages without valid signature, scope owner, promotion review, or deterministic materialization digest | Must remain zero |
| Routines | Duplicate logical fires for one Fire Key or deliveries without a valid destination basis | Must remain zero |
| Runtime | Dispatches on missing, stale, incompatible, or weaker-than-required Runtime Capability Profiles | Must remain zero |
| Deployment | Clauses reported as `ENFORCED` without runtime enforcement evidence | Must remain zero |
| Drift | Released deployments with undeclared artifact, configuration, migration, policy, extension, or runtime-profile drift | Must remain zero |
| Multi-target | Targets executed outside approved immutable snapshot | Must remain zero |

## 9. Documentation incorporation plan

### Repository artifacts to add

1. `docs/v2/ClarityIT_v2_Native_Pattern_Specification_v0.1.md`
2. `docs/v2/ClarityIT_v2_Delivery_Roadmap_v0.2.md`

### Existing artifacts to update

| Artifact | Required change |
|---|---|
| `docs/v2/README.md` | Add both documents to the authority index; state that only WP-00 is currently formal and later package names are proposed until approved. |
| Product Definition section 10.2 | Retain the eight product-delivery decisions; add a concise mapping to WP-00 through WP-10 and point to the roadmap for package gates. |
| Product Definition section 15 | Add Native Pattern Specification as the authority for reusable experience/orchestration patterns and Delivery Roadmap as planning authority. |
| Kernel Appendix B | Add Native Pattern Specification, Context Overlay, Executor Credential Broker, Site Runtime Capability Profile and Migration, Knowledge Governance, Operational Skill Package, Signal and Routine, Project Context, Multi-Target Coordination, Extension SDK, and Executable Deployment Contract specifications. |
| WP-00 plan | Add a boundary note that no later pattern implementation is part of WP-00 and that its G6 remains the prerequisite. |
| Future WP plans | Include pattern IDs, kernel invariants, entry gate, exclusions, acceptance evidence, and sign-off owners. |

### Replacement text for Product Definition section 10.2

> Delivery remains migration-first and proof-gated. WP-00 stabilizes the migration baseline, restore evidence, and blocking CI. WP-01 introduces the authoritative kernel foundation. WP-02 implements the provider-neutral VM-start capability and initial provider conformance. WP-03 delivers the Case experience and accepts the verified non-production pilot. Only after Release R1 may WP-04 add the Site Runtime, WP-05 add reviewed knowledge and operational skills, WP-06 add Signals and Routines, WP-07 add Project and software-delivery contexts, WP-08 add controlled multi-target execution, WP-09 add a second provider and extension SDK, and WP-10 complete production hardening and rollout. The Native Pattern Specification defines the reusable pattern contracts; the Delivery Roadmap defines package ownership and gates.

## 10. Explicitly deferred capabilities

The following remain outside the roadmap until an approved revision assigns a typed capability, pattern, work package, and acceptance gate:

- unrestricted shell or terminal agent execution;
- credential-bearing AI workers;
- production autonomous remediation without policy-defined authority and acceptance;
- automatic playbook or knowledge publication from Case history;
- general workflow/no-code marketplace;
- arbitrary browser or desktop mutation;
- Kubernetes, database, public-cloud, identity, Git, or deployment mutation beyond a separately specified vertical slice;
- organization-wide or irreversible multi-target actions;
- decentralized identity, peer-to-peer execution, custom source-control hosting, or generalized multi-agent swarms;
- provider-specific product states or provider-prefixed v2 capabilities.

## 11. Immediate next actions

1. Approve the Native Pattern Specification and this roadmap as draft v0.1/v0.2 authorities.
2. Add both Markdown artifacts to `docs/v2` and update the v2 document index.
3. Add the boundary note to WP-00 without changing its scope or G6.
4. Create WP-01 only after WP-00 evidence is accepted; use RG-01 as its exit gate.
5. Create the Generic Compute Adapter and Proxmox profile documents during WP-02 planning.
6. Create the First-Release Experience Specification before WP-03 implementation completion.
7. Derive the Context Overlay and Executor Credential Broker contracts during WP-01 planning; complete their live conformance in WP-02 and WP-03.
8. Create the Runtime Capability Profile and Migration Contract before WP-04 route implementation; create the signed Skill Package and Signal/Routine contracts before WP-05 and WP-06 implementation.
9. Create and exercise the Executable Deployment Contract before WP-10 release-candidate approval.
10. Do not schedule WP-04 through WP-10 against calendar dates until R1 evidence and team capacity are known.

## 12. Roadmap acceptance rule

The roadmap is current only when package status, accepted gates, governing specification versions, and explicit deferrals are maintained together. A feature is not considered delivered because code merged, a provider returned success, or a UI state appeared. Delivery means the owning package's semantic, security, failure, evidence, migration, accessibility, and live-environment gates have passed and the decision is recorded.
