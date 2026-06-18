import { vi } from 'vitest';

/**
 * The auth+team surface that the real AuthProvider calls on mount. Legacy
 * knowledge tests mock `vi.mock('../api/client', ...)` with only the specific
 * `api.*` methods they exercise; spread this in so the AuthProvider (mounted
 * via renderWithProviders({ auth: true })) initializes cleanly.
 *
 * Usage in a test:
 *   vi.mock('../api/client', () => ({
 *     api: {
 *       getQualityReport: vi.fn(),
 *       ...legacyApiMethods(),   // <-- the auth surface
 *     },
 *     ...legacyApiExports(),      // <-- ApiError + token/team helpers
 *   }));
 */
export function legacyApiMethods() {
  return {
    me: vi.fn().mockResolvedValue({
      id: 'u1', email: 'owner@test.dev', name: 'Owner', active: true,
      teams: [{ id: 'team-1', name: 'Team', slug: 'team', role: 'owner' }],
    }),
    permissions: vi.fn().mockResolvedValue({ role: 'owner', team_id: 'team-1', permissions: [] }),
  };
}

export function legacyApiExports() {
  return {
    ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
    getStoredTeamId: () => 'team-1',
    setStoredTeamId: () => {},
    setAccessToken: () => {},
    getAccessToken: () => null,
  };
}
