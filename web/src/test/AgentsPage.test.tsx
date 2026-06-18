import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';
import { AgentsPage } from '../features/agents/AgentsPage';

const AGENTS = [
  { id: 'agent-1', name: 'remediation-executor', description: 'Executes approved steps', status: 'active', max_autonomy: 'A4', created_at: '2026-06-17T10:00:00Z' },
  { id: 'agent-2', name: 'observer', description: 'Read-only monitor', status: 'disabled', max_autonomy: 'A1', created_at: '2026-06-17T10:00:00Z' },
];
const RUNS = [
  { id: 'run-1', agent_id: 'agent-1', status: 'completed', triggered_by: 'u1', created_at: '2026-06-17T11:00:00Z' },
];
const GRANTS = [
  { id: 'g-1', tool_name: 'proxmox.start', max_autonomy_level: 'A4', requires_approval: true, expires_at: null, created_at: '2026-06-17T10:00:00Z', revoked_at: null },
];
const INTENTIONS = [
  { id: 'in-1', intention_type: 'action', requested_tool: 'proxmox.start', confidence: 0.9, risk_level: 'medium', autonomy_level: 'A4', status: 'executed', blocked_reason: null, created_at: '2026-06-17T11:00:00Z' },
];

function withReadPerm(extra: string[] = []) {
  server.use(
    http.get('*/api/auth/permissions', () =>
      HttpResponse.json({ role: 'owner', team_id: 'team-1', permissions: ['agents.read', ...extra] }),
    ),
  );
}

function mockAgentsData() {
  server.use(
    // List endpoint has a trailing slash; match it precisely so it doesn't
    // swallow the grants sub-path.
    http.get('*/api/teams/:teamId/agents/', () => HttpResponse.json(AGENTS)),
    http.get('*/api/teams/:teamId/agent-runs', () => HttpResponse.json(RUNS)),
    http.get('*/api/teams/:teamId/agents/:agentId/grants', () => HttpResponse.json(GRANTS)),
    http.get('*/api/teams/:teamId/agent-runs/:runId/intentions', () => HttpResponse.json(INTENTIONS)),
  );
}

describe('AgentsPage', () => {
  it('renders the agent list with status + autonomy badges', async () => {
    setActiveTeam();
    withReadPerm();
    mockAgentsData();
    renderWithProviders(<AgentsPage />, { route: '/agents', auth: true });

    await waitFor(() => expect(screen.getByText('remediation-executor')).toBeInTheDocument());
    expect(screen.getByText('observer')).toBeInTheDocument();
    expect(screen.getByText('A4')).toBeInTheDocument();
    expect(screen.getByText('A1')).toBeInTheDocument();
  });

  it('shows the access-denied state when the user lacks agents.read', async () => {
    setActiveTeam();
    // Default permissions handler returns [] → no agents.read.
    server.use(
      http.get('*/api/auth/permissions', () =>
        HttpResponse.json({ role: 'member', team_id: 'team-1', permissions: [] }),
      ),
    );
    renderWithProviders(<AgentsPage />, { route: '/agents', auth: true });

    await waitFor(() => expect(screen.getByText('Access denied')).toBeInTheDocument());
  });

  it('selects an agent and loads its grants', async () => {
    setActiveTeam();
    withReadPerm();
    mockAgentsData();
    const user = userEvent.setup();
    renderWithProviders(<AgentsPage />, { route: '/agents', auth: true });

    await waitFor(() => expect(screen.getByTestId('agent-row-agent-1')).toBeInTheDocument());
    await user.click(screen.getByTestId('agent-row-agent-1'));

    // The grants query is enabled once an agent is selected; wait for the grant row.
    await waitFor(() => expect(screen.getByText('proxmox.start')).toBeInTheDocument());
    expect(screen.getByText('Yes')).toBeInTheDocument(); // requires_approval
  });

  it('creates an agent via the mutation (with A5 disabled in the selector)', async () => {
    setActiveTeam();
    withReadPerm(['agents.create']);
    let created: Record<string, unknown> | undefined;
    server.use(
      http.get('*/api/teams/:teamId/agents/', () => HttpResponse.json(AGENTS)),
      http.post('*/api/teams/:teamId/agents/*', async ({ request }) => {
        created = await request.json();
        return HttpResponse.json({ id: 'agent-new' });
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<AgentsPage />, { route: '/agents', auth: true });

    await waitFor(() => expect(screen.getByRole('button', { name: /Create Agent/ })).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /Create Agent/ }));
    await user.type(screen.getByTestId('agent-name'), 'new-bot');
    await user.click(screen.getByTestId('agent-create'));

    await waitFor(() => expect(created).toMatchObject({ name: 'new-bot', max_autonomy: 'A3' }));
  });

  it('views a run and loads its intentions', async () => {
    setActiveTeam();
    withReadPerm();
    mockAgentsData();
    const user = userEvent.setup();
    renderWithProviders(<AgentsPage />, { route: '/agents', auth: true });

    await waitFor(() => expect(screen.getByText('Agent Runs')).toBeInTheDocument());
    // Click "View" on the run row.
    const viewBtn = screen.getAllByRole('button', { name: 'View' })[0];
    await user.click(viewBtn);

    await waitFor(() => expect(screen.getByText(/Intentions — run/)).toBeInTheDocument());
    expect(screen.getByText('proxmox.start')).toBeInTheDocument();
    expect(screen.getByText('executed')).toBeInTheDocument();
  });

  it('shows the empty state when there are no agents', async () => {
    setActiveTeam();
    withReadPerm();
    server.use(
      http.get('*/api/teams/:teamId/agents/', () => HttpResponse.json([])),
      http.get('*/api/teams/:teamId/agent-runs', () => HttpResponse.json([])),
    );
    renderWithProviders(<AgentsPage />, { route: '/agents', auth: true });

    await waitFor(() => expect(screen.getByText('No agents')).toBeInTheDocument());
  });
});
