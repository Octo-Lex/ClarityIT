import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor, fireEvent } from '@testing-library/react';
import { legacyApiMethods, legacyApiExports } from './legacyApiMock';
import { renderWithProviders } from './renderWithProviders';

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: vi.fn(() => ({})),
  };
});

vi.mock('../api/client', () => ({
  api: {
    listCollections: vi.fn(),
    createCollection: vi.fn(),
    getCollection: vi.fn(),
    patchCollection: vi.fn(),
    deleteCollection: vi.fn(),
    addCollectionItem: vi.fn(),
    removeCollectionItem: vi.fn(),
    saveAnswer: vi.fn(),
    listSavedAnswers: vi.fn(),
    getSavedAnswer: vi.fn(),
    deleteSavedAnswer: vi.fn(),
    askClarity: vi.fn(),
    knowledgeSearch: vi.fn(),
    ...legacyApiMethods(),
  },
  ...legacyApiExports(),
}));

import { useParams } from 'react-router-dom';
import { api } from '../api/client';
import { KnowledgeCollectionsPage } from '../features/knowledge/KnowledgeCollectionsPage';
import { KnowledgeCollectionDetailPage } from '../features/knowledge/KnowledgeCollectionDetailPage';
import { SaveToCollectionDialog } from '../features/knowledge/SaveToCollectionDialog';
import { SavedKnowledgeAnswersPage } from '../features/knowledge/SavedKnowledgeAnswersPage';
import { AskClarityAnswer } from '../features/knowledge/AskClarityAnswer';
import { AskClaritySourceCard } from '../features/knowledge/AskClaritySourceCard';

function renderWithRouter(el: React.ReactElement, path = '/') {
  return renderWithProviders(el, { auth: true, route: path });
}

describe('Track 6 — Knowledge Collections', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // 1: Collections page lists collections
  it('lists collections', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({
      collections: [
        { id: 'c1', team_id: 't', name: 'Incident Playbooks', description: '', created_by: 'u1', created_at: '', updated_at: '', archived_at: null, item_count: 3 },
        { id: 'c2', team_id: 't', name: 'Runbooks', description: 'Useful docs', created_by: 'u1', created_at: '', updated_at: '', archived_at: null, item_count: 0 },
      ],
    });
    renderWithRouter(<KnowledgeCollectionsPage />);
    await waitFor(() => {
      expect(screen.getAllByTestId('collection-card')).toHaveLength(2);
    });
    expect(screen.getByText('Incident Playbooks')).toBeInTheDocument();
  });

  // 2: Create collection dialog works
  it('creates collection', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({ collections: [] });
    vi.mocked(api.createCollection).mockResolvedValue({
      id: 'c-new', team_id: 't', name: 'New Collection', description: '', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0,
    });

    renderWithRouter(<KnowledgeCollectionsPage />);
    await waitFor(() => {
      expect(screen.getByTestId('collections-empty')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('create-collection-btn'));
    fireEvent.change(screen.getByTestId('collection-name-input'), { target: { value: 'New Collection' } });
    fireEvent.click(screen.getByTestId('collection-create-confirm'));

    await waitFor(() => {
      expect(api.createCollection).toHaveBeenCalledWith('New Collection', undefined);
    });
  });

  // 3: Archive collection works
  it('archives collection', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({
      collections: [
        { id: 'c1', team_id: 't', name: 'To Archive', description: '', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0 },
      ],
    });
    vi.mocked(api.deleteCollection).mockResolvedValue({ status: 'archived' });

    renderWithRouter(<KnowledgeCollectionsPage />);
    await waitFor(() => {
      expect(screen.getByTestId('collection-card')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('archive-collection-c1'));

    await waitFor(() => {
      expect(api.deleteCollection).toHaveBeenCalledWith('c1');
    });
  });

  // 4: Collection detail renders items
  it('collection detail renders items', async () => {
    vi.mocked(api.getCollection).mockResolvedValue({
      id: 'c1', team_id: 't', name: 'Test', description: 'Desc', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 2,
      items: [
        { id: 'i1', collection_id: 'c1', team_id: 't', source_type: 'incident', source_id: 'inc-1', title: 'DB Outage', summary: 'Summary 1', added_by: null, added_at: '' },
        { id: 'i2', collection_id: 'c1', team_id: 't', source_type: 'clarity_document', source_id: 'doc-1', title: 'Backup Doc', summary: 'Summary 2', added_by: null, added_at: '' },
      ],
    });
    vi.mocked(useParams).mockReturnValue({ collectionId: 'c1' });

    renderWithRouter(<KnowledgeCollectionDetailPage />);

    await waitFor(() => {
      expect(screen.getAllByTestId('collection-item-card')).toHaveLength(2);
    });
    expect(screen.getByText('DB Outage')).toBeInTheDocument();
  });

  // 5: Empty collection state renders
  it('renders empty collection state', async () => {
    vi.mocked(api.getCollection).mockResolvedValue({
      id: 'c1', team_id: 't', name: 'Empty', description: '', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0,
      items: [],
    });
    vi.mocked(useParams).mockReturnValue({ collectionId: 'c1' });

    renderWithRouter(<KnowledgeCollectionDetailPage />);

    await waitFor(() => {
      expect(screen.getByTestId('collection-items-empty')).toBeInTheDocument();
    });
  });

  // 6: SaveToCollectionDialog renders
  it('renders save-to-collection dialog', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({
      collections: [{ id: 'c1', team_id: 't', name: 'Collection A', description: '', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0 }],
    });

    renderWithRouter(<SaveToCollectionDialog sourceType="incident" sourceId="inc-1" onClose={() => {}} />);

    await waitFor(() => {
      expect(screen.getByTestId('save-to-collection-dialog')).toBeInTheDocument();
    });
    expect(screen.getByText('Save to Collection')).toBeInTheDocument();
  });

  // 7: SaveToCollectionDialog handles empty collections
  it('renders empty state when no collections', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({ collections: [] });

    renderWithRouter(<SaveToCollectionDialog sourceType="incident" sourceId="inc-1" onClose={() => {}} />);

    await waitFor(() => {
      expect(screen.getByTestId('save-dialog-empty')).toBeInTheDocument();
    });
  });

  // 8: SaveToCollectionDialog handles duplicate
  it('handles duplicate item safely', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({
      collections: [{ id: 'c1', team_id: 't', name: 'Collection A', description: '', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0 }],
    });
    vi.mocked(api.addCollectionItem).mockResolvedValue({
      item: { id: 'i1', collection_id: 'c1', team_id: 't', source_type: 'incident', source_id: 'inc-1', added_by: null, added_at: '' },
      duplicate: true,
    });

    renderWithRouter(<SaveToCollectionDialog sourceType="incident" sourceId="inc-1" onClose={() => {}} />);

    await waitFor(() => {
      expect(screen.getByTestId('save-dialog-select')).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId('save-dialog-select'), { target: { value: 'c1' } });
    fireEvent.click(screen.getByTestId('save-dialog-confirm'));

    await waitFor(() => {
      expect(screen.getByTestId('save-dialog-success')).toBeInTheDocument();
      expect(screen.getByText(/already in this collection/i)).toBeInTheDocument();
    });
  });

  // 9: Edit collection works
  it('edits collection name', async () => {
    vi.mocked(api.getCollection).mockResolvedValue({
      id: 'c1', team_id: 't', name: 'Old Name', description: 'Old desc', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0,
      items: [],
    });
    vi.mocked(api.patchCollection).mockResolvedValue({
      id: 'c1', team_id: 't', name: 'New Name', description: 'New desc', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 0,
    });
    vi.mocked(useParams).mockReturnValue({ collectionId: 'c1' });

    renderWithRouter(<KnowledgeCollectionDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('Old Name')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('edit-collection-btn'));
    fireEvent.change(screen.getByTestId('edit-name-input'), { target: { value: 'New Name' } });
    fireEvent.click(screen.getByTestId('edit-save-btn'));

    await waitFor(() => {
      expect(api.patchCollection).toHaveBeenCalledWith('c1', { name: 'New Name', description: 'Old desc' });
    });
  });

  // 10: Saved answers page renders
  it('renders saved answers page', async () => {
    vi.mocked(api.listSavedAnswers).mockResolvedValue({
      answers: [
        { id: 'a1', team_id: 't', question: 'What is the backup policy?', answer: 'Daily backups.', confidence: 'high', sources: [{ source_type: 'artifact', source_id: 'a-1' }], created_by: null, created_at: '2026-06-01T00:00:00Z' },
        { id: 'a2', team_id: 't', question: 'How to handle incidents?', answer: 'Follow the runbook.', confidence: 'medium', sources: [], created_by: null, created_at: '2026-06-02T00:00:00Z' },
      ],
    });

    renderWithRouter(<SavedKnowledgeAnswersPage />);

    await waitFor(() => {
      expect(screen.getAllByTestId('saved-answer-card')).toHaveLength(2);
    });
    expect(screen.getByText('What is the backup policy?')).toBeInTheDocument();
  });

  // 11: Saved answers empty state
  it('renders saved answers empty state', async () => {
    vi.mocked(api.listSavedAnswers).mockResolvedValue({ answers: [] });

    renderWithRouter(<SavedKnowledgeAnswersPage />);

    await waitFor(() => {
      expect(screen.getByTestId('saved-answers-empty')).toBeInTheDocument();
    });
  });

  // 12: Ask answer save action works
  it('Ask Clarity answer save works', async () => {
    vi.mocked(api.saveAnswer).mockResolvedValue({
      id: 'a1', team_id: 't', question: 'Q', answer: 'A', confidence: 'high', sources: [], created_by: null, created_at: '',
    });

    renderWithRouter(
      <AskClarityAnswer
        response={{ answer: 'Answer text', sources: [], confidence: 'high', missing_info: [] }}
        question="Test question?"
      />
    );

    fireEvent.click(screen.getByTestId('ask-save-answer'));

    await waitFor(() => {
      expect(api.saveAnswer).toHaveBeenCalledWith({
        question: 'Test question?',
        answer: 'Answer text',
        confidence: 'high',
        sources: [],
      });
    });
  });

  // 13: Loading state renders
  it('renders loading state for collections', () => {
    vi.mocked(api.listCollections).mockReturnValue(new Promise(() => {}));
    renderWithRouter(<KnowledgeCollectionsPage />);
    expect(screen.getByTestId('collections-loading')).toBeInTheDocument();
  });

  // 14: Error state renders safely
  it('renders safe error state for collections', async () => {
    vi.mocked(api.listCollections).mockRejectedValue(new Error('Network error'));
    renderWithRouter(<KnowledgeCollectionsPage />);

    await waitFor(() => {
      const err = screen.getByTestId('collections-error');
      expect(err).toBeInTheDocument();
      expect(err.textContent).not.toContain('Network error');
    });
  });

  // 15: Source cards preserved in saved answers
  it('saved answer detail preserves source cards', async () => {
    vi.mocked(api.getSavedAnswer).mockResolvedValue({
      id: 'a1', team_id: 't', question: 'Q', answer: 'A long answer.', confidence: 'high',
      sources: [
        { source_type: 'artifact', source_id: 'a-1', title: 'Doc A', snippet: 'Snippet A' },
        { source_type: 'incident', source_id: 'i-1', title: 'Incident B', snippet: 'Snippet B' },
      ],
      created_by: null, created_at: '',
    });

    vi.mocked(useParams).mockReturnValue({ answerId: 'a1' });

    const { SavedKnowledgeAnswerDetailPage } = await import('../features/knowledge/SavedKnowledgeAnswerDetailPage');
    renderWithRouter(<SavedKnowledgeAnswerDetailPage />, '/knowledge/saved-answers/a1');

    await waitFor(() => {
      expect(screen.getAllByTestId('ask-clarity-source-card')).toHaveLength(2);
    });
  });

  // 16: No public/share/approval/execute controls
  it('does not render share, publish, or execute controls', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({ collections: [] });
    renderWithRouter(<KnowledgeCollectionsPage />);

    await waitFor(() => {
      expect(screen.getByTestId('collections-empty')).toBeInTheDocument();
    });

    expect(screen.queryByText(/share/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/publish/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/execute/i)).not.toBeInTheDocument();
  });

  // 17: No raw prompt/COT fields rendered
  it('does not render prompt or chain_of_thought', () => {
    renderWithRouter(
      <AskClarityAnswer
        response={{ answer: 'Answer', sources: [], confidence: 'high', missing_info: [] }}
        question="Q"
      />
    );
    expect(screen.queryByText(/prompt/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/chain_of_thought/i)).not.toBeInTheDocument();
  });

  // 18: Collections nav renders (verify component can render in router)
  it('collections page renders with nav', async () => {
    vi.mocked(api.listCollections).mockResolvedValue({ collections: [] });
    renderWithRouter(<KnowledgeCollectionsPage />);
    await waitFor(() => {
      expect(screen.getByText('Knowledge Collections')).toBeInTheDocument();
    });
  });

  // 19: Remove item from collection works
  it('removes item from collection', async () => {
    vi.mocked(api.getCollection).mockResolvedValue({
      id: 'c1', team_id: 't', name: 'Test', description: '', created_by: null, created_at: '', updated_at: '', archived_at: null, item_count: 1,
      items: [
        { id: 'i1', collection_id: 'c1', team_id: 't', source_type: 'incident', source_id: 'inc-1', title: 'Item', added_by: null, added_at: '' },
      ],
    });
    vi.mocked(api.removeCollectionItem).mockResolvedValue({ status: 'removed' });
    vi.mocked(useParams).mockReturnValue({ collectionId: 'c1' });

    renderWithRouter(<KnowledgeCollectionDetailPage />);
    await waitFor(() => {
      expect(screen.getByTestId('collection-item-card')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('remove-item-i1'));

    await waitFor(() => {
      expect(api.removeCollectionItem).toHaveBeenCalledWith('c1', 'i1');
    });
  });

  // 20: Source card save-to-collection renders
  it('Ask source card has save-to-collection button', () => {
    const { createElement } = require('react');
    renderWithRouter(
      createElement(AskClaritySourceCard, { source: { source_type: 'incident', source_id: 'inc-1', title: 'Test', snippet: 'Snippet' } })
    );
    expect(screen.getByTestId('ask-source-save-to-collection')).toBeInTheDocument();
  });
});
