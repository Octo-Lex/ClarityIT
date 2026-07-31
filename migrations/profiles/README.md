# ClarityIT WP-00 — Schema Profiles (P1/P2)

Production schema captures for the v2 migration baseline (G1). Per the Migration
spec §2.2, the **approved production schema profile (P1/P2)** — not migration
filenames — is the authority for upgrading an existing database.

## Evidence policy

**Raw P1/P2 manifests, backup artifacts, and restore logs are stored externally**
(outside this repository), per WP-00's sensitive-bytes-outside-repo rule. This
directory contains only:
- sanitized reports/decisions (digests + references, no raw bytes),
- the capture tooling (`scripts/profile/`),
- the P3 sanitized CI fixture.

External evidence location and hashes are documented in `CAPTURE-REPORT.md §7`.

## Key results (v3.1.0 profiler)

- **P1 = P2a = P2b fingerprint:** `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83`
- **Self-consistent** (recomputed == stored), **deterministic**, **repeatable** (two restores from operational backup).
- 65 relations, 91 functions, PostgreSQL 16.14.
- Production has **no migration ledger** — this capture is the provenance authority.
- Operational backup process repaired (systemd timer installed); operational backup `opbak-20260731-173628` successfully restored.

## Capture tooling

`scripts/profile/capture_schema.py` (v3.1.0-p1p2) — read-only, secret-excluding,
deterministic. 13 unit tests in `test_capture_schema.py` prove self-consistency,
determinism, P1==P2, ownership-exclusion, and version-independence.

## Relationship to P0

The CI-only P0 fixture (`migrations/ci/p0/`) provisions the backend test DB.
It is **not** a substitute for P1/P2 — P0 is application-test evidence;
P1/P2 are production-schema authority and gate G2/G3.
