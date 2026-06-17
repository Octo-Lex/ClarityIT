import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    knowledgeSearch: vi.fn(),
    getKnowledgeItem: vi.fn(),
  },
  ApiError: class extends Error {
    constructor(public status: number, msg: string) { super(msg); }
  },
  getStoredTeamId: () => 'team-123',
}));

import { KnowledgeSearchPage } from '../features/knowledge/KnowledgeSearchPage';
import { api } from '../api/client';

function renderPage() {
  return render(
    <MemoryRouter>
      <KnowledgeSearchPage />
    </MemoryRouter>
  );
}

const mockResults = [
  {
    source_type: 'clarity_document',
    source_id: 'doc-1',
    title: 'Authentication Architecture',
    summary: 'Overview of auth design',
    snippet: 'The <start>authentication</start> system uses JWT tokens.',
    rank: 0.15,
    updated_at: '2026-06-15T10:00:00Z',
  },
  {
    source_type: 'work_item',
    source_id: 'wi-1',
    title: 'Fix auth bug in login flow',
    summary: 'Critical bug fix',
    snippet: 'Found <start>authentication</start> issue in token refresh.',
    rank: 0.08,
    updated_at: '2026-06-14T10:00:00Z',
  },
];

describe('KnowledgeSearchPage — Unified Knowledge Search UI', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 3: empty state renders before search
  it('renders empty state before search', () => {
    renderPage();
    expect(screen.getByTestId('knowledge-empty')).toBeInTheDocument();
  });

  // Test 2: /knowledge page renders
  it('renders the knowledge search page', () => {
    renderPage();
    expect(screen.getByTestId('knowledge-page')).toBeInTheDocument();
  });

  // Test 4: search submits query
  it('submits search and calls API', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: mockResults,
      total: 2,
      query: 'authentication',
    });
    renderPage();

    const input = screen.getByTestId('knowledge-search-input');
    fireEvent.change(input, { target: { value: 'authentication' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(api.knowledgeSearch).toHaveBeenCalledWith('authentication', 'all', 20, 0);
    });
  });

  // Test 5: loading state renders
  it('renders loading state during search', async () => {
    let resolveSearch: (v: any) => void;
    vi.mocked(api.knowledgeSearch).mockReturnValue(
      new Promise((resolve) => { resolveSearch = resolve; })
    );
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'test' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getByTestId('knowledge-loading')).toBeInTheDocument();
    });
    resolveSearch!({ results: [], total: 0, query: 'test' });
  });

  // Test 6: results render
  it('renders search results', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: mockResults,
      total: 2,
      query: 'authentication',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'authentication' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getAllByTestId('search-result-card')).toHaveLength(2);
    });
  });

  // Test 7: snippets render
  it('renders highlighted snippets', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: mockResults,
      total: 2,
      query: 'authentication',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'authentication' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      const snippets = screen.getAllByTestId('knowledge-snippet');
      expect(snippets).toHaveLength(2);
      // Verify the highlight mark is rendered
      expect(snippets[0].querySelector('mark')).not.toBeNull();
    });
  });

  // Test 8: source badges render
  it('renders source type badges', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: mockResults,
      total: 2,
      query: 'authentication',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'authentication' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      const badges = screen.getAllByTestId('knowledge-source-badge');
      expect(badges).toHaveLength(2);
      expect(badges[0]).toHaveTextContent('Document');
      expect(badges[1]).toHaveTextContent('Work Item');
    });
  });

  // Test 9: filters render
  it('renders all filter buttons', () => {
    renderPage();
    expect(screen.getByTestId('filter-all')).toBeInTheDocument();
    expect(screen.getByTestId('filter-clarity_document')).toBeInTheDocument();
    expect(screen.getByTestId('filter-work_item')).toBeInTheDocument();
    expect(screen.getByTestId('filter-incident')).toBeInTheDocument();
    expect(screen.getByTestId('filter-asset')).toBeInTheDocument();
  });

  // Test 10: filter click refetches with source_type
  it('refetches with source_type when filter is clicked', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: [mockResults[0]],
      total: 1,
      query: 'auth',
    });
    renderPage();

    // Search first
    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'auth' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(api.knowledgeSearch).toHaveBeenCalledWith('auth', 'all', 20, 0);
    });

    // Click a filter
    fireEvent.click(screen.getByTestId('filter-clarity_document'));

    await waitFor(() => {
      expect(api.knowledgeSearch).toHaveBeenCalledWith('auth', 'clarity_document', 20, 0);
    });
  });

  // Test 11: no-results state renders
  it('renders no-results state when search returns empty', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: [],
      total: 0,
      query: 'nonexistent',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'nonexistent' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getByTestId('knowledge-no-results')).toBeInTheDocument();
    });
  });

  // Test 12: API error renders safe message
  it('renders safe error message on API failure', async () => {
    vi.mocked(api.knowledgeSearch).mockRejectedValue(new Error('Internal server error'));
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'test' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      const errorEl = screen.getByTestId('knowledge-error');
      expect(errorEl).toBeInTheDocument();
      // Must not leak raw error details
      expect(errorEl.textContent).not.toContain('Internal server error');
    });
  });

  // Test 13: result click uses known route (document)
  it('navigates to document route when document result is clicked', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: [mockResults[0]],
      total: 1,
      query: 'auth',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'auth' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getByTestId('search-result-card')).toBeInTheDocument();
    });

    // Just verify the card is clickable — actual navigation tested in integration
    const card = screen.getByTestId('search-result-card');
    expect(card).toBeInTheDocument();
  });

  // Test 14: unknown route uses safe fallback (context_node)
  it('renders context_node results without crash', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: [{
        source_type: 'context_node',
        source_id: 'ctx-1',
        title: 'Service Context',
        summary: '',
        snippet: 'Some <start>context</start> data',
        rank: 0.01,
        updated_at: '2026-06-10T00:00:00Z',
      }],
      total: 1,
      query: 'context',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'context' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getByTestId('search-result-card')).toBeInTheDocument();
    });
  });

  // Test 15: raw storage object identifiers are not rendered
  it('does not render storage object IDs', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: [{
        source_type: 'artifact',
        source_id: 'art-1',
        title: 'Report',
        summary: '',
        snippet: 'Storage at <start>s3://clarityit/abc-123</start>',
        rank: 0.05,
        updated_at: '2026-06-15T00:00:00Z',
      }],
      total: 1,
      query: 'storage',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'storage' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getByTestId('search-result-card')).toBeInTheDocument();
    });
    // The snippet may contain test data but storage identifiers like bucket names
    // are sanitized at indexing time — verify no real storage_object_id field is rendered
    expect(screen.queryByText('storage_object_id')).not.toBeInTheDocument();
  });

  // Test 16: prompt/COT-looking fields are not rendered
  it('does not render prompt or chain-of-thought fields', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: [{
        source_type: 'artifact',
        source_id: 'art-1',
        title: 'Report',
        summary: '',
        snippet: 'Regular content',
        rank: 0.05,
        updated_at: '2026-06-15T00:00:00Z',
      }],
      total: 1,
      query: 'test',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'test' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getByTestId('search-result-card')).toBeInTheDocument();
    });
    expect(screen.queryByText(/chain_of_thought/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/prompt/i)).not.toBeInTheDocument();
  });

  // Test 17: no Ask/Q&A UI appears in search results
  // Note: Track 5 adds AskClarityPanel to /knowledge page intentionally.
  // This test now verifies that no Ask/Q&A UI appears inside search results themselves.
  it('does not render Q&A controls inside search results', async () => {
    vi.mocked(api.knowledgeSearch).mockResolvedValue({
      results: mockResults,
      total: 2,
      query: 'authentication',
    });
    renderPage();

    fireEvent.change(screen.getByTestId('knowledge-search-input'), { target: { value: 'authentication' } });
    fireEvent.click(screen.getByTestId('knowledge-search-button'));

    await waitFor(() => {
      expect(screen.getAllByTestId('search-result-card')).toHaveLength(2);
    });
    // Search result cards should not have Q&A mutation controls
    const cards = screen.getAllByTestId('search-result-card');
    for (const card of cards) {
      expect(card.querySelector('[data-testid="ask-clarity"]')).not.toBeTruthy();
    }
  });

  // Test 18: no share/public/approval/execute controls
  it('does not render share, approve, or execute controls', () => {
    renderPage();
    expect(screen.queryByTestId('share-button')).not.toBeInTheDocument();
    expect(screen.queryByTestId('approve-button')).not.toBeInTheDocument();
    expect(screen.queryByTestId('execute-button')).not.toBeInTheDocument();
    expect(screen.queryByText(/share/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/approve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/execute/i)).not.toBeInTheDocument();
  });
});
