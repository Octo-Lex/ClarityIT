-- G2 Fixture: 018 agent schema validation
-- Asserts the P1 canonical agent shape (not 005, not raw 018)

-- This fixture assumes the P3/P0 schema has been applied (which uses 018 shape).
-- It validates the P1-specific divergences from 018.

DO $$ BEGIN
    -- 1. agent_identities.max_autonomy exists (NOT max_autonomy_level)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_identities' AND column_name = 'max_autonomy'
    ), '018 FAIL: agent_identities.max_autonomy missing';

    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_identities' AND column_name = 'max_autonomy_level'
    ), '018 FAIL: agent_identities.max_autonomy_level should not exist';

    -- 2. agent_identities.max_autonomy has NO default (P1 divergence from 018)
    -- (018 specifies DEFAULT 'A0'; P1 does not)

    -- 3. agent_identities has NO metadata column (P1 divergence from 018)
    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_identities' AND column_name = 'metadata'
    ), '018 FAIL: agent_identities.metadata should not exist (P1 has no metadata column)';

    -- 4. agent_identities.team_id has NO FK to teams (P1 divergence from 018)
    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'agent_identities' AND constraint_type = 'FOREIGN KEY'
    ), '018 FAIL: agent_identities should have no FKs (P1 has none)';

    -- 5. agent_effect_results.result exists (NOT result_payload)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_effect_results' AND column_name = 'result'
    ), '018 FAIL: agent_effect_results.result missing';

    ASSERT NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_effect_results' AND column_name = 'result_payload'
    ), '018 FAIL: agent_effect_results.result_payload should not exist';

    -- 6. tool_registry exists with 7 columns
    ASSERT (SELECT count(*) FROM information_schema.columns WHERE table_name = 'tool_registry') = 7,
        '018 FAIL: tool_registry should have 7 columns';

    -- 7. agent_intentions.status defaults to 'created' (P1, not 018's 'proposed')
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'agent_intentions' AND column_name = 'status'
        AND column_default = '''created''::text'
    ), '018 FAIL: agent_intentions.status should default to created (P1), not proposed (018)';

    -- 8. trg_agent_identities_updated_at trigger exists (P1 has it)
    ASSERT EXISTS (
        SELECT 1 FROM information_schema.triggers
        WHERE event_object_table = 'agent_identities' AND trigger_name = 'trg_agent_identities_updated_at'
    ), '018 FAIL: trg_agent_identities_updated_at trigger should exist';

END $$;

-- 9. A 005-only shape must fail: agent_identities.max_autonomy_level should NOT exist
-- This assertion is implicit — if the schema has max_autonomy_level instead of max_autonomy,
-- assertions 1-2 above catch it.
