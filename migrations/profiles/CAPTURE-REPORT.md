# WP-00 G1 — P1/P2 Schema Capture Report

**Date:** 31 July 2026
**Profiler:** `scripts/profile/capture_schema.py` v`3.0.0-p1p2` (12 unit tests passing)
**Source:** Production database on Proxmox CT 150 (`clarityit-postgres-1`)
**Evidence store:** External (outside this repo) — see §7. Only digests and references are in-repo.
**Status:** Captured and compared; awaiting owner approval (§6).

---

## 1. What was captured

Per Migration spec §2.2 / §4.3 and WP-00 G1:

- **P1:** read-only production schema capture.
- **P2:** the same capture from an independently restored current-production dump (two isolated restores, proving repeatability).
- PostgreSQL version + extensions, schema-only dump, deterministic fingerprint.
- Tables, columns, constraints, indexes, **sequences (with properties)**, functions, triggers, views, **RLS policies (with commands, roles, enabled/forced flags)**.
- Migration-history state, roles, memberships, **ownership (reported, excluded from fingerprint)**, **comprehensive grants (PUBLIC + all object classes via aclexplode)**, default privileges.
- Table counts and integrity checks — **no business data or secrets.**

## 2. Capture facts

| Item | Value |
|---|---|
| PostgreSQL | 16.14 (Alpine) |
| Schema fingerprint (P1 = P2a = P2b) | `32f7a06eb7ae8c8f96547bd494f66b26468fba103dbde50fe08c5bf9d0e1402c` |
| Relations | 65 |
| Functions | 91 |
| Schemas | `public` |
| Self-consistent fingerprint | ✅ (recomputed == stored) |
| Deterministic | ✅ (re-capture identical) |
| Repeatability (P2a == P2b) | ✅ (two independent restores, identical fingerprint) |

**P1 == P2: MATCH.** The production schema restores cleanly and is reproducible.

## 3. Fingerprint properties (corrected from v2)

The v3 profiler fixes the v2 blockers:
- `fingerprint_sha256` is excluded from its own computation → **self-consistent**.
- Overloaded functions are totally ordered by `(schema, name, args)` → **no false diffs**.
- Ownership is excluded (spec §4.3) → reported in manifest, not hashed.
- `pg_version_string` (build-specific compiler/musl label) is excluded from the fingerprint while `server_version_num` (major-version detection) remains fingerprinted.
- Proven by 13 unit tests in `scripts/profile/test_capture_schema.py`.

## 4. Findings

1. **Production has NO migration ledger table.** Schema provenance is unverifiable from the DB — confirming the Migration spec's premise that the live schema (this capture) is the upgrade authority.
2. **The scheduled operational backup is stale — a G1 blocker.** `postgresql_20260614_083025.sql.gz` (the most recent scheduled backup) is **missing 16 tables and 5 functions** added since 2026-06-14 (knowledge/artifact/webauthn/evaluation features). It cannot serve as rollback evidence for the current schema. P2 was captured from a fresh dump (proving current-state restorability), but the Migration spec requires recovery from a *current operational backup*. The backup process must be repaired and the A3 drill repeated before G1 can pass.
3. **No orphan FKs, no invalid constraints.** Schema is structurally sound.
4. **No RLS policies enabled** in production (`rls_state` empty).

## 5. P1↔P2 comparison

Both restores from the fresh current-production dump produce fingerprint `32f7a06e…`, identical to P1. The comparison reports **MATCH — identical canonical schema**.

## 6. Approval (A2)

P1/P2 require Database + Security approval (Migration spec §2.2).

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | | ☐ accept ☐ block | | |
| Security | | ☐ accept ☐ block | | |

**Acceptance condition:** Both sign *accept*. On acceptance, the P1 fingerprint enters the migration runner's source-profile allowlist. Unknown/drifted fingerprints fail closed (G4).

## 7. Evidence storage (external)

Per WP-00 evidence policy, sensitive P1/P2 bytes remain **outside the repository**. The raw manifests, schema dumps, and restore logs are stored externally with immutable hash references:

| Artifact | External path | SHA-256 |
|---|---|---|
| P1 manifest | `clarityit-g1-evidence/p1-production/manifest.json` | `261f1fa47227767ca7abf1d41c0e9b791649459f7eb823f8f9474bcf371d5f35` |
| P2 manifest | `clarityit-g1-evidence/p2-restored/manifest.json` | `59398b8f66e04a92a7a8af7f53e6cf2a54843bd5e559be311781ba518c7b576f` |
| P2 source dump | `clarityit-g1-evidence/p2-restored/source-dump.sql` | `a21085637ca165ad50c27d1cab109a22d0580708f2b797f4e41b3f6df7e9013f` |
| Restore logs | `clarityit-g1-evidence/restore-logs/` | (in EVIDENCE-SHA256SUMS.txt) |
| Evidence manifest | `clarityit-g1-evidence/EVIDENCE-SHA256SUMS.txt` | (self-referential) |

The repo contains **only this report, the capture script, and the unit tests** — no raw manifests or dumps.

## 8. What this does NOT do

- Not a data export (counts only; no row data, credentials, or PII).
- Does not resolve 016/018/029 (that is G2, consuming P1).
- Does not authorize an upgrade (establishes the allowlist profile only).
- The CI-only P0 fixture (`migrations/ci/p0/`) is **not** a substitute for P1/P2.
