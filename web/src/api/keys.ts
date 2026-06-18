/**
 * Query-key factory — the single source of truth for React Query cache keys.
 *
 * Keys are hierarchical so invalidation can be precise or broad:
 *   invalidateQueries({ queryKey: keys.incidents.list(teamId) })  // one list
 *   invalidateQueries({ queryKey: keys.team(teamId) })            // everything for a team
 *   invalidateQueries({ queryKey: ['teams', teamId] })            // equivalent
 *
 * Shape: ['teams', teamId, <domain>, <...args>]  for team-scoped queries
 *        ['global', <domain>, <...args>]         for non-team-scoped (admin/auth)
 *
 * Do NOT construct query keys ad-hoc in feature code. Import from here.
 */

const GLOBAL = 'global' as const;

/** Top-level team scope — invalidate this to refetch everything for a team. */
export const team = (teamId: string) => ['teams', teamId] as const;

export const keys = {
  /** Bootstrap (public) */
  bootstrapStatus: () => [GLOBAL, 'bootstrap-status'] as const,

  /** Auth / session (non-team-scoped) */
  me: () => [GLOBAL, 'me'] as const,
  permissions: () => [GLOBAL, 'permissions'] as const,
  sessions: () => [GLOBAL, 'sessions'] as const,

  /** Team scope root */
  team,

  /** Team settings & membership */
  settings: (teamId: string) => ['teams', teamId, 'settings'] as const,
  members: (teamId: string) => ['teams', teamId, 'members'] as const,
  invitations: (teamId: string) => ['teams', teamId, 'invitations'] as const,
  accessGrants: (teamId: string) => ['teams', teamId, 'access-grants'] as const,

  /** Universal object spine */
  objects: {
    list: (teamId: string, params?: Record<string, string>) =>
      ['teams', teamId, 'objects', 'list', params ?? {}] as const,
    detail: (teamId: string, objectId: string) =>
      ['teams', teamId, 'objects', 'detail', objectId] as const,
    links: (teamId: string, objectId: string) =>
      ['teams', teamId, 'objects', objectId, 'links'] as const,
    comments: (teamId: string, objectId: string) =>
      ['teams', teamId, 'objects', objectId, 'comments'] as const,
    attachments: (teamId: string, objectId: string) =>
      ['teams', teamId, 'objects', objectId, 'attachments'] as const,
  },

  /** Work items & board */
  workItems: {
    list: (teamId: string, params?: Record<string, string>) =>
      ['teams', teamId, 'work-items', 'list', params ?? {}] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'work-items', 'detail', id] as const,
    board: (teamId: string) => ['teams', teamId, 'work-items', 'board'] as const,
  },

  /** Incidents */
  incidents: {
    list: (teamId: string) => ['teams', teamId, 'incidents', 'list'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'incidents', 'detail', id] as const,
    patterns: (teamId: string, params?: { window_days?: number; min_occurrences?: number }) =>
      ['teams', teamId, 'incidents', 'patterns', params ?? {}] as const,
  },

  /** Projects */
  projects: {
    list: (teamId: string) => ['teams', teamId, 'projects', 'list'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'projects', 'detail', id] as const,
  },

  /** Agent runtime */
  agents: {
    list: (teamId: string) => ['teams', teamId, 'agents', 'list'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'agents', 'detail', id] as const,
    grants: (teamId: string, agentId: string) =>
      ['teams', teamId, 'agents', agentId, 'grants'] as const,
  },
  agentRuns: {
    list: (teamId: string) => ['teams', teamId, 'agent-runs', 'list'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'agent-runs', 'detail', id] as const,
    intentions: (teamId: string, runId: string) =>
      ['teams', teamId, 'agent-runs', runId, 'intentions'] as const,
  },

  /** Approvals */
  approvals: {
    list: (teamId: string, status?: string) =>
      ['teams', teamId, 'approvals', 'list', status ?? 'all'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'approvals', 'detail', id] as const,
  },

  /** Integrations & assets */
  integrationKeys: (teamId: string) =>
    ['teams', teamId, 'integration-keys'] as const,
  proxmoxStatus: (teamId: string) =>
    ['teams', teamId, 'proxmox', 'status'] as const,
  assets: {
    list: (teamId: string) => ['teams', teamId, 'assets', 'list'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'assets', 'detail', id] as const,
    riskScore: (teamId: string, assetId: string, action: string) =>
      ['teams', teamId, 'assets', assetId, 'risk-score', action] as const,
  },
  assetActions: {
    list: (teamId: string, status?: string) =>
      ['teams', teamId, 'asset-actions', 'list', status ?? 'all'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'asset-actions', 'detail', id] as const,
    outcome: (teamId: string, actionId: string) =>
      ['teams', teamId, 'asset-actions', actionId, 'outcome'] as const,
  },

  /** Remediations */
  remediations: {
    list: (teamId: string, status?: string) =>
      ['teams', teamId, 'remediations', 'list', status ?? 'all'] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'remediations', 'detail', id] as const,
    outcome: (teamId: string, id: string) =>
      ['teams', teamId, 'remediations', id, 'outcome'] as const,
  },
  evidence: (teamId: string, recommendationId: string) =>
    ['teams', teamId, 'recommendations', recommendationId, 'evidence'] as const,
  contextQuality: (teamId: string, params?: { stale_days?: number; confidence_threshold?: number }) =>
    ['teams', teamId, 'context', 'quality', params ?? {}] as const,

  /** Artifacts & documents */
  artifacts: {
    list: (teamId: string, params?: { type?: string; status?: string; q?: string; include_archived?: boolean }) =>
      ['teams', teamId, 'artifacts', 'list', params ?? {}] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'artifacts', 'detail', id] as const,
    recent: (teamId: string, includeArchived?: boolean) =>
      ['teams', teamId, 'artifacts', 'recent', { includeArchived }] as const,
    search: (teamId: string, query: string) =>
      ['teams', teamId, 'artifacts', 'search', query] as const,
    storageSummary: (teamId: string) =>
      ['teams', teamId, 'artifacts', 'storage-summary'] as const,
    meetingSummaries: (teamId: string) =>
      ['teams', teamId, 'artifacts', 'meeting-summaries'] as const,
    meetingSummary: (teamId: string, id: string) =>
      ['teams', teamId, 'artifacts', 'meeting-summaries', id] as const,
    templates: (teamId: string, type?: string, format?: string) =>
      ['teams', teamId, 'artifacts', 'templates', { type, format }] as const,
  },
  documents: {
    list: (teamId: string, includeArchived?: boolean) =>
      ['teams', teamId, 'documents', 'list', { includeArchived }] as const,
    detail: (teamId: string, id: string) =>
      ['teams', teamId, 'documents', 'detail', id] as const,
    versions: (teamId: string, id: string) =>
      ['teams', teamId, 'documents', id, 'versions'] as const,
    version: (teamId: string, id: string, versionId: string) =>
      ['teams', teamId, 'documents', id, 'versions', versionId] as const,
  },
  presentonStatus: (teamId: string) =>
    ['teams', teamId, 'presenton', 'status'] as const,

  /** Knowledge */
  knowledge: {
    search: (teamId: string, query: string, sourceType?: string) =>
      ['teams', teamId, 'knowledge', 'search', { query, sourceType }] as const,
    detail: (teamId: string, itemId: string) =>
      ['teams', teamId, 'knowledge', 'detail', itemId] as const,
    indexStatus: (teamId: string) =>
      ['teams', teamId, 'knowledge', 'index-status'] as const,
    related: (teamId: string, sourceType: string, sourceId: string) =>
      ['teams', teamId, 'knowledge', 'related', { sourceType, sourceId }] as const,
    quality: (teamId: string) =>
      ['teams', teamId, 'knowledge', 'quality'] as const,
    qualityStale: (teamId: string) =>
      ['teams', teamId, 'knowledge', 'quality', 'stale'] as const,
    qualityDuplicates: (teamId: string) =>
      ['teams', teamId, 'knowledge', 'quality', 'duplicates'] as const,
    qualityOrphans: (teamId: string) =>
      ['teams', teamId, 'knowledge', 'quality', 'orphans'] as const,
    collections: {
      list: (teamId: string) =>
        ['teams', teamId, 'knowledge', 'collections', 'list'] as const,
      detail: (teamId: string, id: string) =>
        ['teams', teamId, 'knowledge', 'collections', 'detail', id] as const,
    },
    savedAnswers: {
      list: (teamId: string) =>
        ['teams', teamId, 'knowledge', 'saved-answers', 'list'] as const,
      detail: (teamId: string, id: string) =>
        ['teams', teamId, 'knowledge', 'saved-answers', 'detail', id] as const,
    },
  },

  /** MFA */
  mfaStatus: () => [GLOBAL, 'mfa', 'status'] as const,
  mfaFactors: () => [GLOBAL, 'mfa', 'factors'] as const,
  webauthnCredentials: () => [GLOBAL, 'webauthn', 'credentials'] as const,

  /** Admin (platform-owner, non-team-scoped) */
  admin: {
    users: () => [GLOBAL, 'admin', 'users'] as const,
    user: (id: string) => [GLOBAL, 'admin', 'users', id] as const,
    teams: () => [GLOBAL, 'admin', 'teams'] as const,
    audit: (params?: Record<string, string>) =>
      [GLOBAL, 'admin', 'audit', params ?? {}] as const,
    settings: () => [GLOBAL, 'admin', 'settings'] as const,
    setupStatus: () => [GLOBAL, 'admin', 'setup-status'] as const,
    metrics: () => [GLOBAL, 'admin', 'metrics'] as const,
    ops: {
      summary: () => [GLOBAL, 'admin', 'ops', 'summary'] as const,
      outbox: () => [GLOBAL, 'admin', 'ops', 'outbox'] as const,
      deadLetters: () => [GLOBAL, 'admin', 'ops', 'dead-letters'] as const,
      workers: () => [GLOBAL, 'admin', 'ops', 'workers'] as const,
      webhookRejections: () => [GLOBAL, 'admin', 'ops', 'webhooks', 'rejections'] as const,
      agentBlocks: () => [GLOBAL, 'admin', 'ops', 'agent-blocks'] as const,
    },
    backupStatus: () => [GLOBAL, 'admin', 'backup-status'] as const,
    deepHealth: () => [GLOBAL, 'health', 'deep'] as const,
    knowledgeIndexStatus: () => [GLOBAL, 'admin', 'knowledge', 'index-status'] as const,
    evaluationResults: () => [GLOBAL, 'admin', 'agent-evaluation', 'results'] as const,
    evaluationRun: (runId: string) =>
      [GLOBAL, 'admin', 'agent-evaluation', 'runs', runId] as const,
    mutationWindow: () => [GLOBAL, 'admin', 'proxmox', 'mutation-window'] as const,
  },
} as const;

export type QueryKeys = typeof keys;
