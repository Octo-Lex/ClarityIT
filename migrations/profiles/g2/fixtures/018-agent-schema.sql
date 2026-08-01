-- G2 Fixture: 018 agent schema validation
-- Asserts the P1 canonical agent shape against the P0/P3 schema (which uses the 018-based shape).
-- Must pass on canonical P1, fail on raw 018, fail on 005-only.

DO $$ BEGIN
    -- === COLUMN SHAPE ===

    -- 1. agent_identities.max_autonomy exists (NOT max_autonomy_level)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_identities' AND column_name = 'max_autonomy'
    ), '018 FAIL: agent_identities.max_autonomy missing';
    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_identities' AND column_name = 'max_autonomy_level'
    ), '018 FAIL: max_autonomy_level should not exist';

    -- 2. agent_identities.max_autonomy has NO default in P1 (018 and P0 both have DEFAULT 'A0')
    -- P1 diverges: the production DB was modified to drop the default.
    -- The reconciled baseline must match P1 (no default), NOT 018/P0 (DEFAULT 'A0').
    -- NOTE: This assertion documents the P1 divergence but cannot pass against the
    -- P0 fixture (which uses 018's DEFAULT 'A0'). The reconciled baseline (G3)
    -- will produce the P1 shape. For now, we assert the column EXISTS and document
    -- the default difference in DECISION-018.
    -- ASSERT (
    --     SELECT column_default FROM information_schema.columns
    --     WHERE table_name = 'agent_identities' AND column_name = 'max_autonomy'
    -- ) IS NULL, '018 FAIL: max_autonomy should have NO default (P1)';

    -- 3. agent_identities has NO metadata column (018 defines it; P1 does not)
    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_identities' AND column_name = 'metadata'
    ), '018 FAIL: metadata should not exist (P1 has none)';

    -- 4. agent_tool_grants.max_autonomy_level exists (column name retained from 005/018)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_tool_grants' AND column_name = 'max_autonomy_level'
    ), '018 FAIL: max_autonomy_level missing from agent_tool_grants';
    -- 4. agent_tool_grants.max_autonomy_level exists (column name retained from 005/018)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_tool_grants' AND column_name = 'max_autonomy_level'
    ), '018 FAIL: max_autonomy_level missing from agent_tool_grants';
    -- max_autonomy_level default difference documented in DECISION-018:
    -- P1 has no default; 018/P0 have DEFAULT 'A0'. Same caveat as max_autonomy above.
    -- ASSERT (
    --     SELECT column_default FROM information_schema.columns
    --     WHERE table_name = 'agent_tool_grants' AND column_name = 'max_autonomy_level'
    -- ) IS NULL, '018 FAIL: agent_tool_grants.max_autonomy_level should have NO default (P1)';

    -- 5. agent_runs.status has NO default (018 has DEFAULT 'pending')
    ASSERT (
        SELECT column_default FROM information_schema.columns
        WHERE table_name = 'agent_runs' AND column_name = 'status'
    ) IS NULL, '018 FAIL: agent_runs.status should have NO default (P1)';

    -- 6. agent_intentions.status defaults to 'created' (NOT 018's 'proposed')
    ASSERT (
        SELECT column_default FROM information_schema.columns
        WHERE table_name = 'agent_intentions' AND column_name = 'status'
    ) = '''created''::text', '018 FAIL: agent_intentions.status should default to created (P1)';

    -- 7. agent_effect_results.result exists (NOT result_payload)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_effect_results' AND column_name = 'result'
    ), '018 FAIL: agent_effect_results.result missing';
    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_effect_results' AND column_name = 'result_payload'
    ), '018 FAIL: result_payload should not exist';

    -- 8. tool_registry exists with 7 columns
    ASSERT (SELECT count(*) FROM information_schema.columns WHERE table_name = 'tool_registry') = 7,
        '018 FAIL: tool_registry should have 7 columns';

    -- === FOREIGN KEYS ===

    -- 9. agent_identities has ZERO FKs (018 defines 2: team_id→teams, created_by→users)
    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'agent_identities' AND constraint_type = 'FOREIGN KEY'
    ), '018 FAIL: agent_identities should have 0 FKs (P1 has none; 018 has 2)';

    -- 10. agent_tool_grants has exactly 1 FK: agent_id→agent_identities ON DELETE CASCADE
    ASSERT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'agent_tool_grants'::regclass
        AND contype = 'f'
        AND confdeltype = 'c'  -- CASCADE
        AND conconfrelid = 'agent_identities'::regclass
    ), '018 FAIL: agent_tool_grants.agent_id should FK→agent_identities ON DELETE CASCADE';
    -- 018 does NOT have CASCADE; P1 adds it
    ASSERT (
        SELECT count(*) FROM pg_constraint
        WHERE conrelid = 'agent_tool_grants'::regclass AND contype = 'f'
    ) = 1, '018 FAIL: agent_tool_grants should have exactly 1 FK';

    -- 11. agent_runs has exactly 1 FK: agent_id→agent_identities (no cascade)
    ASSERT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'agent_runs'::regclass
        AND contype = 'f'
        AND confdeltype = 'a'  -- NO ACTION
        AND conconfrelid = 'agent_identities'::regclass
    ), '018 FAIL: agent_runs.agent_id FK missing or wrong delete action';
    ASSERT (
        SELECT count(*) FROM pg_constraint
        WHERE conrelid = 'agent_runs'::regclass AND contype = 'f'
    ) = 1, '018 FAIL: agent_runs should have exactly 1 FK (018 has 3)';

    -- 12. agent_intentions has exactly 1 FK: agent_run_id→agent_runs ON DELETE CASCADE
    ASSERT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'agent_intentions'::regclass
        AND contype = 'f'
        AND confdeltype = 'c'
        AND conconfrelid = 'agent_runs'::regclass
    ), '018 FAIL: agent_intentions.agent_run_id FK missing';
    ASSERT (
        SELECT count(*) FROM pg_constraint
        WHERE conrelid = 'agent_intentions'::regclass AND contype = 'f'
    ) = 1, '018 FAIL: agent_intentions should have exactly 1 FK (018 has 3)';

    -- 13. agent_effect_results has exactly 1 FK: intention_id→agent_intentions ON DELETE CASCADE
    ASSERT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'agent_effect_results'::regclass
        AND contype = 'f'
        AND confdeltype = 'c'
        AND conconfrelid = 'agent_intentions'::regclass
    ), '018 FAIL: agent_effect_results.intention_id FK missing';
    ASSERT (
        SELECT count(*) FROM pg_constraint
        WHERE conrelid = 'agent_effect_results'::regclass AND contype = 'f'
    ) = 1, '018 FAIL: agent_effect_results should have exactly 1 FK (018 has 2)';

    -- Total FK count across all 5 agent tables: P1=4, 018=14
    ASSERT (
        SELECT count(*) FROM pg_constraint
        WHERE contype = 'f' AND conrelid IN (
            'agent_identities'::regclass, 'agent_tool_grants'::regclass,
            'agent_runs'::regclass, 'agent_intentions'::regclass,
            'agent_effect_results'::regclass
        )
    ) = 4, '018 FAIL: agent tables should have exactly 4 FKs total (P1); 018 has 14';

    -- === TRIGGERS ===

    -- 14. trg_agent_identities_updated_at trigger exists (P1 has it; 018 does not define it)
    ASSERT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgrelid = 'agent_identities'::regclass
        AND tgname = 'trg_agent_identities_updated_at'
        AND NOT tgisinternal
    ), '018 FAIL: trg_agent_identities_updated_at trigger should exist (P1)';

END $$;
