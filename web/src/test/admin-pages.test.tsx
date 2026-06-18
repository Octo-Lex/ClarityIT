import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import AdminUsers from '../features/admin/AdminUsers';
import AdminTeams from '../features/admin/AdminTeams';
import AdminAudit from '../features/admin/AdminAudit';
import AdminSetup from '../features/admin/AdminSetup';
import AdminIntegrations from '../features/admin/AdminIntegrations';

describe('AdminUsers', () => {
  it('renders the user table with role + status badges', async () => {
    server.use(
      http.get('*/api/admin/users', () =>
        HttpResponse.json([
          { id: 'u1', name: 'Alice Owner', email: 'owner@test.dev', active: true, is_platform_owner: true },
          { id: 'u2', name: 'Maya', email: 'maya@test.dev', active: false, is_platform_owner: false },
        ]),
      ),
    );
    renderWithProviders(<AdminUsers />);
    await waitFor(() => expect(screen.getByText('Alice Owner')).toBeInTheDocument());
    expect(screen.getByText('maya@test.dev')).toBeInTheDocument();
    expect(screen.getByText('Inactive')).toBeInTheDocument();
  });

  it('toggles a user active state via the mutation', async () => {
    let patched: Record<string, unknown> | undefined;
    server.use(
      http.get('*/api/admin/users', () =>
        HttpResponse.json([{ id: 'u1', name: 'Owner', email: 'o@t.dev', active: true, is_platform_owner: true }]),
      ),
      http.patch('*/api/admin/users/u1', async ({ request }) => {
        patched = await request.json();
        return HttpResponse.json({ message: 'ok' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<AdminUsers />);
    await waitFor(() => expect(screen.getByTestId('user-toggle-u1')).toBeInTheDocument());
    await user.click(screen.getByTestId('user-toggle-u1'));
    await waitFor(() => expect(patched).toEqual({ is_active: false }));
  });

  it('shows an error state on fetch failure (no silent swallow)', async () => {
    server.use(http.get('*/api/admin/users', () => HttpResponse.json({ detail: 'forbidden' }, { status: 403 })));
    renderWithProviders(<AdminUsers />);
    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
  });
});

describe('AdminTeams', () => {
  it('renders team cards', async () => {
    server.use(
      http.get('*/api/admin/teams', () =>
        HttpResponse.json([
          { id: 't1', name: 'Platform', slug: 'platform', description: 'Core team' },
        ]),
      ),
    );
    renderWithProviders(<AdminTeams />);
    await waitFor(() => expect(screen.getByText('Platform')).toBeInTheDocument());
    expect(screen.getByText('platform')).toBeInTheDocument();
    expect(screen.getByText('Core team')).toBeInTheDocument();
  });

  it('shows the empty state', async () => {
    server.use(http.get('*/api/admin/teams', () => HttpResponse.json([])));
    renderWithProviders(<AdminTeams />);
    await waitFor(() => expect(screen.getByTestId('page-empty')).toBeInTheDocument());
  });
});

describe('AdminAudit', () => {
  it('renders audit events', async () => {
    server.use(
      http.get('*/api/admin/audit', () =>
        HttpResponse.json([
          { id: 'a1', actor_id: 'u1234abcd', action: 'user.login', entity_type: 'user', entity_id: 'u1234abcd', new_value: {}, created_at: '2026-06-17T10:00:00Z' },
        ]),
      ),
    );
    renderWithProviders(<AdminAudit />);
    await waitFor(() => expect(screen.getByText('user.login')).toBeInTheDocument());
    expect(screen.getByText(/user\/u1234abc/)).toBeInTheDocument();
  });

  it('shows the empty state', async () => {
    server.use(http.get('*/api/admin/audit', () => HttpResponse.json([])));
    renderWithProviders(<AdminAudit />);
    await waitFor(() => expect(screen.getByTestId('page-empty')).toBeInTheDocument());
  });
});

describe('AdminSetup', () => {
  it('renders the checklist with done/pending items', async () => {
    server.use(
      http.get('*/api/admin/setup-status', () =>
        HttpResponse.json({
          bootstrap_complete: true,
          first_team_exists: true,
          users_exist: false,
          integration_key_created: false,
          email_configured: false,
          email_mode: 'dev',
          proxmox_mode: 'disabled',
          webhook_signing_enforced: false,
          next_actions: ['Create a user', 'Configure SMTP'],
          agent_profile_required: 'reasoning-worker',
        }),
      ),
    );
    renderWithProviders(<AdminSetup />);
    await waitFor(() => expect(screen.getByText('Platform Bootstrapped')).toBeInTheDocument());
    expect(screen.getAllByText('Complete').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('Pending').length).toBeGreaterThanOrEqual(3);
    expect(screen.getByText(/Create a user/)).toBeInTheDocument();
  });

  it('shows an error state when the fetch fails', async () => {
    server.use(http.get('*/api/admin/setup-status', () => HttpResponse.json({ detail: 'no' }, { status: 500 })));
    renderWithProviders(<AdminSetup />);
    await waitFor(() => expect(screen.getByTestId('page-error')).toBeInTheDocument());
  });
});

describe('AdminIntegrations', () => {
  it('renders the integration keys table', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/integration-keys', () =>
        HttpResponse.json([
          { id: 'k1', name: 'Grafana', prefix: 'clk_', allowed_sources: ['grafana'], allowed_scopes: ['webhooks:ingest'], created_at: '2026-06-17T00:00:00Z', revoked_at: null, rotation_required: false },
        ]),
      ),
    );
    renderWithProviders(<AdminIntegrations />);
    await waitFor(() => expect(screen.getByText('Grafana')).toBeInTheDocument());
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('creates a key and reveals the secret once', async () => {
    setActiveTeam();
    server.use(
      http.get('*/api/teams/:teamId/integration-keys', () => HttpResponse.json([])),
      http.post('*/api/teams/:teamId/integration-keys', async ({ request }) => {
        const body = await request.json() as Record<string, unknown>;
        return HttpResponse.json({ id: 'k-new', key: 'clk_secret123', signing_secret: 'sec_xyz', prefix: 'clk_', name: body.name });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<AdminIntegrations />);
    await waitFor(() => expect(screen.getByRole('button', { name: /Create Key/ })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Create Key/ }));
    await user.type(screen.getByTestId('ik-name'), 'Prometheus');
    await user.type(screen.getByTestId('ik-sources'), 'prometheus');
    await user.type(screen.getByTestId('ik-scopes'), 'alerts:create');
    await user.click(screen.getByTestId('ik-create'));

    // The secret reveal modal appears (shown once).
    await waitFor(() => expect(screen.getByText(/Save These Credentials/i)).toBeInTheDocument());
    expect(screen.getByText('clk_secret123')).toBeInTheDocument();
  });

  it('shows the empty state when there are no keys', async () => {
    setActiveTeam();
    server.use(http.get('*/api/teams/:teamId/integration-keys', () => HttpResponse.json([])));
    renderWithProviders(<AdminIntegrations />);
    await waitFor(() => expect(screen.getByTestId('page-empty')).toBeInTheDocument());
  });
});
