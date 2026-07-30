-- Deterministic principals expected by the repository backend tests.
-- CI fixture only: never apply to a deployed environment.
INSERT INTO users (id, email, password_hash, name, is_active)
VALUES
    (
        '00000000-0000-4000-8000-000000000001',
        'owner@test.dev',
        '$2b$12$kk9Bf9hHSREL6M6wGBht0O0To3JstSR16dGXGj1DzH1uBtTnBaPG.',
        'Test Owner',
        TRUE
    ),
    (
        '00000000-0000-4000-8000-000000000002',
        'member@test.dev',
        '$2b$12$kk9Bf9hHSREL6M6wGBht0O0To3JstSR16dGXGj1DzH1uBtTnBaPG.',
        'Test Member',
        TRUE
    );

INSERT INTO teams (id, name, slug)
VALUES (
    '00000000-0000-4000-8000-000000000010',
    'Test Team',
    'test-team'
);

INSERT INTO team_memberships (id, user_id, team_id, role_id)
SELECT
    '00000000-0000-4000-8000-000000000011',
    '00000000-0000-4000-8000-000000000001',
    '00000000-0000-4000-8000-000000000010',
    id
FROM roles
WHERE name = 'owner';

INSERT INTO team_memberships (id, user_id, team_id, role_id)
SELECT
    '00000000-0000-4000-8000-000000000012',
    '00000000-0000-4000-8000-000000000002',
    '00000000-0000-4000-8000-000000000010',
    id
FROM roles
WHERE name = 'member';

INSERT INTO user_platform_roles (
    id,
    user_id,
    platform_role_id,
    granted_by
)
SELECT
    '00000000-0000-4000-8000-000000000013',
    '00000000-0000-4000-8000-000000000001',
    id,
    '00000000-0000-4000-8000-000000000001'
FROM platform_roles
WHERE name = 'platform_owner';

UPDATE bootstrap_lock
SET is_locked = TRUE,
    locked_by_user_id = '00000000-0000-4000-8000-000000000001',
    locked_at = NOW()
WHERE id = 1;

DO $$
BEGIN
    ASSERT (
        SELECT COUNT(*)
        FROM users
        WHERE email IN ('owner@test.dev', 'member@test.dev')
    ) = 2, 'P0 test users missing';
    ASSERT EXISTS (
        SELECT 1
        FROM user_platform_roles upr
        JOIN users u ON u.id = upr.user_id
        JOIN platform_roles pr ON pr.id = upr.platform_role_id
        WHERE u.email = 'owner@test.dev'
          AND pr.name = 'platform_owner'
          AND upr.revoked_at IS NULL
    ), 'P0 platform owner assignment missing';
END $$;
