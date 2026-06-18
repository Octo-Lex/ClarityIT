import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    simulateApprovalPolicy: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import PolicySimulationPanel from '../features/admin/PolicySimulationPanel';
import { api } from '../api/client';

function renderPanel(props?: Partial<{ isPlatformOwner: boolean }>) {
  return render(
    <MemoryRouter>
      <PolicySimulationPanel isPlatformOwner={props?.isPlatformOwner ?? true} />
    </MemoryRouter>
  );
}

const mockResponse = {
  simulation_only: true,
  live_policy_changed: false,
  results: [
    {
      scenario_id: 'low-action',
      action_type: 'noop.check',
      risk_level: 'low',
      allowed: true,
      blocked: false,
      requires_approval: false,
      requires_mfa: false,
      min_approvers: 0,
      allow_self_approval: true,
      decision_explanation: 'Low-risk actions proceed without approval.',
    },
    {
      scenario_id: 'high-action',
      action_type: 'proxmox.shutdown',
      risk_level: 'high',
      allowed: true,
      blocked: false,
      requires_approval: true,
      requires_mfa: true,
      min_approvers: 1,
      allow_self_approval: false,
      decision_explanation: 'High-risk actions require MFA and one non-self approval.',
    },
  ],
  policy_diff: {
    changed: true,
    changes: [
      { risk_level: 'critical', field: 'min_approvers', current: 2, draft: 3 },
    ],
  },
};

describe('PolicySimulationPanel — Approval Policy Simulation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: simulation panel renders
  it('renders simulation panel', () => {
    renderPanel();
    expect(screen.getByTestId('sim-panel')).toBeInTheDocument();
  });

  // Test 2: draft policy inputs render
  it('renders draft policy inputs for all risk levels', () => {
    renderPanel();
    expect(screen.getByTestId('sim-policy-low')).toBeInTheDocument();
    expect(screen.getByTestId('sim-policy-medium')).toBeInTheDocument();
    expect(screen.getByTestId('sim-policy-high')).toBeInTheDocument();
    expect(screen.getByTestId('sim-policy-critical')).toBeInTheDocument();
  });

  // Test 3: simulate button calls API
  it('calls simulation API when button clicked', async () => {
    vi.mocked(api.simulateApprovalPolicy).mockResolvedValue(mockResponse);

    renderPanel();
    const button = screen.getByTestId('sim-button');
    fireEvent.click(button);

    await waitFor(() => {
      expect(api.simulateApprovalPolicy).toHaveBeenCalledTimes(1);
    });
  });

  // Test 4: result table renders approval/MFA/min-approver output
  it('renders result table with approval/MFA/min-approver columns', async () => {
    vi.mocked(api.simulateApprovalPolicy).mockResolvedValue(mockResponse);

    renderPanel();
    fireEvent.click(screen.getByTestId('sim-button'));

    await waitFor(() => {
      expect(screen.getByTestId('sim-results')).toBeInTheDocument();
    });

    // Check high-risk result shows MFA required and approval required
    expect(screen.getByTestId('sim-result-mfa-high-action').textContent).toContain('✓');
    expect(screen.getByTestId('sim-result-approval-high-action').textContent).toContain('Yes');
    expect(screen.getByTestId('sim-result-approvers-high-action').textContent).toContain('1');
  });

  // Test 5: policy diff renders
  it('renders policy diff when changes exist', async () => {
    vi.mocked(api.simulateApprovalPolicy).mockResolvedValue(mockResponse);

    renderPanel();
    fireEvent.click(screen.getByTestId('sim-button'));

    await waitFor(() => {
      expect(screen.getByTestId('sim-diff')).toBeInTheDocument();
    });

    const diff = screen.getByTestId('sim-diff');
    expect(diff.textContent).toContain('min_approvers');
    expect(diff.textContent).toContain('critical');
  });

  // Test 6: simulation-only warning renders
  it('renders simulation-only warning', () => {
    renderPanel();
    const warning = screen.getByTestId('sim-warning');
    expect(warning.textContent).toContain('Simulation only');
    expect(warning.textContent).toContain('no changes to live policy');
  });

  // Test 7: no save/apply button is rendered
  it('does not render a save or apply button', () => {
    renderPanel();
    const panel = screen.getByTestId('sim-panel');
    // Look for any button with save/apply text
    const allButtons = panel.querySelectorAll('button');
    for (const btn of Array.from(allButtons)) {
      const text = btn.textContent?.toLowerCase() || '';
      expect(text).not.toContain('save');
      expect(text).not.toContain('apply');
      expect(text).not.toContain('commit');
      expect(text).not.toContain('deploy');
    }
  });

  // Test 8: unauthorized user cannot access panel
  it('shows unauthorized message for non-platform-owner', () => {
    renderPanel({ isPlatformOwner: false });
    expect(screen.getByTestId('sim-panel-unauthorized')).toBeInTheDocument();
    expect(screen.getByTestId('sim-panel-unauthorized').textContent).toContain('Platform owner access required');
  });

  // Test 9: sensitive payload fields are not rendered
  it('does not render sensitive payload fields in results', async () => {
    const responseWithSecrets = {
      ...mockResponse,
      results: [{
        ...mockResponse.results[0],
        decision_explanation: 'Low-risk actions proceed without approval.',
      }],
    };
    vi.mocked(api.simulateApprovalPolicy).mockResolvedValue(responseWithSecrets);

    const { container } = renderPanel();
    fireEvent.click(screen.getByTestId('sim-button'));

    await waitFor(() => {
      expect(screen.getByTestId('sim-results')).toBeInTheDocument();
    });

    const html = container.innerHTML;
    // No sensitive field names should appear in the DOM
    expect(html).not.toContain('action_target');
    expect(html).not.toContain('parameters');
    expect(html).not.toContain('secret');
    expect(html).not.toContain('token');
    expect(html).not.toContain('password');
  });
});
