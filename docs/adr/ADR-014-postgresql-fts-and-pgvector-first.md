# ADR-014: PostgreSQL FTS and pgvector First

## Status

Accepted.

## Decision

Use PostgreSQL full-text search and pgvector for Genesis semantic retrieval.

## Rationale

Avoids additional storage engines while enabling search and embeddings.

## Consequences

Typesense/Qdrant can be added later as projections.

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
