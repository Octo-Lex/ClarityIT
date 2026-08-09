# G4 Go Migration Runner — Approval Receipt

Status: **IMPLEMENTED · LINUX CI VERIFIED · ALL EVIDENCE ROWS PASS · ACCEPTED**

## Frozen identities

```text
g4-receipt-identity
product_manifest_blob_sha256 = 1f6e31422461173cd4b4671417809f8b819bad493efec2fb0a5cdd2783d37a63
control_manifest_sha256      = 3fd65e917ded8b7d59a1f42051b69f41e4b5c24f583f9524deaccdfdfb1add66
composite_installation_sha256 = 8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084
database_name                = clarityit
postgres_major               = 16
```

The frozen identities are bound to their generator and manifest files and are unchanged from the G3 signed authority.

## Adoption and governed-target identities

| Identity | Value |
|---|---|
| Governed target fingerprint | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` |
| Adoption artifact SHA-256 | `a89ab852b7add6e130bc9ed941caa4329f3024a5c1d3cabd7b25ba2f89a64359` |
| P3 golden source fingerprint | `cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b` |
| Baseline SQL checksum (rev 0001) | `1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508` |
| P1/P2 source fingerprint (recognized, not executable) | `89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83` |

## Exact-commit live proof

| Field | Value |
|---|---|
| Implementation squash commit (PR #16) | `f769cd3815ea08194b56c267cfa3b30fb7a12fd9` |
| Authority tip (incl. matrix fix PR #17) | `b31a7c5cd0ba132cb179db5751e8e2b8f339639f` |
| Linux CI run | `31336112238` |
| Platform | Ubuntu (GitHub Actions, Linux CI) |
| Runtime | PostgreSQL 16.14-alpine (pinned digest `7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382`) |
| Oracle | Python governed_fingerprint cross-validation: PASS |
| Tested by | `scripts/migration/run_g4_matrix.sh` + Go proof test suite |

## G4 evidence matrix results

```text
G4-01 fresh_install_ab           PASS
G4-02 approved_p3_adoption       PASS
G4-03 unknown_drifted_source     PASS
G4-04 packaged_checksum_mutation PASS
G4-05 advisory_lock_contention   PASS
G4-06 tx_failure_rerun           PASS
G4-07 nontransactional_path      PASS:path_absent
G4-08 verify_mode                PASS
G4-09 privilege_boundary         PASS
G4-10 legacy_exclusion           PASS
G4-11 evidence_hygiene           PASS
```

All 11 G4-AUTHORIZATION-AND-PLAN.md §4 evidence rows pass. PASS: 11 / 11.

## Receipt-only disclaimer

This receipt records the evidence only. It does not change the product, control, or composite identities established by the G3 signed authority. The only producing-tip-to-receipt-tip file changes are `G4-APPROVALS.md` (new) and `PROJECT-COMPLETION-AUTHORITY.md` (gate ledger update).

## Approvals

The decisions below are one transparent role-based assessment, not independent human attestations. The project authority requested one delegated assessment across the required roles.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Database | Domty | **APPROVE** — Runner database behavior, source-profile checks, locking, ledger, adoption, restart, and verification | DO | 2026-08-10 |
| Backend | Tecarty | **APPROVE** — Go command, packaged migration bytes, deterministic diagnostics, CLI surface, and tests | BE | 2026-08-10 |
