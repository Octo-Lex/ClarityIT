# ClarityIT v2 — Layered System Architecture

> **Scope:** Target-state reference architecture. Site Runtime, Native Enforcement,
> and the optional Host Sensor are future architecture and are outside WP-00 and
> the initial central-route Proxmox slice.

```mermaid
flowchart TB
    USER["USER<br/>Operator, employee, reviewer"]
    SIGNAL["SIGNAL<br/>Alerts, requests, schedules and routines"]
    IDP["IDP<br/>Enterprise identity provider"]

    subgraph EXPERIENCE["1. Experience plane"]
        direction TB
        WEB["WEB<br/>React and TypeScript web application"]
        SURFACES["SURFACES<br/>My Work · Case Workspace · Resources · Team · Knowledge"]
        CLIENT["CLIENT<br/>REST/OpenAPI commands · WebSocket progress"]
        WEB --> SURFACES --> CLIENT
    end

    subgraph CONTROL["2. Authoritative control plane"]
        direction TB
        API["CONTROL API<br/>Controlled query, command and artifact interfaces"]
        IAM["IAM<br/>RBAC · workspace isolation · enterprise identity"]
        KERNEL["DOMAIN KERNEL<br/>objective · work item · case · resource · decision · outcome"]
        AUTHORITY["AUTHORITY<br/>Policy · approvals · scoped authority grants"]
        BROKER["EFFECT BROKER<br/>Preflight authorization · attempt creation · dispatch record"]
        EDGE["REMOTE-SITE EDGE GATEWAY<br/>Authenticate sessions · validate signed envelopes"]
        INGRESS["RECEIPT / RESULT INGRESS<br/>Authenticate · validate · deduplicate · normalize"]

        API --> IAM --> KERNEL --> AUTHORITY
        AUTHORITY -->|"Signed immutable Operation Packet + grant"| BROKER
        EDGE -->|"Signed envelopes"| INGRESS
    end

    subgraph PROCESSING["3. Intelligence and processing"]
        direction TB
        REASON["REASON<br/>Investigate and propose through controlled backend APIs"]
        OBSERVE["OBSERVE<br/>Normalize signals and create source-attributed observations"]
        EXECUTE["EXECUTE<br/>Dispatch · poll · reconcile · retry · compensate<br/>route = central | site | native_guard"]
        VERIFY["VERIFY<br/>Independently evaluate versioned postconditions using fresh evidence"]
    end

    subgraph DATA["4. Data and evidence plane"]
        direction TB
        POSTGRES[("POSTGRESQL<br/>Authoritative product state<br/>inbox · outbox · aggregate versions")]
        NATS[("NATS JETSTREAM<br/>Durable commands and events<br/>transport — never product truth")]
        OBJECTS[("OBJECT STORAGE<br/>Immutable evidence, outputs and signed manifests")]
        SEARCH["SEARCH<br/>PostgreSQL FTS and pgvector retrieval"]
        POSTGRES --- SEARCH
    end

    subgraph TRUST["5. Trust services"]
        direction TB
        WID["WORKLOAD IDENTITY<br/>Short-lived identity · mutual TLS · route binding"]
        SECRETS["SECRETS<br/>Vault or KMS · least privilege · executor-only access"]
        BUNDLES["POLICY BUNDLES<br/>Signed · versioned · centrally issued · locally enforced"]
        BOUNDARY["TRUST BOUNDARY<br/>Agents never receive target credentials<br/>Routes revalidate packet, grant, expiry, nonce, identity, resource binding and local policy"]
    end

    subgraph TARGET["6. Target plane"]
        direction TB

        subgraph ZONE["Customer network zone or cluster"]
            direction TB
            SITE["SITE RUNTIME<br/>Signed Go runtime · outbound-initiated mTLS"]
            LOCAL["LOCAL CONTROL<br/>Packet validation · local policy · encrypted offline journal"]
            ADAPTERS["TYPED ADAPTERS<br/>Provider-neutral capability translation"]
            PLATFORM["PLATFORM CONTROL APIs<br/>Compute · cluster · database · network"]
            NATIVE["NATIVE ENFORCEMENT<br/>IAM · admission · gateway · database · OS security hooks"]
            HOST["OPTIONAL HOST SENSOR<br/>Only for required host-local visibility or enforcement"]
            SYSTEMS["MANAGED SYSTEMS<br/>Infrastructure, applications, data and workloads"]

            SITE --> LOCAL
            LOCAL --> ADAPTERS --> PLATFORM --> SYSTEMS
            LOCAL --> NATIVE --> SYSTEMS
            HOST -.->|"Required host-local capabilities only"| SITE
            HOST -.-> SYSTEMS
        end

        CENTRAL["CENTRAL CONNECTORS<br/>Reachable provider, SaaS and public-cloud APIs"]
        EXTERNAL["EXTERNAL SERVICES<br/>SaaS · public cloud · provider control planes"]
        CENTRAL --> EXTERNAL
    end

    subgraph SOURCES["7. Existing operational sources"]
        direction TB
        TELEMETRY["TELEMETRY<br/>Monitoring · logs · OpenTelemetry · event systems"]
        HEALTH["HEALTH<br/>Independent technical and business endpoints"]
    end

    USER --> WEB
    IDP --> IAM
    CLIENT -->|"Commands and queries"| API
    SIGNAL --> OBSERVE

    API -->|"Controlled API"| REASON
    REASON -->|"Structured findings and proposal artifacts"| API
    OBSERVE -->|"Validated observation candidates"| API

    BROKER -->|"Authorized attempt — sole dispatch path"| EXECUTE
    EXECUTE -->|"route = central"| CENTRAL
    EXECUTE -->|"route = site"| EDGE
    EXECUTE -->|"route = native_guard"| NATIVE

    SITE <-->|"Outbound-initiated mTLS session"| EDGE
    CENTRAL -->|"Signed receipts and claims"| INGRESS
    SITE -->|"Signed receipts, observations and outputs"| EDGE

    INGRESS -->|"Validate and persist before publication"| POSTGRES
    KERNEL <-->|"Authoritative transactions"| POSTGRES
    POSTGRES -->|"Committed transactional outbox"| NATS

    NATS -->|"Committed events"| OBSERVE
    NATS -->|"Durable command and lifecycle events"| EXECUTE
    NATS -->|"verification.requested"| VERIFY

    TELEMETRY --> OBSERVE
    VERIFY -.->|"Read-only resulting-state checks"| PLATFORM
    VERIFY -.-> TELEMETRY
    VERIFY -.-> HEALTH
    VERIFY -->|"Verification result through controlled API"| API

    EXECUTE -->|"Outputs and evidence"| OBJECTS
    VERIFY -->|"Evidence and probe results"| OBJECTS

    WID -.-> API
    WID -.-> EDGE
    WID -.-> SITE
    SECRETS -.-> EXECUTE
    SECRETS -.-> SITE
    AUTHORITY --> BUNDLES
    BUNDLES -.-> EXECUTE
    BUNDLES -.-> SITE
    BUNDLES -.-> NATIVE
    BOUNDARY -.-> EXECUTE
    BOUNDARY -.-> SITE
    BOUNDARY -.-> NATIVE

    classDef external fill:#ffffff,stroke:#334155,stroke-width:1.5px,color:#172033;
    classDef experience fill:#ffffff,stroke:#2563eb,stroke-width:1.5px,color:#172033;
    classDef control fill:#ffffff,stroke:#2d7d46,stroke-width:1.5px,color:#172033;
    classDef processing fill:#ffffff,stroke:#6d43c0,stroke-width:1.5px,color:#172033;
    classDef data fill:#ffffff,stroke:#d67a00,stroke-width:1.5px,color:#172033;
    classDef trust fill:#ffffff,stroke:#16818b,stroke-width:1.5px,color:#172033;
    classDef target fill:#ffffff,stroke:#2563c7,stroke-width:1.5px,color:#172033;
    classDef sources fill:#ffffff,stroke:#cc3158,stroke-width:1.5px,color:#172033;

    class USER,SIGNAL,IDP external;
    class WEB,SURFACES,CLIENT experience;
    class API,IAM,KERNEL,AUTHORITY,BROKER,EDGE,INGRESS control;
    class REASON,OBSERVE,EXECUTE,VERIFY processing;
    class POSTGRES,NATS,OBJECTS,SEARCH data;
    class WID,SECRETS,BUNDLES,BOUNDARY trust;
    class SITE,LOCAL,ADAPTERS,PLATFORM,NATIVE,HOST,SYSTEMS,CENTRAL,EXTERNAL target;
    class TELEMETRY,HEALTH sources;

    style EXPERIENCE fill:#f3f6ff,stroke:#2563eb,stroke-width:2px
    style CONTROL fill:#f2faf4,stroke:#2d7d46,stroke-width:2px
    style PROCESSING fill:#f7f3ff,stroke:#6d43c0,stroke-width:2px
    style DATA fill:#fff8ef,stroke:#d67a00,stroke-width:2px
    style TRUST fill:#f1fbfc,stroke:#16818b,stroke-width:2px
    style TARGET fill:#f2f6ff,stroke:#2563c7,stroke-width:2px
    style ZONE fill:#f8faff,stroke:#7aa2e3,stroke-width:1px
    style SOURCES fill:#fff5f7,stroke:#cc3158,stroke-width:2px
```

## Relationship legend

| Notation | Meaning |
|---|---|
| Solid directed line | Authoritative command, transaction, or committed lifecycle flow |
| Dashed directed line | Observation, read-only proof, identity, policy, or trust control |
| Bidirectional line | Mutually authenticated session or authoritative transaction exchange |
| `PostgreSQL → NATS JetStream` | Persist-before-publish path through the committed transactional outbox |

## Binding invariants

1. The Effect Broker dispatches only through Execute.
2. Execute owns route selection and execution-lifecycle tracking.
3. Inbound receipts and results become authoritative records only after validation
   and PostgreSQL commit; persistence does not make a claim Verified or Accepted.
4. NATS JetStream carries committed commands and events; it never creates product
   truth.
5. Reasoning workers submit findings and proposals through controlled backend APIs.
6. Provider, worker and agent outputs remain source-attributed claims after
   persistence. Only independent verification can establish a verified result,
   and only a separate outcome decision can accept it.

> **Execution truth invariant:** Provider, worker and agent outputs remain
> source-attributed claims after persistence. Only independent verification can
> establish a verified result, and only a separate outcome decision can accept it.
