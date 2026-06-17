import { useEffect, useRef, useState, useSyncExternalStore } from 'react';
import { getAccessToken, getStoredTeamId } from '../api/client';

/**
 * Shared WebSocket connection — singleton across the app.
 *
 * The original implementation opened a new WebSocket per useWebSocketInvalidation
 * call site, which would multiply connections (one per consumer). This refactor
 * exposes a single shared connection via an external store: all hooks subscribe
 * to the same connection and its event stream.
 *
 * Events are treated as invalidation signals only — never as source of truth
 * (see useRealtimeInvalidation for the React Query integration). On reconnect,
 * consumers refetch via cache invalidation.
 */

export interface WsEvent {
  team_id: string;
  event_type: string;
  aggregate_type: string;
  aggregate_id: string;
  occurred_at: string;
}

type Listener = (event: WsEvent) => void;

// ─── Singleton connection store ───
let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
let connectedTeam: string | null = null;
let connectedToken: string | null = null;
const listeners = new Set<Listener>();
let connectedState = false;
const connListeners = new Set<() => void>();

function notifyEvent(event: WsEvent) {
  listeners.forEach((l) => {
    try { l(event); } catch { /* listener errors must not break the stream */ }
  });
}

function setConnected(v: boolean) {
  connectedState = v;
  connListeners.forEach((l) => l());
}

function connect() {
  const token = getAccessToken();
  const teamId = getStoredTeamId();
  if (!token || !teamId) return;

  // Already connected to the same team+token — nothing to do.
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) && connectedTeam === teamId && connectedToken === token) {
    return;
  }
  // Tear down any stale connection (token rotated or team switched).
  if (ws) { try { ws.close(); } catch { /* noop */ } ws = null; }

  connectedTeam = teamId;
  connectedToken = token;

  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  const url = `${proto}//${host}/api/ws?token=${token}`;

  ws = new WebSocket(url);

  ws.onopen = () => setConnected(true);

  ws.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data) as WsEvent;
      // Drop cross-team events at the connection level.
      if (data.team_id && data.team_id !== teamId) return;
      notifyEvent(data);
    } catch { /* malformed payload — ignore */ }
  };

  ws.onclose = () => {
    setConnected(false);
    ws = null;
    // Reconnect with backoff (capped) — only if we still have creds.
    if (getAccessToken() && getStoredTeamId()) {
      reconnectTimer = setTimeout(connect, 3000);
    }
  };

  ws.onerror = () => { try { ws?.close(); } catch { /* noop */ } };
}

function disconnect() {
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = undefined; }
  if (ws) { try { ws.close(); } catch { /* noop */ } ws = null; }
  connectedTeam = null;
  connectedToken = null;
  setConnected(false);
}

function subscribeConn(listener: () => void): () => void {
  connListeners.add(listener);
  return () => connListeners.delete(listener);
}

function getConnSnapshot(): boolean {
  return connectedState;
}

// ─── Public hooks ───

/**
 * Subscribe to WS events. The callback fires for each in-team event.
 * Also triggers the singleton connection to open (after auth + team are set).
 * Returns { connected, lastEvent }.
 */
export function useWebSocketInvalidation(onEvent?: (event: WsEvent) => void): {
  connected: boolean;
  lastEvent: WsEvent | null;
} {
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const [lastEvent, setLastEvent] = useState<WsEvent | null>(null);

  // Subscribe to the shared connection state.
  const connected = useSyncExternalStore(subscribeConn, getConnSnapshot, getConnSnapshot);

  // Register this consumer's event listener + ensure connection is open.
  useEffect(() => {
    const handler: Listener = (event) => {
      setLastEvent(event);
      onEventRef.current?.(event);
    };
    listeners.add(handler);
    connect();
    return () => { listeners.delete(handler); };
  }, []);

  return { connected, lastEvent };
}

/**
 * Imperatively force a reconnect — e.g. after login or team switch, when the
 * token/team changed but no event has fired yet to trigger reconnect.
 */
export function reconnectWebSocket() {
  // Force teardown so connect() reopens with fresh creds.
  disconnect();
  connect();
}

/**
 * Subscribe to just the connected boolean (no event handling). Cheaper for
 * components that only show the Live/Offline indicator.
 */
export function useWebSocketConnected(): boolean {
  return useSyncExternalStore(subscribeConn, getConnSnapshot, getConnSnapshot);
}
