import { QueryClientProvider } from '@tanstack/react-query';
import { lazy, Suspense, ReactNode } from 'react';
import { getQueryClient } from './queryClient';

/**
 * Wraps the app with the singleton QueryClient + devtools (dev only).
 *
 * Realtime WS→invalidation is wired at the AppLayout level (inside RequireAuth,
 * where the active team is known), not here, because the bridge needs the teamId.
 *
 * The devtools are loaded via a guarded dynamic import so the production bundle
 * never resolves @tanstack/react-query-devtools (which is a devDependency and
 * not installed in the Docker build image).
 */
const ReactQueryDevtools = lazy(() =>
  import.meta.env.DEV
    ? import('@tanstack/react-query-devtools').then(m => ({ default: m.ReactQueryDevtools }))
    : Promise.resolve({ default: () => null }),
);

export function QueryProvider({ children }: { children: ReactNode }) {
  const queryClient = getQueryClient();
  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <Suspense fallback={null}>
        {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
      </Suspense>
    </QueryClientProvider>
  );
}
