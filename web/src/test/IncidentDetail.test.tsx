import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import IncidentDetail from '../features/incidents/IncidentDetail';

const INCIDENT = {
  id: 'inc-1', title: 'DB pool exhausted', status: 'open', severity: 'sev1', impact: 'high',
  resolved_at: null, created_at: '2026-06-17T10:00:00Z', summary: 'Primary DB refusing connections', version: 2,
};
const COMMENTS = [
  { id: 'c-1', author_id: 'u1', body: 'Investigating', created_at: '2026-06-17T11:00:00Z', updated_at: null },
];

function mockIncident(overrides: Partial<typeof INCIDENT> = {}) {
  server.use(
    http.get('*/api/teams/:teamId/incidents/inc-1', () => HttpResponse.json({ ...INCIDENT, ...overrides })),
    http.get('*/api/teams/:teamId/objects/inc-1/comments', () => HttpResponse.json(COMMENTS)),
  );
}

describe('IncidentDetail', () => {
  it('renders the title, severity, status, version, and details', async () => {
    setActiveTeam();
    mockIncident();
    renderWithProviders(<IncidentDetail />, { route: '/incidents/inc-1', routePath: '/incidents/:id', auth: true });

    await waitFor(() => expect(screen.getByText('DB pool exhausted')).toBeInTheDocument());
    expect(screen.getByText('SEV1')).toBeInTheDocument();
    expect(screen.getByText('open')).toBeInTheDocument();
    expect(screen.getByText('v2')).toBeInTheDocument();
    expect(screen.getByText('Primary DB refusing connections')).toBeInTheDocument();
  });

  it('renders timeline entries (comments)', async () => {
    setActiveTeam();
    mockIncident();
    renderWithProviders(<IncidentDetail />, { route: '/incidents/inc-1', routePath: '/incidents/:id', auth: true });

    await waitFor(() => expect(screen.getByText('Investigating')).toBeInTheDocument());
  });

  it('shows the resolved timestamp when present', async () => {
    setActiveTeam();
    mockIncident({ status: 'resolved', resolved_at: '2026-06-17T12:00:00Z' });
    renderWithProviders(<IncidentDetail />, { route: '/incidents/inc-1', routePath: '/incidents/:id', auth: true });

    await waitFor(() => expect(screen.getByText(/Resolved/)).toBeInTheDocument());
  });

  it('adds a timeline entry via the mutation on Enter', async () => {
    setActiveTeam();
    // Grant incidents.update so the timeline input renders.
    server.use(
      http.get('*/api/auth/permissions', () =>
        HttpResponse.json({ role: 'owner', team_id: 'team-1', permissions: ['incidents.update'] }),
      ),
    );
    mockIncident();
    let postedBody: string | undefined;
    server.use(
      http.post('*/api/teams/:teamId/incidents/inc-1/timeline', async ({ request }) => {
        postedBody = (await request.json() as { body: string }).body;
        return HttpResponse.json({ id: 't-1' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<IncidentDetail />, { route: '/incidents/inc-1', routePath: '/incidents/:id', auth: true });

    await waitFor(() => expect(screen.getByTestId('timeline-input')).toBeInTheDocument());
    await user.type(screen.getByTestId('timeline-input'), 'Rolled back the deploy{Enter}');

    await waitFor(() => expect(postedBody).toBe('Rolled back the deploy'));
  });

  it('resolves the incident with expected_version (optimistic concurrency preserved)', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/auth/permissions', () =>
        HttpResponse.json({ role: 'owner', team_id: 'team-1', permissions: ['incidents.update'] }),
      ),
    );
    mockIncident({ version: 5 });
    let patchBody: Record<string, unknown> | undefined;
    server.use(
      http.patch('*/api/teams/:teamId/incidents/inc-1', async ({ request }) => {
        patchBody = await request.json();
        return HttpResponse.json({ message: 'resolved' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<IncidentDetail />, { route: '/incidents/inc-1', routePath: '/incidents/:id', auth: true });

    await waitFor(() => expect(screen.getByTestId('inc-resolve')).toBeInTheDocument());
    await user.click(screen.getByTestId('inc-resolve'));

    await waitFor(() => expect(patchBody).toBeDefined());
    expect(patchBody).toMatchObject({ status: 'resolved', expected_version: 5 });
  });

  it('shows an error state when the incident fetch fails', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/incidents/inc-1', () =>
        HttpResponse.json({ detail: 'not found' }, { status: 404 }),
      ),
    );
    renderWithProviders(<IncidentDetail />, { route: '/incidents/inc-1', routePath: '/incidents/:id', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
  });
});
