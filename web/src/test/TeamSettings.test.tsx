import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import TeamSettings from '../features/team/TeamSettings';

const MEMBERS = [
  { user_id: 'u1', name: 'Owner', email: 'owner@test.dev', role: 'owner', joined_at: '2026-06-01T00:00:00Z' },
  { user_id: 'u2', name: 'Maya', email: 'maya@test.dev', role: 'manager', joined_at: '2026-06-05T00:00:00Z' },
];
const INVITATIONS = [
  { id: 'inv-1', email: 'new@test.dev', role: 'member', invited_by: 'u1', expires_at: '2026-07-01T00:00:00Z', accepted_at: null, created_at: '2026-06-17T00:00:00Z', status: 'pending' },
];

function mockTeam(members = MEMBERS, invitations = INVITATIONS) {
  server.use(
    http.get('*/api/teams/:teamId/members', () => HttpResponse.json(members)),
    http.get('*/api/teams/:teamId/invitations', () => HttpResponse.json(invitations)),
  );
}

describe('TeamSettings', () => {
  it('renders the members table with role badges', async () => {
    setActiveTeam();
    mockTeam();
    renderWithProviders(<TeamSettings />, { route: '/settings/team', auth: true });

    await waitFor(() => expect(screen.getByText('Owner')).toBeInTheDocument());
    expect(screen.getByText('maya@test.dev')).toBeInTheDocument();
    expect(screen.getByText('owner')).toBeInTheDocument();
    expect(screen.getByText('manager')).toBeInTheDocument();
  });

  it('renders pending invitations', async () => {
    setActiveTeam();
    mockTeam();
    renderWithProviders(<TeamSettings />, { route: '/settings/team', auth: true });

    await waitFor(() => expect(screen.getByText('new@test.dev')).toBeInTheDocument());
    expect(screen.getByText(/Expires/)).toBeInTheDocument();
  });

  it('sends an invitation via the mutation when the form is submitted', async () => {
    setActiveTeam();
    // Grant invitations.create BEFORE render.
    server.use(
      http.get('*/api/auth/permissions', () =>
        HttpResponse.json({ role: 'owner', team_id: 'team-1', permissions: ['team.invitations.create'] }),
      ),
    );
    mockTeam();
    let posted: { email: string; role: string } | undefined;
    server.use(
      http.post('*/api/teams/:teamId/invitations', async ({ request }) => {
        posted = await request.json() as { email: string; role: string };
        return HttpResponse.json({ id: 'inv-2' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<TeamSettings />, { route: '/settings/team', auth: true });

    await waitFor(() => expect(screen.getByTestId('invite-email')).toBeInTheDocument());
    await user.type(screen.getByTestId('invite-email'), 'hire@test.dev');
    await user.click(screen.getByTestId('invite-submit'));

    await waitFor(() => expect(posted).toMatchObject({ email: 'hire@test.dev', role: 'member' }));
    // Input cleared after success.
    expect(screen.getByTestId('invite-email')).toHaveValue('');
  });

  it('shows the error state when members fail to load', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/members', () =>
        HttpResponse.json({ detail: 'forbidden' }, { status: 403 }),
      ),
      http.get('*/api/teams/:teamId/invitations', () => HttpResponse.json([])),
    );
    renderWithProviders(<TeamSettings />, { route: '/settings/team', auth: true });

    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
    expect(screen.getByText('Failed to load members')).toBeInTheDocument();
  });
});
