# G2 Decision — Migration 018 (Agent Runtime)

## Conflict

Migration 005 creates five agent tables (`agent_identities`, `agent_tool_grants`, `agent_runs`, `agent_intentions`, `agent_effect_results`) with `IF NOT EXISTS` and columns:
- `agent_identities.max_autonomy_level TEXT NOT NULL DEFAULT 'A0'`
- `agent_effect_results.result_payload JSONB NOT NULL DEFAULT '{}'`
- No `tool_registry` table

Migration 018 recreates the same five tables WITHOUT `IF NOT EXISTS` (conflict on fresh install) with different columns:
- `agent_identities.max_autonomy TEXT NOT NULL DEFAULT 'A0'` (renamed)
- `agent_effect_results.result JSONB NOT NULL DEFAULT '{}'` (renamed)
- Adds `description`, `deleted_at` columns to `agent_identities`
- Adds `revoked_at`, `revoked_by` to `agent_tool_grants`
- Adds `tool_registry` table (7 columns)
- Different status CHECK constraints (expanded enums)
- Adds `triggered_by`, `error_message`, `correlation_id` to `agent_runs`
- Adds `reasoning_summary`, `evidence_refs`, `approved_by`, `approved_at`, `blocked_reason` to `agent_intentions`

## P1 evidence

P1's agent schema matches the **018 shape exactly**:
- `agent_identities.max_autonomy` (not `max_autonomy_level`) — with CHECK constraint named `agent_identities_max_autonomy_level_check` (constraint name retains legacy suffix; column name is `max_autonomy`)
- `agent_effect_results.result` (not `result_payload`) — with CHECK constraint named `agent_effect_results_result_payload_check`
- `tool_registry` exists with 7 columns
- All expanded CHECK constraints and additional columns from 018 are present

The application code (`services/api/internal/agent/`) queries `max_autonomy` and `result`, confirming the 018 shape is what the running application expects.

## Decision

**Adopt the P1 agent schema — the 018 shape — as canonical.** The reconciled baseline will:

1. Skip migration 005's agent table definitions entirely (as the P0 fixture already does: `SKIP = {"005_agent_esaa.sql"}`).
2. Use migration 018's definitions as the canonical agent runtime schema.
3. A 005-only shape (with `max_autonomy_level`, `result_payload`, no `tool_registry`) must **fail** as an unsupported profile — the application code will not work against it.

## Constraint name anomalies

P1 retains two constraint names that reference the old column names:
- `agent_identities_max_autonomy_level_check` (on column `max_autonomy`)
- `agent_effect_results_result_payload_check` (on column `result`)

These are cosmetic artifacts. The reconciled baseline preserves them as-is (matching P1) because constraint names are part of the schema fingerprint and renaming them would create drift from P1 without functional benefit.

## No dual-column or permissive workaround

The decision does NOT:
- Create both `max_autonomy` and `max_autonomy_level` columns
- Use `IF NOT EXISTS` on 018's CREATE TABLE statements
- Attempt to merge 005 and 018 shapes

## Proof

A validation test will assert:
1. `agent_identities` has column `max_autonomy` (not `max_autonomy_level`)
2. `agent_effect_results` has column `result` (not `result_payload`)
3. `tool_registry` exists with 7 columns
4. A 005-only fixture fails with a clear error when loaded against the canonical schema
