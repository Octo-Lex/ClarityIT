# G3 Approved-Source Adoption Contract

G3 defines the source-adoption boundary; it does not implement the G4 runner.

An approved existing source may be marked as having adopted baseline `0001` only when all of the following are true before mutation:

1. The source fingerprint is present in the G1-approved allowlist.
2. PostgreSQL major version is 16 and the database name is `clarityit`.
3. Product-schema comparison against the signed G2 manifest reports no unexplained object drift.
4. Required extensions are installed and compatible.
5. The privileged bootstrap can establish the signed five-role target without leaving superuser, ambient owner, or delegation authority in an application identity.
6. The legacy checksum inventory is present only as provenance; no file `001–040` is selected or marked as executed.
7. Baseline adoption requires no business-row update, manual repair, backfill, or corrective DDL.

The eventual G4 runner must perform the adoption as one fail-closed operation:

- acquire the migration advisory lock;
- re-profile and match the approved source;
- install or verify the `platform` control schema;
- record the approved `source_profile_id`;
- insert the immutable successful `platform.schema_revisions` row for version `0001` using the committed baseline checksum;
- record reconciliation evidence; and
- verify the product target and control identity before success.

If any precondition fails, G4 must perform no adoption write. This contract authorizes no backfill and supplies no standalone SQL that could bypass the runner's preflight, lock, evidence, or restart semantics.
