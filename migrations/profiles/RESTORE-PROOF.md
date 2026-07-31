# WP-00 G1 — A3 Restore Proof

**Date:** 31 July 2026
**Status:** Restore verified with two isolated restores; P2 schema-equivalent to P1.

---

## 1. Backup source

The most recent **scheduled operational backup** (`postgresql_20260614_083025.sql.gz`, sha256 `a76b22a711c865aab534873ab874ccc1c0e02a4945e5f56d6319b5acaf623d78`) was found to be **stale** — it predates 16 tables and 5 functions added to production since 2026-06-14 (see CAPTURE-REPORT.md §4). It cannot serve as rollback evidence for the current schema.

P2 was therefore captured from a **fresh dump of the current production state**:

| Item | Value |
|---|---|
| Method | `pg_dump -U clarityit -d clarityit --no-owner --no-privileges` (logical, full) |
| Source | `clarityit-postgres-1` on CT 150 (production) |
| sha256 | `a21085637ca165ad50c27d1cab109a22d0580708f2b797f4e41b3f6df7e9013f` |
| Size | 5,223,115 bytes |
| Taken at (UTC) | 2026-07-31 (capture session) |

> **Action item:** The scheduled backup cadence must be tightened so operational backups track production. This is recorded as a finding, not a blocker for G1 (the fresh dump proves current-state restorability).

## 2. Restore environment

Two isolated ephemeral containers, separate from production:

| | Restore #1 (p2a) | Restore #2 (p2b) |
|---|---|---|
| Container | `p2a-restore-pg` | `p2b-restore-pg` |
| Image | `postgres:16-alpine` | `postgres:16-alpine` |
| Network | `clarityit_clarityit-net` (isolated) | `clarityit_clarityit-net` (isolated) |
| Disposition | removed after capture | removed after capture |

## 3. Restore transcripts (both restores)

Both restores applied the dump with `ON_ERROR_STOP=1`:

| | Restore #1 | Restore #2 |
|---|---|---|
| Exit code | 0 | 0 |
| Elapsed | 1s | 1s |
| Output lines | 451 | 451 |
| Errors | none | none |

Full transcripts retained externally: `clarityit-g1-evidence/restore-logs/p2a-restore.log`, `p2b-restore.log` (hashes in EVIDENCE-SHA256SUMS.txt).

## 4. Post-restore verification

- **P2a fingerprint:** `32f7a06eb7ae8c8f96547bd494f66b26468fba103dbde50fe08c5bf9d0e1402c`
- **P2b fingerprint:** `32f7a06eb7ae8c8f96547bd494f66b26468fba103dbde50fe08c5bf9d0e1402c`
- **P1 (production) fingerprint:** `32f7a06eb7ae8c8f96547bd494f66b26468fba103dbde50fe08c5bf9d0e1402c`

**P1 == P2a == P2b: all three identical.** This proves:
1. The current production dump restores cleanly.
2. The restore is **repeatable** (two independent restores produce identical schema).
3. The restored schema matches production.

## 5. Approval (A3)

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | ☐ accept ☐ block | | |
| Operations | | ☐ accept ☐ block | | |

## 6. Disposition

Both restore containers were removed after capture (`docker rm -f`). No persistent state. The fresh dump and transcripts are retained in the external evidence store only (not in the repo).
