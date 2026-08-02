# G3 Contract — Reconciled PostgreSQL Baseline

## Status

Implementation contract for WP-00 G3. Approval is recorded separately in `G3-APPROVALS.md`.

## Frozen inputs

| Input | Frozen value |
|---|---|
| Signed G2 commit | `f04f94faad0105d1c3274e9c7974d44f936a0d28` |
| Product manifest | `migrations/profiles/g2/TARGET-SCHEMA-MANIFEST.json` |
| Product-manifest Git blob | `1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63` |
| Product-manifest size | 284,064 bytes, LF |
| PostgreSQL major | 16 |
| Database name | `clarityit` |
| Legacy migrations | `001–040`, byte-immutable |

The G2 manifest, its detached checksum, its machine-readable receipt identity, and its human signatures are read-only G3 inputs. Any drift blocks generation.

## Equivalence target

`FRESH-EQUIV PASS` has two simultaneous requirements:

1. Two independent fresh PostgreSQL 16 installations using `POSTGRES_DB=clarityit` produce byte-identical canonical profiler fingerprints to each other.
2. Each installation completely conforms to the signed G2 product manifest across 64 tables, 10 application functions, columns, primary/unique/foreign/check constraints, indexes, triggers, the sequence, ownership, per-object grants, five role flags, and all PostgreSQL 16 membership options.

The four `platform` migration-control tables and their supporting constraints, indexes, functions, triggers, ownership, and ACL boundary are validated separately against `CONTROL-SCHEMA-MANIFEST.json`.

The fresh installations are not required to match the P3 golden fingerprint. P3 is a sanitized legacy source with a different operational role posture.

## Installation order

An administrator applies exactly these G3 artifacts, in order:

1. `migrations/v2/bootstrap/0000_roles.sql`
2. `migrations/v2/bootstrap/0000_platform.sql`
3. `migrations/v2/baseline/0001_reconciled.sql`
4. `migrations/v2/baseline/0001_seed.sql`

Legacy migrations under `migrations/legacy/v1/001-040/` are provenance only. They are never part of the supported installation sequence.

## Identity model

G3 keeps three identities distinct:

- Product identity: the unchanged signed G2 manifest blob.
- Control identity: the SHA-256 of the generated control-schema manifest.
- Composite installation identity: the domain-separated, length-framed SHA-256 over the product digest, control-manifest bytes, baseline SQL, seed SQL, role-bootstrap SQL, and legacy checksum inventory.

The current generated identities are recorded in `migrations/v2/manifests/G3-A4-MANIFEST.json`.

## Seed contract

The seed contains only:

- the seven canonical G2 permission names resolved by DECISION-016; and
- one deterministic successful revision row binding version `0001` to the baseline SQL checksum.

It contains no business rows, production identifiers, sample users, credentials, source data, or historical backfill.

## Scope exclusions

G3 does not authorize or contain:

- the Go migration runner, lock management, restart/failpoint behavior, or automatic source adoption (G4);
- blocking GitHub workflow changes (G5);
- forward corrective revisions;
- legacy backfill, cutover, or contract/removal DDL;
- modification or execution of migrations `001–040`;
- changes to the signed G2 product manifest.

## Stop conditions

Stop G3 and return to the applicable prior gate if:

- the signed product manifest, checksum, receipt, or legacy bytes change;
- an object cannot be derived deterministically from approved evidence;
- a clean installation requires manual SQL;
- an environment-specific owner, timestamp, host, credential, or database name enters a generated artifact;
- extension functions lose their intended default `PUBLIC EXECUTE` posture;
- an application function retains `PUBLIC EXECUTE`;
- either fresh profiler fingerprint differs; or
- an adoption path requires source-data repair, legacy replay, or backfill.
