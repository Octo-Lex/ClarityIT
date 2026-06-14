import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../auth/context';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    me: vi.fn().mockResolvedValue({ id: 'u1', email: 'owner@test.dev', name: 'Owner', teams: [{ id: 't1', name: 'Team', slug: 'team', role: 'owner' }] }),
    permissions: vi.fn().mockResolvedValue({ role: 'owner', team_id: 't1', permissions: ['approvals.read', 'approvals.approve', 'approvals.cancel'] }),
    mfaStatus: vi.fn().mockResolvedValue({ enabled: false }),
    mfaListFactors: vi.fn().mockResolvedValue([]),
    listApprovals: vi.fn(),
  },
  setAccessToken: vi.fn(),
  getStoredTeamId: vi.fn().mockReturnValue('t1'),
  setStoredTeamId: vi.fn(),
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

// Mock useRefetch
vi.mock('../hooks/useRefetch', () => ({
  useRefetch: () => ({ bump: vi.fn(), version: 0 }),
}));

// Mock useWebSocketInvalidation - uses a stateful ref that can be triggered
let wsEventSetter: ((evt: any) => void) | null = null;
vi.mock('../hooks/useWebSocket', () => ({
  useWebSocketInvalidation: () => {
    const [lastEvent, setLastEvent] = (require('react') as any).useState(null);
    wsEventSetter = setLastEvent;
    return { lastEvent, connected: true };
  },
}));

import AdminApprovals from '../features/admin/AdminApprovals';
import { api } from '../api/client';

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter>
      <AuthProvider>{ui}</AuthProvider>
    </MemoryRouter>
  );
}

describe('AdminApprovals — Expiry Badges', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    wsEventSetter = null;
  });

  it('shows expiring badge for approval within threshold', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([
      {
        id: 'a1',
        action_type: 'proxmox.start',
        risk_level: 'medium',
        description: 'Start VM 100',
        status: 'pending',
        requested_by: 'u2',
        created_at: new Date(Date.now() - 70 * 1000).toISOString(),
        expires_at: new Date(Date.now() + 10 * 1000).toISOString(),
        remaining_seconds: 10,
        is_expiring: true,
      },
    ]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByTestId('expiring-badge')).toBeInTheDocument();
    });
  });

  it('shows expired badge for expired approval', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([
      {
        id: 'a2',
        action_type: 'proxmox.shutdown',
        risk_level: 'high',
        description: 'Shutdown VM 101',
        status: 'expired',
        requested_by: 'u2',
        created_at: new Date(Date.now() - 2 * 3600 * 1000).toISOString(),
        expires_at: new Date(Date.now() - 3600 * 1000).toISOString(),
        remaining_seconds: 0,
        is_expiring: false,
      },
    ]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByTestId('expired-badge')).toBeInTheDocument();
    });
  });

  it('shows active badge for approval with plenty of time', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([
      {
        id: 'a3',
        action_type: 'proxmox.snapshot',
        risk_level: 'low',
        description: 'Snapshot VM 100',
        status: 'pending',
        requested_by: 'u2',
        created_at: new Date().toISOString(),
        expires_at: new Date(Date.now() + 3600 * 1000).toISOString(),
        remaining_seconds: 3600,
        is_expiring: false,
      },
    ]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByTestId('active-badge')).toBeInTheDocument();
    });
  });

  it('does not show approve/reject buttons for expired approval', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([
      {
        id: 'a4',
        action_type: 'proxmox.start',
        risk_level: 'medium',
        description: 'Start VM 100',
        status: 'expired',
        requested_by: 'u2',
        created_at: new Date().toISOString(),
        expires_at: new Date().toISOString(),
        remaining_seconds: 0,
        is_expiring: false,
      },
    ]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByTestId('expired-badge')).toBeInTheDocument();
    });

    expect(screen.queryByTestId('approve-btn-a4')).not.toBeInTheDocument();
    expect(screen.queryByTestId('reject-btn-a4')).not.toBeInTheDocument();
  });

  it('handles approval.expiring WS event with toast', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByText(/No pending approvals/i)).toBeInTheDocument();
    });

    // Simulate WS event
    act(() => {
      if (wsEventSetter) wsEventSetter({
        event_type: 'approval.expiring',
        action_type: 'proxmox.start',
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId('approval-toast')).toBeInTheDocument();
      expect(screen.getByTestId('approval-toast').textContent).toContain('expiring');
    });
  });

  it('handles approval.expired WS event with toast', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByText(/No pending approvals/i)).toBeInTheDocument();
    });

    // Simulate WS event
    act(() => {
      if (wsEventSetter) wsEventSetter({
        event_type: 'approval.expired',
        action_type: 'proxmox.shutdown',
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId('approval-toast')).toBeInTheDocument();
      expect(screen.getByTestId('approval-toast').textContent).toContain('expired');
    });
  });

  it('status filter includes expired option', async () => {
    vi.mocked(api.listApprovals).mockResolvedValue([]);

    renderWithProviders(<AdminApprovals />);

    await waitFor(() => {
      expect(screen.getByText('expired')).toBeInTheDocument();
    });

    // Click expired filter
    fireEvent.click(screen.getByText('expired'));

    await waitFor(() => {
      expect(api.listApprovals).toHaveBeenCalledWith('expired');
    });
  });
});
