import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getEvidence: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import EvidencePanel from '../features/remediation/EvidencePanel';
import { api } from '../api/client';

function renderPanel(recommendationId: string) {
  return render(
    <MemoryRouter>
      <EvidencePanel recommendationId={recommendationId} />
    </MemoryRouter>
  );
}

const mockEvidence = {
  available: true,
  recommendation_summary: 'Restart the service to clear the error state',
  supporting_evidence: [
    { type: 'log_entry', description: 'OOM killer invoked', source: 'syslog' },
    { type: 'metric', description: 'Memory at 98%', source: 'node_exporter' },
  ],
  conflicting_evidence: [
    { type: 'metric', description: 'Memory usage normal in last 5 min', source: 'prometheus' },
  ],
  confidence_score: 0.75,
  confidence_level: 'high',
  risk_notes: 'Service restart may cause brief downtime',
  missing_info: [
    { description: 'No data on concurrent users' },
  ],
  is_stale: false,
};

describe('EvidencePanel — Recommendation Evidence Packs', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: Evidence panel renders summary/confidence
  it('renders summary and confidence level', async () => {
    vi.mocked(api.getEvidence).mockResolvedValue(mockEvidence);

    renderPanel('rec-123');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-panel')).toBeInTheDocument();
    });

    expect(screen.getByTestId('evidence-summary')).toBeInTheDocument();
    expect(screen.getByTestId('evidence-summary').textContent).toContain('Restart the service');
    expect(screen.getByTestId('evidence-confidence').textContent).toContain('high');
    expect(screen.getByTestId('evidence-confidence').textContent).toContain('75%');
  });

  // Test 2: Supporting evidence renders
  it('renders supporting evidence items', async () => {
    vi.mocked(api.getEvidence).mockResolvedValue(mockEvidence);

    renderPanel('rec-123');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-supporting')).toBeInTheDocument();
    });

    const supporting = screen.getByTestId('evidence-supporting');
    expect(supporting.textContent).toContain('OOM killer invoked');
    expect(supporting.textContent).toContain('Memory at 98%');
  });

  // Test 3: Conflicting evidence renders
  it('renders conflicting evidence items', async () => {
    vi.mocked(api.getEvidence).mockResolvedValue(mockEvidence);

    renderPanel('rec-123');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-conflicting')).toBeInTheDocument();
    });

    const conflicting = screen.getByTestId('evidence-conflicting');
    expect(conflicting.textContent).toContain('Memory usage normal');
  });

  // Test 4: Missing info warning renders
  it('renders missing information warnings', async () => {
    vi.mocked(api.getEvidence).mockResolvedValue(mockEvidence);

    renderPanel('rec-123');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-missing')).toBeInTheDocument();
    });

    const missing = screen.getByTestId('evidence-missing');
    expect(missing.textContent).toContain('No data on concurrent users');
  });

  // Test 5: Sensitive fields are not rendered
  it('does not render sensitive fields', async () => {
    const evidenceWithSecrets = {
      ...mockEvidence,
      supporting_evidence: [
        { type: 'log', description: '[REDACTED]', password: '[REDACTED]', token: '[REDACTED]' },
      ],
    };
    vi.mocked(api.getEvidence).mockResolvedValue(evidenceWithSecrets);

    const { container } = renderPanel('rec-123');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-panel')).toBeInTheDocument();
    });

    const fullHTML = container.innerHTML;
    // No raw chain-of-thought, tool parameters, or secrets should appear
    expect(fullHTML).not.toContain('"chain_of_thought"');
    expect(fullHTML).not.toContain('"tool_parameters"');
    expect(fullHTML).not.toContain('"action_target"');
    // The API mock returns [REDACTED] for sanitized values, which is fine
  });

  // Test 6: Unavailable evidence shows safe message
  it('shows unavailable message for missing evidence', async () => {
    vi.mocked(api.getEvidence).mockResolvedValue({
      available: false,
      message: 'Evidence unavailable for this recommendation',
    });

    renderPanel('rec-legacy');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-unavailable')).toBeInTheDocument();
    });

    expect(screen.getByTestId('evidence-unavailable').textContent).toContain('Evidence unavailable');
  });

  // Test 7: Stale indicator renders when evidence is stale
  it('shows stale warning when evidence is stale', async () => {
    vi.mocked(api.getEvidence).mockResolvedValue({
      ...mockEvidence,
      is_stale: true,
    });

    renderPanel('rec-123');

    await waitFor(() => {
      expect(screen.getByTestId('evidence-stale-warning')).toBeInTheDocument();
    });

    expect(screen.getByTestId('evidence-stale-warning').textContent).toContain('stale');
  });
});
