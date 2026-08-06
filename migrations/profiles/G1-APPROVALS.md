# WP-00 G1 — Approval Record

**Date:** 1 August 2026
**PR:** [#7](https://github.com/Octo-Lex/ClarityIT/pull/7)
**Commit:** `3b4a6fd`
**CI:** [Run 30684751779](https://github.com/Octo-Lex/ClarityIT/actions/runs/30684751779) — success

## Approved artifacts

| Artifact | Identity | Digest |
|---|---|---|
| Custody manifest v4 | Version `c7b86b9b-18fc-4504-942c-fe26ccc83f07` | SHA-256 `75ce4f08f4afdc7cedff0e1d977026ec5b590f8822caefdc079905ed678130c1` |
| P1/P2 schema fingerprint | — | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| P3 golden fingerprint | — | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| Operational backup | `opbak-20260731-173628` | SHA-256 `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |

## Development-exception limitations accepted

1. Single-host storage (MinIO on CT 150) — not independently durable against host failure.
2. KES uses filesystem keystore (not HA KMS).
3. Root credentials exist alongside project IAM (root retains admin bypass for GOVERNANCE retention).
4. No separate MFA for break-glass admin.
5. Audit webhook points to localhost (not a durable external SIEM).

Production deployment must provision a fresh environment with HA enterprise IAM, independently protected KES/KMS, independent evidence storage, tested recovery, and no reuse of CT 150 identities, credentials, keys, or policies.

## Approvals

Each approver confirms the v4 custody manifest identity (`c7b86b9b…` / `75ce4f08…`) and explicitly accepts the documented development limitations above.

### Architecture — APPROVE WITH CONDITIONS (DEVELOPMENT ONLY)

Architecture accepts the physical co-location of MinIO, KES, project IAM, and local audit on CT 150 as a bounded development deployment exception. The logical separation of authoritative product state, evidence storage, and trust services remains mandatory. This approval does not certify independent durability, high availability, disaster recovery, or production readiness. It expires upon scope change or production-readiness review and requires a fresh production deployment across independent trust and failure domains.

| Role | Owner | Decision | Date |
|---|---|---|---|
| Architecture | Architecture Owner | accept (with conditions, development only) | 2026-08-01 |

### Security — APPROVE WITH CONDITIONS (DEVELOPMENT ONLY)

Security explicitly accepts the residual risk created by the filesystem KES keystore, privileged root bypass, absence of separate break-glass MFA, and localhost-only audit receiver. The committee recognizes that compromise of CT 150 or its root credential could defeat confidentiality, retention administration, separation of duties, and independent auditability. Approval depends on continued least-privilege IAM use, non-routine controlled root access, exclusion of production secrets and regulated evidence, and replacement of all four exception controls before production.

| Role | Owner | Decision | Date |
|---|---|---|---|
| Security | Security Owner | accept (with conditions, development only) | 2026-08-01 |

### Operations — APPROVE WITH CONDITIONS (DEVELOPMENT ONLY)

Operations accepts that CT 150 is a single operational failure domain and that host failure may simultaneously make object storage, KES, and audit unavailable or unrecoverable. No HA, RPO, RTO, backup, or disaster-recovery credit is assigned to this deployment. Approval depends on maintaining health and capacity monitoring, configuration/key recovery procedures, tested read-back and decryption, and fail-closed behavior whenever evidence or KES is unavailable. Production requires independent storage, key, audit, and recovery paths with tested failover.

| Role | Owner | Decision | Date |
|---|---|---|---|
| Operations | Operations Owner | accept (with conditions, development only) | 2026-08-01 |

### Database — APPROVE WITH CONDITIONS (DEVELOPMENT ONLY)

Database confirms that PostgreSQL remains the authoritative product-state store and that MinIO contains evidence bytes and migration artifacts only. Any database backup stored solely on CT 150 is accepted as development evidence or rehearsal material and does not qualify as an independent backup, PITR capability, or disaster-recovery copy. Approval depends on preserving encrypted, access-controlled, digest-bound migration evidence and performing production backup, restore, reconciliation, and recovery from an independent failure domain before production authorization.

| Role | Owner | Decision | Date |
|---|---|---|---|
| Database | Database Owner | accept (with conditions, development only) | 2026-08-01 |

## G1 closure

All four gates approved with conditions (development only). G1 is closed.

| Gate | Result |
|---|---|
| Architecture | ✅ accept (conditions, dev only) |
| Security | ✅ accept (conditions, dev only) |
| Operations | ✅ accept (conditions, dev only) |
| Database | ✅ accept (conditions, dev only) |

**G1 status: CLOSED — 2026-08-01**

All approvals are conditioned on development-only deployment. Production authorization requires a fresh environment with independent trust and failure domains, HA enterprise IAM, independently protected KES/KMS, independent evidence storage, tested recovery, and no reuse of CT 150 identities, credentials, keys, or policies.
