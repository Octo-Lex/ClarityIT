# WP-01 G0 — Object Ownership and Prohibited Writes

**Gate:** WP01-G0 — Plan/Contract Freeze  
**Authority:** `WP01-AUTH-2026-08-12`  
**Baseline:** `main@33d3802d93c6d3123d9377566f0f3b6fb1360ecb`  
**Status:** Frozen G0 implementation map candidate

## 1. Purpose

This matrix assigns one authoritative owner to each WP-01 object family and defines prohibited write paths. It exists to prevent accidental dual-authority, browser/agent writes into kernel truth, compatibility fall-through, and provider-specific authority leakage.

The rule is simple: **one authoritative writer per object family at a time**. Derived views, caches, NATS messages, search indexes and UI state are not alternate writers.

## 2. Principal categories

WP-01 recognizes typed `PrincipalRef` categories:

- human;
- reasoning agent;
- service;
- policy service/principal;
- execution workload;
- external source.

Principal category alone does not grant permission. Workspace scope and explicit authorization remain mandatory.

## 3. Authoritative object ownership

| Object family | Authoritative owner | Authoritative store | Allowed authoritative mutation path | Prohibited writers / paths |
|---|---|---|---|---|
| Case | Domain kernel | PostgreSQL | Domain command/service with workspace + expected aggregate version | browser direct SQL, reasoning agent, NATS consumer without domain command, compatibility dual-write |
| Resource | Resource registry/domain service | PostgreSQL | Resource command/service | reasoning agent, browser, provider adapter, context builder |
| ProviderBinding | Resource registry/domain service | PostgreSQL | Binding command/service | provider adapter direct write, browser, reasoning agent |
| Observation | Observation service | PostgreSQL | Typed source-attributed observation ingestion | provider result claim, UI, reasoning synthesis, verifier output masquerading as observation |
| OperationPacket | Domain kernel | PostgreSQL | Draft/propose/successor commands | Effect Broker edits, adapter edits, reasoning direct write after proposal, UI direct write |
| PolicyDecision | Authority service | PostgreSQL | Deterministic policy evaluation | approval UI, reasoning agent, Effect Broker, adapter |
| ApprovalDecision | Authority service | PostgreSQL | Identified approver command bound to exact packet digest | reasoning agent, shared service identity, compatibility backfill, adapter |
| AuthorityGrant | Authority service | PostgreSQL | grant issue/reserve/release/consume/revoke/expire state machine | approval row alone, reasoning agent, browser, adapter, compatibility backfill |
| ExecutionAttempt | Effect Broker | PostgreSQL | sole broker command/preflight/dispatch orchestration | Control API direct provider path, reasoning worker, adapter creating independent attempt, legacy remediation path |
| Dispatch record | Effect Broker | PostgreSQL | atomic with grant reservation + attempt state | adapter-only log, browser, NATS-only event |
| ProviderReceipt | execution fixture/service in WP-01; live adapter later | PostgreSQL | typed receipt ingestion bound to attempt/provider op | reasoning agent, UI, verifier |
| ResultClaim | execution fixture/service in WP-01; live adapter later | PostgreSQL | typed claim bound to attempt and source | reasoning synthesis, UI, verifier |
| VerificationSpec | Domain/verification contract owner | PostgreSQL | versioned immutable spec publication/binding | executor, adapter, UI projection, reasoning agent |
| Verification | Verification service | PostgreSQL | independent verifier workflow against exact spec | executor, adapter, reasoning agent, UI, provider result claim |
| VerificationEvidence | Verification/evidence service | PostgreSQL + immutable artifact refs | sealed evidence append | mutable UI attachment, adapter overwriting prior evidence |
| OutcomeDecision | Domain/outcome service | PostgreSQL | identified accountable human decision after required Verification | provider, adapter, reasoning agent, approval service, UI optimistic state without command |
| EvidenceManifest | Evidence service | PostgreSQL + immutable artifact refs | append/seal immutable lineage manifest | reasoning agent, browser, adapter rewriting history |
| Inbox record | Messaging foundation | PostgreSQL | consumer dedupe transaction | NATS as authoritative dedupe source, UI |
| Outbox record | Authoritative transaction owner | PostgreSQL | same transaction as authoritative state + audit | asynchronous best-effort publisher creating truth |
| Audit record | authoritative command owner/audit service | PostgreSQL | same transaction as state when required | browser log, NATS-only record |
| ContextBundle manifest | Context Builder | PostgreSQL/derived evidence store as selected by G1/G4 | deterministic build/seal from authorized sources | browser-composed context, reasoning self-authorized bundle |
| Context overlay definition | owning policy/context source | PostgreSQL/config authority with explicit version | governed overlay management | retrieved/generated text pretending to be policy |
| Compatibility mapping | Compatibility layer | PostgreSQL | idempotent provenance-bound mapper | v1 writeback from v2 shadow read, browser |
| Writer-ownership registry | Compatibility/kernel migration authority | PostgreSQL/config authority | explicitly governed package transition | runtime guess, bidirectional dual-write |

## 4. Derived/non-authoritative surfaces

The following are rebuildable or transport-only and MUST NOT become a source of product truth:

- NATS subjects/messages;
- search indexes;
- timeline/read projections;
- browser state;
- WebSocket events;
- caches;
- context-selection caches;
- monitoring dashboards;
- logs/traces;
- fake execution fixtures;
- similarity/retrieval ranking.

A failure/loss of one of these may affect latency or visibility but must not alter authoritative state.

## 5. Service boundary rules

### Domain kernel

May own Cases, Resources where assigned, packets, successor relationships and outcome commands. It must not call providers or convert provider claims into Verification.

### Authority service

Owns PolicyDecision, ApprovalDecision and AuthorityGrant. It must keep those record types distinct. Approval does not directly execute.

### Effect Broker

Is the sole dispatch API. It may validate authority, reserve a grant, create/transition an attempt and record dispatch. In WP-01 it may only invoke deterministic fake/no-op routes. It must not mark Verification passed or Outcome accepted.

### Execution fixture

May return typed fake receipts/claims for kernel tests. It must have no real provider credentials, no network mutation capability and no authority to approve/verify/accept.

### Verification service

Owns Verification evaluation against exact immutable VerificationSpec and fresh allowed evidence. It must not use executor flags, agent conclusions, UI state or provider completion alone as proof.

### Evidence service

Seals lineage manifests and artifact references. It is append-oriented and may not rewrite prior authoritative history.

### Context Builder

Builds derived immutable Context Bundles. It cannot grant authority or convert context into current Observation/Verification.

### Compatibility layer

Maps/reads historical v1 semantics under the one-writer contract. It cannot manufacture AuthorityGrants, passed Verification or Accepted outcomes from legacy claims, and cannot fall through into unsafe legacy execution.

## 6. API/browser restrictions

The browser/UI may request commands through authenticated server APIs but MUST NOT:

- write kernel tables directly;
- supply authoritative actor/workload identity solely from client assertions;
- create grants or attempts by posting raw database-shaped records;
- call provider adapters directly;
- mark Verification/Accepted by presentation state;
- choose a workspace solely by client-supplied ID without server authorization.

## 7. Reasoning-agent restrictions

Reasoning agents are credentialless and non-authoritative. They may:

- read bounded governed context;
- produce findings/proposals/draft packet material through typed APIs;
- request permitted read tools.

They MUST NOT:

- approve their own packet where human/SoD policy requires otherwise;
- issue/reserve/consume AuthorityGrants;
- call a provider or credential broker;
- create an ExecutionAttempt outside the Effect Broker;
- complete Verification;
- accept an outcome;
- write authoritative Resource/Observation state directly.

## 8. Workspace enforcement

Every authoritative WP-01 row except explicitly global immutable definitions must bind a workspace directly or through a constraint-verifiable parent. Queries/commands fail closed on missing or ambiguous workspace.

Cross-workspace foreign keys or logical references are prohibited in WP-01 unless a later explicitly governed federation model is introduced.

## 9. Background processing

Every background job/consumer must carry explicit workspace and actor/workload provenance. It must re-authorize server-side rather than trust message/browser scope blindly.

Inbox dedupe and authoritative state transition occur in the same bounded transaction where required by the Kernel contract.

## 10. Direct-provider bypass denylist

During WP-01, the following code paths must have **zero reachable live provider mutation**:

- Control API handlers;
- reasoning/agent workers;
- compatibility/remediation paths;
- context builder;
- verifier;
- UI/browser;
- generic authenticated HTTP clients.

Only the Effect Broker fake/no-op route is permitted for synthetic execution semantics.

## 11. Test obligations

WP-01 implementation must provide negatives proving:

1. reasoning identity cannot issue a grant;
2. approval record alone cannot create an attempt;
3. browser cannot directly dispatch;
4. compatibility mapping cannot promote historical claims;
5. executor/provider claim cannot mark Verification;
6. verifier cannot mark Outcome accepted;
7. cross-workspace object/reference writes fail;
8. duplicate command cannot create a second logical attempt;
9. direct provider-bypass interfaces are absent or unreachable in WP-01;
10. derived caches/NATS/UI state can be discarded/rebuilt without changing truth.

## 12. Change control

Changing an authoritative owner, introducing a second authoritative writer, allowing a prohibited principal to create execution/verification/outcome truth, or bypassing the Effect Broker is a semantic contract change and cannot be made as a routine refactor.
