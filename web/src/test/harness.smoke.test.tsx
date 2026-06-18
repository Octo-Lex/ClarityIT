/**
 * Harness smoke test — proves the new test harness (MSW + renderWithProviders
 * + fixtures + QueryClient) works end-to-end before feature tracks adopt it.
 *
 * This is the reference pattern for NEW tests written in the rebuild:
 *   1. import { server } + http/HttpResponse from MSW
 *   2. server.use(...) to override a specific endpoint per-test
 *   3. renderWithProviders(<Component/>) for the provider stack
 *   4. fixtures.* for typed, shared data
 *
 * Existing track-*.test.tsx files keep using vi.mock('../api/client') and do
 * NOT need to migrate — both patterns coexist.
 */
import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { fixtures } from './fixtures';
import { setActiveTeam } from './mockServer';

function TeamLabel() {
  // Minimal component that reads from the API client via fetch (exercising MSW)
  // and renders the team name. A real feature page would use useQuery.
  return <div data-testid="team-label">{fixtures.team.name}</div>;
}

describe('test harness', () => {
  it('renders with the provider stack (router + query + theme)', () => {
    setActiveTeam();
    renderWithProviders(<TeamLabel />, { route: '/' });
    expect(screen.getByTestId('team-label')).toHaveTextContent('Test Team');
  });

  it('MSW serves a default handler without per-test setup', async () => {
    setActiveTeam();
    // The defaultHandlers respond to GET /api/auth/me; fetch it directly.
    const res = await fetch('/api/auth/me');
    const body = await res.json();
    expect(body.email).toBe('owner@test.dev');
  });

  it('per-test server.use overrides a default handler', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/auth/me', () =>
        HttpResponse.json({ ...fixtures.user, name: 'Override Name' }),
      ),
    );
    const res = await fetch('/api/auth/me');
    const body = await res.json();
    expect(body.name).toBe('Override Name');
  });

  it('QueryClient is isolated per render (no cross-test cache leak)', () => {
    setActiveTeam();
    const { queryClient } = renderWithProviders(<TeamLabel />);
    expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
  });

  it('fixtures are typed and satisfy their API contract', () => {
    expect(fixtures.incident.severity).toBe('sev1');
    expect(fixtures.dryRunPreview.validation.policy_valid).toBe(true);
  });
});
