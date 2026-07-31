# ClarityIT WP-00 — Schema Profiles (P1/P2)

Production schema captures for the v2 migration baseline (G1). Per the Migration
spec §2.2, the **approved production schema profile (P1/P2)** — not migration
filenames — is the authority for upgrading an existing database.

## Evidence policy

**Raw P1/P2 manifests, schema dumps, and restore logs are stored externally**
(outside this repository), per WP-00's sensitive-bytes-outside-repo rule. This
directory contains only:
- sanitized reports/decisions (digests + references, no raw bytes),
- the capture tooling (`scripts/profile/`),
- the P3 sanitized CI fixture (when produced).

External evidence location and hashes are documented in `CAPTURE-REPORT.md §7`.

## Contents

| Path | What | In-repo? |
|---|---|---|
| `CAPTURE-REPORT.md` | P1/P2 facts, comparison, approval (A2) | yes (decisions only) |
| `RESTORE-PROOF.md` | Restore-rehearsal evidence (A3) | yes (decisions only) |
| `p3/` | Sanitized CI fixture (P3) | yes (synthetic, no prod data) |
| P1/P2 manifests + dumps | raw captures | **no** — external store |

## Key results (v3 profiler)

- **P1 = P2a = P2b fingerprint:** `32f7a06eb7ae8c8f96547bd494f66b26468fba103dbde50fe08c5bf9d0e1402c`
- **Self-consistent** (recomputed == stored), **deterministic**, **repeatable** (two restores).
- 65 relations, 91 functions, PostgreSQL 16.14.
- Production has **no migration ledger** — this capture is the provenance authority.
- Scheduled operational backup is **stale** (missing 16 tables since 2026-06-14).

## Capture tooling

`scripts/profile/capture_schema.py` (v3.0.0-p1p2) — read-only, secret-excluding,
deterministic. 12 unit tests in `test_capture_schema.py` prove self-consistency,
determinism, P1==P2, ownership-exclusion, and sensitivity.

## Relationship to P0

The CI-only P0 fixture (`migrations/ci/p0/`) provisions the backend test DB.
It is **not** a substitute for P1/P2 — P0 is application-test evidence;
P1/P2 are production-schema authority and gate G2/G3.
