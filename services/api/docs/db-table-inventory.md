# ClarityIT Database Table Inventory

**Generated:** 2026-06-12
**Total tables:** 37
**Migrations applied:** 001–013

| # | Table | Migration | Purpose | Classification |
|---|-------|-----------|---------|----------------|
| 1 | `objects` | 001_genesis_core | Universal object spine — every domain entity is an object | Canonical |
| 2 | `object_links` | 001_genesis_core | Typed relationships between objects (parent, depends_on, etc.) | Canonical |
| 3 | `object_comments` | 001_genesis_core | Threaded comments on any object | Derived |
| 4 | `object_storage_refs` | 001_genesis_core | Links from objects to storage objects (attachments) | Derived |
| 5 | `outbox_events` | 002_outbox | Transactional outbox for reliable event publishing | Ephemeral |
| 6 | `audit_logs` | 003_audit | Immutable audit trail with HMAC'd PII | Canonical |
| 7 | `context_nodes` | 004_context | Context graph nodes (concepts, tags, facts) | Canonical |
| 8 | `context_edges` | 004_context | Context graph edges (relationships between nodes) | Canonical |
| 9 | `context_edge_evidence` | 004_context | Evidence supporting context edges | Derived |
| 10 | `context_bundles` | 004_context | Named bundles of context for agent injection | Canonical |
| 11 | `agent_identities` | 005_agents | Agent identity records (name, description, autonomy level) | Canonical |
| 12 | `agent_intentions` | 005_agents | Agent intention declarations (what it plans to do) | Ephemeral |
| 13 | `agent_runs` | 005_agents | Agent execution run records | Canonical |
| 14 | `agent_effect_results` | 005_agents | Results of agent effects (tool calls, mutations) | Canonical |
| 15 | `agent_tool_grants` | 005_agents | Tool access grants for agents (scoped, time-limited) | Canonical |
| 16 | `storage_objects` | 006_storage | Object storage metadata (S3/MinIO references) | Canonical |
| 17 | `bootstrap_lock` | 007_iam_core | Single-row table ensuring bootstrap runs exactly once | System |
| 18 | `users` | 007_iam_core | User accounts with email, password hash, token version | Canonical |
| 19 | `teams` | 007_iam_core | Team organizations with slugs and icons | Canonical |
| 20 | `roles` | 007_iam_core | Team-scoped role definitions (owner, admin, member, etc.) | Canonical |
| 21 | `permissions` | 007_iam_core | Named permissions (e.g., incidents.create, teams.manage) | Canonical |
| 22 | `role_permissions` | 007_iam_core | Maps permissions to roles | Derived |
| 23 | `team_memberships` | 007_iam_core | User-team membership with role_id FK | Canonical |
| 24 | `user_sessions` | 008_iam_sessions_tokens | User session tracking with HMAC'd IP | Ephemeral |
| 25 | `refresh_tokens` | 008_iam_sessions_tokens | Token families with rotation and reuse detection | Ephemeral |
| 26 | `password_reset_tokens` | 008_iam_sessions_tokens | Password reset flow tokens with expiry | Ephemeral |
| 27 | `invitations` | 010_iam_invitations | Team invitation tokens | Ephemeral |
| 28 | `team_access_grants` | 010_iam_invitations | External access grants for teams | Canonical |
| 29 | `platform_roles` | 011_iam_platform_roles | Platform-wide roles (platform_owner, platform_admin, auditor) | Canonical |
| 30 | `user_platform_roles` | 011_iam_platform_roles | User-to-platform-role assignments with revocation | Canonical |
| 31 | `integration_api_keys` | 012_iam_constraints | API keys for integrations with allowed_sources/scopes | Canonical |
| 32 | `idempotency_keys` | 013_idempotency | Idempotency key tracking for safe mutation retries | Ephemeral |
| 33 | `alerts` | 001_genesis_core | Alert objects (domain, via objects spine) | Canonical |
| 34 | `assets` | 001_genesis_core | Asset objects (domain, via objects spine) | Canonical |
| 35 | `docs` | 001_genesis_core | Document objects (domain, via objects spine) | Canonical |
| 36 | `incidents` | 001_genesis_core | Incident objects (domain, via objects spine) | Canonical |
| 37 | `work_items` | 001_genesis_core | Work item objects (domain, via objects spine) | Canonical |

## Classification Legend

- **Canonical** — source of truth, must not be derived or rebuilt
- **Derived** — computed from canonical data, can be regenerated
- **Ephemeral** — transient data, safe to truncate (sessions, tokens, outbox)
- **System** — operational control data (bootstrap lock)

## Notes

- Tables 33–37 (alerts, assets, docs, incidents, work_items) are typed extension tables per ADR-013 universal object spine. They share the `objects` table via `object_id` FK.
- The plan expected "~40 tables". Actual: 37. No missing tables — the delta comes from fewer domain extension tables than initially estimated.
