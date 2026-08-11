# ClarityIT v2 — Context Overlay Contract

**Version:** 0.1  
**Package owner:** WP-01 — Authoritative Kernel Foundation  
**Status:** WP01-G0 contract candidate; becomes frozen WP-01 input when the G0 integration is accepted  
**Authority:** `WP01-AUTH-2026-08-12`  
**Package baseline:** `main@33d3802d93c6d3123d9377566f0f3b6fb1360ecb`  

## 1. Purpose

This contract defines how ClarityIT composes bounded reasoning context without allowing contextual material to become policy, authority, execution truth, verification truth, or accepted outcome truth.

A Context Bundle is a derived, immutable, workspace-scoped input to one reasoning turn or bounded reasoning run. It is not an authoritative business record and cannot by itself authorize execution, satisfy a required live Observation, issue an AuthorityGrant, establish Verification, or create an OutcomeDecision.

The contract implements the WP-01-owned portion of Native Pattern P-05 and is subordinate to the Product Definition, Authoritative Execution Kernel, compatibility/migration authority, workspace isolation requirements, and the integrated WP-01 package plan.

## 2. Non-negotiable invariants

1. PostgreSQL-owned authoritative records remain the source of truth; a Context Bundle is a derived projection.
2. Context selection never grants capability or execution authority.
3. Every selected item retains source identity, authority class, workspace, version/time, sensitivity and screening state.
4. Overlay order is deterministic: **organization -> workspace -> Case/Resource -> role/task -> personal draft**.
5. A narrower overlay may tighten an inherited rule but must never relax it.
6. Personal, retrieved, prior-Case and generated text must never shadow reserved authoritative namespaces.
7. Secret values are forbidden. Only opaque secret references may appear where policy explicitly permits them.
8. Missing, stale, omitted, access-restricted and quarantined inputs remain distinguishable; absence is never silently converted to a positive fact.
9. Identical effective inputs plus identical builder version must reproduce the same selection/composition digest.
10. Every material rejection, omission, truncation, collision and policy conflict is evidenced.
11. Workspace scope is mandatory and server-enforced before selection.
12. Context may support investigation under degraded information, but a critical-source gap cannot satisfy an executable precondition.
13. Same-layer precedence is a bound semantic input governed by an identified/versioned precedence policy; no unrecorded local ordering may influence composition.

## 3. Terminology

### 3.1 Context Bundle

An immutable manifest of the exact references and selected fieldsets supplied to one reasoning turn/run, plus the builder inputs, policy decisions, omissions and final digest.

### 3.2 Overlay

A versioned set of contextual instructions, preferences, constraints or presentation guidance applied at one authority layer. An overlay is not automatically authoritative merely because it is earlier in the order; every entry carries an explicit authority class.

### 3.3 Authority class

The semantic class assigned to an overlay/context entry. WP-01 uses at least:

- `authoritative_policy` — organization/workspace policy material owned by an authoritative policy source;
- `authoritative_resource` — canonical Resource/Binding identity/version material;
- `current_observation` — source-attributed Observation material with freshness metadata;
- `approved_knowledge` — exact approved knowledge version when a later package provides one; WP-01 only reserves the class/namespace;
- `case_context` — current Case facts and lineage references;
- `task_context` — bounded role/task guidance that may tighten but not relax higher authority;
- `historical_context` — prior Case or legacy material, explicitly weaker than current verified truth;
- `retrieved_context` — retrieved untrusted/context-only material;
- `personal_draft` — user draft/preferences, never authoritative;
- `generated_context` — model-generated synthesis, never authoritative.

### 3.4 Screening state

One of:

- `screened` — passed the configured content/sensitivity screening process;
- `quarantined` — withheld from normal reasoning use because of a screening/policy problem;
- `unscreened` — known not to have passed screening and handled according to policy.

Screening state is not authority and does not increase truth strength.

## 4. Context Bundle schema contract

Every Context Bundle SHALL bind at least:

- `context_bundle_id`;
- `schema_version`;
- `workspace_id`;
- `case_id`;
- exact objective/purpose identifier or immutable objective digest;
- exact target Resource IDs;
- Resource aggregate versions;
- ProviderBinding IDs/versions where contextually required;
- selected Resource fieldsets rather than unrestricted object dumps;
- selected Observation IDs, source identities, observed/received times and freshness classification;
- bounded topology references;
- ownership/support responsibility references;
- health-contract references where relevant;
- prior Case references when selected;
- approved knowledge-version references when available in later packages;
- allowed read-capability identifiers;
- explicit exclusions;
- topology/resource/byte/token/count limits used by the builder;
- builder version;
- access-scope/policy version inputs;
- same-layer precedence policy identity and version;
- ordered overlay-entry references including each entry's bound declared precedence;
- applied/rejected/omitted/conflicting entry records;
- final canonical composition digest;
- creation time and builder workload PrincipalRef.

The bundle stores references and selected material required for reproducibility. It must not copy secrets or unrestricted sensitive source payloads merely to make the bundle self-contained.

## 5. Deterministic builder sequence

For each build, the Context Builder SHALL execute in this order:

1. bind exact workspace, Case, objective and target Resource scope;
2. resolve stable canonical IDs before names/free text;
3. enforce server-side workspace membership, ACL and data-scope policy;
4. load exact authoritative versions/references required by the objective;
5. select bounded attributed fieldsets rather than full records by default;
6. rank eligible context using deterministic policy over relevance, freshness, authority and explicit task need;
7. expand topology only within the configured relation allowlist, depth and target-count limits;
8. remove secret values and prohibited/sensitivity-disallowed material;
9. classify screening state and quarantine where required;
10. bind the identified/versioned same-layer precedence policy and each eligible entry's declared precedence;
11. apply overlays in the order defined in section 6;
12. reject authority-class collisions and prohibited shadowing;
13. record omissions, truncation, stale inputs, critical gaps and conflicts;
14. serialize the canonical composition manifest;
15. compute and seal the final digest before reasoning begins.

No browser or reasoning agent may bypass the builder and claim an equivalent authoritative Context Bundle.

## 6. Overlay order and semantics

Overlay order is fixed:

1. **Organization**
2. **Workspace**
3. **Case/Resource**
4. **Role/Task**
5. **Personal Draft**

Each overlay entry SHALL bind:

- entry ID;
- workspace ID;
- source PrincipalRef/source system identity;
- layer;
- authority class;
- source object/reference;
- version and/or effective time;
- sensitivity classification;
- screening state;
- immutable content/reference digest;
- intended scope;
- explicit constraints contributed by the entry;
- declared same-layer precedence value/class;
- governing precedence-policy identity and version.

The precedence value and governing policy/version are semantic composition inputs and must be included in evidence/digest inputs. Two builders may not use different unrecorded precedence rules for the same bound entries.

Later layers may specialize presentation or add narrower restrictions. They may not transform contextual instructions into policy authority.

## 7. Monotonic tightening

Composition MUST fail closed when a later layer attempts to:

- remove or weaken an inherited deny;
- broaden an allowed capability/read scope;
- widen workspace/resource scope;
- extend a validity or freshness boundary beyond an authoritative limit;
- weaken sensitivity handling;
- replace a canonical Resource identity or binding version;
- convert an omitted/stale/missing source into a satisfied precondition;
- promote contextual/historical/retrieved/generated material to authoritative policy or current Observation;
- reduce a required approval, verification or human-outcome requirement;
- introduce a generic provider/credential capability.

A later layer MAY:

- add a deny;
- narrow a capability/read scope;
- reduce a time/resource/topology limit;
- request stricter sensitivity handling;
- add a user preference that does not conflict with authority;
- add task-specific context within the already-authorized scope.

The composition record must identify the earlier constraint, conflicting later entry and deterministic rejection reason.

## 8. Reserved authoritative namespaces and anti-shadowing

**Every canonical authoritative Kernel/Compatibility record namespace is reserved**, not only a selected subset. Reserved namespaces/classes may only be populated from their authoritative owner and include at least:

- Case identity, aggregate version, lifecycle/projection source references and canonical Case fields;
- Resource identity/version and ProviderBinding identity/version;
- Observation identity, source, fieldset and freshness metadata;
- OperationPacket identity, canonical bytes/digest/signature metadata, state and successor lineage;
- policy and policy revision identifiers;
- capability definitions and parameter schemas;
- PolicyDecision, ApprovalDecision and AuthorityGrant identities/states/bindings;
- ExecutionAttempt identity/state/idempotency key, dispatch record and provider-operation reference;
- ProviderReceipt and ResultClaim identities/source/bindings;
- VerificationSpec, Verification and VerificationEvidence identities/results/bindings;
- OutcomeDecision identity/state/accountable principal;
- EvidenceManifest identity/digest/artifact references;
- authoritative ownership/support mappings;
- inbox/outbox/audit authoritative message/state records;
- compatibility mapping and one-writer ownership records;
- approved knowledge-version IDs once WP-05 introduces them;
- migration/schema authority and revision identities.

This list names the WP-01 canonical families but does not narrow the rule: if a record/field is authoritative under the Kernel, Compatibility contract, integrated WP-01 ownership matrix, or a later approved authority, its namespace is reserved automatically.

Personal drafts, retrieved text, prior Cases, generated synthesis and unapproved knowledge MUST NOT define or override a reserved authoritative key.

On collision, the builder SHALL either:

- reject the conflicting entry; or
- quarantine it where policy requires retention for review.

It MUST NOT silently choose the lower-authority value. Collision evidence includes both references/digests, authority classes and the resolution reason without leaking prohibited content.

## 9. Observation and freshness handling

Observation context SHALL retain:

- source identity;
- Resource/binding scope;
- observed time;
- received time;
- external revision/fingerprint where available;
- selected fieldset;
- freshness rule/version;
- current freshness classification.

A stale Observation may be shown as stale context when policy permits, but cannot be treated as fresh evidence. A required current Observation that is missing/stale/quarantined remains an explicit gap.

Context screening does not convert an Observation into Verification. Verification remains governed by the exact VerificationSpec and independent verifier path.

## 10. Prior Cases and historical material

Prior Cases and v1 historical rows are context only unless a current authoritative record explicitly references them for a stronger purpose.

The builder SHALL preserve the historical truth class. Legacy provider/agent/operator success claims must not be rendered as passed Verification or Accepted outcome. Ambiguous historical provider outcomes remain ambiguous.

Similarity or retrieval rank does not change authority class.

## 11. Topology expansion

Topology expansion requires an explicit:

- relation allowlist;
- maximum depth;
- maximum unique Resource count;
- maximum edges/records;
- deterministic traversal/order rule.

The builder must record which relations were followed and which candidates were omitted by limits or access policy.

Cross-workspace topology expansion is prohibited unless an explicit higher authority later defines a lawful cross-workspace relationship and corresponding isolation contract; WP-01 assumes no such path.

## 12. Personal overlays

Personal overlay content is draft/contextual only.

It may contain preferences such as display/detail level or task notes. It must never:

- issue policy;
- grant authority;
- alter Resource identity;
- redefine capability semantics;
- weaken approval/verification requirements;
- insert provider credentials;
- become part of Operation Packet canonicalization merely because it was in context.

A personal value may enter a future packet only through an explicit typed, validated mapping into a packet field under the packet/capability schema. The packet then owns the resulting value independently of the personal overlay.

## 13. Secret handling

Secret **values** are prohibited from:

- Context Bundles;
- overlay bodies;
- reasoning prompts;
- packet/context composition evidence;
- cache/search fixtures;
- logs and traces.

Opaque secret references may appear only where policy permits and must carry workspace/scope metadata without revealing the value. WP-01 does not resolve or inject provider credentials.

If a source field is classified secret, the builder records a sanitized omission reason rather than the value or a reversible transformation of it.

## 14. Caching and invalidation

A Context Bundle or intermediate selection cache is derived and non-authoritative.

Every cache key must include at least:

- workspace;
- access-scope/policy identity/version;
- builder version;
- same-layer precedence-policy identity/version;
- relevant Resource/Binding aggregate versions;
- relevant Observation/version/freshness inputs;
- approved knowledge versions when applicable;
- topology-limit configuration where it affects selection.

Invalidation is required when any bound identity/version/access/precedence rule changes. A cache miss/failure may reduce performance; it must not change authority semantics.

Caches are never used as the sole source of an authoritative version assertion if the authoritative record can have advanced.

## 15. Critical-source gaps and degraded investigation

Each builder policy identifies critical source classes for the objective.

When a critical source is unavailable, stale beyond policy, quarantined or access-restricted:

- the bundle records the gap and reason;
- the reasoning path may continue only in a policy-approved degraded investigation mode;
- the gap cannot be represented as a satisfied executable precondition;
- the Effect Broker/preflight remains responsible for fresh execution-time checks.

Reasoning output must distinguish what is known, stale, missing and inferred.

## 16. Canonical composition and digest

Digest computation uses a versioned canonical JSON representation. The canonical form SHALL:

- use explicit schema/builder versions;
- include the same-layer precedence-policy identity/version and every entry's declared precedence;
- sort object/map keys deterministically;
- order set-like reference collections by stable ID/digest;
- preserve the fixed overlay-layer order;
- use deterministic ordering within each layer by the **bound declared precedence** governed by the **bound precedence policy/version**, then stable entry ID;
- normalize times to UTC RFC3339 with documented precision;
- exclude non-semantic runtime data such as trace/span IDs;
- include every semantic applied/rejected/omitted/conflict record required to reproduce the composition decision.

The digest is SHA-256 over the exact canonical bytes with a versioned domain separator, for example:

`clarityit-context-overlay-v0.1\0 || canonical_json`

Changing canonicalization, precedence semantics or semantic fields requires a new builder/schema version; it must not silently reinterpret existing digests.

## 17. Composition evidence

For every build the evidence record SHALL contain, directly or by immutable reference:

- bundle ID/digest;
- builder version;
- workspace/Case/objective scope;
- selected authoritative reference IDs/versions;
- overlay entries and layer/authority metadata;
- each entry's declared same-layer precedence;
- governing precedence-policy identity/version;
- applied entries;
- rejected/quarantined entries;
- omitted/truncated inputs;
- stale/missing/access-restricted critical sources;
- topology traversal/limits;
- policy/access version;
- collision/conflict reason codes;
- final composition digest;
- builder workload identity and build time.

Evidence must be sufficient for an independent reviewer to rerun composition from the same immutable inputs and reproduce the digest.

## 18. Workspace isolation

The Context Builder MUST fail closed on missing or ambiguous workspace scope.

All source queries, caches, topology expansion, search/retrieval and object references are workspace-bound server-side. User-supplied workspace/resource IDs are treated as requests, not proof of access.

Negative tests must cover:

- cross-workspace Resource ID injection;
- cross-workspace prior-Case reference;
- cross-workspace cache-key collision;
- cross-workspace search/retrieval result;
- cross-workspace Observation reference;
- object-storage/evidence reference from another workspace;
- background build without explicit workspace attribution.

## 19. Reason codes

WP-01 Context Builder reasons use stable typed codes under at least:

- `CONTEXT_SCOPE_*`
- `CONTEXT_ACCESS_*`
- `CONTEXT_LIMIT_*`
- `CONTEXT_STALE_*`
- `CONTEXT_MISSING_*`
- `CONTEXT_SCREENING_*`
- `CONTEXT_SHADOW_*`
- `CONTEXT_POLICY_RELAXATION_*`
- `CONTEXT_SECRET_*`
- `CONTEXT_DIGEST_*`

Reason strings are diagnostic; automated behavior keys on the typed code/schema version.

## 20. Required WP-01 conformance scenarios

WP-01 MUST prove at minimum:

1. identical inputs + builder version reproduce the same digest;
2. permutation of raw input arrival order does not change canonical output;
3. overlay-layer permutation is rejected or canonicalized to the fixed order without changing authority;
4. identical entries with identical bound precedence policy/version reproduce the same same-layer order/digest, and changing precedence policy/version changes the bound composition input;
5. workspace/Case scope broadening fails;
6. deny removal/policy relaxation fails;
7. personal/retrieved/generated collision with **any canonical authoritative Kernel/Compatibility namespace** is rejected/quarantined;
8. stale Observation remains stale and cannot satisfy a fresh requirement;
9. critical-source gap remains explicit;
10. topology depth/relation/count limits are enforced and evidenced;
11. cache key includes workspace/access/version/precedence-policy inputs and invalidates on change;
12. secret values are absent from bundle/evidence/log fixtures;
13. cross-workspace source/reference/cache/search cases fail closed;
14. personal-draft data is excluded from packet canonicalization absent explicit typed mapping;
15. screening state does not alter authority class;
16. bundle reconstruction reproduces the final digest.

These scenarios satisfy the WP-01-owned P-05/PC-BC criteria and AC-01-33 through AC-01-37 where applicable.

## 21. Explicit non-goals

This contract does not authorize or define:

- live provider mutation;
- provider credential resolution/injection;
- Site Runtime/offline context behavior;
- WP-05 reviewed knowledge lifecycle;
- generic web/search access outside governed read capabilities;
- Operation Packet authority or Verification semantics beyond preserving their boundaries;
- cross-workspace federation.

## 22. Change control

Once WP01-G0 integrates this contract, implementation may refine internal APIs and storage shapes without changing these semantic requirements.

A change to overlay order, same-layer precedence semantics/policy binding, monotonic-tightening rules, reserved-authority behavior, secret handling, workspace isolation, digest semantics, or the rule that context cannot create authority/truth is a contract change and requires a governed successor under WP-01 change control.
