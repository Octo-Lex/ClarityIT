# WP-00 G1 — Evidence Custody Manifest

**Date:** 1 August 2026
**Commit:** `22050b8`
**Backup:** `opbak-20260731-173628`
**P1/P2 fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
**P3 golden:** `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`

> **This manifest does not contain its own digest or version ID.**
> The uploaded object's immutable version ID and SHA-256 are recorded in the
> separate custody receipt (`CUSTODY-RECEIPT.md`) after upload, so the manifest
> bytes never change after sealing.

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
| Encryption | SSE-KMS via KES (key: `clarityit-evidence-key`) |
| KMS | KES 2025-03-12, filesystem keystore, key `clarityit-evidence-key` (Encryption ✔ Decryption ✔) |
| KMS key identity | `clarityit-evidence-key` (KES master key) |
| Audit logging | `audit_webhook` configured; KES audit to stdout (Docker logs) |
| Writer identity | `evidence-writer` (PutObject, GetObject, ListBucket; no DeleteObject) |
| Verifier identity | `evidence-verifier` (GetObject, ListBucket; PutObject/DeleteObject denied) |

### Deployment profile

This is a **time-bounded development exception** per the approved *Environment Trust and Evidence Custody Deployment Profile v0.1*. Development: KES, project IAM, and MinIO coexist on CT 150 subject to encryption, separate writer/verifier identities, object-lock controls, audit evidence, recovery evidence, and signed risk acceptance. Production: fresh environment with HA enterprise IAM, independently protected KES/KMS, independent evidence storage, tested recovery, and no reuse of CT 150 identities, credentials, keys, or policies.

**Development-exception limitations:**
1. Single-host storage (MinIO on CT 150) — not independently durable against host failure.
2. KES uses filesystem keystore (not HA KMS).
3. Root credentials exist alongside project IAM (root retains admin bypass for GOVERNANCE retention).

## Evidence objects (12 artifacts)

All uploaded 2026-08-01T02:09:46Z–02:09:47Z by principal `evidence-writer`.

| Key | Version ID | Size | SHA-256 | Upload (UTC) | Principal | Retention Expiry | Legal Hold |
|---|---|---|---|---|---|---|---|
| `manifest.json` (P1) | `1fd353ec-2258-4cd5-af57-fdbafc2c7f3a` | 250,239 | `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` | 2026-08-01T02:09:46Z | `evidence-writer` | 2033-07-31 | ON |
| `manifest-p2a.json` | `f7de1fa9-011c-4ee2-bd20-cf6046fbf6c1` | 250,235 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` | 2026-08-01T02:09:46Z | `evidence-writer` | 2033-07-31 | ON |
| `manifest-p2b.json` | `5a1fa1db-d2f8-411b-826c-897e832cd6e3` | 250,235 | `db7578616d1acddc74885a5c67e4724cc83c9fd698bb56765deed260afb1c173` | 2026-08-01T02:09:46Z | `evidence-writer` | 2033-07-31 | ON |
| `opbak.sql.gz` | `b315248b-9ddd-4c32-9886-c7d3035c4a37` | 1,228,736 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `p2a-restore.log` | `6b9093b5-801b-495a-bf6f-a85b04dd3352` | 10,397 | `541ba3cbebbaaa97497bb7e4729ae513bb1d43e0470bf431c2e9d0d24ff69c74` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `p2b-restore.log` | `c87af47e-a492-4203-9be5-c46bef1edf70` | 10,397 | `9c9f5a6454bff50d2110a093233948e4859e128fe52a96cde3843b140363ae3a` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `job-log-sanitized.txt` | `2cb49e99-9584-4294-9227-fb95f9a27a5d` | 703 | `a43e20e30db13e779c18d5e75e3662970a629003033657e92c14b8100eb9a7c8` | 2026-08-01T02:09:46Z | `evidence-writer` | 2033-07-31 | ON |
| `service.conf` | `03669237-d4b3-4d8b-8e8c-90fb8e6e24a4` | 212 | `ecfa4f6c54160917c831eb53fe374392c2d7961eb69c70c51d3467e115fbda8f` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `timer.conf` | `659f578d-04ce-44ea-94e9-11a9425f915f` | 150 | `56c4f90534281cfff2f076e7151cdef57ebab40575aa448f7ba67334a80580ec` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `systemctl-cat.txt` | `c2df7286-5b91-4235-b5d2-0949a6145aeb` | 455 | `99d7378cbacc8c882b74d6baf2002b2db5133159fb90c17719e79f7334b5696d` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `systemctl-list-timers.txt` | `de77826a-10e3-4558-90a9-2982e52ab722` | 165 | `18f5e770160b0bf4ea783ceda3efdf8d20e7ea428e6480982e058a753a098b89` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |
| `timedatectl.txt` | `3fad4ee9-9cfd-4a1e-ab8e-ea8fe4b3117c` | 8 | `f0dcac7b1d721d2f68937a71f0229b4c4f88564fd711339951528889913cd85d` | 2026-08-01T02:09:47Z | `evidence-writer` | 2033-07-31 | ON |

## Independent verification results (read-only verifier `evidence-verifier`)

All 12 artifacts retrieved via `evidence-verifier` identity and SHA-256 verified:

| Key | Result |
|---|---|
| `manifest.json` | ✅ SHA-256 match |
| `manifest-p2a.json` | ✅ SHA-256 match |
| `manifest-p2b.json` | ✅ SHA-256 match |
| `opbak.sql.gz` | ✅ SHA-256 match |
| `p2a-restore.log` | ✅ SHA-256 match |
| `p2b-restore.log` | ✅ SHA-256 match |
| `job-log-sanitized.txt` | ✅ SHA-256 match |
| `service.conf` | ✅ SHA-256 match |
| `timer.conf` | ✅ SHA-256 match |
| `systemctl-cat.txt` | ✅ SHA-256 match |
| `systemctl-list-timers.txt` | ✅ SHA-256 match |
| `timedatectl.txt` | ✅ SHA-256 match |

## Immutability proofs (via `evidence-verifier` identity)

| Operation | Result |
|---|---|
| Delete object | ❌ `Access Denied` — verifier cannot delete |
| Clear legal-hold | ❌ `Access Denied` — verifier cannot modify legal-hold |
| Bypass retention | ❌ Not permitted for verifier identity |
| Put object (write) | ❌ `Access Denied` — verifier cannot upload |

## Risk acceptance

By the approved *Environment Trust and Evidence Custody Deployment Profile v0.1*, this development-exception custody arrangement requires Architecture, Security, Operations, and Database approval.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Security | | | | |
| Operations | | | | |
| Database | | | | |
