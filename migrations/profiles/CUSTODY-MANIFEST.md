# WP-00 G1 — Evidence Custody Manifest (v2)

**Date:** 1 August 2026
**Commit:** `22050b8`
**Backup:** `opbak-20260731-173628`
**P1/P2 fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
**P3 golden:** `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`

> **This manifest does not contain its own digest or version ID.**
> The uploaded object's immutable version ID and SHA-256 are recorded in the
> separate custody receipt (`CUSTODY-RECEIPT.md`) after upload, so the manifest
> bytes never change after sealing. This manifest supersedes v1 (version
> `11e6ab63-bb41-46dc-aff2-4ff50f85b761`) which is preserved as immutable history.

## Storage configuration

| Property | Value |
|---|---|
| Platform | MinIO (S3-compatible) on CT 150 |
| MinIO version | RELEASE.2025-09-07T16-13-09Z |
| Bucket | `clarityit-g1-evidence` |
| Versioning | Enabled (immutable version IDs per upload) |
| Object lock | Enabled (bucket created with `--with-lock`) |
| Retention mode | GOVERNANCE, 2555 days (~7 years); expiry 2033-07-31 |
| Legal hold | ON for all evidence objects |
| Encryption | SSE-KMS via KES (key: `clarityit-evidence-key`); Encryption ✔ Decryption ✔ |
| KMS | KES 2025-03-12, filesystem keystore, key `clarityit-evidence-key` |
| KMS key identity | `clarityit-evidence-key` (KES master key) |
| Audit logging | `audit_webhook` configured (endpoint: localhost:9000); KES audit to stdout (Docker logs) |
| Writer identity | `evidence-writer` (PutObject, GetObject, ListBucket; no DeleteObject/IAM/KMS admin) |
| Verifier identity | `evidence-verifier` (GetObject, ListBucket; PutObject/DeleteObject/IAM/KMS admin denied) |
| Break-glass admin | Root credentials (`clarityit`); no separate MFA configured (development exception) |

### Deployment profile

This is a **time-bounded development exception** per the approved *Environment Trust and Evidence Custody Deployment Profile v0.1*.

**Development-exception limitations:**
1. Single-host storage (MinIO on CT 150) — not independently durable against host failure.
2. KES uses filesystem keystore (not HA KMS).
3. Root credentials exist alongside project IAM (root retains admin bypass for GOVERNANCE retention).
4. No separate MFA for break-glass admin.
5. Audit webhook points to localhost (not a durable external SIEM).

## Original evidence objects (12 artifacts)

All uploaded 2026-08-01T02:09:46Z–02:09:47Z by `evidence-writer`. All SSE-KMS encrypted, GOVERNANCE retention 2555d, legal-hold ON. All SHA-256 verified via `evidence-verifier`.

| Key | Version ID | Size | SHA-256 |
|---|---|---|---|
| `manifest.json` (P1) | `1fd353ec-2258-4cd5-af57-fdbafc2c7f3a` | 250,239 | `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` |
| `manifest-p2a.json` | `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1` | 250,235 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |
| `manifest-p2b.json` | `5a1fa1db-d2f8-411b-826c-897e832cd6e3` | 250,235 | `db7578616d1acddc74885a5c67e4724cc83c9fd698bb56765deed260afb1c173` |
| `opbak.sql.gz` | `b315248b-9ddd-4c32-9886-c7d3035c4a37` | 1,228,736 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| `p2a-restore.log` | `6b9093b5-801b-495a-bf6f-a85b04dd3352` | 10,397 | `541ba3cbebbaaa97497bb7e4729ae513bb1d43e0470bf431c2e9d0d24ff69c74` |
| `p2b-restore.log` | `c87af47e-a492-4203-9be5-c46bef1edf70` | 10,397 | `9c9f5a6454bff50d2110a093233948e4859e128fe52a96cde3843b140363ae3a` |
| `job-log-sanitized.txt` | `2cb49e99-9584-4294-9227-fb95f9a27a5d` | 703 | `a43e20e30db13e779c18d5e75e3662970a629003033657e92c14b8100eb9a7c8` |
| `service.conf` | `03669237-d4b3-4d8b-8e8c-90fb8e6e24a4` | 212 | `ecfa4f6c54160917c831eb53fe374392c2d7961eb69c70c51d3467e115fbda8f` |
| `timer.conf` | `659f578d-04ce-44ea-94e9-11a9425f915f` | 150 | `56c4f90534281cfff2f076e7151cdef57ebab40575aa448f7ba67334a80580ec` |
| `systemctl-cat.txt` | `c2df7286-5b91-4235-b5d2-0949a6145aeb` | 455 | `99d7378cbacc8c882b74d6baf2002b2db5133159fb90c17719e79f7334b5696d` |
| `systemctl-list-timers.txt` | `de77826a-10e3-4558-90a9-2982e52ab722` | 165 | `18f5e770160b0bf4ea783ceda3efdf8d20e7ea428e6480982e058a753a098b89` |
| `timedatectl.txt` | `3fad4ee9-9cfd-4a1e-ab8e-ea8fe4b3117c` | 8 | `f0dcac7b1d721d2f68937a71f0229b4c4f88564fd711339951528889913cd85d` |

## Control evidence objects (6 artifacts)

All uploaded 2026-08-01T04:21:55Z by `evidence-writer`. All SSE-KMS encrypted, GOVERNANCE retention 2555d, legal-hold ON. All SHA-256 verified via `evidence-verifier`.

| Key | Version ID | Size |
|---|---|---|
| `controls/per-object-metadata.txt` | `ba989131-9535-4407-b2b6-f6ccb0b4ae53` | 7,805 |
| `controls/denial-tests.txt` | `5cbd0ade-1815-474c-8c34-73dd5d1f3cfb` | 2,822 |
| `controls/bucket-config.txt` | `0b496a67-435d-4a0b-ae3e-3d5f38283b65` | 742 |
| `controls/recovery-tests.txt` | `35311f6a-fdbc-4725-aca2-ad60d4890dd1` | 1,112 |
| `controls/audit-evidence.txt` | `26b52737-78d9-4238-a0e0-a75576b123e5` | 805 |
| `controls/stat-custody-manifest.json` | `2c807abd-cc73-47cd-a9f6-46bc7634985b` | 390 |

## Denial test results summary

| Identity | Operation | Result |
|---|---|---|
| `evidence-writer` | Delete | ❌ Access Denied |
| `evidence-writer` | Retention reduction | ❌ Not supported for identity |
| `evidence-writer` | Legal-hold clear | ❌ Access Denied |
| `evidence-writer` | IAM policy list | ❌ Access Denied |
| `evidence-writer` | KMS key status | ❌ Access Denied |
| `evidence-verifier` | Delete | ❌ Access Denied |
| `evidence-verifier` | Retention bypass | ❌ Not supported for identity |
| `evidence-verifier` | Legal-hold clear | ❌ Access Denied |
| `evidence-verifier` | Write | ❌ Access Denied |
| `evidence-verifier` | IAM policy list | ❌ Access Denied |
| `evidence-verifier` | KMS key status | ❌ Access Denied |

## Recovery test results summary

| Test | Result |
|---|---|
| Exact-version read-back + decryption | ✅ SHA-256 match |
| KES key material available | ✅ Encryption ✔ Decryption ✔ |
| IAM identities queryable | ✅ Both confirmed |
| Bucket controls functional | ✅ Versioning + object-lock + legal-hold |

## Prior manifest versions (immutable history)

| Version | Version ID | Superseded by |
|---|---|---|
| v1 | `7cfcbb0f-4df2-44af-8aec-90de39a5a4c7` | v2 (`11e6ab63…`) |
| v2 | `11e6ab63-bb41-46dc-aff2-4ff50f85b761` | v3 (this manifest) |

## Risk acceptance

This development-exception custody arrangement requires Architecture, Security, Operations, and Database approval.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Security | | | | |
| Operations | | | | |
| Database | | | | |
