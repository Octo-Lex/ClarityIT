# WP-01 G0 — State and Reason-Code Applicability

**Gate:** WP01-G0 — Plan/Contract Freeze  
**Authority:** `WP01-AUTH-2026-08-12`  
**Baseline:** `main@33d3802d93c6d3123d9377566f0f3b6fb1360ecb`  
**Status:** Frozen semantic implementation map candidate

## 1. Purpose

This document freezes the Kernel state machines and reason-code families that WP-01 must implement or exercise. It does not create new semantic states. Illegal transitions must fail without partial authoritative mutation.

## 2. Operation Packet

Canonical states:

- `draft`
- `proposed`
- `superseded`
- `withdrawn`
- `expired`

Required semantics:

| From | To | WP-01 disposition |
|---|---|---|
| draft | proposed | legal when packet canonicalization, integrity and proposal prerequisites pass |
| proposed | superseded | legal only through an immutable successor relationship; proposed bytes remain unchanged |
| proposed | withdrawn | legal proposer withdrawal only before dispatch/execution has begun; does not erase proposal history |
| proposed | expired | legal only when the governed validity window expires **before dispatch has begun**, with authoritative proof that no attempt crossed the submission boundary |
| draft | withdrawn/superseded/expired | **illegal** under Kernel v0.1; draft may only transition by propose |
| proposed | expired after dispatch/submission began or may have begun | **illegal**; execution lineage remains attached to the proposed packet rather than being rewritten as expired |
| proposed | proposed with modified bound bytes | **illegal**; material change requires successor |
| terminal packet state | mutation back to draft/proposed | **illegal** |

A proposed packet digest binds downstream PolicyDecision, ApprovalDecision and AuthorityGrant scope. Expiry is a pre-dispatch validity outcome, not a mechanism for rewriting the state of a packet whose execution has already begun.

## 3. AuthorityGrant

Canonical states:

- `issued`
- `reserved`
- `consumed`
- `revoked`
- `expired`

There is **no `released` state**. `release` is the Kernel transition action that returns a safe unconsumed reservation to `issued`.

| From | To | WP-01 disposition |
|---|---|---|
| issued | reserved | legal only for exact matching broker preflight/attempt reservation |
| issued | revoked | legal by authoritative revoke command |
| issued | expired | legal when validity expires |
| reserved | consumed | legal once provider submission begins or may have begun under exact grant-use semantics |
| reserved | issued | legal `release` only when submission is provably absent and retry policy permits reuse |
| reserved | revoked | legal; future dispatch blocked |
| reserved | expired | **illegal** under the Kernel v0.1 transition table; expiry is an allowed transition from `issued`, not an invented reserved-state edge |
| consumed/revoked/expired | any nonterminal state | **illegal** |

Reservation/release must not enable blind retry after ambiguous submission. Once submission begins, the grant is consumed; ambiguity never returns it to `issued`.

## 4. ExecutionAttempt

Canonical states:

- `created`
- `preflight`
- `dispatchable`
- `submitting`
- `submitted`
- `running`
- `provider_completed`
- `provider_failed`
- `blocked`
- `cancelled`
- `outcome_unknown`

Required legal flow:

```text
created
  -> preflight
      -> blocked
      -> cancelled
      -> dispatchable
          -> cancelled
          -> submitting
              -> submitted
              -> provider_failed
              -> outcome_unknown
submitted
  -> running
  -> provider_completed
  -> provider_failed
  -> outcome_unknown
running
  -> provider_completed
  -> provider_failed
  -> outcome_unknown
outcome_unknown
  -> submitted
  -> provider_completed
  -> provider_failed
```

`provider_completed`, `provider_failed`, `blocked` and `cancelled` are terminal for that attempt. Kernel v0.1 permits `outcome_unknown` reconciliation to submitted/completed/failed when independent evidence resolves the ambiguity. If reconciliation remains unresolved, **there is no `outcome_unknown -> outcome_unknown` transition**: the attempt stays in the existing terminal-unknown state and new reconciliation evidence/audit records are appended without emitting a duplicate authoritative state transition or transition outbox event. Kernel v0.1 also does **not** introduce a direct `outcome_unknown -> running` edge.

Reconciliation updates that same attempt only when observed provider truth supports one of the legal resolving transitions and never silently creates a blind retry.

WP-01 uses deterministic fake/no-op execution to exercise these states. No real provider mutation occurs.

## 5. Verification

Canonical states:

- `pending`
- `running`
- `passed`
- `failed`
- `inconclusive`

Legal transition shape:

```text
pending -> running -> passed | failed | inconclusive
```

A retry/re-evaluation is represented by governed evidence/retry semantics required by the exact VerificationSpec; prior Verification history is not overwritten to manufacture a different result.

`provider_completed`, executor flags, reasoning conclusions and UI state are not Verification states or sufficient inputs.

## 6. OutcomeDecision

Canonical states:

- `pending`
- `accepted`
- `rejected`
- `correction_required`
- `compensation_required`

First-release `accepted` requires the exact prerequisite passed Verification plus an identified accountable human decision. Provider/executor/reasoning identities cannot directly create acceptance.

`correction_required` and `compensation_required` lead to explicit successor lineage; the original operation history remains unchanged.

## 7. Case lifecycle

Case lifecycle/status is a **projection over typed authoritative records**, not a substitute state machine that may override packet/attempt/Verification/outcome truth.

The Kernel v0.1 projection sequence is:

```text
open -> investigating -> decision_pending -> authorized -> executing -> verifying -> outcome_pending -> accepted | correction_required | closed
```

The projection must be rebuildable from PostgreSQL and preserve source class, causal ordering and unknown/inconclusive states rather than collapsing them into success.

## 8. Successor lineage types

WP-01 shall represent typed immutable relations at least for:

- `supersedes` / packet successor;
- `corrects`;
- `retries_after_safe_failure`;
- `reconciles_unknown`;
- `compensates`.

Lineage must support reconstruction of:

- happy;
- blocked;
- rejected;
- failed;
- cancelled;
- unknown;
- inconclusive;
- superseded;
- compensation-required;
- compensated;
- successor chains.

## 9. Reason-code model

Reason codes are versioned machine-readable values. Human diagnostic text may evolve without changing machine semantics, but code meaning must not be silently reused.

WP-01 shall use the Kernel families below.

### 9.1 `AUTH_*`

Applies to authority/policy/approval/grant rejection and mismatch, including examples such as:

- missing/invalid PolicyDecision;
- missing/rejected approval;
- separation-of-duties conflict;
- grant missing/expired/revoked/consumed;
- workspace/Case/packet/resource/capability/workload/route/policy/nonce/use mismatch.

Authority failures block before synthetic submission.

### 9.2 `PRECONDITION_*`

Applies to stale Resource/binding version, unsafe target state, missing required current Observation, schema/capability mismatch and other execution-time preflight failures.

A reasoning-time Context Bundle cannot satisfy an execution-time precondition merely by containing contextual text.

### 9.3 `SUBMISSION_*`

Applies to synthetic submission boundary behavior:

- definitely not submitted;
- accepted/submitted;
- transport/result ambiguity;
- idempotency conflict/duplicate handling.

Ambiguity maps to `outcome_unknown`, not blind resubmission.

### 9.4 `PROVIDER_*`

In WP-01 these codes are exercised by deterministic fake receipts only. They represent source-attributed provider/fixture claims such as running/completed/failed or provider-reference mismatch. They do not create Verification.

### 9.5 `OBSERVATION_*`

Applies to source/freshness/fieldset/current-state problems, including stale, missing, source mismatch and unusable observation evidence.

### 9.6 `VERIFICATION_*`

Applies to exact VerificationSpec mismatch, stale/missing evidence, verifier unavailable, pass/fail/inconclusive classification and invalid proof inputs.

Verifier unavailability or insufficient exact evidence is `inconclusive` unless the exact VerificationSpec defines a deterministic failure condition.

### 9.7 `OUTCOME_*`

Applies to acceptance/rejection/correction/compensation decision prerequisites, including missing human decision and invalid attempt to accept without passed Verification.

### 9.8 `EVIDENCE_*`

Applies to evidence digest/reference/redaction/reconstruction/sealing failure. Evidence failure cannot be treated as success merely because execution/provider claims exist.

### 9.9 `SITE_*`

WP-01 supports only fail-closed route-unavailable/unimplemented semantics for Site Runtime references. Live Site Runtime disconnection, spool/journal, local polling and recovery behavior are deferred to WP-04.

### 9.10 Context-specific families

The Context Overlay Contract additionally reserves:

- `CONTEXT_SCOPE_*`
- `CONTEXT_ACCESS_*`
- `CONTEXT_LIMIT_*`
- `CONTEXT_STALE_*`
- `CONTEXT_MISSING_*`
- `CONTEXT_SCREENING_*`
- `CONTEXT_SHADOW_*`
- `CONTEXT_POLICY_RELAXATION_*`
- `CONTEXT_SECRET_*`
- `CONTEXT_DIGEST_*`

These must not be overloaded as authority or execution success reasons.

## 10. Cross-state invariants

WP-01 implementation shall prove:

1. an ApprovalDecision cannot transition an Attempt by itself;
2. a provider-completed ResultClaim cannot transition Verification to passed;
3. a passed Verification cannot transition OutcomeDecision to accepted without the required human decision;
4. `outcome_unknown` cannot trigger automatic new attempt submission;
5. unresolved reconciliation appends evidence without a duplicate `outcome_unknown` self-transition;
6. packet expiry cannot rewrite a packet after dispatch has begun or may have begun;
7. cancelled and failed attempts remain historically visible;
8. compensation is a successor operation, not mutation of the original attempt/outcome;
9. stale aggregate-version writes fail before state change;
10. duplicate delivery produces at most one authoritative transition per consumer/logical command;
11. illegal transition failure leaves no partial audit/outbox/state inconsistency;
12. state + audit + outbox atomicity is preserved.

## 11. Evidence requirements

A3/A4/A5/A6 evidence must include the exact state/reason matrices exercised, expected/actual outcomes, transaction rollback proof for negatives where applicable, and immutable references to the tests/runs that established them. Reconciliation evidence must distinguish an actual state transition from an evidence-only no-state-change result.

## 12. Change control

New semantic states, an additional transition edge not permitted by Kernel v0.1, removal of `outcome_unknown`/`inconclusive`, reclassification of provider claims as Verification, a `released` AuthorityGrant state, post-dispatch packet expiry, an `outcome_unknown` self-transition, or a transition that weakens immutable-successor semantics requires a governed semantic successor; it is not a routine implementation detail.
