import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { useWebSocketInvalidation } from '../hooks/useWebSocket';

interface RefetchState {
  version: number;
  triggerRefetch: () => void;
  wsConnected: boolean;
}

const Ctx = createContext<RefetchState>({ version: 0, triggerRefetch: () => {}, wsConnected: false });

export function RefetchProvider({ children }: { children: ReactNode }) {
  const [version, setVersion] = useState(0);
  const { connected } = useWebSocketInvalidation(() => {
    // WebSocket event received — bump version to trigger refetches
    setVersion(v => v + 1);
  });

  const triggerRefetch = useCallback(() => setVersion(v => v + 1), []);

  return (
    <Ctx.Provider value={{ version, triggerRefetch, wsConnected: connected }}>
      {children}
    </Ctx.Provider>
  );
}

export function useRefetch() {
  return useContext(Ctx);
}
