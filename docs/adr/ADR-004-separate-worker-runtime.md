# ADR-004: Separate Worker Runtime

## Status

Accepted.

## Decision

Run background workloads outside the API process.

## Rationale

Workers isolate slow, retryable, probabilistic, or integration-heavy work from user-facing request paths.

## Consequences

Workers include outbox, context ingestion, event handling, integrations, and agents.

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
