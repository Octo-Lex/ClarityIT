# ADR-010: CUE Build-Time Contract Compiler Only

## Status

Accepted.

## Decision

Use CUE for contracts and generation, not request hot-path validation.

## Rationale

Provides strong cross-contract validation without runtime overhead.

## Consequences

Generated artifacts are used by Go, TypeScript, SQL, OpenAPI, and agent tools.

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
