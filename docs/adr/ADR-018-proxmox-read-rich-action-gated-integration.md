# ADR-018: Proxmox Read-Rich, Action-Gated Integration

## Status

Accepted.

## Decision

Start Proxmox integration with inventory, metrics, backup status, and alert correlation; gate actions.

## Rationale

Provides value without unsafe automation defaults.

## Consequences

Start/stop/restart/migrate require approval and MFA; destructive actions disabled by default.

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
