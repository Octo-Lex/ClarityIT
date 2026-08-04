# ClarityIT v2 — Signals and Routines

**Diagram version:** 0.2
**Status:** Proposed target-state companion under Native Pattern P-15; not an approved implementation contract
**Governing constraints:** Product Definition v0.1 and Authoritative Execution Kernel v0.1
**Rendered asset:** [`images/signals-and-routines.png`](images/signals-and-routines.png)

This view separates user work intake, external Signal ingestion, and routine
firing. No alert, webhook, schedule, routine, reasoning output, or transport
acknowledgement may bypass Case accountability or the normal kernel path for a
consequential effect.

```mermaid
flowchart TB
    subgraph INPUTS["Input classes"]
        direction LR
        HUMAN["HUMAN REQUEST<br/>Authenticated user intent"]
        ALERT["EXTERNAL SIGNAL<br/>Alert · webhook · operational event"]
        CLOCK["SCHEDULE<br/>Versioned occurrence"]
    end

    subgraph SIGNALS["Signal ingestion"]
        direction LR
        INGRESS["SIGNAL INGRESS<br/>Authenticate source · schema validate · rate limit"]
        NORMALIZE["NORMALIZE + DEDUPLICATE<br/>Source identity · event identity · observed time"]
        OBS["OBSERVATION<br/>Source-attributed claim<br/>never kernel truth by implication"]
        CORRELATE["CASE CORRELATION<br/>Attach to existing Case or open exception Case"]
        ALERT --> INGRESS --> NORMALIZE --> OBS --> CORRELATE
    end

    subgraph ROUTINES["Routine firing — proposed P-15"]
        direction LR
        TRIGGER["TRIGGER EVALUATION<br/>Schedule or normalized Signal"]
        PRINCIPAL["ROUTINE PRINCIPAL<br/>Dedicated least privilege<br/>never inherits owner permissions"]
        FIRE["FIRE KEY LEDGER<br/>Atomic reserve in PostgreSQL<br/>one logical fire across retry/replay"]
        RESOLVE["TARGET + DESTINATION RESOLUTION<br/>Versioned binding · consent/policy basis<br/>suppression · opt-out · quiet window"]
        WORK["ROUTINE WORK<br/>Create/select Case · Work Item · notification attempt<br/>no provider credential or grant authority"]
        CLOCK --> TRIGGER
        NORMALIZE -->|"Eligible event trigger"| TRIGGER
        PRINCIPAL -.->|"Actor binding"| TRIGGER
        TRIGGER -->|"Derive exact Fire Key"| FIRE
        FIRE -->|"New fire or resume same logical fire"| RESOLVE
        RESOLVE --> WORK
    end

    subgraph GOVERNED["Normal governed kernel path for any consequential effect"]
        direction LR
        CASE["CASE / WORK ITEM<br/>Objective · owner · resolved target set"]
        PACKET["IMMUTABLE OPERATION PACKET<br/>Baseline · typed capability · postconditions"]
        AUTH["POLICY + APPROVAL + GRANT<br/>Bound to packet, resource, route and validity"]
        DISPATCH["OUTBOX DISPATCH<br/>PostgreSQL commit → NATS → Execute"]
        VERIFY["INDEPENDENT VERIFICATION<br/>Fresh evidence · pass/fail/inconclusive"]
        OUTCOME["OUTCOME DECISION<br/>Accountable principal<br/>accept/reject/successor"]
        CASE --> PACKET --> AUTH --> DISPATCH --> VERIFY --> OUTCOME
    end

    subgraph DELIVERY["Notification and exception handling"]
        direction LR
        ATTEMPT["DELIVERY ATTEMPT<br/>Persist destination binding and sanitized receipt"]
        ACK["TRANSPORT ACKNOWLEDGEMENT<br/>Delivery claim only<br/>not human receipt or outcome"]
        EXCEPTION["EXCEPTION CASE<br/>Ambiguity · failure · revocation · suppression<br/>budget breach · routine health failure"]
        ATTEMPT --> ACK
    end

    HUMAN -->|"Direct Case/Control API intake"| CASE
    CORRELATE --> CASE
    WORK -->|"Consequential proposal"| CASE
    WORK -->|"Permitted notification"| ATTEMPT
    INGRESS -->|"Rejected or malformed"| EXCEPTION
    RESOLVE -->|"Ambiguous target or invalid destination"| EXCEPTION
    ATTEMPT -->|"Failure or exhausted retry"| EXCEPTION
    FIRE -->|"Duplicate resumes existing lineage"| WORK

    classDef input fill:#ffffff,stroke:#334155,stroke-width:1.5px,color:#172033;
    classDef signal fill:#fff5f7,stroke:#cc3158,stroke-width:1.5px,color:#172033;
    classDef proposed fill:#fffdf2,stroke:#9a6700,stroke-width:1.5px,stroke-dasharray:5 4,color:#172033;
    classDef governed fill:#f2faf4,stroke:#2d7d46,stroke-width:1.5px,color:#172033;
    classDef delivery fill:#eef6ff,stroke:#285b8f,stroke-width:1.5px,color:#172033;
    classDef failure fill:#fff2f2,stroke:#b42318,stroke-width:1.5px,color:#172033;

    class HUMAN,ALERT,CLOCK input;
    class INGRESS,NORMALIZE,OBS,CORRELATE signal;
    class TRIGGER,PRINCIPAL,FIRE,RESOLVE,WORK proposed;
    class CASE,PACKET,AUTH,DISPATCH,VERIFY,OUTCOME governed;
    class ATTEMPT,ACK delivery;
    class EXCEPTION failure;

    style INPUTS fill:#fafbfc,stroke:#94a3b8,stroke-width:1px
    style SIGNALS fill:#fff5f7,stroke:#cc3158,stroke-width:2px
    style ROUTINES fill:#fffdf7,stroke:#9a6700,stroke-width:2px,stroke-dasharray:5 4
    style GOVERNED fill:#f2faf4,stroke:#2d7d46,stroke-width:2px
    style DELIVERY fill:#f8fbff,stroke:#285b8f,stroke-width:1px
```

## Binding rules

- Human requests enter the Control API/Case path directly; they are not normalized
  as external Signals.
- External Signals are authenticated, schema-validated, deduplicated, and preserved
  as source-attributed Observations before Case correlation.
- Each Routine version executes as a dedicated Routine Principal and reserves one
  deterministic Fire Key atomically before downstream work.
- Target and person-directed destination bindings are versioned and revalidated;
  ambiguity, revocation, suppression, lack of consent/policy basis, or purpose
  mismatch blocks the action and creates or updates an exception Case.
- A routine proposing a consequential effect must create or select a Case and use
  the same packet, policy, approval, grant, outbox, execution, verification, and
  outcome-decision sequence as human-initiated work.
- Transport acceptance is a delivery claim, not proof that a person received,
  read, or acted on a notification.
