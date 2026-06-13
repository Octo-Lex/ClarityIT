# ADR-013: Universal Object Spine with Typed Extensions

## Status

Accepted.

## Decision

Represent all important entities through a shared object spine and domain-specific extension tables.

## Rationale

Prevents internal tool sprawl while preserving domain semantics.

## Consequences

Objects share links, comments, refs, ownership, status, permissions, context, and audit behavior.

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
