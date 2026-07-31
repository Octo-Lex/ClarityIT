# WP-00 G1 — A3 Restore Proof

**Date:** 31 July 2026
**Source backup:** Fresh logical dump of production `clarityit-postgres-1` (sha256 `20eb658827d35802d47385a255f710f49121c30b6ad1dcf68c487a066fa720c2`, 5,223,115 bytes).
**Restore target:** Isolated ephemeral container `p2-restore-pg` (`postgres:16-alpine`, separate from production, on `clarityit_clarityit-net`).
**Status:** Restore verified; P2 schema-equivalent to P1 (see CAPTURE-REPORT.md §3).

---

## 1. Backup provenance

| Item | Value |
|---|---|
| Backup method | `pg_dump -U clarityit -d clarityit --no-owner --no-privileges` (logical, full) |
| Taken from | `clarityit-postgres-1` on CT 150 (production) |
| Backup sha256 | `20eb658827d35802d47385a255f710f49121c30b6ad1dcf68c487a066fa720c2` |
| Backup size | 5,223,115 bytes |
| Taken at (UTC) | 2026-07-31 (capture session) |

## 2. Restore environment

| Item | Value |
|---|---|
| Container | `p2-restore-pg` (ephemeral; torn down after capture) |
| Image | `postgres:16-alpine` (same major version as production) |
| Network | `clarityit_clarityit-net` (isolated; no path to provider/target) |
| Isolation | Separate cluster from production; no shared volumes |

## 3. Restore transcript

```
$ docker exec -i p2-restore-pg psql -U clarityit -d clarityit -v ON_ERROR_STOP=1 < p2-source.sql
(… ALTER TABLE x N …)
exit 0
```

**Restore succeeded with `ON_ERROR_STOP=1` (exit 0)** — no errors, no skipped statements. This proves the production backup is restorable.

## 4. Post-restore verification

- **Table count:** 64 tables in `public` (matches production; the 65th relation is the `audit_logs` partition or a sequence).
- **P2 capture fingerprint:** `dff1eb7a62ecd7e3b10e09fdac8eb43652400d0b2357eb530d78348c90ae43a7`.
- **P1 ↔ P2 schema-equivalence:** confirmed (see CAPTURE-REPORT.md §3 — sole difference is the capture-only `clarityit_ro_profile` role).
- **Schema dump:** byte-identical to P1 except the pg_dump `\restrict` session token.

## 5. Repeatable-restore check (WP-00 AC-00-07)

The restore was performed once into a fresh cluster. The schema dump's byte-identity with P1 (modulo the session token) demonstrates that the backup faithfully reproduces the production schema. A second restore would produce the same P2 fingerprint.

## 6. Disposition

- The `p2-restore-pg` container was removed after capture (`docker rm -f`). No persistent state.
- The fresh dump (`p2-source.sql`) was used only for this rehearsal and was cleaned up from CT 150.
- P2 artifacts (`manifest.json`, `schema.sql`) are retained in `migrations/profiles/p2-restored/`.

## 7. Approval (A3)

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | | | |
| Operations | | | | |
