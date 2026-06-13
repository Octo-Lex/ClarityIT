# ADR-012: Web-First PWA

## Status

Accepted.

## Decision

Build the primary interface as a web-first PWA.

## Rationale

Matches Cloudflare Tunnel access and avoids premature desktop/mobile complexity.

## Consequences

Tauri and native mobile remain future clients.

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
