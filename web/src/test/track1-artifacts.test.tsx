import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
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
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ArtifactsPage from '../features/artifacts/ArtifactsPage';
import { api } from '../api/client';

function renderPage() {
  return render(
    <MemoryRouter>
      <ArtifactsPage />
    </MemoryRouter>
  );
}

const mockArtifacts = [
  {
    id: 'art-1',
    artifact_type: 'report',
    title: 'Weekly Status Report',
    description: 'Week of June 10',
    status: 'published',
    content_markdown: '## Status\nAll good.',
    updated_at: '2026-06-15T00:00:00Z',
  },
  {
    id: 'art-2',
    artifact_type: 'document',
    title: 'Architecture Overview',
    description: 'System architecture',
    status: 'draft',
    content_markdown: '# Architecture',
    updated_at: '2026-06-14T00:00:00Z',
  },
];

describe('ArtifactsPage — Internal Document/Report Workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: artifacts page renders
  it('renders artifacts page', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-page')).toBeInTheDocument());
  });

  // Test 2: empty state renders
  it('renders empty state when no artifacts', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue([]);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-empty')).toBeInTheDocument());
  });

  // Test 3: artifact list renders type/status badges
  it('renders type and status badges', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-item-art-1')).toBeInTheDocument());
    expect(screen.getByTestId('artifacts-type-art-1').textContent).toContain('report');
    expect(screen.getByTestId('artifacts-status-art-1').textContent).toContain('published');
    expect(screen.getByTestId('artifacts-type-art-2').textContent).toContain('document');
    expect(screen.getByTestId('artifacts-status-art-2').textContent).toContain('draft');
  });

  // Test 4: type filter works
  it('calls listArtifacts with type filter', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-type-filter')).toBeInTheDocument());
    fireEvent.change(screen.getByTestId('artifacts-type-filter'), { target: { value: 'report' } });
    expect(api.listArtifacts).toHaveBeenCalledWith({ type: 'report', q: undefined });
  });

  // Test 5: search input calls search API path
  it('calls listArtifacts on search', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.searchArtifacts).mockResolvedValue(mockArtifacts);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-search-input')).toBeInTheDocument());
    fireEvent.change(screen.getByTestId('artifacts-search-input'), { target: { value: 'weekly' } });
    fireEvent.click(screen.getByTestId('artifacts-search-btn'));
    await waitFor(() => expect(api.searchArtifacts).toHaveBeenCalledWith('weekly'));
  });

  // Test 6: create artifact form renders
  it('renders create form on button click', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-create-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-create-btn'));
    await waitFor(() => expect(screen.getByTestId('artifact-editor')).toBeInTheDocument());
    expect(screen.getByTestId('editor-type')).toBeInTheDocument();
    expect(screen.getByTestId('editor-title')).toBeInTheDocument();
  });

  // Test 7: editor renders markdown content (edit mode)
  it('renders editor with markdown content in edit mode', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getArtifact).mockResolvedValue(mockArtifacts[0]);
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-item-art-1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-item-art-1'));
    await waitFor(() => expect(screen.getByTestId('editor-content')).toBeInTheDocument());
    expect((screen.getByTestId('editor-content') as HTMLTextAreaElement).value).toContain('## Status');
  });

  // Test 8: save/update calls API
  it('calls create API on save', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.createArtifact).mockResolvedValue({ id: 'new-id' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-create-btn')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-create-btn'));
    await waitFor(() => expect(screen.getByTestId('editor-save')).toBeInTheDocument());
    fireEvent.change(screen.getByTestId('editor-title'), { target: { value: 'New Doc' } });
    fireEvent.click(screen.getByTestId('editor-save'));
    await waitFor(() => expect(api.createArtifact).toHaveBeenCalledTimes(1));
  });

  // Test 9: archive/delete action calls API
  it('calls archive API on archive button', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getArtifact).mockResolvedValue(mockArtifacts[0]);
    vi.mocked(api.archiveArtifact).mockResolvedValue({ status: 'archived' });
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-item-art-1')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('artifacts-item-art-1'));
    await waitFor(() => expect(screen.getByTestId('editor-archive')).toBeInTheDocument());
    fireEvent.click(screen.getByTestId('editor-archive'));
    await waitFor(() => expect(api.archiveArtifact).toHaveBeenCalledTimes(1));
  });

  // Test 10: unauthorized state renders
  it('renders error state when API fails', async () => {
    vi.mocked(api.listArtifacts).mockRejectedValue(new Error('Unauthorized'));
    renderPage();
    await waitFor(() => expect(screen.getByTestId('artifacts-unauthorized')).toBeInTheDocument());
  });
});
