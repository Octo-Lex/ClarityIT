# WP-00 G1 — P1/P2 Schema Capture Report

**Date:** 31 July 2026
**Source:** Production database on Proxmox CT 150 (`root@192.168.3.5`, container `clarityit-postgres-1`)
**Profiler:** `scripts/profile/capture_schema.py` v`2.0.0-p1p2`
**Status:** Captured and compared; awaiting owner approval (see §6).

---

## 1. What was captured

Per the Migration spec §2.2 / §4.3 and the WP-00 G1 requirements:

- **P1:** read-only production schema capture (live `clarityit-postgres-1`).
- **P2:** the same capture from an independently restored production backup (isolated `p2-restore-pg` container, ephemeral, torn down after capture).
- PostgreSQL version and extensions.
- Schema-only dump (`pg_dump --schema-only`) and deterministic schema fingerprint.
- Tables, columns, constraints, indexes, sequences, functions, triggers, views, RLS policies.
- Migration-history state (migration ledger table).
- Roles, memberships, ownership, grants digest, default privileges.
- Required table counts and integrity checks — **no business data or secrets exported.**

## 2. Capture facts

| Item | P1 (production) | P2 (restored) |
|---|---|---|
| PostgreSQL | 16.14 (Alpine) | 16.14 (Alpine) |
| Schemas | `public` | `public` |
| Relations | 65 | 65 |
| Functions | 91 | 91 |
| Fingerprint (sha256) | `aaf09d492e3d561b760338a410567007ec0dcf14bf347d0e0c00469d48f33cf0` | `dff1eb7a62ecd7e3b10e09fdac8eb43652400d0b2357eb530d78348c90ae43a7` |
| Schema dump bytes | 119,707 | 119,707 |
| Schema dump sha256 | `5e9e19b4fb1e9161def563ddf50537aa3e7edb03b773a5ba0203ab6c64770361` | `5de6005367cde9d4fa6a4d6f6272b5da9d11efa4df0689a57b766f67693c1135` |
| Deterministic (re-capture) | ✅ identical fingerprint | — |

> **Note on fingerprint difference:** P1 and P2 fingerprints differ by **one role only**: `clarityit_ro_profile` (the read-only role created for this capture, present in P1's cluster but absent from P2's restored cluster because `pg_dump` does not export cluster-level roles). This is a **benign, documented capture artifact**, not a schema drift. See §3.

## 3. P1 ↔ P2 comparison

`compare` reports `roles_and_grants` as the only differing section. Drilling in:

- **functions:** identical (91 functions, confirmed by canonical sort — the raw comparison's initial false positive was list ordering).
- **roles:** P1 has `clarityit` + `clarityit_ro_profile`; P2 has `clarityit` only. Difference = the capture role.
- **memberships, ownership, grants_sha256, default_privileges:** identical.
- **schema dumps:** byte-identical except for the `\restrict`/`\unrestrict` session token (a pg_dump 16.4 random guard, not schema content).

**Conclusion: P1 and P2 are schema-equivalent.** The production database restores cleanly and its schema is reproducible from backup.

## 4. Migration-history state

The production database has **no migration ledger table** (`schema_migrations`, `platform.schema_revisions`, etc. all absent). This confirms the Migration spec's finding: the deployed schema was produced by the broken `make migrate` psql loop with no durable record of which migrations applied. The 65 relations exist but their provenance is unverifiable from the database itself — which is why this P1 capture (not the migration files) is the upgrade authority.

## 5. Integrity checks

- **Orphan foreign keys:** none (all FKs reference tables within `public`).
- **Invalid constraints:** none (all constraints validated).

## 6. Approval

P1/P2 require Database + Operations + Security approval (Migration spec §2.2).

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | | | |
| Operations | | | | |
| Security | | | | |

**Acceptance condition:** All three sign *accept*. On acceptance, P1/P2 fingerprints are added to the migration runner's source-profile allowlist. Unknown or drifted fingerprints fail closed (G4 preflight).

## 7. Artifacts

```
migrations/profiles/
├── p1-production/
│   ├── manifest.json     # full catalog capture + fingerprint
│   └── schema.sql        # pg_dump --schema-only
├── p2-restored/
│   ├── manifest.json
│   └── schema.sql
├── CAPTURE-REPORT.md     # this file
├── RESTORE-PROOF.md      # A3 restore-rehearsal evidence
└── SHA256SUMS.txt        # artifact checksum manifest
```

## 8. What this capture does NOT do

- It is **not** a data export. Only counts; no row data, no credentials, no PII.
- It does **not** resolve migrations 016/018/029 (that is G2, which consumes P1 as input).
- It does **not** authorize an upgrade. It establishes the approved source profile on the allowlist.
- The CI-only P0 fixture (`migrations/ci/p0/`) is **not** a substitute for P1/P2. P0 is application-test evidence; P1/P2 are production-schema authority.
