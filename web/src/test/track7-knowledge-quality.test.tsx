import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor, fireEvent } from '@testing-library/react';
import { legacyApiMethods, legacyApiExports } from './legacyApiMock';
import { renderWithProviders } from './renderWithProviders';

vi.mock('../api/client', () => ({
  api: {
    getQualityReport: vi.fn(),
    ...legacyApiMethods(),
  },
  ...legacyApiExports(),
}));

import { api } from '../api/client';
import { KnowledgeQualityPage } from '../features/knowledge/KnowledgeQualityPage';

function renderPage() {
  return renderWithProviders(<KnowledgeQualityPage />, { auth: true });
}

const mockReport = {
  team_id: 't1',
  total_items: 10,
  stale_count: 2,
  duplicate_count: 3,
  orphan_count: 1,
  by_type: { artifact: 5, incident: 3, clarity_document: 2 },
  stale_items: [
    { knowledge_item_id: 'k1', source_type: 'artifact', source_id: 'a-1', title: 'Old Doc', summary: 'Old', indexed_at: '', days_stale: 45 },
  ],
  duplicate_groups: [
    { content_hash: 'abcdef1234…', count: 3, items: [
      { knowledge_item_id: 'k2', source_type: 'artifact', source_id: 'a-2', title: 'Dup A', summary: '', indexed_at: '' },
      { knowledge_item_id: 'k3', source_type: 'artifact', source_id: 'a-3', title: 'Dup B', summary: '', indexed_at: '' },
    ] },
  ],
  orphan_items: [
    { knowledge_item_id: 'k4', source_type: 'clarity_document', source_id: 'd-1', title: 'Orphan Doc', summary: '', indexed_at: '' },
  ],
  generated_at: '2026-06-17T00:00:00Z',
};

describe('Track 7 — Knowledge Quality Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // 1: Quality page renders
  it('renders quality page', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Knowledge Quality')).toBeInTheDocument();
    });
  });

  // 2: Summary cards render
  it('renders summary cards with counts', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByTestId('quality-total')).toHaveTextContent('10');
      expect(screen.getByTestId('quality-stale')).toHaveTextContent('2');
      expect(screen.getByTestId('quality-duplicates')).toHaveTextContent('3');
      expect(screen.getByTestId('quality-orphans')).toHaveTextContent('1');
    });
  });

  // 3: By type breakdown renders
  it('renders by-type breakdown', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('By Source Type')).toBeInTheDocument();
    });
  });

  // 4: Stale items render
  it('renders stale items section', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByTestId('quality-stale-section')).toBeInTheDocument();
      expect(screen.getByTestId('quality-stale-item')).toBeInTheDocument();
    });
  });

  // 5: Duplicate groups render
  it('renders duplicate groups', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByTestId('quality-dup-section')).toBeInTheDocument();
      expect(screen.getByTestId('quality-dup-group')).toBeInTheDocument();
    });
  });

  // 6: Orphan items render
  it('renders orphan items', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByTestId('quality-orphan-section')).toBeInTheDocument();
      expect(screen.getByTestId('quality-orphan-item')).toBeInTheDocument();
    });
  });

  // 7: Loading state renders
  it('renders loading state', () => {
    vi.mocked(api.getQualityReport).mockReturnValue(new Promise(() => {}));
    renderPage();
    expect(screen.getByTestId('quality-loading')).toBeInTheDocument();
  });

  // 8: Error state renders safely
  it('renders safe error state', async () => {
    vi.mocked(api.getQualityReport).mockRejectedValue(new Error('DB error'));
    renderPage();
    await waitFor(() => {
      const err = screen.getByTestId('quality-error');
      expect(err).toBeInTheDocument();
      expect(err.textContent).not.toContain('DB error');
    });
  });

  // 9: All-clean state
  it('renders all-clean when no issues', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue({
      team_id: 't1', total_items: 5, stale_count: 0, duplicate_count: 0, orphan_count: 0,
      by_type: { artifact: 5 }, stale_items: [], duplicate_groups: [], orphan_items: [],
      generated_at: '',
    });
    renderPage();
    await waitFor(() => {
      expect(screen.getByTestId('quality-all-clean')).toBeInTheDocument();
    });
  });

  // 10: No operational controls
  it('does not render share, publish, or execute controls', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Knowledge Quality')).toBeInTheDocument();
    });
    expect(screen.queryByText(/share/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/publish/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/execute/i)).not.toBeInTheDocument();
  });

  // 11: No COT/raw prompt fields
  it('does not render chain_of_thought or prompt', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Knowledge Quality')).toBeInTheDocument();
    });
    expect(screen.queryByText(/chain_of_thought/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/^prompt$/i)).not.toBeInTheDocument();
  });

  // 12: Refresh button works
  it('refreshes report on button click', async () => {
    vi.mocked(api.getQualityReport).mockResolvedValue(mockReport);
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Knowledge Quality')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /Refresh/ }));
    await waitFor(() => {
      expect(api.getQualityReport).toHaveBeenCalledTimes(2);
    });
  });
});
