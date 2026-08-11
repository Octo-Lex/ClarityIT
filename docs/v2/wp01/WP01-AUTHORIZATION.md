# WP-01 Authorization and Authority-Set Ratification

**Authorization ID:** `WP01-AUTH-2026-08-12`  
**Date:** 12 August 2026  
**Package:** WP-01 — Authoritative Kernel Foundation  
**Baseline:** `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f`  
**Status:** **AUTHORIZED FOR PACKAGE-PLAN CREATION; IMPLEMENTATION ACTIVATES WHEN THE WP-01 PLAN IS INTEGRATED**

## 1. Authorization

The client explicitly authorized WP-01 on 12 August 2026 and directed creation of the package plan.

This authorization permits:

- creation, review and integration of the formal WP-01 package plan;
- ratification of the exact existing v2 authority set needed by the Delivery Roadmap WP-01 entry criteria;
- after integration of the package plan, implementation of the bounded WP-01 scope through RG-01 without further routine micro-authorization;
- normal code, schema, test, CI, evidence and documentation changes required by the integrated plan;
- bounded experiments needed to resolve routine implementation uncertainty while preserving the plan's frozen semantics and exclusions.

This authorization does **not** permit WP-02 or later-package implementation, a live provider mutation, Site Runtime execution, production cutover, a second provider, multi-target execution, or any bypass of the authoritative kernel.

## 2. WP-01 governing authority set

For WP-01 implementation and RG-01 acceptance, the following exact repository artifacts are ratified together as the governing authority set required by Delivery Roadmap v0.2:

| Authority | Exact repository identity | WP-01 status |
|---|---|---|
| Product Definition v0.1 | `docs/v2/ClarityIT_v2_Product_Definition_v0.1.md`, blob `d44975d1557e8499c4e7613a5cd49115126266b0` | Ratified for WP-01 product scope and first-release semantics |
| Authoritative Execution Kernel v0.1 | `docs/v2/ClarityIT_v2_Authoritative_Execution_Kernel_Specification_v0.1.md`, blob `1153fb3bfadb1e603307354dc8b6e361eb44167d` | Ratified as highest WP-01 execution-semantics authority |
| v1-to-v2 Compatibility and Migration v0.1 | `docs/v2/ClarityIT_v2_v1-to-v2_Compatibility_and_Migration_Specification_v0.1.md`, blob `bdf179c677f283591842f5a52e41092a70e0b660` | Ratified for coexistence, historical truth and additive migration constraints |
| Layered System Architecture | `docs/v2/ClarityIT-v2-Layered-System-Architecture.md`, blob `9d42a74b39e941509725c1c5dd42a87c9126b9e8` | Ratified as WP-01 architecture baseline where consistent with higher authorities |
| Native Pattern Specification v0.1 | `docs/v2/ClarityIT_v2_Native_Pattern_Specification_v0.1.md`, blob `00ce72fab791e8b959549b4845d40b4a48954044` | Ratified for the WP-01-owned patterns and skeletons named by the roadmap/plan |
| Delivery Roadmap v0.2 | `docs/v2/ClarityIT_v2_Delivery_Roadmap_v0.2.md`, blob `89911eb29972d813d75f22d98cf239d2b61784b6` | Ratified for WP-01 boundary, RG-01 and sequencing only |
| Environment Trust and Evidence Custody Profile v0.1 | `docs/v2/ClarityIT_v2_Environment_Trust_and_Evidence_Custody_Deployment_Profile_v0.1.md`, blob `8a6d28d538fd0d5525114958329b0592829806a9` | Existing adopted development trust/custody profile remains in force |
| WP-00 final evidence | `main@e13c8b734b39afb32ff5e3e4a7281543f33d8a1f` | Accepted immutable foundation input |

This ratification is **bounded to WP-01**. It does not silently approve future WP-02 through WP-10 implementation, nor does it prevent a later formally governed successor revision to any authority.

## 3. Activation boundary

Package planning is active immediately under this authorization.

WP-01 implementation authority activates only when:

1. `ClarityIT_v2_WP-01_Authoritative_Kernel_Foundation_Plan_v0.1.md` is integrated to `main`;
2. the integrated plan binds this authorization ID and the exact authority set above;
3. the accepted WP-00 required CI controls remain in force;
4. no demonstrated contradiction invalidates a bound authority or WP-00 foundation identity.

Once activated, routine implementation choices do not require further authorization. A new client decision is required only for a demonstrated blocker that requires changing a frozen package semantic, acceptance criterion, authority identity, or explicit exclusion.

## 4. Non-negotiable boundary

WP-01 is a **provider-mutation-free kernel foundation package**.

The package may use deterministic fake/no-op adapter, route, provider-receipt and verifier fixtures to prove kernel behavior. It must not use real provider credentials or perform a live consequential provider call. The first live provider-neutral `compute.virtual_machine.start@1` proof remains WP-02.

## 5. Terminal plan outcome

Package-plan creation ends with one of:

```text
WP01_PLAN=INTEGRATED
WP01_IMPLEMENTATION_AUTHORITY=ACTIVE
```

or a concrete demonstrated planning blocker that prevents a coherent WP-01 plan under the ratified authority set.
