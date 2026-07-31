-- P3 seed: minimal legacy-truth cases for migration classification tests.
-- All values are synthetic fixed literals. NO production data.

-- Bootstrap: one team, one user, one object (required by FKs)
INSERT INTO teams (id, name, slug, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'p3-synthetic-team', 'p3-synthetic', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, password_hash, name, is_active, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000002', 'p3-synthetic@example.invalid',
        'SYNTHETIC-HASH-NOT-A-REAL-CREDENTIAL', 'P3 Synthetic User', true, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO objects (id, team_id, object_type, title, status, created_by, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'asset', 'p3-synthetic-asset', 'active', '00000000-0000-0000-0000-000000000002', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 1: agent_effect_results.status='succeeded' (→ legacy_unverified)
INSERT INTO agent_identities (id, team_id, name, agent_type, status, max_autonomy, created_by, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'p3-synthetic-agent', 'assistant', 'active', 'A0', '00000000-0000-0000-0000-000000000002', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_runs (id, team_id, agent_id, triggered_by, triggered_by_actor_type, status, started_at, correlation_id, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000002', 'user', 'completed', '2026-01-01T00:00:00Z', '00000000-0000-0000-0000-000000000003', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_intentions (id, team_id, agent_run_id, intention_type, risk_level, autonomy_level, status, created_at)
VALUES ('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000003', 'action', 'medium', 'A2', 'executed', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agent_effect_results (id, team_id, intention_id, tool_name, status, approval_id, result, created_at)
VALUES ('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000006', 'synthetic_tool', 'succeeded', NULL,
        '{"synthetic": true}'::jsonb, '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 2: asset_actions succeeded WITH proxmox_task_id (→ legacy_submitted_unverified)
-- assets uses object_id (not a separate id column); asset_actions references asset_id
INSERT INTO asset_actions (id, team_id, asset_id, action_type, status, proxmox_task_id,
                           requested_by, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000003', 'proxmox.start', 'succeeded',
        'UPID:p3:0000ABC:00000000:00000000', '00000000-0000-0000-0000-000000000002', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;

-- Legacy-truth case 3: approval_requests approved (→ legacy_decision_evidence)
INSERT INTO approval_requests (id, team_id, action_type, action_target, risk_level,
                               description, status, requested_by, expires_at, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'proxmox.start',
        '{"asset_id":"00000000-0000-0000-0000-000000000003"}'::jsonb, 'medium',
        'p3 synthetic approval', 'approved', '00000000-0000-0000-0000-000000000002', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (id) DO NOTHING;
