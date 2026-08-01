# G2 Decision — Migration 018 (Agent Runtime) — Corrected v3

## Conflict

Migration 005 creates 5 agent tables with `IF NOT EXISTS`. Migration 018 recreates them without `IF NOT EXISTS`, with different columns. P1 has a shape close to but **not identical** to 018.

## Decision

**Adopt the complete P1 agent shape as canonical.** P1 is the production authority. Both 005 and 018 are source artifacts, not the final word.

## Exhaustive 005↔018↔P1 differences (complete)

### Column differences (agent_identities)

| Column | 005 | 018 | P1 (authoritative) |
|---|---|---|---|
| `max_autonomy` / `max_autonomy_level` | `max_autonomy_level TEXT NOT NULL DEFAULT 'A0'` | `max_autonomy TEXT NOT NULL DEFAULT 'A0'` | `max_autonomy TEXT NOT NULL` (**NO default**) |
| `description` | absent | `TEXT DEFAULT ''` | `TEXT DEFAULT ''` |
| `deleted_at` | absent | `TIMESTAMPTZ` | `TIMESTAMPTZ` |
| `metadata` | absent | `JSONB DEFAULT '{}'` | **ABSENT** (018 defines it; P1 does not) |

### Column differences (agent_tool_grants)

| Column | 005 | 018 | P1 |
|---|---|---|---|
| `max_autonomy_level` | `TEXT NOT NULL DEFAULT 'A0'` | `TEXT NOT NULL DEFAULT 'A0'` | `TEXT NOT NULL` (**NO default**) |
| `revoked_at` | absent | `TIMESTAMPTZ` | `TIMESTAMPTZ` |
| `revoked_by` | absent | `UUID REFERENCES users(id)` | `UUID` (no FK) |
| `requires_approval` | `BOOLEAN NOT NULL DEFAULT TRUE` | `BOOLEAN NOT NULL DEFAULT true` | `BOOLEAN NOT NULL DEFAULT true` |
| `requires_mfa` | `BOOLEAN NOT NULL DEFAULT FALSE` | `BOOLEAN NOT NULL DEFAULT false` | `BOOLEAN NOT NULL DEFAULT false` |

### Column differences (agent_runs)

| Column | 005 | 018 | P1 |
|---|---|---|---|
| `status` | `TEXT NOT NULL CHECK (...) DEFAULT 'pending'` | `TEXT NOT NULL DEFAULT 'pending'` | `TEXT NOT NULL` (**NO default**) |
| `triggered_by` | absent | `UUID REFERENCES users(id)` | `UUID` (no FK) |
| `triggered_by_actor_type` | absent | `TEXT` with CHECK | `TEXT` with CHECK |
| `error_message` | absent | `TEXT` | `TEXT` |
| `correlation_id` | absent | `UUID` | `UUID` |
| `created_at` | absent | `TIMESTAMPTZ DEFAULT now()` | `TIMESTAMPTZ DEFAULT now()` |
| `updated_at` | absent | `TIMESTAMPTZ DEFAULT now()` | `TIMESTAMPTZ DEFAULT now()` |

### Column differences (agent_intentions)

| Column | 005 | 018 | P1 |
|---|---|---|---|
| `status` | `TEXT DEFAULT 'created'` | `TEXT NOT NULL DEFAULT 'proposed'` | `TEXT NOT NULL DEFAULT `'created'` (**P1 default differs from 018**) |
| `reasoning_summary` | absent | `TEXT DEFAULT ''` | `TEXT DEFAULT ''` |
| `evidence_refs` | absent | `JSONB DEFAULT '[]'` | `JSONB DEFAULT '[]'` |
| `approved_by` | absent | `UUID REFERENCES users(id)` | `UUID` (no FK) |
| `approved_at` | absent | `TIMESTAMPTZ` | `TIMESTAMPTZ` |
| `blocked_reason` | absent | `TEXT` | `TEXT` |

### Column differences (agent_effect_results)

| Column | 005 | 018 | P1 |
|---|---|---|---|
| `result` / `result_payload` | `result_payload JSONB` | `result JSONB NOT NULL DEFAULT '{}'` | `result JSONB NOT NULL DEFAULT '{}'` |

### Default differences (P1 vs 018)

| Table.Column | 018 default | P1 default |
|---|---|---|
| `agent_identities.max_autonomy` | `DEFAULT 'A0'` | **None** |
| `agent_tool_grants.max_autonomy_level` | `DEFAULT 'A0'` | **None** |
| `agent_runs.status` | `DEFAULT 'pending'` | **None** |
| `agent_intentions.status` | `DEFAULT 'proposed'` | **`DEFAULT 'created'`** |

### Foreign keys: 018 defines 13 inline REFERENCES; P1 has only 4 FKs

**FKs present in both 018 and P1 (4):**
- `agent_tool_grants.agent_id → agent_identities(id)` — but **P1 adds `ON DELETE CASCADE`**; 018 does **NOT** have CASCADE
- `agent_runs.agent_id → agent_identities(id)` — matches
- `agent_intentions.agent_run_id → agent_runs(id) ON DELETE CASCADE` — matches
- `agent_effect_results.intention_id → agent_intentions(id) ON DELETE CASCADE` — matches

**FKs in 018 but ABSENT from P1 (10):**
1. `agent_identities.team_id → teams(id)`
2. `agent_identities.created_by → users(id)`
3. `agent_tool_grants.team_id → teams(id)`
4. `agent_tool_grants.created_by → users(id)`
5. `agent_tool_grants.revoked_by → users(id)`
6. `agent_runs.team_id → teams(id)`
7. `agent_runs.triggered_by → users(id)`
8. `agent_intentions.team_id → teams(id)`
9. `agent_intentions.approved_by → users(id)`
10. `agent_effect_results.team_id → teams(id)`

**FK differences (1):**
- `agent_tool_grants.agent_id`: 018 has bare `REFERENCES agent_identities(id)`; P1 has `REFERENCES agent_identities(id) ON DELETE CASCADE` — P1 adds CASCADE that 018 does not define

### Triggers

- `trg_agent_identities_updated_at`: **present in P1**, not defined in 018

### tool_registry

P1 has 7 columns matching 018. ✅

## Resolution

The reconciled baseline will adopt the complete P1 agent shape. This means:

1. `max_autonomy TEXT NOT NULL` with **no default** (P1, not 018's `DEFAULT 'A0'`)
2. **No `metadata` column** (P1, not 018)
3. **10 FKs from 018 are absent** (P1 does not have `team_id`, `created_by`, `triggered_by`, `approved_by`, `revoked_by` FKs)
4. `agent_tool_grants.agent_id` has `ON DELETE CASCADE` (P1 adds this; 018 does not)
5. `agent_runs.status` has **no default** (P1, not 018's `DEFAULT 'pending'`)
6. `agent_intentions.status` defaults to **`'created'`** (P1, not 018's `'proposed'`)
7. `trg_agent_identities_updated_at` trigger is **present** (P1)
8. 005's agent tables are **skipped entirely**
9. A 005-only shape must **fail** as unsupported
10. A raw-018 shape must **fail** validation (10 missing FKs, wrong defaults, missing trigger, extra metadata column)
