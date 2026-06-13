# ADR-007: ESAA Structured Agent Intentions

## Status

Accepted.

## Decision

Agents emit structured intentions rather than directly mutating state.

## Rationale

Separates probabilistic reasoning from deterministic effect application.

## Consequences

Intentions are validated by schema, IAM, autonomy policy, approvals, MFA, SoD, idempotency, and audit before execution.

## Implementation notes

- This ADR is part of the ClarityIT Genesis Build baseline.
- Contracts should be represented in `schemas/cue` where applicable.
- Runtime behavior should be enforced in Go services, PostgreSQL constraints, migrations, and CI contract tests.

## Review triggers

Revisit this ADR if:

- the decision causes measurable operational risk,
- the implementation becomes inconsistent with the Genesis Build baseline,
- scaling pressure proves the initial boundary insufficient,
- or a security review identifies a stronger alternative.
