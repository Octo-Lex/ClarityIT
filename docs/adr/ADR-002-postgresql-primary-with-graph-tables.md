# ADR-002: PostgreSQL Primary with Graph Tables

## Status

Accepted.

## Decision

Use PostgreSQL as the canonical data store with relational tables plus graph-style context tables.

## Rationale

Keeps source-of-truth, audit, IAM, context, idempotency, and transactional integrity in one operational database.

## Consequences

Graph database can be introduced later as a projection if traversal complexity exceeds PostgreSQL recursive CTE capabilities.

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
