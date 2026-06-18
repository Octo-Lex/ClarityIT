/**
 * API client — typed access to the ClarityIT Go backend.
 *
 * Contract preserved from the original:
 *  - Access token held in memory only; refresh token in httpOnly cookie.
 *  - Transparent 401 → refresh → retry once.
 *  - Idempotency-Key (UUID v4) on all mutations via the `mutation()` helper.
 *  - Team scoping via `teamPath()` using the stored active team id.
 *
 * Backend quirks honored (see route study):
 *  - Error envelope is {"detail": ...} everywhere EXCEPT the knowledge package,
 *    which uses {"error": ...}. We accept both.
 *  - Trailing slashes are literal for some chi routes (agents/approvals/etc.).
 *  - Timestamps are ISO-8601 strings.
 *
 * Fixes vs original:
 *  - updateDocument now sends If-Match:<updated_at> to activate the 409
 *    conflict path (the DocumentEditorPage modal was previously dead code).
 *  - request() accepts an AbortSignal so React Query can cancel in-flight
 *    queries on team switch / unmount.
 */
import type * as T from './types';

const BASE = '/api';

let accessToken: string | null = null;
let refreshPromise: Promise<boolean> | null = null;

// Token storage — memory only for access, httpOnly cookie for refresh (server-managed)
export function setAccessToken(t: string | null) { accessToken = t; }
export function getAccessToken() { return accessToken; }

// Generate UUID v4 for idempotency keys
// Fallback for non-secure contexts (HTTP) where crypto.randomUUID is unavailable
function uuid(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}

// Persist/restore team across reloads (not sensitive)
const TEAM_KEY = 'clarityit_team';
export function getStoredTeamId(): string | null { return localStorage.getItem(TEAM_KEY); }
export function setStoredTeamId(id: string | null) {
  if (id) localStorage.setItem(TEAM_KEY, id); else localStorage.removeItem(TEAM_KEY);
}

// Attempt token refresh once
async function tryRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${BASE}/auth/refresh`, { method: 'POST', credentials: 'include' });
      if (!res.ok) return false;
      const data = await res.json() as T.AuthTokens;
      accessToken = data.access_token;
      return true;
    } catch { return false; } finally { refreshPromise = null; }
  })();
  return refreshPromise;
}

// Core fetch wrapper. Supports AbortSignal for cancellation.
interface RequestOptions extends RequestInit {
  /** Extra headers to merge (e.g. If-Match). */
  signal?: AbortSignal;
}

async function request<R>(path: string, opts: RequestOptions = {}): Promise<R> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...(opts.headers as Record<string, string> ?? {}) };
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;

  let res = await fetch(`${BASE}${path}`, { ...opts, headers, credentials: 'include' });

  // Auto-refresh on 401 (only when we had a token and the request isn't itself a refresh)
  if (res.status === 401 && accessToken && !path.includes('/auth/refresh')) {
    const refreshed = await tryRefresh();
    if (refreshed && accessToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
      res = await fetch(`${BASE}${path}`, { ...opts, headers, credentials: 'include' });
    } else {
      accessToken = null;
      window.dispatchEvent(new Event('auth:logout'));
      throw new ApiError(401, 'Session expired');
    }
  }

  if (res.status === 204) return undefined as R;
  const body = await res.text();
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const parsed = JSON.parse(body);
      // Accept both {"detail": ...} (default) and {"error": ...} (knowledge pkg)
      detail = parsed.detail || parsed.error || detail;
    } catch { /* keep statusText */ }
    throw new ApiError(res.status, detail);
  }
  return body ? (JSON.parse(body) as R) : (undefined as R);
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); this.name = 'ApiError'; }
}

// Mutation helper — adds Idempotency-Key
function mutation<R>(method: string, path: string, body?: unknown): Promise<R> {
  const headers: Record<string, string> = { 'Idempotency-Key': uuid() };
  return request<R>(path, { method, headers, body: body ? JSON.stringify(body) : undefined });
}

function teamPath(path: string): string {
  const tid = getStoredTeamId();
  if (!tid) throw new Error('No active team');
  return `/teams/${tid}${path}`;
}

// ─── API ───
export const api = {
  // Bootstrap
  bootstrapStatus: () => request<T.BootstrapStatus>('/bootstrap/status').catch(() => ({ bootstrapped: false })),
  bootstrap: (data: { name: string; email: string; password: string; team_name: string }) =>
    mutation<T.BootstrapResponse>('POST', '/bootstrap', data),

  // Password reset
  forgotPassword: (email: string) =>
    request<T.ForgotPasswordResponse>('/auth/forgot-password', { method: 'POST', body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    mutation<T.MessageResponse>('POST', '/auth/reset-password', { token, password }),

  // Auth
  register: (data: { name: string; email: string; password: string }) =>
    mutation<T.AuthTokens>('POST', '/auth/register', data),
  login: (data: { email: string; password: string }) =>
    request<T.AuthTokens>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  refresh: () => request<T.AuthTokens>('/auth/refresh', { method: 'POST' }),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  me: (signal?: AbortSignal) => request<T.User>('/auth/me', { signal }),
  switchTeam: (teamId: string) => mutation<T.SwitchTeamResponse>('POST', '/auth/switch-team', { team_id: teamId }),
  permissions: (signal?: AbortSignal) => request<T.Permissions>('/auth/permissions', { signal }),
  sessions: () => request<T.Session[]>('/auth/sessions'),
  revokeSession: (id: string) => mutation<void>('DELETE', `/auth/sessions/${id}`),

  // Objects
  createObject: (data: { object_type: string; title: string; status?: string; priority?: string }) =>
    mutation<T.CreateObjectResponse>('POST', teamPath('/objects'), data),
  listObjects: (params?: Record<string, string>, signal?: AbortSignal) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<T.ObjectDetail[]>(teamPath(`/objects${qs}`), { signal });
  },
  getObject: (id: string, signal?: AbortSignal) => request<T.ObjectDetail>(teamPath(`/objects/${id}`), { signal }),
  updateObject: (id: string, data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/objects/${id}`), data),
  deleteObject: (id: string) => mutation<T.MessageResponse>('DELETE', teamPath(`/objects/${id}`)),

  // Links
  createLink: (objectId: string, data: { to_object_id: string; relation_type: string }) =>
    mutation<{ id: string }>('POST', teamPath(`/objects/${objectId}/links`), data),
  listLinks: (objectId: string) => request<T.Link[]>(teamPath(`/objects/${objectId}/links`)),
  deleteLink: (objectId: string, linkId: string) =>
    mutation<T.MessageResponse>('DELETE', teamPath(`/objects/${objectId}/links/${linkId}`)),

  // Comments
  createComment: (objectId: string, body: string) =>
    mutation<{ id: string }>('POST', teamPath(`/objects/${objectId}/comments`), { body }),
  listComments: (objectId: string) => request<T.Comment[]>(teamPath(`/objects/${objectId}/comments`)),
  updateComment: (objectId: string, commentId: string, body: string) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/objects/${objectId}/comments/${commentId}`), { body }),
  deleteComment: (objectId: string, commentId: string) =>
    mutation<T.MessageResponse>('DELETE', teamPath(`/objects/${objectId}/comments/${commentId}`)),

  // Work Items
  createWorkItem: (data: Record<string, unknown>) =>
    mutation<{ id: string }>('POST', teamPath('/work-items'), data),
  listWorkItems: (params?: Record<string, string>, signal?: AbortSignal) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<T.WorkItem[]>(teamPath(`/work-items${qs}`), { signal });
  },
  getWorkItem: (id: string) => request<T.WorkItem>(teamPath(`/work-items/${id}`)),
  updateWorkItem: (id: string, data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/work-items/${id}`), data),
  deleteWorkItem: (id: string) => mutation<T.MessageResponse>('DELETE', teamPath(`/work-items/${id}`)),
  getBoard: () => request<Record<string, T.WorkItem[]>>(teamPath('/work-items/board')),

  // Incidents
  createIncident: (data: Record<string, unknown>) =>
    mutation<{ id: string }>('POST', teamPath('/incidents'), data),
  listIncidents: (signal?: AbortSignal) => request<T.Incident[]>(teamPath('/incidents'), { signal }),
  getIncidentPatterns: (params?: { window_days?: number; min_occurrences?: number }, signal?: AbortSignal) => {
    let qs = '';
    if (params?.window_days) qs += `?window_days=${params.window_days}`;
    if (params?.min_occurrences) qs += `${qs ? '&' : '?'}min_occurrences=${params.min_occurrences}`;
    return request<T.IncidentPatternsResponse>(teamPath(`/incidents/patterns${qs}`), { signal });
  },
  getIncident: (id: string, signal?: AbortSignal) => request<T.Incident>(teamPath(`/incidents/${id}`), { signal }),
  updateIncident: (id: string, data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/incidents/${id}`), data),
  addTimeline: (id: string, body: string) =>
    mutation<{ id: string }>('POST', teamPath(`/incidents/${id}/timeline`), { body }),

  // Projects
  createProject: (data: Record<string, unknown>) =>
    mutation<{ id: string }>('POST', teamPath('/projects'), data),
  listProjects: () => request<T.Project[]>(teamPath('/projects')),
  updateProject: (id: string, data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/projects/${id}`), data),
  deleteProject: (id: string) => mutation<T.MessageResponse>('DELETE', teamPath(`/projects/${id}`)),

  // Team
  getSettings: () => request<T.TeamSettings>(teamPath('/settings')),
  updateSettings: (data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', teamPath('/settings'), data),
  listMembers: () => request<T.Member[]>(teamPath('/members')),
  updateMemberRole: (id: string, role: string) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/members/${id}`), { role }),
  removeMember: (id: string) => mutation<void>('DELETE', teamPath(`/members/${id}`)),
  createInvitation: (email: string, role: string) =>
    mutation<T.Invitation>('POST', teamPath('/invitations'), { email, role }),
  listInvitations: () => request<T.Invitation[]>(teamPath('/invitations')),
  revokeInvitation: (id: string) => mutation<void>('DELETE', teamPath(`/invitations/${id}`)),
  listAccessGrants: () => request<T.AccessGrant[]>(teamPath('/access-grants')),
  createAccessGrant: (data: Record<string, unknown>) =>
    mutation<T.AccessGrant>('POST', teamPath('/access-grants'), data),
  revokeAccessGrant: (id: string) => mutation<void>('DELETE', teamPath(`/access-grants/${id}`)),

  // Agents (trailing slashes are literal in chi)
  listAgents: (signal?: AbortSignal) => request<T.Agent[]>(teamPath('/agents/'), { signal }),
  createAgent: (data: { name: string; max_autonomy: string; description?: string }) =>
    mutation<{ id: string }>('POST', teamPath('/agents/'), data),
  getAgent: (id: string) => request<T.Agent>(teamPath(`/agents/${id}/`)),
  updateAgent: (id: string, data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', teamPath(`/agents/${id}/`), data),
  disableAgent: (id: string) => mutation<T.MessageResponse>('DELETE', teamPath(`/agents/${id}/`)),

  // Agent Grants
  listGrants: (agentId: string) => request<T.AgentGrant[]>(teamPath(`/agents/${agentId}/grants`)),
  createGrant: (agentId: string, data: { tool_name: string; max_autonomy_level: string; requires_approval?: boolean }) =>
    mutation<{ id: string }>('POST', teamPath(`/agents/${agentId}/grants`), data),
  revokeGrant: (agentId: string, grantId: string) =>
    mutation<T.MessageResponse>('DELETE', teamPath(`/agents/${agentId}/grants/${grantId}`)),

  // Agent Runs
  listRuns: () => request<T.AgentRun[]>(teamPath('/agent-runs')),
  createRun: (agentId: string) =>
    mutation<{ id: string }>('POST', teamPath('/agent-runs'), { agent_id: agentId }),
  getRun: (id: string) => request<T.AgentRun>(teamPath(`/agent-runs/${id}`)),

  // Agent Intentions
  listIntentions: (runId: string) => request<T.AgentIntention[]>(teamPath(`/agent-runs/${runId}/intentions`)),

  // Admin
  listUsers: () => request<T.AdminUser[]>('/admin/users'),
  getUser: (id: string) => request<T.AdminUser>(`/admin/users/${id}`),
  updateUser: (id: string, data: Record<string, unknown>) =>
    mutation<T.MessageResponse>('PATCH', `/admin/users/${id}`, data),
  listTeams: () => request<T.AdminTeam[]>('/admin/teams'),
  listAudit: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<T.AuditEvent[]>(`/admin/audit${qs}`);
  },

  // Ops Dashboard
  opsSummary: (signal?: AbortSignal) => request<T.OpsSummary>('/admin/ops/summary', { signal }),
  opsOutbox: () => request<T.AuditEvent[]>('/admin/ops/outbox'),
  opsDeadLetters: () => request<T.AuditEvent[]>('/admin/ops/dead-letters'),
  opsWorkers: () => request<T.WorkerInfo[]>('/admin/ops/workers'),
  opsWebhookRejections: () => request<T.AuditEvent[]>('/admin/ops/webhooks/rejections'),
  opsAgentBlocks: () => request<T.AuditEvent[]>('/admin/ops/agent-blocks'),
  backupStatus: () => request<T.BackupStatus>('/admin/backup-status'),
  deepHealth: () => request<T.DeepHealth>('/health/deep'),

  // Integration keys
  listIntegrationKeys: (teamId: string) => request<T.IntegrationKey[]>(`/teams/${teamId}/integration-keys`),
  createIntegrationKey: (teamId: string, data: { name: string; allowed_sources: string[]; allowed_scopes: string[]; allow_unsigned_dev?: boolean }) =>
    mutation<T.IntegrationKeySecret>('POST', `/teams/${teamId}/integration-keys`, data),
  revokeIntegrationKey: (teamId: string, keyId: string) =>
    mutation<T.MessageResponse>('DELETE', `/teams/${teamId}/integration-keys/${keyId}`),
  rotateIntegrationKey: (teamId: string, keyId: string) =>
    mutation<T.RotatedKey>('POST', `/teams/${teamId}/integration-keys/${keyId}/rotate`),

  // Setup status
  setupStatus: () => request<T.SetupStatus>('/admin/setup-status'),

  // ─── MFA ───
  mfaEnroll: () =>
    mutation<T.MFaEnrollResponse>('POST', '/auth/mfa/totp/enroll'),
  mfaVerifyEnrollment: (factorId: string, code: string) =>
    mutation<T.MessageResponse>('POST', '/auth/mfa/totp/verify-enrollment', { factor_id: factorId, code }),
  mfaChallenge: (password: string) =>
    mutation<T.MFAChallengeResponse>('POST', '/auth/mfa/challenge', { password }),
  mfaVerify: (challengeToken: string, code: string) =>
    mutation<T.MessageResponse>('POST', '/auth/mfa/verify', { challenge_token: challengeToken, code }),
  mfaRegenerateRecovery: () =>
    mutation<T.MFARegenerateResponse>('POST', '/auth/mfa/recovery-codes/regenerate'),
  mfaDisableFactor: (factorId: string) =>
    mutation<T.MessageResponse>('DELETE', `/auth/mfa/factors/${factorId}`),
  mfaListFactors: () =>
    request<T.MFAFactor[]>('/auth/mfa/factors'),
  mfaStatus: () =>
    request<T.MFAStatusResponse>('/auth/mfa/status'),

  // ─── WebAuthn ───
  webauthnRegisterStart: (label: string) =>
    mutation<T.WebAuthnStartResponse>('POST', '/auth/mfa/webauthn/register/start', { label }),
  webauthnRegisterFinish: async (label: string, credential: Record<string, unknown>) => {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;
    const res = await fetch(`${BASE}/auth/mfa/webauthn/register/finish?label=${encodeURIComponent(label)}`, {
      method: 'POST', headers, body: JSON.stringify(credential), credentials: 'include',
    });
    const j = await res.json(); if (!res.ok) throw new ApiError(res.status, j.detail || 'Failed'); return j;
  },
  webauthnAuthStart: () =>
    mutation<T.WebAuthnStartResponse>('POST', '/auth/mfa/webauthn/authenticate/start', {}),
  webauthnAuthFinish: async (credential: Record<string, unknown>) => {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;
    const res = await fetch(`${BASE}/auth/mfa/webauthn/authenticate/finish`, {
      method: 'POST', headers, body: JSON.stringify(credential), credentials: 'include',
    });
    const j = await res.json(); if (!res.ok) throw new ApiError(res.status, j.detail || 'Failed'); return j;
  },
  webauthnListCredentials: () =>
    request<T.WebAuthnCredential[]>('/auth/mfa/webauthn/credentials'),
  webauthnDisableCredential: async (credentialId: string) => {
    const headers: Record<string, string> = { 'Idempotency-Key': uuid() };
    if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;
    const res = await fetch(`${BASE}/auth/mfa/webauthn/credentials/${credentialId}`, {
      method: 'DELETE', headers, credentials: 'include',
    });
    const j = await res.json(); if (!res.ok) throw new ApiError(res.status, j.detail || 'Failed'); return j;
  },

  // ─── Approvals ───
  listApprovals: (status?: string) => {
    const qs = status ? `?status=${status}` : '';
    return request<T.Approval[]>(teamPath(`/approvals/${qs}`));
  },
  getApproval: (id: string) => request<T.Approval>(teamPath(`/approvals/${id}`)),
  createApproval: (data: T.CreateApprovalRequest) =>
    mutation<T.Approval>('POST', teamPath('/approvals/'), data),
  approveApproval: (id: string, reason?: string) =>
    mutation<T.Approval>('POST', teamPath(`/approvals/${id}/approve`), { reason: reason || '' }),
  rejectApproval: (id: string, reason?: string) =>
    mutation<T.Approval>('POST', teamPath(`/approvals/${id}/reject`), { reason: reason || '' }),
  cancelApproval: (id: string) =>
    mutation<T.Approval>('POST', teamPath(`/approvals/${id}/cancel`)),

  // ─── Asset Actions ───
  createAssetAction: (assetId: string, action: string, snapshotName?: string) =>
    mutation<T.AssetAction>('POST', teamPath(`/assets/${assetId}/actions/proxmox/${action}`),
      snapshotName ? { snapshot_name: snapshotName } : {}),
  dryRunAssetAction: (assetId: string, action: string, snapshotName?: string) =>
    mutation<T.DryRunPreview>('POST', teamPath(`/assets/${assetId}/actions/proxmox/${action}`),
      { dry_run: true, ...(snapshotName ? { snapshot_name: snapshotName } : {}) }),
  listAssetActions: (status?: string) => {
    const qs = status ? `?status=${status}` : '';
    return request<T.AssetAction[]>(teamPath(`/assets/asset-actions${qs}`));
  },
  getAssetAction: (id: string) => request<T.AssetAction>(teamPath(`/assets/asset-actions/${id}`)),
  executeAssetAction: (id: string) =>
    mutation<T.AssetAction>('POST', teamPath(`/assets/asset-actions/${id}/execute`)),

  // ─── Remediations ───
  listRemediations: (status?: string) => {
    const qs = status ? `?status=${status}` : '';
    return request<T.Remediation[]>(teamPath(`/remediations/${qs}`));
  },
  getRemediation: (id: string) => request<T.Remediation>(teamPath(`/remediations/${id}`)),
  createRemediation: (data: Record<string, unknown>) =>
    mutation<T.Remediation>('POST', teamPath('/remediations/'), data),
  approveRemediation: (id: string) =>
    mutation<T.Remediation>('POST', teamPath(`/remediations/${id}/approve`)),
  executeRemediation: (id: string) =>
    mutation<T.Remediation>('POST', teamPath(`/remediations/${id}/execute`)),
  cancelRemediation: (id: string) =>
    mutation<T.Remediation>('POST', teamPath(`/remediations/${id}/cancel`)),

  // v1.2 Track 6: Context Graph Quality Controls
  getContextQuality: (params?: { stale_days?: number; confidence_threshold?: number }) => {
    let qs = '';
    if (params?.stale_days) qs += `?stale_days=${params.stale_days}`;
    if (params?.confidence_threshold) qs += `${qs ? '&' : '?'}confidence_threshold=${params.confidence_threshold}`;
    return request<T.ContextQuality>(teamPath(`/context/quality${qs}`));
  },
  confirmRelation: (relationId: string, reason: string) =>
    mutation<T.ContextQuality>('POST', teamPath(`/context/relations/${relationId}/confirm`), { reason }),
  dismissRelation: (relationId: string, reason: string) =>
    mutation<T.ContextQuality>('POST', teamPath(`/context/relations/${relationId}/dismiss`), { reason }),

  // v1.2 Track 5: Post-Action Outcome Tracking
  getAssetActionOutcome: (actionId: string) =>
    request<T.ActionOutcome>(teamPath(`/asset-actions/${actionId}/outcome`)),
  saveAssetActionOutcome: (actionId: string, data: Record<string, unknown>) =>
    mutation<T.ActionOutcome>('POST', teamPath(`/asset-actions/${actionId}/outcome`), data),
  getRemediationOutcome: (proposalId: string) =>
    request<T.ActionOutcome>(teamPath(`/remediations/${proposalId}/outcome`)),
  saveRemediationOutcome: (proposalId: string, data: Record<string, unknown>) =>
    mutation<T.ActionOutcome>('POST', teamPath(`/remediations/${proposalId}/outcome`), data),

  // v1.2 Track 4: Change-Risk Scoring
  getAssetRiskScore: (assetId: string, action: string) =>
    request<T.RiskScoreResponse>(teamPath(`/assets/${assetId}/risk-score?action=${action}`)),

  // v1.2 Track 3: Approval Policy Simulation
  simulateApprovalPolicy: (data: Record<string, unknown>) =>
    mutation<T.PolicySimulationResponse>('POST', `/api/admin/approval-policy/simulate`, data),

  // v1.2 Track 7: Agent Recommendation Evaluation Harness
  runEvaluation: () =>
    mutation<Record<string, unknown>>('POST', `/api/admin/agent-evaluation/run`, {}),
  getEvaluationResults: () =>
    request<T.EvaluationResults>(`/api/admin/agent-evaluation/results`),
  getEvaluationRun: (runId: string) =>
    request<T.EvaluationResults>(`/api/admin/agent-evaluation/runs/${runId}`),

  // ─── Recommendation Evidence (v1.2 Track 1) ───
  getEvidence: (recommendationId: string) =>
    request<T.RecommendationEvidence>(teamPath(`/recommendations/${recommendationId}/evidence`)),
  getMetrics: () => request<T.AdminMetrics>('/api/admin/metrics'),

  // v1.3 Track 1: Artifacts
  listArtifacts: (params?: { type?: string; status?: string; q?: string; include_archived?: boolean }) => {
    let qs = '';
    if (params?.type) qs += `?type=${params.type}`;
    if (params?.status) qs += `${qs ? '&' : '?'}status=${params.status}`;
    if (params?.q) qs += `${qs ? '&' : '?'}q=${encodeURIComponent(params.q)}`;
    if (params?.include_archived) qs += `${qs ? '&' : '?'}include_archived=true`;
    return request<T.Artifact[]>(teamPath(`/artifacts${qs}`));
  },
  getArtifact: (artifactId: string) =>
    request<T.Artifact>(teamPath(`/artifacts/${artifactId}`)),
  createArtifact: (data: { artifact_type: string; title: string; content_markdown?: string; description?: string; source_data?: Record<string, unknown> }) =>
    mutation<T.Artifact>('POST', teamPath('/artifacts'), data),
  updateArtifact: (artifactId: string, data: Record<string, unknown>) =>
    mutation<T.Artifact>('PATCH', teamPath(`/artifacts/${artifactId}`), data),
  archiveArtifact: (artifactId: string) =>
    mutation<T.Artifact>('DELETE', teamPath(`/artifacts/${artifactId}`), {}),

  // v1.3 Track 3: Meeting Summaries
  listMeetingSummaries: () =>
    request<T.MeetingSummary[]>(teamPath('/artifacts/meeting-summaries')),
  getMeetingSummary: (id: string) =>
    request<T.MeetingSummary>(teamPath(`/artifacts/meeting-summaries/${id}`)),
  createMeetingSummary: (data: Record<string, unknown>) =>
    mutation<T.MeetingSummary>('POST', teamPath('/artifacts/meeting-summaries'), data),
  updateMeetingSummary: (id: string, data: Record<string, unknown>) =>
    mutation<T.MeetingSummary>('PATCH', teamPath(`/artifacts/meeting-summaries/${id}`), data),

  // v1.3 Track 4: Status Reports
  generateStatusReport: (data: { title: string; project_id?: string; period_start: string; period_end: string; include_sections: string[] }) =>
    mutation<T.Artifact>('POST', teamPath('/status-reports/generate'), data),

  // v1.4 Track 6: Export endpoints
  exportDocumentUrl: (artifactId: string, format: 'markdown' | 'pdf' | 'docx') =>
    teamPath(`/artifacts/${artifactId}/export/${format}`),

  // v1.4 Track 7: Document Versions
  listVersions: (artifactId: string) =>
    request<T.DocumentVersionsResponse>(teamPath(`/artifacts/documents/${artifactId}/versions`)),
  getVersion: (artifactId: string, versionId: string) =>
    request<T.DocumentVersion>(teamPath(`/artifacts/documents/${artifactId}/versions/${versionId}`)),
  restoreVersion: (artifactId: string, versionId: string) =>
    request<T.DocumentVersion>(teamPath(`/artifacts/documents/${artifactId}/versions/${versionId}/restore`), { method: 'POST' }),

  // v1.3 Track 5: Template Library
  listTemplates: (typeFilter?: string, formatFilter?: string) => {
    let qs = '';
    if (typeFilter) qs += `?type=${typeFilter}`;
    if (formatFilter) qs += `${typeFilter ? '&' : '?'}format=${formatFilter}`;
    return request<T.ArtifactTemplate[]>(teamPath(`/artifact-templates${qs}`));
  },
  createTemplate: (data: { template_type: string; name: string; content_markdown?: string; description?: string; metadata?: Record<string, unknown>; template_format?: string; document_json?: unknown; schema_version?: number }) =>
    mutation<T.ArtifactTemplate>('POST', teamPath('/artifact-templates'), data),
  instantiateTemplate: (templateId: string, data: { title?: string; description?: string; status?: string }) =>
    mutation<T.Artifact>('POST', teamPath(`/artifact-templates/${templateId}/instantiate`), data),

  // v1.3 Track 6: Storage and Recent Files
  listArtifactsWithFiles: (typeFilter?: string) => {
    let qs = 'include_files=true';
    if (typeFilter) qs += `&type=${typeFilter}`;
    return request<T.Artifact[]>(teamPath(`/artifacts?${qs}`));
  },
  getRecentArtifacts: (includeArchived?: boolean) => {
    let qs = '';
    if (includeArchived) qs = '?include_archived=true';
    return request<T.Artifact[]>(teamPath(`/artifacts/recent${qs}`));
  },
  searchArtifacts: (query: string) =>
    request<T.Artifact[]>(teamPath(`/artifacts/search?q=${encodeURIComponent(query)}`)),
  getStorageSummary: () =>
    request<T.StorageSummary>(teamPath('/artifacts/storage-summary')),

  // v1.3 Track 7: Download and Export
  downloadArtifact: (artifactId: string) =>
    request<Record<string, unknown>>(teamPath(`/artifacts/${artifactId}/download`)),
  exportArtifactUrl: (artifactId: string, format: 'markdown' | 'pdf') =>
    `${teamPath(`/artifacts/${artifactId}/export/${format}`)}`,

  // v1.4 Track 1: Native Documents
  createDocument: (data: { title: string; document_type: string; document_json: unknown; description?: string; status?: string }) =>
    mutation<T.DocumentArtifact>('POST', teamPath('/artifacts/documents'), data),
  listDocuments: (includeArchived?: boolean) => {
    const qs = includeArchived ? '?include_archived=true' : '';
    return request<T.DocumentArtifact[]>(teamPath(`/artifacts/documents${qs}`));
  },
  getDocument: (artifactId: string) =>
    request<T.DocumentArtifact>(teamPath(`/artifacts/documents/${artifactId}`)),
  /**
   * PATCH a document. Sends `If-Match: <updatedAt>` so the backend's 409
   * conflict check (artifact/document.go:651) is activated — previously this
   * was never sent, making the editor's conflict modal dead code.
   */
  updateDocument: (artifactId: string, data: { title?: string; description?: string; document_type?: string; document_json?: unknown }, updatedAt?: string) => {
    const headers: Record<string, string> = { 'Idempotency-Key': uuid() };
    if (updatedAt) headers['If-Match'] = updatedAt;
    return request<T.DocumentArtifact>(teamPath(`/artifacts/documents/${artifactId}`), {
      method: 'PATCH',
      headers,
      body: JSON.stringify(data),
    });
  },

  // v1.4 Track 3: Document Agent Assist
  documentAssist: (artifactId: string, data: { mode: string; block_id?: string; selected_text?: string; instruction?: string; document_type?: string; max_words?: number }) =>
    request<Record<string, unknown>>(teamPath(`/artifacts/documents/${artifactId}/document-assist`), {
      method: 'POST',
      headers: { 'Idempotency-Key': uuid() },
      body: JSON.stringify(data),
    }),

  // v1.4 Track 4: Generate Document
  generateDocument: (data: { title: string; document_type: string; prompt: string; tone: string; sections: string[] }) =>
    request<T.DocumentArtifact>(teamPath(`/artifacts/generate-document`), {
      method: 'POST',
      headers: { 'Idempotency-Key': uuid() },
      body: JSON.stringify(data),
    }),

  // v1.3 Track 2: Presenton
  getPresentonStatus: () =>
    request<T.PresentonStatus>(teamPath('/artifacts/presenton/status')),
  generatePresentation: (data: { title: string; content: string; num_slides: number; template?: string; tone?: string; language?: string; export_as: string; instructions?: string }) =>
    mutation<T.Artifact>('POST', teamPath('/artifacts/generate-presentation'), data),

  // v1.5 Track 1-3: Knowledge Search
  knowledgeSearch: (query: string, sourceType?: string, limit?: number, offset?: number) => {
    let qs = `q=${encodeURIComponent(query)}`;
    if (sourceType && sourceType !== 'all') qs += `&source_type=${sourceType}`;
    if (limit) qs += `&limit=${limit}`;
    if (offset) qs += `&offset=${offset}`;
    return request<T.KnowledgeSearchResponse>(teamPath(`/knowledge/search?${qs}`));
  },
  getKnowledgeItem: (itemId: string) =>
    request<T.KnowledgeItem>(teamPath(`/knowledge/${itemId}`)),
  getRelatedKnowledge: (sourceType: string, sourceId: string, limit?: number) => {
    let qs = `source_type=${sourceType}&source_id=${sourceId}`;
    if (limit) qs += `&limit=${limit}`;
    return request<T.RelatedKnowledgeResponse>(teamPath(`/knowledge/related?${qs}`));
  },
  askClarity: (question: string, sourceTypes?: string[], maxSources?: number) =>
    mutation<T.AskClarityResponse>('POST', teamPath('/knowledge/ask'), {
      question,
      source_types: sourceTypes,
      max_sources: maxSources,
    }),

  // v1.5 Track 6: Knowledge Collections
  listCollections: () =>
    request<T.CollectionsListResponse>(teamPath('/knowledge/collections')),
  createCollection: (name: string, description?: string) =>
    mutation<T.KnowledgeCollection>('POST', teamPath('/knowledge/collections'), { name, description }),
  getCollection: (collectionId: string) =>
    request<T.KnowledgeCollectionDetail>(teamPath(`/knowledge/collections/${collectionId}`)),
  patchCollection: (collectionId: string, data: { name?: string; description?: string }) =>
    mutation<T.KnowledgeCollection>('PATCH', teamPath(`/knowledge/collections/${collectionId}`), data),
  deleteCollection: (collectionId: string) =>
    mutation<{ status: string }>('DELETE', teamPath(`/knowledge/collections/${collectionId}`)),
  addCollectionItem: (collectionId: string, data: { source_type: string; source_id: string; knowledge_item_id?: string; note?: string }) =>
    mutation<T.AddCollectionItemResponse>('POST', teamPath(`/knowledge/collections/${collectionId}/items`), data),
  removeCollectionItem: (collectionId: string, itemId: string) =>
    mutation<{ status: string }>('DELETE', teamPath(`/knowledge/collections/${collectionId}/items/${itemId}`)),

  // v1.5 Track 6: Saved Answers
  saveAnswer: (data: { question: string; answer: string; confidence: string; sources: T.AskClaritySource[]; collection_id?: string }) =>
    mutation<T.SavedKnowledgeAnswer>('POST', teamPath('/knowledge/saved-answers'), data),
  listSavedAnswers: () =>
    request<T.SavedAnswersListResponse>(teamPath('/knowledge/saved-answers')),
  getSavedAnswer: (answerId: string) =>
    request<T.SavedKnowledgeAnswer>(teamPath(`/knowledge/saved-answers/${answerId}`)),
  deleteSavedAnswer: (answerId: string) =>
    mutation<{ status: string }>('DELETE', teamPath(`/knowledge/saved-answers/${answerId}`)),

  // v1.5 Track 7: Knowledge Quality
  getQualityReport: () =>
    request<T.KnowledgeQualityReport>(teamPath('/knowledge/quality')),
  getStaleItems: () =>
    request<T.StaleItemsResponse>(teamPath('/knowledge/quality/stale')),
  getDuplicateItems: () =>
    request<T.DuplicateItemsResponse>(teamPath('/knowledge/quality/duplicates')),
  getOrphanItems: () =>
    request<T.OrphanItemsResponse>(teamPath('/knowledge/quality/orphans')),
};

// Re-export types for convenience
export type * from './types';
