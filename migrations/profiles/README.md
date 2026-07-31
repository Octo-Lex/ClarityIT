# ClarityIT WP-00 — Schema Profiles (P1/P2)

Production schema captures for the v2 migration baseline. Per the v1-to-v2
Compatibility & Migration Specification §2.2, the **approved production schema
profile (P1/P2)** — not migration filenames — is the authority for upgrading an
existing database.

## Contents

| Path | What | Authority |
|---|---|---|
| `p1-production/manifest.json` | Full catalog capture of production DB + fingerprint | P1 (production source) |
| `p1-production/schema.sql` | `pg_dump --schema-only` of production | P1 |
| `p2-restored/manifest.json` | Same capture from restored backup | P2 (rehearsal/rollback evidence) |
| `p2-restored/schema.sql` | `pg_dump --schema-only` of restored | P2 |
| `CAPTURE-REPORT.md` | P1/P2 comparison, facts, approval (A2) | G1 decision input |
| `RESTORE-PROOF.md` | Restore-rehearsal evidence (A3) | G1 decision input |
| `SHA256SUMS.txt` | Artifact checksum manifest | integrity |

## Key results

- **P1 fingerprint:** `aaf09d492e3d561b760338a410567007ec0dcf14bf347d0e0c00469d48f33cf0`
- **P2 fingerprint:** `dff1eb7a62ecd7e3b10e09fdac8eb43652400d0b2357eb530d78348c90ae43a7`
- **P1 ↔ P2 schema-equivalent:** YES. Sole difference is the `clarityit_ro_profile` capture role (cluster-level, not in DB dumps).
- **Production has no migration ledger** — schema provenance unverifiable from the DB; this capture is the authority.

## Capture tooling

`scripts/profile/capture_schema.py` — read-only, secret-excluding, deterministic.
Usage and invariants documented in its header.

## Relationship to P0

The CI-only P0 fixture (`migrations/ci/p0/`) provisions the backend test DB.
It is **not** a substitute for P1/P2. P0 is application-test evidence;
P1/P2 are production-schema authority and gate G2/G3.
