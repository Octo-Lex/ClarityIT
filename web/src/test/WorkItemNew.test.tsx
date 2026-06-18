import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import WorkItemNew from '../features/work-items/WorkItemNew';

describe('WorkItemNew', () => {
  it('blocks submit and shows a validation error when title is empty', async () => {
    setActiveTeam();
    server.use(http.post('*/api/teams/:teamId/work-items', () => HttpResponse.json({ id: 'x' })));
    const user = userEvent.setup();
    renderWithProviders(<WorkItemNew />, { route: '/work-items/new', auth: true });

    await user.click(screen.getByTestId('wi-create'));
    expect(screen.getByText('Title is required')).toBeInTheDocument();
  });

  it('creates a work item and navigates to its detail page on success', async () => {
    setActiveTeam();
    let createdBody: Record<string, unknown> | undefined;
    server.use(
      http.post('*/api/teams/:teamId/work-items', async ({ request }) => {
        createdBody = await request.json();
        return HttpResponse.json({ id: 'wi-new' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<WorkItemNew />, { route: '/work-items/new', auth: true });

    await user.type(screen.getByTestId('wi-title'), 'Patch the build server');
    await user.click(screen.getByTestId('wi-create'));

    await waitFor(() => expect(createdBody).toBeDefined());
    expect(createdBody).toMatchObject({
      title: 'Patch the build server',
      work_item_type: 'task',
      priority: 'none',
      status: 'open',
    });
  });

  it('shows a toast on creation failure', async () => {
    setActiveTeam();
    server.use(
      http.post('*/api/teams/:teamId/work-items', () =>
        HttpResponse.json({ detail: 'Validation failed' }, { status: 422 }),
      ),
    );
    const user = userEvent.setup();
    renderWithProviders(<WorkItemNew />, { route: '/work-items/new', auth: true });

    await user.type(screen.getByTestId('wi-title'), 'A valid title');
    await user.click(screen.getByTestId('wi-create'));

    await waitFor(() => expect(screen.getByText('Create failed')).toBeInTheDocument());
  });
});
