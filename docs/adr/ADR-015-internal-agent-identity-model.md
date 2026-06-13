# ADR-015: Internal Agent Identity Model

## Status

Accepted.

## Decision

Create internal agent identities with tool grants and team scope.

## Rationale

Agents need auditable authority without introducing broad service principals.

## Consequences

Agent authority is bounded by team, tool, autonomy level, risk, approval, MFA, and expiry.

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
