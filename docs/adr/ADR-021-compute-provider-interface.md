# ADR-021: Compute Provider Interface — Proxmox First, Not Proxmox Only

## Status

Accepted.

## Context

ADR-018 established Proxmox as the first compute integration — read-rich,
action-gated, behind approval + MFA + mutation windows. As the platform
prepares for public release, a question arises: is Proxmox a hardcoded
binding, or is it the first implementation of a general provider boundary?

The codebase already has the right structure:

- `ProxmoxClient` is an **interface** (`services/api/internal/proxmox/handler.go`)
  with 7 methods: 2 read (ListNodes, ListVMs) and 5 mutation
  (Start/Shutdown/Stop/Snapshot/GetTaskStatus).
- A `FakeProxmoxClient` stub is the **default** when `PROXMOX_ENABLED=false`.
- Only `main.go` imports the `proxmox` package. Every other package — the Tool
  Gateway, approval workflow, agent runtime, event outbox — is unaware of
  Proxmox. Dependency injection is clean.

However, the coupling is in the **contract surface**, not the code structure:

- Route paths contain `proxmox` (`/integrations/proxmox/...`,
  `/assets/{id}/actions/proxmox/start`).
- Action types stored in the database are `proxmox.start`, `proxmox.shutdown`,
  etc.
- Permission strings reference `integrations.proxmox.*`.
- Domain types are named `ProxmoxNode`, `ProxmoxVM`.

## Decision

**Proxmox is the first provider, not the only provider.** The platform treats
compute integrations as additive: each provider implements a client interface
and registers its own routes. We do **not** rename the existing Proxmox-named
surface to be provider-agnostic prematurely.

The extensibility boundary is made explicit in the README and this ADR so that
the public repository reads correctly to teams evaluating the platform for
non-Proxmox infrastructure.

## Rationale

- **The interface already exists.** Adding a provider does not require core
  changes — implement the interface, register routes, wire behind a config flag.
- **The safety model is provider-agnostic.** The Tool Gateway, A0–A4 autonomy
  ladder, mutation windows, approval + MFA requirements, and the "no destructive
  mutations" rule apply to all compute integrations equally. This is the
  platform's core IP, not a Proxmox feature.
- **Premature generalization is costly.** Renaming routes, action types,
  permissions, and domain types — and migrating the stored data + frontend +
  E2E tests — would touch every layer for a second provider that does not yet
  exist.
- **Named-first is the proven pattern.** Terraform providers, Packer builders,
  and Kubernetes cloud-controller-managers all name the first implementation
  and use the interface as the extension point.

## Consequences

- New providers (Hetzner Cloud, Kubernetes, libvirt, AWS, etc.) are added
  alongside Proxmox, not by replacing it.
- The `proxmox` package name and route prefix remain — they are honest about
  what the first implementation targets.
- A future provider gets its own package (`internal/hetzner/`,
  `internal/kubernetes/`), its own route prefix (`/integrations/hetzner/`),
  and its own action-type prefix (`hetzner.start`).
- If a common abstraction is needed later (e.g., a shared `ComputeProvider`
  interface), it can be extracted from the existing `ProxmoxClient` interface
  without breaking existing integrations.

## Review triggers

Revisit this ADR if:

- a second provider is implemented and the duplication becomes costly,
- a third provider makes a shared interface clearly necessary,
- or the Proxmox-specific naming causes measurable adoption friction.
