// API client — tokens in memory, auto-refresh on 401, idempotency on mutations
const BASE = '/api';

let accessToken: string | null = null;
let refreshPromise: Promise<boolean> | null = null;

// Token storage — memory only for access, httpOnly cookie for refresh (server-managed)
export function setAccessToken(t: string | null) { accessToken = t; }
export function getAccessToken() { return accessToken; }

// Generate UUID v4 for idempotency keys
function uuid(): string {
  return crypto.randomUUID();
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
      const data = await res.json();
      accessToken = data.access_token;
      return true;
    } catch { return false; } finally { refreshPromise = null; }
  })();
  return refreshPromise;
}

// Core fetch wrapper
async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...(opts.headers as any ?? {}) };
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;

  let res = await fetch(`${BASE}${path}`, { ...opts, headers, credentials: 'include' });

  // Auto-refresh on 401
  if (res.status === 401 && accessToken) {
    const refreshed = await tryRefresh();
    if (refreshed && accessToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
      res = await fetch(`${BASE}${path}`, { ...opts, headers, credentials: 'include' });
    } else {
      accessToken = null;
      window.dispatchEvent(new Event('auth:logout'));
      throw new Error('Session expired');
    }
  }

  if (res.status === 204) return undefined as T;
  const body = await res.text();
  if (!res.ok) {
    let detail = res.statusText;
    try { detail = JSON.parse(body).detail || detail; } catch {}
    throw new ApiError(res.status, detail);
  }
  return body ? JSON.parse(body) : (undefined as T);
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); this.name = 'ApiError'; }
}

// Mutation helper — adds Idempotency-Key
function mutation<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Idempotency-Key': uuid() };
  return request<T>(path, { method, headers, body: body ? JSON.stringify(body) : undefined });
}

function teamPath(path: string): string {
  const tid = getStoredTeamId();
  if (!tid) throw new Error('No active team');
  return `/teams/${tid}${path}`;
}

// ─── Types ───
export interface User {
  id: string; email: string; name: string;
  active: boolean;
  teams: { id: string; name: string; slug: string; role: string }[];
}

export interface Permissions { role: string; team_id: string; permissions: string[]; }

export interface WorkItem {
  id: string; title: string; summary: string; status: string; priority: string;
  owner_id: string | null; version: number; work_item_type: string;
  assignee_id: string | null; project_id: string | null; created_at: string;
}

export interface Incident {
  id: string; title: string; summary: string; status: string;
  severity: string; impact: string; resolved_at: string | null;
  created_at: string; version: number;
}

export interface ObjectDetail {
  id: string; team_id: string; object_type: string; title: string; summary: string;
  status: string; priority: string; owner_id: string | null; created_by: string | null;
  version: number; metadata: Record<string, unknown>; created_at: string; updated_at: string;
}

export interface Comment {
  id: string; author_id: string; body: string; created_at: string; updated_at: string | null;
}

export interface Link {
  id: string; from_object_id: string; to_object_id: string;
  relation_type: string; created_at: string;
}

export interface Member {
  user_id: string; name: string; email: string; role: string; joined_at: string;
}

export interface Invitation {
  id: string; email: string; role: string; invited_by: string;
  expires_at: string; accepted_at: string | null; created_at: string; status: string;
}

export interface AuditEvent {
  id: string; actor_id: string; action: string; entity_type: string;
  entity_id: string; new_value: Record<string, unknown>; created_at: string;
}

export interface Project {
  id: string; title: string; summary: string; status: string; version: number; created_at: string;
}

export interface Agent {
  id: string; name: string; description: string; status: string; max_autonomy: string; created_at: string;
}

export interface AgentGrant {
  id: string; tool_name: string; max_autonomy_level: string; requires_approval: boolean;
  requires_mfa: boolean; expires_at: string | null; created_at: string; revoked_at: string | null;
}

export interface AgentRun {
  id: string; agent_id: string; status: string; triggered_by: string; created_at: string;
  started_at?: string; completed_at?: string | null; error_message?: string | null;
}

export interface AgentIntention {
  id: string; intention_type: string; requested_tool: string; confidence: number;
  risk_level: string; autonomy_level: string; status: string; blocked_reason: string | null;
  created_at: string;
}

// ─── API ───
export const api = {
  // Bootstrap
  bootstrapStatus: () => request<{ bootstrapped: boolean }>('/bootstrap/status').catch(() => ({ bootstrapped: false })),
  bootstrap: (data: { name: string; email: string; password: string; team_name: string }) =>
    mutation<{ access_token: string }>('POST', '/bootstrap', data),

  // Password reset
  forgotPassword: (email: string) =>
    request<{ message: string }>('/auth/forgot-password', { method: 'POST', body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    mutation<{ message: string }>('POST', '/auth/reset-password', { token, password }),

  // Auth
  register: (data: { name: string; email: string; password: string }) =>
    mutation<{ access_token: string }>('POST', '/auth/register', data),
  login: (data: { email: string; password: string }) =>
    request<{ access_token: string }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  refresh: () => request<{ access_token: string }>('/auth/refresh', { method: 'POST' }),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  me: () => request<User>('/auth/me'),
  switchTeam: (teamId: string) => mutation<{ access_token: string }>('POST', '/auth/switch-team', { team_id: teamId }),
  permissions: () => request<Permissions>('/auth/permissions'),

  // Objects
  createObject: (data: { object_type: string; title: string; status?: string; priority?: string }) =>
    mutation<{ id: string }>('POST', teamPath('/objects'), data),
  listObjects: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<ObjectDetail[]>(teamPath(`/objects${qs}`));
  },
  getObject: (id: string) => request<ObjectDetail>(teamPath(`/objects/${id}`)),
  updateObject: (id: string, data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', teamPath(`/objects/${id}`), data),
  deleteObject: (id: string) => mutation<{ message: string }>('DELETE', teamPath(`/objects/${id}`)),

  // Links
  createLink: (objectId: string, data: { to_object_id: string; relation_type: string }) =>
    mutation<{ id: string }>('POST', teamPath(`/objects/${objectId}/links`), data),
  listLinks: (objectId: string) => request<Link[]>(teamPath(`/objects/${objectId}/links`)),
  deleteLink: (objectId: string, linkId: string) =>
    mutation<{ message: string }>('DELETE', teamPath(`/objects/${objectId}/links/${linkId}`)),

  // Comments
  createComment: (objectId: string, body: string) =>
    mutation<{ id: string }>('POST', teamPath(`/objects/${objectId}/comments`), { body }),
  listComments: (objectId: string) => request<Comment[]>(teamPath(`/objects/${objectId}/comments`)),
  updateComment: (objectId: string, commentId: string, body: string) =>
    mutation<{ message: string }>('PATCH', teamPath(`/objects/${objectId}/comments/${commentId}`), { body }),
  deleteComment: (objectId: string, commentId: string) =>
    mutation<{ message: string }>('DELETE', teamPath(`/objects/${objectId}/comments/${commentId}`)),

  // Work Items
  createWorkItem: (data: Record<string, unknown>) =>
    mutation<{ id: string }>('POST', teamPath('/work-items'), data),
  listWorkItems: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<WorkItem[]>(teamPath(`/work-items${qs}`));
  },
  getWorkItem: (id: string) => request<WorkItem>(teamPath(`/work-items/${id}`)),
  updateWorkItem: (id: string, data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', teamPath(`/work-items/${id}`), data),
  deleteWorkItem: (id: string) => mutation<{ message: string }>('DELETE', teamPath(`/work-items/${id}`)),
  getBoard: () => request<Record<string, WorkItem[]>>(teamPath('/work-items/board')),

  // Incidents
  createIncident: (data: Record<string, unknown>) =>
    mutation<{ id: string }>('POST', teamPath('/incidents'), data),
  listIncidents: () => request<Incident[]>(teamPath('/incidents')),
  getIncident: (id: string) => request<Incident>(teamPath(`/incidents/${id}`)),
  updateIncident: (id: string, data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', teamPath(`/incidents/${id}`), data),
  addTimeline: (id: string, body: string) =>
    mutation<{ id: string }>('POST', teamPath(`/incidents/${id}/timeline`), { body }),

  // Projects
  createProject: (data: Record<string, unknown>) =>
    mutation<{ id: string }>('POST', teamPath('/projects'), data),
  listProjects: () => request<Project[]>(teamPath('/projects')),
  updateProject: (id: string, data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', teamPath(`/projects/${id}`), data),
  deleteProject: (id: string) => mutation<{ message: string }>('DELETE', teamPath(`/projects/${id}`)),

  // Team
  getSettings: () => request<any>(teamPath('/settings')),
  updateSettings: (data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', teamPath('/settings'), data),
  listMembers: () => request<Member[]>(teamPath('/members')),
  updateMemberRole: (id: string, role: string) =>
    mutation<{ message: string }>('PATCH', teamPath(`/members/${id}`), { role }),
  removeMember: (id: string) => mutation<void>('DELETE', teamPath(`/members/${id}`)),
  createInvitation: (email: string, role: string) =>
    mutation<any>('POST', teamPath('/invitations'), { email, role }),
  listInvitations: () => request<Invitation[]>(teamPath('/invitations')),
  revokeInvitation: (id: string) => mutation<void>('DELETE', teamPath(`/invitations/${id}`)),
  listAccessGrants: () => request<any[]>(teamPath('/access-grants')),
  createAccessGrant: (data: Record<string, unknown>) =>
    mutation<any>('POST', teamPath('/access-grants'), data),
  revokeAccessGrant: (id: string) => mutation<void>('DELETE', teamPath(`/access-grants/${id}`)),

  // Agents
  listAgents: () => request<Agent[]>(teamPath('/agents')),
  createAgent: (data: { name: string; max_autonomy: string; description?: string }) =>
    mutation<{ id: string }>('POST', teamPath('/agents'), data),
  getAgent: (id: string) => request<Agent>(teamPath(`/agents/${id}`)),
  updateAgent: (id: string, data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', teamPath(`/agents/${id}`), data),
  disableAgent: (id: string) => mutation<{ message: string }>('DELETE', teamPath(`/agents/${id}`)),

  // Agent Grants
  listGrants: (agentId: string) => request<AgentGrant[]>(teamPath(`/agents/${agentId}/grants`)),
  createGrant: (agentId: string, data: { tool_name: string; max_autonomy_level: string; requires_approval?: boolean }) =>
    mutation<{ id: string }>('POST', teamPath(`/agents/${agentId}/grants`), data),
  revokeGrant: (agentId: string, grantId: string) =>
    mutation<{ message: string }>('DELETE', teamPath(`/agents/${agentId}/grants/${grantId}`)),

  // Agent Runs
  listRuns: () => request<AgentRun[]>(teamPath('/agent-runs')),
  createRun: (agentId: string) =>
    mutation<{ id: string }>('POST', teamPath('/agent-runs'), { agent_id: agentId }),
  getRun: (id: string) => request<AgentRun>(teamPath(`/agent-runs/${id}`)),

  // Agent Intentions
  listIntentions: (runId: string) => request<AgentIntention[]>(teamPath(`/agent-runs/${runId}/intentions`)),

  // Admin
  listUsers: () => request<any[]>('/admin/users'),
  getUser: (id: string) => request<any>(`/admin/users/${id}`),
  updateUser: (id: string, data: Record<string, unknown>) =>
    mutation<{ message: string }>('PATCH', `/admin/users/${id}`, data),
  listTeams: () => request<any[]>('/admin/teams'),
  listAudit: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return request<any[]>(`/admin/audit${qs}`);
  },

  // Ops Dashboard
  opsSummary: () => request<Record<string, number>>('/admin/ops/summary'),
  opsOutbox: () => request<any[]>('/admin/ops/outbox'),
  opsDeadLetters: () => request<any[]>('/admin/ops/dead-letters'),
  opsWorkers: () => request<{ name: string; status: string; last_seen?: string }[]>('/admin/ops/workers'),
  opsWebhookRejections: () => request<any[]>('/admin/ops/webhooks/rejections'),
  opsAgentBlocks: () => request<any[]>('/admin/ops/agent-blocks'),
  deepHealth: () => request<any>('/health/deep'),

  // Integration keys (direct, no team-scoping helper needed)
  listIntegrationKeys: (teamId: string) => request<any[]>(`/teams/${teamId}/integration-keys`),
  createIntegrationKey: (teamId: string, data: { name: string; allowed_sources: string[]; allowed_scopes: string[]; allow_unsigned_dev?: boolean }) =>
    mutation<any>('POST', `/teams/${teamId}/integration-keys`, data),
  revokeIntegrationKey: (teamId: string, keyId: string) =>
    mutation<{ message: string }>('DELETE', `/teams/${teamId}/integration-keys/${keyId}`),
  rotateIntegrationKey: (teamId: string, keyId: string) =>
    mutation<{ id: string; key: string; signing_secret: string; prefix: string; name: string }>('POST', `/teams/${teamId}/integration-keys/${keyId}/rotate`),
};
