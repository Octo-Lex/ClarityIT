import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../auth/context';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    me: vi.fn().mockResolvedValue({ id: 'u1', email: 'owner@test.dev', name: 'Owner', teams: [{ id: 't1', name: 'Team', slug: 'team', role: 'owner' }] }),
    permissions: vi.fn().mockResolvedValue({ role: 'owner', team_id: 't1', permissions: [] }),
    getMetrics: vi.fn(),
  },
  setAccessToken: vi.fn(),
  getStoredTeamId: vi.fn().mockReturnValue('t1'),
  setStoredTeamId: vi.fn(),
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import AdminMetrics from '../features/admin/AdminMetrics';
import { api } from '../api/client';

const mockMetricsData = {
  approvals: {
    pending: 3,
    approved: 10,
    rejected: 2,
    expired: 1,
    executed: 8,
    failed: 1,
    avg_time_to_decision_seconds: 120.5,
  },
  remediations: {
    draft: 2,
    proposed: 1,
    approved: 3,
    executing: 1,
    completed: 5,
    failed: 0,
    cancelled: 2,
  },
  asset_actions: {
    by_status: { pending: 4, approved: 2, executing: 1, succeeded: 8, failed: 2, cancelled: 1 },
    by_type: { 'proxmox.start': 3, 'proxmox.shutdown': 2, 'proxmox.stop': 1, 'proxmox.snapshot': 5 },
    success_rate_percent: 80.0,
  },
  agents: {
    runs_pending: 1,
    runs_running: 0,
    runs_completed: 15,
    runs_failed: 2,
    avg_reasoning_time_seconds: 45.3,
  },
};

const emptyMetricsData = {
  approvals: {
    pending: 0, approved: 0, rejected: 0, expired: 0, executed: 0, failed: 0,
    avg_time_to_decision_seconds: 0,
  },
  remediations: {
    draft: 0, proposed: 0, approved: 0, executing: 0, completed: 0, failed: 0, cancelled: 0,
  },
  asset_actions: {
    by_status: {},
    by_type: {},
    success_rate_percent: 0,
  },
  agents: {
    runs_pending: 0, runs_running: 0, runs_completed: 0, runs_failed: 0,
    avg_reasoning_time_seconds: 0,
  },
};

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter>
      <AuthProvider>{ui}</AuthProvider>
    </MemoryRouter>
  );
}

describe('AdminMetrics — Operational Metrics', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset default mock implementations
    vi.mocked(api.me).mockResolvedValue({ id: 'u1', email: 'owner@test.dev', name: 'Owner', teams: [{ id: 't1', name: 'Team', slug: 'team', role: 'owner' }] });
    vi.mocked(api.permissions).mockResolvedValue({ role: 'owner', team_id: 't1', permissions: [] });
    vi.mocked(api.getMetrics).mockResolvedValue(mockMetricsData);
  });

  // Test 1: Metrics page renders approval cards
  it('renders approval metric cards', async () => {
    vi.mocked(api.getMetrics).mockResolvedValue(mockMetricsData);

    renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-approvals')).toBeInTheDocument();
    });

    expect(screen.getByTestId('approvals-pending').textContent).toBe('3');
    expect(screen.getByTestId('approvals-approved').textContent).toBe('10');
    expect(screen.getByTestId('approvals-rejected').textContent).toBe('2');
    expect(screen.getByTestId('approvals-expired').textContent).toBe('1');
    expect(screen.getByTestId('approvals-executed').textContent).toBe('8');
    expect(screen.getByTestId('approvals-failed').textContent).toBe('1');
  });

  // Test 2: Metrics page renders remediation cards
  it('renders remediation metric cards', async () => {
    vi.mocked(api.getMetrics).mockResolvedValue(mockMetricsData);

    renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-remediations')).toBeInTheDocument();
    });

    expect(screen.getByTestId('remediations-draft').textContent).toBe('2');
    expect(screen.getByTestId('remediations-completed').textContent).toBe('5');
    expect(screen.getByTestId('remediations-failed').textContent).toBe('0');
  });

  // Test 3: Metrics page renders asset action cards
  it('renders asset action metric cards', async () => {
    vi.mocked(api.getMetrics).mockResolvedValue(mockMetricsData);

    renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-asset-actions')).toBeInTheDocument();
    });

    expect(screen.getByTestId('assets-succeeded').textContent).toBe('8');
    expect(screen.getByTestId('assets-start').textContent).toBe('3');
    expect(screen.getByTestId('assets-shutdown').textContent).toBe('2');
    expect(screen.getByTestId('assets-stop').textContent).toBe('1');
    expect(screen.getByTestId('assets-snapshot').textContent).toBe('5');
    expect(screen.getByTestId('metrics-success-rate').textContent).toContain('80.0%');
  });

  // Test 4: Metrics page renders agent cards
  it('renders agent run metric cards', async () => {
    vi.mocked(api.getMetrics).mockResolvedValue(mockMetricsData);

    renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-agents')).toBeInTheDocument();
    });

    expect(screen.getByTestId('agents-completed').textContent).toBe('15');
    expect(screen.getByTestId('agents-failed').textContent).toBe('2');
    expect(screen.getByTestId('metrics-avg-reasoning').textContent).toContain('45.3s');
  });

  // Test 5: Empty state renders zeros
  it('renders zeros for empty state', async () => {
    vi.mocked(api.getMetrics).mockResolvedValue(emptyMetricsData);

    renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-page')).toBeInTheDocument();
    });

    // All values should be 0
    expect(screen.getByTestId('approvals-pending').textContent).toBe('0');
    expect(screen.getByTestId('approvals-approved').textContent).toBe('0');
    expect(screen.getByTestId('remediations-draft').textContent).toBe('0');
    expect(screen.getByTestId('remediations-completed').textContent).toBe('0');
    expect(screen.getByTestId('metrics-success-rate').textContent).toContain('0.0%');
  });

  // Test 6: Permission gating hides page from unauthorized user
  it('shows permission denied for non-admin user', async () => {
    // Mock permissions to return non-owner role
    vi.mocked(api.permissions).mockResolvedValue({ role: 'member', team_id: 't1', permissions: [] });

    renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-permission-denied')).toBeInTheDocument();
    });
  });

  // Test 7: No raw sensitive fields rendered
  it('does not render sensitive fields', async () => {
    vi.mocked(api.getMetrics).mockResolvedValue(mockMetricsData);

    const { container } = renderWithProviders(<AdminMetrics />);

    await waitFor(() => {
      expect(screen.getByTestId('metrics-page')).toBeInTheDocument();
    });

    const fullHTML = container.innerHTML;
    const sensitiveFields = [
      'action_target', 'payload', 'parameters', 'tool_parameters',
      'comment', 'reasoning_summary', 'token', 'secret', 'password',
      'public_key', 'credential_id', 'snapshot_name',
    ];
    for (const field of sensitiveFields) {
      expect(fullHTML).not.toContain(`"${field}"`);
    }
  });
});
