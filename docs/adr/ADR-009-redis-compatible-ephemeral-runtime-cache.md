# ADR-009: Redis-Compatible Ephemeral Runtime Cache

## Status

Accepted.

## Decision

Use Redis-compatible backend, preferably Valkey by default, for ephemeral acceleration.

## Rationale

Supports cache, rate limiting, presence, leases, and short-lived context bundles.

## Consequences

Redis/Valkey must not store canonical audit, permissions, domain state, or idempotency truth.

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
