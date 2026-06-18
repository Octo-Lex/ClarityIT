import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

vi.mock('../api/client', () => ({
  api: {
    listVersions: vi.fn(),
    getVersion: vi.fn(),
    restoreVersion: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

vi.mock('../auth/context', () => ({
  useAuth: () => ({ token: 'test-token', user: { id: 'u1', email: 'test@test.dev' } }),
}));

import { api } from '../api/client';
import VersionHistoryDrawer from '../features/artifacts/VersionHistoryDrawer';

const mockVersions = [
  { id: 'v3', version_number: 3, word_count: 420, source: 'user_save', change_summary: 'Updated risks', created_at: '2026-06-16T10:00:00Z' },
  { id: 'v2', version_number: 2, word_count: 389, source: 'agent_assisted_edit', created_at: '2026-06-15T10:00:00Z' },
  { id: 'v1', version_number: 1, word_count: 350, source: 'generated', created_at: '2026-06-14T10:00:00Z' },
];

const mockVersionDetail = {
  id: 'v2',
  version_number: 2,
  document_json: { schema_version: 1, title: 'Test', document_type: 'general_document', blocks: [{ id: 'b1', type: 'paragraph', text: 'Version 2 content' }] },
  word_count: 389,
  source: 'agent_assisted_edit',
  created_at: '2026-06-15T10:00:00Z',
};

beforeEach(() => {
  vi.clearAllMocks();
  (api.listVersions as any).mockResolvedValue({ versions: mockVersions });
  (api.getVersion as any).mockResolvedValue(mockVersionDetail);
  (api.restoreVersion as any).mockResolvedValue({
    artifact_id: 'a1',
    restored_from_version: 2,
    new_version_number: 4,
    document_json: mockVersionDetail.document_json,
    word_count: 389,
  });
});

describe('Version History (Track 7)', () => {
  const defaultProps = {
    artifactId: 'art-1',
    open: true,
    onClose: vi.fn(),
    archived: false,
    onRestored: vi.fn(),
  };

  it('1. Version History button area renders when drawer open', () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    expect(screen.getByTestId('version-drawer')).toBeTruthy();
  });

  it('2. drawer renders version list', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId('version-list')).toBeTruthy();
    });
  });

  it('3. version list shows version numbers', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText('v3')).toBeTruthy();
      expect(screen.getByText('v2')).toBeTruthy();
      expect(screen.getByText('v1')).toBeTruthy();
    });
  });

  it('4. source badges render correctly', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId('version-badge-3').textContent).toBe('Saved');
      expect(screen.getByTestId('version-badge-2').textContent).toBe('AI Assist');
      expect(screen.getByTestId('version-badge-1').textContent).toBe('Generated');
    });
  });

  it('5. word count renders', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/420 words/)).toBeTruthy();
    });
  });

  it('6. selecting version shows preview', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-item-2')).toBeTruthy());
    fireEvent.click(screen.getByTestId('version-item-2'));
    await waitFor(() => {
      expect(screen.getByTestId('version-preview')).toBeTruthy();
    });
  });

  it('7. restore opens confirmation dialog', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-item-2')).toBeTruthy());
    fireEvent.click(screen.getByTestId('version-item-2'));
    await waitFor(() => expect(screen.getByTestId('restore-button')).toBeTruthy());
    fireEvent.click(screen.getByTestId('restore-button'));
    expect(screen.getByTestId('restore-confirm')).toBeTruthy();
  });

  it('8. confirm restore calls API', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-item-2')).toBeTruthy());
    fireEvent.click(screen.getByTestId('version-item-2'));
    await waitFor(() => expect(screen.getByTestId('restore-button')).toBeTruthy());
    fireEvent.click(screen.getByTestId('restore-button'));
    fireEvent.click(screen.getByTestId('restore-confirm-button'));
    await waitFor(() => {
      expect(api.restoreVersion).toHaveBeenCalledWith('art-1', 'v2');
    });
  });

  it('9. successful restore calls onRestored callback', async () => {
    const onRestored = vi.fn();
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} onRestored={onRestored} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-item-2')).toBeTruthy());
    fireEvent.click(screen.getByTestId('version-item-2'));
    await waitFor(() => expect(screen.getByTestId('restore-button')).toBeTruthy());
    fireEvent.click(screen.getByTestId('restore-button'));
    fireEvent.click(screen.getByTestId('restore-confirm-button'));
    await waitFor(() => {
      expect(onRestored).toHaveBeenCalled();
    });
  });

  it('10. successful restore refreshes version list', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(api.listVersions).toHaveBeenCalledTimes(1));
    // After restore, listVersions should be called again
    fireEvent.click(await screen.findByTestId('version-item-2'));
    await waitFor(() => expect(screen.getByTestId('restore-button')).toBeTruthy());
    fireEvent.click(screen.getByTestId('restore-button'));
    fireEvent.click(screen.getByTestId('restore-confirm-button'));
    await waitFor(() => {
      expect(api.listVersions).toHaveBeenCalledTimes(2);
    });
  });

  it('11. restore error renders safely', async () => {
    (api.restoreVersion as any).mockRejectedValueOnce(new Error('fail'));
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-item-2')).toBeTruthy());
    fireEvent.click(screen.getByTestId('version-item-2'));
    await waitFor(() => expect(screen.getByTestId('restore-button')).toBeTruthy());
    fireEvent.click(screen.getByTestId('restore-button'));
    fireEvent.click(screen.getByTestId('restore-confirm-button'));
    await waitFor(() => {
      expect(screen.getByTestId('version-error')).toBeTruthy();
    });
  });

  it('12. archived document shows restore unavailable', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} archived /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-item-2')).toBeTruthy());
    fireEvent.click(screen.getByTestId('version-item-2'));
    await waitFor(() => {
      expect(screen.getByTestId('restore-archived-notice')).toBeTruthy();
    });
  });

  it('13. no delete version button rendered', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-list')).toBeTruthy());
    expect(screen.queryByTestId('delete-version')).toBeNull();
  });

  it('14. no approval/share/execute buttons rendered', async () => {
    render(<MemoryRouter><VersionHistoryDrawer {...defaultProps} /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-list')).toBeTruthy());
    expect(screen.queryByTestId('approve-button')).toBeNull();
    expect(screen.queryByTestId('share-button')).toBeNull();
    expect(screen.queryByTestId('execute-button')).toBeNull();
  });
});
