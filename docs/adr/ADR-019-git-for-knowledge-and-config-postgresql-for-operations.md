# ADR-019: Git for Knowledge and Config, PostgreSQL for Operations

## Status

Accepted.

## Decision

Use Git for runbooks, docs, skills, schemas, deployment templates, and config snapshots.

## Rationale

Git is excellent for human-readable knowledge and configuration history, not live operations.

## Consequences

Tickets, incidents, audit, sessions, permissions, idempotency, and agent runs remain in PostgreSQL.

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
