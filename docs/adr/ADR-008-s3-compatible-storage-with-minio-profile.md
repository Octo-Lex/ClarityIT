# ADR-008: S3-Compatible Storage with MinIO Profile

## Status

Accepted.

## Decision

Use S3-compatible object storage and provide a MinIO deployment profile.

## Rationale

Supports self-hosting on Proxmox while avoiding hard dependency on one object store.

## Consequences

PostgreSQL owns metadata and permissions for every object.

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
