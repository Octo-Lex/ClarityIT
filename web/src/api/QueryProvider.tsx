import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { ReactNode } from 'react';
import { getQueryClient } from './queryClient';

/**
 * Wraps the app with the singleton QueryClient + devtools (dev only).
 *
 * Realtime WS→invalidation is wired at the AppLayout level (inside RequireAuth,
 * where the active team is known), not here, because the bridge needs the teamId.
 */
export function QueryProvider({ children }: { children: ReactNode }) {
  const queryClient = getQueryClient();
  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  );
}
