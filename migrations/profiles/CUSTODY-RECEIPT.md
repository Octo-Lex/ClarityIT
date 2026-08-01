# WP-00 G1 — Evidence Custody Receipt

**Date:** 1 August 2026
**Receipt author:** `evidence-writer` (MinIO IAM principal)
**Bound commit:** `22050b8`
**Bound backup:** `opbak-20260731-173628`
**Bound P1/P2 fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
**Bound P3 golden:** `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`

This receipt is a **repository-side record** of the externally stored custody
manifest. It does not alter the manifest's bytes. The manifest itself does not
contain its own digest or version ID (see its self-referential seal note).

## Sealed custody manifest (13th evidence object)

| Property | Value |
|---|---|
| Object key | `custody-manifest.md` |
| Bucket | `clarityit-g1-evidence` |
| Version ID | `11e6ab63-bb41-46dc-aff2-4ff50f85b761` |
| Size | 6,469 bytes |
| SHA-256 | `36eac69fd1c68204eb2450b0957ace64db51f8f106a754f340e8fb081ea021d0` |
| Uploaded | 2026-08-01T02:48:55Z |
| Principal | `evidence-writer` |
| Encryption | SSE-KMS (key: `clarityit-evidence-key`) |
| Retention | GOVERNANCE, 2555 days (expiry 2033-07-31) |
| Legal hold | ON |
| Verifier result | ✅ Retrieved via `evidence-verifier`; content matches first 2 lines |
| Prior version (v1) | `7cfcbb0f-4df2-44af-8aec-90de39a5a4c7` (superseded, immutable) |

## Verification evidence

The sealed manifest was:
1. Computed to SHA-256 `36eac69f…` **before** upload (manifest bytes unchanged post-seal).
2. Uploaded by `evidence-writer` identity with SSE-KMS, retention, and legal-hold.
3. Retrieved by `evidence-verifier` (read-only) identity and content verified.
4. Version ID `11e6ab63…` assigned immutably by MinIO.

## Summary of all 13 evidence objects

12 artifacts + 1 custody manifest = 13 immutable objects in `clarityit-g1-evidence`,
all encrypted (SSE-KMS), retained (GOVERNANCE 2555d), legal-hold ON, and SHA-256
verified via independent read-only `evidence-verifier` identity.

## Approvals required

This development-exception custody arrangement requires Architecture, Security,
Operations, and Database approval against this exact receipt.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Security | | | | |
| Operations | | | | |
| Database | | | | |
