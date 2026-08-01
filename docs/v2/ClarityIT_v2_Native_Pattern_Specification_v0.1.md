# ClarityIT v2 Native Pattern Specification

**Canonical patterns for shared work, governed execution, reviewed knowledge, and controlled scale**

**Version:** 0.1  
**Status:** Draft normative specification  
**Date:** 1 August 2026  
**Applies to:** Experience plane, authoritative control plane, intelligence and processing, data and evidence plane, trust services, target runtimes, and operational-source integrations

> **Specification purpose:** Define the reusable patterns ClarityIT v2 will implement after the migration baseline, so that experience expansion, operational intelligence, private access, routines, projects, and multi-target work extend the same authority and evidence model rather than creating parallel execution semantics.

## Normative decision snapshot

- A Case is the shared governed work context for consequential work. Conversation coordinates activity but does not create authority or truth.
- People, agents, services, policies, and execution workloads are separately identifiable principals. Agents may investigate and propose but cannot approve, issue authority, hold target credentials, verify their own work, or accept an outcome.
- Operational records remain typed. Findings, Observations, Operation Packets, Policy Decisions, approvals, grants, attempts, receipts, Result Claims, Verifications, Outcome Decisions, and evidence manifests must not collapse into a generic event.
- Natural-language intent must compile into a provider-neutral, machine-checkable Operation Packet before any consequential effect is eligible for authorization.
- Resources, topology, observations, prior Cases, reviewed knowledge, and health contracts form bounded context. Retrieval output is derived context, not authoritative state.
- Provider access is implemented through typed capabilities and conforming adapters. Terminal commands, provider syntax, route type, and plugin names are never the approval subject.
- Reasoning is credentialless. Secrets are resolved only by trusted executors at the narrowest execution boundary.
- Provider completion remains a claim. Independent versioned verification and, for the first release, human acceptance are required to establish the outcome.
- Correction and compensation create successor lineages. No failed, rejected, unknown, or superseded record is rewritten to make history appear successful.
- Site-local execution is deterministic and outbound-connected. It contains no language model and cannot create or extend authority while disconnected.
- Knowledge and operational skills become reusable only through explicit review, versioning, scope, evidence, and retirement controls.
- Signals and schedules may open or advance work, but they cannot bypass policy, authority, execution, verification, or acceptance.
- Multi-target execution is deferred until the single-target verified pilot passes. Every target then retains its own authority, attempt, evidence, verification, and outcome.
- Workspace isolation applies to authoritative state, search, evidence, messages, object storage, caches, identities, and target routes.

## 1. Authority, scope, and terminology

### 1.1 Authority hierarchy

This specification is normative for the semantics of the patterns it defines. It does not weaken higher-order ClarityIT authorities.

| Artifact | Governs | Authority relationship |
|---|---|---|
| Product Definition v0.1 or successor | Product category, scope, first-release outcome, surfaces, and acceptance | Highest product authority |
| Authoritative Execution Kernel Specification v0.1 or successor | Execution objects, states, authority, claims, verification, outcomes, and evidence | Highest execution-semantics authority |
| v1-to-v2 Compatibility and Migration Specification v0.1 or successor | Source profiles, migration, coexistence, cutover, rollback, and historical truth | Highest migration authority |
| Layered System Architecture | Component placement, trust boundaries, routing, persistence, and read/write relationships | Architecture baseline |
| This specification | Reusable native patterns and their conformance requirements | Must conform to all authorities above |
| Delivery Roadmap | Sequence, ownership, gates, and release packaging | Planning authority; cannot change semantics |

If a pattern conflicts with the kernel, the kernel governs. If a roadmap item conflicts with this specification, the pattern specification governs until an approved revision changes it.

### 1.2 Scope

This document specifies:

- the canonical pattern catalog and pattern dependencies;
- required domain, API, UI, persistence, trust, evidence, and failure behavior;
- minimum data contracts where the kernel does not already define a stronger contract;
- conformance and acceptance criteria for each pattern;
- prohibited shortcuts and explicit non-goals;
- ownership by proposed roadmap work package.

This document does not specify detailed screen layouts, provider request payloads, model selection, a general workflow language, calendar dates, staffing estimates, pricing, or provider-specific implementation code.

### 1.3 Pattern conformance language

**MUST** and **MUST NOT** are mandatory. **SHOULD** and **SHOULD NOT** require a recorded architecture decision and risk rationale when deviated from. **MAY** is optional but must preserve all invariants.

Each pattern uses the following fields:

- **Intent:** the result the pattern exists to produce.
- **Use when:** conditions under which the pattern applies.
- **Do not use as:** boundaries that prevent category drift.
- **Participants:** components and principals that perform distinct roles.
- **Contract:** normative behavior, data, state, and trust requirements.
- **Failure and recovery:** behavior under stale, partial, duplicate, disconnected, or unknown conditions.
- **Evidence:** the record required to reconstruct what happened.
- **Conformance:** externally testable acceptance criteria.
- **Roadmap:** work package that introduces or matures the pattern.

## 2. Global invariants for every pattern

| ID | Invariant |
|---|---|
| GP-01 | PostgreSQL is authoritative for product state. NATS JetStream transports committed commands and events but never creates product truth. |
| GP-02 | Every authoritative transition commits with audit and transactional outbox data before publication. Every consumer deduplicates before applying state. |
| GP-03 | Every record and message carries workspace, actor or workload identity, correlation, causation, schema version, and time. |
| GP-04 | Conversation, retrieval, agent output, executor output, provider status, and operator annotations remain source-attributed claims unless a stronger typed record establishes authority. |
| GP-05 | No consequential provider call occurs outside the Effect Broker -> Execution Worker -> selected trusted route path. |
| GP-06 | Credentials are absent from prompts, browser payloads, packets, messages, logs, receipts, evidence exports, and search indexes. |
| GP-07 | Freshness is evaluated when state is used, not when it was originally stored. |
| GP-08 | Provider completion cannot directly produce Verification or Accepted. |
| GP-09 | Any material change to target, capability, parameters, baseline, policy revision, route, verifier, or risk creates a successor decision subject as required by kernel binding rules. |
| GP-10 | Unknown, inconclusive, partial, and blocked are legitimate states and cannot be coerced into succeeded or failed for presentation convenience. |
| GP-11 | Workspace isolation is enforced server-side and preserved in data, search, transport, storage, caches, and runtime routes. |
| GP-12 | Projections, timelines, summaries, search results, and dashboards are derived and rebuildable. |

## 3. Pattern catalog and dependency map

| ID | Native pattern | Primary outcome | First owning package |
|---|---|---|---|
| P-01 | Governed Case Workspace | One shared context from objective to accepted outcome | WP-03 |
| P-02 | Identifiable Human-Agent Participation | Every participant and artifact has attributable identity and bounded rights | WP-01 |
| P-03 | Typed Work Record and Provenance Timeline | Durable history without collapsing authority levels | WP-01 |
| P-04 | Evidence-Backed Operational Answer | Answers remain source-linked, freshness-aware, and explicitly uncertain | WP-03 |
| P-05 | Resource-Aware Bounded Context | Reasoning receives relevant operational context without unbounded access | WP-01 |
| P-06 | Intent-to-Immutable Operation Packet | Natural intent becomes a reviewable, machine-checkable effect subject | WP-01 |
| P-07 | Typed Capability and Adapter Boundary | Product semantics remain provider-neutral and testable | WP-02 |
| P-08 | Scoped Authority and Trusted Dispatch | Every consequential effect executes under exact, expiring authority | WP-01 |
| P-09 | Credentialless Reasoning and Executor-Held Secrets | AI can assist without receiving operational credentials | WP-01 |
| P-10 | Independent Verification and Human Acceptance | Outcomes are proven separately from execution claims | WP-01 |
| P-11 | Successor Correction and Compensation | Recovery preserves immutable history and receives fresh authority | WP-01 |
| P-12 | Deterministic Site Runtime and Private Access | Private systems are reached through a constrained local route | WP-04 |
| P-13 | Reviewed Operational Knowledge | Institutional knowledge is attributable, reviewed, versioned, and retireable | WP-05 |
| P-14 | Versioned Operational Skill and Playbook | Reusable procedures cannot bypass capability, authority, or evidence controls | WP-05 |
| P-15 | Signal-Triggered Routine and Exception Case | Schedules and events create accountable work without direct unsafe automation | WP-06 |
| P-16 | Project and Software-Delivery Context Binding | Projects and delivery objects organize work without becoming authority | WP-07 |
| P-17 | Controlled Multi-Target Execution | Bounded fan-out preserves per-target control and proof | WP-08 |
| P-18 | Workspace Isolation and Sovereign Deployment | Every layer remains tenant-safe and customer-controlled | WP-01 |

The dependency rule is strict: P-06 through P-11 must pass for one target before P-12, P-15, P-16, or P-17 may introduce a new mutation path. P-17 cannot begin before the verified single-target pilot is accepted.

## 4. P-01 - Governed Case Workspace

### Intent

Provide one coherent operational work context in which an accountable person and bounded agents can understand an objective, examine evidence, propose a change, obtain authority, observe execution, verify the result, and decide the outcome without relying on an external ticket or terminal to reconstruct the chain.

### Use when

Use a Case for controlled, consequential, or critical work; uncertain investigation; cross-role decisions; effects on real systems; evidence obligations; or work that may require correction. Assisted or collaborative work MAY remain a lighter Work Item until policy, risk, or exception promotes it to a Case.

Do not use a Case as a generic chat room, raw log stream, credential container, or substitute for the external managed system.

### Participants and contract

- **Accountable owner:** owns success criteria and outcome acceptance or rejection.
- **Participants:** humans and agents with explicit workspace membership and Case role.
- **Case service:** owns objective, participants, affected resources, success criteria, lifecycle projection, and lineage links.
- **Control API:** is the only write interface for Case commands and artifacts.
- **My Work:** projects assigned decisions, exceptions, agent-prepared work, execution uncertainty, verification failure, and acceptance backlog.
- **Case Workspace:** presents distinct sections for context, findings, proposal, authority, execution, verification, outcome, and evidence.

A Case MUST have one workspace, stable Case ID, objective, accountable owner, success criteria, participants, affected resource references, current projection state, aggregate version, and correlation ID. Consequential execution cannot begin until the exact target Resource and success criteria exist.

The Case timeline MUST be a projection over typed records. It MUST display authority class and source for each item. Discussion, a finding, a provider receipt, and a passed Verification cannot use the same semantic status.

The first-release lifecycle projection is:

`open -> investigating -> decision_pending -> authorized -> executing -> verifying -> outcome_pending -> accepted | correction_required | closed`

Projection state MUST be derived from underlying immutable records and MUST be rebuildable. Optimistic UI MAY be used for drafts and comments; it MUST NOT be used for approval, grant, attempt, verification, or outcome state.

### Failure, evidence, and conformance

Concurrent commands use expected aggregate version and return a conflict without overwriting later work. If a projection consumer fails, the authoritative records continue and the projection rebuilds. If a target becomes unresolved, the Case remains open but mutation controls fail closed.

The sealed Case evidence must identify objective, owner, participants, resources, baseline, findings, packet, policy, decisions, grant, attempts, claims, Verifications, outcome, successors, and manifest digest.

- **PC-CW-01:** A governed Case cannot be created without objective, owner, workspace, and success criteria.
- **PC-CW-02:** Every consequential lineage is reachable from one Case and every Case artifact is workspace-scoped.
- **PC-CW-03:** Submitted, provider completed, observed, Verified, and Accepted appear as distinct states.
- **PC-CW-04:** Rebuilding projections from authoritative records reproduces the same lifecycle and timeline order.
- **PC-CW-05:** A user can inspect raw receipts and evidence while the default view remains outcome-centered.
- **PC-CW-06:** The first-release happy path requires no terminal, provider login, manual database edit, or manual evidence assembly.

**Roadmap:** domain skeleton in WP-01; complete experience and live acceptance in WP-03.

## 5. P-02 - Identifiable Human-Agent Participation

### Intent

Allow humans and agents to work in the same Case while keeping identity, provenance, responsibility, and prohibited powers explicit.

### Contract

Every actor MUST use a typed `PrincipalRef` with principal ID, principal type, workspace, authentication or workload identity context, status, and display metadata. Principal types are human, reasoning agent, service, execution workload, policy, and external source. Shared identities without workload attribution do not satisfy this pattern.

A reasoning agent identity MUST additionally record agent definition/version, run ID, model and policy profile references, context-bundle digest, tool or capability declarations, creator or owner, and lifecycle status. Sensitive model-provider details MAY be stored operationally but do not replace the ClarityIT principal.

Agents MAY:

- read bounded Case and Resource context;
- perform explicitly permitted read-only investigation;
- create findings, hypotheses, drafts, summaries, and Operation Packet proposals;
- request review or authority through the Control API.

Agents MUST NOT:

- approve their own or another packet;
- issue, reserve, consume, revoke, or extend an Authority Grant;
- receive target credentials or secret values;
- call a provider mutation interface directly;
- write Provider Receipts, Verification, or Outcome Decisions;
- mark knowledge or a playbook approved;
- alter their own capability or workspace scope.

Human participation MUST distinguish proposer, approver, accountable owner, executor administrator, verifier administrator, and outcome accepter where policy requires separation of duties. A human UI session does not become an execution workload identity.

Agent-produced prose MUST retain run provenance even when edited by a human. Human adoption of a draft is a new authored revision or explicit endorsement, not silent provenance replacement.

### Failure, evidence, and conformance

Revoked or disabled principals cannot create new artifacts or commands. Existing records remain attributable. If workload attestation is unavailable, dispatch fails before provider submission. If an agent run loses its context or policy binding, its draft remains evidence but cannot continue under a new binding without a successor run.

- **PC-ID-01:** Every artifact and state transition resolves to one authenticated principal or workload identity.
- **PC-ID-02:** Agent API credentials cannot invoke approval, grant, dispatch, verification completion, or acceptance commands.
- **PC-ID-03:** Provenance survives human editing through revision lineage.
- **PC-ID-04:** Separation-of-duties policies reject conflicting principal roles.
- **PC-ID-05:** Secret scanning finds no operational credential in agent context or output.

**Roadmap:** WP-01, surfaced and usability-tested in WP-03.

## 6. P-03 - Typed Work Record and Provenance Timeline

### Intent

Create a durable, reconstructable record of work while preserving the different evidentiary and authoritative meanings of collaboration, decisions, execution claims, verification, and outcomes.

### Contract

The canonical record families are:

| Class | Examples | Authority meaning |
|---|---|---|
| Collaboration | comment, question, hypothesis, summary | Context only |
| Finding | structured finding, uncertainty, source references | Reasoned claim |
| Observation | source, observed time, typed state, fingerprint | Source-attributed state claim |
| Proposal | draft or immutable Operation Packet | Intent; no authority until separately granted |
| Authority | Policy Decision, Approval Decision, Authority Grant | Permission conditions and exact scope |
| Execution | Attempt, Provider Receipt, Result Claim | What was attempted and what route/provider reported |
| Verification | Verification Spec, evidence items, result | Independent evaluation of versioned postconditions |
| Outcome | acceptance, rejection, correction, compensation decision | Accountable disposition |
| Evidence | manifest, artifact reference, digest, signature | Integrity and reconstruction |

Each record MUST include immutable identity, workspace, schema version, actor, created or observed time, recorded time, correlation, causation, aggregate version where applicable, and successor or supersession linkage. Records MUST NOT be retyped after creation to promote their authority.

Timeline order uses authoritative recorded time and explicit causal links, not browser arrival order. Source time remains visible when different. Late observations are inserted into the causal view without rewriting audit order.

NATS messages MUST reference already committed records. A message replay may rebuild a projection but may not create a second authoritative transition. Search and summaries MUST preserve record class and source in their result schema.

### Failure, evidence, and conformance

Malformed or unsupported record versions are quarantined with reason codes. Duplicate message delivery returns the recorded result. An absent artifact is represented as missing evidence, not by deleting the manifest reference.

- **PC-TR-01:** A generic timeline event cannot substitute for a typed authority or verification record.
- **PC-TR-02:** Duplicate delivery produces one authoritative record and one aggregate version increment.
- **PC-TR-03:** A full timeline can be rebuilt from PostgreSQL without NATS history.
- **PC-TR-04:** Search results expose record class, source, workspace, and time.
- **PC-TR-05:** Corrections add successors and preserve the original bytes and digest.

**Roadmap:** WP-01; evidence-centered rendering in WP-03; extended record classes in later packages.

## 7. P-04 - Evidence-Backed Operational Answer

### Intent

Answer operational questions from bounded, source-attributed evidence while making freshness, uncertainty, disagreement, and missing evidence visible.

### Contract

An operational answer MUST be an artifact, not an authoritative state transition. Its minimum contract is:

| Field | Requirement |
|---|---|
| answer_id, case_id, workspace_id | Stable identity and scope |
| question | Exact user or system question |
| generated_by | Human or reasoning principal and run |
| context_bundle_digest | Exact bounded inputs used |
| statements[] | Individually addressable conclusions |
| source_refs[] per statement | Observations, evidence, knowledge, external references, or prior Cases |
| source_time and freshness | Observed/published time and freshness decision |
| confidence and uncertainty | Calibrated rationale, not an unsupported number |
| alternatives | Material competing explanations when present |
| missing_evidence | What could change the answer |
| recorded_at and revision | Audit and successor lineage |

The answer generator MUST prefer current Resource Observations and approved knowledge over unsourced summaries. It MUST distinguish real-time state, historical evidence, reviewed knowledge, and prior reasoning. Retrieval rank is not truth rank.

If sources disagree, the answer MUST preserve both claims, identify their source times, and state the resolution rule or unresolved status. Stale evidence may be discussed but cannot satisfy a fresh precondition or Verification input.

Answers MAY propose read-only next checks or draft an Operation Packet. They MUST NOT execute a mutation, issue authority, or mark the question resolved solely because prose was generated.

### Failure, evidence, and conformance

If no adequate source exists, the result is `insufficient_evidence`, not a fabricated answer. Retrieval or source failure records the missing source and may remain partially useful. Restricted sources are omitted with a visible access-limited indication; access control is not weakened to improve completeness.

- **PC-EA-01:** Every material answer statement has at least one inspectable source reference or is explicitly labeled inference.
- **PC-EA-02:** Stale and current observations are visually and semantically distinct.
- **PC-EA-03:** Conflicting sources cannot be silently collapsed.
- **PC-EA-04:** Answer generation cannot change Resource, Case authority, execution, verification, or outcome state.
- **PC-EA-05:** A reviewer can reproduce the bounded source set from the context-bundle digest and references.

**Roadmap:** basic Case answers in WP-03; mature reviewed retrieval in WP-05.

## 8. P-05 - Resource-Aware Bounded Context

### Intent

Provide reasoning workers only the operational context needed for the current objective, resource, and policy scope, with explicit provenance and deterministic limits.

### Contract

A Context Bundle is derived and immutable for one reasoning turn or run. It MUST include bundle ID and digest, workspace, Case, objective, target Resource IDs, resource and binding versions, selected fieldsets, Observation references and freshness, topology edges, ownership, health contracts, relevant prior Cases, approved knowledge versions, allowed read capabilities, exclusions, size limits, and builder version.

The context builder MUST:

1. start from the Case objective and exact Resource scope;
2. resolve stable resource identities before labels;
3. apply server-side workspace and record-level access control;
4. select source-attributed fields rather than unbounded raw dumps;
5. rank by relevance, freshness, authority class, and policy;
6. remove secret values and prohibited data classes;
7. record omitted or truncated categories;
8. seal the selected references and digest before dispatch to a reasoning worker.

Topology expansion MUST have explicit depth, relationship allowlist, and target-count limits. A context bundle cannot expand from one target to an entire environment through an unbounded relation. Prior Cases are context; their accepted outcomes do not prove current state.

Context cache entries MUST be workspace-keyed, access-scope-keyed, versioned, and invalidated by relevant Resource, Observation, permission, or knowledge-version changes. Cache content is non-authoritative.

### Subordinate contract: deterministic context overlays

Every Context Bundle MUST be composed from ordered overlays using a versioned composition algorithm. The overlay order is:

1. organization policy, approved capability definitions, and organization-approved knowledge;
2. workspace policy, workspace configuration, and workspace-scoped published knowledge;
3. Case, Resource, binding, topology, Observation, and evidence context;
4. role- and task-specific read scope; and
5. personal drafts, preferences, and non-authoritative working memory.

Each overlay entry MUST carry workspace, source identity, authority class, version or observation time, sensitivity, screening state, and immutable content or reference digest. The composition record MUST identify all applied overlays, rejected entries, omissions, conflicts, and the builder version used to produce the final bundle.

Policy composition is monotonic. A later or narrower overlay MAY remove data, reduce scope, add a deny, require stronger approval, shorten freshness, restrict egress, or impose a tighter resource limit. It MUST NOT relax an organization or workspace deny, broaden an allowed capability, weaken a sensitivity rule, replace an approved Resource identity, or turn contextual text into policy or authority.

Authoritative namespaces are reserved. Personal drafts, prior Case prose, retrieved documents, model output, and other non-authoritative content MUST NOT shadow or replace policy, capability definitions, Resource IDs, binding versions, published knowledge versions, or current Observations. A name or key collision is rejected or quarantined and recorded; precedence alone cannot silently resolve it.

External and generated content MUST be source-labelled and assigned `screened`, `quarantined`, or `unscreened` handling state before inclusion. Screening may restrict or exclude content, but a screening result cannot grant authority, change a source's authority class, or satisfy a required Observation. Personal overlays remain writable only as drafts and are excluded from packet canonicalization unless an explicit field is validated through the normal proposal contract.

### Failure, evidence, and conformance

If a critical source is unavailable or freshness cannot be determined, the bundle records the gap. A reasoning worker may continue only if policy permits degraded investigation; it cannot produce an executable packet that depends on missing preconditions.

- **PC-BC-01:** Cross-workspace records never enter a bundle, search result, cache key, or model request.
- **PC-BC-02:** Rebuilding with the same input versions and builder version produces the same selected-reference digest.
- **PC-BC-03:** Topology expansion obeys depth and target-count limits.
- **PC-BC-04:** Secret references may appear; secret values do not.
- **PC-BC-05:** Missing, stale, omitted, and access-restricted context is distinguishable.
- **PC-BC-06:** Overlay permutation, policy-relaxation, and deny-removal attempts fail; the same ordered versions produce the same composition digest.
- **PC-BC-07:** Personal, retrieved, prior-Case, and generated content cannot shadow authoritative namespaces or change authority class.
- **PC-BC-08:** Quarantined content is excluded from packet inputs and unscreened content is visibly constrained by policy.
- **PC-BC-09:** The evidence record reconstructs applied overlays, rejected collisions, omissions, screening states, and the final bundle digest.

**Roadmap:** context contract in WP-01; Case implementation in WP-03; reviewed knowledge expansion in WP-05.

## 9. P-06 - Intent-to-Immutable Operation Packet

### Intent

Translate a natural-language objective into a provider-neutral, reviewable, machine-checkable proposal before any consequential action can be authorized.

### Contract

The user may express the objective in natural language. A reasoning worker MAY investigate and draft a proposal, but the Control API must validate and canonicalize the proposal into the kernel Operation Packet contract.

The transformation stages are:

1. **Objective capture:** record intended outcome, owner, target scope, and success criteria.
2. **Read-only investigation:** collect bounded Observations and evidence.
3. **Capability resolution:** choose one declared provider-neutral capability supported by the exact Resource binding.
4. **Packet draft:** populate rationale, baseline, parameters, predicted effects, preconditions, postconditions, stop conditions, risk, authority requirement, verifier, and compensation candidate.
5. **Prepare preview:** adapter validates translation and exposes a redacted provider preview without mutation.
6. **Proposal:** canonicalize, sign, digest, and freeze the packet.

Free-form shell commands, SQL, provider URLs, or opaque tool names MUST NOT be the approved effect. If a later capability legitimately contains a script or query, its schema, interpreter, target, allowed operations, content digest, safety class, and verifier must be explicit and separately governed.

Any material edit after proposal creates a successor packet and invalidates bound approval or grant as required by the kernel. The UI MUST show both the generic capability meaning and the redacted provider translation before an approver decides.

### Failure, evidence, and conformance

Unsupported capability, unresolved target, stale baseline, adapter prepare failure, missing verifier, or ambiguous parameter blocks proposal or authority. A draft may remain editable, but it has no execution meaning.

- **PC-IP-01:** Natural-language text alone cannot reach the Effect Broker.
- **PC-IP-02:** Proposal produces a deterministic digest and becomes immutable.
- **PC-IP-03:** Provider-specific syntax is absent from the core capability and approval subject.
- **PC-IP-04:** Changing a bound field produces a successor and invalidates prior decisions.
- **PC-IP-05:** Adapter Prepare performs no mutation and exposes required permissions and translation.

**Roadmap:** WP-01 contract; WP-02 first capability; WP-03 approval and preview experience.

## 10. P-07 - Typed Capability and Adapter Boundary

### Intent

Keep ClarityIT product behavior stable across providers by separating provider-neutral capability semantics from provider translation and deployment route.

### Contract

The capability registry MUST define capability name and version, resource type and version, parameter schema, required Observations, preconditions, direct effects, expected result fields, verifier requirements, idempotency semantics, cancellation semantics, risk inputs, and compensation candidates.

Adapters MUST implement the kernel methods `DescribeCapabilities`, `Observe`, `Prepare`, `Submit`, `Poll`, `Cancel`, `ObserveResult`, and `Reconcile` as applicable. `Verify` is never an adapter method. Unsupported critical fields are rejected rather than ignored.

Every adapter release MUST declare adapter ID, version, build digest, supported capability versions, provider class, route compatibility, permission requirements, provider idempotency behavior, raw status mapping, reconciliation behavior, and limitations.

An extension or plugin MAY register an adapter, observer, verifier, connector, or read-only context source through a signed manifest. It MUST NOT:

- create or reinterpret Authority Grants;
- call the kernel database directly;
- expose credentials to reasoning workers or browsers;
- publish an unpersisted receipt as truth;
- invent a new success state;
- bypass conformance tests or workspace isolation;
- broaden targets or parameters beyond the packet.

Provider-prefixed compatibility APIs may exist during migration but MUST translate to generic v2 objects and commands. Route changes are deployment changes and require authority re-evaluation; they do not create new product capabilities.

### Failure, evidence, and conformance

Adapters return precise rejection, pre-send failure, accepted, running, terminal, unknown, and reconciliation states with raw provider codes preserved and secrets redacted. An ambiguous submission cannot be automatically resubmitted.

- **PC-AB-01:** The generic capability contract passes without provider-specific core fields.
- **PC-AB-02:** Contract tests cover observe, prepare, submit, poll, cancel, result observation, failure, duplicate delivery, and reconciliation.
- **PC-AB-03:** Adapter version and configuration digest appear in every receipt.
- **PC-AB-04:** A second conforming adapter can implement the same capability without changing packet or Case semantics.
- **PC-AB-05:** Extension permissions cannot invoke authority or database write paths outside controlled APIs.

**Roadmap:** first conformance in WP-02; extension SDK and second provider in WP-09.

## 11. P-08 - Scoped Authority and Trusted Dispatch

### Intent

Ensure that every consequential effect is explicitly permitted for the exact packet, resource, capability, route, workload identity, policy revision, time, and use count.

### Contract

Policy Decision, Approval Decision, Authority Grant, and execution preflight answer separate questions and remain separate records. Approval does not itself dispatch, provide a credential, or become a grant.

The Effect Broker is the sole dispatch entry point. It MUST atomically or serializably:

1. resolve exact packet, resource binding, adapter, route, and verifier;
2. verify packet digest and signature;
3. evaluate policy and required approval references;
4. validate grant scope, subject, route, time, nonce, state, and use limit;
5. obtain a fresh Observation and evaluate preconditions and stop conditions;
6. reserve the grant and create one idempotent Execution Attempt;
7. commit attempt, dispatch record, audit, and outbox before transport.

The execution route MUST independently revalidate packet signature, grant, route identity, expiry, nonce, target binding, and local policy. Client state and reasoning output can never weaken these checks.

Changing route after approval requires a new grant and policy evaluation. Changing target binding, provider translation, risk, or packet content requires a successor packet when the approved subject changes.

### Failure, evidence, and conformance

Denied policy creates no grant. Expired, revoked, consumed, mismatched, or replayed grants block before provider submission. Duplicate commands return the existing attempt. A failure after submission may have occurred consumes the grant and enters `outcome_unknown` until reconciliation.

- **PC-AD-01:** Direct provider calls from the Control API, browser, or reasoning worker are technically unreachable.
- **PC-AD-02:** Scope, route, subject, expiry, nonce, and use-limit mismatch each block before submission.
- **PC-AD-03:** Concurrent duplicate commands create one attempt and at most one logical provider submission.
- **PC-AD-04:** Route validation repeats critical authority checks.
- **PC-AD-05:** Audit reconstructs why dispatch was allowed at that moment.

**Roadmap:** WP-01 kernel foundation; WP-02 live compute proof.

## 12. P-09 - Credentialless Reasoning and Executor-Held Secrets

### Intent

Allow agents to investigate and propose operations without receiving reusable or target-operational credentials.

### Contract

Reasoning workers receive capability descriptions and secret references only. Secret values are resolved at the narrowest trusted executor: central connector, Site Runtime, verifier, or native enforcement integration.

Secret references MUST identify workspace, secret purpose, target or provider scope, allowed workload identities, validity or rotation metadata, and resolving trust boundary. They MUST NOT reveal the underlying value or enable browser-side resolution.

Operational credentials SHOULD be short-lived, least-privilege, target-scoped, capability-scoped where supported, and obtained just in time. Long-lived provider credentials require a recorded exception and compensating controls.

The Secrets service MUST enforce workload identity, route, workspace, and purpose. It MUST emit a sanitized access audit without secret value. Secret values MUST be excluded from packets, prompts, client payloads, events, receipts, logs, error text, evidence exports, object metadata, and search indexing.

Verifiers use independent secret references where practical so that executor compromise does not automatically compromise proof. Secret rotation cannot mutate sealed historical packets; historical records retain key identifiers and redaction metadata, not values.

### Subordinate contract: destination-bound credential broker

Provider credentials used for HTTPS integrations MUST be injected by a credential broker inside the trusted executor or typed adapter. The broker MUST NOT expose a general authenticated HTTP tool to a reasoning worker, browser, client, playbook, or routine.

A broker entitlement MUST bind:

- entitlement ID and digest, workspace, workload identity, route, adapter, target, and capability version;
- Operation Packet, Authority Grant, attempt, validity interval, use limit, and revocation state when the call is consequential;
- HTTPS scheme, exact destination host and port, TLS requirements, allowed HTTP methods, normalized path prefixes, and redirect policy;
- allowed request headers, headers the broker may inject, response metadata that may be returned, and request/response size limits; and
- secret reference, credential purpose, provider account scope, and sanitization profile.

The broker MUST validate the entitlement after URL parsing and path normalization, reject user-info and ambiguous host forms, prevent redirect or DNS handling from escaping the approved destination policy, and inject credential material only after all checks pass. Headers not on the allowlist are removed or rejected. Secret-bearing headers and raw response bodies MUST NOT enter prompts, routine inputs, generic logs, or evidence exports.

Every brokered call emits a sanitized audit record containing entitlement digest, workload, packet and attempt references where applicable, destination class, method, normalized path class, timing, outcome, provider correlation identifiers, and redaction profile. It MUST contain neither the secret nor a replayable authenticated request.

### Failure, evidence, and conformance

Secret resolution failure, expired credential, identity mismatch, or excessive permission fails the operation at a precise stage. It cannot fall back to a broader credential. Logs preserve provider error classification without including values.

- **PC-CS-01:** Reasoning and browser processes cannot call secret-value resolution interfaces.
- **PC-CS-02:** Secret access requires matching workspace, workload, route, and purpose.
- **PC-CS-03:** Automated scanning finds no secret value in all prohibited surfaces.
- **PC-CS-04:** Credential permissions cannot perform effects outside the packet's pilot boundary.
- **PC-CS-05:** Rotation and revocation are testable without changing historical evidence.
- **PC-CS-06:** Host, port, method, normalized path, header, redirect, workload, route, packet, and validity mismatches each fail before credential injection.
- **PC-CS-07:** Reasoning, browser, playbook, and routine paths cannot invoke a generic authenticated transport or retrieve broker-injected headers.
- **PC-CS-08:** Broker audit records reconstruct why a call was permitted without containing credentials or replayable request material.
- **PC-CS-09:** Central and Site Runtime broker implementations pass the same entitlement and sanitization conformance fixtures.

**Roadmap:** trust foundation in WP-01; connector enforcement in WP-02; local resolution in WP-04.

## 13. P-10 - Independent Verification and Human Acceptance

### Intent

Prove the intended outcome through fresh, versioned, read-only checks that are independent from the executor's success report, then obtain the accountable outcome decision.

### Contract

Every consequential packet MUST reference an immutable Verification Spec version. The spec defines required sources, predicates, freshness, aggregation, thresholds, timeouts, secret references, result classification, and evidence sealing.

The Verification Worker MUST be separate from the execution adapter interface. It MAY use the same external platform for a provider-state postcondition, but consequential workload health SHOULD use a different observation path. Executor-written status, provider task success, agent conclusions, and UI state are prohibited Verification inputs.

Verification results are `passed`, `failed`, or `inconclusive`. Missing, stale, unreachable, timed-out, or unauthorized inputs are inconclusive unless the exact spec defines them as failure. Retried probes remain separate evidence items.

For the first release, `Accepted` requires passed Verification and an explicit accountable human decision. Acceptance records the principal, rationale, time, exact Verification, and lineage. Verification does not automatically close the Case.

### Failure, evidence, and conformance

If provider completion succeeds but result state or health fails, execution completion remains visible and the outcome cannot be Accepted. If the verifier is unavailable, the system remains outcome-pending or requires correction under policy; it cannot infer pass from elapsed time.

- **PC-IV-01:** Provider completion cannot directly create Verification.
- **PC-IV-02:** The verifier uses the exact spec version and fresh recorded inputs.
- **PC-IV-03:** Pass, fail, and inconclusive paths preserve individual evidence and reason codes.
- **PC-IV-04:** Threshold or source changes create a new spec version and successor packet when bound.
- **PC-IV-05:** First-release acceptance requires an identified accountable human.

**Roadmap:** contracts in WP-01; compute verifier in WP-02; outcome experience and pilot in WP-03.

## 14. P-11 - Successor Correction and Compensation

### Intent

Correct failed, partial, rejected, stale, or harmful outcomes without erasing prior attempts or treating a rollback mechanism as proof of recovery.

### Contract

Correction changes the plan for achieving the objective. Compensation proposes an explicit effect intended to counter a prior effect. Both are successor Operation Packets with new digest, current baseline, policy evaluation, authority, attempt, Verification, and outcome.

Every successor MUST reference predecessor Case, packet, attempt, outcome, reason, and relationship type: `corrects`, `retries_after_safe_failure`, `reconciles_unknown`, or `compensates`. A blind retry is prohibited after ambiguous submission.

A compensation candidate in an earlier packet is advisory. It MUST NOT carry standing authority. Automatic compensation MAY be introduced only under a future policy profile that pre-authorizes the exact successor capability, trigger, scope, freshness, and verifier; it is excluded from the first release.

The UI MUST preserve both original and successor lineages, explain why the successor exists, and avoid a single mutable success label. Aggregate Case status is derived from the active lineage and does not alter historical outcomes.

### Failure, evidence, and conformance

Compensation failure creates its own failed or unknown outcome and escalation. It cannot change the original result. Reconciliation may resolve an unknown provider submission but cannot fabricate a missing receipt.

- **PC-SC-01:** Editing a failed or accepted historical packet or attempt is rejected.
- **PC-SC-02:** Correction and compensation each require a new packet and authority cycle.
- **PC-SC-03:** Ambiguous submission blocks automatic resubmission and invokes reconciliation.
- **PC-SC-04:** Evidence manifests link every predecessor and successor without deleting prior artifacts.
- **PC-SC-05:** Compensation failure remains separately visible and reviewable.

**Roadmap:** lineage model in WP-01; first correction experience in WP-03; automated profiles remain deferred.

## 15. P-12 - Deterministic Site Runtime and Private Access

### Intent

Reach private network zones and clusters without placing an AI agent on every target or exposing target credentials to the central reasoning plane.

### Contract

One signed Site Runtime is deployed per approved network zone, data center, or cluster when direct central connectivity is insufficient. It initiates outbound mTLS, authenticates to the Remote-Site Edge Gateway, advertises signed route and adapter inventory, and executes only typed authorized commands.

The Site Runtime contains Local Control, encrypted append-only journal, typed adapters, credential resolver, observation spool, health reporting, and upgrade manager. It contains no language model, prompt, autonomous planner, grant issuer, or arbitrary remote shell endpoint.

Every command MUST bind packet, grant, workspace, route, runtime identity, target binding, adapter version, nonce, expiry, and local sequence. Local Control revalidates signature, authority, policy bundle, stop conditions, clock health, and target scope before execution.

Private SSH, bastion, database, or platform access MAY be added only as a typed adapter with an explicit capability schema. A desktop terminal session is not the execution model. Host sensors are optional and permitted only for required host-local observation or enforcement.

During disconnection:

- permitted observations and protective deny rules may continue and spool;
- already submitted operations may continue polling and journaling;
- new consequential mutations, grants, approvals, or expiry extensions are denied by default;
- pre-authorized containment is allowed only under signed emergency policy.

### Subordinate contract: runtime capability profile and migration safety

Each Site Runtime and central connector substrate MUST publish a signed, versioned Runtime Capability Profile bound to its workload identity and route. The profile MUST declare:

- supported typed-effect, observation, verification, adapter, and protocol versions;
- durable writable state, encrypted journal, receipt spool, observation spool, backup/restore, and blob-staging support;
- cryptographic identity, mTLS, local policy enforcement, secret-resolution, attestation, and revocation support;
- egress-enforcement grade, network reachability class, operating-system and tool profile, clock-health requirements, and resource budgets;
- process-session support only where an approved typed adapter requires it; and
- upgrade, rollback, compatibility, retention, and maximum-offline guarantees.

Every route and Operation Packet evaluation MUST compare its minimum runtime requirements with the active profile digest before dispatch. Absence, staleness, incompatibility, or a weaker enforcement grade fails closed. A runtime MAY advertise capabilities it does not currently route, but an unadvertised or disabled capability cannot be inferred from installed software.

A route or runtime migration MUST produce a signed Migration Gap Report comparing source, destination, and required profiles. The report identifies preserved, weakened, unavailable, and behaviorally different capabilities; affected in-flight work; journal and evidence transfer; rollback boundary; and required operator decisions. Migration is refused when any required authority, isolation, durability, secret, egress, verification, evidence, or recovery capability would be weakened. Waivers cannot override kernel invariants or convert an unsupported destination into a conformant route.

In-flight attempts MUST remain bound to the runtime and journal that accepted them unless a separately specified handoff protocol proves single ownership, sequence continuity, and evidence continuity. Migration does not erase uncertainty or authorize resubmission.

### Failure, evidence, and conformance

Unhealthy clock, invalid identity, incompatible runtime version, queue exhaustion, policy mismatch, or disconnected pre-dispatch state blocks mutation. Local receipts are signed, sequenced, deduplicated at ingress, and committed before progress publication.

- **PC-SR-01:** Runtime inspection proves no model runtime or autonomous selection path.
- **PC-SR-02:** Route identity mismatch and expired authority block locally.
- **PC-SR-03:** Disconnect-before-dispatch denies mutation; disconnect-after-submit preserves polling and central uncertainty.
- **PC-SR-04:** Encrypted journal survives restart and replays receipts exactly once centrally.
- **PC-SR-05:** Signed upgrade, rollback, compatibility handshake, and compromised-runtime revocation pass.
- **PC-SR-06:** Dispatch fails when the profile is absent, stale, unsigned, identity-mismatched, or weaker than the packet and route requirements.
- **PC-SR-07:** Profile conformance covers persistence, journal, spool, identity, policy, secrets, egress, backup, staging, clock, resources, and supported protocol versions.
- **PC-SR-08:** Migration produces a signed gap report and refuses every required-capability downgrade without changing historical route evidence.
- **PC-SR-09:** In-flight migration, restart, and rollback tests prove single ownership, sequence continuity, and no duplicate provider submission.
- **PC-SR-10:** Capability loss and disabled features remain visible to operators and cannot be hidden by a compatibility fallback.

**Roadmap:** WP-04, only after the central single-target pilot passes.

## 16. P-13 - Reviewed Operational Knowledge

### Intent

Convert evidence-backed experience into reusable institutional knowledge without allowing unreviewed Case content or generated summaries to become operational authority.

### Contract

Knowledge items use the lifecycle:

`candidate -> review_pending -> approved -> published -> superseded | retired`

A candidate may originate from a Case, document, observation set, operator submission, or curated external source. It MUST include source references, author or generator, workspace, subject Resources or types, validity scope, review owner, created time, and proposed expiry or review date.

Published knowledge MUST have stable item ID, immutable version, title, content, source manifest, applicability, exclusions, owner, reviewer, approval time, sensitivity, access scope, effective time, review-by time, supersession lineage, and content digest.

Retrieval indexes are derived. Search results MUST enforce workspace and access scope before ranking, return exact item versions and source references, and distinguish published knowledge from candidates and prior Case prose. An approved knowledge item informs reasoning; it does not satisfy a fresh Resource Observation or grant authority.

Material content change, applicability change, source change, or review-policy change creates a new version. Retirement removes an item from default retrieval but preserves history and references from prior Cases.

### Failure, evidence, and conformance

Expired or review-overdue knowledge is visibly stale and excluded from high-confidence operational guidance by policy. Missing source artifacts block publication or mark the item incomplete. Conflicting approved items are surfaced for resolution rather than silently ranked away.

- **PC-KN-01:** Unreviewed Case output is never returned as approved knowledge.
- **PC-KN-02:** Every published version has source manifest, owner, reviewer, scope, and digest.
- **PC-KN-03:** Search enforces workspace and sensitivity before retrieval.
- **PC-KN-04:** Retirement and supersession preserve references from historical Cases.
- **PC-KN-05:** Approved knowledge cannot substitute for current Observation, Verification, or authority.

**Roadmap:** WP-05; basic attached context remains available in WP-03.

## 17. P-14 - Versioned Operational Skill and Playbook

### Intent

Make repeatable operational procedures reusable while ensuring that each consequential step still becomes a typed packet with current authority and evidence.

### Contract

An operational skill or playbook is a reviewed plan template, not executable authority. Its minimum manifest is:

| Field | Requirement |
|---|---|
| skill_id, name, version, digest | Stable versioned identity |
| purpose and expected outcome | Human-readable intent |
| owners and reviewers | Accountable lifecycle roles |
| supported resource types | Exact normalized types |
| steps[] | Read, decide, propose, execute, verify, or notify classification |
| capability refs | Exact provider-neutral capability versions |
| input schema | Typed, validated, secret-free values |
| preconditions and stop conditions | Machine-checkable controls |
| authority profile | Risk and approval requirements per step |
| evidence requirements | Required observations, receipts, and manifests |
| verifier refs | Exact postcondition specifications |
| correction and compensation guidance | Successor candidates only |
| applicability and exclusions | Environments, providers, limits, and forbidden uses |
| source Cases and review record | Provenance |
| effective and review dates | Lifecycle control |

Instantiation MUST resolve exact Resources, current Observations, capability support, and policy. Each consequential step produces its own Operation Packet or an explicitly modeled compound packet supported by the kernel. A playbook cannot reuse historical approval or grant.

Automatic generation from Case history may create a candidate only. Publication requires human review, conformance tests, and a recorded owner. Scripts or commands embedded in a future skill must use a separately governed typed capability and content digest.

### Subordinate contract: signed scope-owned skill package

Every skill MUST be distributed as a signed package owned by exactly one workspace or organization scope. Its lifecycle is:

`draft -> review_pending -> reviewed -> published -> superseded | archived`

Only a published version may be instantiated for governed work. Promotion from a workspace to organization scope creates a new scoped version, requires organization review, and produces a new signature and digest; ownership or approval is never inherited implicitly. Archived and superseded packages remain addressable for historical reconstruction but are excluded from new default selection.

The signed manifest MUST additionally declare package scope, signing identity and algorithm, signature, manifest digest, content and dependency digests, required capabilities, approved capability ceiling, materializer version range, file allowlist, execution-content classification, and promotion lineage. Required capabilities MUST be a subset of the approved ceiling. Neither field is an Authority Grant, and publication creates no standing execution authority.

Materialization MUST be deterministic and isolated. The materialization record binds the published package digest, exact dependency digests, selected files, normalized manifest, materializer version, and resulting tree digest. Instantiation separately binds validated inputs, exact Resources, current policy, capability versions, and generated packet references. Network retrieval, floating dependencies, mutable branch references, undeclared files, and scope-precedence shadowing are prohibited during materialization.

Signature, ownership, lifecycle, capability, dependency, and digest checks occur before a skill enters reasoning context or creates a packet draft. A package can provide instructions and templates, but executable content remains subject to a separately governed typed capability and cannot gain a secret or generic command interface through packaging.

### Failure, evidence, and conformance

Unsupported version, stale applicability, missing capability, failed precondition, or changed resource binding blocks instantiation or the affected step. Partial completion leaves per-step evidence and requires a successor decision.

- **PC-SK-01:** Executing a playbook creates no provider call without a valid packet and grant for each effect.
- **PC-SK-02:** Input validation rejects undeclared fields and secret values.
- **PC-SK-03:** Draft, review-pending, reviewed, published, superseded, and archived versions are distinct and transition only through authorized lifecycle decisions.
- **PC-SK-04:** Conformance fixtures cover happy, stale, unsupported, partial, and compensation-required paths.
- **PC-SK-05:** A reviewer can trace every skill version to source evidence and approval.
- **PC-SK-06:** Package signature, scope ownership, promotion lineage, capability ceiling, dependency digests, and file allowlist are verified before use.
- **PC-SK-07:** Repeated materialization from identical signed inputs produces the same tree digest and performs no undeclared network access.
- **PC-SK-08:** A workspace package cannot shadow an organization package or become organization-scoped without a new reviewed and signed version.
- **PC-SK-09:** Package publication and instantiation create no Authority Grant, secret access, or provider call.

**Roadmap:** WP-05; general authoring marketplace remains excluded.

## 18. P-15 - Signal-Triggered Routine and Exception Case

### Intent

Turn schedules, alerts, webhooks, and operational events into deduplicated accountable work while reserving consequential effects for the normal kernel path.

### Contract

A Signal is a source-attributed observation candidate with signal ID, workspace, source, type, occurred and received times, subject hints, payload digest, severity, deduplication key, correlation key, schema version, and artifact references. Signal normalization validates source and maps exact Resources where possible; ambiguous subjects remain unresolved.

A Routine is a versioned procedure definition with trigger, schedule or event filter, owner, target-resolution rule, read-only steps, exception conditions, Case template, allowed playbook references, concurrency policy, quiet window, deduplication window, and review date.

The trigger path is:

`source -> normalize Signal -> deduplicate/correlate -> instantiate Routine -> observe -> confirm normal completion or open/advance Case`

Signals and routines MAY create Work Items, observations, notifications, knowledge-review candidates, or packet drafts. They MUST NOT directly issue grants, call providers, mark Verification passed, or accept outcomes.

If a routine proposes a consequential effect, the exact target set is resolved, a Case is created or selected, and the normal packet, policy, approval, grant, execution, verification, and outcome sequence applies.

### Subordinate contract: routine actor, fire key, and destination consent

Every Routine version MUST execute as a dedicated Routine Principal, never as its human owner or a reasoning identity. The principal is bound to workspace, routine ID and version, allowed read capabilities, permitted Case and Work Item operations, notification destinations, and expiry or review date. It holds no provider credential, approval role, or grant-issuing permission.

Every trigger evaluation MUST derive a deterministic Fire Key from routine ID and version, trigger identity and version, scheduled occurrence or source-event identity, resolved destination binding, and exact target-resolution result. The Fire Key is reserved atomically in a durable firing ledger before downstream work begins. Retries resume the same logical fire and cannot create a second Case, notification, packet draft, or playbook instance for that fire.

Notification and person-directed destinations MUST be versioned bindings with verified identity, workspace, purpose, permitted message class, and either recipient consent or an approved organizational policy basis. Applicable suppression, opt-out, quiet-window, and destination-revocation state is checked at delivery time. A Routine owner name, channel label, or address in generated text is not a destination binding.

Delivery attempts, acknowledgements, and failures are persisted separately. Transport acceptance is a delivery claim, not proof that a person received, read, or acted on the notification. Destination changes create a successor Routine version and do not redirect already reserved fires silently.

### Failure, evidence, and conformance

Duplicate or storming signals collapse into one correlation group without losing source counts. Source outage creates a visible routine-health exception. Missed schedules are handled by an explicit catch-up policy; the system does not silently run an accumulated set of consequential actions.

- **PC-RT-01:** Duplicate signals produce one routine instance within the configured window.
- **PC-RT-02:** Ambiguous target resolution cannot create a consequential packet.
- **PC-RT-03:** Routine execution has no direct provider credential or mutation interface.
- **PC-RT-04:** Signal storms respect concurrency, rate, and Case-correlation budgets.
- **PC-RT-05:** Routine, Signal, Case, and any later packets remain fully traceable.
- **PC-RT-06:** Routine work is attributable to a dedicated least-privilege Routine Principal and never inherits the owner's ambient permissions.
- **PC-RT-07:** Duplicate delivery, worker restart, lease loss, catch-up, and replay produce one logical fire for each Fire Key.
- **PC-RT-08:** Unverified, revoked, suppressed, non-consenting, cross-workspace, or purpose-mismatched destinations block delivery.
- **PC-RT-09:** Destination changes create a successor version; reserved fires retain their original destination evidence.
- **PC-RT-10:** Transport acknowledgement cannot mark a Case outcome, Verification, or human response as complete.

**Roadmap:** WP-06, after reviewed knowledge and playbook foundations.

## 19. P-16 - Project and Software-Delivery Context Binding

### Intent

Organize longer-lived objectives, milestones, repositories, branches, changes, reviews, releases, and linked Cases without turning project metadata or source-control state into execution authority.

### Contract

A Project is a coordination aggregate with workspace, objective, owners, participants, milestones, dependencies, linked Work Items and Cases, resource scope, delivery-context bindings, status, and version. Delivery-context bindings may reference external repositories, branches, commits, pull requests, builds, deployments, environments, and releases by stable external identity and observed revision.

External delivery systems remain authoritative for repository and pipeline state. ClarityIT records source-attributed observations and links. A branch, pull request approval, issue status, or pipeline result does not issue a ClarityIT Authority Grant or prove a managed-system outcome.

Project views MAY provide discussion, agent findings, review requests, evidence, and release coordination. Consequential delivery actions require future typed capabilities and the same packet and verification model. A Project may group many Cases, but each governed effect remains owned by a Case and exact Resource scope.

Repository content included in reasoning context MUST be bound to repository identity, commit digest, path scope, access policy, and context-bundle digest. Uncommitted or mutable working-copy content is explicitly labeled.

### Failure, evidence, and conformance

Deleted branches, rebases, force pushes, renamed repositories, or missing permissions produce updated observations and unresolved bindings; historical references retain the original commit identity. Project status cannot hide a failed or unknown Case lineage.

- **PC-PJ-01:** Every delivery-context reference includes provider, external identity, and observed revision.
- **PC-PJ-02:** Repository or pipeline events cannot directly create execution authority or accepted outcome.
- **PC-PJ-03:** A governed effect remains linked to one Case even when surfaced in a Project.
- **PC-PJ-04:** Rebased or deleted references preserve historical commit evidence.
- **PC-PJ-05:** Context selection enforces repository path, workspace, and revision scope.

**Roadmap:** WP-07; Git or deployment mutation capabilities remain separately gated.

## 20. P-17 - Controlled Multi-Target Execution

### Intent

Execute a bounded operation across multiple resources without losing per-target authority, idempotency, progress, failure isolation, verification, or evidence.

### Entry gate

This pattern MUST NOT be implemented until the single-target `compute.virtual_machine.start` pilot has passed all product and kernel acceptance criteria. It begins with one capability, one provider profile, one environment class, and a small policy-defined target limit.

### Contract

A target set MUST resolve to an immutable snapshot of exact Resource IDs and binding versions before approval. Dynamic labels such as `all`, `production`, or a mutable query cannot be the execution target. The packet or batch plan records selection query for explanation and resolved targets for authority.

The batch coordinator MUST define maximum targets, concurrency, ordering, failure threshold, stop policy, rate limits, maintenance window, per-zone budgets, and aggregate success rule. It creates or references per-target packet/grant/attempt lineages unless a future compound-packet kernel contract proves equivalent isolation.

Each target retains:

- baseline Observation and freshness decision;
- target-bound authority and idempotency;
- independent submit and provider-operation tracking;
- result Observation and Verification;
- correction or compensation lineage;
- evidence manifest or manifest section with independently verifiable digest.

Aggregate progress is a derived projection. `80% succeeded` cannot convert failed, unknown, or inconclusive targets into success. Stop-on-threshold prevents new submissions at a safe boundary; it does not claim cancellation of submitted operations.

### Failure, evidence, and conformance

Partial failure preserves successful targets and separately handles failed or unknown targets. Worker restart resumes from persisted per-target checkpoints without resubmission. A target whose baseline changed is blocked without blocking unrelated targets unless policy defines atomic group semantics.

- **PC-MT-01:** Mutable target selectors resolve to an immutable approved target snapshot.
- **PC-MT-02:** Target count, concurrency, rate, zone, and failure budgets are enforced server-side.
- **PC-MT-03:** Duplicate or restarted batch processing produces at most one logical submission per target.
- **PC-MT-04:** Aggregate status never overwrites per-target truth.
- **PC-MT-05:** Partial, threshold-stop, unknown, and successor-recovery scenarios pass live conformance.

**Roadmap:** WP-08, after WP-03 acceptance and preferably after WP-04 route conformance.

## 21. P-18 - Workspace Isolation and Sovereign Deployment

### Intent

Keep each organization's product state, evidence, knowledge, messages, runtime routes, and secrets isolated while supporting customer-controlled deployment and provider-neutral operation.

### Contract

Workspace scope MUST be enforced in PostgreSQL queries and constraints, API authorization, NATS subject or envelope handling, object-storage paths and access policies, search indexes, vector retrieval, caches, WebSocket subscriptions, background jobs, secret references, adapter configuration, Site Runtime routes, evidence exports, and operational telemetry.

Every aggregate and message includes workspace ID. Cross-workspace relationships are prohibited unless a future explicit federation contract defines identity, consent, filtering, and evidence. Administrative support access is time-bound, approved, attributed, and auditable.

Deployment artifacts MUST be signed, versioned, reproducible, and accompanied by SBOM, provenance, configuration schema, migration compatibility, and verification guidance. Core product semantics cannot depend on one SaaS, cloud, identity provider, message provider, object store, model provider, or infrastructure provider.

Customer-controlled deployments MUST preserve the same authority, evidence, and conformance requirements as centrally hosted deployments. Offline or restricted deployment does not permit local agents or administrators to manufacture Verification or authority.

Data export and retention MUST preserve workspace policy, redaction, manifest digests, and lifecycle records. Deletion and legal hold are explicit records, not silent disappearance.

### Subordinate contract: executable deployment directory

Every releasable deployment MUST be represented by a version-controlled Deployment Contract Directory that binds the exact product release, deployment tool version, configuration schema, migration range, policy and extension descriptors, runtime profiles, secret-reference routes, and immutable artifact or image digests. Environment-specific values MAY be supplied through validated overlays, but the resolved configuration digest and source of each value MUST be recorded without exposing secrets.

The directory MUST provide deterministic operations with these semantics:

- `check`: offline validation of schema, references, signatures, digests, versions, compatibility, and required clauses;
- `doctor`: external read-only checks of identity, connectivity, permissions, storage, database, queues, object storage, clocks, and target environment prerequisites;
- `plan`: a rendered, reviewable change plan identifying creates, updates, migrations, restarts, removals, risks, evidence steps, and rollback boundaries;
- `up`: controlled application of the approved deployment plan with step identity, idempotency, checkpoints, and persisted results; it grants no managed-system operational authority; and
- `verify-drift`: comparison of the declared manifest with live deployment identity, versions, digests, configuration classes, runtime profiles, and policy state.

Before a mutating deployment, the system MUST seal an immutable deployment manifest, approved plan digest, database migration position, backup or snapshot reference where required, rollback manifest, forward-recovery instructions, and responsible decision record. Rollback MUST preserve kernel history and evidence; database rollback is permitted only when the migration contract declares it safe. Otherwise the response is a forward correction under an approved successor plan.

Every contract clause MUST be registered as `ENFORCED`, `VALIDATED-ONLY`, or `RESERVED`. `ENFORCED` means the running system prevents or detects violation at the governing boundary. `VALIDATED-ONLY` means a deterministic preflight check exists but runtime enforcement is not yet proven. `RESERVED` means the clause is named but unavailable and cannot be claimed as implemented. Release evidence MUST not treat `VALIDATED-ONLY` or `RESERVED` as `ENFORCED`.

Deployment commands and live systems MUST emit source-attributed evidence sufficient to reconstruct requested plan, approved digest, applied steps, resulting manifest, drift status, database position, and rollback or recovery decision. Deployment success is not Verification of managed operational outcomes.

### Failure, evidence, and conformance

A missing workspace scope or ambiguous route is a security failure and fails closed. Cache, index, or transport partition mismatch triggers quarantine and incident evidence. Backup and restore tests must prove workspace separation and preserve sealed evidence.

- **PC-WS-01:** Cross-workspace API, search, vector, WebSocket, cache, object, message, and secret tests all fail closed.
- **PC-WS-02:** Every Site Runtime and connector route is bound to approved workspace and resource scopes.
- **PC-WS-03:** Fresh installation, upgrade, backup, restore, and export reproduce required integrity and isolation controls.
- **PC-WS-04:** Release artifacts include signatures, SBOM, provenance, and version manifest.
- **PC-WS-05:** Provider substitution does not change product capability, authority, or evidence semantics.
- **PC-WS-06:** `check`, `doctor`, `plan`, `up`, and `verify-drift` have deterministic exit classification and preserve their evidence records.
- **PC-WS-07:** The applied deployment matches immutable artifact, configuration, migration, policy, extension, and Runtime Capability Profile digests or reports explicit drift.
- **PC-WS-08:** Rollback and forward-recovery exercises preserve authoritative history, evidence, workspace isolation, and database compatibility.
- **PC-WS-09:** Every deployment clause has one truthful status, and release checks reject unsupported `ENFORCED` claims.
- **PC-WS-10:** Deployment tooling cannot issue operational Authority Grants, call managed-system capabilities, or manufacture Verification.

**Roadmap:** foundation in WP-01; exercised by every package; production hardening in WP-10.

## 22. Cross-pattern conformance matrix

| Concern | Governing patterns | Required release evidence |
|---|---|---|
| Shared work and experience | P-01, P-02, P-03 | Case lifecycle tests, role tests, projection rebuild, accessibility evidence |
| Operational intelligence | P-04, P-05, P-13, P-14 | Source attribution, ordered overlay digest, anti-shadowing, review lifecycle, signed skill package and materialization fixtures |
| Governed effects | P-06, P-07, P-08, P-09 | Packet digest, adapter contract, grant scope, destination-bound credential broker, secret scan |
| Proven outcomes | P-10, P-11 | Verifier conformance, outcome decisions, successor lineage, evidence manifests |
| Private and local execution | P-12 | Runtime identity and capability profile, migration-gap refusal, offline journal, disconnection, upgrade, local-policy tests |
| Evented and recurring work | P-15 | Signal normalization, Fire Key dedupe, Routine Principal, destination consent, routine health, exception Case evidence |
| Longer-lived delivery context | P-16 | Project/Case bindings, revision provenance, external-state truth tests |
| Controlled scale | P-17 | Resolved target snapshots, budgets, per-target attempts and proof |
| Trust and deployment | P-18 plus all | Isolation matrix, executable deployment contract, clause-status truth, signed artifacts, drift, rollback, restore and export proof |

## 23. Prohibited shortcuts

- No terminal-first or unrestricted shell-agent execution model.
- No credential-bearing reasoning worker, prompt, browser session, or general agent tool.
- No direct provider mutation from a Case comment, answer, routine, Project event, playbook, plugin, or UI action.
- No generic `succeeded` field shared by proposal, provider completion, verification, and outcome.
- No provider receipt, process exit code, signed message, schedule completion, pipeline status, or agent confidence treated as Verification.
- No dynamic multi-target selector evaluated after approval.
- No automatic publication of knowledge or playbooks from operational history.
- No hidden compensation callback or destructive rollback that rewrites the original record.
- No new SSH, Kubernetes, database, public-cloud, browser, desktop, Git, or multi-target mutation before the verified VM-start release gate.
- No cross-workspace retrieval, transport, caching, runtime, or evidence shortcut.

## 24. Required follow-on detailed specifications

| Specification | Required before | Minimum content |
|---|---|---|
| First-Release Experience Specification | WP-03 build completion | Screen states, roles, command language, evidence inspection, intervention, accessibility |
| Context Overlay Contract | WP-01 completion | Overlay order, authority classes, monotonic tightening, anti-shadowing, screening states, deterministic digest |
| Executor Credential Broker Contract | WP-02 live connector | Workload and packet binding, HTTPS destination rules, header injection, redirects, audit sanitization, conformance fixtures |
| Site Runtime Protocol | WP-04 implementation | Handshake, envelopes, sequence/ack, queue, attestation, upgrade, disconnection |
| Runtime Capability Profile and Migration Contract | WP-04 route migration | Profile schema, minimum requirements, enforcement grades, gap report, refusal and in-flight ownership |
| Knowledge Governance Specification | WP-05 implementation | Lifecycle, review, sensitivity, indexing, retention, conflicts, retirement |
| Operational Skill Package Specification | WP-05 implementation | Signed manifest, scope ownership, lifecycle and promotion, deterministic materialization, inputs, steps, evidence, capability binding |
| Signal and Routine Specification | WP-06 implementation | Source trust, normalization, Fire Key, Routine Principal, destination consent, dedupe, correlation, scheduling, delivery and exception semantics |
| Project and Delivery Context Specification | WP-07 implementation | Project schema, external bindings, revision provenance, Case relationships |
| Multi-Target Coordination Specification | WP-08 implementation | Target snapshot, budgets, concurrency, partial failure, aggregate projection |
| Extension SDK and Conformance Specification | WP-09 implementation | Signed manifests, APIs, sandboxing, versioning, isolation, compatibility tests |
| Executable Deployment Contract | WP-10 release candidate | Version binding, configuration schema, check/doctor/plan/up, immutable manifests, clause status, drift, rollback and recovery evidence |

## 25. Release rule

No pattern is accepted because its UI exists or its happy path runs. Acceptance requires its normative conformance criteria, cross-pattern invariants, security tests, failure scenarios, evidence reconstruction, and owning roadmap gate to pass under blocking CI and, where the pattern touches real systems, a controlled non-production environment.

The first implementation priority remains unchanged: prove one provider-neutral `compute.virtual_machine.start` lineage from Case objective through human acceptance before widening target types, providers, routes, routines, projects, or target counts.
