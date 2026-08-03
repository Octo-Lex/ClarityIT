# G3 Approved-Source Adoption Contract

G3 defines the source-adoption boundary and supplies the deterministic
adoption artifact (`migrations/v2/adoption/0001_adopt_p3.sql`).  G4
implements the same contract in the Go migration runner.

## Precondition checks (before mutation)

An approved existing source may be marked as having adopted baseline
`0001` only when all of the following are true before mutation:

1. The source fingerprint is present in the G1-approved allowlist.
2. PostgreSQL major version is 16 and the database name is `clarityit`.
3. Product-schema comparison against the signed G2 manifest reports no unexplained object drift.
4. Required extensions are installed and compatible; the bootstrap identity owns them.
5. The privileged bootstrap can establish the signed five-role target without leaving superuser, ambient owner, or delegation authority in an application identity.
6. The legacy checksum inventory is present only as provenance; no file `001–040` is selected or marked as executed.
7. Adoption requires no mutation of pre-existing source business rows; the only permitted product-row write is the seven canonical permission inserts.

## G3 deterministic adoption artifact

G3 supplies `0001_adopt_p3.sql`, a single-transaction artifact that
reconciles an existing P3 source to the signed G2 governed posture.  It
performs no legacy-replay, no product-table creation, and no backfill.
It is fail-closed: the runtime `g3.source_fingerprint` setting must
match the G1-approved P3 golden, or the entire transaction aborts and
rolls back with zero writes.

The G3 adoption proof runs this artifact against a live P3 source on
the pinned PostgreSQL 16 image and requires the resulting governed
fingerprint to equal the fresh-install governed fingerprint (convergence).

## G4 runner contract

The eventual G4 runner must perform the adoption as one fail-closed
operation:

- acquire the migration advisory lock;
- re-profile and match the approved source;
- install or verify the `platform` control schema;
- record the approved `source_profile_id`;
- insert the immutable successful `platform.schema_revisions` row for version `0001` using the committed baseline checksum;
- record reconciliation evidence; and
- verify the product target and control identity before success.

If any precondition fails, G4 must perform no adoption write.  This
contract authorizes no backfill.
