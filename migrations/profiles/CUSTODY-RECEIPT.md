# WP-00 G1 — Evidence Custody Receipt

**Date:** 1 August 2026
**Receipt author:** `evidence-writer` (MinIO IAM principal)
**Bound commit:** `22050b8`
**Bound backup:** `opbak-20260731-173628`
**Bound P1/P2 fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
**Bound P3 golden:** `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`

This receipt is a **repository-side record** of the externally stored custody
manifest. It does not alter the manifest's bytes.

## Sealed custody manifest v3 (current)

| Property | Value |
|---|---|
| Object key | `custody-manifest.md` |
| Bucket | `clarityit-g1-evidence` |
| Version ID | `1eaa75e6-e857-4cba-8b5c-af8cfe6b3f7b` |
| Size | 6,675 bytes |
| SHA-256 | `2f6aa8f192ebe08268b5d4833d46b5170e4f6464f1c77fdbeb1b13e22cbec1ee` |
| Uploaded | 2026-08-01T04:23:26Z |
| Principal | `evidence-writer` |
| Encryption | SSE-KMS (key: `clarityit-evidence-key`) |
| Retention | GOVERNANCE, 2555 days (expiry 2033-07-31) |
| Legal hold | ON |
| Verifier result | ✅ Full SHA-256 match via `evidence-verifier` (size=6675, sha256=2f6aa8f1…) |

## Prior manifest versions (immutable history)

| Version | Version ID | SHA-256 | Superseded by |
|---|---|---|---|
| v1 | `7cfcbb0f-4df2-44af-8aec-90de39a5a4c7` | (pre-seal construction) | v2 |
| v2 | `11e6ab63-bb41-46dc-aff2-4ff50f85b761` | `36eac69fd1c68204eb2450b0957ace64db51f8f106a754f340e8fb081ea021d0` | v3 |

## Evidence inventory (19 objects total)

### Original artifacts (12)
All SSE-KMS encrypted, GOVERNANCE retention 2555d, legal-hold ON, SHA-256 verified via `evidence-verifier`.

| Key | Version ID | Size | SHA-256 |
|---|---|---|---|
| `manifest.json` | `1fd353ec-2258-4cd5-af57-fdbafc2c7f3a` | 250,239 | `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` |
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

### Control evidence (6)
All SSE-KMS encrypted, GOVERNANCE retention 2555d, legal-hold ON, SHA-256 verified via `evidence-verifier`.

| Key | Version ID | Size |
|---|---|---|
| `controls/per-object-metadata.txt` | `ba989131-9535-4407-b2b6-f6ccb0b4ae53` | 7,805 |
| `controls/denial-tests.txt` | `5cbd0ade-1815-474c-8c34-73dd5d1f3cfb` | 2,822 |
| `controls/bucket-config.txt` | `0b496a67-435d-4a0b-ae3e-3d5f38283b65` | 742 |
| `controls/recovery-tests.txt` | `35311f6a-fdbc-4725-aca2-ad60d4890dd1` | 1,112 |
| `controls/audit-evidence.txt` | `26b52737-78d9-4238-a0e0-a75576b123e5` | 805 |
| `controls/stat-custody-manifest.json` | `2c807abd-cc73-47cd-a9f6-46bc7634985b` | 390 |

### Custody manifest (1)
Version ID `1eaa75e6-e857-4cba-8b5c-af8cfe6b3f7b`, SHA-256 `2f6aa8f1…`, size 6,675 bytes.

## Verification summary

- All 12 original artifacts: SHA-256 verified ✅
- All 6 control evidence artifacts: SHA-256 verified ✅
- Manifest v3: full SHA-256 verified via verifier ✅ (size=6675, sha256=2f6aa8f1…)
- Writer denial tests: 5/5 denied ✅
- Verifier denial tests: 6/6 denied ✅
- Recovery tests: 4/4 passed ✅

## Approvals required

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Architecture | | | | |
| Security | | | | |
| Operations | | | | |
| Database | | | | |
