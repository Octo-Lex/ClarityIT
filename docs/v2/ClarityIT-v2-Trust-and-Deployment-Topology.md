# ClarityIT v2 — Trust and Deployment Topology

**Diagram version:** 0.2
**Status:** Target-state trust companion; deployment placement is governed by the Environment Trust and Evidence Custody Profile v0.1
**Draft dependencies:** Native Pattern P-09 destination-bound credential broker and P-12 Runtime Capability Profile; proposed until separately approved
**Rendered asset:** [`images/trust-and-deployment-topology.png`](images/trust-and-deployment-topology.png)

This diagram expands the cross-plane controls intentionally compressed in the
layered overview. Trust does not flow from a worker's location: each request is
bound to workspace, workload identity, packet, grant, route, destination, policy,
and compatible runtime capabilities.

```mermaid
flowchart LR
    subgraph TOP[" "]
        direction LR
    subgraph CONTROL["Central logical roles — separately scalable roles in one Go modular monolith"]
        direction LR
        API["CONTROL API / DOMAIN KERNEL<br/>Authoritative state transitions"]
        REASON["REASON<br/>Findings and proposals only<br/>credentialless"]
        EXEC["EXECUTE<br/>Committed dispatch consumer<br/>route/profile validation"]
        VERIFY["VERIFY<br/>Independent probes and postconditions"]
        EDGE["EDGE GATEWAY<br/>Authenticated Site Runtime sessions"]
    end

    subgraph TRUST["Trust services"]
        direction LR
        WID["WORKLOAD IDENTITY<br/>Short-lived identity · mTLS · route binding"]
        POLICY["SIGNED POLICY BUNDLES<br/>Versioned · centrally issued · locally enforced"]
        VAULT["VAULT / KMS<br/>Secret material and key custody"]
        CRED["DESTINATION-BOUND CREDENTIAL BROKER<br/>Workload + packet + grant + route + destination binding<br/>internal injection only · proposed P-09"]
        RCP["RUNTIME CAPABILITY PROFILE<br/>Signed versions · enforcement grade · migration-gap refusal<br/>proposed P-12"]
        VAULT --> CRED
    end
    end

    subgraph MIDDLE[" "]
        direction LR
    subgraph PROTECTED["Workspace isolation boundary — enforced server-side at every hop"]
        direction TB
        ISOLATION["CROSS-CUTTING ISOLATION CONTRACT<br/>API + WebSockets · PostgreSQL + search · NATS<br/>object prefixes/policies/manifests · credential requests<br/>caches · central/site/native routes · resource binding"]
    end

    subgraph ROUTES["Trusted execution and observation routes"]
        direction LR
        CENTRAL["CENTRAL TYPED CONNECTOR<br/>Reachable provider APIs"]
        SITE["SITE RUNTIME<br/>Outbound mTLS · local policy · offline journal"]
        ADAPTER["SITE TYPED ADAPTER<br/>Private control APIs"]
        NATIVE["NATIVE ENFORCEMENT<br/>Deterministic allow / deny / contain"]
        PROBE["VERIFIER PROBE<br/>Independent read path"]
        SITE --> ADAPTER
    end
    end

    subgraph PLACEMENT["Environment placement and custody"]
        direction LR
        DEV["CT 150 DEVELOPMENT EXCEPTION<br/>Approved bounded co-location<br/>not production evidence"]
        PROD["FRESH PRODUCTION TOPOLOGY<br/>Independent trust, application, PostgreSQL,<br/>evidence, audit and recovery failure domains"]
        NOPROMOTE["NO IN-PLACE PROMOTION<br/>Do not reuse CT 150 realm, identities, root keys,<br/>service credentials, bucket credentials or evidence keys"]
        DEV -.->|"Rebuild; never promote"| NOPROMOTE --> PROD
    end

    API --> REASON
    API --> EXEC
    EXEC --> EDGE

    WID -.-> API
    WID -.-> EXEC
    WID -.-> VERIFY
    WID -.-> EDGE
    WID -.-> CENTRAL
    WID -.-> SITE

    POLICY -.-> EXEC
    POLICY -.-> SITE
    POLICY -.-> NATIVE

    RCP -.->|"Fail closed on missing, stale, incompatible or weaker profile"| EXEC
    RCP -.-> CENTRAL
    RCP -.-> SITE

    CRED -.->|"Connector-internal injection"| CENTRAL
    CRED -.->|"Adapter-internal injection"| ADAPTER
    CRED -.->|"Probe-internal injection"| PROBE

    EXEC --> CENTRAL
    EXEC --> SITE
    EXEC --> NATIVE
    VERIFY --> PROBE

    API -.-> ISOLATION
    EXEC -.-> ISOLATION
    CRED -.-> ISOLATION
    VERIFY -.-> ISOLATION

    classDef role fill:#ffffff,stroke:#285b8f,stroke-width:1.5px,color:#172033;
    classDef trust fill:#f1fbfc,stroke:#16818b,stroke-width:1.5px,color:#172033;
    classDef proposed fill:#fffdf2,stroke:#9a6700,stroke-width:1.5px,stroke-dasharray:5 4,color:#172033;
    classDef boundary fill:#f7f8fb,stroke:#5f6b7a,stroke-width:1.5px,color:#172033;
    classDef route fill:#f2f6ff,stroke:#2563c7,stroke-width:1.5px,color:#172033;
    classDef dev fill:#fff8ef,stroke:#d67a00,stroke-width:1.5px,color:#172033;
    classDef prod fill:#eefaf3,stroke:#1f7a46,stroke-width:1.5px,color:#172033;
    classDef blocked fill:#fff2f2,stroke:#b42318,stroke-width:1.5px,color:#172033;

    class API,REASON,EXEC,VERIFY,EDGE role;
    class WID,POLICY,VAULT trust;
    class CRED,RCP proposed;
    class ISOLATION boundary;
    class CENTRAL,SITE,ADAPTER,NATIVE,PROBE route;
    class DEV dev;
    class PROD prod;
    class NOPROMOTE blocked;

    style CONTROL fill:#f8fbff,stroke:#7aa2c7,stroke-width:1px
    style TRUST fill:#f1fbfc,stroke:#16818b,stroke-width:2px
    style PROTECTED fill:#fafbfc,stroke:#5f6b7a,stroke-width:2px
    style ROUTES fill:#f2f6ff,stroke:#2563c7,stroke-width:2px
    style PLACEMENT fill:#fffdf8,stroke:#d67a00,stroke-width:2px
    style TOP fill:transparent,stroke:transparent
    style MIDDLE fill:transparent,stroke:transparent
```

## Binding rules

- Workload identity authenticates a role; it does not by itself grant an effect.
- Credentials are absent from prompts, packets, browser payloads, messages, logs,
  receipts, evidence exports, and search. A broker resolves and injects a scoped
  credential only within the bound typed adapter, connector, or verifier probe.
- Missing, stale, incompatible, or weaker-than-required runtime capability
  profiles fail closed before route dispatch.
- Workspace isolation is enforced independently across API/WebSockets, product
  state, transport, object bytes, search, credentials, caches, and runtime routes.
- CT 150 is an approved development exception only. Production is provisioned
  fresh across independent trust and evidence-custody failure domains.
