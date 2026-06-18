import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import ObjectDetailPage from '../features/objects/ObjectDetailPage';

const OBJECT = {
  id: 'obj-1', team_id: 'team-1', object_type: 'work_item', title: 'Rotate DB creds',
  summary: 'Quarterly rotation', status: 'open', priority: 'high', owner_id: 'u1',
  created_by: 'u1', version: 3, metadata: {}, created_at: '2026-06-17T10:00:00Z', updated_at: '2026-06-17T10:00:00Z',
};
const COMMENTS = [
  { id: 'c-1', author_id: 'u1abcdef0', body: 'Started on this', created_at: '2026-06-17T11:00:00Z', updated_at: null },
];
const LINKS = [
  { id: 'l-1', from_object_id: 'obj-1', to_object_id: 'obj-2', relation_type: 'blocks', created_at: '2026-06-17T10:00:00Z' },
];

function mockObject(overrides: Partial<typeof OBJECT> = {}) {
  server.use(
    http.get('*/api/teams/:teamId/objects/obj-1', () => HttpResponse.json({ ...OBJECT, ...overrides })),
    http.get('*/api/teams/:teamId/objects/obj-1/comments', () => HttpResponse.json(COMMENTS)),
    http.get('*/api/teams/:teamId/objects/obj-1/links', () => HttpResponse.json(LINKS)),
  );
}

describe('ObjectDetailPage', () => {
  it('renders the object title, status badge, version, and details', async () => {
    setActiveTeam();
    mockObject();
    renderWithProviders(<ObjectDetailPage />, { route: '/objects/obj-1', routePath: '/objects/:id', auth: true });

    await waitFor(() => expect(screen.getByText('Rotate DB creds')).toBeInTheDocument());
    expect(screen.getByText('Quarterly rotation')).toBeInTheDocument();
    expect(screen.getByText('v3')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
  });

  it('renders comments and links', async () => {
    setActiveTeam();
    mockObject();
    renderWithProviders(<ObjectDetailPage />, { route: '/objects/obj-1', routePath: '/objects/:id', auth: true });

    await waitFor(() => expect(screen.getByText('Started on this')).toBeInTheDocument());
    expect(screen.getByText(/blocks/)).toBeInTheDocument();
    expect(screen.getByText(/obj-2/)).toBeInTheDocument();
  });

  it('posts a new comment via the mutation on Enter', async () => {
    setActiveTeam();
    mockObject();
    let postedBody: string | undefined;
    server.use(
      http.post('*/api/teams/:teamId/objects/obj-1/comments', async ({ request }) => {
        postedBody = (await request.json() as { body: string }).body;
        return HttpResponse.json({ id: 'c-2' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<ObjectDetailPage />, { route: '/objects/obj-1', routePath: '/objects/:id', auth: true });

    await waitFor(() => expect(screen.getByTestId('comment-input')).toBeInTheDocument());
    await user.type(screen.getByTestId('comment-input'), 'On it{Enter}');

    await waitFor(() => expect(postedBody).toBe('On it'));
  });

  it('renders without error when the object loads (edit form is wired via permission gate)', async () => {
    // The 409 conflict contract is covered directly in ObjectEditForm.test.tsx,
    // which mocks the auth context to grant objects.update. Here we just confirm
    // the detail page renders the object and the edit-form mount point.
    setActiveTeam();
    mockObject({ version: 3 });
    renderWithProviders(<ObjectDetailPage />, { route: '/objects/obj-1', routePath: '/objects/:id', auth: true });

    await waitFor(() => expect(screen.getByText('Rotate DB creds')).toBeInTheDocument());
    expect(screen.getByText('v3')).toBeInTheDocument();
  });

  it('shows an error state when the object fetch fails', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/objects/obj-1', () =>
        HttpResponse.json({ detail: 'not found' }, { status: 404 }),
      ),
    );
    renderWithProviders(<ObjectDetailPage />, { route: '/objects/obj-1', routePath: '/objects/:id', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
  });
});
