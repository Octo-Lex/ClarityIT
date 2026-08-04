# ClarityIT v2 Environment Trust and Evidence Custody Deployment Profile

**Version:** 0.1  
**Status:** Adopted for development; normative production exit criteria  
**Decision date:** 1 August 2026  
**Applies to:** IAM, KES/KMS, evidence object storage, audit custody, administrative access, and environment promotion  
**Development placement:** CT 150  
**Production authority:** WP-10 RG-10 / Release R5

> **Decision:** KES and project IAM may be deployed with the development evidence store on CT 150 during the ClarityIT v2 project development phase. This is a bounded, time-limited development exception. It does not establish independent durability or production conformance. Production must be provisioned fresh with separate trust and evidence-custody failure domains and must not inherit CT 150 identities, root keys, or storage credentials.

## 1. Purpose and authority

This profile translates the logical Trust Services and Data and Evidence planes into environment-specific deployment rules. It governs the development exception on CT 150, the controls required before that exception can support WP-00/G1 development evidence, and the production architecture and tests required by WP-10 RG-10.

The Product Definition remains authoritative for product scope. The Authoritative Execution Kernel Specification remains authoritative for identity, authority, secret, and evidence semantics. The v1-to-v2 Compatibility and Migration Specification remains authoritative for WP-00 migration evidence. This profile governs physical trust placement, environment classification, and promotion boundaries.

The layered architecture diagram is logical. Separate planes identify separate responsibilities and trust boundaries; they do not require a separate host for every development component. Production physical placement is governed by this profile and cannot be inferred from development co-location.

## 2. Environment classifications

| Profile | Permitted placement | Assurance statement | Promotion status |
|---|---|---|---|
| Development bootstrap | MinIO evidence storage, KES, and project IAM may share CT 150. | Suitable for development after the controls in section 3 pass. The shared-host durability risk is accepted and recorded. | Cannot be promoted in place. |
| Controlled non-production | May use the development bootstrap or a production-shaped test topology, as declared by the release plan. | Suitable only for the declared pilot scope; never evidence of production readiness by itself. | Configuration may be reviewed for reuse; credentials and key material cannot be reused. |
| Production | Dedicated trust and custody services across approved independent failure domains. | Must satisfy section 5 and RG-10 before any production enrollment. | Fresh deployment only. |

Environment labels are authoritative attributes of the deployment contract and Resource bindings. A deployment cannot self-declare a higher profile because encryption is enabled or because components run in separate containers.

## 3. Development bootstrap profile - CT 150

### 3.1 Permitted placement

CT 150 may host the non-production ClarityIT application, development MinIO evidence store, KES service, and project IAM service in one failure domain. Logical service boundaries, separate processes/containers, separate credentials, and explicit network policy remain mandatory even when the host is shared.

This exception applies only during project development. CT 150 must not hold production user data, production service credentials, production identity signing keys, production KES root material, or production evidence keys.

### 3.2 Mandatory controls

Before WP-00/G1 development acceptance, the CT 150 profile must demonstrate:

1. KES-backed server-side encryption for every restricted evidence object, with key identifiers and rotation metadata recorded in the custody manifest.
2. A dedicated least-privilege evidence-writer identity that can create required versions but cannot administer IAM, KES, retention, legal hold, or bucket policy.
3. A separate read-only verifier identity that retrieves exact object version IDs and cannot write, delete, bypass retention, remove legal hold, change policy, administer KES, or issue identities.
4. Separate human administration identities protected by MFA where supported; shared root credentials are prohibited for routine operation and retained only as controlled break-glass access.
5. Versioning, object lock, declared retention, and legal hold for the development evidence set.
6. Version-ID-specific read-back of every evidence object through the verifier identity, with SHA-256 and size matched to the sealed custody manifest.
7. Tests proving that the writer and verifier cannot explicitly delete a retained version, bypass retention, remove legal hold, alter the custody manifest version, or access KES root material.
8. Audit events for upload, read-back, denied control operations, IAM administration, KES administration, key use, retention, and legal-hold changes, copied to the declared audit receiver and included in the evidence pack.
9. Recovery evidence for IAM configuration, KES configuration and key references, bucket policy, object versions, and the custody manifest. Recovery documentation must state that a CT 150 host loss can still cause development evidence unavailability or loss.
10. A signed development risk acceptance identifying the owner, expiry/review point, permitted data class, shared-host failure mode, recovery limitation, and production exit obligation.

### 3.3 Accepted development limitations

The following limitations are accepted only for this development profile:

- IAM, KES, object storage, and application components share one host failure domain.
- The CT 150 evidence repository is not independently durable against loss or compromise of that host.
- Availability and disaster-recovery objectives are development-grade.
- Administrative separation is logical and identity-based rather than physically isolated.

These limitations must remain visible in the custody manifest, the WP-00/G1 decision, status reporting, and any exported evidence. They cannot be reclassified as remediated merely because KES and IAM are installed.

### 3.4 Development acceptance decision

WP-00/G1 may be accepted for development after all section 3.2 controls pass and Architecture, Security, Operations, and Database owners sign the exact evidence manifest and risk acceptance. The decision authorizes continued development work only. It does not authorize production enrollment, production data, or G2 activities that are explicitly dependent on production-grade custody.

## 4. Development implementation boundary

The development deployment should use explicit service and network boundaries:

- IAM authenticates human and workload principals and issues environment-scoped identities.
- KES provides the key service used by the evidence store; application and evidence-writer identities never receive KES root credentials.
- MinIO stores encrypted, versioned evidence objects and enforces object lock, retention, and legal hold.
- ClarityIT stores immutable object references, version IDs, digests, key identifiers, and custody-manifest references; it does not become the key authority.
- Audit records are treated as evidence claims until retrieved from the declared receiver and sealed into the evidence manifest.

The development profile must use environment-qualified issuers, audiences, key names, bucket names, policies, and service identities so no development identity can authenticate to production.

## 5. Production target profile

### 5.1 Required topology

Production must be provisioned as a fresh deployment with these distinct trust and failure-domain boundaries:

1. **Enterprise identity boundary:** federation with the approved enterprise IdP, a production IAM service deployed for high availability across approved failure domains, MFA for privileged human access, short-lived workload identity, environment-bound audiences, and separation of security administration from application administration.
2. **Key-management boundary:** an HA KES tier separate from the evidence store and ClarityIT application, backed by an approved external root of trust such as an HSM, enterprise KMS, or approved secrets platform. KES administrators must not be evidence-store administrators.
3. **Evidence-storage boundary:** production S3-compatible storage or managed object storage with encryption under externally protected keys, versioning, object lock/WORM retention, legal hold, independent durable replication, integrity checking, and capacity/retention monitoring.
4. **Independent verification boundary:** dedicated writer and read-only verifier workload identities with non-overlapping permissions. The verifier retrieves exact versions and verifies digests but cannot mutate evidence or administer trust services.
5. **Audit boundary:** IAM, KES, storage, administrator, and ClarityIT audit events delivered to an external durable audit/SIEM receiver outside the protected system's primary failure domain and retained under the approved policy.
6. **Recovery boundary:** tested backup and recovery for IAM configuration, identity signing material as permitted, KES configuration and root-of-trust references, storage metadata, object replicas, policies, custody manifests, and audit evidence.

Separate containers or LXCs on the same Proxmox host do not create an independent production failure domain. The production design must identify the actual compute, storage, administrative, and site failure domains and the failure each boundary is intended to survive.

### 5.2 Production identity and key rules

- Production uses a distinct IAM realm/tenant, issuers, audiences, clients, service identities, groups, roles, and break-glass accounts.
- Production uses new KES identities, root-of-trust bindings, wrapping keys, evidence keys, and rotation schedules.
- Key material is non-exportable where the selected platform supports it. Applications receive only the narrow cryptographic operation or data-key service needed for their role.
- Privileged access is time-bounded, approved, attributable, audited, and unavailable to reasoning workers.
- No single routine administrator can administer IAM, KES, evidence retention, application deployment, and independent verification.

### 5.3 Production evidence-custody rules

For every protected evidence set, the production custody manifest must bind object key, immutable version ID, SHA-256, size, creation time, writer principal, encryption mode, KMS/KES key identifier, retention mode and expiry, legal-hold state, replication state, audit references, and the governing ClarityIT commit/release.

The manifest itself must be signed, stored under the same immutable controls, independently replicated, and verified through the read-only verifier identity. Credentials, private keys, raw secrets, and presigned URLs must never appear in the repository or manifest.

## 6. No-in-place-promotion rule

CT 150 is not upgraded into production. Production is built from approved, signed artifacts and version-controlled configuration in a fresh trust domain.

The following must never be copied or reused from CT 150 in production:

- IAM realm databases, signing keys, client secrets, refresh tokens, sessions, service-account credentials, or break-glass credentials;
- KES root credentials, root keys, wrapping keys, evidence keys, API keys, or administrator identities;
- MinIO root credentials, bucket credentials, service tokens, presigned URLs, or development policy bindings;
- development certificates, trust stores, recovery codes, or audit-receiver credentials.

Only reviewed declarative configuration, signed release artifacts, policy intent, schema definitions, and sanitized manifests may be promoted. Every promoted item receives a new production identity, signature/digest, approval, and deployment evidence record.

## 7. Production acceptance gate - WP-10 RG-10

Production enrollment is blocked until all of the following are demonstrated and approved:

1. The topology survives the declared loss of one IAM instance/failure domain, one KES instance/failure domain, and one evidence-storage failure domain without losing authoritative identity, key service, retained evidence, or audit history.
2. Every protected object is encrypted, retained, independently replicated, retrieved by exact version through the verifier identity, and hash-verified.
3. Rotation of IAM signing material and KES/evidence keys completes without loss of access to retained evidence; revoked identities and old keys behave according to policy.
4. Explicit version deletion, retention bypass, legal-hold removal, policy change, and key administration are denied to writer, verifier, application, and ordinary administrator identities.
5. Break-glass use requires approved procedure, produces durable audit evidence, and is followed by credential rotation and review.
6. IAM, KES, evidence-store, application, and audit recovery exercises meet approved RTO/RPO and reproduce the declared manifests.
7. The executable deployment contract proves that no CT 150 identity, credential, root key, evidence key, or bucket policy was reused.
8. Independent Security and Operations reviews pass, and Product, Engineering, Operations, Security, Database, and Quality owners record the production go/no-go against the exact evidence manifest.

No approval can convert the development CT 150 profile into production conformance without this evidence.

## 8. Accountability and review

| Owner | Accountability |
|---|---|
| Architecture | Approve environment classification, failure domains, trust boundaries, and no-in-place-promotion design. |
| Security | Approve IAM, KES/KMS, privileged access, separation of duties, encryption, retention, legal hold, and audit controls. |
| Operations | Operate CT 150 controls during development; design and prove production HA, monitoring, recovery, and incident procedures. |
| Database | Approve migration evidence handling, backup custody, restore proof, and data classification. |
| Engineering | Implement environment-bound identities, secret references, evidence metadata, and deployment checks without embedding credentials. |
| Quality | Verify conformance scenarios, negative authorization tests, recovery exercises, and evidence reconstruction. |
| Product | Accept the development risk for project scope and record the production go/no-go with the required owners. |

The development exception must be reviewed at the earlier of: a material topology or data-class change, production-planning start, any CT 150 security incident, failure of recovery evidence, or WP-10 entry.

## 9. Required evidence

The environment-profile evidence pack must include:

- approved topology and failure-domain register;
- IAM realm/tenant, issuer, audience, role, service-identity, MFA, and break-glass policy identifiers without secret values;
- KES/KMS configuration digest, root-of-trust identifier, key identifiers, rotation result, and recovery evidence without key material;
- storage versioning, object-lock, retention, legal-hold, encryption, replication, and policy evidence;
- exact-version read-back and hash-verification results;
- denied deletion, bypass, legal-hold, policy, and key-administration tests;
- durable audit receiver evidence for successful and denied operations;
- signed custody manifest and immutable version reference;
- development risk acceptance or production go/no-go decision, as applicable.

## 10. Relationship to the reference architecture

The Trust Services plane continues to contain workload identity, secrets/key services, signed policy bundles, and the trust boundary. The Data and Evidence plane continues to contain the object store and evidence references. CT 150 co-location changes only the development deployment profile; it does not merge those logical responsibilities, move authority into the storage service, or allow an agent to hold credentials.

For production, the same logical architecture is realized through physically and administratively separated trust services and evidence custody. The layered architecture diagram therefore remains valid and does not require a provider-specific redraw.
