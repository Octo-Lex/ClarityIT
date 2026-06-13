# ADR-003: Go Modular Monolith API

## Status

Accepted.

## Decision

Use a Go modular monolith for the primary API and domain services.

## Rationale

Avoids premature microservice complexity while preserving module boundaries.

## Consequences

Modules include IAM, Work, Queue, Project, Wiki, Hub, Grid, Context API, and Tool Gateway.

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
