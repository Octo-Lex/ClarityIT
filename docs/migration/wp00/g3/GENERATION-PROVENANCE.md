# G3 Generation Provenance

## Generator

- Entrypoint: `scripts/migration/generate_g3.py`
- Version: `g3-baseline-generator-v1`
- Frozen source commit: `f04f94faad0105d1c3274e9c7974d44f936a0d28`
- Determinism check: `python3 scripts/migration/generate_g3.py --check`
- Static verification: `bash scripts/migration/validate_g3.sh --static`
- Full two-install verification: `bash scripts/migration/validate_g3.sh`

## Why the P3 generator is not reused

The signed G2 manifest and the P3 capture manifest are structurally different:

- G2 stores `columns`, `constraints`, `indexes`, and `triggers` inside each `tables["schema.table"]` record.
- P3 stores those collections in top-level dictionaries keyed by `schema.table`.

`generate_p3.py` therefore cannot consume the signed G2 bytes without silently reading the wrong structure. G3 has a new reader for the nested shape. Its emission order deliberately follows the proven P3 pattern: extensions, sequence, application functions, tables with non-FK constraints, non-constraint indexes, triggers, deferred foreign keys, then signature-scoped grants.

## Signed selections

The generator does not infer application functions by a mutable denylist. It selects the exact 10 `(schema, name, identity arguments)` signatures in `target_grants.application_functions` and requires a one-to-one match to function bodies in the signed manifest. The remaining 81 extension-provided function records are not emitted as application DDL.

Extensions are installed by the bootstrap installer before `SET ROLE clarityit_owner`. This preserves their installer-default ACL. Product objects are then created after the role switch so their owner is `clarityit_owner`; each application function receives the signed per-signature `PUBLIC` revoke and `clarityit_app` grant.

## Legacy custody

The archive is read from `git cat-file blob f04f94f:<path>`, not copied from potentially converted working-tree bytes. Generation requires exactly the ordered numeric range `001–040`. Static verification compares, for every file:

1. the signed G2 Git blob;
2. the still-active repository path;
3. the provenance archive copy; and
4. the ordered SHA-256 inventory entry.

Any mismatch fails closed.

## Composite framing

The composite digest is SHA-256 over:

1. domain bytes `clarityit-g3-composite-v1\0`;
2. for every ordered component: unsigned 32-bit big-endian label length, UTF-8 label, unsigned 64-bit big-endian data length, and raw data bytes.

The ordered component labels are:

1. `product_manifest_blob_sha256`
2. `control_manifest`
3. `baseline_sql`
4. `seed_sql`
5. `role_bootstrap_sql`
6. `legacy_checksum_inventory`

Length framing makes concatenation unambiguous. The A4 manifest is not an input to its own composite digest; its detached checksum is recorded afterward in `migrations/v2/manifests/SHA256SUMS`.

## Reproducibility

The generator uses no current time, random value, environment variable, hostname, branch, or working-directory identity. Canonical permission UUIDs are UUIDv5 values derived from fixed names. The revision timestamp is the fixed G3 artifact date, not an execution timestamp. Actual run timing and actor evidence belong to the G4 run ledger.
