import { useQueryClient } from '@tanstack/react-query';
import { useWebSocketInvalidation, useWebSocketConnected } from './useWebSocket';

/**
 * WS → React Query invalidation bridge.
 *
 * Replaces the legacy useRefetch version-counter. The architecture's stated
 * philosophy (useWebSocket.ts:9-13: "events are treated as invalidation signals
 * only — never as source of truth") maps directly onto React Query's
 * invalidateQueries. WS events now invalidate the precise cache slice for the
 * affected domain instead of bumping a global counter.
 *
 * The subject pattern is `clarity.v1.<domain>.<entity>.<action>`, so we map
 * the event's aggregate_type to a query-key prefix.
 */
const DOMAIN_KEY_MAP: Record<string, readonly string[]> = {
  work_item: ['work-items'],
  workitem: ['work-items'],
  incident: ['incidents'],
  project: ['projects'],
  object: ['objects'],
  agent: ['agents'],
  agent_run: ['agent-runs'],
  agentrun: ['agent-runs'],
  intention: ['agent-runs'],
  approval: ['approvals'],
  remediation: ['remediations'],
  asset_action: ['asset-actions'],
  assetaction: ['asset-actions'],
  asset: ['assets'],
  artifact: ['artifacts'],
  document: ['documents'],
  meeting_summary: ['artifacts', 'meeting-summaries'],
  status_report: ['artifacts'],
  knowledge_item: ['knowledge'],
  knowledge: ['knowledge'],
  collection: ['knowledge', 'collections'],
  saved_answer: ['knowledge', 'saved-answers'],
  integration_key: ['integration-keys'],
  member: ['members'],
  invitation: ['invitations'],
  access_grant: ['access-grants'],
  settings: ['settings'],
};

export function useRealtimeInvalidation(teamId: string | null) {
  const queryClient = useQueryClient();

  useWebSocketInvalidation((event) => {
    if (!teamId) return;
    // Ignore cross-team events (also guarded in the hook, but defense-in-depth).
    if (event.team_id && event.team_id !== teamId) return;

    const domain = DOMAIN_KEY_MAP[event.aggregate_type];
    if (domain) {
      // Invalidate just the affected domain(s) under this team.
      void queryClient.invalidateQueries({
        queryKey: ['teams', teamId, ...domain],
        refetchType: 'active',
      });
    } else {
      // Unknown domain — conservatively invalidate the whole team scope.
      void queryClient.invalidateQueries({
        queryKey: ['teams', teamId],
        refetchType: 'active',
      });
    }
  });

  // Expose connected state for the layout indicator.
  // We use a re-subscription via the same hook's returned state through a
  // tiny wrapper component pattern; here we just ensure the invalidation
  // subscription is active. The connected indicator is provided separately.
}

/**
 * Subscribe to WS connectivity for the layout "Live/Offline" indicator without
 * triggering invalidation. Returns the connected boolean.
 */
export function useRealtimeConnected() {
  return useWebSocketConnected();
}
