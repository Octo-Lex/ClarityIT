import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    generateDocument: vi.fn(),
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
    documentAssist: vi.fn(),
    listMeetingSummaries: vi.fn(),
    getMeetingSummary: vi.fn(),
    createMeetingSummary: vi.fn(),
    updateMeetingSummary: vi.fn(),
    generateStatusReport: vi.fn(),
    generatePresentation: vi.fn(),
    listTemplates: vi.fn(),
    createTemplate: vi.fn(),
    instantiateTemplate: vi.fn(),
    downloadArtifact: vi.fn(),
    exportArtifactUrl: vi.fn(),
    getPresentonStatus: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import DocumentGenerateModal from '../features/artifacts/DocumentGenerateModal';
import { api } from '../api/client';

function renderModal(props: Partial<Parameters<typeof DocumentGenerateModal>[0]> = {}) {
  const defaultProps = {
    onClose: vi.fn(),
    onGenerated: vi.fn(),
    ...props,
  };
  return render(
    <MemoryRouter>
      <DocumentGenerateModal {...defaultProps as any} />
    </MemoryRouter>
  );
}

const mockGeneratedResponse = {
  artifact_id: 'new-doc-123',
  artifact_type: 'document',
  document_type: 'implementation_plan',
  title: 'Test Doc',
  status: 'draft',
  schema_version: 1,
  word_count: 50,
  document_json: { schema_version: 1, title: 'Test Doc', document_type: 'implementation_plan', blocks: [] },
};

beforeEach(() => {
  vi.clearAllMocks();
  (api.generateDocument as any).mockResolvedValue(mockGeneratedResponse);
});

describe('DocumentGenerateModal', () => {
  it('1. renders', () => {
    renderModal();
    expect(screen.getByTestId('generate-modal')).toBeTruthy();
  });

  it('2. title input works', () => {
    renderModal();
    const input = screen.getByTestId('generate-title') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'My Doc' } });
    expect(input.value).toBe('My Doc');
  });

  it('3. document_type selector works', () => {
    renderModal();
    const select = screen.getByTestId('generate-doc-type') as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'decision_memo' } });
    expect(select.value).toBe('decision_memo');
  });

  it('4. tone selector works', () => {
    renderModal();
    const select = screen.getByTestId('generate-tone') as HTMLSelectElement;
    fireEvent.change(select, { target: { value: 'executive' } });
    expect(select.value).toBe('executive');
  });

  it('5. prompt textarea works', () => {
    renderModal();
    const textarea = screen.getByTestId('generate-prompt') as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: 'Write about X' } });
    expect(textarea.value).toBe('Write about X');
  });

  it('6. sections add/remove works', () => {
    renderModal();
    // Add section
    fireEvent.click(screen.getByTestId('generate-add-section'));
    expect(screen.getByTestId('generate-section-0')).toBeTruthy();

    // Add another
    fireEvent.click(screen.getByTestId('generate-add-section'));
    expect(screen.getByTestId('generate-section-1')).toBeTruthy();

    // Remove first
    fireEvent.click(screen.getByTestId('generate-remove-section-0'));
    // After removal, remaining section is reindexed to 0
    expect(screen.queryByTestId('generate-section-1')).toBeNull();
    expect(screen.queryByTestId('generate-section-0')).toBeTruthy();
  });

  it('7. submit calls API', async () => {
    renderModal();
    fireEvent.change(screen.getByTestId('generate-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('generate-prompt'), { target: { value: 'Write something' } });
    await act(async () => {
      fireEvent.click(screen.getByTestId('generate-submit'));
    });
    expect(api.generateDocument).toHaveBeenCalled();
  });

  it('8. loading state renders', async () => {
    let resolvePromise: (v: any) => void;
    (api.generateDocument as any).mockReturnValue(new Promise(r => { resolvePromise = r; }));
    renderModal();
    fireEvent.change(screen.getByTestId('generate-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('generate-prompt'), { target: { value: 'Write' } });
    fireEvent.click(screen.getByTestId('generate-submit'));
    await waitFor(() => expect(screen.getByTestId('generate-loading')).toBeTruthy());
    resolvePromise!(mockGeneratedResponse);
  });

  it('9. error state renders safely', async () => {
    (api.generateDocument as any).mockRejectedValue(new Error('Server error'));
    renderModal();
    fireEvent.change(screen.getByTestId('generate-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('generate-prompt'), { target: { value: 'Write' } });
    await act(async () => {
      fireEvent.click(screen.getByTestId('generate-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('generate-error')).toBeTruthy());
    expect(screen.getByTestId('generate-error').textContent).toContain('Failed');
  });

  it('10. successful generation calls onGenerated with artifact ID', async () => {
    const onGenerated = vi.fn();
    renderModal({ onGenerated });
    fireEvent.change(screen.getByTestId('generate-title'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByTestId('generate-prompt'), { target: { value: 'Write' } });
    await act(async () => {
      fireEvent.click(screen.getByTestId('generate-submit'));
    });
    expect(onGenerated).toHaveBeenCalledWith('new-doc-123');
  });

  it('11. no export/template/version/share/approval/execute buttons rendered', () => {
    renderModal();
    expect(screen.queryByTestId('export-docx')).toBeNull();
    expect(screen.queryByTestId('export-pdf')).toBeNull();
    expect(screen.queryByTestId('export-markdown')).toBeNull();
    expect(screen.queryByTestId('version-history')).toBeNull();
    expect(screen.queryByTestId('share-button')).toBeNull();
    expect(screen.queryByTestId('approve-button')).toBeNull();
    expect(screen.queryByTestId('execute-button')).toBeNull();
    expect(screen.queryByTestId('template-gallery')).toBeNull();
  });
});
