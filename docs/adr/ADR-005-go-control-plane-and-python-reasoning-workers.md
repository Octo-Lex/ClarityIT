# ADR-005: Go Control Plane and Python Reasoning Workers

## Status

Accepted.

## Decision

Use Go for deterministic control-plane services and Python for agent reasoning workers.

## Rationale

Go provides strong operational reliability; Python accelerates AI/LLM development.

## Consequences

Python workers cannot mutate state directly and must call the Go Tool Gateway.

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
