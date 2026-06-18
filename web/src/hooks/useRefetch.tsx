/**
 * BACKWARD-COMPAT SHIM — do not use in new code.
 *
 * The original useRefetch was a global version-counter that bumped on every WS
 * event, with consumers putting `version` in useEffect deps to refetch. That
 * role is now filled by React Query + useRealtimeInvalidation (precise cache
 * invalidation keyed off the WS aggregate_type).
 *
 * This shim keeps the old `useRefetch()` call sites (Dashboard, AdminApprovals)
 * working until their tracks migrate them to useQuery. It returns:
 *  - version: a monotonically-increasing counter that still bumps on WS events
 *    (so legacy `useEffect([…, version])` deps keep refetching).
 *  - bump / triggerRefetch: no-ops (invalidation now happens at the RQ layer).
 *  - wsConnected: derived from the realtime hook.
 *
 * No provider is required — it self-subscribes to the WS hook directly.
 */
import { useState, useCallback } from 'react';
import type { ReactNode } from 'react';
import { useWebSocketInvalidation } from '../hooks/useWebSocket';

interface RefetchState {
  version: number;
  triggerRefetch: () => void;
  wsConnected: boolean;
  /** Legacy alias for triggerRefetch; kept for AdminApprovals. */
  bump: () => void;
}

export function useRefetch(): RefetchState {
  const [version, setVersion] = useState(0);
  const { connected } = useWebSocketInvalidation(() => {
    // Legacy behavior: bump version on any WS event for this team.
    setVersion((v) => v + 1);
  });

  const triggerRefetch = useCallback(() => setVersion((v) => v + 1), []);
  const bump = triggerRefetch;

  return { version, triggerRefetch, wsConnected: connected, bump };
}

/** Kept for import-compat; no longer needed (no provider required). */
export function RefetchProvider({ children }: { children: ReactNode }) {
  return children;
}

// Backward-compat default export shape (unused but avoids dead-code errors).
export default useRefetch;
