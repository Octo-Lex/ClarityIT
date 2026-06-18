import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';

// Mock the API client (legacy coexisting pattern; the component uses useQuery
// which calls api.getIncidentPatterns). getStoredTeamId is needed so the auth
// provider's activeTeamId enables the query.
vi.mock('../api/client', () => ({
  api: {
    getIncidentPatterns: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
  getStoredTeamId: () => 'team-1',
  setStoredTeamId: () => {},
  setAccessToken: () => {},
  getAccessToken: () => null,
}));

import PatternCards from '../features/incidents/PatternCards';
import { api } from '../api/client';
import { renderWithProviders } from './renderWithProviders';

function renderCards() {
  return renderWithProviders(<PatternCards />, { auth: true });
}

const mockPatterns = {
  patterns: [
    {
      pattern_id: 'abc123',
      pattern_type: 'recurring_asset',
      pattern_description: 'Asset vm-109 has 3 incidents in the analysis window.',
      confidence: 0.82,
      incident_ids: ['inc-1', 'inc-2', 'inc-3'],
      asset_ids: ['asset-109'],
      affected_assets: [{ asset_id: 'asset-109', name: 'vm-109', provider: 'proxmox' }],
      severity_mix: { critical: 0, high: 2, medium: 1, low: 0 },
      first_seen: '2026-06-10T10:00:00Z',
      last_seen: '2026-06-14T10:00:00Z',
      occurrence_count: 3,
      advisory_only: true,
    },
    {
      pattern_id: 'def456',
      pattern_type: 'cluster',
      pattern_description: '5 incidents occurred in close succession within the window.',
      confidence: 0.75,
      incident_ids: ['inc-4', 'inc-5', 'inc-6', 'inc-7', 'inc-8'],
      severity_mix: { critical: 1, high: 2, medium: 2, low: 0 },
      first_seen: '2026-06-13T08:00:00Z',
      last_seen: '2026-06-13T12:00:00Z',
      occurrence_count: 5,
      advisory_only: true,
    },
  ],
};

describe('PatternCards — Incident Pattern Detection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: pattern cards render on incidents list
  it('renders pattern cards when patterns exist', async () => {
    vi.mocked(api.getIncidentPatterns).mockResolvedValue(mockPatterns);

    renderCards();

    await waitFor(() => {
      expect(screen.getByTestId('pattern-cards')).toBeInTheDocument();
    });

    expect(screen.getByTestId('pattern-card-recurring_asset')).toBeInTheDocument();
    expect(screen.getByTestId('pattern-card-cluster')).toBeInTheDocument();
  });

  // Test 2: recurring asset pattern displays affected asset
  it('displays affected assets for recurring_asset pattern', async () => {
    vi.mocked(api.getIncidentPatterns).mockResolvedValue(mockPatterns);

    renderCards();

    await waitFor(() => {
      expect(screen.getByTestId('pattern-assets-recurring_asset')).toBeInTheDocument();
    });

    const assets = screen.getByTestId('pattern-assets-recurring_asset');
    expect(assets.textContent).toContain('vm-109');
  });

  // Test 3: confidence and occurrence count render
  it('renders confidence percentage and occurrence count', async () => {
    vi.mocked(api.getIncidentPatterns).mockResolvedValue(mockPatterns);

    renderCards();

    await waitFor(() => {
      expect(screen.getByTestId('pattern-confidence-recurring_asset')).toBeInTheDocument();
    });

    const confidence = screen.getByTestId('pattern-confidence-recurring_asset');
    expect(confidence.textContent).toContain('82%');

    const count = screen.getByTestId('pattern-count-recurring_asset');
    expect(count.textContent).toContain('3 incidents');
  });

  // Test 4: advisory badge renders
  it('renders advisory badge on each pattern card', async () => {
    vi.mocked(api.getIncidentPatterns).mockResolvedValue(mockPatterns);

    renderCards();

    await waitFor(() => {
      expect(screen.getByTestId('pattern-advisory-recurring_asset')).toBeInTheDocument();
    });

    const badge = screen.getByTestId('pattern-advisory-recurring_asset');
    expect(badge.textContent).toContain('Pattern detected');
    expect(badge.textContent).toContain('review recommended');
  });

  // Test 5: empty pattern state renders
  it('renders empty state when no patterns exist', async () => {
    vi.mocked(api.getIncidentPatterns).mockResolvedValue({ patterns: [] });

    renderCards();

    await waitFor(() => {
      expect(screen.getByTestId('pattern-empty')).toBeInTheDocument();
    });

    expect(screen.getByTestId('pattern-empty').textContent).toContain('No incident patterns detected');
  });

  // Test 6: raw incident body is not rendered
  it('does not render raw incident body content', async () => {
    const patternsWithSecrets = {
      patterns: [{
        pattern_id: 'secret1',
        pattern_type: 'recurring_symptom',
        pattern_description: '3 incidents share a common symptom pattern.',
        confidence: 0.7,
        incident_ids: ['inc-1', 'inc-2', 'inc-3'],
        severity_mix: { critical: 0, high: 1, medium: 2, low: 0 },
        first_seen: '2026-06-10T10:00:00Z',
        last_seen: '2026-06-14T10:00:00Z',
        occurrence_count: 3,
        advisory_only: true,
      }],
    };
    vi.mocked(api.getIncidentPatterns).mockResolvedValue(patternsWithSecrets);

    const { container } = renderCards();

    await waitFor(() => {
      expect(screen.getByTestId('pattern-card-recurring_symptom')).toBeInTheDocument();
    });

    const html = container.innerHTML;
    // No raw summary, body, chain_of_thought, or secret patterns should appear
    expect(html).not.toContain('"summary"');
    expect(html).not.toContain('"body"');
    expect(html).not.toContain('password');
    expect(html).not.toContain('secret');
    expect(html).not.toContain('token');
  });
});
