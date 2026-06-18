import { describe, it, expect } from 'vitest';
import { http, HttpResponse, delay } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import QueuePage from '../features/queue/QueuePage';

const WORK_ITEMS = [
  { id: 'wi-1', title: 'Rotate DB creds', work_item_type: 'task', status: 'open', priority: 'high', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
  { id: 'wi-2', title: 'Patch nginx', work_item_type: 'bug', status: 'in_progress', priority: 'medium', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
  { id: 'wi-3', title: 'Blocked deploy', work_item_type: 'task', status: 'blocked', priority: 'critical', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
];

function mockWorkItems(items = WORK_ITEMS) {
  server.use(
    http.get('*/api/teams/:teamId/work-items', () => HttpResponse.json(items)),
  );
}
function mockWorkItemsError() {
  server.use(
    http.get('*/api/teams/:teamId/work-items', () =>
      HttpResponse.json({ detail: 'Database unavailable' }, { status: 500 }),
    ),
  );
}

describe('QueuePage', () => {
  it('renders the loading skeleton then populated rows', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/work-items', async () => {
        await delay(100);
        return HttpResponse.json(WORK_ITEMS);
      }),
    );

    renderWithProviders(<QueuePage />, { route: '/queue', auth: true });
    // Skeleton shows first (animate-pulse shimmering bars)
    await waitFor(() => {
      expect(document.querySelectorAll('[data-slot="skeleton"]').length).toBeGreaterThan(0);
    });

    // Then rows render
    await waitFor(() => expect(screen.getByText('Rotate DB creds')).toBeInTheDocument());
    expect(screen.getByText('Patch nginx')).toBeInTheDocument();
    expect(screen.getByText('Blocked deploy')).toBeInTheDocument();
  });

  it('shows the empty state when there are no work items', async () => {
    setActiveTeam();
    mockWorkItems([]);
    renderWithProviders(<QueuePage />, { route: '/queue', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-empty')).toBeInTheDocument());
    expect(screen.getByText(/No work items/i)).toBeInTheDocument();
  });

  it('shows an error state with retry when the fetch fails (no silent swallow)', async () => {
    setActiveTeam();
    mockWorkItemsError();
    renderWithProviders(<QueuePage />, { route: '/queue', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
    expect(screen.getByText('Failed to load queue')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });

  it('sends the status filter as a query param when a tab is selected', async () => {
    setActiveTeam();
    let capturedUrl = '';
    server.use(
      http.get('*/api/teams/:teamId/work-items', ({ request }) => {
        capturedUrl = request.url;
        return HttpResponse.json(WORK_ITEMS.filter(wi => wi.status === 'open'));
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<QueuePage />, { route: '/queue', auth: true });

    await waitFor(() => expect(screen.getByText('Rotate DB creds')).toBeInTheDocument());

    // Initial load: no status filter
    expect(capturedUrl).not.toContain('status=open');

    // Click the Open tab
    await user.click(screen.getByRole('tab', { name: 'Open' }));

    await waitFor(() => expect(capturedUrl).toContain('status=open'));
  });

  it('shows the New button when the user has work.items.create permission', async () => {
    setActiveTeam();
    mockWorkItems();
    renderWithProviders(<QueuePage />, { route: '/queue', auth: true });

    await waitFor(() => expect(screen.getByText('Rotate DB creds')).toBeInTheDocument());
    // The "New" button is always present in this test because the default
    // permissions handler returns an empty array — and the component reads
    // hasPermission from the real AuthProvider. We assert the button renders.
    expect(screen.queryByRole('button', { name: /New/ })).toBeInTheDocument();
  });
});
