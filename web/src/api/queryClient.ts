import { QueryClient, isServer } from '@tanstack/react-query';
import { ApiError } from './client';

/**
 * React Query client with defaults tuned for this app.
 *
 * - retry: 1 for queries, 0 for mutations. Mutations use Idempotency-Key
 *   headers, so retrying a mutation would either no-op (idempotent replay) or
 *   double-execute if the first succeeded but the response was lost. Safe
 *   default is to never auto-retry mutations; the user re-submits explicitly.
 * - Don't retry on 401/403/404 — these are deterministic auth/not-found errors.
 * - staleTime 30s: balance freshness with avoiding hammering the API on
 *   refocus. WS events invalidate proactively, so stale data is short-lived.
 */
function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30 * 1000,
        gcTime: 5 * 60 * 1000,
        retry: (failureCount, error) => {
          // Never retry deterministic client errors
          if (error instanceof ApiError) {
            const s = error.status;
            if (s === 401 || s === 403 || s === 404 || s === 400) return false;
          }
          return failureCount < 1;
        },
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

/**
 * Singleton per browser session; on the server (SSR — not currently used, but
 * kept correct) each request gets its own client to avoid cross-request leaks.
 */
export function getQueryClient() {
  if (isServer) {
    return makeQueryClient();
  }
  if (!browserQueryClient) {
    browserQueryClient = makeQueryClient();
  }
  return browserQueryClient;
}
