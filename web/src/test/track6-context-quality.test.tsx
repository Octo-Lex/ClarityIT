import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getContextQuality: vi.fn(),
    confirmRelation: vi.fn(),
    dismissRelation: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ContextQualityPanel from '../features/admin/ContextQualityPanel';
import { api } from '../api/client';

function renderPanel() {
  return render(
    <MemoryRouter>
      <ContextQualityPanel />
    </MemoryRouter>
  );
}

const mockQuality = {
  quality_score: 84,
  advisory_only: true,
  summary: {
    total_nodes: 42,
    total_relations: 91,
    stale_nodes: 3,
    low_confidence_relations: 5,
    conflicting_relations: 1,
    confirmed_relations: 18,
    dismissed_relations: 3,
  },
  stale_nodes: [
    { node_id: 'n1', node_type: 'asset', label: 'vm-109', days_stale: 41, reason: 'Not updated in 30+ days' },
    { node_id: 'n2', node_type: 'service', label: 'api-gw', days_stale: 52, reason: 'Not updated in 30+ days' },
  ],
  low_confidence_relations: [
    { relation_id: 'r1', relation_type: 'depends_on', confidence: 0.42, reason: 'Below threshold' },
    { relation_id: 'r2', relation_type: 'related_to', confidence: 0.35, reason: 'Below threshold' },
  ],
  conflicting_relations: [
    { relation_id: 'r3', relation_type: 'blocks', conflict_reason: 'Multiple contradictory relation types' },
  ],
};

describe('ContextQualityPanel — Context Graph Quality Controls', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: context quality card renders
  it('renders quality panel', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-panel')).toBeInTheDocument());
  });

  // Test 2: quality score and summary counts render
  it('renders score and summary counts', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-score')).toBeInTheDocument());
    expect(screen.getByTestId('quality-score').textContent).toContain('84');
    expect(screen.getByTestId('quality-summary').textContent).toContain('42');
    expect(screen.getByTestId('quality-summary').textContent).toContain('91');
  });

  // Test 3: stale node list renders
  it('renders stale node list', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-stale')).toBeInTheDocument());
    expect(screen.getByTestId('quality-stale').textContent).toContain('vm-109');
    expect(screen.getByTestId('quality-stale').textContent).toContain('api-gw');
  });

  // Test 4: low-confidence relation list renders
  it('renders low-confidence relations', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-low-conf')).toBeInTheDocument());
    expect(screen.getByTestId('lowconf-r1')).toBeInTheDocument();
    expect(screen.getByTestId('lowconf-r2')).toBeInTheDocument();
  });

  // Test 5: conflicting relation list renders
  it('renders conflicting relations', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-conflicts')).toBeInTheDocument());
    expect(screen.getByTestId('conflict-r3')).toBeInTheDocument();
  });

  // Test 6: advisory-only warning renders
  it('renders advisory warning', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-warning')).toBeInTheDocument());
    const warning = screen.getByTestId('quality-warning');
    expect(warning.textContent).toContain('advisory only');
    expect(warning.textContent).toContain('does not delete graph data');
  });

  // Test 7: confirm relation calls API
  it('calls confirm API on confirm button click', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    vi.mocked(api.confirmRelation).mockResolvedValue({ quality_status: 'confirmed' });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('confirm-r1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('confirm-r1'));
    await waitFor(() => expect(api.confirmRelation).toHaveBeenCalledTimes(1));
  });

  // Test 8: dismiss relation calls API
  it('calls dismiss API on dismiss button click', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    vi.mocked(api.dismissRelation).mockResolvedValue({ quality_status: 'dismissed' });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId('dismiss-r1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('dismiss-r1'));
    await waitFor(() => expect(api.dismissRelation).toHaveBeenCalledTimes(1));
  });

  // Test 9: no delete/auto-fix/agent-run buttons rendered
  it('does not render delete or auto-fix buttons', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    const { container } = renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-panel')).toBeInTheDocument());
    const allButtons = container.querySelectorAll('button');
    for (const btn of Array.from(allButtons)) {
      const text = (btn.textContent || '').toLowerCase();
      expect(text).not.toContain('delete');
      expect(text).not.toContain('auto-fix');
      expect(text).not.toContain('regenerate');
      expect(text).not.toContain('agent-run');
    }
  });

  // Test 10: sensitive metadata is not rendered
  it('does not render sensitive metadata', async () => {
    vi.mocked(api.getContextQuality).mockResolvedValue(mockQuality);
    const { container } = renderPanel();
    await waitFor(() => expect(screen.getByTestId('quality-panel')).toBeInTheDocument());
    const html = container.innerHTML.toLowerCase();
    expect(html).not.toContain('password');
    expect(html).not.toContain('secret');
    expect(html).not.toContain('token');
    expect(html).not.toContain('tool_parameters');
  });
});
