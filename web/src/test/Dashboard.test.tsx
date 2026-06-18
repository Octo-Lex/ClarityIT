import { describe, it, expect } from 'vitest';
import { http, HttpResponse, delay } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import Dashboard from '../features/dashboard/Dashboard';

const WORK_ITEMS = [
  { id: 'wi-1', title: 'Rotate DB creds', status: 'open', priority: 'high', work_item_type: 'task', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
  { id: 'wi-2', title: 'Patch nginx', status: 'open', priority: 'medium', work_item_type: 'bug', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
];
const INCIDENTS = [
  { id: 'inc-1', title: 'DB pool exhausted', status: 'open', severity: 'sev1', impact: 'high', resolved_at: null, created_at: '2026-06-17T10:00:00Z', summary: '', version: 1 },
];

function mockDashboardData(work = WORK_ITEMS, incidents = INCIDENTS) {
  server.use(
    http.get('*/api/teams/:teamId/work-items', ({ request }) => {
      // Dashboard sends status=open; honor it
      const url = new URL(request.url);
      if (url.searchParams.get('status') === 'open') return HttpResponse.json(work);
      return HttpResponse.json([]);
    }),
    http.get('*/api/teams/:teamId/incidents', () => HttpResponse.json(incidents)),
  );
}

describe('Dashboard', () => {
  it('renders the welcome header', () => {
    setActiveTeam();
    mockDashboardData();
    renderWithProviders(<Dashboard />, { route: '/', auth: true });
    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeInTheDocument();
    expect(screen.getByText(/Welcome back/)).toBeInTheDocument();
  });

  it('shows stat card counts once data loads', async () => {
    setActiveTeam();
    mockDashboardData();
    renderWithProviders(<Dashboard />, { route: '/', auth: true });

    await waitFor(() => {
      // 2 open work items
      expect(screen.getByText('Open Work Items')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
    });
    // 1 active incident (open)
    expect(screen.getByText('Active Incidents')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
    // Operational badge always shows
    expect(screen.getByText('Operational')).toBeInTheDocument();
  });

  it('renders the recent work items list', async () => {
    setActiveTeam();
    mockDashboardData();
    renderWithProviders(<Dashboard />, { route: '/', auth: true });

    await waitFor(() => expect(screen.getByText('Rotate DB creds')).toBeInTheDocument());
    expect(screen.getByText('Patch nginx')).toBeInTheDocument();
  });

  it('isolates a work-items failure (no silent swallow) while incidents still load', async () => {
    setActiveTeam();
    // Work items fail; incidents succeed.
    server.use(
      http.get('*/api/teams/:teamId/work-items', () =>
        HttpResponse.json({ detail: 'boom' }, { status: 500 }),
      ),
      http.get('*/api/teams/:teamId/incidents', () => HttpResponse.json(INCIDENTS)),
    );
    renderWithProviders(<Dashboard />, { route: '/', auth: true });

    // Incidents stat still resolves to 1
    await waitFor(() => expect(screen.getByText('1')).toBeInTheDocument());
    // Work items show the failure indicator + "Failed to load" copy
    expect(screen.getByText('Failed to load work items')).toBeInTheDocument();
  });

  it('shows the empty state when there are no open work items', async () => {
    setActiveTeam();
    mockDashboardData([], []);
    renderWithProviders(<Dashboard />, { route: '/', auth: true });

    // Both stat counts render as 0; the recent-items card shows the empty state.
    await waitFor(() => expect(screen.getByText('No open work items')).toBeInTheDocument());
    expect(screen.getAllByText('0').length).toBeGreaterThanOrEqual(2);
  });
});
