# ADR-001: Sovereign Hybrid Deployment

## Status

Accepted.

## Decision

Adopt self-hosted Proxmox origin with Cloudflare Tunnel and web-first access.

## Rationale

Data sovereignty, secure ingress, operational realism, and low deployment friction.

## Consequences

Only web/API ingress is exposed through Cloudflare Tunnel. Databases, NATS, Redis/Valkey, MinIO, Proxmox APIs, and workers remain private.

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
