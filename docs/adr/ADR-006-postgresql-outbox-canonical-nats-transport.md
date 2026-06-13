# ADR-006: PostgreSQL Outbox Canonical, NATS Transport

## Status

Accepted.

## Decision

Persist domain events in PostgreSQL outbox inside the transaction, then publish to NATS JetStream asynchronously.

## Rationale

Prevents event loss and supports replay, retries, dedupe, and auditability.

## Consequences

NATS is transport, not canonical state.

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
