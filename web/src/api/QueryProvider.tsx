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
 * not installed in the Docker build image). The fallback stub accepts any props
 * so callers can pass `initialIsOpen` regardless of which branch resolves.
 */
const ReactQueryDevtools = lazy(async () => {
  if (!import.meta.env.DEV) {
    return { default: (_props: unknown) => null };
  }
  const m = await import('@tanstack/react-query-devtools');
  return { default: m.ReactQueryDevtools };
});

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
