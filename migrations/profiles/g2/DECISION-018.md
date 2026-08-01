# G2 Decision — Migration 018 (Agent Runtime) — Corrected

## Conflict

Migration 005 creates 5 agent tables with `IF NOT EXISTS` and columns matching the 005 shape. Migration 018 recreates the same 5 tables WITHOUT `IF NOT EXISTS` (conflict on fresh install) with different columns and adds `tool_registry`. P1 has a shape that is close to but **not identical** to 018.

## Decision

**Adopt the complete P1 agent shape as canonical.** P1 is the production authority; 018 is a source artifact, not the final word.

## Exhaustive 005↔018↔P1 differences

### agent_identities

| Column | 005 | 018 | P1 (authoritative) |
|---|---|---|---|
| `max_autonomy` / `max_autonomy_level` | `max_autonomy_level TEXT NOT NULL DEFAULT 'A0'` | `max_autonomy TEXT NOT NULL DEFAULT 'A0'` | `max_autonomy TEXT NOT NULL` (NO default) |
| `description` | absent | `TEXT DEFAULT ''` | `TEXT DEFAULT ''` ✅ |
| `deleted_at` | absent | `TIMESTAMPTZ` | `TIMESTAMPTZ` ✅ |
| `metadata` | absent | `JSONB DEFAULT '{}'` | **ABSENT** (P1 does not have `metadata`; 018 defines it) |
| `created_by` | `UUID` (no FK) | `UUID REFERENCES users(id)` | `UUID` (no FK in P1) |
| `team_id` | `UUID` (no FK) | `UUID REFERENCES teams(id)` | `UUID` (no FK in P1 — **018's FK is absent**) |

**Key P1 divergences from 018:**
- `max_autonomy` has **NO default** in P1 (018 specifies `DEFAULT 'A0'`).
- `metadata` column is **absent** in P1 (018 defines it).
- `team_id` has **no FK** to `teams(id)` in P1 (018 defines the FK).
- `created_by` has **no FK** to `users(id)` in P1 (018 defines the FK).

### agent_intentions

| Property | 018 | P1 (authoritative) |
|---|---|---|
| `status` DEFAULT | `'proposed'` | **`'created'`** (P1 default differs from 018) |

### agent_tool_grants

| Property | 018 | P1 |
|---|---|---|
| `agent_id` FK delete action | `ON DELETE CASCADE` | `ON DELETE CASCADE` ✅ (matches) |

### Foreign keys

| Table | 018 defines | P1 has |
|---|---|---|
| `agent_identities` | FK to `teams(id)`, FK to `users(id)` for `created_by` | **Neither FK exists** |
| `agent_tool_grants` | FK to `agent_identities(id) ON DELETE CASCADE` | ✅ present |
| `agent_runs` | FK to `agent_identities(id)` | ✅ present |
| `agent_intentions` | FK to `agent_runs(id) ON DELETE CASCADE` | ✅ present |
| `agent_effect_results` | FK to `agent_intentions(id) ON DELETE CASCADE` | ✅ present |

### Triggers

| Table | 018 | P1 |
|---|---|---|
| `agent_identities` | `trg_agent_identities_updated_at` | ✅ present (P1 has this trigger; 018 does not define it — it was likely added by a later migration or manually) |

### tool_registry

P1 has 7 columns matching 018. ✅

## Resolution

The reconciled baseline will **adopt the complete P1 agent shape**, which means:

1. `agent_identities.max_autonomy TEXT NOT NULL` (no default — matching P1, not 018's `DEFAULT 'A0'`)
2. `agent_identities` has **no `metadata` column** (matching P1, not 018)
3. `agent_identities.team_id` and `created_by` have **no FK constraints** (matching P1)
4. `agent_intentions.status` defaults to **`'created'`** (matching P1, not 018's `'proposed'`)
5. `trg_agent_identities_updated_at` trigger is **present** (matching P1)
6. 005's agent tables are **skipped entirely** (not used as a base shape)
7. 018 is used as the **starting source** but corrected to match P1 exactly

**A 005-only shape must fail** as an unsupported profile.
**A raw 018 shape must also fail** validation against the target manifest — the baseline must match P1, not 018 verbatim.

## No dual-column or permissive workaround

- No `IF NOT EXISTS` on 018's CREATE TABLE statements
- No attempt to merge 005 and 018 shapes
- No creating both `max_autonomy` and `max_autonomy_level`
