# G2 Decision — Migration 018 (Agent Runtime) — Corrected v3

## Conflict

Migration 005 creates 5 agent tables with `IF NOT EXISTS`. Migration 018 recreates them without `IF NOT EXISTS` with different columns. P1 has a shape close to but **not identical** to 018.

## Decision

**Adopt the complete P1 agent shape as canonical.**

## Exhaustive 005↔018↔P1 differences

### Default differences (4 — P1 vs 018)

| Table.Column | 018 default | P1 default |
|---|---|---|
| `agent_identities.max_autonomy` | `DEFAULT 'A0'` | **None** |
| `agent_tool_grants.max_autonomy_level` | `DEFAULT 'A0'` | **None** |
| `agent_runs.status` | `DEFAULT 'pending'` | **None** |
| `agent_intentions.status` | `DEFAULT 'proposed'` | **`DEFAULT 'created'`** |

### Column shape differences

| Difference | 005 | 018 | P1 |
|---|---|---|---|
| `agent_identities.max_autonomy` | `max_autonomy_level` | `max_autonomy DEFAULT 'A0'` | `max_autonomy` (no default) |
| `agent_identities.metadata` | absent | `JSONB DEFAULT '{}'` | **absent** |
| `agent_identities.description` | absent | `TEXT DEFAULT ''` | present |
| `agent_identities.deleted_at` | absent | `TIMESTAMPTZ` | present |
| `agent_effect_results.result` | `result_payload` | `result` | `result` |
| `tool_registry` | absent | 7 columns | present |

### Foreign keys: 018 defines 14 inline REFERENCES; P1 has 4

**018 defines 14 inline REFERENCES clauses (column-level):**
1. `agent_identities.team_id REFERENCES teams(id)`
2. `agent_identities.created_by REFERENCES users(id)`
3. `agent_tool_grants.agent_id REFERENCES agent_identities(id)` (no cascade)
4. `agent_tool_grants.team_id REFERENCES teams(id)`
5. `agent_tool_grants.created_by REFERENCES users(id)`
6. `agent_tool_grants.revoked_by REFERENCES users(id)`
7. `agent_runs.team_id REFERENCES teams(id)`
8. `agent_runs.agent_id REFERENCES agent_identities(id)`
9. `agent_runs.triggered_by REFERENCES users(id)`
10. `agent_intentions.agent_run_id REFERENCES agent_runs(id) ON DELETE CASCADE`
11. `agent_intentions.team_id REFERENCES teams(id)`
12. `agent_intentions.approved_by REFERENCES users(id)`
13. `agent_effect_results.intention_id REFERENCES agent_intentions(id) ON DELETE CASCADE`
14. `agent_effect_results.team_id REFERENCES teams(id)`

**P1 has 4 FKs:**
- `agent_tool_grants.agent_id → agent_identities(id) ON DELETE CASCADE` — **P1 adds CASCADE that 018 does NOT have**
- `agent_runs.agent_id → agent_identities(id)` — matches (no cascade)
- `agent_intentions.agent_run_id → agent_runs(id) ON DELETE CASCADE` — matches
- `agent_effect_results.intention_id → agent_intentions(id) ON DELETE CASCADE` — matches

**10 FKs in 018 absent from P1:** all `team_id`, `created_by`, `triggered_by`, `approved_by`, `revoked_by` FKs.

**1 FK delete-action difference:** `agent_tool_grants.agent_id` — 018 has bare REFERENCES; P1 has `ON DELETE CASCADE`.

### Triggers

- `trg_agent_identities_updated_at`: **present in P1**, not defined in 018.

## Resolution

Adopt complete P1 shape. Skip 005 agent tables. 018 is source but corrected to match P1 exactly. Both 005-only and raw-018 fail validation.

## Fixture

`fixtures/018-agent-schema.sql` asserts all 4 defaults, all 10 absent FKs, all 4 retained relationships with delete actions, metadata absence, P1-only trigger, and total FK count (4).
