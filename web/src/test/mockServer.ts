import { setupServer } from 'msw/node';
import { http, HttpResponse, HttpHandler } from 'msw';

/**
 * Shared MSW server. Mounted once in setup.ts (listen/reset/close lifecycle).
 *
 * Default handlers cover the unauthenticated endpoints + the auth flow that
 * AuthProvider hits on mount. Per-test, override specific endpoints with
 * server.use(...) — see renderWithProviders helpers for the common shapes.
 *
 * IMPORTANT: `onUnhandledRequest: 'error'` means every fetch the test code
 * triggers MUST be handled. Either add a default handler here or a per-test
 * override. This catches forgotten mocks (a common source of false greens).
 */

const DEFAULT_TEAM = {
  id: 'team-1',
  name: 'Test Team',
  slug: 'test-team',
  icon: '',
  role: 'owner',
};

/** Default handlers — the auth/session surface AuthProvider depends on. */
export const defaultHandlers: HttpHandler[] = [
  // Bootstrap status — not bootstrapped by default
  http.get('*/api/bootstrap/status', () =>
    HttpResponse.json({ bootstrapped: false }),
  ),

  // Auth: me + permissions (AuthProvider calls these on mount)
  http.get('*/api/auth/me', () =>
    HttpResponse.json({
      id: 'user-1',
      email: 'owner@test.dev',
      name: 'Owner',
      active: true,
      teams: [DEFAULT_TEAM],
    }),
  ),
  http.get('*/api/auth/permissions', () =>
    HttpResponse.json({
      role: 'owner',
      team_id: DEFAULT_TEAM.id,
      permissions: [],
    }),
  ),

  // Empty-list defaults for the common list endpoints, so pages that fetch on
  // mount don't blow up with unhandled requests. Tests override these as needed.
  http.get('*/api/teams/:teamId/work-items', () => HttpResponse.json([])),
  http.get('*/api/teams/:teamId/incidents', () => HttpResponse.json([])),
  http.get('*/api/teams/:teamId/agents/*', () => HttpResponse.json([])),
];

export const server = setupServer(...defaultHandlers);

/** Helper to set the active team in localStorage (most tests need this). */
export function setActiveTeam(teamId = DEFAULT_TEAM.id) {
  localStorage.setItem('clarityit_team', teamId);
}

/** Helper to register a one-off JSON response for a method+path matcher. */
export function jsonHandler(
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE',
  path: string,
  body: unknown,
  status = 200,
): HttpHandler {
  return http[method.toLowerCase() as 'get'](`*${path}`, () =>
    HttpResponse.json(body, { status }),
  );
}
