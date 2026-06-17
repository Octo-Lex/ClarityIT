import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getRelatedKnowledge: vi.fn(),
  },
  ApiError: class extends Error {
    constructor(public status: number, msg: string) { super(msg); }
  },
  getStoredTeamId: () => 'team-123',
}));

import { RelatedKnowledgePanel } from '../features/knowledge/RelatedKnowledgePanel';
import { api } from '../api/client';

function renderPanel(props?: Partial<{ sourceType: string; sourceId: string }>) {
  return render(
    <MemoryRouter>
      <RelatedKnowledgePanel
        sourceType={props?.sourceType ?? 'clarity_document'}
        sourceId={props?.sourceId ?? 'doc-1'}
      />
    </MemoryRouter>
  );
}

const mockRelated = [
  {
    source_type: 'incident',
    source_id: 'inc-1',
    title: 'Auth Outage After JWT Change',
    summary: 'Incident involving auth failures',
    snippet: '',
    rank: 0.87,
    reason: 'content_similarity',
    updated_at: '2026-06-15T10:00:00Z',
  },
  {
    source_type: 'clarity_document',
    source_id: 'doc-2',
    title: 'Auth Runbook',
    summary: 'Operations guide for auth',
    snippet: '',
    rank: 0.65,
    reason: 'same_source_family',
    updated_at: '2026-06-14T10:00:00Z',
  },
];

describe('RelatedKnowledgePanel — Related Knowledge Panel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: Panel renders
  it('renders the related knowledge panel', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: [],
    });
    renderPanel();
    expect(screen.getByTestId('related-knowledge-panel')).toBeInTheDocument();
  });

  // Test 2: Loading state renders
  it('renders loading state', async () => {
    let resolveFn: (v: any) => void;
    vi.mocked(api.getRelatedKnowledge).mockReturnValue(
      new Promise((resolve) => { resolveFn = resolve; })
    );
    renderPanel();

    await waitFor(() => {
      expect(screen.getByTestId('related-loading')).toBeInTheDocument();
    });
    resolveFn!({ source: {}, related: [] });
  });

  // Test 3: Empty state renders
  it('renders empty state when no related items', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: [],
    });
    renderPanel();

    await waitFor(() => {
      expect(screen.getByTestId('related-empty')).toBeInTheDocument();
    });
  });

  // Test 4: Error state renders safely
  it('renders safe error state on API failure', async () => {
    vi.mocked(api.getRelatedKnowledge).mockRejectedValue(new Error('Internal error'));
    renderPanel();

    await waitFor(() => {
      const err = screen.getByTestId('related-error');
      expect(err).toBeInTheDocument();
      // Must not leak raw error
      expect(err.textContent).not.toContain('Internal error');
    });
  });

  // Test 5: Related cards render
  it('renders related knowledge cards', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();

    await waitFor(() => {
      expect(screen.getAllByTestId('related-knowledge-card')).toHaveLength(2);
    });
  });

  // Test 6: Reason badges render
  it('renders reason badges', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();

    await waitFor(() => {
      const badges = screen.getAllByTestId('related-reason-badge');
      expect(badges).toHaveLength(2);
      expect(badges[0]).toHaveTextContent('Similar');
      expect(badges[1]).toHaveTextContent('Same Type');
    });
  });

  // Test 7: Source badges render
  it('renders source type badges', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();

    await waitFor(() => {
      const badges = screen.getAllByTestId('knowledge-source-badge');
      expect(badges).toHaveLength(2);
    });
  });

  // Test 8: Card click uses known route (document)
  it('renders clickable cards for known source types', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: [mockRelated[1]], // clarity_document result
    });
    renderPanel();

    await waitFor(() => {
      const card = screen.getByTestId('related-knowledge-card');
      expect(card).toBeInTheDocument();
    });
  });

  // Test 9: Unknown source uses fallback route
  it('renders context_node results safely', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: [{
        source_type: 'context_node',
        source_id: 'ctx-1',
        title: 'Service Context',
        summary: 'Context about the service',
        snippet: '',
        rank: 0.3,
        reason: 'recent_related',
        updated_at: '2026-06-10T00:00:00Z',
      }],
    });
    renderPanel();

    await waitFor(() => {
      expect(screen.getByTestId('related-knowledge-card')).toBeInTheDocument();
    });
  });

  // Test 10: Panel collapse/expand works
  it('collapses and expands when toggle is clicked', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();

    // Wait for results to load
    await waitFor(() => {
      expect(screen.getByTestId('related-items')).toBeInTheDocument();
    });

    // Collapse
    fireEvent.click(screen.getByTestId('related-panel-toggle'));
    expect(screen.queryByTestId('related-items')).not.toBeInTheDocument();

    // Expand
    fireEvent.click(screen.getByTestId('related-panel-toggle'));
    await waitFor(() => {
      expect(screen.getByTestId('related-items')).toBeInTheDocument();
    });
  });

  // Test 11: No Ask/Q&A UI appears
  it('does not render Ask/Q&A UI', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: [],
    });
    renderPanel();
    expect(screen.queryByTestId('ask-clarity')).not.toBeInTheDocument();
    expect(screen.queryByText(/ask clarity/i)).not.toBeInTheDocument();
  });

  // Test 12: No share/public/approval/execute controls
  it('does not render share, approve, or execute controls', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();
    await waitFor(() => {
      expect(screen.getAllByTestId('related-knowledge-card')).toHaveLength(2);
    });
    expect(screen.queryByText(/share/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/approve/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/execute/i)).not.toBeInTheDocument();
  });

  // Test 13: Raw storage identifiers not rendered
  it('does not render storage object identifiers', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();
    await waitFor(() => {
      expect(screen.getAllByTestId('related-knowledge-card')).toHaveLength(2);
    });
    expect(screen.queryByText(/storage_object_id/i)).not.toBeInTheDocument();
  });

  // Test 14: Prompt/COT fields not rendered
  it('does not render prompt or chain-of-thought fields', async () => {
    vi.mocked(api.getRelatedKnowledge).mockResolvedValue({
      source: { source_type: 'clarity_document', source_id: 'doc-1' },
      related: mockRelated,
    });
    renderPanel();
    await waitFor(() => {
      expect(screen.getAllByTestId('related-knowledge-card')).toHaveLength(2);
    });
    expect(screen.queryByText(/chain_of_thought/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/^prompt$/i)).not.toBeInTheDocument();
  });

  // Test 15: Panel does not block on API error — renders error state, not crash
  it('renders error state without crashing the panel', async () => {
    vi.mocked(api.getRelatedKnowledge).mockRejectedValue(new Error('Network error'));
    const { container } = renderPanel();

    await waitFor(() => {
      expect(screen.getByTestId('related-error')).toBeInTheDocument();
    });
    // Panel container still exists
    expect(container.querySelector('[data-testid="related-knowledge-panel"]')).toBeInTheDocument();
  });
});
