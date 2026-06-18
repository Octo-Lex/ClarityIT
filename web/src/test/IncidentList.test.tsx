import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import IncidentList from '../features/incidents/IncidentList';

const INCIDENTS = [
  { id: 'inc-1', title: 'DB pool exhausted', status: 'open', severity: 'sev1', impact: 'high', resolved_at: null, created_at: '2026-06-17T10:00:00Z', summary: '', version: 1 },
  { id: 'inc-2', title: 'Auth latency', status: 'resolved', severity: 'sev2', impact: 'medium', resolved_at: '2026-06-17T12:00:00Z', created_at: '2026-06-17T09:00:00Z', summary: '', version: 2 },
];

function mockIncidents(list = INCIDENTS) {
  server.use(http.get('*/api/teams/:teamId/incidents', () => HttpResponse.json(list)));
}

describe('IncidentList', () => {
  it('renders the list with severity + status badges', async () => {
    setActiveTeam();
    mockIncidents();
    renderWithProviders(<IncidentList />, { route: '/incidents', auth: true });

    await waitFor(() => expect(screen.getByText('DB pool exhausted')).toBeInTheDocument());
    expect(screen.getByText('Auth latency')).toBeInTheDocument();
    expect(screen.getByText('SEV1')).toBeInTheDocument();
    expect(screen.getByText('resolved')).toBeInTheDocument();
  });

  it('shows the empty state when there are no incidents', async () => {
    setActiveTeam();
    mockIncidents([]);
    renderWithProviders(<IncidentList />, { route: '/incidents', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-empty')).toBeInTheDocument());
  });

  it('shows an error state with retry when the fetch fails (no silent swallow)', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/incidents', () =>
        HttpResponse.json({ detail: 'db down' }, { status: 500 }),
      ),
    );
    renderWithProviders(<IncidentList />, { route: '/incidents', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
    expect(screen.getByText('Failed to load incidents')).toBeInTheDocument();
  });

  it('creates an incident via the mutation + invalidates the list', async () => {
    setActiveTeam();
    // Grant incidents.create BEFORE render so the AuthProvider picks it up.
    server.use(
      http.get('*/api/auth/permissions', () =>
        HttpResponse.json({ role: 'owner', team_id: 'team-1', permissions: ['incidents.create'] }),
      ),
    );
    let created: Record<string, unknown> | undefined;
    let listCalls = 0;
    server.use(
      http.get('*/api/teams/:teamId/incidents', () => {
        listCalls++;
        return HttpResponse.json(listCalls === 1 ? [] : [{ id: 'inc-new', title: 'New outage', status: 'open', severity: 'sev1', impact: '', resolved_at: null, created_at: '2026-06-17T10:00:00Z', summary: '', version: 1 }]);
      }),
      http.post('*/api/teams/:teamId/incidents', async ({ request }) => {
        created = await request.json();
        return HttpResponse.json({ id: 'inc-new' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<IncidentList />, { route: '/incidents', auth: true });

    await waitFor(() => expect(screen.getByRole('button', { name: /New Incident/ })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /New Incident/ }));
    await user.type(screen.getByTestId('inc-title'), 'New outage');
    await user.click(screen.getByTestId('inc-create'));

    await waitFor(() => expect(created).toBeDefined());
    expect(created).toMatchObject({ title: 'New outage', severity: 'sev3' });
    // The list re-fetched after invalidation (listCalls > 1).
    await waitFor(() => expect(listCalls).toBeGreaterThan(1));
  });
});
