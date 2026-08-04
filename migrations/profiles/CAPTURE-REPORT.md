# WP-00 G1 — P1/P2 Schema Capture Report

**Date:** 1 August 2026
**Profiler:** `scripts/profile/capture_schema.py` v`3.2.0-p1p2` (13 unit tests passing)
**Source:** Production database on Proxmox CT 150 (`clarityit-postgres-1`)
**Evidence store:** External (outside this repo) — see §7. Only digests and references are in-repo.
**Status:** Captured and compared; awaiting owner approval (§6).

---

## 1. What was captured

Per Migration spec §2.2 / §4.3 and WP-00 G1:

- **P1:** read-only production schema capture.
- **P2:** the same capture from an independently restored **operational backup** (produced by the repaired systemd job, not an ad hoc dump). Two isolated restores, proving repeatability.
- PostgreSQL version + extensions, deterministic fingerprint.
- Tables, columns, constraints, indexes, **sequences (with properties)**, functions, triggers, views, **RLS policies (with commands, roles, enabled/forced flags)**.
- Migration-history state, roles, memberships, **ownership (reported, excluded from fingerprint)**, **comprehensive grants (PUBLIC + all object classes via aclexplode)**, default privileges.
- Table counts and integrity checks — **no business data or secrets.**

## 2. Capture facts

| Item | Value |
|---|---|
| PostgreSQL | 16.14 (Alpine) |
| Schema fingerprint (P1 = P2a = P2b) | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |
| Relations | 65 |
| Functions | 91 |
| Schemas | `public` |
| Profiler version | `3.2.0-p1p2` |
| Self-consistent fingerprint | ✅ (recomputed == stored) |
| Deterministic | ✅ (re-capture identical) |
| Repeatability (P2a == P2b) | ✅ (two independent restores from operational backup, identical fingerprint) |

**P1 == P2: MATCH.** The production schema restores cleanly and is reproducible from the operational backup.

## 3. Fingerprint properties (v3.2.0)

The v3.2.0 profiler excludes ALL version metadata from the fingerprint:
- `pg_version_string`, `server_version`, AND `server_version_num` are excluded.
- `fingerprint_sha256` is excluded from its own computation (self-consistent).
- Overloaded functions are totally ordered by `(schema, name, args)`.
- Ownership is excluded (spec §4.3) → reported in manifest, not hashed.
- The fingerprint is purely schema — stable across PG patch versions.
- Proven by 13 unit tests in `scripts/profile/test_capture_schema.py`.

## 4. Findings

1. **Production has NO migration ledger table.** Schema provenance is unverifiable from the DB — confirming the Migration spec's premise that the live schema (this capture) is the upgrade authority.
2. **Operational backup process was broken** (no scheduling — script present but no cron/timer). Repaired: installed `clarityit-backup.service` + `.timer` (daily 03:00 UTC). The operational backup `opbak-20260731-173628` was produced by the repaired systemd job and successfully restored.
3. **No orphan FKs, no invalid constraints.** Schema is structurally sound.
4. **No RLS policies enabled** in production.

## 5. P1↔P2 comparison

Both restores from the operational backup produce fingerprint `89b7792d…`, identical to P1. The comparison reports **MATCH — identical canonical schema**.

## 6. Approval (A2)

P1/P2 require Database + Security approval (Migration spec §2.2).

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | ☐ accept ☐ block | | |
| Security | | ☐ accept ☐ block | | |

## 7. Evidence storage (external)

Per WP-00 evidence policy, sensitive P1/P2 bytes remain **outside the repository**. The raw manifests, backup artifact, and restore logs are stored externally with immutable hash references:

| Artifact | External path | SHA-256 |
|---|---|---|
| P1 manifest (v3.2.0) | `clarityit-g1-evidence/p1-production/manifest.json` | `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` |
| P2a manifest | `clarityit-g1-evidence/p2-restored/manifest-p2a.json` | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |
| P2b manifest | `clarityit-g1-evidence/p2-restored/manifest-p2b.json` | `db7578616d1acddc74885a5c67e4724cc83c9fd698bb56765deed260afb1c173` |
| Operational backup | `postgresql_20260731_173628.sql.gz` (on CT 150) | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| Restore log #1 | `clarityit-g1-evidence/restore-logs/p2a-restore.log` | `541ba3cbebbaaa97497bb7e4729ae513bb1d43e0470bf431c2e9d0d24ff69c74` |
| Restore log #2 | `clarityit-g1-evidence/restore-logs/p2b-restore.log` | `9c9f5a6454bff50d2110a093233948e4859e128fe52a96cde3843b140363ae3a` |
| Sanitized backup-job log | `clarityit-g1-evidence/backup-job/job-log-sanitized.txt` | `a43e20e30db13e779c18d5e75e3662970a629003033657e92c14b8100eb9a7c8` |
| Service unit configuration | `clarityit-g1-evidence/backup-job/service.conf` | `ecfa4f6c54160917c831eb53fe374392c2d7961eb69c70c51d3467e115fbda8f` |
| Timer unit configuration | `clarityit-g1-evidence/backup-job/timer.conf` | `56c4f90534281cfff2f076e7151cdef57ebab40575aa448f7ba67334a80580ec` |
| `systemctl cat` capture | `clarityit-g1-evidence/backup-job/systemctl-cat.txt` | `99d7378cbacc8c882b74d6baf2002b2db5133159fb90c17719e79f7334b5696d` |
| `systemctl list-timers` capture | `clarityit-g1-evidence/backup-job/systemctl-list-timers.txt` | `18f5e770160b0bf4ea783ceda3efdf8d20e7ea428e6480982e058a753a098b89` |
| `timedatectl` capture | `clarityit-g1-evidence/backup-job/timedatectl.txt` | `f0dcac7b1d721d2f68937a71f0229b4c4f88564fd711339951528889913cd85d` |

The repo contains **only this report, the capture script, the P3 fixture, and the unit tests** — no raw manifests or dumps.

## 8. What this does NOT do

- Not a data export (counts only; no row data, credentials, or PII).
- Does not resolve 016/018/029 (that is G2, consuming P1).
- Does not authorize an upgrade (establishes the allowlist profile only).
- The CI-only P0 fixture (`migrations/ci/p0/`) is **not** a substitute for P1/P2.
