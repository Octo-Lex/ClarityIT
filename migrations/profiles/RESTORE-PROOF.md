# WP-00 G1 — A3 Restore Proof (Operational Backup)

**Date:** 1 August 2026
**Profiler:** v`3.1.0-p1p2`
**Status:** ✅ Restore verified — operational backup recovery proven from two isolated restores.

---

## 1. Backup-process diagnosis and repair

**Root cause:** No scheduling existed for the ClarityIT backup script. The `backup-postgres.sh` script was functional but had no cron or systemd timer — backups were run manually, with the most recent from 2026-06-14 (47 days stale).

**Repair:** Installed a systemd timer inside CT 150:
- `clarityit-backup.service` — runs `/opt/clarityit/scripts/backup-postgres.sh`
- `clarityit-backup.timer` — daily at 03:00 UTC (`OnCalendar=*-*-* 03:00:00`), `Persistent=true`
- Timezone: `Etc/UTC`
- Enabled and started; first operational backup produced via the service

### Service/timer configuration evidence

```ini
# /etc/systemd/system/clarityit-backup.service
[Unit]
Description=ClarityIT PostgreSQL operational backup
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/opt/clarityit/scripts/backup-postgres.sh
WorkingDirectory=/opt/clarityit

# /etc/systemd/system/clarityit-backup.timer
[Unit]
Description=ClarityIT PostgreSQL daily operational backup

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

```
$ systemctl list-timers clarityit-backup.timer
NEXT                        LEFT          UNIT                   ACTIVATES
Sat 2026-08-01 03:00:00 UTC 4h 42min left clarityit-backup.timer clarityit-backup.service

$ timedatectl show -p Timezone --value
Etc/UTC
```

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

### Sanitized backup-job log

```
Jul 31 17:36:28 clarityit systemd[1]: Starting clarityit-backup.service...
Jul 31 17:36:28 clarityit backup-postgres.sh[3020531]: === ClarityIT PostgreSQL Backup ===
Jul 31 17:36:28 clarityit backup-postgres.sh[3020531]: Timestamp: 20260731_173628
Jul 31 17:36:28 clarityit backup-postgres.sh[3020531]: Output: /opt/clarityit/backups/postgresql_20260731_173628.sql.gz
Jul 31 17:36:28 clarityit backup-postgres.sh[3020531]: Backup complete: 1.2M
Jul 31 17:36:28 clarityit backup-postgres.sh[3020531]: Old backups rotated (keeping last 30)
Jul 31 17:36:28 clarityit systemd[1]: clarityit-backup.service: Deactivated successfully.
Jul 31 17:36:28 clarityit systemd[1]: Finished clarityit-backup.service...
```

## 3. Restore environment

Two isolated ephemeral containers, separate from production:

| | Restore #1 (P2a) | Restore #2 (P2b) |
|---|---|---|
| Container | `p2a-restore-pg` | `p2b-restore-pg` |
| Image | `postgres@sha256:d845e7f0ac8517b9d9868b6d20379f9688ba3676595e50ca7c0b664964b2a760` (16-alpine) | same |
| Network | `a3-net` (isolated) | `a3-net` (isolated) |
| Disposition | removed after capture | removed after capture |

## 4. Restore transcripts

| | Restore #1 | Restore #2 |
|---|---|---|
| Exit code | 0 | 0 |
| Elapsed | 1160 ms | 1306 ms |
| Output lines | 884 | 884 |
| Errors | none | none |
| Log SHA-256 | `541ba3cbebbaaa97497bb7e4729ae513bb1d43e0470bf431c2e9d0d24ff69c74` | `9c9f5a6454bff50d2110a093233948e4859e128fe52a96cde3843b140363ae3a` |

Full transcripts retained externally: `clarityit-g1-evidence/restore-logs/`.

## 5. Fingerprint verification

| Profile | Fingerprint | Manifest SHA-256 |
|---|---|---|
| P1 (production, v3.1.0) | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` | `0f81cf9369c5139ce680b049981676adc5ff9811037dba866326886579c4d994` |
| P2a (restore #1) | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` | `d32f4b9c4d85a66c7c095adec7b1a11cb1b03271a7916b6134d797535a521ecb` |
| P2b (restore #2) | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` | `db7578616d1acddc74885a5c67e4724cc83c9fd698bb56765deed260afb1c173` |

**P1 == P2a == P2b: MATCH.** Self-consistent, deterministic, repeatable.

## 6. Data-condition checks

| Check | Result |
|---|---|
| Orphan foreign keys | 0 ✅ |
| Invalid constraints | 0 ✅ |
| Table count | 64 ✅ |
| Functions | 91 ✅ |
| Migration ledger | Not present (expected) |

## 7. Approval (A3)

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | ☐ accept ☐ block | | |
| Operations | | ☐ accept ☐ block | | |

## 8. Disposition

Both restore containers and the isolated network were removed after capture. The backup artifact, manifests, and restore logs are retained in the external evidence store only (not in the repo).
