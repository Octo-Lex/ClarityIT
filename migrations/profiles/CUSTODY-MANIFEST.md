# WP-00 G1 — Evidence Custody Manifest

**Date:** 1 August 2026
**Commit:** `22050b8`
**Backup:** `opbak-20260731-173628`
**P1/P2 fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
**P3 golden:** `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`

## Storage configuration

| Property | Value |
|---|---|
| Platform | MinIO (S3-compatible) on CT 150 |
| Bucket | `clarityit-g1-evidence` |
| Versioning | Enabled (all uploads create immutable versions) |
| Object lock | Enabled (bucket created with `--with-lock`) |
| Retention mode | GOVERNANCE, 2555 days (~7 years) |
| Legal hold | ON for all evidence objects |
| Encryption | SSE-S3 not available (no KMS configured on this MinIO) — **limitation documented** |
| Audit logging | Configured (audit_webhook) |
| Access | `clarityit` root credentials (least-privilege IAM not yet configured) — **limitation documented** |

## Evidence objects (11 artifacts, 12th = P1 manifest under key `manifest.json`)

| Key | Version ID | Size | SHA-256 (local) | Legal Hold |
|---|---|---|---|---|
| `manifest.json` (P1) | `fcfec6e9-4db0-4e97-b781-9f67868130de` | 250,239 | `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` | ON |
| `manifest-p2a.json` | `b50045f3-b44a-4b26-b3cd-71a75ad6635e` | 250,235 | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` | ON |
| `manifest-p2b.json` | `68b0f2cd-67a6-4e5f-a697-05520bf2faaa` | 250,235 | `db7578616d1acddc74885a5c67e4724cc83c9fd698bb56765deed260afb1c173` | ON |
| `p2a-restore.log` | `d3490656-d741-4d65-9328-b9de2921f7e0` | 10,397 | `541ba3cbebbaaa97497bb7e4729ae513bb1d43e0470bf431c2e9d0d24ff69c74` | ON |
| `p2b-restore.log` | `e9470d4c-fabf-4d36-9cf8-bff6aa25d332` | 10,397 | `9c9f5a6454bff50d2110a093233948e4859e128fe52a96cde3843b140363ae3a` | ON |
| `job-log-sanitized.txt` | `0235e5f2-3b7a-40a4-8d7d-e63ea317ebe3` | 703 | `a43e20e30db13e779c18d5e75e3662970a629003033657e92c14b8100eb9a7c8` | ON |
| `service.conf` | `7a990ac6-3bbd-407e-a9f4-de6476117854` | 212 | `ecfa4f6c54160917c831eb53fe374392c2d7961eb69c70c51d3467e115fbda8f` | ON |
| `timer.conf` | `b0613398-b2f5-4b25-bb47-30ad1c806b34` | 150 | `56c4f90534281cfff2f076e7151cdef57ebab40575aa448f7ba67334a80580ec` | ON |
| `systemctl-cat.txt` | `1c4245fa-cf1a-4545-bc97-e4744e56d25e` | 455 | `99d7378cbacc8c882b74d6baf2002b2db5133159fb90c17719e79f7334b5696d` | ON |
| `systemctl-list-timers.txt` | `e554a610-dc17-4f97-93ee-7fadc8e806bb` | 165 | `18f5e770160b0bf4ea783ceda3efdf8d20e7ea428e6480982e058a753a098b89` | ON |
| `timedatectl.txt` | `ecd71e7f-1bea-4e62-865f-7ff3333460e4` | 8 | `f0dcac7b1d721d2f68937a71f0229b4c4f88564fd711339951528889913cd85d` | ON |

## Operational backup artifact

| Property | Value |
|---|---|
| Key | `postgresql_20260731_173628.sql.gz` |
| Location | `/opt/clarityit/backups/` on CT 150 (not in MinIO — too large for `/tmp` transfer) |
| SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| Size | 1,228,736 bytes |
| Retention | Managed by `clarityit-backup.timer` (30-backup rotation) |

## Immutability verification

- **Versioning:** Every upload creates a unique version ID. Overwriting a key creates a NEW version (original preserved).
- **Legal hold ON:** Prevents deletion of the specific version.
- **Retention GOVERNANCE 2555d:** Prevents deletion until ~August 2033 (admin bypass possible but logged).
- **Delete marker test:** Deleting a key creates a delete marker (version preserved).

## Limitations (must be resolved before G1 acceptance)

1. **No KMS encryption** — MinIO on CT 150 lacks KMS configuration. SSE-S3 encryption is unavailable. Objects are stored unencrypted at rest (MinIO disk encryption is the container's responsibility).
2. **Single-host storage** — MinIO runs on CT 150 (same host as production). Not independently durable against host failure.
3. **Root credentials** — Using `clarityit` root credentials, not a dedicated least-privilege IAM identity.
4. **Operational backup not in MinIO** — Too large for the `/tmp` transfer path; stored on CT 150's `/opt/clarityit/backups/` with filesystem-level rotation only.

These limitations mean the custody is **better than a local directory but not yet meeting the full spec**. G1 acceptance requires Database, Operations, and Security to approve these custody limitations or require a dedicated evidence-storage target.
