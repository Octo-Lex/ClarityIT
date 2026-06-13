# ADR-016: A0-A5 Autonomy Ladder

## Status

Accepted.

## Decision

Classify every agent capability by autonomy level A0 through A5.

## Rationale

Enables full-system design while controlling operational risk.

## Consequences

High-risk actions default to A4 approval-gated with recent MFA.

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
