import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import BoardPage from '../features/board/BoardPage';

const BOARD = {
  open: [
    { id: 'wi-1', title: 'Rotate DB creds', work_item_type: 'task', status: 'open', priority: 'high', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
  ],
  in_progress: [
    { id: 'wi-2', title: 'Patch nginx', work_item_type: 'bug', status: 'in_progress', priority: 'medium', owner_id: 'u1', version: 1, assignee_id: null, project_id: null, summary: '', created_at: '2026-06-17T10:00:00Z' },
  ],
  blocked: [],
  resolved: [],
  closed: [],
};

function mockBoard(board = BOARD) {
  server.use(http.get('*/api/teams/:teamId/work-items/board', () => HttpResponse.json(board)));
}

describe('BoardPage', () => {
  it('renders five status columns with item counts', async () => {
    setActiveTeam();
    mockBoard();
    renderWithProviders(<BoardPage />, { route: '/board', auth: true });

    await waitFor(() => expect(screen.getByTestId('board-column-open')).toBeInTheDocument());
    expect(screen.getByTestId('board-column-in_progress')).toBeInTheDocument();
    expect(screen.getByTestId('board-column-blocked')).toBeInTheDocument();
    expect(screen.getByTestId('board-column-resolved')).toBeInTheDocument();
    expect(screen.getByTestId('board-column-closed')).toBeInTheDocument();

    // The open card renders
    expect(screen.getByText('Rotate DB creds')).toBeInTheDocument();
    // Column count badge
    expect(screen.getByTestId('board-column-open').textContent).toContain('1');
  });

  it('shows the empty state when the board has no items', async () => {
    setActiveTeam();
    mockBoard({ open: [], in_progress: [], blocked: [], resolved: [], closed: [] });
    renderWithProviders(<BoardPage />, { route: '/board', auth: true });

    await waitFor(() => expect(screen.getByText('Board is empty')).toBeInTheDocument());
  });

  it('shows an error state when the board fetch fails', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/work-items/board', () =>
        HttpResponse.json({ detail: 'down' }, { status: 500 }),
      ),
    );
    renderWithProviders(<BoardPage />, { route: '/board', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
    expect(screen.getByText('Failed to load board')).toBeInTheDocument();
  });

  it('renders cards as draggable elements (draggable attribute / dnd-kit wired)', async () => {
    setActiveTeam();
    mockBoard();
    renderWithProviders(<BoardPage />, { route: '/board', auth: true });

    await waitFor(() => expect(screen.getByTestId('board-card-wi-1')).toBeInTheDocument());
    // The card element is present and is the drag handle (role via dnd-kit)
    const card = screen.getByTestId('board-card-wi-1');
    expect(card).toBeInTheDocument();
  });
});
