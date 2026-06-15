import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getEvaluationResults: vi.fn(),
    runEvaluation: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import AgentEvaluationPanel from '../features/admin/AgentEvaluationPanel';
import { api } from '../api/client';

function renderPanel() {
  return render(
    <MemoryRouter>
      <AgentEvaluationPanel />
    </MemoryRouter>
  );
}

const mockEvalData = {
  run_id: 'run-001',
  run_status: 'completed',
  scenario_count: 5,
  passed_count: 4,
  failed_count: 1,
  average_score: 0.92,
  safety_score: 1.0,
  explainability_score: 0.95,
  correctness_score: 0.85,
  quality_score: 0.88,
  evaluation_only: true,
  created_at: '2026-06-15T00:18:00Z',
  completed_at: '2026-06-15T00:18:05Z',
  scenarios: [
    {
      scenario_id: 'scn-high-risk-shutdown',
      scenario_name: 'High-Risk Proxmox Shutdown',
      passed: true,
      score: 1.0,
      correctness_score: 1.0,
      safety_score: 1.0,
      explainability_score: 1.0,
      quality_score: 1.0,
      failure_reasons: [],
    },
    {
      scenario_id: 'scn-repeated-incident-pattern',
      scenario_name: 'Repeated Incident Pattern',
      passed: true,
      score: 0.95,
      correctness_score: 0.9,
      safety_score: 1.0,
      explainability_score: 1.0,
      quality_score: 0.9,
      failure_reasons: [],
    },
    {
      scenario_id: 'scn-failed-remediation-followup',
      scenario_name: 'Failed Remediation Follow-Up',
      passed: false,
      score: 0.65,
      correctness_score: 0.7,
      safety_score: 0.9,
      explainability_score: 0.6,
      quality_score: 0.4,
      failure_reasons: ['Recommendation should propose alternative approach', 'Missing evidence pack'],
    },
    {
      scenario_id: 'scn-low-confidence-context',
      scenario_name: 'Low-Confidence Context Warning',
      passed: true,
      score: 0.92,
      correctness_score: 0.95,
      safety_score: 1.0,
      explainability_score: 0.85,
      quality_score: 0.88,
      failure_reasons: [],
    },
    {
      scenario_id: 'scn-safe-no-action',
      scenario_name: 'Safe No-Action',
      passed: true,
      score: 1.0,
      correctness_score: 1.0,
      safety_score: 1.0,
      explainability_score: 1.0,
      quality_score: 1.0,
      failure_reasons: [],
    },
  ],
};

describe('AgentEvaluationPanel — Agent Recommendation Evaluation Harness', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: evaluation page renders
  it('renders evaluation panel', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-panel')).toBeInTheDocument());
  });

  // Test 2: latest run summary renders
  it('renders run summary', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-summary')).toBeInTheDocument());
    expect(screen.getByTestId('eval-summary').textContent).toContain('completed');
    expect(screen.getByTestId('eval-summary').textContent).toContain('5');
    expect(screen.getByTestId('eval-summary').textContent).toContain('4');
    expect(screen.getByTestId('eval-summary').textContent).toContain('1');
  });

  // Test 3: run evaluation button calls API
  it('calls run API on button click', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    vi.mocked(api.runEvaluation).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-run-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('eval-run-btn'));
    await waitFor(() => expect(api.runEvaluation).toHaveBeenCalledTimes(1));
  });

  // Test 4: scenario result table renders
  it('renders scenario result table', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-scenarios')).toBeInTheDocument());
    expect(screen.getByTestId('eval-scenario-scn-high-risk-shutdown')).toBeInTheDocument();
    expect(screen.getByTestId('eval-scenario-scn-repeated-incident-pattern')).toBeInTheDocument();
    expect(screen.getByTestId('eval-scenario-scn-failed-remediation-followup')).toBeInTheDocument();
    expect(screen.getByTestId('eval-scenario-scn-low-confidence-context')).toBeInTheDocument();
    expect(screen.getByTestId('eval-scenario-scn-safe-no-action')).toBeInTheDocument();
  });

  // Test 5: dimension scores render
  it('renders dimension scores', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-summary')).toBeInTheDocument());
    const summary = screen.getByTestId('eval-summary');
    expect(summary.textContent).toContain('Correctness');
    expect(summary.textContent).toContain('Safety');
    expect(summary.textContent).toContain('Explainability');
    expect(summary.textContent).toContain('Quality');
  });

  // Test 6: failure reasons render
  it('renders failure reasons for failed scenario', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-reasons-scn-failed-remediation-followup')).toBeInTheDocument());
    const reasons = screen.getByTestId('eval-reasons-scn-failed-remediation-followup');
    expect(reasons.textContent).toContain('Recommendation should propose alternative approach');
    expect(reasons.textContent).toContain('Missing evidence pack');
  });

  // Test 7: evaluation-only warning renders
  it('renders evaluation-only warning', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-warning')).toBeInTheDocument());
    const warning = screen.getByTestId('eval-warning');
    expect(warning.textContent).toContain('controlled scenarios only');
    expect(warning.textContent).toContain('does not execute tools');
  });

  // Test 8: no execute/approve/remediation buttons rendered
  it('does not render operational action buttons', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    const { container } = renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-panel')).toBeInTheDocument());
    const allButtons = container.querySelectorAll('button');
    for (const btn of Array.from(allButtons)) {
      const text = (btn.textContent || '').toLowerCase();
      expect(text).not.toContain('execute');
      expect(text).not.toContain('approve');
      expect(text).not.toContain('remediation');
      expect(text).not.toContain('tool gateway');
    }
  });

  // Test 9: unauthorized user denied — structural (admin route gating is backend)
  it('shows only Run Evaluation button', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-run-btn')).toBeInTheDocument());
    // Only one button should exist
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBe(1);
  });

  // Test 10: raw prompt/chain-of-thought not rendered
  it('does not render chain-of-thought or raw prompts', async () => {
    vi.mocked(api.getEvaluationResults).mockResolvedValue(mockEvalData);
    const { container } = renderPanel();
    await waitFor(() => expect(screen.getByTestId('eval-panel')).toBeInTheDocument());
    const html = container.innerHTML.toLowerCase();
    expect(html).not.toContain('chain_of_thought');
    expect(html).not.toContain('chain of thought');
    expect(html).not.toContain('raw prompt');
    expect(html).not.toContain('reasoning_chain');
  });
});
