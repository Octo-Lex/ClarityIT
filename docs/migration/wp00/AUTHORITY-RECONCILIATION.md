# ClarityIT v2 Authority Reconciliation Manifest

**Reconciliation date:** 2026-08-06  
**Destination branch:** `reconcile/v2-authority-main`  
**Destination base:** `main@7d18754e4503ac656469d4388405db19e1eaaa10`  
**Authority source:** `wp00/g2-schema-decisions@ca3bfcc720669e33dfe6255fa6b5ac8058117281`

## Purpose

This manifest records the documentation-only integration of the current ClarityIT v2 authority set onto `main`. It does not merge WP-00 runtime implementation, migration SQL, schema tooling, provider behavior, deployment changes, or production authority.

The integration preserves source Git blob identities for imported files wherever possible. A destination commit identity is packaging provenance and does not replace the source decision, evidence, or approval identity recorded by the authoritative documents.

## Status boundary

- WP-00 G1, G2, and G3 are recorded as closed by their controlling source records.
- WP-00 G4 is authorized to implement under `G4-AUTH-2026-08-05` and is not accepted.
- G5 and G6 remain unauthorized and incomplete.
- Target-state architecture does not assert delivered runtime capability.
- This reconciliation grants no provider mutation, production migration, cutover, Site Runtime, host-agent, adapter, or WP-01 through WP-10 implementation authority.

## Integrated authority set

| Destination | Source classification | Reconciliation treatment |
|---|---|---|
| `docs/v2/PROJECT-COMPLETION-AUTHORITY.md` | Current project handoff and gate authority | Imported from the authority source commit |
| `docs/v2/README.md` | v2 authority index | Imported as the current entry point |
| Product Definition | Normative product authority | Imported without semantic rewriting |
| Authoritative Execution Kernel | Normative execution authority | Imported without semantic rewriting |
| v1-to-v2 Compatibility and Migration Specification | Normative migration authority | Imported without semantic rewriting |
| WP-00 Migration Baseline and CI Stabilization Plan | Historical plan and acceptance contract | Imported with its recorded revision history intact |
| Delivery Roadmap v0.2 | Planning and dependency authority | Imported from the authority source commit |
| Native Pattern Specification v0.1 | Cross-cutting implementation-pattern authority | Imported from the authority source commit |
| Environment Trust and Evidence Custody Profile v0.1 | Deployment trust and evidence-handling authority | Imported from the authority source commit |
| Layered System Architecture | Target-state architecture | Imported; does not assert delivery |
| Authoritative Operation Sequence | Target-state execution sequence | Imported; does not assert delivery |
| Trust and Deployment Topology | Target-state trust topology | Imported; does not assert delivery |
| Signals and Routines | Target-state signal-processing view | Imported; does not assert delivery |
| `docs/v2/SHA256SUMS.txt` | Source document-set integrity manifest | Imported from the authority source commit |

## Source evidence references

Some controlling WP-00 evidence remains on the governed source branch until its implementation/evidence integration path is separately reviewed. Relative references in the imported authority documents identify those canonical repository paths. Their absence from this docs-only reconciliation must not be interpreted as revocation, duplication, or re-signing.

The controlling source paths include:

- `migrations/profiles/G1-APPROVALS.md`
- `migrations/profiles/g2/G2-APPROVALS.md`
- `docs/migration/wp00/g3/G3-APPROVALS.md`
- `docs/migration/wp00/g4/G4-AUTHORIZATION-AND-PLAN.md`

## Reconciliation rules

1. Historical statements remain historical; current status is read from `PROJECT-COMPLETION-AUTHORITY.md`.
2. Authorization is not acceptance.
3. Source-attributed claims are not Verification, and Verification is not acceptance.
4. Frozen evidence is not edited merely to improve presentation.
5. Later gate changes require a successor authority record bound to exact implementation and evidence identities.
6. The docs-only integration may merge independently of G4 implementation because it records G4 as authorized but unaccepted.
