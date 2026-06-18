# Security Policy

## Reporting a Vulnerability

ClarityIT takes security reports seriously. **Please do not file public GitHub issues for security vulnerabilities.**

Instead, report suspected vulnerabilities privately to the maintainers. Include:

- A description of the issue and its potential impact
- Steps to reproduce (proof of concept if possible)
- The affected version/commit

You should receive an acknowledgement within 72 hours. We will coordinate a fix and disclosure timeline with you.

## Security Model

ClarityIT is designed around a sovereign-hybrid deployment model with an explicit agent autonomy boundary:

- **Agent autonomy is capped at A4.** A5 (full autonomy) is hardcoded-disabled — rejected before any database lookup and excluded by a `CHECK` constraint.
- **The reasoning worker is isolated.** It has no `DATABASE_URL`, `NATS_URL`, `REDIS_URL`, or `MINIO_ENDPOINT`. It fails closed at startup if any are set and communicates only over HTTP to the Go API.
- **No destructive mutations via agents.** Agents emit *intentions* through a 13-check Tool Gateway policy chain; allowed mutations are limited to start/shutdown/stop/snapshot.
- **High-risk operations require MFA + approval + policy + audit + outbox** before execution.
- `chain_of_thought` is always stripped from model output; only `reasoning_summary` is retained.

See [`docs/security/agent-autonomy-boundary.md`](docs/security/agent-autonomy-boundary.md) and [`docs/architecture/`](docs/architecture/) for the full threat model.

## Scope

This policy covers the ClarityIT source repository. It does not cover third-party dependencies (report those upstream) or deployments you do not control.
