# WP-00 G6 — P2 Adoption-Path Blocker

**Status:** **DEMONSTRATED BLOCKER · CURRENT CORRECTIVE AUTHORIZATION INSUFFICIENT**  
**Date:** 2026-08-11  
**Parent authorization:** `G6-P2-SUCCESSOR-AUTH-2026-08-11`

## 1. Finding

The authorized P1/P2 v3.2 source-identity successor correction cannot safely be implemented as a classifier-only change because the only existing executable adoption path is a P3-specific artifact and ledger path.

This is a demonstrated implementation fact, not a speculative concern.

## 2. Existing runner behavior

`services/api/internal/migration/preflight.go` intentionally defines:

- P3 source fingerprint as executable through `PathAdopt`;
- historical P1/P2 fingerprint as recognized but non-executable;
- `PathAdopt` as `adopt_p3`.

`services/api/internal/migration/apply.go` executes `assets.AssetAdoptP3` for every `PathAdopt` decision and `profileIDForPath(PathAdopt)` always returns the fixed P3 profile ID.

Therefore making a P2 successor fingerprint executable through the existing `PathAdopt` would route P2 through P3-specific evidence semantics.

## 3. Frozen adoption artifact is P3-specific

`migrations/v2/adoption/0001_adopt_p3.sql` is explicitly a deterministic P3 approved-source adoption artifact.

Its preflight requires, among other conditions:

- `clarityit` exists as the P3 bootstrap **superuser**;
- `clarityit` owns `pgcrypto`, `citext`, and `pg_trgm`;
- P3 structural digests and P3 single-shot identity posture.

The real approved P2 restore diagnostic established the production-matching source role posture using `postgres` as the superuser. Manually converting the restored source into P3 bootstrap posture before migration would violate AC-00-14's no-manual-correction requirement.

Even if the P3 structural preflight could otherwise pass, the artifact hardcodes this provenance row:

- profile ID `7c5cb0b9-1fb4-540d-9433-f0196ff6f7bb`;
- schema fingerprint `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b`;
- P3 roles digest/source commit/approval metadata.

It also records schema revision name `adopt-p3`.

Using those bytes on a P2 source would create false source provenance.

## 4. Direct acceptance impact

A classifier-only change cannot satisfy AC-00-14 safely:

1. with the real P2 role posture, the P3 artifact is expected to reject its P3-bootstrap preconditions; or
2. altering P2 manually to satisfy those preconditions is prohibited; and
3. routing P2 through the P3 artifact would record P3 source identity/provenance rather than the actual P2 successor.

The existing fail-closed P1/P2 non-executable behavior was therefore materially meaningful, not merely an allowlist omission.

## 5. Triggered stop condition

`G6-P2-SUCCESSOR-AUTH-2026-08-11` permits a classifier-only correction and explicitly forbids:

- modifying `0001_adopt_p3.sql`;
- new migration/backfill semantics;
- weakening frozen provenance/checksum controls.

It also states that if the existing adoption contract cannot safely consume real P2, stop for a new governed decision.

That stop condition is now met.

No classifier or migration code has been changed under the current corrective authorization.

## 6. Smallest safe successor scope

A further explicit authorization is required for a **P2-specific adoption successor**. The smallest safe scope should permit:

1. freeze the deterministic v3.2 P2 source fingerprint after repeat evidence;
2. define a distinct P2 successor profile identity and metadata;
3. create a new deterministic P2 adoption artifact or equivalently governed P2-specific execution path without modifying the frozen P3 artifact;
4. use the real P2 bootstrap/role posture as a precondition rather than manufacturing P3 posture;
5. preserve product business rows and prohibit legacy `001-040` replay;
6. converge to the same frozen baseline checksum and governed target fingerprint;
7. record truthful P2 source profile identity in `platform.source_profiles` and `platform.migration_runs`;
8. preserve the frozen P3 adoption path byte-for-byte;
9. rerun all G4/G5 regression evidence plus the complete real P2 G6 rehearsal;
10. stop if P2 requires a data/schema correction beyond the already-governed reconciliation decisions.

Until that successor adoption path is separately authorized and proven, **AC-00-14 remains blocked and WP-00 cannot be accepted**.
