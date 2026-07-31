# WP-00 G1 — A3 Restore Proof (Operational Backup)

**Date:** 31 July 2026
**Status:** ✅ Restore verified — operational backup recovery proven.

---

## 1. Backup-process diagnosis and repair

**Root cause:** No scheduling existed for the ClarityIT backup script. The `backup-postgres.sh` script was functional but had no cron or systemd timer — backups were run manually, with the most recent from 2026-06-14 (47 days stale).

**Repair:** Installed a systemd timer inside CT 150:
- `clarityit-backup.service` — runs `/opt/clarityit/scripts/backup-postgres.sh`
- `clarityit-backup.timer` — daily at 03:00 UTC, `Persistent=true`
- Enabled and started; first operational backup produced via the service

## 2. Operational backup reference

| Item | Value |
|---|---|
| Backup ID | `opbak-20260731-173628` |
| File | `postgresql_20260731_173628.sql.gz` |
| Timestamp | 2026-07-31T17:36:28Z |
| Produced by | `clarityit-backup.service` (systemd, not ad hoc) |
| Size | 1,228,736 bytes |
| SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` |
| PostgreSQL | 16.14 |
| Format | `pg_dump --clean --if-exists` (gzipped) |

## 3. Restore environment

Two isolated ephemeral containers, separate from production:

| | Restore #1 (P2a) | Restore #2 (P2b) |
|---|---|---|
| Container | `p2a-restore-pg` | `p2b-restore-pg` |
| Image | `postgres:16-alpine` | `postgres:16-alpine` |
| Network | `a3-restore-net` (isolated) | `a3-restore-net` (isolated) |
| Disposition | removed after capture | removed after capture |

## 4. Restore transcripts

| | Restore #1 | Restore #2 |
|---|---|---|
| Exit code | 0 | 0 |
| Elapsed | 1155 ms | 1199 ms |
| Output lines | 884 | 884 |
| Errors | none | none |

Full transcripts retained externally: `clarityit-g1-evidence/restore-logs/p2a-restore.log`, `p2b-restore.log`.

## 5. Fingerprint verification

| Profile | Fingerprint | Source |
|---|---|---|
| P1 (production, recaptured) | `100cb5f30a45b77728da369291eae60a7538a7545c521308548d8b2570c48dab` | Current profiler, production DB |
| P2a (restore #1) | `100cb5f30a45b77728da369291eae60a7538a7545c521308548d8b2570c48dab` | Same profiler, restored from opbak-20260731 |
| P2b (restore #2) | `100cb5f30a45b77728da369291eae60a7538a7545c521308548d8b2570c48dab` | Same profiler, restored from opbak-20260731 |

**P1 == P2a == P2b: MATCH.** Self-consistent, deterministic, repeatable.

> **Note on P1 fingerprint change:** P1 was recaptured with the current profiler (v3.0.0-p1p2, which excludes all version metadata from the fingerprint). The earlier P1 fingerprint (`32f7a06e…`) was produced by an intermediate profiler version that still included `server_version`/`server_version_num` in the fingerprinted settings. The current fingerprint (`100cb5f3…`) is the authoritative one — purely schema, version-independent.

## 6. Data-condition checks

| Check | Result |
|---|---|
| Orphan foreign keys | 0 ✅ |
| Invalid constraints | 0 ✅ |
| Table count | 64 ✅ |
| Functions | 91 ✅ |
| Migration ledger | Not present (expected — no ledger in production) |

## 7. Approval (A3)

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | ☐ accept ☐ block | | |
| Operations | | ☐ accept ☐ block | | |

## 8. Disposition

Both restore containers and the isolated network were removed after capture. The backup artifact, manifests, and restore logs are retained in the external evidence store only (not in the repo).
