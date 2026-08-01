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

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | ☐ accept ☐ block | | |
| Security | | ☐ accept ☐ block | | |
| Operations | | ☐ accept ☐ block | | |
| Database | | ☐ accept ☐ block | | |
