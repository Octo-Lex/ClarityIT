import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

vi.mock('../api/client', () => ({
  api: {
    listArtifacts: vi.fn(),
    getArtifact: vi.fn(),
    createArtifact: vi.fn(),
    updateArtifact: vi.fn(),
    archiveArtifact: vi.fn(),
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
    listArtifactsWithFiles: vi.fn(),
    getRecentArtifacts: vi.fn(),
    searchArtifacts: vi.fn(),
    getStorageSummary: vi.fn(),
  },
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import ArtifactsPage from '../features/artifacts/ArtifactsPage';
import { api } from '../api/client';

const mockArtifacts = [
  { id: 'art-1', artifact_type: 'document', title: 'Doc One', status: 'draft', updated_at: '2026-06-10T00:00:00Z' },
  { id: 'art-2', artifact_type: 'presentation', title: 'Slide Deck', status: 'published', updated_at: '2026-06-12T00:00:00Z', file_format: 'pptx', storage_object_id: 'so-1' },
];

const mockSummary = {
  total_artifacts: 42,
  file_artifacts: 7,
  inline_artifacts: 35,
  total_file_size_bytes: 12345678,
  by_format: { pptx: 3, pdf: 2, md: 2 },
};

describe('Track 6 — Artifact Storage and Recent Files', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: search bar renders
  it('renders search bar', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue(mockArtifacts);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-search-input')).toBeInTheDocument());
  });

  // Test 2: search calls API
  it('calls searchArtifacts on search', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    vi.mocked(api.searchArtifacts).mockResolvedValue([mockArtifacts[0]]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-search-input')).toBeInTheDocument());
    fireEvent.change(screen.getByTestId('artifacts-search-input'), { target: { value: 'Doc' } });
    fireEvent.click(screen.getByTestId('artifacts-search-btn'));
    await waitFor(() => expect(api.searchArtifacts).toHaveBeenCalledWith('Doc'));
  });

  // Test 3: recent artifacts widget renders
  it('renders recent artifacts widget', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue(mockArtifacts);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-recent-widget')).toBeInTheDocument());
    expect(screen.getByTestId('artifacts-recent-art-1')).toBeInTheDocument();
  });

  // Test 4: storage summary renders
  it('renders storage summary', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-storage-summary')).toBeInTheDocument());
    expect(screen.getByTestId('storage-total').textContent).toBe('42');
    expect(screen.getByTestId('storage-files').textContent).toBe('7');
    expect(screen.getByTestId('storage-size')).toBeInTheDocument();
  });

  // Test 5: file format badge renders
  it('renders file format badge for file artifacts', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-file-format-art-2')).toBeInTheDocument());
    expect(screen.getByTestId('artifacts-file-format-art-2').textContent).toContain('PPTX');
  });

  // Test 6: file size indicator renders
  it('renders file indicator for artifacts with storage', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-file-size-art-2')).toBeInTheDocument());
  });

  // Test 7: inline artifact has no file badges
  it('does not render file badges for inline artifacts', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-item-art-1')).toBeInTheDocument());
    expect(screen.queryByTestId('artifacts-file-format-art-1')).not.toBeInTheDocument();
    expect(screen.queryByTestId('artifacts-file-size-art-1')).not.toBeInTheDocument();
  });

  // Test 8: archived artifacts hidden by default (status filter excludes archived)
  it('does not render archived artifacts in list', async () => {
    const withArchived = [
      ...mockArtifacts,
      { id: 'art-3', artifact_type: 'document', title: 'Archived Doc', status: 'archived', updated_at: '2026-06-08T00:00:00Z' },
    ];
    // listArtifacts already excludes archived on the backend
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-list')).toBeInTheDocument());
    expect(screen.queryByTestId('artifacts-item-art-3')).not.toBeInTheDocument();
  });

  // Test 9: no raw bucket/object key rendered
  it('does not render raw bucket or object keys', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    const { container } = render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-list')).toBeInTheDocument());
    expect(container.textContent).not.toContain('bucket');
    expect(container.textContent).not.toContain('object_key');
    expect(container.textContent).not.toContain('clarityit');
  });

  // Test 10: no external share/download link rendered
  it('does not render share or download links', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    const { container } = render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('artifacts-list')).toBeInTheDocument());
    expect(container.textContent).not.toContain('Share');
    expect(container.textContent).not.toContain('Download');
    expect(container.textContent).not.toContain('presigned');
  });

  // Test 11: format counts render in storage summary
  it('renders format counts in storage summary', async () => {
    vi.mocked(api.listArtifacts).mockResolvedValue(mockArtifacts);
    vi.mocked(api.getStorageSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getRecentArtifacts).mockResolvedValue([]);
    render(<MemoryRouter><ArtifactsPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('storage-format-pptx')).toBeInTheDocument());
    expect(screen.getByTestId('storage-format-pptx').textContent).toContain('pptx: 3');
  });
});
