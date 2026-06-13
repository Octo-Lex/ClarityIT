import { useCallback, useEffect, useRef, useState } from 'react';
import { getAccessToken, getStoredTeamId } from '../api/client';

interface UseWebSocketResult {
  connected: boolean;
  lastEvent: { team_id: string; event_type: string; aggregate_type: string; aggregate_id: string; occurred_at: string } | null;
}

/**
 * WebSocket hook that connects after auth + team selection.
 * Events are treated as invalidation signals only — never as source of truth.
 * On reconnect, consumers should refetch current page data.
 */
export function useWebSocketInvalidation(
  onEvent?: (event: { team_id: string; event_type: string; aggregate_type: string; aggregate_id: string; occurred_at: string }) => void,
): UseWebSocketResult {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<UseWebSocketResult['lastEvent']>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    const token = getAccessToken();
    const teamId = getStoredTeamId();
    if (!token || !teamId) return;

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${proto}//${host}/api/ws?token=${token}`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        // Ignore cross-team events
        if (data.team_id && data.team_id !== teamId) return;
        setLastEvent(data);
        onEventRef.current?.(data);
      } catch {}
    };

    ws.onclose = () => {
      setConnected(false);
      reconnectTimer.current = setTimeout(connect, 3000);
    };

    ws.onerror = () => ws.close();
  }, []);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
    };
  }, [connect]);

  return { connected, lastEvent };
}
