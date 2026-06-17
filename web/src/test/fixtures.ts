/**
 * Shared typed test fixtures. Consolidates the objects previously copy-pasted
 * across track test files (e.g. the DryRunPreview repeated 4× in track6-dryrun).
 *
 * Import from here rather than redeclaring per-file. Keep these minimal and
 * realistic — they mirror backend response shapes (see api/types.ts).
 */
import type {
  User, Permissions, Incident, WorkItem, Agent, DryRunPreview,
  Artifact, KnowledgeSearchResponse, KnowledgeCollection,
} from '@/api/types';

export const fixtures = {
  team: { id: 'team-1', name: 'Test Team', slug: 'test-team', icon: '', role: 'owner' },

  user: {
    id: 'user-1',
    email: 'owner@test.dev',
    name: 'Owner',
    active: true,
    teams: [{ id: 'team-1', name: 'Test Team', slug: 'test-team', role: 'owner' }],
  } satisfies User,

  ownerPermissions: {
    role: 'owner',
    team_id: 'team-1',
    permissions: [
      'work.items.list', 'work.items.create', 'incidents.list', 'incidents.create',
      'agents.read', 'agents.create', 'artifacts.read', 'knowledge.search',
      'knowledge.collections.read', 'team.settings.view',
    ],
  } satisfies Permissions,

  incident: {
    id: 'inc-1',
    title: 'DB connection pool exhausted',
    summary: 'Primary DB refusing connections',
    status: 'open',
    severity: 'sev1',
    impact: 'high',
    resolved_at: null,
    created_at: '2026-06-17T10:00:00Z',
    version: 1,
  } satisfies Incident,

  workItem: {
    id: 'wi-1',
    title: 'Rotate DB credentials',
    summary: 'Quarterly rotation',
    status: 'open',
    priority: 'high',
    owner_id: 'user-1',
    version: 1,
    work_item_type: 'task',
    assignee_id: 'user-1',
    project_id: null,
    created_at: '2026-06-17T10:00:00Z',
  } satisfies WorkItem,

  agent: {
    id: 'agent-1',
    name: 'remediation-executor',
    description: 'Executes approved remediation steps',
    status: 'active',
    max_autonomy: 'A4',
    created_at: '2026-06-17T10:00:00Z',
  } satisfies Agent,

  /** The DryRunPreview shape repeated verbatim across track6-dryrun tests. */
  dryRunPreview: {
    dry_run: true,
    action_type: 'proxmox.start',
    target: { asset_id: 'asset-1', name: 'test-vm', provider: 'proxmox', node: 'pve1', vmid: 100, vm_type: 'qemu' },
    risk_level: 'medium',
    requires_approval: true,
    requires_mfa: false,
    min_approvers: 1,
    mutation_window_required: true,
    mutation_window_active: false,
    feature_flag_enabled: false,
    would_create_approval: true,
    would_create_asset_action: false,
    would_call_proxmox: false,
    validation: { asset_valid: true, target_valid: true, snapshot_name_valid: true, policy_valid: true },
  } satisfies DryRunPreview,

  artifact: {
    id: 'art-1',
    artifact_type: 'document',
    title: 'Incident Report',
    description: 'Post-mortem draft',
    status: 'active',
    team_id: 'team-1',
    created_at: '2026-06-17T10:00:00Z',
    updated_at: '2026-06-17T10:00:00Z',
  } satisfies Artifact,

  knowledgeSearchResponse: {
    results: [
      {
        source_type: 'incident',
        source_id: 'inc-1',
        title: 'DB connection pool exhausted',
        summary: 'Primary DB refusing connections',
        snippet: 'Pool <mark>exhausted</mark> at 03:00 UTC',
        rank: 0.9,
        updated_at: '2026-06-17T10:00:00Z',
      },
    ],
    total: 1,
    query: 'database',
  } satisfies KnowledgeSearchResponse,

  collection: {
    id: 'col-1',
    team_id: 'team-1',
    name: 'Runbooks',
    description: 'Operational runbooks',
    created_by: 'user-1',
    created_at: '2026-06-17T10:00:00Z',
    updated_at: '2026-06-17T10:00:00Z',
    archived_at: null,
    item_count: 0,
  } satisfies KnowledgeCollection,
};

export type Fixtures = typeof fixtures;
