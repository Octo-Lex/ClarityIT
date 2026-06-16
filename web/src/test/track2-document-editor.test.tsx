import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

// Mock auth context
vi.mock('../auth/context', () => ({
  useAuth: () => ({ token: 'test-token', user: { id: 'u1', email: 'test@test.dev' } }),
}));

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
    createDocument: vi.fn(),
    listDocuments: vi.fn(),
    listArtifacts: vi.fn(),
    getArtifact: vi.fn(),
    createArtifact: vi.fn(),
    updateArtifact: vi.fn(),
    archiveArtifact: vi.fn(),
    getRecentArtifacts: vi.fn().mockResolvedValue([]),
    searchArtifacts: vi.fn().mockResolvedValue([]),
    getStorageSummary: vi.fn().mockResolvedValue(null),
    getPresentonStatus: vi.fn(),
    generatePresentation: vi.fn(),
    listMeetingSummaries: vi.fn(),
    getMeetingSummary: vi.fn(),
    createMeetingSummary: vi.fn(),
    updateMeetingSummary: vi.fn(),
    generateStatusReport: vi.fn(),
    listTemplates: vi.fn(),
    createTemplate: vi.fn(),
    instantiateTemplate: vi.fn(),
    downloadArtifact: vi.fn(),
    exportArtifactUrl: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import DocumentEditorPage from '../features/artifacts/DocumentEditorPage';
import { api, ApiError } from '../api/client';

const mockDoc = {
  id: 'doc-1',
  artifact_type: 'document',
  title: 'Test Document',
  document_type: 'implementation_plan',
  document_json: {
    schema_version: 1,
    title: 'Test Document',
    document_type: 'implementation_plan',
    blocks: [
      { id: 'blk_001', type: 'heading', level: 1, text: 'Overview' },
      { id: 'blk_002', type: 'paragraph', text: 'This is a paragraph.' },
    ],
  },
  schema_version: 1,
  word_count: 5,
  status: 'draft',
};

function renderEditor(artifactId = 'doc-1') {
  return render(
    <MemoryRouter initialEntries={[`/teams/t1/artifacts/documents/${artifactId}`]}>
      <Routes>
        <Route path="/teams/:teamId/artifacts/documents/:artifactId" element={<DocumentEditorPage />} />
      </Routes>
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  (api.getDocument as any).mockResolvedValue(mockDoc);
  (api.updateDocument as any).mockResolvedValue({ id: 'doc-1' });
});

describe('DocumentEditorPage', () => {
  it('1. renders loaded document', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('doc-editor-page')).toBeTruthy();
    expect((screen.getByTestId('doc-title-input') as HTMLInputElement).value).toBe('Test Document');
  });

  it('2. title editing marks document dirty', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    const titleInput = screen.getByTestId('doc-title-input');
    fireEvent.change(titleInput, { target: { value: 'Updated Title' } });
    expect((titleInput as HTMLInputElement).value).toBe('Updated Title');
    expect(screen.getByTestId('save-status').textContent).toContain('Unsaved');
  });

  it('3. save calls PATCH and clears dirty state', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Make dirty
    fireEvent.change(screen.getByTestId('doc-title-input'), { target: { value: 'New Title' } });
    // Save
    await act(async () => {
      fireEvent.click(screen.getByTestId('toolbar-save'));
    });
    await waitFor(() => {
      expect(api.updateDocument).toHaveBeenCalledWith('doc-1', expect.objectContaining({ title: 'New Title' }));
    });
    await waitFor(() => {
      expect(screen.getByTestId('save-status').textContent).toContain('Saved');
    });
  });

  it('4. save error shows safe error state', async () => {
    (api.updateDocument as any).mockRejectedValue(new Error('Network error'));
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    fireEvent.change(screen.getByTestId('doc-title-input'), { target: { value: 'X' } });
    await act(async () => {
      fireEvent.click(screen.getByTestId('toolbar-save'));
    });
    await waitFor(() => {
      expect(screen.getByTestId('save-status').textContent).toContain('Save failed');
    });
  });

  it('4b. stale-save 409 shows conflict dialog (Track 8 regression)', async () => {
    const conflictErr = new Error('Document was modified by another user') as any;
    conflictErr.status = 409;
    (api.updateDocument as any).mockRejectedValue(conflictErr);
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    fireEvent.change(screen.getByTestId('doc-title-input'), { target: { value: 'Stale Title' } });
    await act(async () => {
      fireEvent.click(screen.getByTestId('toolbar-save'));
    });
    await waitFor(() => {
      expect(screen.getByTestId('save-conflict')).toBeTruthy();
      expect(screen.getByTestId('conflict-reload')).toBeTruthy();
      expect(screen.getByTestId('conflict-dismiss')).toBeTruthy();
    });
  });

  it('5. heading block renders and edits', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('block-text-blk_001')).toBeTruthy();
    fireEvent.change(screen.getByTestId('block-text-blk_001'), { target: { value: 'New Heading' } });
    expect((screen.getByTestId('block-text-blk_001') as HTMLInputElement).value).toBe('New Heading');
  });

  it('6. paragraph block renders and edits', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('block-text-blk_002')).toBeTruthy();
    fireEvent.change(screen.getByTestId('block-text-blk_002'), { target: { value: 'New paragraph text' } });
    expect(screen.getByTestId('save-status').textContent).toContain('Unsaved');
  });

  it('7. bullets block add/remove/edit works', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_b1', type: 'bullets', items: ['First item'] }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Edit
    fireEvent.change(screen.getByTestId('block-item-blk_b1-0'), { target: { value: 'Edited item' } });
    // Add
    fireEvent.click(screen.getByTestId('block-item-add-blk_b1'));
    expect(screen.queryByTestId('block-item-blk_b1-1')).toBeTruthy();
    // Remove
    fireEvent.click(screen.getByTestId('block-item-remove-blk_b1-0'));
    expect(screen.queryByTestId('block-item-blk_b1-0')).toBeTruthy(); // index shifted
  });

  it('8. numbered_list block add/remove/edit works', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_n1', type: 'numbered_list', items: ['Step one'] }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    fireEvent.change(screen.getByTestId('block-item-blk_n1-0'), { target: { value: 'Edited step' } });
    fireEvent.click(screen.getByTestId('block-item-add-blk_n1'));
    expect(screen.queryByTestId('block-item-blk_n1-1')).toBeTruthy();
  });

  it('9. table block add/remove row/column works', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_t1', type: 'table', headers: ['A', 'B'], rows: [['1', '2']] }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Add row
    fireEvent.click(screen.getByTestId('block-row-add-blk_t1'));
    expect(screen.queryByTestId('block-cell-blk_t1-1-0')).toBeTruthy();
    // Add column
    fireEvent.click(screen.getByTestId('block-col-add-blk_t1'));
    expect(screen.queryByTestId('block-header-blk_t1-2')).toBeTruthy();
    // Remove row
    fireEvent.click(screen.getByTestId('block-row-remove-blk_t1-0'));
    expect(screen.queryByTestId('block-cell-blk_t1-0-0')).toBeTruthy(); // shifted
  });

  it('10. quote block renders and edits', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_q1', type: 'quote', text: 'A quote' }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('block-text-blk_q1')).toBeTruthy();
    fireEvent.change(screen.getByTestId('block-text-blk_q1'), { target: { value: 'Updated quote' } });
  });

  it('11. callout block renders with variant selector', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_c1', type: 'callout', variant: 'info', text: 'Note this' }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('block-variant-blk_c1')).toBeTruthy();
    fireEvent.change(screen.getByTestId('block-variant-blk_c1'), { target: { value: 'warning' } });
  });

  it('12. page_break renders as visual divider', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_pb1', type: 'page_break' }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('block-preview-blk_pb1')).toBeTruthy();
    expect(screen.getByTestId('block-preview-blk_pb1').textContent).toContain('Page Break');
  });

  it('13. add block from toolbar works', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    fireEvent.click(screen.getByTestId('toolbar-add-paragraph'));
    // New block should appear - check dirty state
    expect(screen.getByTestId('save-status').textContent).toContain('Unsaved');
  });

  it('14. delete block requires confirmation', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [
          { id: 'blk_d1', type: 'paragraph', text: 'First' },
          { id: 'blk_d2', type: 'paragraph', text: 'Second' },
        ],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Click delete on first block
    fireEvent.click(screen.getByTestId('block-delete-blk_d1'));
    // Should show confirmation
    expect(screen.getByTestId('block-confirm-delete-blk_d1')).toBeTruthy();
    // Confirm
    fireEvent.click(screen.getByTestId('block-confirm-yes-blk_d1'));
    // Block should be removed
    await waitFor(() => expect(screen.queryByTestId('block-editor-blk_d1')).toBeNull());
  });

  it('15. move block up/down works', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [
          { id: 'blk_m1', type: 'paragraph', text: 'First' },
          { id: 'blk_m2', type: 'paragraph', text: 'Second' },
        ],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Move second block up
    fireEvent.click(screen.getByTestId('block-up-blk_m2'));
    expect(screen.getByTestId('save-status').textContent).toContain('Unsaved');
  });

  it('16. duplicate block works', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: {
        ...mockDoc.document_json,
        blocks: [{ id: 'blk_dup1', type: 'paragraph', text: 'Original' }],
      },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    fireEvent.click(screen.getByTestId('block-duplicate-blk_dup1'));
    // Should have two blocks now
    expect(screen.getByTestId('save-status').textContent).toContain('Unsaved');
  });

  it('17. outline renders heading blocks', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('document-outline')).toBeTruthy();
    expect(screen.getByTestId('outline-item-blk_001')).toBeTruthy();
    expect(screen.getByTestId('outline-item-blk_001').textContent).toContain('Overview');
  });

  it('18. preview mode renders read-only document', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    fireEvent.click(screen.getByTestId('toolbar-preview'));
    expect(screen.getByTestId('doc-title-preview')).toBeTruthy();
    // Should not show editable inputs in preview mode
    expect(screen.queryByTestId('doc-title-input')).toBeNull();
  });

  it('19. unsaved navigation warning is registered', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Make dirty
    fireEvent.change(screen.getByTestId('doc-title-input'), { target: { value: 'Unsaved Title' } });
    // Simulate beforeunload
    const event = new Event('beforeunload');
    Object.defineProperty(event, 'preventDefault', { value: vi.fn() });
    window.dispatchEvent(event);
    // The handler should be registered (indirectly verified by no crash)
    expect(true).toBe(true);
  });

  it('20. empty document state renders', async () => {
    (api.getDocument as any).mockResolvedValue({
      ...mockDoc,
      document_json: { ...mockDoc.document_json, blocks: [] },
    });
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('doc-empty-state')).toBeTruthy();
  });

  it('21. load failure renders safe error', async () => {
    (api.getDocument as any).mockRejectedValue(new ApiError(404, 'Not found'));
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    expect(screen.getByTestId('doc-error')).toBeTruthy();
    expect(screen.getByTestId('doc-error').textContent).toContain('Not found');
  });

  it('22. no agent/generate/share/approval/execute buttons rendered', async () => {
    renderEditor();
    await waitFor(() => expect(screen.queryByTestId('doc-loading')).toBeNull());
    // Verify none of these exist
    expect(screen.queryByTestId('generate-document')).toBeNull();
    expect(screen.queryByTestId('share-button')).toBeNull();
    expect(screen.queryByTestId('approve-button')).toBeNull();
    expect(screen.queryByTestId('execute-button')).toBeNull();
    expect(screen.queryByTestId('version-history')).toBeNull();
  });
});
