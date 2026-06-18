/**
 * Backend permission strings — the single source of truth for the client.
 *
 * Extracted from every RequirePermission() call in services/api/cmd/api/main.go.
 * When the backend adds/changes a permission, update this module; the
 * accompanying test (permissions.test.ts) fails if a literal used anywhere in
 * the UI is not in this set.
 *
 * Usage:
 *   import { Perm } from '@/auth/permissions';
 *   hasPermission(Perm.WorkItems.View)   // not hasPermission('work.items.view')
 *
 * Never inline a raw string. Always reference Perm.*.
 */

export const Perm = {
  // Team
  TeamSettingsRead: 'team.settings.read',
  TeamSettingsUpdate: 'team.settings.update',
  TeamMembersRead: 'team.members.read',
  TeamMembersUpdate: 'team.members.update',
  TeamMembersRemove: 'team.members.remove',
  TeamInvitationsCreate: 'team.invitations.create',
  TeamInvitationsRead: 'team.invitations.read',
  TeamInvitationsRevoke: 'team.invitations.revoke',
  TeamAccessGrantsRead: 'team.access_grants.read',
  TeamAccessGrantsCreate: 'team.access_grants.create',
  TeamAccessGrantsRevoke: 'team.access_grants.revoke',

  // Objects (universal spine)
  ObjectsCreate: 'objects.create',
  ObjectsRead: 'objects.read',
  ObjectsUpdate: 'objects.update',
  ObjectsDelete: 'objects.delete',
  ObjectsLinksCreate: 'objects.links.create',
  ObjectsLinksRead: 'objects.links.read',
  ObjectsLinksDelete: 'objects.links.delete',
  ObjectsCommentsCreate: 'objects.comments.create',
  ObjectsCommentsRead: 'objects.comments.read',
  ObjectsCommentsUpdateOwn: 'objects.comments.update.own',
  ObjectsCommentsDeleteOwn: 'objects.comments.delete.own',
  ObjectsAttachmentsCreate: 'objects.attachments.create',
  ObjectsAttachmentsRead: 'objects.attachments.read',

  // Work items
  WorkItemsCreate: 'work.items.create',
  WorkItemsView: 'work.items.view',
  WorkItemsUpdateOwn: 'work.items.update.own',
  WorkItemsDeleteOwn: 'work.items.delete.own',

  // Incidents
  IncidentsCreate: 'incidents.create',
  IncidentsRead: 'incidents.read',
  IncidentsUpdate: 'incidents.update',
  IncidentsTimelineCreate: 'incidents.timeline.create',

  // Projects
  ProjectsCreate: 'projects.create',
  ProjectsView: 'projects.view',
  ProjectsUpdate: 'projects.update',
  ProjectsDelete: 'projects.delete',

  // Agents
  AgentsCreate: 'agents.create',
  AgentsRead: 'agents.read',
  AgentsUpdate: 'agents.update',
  AgentsDisable: 'agents.disable',
  AgentsGrantsCreate: 'agents.grants.create',
  AgentsGrantsRead: 'agents.grants.read',
  AgentsGrantsRevoke: 'agents.grants.revoke',
  AgentsRunsCreate: 'agents.runs.create',
  AgentsRunsRead: 'agents.runs.read',
  AgentsIntentionsCreate: 'agents.intentions.create',
  AgentsIntentionsRead: 'agents.intentions.read',
  AgentsToolsExecute: 'agents.tools.execute',

  // Approvals
  ApprovalsCreate: 'approvals.create',
  ApprovalsRead: 'approvals.read',
  ApprovalsApprove: 'approvals.approve',

  // Integrations
  IntegrationsKeysCreate: 'integrations.keys.create',
  IntegrationsKeysRead: 'integrations.keys.read',
  IntegrationsKeysRevoke: 'integrations.keys.revoke',
  IntegrationsProxmoxRead: 'integrations.proxmox.read',
  IntegrationsProxmoxSync: 'integrations.proxmox.sync',

  // Assets
  AssetsRead: 'assets.read',
  AssetsActionsCreate: 'assets.actions.create',
  AssetsActionsRead: 'assets.actions.read',
  AssetsActionsExecute: 'assets.actions.execute',

  // Remediations
  RemediationsCreate: 'remediations.create',
  RemediationsRead: 'remediations.read',
  RemediationsApprove: 'remediations.approve',
  RemediationsExecute: 'remediations.execute',
  RemediationsCancel: 'remediations.cancel',

  // Artifacts
  ArtifactsCreate: 'artifacts.create',
  ArtifactsRead: 'artifacts.read',
  ArtifactsUpdate: 'artifacts.update',
  ArtifactsDelete: 'artifacts.delete',

  // Knowledge
  KnowledgeSearch: 'knowledge.search',
  KnowledgeRead: 'knowledge.read',
  KnowledgeAsk: 'knowledge.ask',
  KnowledgeCollectionsRead: 'knowledge.collections.read',
  KnowledgeCollectionsCreate: 'knowledge.collections.create',
  KnowledgeCollectionsUpdate: 'knowledge.collections.update',
  KnowledgeCollectionsDelete: 'knowledge.collections.delete',
} as const;

export type Permission = (typeof Perm)[keyof typeof Perm];

/** The complete authoritative set — used by tests to validate UI literals. */
export const ALL_PERMISSIONS: readonly Permission[] = Object.values(Perm);

/** Type guard: is this string a real backend permission? */
export function isPermission(value: string): value is Permission {
  return (ALL_PERMISSIONS as readonly string[]).includes(value);
}
