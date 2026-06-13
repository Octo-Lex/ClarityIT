# ADR-017: Least-Privilege IT Role Matrix

## Status

Accepted.

## Decision

Seed explicit IT roles and permissions with least privilege.

## Rationale

ClarityIT touches infrastructure and must prevent broad admin authority.

## Consequences

Roles include owner, admin, manager, member, viewer, on_call_engineer, infrastructure_engineer, security_admin, auditor, and automation_operator.

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
