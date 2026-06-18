import { ReactElement, ReactNode } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '@/components/theme/ThemeProvider';
import { AuthProvider } from '@/auth/context';

/**
 * Canonical test renderer. Wraps a component in the same provider stack the
 * app uses (Router + QueryClient + Theme), with a per-test QueryClient so
 * cache state never leaks between tests.
 *
 * Usage:
 *   const { getByText } = renderWithProviders(<MyPage />, {
 *     route: '/incidents',
 *   });
 *
 * For components that read useAuth, pass a mock auth context via the
 * `mockAuth` option — this stubs the AuthProvider.
 */

export interface RenderProviderOptions extends Omit<RenderOptions, 'wrapper'> {
  /** Initial router entries (defaults to ['/']). */
  route?: string;
  /** Router props (e.g. initial search params). */
  routerProps?: Partial<MemoryRouterProps>;
  /** Wrap with the real AuthProvider (default false). The MSW default handlers
   *  serve /auth/me and /auth/permissions, so the provider initializes cleanly.
   *  Use this for components that call useAuth(). */
  auth?: boolean;
}

function makeTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // No retries + no gc delay in tests for deterministic, fast runs.
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: { retry: false },
    },
  });
}

export function renderWithProviders(
  ui: ReactElement,
  { route = '/', routerProps, auth = false, ...renderOptions }: RenderProviderOptions = {},
) {
  const queryClient = makeTestQueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    const content = (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[route]} {...routerProps}>
          {children}
        </MemoryRouter>
      </QueryClientProvider>
    );
    return (
      <ThemeProvider>
        {auth ? <AuthProvider>{content}</AuthProvider> : content}
      </ThemeProvider>
    );
  }
  return { ...render(ui, { wrapper: Wrapper, ...renderOptions }), queryClient };
}

/** Re-export for tests that only need the QueryClient wrapper (no router). */
export function renderWithQueryClient(
  ui: ReactElement,
  renderOptions?: RenderOptions,
) {
  const queryClient = makeTestQueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return { ...render(ui, { wrapper: Wrapper, ...renderOptions }), queryClient };
}
