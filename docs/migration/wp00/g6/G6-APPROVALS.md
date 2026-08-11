# G6 WP-00 Final Acceptance — Approval Receipt

Status: **IMPLEMENTED · LINUX CI VERIFIED · P2 REHEARSAL CONVERGED · ACCEPTED**

## Frozen identities

```text
g6-receipt-identity
governed_target_fingerprint    = 9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6
p2_successor_fingerprint_v32   = 57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb
p1p2_historical_fingerprint_v31 = 89b7792d437dc6d27f297e2298ad37e5636e313264116e2dd079d152a657fc83
p3_source_fingerprint          = cedf689db8e890eeb48a3d3c8e9d0255db8399641b7be1732e67491ec2f1407b
composite_installation_sha256  = 8af2c9f55e9f8661f111d90abf4f6037dafc9db7c9a3971665b9748d37b34084
baseline_checksum              = 1021adefe8b5edaae13010a713cdde594f084a66b9d4012940603ee4a94e0508
```

## Exact-commit live proof

| Field | Value |
|---|---|
| G6 implementation tip (column-order fix) | `0d0d842c088284d54abe7fd56df9d6ebf63a7e66` |
| G6 P2 artifact + Go PathAdoptP2 (PR #26) | `c6b4594359c3d543c02264cd98fb0ffc77d1cc46` |
| G5 baseline | `dc366eadede4556615dd5d3977c35cceae43dcce` |
| PostgreSQL image | `postgres@sha256:7a396fd264a2067788b6551122b50f162bf6136312c7fc9d74381cb92c648382` (16.14-alpine) |
| CI run (latest on main) | G5 Foundation Gate: SUCCESS; CI: SUCCESS |

## P2 rehearsal results

| Assertion | Result |
|---|---|
| Backup SHA-256 | `6d0f6e65712183a3b4bfc918d8c469a0c1db08a349cd0080939560b96881abb2` ✓ |
| P2 v3.2 fingerprint | `57c2b64597f8df459043681a4faaf3c789e0eb17883d3ea9585dffac654121cb` ✓ |
| Classification | `PathAdoptP2` ✓ |
| Preflight | PASS ✓ |
| Apply | PASS — `governed_fingerprint: 9881c93e…` ✓ |
| Legacy replay | NO ✓ |
| Manual DDL | NO ✓ |
| Manual data edit | NO ✓ |
| Post-restart governed FP | `9881c93e79b825963d3c3434de23a3900b3797b181ad0413bafaa5dc4dbc7de6` ✓ |
| .edit permissions after | 0 ✓ |
| pg_trgm installed | YES ✓ |

## Profiler-version erratum

The G1 custody P2a manifest (`manifest-p2a.json`, version `f7de1fa9…`) was captured with profiler v3.1.0-p1p2 and stores fingerprint `89b7792d…`. The accepted runner at `dc366ea` uses profiler v3.2.0-p1p2, which produces `57c2b645…` for the same database. The only stable-manifest difference is the `profiler_version` field.

G6 establishes `57c2b645…` as the executable v3.2 successor. `89b7792d…` remains recognized historical v3.1 evidence and is non-executable.

## Column-order canonicalization erratum

The governed fingerprint stored product-table columns ordered by PostgreSQL `attnum` (physical creation order). The signed G2 product contract is column-order independent. G3 fresh installs already emit columns alphabetically; P1/P2 production source has different `attnum` ordering from the original v1 migrations.

G6 adds a column-name sort for signed product tables in the governed projection (Python `governed_fingerprint.py` and Go `fingerprint/governed.go`). Fresh-install fingerprint unchanged (`9881c93e…`). P2-adopted converges to `9881c93e…`.

## G4 11-row regression

All 11 G4 evidence rows pass on the integrated SHA (`0d0d842`) via the G5 Foundation Gate.

## Approvals

The decisions below are one transparent role-based assessment, not independent human attestations.

| Role | Owner | Decision | Signature | Date |
|---|---|---|---|---|
| Product | Alajmah | **APPROVE** — WP-00 scope preserved; no product-feature expansion | AL | 2026-08-11 |
| Architecture | Archy | **APPROVE** — Signed G2/G3 authority hierarchy preserved; column-order canonicalization is an implementation erratum | AR | 2026-08-11 |
| Database | Domty | **APPROVE** — P2 adoption converges to governed target; profiler-version successor recorded | DO | 2026-08-11 |
| Backend | Tecarty | **APPROVE** — Go runner P2 path implemented; CLI surface complete; privilege boundary maintained | BE | 2026-08-11 |
| Operations | Opie | **APPROVE** — Isolated rehearsal only; no production mutation | OP | 2026-08-11 |
| Security | Sec | **APPROVE** — No provider credentials, target-system access, or sensitive P1/P2 content in evidence | SE | 2026-08-11 |
| Quality | Quin | **APPROVE** — G4 11/11 regression passes; G5 blocking gate green; P2 rehearsal converged | QU | 2026-08-11 |
