# ADR-020: Golden Thread Scenario Packs

## Status

Accepted.

## Decision

Use scenario packs as integration validation, not as MVP scope.

## Rationale

Tests whether identity, events, context, agents, views, audit, approvals, storage, and workflows compose correctly.

## Consequences

Required packs include incident response, service desk triage, infrastructure action approval, knowledge drift correction, project risk detection, and denied agent action.

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
