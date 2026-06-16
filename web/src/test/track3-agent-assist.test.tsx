import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
    createDocument: vi.fn(),
    listDocuments: vi.fn(),
    documentAssist: vi.fn(),
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

import AgentAssistPanel from '../features/artifacts/AgentAssistPanel';
import { api } from '../api/client';

function renderPanel(props: Partial<Parameters<typeof AgentAssistPanel>[0]> = {}) {
  const defaultProps = {
    artifactId: 'doc-1',
    selectedBlockId: 'blk_001',
    selectedBlockText: 'Some text to work with.',
    documentType: 'implementation_plan',
    onInsertBelow: vi.fn(),
    onReplaceBlock: vi.fn(),
    ...props,
  };
  return render(
    <MemoryRouter>
      <AgentAssistPanel {...defaultProps as any} />
    </MemoryRouter>
  );
}

const mockSuggestion = {
  suggested_blocks: [
    { type: 'paragraph', text: 'Rewritten text for clarity.' },
  ],
  summary: 'Rewrote selected text for clarity.',
};

beforeEach(() => {
  vi.clearAllMocks();
  (api.documentAssist as any).mockResolvedValue(mockSuggestion);
});

describe('AgentAssistPanel', () => {
  it('1. renders', () => {
    renderPanel();
    expect(screen.getByTestId('agent-assist')).toBeTruthy();
  });

  it('2. mode selector renders all modes', () => {
    renderPanel();
    const select = screen.getByTestId('assist-mode-select') as HTMLSelectElement;
    const options = select.querySelectorAll('option');
    expect(options.length).toBe(10);
    expect(Array.from(options).map(o => o.value)).toContain('rewrite');
    expect(Array.from(options).map(o => o.value)).toContain('create_outline');
    expect(Array.from(options).map(o => o.value)).toContain('extract_action_items');
  });

  it('3. selecting block populates context', () => {
    renderPanel({ selectedBlockId: 'blk_test42', selectedBlockText: 'Block text here' });
    expect(screen.getByText(/blk_test42/)).toBeTruthy();
  });

  it('4. submit calls document-assist API', async () => {
    renderPanel();
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    expect(api.documentAssist).toHaveBeenCalled();
    const call = (api.documentAssist as any).mock.calls[0];
    expect(call[0]).toBe('doc-1');
    expect(call[1].mode).toBe('rewrite');
  });

  it('5. loading state renders', async () => {
    let resolvePromise: (v: any) => void;
    (api.documentAssist as any).mockReturnValue(new Promise(r => { resolvePromise = r; }));
    renderPanel();
    fireEvent.click(screen.getByTestId('assist-submit'));
    await waitFor(() => expect(screen.getByTestId('assist-loading')).toBeTruthy());
    resolvePromise!(mockSuggestion);
  });

  it('6. suggestion preview renders', async () => {
    renderPanel();
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('assist-suggestion')).toBeTruthy());
    expect(screen.getByText('Rewrote selected text for clarity.')).toBeTruthy();
    expect(screen.getByText('Rewritten text for clarity.')).toBeTruthy();
  });

  it('7. Insert Below inserts suggested block', async () => {
    const onInsertBelow = vi.fn();
    renderPanel({ onInsertBelow });
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('assist-insert-below')).toBeTruthy());
    fireEvent.click(screen.getByTestId('assist-insert-below'));
    expect(onInsertBelow).toHaveBeenCalled();
  });

  it('8. Replace Block replaces selected block', async () => {
    const onReplaceBlock = vi.fn();
    renderPanel({ onReplaceBlock });
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('assist-replace-block')).toBeTruthy());
    fireEvent.click(screen.getByTestId('assist-replace-block'));
    expect(onReplaceBlock).toHaveBeenCalled();
  });

  it('9. Copy action works', async () => {
    renderPanel();
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('assist-copy')).toBeTruthy());
    // Mock clipboard
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
    fireEvent.click(screen.getByTestId('assist-copy'));
    expect(navigator.clipboard.writeText).toHaveBeenCalled();
  });

  it('10. Dismiss clears suggestion', async () => {
    renderPanel();
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('assist-suggestion')).toBeTruthy());
    fireEvent.click(screen.getByTestId('assist-dismiss'));
    expect(screen.queryByTestId('assist-suggestion')).toBeNull();
  });

  it('11. error state renders safely', async () => {
    (api.documentAssist as any).mockRejectedValue(new Error('Network error'));
    renderPanel();
    await act(async () => {
      fireEvent.click(screen.getByTestId('assist-submit'));
    });
    await waitFor(() => expect(screen.getByTestId('assist-error')).toBeTruthy());
    expect(screen.getByTestId('assist-error').textContent).toContain('Failed');
  });

  it('12. no generate/export/version/share/approval/execute buttons rendered', () => {
    renderPanel();
    expect(screen.queryByTestId('generate-document')).toBeNull();
    expect(screen.queryByTestId('export-docx')).toBeNull();
    expect(screen.queryByTestId('export-pdf')).toBeNull();
    expect(screen.queryByTestId('export-markdown')).toBeNull();
    expect(screen.queryByTestId('version-history')).toBeNull();
    expect(screen.queryByTestId('share-button')).toBeNull();
    expect(screen.queryByTestId('approve-button')).toBeNull();
    expect(screen.queryByTestId('execute-button')).toBeNull();
  });
});
