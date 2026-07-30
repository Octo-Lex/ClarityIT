-- Minimum invariants required by the repository-derived backend test schema.
DO $$
BEGIN
    ASSERT NOT EXISTS (SELECT 1 FROM permissions WHERE name LIKE '%.edit%'),
        'legacy edit permissions remain';
    ASSERT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'agent_identities'
          AND column_name = 'max_autonomy'
    ), 'agent_identities.max_autonomy missing';
    ASSERT NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'agent_identities'
          AND column_name = 'max_autonomy_level'
    ), 'obsolete agent_identities.max_autonomy_level remains';
    ASSERT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'agent_effect_results'
          AND column_name = 'result'
    ), 'agent_effect_results.result missing';
    ASSERT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'tool_registry'
    ), 'tool_registry missing';
    ASSERT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clarityit_app'),
        'clarityit_app test role missing';
    ASSERT has_table_privilege('clarityit_app', 'public.recommendation_evidence', 'SELECT'),
        'clarityit_app lacks SELECT on recommendation_evidence';
    ASSERT has_table_privilege('clarityit_app', 'public.recommendation_evidence', 'INSERT'),
        'clarityit_app lacks INSERT on recommendation_evidence';
    ASSERT has_table_privilege('clarityit_app', 'public.recommendation_evidence', 'UPDATE'),
        'clarityit_app lacks UPDATE on recommendation_evidence';
END $$;
