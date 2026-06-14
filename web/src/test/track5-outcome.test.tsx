import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getAssetActionOutcome: vi.fn(),
    saveAssetActionOutcome: vi.fn(),
    getRemediationOutcome: vi.fn(),
    saveRemediationOutcome: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import OutcomePanel from '../features/shared/OutcomePanel';
import { api } from '../api/client';

function renderPanel(props?: Partial<{ sourceType: string; sourceId: string; sourceStatus: string }>) {
  return render(
    <MemoryRouter>
      <OutcomePanel
        sourceType={(props?.sourceType as any) || 'asset-action'}
        sourceId={props?.sourceId || 'test-action-1'}
        sourceStatus={props?.sourceStatus || 'succeeded'}
      />
    </MemoryRouter>
  );
}

describe('OutcomePanel — Post-Action Outcome Tracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: outcome form renders for terminal asset action
  it('renders outcome form for succeeded action', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({ available: false });

    renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-form')).toBeInTheDocument();
    });
  });

  // Test 2: outcome form hidden for pending action
  it('shows not-terminal message for pending action', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({ available: false });

    renderPanel({ sourceStatus: 'pending' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-not-terminal')).toBeInTheDocument();
    });
  });

  // Test 3: outcome submit calls API
  it('calls save API on submit', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({ available: false });
    vi.mocked(api.saveAssetActionOutcome).mockResolvedValue({
      id: 'out-1',
      outcome_status: 'successful',
      available: true,
    });

    renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-form-submit')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('outcome-form-submit'));

    await waitFor(() => {
      expect(api.saveAssetActionOutcome).toHaveBeenCalledTimes(1);
    });
  });

  // Test 4: outcome detail renders when available
  it('renders outcome detail when available', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({
      available: true,
      id: 'out-1',
      outcome_status: 'successful',
      expected_result: 'VM shuts down',
      actual_result: 'VM shut down cleanly',
      operator_feedback: 'No issues',
      follow_up_recommendation: 'Monitor for 15 min',
      created_at: '2026-06-14T10:00:00Z',
    });

    renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-display')).toBeInTheDocument();
    });

    const display = screen.getByTestId('outcome-display');
    expect(display.textContent).toContain('successful');
    expect(display.textContent).toContain('VM shuts down');
    expect(display.textContent).toContain('VM shut down cleanly');
  });

  // Test 5: remediation outcome form renders for completed remediation
  it('renders form for completed remediation', async () => {
    vi.mocked(api.getRemediationOutcome).mockResolvedValue({ available: false });

    renderPanel({ sourceType: 'remediation', sourceStatus: 'completed' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-form')).toBeInTheDocument();
    });
  });

  // Test 6: warning text renders
  it('renders warning about no automatic retry', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({ available: false });

    renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-warning')).toBeInTheDocument();
    });

    const warning = screen.getByTestId('outcome-warning');
    expect(warning.textContent).toContain('not trigger any automatic retry');
  });

  // Test 7: no retry/follow-up execute button rendered
  it('does not render retry or execute-follow-up buttons', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({ available: false });

    renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-form')).toBeInTheDocument();
    });

    const panel = screen.getByTestId('outcome-panel');
    const allButtons = panel.querySelectorAll('button');
    for (const btn of Array.from(allButtons)) {
      const text = (btn.textContent || '').toLowerCase();
      expect(text).not.toContain('retry');
      expect(text).not.toContain('execute');
      expect(text).not.toContain('run follow');
    }
  });

  // Test 8: error state renders safely
  it('renders error state when save fails', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({ available: false });
    vi.mocked(api.saveAssetActionOutcome).mockRejectedValue(new Error('Network error'));

    renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-form-submit')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('outcome-form-submit'));

    await waitFor(() => {
      expect(screen.getByTestId('outcome-error')).toBeInTheDocument();
    });
  });

  // Test 9: sensitive fields are not rendered
  it('does not render sensitive fields in outcome display', async () => {
    vi.mocked(api.getAssetActionOutcome).mockResolvedValue({
      available: true,
      outcome_status: 'successful',
      actual_result: '[REDACTED]',
      expected_result: 'Expected behavior',
    });

    const { container } = renderPanel({ sourceStatus: 'succeeded' });

    await waitFor(() => {
      expect(screen.getByTestId('outcome-display')).toBeInTheDocument();
    });

    const html = container.innerHTML.toLowerCase();
    expect(html).not.toContain('password');
    expect(html).not.toContain('secret');
    expect(html).not.toContain('token');
  });
});
